package e2e

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var variable *regexp.Regexp = regexp.MustCompile(`\$\w+`)

type (
	// Sequence is an ordered, named group of steps that run one after another, sharing a
	// common data context. Declare it directly, alongside standalone Tests.
	Sequence struct {
		// The steps to run, in order.
		Steps Steps
	}
	// Steps is an ordered slice of tests. Steps run sequentially and share a common data
	// context, making it possible to pass captured values from one step to the next. Each
	// step is displayed under its own "Step 1", "Step 2", etc. banner in output.
	Steps []Test
)

// run makes Sequence satisfy Runnable, letting it be declared alongside standalone Tests.
// name is the key it was declared under in the map passed to Runner.Run. client, buf, and
// data all come from Runner.Run too; Sequence just shares them with each of its own steps
// in turn, giving each one its "Step N" label as its name.
func (s Sequence) run(name string, verbose bool, client *http.Client, buf *bytes.Buffer, data map[string]string) result {
	allPassed := true

	fmt.Fprintln(buf, yellow("\n", center(strings.ToUpper(name), 31)))

	numRun := 0
	for i, step := range s.Steps {
		numRun = i + 1
		if res := step.run(fmt.Sprintf("Step %d", i+1), verbose, client, buf, data); !res.passed {
			allPassed = false
			break
		}
		fmt.Fprintln(buf)
	}
	fmt.Fprintf(buf, "---------------------------------\nRESULT: %s\n", resultText(allPassed))
	return result{buf, allPassed, numRun}
}
