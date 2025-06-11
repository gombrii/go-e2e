package e2e

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var variable *regexp.Regexp = regexp.MustCompile(`\$\w+`)

type (
	Sequence struct {
		Name  string
		Steps Steps
	}
	Steps []test
)

func (s Sequence) run(client *http.Client) result {
	buf := &bytes.Buffer{}
	allPassed := true
	data := make(map[string]string)

	fmt.Fprintln(buf, yellow("\n---------------------------------"))
	fmt.Fprintln(buf, yellow(" TEST SEQUENCE - ", strings.ToUpper(s.Name)))
	fmt.Fprintln(buf, yellow("---------------------------------"))

	numRun := 0
	for i, step := range s.Steps {
		fmt.Fprintln(buf, "Step", i+1)
		numRun = i + 1
		if result := step.run(client, buf, data); !result.passed {
			allPassed = false
			break
		}
		fmt.Fprintln(buf)
	}
	fmt.Fprintf(buf, "---------------------------------\nSEQUENCE RESULT: %s\n", resultText(allPassed))
	return result{buf, allPassed, numRun}
}

func Input(text string, mapTo string) action {
	return func(data map[string]string) error {
		progressBarMutex.Lock()
		defer progressBarMutex.Unlock()
		reader := bufio.NewReader(os.Stdin)

		moveDown(1) // To one line below progress bar
		clearLine() // Clear line where prompt will be drawn

		fmt.Print("\rInput required - ", text, ": ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %v", err)
		}

		moveUp(1)   // Back to the line where the prompt was drawn
		clearLine() // Clear line where prompt was drawn
		moveUp(1)   // To line where progress bar is drawn

		if mapTo != "" {
			data[mapTo] = strings.TrimSpace(input)
		}

		return nil
	}
}

func Command(command string, args ...string) action { // Can add mapTo as first argument to be able to capture output
	return func(data map[string]string) error {
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
			return fmt.Errorf("executing command: %v", err)
		}

		qr := string(out)
		numLines := strings.Count(qr, "\n")
		if !strings.HasSuffix(qr, "\n") && len(qr) > 0 {
			numLines++
		}

		fmt.Print("\r", qr, "Continue with Enter")
		reader.ReadString('\n')

		for range numLines + 1 { // Remove all lines printed by the executed command
			moveUp(1)
			clearLine()
		}

		moveUp(1) // To line where progress bar is drawn

		return nil
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
