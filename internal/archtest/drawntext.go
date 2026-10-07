package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// The drawn-text scan: no string literal in the library packages may hold a
// non-ASCII rune.
//
// WHY IT IS A RULE AND NOT A STYLE. This project's fonts are the game's own
// bitmap atlases and they are indexed BY BYTE: render/text's Font.walk steps
// `for i := 0; i < len(s); i++` and each byte selects a record through the
// install's code page (render/text.Convert). A Go source literal is UTF-8, so a
// single em dash in a drawn string is three bytes, and those three bytes select
// three unrelated records. On the English install U+2014 is E2 80 94, which the
// atlas draws as the three CP437 glyphs at those codes.
//
// THAT DEFECT SHIPPED. Replacing them fixes what he saw; this scan is what
// stops the next one.
//
// SCOPE AND BLIND SPOTS. It reads parsed syntax, so a non-ASCII rune in a
// comment is not a finding. It covers `pkg/...` production sources: that is
// where every drawn string is composed. It does NOT cover `cmd/...`, whose
// output goes to a console that reads UTF-8, and it cannot see a non-ASCII
// string arriving from anywhere other than a literal — an install's own bytes,
// which are the code page's already and are correct. It is lexical: it proves
// no such literal occurs, not that no mojibake can reach a font another way.

// CheckDrawnText reports every string literal holding a byte above 0x7f. Files
// are keyed by module-relative slash path, and an empty file set is itself a
// violation for LoadDrawnTextSources' own reason: a scan that has stopped
// finding sources reports nothing.
func CheckDrawnText(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{
			From:   "pkg",
			Reason: "no sources scanned - the drawn-text scan found nothing to read",
		}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	var vs []Violation
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, uerr := strconv.Unquote(lit.Value)
			if uerr != nil {
				return true
			}
			for i := 0; i < len(s); i++ {
				if s[i] < 0x80 {
					continue
				}
				vs = append(vs, Violation{
					From: name + ":" + strconv.Itoa(fset.Position(lit.Pos()).Line),
					Reason: "string literal holds a non-ASCII byte; the game's fonts are " +
						"indexed by byte, so a UTF-8 rune drawn through one becomes " +
						"one garbage glyph per byte",
				})
				return true
			}
			return true
		})
	}
	return vs
}

// LoadDrawnTextSources reads every production .go file under pkg/, keyed by its
// module-relative slash path. Test files are excluded: a fixture states a code
// page's bytes as hex by golden rule 2 and never draws them.
func LoadDrawnTextSources(moduleRoot string) (map[string]string, error) {
	root := filepath.Join(moduleRoot, "pkg")
	out := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		rel, rerr := filepath.Rel(moduleRoot, path)
		if rerr != nil {
			return rerr
		}
		out[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
