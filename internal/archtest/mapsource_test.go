package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"testing"
)

// mapSourceFile holds the one parser of an editor map source; no other
// production file spells a source kind's prefix.
const mapSourceFile = "pkg/ui/mapsource.go"

var mapSourcePrefixes = map[string]bool{`"new/"`: true, `"scenario/"`: true, `"loose/"`: true}

// mapSourcePrefixSpellers are the files outside the parser that spell a map
// source prefix as a string literal, in a HasPrefix call or anywhere else.
func mapSourcePrefixSpellers(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for name, source := range files {
		if name == mapSourceFile {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING && mapSourcePrefixes[lit.Value] {
				out = append(out, name+":"+lit.Value)
			}
			return true
		})
	}
	sort.Strings(out)
	return out
}

func TestMapSourceScanNamesEachSpeller(t *testing.T) {
	got := mapSourcePrefixSpellers(t, map[string]string{
		mapSourceFile:          "package ui\nvar p = \"loose/\"\n",
		"pkg/ui/x.go":          "package ui\nvar b = strings.HasPrefix(s, \"scenario/\")\nvar c = \"scenario\"\n",
		"cmd/mapedit/x.go":     "package main\nvar p = \"loose/\" + s\n",
		"pkg/game/mapcheck.go": "package game\nvar p = \"new\"\n",
	})
	if len(got) != 2 || got[0] != `cmd/mapedit/x.go:"loose/"` || got[1] != `pkg/ui/x.go:"scenario/"` {
		t.Fatalf("mapSourcePrefixSpellers = %v", got)
	}
}

// TestMapSourceIsParsedOnce holds every map source prefix to the parser.
func TestMapSourceIsParsedOnce(t *testing.T) {
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
	if _, ok := files[mapSourceFile]; !ok {
		t.Fatalf("%s is missing; the scan guards nothing", mapSourceFile)
	}
	for _, s := range mapSourcePrefixSpellers(t, files) {
		t.Errorf("a map source prefix is spelled outside %s: %s", mapSourceFile, s)
	}
}
