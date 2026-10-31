package archtest

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"
	"strconv"
)

const (
	uiPackage         = "againrom/pkg/ui"
	menuPlacementFile = "pkg/ui/missionmenu.go"
)

// menuPlacementTypes are the menu window placement and its operation types,
// all declared in menuPlacementFile; menuPlacementOps are the row's fields a
// surface may only call an operation on.
var (
	menuPlacementTypes = map[string]bool{"menuWindowPlacement": true, "menuWindowHost": true, "menuWindowAnchor": true,
		"menuUnderlay": true, "menuBackdrop": true, "menuTransform": true, "menuTip": true}
	menuPlacementOps = map[string]bool{"underlay": true, "backdrop": true, "transform": true, "tipOnPanel": true}
)

// CheckMenuPlacement reports, in pkg/ui production files outside
// missionmenu.go, each way a surface decides by where a menu window is
// placed: a call of a placement method returning bool, a reference to the
// placement's anchor, centred or host row, a placement constant or host
// table, a read of NativeMissionWindows, a call of menuWindowShown, an
// operation field used other than to call its operation, and an if statement
// whose condition or init calls something returning a placement. It reads
// go/types answers, so another type's anchor field does not count.
func CheckMenuPlacement(m *TypeCheckedModule) []string {
	p := m.Packages[uiPackage]
	if p == nil {
		return []string{uiPackage + " is not in the module; the placement scan guards nothing"}
	}
	var out []string
	for i, file := range p.Files[:p.NumProd] {
		rel := p.RelPaths[i]
		if rel == menuPlacementFile {
			continue
		}
		at := func(n ast.Node) string { return rel + ":" + strconv.Itoa(m.Fset.Position(n.Pos()).Line) }
		var stack []ast.Node
		ast.Inspect(file, func(n ast.Node) bool {
			if n == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			var parent ast.Node
			if len(stack) > 0 {
				parent = stack[len(stack)-1]
			}
			stack = append(stack, n)
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if f := menuPlacementSelection(p.Info, x, parent, stack); f != "" {
					out = append(out, at(x)+": "+f)
				}
			case *ast.Ident:
				if f := menuPlacementName(p.Info, x); f != "" {
					out = append(out, at(x)+": "+f)
				}
			case *ast.IfStmt:
				for _, part := range []ast.Node{x.Init, x.Cond} {
					if part != nil && callsPlacement(p.Info, part) {
						out = append(out, at(x)+": an if statement asks a menu placement; call the placement's operation instead")
						break
					}
				}
			}
			return true
		})
	}
	sort.Strings(out)
	return out
}

// menuPlacementSelection names the placement shape selector x makes, or "".
func menuPlacementSelection(info *types.Info, x *ast.SelectorExpr, parent ast.Node, stack []ast.Node) string {
	sel := info.Selections[x]
	if sel == nil {
		return ""
	}
	recv := namedUI(sel.Recv())
	switch sel.Kind() {
	case types.MethodVal, types.MethodExpr:
		if recv == "flow" && x.Sel.Name == "menuWindowShown" {
			return "calls menuWindowShown; whether a window stands over the mission is the placement's"
		}
		if !menuPlacementTypes[recv] {
			return ""
		}
		if sig, ok := sel.Obj().Type().(*types.Signature); ok && sig.Results().Len() == 1 && isBool(sig.Results().At(0).Type()) {
			return fmt.Sprintf("calls %s.%s, a placement answer in bool; call the operation that does the work", recv, x.Sel.Name)
		}
	case types.FieldVal:
		if recv == "GameMenuProfile" && x.Sel.Name == "NativeMissionWindows" {
			return "reads NativeMissionWindows; only the placement builder reads it"
		}
		if !menuPlacementTypes[recv] {
			return ""
		}
		switch name := x.Sel.Name; {
		case name == "anchor" || name == "centred":
			return "reads the placement's " + name + "; call the placement's operation instead"
		case name == "menuWindowHost":
			return "reads the placement's host row; call the placement's operation instead"
		case menuPlacementOps[name]:
			if call, ok := grandparent(stack).(*ast.CallExpr); ok && call.Fun == parent {
				if _, ok := parent.(*ast.SelectorExpr); ok {
					return ""
				}
			}
			return "uses the placement's " + name + " other than to call its operation"
		}
	}
	return ""
}

// menuPlacementName names a placement constant or host table id refers to.
func menuPlacementName(info *types.Info, id *ast.Ident) string {
	obj := info.Uses[id]
	if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != uiPackage || obj.Parent() != obj.Pkg().Scope() {
		return ""
	}
	t := namedUI(obj.Type())
	switch obj.(type) {
	case *types.Const:
		if menuPlacementTypes[t] {
			return "names the placement constant " + id.Name + "; call the placement's operation instead"
		}
	case *types.Var:
		if t == "menuWindowHost" {
			return "names the host row " + id.Name + "; the placement builder alone picks a row"
		}
	}
	return ""
}

// callsPlacement reports a call in n whose result is a menu placement.
func callsPlacement(info *types.Info, n ast.Node) bool {
	found := false
	ast.Inspect(n, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok && namedUI(info.TypeOf(call)) == "menuWindowPlacement" {
			found = true
		}
		return !found
	})
	return found
}

// namedUI is the name of t, or of the type t points to, when it is declared
// in pkg/ui; "" otherwise.
func namedUI(t types.Type) string {
	if t == nil {
		return ""
	}
	t = types.Unalias(t)
	if ptr, ok := t.(*types.Pointer); ok {
		t = types.Unalias(ptr.Elem())
	}
	named, ok := t.(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != uiPackage {
		return ""
	}
	return named.Obj().Name()
}

func isBool(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Kind() == types.Bool
}

func grandparent(stack []ast.Node) ast.Node {
	if len(stack) < 3 {
		return nil
	}
	return stack[len(stack)-3]
}
