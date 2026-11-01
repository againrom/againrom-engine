package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// The record-kind scan. Production code in the editor roots names a
// ui.RecordKind by its constant and asks the kind for a capability; it never
// writes a kind word as a string. The words come from the constants of
// pkg/ui/recordkind.go. The scan refuses a word literal in these shapes:
//
//	compare   kind == "Unit", r.Kind != "Unit"
//	case      case "Unit": in a switch on a kind
//	field     Kind: "Unit", OpenKind: "Unit", Pick: "Unit"
//	assign    kind = "Unit", kind := "Unit", x.Kind = "Unit"
//	return    return "Unit" from a function that returns a RecordKind
//	mapkey    "Unit": inside a map literal
//	argument  a word passed to a call, other than copy and the fmt family

// The exceptions are RecordKindAllowed. Tests are not scanned.

// RecordKindRoots are the directories and file name prefixes the scan reads.
var RecordKindRoots = []struct{ Dir, Prefix string }{
	{"pkg/ui", ""},
	{"pkg/game", "map"},
	{"pkg/mapedit", ""},
	{"cmd/mapedit", ""},
}

// recordKindFile holds the constants that name the kinds.
const recordKindFile = "pkg/ui/recordkind.go"

// RecordKindAllowed are the permitted uses, keyed file:line, each with its
// reason.
var RecordKindAllowed = map[string]string{}

// RecordKindUse is one refused literal.
type RecordKindUse struct {
	File  string
	Line  int
	Shape string
	Word  string
}

func (u RecordKindUse) String() string {
	return fmt.Sprintf("%s:%d: %s %q; use the ui kind constant and a capability of ui.RecordKind", u.File, u.Line, u.Shape, u.Word)
}

// RecordKindWords reads the kind words from the RecordKind constants of src.
func RecordKindWords(src []byte) (map[string]bool, error) {
	f, err := parser.ParseFile(token.NewFileSet(), recordKindFile, src, 0)
	if err != nil {
		return nil, err
	}
	words := map[string]bool{}
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST {
			continue
		}
		for _, s := range g.Specs {
			v := s.(*ast.ValueSpec)
			id, ok := v.Type.(*ast.Ident)
			if !ok || id.Name != "RecordKind" {
				continue
			}
			for _, e := range v.Values {
				if w, ok := stringLit(e); ok {
					words[w] = true
				}
			}
		}
	}
	return words, nil
}

func stringLit(e ast.Expr) (string, bool) {
	b, ok := e.(*ast.BasicLit)
	if !ok || b.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(b.Value)
	return s, err == nil
}

// kindName reports whether e names a kind: an identifier or a field called
// kind, Kind, OpenKind or Pick.
func kindName(e ast.Expr) bool {
	var n string
	switch x := e.(type) {
	case *ast.Ident:
		n = x.Name
	case *ast.SelectorExpr:
		n = x.Sel.Name
	default:
		return false
	}
	switch n {
	case "kind", "Kind", "OpenKind", "Pick":
		return true
	}
	return false
}

func returnsKind(t *ast.FuncType) bool {
	if t == nil || t.Results == nil {
		return false
	}
	for _, r := range t.Results.List {
		switch x := r.Type.(type) {
		case *ast.Ident:
			if x.Name == "RecordKind" {
				return true
			}
		case *ast.SelectorExpr:
			if x.Sel.Name == "RecordKind" {
				return true
			}
		}
	}
	return false
}

// ScanRecordKindSource lists the kind word literals of one source file.
func ScanRecordKindSource(name string, src []byte, words map[string]bool) ([]RecordKindUse, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil, err
	}
	var uses []RecordKindUse
	seen := map[token.Pos]bool{}
	add := func(e ast.Expr, shape string) {
		w, ok := stringLit(e)
		if !ok || !words[w] || seen[e.Pos()] {
			return
		}
		seen[e.Pos()] = true
		uses = append(uses, RecordKindUse{File: name, Line: fset.Position(e.Pos()).Line, Shape: shape, Word: w})
	}
	var result []bool
	var walk func(n ast.Node)
	walk = func(root ast.Node) {
		ast.Inspect(root, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				if x.Body != nil {
					result = append(result, returnsKind(x.Type))
					walk(x.Body)
					result = result[:len(result)-1]
				}
				return false
			case *ast.FuncLit:
				result = append(result, returnsKind(x.Type))
				walk(x.Body)
				result = result[:len(result)-1]
				return false
			case *ast.BinaryExpr:
				if x.Op == token.EQL || x.Op == token.NEQ {
					if kindName(x.X) {
						add(x.Y, "compare")
					}
					if kindName(x.Y) {
						add(x.X, "compare")
					}
				}
			case *ast.SwitchStmt:
				if x.Tag != nil && kindName(x.Tag) {
					for _, c := range x.Body.List {
						for _, e := range c.(*ast.CaseClause).List {
							add(e, "case")
						}
					}
				}
			case *ast.CompositeLit:
				_, isMap := x.Type.(*ast.MapType)
				for _, e := range x.Elts {
					kv, ok := e.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					if isMap {
						add(kv.Key, "mapkey")
					} else if kindName(kv.Key) {
						add(kv.Value, "field")
					}
				}
			case *ast.AssignStmt:
				if len(x.Lhs) == len(x.Rhs) {
					for i := range x.Lhs {
						if kindName(x.Lhs[i]) {
							add(x.Rhs[i], "assign")
						}
					}
				}
			case *ast.ReturnStmt:
				if len(result) > 0 && result[len(result)-1] {
					for _, e := range x.Results {
						add(e, "return")
					}
				}
			case *ast.CallExpr:
				if skipKindCall(x.Fun) {
					break
				}
				for _, a := range x.Args {
					add(a, "argument")
				}
			}
			return true
		})
	}
	walk(f)
	sort.Slice(uses, func(i, j int) bool { return uses[i].Line < uses[j].Line })
	return uses, nil
}

// skipKindCall are the calls whose word argument is text, not a kind.
func skipKindCall(fun ast.Expr) bool {
	switch x := fun.(type) {
	case *ast.Ident:
		return x.Name == "copy" || x.Name == "panic" || x.Name == "string"
	case *ast.SelectorExpr:
		if p, ok := x.X.(*ast.Ident); ok {
			return p.Name == "fmt" || p.Name == "errors" || p.Name == "strings"
		}
	}
	return false
}

// LoadRecordKindUses measures the real tree under root.
func LoadRecordKindUses(root string) ([]RecordKindUse, error) {
	src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(recordKindFile)))
	if err != nil {
		return nil, err
	}
	words, err := RecordKindWords(src)
	if err != nil {
		return nil, err
	}
	var all []RecordKindUse
	for _, r := range RecordKindRoots {
		dir := filepath.Join(root, filepath.FromSlash(r.Dir))
		ents, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, e := range ents {
			n := e.Name()
			if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") || !strings.HasPrefix(n, r.Prefix) {
				continue
			}
			rel := r.Dir + "/" + n
			if rel == recordKindFile {
				continue
			}
			b, err := os.ReadFile(filepath.Join(dir, n))
			if err != nil {
				return nil, err
			}
			uses, err := ScanRecordKindSource(rel, b, words)
			if err != nil {
				return nil, err
			}
			for _, u := range uses {
				if _, ok := RecordKindAllowed[fmt.Sprintf("%s:%d", u.File, u.Line)]; !ok {
					all = append(all, u)
				}
			}
		}
	}
	return all, nil
}
