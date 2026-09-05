package e2e

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Delay pauses execution for the given duration before the test runs. The duration is
// parsed using Go's standard format, e.g. "500ms" or "2s". Progress is shown with a
// spinner while waiting.
//
// Useful when a previous step triggers something asynchronous that needs time to settle
// before the next assertion — for example waiting for a cache to populate, for an
// eventual consistency window to close, or for a background job to complete.
func Delay(delay string) func(data map[string]string) (string, error) {
	return func(data map[string]string) (string, error) {
		progressBarMutex.Lock()
		defer progressBarMutex.Unlock()

		del, err := time.ParseDuration(delay)
		if err != nil {
			return fmt.Sprintf("delay %s", delay), fmt.Errorf("parsing delay: %v", err)
		}
		moveDown(1)
		clearLine()

		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧"}
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		start := time.Now()
		frame := 0

		for time.Since(start) < del {
			fmt.Printf("\rDelay: %s %.1fs / %.1fs", spinner[frame%len(spinner)], time.Since(start).Seconds(), del.Seconds())
			frame++
			<-ticker.C
		}

		clearLine()
		moveUp(1)

		return fmt.Sprintf("delay: %s", delay), nil
	}
}

// Input prompts the user for a string value before the test runs. prompt is the message
// shown to the user. The entered value is stored under mapTo and can be referenced
// elsewhere in the test using the $-prefix, e.g. "$mapTo".
func Input(prompt string, mapTo string) func(data map[string]string) (string, error) {
	return func(data map[string]string) (string, error) {
		progressBarMutex.Lock()
		defer progressBarMutex.Unlock()
		reader := bufio.NewReader(os.Stdin)

		moveDown(1) // To one line below progress bar
		clearLine() // Clear line where prompt will be drawn

		fmt.Print("\r? ", prompt, ": ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Sprintf("manual input %q", prompt), fmt.Errorf("reading input: %v", err)
		}

		moveUp(1)   // Back to the line where the prompt was drawn
		clearLine() // Clear line where prompt was drawn
		moveUp(1)   // To line where progress bar is drawn

		if mapTo != "" {
			data[mapTo] = strings.TrimSpace(input)
		}

		return fmt.Sprintf("manual input: %q", prompt), nil
	}
}

// Command runs a terminal command before the test runs. Its output is printed and the
// user is prompted to press Enter to continue. Args support the $-prefix to reference
// values captured from earlier steps.
//
// Useful for fetching local dynamic data, displaying a QR code, or any other side
// effect that should happen and be confirmed before the test proceeds.
func Command(command string, args ...string) func(data map[string]string) (string, error) {
	return func(data map[string]string) (string, error) {
		progressBarMutex.Lock()
		defer progressBarMutex.Unlock()
		reader := bufio.NewReader(os.Stdin)

		moveDown(1) // To one line below progress bar
		clearLine() // Clear line where prompt will be drawn

		for i, s := range args {
			args[i] = variable.ReplaceAllStringFunc(s, func(str string) string {
				str = strings.TrimPrefix(str, "$")
				return data[str]
			})
		}

		cmd := exec.Command(command, args...)
		out, err := cmd.Output()
		if err != nil {
			return fmt.Sprintf("command run %q", command), fmt.Errorf("executing command: %v", err)
		}

		outStr := strings.TrimSuffix(string(out), "\n")
		numLines := strings.Count(outStr, "\n")

		if len(strings.TrimSpace(outStr)) > 0 {
			numLines++
			fmt.Print("\r", outStr, "\nContinue with Enter")
		} else {
			fmt.Print("\rContinue with Enter")
		}
		reader.ReadString('\n')

		for range numLines + 1 { // Remove all lines printed by the executed command
			moveUp(1)
			clearLine()
		}

		moveUp(1) // To line where progress bar is drawn

		return fmt.Sprintf("command run: %q", command), nil
	}
}
