package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"regexp"
	"sort"
	"strings"
)

// The widget kit scan: frames, push buttons and press latches have one
// builder each.

// A FRAME PAINTER is a production function in pkg/ui outside the kit files
// (widget*.go) that is named like one (draw, paint, fill or stamp, then
// Frame, Border, Box, Outline or NinePatch), calls the one-pixel outline
// primitive, or reads a frame art's Pieces. widgetFrameDebt names the
// painters the kit has not absorbed yet; widgetNotAFrame names the matches
// that draw no frame, with the reason.

// A PUSH BUTTON PAINTER is a production function in pkg/ui outside the kit
// that paints a push button's face or press state itself instead of calling
// drawPushButton: it is named like one (draw, paint, fill or stamp, then
// Button), calls the plaque ink or press offset (buttonInk,
// buttonTextOffset), reads a button colour (a name ending in ButtonFill,
// ButtonSelected, ButtonBorder, ButtonFace or ButtonBevel), or picks a
// button picture by its state (an index into an index into a set named
// ...Buttons). A closure counts for the
// function that declares it, so a canvas painter and its image twin are both
// found. widgetButtonDebt names the painters the kit has not absorbed yet;
// widgetNotAButton names the matches that paint no button, with the reason.

// A PRESS LATCH is a struct field in pkg/ui or pkg/render named press or
// ending in Press. Outside the kit and the latch package its type is the kit
// latch (buttonLatch or latch.Latch) unless widgetLatchDebt or
// widgetNotALatch names it.

// A list entry that no longer matches is a violation, so each list only
// falls. The scan reads parsed syntax: it finds painters and latches by name,
// call and field type, not every pixel loop that could draw an edge.

// Every list is keyed by the declaring file and the name, "pkg/ui/x.go:name".

// widgetFrameDebt is every frame painter outside the kit. It may only fall.
var widgetFrameDebt = map[string]string{
	"pkg/ui/shopscreen.go:outline":              "the one-pixel outline primitive the debt painters share",
	"pkg/ui/shopscreen.go:drawShopHover":        "shop characteristics box",
	"pkg/ui/shopscreen.go:drawShopChevron":      "shop chevron box",
	"pkg/ui/townshell.go:drawTownShellBox":      "town shell control box",
	"pkg/ui/townshell.go:drawTownShellOutline":  "town shell control outline",
	"pkg/ui/townshell.go:drawCharacterPaneBody": "character pane fallback outline",
	"pkg/ui/townshell.go:ComposeTownSurface":    "town surface control outlines",
	"pkg/ui/worldmap.go:drawWorldMapBorder":     "world map border",
}

// widgetNotAFrame is every function the frame rule matches that draws no
// frame.
var widgetNotAFrame = map[string]string{
	"pkg/ui/framesmoothing.go:drawFinalFrame":      "presents the finished video frame",
	"pkg/ui/viewer.go:drawFrame":                   "presents one video frame of the map",
	"pkg/ui/dialoguebackdrop.go:drawFrameShadows":  "the dialogue backdrop's GPU twin of the kit's shadow pass",
	"pkg/ui/spellpopup.go:drawAutocastBorder":      "the animated autocast mark around a spell cell",
	"pkg/ui/sackhighlight.go:drawSackOutline":      "an item contour from the outline texture",
	"pkg/ui/dialogueportrait.go:drawFlippedBorder": "the dialogue portrait's art strip, mirrored",
}

// widgetButtonDebt is every push button painter outside the kit. It may only
// fall.
var widgetButtonDebt = map[string]string{
	"pkg/ui/app.go:drawTownButton":               "second-game town list button on the canvas",
	"pkg/ui/townlist.go:composeTownList":         "second-game town list button, the image twin in a closure",
	"pkg/ui/chargen_page.go:drawChargenCommands": "generator Accept, Reset and Back plaques",
	"pkg/ui/chargen_page.go:detailedStatButton":  "generator stat plus and minus pictures by state",
	"pkg/ui/save_dialog_draw.go:saveDialogPaint": "save dialog buttons without the menu font",
	"pkg/ui/shopscreen.go:ComposeShopScreen":     "shop control captions, ink and press offset",
	"pkg/ui/townshell.go:ComposeTownSurface":     "town surface plaques, ink and press offset",
}

