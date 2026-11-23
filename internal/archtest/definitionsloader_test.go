package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func exportedDefinitionLoaders(name string, src []byte) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	maploadNames, dotMapload := maploadImportBindings(f)
	var out []string
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !fn.Name.IsExported() {
			continue
		}
		if strings.HasPrefix(fn.Name.Name, "LoadDefinitions") || returnsDefinitionsOrTable(fn.Type.Results, maploadNames, dotMapload) {
			out = append(out, name+":"+fn.Name.Name)
		}
	}
	return out, nil
}

func maploadImportBindings(f *ast.File) (names map[string]bool, dot bool) {
	names = map[string]bool{}
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil || path != "againrom/pkg/mapload" {
			continue
		}
		if imp.Name == nil {
			names["mapload"] = true
			continue
		}
		switch imp.Name.Name {
		case ".":
			dot = true
		case "_":
		default:
			names[imp.Name.Name] = true
		}
	}
	return names, dot
}

func returnsDefinitionsOrTable(results *ast.FieldList, maploadNames map[string]bool, dotMapload bool) bool {
	if results == nil {
		return false
	}
	for _, field := range results.List {
		t := field.Type
		if star, ok := t.(*ast.StarExpr); ok {
			t = star.X
		}
		switch t := t.(type) {
		case *ast.Ident:
			if t.Name == "Definitions" || (dotMapload && t.Name == "Table") {
				return true
			}
		case *ast.SelectorExpr:
			if id, ok := t.X.(*ast.Ident); ok && maploadNames[id.Name] && t.Sel.Name == "Table" {
				return true
			}
		}
	}
	return false
}

// TestDefinitionLoaderHasOneExportedEntry checks every production file of
// pkg/game for exported definition loaders: only LoadDefinitionsOf in table.go
// may exist.
func TestDefinitionLoaderHasOneExportedEntry(t *testing.T) {
	root := moduleRootForTest(t)
	dir := filepath.Join(root, "pkg", "game")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	scanned := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		scanned++
		hits, err := exportedDefinitionLoaders(name, src)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		found = append(found, hits...)
	}
	if scanned == 0 {
		t.Fatal("pkg/game holds no production Go file; the scan went blind")
	}
	sort.Strings(found)
	if len(found) != 1 || found[0] != "table.go:LoadDefinitionsOf" {
		t.Errorf("exported definition loaders = %v, want [table.go:LoadDefinitionsOf]", found)
	}
}

func TestDefinitionLoaderGuardSeesEachShape(t *testing.T) {
	const src = `package game
import (
	"againrom/pkg/mapload"
	ml "againrom/pkg/mapload"
	"other/pkg"
)
type Definitions struct{}
func LoadDefinitionsOf() (*Definitions, error) { return nil, nil }
func LoadDefinitions() (*Definitions, error) { return nil, nil }
func LoadDefinitionsFor() (*Definitions, error) { return nil, nil }
func OpenDefinitions() (*Definitions, error) { return nil, nil }
func CopyDefinitions() Definitions { return Definitions{} }
func LoadTable() (*mapload.Table, error) { return nil, nil }
func OpenTable() (mapload.Table, error) { return mapload.Table{}, nil }
func ReadAlias() (*ml.Table, error) { return nil, nil }
func loadDefinitionsFor() (*Definitions, error) { return nil, nil }
func loadTable() (*mapload.Table, error) { return nil, nil }
func OtherTable() (*pkg.Table, error) { return nil, nil }
func LoadInt() (int, error) { return 0, nil }
`
	got, err := exportedDefinitionLoaders("table.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	want := "table.go:LoadDefinitionsOf table.go:LoadDefinitions table.go:LoadDefinitionsFor table.go:OpenDefinitions table.go:CopyDefinitions table.go:LoadTable table.go:OpenTable table.go:ReadAlias"
	if strings.Join(got, " ") != want {
		t.Fatalf("loaders = %q, want %q", strings.Join(got, " "), want)
	}
}
