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
		// A name identifying this suite in test output.
		Name  string
		// The tests to run. Each key is the test name shown in output.
		Tests Tests
	}
	// Tests is an unordered map of named tests. Tests in a Suite run concurrently, so they
	// must be independent of each other.
	Tests map[string]test
)

func (s Suite) run(client *http.Client, verbose bool) result {
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
			result := test.run(client, buf, map[string]string{}, verbose)
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
