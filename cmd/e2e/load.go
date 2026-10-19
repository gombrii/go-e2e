package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

type setup struct {
	PkgPath string
	Dir     string
	// Config is the name of the exported var of type e2e.Config declared in the root package,
	// if any.
	Config string
}

type packageInfo struct {
	PkgPath string
	PkgName string
	Dir     string
	// ImportAlias is the identifier this package is imported under in the generated runner.
	// It's the same as PkgName, unless that name collided with another package's, in which
	// case it's suffixed with a number to stay a unique Go identifier.
	ImportAlias  string
	ExportedVars []exportedVar
}

type exportedVar struct {
	VarName string
	// DisplayName is what the test is keyed and printed under. It's the same as VarName,
	// unless that name collided with another package's, in which case it's prefixed with
	// the package name to disambiguate.
	DisplayName string
	TypeName    string
}

func load(wd, pattern string) (setup, []packageInfo, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedFiles,
		Dir:  wd,
	}

	stp, err := loadSetup(cfg)
	if err != nil {
		return setup{}, nil, err
	}

	pkgs, err := loadPackages(cfg, wd, pattern)
	if err != nil {
		return setup{}, nil, err
	}

	assignImportAliases(stp.PkgPath, pkgs)
	disambiguateNames(pkgs)

	return stp, pkgs, nil
}

func loadPackages(cfg *packages.Config, wd, pattern string) ([]packageInfo, error) {
	pattern, targetFile := separate(wd, pattern)
	pkgs, err := packages.Load(cfg, pattern)
	if err != nil || packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("loading packages: %v", err)
	}

	packages := make([]packageInfo, 0)
	containsTests := false

	for _, pkg := range pkgs {
		var exportedVars []exportedVar
		for _, file := range pkg.Syntax {
			if targetFile != "" {
				tf := pkg.Fset.File(file.Pos())
				if tf == nil {
					continue
				}
				name := tf.Name()
				if !filepath.IsAbs(name) {
					absName, err := filepath.Abs(name)
					if err != nil {
						continue
					}
					name = absName
				}
				if filepath.Clean(name) != targetFile {
					continue
				}
			}
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.VAR {
					continue
				}
				for _, spec := range gen.Specs {
					vs := spec.(*ast.ValueSpec)
					for _, name := range vs.Names {
						obj := pkg.TypesInfo.Defs[name]
						if obj == nil || !obj.Exported() {
							continue
						}
						typ := obj.Type()

						typeName := e2eTypeName(typ)
						if typeName != "Sequence" && typeName != "Test" {
							continue
						}

						exportedVars = append(exportedVars, exportedVar{
							VarName:     name.Name,
							DisplayName: name.Name,
							TypeName:    typeName,
						})
						containsTests = true
					}
				}
			}
		}

		if len(exportedVars) > 0 {
			packages = append(packages, packageInfo{PkgPath: pkg.PkgPath, PkgName: pkg.Name, Dir: pkg.Dir, ExportedVars: exportedVars})
		}
	}

	if !containsTests {
		return nil, errors.New("no tests provided")
	}

	return packages, nil
}

// assignImportAliases gives each package a unique Go import identifier to use in the
// generated runner. Two different import paths can share a package name. E.g. two "tests"
// packages living at different paths. Importing both under that same bare name wouldn't
// compile, so every package after the first gets its name suffixed with a number.
//
// The root package is always imported as "root", and the go-e2e package as "e2e", so both
// names are reserved. A user package named either of them gets renumbered too.
func assignImportAliases(setupPkgPath string, packages []packageInfo) {
	seen := map[string]int{"root": 1, "e2e": 1}
	for i, pkg := range packages {
		if pkg.PkgPath == setupPkgPath {
			packages[i].ImportAlias = "root"
			continue
		}
		seen[pkg.PkgName]++
		if n := seen[pkg.PkgName]; n > 1 {
			packages[i].ImportAlias = fmt.Sprintf("%s%d", pkg.PkgName, n)
		} else {
			packages[i].ImportAlias = pkg.PkgName
		}
	}
}

// disambiguateNames guards against a name collision the generated runner can't recover
// from: every exported var's own name becomes its key in the map passed to Runner.Run, so
// two vars sharing a name, even across different packages, would otherwise produce a
// duplicate map key and fail to compile. Rather than rejecting that, every var sharing a
// colliding name gets its DisplayName prefixed with its own (already-unique) import alias,
// e.g. "Ping" declared in both "smoketests" and "manualtests" becomes "smoketests.Ping" and
// "manualtests.Ping", or, if both happen to be named "tests" too, "tests.Ping" and
// "tests2.Ping". Names with no collision are left as-is. Must run after assignImportAliases.
func disambiguateNames(packages []packageInfo) {
	type loc struct{ pkgIdx, varIdx int }
	locs := make(map[string][]loc)
	for pi, pkg := range packages {
		for vi, v := range pkg.ExportedVars {
			locs[v.VarName] = append(locs[v.VarName], loc{pi, vi})
		}
	}
	for name, ls := range locs {
		if len(ls) < 2 {
			continue
		}
		for _, l := range ls {
			pkg := &packages[l.pkgIdx]
			pkg.ExportedVars[l.varIdx].DisplayName = pkg.ImportAlias + "." + name
		}
	}
}

func loadSetup(cfg *packages.Config) (setup, error) {
	pkgs, err := packages.Load(cfg, ".")
	if err != nil || len(pkgs) == 0 {
		return setup{}, fmt.Errorf("loading root package: %v", err)
	}

	stp := setup{}
	root := pkgs[0]
	for _, file := range root.Syntax {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				for _, name := range spec.(*ast.ValueSpec).Names {
					obj := root.TypesInfo.Defs[name]
					if obj == nil || !obj.Exported() || e2eTypeName(obj.Type()) != "Config" {
						continue
					}
					if stp.Config != "" {
						return setup{}, fmt.Errorf("multiple e2e.Config declared in root package: %s and %s", stp.Config, name.Name)
					}
					stp.Config = name.Name
					stp.PkgPath = root.PkgPath
					stp.Dir = root.Dir
				}
			}
		}
	}

	return stp, nil
}

// e2eTypeName returns the name of typ if it's a named type declared in the e2e package, or ""
// otherwise. Pointers aren't followed, so *e2e.Test returns "".
func e2eTypeName(typ types.Type) string {
	named, ok := typ.(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != "github.com/gombrii/go-e2e" {
		return ""
	}
	return named.Obj().Name()
}

func separate(wd, target string) (dir string, file string) {
	if !filepath.IsAbs(target) {
		target = filepath.Join(wd, target)
	}
	if strings.HasSuffix(target, ".go") {
		return filepath.Dir(target), filepath.Clean(target)
	}
	return target, ""
}
