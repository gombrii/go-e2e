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
	Sequence struct {
		// A name identifying this sequence in test output.
		Name string
		// The steps to run, in order.
		Steps Steps
	}
	// Steps is an ordered slice of tests. Steps run sequentially and share a common data
	// context, making it possible to pass captured values from one step to the next.
	// Each step is displayed as "step 1", "step 2", etc. in output.
	Steps []test
)

func (s Sequence) run(client *http.Client, verbose bool) result {
	buf := &bytes.Buffer{}
	allPassed := true
	data := make(map[string]string)

	fmt.Fprintln(buf, yellow("\n---------------------------------"))
	fmt.Fprintln(buf, yellow(" TEST SEQUENCE - ", strings.ToUpper(s.Name)))
	fmt.Fprintln(buf, yellow("---------------------------------"))

	numRun := 0
	for i, step := range s.Steps {
		fmt.Fprintln(buf, "Step", i+1)
		numRun = i + 1
		if result := step.run(client, buf, data, verbose); !result.passed {
			allPassed = false
			break
		}
		fmt.Fprintln(buf)
	}
	fmt.Fprintf(buf, "---------------------------------\nSEQUENCE RESULT: %s\n", resultText(allPassed))
	return result{buf, allPassed, numRun}
}
