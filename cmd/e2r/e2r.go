package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"
)

const usageInstructions = `Usage: e2r <pattern> [env]

<pattern> follows the same rules as go test:
  .            current package
  ./tests      specific package
  ./tests.go   specific file
  ./...        current package and all subpackages

[env] is optional:
  Specify an environment name (e.g. DEV, PROD) to pass to your tests.

Examples:
  e2r .                # Run tests in current package
  e2r ./tests          # Run tests in ./tests
  e2r ./tests.go       # Run tests only in tests.go
  e2r ./... DEV        # Run tests recursively, passing env=DEV`

const (
	errorExit   = 1
	badArgument = 2
)

const (
	patternArg = 1
	envArg     = 2
)

type data struct {
	Noise    int64
	Setup    setup
	Packages []packageInfo
}

func main() {
	wd, _ := os.Getwd()
	var pattern string
	var env string
	switch len(os.Args) {
	case 3:
		env = os.Args[envArg]
		fallthrough
	case 2:
		pattern = os.Args[patternArg]
	default:
		fmt.Println(usageInstructions)
		os.Exit(badArgument)
	}

	setup, packages, err := load(wd, pattern)
	if err != nil {
		fmt.Printf("Error setting up runner: %v\n", err)
		os.Exit(errorExit)
	}
	data := data{time.Now().Unix(), setup, packages}
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
