package e2e

import (
	"bytes"
	"context"
	"encoding/json"
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

func performTest(client *http.Client, buf *bytes.Buffer, req Request, expected Expect) (parsedBody map[string]any, res testResult) {
	printReq(buf, req)

	resp, err := makeRequest(client, req)
	if err != nil {
		fmt.Fprintf(buf, "\n%s: making request: %v\n", pink("ERROR"), err)
		return map[string]any{}, testResult{buf, false}
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(buf, "\n%s: reading response body: %v\n", pink("ERROR"), err)
		return map[string]any{}, testResult{buf, false}
	}

	printResp(buf, resp, body, expected)

	parsedBody = make(map[string]any)
	json.Unmarshal(body, &parsedBody)

	if err := assertStatus(expected.Status, resp.StatusCode); err != nil {
		fmt.Fprintf(buf, "\n%s: asserting status: %v\n", pink("FAIL"), err)
		return map[string]any{}, testResult{buf, false}
	}
	if err := assertHeaders(expected.Headers, resp.Header); err != nil {
		fmt.Fprintf(buf, "\n%s: asserting header: %v\n", pink("FAIL"), err)
		return map[string]any{}, testResult{buf, false}
	}
	if err := assertBody(expected.Body, parsedBody); err != nil {
		fmt.Fprintf(buf, "\n%s: asserting body: %v\n", pink("FAIL"), err)
		return map[string]any{}, testResult{buf, false}
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

func assertBody(expected Body, actual map[string]any) error {
	for field, exp := range expected {
		res, ok := actual[field]
		if !ok {
			return fmt.Errorf("missing field %q", field)
		}

		if !strings.Contains(fmt.Sprint(res), fmt.Sprint(exp)) {
			return fmt.Errorf("unexpected value of field %q, \ngot: %v \nwant at least: %v", field, res, exp)
		}
	}
	return nil
}
