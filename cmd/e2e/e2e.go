package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

const usageInstructions = `Usage: e2e [-v] <pattern> [env]

[-v] is optional:
  Print all response headers, not only the expected ones.

<pattern> follows the same rules as go test:
  .            current package
  ./tests      specific package
  ./tests.go   specific file
  ./...        current package and all subpackages

[env] is optional:
  Specify a custom environment name (e.g. DEV, PROD) to pass to your tests.

Examples:
  e2e .                # Run tests in current package
  e2e ./tests          # Run tests in ./tests
  e2e ./tests.go       # Run tests only in tests.go
  e2e ./... DEV        # Run tests recursively, passing env=DEV
  e2e -v ./... DEV     # Run tests with verbose header output`

const (
	errorExit   = 1
	badArgument = 2
)

type data struct {
	Noise    int64
	Setup    setup
	Packages []packageInfo
	Verbose  bool
}

func main() {
	wd, _ := os.Getwd()
	pattern, env, verbose, err := args(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(badArgument)
	}

	setup, packages, err := load(wd, pattern)
	if err != nil {
		fmt.Printf("Error setting up runner: %v\n", err)
		os.Exit(errorExit)
	}
	data := data{time.Now().Unix(), setup, packages, verbose}
	dir, err := os.MkdirTemp("", "e2e-runner-*")
	if err != nil {
		fmt.Printf("Error setting up runner: %v\n", err)
		os.Exit(errorExit)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "runner.go")
	file, err := os.Create(path)
	if err != nil {
		fmt.Printf("Error setting up runner: %v\n", err)
		os.Exit(errorExit)
	}
	defer file.Close()

	err = template.Must(template.New("runner").Parse(runner)).Execute(file, data)
	if err != nil {
		fmt.Printf("Error setting up runner: %v\n", err)
		os.Exit(errorExit)
	}

	cmd := exec.Command("go", "run", path, env)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error executing runner: %v\n", err)
		os.Exit(errorExit)
	}
}

func args(args []string) (pattern, env string, verbose bool, err error) {
	verbose, rest, err := stripVerbose(args)
	if err != nil {
		return "", "", false, err
	}
	switch len(rest) {
	case 3:
		env = rest[2]
		fallthrough
	case 2:
		pattern = rest[1]
	default:
		if len(rest) > 3 {
			return "", "", false, fmt.Errorf("unexpected argument: %q", rest[3])
		}
		return "", "", false, fmt.Errorf("%s", usageInstructions)
	}
	return pattern, env, verbose, nil
}

func stripVerbose(args []string) (bool, []string, error) {
	verbose := false
	rest := []string{}
	for _, a := range args {
		switch {
		case a == "-v":
			verbose = true
		case strings.HasPrefix(a, "-"):
			return false, nil, fmt.Errorf("unknown flag: %q", a)
		default:
			rest = append(rest, a)
		}
	}
	return verbose, rest, nil
}
