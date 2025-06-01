package e2e

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
)

var variable *regexp.Regexp = regexp.MustCompile(`\$\w+`)
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

func ifTrue(text string, condition bool) string {
	if condition {
		return text
	}
	return ""
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
	barWidth := 40
	segmentWidth := barWidth / total
	filled := len(results)
	fraction := float64(filled) / float64(total)

	var bar string
	for _, result := range results {
		color := green
		if !result.passed {
			color = red
		}
		bar += color(strings.Repeat("=", segmentWidth))
	}

	head := ""
	if filled < total {
		head = ">"
	}

	remaining := total - filled
	bar += head + strings.Repeat(" ", remaining*segmentWidth-len(head))

	percent := int(fraction * 100)
	progressBarMutex.Lock()
	defer progressBarMutex.Unlock()
	fmt.Print("\n\n") // Ensure two lines exist
	moveUp(2)         // Move up to second-to-last line
	clearLine()       // Clear line where progress bar will be drawn
	fmt.Printf("\r[%s] %d%% (%d/%d)", bar, percent, filled, total)
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
