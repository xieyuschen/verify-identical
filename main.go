package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func main() {
	checkEquality(codeSnippet)
}

const codeSnippet = `
package main

import "sync"

var mutex sync.Mutex

func main() {}
`

func checkEquality(content string) {
	// Parse the source files.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "codeSnippet.go", content, parser.ParseComments)
	if err != nil {
		fmt.Print(err) // parse error
		return
	}
	files := []*ast.File{f}

	pkg := types.NewPackage("codeSnippet", "")
	p, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()}, fset, pkg, files, ssa.SanityCheckFunctions)
	if err != nil {
		panic(err)
	}

	actualTyp := p.Var("mutex").Type()
	expected := loadPackage("sync").Type("Mutex").Type()

	if !types.Identical(expected, actualTyp) {
		panic("actual and expected type are different")
	}
}

func loadPackage(name string) *ssa.Package {
	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax | packages.NeedExportsFile,
	}

	pkgs, err := packages.Load(cfg, name)
	if err != nil {
		panic(-1)
	}
	packages.Visit(pkgs, nil, func(p *packages.Package) {
		if len(p.Errors) != 0 {
			for _, e := range p.Errors {
				fmt.Println(e)
			}
			panic("error happens")
		}
	})
	program, spkg := ssautil.Packages(pkgs, ssa.SanityCheckFunctions)
	program.Build()

	return spkg[0]
}