// widgetNotAButton is every function the push button rule matches that
// paints no push button.
var widgetNotAButton = map[string]string{
	"pkg/ui/modscreens.go:drawModButton": "draws a mod page button through drawPushButton",
}

// widgetLatchDebt is every press latch outside the kit. It may only fall.
var widgetLatchDebt = map[string]string{
	"pkg/ui/app.go:townSurfacePress":  "town surface controls",
	"pkg/ui/app.go:townTipPress":      "room tip close button",
	"pkg/ui/townpointer.go:tipPress":  "room tip close button, as the town room draws it",
	"pkg/ui/townpointer.go:shopPress": "shop controls",
}

// widgetNotALatch is every field the latch rule matches that is not a push
// button latch, or that holds the kit latch inside a wider record.
var widgetNotALatch = map[string]string{
	"pkg/ui/mapeditor.go:press":     "map editor drag origin, a point",
	"pkg/ui/mapeditormove.go:press": "map editor move origin, a point",
	"pkg/ui/app.go:dialoguePress":   "dialogue pointer capture holding the kit latch",
}

var (
	framePainterName  = regexp.MustCompile(`^(draw|paint|fill|stamp)[A-Za-z]*(Frame|Border|Box|Outline|NinePatch)[A-Za-z]*$`)
	buttonPainterName = regexp.MustCompile(`^(draw|paint|fill|stamp)[A-Za-z]*Button[A-Za-z]*$`)
	buttonColourName  = regexp.MustCompile(`[bB]utton(Fill|Selected|Border|Face|Bevel)$`)
	buttonPictureSet  = regexp.MustCompile(`Buttons$`)
	pressFieldName    = regexp.MustCompile(`^(press|[a-z][A-Za-z]*Press)$`)
)

func widgetKitFile(name string) bool {
	base := path.Base(name)
	return strings.HasPrefix(name, "pkg/render/latch/") ||
		strings.HasPrefix(name, "pkg/ui/") && strings.HasPrefix(base, "widget")
}

// CheckWidgetKit reports every frame painter and press latch outside the kit
// that the lists do not name, and every list entry nothing matches. files is
// keyed by module-relative slash path, as LoadDrawnTextSources returns it.
func CheckWidgetKit(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the widget kit scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	seenFrame, seenButton, seenLatch := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, name := range names {
		inUI := strings.HasPrefix(name, "pkg/ui/")
		inRender := strings.HasPrefix(name, "pkg/render/")
		if !inUI && !inRender || widgetKitFile(name) {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return fmt.Sprintf("%s:%d", name, fset.Position(n.Pos()).Line) }
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncDecl:
				if !inUI {
					return true
				}
				if why := framePainter(n); why != "" {
					key := name + ":" + n.Name.Name
					seenFrame[key] = true
					if widgetFrameDebt[key] == "" && widgetNotAFrame[key] == "" {
						vs = append(vs, Violation{From: at(n), Reason: n.Name.Name + " is a frame painter outside the kit (" + why + "); draw it with drawFrame"})
					}
				}
				if why := buttonPainter(n); why != "" {
					key := name + ":" + n.Name.Name
					seenButton[key] = true
					if widgetButtonDebt[key] == "" && widgetNotAButton[key] == "" {
						vs = append(vs, Violation{From: at(n), Reason: n.Name.Name + " is a push button painter outside the kit (" + why + "); draw it with drawPushButton"})
					}
				}
			case *ast.StructType:
				for _, field := range n.Fields.List {
					vs = append(vs, checkPressField(name, field, at, seenLatch)...)
				}
			}
			return true
		})
	}
	stale := func(list map[string]string, seen map[string]bool, what string) {
		keys := make([]string, 0, len(list))
		for k := range list {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if !seen[k] {
				vs = append(vs, Violation{From: "internal/archtest/widgetkit.go", Reason: k + " is listed as " + what + " but nothing matches it; remove the entry"})
			}
		}
	}
	stale(widgetFrameDebt, seenFrame, "frame debt")
	stale(widgetNotAFrame, seenFrame, "not a frame")
	stale(widgetButtonDebt, seenButton, "button debt")
	stale(widgetNotAButton, seenButton, "not a button")
	stale(widgetLatchDebt, seenLatch, "latch debt")
	stale(widgetNotALatch, seenLatch, "not a latch")
	return vs
}

