//go:build ignore

// One-shot helper: split handler method groups into sibling files (same package).
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type groupSpec struct {
	file string
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: go run ./tools/split_handler/main.go <src.go> <file:M1,M2|file2:M3,...>\n")
		os.Exit(2)
	}
	srcPath := os.Args[1]
	specArg := os.Args[2]
	outDir := filepath.Dir(srcPath)
	keepFile := filepath.Base(srcPath)

	var groups []groupSpec
	assigned := map[string]string{}
	for _, part := range strings.Split(specArg, "|") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		i := strings.Index(part, ":")
		if i < 0 {
			panic("bad group: " + part)
		}
		file := part[:i]
		for _, m := range strings.Split(part[i+1:], ",") {
			m = strings.TrimSpace(m)
			if m == "" {
				continue
			}
			assigned[m] = file
		}
		groups = append(groups, groupSpec{file: file})
	}

	src, err := os.ReadFile(srcPath)
	if err != nil {
		panic(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, srcPath, src, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	pkg := file.Name.Name
	imports := importSpecs(file)

	var keepDecls []ast.Decl
	byFile := map[string][]*ast.FuncDecl{}
	for _, g := range groups {
		byFile[g.file] = nil
	}

	for _, d := range file.Decls {
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			continue
		}
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
			keepDecls = append(keepDecls, d)
			continue
		}
		name := fd.Name.Name
		if dest, ok := assigned[name]; ok {
			byFile[dest] = append(byFile[dest], fd)
			continue
		}
		keepDecls = append(keepDecls, d)
	}

	write := func(path string, decls []ast.Decl, funcs []*ast.FuncDecl) {
		out := rebuild(pkg, imports, decls, funcs, fset)
		if err := os.WriteFile(path, out, 0o644); err != nil {
			panic(err)
		}
		fmt.Println("wrote", path)
	}

	write(filepath.Join(outDir, keepFile), keepDecls, nil)
	for _, g := range groups {
		write(filepath.Join(outDir, g.file), nil, byFile[g.file])
	}
}

func importSpecs(f *ast.File) []*ast.ImportSpec {
	var out []*ast.ImportSpec
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		for _, s := range gd.Specs {
			out = append(out, s.(*ast.ImportSpec))
		}
	}
	return out
}

func rebuild(pkg string, imports []*ast.ImportSpec, decls []ast.Decl, funcs []*ast.FuncDecl, fset *token.FileSet) []byte {
	var all []ast.Decl
	if len(imports) > 0 {
		gd := &ast.GenDecl{Tok: token.IMPORT}
		for _, im := range imports {
			gd.Specs = append(gd.Specs, im)
		}
		all = append(all, gd)
	}
	all = append(all, decls...)
	for _, fd := range funcs {
		all = append(all, fd)
	}
	af := &ast.File{
		Name:  ast.NewIdent(pkg),
		Decls: all,
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, af); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
