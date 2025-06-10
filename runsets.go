package e2e

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type Runner struct {
	Before func() any
	After  func(any)
}

type Set interface {
	run(*http.Client) result
}

type result struct {
	buf    *bytes.Buffer
	passed bool
	numRun int
}

func (r Runner) Run(sets ...Set) {
	r.ensureHooks()
	before := r.Before()
	defer r.After(before)

	ch := make(chan result)
	wg := sync.WaitGroup{}
	client := &http.Client{}
	numRun := 0
	numPassed := 0
	results := []result{}

	drawProgressBar(results, len(sets))
	for _, set := range sets {
		wg.Add(1)
		go func(set Set) {
			defer wg.Done()
			ch <- set.run(client)
		}(set)
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
MULTI-SET RESULT: %s
Total tests run: %d
Successful sets: %d
Failed sets: %d
`, resultText(allPassed), numRun, numPassed, numFailed)

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
	if r.Before == nil {
		r.Before = func() any { return nil }
	}
	if r.After == nil {
		r.After = func(any) {}
	}
}