// framePainter says why fn is a frame painter, or "" when it is not one.
func framePainter(fn *ast.FuncDecl) string {
	if framePainterName.MatchString(fn.Name.Name) {
		return "named as a frame painter"
	}
	if fn.Name.Name == "outline" {
		return "the one-pixel outline primitive"
	}
	why := ""
	if fn.Body != nil {
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "outline" {
					why = "calls outline"
				}
			case *ast.SelectorExpr:
				if n.Sel.Name == "Pieces" {
					why = "reads frame pieces"
				}
			}
			return why == ""
		})
	}
	return why
}

// buttonPainter says why fn paints a push button's face or press state, or
// "" when it does not. A finding inside a closure says so.
func buttonPainter(fn *ast.FuncDecl) string {
	if buttonPainterName.MatchString(fn.Name.Name) {
		return "named as a push button painter"
	}
	if fn.Body == nil {
		return ""
	}
	why := ""
	var walk func(n ast.Node, closure bool)
	walk = func(n ast.Node, closure bool) {
		ast.Inspect(n, func(n ast.Node) bool {
			if why != "" {
				return false
			}
			found := ""
			switch n := n.(type) {
			case *ast.FuncLit:
				if !closure {
					walk(n.Body, true)
					return false
				}
			case *ast.KeyValueExpr:
				// A composite literal's key names a field; only its value is read.
				walk(n.Value, closure)
				return false
			case *ast.CallExpr:
				if id, ok := n.Fun.(*ast.Ident); ok && (id.Name == "buttonInk" || id.Name == "buttonTextOffset") {
					found = "calls " + id.Name
				}
			case *ast.SelectorExpr:
				if buttonColourName.MatchString(n.Sel.Name) {
					found = "reads the button colour " + n.Sel.Name
				}
			case *ast.Ident:
				if buttonColourName.MatchString(n.Name) {
					found = "reads the button colour " + n.Name
				}
			case *ast.IndexExpr:
				if inner, ok := n.X.(*ast.IndexExpr); ok && buttonPictureSet.MatchString(exprName(inner.X)) {
					found = "picks a button picture by state from " + exprName(inner.X)
				}
			}
			if found != "" {
				if closure {
					found += " in a closure"
				}
				why = found
			}
			return why == ""
		})
	}
	walk(fn.Body, false)
	return why
}

// exprName is the name an identifier or selector ends in, or "".
func exprName(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	}
	return ""
}

// checkPressField reports a struct field named as a press latch that is not
// the kit latch and no list names.
func checkPressField(name string, field *ast.Field, at func(ast.Node) string, seen map[string]bool) []Violation {
	var vs []Violation
	for _, id := range field.Names {
		if !pressFieldName.MatchString(id.Name) || kitLatchType(field.Type) {
			continue
		}
		key := name + ":" + id.Name
		seen[key] = true
		if widgetLatchDebt[key] == "" && widgetNotALatch[key] == "" {
			vs = append(vs, Violation{From: at(id), Reason: id.Name + " is a press latch outside the kit; make it a buttonLatch"})
		}
	}
	return vs
}

func kitLatchType(e ast.Expr) bool {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name == "buttonLatch"
	case *ast.SelectorExpr:
		x, ok := t.X.(*ast.Ident)
		return ok && x.Name == "latch" && t.Sel.Name == "Latch"
	}
	return false
}
