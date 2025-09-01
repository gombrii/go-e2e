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

	parsedBody = parseBody(body, resp.Header.Get("Content-Type"))

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

	if reqSetup.Content != "" {
		req.Header.Set("Content-Type", reqSetup.Content)
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing: %v", err)
	}

	return resp, nil
}

func printReq(buf *bytes.Buffer, req Request) {
	fmt.Fprintln(buf, "->", req.Method, req.URL)
	for _, h := range req.Headers {
		fmt.Fprintf(buf, "-> %s: %s\n", h.Key, h.Val)
	}
	if len(req.Body) > 0 {
		fmt.Fprint(buf, "-> "+format([]byte(req.Body)))
	}
}
func printResp(buf *bytes.Buffer, resp *http.Response, body []byte, expected Expect) {
	fmt.Fprintln(buf, "<-", resp.StatusCode)
	for k, v := range resp.Header {
		if slices.ContainsFunc(expected.Headers, func(header header) bool {
			return header.Key == k
		}) {
			fmt.Fprintf(buf, "<- %s: %s\n", k, strings.Join(v, "; "))
		}
	}
	formattedBody := ""
	if len(body) > 0 {
		formattedBody = "<- " + format(body)
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
			return fmt.Errorf("missing value for %q. Want at least:%q", h.Key, h.Val)
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
			return fmt.Errorf("unexpected value of field %q,\nno match among: %v\nwant at least: %v", field, vals, want)
		}
	}
	return nil
}

func flattenJSON(v any, prefix string, out map[string][]string) {
	switch x := v.(type) {
	case map[string]any:
		for k, vv := range x {
			p := k
			if prefix != "" {
				p = prefix + "." + k
			}
			flattenJSON(vv, p, out)
		}
	case []any:
		for _, vv := range x {
			// same prefix, no indices → collect all leaves under same path
			flattenJSON(vv, prefix, out)
		}
	default:
		if prefix != "" {
			out[prefix] = append(out[prefix], fmt.Sprint(x))
		}
	}
}

func xmlToFlat(b []byte) (map[string][]string, error) {
	dec := xml.NewDecoder(bytes.NewReader(b))
	out := map[string][]string{}
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
			// attributes become path@attr
			if len(t.Attr) > 0 {
				key := strings.Join(stack, ".")
				for _, a := range t.Attr {
					out[key+"@"+a.Name.Local] = append(out[key+"@"+a.Name.Local], a.Value)
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

func parseBody(body []byte, contentType string) map[string][]string {
	flat := make(map[string][]string)

	switch {
	case strings.Contains(contentType, "json"):
		var v any
		if err := json.Unmarshal(body, &v); err == nil {
			flattenJSON(v, "", flat)
		}
	case strings.Contains(contentType, "xml"):
		if m, err := xmlToFlat(body); err == nil {
			flat = m
		}
	}

	// TODO: Needs error handling for when unmarshalling isn't possible or content-type is not supported

	return flat
}
