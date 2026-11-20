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

// mapHostType is the map editor's one edit host; what a format lets its
// tools do is a row of the format, not a host type of its own.
const mapHostType = "mapInspectionEdits"

// mapHostCalls are the host calls a wrapper type would re-implement.
var mapHostCalls = map[string]bool{"Move": true, "Clone": true, "Delete": true, "SetFields": true,
	"AddRecord": true, "CheckNewRecord": true, "NewRecordStep": true}

// mapHostBreaches lists, over the production sources of pkg/game and
// pkg/mapedit, each call of NarrowTools and each type that holds the host and
// declares one of its calls.
func mapHostBreaches(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for name, source := range files {
		if !strings.HasPrefix(name, "pkg/game/") && !strings.HasPrefix(name, "pkg/mapedit/") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		holders, methods := map[string]bool{}, map[string]map[string]bool{}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.SelectorExpr:
				if n.Sel.Name == "NarrowTools" {
					out = append(out, name+": NarrowTools is read")
				}
			case *ast.TypeSpec:
				if st, ok := n.Type.(*ast.StructType); ok && n.Name.Name != mapHostType && holdsMapHost(st) {
					holders[n.Name.Name] = true
				}
			case *ast.FuncDecl:
				if recv := methodOwner(n); recv != "" && mapHostCalls[n.Name.Name] {
					if methods[recv] == nil {
						methods[recv] = map[string]bool{}
					}
					methods[recv][n.Name.Name] = true
				}
			}
			return true
		})
		for typ := range holders {
			for call := range methods[typ] {
				out = append(out, name+": "+typ+" wraps "+mapHostType+" and declares "+call)
			}
		}
	}
	sort.Strings(out)
	return out
}

func holdsMapHost(st *ast.StructType) bool {
	for _, field := range st.Fields.List {
		typ := field.Type
		if star, ok := typ.(*ast.StarExpr); ok {
			typ = star.X
		}
		if id, ok := typ.(*ast.Ident); ok && id.Name == mapHostType {
			return true
		}
	}
	return false
}

func methodOwner(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) != 1 {
		return ""
	}
	typ := fn.Recv.List[0].Type
	if star, ok := typ.(*ast.StarExpr); ok {
		typ = star.X
	}
	if id, ok := typ.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func TestMapHostBreachesNameEachBranchAndWrapper(t *testing.T) {
	got := mapHostBreaches(t, map[string]string{
		"pkg/game/a.go": "package game\ntype wrap struct{ s *mapInspectionEdits }\nfunc (w wrap) Move() {}\nfunc (w wrap) Dirty() {}\n" +
			"type mapInspectionEdits struct{}\nfunc (s *mapInspectionEdits) Move() {}\nfunc f(m *E) bool { return m.NarrowTools() }\n",
		"pkg/mapedit/b.go": "package mapedit\ntype holder struct{ s mapInspectionEdits }\nfunc (h holder) Name() {}\n",
		"pkg/ui/c.go":      "package ui\nfunc f(m *E) bool { return m.NarrowTools() }\n",
	})
	want := []string{"pkg/game/a.go: NarrowTools is read", "pkg/game/a.go: wrap wraps mapInspectionEdits and declares Move"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("mapHostBreaches = %v, want %v", got, want)
	}
}

// TestMapEditHostIsOne states that no production code in pkg/game or
// pkg/mapedit branches on NarrowTools and that no second host type wraps
// the map edit host: what a map format allows is read from its row.
func TestMapEditHostIsOne(t *testing.T) {
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
	if _, ok := files["pkg/game/mapinspectionedit.go"]; !ok {
		t.Fatal("pkg/game/mapinspectionedit.go is missing; the scan guards nothing")
	}
	for _, b := range mapHostBreaches(t, files) {
		t.Errorf("the map edit host is not one: %s", b)
	}
}
