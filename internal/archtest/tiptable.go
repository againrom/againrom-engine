package archtest

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The tip table scan: every tip popup is a row of its game's table under
// TipTableDir, so no other home names a tip's text or rectangle. A
// production pkg/ Go file holds no string literal naming a tip text (a path
// through a tips directory or file, or a second-game tips section) and
// declares no package-level tip rectangle; no other description under pkg/
// holds a "tip" or "tips" object or array, or a string naming a tip text. A
// hotspot's numeric "tip" hover slot is not a popup and passes. The scan is
// lexical: a tip text assembled from parts at run time is not seen.
const TipTableDir = "pkg/game/tips/"

var (
	tipTextLiteral = regexp.MustCompile(`(^|/)tips(/|\d|%|$)|^tips\d+$`)
	tipRectName    = regexp.MustCompile(`(?i)tip.*rect|rect.*tip`)
)

// CheckTipHomes reports every tip text or rectangle named outside the tip
// tables: goFiles are production Go sources and jsonFiles descriptions, both
// by slash path from the module root.
func CheckTipHomes(goFiles, jsonFiles map[string]string) []Violation {
	if len(goFiles) == 0 || len(jsonFiles) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the tip table scan found nothing to read"}}
	}
	var vs []Violation
	for _, name := range sortedNames(goFiles) {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, goFiles[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		ast.Inspect(f, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if s, err := strconv.Unquote(lit.Value); err == nil && tipTextLiteral.MatchString(s) {
					vs = append(vs, Violation{From: at(lit), Reason: "names tip text " + strconv.Quote(s) + "; give the tip a row in its game's table under " + TipTableDir})
				}
			}
			return true
		})
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || (gen.Tok != token.VAR && gen.Tok != token.CONST) {
				continue
			}
			for _, spec := range gen.Specs {
				for _, id := range spec.(*ast.ValueSpec).Names {
					if tipRectName.MatchString(id.Name) {
						vs = append(vs, Violation{From: at(id), Reason: "declares tip rectangle " + id.Name + "; the rectangle belongs to the tip's row under " + TipTableDir})
					}
				}
			}
		}
	}
	for _, name := range sortedNames(jsonFiles) {
		if strings.HasPrefix(name, TipTableDir) {
			continue
		}
		var v any
		if err := json.Unmarshal([]byte(jsonFiles[name]), &v); err != nil {
			vs = append(vs, Violation{From: name, Reason: "description does not decode: " + err.Error()})
			continue
		}
		vs = append(vs, tipHomesIn(name, "", v)...)
	}
	return vs
}

func tipHomesIn(file, path string, v any) []Violation {
	var vs []Violation
	switch x := v.(type) {
	case map[string]any:
		for _, k := range sortedNames(x) {
			p := path + "." + k
			switch x[k].(type) {
			case map[string]any, []any:
				if k == "tip" || k == "tips" {
					vs = append(vs, Violation{From: file + ":" + p, Reason: "holds a tip popup; give it a row in its game's table under " + TipTableDir})
					continue
				}
			}
			vs = append(vs, tipHomesIn(file, p, x[k])...)
		}
	case []any:
		for i, e := range x {
			vs = append(vs, tipHomesIn(file, path+"["+strconv.Itoa(i)+"]", e)...)
		}
	case string:
		if tipTextLiteral.MatchString(x) {
			vs = append(vs, Violation{From: file + ":" + path, Reason: "names tip text " + strconv.Quote(x) + "; give the tip a row in its game's table under " + TipTableDir})
		}
	}
	return vs
}

func sortedNames[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// LoadDescriptions reads every JSON file under pkg/ outside testdata
// directories, by slash path from the module root.
func LoadDescriptions(moduleRoot string) (map[string]string, error) {
	out := make(map[string]string)
	err := filepath.WalkDir(filepath.Join(moduleRoot, "pkg"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "testdata" {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
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
	return out, err
}
