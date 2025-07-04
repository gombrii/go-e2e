package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"
)

const (
	errorExit   = 1
	badArgument = 2
)

type data struct {
	Noise    int64
	Hooks    hooks
	Packages []packageInfo
}

func main() {
	wd, _ := os.Getwd()
	if len(os.Args) != 2 {
		fmt.Println("Usage: e2r <pattern>\nEg.\ne2r . current package\ne2r ./tests specific package\ne2r ./tests.go specific file\ne2r ./... current package recursively")
		os.Exit(badArgument)
	}
	pattern := os.Args[1]
	hooks, packages, err := load(wd, pattern)
	if err != nil {
		fmt.Printf("Error setting up runner: %v", err)
		os.Exit(errorExit)
	}
	data := data{time.Now().Unix(), hooks, packages}

	dir, err := os.MkdirTemp("", "e2e-runner-*")
	if err != nil {
		fmt.Printf("Error setting up runner: %v", err)
		os.Exit(errorExit)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "runner.go")
	file, err := os.Create(path)
	if err != nil {
		fmt.Printf("Error setting up runner: %v", err)
		os.Exit(errorExit)
	}
	defer file.Close()

	err = template.Must(template.New("runner").Parse(runner)).Execute(file, data)
	if err != nil {
		fmt.Printf("Error setting up runner: %v ", err)
		os.Exit(errorExit)
	}

	cmd := exec.Command("go", "run", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error executing runner: %v", err)
		os.Exit(errorExit)
	}
}
