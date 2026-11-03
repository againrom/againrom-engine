package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The shop binding scan: the shop page of the first game's chapter shop, the
// second game's bank shop and a mission Shop visit is bound by one function,
// townScreen.bindShop, from a shop binding row. A second bind*Shop function on
// the town screen is a second binder, a function-valued field on
// shopController that maps one item to another is a described form held as a
// hook, and a nil test on the settle step or the described form puts back a
// default that the row's value stands for. The pack unit bagUnit is a row
// value of that function type and lives on the row.
const (
	shopBindingPackage = "pkg/game/"
	shopBinder         = "bindShop"
)

// shopRowHooks are the row values a read must not test against nil.
var shopRowHooks = map[string]bool{"settle": true, "described": true}

// CheckShopBinding reports every function on *townScreen named bind...Shop
// other than bindShop, every shopController field of type
// func(sim.ItemInstance) sim.ItemInstance, and every
// comparison of a settle or described selector with nil, in the non-test
// sources of pkg/game. files is keyed by module-relative slash path, as
// LoadDrawnTextSources returns it.
func CheckShopBinding(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the shop binding scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	binders := 0
	for _, name := range names {
		if !strings.HasPrefix(name, shopBindingPackage) || strings.Contains(name[len(shopBindingPackage):], "/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if isTownScreenBinder(d) {
					if d.Name.Name == shopBinder {
						binders++
					} else {
						vs = append(vs, Violation{From: at(d), Reason: d.Name.Name + " is a second shop page binder; bind from a shop binding row with bindShop"})
					}
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == "shopController" {
						vs = append(vs, shopControllerHooks(ts, at)...)
					}
				}
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if b, ok := n.(*ast.BinaryExpr); ok && (b.Op == token.NEQ || b.Op == token.EQL) {
				if hook, ok := nilTestedHook(b); ok {
					vs = append(vs, Violation{From: at(b), Reason: hook + " is tested against nil; the row holds the first game's value"})
				}
			}
			return true
		})
	}
	if binders != 1 {
		vs = append(vs, Violation{From: "pkg/game", Reason: "the shop binding scan found " + strconv.Itoa(binders) + " townScreen." + shopBinder + " functions, want one"})
	}
	return vs
}

// isTownScreenBinder reports whether fn is a method of *townScreen whose name
// begins bind and ends Shop.
func isTownScreenBinder(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || len(fn.Recv.List) != 1 || !strings.HasPrefix(fn.Name.Name, "bind") || !strings.HasSuffix(fn.Name.Name, "Shop") {
		return false
	}
	star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	id, ok := star.X.(*ast.Ident)
	return ok && id.Name == "townScreen"
}

// shopControllerHooks reports the item-to-item function fields of the
// shopController type spec.
func shopControllerHooks(ts *ast.TypeSpec, at func(ast.Node) string) []Violation {
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return nil
	}
	var vs []Violation
	for _, field := range st.Fields.List {
		if !isItemToItem(field.Type) {
			continue
		}
		for _, id := range field.Names {
			vs = append(vs, Violation{From: at(id), Reason: id.Name + " is a func(sim.ItemInstance) sim.ItemInstance field on shopController; the caption form is a shop binding row value"})
		}
	}
	return vs
}

// isItemToItem reports whether t is func(sim.ItemInstance) sim.ItemInstance.
func isItemToItem(t ast.Expr) bool {
	fn, ok := t.(*ast.FuncType)
	if !ok || fn.Params == nil || fn.Results == nil || len(fn.Params.List) != 1 || len(fn.Results.List) != 1 {
		return false
	}
	return isSimItemInstance(fn.Params.List[0].Type) && isSimItemInstance(fn.Results.List[0].Type)
}

func isSimItemInstance(t ast.Expr) bool {
	sel, ok := t.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "ItemInstance" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "sim"
}

// nilTestedHook answers the row value b compares with nil.
func nilTestedHook(b *ast.BinaryExpr) (string, bool) {
	isNil := func(e ast.Expr) bool { id, ok := e.(*ast.Ident); return ok && id.Name == "nil" }
	hook := func(e ast.Expr) (string, bool) {
		sel, ok := e.(*ast.SelectorExpr)
		if !ok || !shopRowHooks[sel.Sel.Name] {
			return "", false
		}
		return sel.Sel.Name, true
	}
	if isNil(b.Y) {
		return hook(b.X)
	}
	if isNil(b.X) {
		return hook(b.Y)
	}
	return "", false
}
