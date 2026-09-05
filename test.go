package e2e

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

type test struct {
	// Before lists the before-actions to run before this test, in order. Use the helper functions
	// [Command], [Input], and [Delay] to create them.
	Before Before
	// Request defines the HTTP call this test makes.
	Request Request
	// Expect defines expectations on the HTTP response. Only the fields you set are validated —
	// unset fields accept any value.
	Expect Expect
	// Capture lists keys of response body fields whose values should be stored and made available
	// to later steps via the $-prefix.
	Capture Captors
}

type (
	// Before is a slice of before-action functions. Use [Command], [Input], or [Delay] to create them.
	Before []func(data map[string]string) (string, error)
	// Request defines the HTTP call to make for this test.
	Request struct {
		// The HTTP method to use, e.g. "GET" or "POST".
		Method string
		// The URL to send the request to. Can be a hard-coded string or a value looked up
		// dynamically using [addr.Lookup] or [addr.EnvLookup].
		URL string
		// Additional headers to send with the request.
		Headers Headers
		// Shorthand for the Content-Type header. Setting this is equivalent to adding a
		// Content-Type entry to [Request.Headers].
		Content string
		// The request body as a string. Raw string literals are recommended for readability.
		Body string
	}
	// Expect defines expectations on the HTTP response. Leave a field unset to accept any value
	// for it.
	Expect struct {
		// The HTTP status code the response must return. If set, any other status code will
		// fail the test. Leave unset to accept any status.
		Status int
		// Headers to assert in the response. Each entry is matched as follows:
		// - the key must exactly match a header name in the response.
		// - the value must be contained within the matched header's value (not an exact match).
		//
		// The partial-value rule means you don't need to spell out the full header value, which
		// is handy when values contain generated codes or other dynamic parts. Setting the value
		// to "" asserts only that the header is present, regardless of its value.
		Headers Headers
		// Body specifies expectations on the response body using dot-separated paths to fields.
		// Both JSON and XML responses are supported.
		//
		// For JSON, use dot notation to reach nested fields:
		//
		//	// Matches
		//	// {
		//	//   "field": {
		//	//     "leaf": "value"
		//	//   }
		//	// }
		//	Body{"field.leaf": "value"}
		//
		// For XML, use dot notation to reach nested tags. Append @attr to assert an attribute:
		//
		//	// Matches
		//	// <root>
		//	//   <item attr="attrval">value</item>
		//	//   <item>othervalue</item>
		//	// </root>
		//	Body{
		//		"root.item":      "othervalue",
		//		"root.item@attr": "attrval",
		//	}
		//
		// In both formats the expected value only needs to be contained within the actual value,
		// not match it exactly. Setting the expected value to "" asserts only that the field
		// exists.
		Body Body
	}
	Captors []string
)

type (
	// Headers is a map of header names to values for use in HTTP requests and response assertions.
	Headers map[string]string
	// Body is a map of dot-separated field paths to expected values. See [Expect.Body].
	Body map[string]any
)

func (t test) run(client *http.Client, buf *bytes.Buffer, data map[string]string, verbose bool) (result testResult) {
	if t.Request.Content != "" {
		if t.Request.Headers == nil {
			t.Request.Headers = make(Headers)
		}
		t.Request.Headers["Content-Type"] = t.Request.Content
	}

	for _, action := range t.Before {
		description, err := action(data)
		fmt.Fprintf(buf, "Before test: %v\n", description)
		if err != nil {
			fmt.Fprintf(buf, "\n%s: performing pre test action: %v\n", pink("ERROR"), err)
			return testResult{
				buf:    buf,
				passed: false,
			}
		}
	}

	t.Request = inject(t.Request, data)

	body, result := performTest(client, buf, t.Request, t.Expect, verbose)
	if !result.passed {
		return result
	}

	capture(body, data, t.Capture)

	return result
}

func inject(req Request, data map[string]string) Request {
	if len(data) == 0 {
		return req
	}

	req.URL = variable.ReplaceAllStringFunc(req.URL, func(s string) string {
		s = strings.TrimPrefix(s, "$")
		return data[s]
	})
	for k, v := range req.Headers {
		req.Headers[k] = variable.ReplaceAllStringFunc(v, func(s string) string {
			s = strings.TrimPrefix(s, "$")
			return data[s]
		})
	}
	req.Body = variable.ReplaceAllStringFunc(req.Body, func(s string) string {
		s = strings.TrimPrefix(s, "$")
		return data[s]
	})

	return req
}

func capture(body map[string][]string, data map[string]string, captors Captors) {
	for _, c := range captors {
		if val, ok := body[c]; ok {
			data[c] = fmt.Sprint(val[0]) ////TODO: Only loops through surface level fields.
		}
	}
}
