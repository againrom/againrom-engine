package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// wizardRowCode is the trigger wizard's reader and writer, by package: they
// read and write every kind through the item table, so none may test one
// opcode or one item's label word.
var wizardRowCode = map[string][]string{
	"pkg/ui/":   {"readParams", "readCondition", "readPair", "reads", "readAction", "rowsPlan", "wizardNode"},
	"pkg/game/": {"writeWizardTrigger", "wizardNode"},
}

// wizardTables are the one item table's two halves.
var wizardTables = map[string]bool{"wizConditions": true, "wizActions": true}

func wizardSelector(e ast.Expr, names ...string) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	for _, n := range names {
		if sel.Sel.Name == n {
			return true
		}
	}
	return false
}

func wizardLiteral(e ast.Expr, kind token.Token) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == kind && lit.Value != "0"
}

func wizardItemType(e ast.Expr) bool {
	if a, ok := e.(*ast.ArrayType); ok {
		e = a.Elt
	}
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "wizItem"
}

// checkWizardItems reports, in one file, a per-kind branch in the wizard's
// reader or writer and an item built outside the item table; found counts
// the reader and writer functions it saw.
func checkWizardItems(name, src string) (found int, vs []string) {
	var funcs []string
	for prefix, fs := range wizardRowCode {
		if strings.HasPrefix(name, prefix) {
			funcs = fs
		}
	}
	f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.SkipObjectResolution)
	if err != nil {
		return 0, []string{name + ": " + err.Error()}
	}
	table := map[ast.Node]bool{}
	for _, d := range f.Decls {
		if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.VAR {
			for _, sp := range g.Specs {
				v := sp.(*ast.ValueSpec)
				for i, n := range v.Names {
					if i < len(v.Values) && wizardTables[n.Name] {
						table[v.Values[i]] = true
					}
				}
			}
		}
	}
	if strings.HasPrefix(name, "pkg/ui/") {
		ast.Inspect(f, func(n ast.Node) bool {
			if table[n] {
				return false
			}
			if lit, ok := n.(*ast.CompositeLit); ok && lit.Type != nil && wizardItemType(lit.Type) {
				vs = append(vs, name+": a wizard item built outside wizConditions and wizActions; an item is a row of the one table")
			}
			return true
		})
	}
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		mine := false
		for _, want := range funcs {
			mine = mine || fn.Name.Name == want
		}
		if !mine {
			continue
		}
		found++
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.BinaryExpr:
				if n.Op != token.EQL && n.Op != token.NEQ {
					return true
				}
				for _, p := range [][2]ast.Expr{{n.X, n.Y}, {n.Y, n.X}} {
					if wizardSelector(p[0], "Opcode", "opcode") && wizardLiteral(p[1], token.INT) {
						vs = append(vs, name+": "+fn.Name.Name+" tests one opcode; a kind is a row of the item table")
					}
					if wizardSelector(p[0], "labelWord") && wizardLiteral(p[1], token.STRING) {
						vs = append(vs, name+": "+fn.Name.Name+" tests one item's label; a kind is a row of the item table")
					}
				}
			case *ast.SwitchStmt:
				if n.Tag != nil && wizardSelector(n.Tag, "Opcode", "opcode", "labelWord") {
					vs = append(vs, name+": "+fn.Name.Name+" switches on the kind; a kind is a row of the item table")
				}
			}
			return true
		})
	}
	return found, vs
}

func TestCheckWizardItems(t *testing.T) {
	cases := []struct {
		name, file, src string
		bad             bool
	}{
		{"the table read", "pkg/ui/a.go", "package ui\nvar wizConditions = []wizItem{{opcode: 3}}\nfunc (e *E) readPair(l, r, c uint32) int { if n.Opcode != it.opcode || n.Opcode == 0 { return -1 }; return 0 }\n", false},
		{"an opcode branch", "pkg/ui/a.go", "package ui\nfunc (e *E) readPair(l, r, c uint32) int { if n.Opcode == 19 { return 21 }; return 0 }\n", true},
		{"a label branch", "pkg/ui/a.go", "package ui\nfunc (e *E) wizardNode(r R) N { if it.labelWord == \"editor.wizard.c.variable\" { return N{} }; return N{} }\n", true},
		{"a kind switch", "pkg/game/a.go", "package game\nfunc writeWizardTrigger(m *M, t T) error { switch n.Opcode { case 34: return nil }; return nil }\n", true},
		{"a second table", "pkg/ui/a.go", "package ui\nvar wizMore = []wizItem{{opcode: 3}}\n", true},
		{"an item built in code", "pkg/ui/a.go", "package ui\nfunc more() { _ = wizItem{opcode: 3} }\n", true},
	}
	for _, tc := range cases {
		_, vs := checkWizardItems(tc.file, tc.src)
		if (len(vs) != 0) != tc.bad {
			t.Errorf("%s: violations %v, want bad %v", tc.name, vs, tc.bad)
		}
	}
}

func TestTriggerWizardReadsAndWritesEveryKindThroughTheItemTable(t *testing.T) {
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
	found, want := 0, 0
	for _, fs := range wizardRowCode {
		want += len(fs)
	}
	for name, src := range files {
		if !strings.HasPrefix(name, "pkg/ui/") && !strings.HasPrefix(name, "pkg/game/") {
			continue
		}
		n, vs := checkWizardItems(name, src)
		found += n
		for _, v := range vs {
			t.Error(v)
		}
	}
	if found != want {
		t.Fatalf("wizard reader and writer functions = %d, want %d; the scan guards nothing", found, want)
	}
}
