package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

type data struct {
	Hooks    hooks
	Packages []packageInfo
}

func main() {
	wd, _ := os.Getwd()
	pattern := os.Args[1]
	hooks, packages := load(wd, pattern)
	data := data{hooks, packages}
	fmt.Printf("%+v\n", data)

	dir, err := os.MkdirTemp("", "e2e-runner-*")
	if err != nil {
		log.Fatalf("Error seting up runner: %v", err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "runner.go")
	file, err := os.Create(path)
	if err != nil {
		log.Fatalf("Error seting up runner: %v", err)
	}
	fmt.Println(path)
	defer file.Close()

	err = template.Must(template.New("runner").Parse(runner)).Execute(file, data)
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command("go", "run", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Error executing runner: %v", err)
	}
}
