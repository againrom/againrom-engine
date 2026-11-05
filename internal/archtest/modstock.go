package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// The mod stock scan: one stock reader, modStock in pkg/game/moditems.go,
// reads the merged mod items for both games' shops, with each game's policy
// as profile data; the layer lookup is mapload.LayerItem over the same entry
// form. Outside the named readers no production function reads the merged
// item list, tests a mod item's stock flag or price, or classifies a mod code
// by secondshop.Stocked; only the two shops call the reader.
const (
	modStockFile   = "pkg/game/moditems.go"
	modStockReader = "modStock"
)

// modStockReaders are the functions that may read the merged mod item list,
// keyed by file and function, with their part.
var modStockReaders = map[string]string{
	modStockFile + " " + modStockReader:    "the stock reader",
	"pkg/game/moditemlayer.go setModItems": "the writer",
	"pkg/game/mods.go SetMods":             "the carry into the session's mod context",
	"pkg/mapload/layers.go LayerItem":      "the layer lookup",
}

// modStockEntries are the named entries: a function that reads the session's
// mod context item list for a save, with its reason. Neither stocks a shop.
var modStockEntries = map[string]string{
	// A SAVE writes each mod item's stand-in code and the worn layers' mark.
	"pkg/game/currentsave.go ExportCurrentSave context": "",
	// A LOAD checks the mark's layers against the mod items as layers.
	"pkg/game/originalrestore.go restoreOriginalTown context": "",
	"pkg/game/originalrestore.go planMission context":         "",
}

// modStockShops are the functions that call the stock reader: the first
// game's stock profile and the second game's shop tables.
var modStockShops = map[string]bool{
	"pkg/game/shopstockprofile.go stockProfile":   true,
	"pkg/game/secondtownshop.go secondShopTables": true,
}

// CheckModStock reports every second reader of the mods' stocked items in
// files, the non-test sources under pkg and cmd.
func CheckModStock(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the mod stock scan found nothing to read"}}
	}
	var vs []Violation
	readers := 0
	for _, name := range sortedNames(files) {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		inSecondShop := strings.HasPrefix(name, "pkg/secondshop/")
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			key := name + " " + fn.Name.Name
			if key == modStockFile+" "+modStockReader {
				readers++
				continue
			}
			vs = append(vs, modStockOutside(key, fn, inSecondShop, at)...)
		}
	}
	if readers != 1 {
		vs = append(vs, Violation{From: modStockFile, Reason: "the mod stock scan found " + strconv.Itoa(readers) + " " + modStockReader + " functions, want one"})
	}
	return vs
}

// modStockOutside reports the shapes of a second mod stock reader in one
// function other than the stock reader.
func modStockOutside(key string, fn *ast.FuncDecl, inSecondShop bool, at func(ast.Node) string) []Violation {
	var vs []Violation
	name := fn.Name.Name
	_, reader := modStockReaders[key]
	bound := modItemIdents(fn)
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			switch {
			case isModItemList(n) && !reader:
				vs = append(vs, Violation{From: at(n), Reason: name + " reads the merged mod items; a shop calls " + modStockReader + ", a layer lookup mapload.LayerItem"})
			case isModContextList(n) && !reader:
				if _, ok := modStockEntries[key+" context"]; !ok {
					vs = append(vs, Violation{From: at(n), Reason: name + " reads the mod context's items; a shop calls " + modStockReader + ", a layer lookup mapload.LayerItemIn"})
				}
			case (n.Sel.Name == "Stock" || n.Sel.Name == "Price") && isBoundIdent(n.X, bound):
				vs = append(vs, Violation{From: at(n), Reason: name + " reads a mod item's " + n.Sel.Name + "; the stock reader " + modStockReader + " places the mods' stocked items"})
			}
		case *ast.CallExpr:
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Stocked" && isIdent(sel.X, "secondshop") ||
				inSecondShop && isIdent(n.Fun, "Stocked") {
				vs = append(vs, Violation{From: at(n), Reason: name + " classifies a mod code by secondshop.Stocked; only " + modStockReader + " does"})
			}
			if isIdent(n.Fun, modStockReader) && !modStockShops[key] {
				vs = append(vs, Violation{From: at(n), Reason: name + " calls " + modStockReader + "; only the two shops' stock builders call it"})
			}
		}
		return true
	})
	return vs
}

// isModItemList reports a read of X.Mods.Items.
func isModItemList(s *ast.SelectorExpr) bool {
	inner, ok := s.X.(*ast.SelectorExpr)
	return ok && s.Sel.Name == "Items" && inner.Sel.Name == "Mods"
}

// isModContextList reports a read of the item list of the session's mod
// context: modContext().Items or tableModContext(t).Items.
func isModContextList(s *ast.SelectorExpr) bool {
	call, ok := s.X.(*ast.CallExpr)
	if !ok || s.Sel.Name != "Items" {
		return false
	}
	callee := calleeName(call)
	return callee == "modContext" || callee == "tableModContext"
}

// modItemIdents are the names fn binds to a mod item: a parameter or variable
// of the ModItem type, the value of a range over a ModItem list or over the
// merged item list, and the first result of a layer lookup.
func modItemIdents(fn *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	lists := map[string]bool{}
	declare := func(names []*ast.Ident, typ ast.Expr) {
		for _, n := range names {
			if isModItemType(typ) {
				out[n.Name] = true
			}
			if arr, ok := typ.(*ast.ArrayType); ok && isModItemType(arr.Elt) {
				lists[n.Name] = true
			}
		}
	}
	for _, fl := range []*ast.FieldList{fn.Recv, fn.Type.Params} {
		if fl != nil {
			for _, f := range fl.List {
				declare(f.Names, f.Type)
			}
		}
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ValueSpec:
			declare(n.Names, n.Type)
		case *ast.AssignStmt:
			if len(n.Rhs) == 1 && len(n.Lhs) > 0 {
				if call, ok := n.Rhs[0].(*ast.CallExpr); ok {
					if c := calleeName(call); c == "LayerItem" || c == "LayerItemIn" {
						if id, ok := n.Lhs[0].(*ast.Ident); ok {
							out[id.Name] = true
						}
					}
				}
			}
		case *ast.RangeStmt:
			id, ok := n.Value.(*ast.Ident)
			if !ok {
				return true
			}
			if sel, ok := n.X.(*ast.SelectorExpr); ok && (isModItemList(sel) || isModContextList(sel)) {
				out[id.Name] = true
			}
			if x, ok := n.X.(*ast.Ident); ok && lists[x.Name] {
				out[id.Name] = true
			}
		}
		return true
	})
	return out
}

func isModItemType(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name == "ModItem"
	case *ast.SelectorExpr:
		return e.Sel.Name == "ModItem" && isIdent(e.X, "mapload")
	}
	return false
}

func isBoundIdent(e ast.Expr, bound map[string]bool) bool {
	id, ok := e.(*ast.Ident)
	return ok && bound[id.Name]
}
