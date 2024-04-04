package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
	"testing"
)

func TestFailCase(t *testing.T) {
	testCases := []struct {
		name   string
		testFn func() error
	}{
		{
			"Mutex is not identical when load sync twice",
			failCase1,
		},
		{
			"Mutex is not identical between loading from sync and retrieving from code",
			failCase2,
		},
		{
			"Load Mutex from the program that loaded once",
			successfulCase,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.testFn()
			if err != nil {
				t.Error(err)
			}
		})
	}
}

// the results of same type sync.Mutex loaded from the sync package twice are not identical
func failCase1() error {
	actualTyp := loadPackage("sync").Type("Mutex").Type()
	expectedTyp := loadPackage("sync").Type("Mutex").Type()

	fmt.Println(actualTyp.String())
	fmt.Println(expectedTyp.String())
	if !types.Identical(expectedTyp, actualTyp) {
		return errors.New("actual and expected type are different")
	}
	return nil
}

const codeSnippet = `
package main

import "sync"

var mutex sync.Mutex

func main() {}
`

// failCase2 uses SSA to load packages in different ways, then compare the type which should be equal
func failCase2() error {
	// Parse the source files.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "codeSnippet.go", codeSnippet, parser.ParseComments)
	if err != nil {
		return err
	}
	files := []*ast.File{f}

	pkg := types.NewPackage("codeSnippet", "")
	p, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()}, fset, pkg, files, ssa.SanityCheckFunctions)
	if err != nil {
		return err
	}

	actualTyp := p.Var("mutex").Object().Type()
	expected := loadPackage("sync").Type("Mutex").Type()

	fmt.Println(actualTyp.String())
	fmt.Println(expected.String())
	if !types.Identical(expected, actualTyp) {
		return errors.New("actual and expected type are different")
	}
	return nil
}

func successfulCase() error {
	p := loadPackage("github.com/xieyuschen/verify-identical/demo")

	typ := syncPkg.Type("Mutex").Type()
	vTyp := p.Var("mutex").Object().Type()

	fmt.Println(typ.String())
	fmt.Println(vTyp.String())
	if !types.Identical(typ, vTyp) {
		return errors.New("actual and expected type are different")
	}
	return nil
}

var syncPkg *ssa.Package

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
	for _, pkg := range program.AllPackages() {
		if pkg.Pkg.Path() == "sync" {
			syncPkg = pkg
		}
	}
	return spkg[0]
}
