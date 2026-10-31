package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"testing"
)

const checkSpecFile = "pkg/ui/checkspec.go"

var checkMarkLiterals = map[string]bool{`"[x] "`: true, `"[ ] "`: true, `"(*) "`: true, `"( ) "`: true}

func checkMarkSpellers(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for name, source := range files {
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING && checkMarkLiterals[lit.Value] {
				out = append(out, name+":"+lit.Value)
			}
			return true
		})
	}
	sort.Strings(out)
	return out
}

func TestCheckMarkScanNamesEachSpeller(t *testing.T) {
	got := checkMarkSpellers(t, map[string]string{
		checkSpecFile:   "package ui\nvar m = [2]string{\"[ ] \", \"[x] \"}\n",
		"cmd/x/main.go": "package main\nvar m = \"[x] \"\nvar n = \"[x]\"\n",
	})
	if len(got) != 3 || got[0] != `cmd/x/main.go:"[x] "` {
		t.Fatalf("checkMarkSpellers = %v", got)
	}
}

func TestCheckBoxMarksAreSpelledInTheSpecOnly(t *testing.T) {
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
	if _, ok := files[checkSpecFile]; !ok {
		t.Fatalf("%s is missing; the scan guards nothing", checkSpecFile)
	}
	for _, s := range checkMarkSpellers(t, files) {
		if len(s) < len(checkSpecFile) || s[:len(checkSpecFile)+1] != checkSpecFile+":" {
			t.Errorf("a check mark is spelled outside %s: %s", checkSpecFile, s)
		}
	}
}
