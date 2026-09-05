// Package e2e is the main package of the go-e2e library. It contains all types needed to
// declare tests. The engine that runs them is used by the e2e tool and does not need to
// be interacted with directly.
package e2e

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Runner is the engine that executes tests. It is used internally by the e2e tool
// and does not need to be instantiated directly.
type Runner struct {
	BeforeRun func() any // Called before any tests run. Corresponds to BeforeRun in the project root.
	AfterRun  func(any)  // Called after all tests have run. Corresponds to AfterRun in the project root.
	Verbose   bool       // When true, all response headers are printed, not only expected ones.
}

type result struct {
	buf    *bytes.Buffer
	passed bool
}

// Runnable is satisfied by Sequence and Test, letting Runner.Run declare a container type
// for either. Its method is unexported, so nothing outside this package can implement it.
type Runnable interface {
	run(name string, verbose bool, client *http.Client, buf *bytes.Buffer, data map[string]string) result
}

// Run executes the given tests, prints the output, and prompts for confirmation before
// showing full results. Called by the e2e tool, which passes each Sequence or Test keyed by
// the name of the exported variable it was declared under (or, if that name collided with
// another package's, by that name prefixed with the package name).
func (r Runner) Run(tests map[string]Runnable) {
	r.ensureHooks()
	before := r.BeforeRun()
	defer r.AfterRun(before)

	ch := make(chan result)
	wg := sync.WaitGroup{}
	client := &http.Client{
		Timeout: 10 * time.Second,
		// Don't follow redirects
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	numPassed := 0
	results := []result{}

	drawProgressBar(results, len(tests))
	for name, t := range tests {
		wg.Add(1)
		go func(name string, t Runnable) {
			defer wg.Done()
			buf := &bytes.Buffer{}
			fmt.Fprintln(buf, yellow(center(strings.ToUpper(name), 31)))
			ch <- t.run(name, r.Verbose, client, buf, make(map[string]string))
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
		results = append(results, result)
		drawProgressBar(results, len(tests))
	}

	allPassed := numPassed == len(tests)
	numFailed := len(tests) - numPassed

	fmt.Printf(`
---------------------------------
TOTAL RESULT: %s
Num tests run: %5d
Failed tests: %6d

`, resultText(allPassed), len(tests), numFailed)

	input := confirm(`Do you want to see full output (vs only failed)? [y/N]: `)
	full := strings.ToLower(strings.Trim(input, "\n")) == "y"

	for _, result := range results {
		switch full {
		case true:
			fmt.Print(result.buf.String())
		case false:
			if !result.passed {
				fmt.Print(result.buf.String())
			}
		}
	}
}

func (r *Runner) ensureHooks() {
	if r.BeforeRun == nil {
		r.BeforeRun = func() any { return nil }
	}
	if r.AfterRun == nil {
		r.AfterRun = func(any) {}
	}
}
