package e2e

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Matches strings such as "$user" and "$var.field.subfield".
var refMatcher *regexp.Regexp = regexp.MustCompile(`\$\w+(?:\.\w+)*`)

// Test defines a single HTTP call and the expectations against its response. Declare it
// directly as a standalone test, or use it as a step within a Sequence. Fields not set
// are ignored, but at least setting Method and URL for Request is required to make a
// proper request.
type Test struct {
	// Before is an optional pre-test action. Use the helper functions [Command], [Input],
	// and [Delay] to create it.
	Before Action
	// Request defines the HTTP call this test makes.
	Request Request
	// Expect defines expectations on the HTTP response. Only the fields you set are validated,
	// unset fields accept any value.
	Expect Expect
	// Capture lists Captors naming response body fields or headers whose values should be stored
	// and made available to later steps via the $-prefix. Has no effect on a standalone Test,
	// since there is no later step to receive it.
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
		// Arrays and repeated tags don't add an index to the path, every element is flattened
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
	Captors []Captor
)

type (
	// Headers is a map of header names to values for use in HTTP requests and response assertions.
	Headers map[string]string
	// Body is a map of dot-separated field paths to expected values. See [Expect.Body].
	Body map[string]any
	// A Captor captures a value from an HTTP response for later reference via the $-prefix.
	Captor struct {
		// Name is the response body field or header to capture, using the same dot-separated
		// path syntax as Expect.Body for body fields.
		Name string
		// As is the key the captured value is stored under. If empty, Name is used instead.
		// Reference the stored value elsewhere using the $-prefix, e.g. $Name or $As.
		As string
		// Regex, if set, cherry-picks part of the captured value instead of capturing it in
		// full: the pattern's first capturing group is captured if it has one, otherwise the
		// whole match is. Use non-capturing groups, (?:...), for whichever part of the pattern
		// isn't what should be captured.
		Regex string
	}
)

// validate reports a structural problem that could never be intentional, e.g. a Request with
// no URL. run calls it first, before doing anything else, so a mistake like that fails
// immediately instead of surfacing as a confusing failure partway through a request that was
// never going to work. This is different from the warnings inject/capture print at runtime for
// things that could plausibly be fine, such as a $-reference with nothing captured for it yet.
func (t Test) validate() error {
	var errs []error
	if t.Request.Method == "" {
		errs = append(errs, errors.New("Request.Method is not set"))
	}
	if t.Request.URL == "" {
		errs = append(errs, errors.New("Request.URL is not set"))
	}
	for _, c := range t.Capture {
		if c.Name == "" {
			errs = append(errs, errors.New("a Captor has no Name set"))
		}
	}
	return errors.Join(errs...)
}

// run makes Test satisfy Runnable, letting it be declared standalone alongside Sequence, or
// used as a step within one. Everything it needs is passed in rather than created
// internally: a standalone Test gets its client/log/data straight from Runner.Run, while a
// step gets them from the Sequence it belongs to.
func (t Test) run(verbose bool, client *http.Client, log *log, data map[string]string) result {
	if err := t.validate(); err != nil {
		log.error(err)
		return result{log: log, passed: false}
	}

	if t.Before != nil {
		description, err := t.Before(data, log)
		log.print(fmt.Sprintf("Pre-test action: %v", description))
		if err != nil {
			log.print() // newline
			log.error(fmt.Errorf("performing pre test action: %w", err))
			return result{log: log, passed: false}
		}
	}

	t.Request = injectRequest(t.Request, data, log)
	t.Expect = injectExpect(t.Expect, data, log)
	body, headers, passed := performTest(client, log, t.Request, t.Expect, verbose)
	if passed {
		capture(body, headers, data, t.Capture, log)
	}

	return result{log: log, passed: passed}
}

func injectRequest(req Request, data map[string]string, log *log) Request {
	req.URL = refMatcher.ReplaceAllStringFunc(req.URL, func(m string) string {
		key := strings.TrimPrefix(m, "$")
		val, ok := data[key]
		if !ok {
			log.warning("%q in URL has no captured value, leaving reference as is", m)
			return m
		}
		return val
	})
	for k, v := range req.Headers {
		req.Headers[k] = refMatcher.ReplaceAllStringFunc(v, func(m string) string {
			key := strings.TrimPrefix(m, "$")
			val, ok := data[key]
			if !ok {
				log.warning("%q in header %q has no captured value, leaving reference as is", m, k)
				return m
			}
			return val
		})
	}
	req.Body = refMatcher.ReplaceAllStringFunc(req.Body, func(m string) string {
		key := strings.TrimPrefix(m, "$")
		val, ok := data[key]
		if !ok {
			log.warning("%q in body has no captured value, leaving reference as is", m)
			return m
		}
		return val
	})

	// TODO: should the t.Before injection be managed together with this and injectExpect?

	return req
}

func injectExpect(exp Expect, data map[string]string, log *log) Expect {
	for k, v := range exp.Headers {
		exp.Headers[k] = refMatcher.ReplaceAllStringFunc(v, func(m string) string {
			key := strings.TrimPrefix(m, "$")
			val, ok := data[key]
			if !ok {
				log.warning("%q in expected header %q has no captured value, leaving reference as is", m, k)
				return m
			}
			return val
		})
	}
	for k, v := range exp.Body {
		s, isString := v.(string)
		if !isString {
			continue
		}
		exp.Body[k] = refMatcher.ReplaceAllStringFunc(s, func(m string) string {
			key := strings.TrimPrefix(m, "$")
			val, ok := data[key]
			if !ok {
				log.warning("%q in expected body field %q has no captured value, leaving reference as is", m, k)
				return m
			}
			return val
		})
	}

	return exp
}

func capture(body map[string][]string, headers http.Header, data map[string]string, captors Captors, log *log) {
	//TODO: Warn (Error?) if captor contains anything else but the allowed set of alphabetical characters and dots

	for _, c := range captors {
		// Only search headers if no match found in body.
		val, foundMatch := body[c.Name]
		if !foundMatch {
			//TODO: Add warning for when both header and body matches and header is skipped??
			val, foundMatch = headers[http.CanonicalHeaderKey(c.Name)]
		}

		if foundMatch {
			if len(val) > 1 {
				log.warning("capturing %q: matched multiple values. Captures first one.", c.Name)
			}

			value := fmt.Sprint(val[0])

			if c.Regex != "" {
				re, err := regexp.Compile(c.Regex)
				if err != nil {
					log.warning("capturing %q: invalid Regex %q: %v", c.Name, c.Regex, err)
					continue
				}
				match := re.FindStringSubmatch(value)
				if match == nil {
					log.warning("capturing %q: regex %q matched nothing in the captured value.", c.Name, c.Regex)
					continue
				}
				// match[1] is the first capturing group, if the pattern has one; otherwise
				// match[0], the whole match, is all there is.
				if len(match) > 1 {
					value = match[1]
				} else {
					value = match[0]
				}
			}

			// c.As is used as key if set, otherwise c.Name is the key.
			key := c.Name
			if c.As != "" {
				key = c.As
			}
			data[key] = value
		} else {
			log.warning("capturing %q: matched nothing.", c.Name)
		}
	}
}
