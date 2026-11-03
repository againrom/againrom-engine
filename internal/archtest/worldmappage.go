package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

var worldMapPageMethods = []string{"atMap", "WorldMapView", "worldMapNow", "chooseWorldRecord", "leaveWorldMap", "arriveWorldMap", "tick"}

// CheckWorldMapPage checks production game sources without compiling them.
// The host forwards input; only worldMapPage implements the page contract.
// Campaign location entries remain named boundaries, not map page builders.
func CheckWorldMapPage(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg/game", Reason: "no world-map page sources"}}
	}
	type source struct {
		name string
		file *ast.File
		set  *token.FileSet
	}
	var sources []source
	methods := map[string]map[string]bool{}
	embedded := map[string][]string{}
	interfaces := map[string]bool{}
	aliases := map[string]string{}
	locations := map[string]string{}
	var vs []Violation
	var names []string
	for name := range files {
		if strings.HasPrefix(name, "pkg/game/") && !strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	addMethod := func(typ, method string) {
		if methods[typ] == nil {
			methods[typ] = map[string]bool{}
		}
		methods[typ][method] = true
	}
	for _, name := range names {
		set := token.NewFileSet()
		f, err := parser.ParseFile(set, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		sources = append(sources, source{name, f, set})
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(set.Position(n.Pos()).Line) }
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if typ := receiverName(d.Recv); typ != "" {
					addMethod(typ, d.Name.Name)
					locations[typ] = at(d)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					t, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					locations[t.Name.Name] = at(t)
					switch typ := t.Type.(type) {
					case *ast.InterfaceType:
						interfaces[t.Name.Name] = true
						for _, field := range typ.Methods.List {
							for _, n := range field.Names {
								addMethod(t.Name.Name, n.Name)
							}
							if len(field.Names) == 0 {
								embedded[t.Name.Name] = append(embedded[t.Name.Name], worldMapTypeName(field.Type))
							}
						}
					case *ast.StructType:
						for _, field := range typ.Fields.List {
							if len(field.Names) == 0 {
								embedded[t.Name.Name] = append(embedded[t.Name.Name], worldMapTypeName(field.Type))
							}
						}
					default:
						parent := worldMapTypeName(t.Type)
						embedded[t.Name.Name] = append(embedded[t.Name.Name], parent)
						aliases[t.Name.Name] = parent
					}
				}
			}
		}
	}
	var collect func(string, map[string]bool) map[string]bool
	collect = func(name string, seen map[string]bool) map[string]bool {
		out := map[string]bool{}
		if seen[name] {
			return out
		}
		seen[name] = true
		for m := range methods[name] {
			out[m] = true
		}
		for _, parent := range embedded[name] {
			for m := range collect(parent, seen) {
				out[m] = true
			}
		}
		return out
	}
	sets := map[string]map[string]bool{}
	for typ := range locations {
		sets[typ] = collect(typ, map[string]bool{})
	}
	has := func(set map[string]bool, wanted map[string]bool) bool {
		for m := range wanted {
			if !set[m] {
				return false
			}
		}
		return true
	}
	pageSet := map[string]bool{}
	for _, m := range worldMapPageMethods {
		pageSet[m] = true
	}
	for typ, set := range sets {
		if typ != "worldMapPage" && !interfaces[typ] && has(set, pageSet) {
			vs = append(vs, Violation{From: locations[typ], Reason: "second world-map page: " + typ + "; use worldMapPage"})
		}
	}
	for iface := range interfaces {
		set := sets[iface]
		if len(set) == 0 || !has(pageSet, set) {
			continue
		}
		for typ, methods := range sets {
			if typ != "worldMapPage" && !interfaces[typ] && has(methods, set) {
				vs = append(vs, Violation{From: locations[typ], Reason: "second implementation of world-map page interface " + iface + ": " + typ})
			}
		}
	}
	for _, src := range sources {
		at := func(n ast.Node) string { return src.name + ":" + strconv.Itoa(src.set.Position(n.Pos()).Line) }
		for _, decl := range src.file.Decls {
			if gen, ok := decl.(*ast.GenDecl); ok {
				for _, spec := range gen.Specs {
					if value, ok := spec.(*ast.ValueSpec); ok {
						for _, name := range value.Names {
							if name.Name == "worldMapLocationEntries" {
								vs = append(vs, checkWorldMapLocationEntries(src.name, value, at)...)
							}
						}
					}
				}
			}
			recv, function := "", ""
			if fn, ok := decl.(*ast.FuncDecl); ok {
				recv, function = receiverName(fn.Recv), fn.Name.Name
				if recv != "" && recv != "worldMapPage" && recv != "townScreen" && function != "tick" && pageSet[function] {
					vs = append(vs, Violation{From: at(fn), Reason: "second map page operation " + recv + "." + function})
				}
			}
			builder := recv == "worldMapEntryBuilder" && src.name == "pkg/game/worldmap.go"
			ast.Inspect(decl, func(n ast.Node) bool {
				reason := ""
				switch n := n.(type) {
				case *ast.AssignStmt:
					for i, lhs := range n.Lhs {
						if sel, ok := lhs.(*ast.SelectorExpr); ok && sel.Sel.Name == "missions" && !builder {
							if i < len(n.Rhs) {
								if lit, ok := n.Rhs[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
									continue
								}
							}
							reason = "second map record assignment; use worldMapEntryBuilder.open"
						}
					}
				case *ast.Ident:
					if n.Name == "townMapPoint" && function != "townMapPoint" && !(builder && function == "record") {
						reason = "townMapPoint is used outside worldMapEntryBuilder.record"
					}
				case *ast.CompositeLit:
					typ := worldMapTypeName(n.Type)
					seen := map[string]bool{}
					for aliases[typ] != "" && !seen[typ] {
						seen[typ], typ = true, aliases[typ]
					}
					switch typ {
					case "WorldMapMission":
						if !builder || function != "record" {
							reason = "WorldMapMission record outside worldMapEntryBuilder.record"
						}
					case "worldMapState":
						if !builder || function != "open" {
							reason = "worldMapState outside worldMapEntryBuilder.open"
						}
					case "worldMapPage":
						if recv != "townScreen" || function != "mapPage" || src.name != "pkg/game/townhost.go" {
							reason = "worldMapPage constructed outside townScreen.mapPage"
						}
					}
				case *ast.CallExpr:
					if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "append" && len(n.Args) > 0 {
						if sel, ok := n.Args[0].(*ast.SelectorExpr); ok && sel.Sel.Name == "missions" && !builder {
							reason = "second map record producer; use worldMapEntryBuilder.open"
						}
					}
					if recv == "worldMapPage" {
						if sel, ok := n.Fun.(*ast.SelectorExpr); ok && (sel.Sel.Name == "hostRules" || sel.Sel.Name == "IsSecond" || sel.Sel.Name == "IsFirst") {
							reason = "world-map page selects game identity; use policy rows"
						}
					}
				case *ast.TypeSwitchStmt:
					ast.Inspect(n, func(c ast.Node) bool {
						if clause, ok := c.(*ast.CaseClause); ok {
							for _, typ := range clause.List {
								if name := worldMapTypeName(typ); name == "worldMapPage" || has(sets[name], pageSet) {
									reason = "world-map page type switch; use policy rows"
								}
							}
						}
						return true
					})
				case *ast.BasicLit:
					if (recv == "worldMapPage" || function == "mapPage") && (n.Value == `"rom1"` || n.Value == `"rom2"` || n.Value == `"room-map"`) {
						reason = "world-map page game discriminator; use policy rows"
					}
				}
				if reason != "" {
					vs = append(vs, Violation{From: at(n), Reason: reason})
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

func checkWorldMapLocationEntries(file string, value *ast.ValueSpec, at func(ast.Node) string) []Violation {
	allowed := map[string]string{"town-state": "arriveInTown", "mission-action": "worldMapMissionAction", "location": "enterLocation"}
	seen := map[string]bool{}
	var vs []Violation
	if file != "pkg/game/secondworldmap.go" {
		vs = append(vs, Violation{From: at(value), Reason: "campaign location entry table moved"})
	}
	for _, v := range value.Values {
		lit, ok := v.(*ast.CompositeLit)
		if !ok {
			continue
		}
		for _, e := range lit.Elts {
			pair, ok := e.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := pair.Key.(*ast.BasicLit)
			if !ok {
				continue
			}
			name, _ := strconv.Unquote(key.Value)
			target := allowed[name]
			found := 0
			ast.Inspect(pair.Value, func(n ast.Node) bool {
				if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == target {
					found++
				}
				return true
			})
			if target == "" || seen[name] || found != 1 {
				vs = append(vs, Violation{From: at(pair), Reason: "unnamed campaign location entry " + name})
			}
			seen[name] = true
		}
	}
	for name := range allowed {
		if !seen[name] {
			vs = append(vs, Violation{From: at(value), Reason: "missing named campaign location entry " + name})
		}
	}
	return vs
}

func worldMapTypeName(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return worldMapTypeName(e.X)
	case *ast.SelectorExpr:
		return e.Sel.Name
	}
	return ""
}
