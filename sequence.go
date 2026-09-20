package e2e

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
)

// Sequence is a list of Test steps that run one after another, sharing a single context so
// that data captured in one step (see Test.Capture) can be referenced by a later one using
// the $-prefix. Sequence satisfies [Runnable], same as a standalone Test, so it's declared
// directly as its own top-level exported variable.
type Sequence []Test

// validate checks every one of Sequence's steps the same way a standalone Test checks itself,
// combining every problem found across all of them into one error. run calls it first, before
// executing any step, so a broken step fails the whole Sequence immediately rather than
// partway through.
func (s Sequence) validate() error {
	var errs []error
	for i, step := range s {
		if err := step.validate(); err != nil {
			errs = append(errs, fmt.Errorf("step %d:\n%w", i+1, err))
		}
	}
	return errors.Join(errs...)
}

// run makes Sequence satisfy Runnable, letting it be declared alongside standalone Tests.
// client, buf, and data all come from Runner.Run; Sequence just shares them with each of its
// own steps in turn.
func (s Sequence) run(verbose bool, client *http.Client, buf *bytes.Buffer, data map[string]string) result {
	if err := s.validate(); err != nil {
		fmt.Fprintf(buf, "%s: %v\n", pink("ERROR"), err)
		return result{buf, false}
	}

	allPassed := true
	for i, step := range s {
		fmt.Fprintf(buf, "Step %d\n", i+1)
		if res := step.run(verbose, client, buf, data); !res.passed {
			allPassed = false
			break
		}
		if i < len(s)-1 {
			fmt.Fprintln(buf)
		}
	}
	fmt.Fprintln(buf, yellow("---------------------------------\n"))

	return result{buf, allPassed}
}
