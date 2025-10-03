package e2e

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type test struct {
	// Before contains a function that will be fun before this test. There are two helper functions
	// that can be used to create Before functions, [Command] and [Input].
	Before Before
	// Request contains all details necessary to perform the test's HTTP call.
	Request Request
	// Expect contains information on the expected shape of the HTTP response.
	Expect Expect
	// Capture contains strings matching fields in the HTTP response of which you'd like to capture
	// the value.
	Capture Captors
}

type (
	// Before is a function type that can used to perform a pre test action.
	Before []func(data map[string]string) (string, error)
	// Request contains all details necessary to perform a test's HTTP call.
	Request struct {
		// CTX is the context provided to the http.Client upon making the test's HTTP call.
		// It defaults to context.Background().
		CTX context.Context
		// The HTTP method of the request.
		Method string
		// The URL to which to make the HTTP call. It can either be hard coded as a string or looked
		// up dynamically using the [addr.AddressBook].
		URL string
		// Headers contains a slice of key value pairs. Duplicate keys will be added together to
		// multi value headers in runtime.
		Headers Headers
		// Content is a special field for the "Content-Type" header for easy access.
		Content string
		// The body in string format. It is recommended to use raw strings.
		Body string
	}
	// Expect contains information on the expected shape of the HTTP response. If a field is left
	// unset it means the test will accept any value as a successful response.
	Expect struct {
		// Status is set if a specific response status is expected as a result of the test. If set
		// then the resulting status of the test must exactly match what is expected or the test
		// will count as a failure.
		Status int
		// Headers contains a slice of key value pairs. The key and the value is treated differently
		// in terms of strictness. A test with an expected header set will only succeed if the
		// following two conditions are met.
		// - the key exactly matches a key present in the HTTP response of the test.
		// - the value is contained within the string value of the header being matched with its
		// key in the previous condition.
		//
		// "Contained within" in the second condition  means that the expected value does not
		// need to state the entirity of the actual value in the HTTP response. This is useful when
		// values in response headers contains generated codes, etc. This also means that setting
		// the expected value to "" means that any value is accepted, only asserting the presense
		// of the key.
		Headers Headers
		// Body is a map representing expectations on response bodies.
		// The keys match fields or paths to leafs in nested response bodies.
		// Body supports both JSON and XML.
		//
		//	{
		// 		"field": {
		//			"leaf": "value"
		//		}
		// 	}
		//
		// This JSON example matches the following body object.
		//
		//	Body{
		//		"field.leaf": "value",
		//	}
		//
		// This asserts that "leaf" contains the string "value". If the value of "leaf" was a longer
		// string, eg. "everybody has values", it would still match.
		//
		//	<root>
		// 		<item attr="attrval">value</item>
		// 		<item>othervalue</item>
		// 	</root>
		//
		// For this XML example the following Body object asserts that at least one <item> tag under
		// the tag <root> contains the text "othervalue". It also asserts that at least one <item>
		// tag under the tag <root> has en attribute "attr" with the value of "attrval".
		//
		//	Body{
		//		"root.item":      "othervalue",
		//		"root.item@attr": "attrval",
		//	}
		Body Body
	}
	Captors []string
)

type (
	// Headers contains a slice of key value pairs representing headers of an HTTP request or
	// response. Duplicate keys are allowed.
	Headers []header
	header  struct {
		Key string
		Val string
	}
	// Body is a map representing expectations on response bodies.
	Body map[string]any
)

func (t test) run(client *http.Client, buf *bytes.Buffer, data map[string]string) (result testResult) {
	if t.Request.Content != "" {
		t.Request.Headers = append(t.Request.Headers, header{"Content-Type", t.Request.Content})
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

	body, result := performTest(client, buf, t.Request, t.Expect)
	if !result.passed {
		return result
	}

	capture(body, data, t.Capture)

	return result
}

func Input(text string, mapTo string) func(data map[string]string) (string, error) {
	return func(data map[string]string) (string, error) {
		progressBarMutex.Lock()
		defer progressBarMutex.Unlock()
		reader := bufio.NewReader(os.Stdin)

		moveDown(1) // To one line below progress bar
		clearLine() // Clear line where prompt will be drawn

		fmt.Print("\rInput required - ", text, ": ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Sprintf("manual input %q", text), fmt.Errorf("reading input: %v", err)
		}

		moveUp(1)   // Back to the line where the prompt was drawn
		clearLine() // Clear line where prompt was drawn
		moveUp(1)   // To line where progress bar is drawn

		if mapTo != "" {
			data[mapTo] = strings.TrimSpace(input)
		}

		return fmt.Sprintf("manual input: %q", text), nil
	}
}

func Command(command string, args ...string) func(data map[string]string) (string, error) { // Can add mapTo as first argument to be able to capture output
	return func(data map[string]string) (string, error) {
		progressBarMutex.Lock()
		defer progressBarMutex.Unlock()
		reader := bufio.NewReader(os.Stdin)

		moveDown(1) // To one line below progress bar
		clearLine() // Clear line where prompt will be drawn

		for i, s := range args {
			args[i] = variable.ReplaceAllStringFunc(s, func(str string) string {
				str = strings.TrimPrefix(str, "$")
				return data[str]
			})
		}

		cmd := exec.Command(command, args...)
		out, err := cmd.Output()
		if err != nil {
			return fmt.Sprintf("command run %q", command), fmt.Errorf("executing command: %v", err)
		}

		outStr := strings.TrimSuffix(string(out), "\n")
		numLines := strings.Count(outStr, "\n")

		if len(strings.TrimSpace(outStr)) > 0 {
			numLines++
			fmt.Print("\r", outStr, "\nContinue with Enter")
		} else {
			fmt.Print("\rContinue with Enter")
		}
		reader.ReadString('\n')

		for range numLines + 1 { // Remove all lines printed by the executed command
			moveUp(1)
			clearLine()
		}

		moveUp(1) // To line where progress bar is drawn

		return fmt.Sprintf("command run: %q", command), nil
	}
}

func inject(req Request, data map[string]string) Request {
	if len(data) == 0 {
		return req
	}

	req.URL = variable.ReplaceAllStringFunc(req.URL, func(s string) string {
		s = strings.TrimPrefix(s, "$")
		return data[s]
	})
	for i, h := range req.Headers {
		h.Val = variable.ReplaceAllStringFunc(h.Val, func(s string) string {
			s = strings.TrimPrefix(s, "$")
			return data[s]
		})
		req.Headers[i] = h
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
