package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strconv"
)

// composerSceneOne names the one composer-scene builder of each kind: a
// pkg/town type with Advance and Paint, a pkg/town interface with Art, a
// pkg/town function answering *Art, and a type elsewhere with Art() *town.Art.
var composerSceneOne = map[string]string{
	"runtime":         "pkg/town:Scene",
	"host interface":  "pkg/town:Host",
	"manifest loader": "pkg/town:LoadArt",
	"adapter":         "pkg/game:townSceneHost",
}

const composerPackage = "pkg/town"

// CheckComposerScene reports each builder beyond composerSceneOne and each
// one missing, over files as LoadDrawnTextSources returns them.
func CheckComposerScene(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the composer-scene scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	found := map[string]map[string]string{}
	add := func(kind, key, at string) {
		if found[kind] == nil {
			found[kind] = map[string]string{}
		}
		if _, ok := found[kind][key]; !ok {
			found[kind][key] = at
		}
	}
	var vs []Violation
	methods := map[string]map[string]string{}
	for _, name := range names {
		dir := path.Dir(name)
		if len(name) < 4 || name[:4] != "pkg/" {
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
				if d.Recv == nil {
					if dir == composerPackage && resultsHold(d.Type.Results, artType) {
						add("manifest loader", dir+":"+d.Name.Name, at(d))
					}
					continue
				}
				recv := receiverName(d.Recv)
				if dir == composerPackage {
					key := dir + ":" + recv
					if methods[key] == nil {
						methods[key] = map[string]string{}
					}
					methods[key][d.Name.Name] = at(d)
					continue
				}
				if d.Name.Name == "Art" && len(d.Type.Params.List) == 0 && resultsHold(d.Type.Results, townArtType) {
					add("adapter", dir+":"+recv, at(d))
				}
			case *ast.GenDecl:
				if dir != composerPackage {
					continue
				}
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					it, ok := ts.Type.(*ast.InterfaceType)
					if !ok {
						continue
					}
					for _, m := range it.Methods.List {
						for _, id := range m.Names {
							if id.Name == "Art" {
								add("host interface", dir+":"+ts.Name.Name, at(ts))
							}
						}
					}
				}
			}
		}
	}
	for key, set := range methods {
		if set["Advance"] != "" && set["Paint"] != "" {
			add("runtime", key, set["Advance"])
		}
	}
	kinds := make([]string, 0, len(composerSceneOne))
	for kind := range composerSceneOne {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	for _, kind := range kinds {
		one := composerSceneOne[kind]
		keys := make([]string, 0, len(found[kind]))
		for key := range found[kind] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if key != one {
				vs = append(vs, Violation{From: found[kind][key],
					Reason: key + " is a second composer-scene " + kind + "; the one is " + one})
			}
		}
		if _, ok := found[kind][one]; !ok {
			vs = append(vs, Violation{From: one, Reason: "the one composer-scene " + kind + " " + one + " is missing"})
		}
	}
	return vs
}

func receiverName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	t := recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func artType(e ast.Expr) bool {
	star, ok := e.(*ast.StarExpr)
	if !ok {
		return false
	}
	id, ok := star.X.(*ast.Ident)
	return ok && id.Name == "Art"
}

func townArtType(e ast.Expr) bool {
	star, ok := e.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Art" {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == "town"
}

func resultsHold(results *ast.FieldList, match func(ast.Expr) bool) bool {
	if results == nil {
		return false
	}
	for _, r := range results.List {
		if match(r.Type) {
			return true
		}
	}
	return false
}
