package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
)

type testResult struct {
	buf    *bytes.Buffer
	passed bool
}

func performTest(client *http.Client, buf *bytes.Buffer, req Request, expected Expect) (parsedBody map[string][]string, res testResult) {
	printReq(buf, req)

	resp, err := makeRequest(client, req)
	if err != nil {
		fmt.Fprintf(buf, "\n%s: making request: %v\n", pink("ERROR"), err)
		return map[string][]string{}, testResult{buf, false}
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(buf, "\n%s: reading response body: %v\n", pink("ERROR"), err)
		return map[string][]string{}, testResult{buf, false}
	}

	printResp(buf, resp, body, expected)

	parsedBody, err = parseBody(body, resp.Header.Get("Content-Type"))
	if err != nil {
		fmt.Fprintf(buf, "\n%s: parsing response body: %v\n", pink("ERROR"), err)
		return map[string][]string{}, testResult{buf, false}
	}

	if err := assertStatus(expected.Status, resp.StatusCode); err != nil {
		fmt.Fprintf(buf, "\n%s: asserting status: %v\n", pink("FAIL"), err)
		return map[string][]string{}, testResult{buf, false}
	}
	if err := assertHeaders(expected.Headers, resp.Header); err != nil {
		fmt.Fprintf(buf, "\n%s: asserting header: %v\n", pink("FAIL"), err)
		return map[string][]string{}, testResult{buf, false}
	}
	if err := assertBody(expected.Body, parsedBody); err != nil {
		fmt.Fprintf(buf, "\n%s: asserting body: %v\n", pink("FAIL"), err)
		return map[string][]string{}, testResult{buf, false}
	}

	return parsedBody, testResult{buf, true}
}

func makeRequest(client *http.Client, reqSetup Request) (*http.Response, error) {
	if reqSetup.CTX == nil {
		reqSetup.CTX = context.Background()
	}

	req, err := http.NewRequestWithContext(reqSetup.CTX, reqSetup.Method, reqSetup.URL, io.NopCloser(strings.NewReader(reqSetup.Body)))
	if err != nil {
		return nil, fmt.Errorf("setting up: %v", err)
	}

	for _, h := range reqSetup.Headers {
		req.Header.Add(h.Key, h.Val)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing: %v", err)
	}

	return resp, nil
}

func printReq(buf *bytes.Buffer, req Request) {
	fmt.Fprintln(buf, grey("->"), req.Method, req.URL)
	for _, h := range req.Headers {
		fmt.Fprintf(buf, grey("-> ")+"%s: %s\n", h.Key, h.Val)
	}
	if len(req.Body) > 0 {
		fmt.Fprint(buf, grey("-> ")+format([]byte(req.Body), req.Content))
	}
}
func printResp(buf *bytes.Buffer, resp *http.Response, body []byte, expected Expect) {
	fmt.Fprintln(buf, grey("<-"), resp.StatusCode)
	for k, v := range resp.Header {
		if slices.ContainsFunc(expected.Headers, func(header header) bool {
			return header.Key == k
		}) {
			fmt.Fprintf(buf, grey("<- ")+"%s: %s\n", k, strings.Join(v, "; "))
		}
	}
	formattedBody := ""
	if len(body) > 0 {
		formattedBody = grey("<- ") + format(body, resp.Header.Get("Content-Type"))
	}
	fmt.Fprint(buf, formattedBody)
}

func assertStatus(expected int, actual int) error {
	if expected != 0 && expected != actual {
		return fmt.Errorf("unexpected code, got: %d want: %d", actual, expected)
	}
	return nil
}

func assertHeaders(expected []header, actual http.Header) error {
	for _, h := range expected {
		res, ok := actual[h.Key]
		if !ok {
			return fmt.Errorf("missing %q", h.Key)
		}

		hasValue := false
		for _, v := range res {
			if strings.Contains(fmt.Sprint(v), fmt.Sprint(h.Val)) {
				hasValue = true
			}
		}
		if !hasValue {
			return fmt.Errorf("missing value for %q. Want at least: %q", h.Key, h.Val)
		}
	}
	return nil
}

func assertBody(expected Body, actual map[string][]string) error {
	for field, exp := range expected {
		vals, ok := actual[field]
		if !ok || len(vals) == 0 {
			return fmt.Errorf("missing field %q", field)
		}
		want := fmt.Sprint(exp)
		found := false
		for _, got := range vals {
			if strings.Contains(got, want) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unexpected value of field %q,\nno match among: %v\nwant at least: %v", field, strings.Join(vals, ", "), want)
		}
	}
	return nil
}

func flattenJSON(body any, prefix string, out map[string][]string) {
	switch x := body.(type) {
	case map[string]any:
		// Adds entries for all non leaf nodes as well to be asserted with ""
		if prefix != "" {
			out[prefix] = []string{fmt.Sprintf("EXISTS_%d", time.Now().Unix())}
		}
		for key, value := range x {
			p := key
			if prefix != "" {
				p = prefix + "." + key
			}
			flattenJSON(value, p, out)
		}
	case []any:
		for _, values := range x {
			flattenJSON(values, prefix, out)
		}
		// We want an empty array to count as a leaf
		if prefix != "" && len(x) == 0 {
			out[prefix] = append(out[prefix], fmt.Sprintf("EXISTS_%d", time.Now().Unix()))
		}
	default:
		if prefix != "" {
			out[prefix] = append(out[prefix], fmt.Sprint(x))
		}
	}
}

func xmlToFlat(b []byte) (map[string][]string, error) {
	dec := xml.NewDecoder(bytes.NewReader(b))
	out := make(map[string][]string)
	var stack []string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			stack = append(stack, t.Name.Local)
			if len(t.Attr) > 0 {
				path := strings.Join(stack, ".")
				for _, a := range t.Attr {
					key := path + "@" + a.Name.Local
					out[key] = append(out[key], a.Value)
				}
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			s := strings.TrimSpace(string(t))
			if s == "" {
				continue
			}
			key := strings.Join(stack, ".")
			out[key] = append(out[key], s)
		}
	}
}

func parseBody(body []byte, contentType string) (map[string][]string, error) {
	if len(body) == 0 {
		return nil, nil
	}

	flat := make(map[string][]string)

	switch {
	case strings.Contains(contentType, "json"):
		var v any
		err := json.Unmarshal(body, &v)
		if err != nil {
			return nil, err
		}
		flattenJSON(v, "", flat)
	case strings.Contains(contentType, "xml"):
		m, err := xmlToFlat(body)
		if err != nil {
			return nil, err
		}
		flat = m
	default:
		return nil, fmt.Errorf("unsupported Content-Type %v", contentType)
	}

	return flat, nil
}
