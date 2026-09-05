package e2e

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func performTest(client *http.Client, buf *bytes.Buffer, req Request, expected Expect, verbose bool) (parsedBody map[string][]string, res bool) {
	printReq(buf, req)

	resp, err := makeRequest(client, req)
	if err != nil {
		fmt.Fprintf(buf, "%s: making request: %v\n", pink("ERROR"), err)
		return map[string][]string{}, false
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(buf, "%s: reading response body: %v\n", pink("ERROR"), err)
		return map[string][]string{}, false
	}

	printResp(buf, resp, body, expected, verbose)

	parsedBody, err = parseBody(body, resp.Header.Get("Content-Type"))
	if err != nil {
		fmt.Fprintf(buf, "%s: parsing response body: %v\n", yellow("WARNING"), err)
	}

	if err := assertStatus(expected.Status, resp.StatusCode); err != nil {
		fmt.Fprintf(buf, "%s: status: %v\n", pink("FAIL"), err)
		return map[string][]string{}, false
	}
	if err := assertHeaders(expected.Headers, resp.Header); err != nil {
		fmt.Fprintf(buf, "%s: header: %v\n", pink("FAIL"), err)
		return map[string][]string{}, false
	}
	if err := assertBody(expected.Body, parsedBody); err != nil {
		fmt.Fprintf(buf, "%s: body: %v\n", pink("FAIL"), err)
		return map[string][]string{}, false
	}

	fmt.Fprintln(buf, green("SUCCESS"))
	return parsedBody, true
}

func makeRequest(client *http.Client, reqSetup Request) (*http.Response, error) {
	req, err := http.NewRequest(reqSetup.Method, reqSetup.URL, io.NopCloser(strings.NewReader(reqSetup.Body)))
	if err != nil {
		return nil, fmt.Errorf("setting up: %v", err)
	}

	for k, v := range reqSetup.Headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing: %v", err)
	}

	return resp, nil
}

func printReq(buf *bytes.Buffer, req Request) {
	fmt.Fprintln(buf, grey("->"), req.Method, req.URL)
	for k, v := range req.Headers {
		fmt.Fprintf(buf, grey("-> ")+"%s: %s\n", k, v)
	}
	if len(req.Body) > 0 {
		fmt.Fprint(buf, grey("-> ")+ensureEndingNL(format([]byte(req.Body), req.Headers["Content-Type"])))
	}
}
func printResp(buf *bytes.Buffer, resp *http.Response, body []byte, expected Expect, verbose bool) {
	fmt.Fprintln(buf, grey("<-"), resp.StatusCode)
	for k, v := range resp.Header {
		if _, inExpected := expected.Headers[k]; verbose || inExpected {
			fmt.Fprintf(buf, grey("<- ")+"%s: %s\n", k, strings.Join(v, "; "))
		}
	}
	formattedBody := ""
	if len(body) > 0 {
		formattedBody = grey("<- ") + ensureEndingNL(format(body, resp.Header.Get("Content-Type")))
	}
	fmt.Fprint(buf, formattedBody)
}

func assertStatus(expected int, actual int) error {
	if expected != 0 && expected != actual {
		return fmt.Errorf("got %d, expected %d", actual, expected)
	}
	return nil
}

func assertHeaders(expected Headers, actual http.Header) error {
	for key, val := range expected {
		res, ok := actual[key]
		if !ok {
			return fmt.Errorf("%q not present", key)
		}

		hasValue := false
		for _, v := range res {
			if strings.Contains(v, val) {
				hasValue = true
			}
		}
		if !hasValue {
			if len(res) == 1 {
				return fmt.Errorf("%q: %q does not contain %q", key, res[0], val)
			}
			return fmt.Errorf("%q: none of %v contains %q", key, res, val)
		}
	}
	return nil
}

func assertBody(expected Body, actual map[string][]string) error {
	for field, exp := range expected {
		vals, ok := actual[field]
		if !ok || len(vals) == 0 {
			return fmt.Errorf("%q not present", field)
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
			// Filter out branch sentinels ("") before displaying actual values
			display := make([]string, 0, len(vals))
			for _, v := range vals {
				if v != "" {
					display = append(display, v)
				}
			}
			if len(display) == 1 {
				return fmt.Errorf("%q: %q does not contain %q", field, display[0], want)
			}
			return fmt.Errorf("%q: none of %v contains %q", field, display, want)
		}
	}
	return nil
}

func flattenJSON(body any, prefix string, out map[string][]string) {
	switch x := body.(type) {
	case map[string]any:
		// Adds entries for all non leaf nodes as well to be asserted with "".
		if prefix != "" {
			out[prefix] = []string{""}
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
		// We want an empty array to count as a leaf.
		if prefix != "" && len(x) == 0 {
			out[prefix] = append(out[prefix], "")
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
			path := strings.Join(stack, ".")
			// Adds entries for all non leaf nodes as well to be asserted with "".
			out[path] = append(out[path], "")
			if len(t.Attr) > 0 {
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
