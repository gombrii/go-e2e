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
	PkgPath   string
	PkgName   string
	BeforeRun string
	AfterRun  string
}

type packageInfo struct {
	PkgPath string
	PkgName string
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

	assignImportAliases(stp.PkgPath, stp.PkgName, pkgs)
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

						named, ok := typ.(*types.Named)
						if !ok {
							continue
						}
						if named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != "github.com/gombrii/go-e2e" {
							continue
						}
						typeName := named.Obj().Name()
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
			packages = append(packages, packageInfo{PkgPath: pkg.PkgPath, PkgName: pkg.Name, ExportedVars: exportedVars})
		}
	}

	if !containsTests {
		return nil, errors.New("no tests provided")
	}

	return packages, nil
}

// assignImportAliases gives each package a unique Go import identifier to use in the
// generated runner. Two different import paths can share a package name. E.g. two "tests"
// packages living at different paths. Importing both under that same bare name would
//
// setupPkgPath and setupPkgName reserve the setup package's own name so that a different
// package sharing it gets renumbered, and, when the setup package is itself one of these.
func assignImportAliases(setupPkgPath, setupPkgName string, packages []packageInfo) {
	seen := map[string]int{}
	if setupPkgName != "" {
		seen[setupPkgName] = 1
	}
	for i, pkg := range packages {
		if pkg.PkgPath == setupPkgPath {
			packages[i].ImportAlias = setupPkgName
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

	hooks := setup{}
	root := pkgs[0]
	for _, file := range root.Syntax {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name == nil || !fn.Name.IsExported() {
				continue
			}

			switch fn.Name.Name {
			case "BeforeRun":
				if fn.Type.Results != nil && len(fn.Type.Params.List) == 0 && len(fn.Type.Results.List) == 1 {
					result := root.TypesInfo.TypeOf(fn.Type.Results.List[0].Type)
					if iface, ok := result.Underlying().(*types.Interface); ok && iface.NumMethods() == 0 {
						hooks.BeforeRun = "BeforeRun"
						hooks.PkgPath = root.PkgPath
						hooks.PkgName = root.Name
					}
				}
			case "AfterRun":
				if fn.Type.Results == nil && len(fn.Type.Params.List) == 1 {
					param := root.TypesInfo.TypeOf(fn.Type.Params.List[0].Type)
					if iface, ok := param.Underlying().(*types.Interface); ok && iface.NumMethods() == 0 {
						hooks.AfterRun = "AfterRun"
						hooks.PkgPath = root.PkgPath
						hooks.PkgName = root.Name
					}
				}
			}
		}
	}

	return hooks, nil
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
