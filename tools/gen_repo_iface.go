//go:build ignore

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	dir := filepath.Join("internal", "repository")
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		name := fi.Name()
		return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") &&
			name != "interfaces.go" && name != "pagination_helper.go"
	}, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	var out strings.Builder
	out.WriteString("package repository\n\nimport (\n\t\"context\"\n\t\"time\"\n\n\t\"yunshu/internal/model\"\n\t\"yunshu/internal/pkg/k8sauth\"\n\n\t\"gorm.io/gorm\"\n)\n\n")
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, decl := range f.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.TYPE {
					continue
				}
				for _, spec := range gen.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || !strings.HasSuffix(ts.Name.Name, "Repository") {
						continue
					}
					structName := ts.Name.Name
					iface := strings.TrimSuffix(structName, "Repository") + "Repo"
					var methods []string
					for _, d := range f.Decls {
						fn, ok := d.(*ast.FuncDecl)
						if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
							continue
						}
						star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
						if !ok {
							continue
						}
						ident, ok := star.X.(*ast.Ident)
						if !ok || ident.Name != structName {
							continue
						}
						if !fn.Name.IsExported() {
							continue
						}
						params := fieldsExpr(fset, fn.Type.Params)
						results := fieldsExpr(fset, fn.Type.Results)
						sig := fmt.Sprintf("\t%s(%s)", fn.Name.Name, params)
						if results != "" {
							sig += " (" + results + ")"
						}
						methods = append(methods, sig)
					}
					if len(methods) == 0 {
						continue
					}
					out.WriteString(fmt.Sprintf("// %s is implemented by *%s.\n", iface, structName))
					out.WriteString(fmt.Sprintf("type %s interface {\n", iface))
					for _, m := range methods {
						out.WriteString(m + "\n")
					}
					out.WriteString("}\n\n")
					out.WriteString(fmt.Sprintf("var _ %s = (*%s)(nil)\n\n", iface, structName))
				}
			}
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "interfaces.go"), []byte(out.String()), 0644); err != nil {
		panic(err)
	}
	fmt.Println("wrote internal/repository/interfaces.go")
}

func fieldsExpr(fset *token.FileSet, fl *ast.FieldList) string {
	if fl == nil {
		return ""
	}
	var parts []string
	for _, field := range fl.List {
		typ := typeString(fset, field.Type)
		if len(field.Names) == 0 {
			parts = append(parts, typ)
			continue
		}
		for _, n := range field.Names {
			parts = append(parts, n.Name+" "+typ)
		}
	}
	return strings.Join(parts, ", ")
}

func typeString(fset *token.FileSet, e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeString(fset, t.X)
	case *ast.SelectorExpr:
		return typeString(fset, t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeString(fset, t.Elt)
		}
	case *ast.MapType:
		return "map[" + typeString(fset, t.Key) + "]" + typeString(fset, t.Value)
	case *ast.InterfaceType:
		if t.Methods == nil || t.Methods.NumFields() == 0 {
			return "any"
		}
	case *ast.FuncType:
		p := fieldsExpr(fset, t.Params)
		r := fieldsExpr(fset, t.Results)
		if r != "" {
			return "func(" + p + ") (" + r + ")"
		}
		return "func(" + p + ")"
	case *ast.StructType:
		return "struct{}"
	}
	return "any"
}
