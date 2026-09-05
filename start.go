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

// Runner is the engine that executes test suites and sequences. It is used internally
// by the e2e tool and does not need to be instantiated directly.
type Runner struct {
	BeforeRun func() any // Called before any tests run. Corresponds to BeforeRun in the project root.
	AfterRun  func(any)  // Called after all tests have run. Corresponds to AfterRun in the project root.
	Verbose   bool       // When true, all response headers are printed, not only expected ones.
}

type set interface {
	run(*http.Client, bool) result
}

type result struct {
	buf    *bytes.Buffer
	passed bool
	numRun int
}

// Run executes the given suites and sequences, prints the output, and prompts for
// confirmation before showing full results. Called by the e2e tool.
func (r Runner) Run(sets ...set) {
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
	numRun := 0
	numPassed := 0
	results := []result{}

	drawProgressBar(results, len(sets))
	for _, s := range sets {
		wg.Add(1)
		go func(set set) {
			defer wg.Done()
			ch <- set.run(client, r.Verbose)
		}(s)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for result := range ch {
		if result.passed {
			numPassed++
		}
		numRun += result.numRun
		results = append(results, result)
		drawProgressBar(results, len(sets))
	}

	allPassed := numPassed == len(sets)
	numFailed := len(sets) - numPassed

	fmt.Printf(`
---------------------------------
TOTAL RESULT: %s
Num sets run: %5d (%d tests)
Failed sets: %6d
`, resultText(allPassed), len(sets), numRun, numFailed)

	input := confirm(`Do you want to see full test logs (vs only failed)? [y/N]: `)
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
