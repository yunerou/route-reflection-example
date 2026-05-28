package main

import (
	"go/ast"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// structDoc records the comments attached to the exported fields of
// one named struct type. The struct is referenced by qualified name
// so generated code can import it.
type structDoc struct {
	Named      *types.Named
	PkgPath    string // import path
	PkgName    string // identifier used in the source package's declaration
	TypeName   string // unqualified type name
	FieldDocs  map[string]string
}

// extractStructDocs gathers field comments for every struct type
// transitively referenced by the three generic arguments of each
// RegisterRoute call.
func extractStructDocs(pkgs []*packages.Package, calls []registerRouteCall) []structDoc {
	index := buildStructASTIndex(pkgs)

	visited := map[*types.Named]bool{}
	var ordered []*types.Named

	enqueue := func(t types.Type) {
		for _, n := range collectNamedStructs(t) {
			if visited[n] {
				continue
			}
			visited[n] = true
			ordered = append(ordered, n)
		}
	}

	// Seed from every observed call.
	for _, c := range calls {
		enqueue(c.ReqParam)
		enqueue(c.ReqBody)
		enqueue(c.RespBody)
	}

	// Walk transitively: fields of each struct may reference further
	// struct types whose comments are also interesting.
	for i := 0; i < len(ordered); i++ {
		named := ordered[i]
		st, ok := named.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		for j := 0; j < st.NumFields(); j++ {
			enqueue(st.Field(j).Type())
		}
	}

	docs := make([]structDoc, 0, len(ordered))
	for _, n := range ordered {
		obj := n.Obj()
		if obj == nil || obj.Pkg() == nil {
			continue
		}
		st, ok := index[n]
		if !ok {
			continue
		}
		fields := fieldDocComments(st)
		if len(fields) == 0 {
			continue
		}
		docs = append(docs, structDoc{
			Named:     n,
			PkgPath:   obj.Pkg().Path(),
			PkgName:   obj.Pkg().Name(),
			TypeName:  obj.Name(),
			FieldDocs: fields,
		})
	}

	// Stable ordering so generated output does not churn between runs.
	sort.Slice(docs, func(i, j int) bool {
		if docs[i].PkgPath != docs[j].PkgPath {
			return docs[i].PkgPath < docs[j].PkgPath
		}
		return docs[i].TypeName < docs[j].TypeName
	})

	return docs
}

// collectNamedStructs returns every named struct type reachable from t
// by dereferencing pointers, slices, arrays, maps and channels.
func collectNamedStructs(t types.Type) []*types.Named {
	var out []*types.Named
	seen := map[types.Type]bool{}
	var walk func(types.Type)
	walk = func(t types.Type) {
		if t == nil || seen[t] {
			return
		}
		seen[t] = true
		switch u := t.(type) {
		case *types.Named:
			if _, ok := u.Underlying().(*types.Struct); ok && u.Obj() != nil && u.Obj().Pkg() != nil {
				out = append(out, u)
			}
			walk(u.Underlying())
		case *types.Pointer:
			walk(u.Elem())
		case *types.Slice:
			walk(u.Elem())
		case *types.Array:
			walk(u.Elem())
		case *types.Map:
			walk(u.Key())
			walk(u.Elem())
		case *types.Chan:
			walk(u.Elem())
		case *types.Struct:
			for i := 0; i < u.NumFields(); i++ {
				walk(u.Field(i).Type())
			}
		}
	}
	walk(t)
	return out
}

// buildStructASTIndex maps every named struct type defined in any of
// the loaded packages to the *ast.StructType node that backs it.
//
// We need the AST (not just the types.Struct) to recover the Doc and
// trailing Comment groups attached to each field, which are lost in
// the resolved type information.
func buildStructASTIndex(pkgs []*packages.Package) map[*types.Named]*ast.StructType {
	index := map[*types.Named]*ast.StructType{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				ts, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					return true
				}
				obj := pkg.TypesInfo.Defs[ts.Name]
				if obj == nil {
					return true
				}
				named, ok := obj.Type().(*types.Named)
				if !ok {
					return true
				}
				index[named] = st
				return true
			})
		}
	}
	return index
}

// fieldDocComments extracts the documentation attached to each
// exported field of a struct, preferring the leading doc block over a
// trailing line comment.
func fieldDocComments(st *ast.StructType) map[string]string {
	docs := map[string]string{}
	if st == nil || st.Fields == nil {
		return docs
	}
	for _, field := range st.Fields.List {
		text := commentText(field.Doc)
		if text == "" {
			text = commentText(field.Comment)
		}
		if text == "" {
			continue
		}
		for _, name := range field.Names {
			if !name.IsExported() {
				continue
			}
			docs[name.Name] = text
		}
	}
	return docs
}

func commentText(g *ast.CommentGroup) string {
	if g == nil {
		return ""
	}
	return strings.TrimSpace(g.Text())
}
