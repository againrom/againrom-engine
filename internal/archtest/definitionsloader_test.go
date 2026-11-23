package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func exportedDefinitionLoaders(name string, src []byte) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !fn.Name.IsExported() {
			continue
		}
		if strings.HasPrefix(fn.Name.Name, "LoadDefinitions") || returnsDefinitions(fn.Type.Results) {
			out = append(out, name+":"+fn.Name.Name)
		}
	}
	return out, nil
}

func returnsDefinitions(results *ast.FieldList) bool {
	if results == nil {
		return false
	}
	for _, field := range results.List {
		t := field.Type
		if star, ok := t.(*ast.StarExpr); ok {
			t = star.X
		}
		if id, ok := t.(*ast.Ident); ok && id.Name == "Definitions" {
			return true
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
type Definitions struct{}
func LoadDefinitionsOf() (*Definitions, error) { return nil, nil }
func LoadDefinitions() (*Definitions, error) { return nil, nil }
func LoadDefinitionsFor() (*Definitions, error) { return nil, nil }
func OpenDefinitions() (*Definitions, error) { return nil, nil }
func CopyDefinitions() Definitions { return Definitions{} }
func loadDefinitionsFor() (*Definitions, error) { return nil, nil }
func LoadTable() (int, error) { return 0, nil }
`
	got, err := exportedDefinitionLoaders("table.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	want := "table.go:LoadDefinitionsOf table.go:LoadDefinitions table.go:LoadDefinitionsFor table.go:OpenDefinitions table.go:CopyDefinitions"
	if strings.Join(got, " ") != want {
		t.Fatalf("loaders = %q, want %q", strings.Join(got, " "), want)
	}
}
