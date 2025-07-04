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
	Before  Before
	Request Request
	Expect  Expect
	Capture Captors
}

type (
	Before  []func(data map[string]string) (string, error)
	Request struct {
		CTX     context.Context
		Method  string
		URL     string
		Headers Headers
		Content string
		Body    string
	}
	Expect struct {
		Status  int
		Headers Headers
		Body    Body
	}
	Captors []string
)

type (
	Headers []header
	header  struct {
		Key string
		Val string
	}
	Body map[string]any
)

func (t test) run(client *http.Client, buf *bytes.Buffer, data map[string]string) (result testResult) {
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

func capture(body map[string]any, data map[string]string, captors Captors) {
	for _, c := range captors {
		if val, ok := body[c]; ok {
			data[c] = fmt.Sprint(val) ////TODO: Only loops through surface level fields.
		}
	}
}
