package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The inn offers scan: innOffers builds an inn's or tavern's options from
// the catalog's offer rows, innHires the hires and ChapterOffer a tavern
// row. An option or row literal elsewhere under pkg/ is a second producer;
// a type TavernRow or InnRow is a second row type.
var innOfferBuilders = map[string]bool{"pkg/game/innoffers.go:innOffers": true, "pkg/game/innoffers.go:innHires": true, "pkg/town/inn.go:ChapterOffer": true}

var innOfferTypes = map[string]bool{"innOffer": true, "OfferRow": true}

var secondOfferRowTypes = map[string]bool{"TavernRow": true, "InnRow": true}

// CheckInnOffers reports every production composite literal with values of
// an inn option or offer row, or of a list or map of them, outside the
// builders, and every type declaration of a second offer row.
// files is keyed by module-relative slash path, as LoadDrawnTextSources
// returns it.
func CheckInnOffers(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the inn offers scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		for _, decl := range f.Decls {
			owner := "package scope"
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if innOfferBuilders[name+":"+fn.Name.Name] {
					continue
				}
				owner = fn.Name.Name
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				if spec, ok := n.(*ast.TypeSpec); ok && secondOfferRowTypes[spec.Name.Name] {
					vs = append(vs, Violation{From: name + ":" + strconv.Itoa(fset.Position(spec.Pos()).Line),
						Reason: "type " + spec.Name.Name + " is a second offer row type; tavern and inn rows are town.OfferRow in their catalog's offer table"})
					return false
				}
				lit, ok := n.(*ast.CompositeLit)
				if !ok || len(lit.Elts) == 0 || !innOfferType(lit.Type) {
					return true
				}
				vs = append(vs, Violation{From: name + ":" + strconv.Itoa(fset.Position(lit.Pos()).Line),
					Reason: owner + " writes an inn option or row literal; inn options are built by innOffers from the town catalog's inn rows"})
				return false
			})
		}
	}
	return vs
}

// innOfferType reports whether a literal's type is an inn option or row, or
// a list, array or map of them.
func innOfferType(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.Ident:
		return innOfferTypes[x.Name]
	case *ast.SelectorExpr:
		return innOfferTypes[x.Sel.Name]
	case *ast.ArrayType:
		return innOfferType(x.Elt)
	case *ast.MapType:
		return innOfferType(x.Value)
	case *ast.StarExpr:
		return innOfferType(x.X)
	}
	return false
}
