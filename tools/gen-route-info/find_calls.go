package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

const reflectionMuxPkgPath = "github.com/yunerou/niarb/pkg/reflection-mux"

// loadPackages loads the requested patterns with enough information
// to resolve generic type arguments and follow imports across the
// module.
func loadPackages(patterns []string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedSyntax |
			packages.NeedTypesSizes,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}
	// Surface load errors but do not abort: a transient compile error in
	// one package should not prevent the generator from inspecting the
	// rest of the module.
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			fmt.Printf("warning: %s: %s\n", pkg.PkgPath, e)
		}
	}
	return pkgs, nil
}

// registerRouteCall captures one observed call to RegisterRoute and
// the three relevant generic type arguments at that site.
type registerRouteCall struct {
	Pkg       *packages.Package
	Position  string
	ReqParam  types.Type
	ReqBody   types.Type
	RespBody  types.Type
}

// findRegisterRouteCalls walks the syntax of every loaded package
// and collects every observed call to the RegisterRoute generic
// function from the reflectionmux package.
func findRegisterRouteCalls(pkgs []*packages.Package) []registerRouteCall {
	var calls []registerRouteCall
	for _, pkg := range pkgs {
		if pkg.PkgPath == reflectionMuxPkgPath {
			// The package itself declares RegisterRoute; skip to avoid
			// recursing on its own definition.
			continue
		}
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				ident := registerRouteIdent(pkg.TypesInfo, call.Fun)
				if ident == nil {
					return true
				}
				inst, ok := pkg.TypesInfo.Instances[ident]
				if !ok || inst.TypeArgs == nil || inst.TypeArgs.Len() < 3 {
					return true
				}
				calls = append(calls, registerRouteCall{
					Pkg:      pkg,
					Position: pkg.Fset.Position(call.Pos()).String(),
					ReqParam: inst.TypeArgs.At(0),
					ReqBody:  inst.TypeArgs.At(1),
					RespBody: inst.TypeArgs.At(2),
				})
				return true
			})
		}
	}
	return calls
}

// registerRouteIdent returns the *ast.Ident referring to the
// RegisterRoute function when expr is a call to it, or nil otherwise.
// It deals with the various AST shapes a generic call can take:
//
//   - RegisterRoute(...)                          — bare ident (after dot import)
//   - reflectionmux.RegisterRoute(...)            — selector
//   - RegisterRoute[A,B,C,D](...)                 — IndexListExpr / IndexExpr
//   - reflectionmux.RegisterRoute[A,B,C,D](...)   — selector wrapped in index
func registerRouteIdent(info *types.Info, fun ast.Expr) *ast.Ident {
	switch f := fun.(type) {
	case *ast.IndexExpr:
		return registerRouteIdent(info, f.X)
	case *ast.IndexListExpr:
		return registerRouteIdent(info, f.X)
	case *ast.SelectorExpr:
		if !isRegisterRouteObject(info, f.Sel) {
			return nil
		}
		return f.Sel
	case *ast.Ident:
		if !isRegisterRouteObject(info, f) {
			return nil
		}
		return f
	}
	return nil
}

func isRegisterRouteObject(info *types.Info, ident *ast.Ident) bool {
	if ident == nil || ident.Name != "RegisterRoute" {
		return false
	}
	obj := info.ObjectOf(ident)
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return strings.TrimSuffix(obj.Pkg().Path(), "/") == reflectionMuxPkgPath
}
