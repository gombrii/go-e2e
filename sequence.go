package e2e

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
)

var variable *regexp.Regexp = regexp.MustCompile(`\$\w+`)

// Sequence is a list of Test steps that run one after another, sharing a single context so
// that data captured in one step (see Test.Capture) can be referenced by a later one using
// the $-prefix. Sequence satisfies [Runnable], same as a standalone Test, so it's declared
// directly as its own top-level exported variable.
type Sequence []Test

// run makes Sequence satisfy Runnable, letting it be declared alongside standalone Tests.
// name is the key it was declared under in the map passed to Runner.Run. client, buf, and
// data all come from Runner.Run too; Sequence just shares them with each of its own steps
// in turn, giving each one its "Step N" label as its name.
func (s Sequence) run(name string, verbose bool, client *http.Client, buf *bytes.Buffer, data map[string]string) result {
	//fmt.Fprintln(buf, yellow(center(strings.ToUpper("Sequence "+name), 31)))

	allPassed := true
	for i, step := range s {
		fmt.Fprintf(buf, "Step %d\n", i+1)
		if res := step.run(name, verbose, client, buf, data); !res.passed {
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
