package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	//go:embed templates/usage.txt
	usageInstructions string

	//go:embed templates/main.go.tmpl
	runnerTpl string

	//go:embed templates/boot.go.tmpl
	bootTpl string

	//go:embed templates/inject.go.tmpl
	injectTpl string
)

const (
	errorExit   = 1
	badArgument = 2
)

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

	dir, err := os.MkdirTemp("", "e2e-runner-*")
	if err != nil {
		fmt.Printf("Error setting up runner: %v\n", err)
		os.Exit(errorExit)
	}
	defer os.RemoveAll(dir)

	runnerPath, err := generateFile("runner", dir, runnerTpl, map[string]any{
		"Setup":    setup,
		"Packages": packages,
		"Verbose":  verbose,
	})
	if err != nil {
		fmt.Printf("Error setting up runner: %v\n", err)
		os.Exit(errorExit)
	}

	goArgs := []string{"run"}
	if setup.Config != "" {
		overlayPath, err := generateAddressBookFiles(dir, setup, packages)
		if err != nil {
			fmt.Printf("Error setting up runner: %v\n", err)
			os.Exit(errorExit)
		}

		goArgs = append(goArgs, "-overlay="+overlayPath)
	}

	cmd := exec.Command("go", append(goArgs, runnerPath, env)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error executing runner: %v\n", err)
		os.Exit(errorExit)
	}
}

func generateFile(name, dir, tpl string, data any) (string, error) {
	path := filepath.Join(dir, name+".go")
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	return path, template.Must(template.New(name).Option("missingkey=error").Parse(tpl)).Execute(file, data)
}

func generateAddressBookFiles(dir string, setup setup, packages []packageInfo) (string, error) {
	replace := make(map[string]string)

	bootPath, err := generateFile("boot", dir, bootTpl, setup)
	if err != nil {
		return "", err
	}
	replace[filepath.Join(setup.Dir, "e2e_boot", "e2e_boot.go")] = bootPath

	for _, pkg := range packages {
		if pkg.PkgPath == setup.PkgPath {
			continue // the root can't import boot
		}
		path, err := generateFile("inject_"+pkg.ImportAlias, dir, injectTpl,
			map[string]any{
				"SetupPath": setup.PkgPath,
				"PkgName":   pkg.PkgName,
			})
		if err != nil {
			return "", err
		}
		replace[filepath.Join(pkg.Dir, "e2e_inject.go")] = path
	}

	overlay, err := json.Marshal(struct{ Replace map[string]string }{replace})
	if err != nil {
		return "", err
	}

	overlayPath := filepath.Join(dir, "overlay.json")
	if err := os.WriteFile(overlayPath, overlay, 0o666); err != nil {
		return "", err
	}

	return overlayPath, nil
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
