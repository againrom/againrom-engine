package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"testing"
)

func TestScriptPayloadHasOneWalkerAndOneTriggerWriter(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	module, err := loadTypeCheckedModule(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range CheckScriptFraming(module) {
		t.Error(finding)
	}
}

func TestCheckScriptFramingNamesEachShape(t *testing.T) {
	fset := token.NewFileSet()
	almFile, err := parser.ParseFile(fset, "alm.go", `package alm
type Triggers struct {
	EntryCount uint32
	Body       []byte
}
type LootSection struct{ Body []byte }
type Map struct {
	Triggers Triggers
	Loot     LootSection
}
const ScriptTriggerOnce = 0xb4
const ScriptTriggerSize = 184
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	almPkg, err := (&types.Config{}).Check(almPath, fset, []*ast.File{almFile}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, path, source string
		bad                bool
	}{
		{"a walk of the body", "caller", `package caller
import "againrom/pkg/formats/alm"
func Bad(m *alm.Map) byte { return m.Triggers.Body[0] }
`, true},
		{"the body's length", "caller", `package caller
import "againrom/pkg/formats/alm"
func Fine(m *alm.Map) (uint32, int) { return m.Triggers.EntryCount, len(m.Triggers.Body) }
`, false},
		{"another section's body", "caller", `package caller
import "againrom/pkg/formats/alm"
func Fine(m *alm.Map) []byte { return m.Loot.Body }
`, false},
		{"a slot offset outside the writer", "caller", `package caller
import "againrom/pkg/formats/alm"
func Bad(rec []byte) byte { return rec[alm.ScriptTriggerOnce] }
`, true},
		{"a slot offset in the writer", "againrom/pkg/mapedit", `package mapedit
import "againrom/pkg/formats/alm"
func Fine(rec []byte) byte { return rec[alm.ScriptTriggerOnce] }
`, false},
		{"the record size", "caller", `package caller
import "againrom/pkg/formats/alm"
func Fine() []byte { return make([]byte, alm.ScriptTriggerSize) }
`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := parser.ParseFile(fset, "caller.go", tc.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			info := &types.Info{Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
			conf := &types.Config{Importer: fixedImporter{path: almPath, pkg: almPkg}}
			if _, err := conf.Check(tc.path, fset, []*ast.File{file}, info); err != nil {
				t.Fatal(err)
			}
			module := &TypeCheckedModule{Fset: fset, Packages: map[string]*CheckedPackage{
				tc.path: {ImportPath: tc.path, Files: []*ast.File{file}, RelPaths: []string{"pkg/caller/caller.go"}, NumProd: 1, Info: info},
			}}
			if got := CheckScriptFraming(module); (len(got) != 0) != tc.bad {
				t.Fatalf("findings %v, want bad=%v", got, tc.bad)
			}
		})
	}
}
