package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

var shopStateFiles = map[string]bool{
	"pkg/game/shop.go":           true,
	"pkg/game/shopprofile.go":    true,
	"pkg/game/secondtownshop.go": true,
	"pkg/game/sleep.go":          true,
}

var shopStateSnapshots = map[string]string{
	"pkg/game/shopdeal.go shopDeal":                    "the one subordinate deal core, embedded only in Shop",
	"pkg/game/secondtownshop.go currentSecondShop":     "the native record codec; no live stock or pending deal",
	"pkg/game/originalcity_sales.go originalCityTrade": "an immutable source-sale comparison snapshot; not the live table",
}

func ShopStateSites(files map[string]string) []Violation {
	var out []Violation
	states, rules := 0, 0
	for _, name := range sortedNames(files) {
		if !strings.HasPrefix(name, "pkg/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		set := token.NewFileSet()
		f, err := parser.ParseFile(set, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			out = append(out, Violation{From: name, Reason: err.Error()})
			continue
		}
		report := func(n ast.Node, shape, reason string) {
			out = append(out, Violation{From: name + ":" + strconv.Itoa(set.Position(n.Pos()).Line) + " " + shape, Reason: reason})
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.TypeSpec:
				st, ok := x.Type.(*ast.StructType)
				if !ok {
					break
				}
				if x.Name.Name == "Shop" {
					states++
					if name != "pkg/game/shop.go" {
						report(x, "shop-state", "Shop belongs in pkg/game/shop.go")
					}
					break
				}
				if shopStateSnapshots[name+" "+x.Name.Name] != "" {
					break
				}
				for _, field := range st.Fields.List {
					bad := false
					ast.Inspect(field.Type, func(n ast.Node) bool {
						switch t := n.(type) {
						case *ast.Ident:
							if t.Name == "shopRecords" {
								bad = true
							}
						case *ast.FuncType:
							return false
						case *ast.IndexExpr:
							if id, ok := t.X.(*ast.Ident); ok && id.Name == "shopDeal" {
								bad = true
							}
						case *ast.IndexListExpr:
							if id, ok := t.X.(*ast.Ident); ok && id.Name == "shopDeal" {
								bad = true
							}
						case *ast.ArrayType:
							if id, ok := t.Elt.(*ast.Ident); ok && (id.Name == "ShopItem" || id.Name == "ShopPlace") {
								bad = true
							}
						case *ast.SelectorExpr:
							if id, ok := t.X.(*ast.Ident); ok && id.Name == "secondshop" && (t.Sel.Name == "Record" || t.Sel.Name == "Stock") {
								bad = true
							}
						}
						return true
					})
					if len(field.Names) == 0 {
						if id, ok := field.Type.(*ast.Ident); ok && id.Name == "Shop" {
							bad = true
						}
					}
					if bad {
						report(field, "shop-state", "another state holds shop records, stock or a deal: "+x.Name.Name)
					}
				}
			case *ast.FuncDecl:
				if x.Type.Results == nil {
					break
				}
				for _, result := range x.Type.Results.List {
					isRules := false
					ast.Inspect(result.Type, func(n ast.Node) bool {
						if id, ok := n.(*ast.Ident); ok && id.Name == "shopRules" {
							isRules = true
						}
						return true
					})
					if isRules {
						rules++
						var star *ast.StarExpr
						ok := false
						if x.Recv != nil {
							star, ok = x.Recv.List[0].Type.(*ast.StarExpr)
						}
						id, ptr := (*ast.Ident)(nil), false
						if ok {
							id, ptr = star.X.(*ast.Ident)
						}
						if name != "pkg/game/shop.go" || x.Name.Name != "rules" || !ptr || id.Name != "Shop" {
							report(x, "shop-rules", "another method builds shop rules: "+x.Name.Name)
						}
					}
				}
			case *ast.IfStmt:
				if !shopStateFiles[name] && shopGameChoice(x.Cond) && shopRuleUse(x) {
					report(x, "shop-game-choice", "a caller chooses a shop rule by game; read the shop profile")
				}
			case *ast.SwitchStmt:
				if !shopStateFiles[name] && (shopGameChoice(x.Tag) || shopGameChoice(x.Body)) && shopRuleUse(x.Body) {
					report(x, "shop-game-choice", "a caller chooses a shop rule by game; read the shop profile")
				}
			}
			return true
		})
	}
	if states != 1 {
		out = append(out, Violation{From: "pkg/game shop-state-count", Reason: "shop state types: " + strconv.Itoa(states) + ", want 1"})
	}
	if rules != 1 {
		out = append(out, Violation{From: "pkg/game shop-rules-count", Reason: "shop rules methods: " + strconv.Itoa(rules) + ", want 1"})
	}
	return out
}

func shopGameChoice(n ast.Node) bool {
	if n == nil {
		return false
	}
	found := false
	ast.Inspect(n, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			switch id.Name {
			case "Game", "GameROM1", "GameROM2", "SpellGameFirst", "SpellGameSecond", "ShopBank", "ShopChapter", "second", "isSecond", "secondGame":
				found = true
			}
		}
		return true
	})
	return found
}

func shopRuleUse(n ast.Node) bool {
	found := false
	ast.Inspect(n, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			switch id.Name {
			case "shopRules", "NewShop", "newSecondShop", "wholeBasket", "unpricedBack", "keepEmptied", "RestockWith", "drawStock", "chapterShopRow", "bankShopRow", "ShopBank", "ShopChapter", "bankShopShown":
				found = true
			}
		}
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "rules" {
				ast.Inspect(sel.X, func(n ast.Node) bool {
					if id, ok := n.(*ast.Ident); ok && (id.Name == "shop" || id.Name == "Shop" || id.Name == "deal") {
						found = true
					}
					return true
				})
			}
		}
		return true
	})
	return found
}
