package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The control layout scan: a control's rectangle is a controlLayout row that
// the draw reads and the hit test asks. It refuses three shapes:
//   - a pkg/ui or pkg/game function returning a rectangle from a switch whose
//     cases return rectangle literals, unless listed as a remaining producer;
//   - on the save surfaces, a hit test whose argument is a rectangle literal,
//     a package rectangle variable or a page field, not a layout row;
//   - on the save surfaces, a package rectangle variable or a rectangle
//     struct that the lists below do not name (a second descriptor shape).

// controlLayoutSurface reports the files whose controls all read the descriptor.
func controlLayoutSurface(name string) bool {
	if !strings.HasPrefix(name, "pkg/ui/") || strings.HasSuffix(name, "_test.go") {
		return false
	}
	base := strings.TrimPrefix(name, "pkg/ui/")
	return strings.HasPrefix(base, "save_dialog") || strings.HasPrefix(base, "savepage") ||
		base == "load_window.go" || base == "load_draw.go" || base == "controllayout.go" ||
		base == "tippanel.go" || base == "goldmodal.go" || base == "documents.go" || base == "chargen_page.go"
}

// controlHitFuncsRemaining ratchets the functions in pkg/ui and pkg/game that
// hold a hit test outside the surfaces above. It may only fall: a surface
// that moves to controlLayout lowers it in the same commit.
const controlHitFuncsRemaining = 162

// tipPlacers are the functions that place a tip box in its page; each asks
// placeTip and holds no page edge test of its own.
var tipPlacers = map[string]bool{
	"pkg/ui/widgethover.go:placeHover": true,
	"pkg/ui/editorpopup.go:popupTipAt": true,
}

// controlSwitchProducers are the functions that still return a rectangle
// from a switch; each is a producer not yet moved to a descriptor. The list
// may only shrink.
var controlSwitchProducers = map[string]string{
	"pkg/ui/characterpane.go:characterPaneArtRect":          "remaining producer: the character pane corner art's blit rectangles",
	"pkg/ui/mapeditorsidepanels.go:MapEditor.sidePanelRect": "the side panel's builder, unchanged by decision",
}

// controlSurfaceRects names the package rectangle variables on the save
// surfaces. None is a control: they are panels, argument rectangles and
// message boxes.
var controlSurfaceRects = map[string]string{
	"pkg/ui/save_dialog_draw.go:saveDialogLayout": "the dialog's descriptor rows",
	"pkg/ui/save_dialog_draw.go:savePanelRect":    "the dialog's panel body",
	"pkg/ui/save_dialog_draw.go:saveListArg":      "the list box's argument, the list control less its bar",
	"pkg/ui/load_window.go:loadPanel":             "the first game's load page panel",
	"pkg/ui/load_window.go:loadListArg":           "the first game's load list argument",
	"pkg/ui/load_window.go:loadMessageBox":        "the first game's load message line",
	"pkg/ui/documents.go:docControls":             "the documents panel's descriptor rows",
	"pkg/ui/documents.go:docSheetRect":            "the documents panel's sheet, not a control",
}

// controlSurfaceTypes names the struct types on the save surfaces that hold a
// rectangle: the descriptor row, and the page data whose button and name box
// rectangles the descriptor reads.
var controlSurfaceTypes = map[string]string{
	"controlRow":   "the descriptor row",
	"SavePage":     "page data: the layout reads its button and name box rectangles",
	"PageText":     "a caption box, not a control",
	"PageButton":   "page data: a button row of the page",
	"PageConfirm":  "the confirmation's label box, not a control",
	"pageView":     "no rectangle field of its own",
	"TipPanelView": "the panel's own rectangle; its controls are rows placed from it",
}

