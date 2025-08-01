package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

type setup struct {
	PkgPath   string
	PkgName   string
	BeforeRun string
	AfterRun  string
}

type packageInfo struct {
	PkgPath      string
	PkgName      string
	ExportedVars []exportedVar
}

type exportedVar struct {
	VarName  string
	TypeName string
}

func load(wd, target string) (setup, []packageInfo, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:  wd,
	}

	hks, err := loadSetup(cfg)
	if err != nil {
		return setup{}, nil, err
	}

	pkgs, err := loadPackages(cfg, target)
	if err != nil {
		return setup{}, nil, err
	}

	return hks, pkgs, nil
}

func loadPackages(cfg *packages.Config, target string) ([]packageInfo, error) {
	pkgs, err := packages.Load(cfg, target)
	if err != nil || packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("loading packages: %v", err)
	}

	packages := make([]packageInfo, 0)

	for _, pkg := range pkgs {
		var exportedVars []exportedVar
		for _, file := range pkg.Syntax {
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
						if typeName != "Suite" && typeName != "Sequence" {
							continue
						}

						exportedVars = append(exportedVars, exportedVar{
							VarName:  name.Name,
							TypeName: typeName,
						})
					}
				}
			}
		}

		if len(exportedVars) > 0 {
			packages = append(packages, packageInfo{pkg.PkgPath, pkg.Name, exportedVars})
		}
	}

	return packages, nil
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
