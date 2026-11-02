package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The saved-equipment slot scan: every slot routing of a saved actor's held
// and worn items reads sav's one slot table. In pkg, outside the table's file,
// it refuses:
//   - an equipment field named by a string literal;
//   - a list of a record's HeldWeapon and HeldShield, which orders the held
//     slots by position;
//   - in pkg/formats/sav and in a file that imports it, a Worn or WornSlots
//     position given as an integer literal, a range key over one compared
//     with an integer literal, and its length compared with one;
//   - in pkg/formats/sav, a city unit's held reference set from a slot list
//     at an integer literal.

const equipmentPlacesFile = "pkg/formats/sav/equipmentplaces.go"

const savImportPath = `"againrom/pkg/formats/sav"`

var equipmentFieldNames = map[string]bool{"HeldWeapon": true, "HeldShield": true, "Worn": true}

var wornFieldNames = map[string]bool{"Worn": true, "WornSlots": true}

var cityHeldNames = map[string]bool{"reference74": true, "reference78": true}

// CheckEquipmentPlaces reports every slot routing of a saved actor's
// equipment outside sav's slot table. files is keyed by module-relative slash
// path.
func CheckEquipmentPlaces(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the saved-equipment slot scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == equipmentPlacesFile {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		saved := strings.HasPrefix(name, "pkg/formats/sav/")
		for _, spec := range f.Imports {
			saved = saved || spec.Path.Value == savImportPath
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.ImportSpec:
				return false
			case *ast.BasicLit:
				if value, err := strconv.Unquote(n.Value); n.Kind == token.STRING && err == nil && equipmentFieldNames[value] {
					vs = append(vs, Violation{From: at(n), Reason: "an equipment field is named by a literal here; read sav's slot table (sav.EquipmentPlaces, sav.EquipmentFields)"})
				}
			case *ast.CompositeLit:
				if slotScanSelectsBoth(n.Elts, "HeldWeapon", "HeldShield") {
					vs = append(vs, Violation{From: at(n), Reason: "the held fields are listed by slot position here; route each slot through sav.EquipmentPlaces"})
				}
			case *ast.IndexExpr:
				if saved && wornSelector(n.X) && slotScanInt(n.Index) {
					vs = append(vs, Violation{From: at(n), Reason: "a Worn position is a slot-index literal here; ask sav.EquipmentFieldSlot or sav.WornOverlap"})
				}
			case *ast.RangeStmt:
				key, ok := n.Key.(*ast.Ident)
				if !saved || !ok || !wornSelector(n.X) {
					return true
				}
				ast.Inspect(n.Body, func(m ast.Node) bool {
					if cmp, ok := m.(*ast.BinaryExpr); ok && slotScanCompare(cmp.Op) && (slotScanIdent(cmp.X, key) && slotScanInt(cmp.Y) || slotScanIdent(cmp.Y, key) && slotScanInt(cmp.X)) {
						vs = append(vs, Violation{From: at(cmp), Reason: "a Worn position is compared with a slot-index literal here; ask sav.WornOverlap or sav.EquipmentFieldSlot"})
					}
					return true
				})
			case *ast.AssignStmt:
				for i, lhs := range n.Lhs {
					s, ok := lhs.(*ast.SelectorExpr)
					if !strings.HasPrefix(name, "pkg/formats/sav/") || !ok || !cityHeldNames[s.Sel.Name] || i >= len(n.Rhs) {
						continue
					}
					if index, ok := n.Rhs[i].(*ast.IndexExpr); ok && slotScanInt(index.Index) {
						vs = append(vs, Violation{From: at(n), Reason: "a city unit's held reference is taken by slot position here; route each slot through sav.EquipmentPlaces"})
					}
				}
			case *ast.BinaryExpr:
				if saved && slotScanCompare(n.Op) && (wornLength(n.X) && slotScanInt(n.Y) || wornLength(n.Y) && slotScanInt(n.X)) {
					vs = append(vs, Violation{From: at(n), Reason: "the Worn width is a literal here; compare with sav.EquipmentSlots"})
				}
			}
			return true
		})
	}
	if _, ok := files[equipmentPlacesFile]; !ok {
		vs = append(vs, Violation{From: equipmentPlacesFile, Reason: "sav's slot table is gone - the saved-equipment slot scan would hold nothing"})
	}
	return vs
}

// slotScanSelectsBoth reports whether a composite literal's elements select both
// fields.
func slotScanSelectsBoth(elts []ast.Expr, a, b string) bool {
	found := map[string]bool{}
	for _, e := range elts {
		if s, ok := e.(*ast.SelectorExpr); ok {
			found[s.Sel.Name] = true
		}
	}
	return found[a] && found[b]
}

func wornSelector(e ast.Expr) bool {
	s, ok := e.(*ast.SelectorExpr)
	return ok && wornFieldNames[s.Sel.Name]
}

func wornLength(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	fn, ok := call.Fun.(*ast.Ident)
	return ok && fn.Name == "len" && wornSelector(call.Args[0])
}

func slotScanInt(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.INT
}

func slotScanIdent(e ast.Expr, id *ast.Ident) bool {
	other, ok := e.(*ast.Ident)
	return ok && other.Name == id.Name
}

func slotScanCompare(op token.Token) bool {
	switch op {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	}
	return false
}
