package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

var townHostSeams = []string{"Header", "Rows", "Footer", "Choose", "Back"}
var townOwnedSeams = []string{"TownMusic", "TownMusicTrack", "TownSquareView", "sceneArt", "sceneNow", "sceneDraw", "sceneSeed", "sceneCondition", "sceneValue", "scenePlay", "sceneStartLoop", "sceneStopLoop", "sceneLeave", "sceneHook", "sceneDialogue"}

// CheckTownHost keeps town input, scenes, music and the square on one host.
func CheckTownHost(files map[string]string) []Violation {
	methods := map[string]map[string]bool{}
	var vs []Violation
	for name, source := range files {
		if !strings.HasPrefix(name, "pkg/game/") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: err.Error()})
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range st.Fields.List {
				if len(field.Names) != 0 {
					continue
				}
				typ := field.Type
				if p, ok := typ.(*ast.StarExpr); ok {
					typ = p.X
				}
				embeds := false
				if id, ok := typ.(*ast.Ident); ok {
					embeds = id.Name == "townScreen"
				}
				if s, ok := typ.(*ast.SelectorExpr); ok {
					if id, ok := s.X.(*ast.Ident); ok {
						embeds = id.Name == "ui" && s.Sel.Name == "TownScreen"
					}
				}
				if embeds {
					vs = append(vs, Violation{From: name + ":" + ts.Name.Name, Reason: "town host methods must not be promoted onto another type"})
				}
			}
			return true
		})
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil {
				continue
			}
			typ := fn.Recv.List[0].Type
			if p, ok := typ.(*ast.StarExpr); ok {
				typ = p.X
			}
			id, ok := typ.(*ast.Ident)
			if !ok {
				continue
			}
			if methods[id.Name] == nil {
				methods[id.Name] = map[string]bool{}
			}
			methods[id.Name][fn.Name.Name] = true
		}
	}
	has := func(typ string, names []string) bool {
		for _, name := range names {
			if !methods[typ][name] {
				return false
			}
		}
		return true
	}
	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if name != "townScreen" && has(name, townHostSeams) {
			vs = append(vs, Violation{From: "pkg/game:" + name, Reason: "town input has a second screen host"})
		}
		if name != "townRoomView" && has(name, []string{"Size", "Paint", "ControlAt", "TipAt"}) {
			vs = append(vs, Violation{From: "pkg/game:" + name, Reason: "town square has a second scene view"})
		}
		for _, seam := range townOwnedSeams {
			if name != "townScreen" && methods[name][seam] {
				vs = append(vs, Violation{From: "pkg/game:" + name + "." + seam, Reason: "town seam belongs to townScreen"})
			}
		}
	}
	for _, seam := range append(append([]string{}, townHostSeams...), townOwnedSeams...) {
		if !methods["townScreen"][seam] {
			vs = append(vs, Violation{From: "pkg/game:townScreen", Reason: "missing town seam " + seam})
		}
	}
	if !has("townRoomView", []string{"Size", "Paint", "ControlAt", "TipAt"}) {
		vs = append(vs, Violation{From: "pkg/game:townRoomView", Reason: "missing town square scene view"})
	}
	return vs
}
