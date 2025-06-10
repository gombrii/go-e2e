package e2e

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type (
	Suite struct {
		Name  string
		Tests Tests
	}
	Tests map[string]test
	test  struct {
		Request  Request
		Expect Expect
	}
)

func (s Suite) run(client *http.Client) result {
	buf := &bytes.Buffer{}
	ch := make(chan testResult)
	wg := sync.WaitGroup{}
	numPassed := 0

	fmt.Fprintln(buf, yellow("\n---------------------------------"))
	fmt.Fprintln(buf, yellow(" TEST SUITE - ", strings.ToUpper(s.Name)))
	fmt.Fprintln(buf, yellow("---------------------------------"))

	for name, t := range s.Tests {
		wg.Add(1)
		go func(name string, test test) {
			defer wg.Done()
			buf := &bytes.Buffer{}
			fmt.Fprintln(buf, "--------", name, "--------")
			_, result := test.run(client, buf)
			if result.passed {
				fmt.Fprintln(buf, "\nSuccess!")
			}
			ch <- result
		}(name, t)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for result := range ch {
		if result.passed {
			numPassed++
		}
		buf.Write(result.buf.Bytes())
	}

	allPassed := numPassed == len(s.Tests)
	numFailed := len(s.Tests) - numPassed

	fmt.Fprintf(buf, `---------------------------------
SUITE RESULT: %s
Success: %d
Fail: %d
`, resultText(allPassed), numPassed, numFailed)
	return result{buf, allPassed, len(s.Tests)}
}

func (t test) run(client *http.Client, buf *bytes.Buffer) (parsedBody map[string]any, result testResult) {
	return run(client, buf, t.Request, t.Expect)
}
