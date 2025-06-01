package e2e

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type Set interface {
	runWithClient(*http.Client) result
}

type result struct {
	buf    *bytes.Buffer
	passed bool
	numRun int
}

func RunConcurrent(hooks *Hooks, sets ...Set) {
	ensureHooks(hooks)
	envs := hooks.Before()
	defer hooks.After(envs)

	ch := make(chan result)
	wg := sync.WaitGroup{}
	client := &http.Client{}
	numRun := 0
	numPassed := 0
	results := []result{}

	for _, set := range sets {
		wg.Add(1)
		go func(set Set) {
			defer wg.Done()
			ch <- set.runWithClient(client)
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

	input := confirm(`Press enter to see failed test logs, or type "f" and press enter to see full test logs: `)
	full := strings.Trim(input, "\n") == "f"

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
