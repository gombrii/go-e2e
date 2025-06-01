package e2e

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

type (
	Sequence struct {
		Name  string
		Steps []Step
	}
	Step struct {
		Inputs  []InputFunc
		Setup   Setup
		Expect  Expect
		Capture Captors
	}
	InputFunc func(data map[string]string) error
	Captors   []string
)

func (s Sequence) run(client *http.Client) result {
	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)
	allPassed := true
	data := make(map[string]string)

	logger.Println(yellow("\n---------------------------------"))
	logger.Println(yellow("TEST SEQUENCE - ", strings.ToUpper(s.Name)))
	logger.Println(yellow("---------------------------------"))

	numRun := 0
	for i, step := range s.Steps {
		logger.Println("Step", i+1)
		numRun = i + 1
		if passed := step.run(client, buf, data); !passed {
			allPassed = false
			break
		}
	}
	logger.Printf("---------------------------------\nSEQUENCE RESULT: %s\n", resultText(allPassed))
	return result{buf, allPassed, numRun}
}

func (s Step) run(client *http.Client, buf *bytes.Buffer, data map[string]string) (passed bool) {
	logger := log.New(buf, "", 0)
	for _, fun := range s.Inputs {
		err := fun(data)
		if err != nil {
			logger.Printf("%s: asking for user input: %v\n", pink("ERROR"), err)
			return false
		}
	}

	s.Setup = inject(s.Setup, data)

	body, result := run(client, buf, s.Setup, s.Expect)
	if !result.passed {
		return false
	}

	capture(body, data, s.Capture)

	return true
}

func Input(text string, mapTo string) InputFunc {
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

		data[mapTo] = strings.TrimSpace(input)

		return nil
	}
}

func inject(setup Setup, data map[string]string) Setup {
	if len(data) == 0 {
		return setup
	}

	setup.URL = variable.ReplaceAllStringFunc(setup.URL, func(s string) string {
		s = strings.TrimPrefix(s, "$")
		return data[s]
	})
	for i, h := range setup.Headers {
		h.Val = variable.ReplaceAllStringFunc(h.Val, func(s string) string {
			s = strings.TrimPrefix(s, "$")
			return data[s]
		})
		setup.Headers[i] = h
	}
	setup.Body = variable.ReplaceAllStringFunc(setup.Body, func(s string) string {
		s = strings.TrimPrefix(s, "$")
		return data[s]
	})

	return setup
}

func capture(body map[string]any, data map[string]string, captors Captors) {
	for _, c := range captors {
		if val, ok := body[c]; ok {
			data[c] = fmt.Sprint(val) ////TODO: Only loops through surface level fields.
		}
	}
}
