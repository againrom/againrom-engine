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

// The map editor's per-install tables live in pkg/game's gameFiles. No
// package-level variable of pkg/game or pkg/ui carries a name of a table that
// held first-game literals.
var editorTableNames = map[string]bool{
	"objectsKinds": true, "handoutBuildings": true, "heroChoices": true, "itemSuits": true, "addKinds": true,
}

// editorTableVars are the package-level variables of the production files of
// pkg/game and pkg/ui that carry an editor table name.
func editorTableVars(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for name, source := range files {
		if !strings.HasPrefix(name, "pkg/game/") && !strings.HasPrefix(name, "pkg/ui/") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.VAR {
				continue
			}
			for _, s := range g.Specs {
				for _, n := range s.(*ast.ValueSpec).Names {
					if editorTableNames[n.Name] {
						out = append(out, name+":"+n.Name)
					}
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

func TestEditorTableScanNamesEachVar(t *testing.T) {
	got := editorTableVars(t, map[string]string{
		"pkg/game/a.go": "package game\nvar itemSuits = 1\nvar other = 2\n",
		"pkg/ui/b.go":   "package ui\nvar (\n\taddKinds = 1\n)\n",
		"pkg/mod/c.go":  "package mod\nvar heroChoices = 1\n",
	})
	if len(got) != 2 || got[0] != "pkg/game/a.go:itemSuits" || got[1] != "pkg/ui/b.go:addKinds" {
		t.Fatalf("editorTableVars = %v", got)
	}
}

func TestEditorTablesAreNotPackageLiterals(t *testing.T) {
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
	if _, ok := files["pkg/game/mapeditortables.go"]; !ok {
		t.Fatal("pkg/game/mapeditortables.go is missing; the scan guards nothing")
	}
	for _, s := range editorTableVars(t, files) {
		t.Errorf("an editor table is a package variable again: %s", s)
	}
}
