package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditorWordClassNamesSecondClassifier(t *testing.T) {
	const builder = "package ui\nimport \"unicode\"\nfunc editorWordClass(r rune) int { if unicode.IsSpace(r) { return 0 }; if unicode.IsLetter(r) || unicode.IsDigit(r) { return 1 }; return 2 }\n"
	for _, row := range []struct {
		name, source, want string
	}{
		{"clean", "package ui\nfunc (m *TextInputState) DeleteWordEdge(back bool) int { return editorWordClass('a') }", ""},
		{"letter", "package ui\nimport \"unicode\"\nfunc secondWord(r rune) bool { return unicode.IsLetter(r) }", "secondWord classifies a word rune"},
		{"digit", "package ui\nimport \"unicode\"\nfunc secondWord(r rune) bool { return unicode.IsDigit(r) }", "secondWord classifies a word rune"},
		{"number", "package ui\nimport \"unicode\"\nfunc secondWord(r rune) bool { return unicode.IsNumber(r) }", "secondWord classifies a word rune"},
		{"space closure", "package ui\nimport \"unicode\"\nfunc secondWord() { class := func(r rune) bool { return unicode.IsSpace(r) }; _ = class }", "secondWord classifies a word rune"},
		{"alias", "package ui\nimport u \"unicode\"\nfunc secondWord(r rune) bool { return u.IsPunct(r) }", "secondWord classifies a word rune"},
		{"function value", "package ui\nimport \"unicode\"\nvar secondWord = unicode.IsSymbol", "package declaration classifies a word rune"},
		{"unicode table", "package ui\nimport \"unicode\"\nfunc secondWord(r rune) bool { return unicode.In(r, unicode.Letter) }", "secondWord classifies a word rune"},
		{"dot import", "package ui\nimport . \"unicode\"\nfunc secondWord(r rune) bool { return IsLetter(r) }", "dot import hides word classification"},
		{"underscore", "package ui\nfunc secondWord(r rune) bool { return r == '_' }", "secondWord compares a word rune"},
		{"ASCII range", "package ui\nfunc secondWord(r rune) bool { return r >= 'a' && r <= 'z' }", "secondWord compares a word rune"},
		{"switch", "package ui\nfunc secondWord(r rune) int { switch r { case '_', '.': return 2 }; return 1 }", "secondWord switches on a word rune"},
		{"per key class", "package ui\nfunc (m *TextInputState) move() { switch editorWordClass('a') { case 1: } }", "move uses editorWordClass outside the shared"},
		{"line boundary", "package ui\nfunc (m *TextInputState) move(r rune) bool { return r == '\\n' || r == '\\r' }", ""},
		{"input acceptance", "package ui\nimport \"unicode\"\nfunc accepted(r rune) bool { return !unicode.IsControl(r) }", ""},
		{"coordinate acceptance", "package ui\nfunc coordinate(r rune) int { if r == ',' || r == ' ' { return 1 }; return 0 }", ""},
		{"line layout", "package ui\nfunc gapWords(s string) []string { return strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == '\\t' }) }", ""},
	} {
		t.Run(row.name, func(t *testing.T) {
			vs := checkEditorWordClasses(map[string]string{
				"pkg/ui/widgetkit/editortextmodel.go": builder,
				"pkg/ui/planted.go":                   row.source,
			})
			if row.want == "" {
				if len(vs) != 0 {
					t.Fatalf("clean source: %v", vs)
				}
				return
			}
			for _, v := range vs {
				if strings.Contains(v.Reason, row.want) && strings.HasPrefix(v.From, "pkg/ui/planted.go") {
					return
				}
			}
			t.Fatalf("violations %v do not name %q in planted.go", vs, row.want)
		})
	}
	if vs := checkEditorWordClasses(nil); len(vs) != 1 || !strings.Contains(vs[0].Reason, "found 0") {
		t.Fatalf("missing builder: %v", vs)
	}
}

func TestEditorWordClassHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, rel := range []string{"pkg/ui", "pkg/ui/widgetkit"} {
		dir := filepath.Join(root, filepath.FromSlash(rel))
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			files[rel+"/"+entry.Name()] = string(data)
		}
	}
	for _, v := range checkEditorWordClasses(files) {
		t.Error(v)
	}
}
