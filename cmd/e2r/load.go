package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"log"

	"golang.org/x/tools/go/packages"
)

type hooks struct {
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

func load(wd, target string) (hooks, []packageInfo) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:  wd,
	}

	return loadHooks(cfg), loadPackages(cfg, target)
}

func loadPackages(cfg *packages.Config, target string) []packageInfo {
	pkgs, err := packages.Load(cfg, target)
	if err != nil || packages.PrintErrors(pkgs) > 0 {
		log.Fatal("error loading packages")
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

	return packages
}

func loadHooks(cfg *packages.Config) hooks {
	pkgs, err := packages.Load(cfg, ".")
	if err != nil || len(pkgs) == 0 {
		log.Fatal("error loading root package")
	}

	hooks := hooks{}
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

	return hooks
}
