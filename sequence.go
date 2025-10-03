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
		// The name of the sequence. Used for test logs.
		Name string
		// The tests/steps contained within this Sequence.
		Steps Steps
	}
	// Steps is an ordered slice. In sequences tests/steps are unnamed and simply displayed as
	// "step 1", "step 2", etc. in logs.
	Steps []test
)

func (s Sequence) run(client *http.Client) result {
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
		if result := step.run(client, buf, data); !result.passed {
			allPassed = false
			break
		}
		fmt.Fprintln(buf)
	}
	fmt.Fprintf(buf, "---------------------------------\nSEQUENCE RESULT: %s\n", resultText(allPassed))
	return result{buf, allPassed, numRun}
}
