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

	"github.com/gombrii/go-e2e/addr"
)

// Config is project wide configuration. The e2e tool picks it up automatically when it's
// declared in an exported variable in the project root, meaning the package in the directory
// the tool is run from. Only one Config may be declared there. All fields are optional, as is
// Config in its entirety.
type Config struct {
	// AddressBook enables addr.Lookup and addr.EnvLookup. Lookups work in all packages except the
	// root.
	AddressBook addr.AddressBook
	// BeforeRun is called before any tests run. Whatever it returns is passed to AfterRun.
	BeforeRun func() any
	// AfterRun is called after all tests have run, receiving what BeforeRun returned, or nil
	// if BeforeRun isn't set.
	AfterRun func(any)
	// TLS is used by the http.Client that runs all tests, e.g. to present a client
	// certificate to servers requiring mTLS, or to trust a private CA. If nil, Go's default
	// TLS behavior is used.
	TLS *tls.Config
}

// Runner is the engine that executes tests. It is used internally by the e2e tool
// and does not need to be instantiated directly.
type Runner struct {
	Config  Config // The project's Config, if one is declared in the project root.
	Verbose bool   // When true, all response headers are printed, not only expected ones.
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
