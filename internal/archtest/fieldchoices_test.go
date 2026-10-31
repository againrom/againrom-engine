package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"
)

func checkFieldChoices(name, src string) (found int, vs []string) {
	f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.SkipObjectResolution)
	if err != nil {
		return 0, []string{name + ": " + err.Error()}
	}
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		switch fn.Name.Name {
		case "scriptFieldChoices":
			vs = append(vs, name+": scriptFieldChoices is back; script field choices are registry rows")
		case "RecordChoices":
			found++
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					vs = append(vs, name+": RecordChoices holds "+lit.Value+"; a field's choices are a registry row")
				}
				return true
			})
		case "Fields":
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && (sel.Sel.Name == "RecordChoices" || sel.Sel.Name == "scriptParameterChoices") {
					vs = append(vs, name+": Fields asks "+sel.Sel.Name+"; it asks fieldChoices")
				}
				return true
			})
		}
	}
	return found, vs
}

func TestCheckFieldChoices(t *testing.T) {
	cases := []struct {
		name, src string
		bad       bool
	}{
		{"the lookup", "package game\nfunc (s *E) RecordChoices(kind string, index int, key string) []C { return s.fieldChoices(A{kind: kind, index: index, key: key}) }\n", false},
		{"a key branch", "package game\nfunc (s *E) RecordChoices(kind string, index int, key string) []C { switch key { case \"sex\": return nil }; return nil }\n", true},
		{"a second script producer", "package game\nfunc scriptFieldChoices(kind, key string) []C { return nil }\n", true},
		{"a direct ask", "package game\nfunc (s *E) Fields(kind string, index int) []F { _ = s.RecordChoices(kind, index, \"k\"); return nil }\n", true},
	}
	for _, tc := range cases {
		_, vs := checkFieldChoices("pkg/game/a.go", tc.src)
		if (len(vs) != 0) != tc.bad {
			t.Errorf("%s: violations %v, want bad %v", tc.name, vs, tc.bad)
		}
	}
}

func TestEditorFieldChoicesHaveOneRegistry(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for name, src := range files {
		if len(name) < len("pkg/game/") || name[:len("pkg/game/")] != "pkg/game/" {
			continue
		}
		n, vs := checkFieldChoices(name, src)
		found += n
		for _, v := range vs {
			t.Error(v)
		}
	}
	if found != 1 {
		t.Fatalf("RecordChoices methods = %d, want 1; the scan guards nothing", found)
	}
}
