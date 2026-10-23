package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

// notCaption lists the string literals the window files may hold because they
// are not words the player reads: program arguments, a result kind, a data
// sentinel, a temporary file pattern, a character set.
var notCaption = map[string]string{
	"-assets":               "program argument",
	"-list":                 "program argument",
	"againrom":              "program name in the version line",
	"check":                 "background result kind",
	"version":               "background result kind",
	"editor":                "background result kind",
	"unknown":               "game version sentinel compared against, shown through a caption",
	".starter-write-test-*": "temporary file pattern",
	" \t\"":                 "characters that need quoting",
	"\r\n":                  "line break set",
}

var verbPattern = regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z]`)

// wordLike reports whether s reads as words: a letter outside ASCII, or two
// ASCII letters in a row once format verbs are taken out.
func wordLike(s string) bool {
	s = verbPattern.ReplaceAllString(s, "")
	run := 0
	for _, r := range s {
		if r > unicode.MaxASCII && unicode.IsLetter(r) {
			return true
		}
		if r < unicode.MaxASCII && unicode.IsLetter(r) {
			run++
			if run == 2 {
				return true
			}
		} else {
			run = 0
		}
	}
	return false
}

// errorBuilders are the calls whose first argument is an error text.
var errorBuilders = map[string]bool{"errors.New": true, "fmt.Errorf": true}

func callName(c *ast.CallExpr) string {
	if sel, ok := c.Fun.(*ast.SelectorExpr); ok {
		if x, ok := sel.X.(*ast.Ident); ok {
			return x.Name + "." + sel.Sel.Name
		}
	}
	return ""
}

// captionLiterals lists the word-like string literals of one source file that
// are not in notCaption. With errorsOnly it lists only the text of errors the
// file builds, for a file whose other literals are settings keys.
func captionLiterals(name string, src []byte, errorsOnly bool) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		panic(err)
	}
	skip := map[*ast.BasicLit]bool{}
	for _, imp := range f.Imports {
		skip[imp.Path] = true
	}
	var found []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Field:
			if n.Tag != nil {
				skip[n.Tag] = true
			}
		case *ast.CallExpr:
			if errorsOnly && errorBuilders[callName(n)] && len(n.Args) > 0 {
				if lit, ok := n.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					if s, _ := strconv.Unquote(lit.Value); wordLike(s) {
						found = append(found, fset.Position(lit.Pos()).String()+" "+lit.Value)
					}
				}
			}
		case *ast.BasicLit:
			if errorsOnly || n.Kind != token.STRING || skip[n] {
				return true
			}
			s, err := strconv.Unquote(n.Value)
			if err != nil {
				return true
			}
			if _, ok := notCaption[s]; !ok && wordLike(s) {
				found = append(found, fset.Position(n.Pos()).String()+" "+n.Value)
			}
		}
		return true
	})
	sort.Strings(found)
	return found
}

// starterFiles are the files that put text into the window: the whole file for
// the three window files, and the error texts of the settings file.
var starterFiles = []struct {
	name       string
	errorsOnly bool
}{
	{"app.go", false}, {"render.go", false}, {"window.go", false}, {"folderdialog.go", false}, {"settings.go", true},
}

// TestNoCaptionLiteralOutsideTable fails when a starter file holds a caption
// as a string literal instead of a row of the caption table.
func TestNoCaptionLiteralOutsideTable(t *testing.T) {
	for _, f := range starterFiles {
		src, err := os.ReadFile(f.name)
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range captionLiterals(f.name, src, f.errorsOnly) {
			t.Errorf("caption literal outside the table: %s", l)
		}
	}
}

// TestCaptionScannerFindsLiterals is the loss control: the scanner reports the
// literals of the shapes the starter used before the table.
func TestCaptionScannerFindsLiterals(t *testing.T) {
	src := `package main
import "errors"
func f(a *app) {
	button(r, "Play", act)
	a.fail("cannot play: %v", err)
	text(1, 2, "settings: "+path, tone)
	_ = errors.New("no base game selected")
	_ = "-assets"
	_ = "%d. %s"
}`
	got := captionLiterals("x.go", []byte(src), false)
	if len(got) != 4 {
		t.Fatalf("scanner found %d literals, want 4: %q", len(got), got)
	}
	got = captionLiterals("x.go", []byte(src), true)
	if len(got) != 1 || !strings.Contains(got[0], "no base game selected") {
		t.Fatalf("errors-only scan found %q", got)
	}
}

func TestNotCaptionIsUsed(t *testing.T) {
	var all strings.Builder
	for _, f := range starterFiles {
		b, err := os.ReadFile(f.name)
		if err != nil {
			t.Fatal(err)
		}
		all.Write(b)
	}
	for lit, why := range notCaption {
		if why == "" || !strings.Contains(all.String(), strconv.Quote(lit)) {
			t.Errorf("notCaption entry %q is unused or has no reason", lit)
		}
	}
}
