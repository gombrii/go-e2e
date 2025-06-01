package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type Setup struct {
	CTX     context.Context
	Method  string
	URL     string
	Headers []Header
	Content string
	Body    string
}

type Expect struct {
	Status  int
	Headers []Header
	Fields  Fields
}

type Header struct {
	Key string
	Val string
}
type Fields map[string]any

type testResult struct {
	buf    *bytes.Buffer
	passed bool
}

func run(client *http.Client, buf *bytes.Buffer, setup Setup, expected Expect) (parsedBody map[string]any, res testResult) {
	logger := log.New(buf, "", 0)

	resp, err := makeRequest(client, setup)
	if err != nil {
		logger.Printf("%s: making request: %v\n", pink("ERROR"), err)
		return map[string]any{}, testResult{buf, false}
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Printf("%s: reading response body: %v\n", pink("ERROR"), err)
		return map[string]any{}, testResult{buf, false}
	}

	parsedBody = make(map[string]any)
	json.Unmarshal(body, &parsedBody)

	logger.Println("->", setup.URL)
	logger.Println("<-", resp.StatusCode)
	logger.Println(ifTrue("<- "+format(body), len(body) > 0))

	if err := assertStatus(expected.Status, resp.StatusCode); err != nil {
		logger.Printf("%s: asserting status: %v\n", pink("FAIL"), err)
		return map[string]any{}, testResult{buf, false}
	}
	if err := assertHeaders(expected.Headers, resp.Header); err != nil {
		logger.Printf("%s: asserting header: %v\n", pink("FAIL"), err)
		return map[string]any{}, testResult{buf, false}
	}
	if err := assertBody(expected.Fields, parsedBody); err != nil {
		logger.Printf("%s: asserting body: %v\n", pink("FAIL"), err)
		return map[string]any{}, testResult{buf, false}
	}

	return parsedBody, testResult{buf, true}
}

func makeRequest(client *http.Client, setup Setup) (*http.Response, error) {
	if setup.CTX == nil {
		setup.CTX = context.Background()
	}

	req, err := http.NewRequestWithContext(setup.CTX, setup.Method, setup.URL, io.NopCloser(strings.NewReader(setup.Body)))
	if err != nil {
		return nil, fmt.Errorf("setting up: %v", err)
	}

	for _, h := range setup.Headers {
		req.Header.Add(h.Key, h.Val)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing: %v", err)
	}

	return resp, nil
}

func assertStatus(expected int, actual int) error {
	if expected != actual {
		return fmt.Errorf("unexpected code, got: %d want: %d", actual, expected)
	}
	return nil
}

func assertHeaders(expected []Header, actual http.Header) error {
	for _, h := range expected {
		res, ok := actual[h.Key]
		if !ok {
			return fmt.Errorf("missing %q", res)
		}

		hasValue := false
		for _, v := range res {
			if strings.Contains(fmt.Sprint(v), fmt.Sprint(h.Val)) {
				hasValue = true
			}
		}
		if !hasValue {
			return fmt.Errorf("missing value. Want at least:%q", h.Val)
		}
	}
	return nil
}

func assertBody(expected Fields, actual map[string]any) error {
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
