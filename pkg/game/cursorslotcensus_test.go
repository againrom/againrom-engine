package game

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func literalsOf(t *testing.T, dir string) map[string]bool {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no source in %s: %v", dir, err)
	}
	out := map[string]bool{}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if l, ok := n.(*ast.BasicLit); ok && l.Kind == token.STRING {
				if s, err := strconv.Unquote(l.Value); err == nil {
					out[s] = true
				}
			}
			return true
		})
	}
	return out
}

// Slots whose name is a string literal nowhere in pkg/ui.
func TestRegisteredCursorSlotsNamedByProductionCode(t *testing.T) {
	named := literalsOf(t, filepath.Join("..", "ui"))
	var unnamed []string
	for _, r := range cursorRegistrations {
		if !named[r.name] {
			unnamed = append(unnamed, r.name)
		}
	}
	sort.Strings(unnamed)
	want := []string{"backpack", "dice", "sattack", "scast", "sdefault", "sdefend", "smove", "spatrol"}
	if len(cursorRegistrations) != 28 || strings.Join(unnamed, " ") != strings.Join(want, " ") {
		t.Fatalf("of %d slots, unnamed = %v, want %v", len(cursorRegistrations), unnamed, want)
	}
	if literalsOf(t, filepath.Join("..", "mapload"))["smove"] {
		t.Fatal("scan control: smove found in an unrelated package")
	}
}

func TestReleaseStructureClassesWithUsableFlag(t *testing.T) {
	f := releaseFront(t)
	var got []int
	for id, c := range f.Structures.Classes {
		if c != nil && c.Usable {
			got = append(got, id)
		}
	}
	want := []int{15, 16, 28, 29, 34, 35, 39, 40, 66}
	if len(got) != len(want) {
		t.Fatalf("usable classes %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("usable classes %v, want %v", got, want)
		}
	}
}
