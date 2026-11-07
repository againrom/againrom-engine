package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// CheckTownState keeps town allocation and deferred art reads in their builders.
func CheckTownState(files map[string]string) []Violation {
	parsed := map[string]*ast.File{}
	aliases := map[string]string{}
	var vs []Violation
	for name, source := range files {
		if !strings.HasPrefix(name, "pkg/game/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: err.Error()})
			continue
		}
		parsed[name] = f
		ast.Inspect(f, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok {
				if id, ok := ts.Type.(*ast.Ident); ok {
					aliases[ts.Name.Name] = id.Name
				}
			}
			return true
		})
	}
	resolve := func(name string) string {
		for range len(aliases) + 1 {
			if aliases[name] == "" {
				break
			}
			name = aliases[name]
		}
		return name
	}
	builders := map[string]string{
		"townState":      "pkg/game/town.go:newTownState",
		"secondCampaign": "pkg/game/town.go:newTownState",
		"townArtLoads":   "pkg/game/townart.go:newTownArtLoads",
	}
	builders["townRoomArt"] = "pkg/game/townart.go:newTownArtLoads"
	reads := map[string]bool{"LoadTownSquareArtFor": true, "LoadTownTavernArt": true, "LoadTownSchoolArt": true, "loadShopArt": true}
	leafReads := map[string]map[string]bool{
		"roomPageArt":      {"pkg/game/shopart.go:loadShopArt": true, "pkg/game/towntavernart.go:LoadTownTavernArt": true, "pkg/game/townschoolart.go:LoadTownSchoolArt": true},
		"loadRoomSceneArt": {"pkg/game/shopart.go:loadShopArt": true, "pkg/game/towntavernart.go:LoadTownTavernArt": true, "pkg/game/townschoolart.go:LoadTownSchoolArt": true},
		"town.LoadArt":     {"pkg/game/townart.go:newTownArtLoads": true, "pkg/game/townsquare.go:LoadTownSquareArtFor": true, "pkg/game/roompage.go:loadRoomSceneArt": true},
	}
	for name, f := range parsed {
		townPackages := map[string]bool{"town": true}
		for _, imp := range f.Imports {
			if strings.Trim(imp.Path.Value, "\"") == "againrom/pkg/town" && imp.Name != nil {
				townPackages[imp.Name.Name] = true
			}
		}
		for _, decl := range f.Decls {
			where := name
			var node ast.Node = decl
			if fn, ok := decl.(*ast.FuncDecl); ok {
				where += ":" + fn.Name.Name
				node = fn.Body
			}
			ast.Inspect(node, func(n ast.Node) bool {
				var allocation string
				switch n := n.(type) {
				case *ast.CompositeLit:
					if id, ok := n.Type.(*ast.Ident); ok {
						allocation = resolve(id.Name)
					} else if st, ok := n.Type.(*ast.StructType); ok {
						allocation = townStateShape(st)
					}
				case *ast.TypeSpec:
					if st, ok := n.Type.(*ast.StructType); ok {
						shape := townStateShape(st)
						if shape != "" && n.Name.Name != shape {
							vs = append(vs, Violation{From: where + ":" + n.Name.Name, Reason: "second town storage shape: " + shape})
						}
					}
				case *ast.CallExpr:
					if townManifestRead(n) && where != "pkg/game/townart.go:newTownArtLoads" && where != "pkg/game/roompage.go:roomPageArt" {
						vs = append(vs, Violation{From: where, Reason: "town room manifest read outside bundle"})
					}
					if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "new" && len(n.Args) == 1 {
						if typ, ok := n.Args[0].(*ast.Ident); ok {
							allocation = resolve(typ.Name)
						}
					}
				case *ast.ValueSpec:
					if id, ok := n.Type.(*ast.Ident); ok {
						allocation = resolve(id.Name)
					}
				case *ast.Ident:
					if allowed, ok := leafReads[n.Name]; ok && !allowed[where] {
						vs = append(vs, Violation{From: where, Reason: "town art decoder outside bundle: " + n.Name})
					}
					if reads[n.Name] && where != "pkg/game/townart.go:newTownArtLoads" {
						vs = append(vs, Violation{From: where, Reason: "town art read outside newTownArtLoads: " + n.Name})
					}
				case *ast.SelectorExpr:
					if id, ok := n.X.(*ast.Ident); ok && townPackages[id.Name] && n.Sel.Name == "LoadArt" && !leafReads["town.LoadArt"][where] {
						vs = append(vs, Violation{From: where, Reason: "town scene read outside bundle"})
					}
				}
				if builder := builders[allocation]; builder != "" && where != builder {
					vs = append(vs, Violation{From: where, Reason: "town allocation outside " + builder + ": " + allocation})
				}
				return true
			})
		}
	}
	sort.Slice(vs, func(i, j int) bool {
		return vs[i].From+vs[i].Reason < vs[j].From+vs[j].Reason
	})
	return vs
}

func townStateShape(st *ast.StructType) string {
	names := map[string]bool{}
	for _, field := range st.Fields.List {
		for _, name := range field.Names {
			names[name.Name] = true
		}
	}
	if names["id"] && names["taken"] && names["mercPool"] {
		return "townState"
	}
	if names["bank"] && names["shop"] && names["available"] && names["current"] {
		return "secondCampaign"
	}
	if names["square"] && names["tavern"] && names["school"] && names["shop"] {
		return "townArtLoads"
	}
	if names["scene"] && names["page"] && names["shop"] && names["err"] {
		return "townRoomArt"
	}
	return ""
}

// townManifestRead reports screenArtOf(...).Load or .Bind over a room's Art
// field, the manifest a town description names. The other readers of the
// library pass a generated list or a named one, never a room's Art.
func townManifestRead(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Load" && sel.Sel.Name != "Bind" || len(call.Args) < 2 {
		return false
	}
	lib, ok := sel.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	if id, ok := lib.Fun.(*ast.Ident); !ok || id.Name != "screenArtOf" {
		return false
	}
	art, ok := call.Args[1].(*ast.SelectorExpr)
	return ok && art.Sel.Name == "Art"
}
