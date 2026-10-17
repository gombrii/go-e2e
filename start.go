// Package e2e is the main package of the go-e2e library. It contains all types needed to
// declare tests. The engine that runs them is used by the e2e tool and does not need to
// be interacted with directly.
package e2e

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Config is global configuration for the project that is automatically used by the e2e tool
// if declared in an exported variable in the project root.
type Config struct {
	BeforeRun func() any  // Called before any tests run.
	AfterRun  func(any)   // Called after all tests have run.
	TLS       *tls.Config // Used when initiating the http.Client that will run the tests.
}

// Runner is the engine that executes tests. It is used internally by the e2e tool
// and does not need to be instantiated directly.
type Runner struct {
	Config  Config
	Verbose bool // When true, all response headers are printed, not only expected ones.
}

type result struct {
	log    *log
	passed bool
}

// Runnable is satisfied by Sequence and Test, letting Runner.Run declare a container type
// for either. Its method is unexported, so nothing outside this package can implement it.
type Runnable interface {
	run(verbose bool, client *http.Client, log *log, cache cache) result
}

// Run executes the given tests, prints the output, and prompts for confirmation before
// showing full results. Called by the e2e tool, which passes each Sequence or Test keyed by
// the name of the exported variable it was declared under (or, if that name collided with
// another package's, by that name prefixed with the package name).
func (r Runner) Run(tests map[string]Runnable) {
	r.ensureHooks()
	before := r.Config.BeforeRun()
	defer r.Config.AfterRun(before)

	ch := make(chan result)
	wg := sync.WaitGroup{}
	numPassed := 0
	results := []result{}

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = r.Config.TLS
	client := &http.Client{
		Timeout: 10 * time.Second,
		// Don't follow redirects
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: tr,
	}

	drawProgressBar(results, len(tests))
	for name, t := range tests {
		wg.Add(1)
		go func(name string, t Runnable) {
			defer wg.Done()
			log := &log{}
			log.banner(name)
			ch <- t.run(r.Verbose, client, log, cache{})
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
			fmt.Print(result.log.String())
		case false:
			if !result.passed {
				fmt.Print(result.log.String())
			}
		}
	}
}

func (r *Runner) ensureHooks() {
	if r.Config.BeforeRun == nil {
		r.Config.BeforeRun = func() any { return nil }
	}
	if r.Config.AfterRun == nil {
		r.Config.AfterRun = func(any) {}
	}
}
