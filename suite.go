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
		// The name of the suite. Used for test logs.
		Name  string
		// The tests contained within this Suite.
		Tests Tests
	}
	// Tests is an unordered map. Each key is a test name and each value is a Test. The test names
	// are used for test logs.
	Tests map[string]test
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
			result := test.run(client, buf, map[string]string{})
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
