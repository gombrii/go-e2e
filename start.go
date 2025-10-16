// Package e2e is the main package of the go-e2e library. It contains all types necessary to
// construct tests as well as the engine running the tests.
package e2e

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// The runner is the core component that run tests. It is mostly called by the [e2r] application
// but can also be instantiated and run programmatically by a third party if needed.
type Runner struct {
	BeforeRun func() any // Sets up environment before running any tests.
	AfterRun  func(any)  // Tears down environment after running all tests.
}

type set interface {
	run(*http.Client) result
}

type result struct {
	buf    *bytes.Buffer
	passed bool
	numRun int
}

// Run starts the engine, runs suites and sequences concurrently or sequentially depending on their
// type. It handles the whole run from start to finish including printing output.
func (r Runner) Run(sets ...set) {
	r.ensureHooks()
	before := r.BeforeRun()
	defer r.AfterRun(before)

	ch := make(chan result)
	wg := sync.WaitGroup{}
	client := &http.Client{
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
			ch <- set.run(client)
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
