package e2e

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

// Test defines a single HTTP call and the expectations against its response. Declare it
// directly as a standalone test, or use it as a step within a Sequence.
type Test struct {
	// Before is an optional pre-test action. Use the helper functions [Command], [Input],
	// and [Delay] to create it.
	Before Action
	// Request defines the HTTP call this test makes.
	Request Request
	// Expect defines expectations on the HTTP response. Only the fields you set are validated —
	// unset fields accept any value.
	Expect Expect
	// Capture lists keys of response body fields whose values should be stored and made available
	// to later steps via the $-prefix. Has no effect on a standalone Test, since there is no
	// later step to receive it.
	Capture Captors
}

type (
	// Request defines the HTTP call to make for this test.
	Request struct {
		// The HTTP method to use, e.g. "GET" or "POST".
		Method string
		// The URL to send the request to. Can be a hard-coded string or a value looked up
		// dynamically using [addr.Lookup] or [addr.EnvLookup].
		URL string
		// Additional headers to send with the request.
		Headers Headers
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
		// Arrays and repeated tags don't add an index to the path — every element is flattened
		// onto the same path as its parent field. A path can therefore resolve to multiple
		// values, and the assertion passes if any one of them contains the expected value:
		//
		//	// Matches
		//	// {
		//	//   "items": [
		//	//     {"id": 1},
		//	//     {"id": 2}
		//	//   ]
		//	// }
		//	Body{"items.id": "2"} // Passes: one of the items has id 2
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

// run makes Test satisfy Runnable, letting it be declared standalone alongside Sequence, or
// used as a step within one. Everything it needs is passed in rather than created
// internally: a standalone Test gets its client/buf/data straight from Runner.Run, while a
// step gets them from the Sequence it belongs to.
//
// name is the key this Test was declared under when standalone, or the "Step N" label a
// Sequence assigns it when it's one of its steps. Either way it's printed as this run's
// banner.
func (t Test) run(name string, verbose bool, client *http.Client, buf *bytes.Buffer, data map[string]string) result {
	//fmt.Fprintln(buf, center(name, 16))
	//fmt.Fprintln(buf, name)

	if t.Before != nil {
		description, err := t.Before(data)
		fmt.Fprintf(buf, "Pre-test action: %v\n", description)
		if err != nil {
			fmt.Fprintf(buf, "\n%s: performing pre test action: %v\n", pink("ERROR"), err)
			return result{buf: buf, passed: false}
		}
	}

	t.Request = inject(t.Request, data)
	body, passed := performTest(client, buf, t.Request, t.Expect, verbose)
	capture(body, data, t.Capture, buf)

	return result{buf: buf, passed: passed}
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

func capture(body map[string][]string, data map[string]string, captors Captors, buf *bytes.Buffer) {
	for _, c := range captors {
		if val, ok := body[c]; ok {
			if len(val) > 1 {
				fmt.Fprintf(buf, "%s: capturing field %q: %v\n", yellow("WARNING"), c, "response field contains multiple values. Captures first one.")
			}
			data[c] = fmt.Sprint(val[0])
		}
	}
}
