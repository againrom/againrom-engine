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

// cellAdmissionFile holds the editor's one cell admission table; the ring
// rule and the old bounds helpers are spelled nowhere else in pkg/game.
const cellAdmissionFile = "pkg/game/mapcelladmission.go"

var cellAdmissionNames = []string{"ringRefuses", "insideMap(", "cellError(", "InBorderRing("}

// groundArmTableFile holds the one arm table {bit, word ID}.
const groundArmTableFile = "pkg/mapload/groundarms.go"

var groundArmWords = map[string]bool{`"editor.pass.border"`: true, `"editor.pass.mountain"`: true, `"editor.pass.water"`: true,
	`"editor.pass.flag"`: true, `"editor.pass.object"`: true, `"editor.pass.structure"`: true}

// cellAdmissionSpellers are the pkg/game files outside the admission file
// that spell one of cellAdmissionNames.
func cellAdmissionSpellers(files map[string]string) []string {
	var out []string
	for name, source := range files {
		if !strings.HasPrefix(name, "pkg/game/") || name == cellAdmissionFile {
			continue
		}
		for _, n := range cellAdmissionNames {
			if strings.Contains(source, n) {
				out = append(out, name+":"+n)
			}
		}
	}
	sort.Strings(out)
	return out
}

// groundArmWordSpellers are the files outside the arm table that spell a
// passability arm's word ID as a string literal.
func groundArmWordSpellers(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for name, source := range files {
		if name == groundArmTableFile {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING && groundArmWords[lit.Value] {
				out = append(out, name+":"+lit.Value)
			}
			return true
		})
	}
	sort.Strings(out)
	return out
}

func TestCellAdmissionScansNameEachSpeller(t *testing.T) {
	got := cellAdmissionSpellers(map[string]string{
		cellAdmissionFile:      "package game\nvar ringRefuses = 1\n",
		"pkg/game/x.go":        "package game\nfunc f() { insideMap(m) }\n",
		"pkg/ui/x.go":          "package ui\nvar ringRefuses = 1\n",
		"pkg/game/mapcheck.go": "package game\nvar _ = mapload.InBorderRing(m, 0, 0)\n",
	})
	if len(got) != 2 || got[0] != "pkg/game/mapcheck.go:InBorderRing(" || got[1] != "pkg/game/x.go:insideMap(" {
		t.Fatalf("cellAdmissionSpellers = %v", got)
	}
	words := groundArmWordSpellers(t, map[string]string{
		groundArmTableFile: "package mapload\nvar w = \"editor.pass.water\"\n",
		"pkg/ui/x.go":      "package ui\nvar w = \"editor.pass.water\"\nvar o = \"editor.pass.open\"\n",
	})
	if len(words) != 1 || words[0] != `pkg/ui/x.go:"editor.pass.water"` {
		t.Fatalf("groundArmWordSpellers = %v", words)
	}
}

// TestCellAdmissionIsOneTable holds the editor's cell admission to its table
// and the passability arms' words to mapload's arm table.
func TestCellAdmissionIsOneTable(t *testing.T) {
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
	for _, f := range []string{cellAdmissionFile, groundArmTableFile} {
		if _, ok := files[f]; !ok {
			t.Fatalf("%s is missing; the scan guards nothing", f)
		}
	}
	for _, s := range cellAdmissionSpellers(files) {
		t.Errorf("a cell admission is judged outside %s: %s", cellAdmissionFile, s)
	}
	for _, s := range groundArmWordSpellers(t, files) {
		t.Errorf("a passability arm word is spelled outside %s: %s", groundArmTableFile, s)
	}
}
