package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The tavern talk text scan: tavernTalkTextPath in pkg/game decides which
// text a tavern row speaks, its mission's or the chapter's, and the town
// screen and the towns layer's row check both call it. A production call of
// TownNPCTextPath anywhere else under pkg/ makes that choice a second time;
// isTavernTalkLeaf, which re-forms a path to recognise it, is the one other
// caller.
var tavernTalkTextCallers = map[string]bool{"tavernTalkTextPath": true, "isTavernTalkLeaf": true}

// CheckTavernTalkText reports every production call of TownNPCTextPath
// outside its two callers. files is keyed by module-relative slash path, as
// LoadDrawnTextSources returns it.
func CheckTavernTalkText(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the tavern talk text scan found nothing to read"}}
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
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || tavernTalkTextCallers[fn.Name.Name] {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok && schoolCallName(call.Fun) == "TownNPCTextPath" {
					vs = append(vs, Violation{From: name + ":" + strconv.Itoa(fset.Position(call.Pos()).Line),
						Reason: fn.Name.Name + " calls TownNPCTextPath; a tavern row's talk text is chosen by tavernTalkTextPath"})
				}
				return true
			})
		}
	}
	return vs
}
