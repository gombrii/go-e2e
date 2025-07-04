package e2e

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
)

var progressBarMutex = sync.Mutex{}

func moveUp(lines int) {
	fmt.Printf("\033[%dA", lines)
}
func moveDown(lines int) {
	fmt.Printf("\033[%dB", lines)
}
func clearLine() {
	fmt.Print("\033[2K")
}

func red(text ...any) string {
	return fmt.Sprintf("\033[31m%v\033[0m", fmt.Sprint(text...))
}

func pink(text ...any) string {
	return fmt.Sprintf("\033[38;5;210m%v\033[0m", fmt.Sprint(text...))
}

func green(text ...any) string {
	return fmt.Sprintf("\033[32m%v\033[0m", fmt.Sprint(text...))
}

func yellow(text ...any) string {
	return fmt.Sprintf("\033[33m%v\033[0m", fmt.Sprint(text...))
}

func resultText(success bool) string {
	if success {
		return green("SUCCESS")
	}
	return red("FAIL")
}

func format(data []byte) string {
	var out bytes.Buffer
	err := json.Indent(&out, data, "", "  ")
	if err != nil {
		return strings.TrimSpace(string(data)) + "\n"
	}
	return strings.TrimSpace(out.String()) + "\n"
}

func drawProgressBar(results []result, total int) {
	numDash := 48
	segSize := int(math.Max(float64(numDash)/float64(total), 1))
	numDash = int(math.Min(float64(total*segSize), float64(numDash)))
	testsPerDash := float64(total) / float64(numDash)
	numRun := len(results)
	progress := float64(numRun) / float64(total)
	filledLen := int(progress * float64(numDash))

	bar := ""
	for segStart := 0; segStart < filledLen; segStart += segSize {
		start := int(float64(segStart) * testsPerDash)
		end := int(float64(segStart+segSize) * testsPerDash)

		fail := false
		for _, test := range results[start:end] {
			if !test.passed {
				fail = true
				break
			}
		}

		if fail {
			bar += red(strings.Repeat("=", segSize))
		} else {
			bar += green(strings.Repeat("=", segSize))
		}
	}

	head := ""
	if progress < 1 {
		head = ">"
	}

	bar += head + strings.Repeat(" ", numDash-filledLen-len(head))

	progressBarMutex.Lock()
	defer progressBarMutex.Unlock()
	fmt.Print("\n\n") // Ensure two lines exist
	moveUp(2)         // Move up to second-to-last line
	clearLine()       // Clear line where progress bar will be drawn
	fmt.Printf("\r[%s] %d%% (%d/%d)", bar, int(progress*100), numRun, total)
}

func confirm(text string) string {
	progressBarMutex.Lock()
	defer progressBarMutex.Unlock()
	reader := bufio.NewReader(os.Stdin)

	moveDown(1) // To one line below progress bar
	clearLine() // Clear line where prompt will be drawn

	fmt.Print("\n\r", text)
	s, _ := reader.ReadString('\n') //TODO: Change to scanner?

	moveUp(1)   // Back to the line where the prompt was drawn
	clearLine() // Clear line where prompt was drawn
	moveUp(1)   // To line where progress bar is drawn

	return s
}
