package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"sort"
	"strings"
)

// townSquareProducers are the functions of pkg/game that may build the town
// square's view: the one view producer and its pane and doll filler.
var townSquareProducers = map[string]string{
	"townScreen.TownSquareView": "TownSquareView",
	"townScreen.paneSquareView": "TownSquareDoll",
}

// townSquareIdentity are the names a square view adapter must not read: a
// game's identity or a host rule, since what differs between the games is
// the description's data.
var townSquareIdentity = []string{"SecondGame", "GameOf", "Edition", "profile", "hostRules"}

// CheckTownSquareView keeps the town square's view on one adapter: in
// pkg/game only townScreen.TownSquareView builds a ui.TownSquareView, only
// townScreen.paneSquareView builds a ui.TownSquareDoll, no other function
// answers a ui.TownSquareView, and neither of them nor a townRoomView method
// reads a game's identity or a host rule.
func CheckTownSquareView(files map[string]string) []Violation {
	var vs []Violation
	built := map[string]int{}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/game/") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: err.Error()})
			continue
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			key := receiverName(fn.Recv) + "." + fn.Name.Name
			if fn.Recv == nil {
				key = fn.Name.Name
			}
			_, producer := townSquareProducers[key]
			if key != "townScreen.TownSquareView" && answersSquareView(fn.Type.Results) {
				vs = append(vs, Violation{From: name + ":" + key, Reason: "a second producer answers the town square view"})
			}
			if fn.Body == nil {
				continue
			}
			adapter := producer || receiverName(fn.Recv) == "townRoomView"
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.CompositeLit:
					if typ, ok := uiSelector(n.Type); ok && (typ == "TownSquareView" || typ == "TownSquareDoll") {
						built[typ]++
						if townSquareProducers[key] != typ {
							vs = append(vs, Violation{From: name + ":" + key, Reason: "builds ui." + typ + " outside the one town square view adapter"})
						}
					}
				case *ast.Ident:
					if adapter && slices.Contains(townSquareIdentity, n.Name) {
						vs = append(vs, Violation{From: name + ":" + key, Reason: "the town square view adapter reads " + n.Name + "; the games differ by description data"})
					}
				}
				return true
			})
		}
	}
	for _, typ := range []string{"TownSquareView", "TownSquareDoll"} {
		if built[typ] == 0 {
			vs = append(vs, Violation{From: "pkg/game", Reason: "no producer builds ui." + typ})
		}
	}
	return vs
}

// answersSquareView reports whether a result list holds a ui.TownSquareView.
func answersSquareView(results *ast.FieldList) bool {
	if results == nil {
		return false
	}
	for _, r := range results.List {
		if typ, ok := uiSelector(r.Type); ok && typ == "TownSquareView" {
			return true
		}
	}
	return false
}

// uiSelector answers the name of a ui package type expression, or of the
// element type of a slice, array or map of one.
func uiSelector(e ast.Expr) (string, bool) {
	switch t := e.(type) {
	case *ast.ArrayType:
		e = t.Elt
	case *ast.MapType:
		e = t.Value
	}
	s, ok := e.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	if id, ok := s.X.(*ast.Ident); ok && id.Name == "ui" {
		return s.Sel.Name, true
	}
	return "", false
}
