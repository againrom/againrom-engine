package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

// The starter builds the map editor's arguments from launch.Editor's rows;
// no starter file spells an editor flag.
var editorFlagLiterals = map[string]bool{`"-assets"`: true, `"-list"`: true}

// editorFlagSpellers are the cmd/starter files that spell an editor flag as a
// string literal.
func editorFlagSpellers(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for name, source := range files {
		if !strings.HasPrefix(name, "cmd/starter/") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING && editorFlagLiterals[lit.Value] {
				out = append(out, name+":"+lit.Value)
			}
			return true
		})
	}
	sort.Strings(out)
	return out
}

func TestEditorFlagScanNamesEachSpeller(t *testing.T) {
	got := editorFlagSpellers(t, map[string]string{
		"cmd/starter/app.go":  "package main\nvar a = []string{\"-assets\", root}\n",
		"cmd/starter/x.go":    "package main\nvar b = \"-assetsx\"\n",
		"cmd/mapedit/main.go": "package main\nvar c = \"-assets\"\n",
	})
	if len(got) != 1 || got[0] != `cmd/starter/app.go:"-assets"` {
		t.Fatalf("editorFlagSpellers = %v", got)
	}
}

// TestStarterBuildsEditorArgumentsFromRows holds the starter's editor
// arguments to launch.Build over launch.Editor's rows.
func TestStarterBuildsEditorArgumentsFromRows(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["cmd/starter/app.go"]; !ok {
		t.Fatal("cmd/starter/app.go is missing; the scan guards nothing")
	}
	for _, s := range editorFlagSpellers(t, files) {
		t.Errorf("an editor flag is spelled in the starter: %s", s)
	}
}
