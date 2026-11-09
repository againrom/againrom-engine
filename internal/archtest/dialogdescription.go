package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// dialogDescription is the one type that describes a modal dialog.
const dialogDescription = "popupSpec"

// retiredDialogTypes are the description types merged into it.
var retiredDialogTypes = map[string]bool{"NoticeLayout": true, "noticeWindow": true, "NoticeShape": true}

// retiredDialogPainters are the painters drawPopup replaced.
var retiredDialogPainters = map[string]bool{
	"renderDialogueNotice": true, "drawNoticeWindow": true, "dialogueButton": true, "drawHelpScrollbar": true,
}

// dialogPainter is the one function that paints a dialog's frame and buttons.
const dialogPainter = "drawPopup"

// DialogDescriptionSites lists the places in production code of pkg/ui and
// pkg/game that are a second dialog description or a second dialog painter: a
// retired type or painter declared again, another struct holding both a Box
// and a *DialogFrame Frame, a comparison against a NoticeStyle constant (a
// style is a row of data, not a branch), and a function that draws a frame
// and pushes a button itself. A function in dialogPainterBaseline is not
// reported while it still does that; an entry that no longer does is, so the
// list can only fall.
func DialogDescriptionSites(root string) ([]string, error) {
	var sites []string
	seen := map[string]bool{}
	for _, dir := range []string{"pkg/ui", "pkg/game"} {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			rel := dir + "/" + name
			src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
			if err != nil {
				return nil, err
			}
			sites = append(sites, dialogDescriptionInSource(rel, src, seen)...)
		}
	}
	for key := range dialogPainterBaseline {
		if !seen[key] {
			sites = append(sites, key+" is in dialogPainterBaseline but no longer draws a frame and a button; drop the entry")
		}
	}
	sort.Strings(sites)
	return sites, nil
}

func dialogDescriptionInSource(name string, src []byte, seen map[string]bool) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
	if err != nil {
		return []string{name + " does not parse: " + err.Error()}
	}
	var hits []string
	add := func(pos token.Pos, what string) {
		hits = append(hits, name+":"+strconv.Itoa(fset.Position(pos).Line)+" "+what)
	}
	isUI := strings.HasPrefix(name, "pkg/ui/")
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !isUI {
					continue
				}
				if retiredDialogTypes[ts.Name.Name] {
					add(ts.Pos(), "declares "+ts.Name.Name+", a dialog description beside "+dialogDescription)
				}
				if st, ok := ts.Type.(*ast.StructType); ok && ts.Name.Name != dialogDescription && holdsBoxAndFrame(st) {
					add(ts.Pos(), "declares "+ts.Name.Name+" with a Box and a DialogFrame, a dialog description beside "+dialogDescription)
				}
			}
		case *ast.FuncDecl:
			if isUI && retiredDialogPainters[d.Name.Name] {
				add(d.Pos(), "declares "+d.Name.Name+", a dialog painter beside "+dialogPainter)
			}
			if d.Body == nil {
				continue
			}
			if isUI && d.Name.Name != dialogPainter && paintsFrameAndButton(d.Body) {
				key := name + " " + d.Name.Name
				if _, known := dialogPainterBaseline[key]; known {
					if seen != nil {
						seen[key] = true
					}
				} else {
					add(d.Pos(), d.Name.Name+" draws a frame and pushes a button itself; make it a value "+dialogPainter+" paints")
				}
			}
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BinaryExpr:
			if (n.Op == token.EQL || n.Op == token.NEQ) && (namesNoticeStyle(n.X) || namesNoticeStyle(n.Y)) {
				add(n.Pos(), "branches on a notice style; read a row of noticeStyles")
			}
		case *ast.CaseClause:
			for _, x := range n.List {
				if namesNoticeStyle(x) {
					add(n.Pos(), "branches on a notice style; read a row of noticeStyles")
				}
			}
		}
		return true
	})
	return hits
}

// holdsBoxAndFrame reports a struct with a field Box and a field Frame of
// type *DialogFrame.
func holdsBoxAndFrame(st *ast.StructType) bool {
	box, frame := false, false
	for _, f := range st.Fields.List {
		for _, nm := range f.Names {
			if nm.Name == "Box" {
				box = true
			}
			if star, ok := f.Type.(*ast.StarExpr); ok && nm.Name == "Frame" {
				if id, ok := star.X.(*ast.Ident); ok && id.Name == "DialogFrame" {
					frame = true
				}
			}
		}
	}
	return box && frame
}

// paintsFrameAndButton reports a body that calls a frame drawer and the push
// button drawer.
func paintsFrameAndButton(body *ast.BlockStmt) bool {
	frame, button := false, false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if id, ok := call.Fun.(*ast.Ident); ok {
			switch id.Name {
			case "drawFrame", "drawFrameBody":
				frame = true
			case "drawPushButton":
				button = true
			}
		}
		return true
	})
	return frame && button
}

func namesNoticeStyle(x ast.Expr) bool {
	switch v := x.(type) {
	case *ast.Ident:
		return strings.HasPrefix(v.Name, "NoticeStyle")
	case *ast.SelectorExpr:
		return strings.HasPrefix(v.Sel.Name, "NoticeStyle")
	}
	return false
}

// dialogPainterBaseline names, with the reason, the functions outside the
// one painter that draw a frame and push a button: parts of that painter, a
// widget, and windows the dialog census did not take in. It may only fall.
var dialogPainterBaseline = map[string]string{
	"pkg/ui/editorpopup.go drawPopupGrid":             "the grid part of the one painter",
	"pkg/ui/widgetdropdown.go drawDropdown":           "a widget's open list",
	"pkg/ui/mapeditorwindow.go chromeCompose":         "the map editor's window chrome",
	"pkg/ui/load_draw.go composePage":                 "the load page, outside the dialog census",
	"pkg/ui/gameoptions_draw.go gameOptionsPicture":   "the game options window, outside the dialog census",
	"pkg/ui/soundoptions_draw.go soundOptionsPicture": "the sound options window, outside the dialog census",
	"pkg/ui/media_menu.go composeCutsceneLibrary":     "the cutscene library, outside the dialog census",
	"pkg/ui/questobjectives.go questPicture":          "the objectives window, outside the dialog census",
	"pkg/ui/tippanel.go ComposeTipPanel":              "the tip panel, outside the dialog census",
}
