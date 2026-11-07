package archtest

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"
)

const almPath = "againrom/pkg/formats/alm"

// triggerSlotOffsets are the alm offsets of a trigger record's slots. The
// trigger writer in pkg/mapedit is the one package outside the format that
// writes at them.
var triggerSlotOffsets = map[string]bool{
	"ScriptTriggerPairs": true, "ScriptTriggerActions": true,
	"ScriptTriggerCompare": true, "ScriptTriggerOnce": true,
}

// CheckScriptFraming keeps the type-7 script payload to one walker and the
// trigger record to one writer. Outside pkg/formats/alm no production file
// reads alm.Triggers.Body except for its length, and outside pkg/formats/alm
// and pkg/mapedit none names a trigger slot offset.
func CheckScriptFraming(m *TypeCheckedModule) []string {
	var findings []string
	for path, p := range m.Packages {
		if path == almPath || p.Info == nil {
			continue
		}
		for i, file := range p.Files[:p.NumProd] {
			rel := p.RelPaths[i]
			measured := map[ast.Expr]bool{}
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) != 1 {
					return true
				}
				if id, ok := call.Fun.(*ast.Ident); ok {
					if _, builtin := p.Info.Uses[id].(*types.Builtin); builtin && id.Name == "len" {
						measured[call.Args[0]] = true
					}
				}
				return true
			})
			ast.Inspect(file, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.SelectorExpr:
					if n.Sel.Name == "Body" && !measured[n] && selectsTriggers(p.Info.Selections[n]) {
						findings = append(findings, fmt.Sprintf("%s reads alm.Triggers.Body; the type-7 payload is walked by alm.FrameScript", rel))
					}
				case *ast.Ident:
					c, ok := p.Info.Uses[n].(*types.Const)
					if ok && c.Pkg() != nil && c.Pkg().Path() == almPath && triggerSlotOffsets[c.Name()] && path != "againrom/pkg/mapedit" {
						findings = append(findings, fmt.Sprintf("%s names alm.%s; a trigger record is written by mapedit.NewScriptTrigger", rel, c.Name()))
					}
				}
				return true
			})
		}
	}
	sort.Strings(findings)
	return findings
}

// selectsTriggers reports whether s selects a field of alm.Triggers.
func selectsTriggers(s *types.Selection) bool {
	if s == nil || s.Kind() != types.FieldVal {
		return false
	}
	t := s.Recv()
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	return ok && named.Obj().Name() == "Triggers" && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == almPath
}