// CheckControlLayout reports every control rectangle shape outside the
// descriptor. files is keyed by module-relative slash path.
func CheckControlLayout(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg/ui", Reason: "no sources scanned - the control layout scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	usedProducer := map[string]bool{}
	usedRect := map[string]bool{}
	usedPlacer := map[string]bool{}
	hitFuncs := 0
	for _, name := range names {
		if !(strings.HasPrefix(name, "pkg/ui/") || strings.HasPrefix(name, "pkg/game/")) || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(p token.Pos) string { return name + ":" + strconv.Itoa(fset.Position(p).Line) }
		surface := controlLayoutSurface(name)
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Body == nil {
					continue
				}
				fn := cadenceFuncName(d)
				if !surface && holdsHitTest(d.Body) {
					hitFuncs++
				}
				if tipPlacers[name+":"+fn] {
					usedPlacer[name+":"+fn] = true
					if what := tipPlacerFault(d.Body); what != "" {
						vs = append(vs, Violation{From: at(d.Pos()), Reason: fn + " " + what + "; a tip takes its place from placeTip"})
					}
				}
				if returnsRect(d) && switchReturnsRects(d.Body) {
					if controlSwitchProducers[name+":"+fn] != "" {
						usedProducer[name+":"+fn] = true
					} else {
						vs = append(vs, Violation{From: at(d.Pos()), Reason: fn + " returns a rectangle from a switch over control identities; make the controls rows of a controlLayout"})
					}
				}
				if surface {
					ast.Inspect(d.Body, func(n ast.Node) bool {
						call, ok := n.(*ast.CallExpr)
						if !ok {
							return true
						}
						sel, ok := call.Fun.(*ast.SelectorExpr)
						if !ok || (sel.Sel.Name != "In" && sel.Sel.Name != "Overlaps") || len(call.Args) != 1 {
							return true
						}
						if what := ownRect(call.Args[0]); what != "" {
							vs = append(vs, Violation{From: at(call.Pos()), Reason: "a hit test in " + fn + " reads " + what + ", not a controlLayout row"})
						}
						return true
					})
				}
			case *ast.GenDecl:
				if !surface {
					continue
				}
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.ValueSpec:
						for i, id := range s.Names {
							if i < len(s.Values) && mentionsRectCall(s.Values[i]) {
								key := name + ":" + id.Name
								if controlSurfaceRects[key] == "" {
									vs = append(vs, Violation{From: at(id.Pos()), Reason: id.Name + " holds a rectangle in a package variable; control rectangles are controlLayout rows"})
								} else {
									usedRect[key] = true
								}
							}
						}
					case *ast.TypeSpec:
						st, ok := s.Type.(*ast.StructType)
						if ok && structHoldsRect(st) && controlSurfaceTypes[s.Name.Name] == "" {
							vs = append(vs, Violation{From: at(s.Pos()), Reason: s.Name.Name + " is a second descriptor shape: a struct holding a rectangle; use controlLayout"})
						}
					}
				}
			}
		}
	}
	if _, whole := files["pkg/ui/controllayout.go"]; whole {
		for k := range tipPlacers {
			if !usedPlacer[k] {
				vs = append(vs, Violation{From: "internal/archtest/controllayout.go", Reason: k + " is listed as a tip placer but nothing there needs it; remove the entry"})
			}
		}
		if hitFuncs > controlHitFuncsRemaining {
			vs = append(vs, Violation{From: "internal/archtest/controllayout.go", Reason: "functions with a hit test outside the controlLayout surfaces rose to " + strconv.Itoa(hitFuncs) + " from " + strconv.Itoa(controlHitFuncsRemaining) + "; read a controlLayout row"})
		}
		if hitFuncs < controlHitFuncsRemaining {
			vs = append(vs, Violation{From: "internal/archtest/controllayout.go", Reason: "functions with a hit test outside the controlLayout surfaces fell to " + strconv.Itoa(hitFuncs) + "; lower controlHitFuncsRemaining from " + strconv.Itoa(controlHitFuncsRemaining)})
		}
	}
	for _, k := range sortedKeys(controlSwitchProducers) {
		if !usedProducer[k] {
			vs = append(vs, Violation{From: "internal/archtest/controllayout.go", Reason: k + " is listed as a remaining producer but nothing there needs it; remove the entry"})
		}
	}
	for _, k := range sortedKeys(controlSurfaceRects) {
		if !usedRect[k] {
			vs = append(vs, Violation{From: "internal/archtest/controllayout.go", Reason: k + " is listed as a non-control rectangle but nothing there needs it; remove the entry"})
		}
	}
	return vs
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// holdsHitTest reports a one-argument .In or .Overlaps call in body.
func holdsHitTest(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok && len(call.Args) == 1 {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && (sel.Sel.Name == "In" || sel.Sel.Name == "Overlaps") {
				found = true
			}
		}
		return true
	})
	return found
}

