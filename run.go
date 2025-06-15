package e2e

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type Runner struct {
	BeforeRun func() any
	AfterRun  func(any)
}

type set interface {
	run(*http.Client) result
}

type result struct {
	buf    *bytes.Buffer
	passed bool
	numRun int
}

func (r Runner) Run(sets ...set) {
	r.ensureHooks()
	before := r.BeforeRun()
	defer r.AfterRun(before)

	ch := make(chan result)
	wg := sync.WaitGroup{}
	client := &http.Client{}
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