// tipPlacerFault names what keeps a tip placer from placeTip: no call to it,
// or a bound's Max read that an edge test would need.
func tipPlacerFault(body *ast.BlockStmt) string {
	calls, edge := false, false
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "placeTip" {
				calls = true
			}
		case *ast.SelectorExpr:
			if n.Sel.Name == "Max" {
				switch x := n.X.(type) {
				case *ast.Ident:
					edge = edge || x.Name == "bounds"
				case *ast.SelectorExpr:
					edge = edge || x.Sel.Name == "bounds"
				}
			}
		}
		return true
	})
	switch {
	case !calls:
		return "does not call placeTip"
	case edge:
		return "tests a page edge itself"
	}
	return ""
}

func returnsRect(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil {
		return false
	}
	for _, r := range fn.Type.Results.List {
		if sel, ok := r.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Rectangle" {
			return true
		}
	}
	return false
}

// switchReturnsRects reports a switch with two or more cases that each
// return an image.Rect call.
func switchReturnsRects(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		var clauses []ast.Stmt
		switch s := n.(type) {
		case *ast.SwitchStmt:
			clauses = s.Body.List
		case *ast.TypeSwitchStmt:
			clauses = s.Body.List
		default:
			return true
		}
		count := 0
		for _, c := range clauses {
			cc, ok := c.(*ast.CaseClause)
			if !ok {
				continue
			}
			for _, st := range cc.Body {
				if ret, ok := st.(*ast.ReturnStmt); ok {
					for _, r := range ret.Results {
						if mentionsRectCall(r) {
							count++
						}
					}
				}
			}
		}
		if count >= 2 {
			found = true
		}
		return true
	})
	return found
}

// mentionsRectCall reports an image.Rect call or image.Rectangle literal in e.
func mentionsRectCall(e ast.Expr) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Rect" {
				if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "image" {
					found = true
				}
			}
		case *ast.CompositeLit:
			if sel, ok := n.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Rectangle" {
				found = true
			}
		}
		return true
	})
	return found
}

// ownRect names the rectangle a hit test's argument reads when it is not a
// layout row: a literal, a package rectangle variable or a page field.
func ownRect(e ast.Expr) string {
	if mentionsRectCall(e) {
		return "a rectangle literal"
	}
	switch e := e.(type) {
	case *ast.Ident:
		if strings.HasSuffix(e.Name, "Rect") || strings.HasSuffix(e.Name, "Box") || strings.HasSuffix(e.Name, "Arg") {
			return "the variable " + e.Name
		}
	case *ast.SelectorExpr:
		switch e.Sel.Name {
		case "NameBox", "List", "Message", "Panel":
			return "the page field " + e.Sel.Name
		}
	case *ast.IndexExpr:
		if sel, ok := e.X.(*ast.SelectorExpr); ok && sel.Sel.Name == "Buttons" {
			return "the page's button row"
		}
	}
	return ""
}

func structHoldsRect(st *ast.StructType) bool {
	for _, f := range st.Fields.List {
		t := f.Type
		if arr, ok := t.(*ast.ArrayType); ok {
			t = arr.Elt
		}
		if sel, ok := t.(*ast.SelectorExpr); ok && sel.Sel.Name == "Rectangle" {
			return true
		}
	}
	return false
}
