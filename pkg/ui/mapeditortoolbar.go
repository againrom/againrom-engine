package ui

import (
	"image"
)

// The toolbar: one row across the window between the top bar and the map,
// or a floating window. toolButtons is its one table and toolbarEntries its
// one builder: the left group holds the tools, one of them current, the
// right group the layer switches, the other switches and the actions; each
// button's cell places it in the floating window's grid. A button performs
// its row key through menuAct, the one path a menu row takes. A drag from the
// grip at the row's left end floats the toolbar, and the window's close
// button docks it again. The same table gives the bar's Tools, View and
// Window menus their rows; a row with no icon stands in a menu only.

// toolKind is what a toolbar button is.
type toolKind uint8

const (
	toolPick   toolKind = iota // a tool: the current one shows on
	toolSwitch                 // a switch: on while its state is
	toolAction                 // a press does one thing
)

// toolGroup is the toolbar group a button stands in.
type toolGroup uint8

const (
	toolLeft  toolGroup = iota // from the window's left edge
	toolRight                  // against the window's right edge
)

// toolButton is one editor command as data: the menu row key it performs, its icon (nil for a row with no toolbar button),
// its tip's word, the key row whose label the tip names, its group and kind, its column and row in the floating window, a
// switch's state and when it is disabled; the bar menu whose row it is, and the word of a heading over it in that menu;
// whether a rule row stands over it; a row with no icon names its menu word as its word.
type toolButton struct {
	key         string
	icon        *editorIcon
	word        string
	keyRow      string
	group       toolGroup
	kind        toolKind
	cell        image.Point
	on          func(e *MapEditor) bool
	off         func(e *MapEditor) bool
	menu        string
	headingWord string
	ruleBefore  bool
}

// onToolbar reports that the row has a toolbar button.
func (b toolButton) onToolbar() bool { return b.icon != nil }

// toolSize is one size of the toolbar's buttons: the scale their icons draw
// at and a button's side.
type toolSize struct{ scale, side int }

// toolSizes are the toolbar's sizes, largest first: each icon pixel a 2 by 2
// block where the toolbar fits that way, else one screen pixel.
var toolSizes = [...]toolSize{{scale: 2, side: 36}, {scale: 1, side: 22}}

// The toolbar's geometry: the gaps between buttons and between the groups,
// the row's padding above and below its buttons, its inset at each end and
// the grip's width.
const (
	toolGap      = 2
	toolGroupGap = 8
	toolbarPad   = 3
	toolInset    = 4
	toolGripW    = 8
)

var toolButtons []toolButton

func init() {
	noMap := func(e *MapEditor) bool { return e.doc == nil }
	noBrush := func(e *MapEditor) bool { return e.brushHost() == nil || e.catalog }
	noEdit := func(e *MapEditor) bool { return !e.editable() }
	noRandom := func(e *MapEditor) bool { return e.doc == nil || e.catalog }
	toolButtons = []toolButton{
		{key: "new", word: "editor.menu.new", kind: toolAction, menu: "file"},
		{key: "open", word: "editor.menu.open", kind: toolAction, menu: "file"},
		{key: "saveas", word: "editor.menu.save", kind: toolAction, off: noEdit, menu: "file"},
		{key: "saveto", word: "editor.menu.saveas", kind: toolAction, off: noEdit, menu: "file"},
		{key: "catalog", word: "editor.menu.builtin", kind: toolAction, menu: "file"},
		{key: "testplay", word: "editor.menu.testplay", kind: toolAction, off: noEdit, menu: "file"},
		{key: "undo", word: "editor.bar.undo", keyRow: "undo", kind: toolAction, off: func(e *MapEditor) bool { return !e.editable() || !e.doc.Edits.CanUndo() }, menu: "file"},
		{key: "redo", word: "editor.bar.redo", keyRow: "redo", kind: toolAction, off: func(e *MapEditor) bool { return !e.editable() || !e.doc.Edits.CanRedo() }, menu: "file"},
		{key: searchKey, word: "editor.bar.search", kind: toolAction, off: noMap, menu: "file"},
		{key: "records", word: "editor.view.records", kind: toolAction, off: noMap, menu: "file"},
		{key: "close", word: "editor.menu.exit", ruleBefore: true, kind: toolAction, menu: "file"},
		{key: "select", icon: &iconSelect, word: "editor.tool.select", kind: toolPick, cell: image.Pt(1, 0), off: noMap, menu: "tools"},
		{key: brushMenuKeys[brushTerrain], icon: &iconTerrain, word: "editor.tool.terrain", keyRow: "terrain", kind: toolPick, cell: image.Pt(1, 1), off: noBrush, menu: "tools", headingWord: "editor.tool.brushes"},
		{key: brushMenuKeys[brushHeight], icon: &iconHeight, word: "editor.tool.height", keyRow: "height", kind: toolPick, cell: image.Pt(1, 2), off: noBrush, menu: "tools"},
		{key: brushMenuKeys[brushObjects], icon: &iconObjects, word: "editor.tool.objects", keyRow: "objects", kind: toolPick, cell: image.Pt(1, 3), off: noBrush, menu: "tools"},
		{key: brushMenuKeys[brushPassability], icon: &iconPassability, word: "editor.tool.passability", keyRow: "passability", kind: toolPick, cell: image.Pt(1, 4), off: noBrush, menu: "tools"},
		{key: brushMenuKeys[brushBridge], icon: &iconBridge, word: "editor.tool.bridge", kind: toolPick, cell: image.Pt(1, 6), off: noBrush, menu: "tools"},
		{key: "drawarea", icon: &iconDrawArea, word: "editor.tool.drawarea", keyRow: "drawarea", kind: toolPick, cell: image.Pt(1, 5), off: func(e *MapEditor) bool { return !e.editable() }, menu: "tools"},
		{key: "newtrigger", word: "editor.tool.newtrigger", keyRow: "newtrigger", kind: toolAction, off: func(e *MapEditor) bool { return !e.addNowOpen(KindTrigger) }, menu: "tools"},
		{key: "traparea", word: "editor.cast.area", kind: toolPick, off: noEdit, menu: "tools"},
		{key: "randomterrain", word: "editor.tools.randomterrain", headingWord: "editor.menu.generate", kind: toolAction, off: noRandom, menu: "tools"},
		{key: "randomheights", word: "editor.tools.randomheights", kind: toolAction, off: noRandom, menu: "tools"},
		{key: "border", icon: &iconBorder, word: "editor.tool.border", group: toolRight, kind: toolSwitch, cell: image.Pt(0, 0), on: func(e *MapEditor) bool { return !e.view.hideBorder }, off: noMap, menu: "view"},
		{key: "areas", icon: &iconAreas, word: "editor.tool.areas", group: toolRight, kind: toolSwitch, cell: image.Pt(0, 1), on: func(e *MapEditor) bool { return !e.view.hideAreas }, off: noMap, menu: "view"},
		{key: "drops", icon: &iconDrops, word: "editor.tool.drops", group: toolRight, kind: toolSwitch, cell: image.Pt(0, 2), on: func(e *MapEditor) bool { return !e.view.hideDrops }, off: noMap, menu: "view"},
		{key: "footprints", icon: &iconFootprints, word: "editor.tool.footprints", group: toolRight, kind: toolSwitch, cell: image.Pt(0, 3), on: func(e *MapEditor) bool { return !e.view.hideFootprints }, off: noMap, menu: "view"},
		{key: "issues", icon: &iconIssues, word: "editor.tool.issues", group: toolRight, kind: toolSwitch, cell: image.Pt(0, 4), on: func(e *MapEditor) bool { return !e.view.hideIssues }, off: noMap, menu: "view"},
		{key: "heightlabels", icon: &iconHeightLabels, word: "editor.tool.heightlabels", keyRow: "heightlabels", group: toolRight, kind: toolSwitch, cell: image.Pt(0, 5), on: func(e *MapEditor) bool { return e.view.heights }, off: noMap, menu: "view"},
		{key: "passview", icon: &iconPassView, word: "editor.tool.passview", keyRow: "passview", group: toolRight, kind: toolSwitch, cell: image.Pt(0, 6), on: func(e *MapEditor) bool { return e.view.passability }, off: noMap, menu: "view"},
		{key: "playertint", icon: &iconPlayerTint, word: "editor.tool.playertint", keyRow: "playertint", group: toolRight, kind: toolSwitch, cell: image.Pt(0, 7), on: func(e *MapEditor) bool { return e.view.playerColours }, off: noMap, menu: "view"},
		{key: "grid", icon: &iconGrid, word: "editor.tool.grid", group: toolRight, kind: toolSwitch, cell: image.Pt(0, 8), on: func(e *MapEditor) bool { return e.view.grid }, off: noMap, menu: "view"},
		{key: "fit", icon: &iconFit, word: "editor.tool.fit", keyRow: "fit", group: toolRight, kind: toolAction, cell: image.Pt(0, 9), off: noMap, menu: "view"},
		{key: "report", icon: &iconReport, word: "editor.tool.report", keyRow: "report", group: toolRight, kind: toolSwitch, cell: image.Pt(0, 10),
			on: (*MapEditor).checkShown,
			off: func(e *MapEditor) bool {
				_, ok := e.checker()
				return !ok && !e.checkShown()
			}, menu: "view"},
		{key: "toolbar", word: "editor.tool.toolbar", group: toolRight, kind: toolSwitch, on: (*MapEditor).toolbarOn, menu: "window"},
		{key: sidePanels[panelLeft].key, icon: &iconObjectsPanel, word: "editor.tool.objectspanel", keyRow: sidePanels[panelLeft].key, group: toolRight, kind: toolSwitch, cell: image.Pt(0, 11), on: func(e *MapEditor) bool { return e.panelOn(panelLeft) }, menu: "window"},
		{key: sidePanels[panelRight].key, icon: &iconPropertiesPanel, word: "editor.tool.propertiespanel", keyRow: sidePanels[panelRight].key, group: toolRight, kind: toolSwitch, cell: image.Pt(0, 12), on: func(e *MapEditor) bool { return e.panelOn(panelRight) }, menu: "window"},
		{key: sidePanels[panelBottom].key, word: "editor.tool.reportpanel", keyRow: sidePanels[panelBottom].key, group: toolRight, kind: toolSwitch, on: func(e *MapEditor) bool { return e.panelOn(panelBottom) }, menu: "window"},
	}
}

// toolEntry is one toolbar button as the window shows it: its row, its
// rectangle, and whether it shows on and disabled.
type toolEntry struct {
	toolButton
	rect          image.Rectangle
	shown, closed bool
}

// toolbarFloats reports that the toolbar stands as a floating window.
func (e *MapEditor) toolbarFloats() bool { return e.panels.floats[popToolbar] }

// toolbarOn reports that the toolbar shows, docked or floating.
func (e *MapEditor) toolbarOn() bool { return !e.panels.toolbarHidden }

// toggleToolbar hides the toolbar, or shows it again where it stood: in its
// row or as its floating window at the placement store's corner.
func (e *MapEditor) toggleToolbar() {
	e.panels.toolbarHidden = !e.panels.toolbarHidden
	e.relayout()
	e.savePanels()
}

// toolbarRect is the toolbar's row: the window's width under the top bar,
// as high as its buttons and padding; no height while the toolbar floats or
// is hidden.
func (e *MapEditor) toolbarRect() image.Rectangle {
	top, high := e.barHeight(), 0
	if !e.toolbarFloats() && e.toolbarOn() {
		high = e.toolSizeNow().side + 2*toolbarPad
	}
	return image.Rect(0, top, e.window.X, top+high)
}

// toolSizeNow is the size the toolbar's buttons take: the largest whose
// docked row fits the window's width, or whose floating window fits the
// height floating windows stand in; else the smallest.
func (e *MapEditor) toolSizeNow() toolSize {
	last := toolSizes[len(toolSizes)-1]
	for _, z := range toolSizes[:len(toolSizes)-1] {
		if e.toolbarFloats() && e.toolFloatSize(z).Y <= e.popupBounds().Dy() || !e.toolbarFloats() && toolRowWidth(z) <= e.window.X {
			return z
		}
	}
	return last
}

// toolRowWidth is the width the docked row takes at size z: the grip, the
// left group, the gap between the groups and the right group, inset at each
// end.
func toolRowWidth(z toolSize) int {
	var n [2]int
	for _, b := range toolButtons {
		if b.onToolbar() {
			n[b.group]++
		}
	}
	run := func(k int) int { return k*z.side + (k-1)*toolGap }
	return 2*toolInset + toolGripW + toolGap + run(n[toolLeft]) + toolGroupGap + run(n[toolRight])
}

// toolGrid is the floating window's columns and rows: one past each
// button's greatest cell.
func toolGrid() image.Point {
	var g image.Point
	for _, b := range toolButtons {
		if b.onToolbar() {
			g = image.Pt(max(g.X, b.cell.X+1), max(g.Y, b.cell.Y+1))
		}
	}
	return g
}

// toolFloatSize is the floating window's outer size at size z: the popup
// builder's title row as its drag strip, over the button grid with popupPad
// around it.
func (e *MapEditor) toolFloatSize(z toolSize) image.Point {
	g := toolGrid()
	strip := layoutPopup(popupSpec{}, e.font, e.bodyFont()).Title.Dy()
	return image.Pt(2*popupPad+g.X*z.side+(g.X-1)*toolGap, strip+2*popupPad+g.Y*z.side+(g.Y-1)*toolGap)
}

// toolbarSpec is the floating toolbar's window: a drag strip whose button
// docks the toolbar, over the buttons, standing in its row of the placement
// store at the size its buttons give; none while the toolbar is hidden.
func (e *MapEditor) toolbarSpec() (popupSpec, bool) {
	if !e.toolbarFloats() || !e.toolbarOn() {
		return popupSpec{}, false
	}
	return e.contentSizedSpec(popupSpec{}, popToolbar, e.toolFloatSize(e.toolSizeNow())), true
}

// toolbarGrip is the docked row's grip at its left end; empty while the
// toolbar floats or is hidden.
func (e *MapEditor) toolbarGrip() image.Rectangle {
	if e.toolbarFloats() || !e.toolbarOn() {
		return image.Rectangle{}
	}
	y := e.toolbarRect().Min.Y + toolbarPad
	return image.Rect(toolInset, y, toolInset+toolGripW, y+e.toolSizeNow().side)
}

// undockToolbar turns the toolbar into a floating window whose drag strip
// holds the pointer at p: its row then shows nothing.
func (e *MapEditor) undockToolbar(p image.Point) {
	e.panels.floats[popToolbar] = true
	e.relayout()
	at := p.Sub(image.Pt(popupPad+toolGripW/2, popupPad))
	e.floatPlace(popToolbar, image.Rectangle{Min: at, Max: at.Add(e.toolFloatSize(e.toolSizeNow()))})
	e.placePopup(popToolbar, at)
}

// dockToolbar puts the floating toolbar back in its row.
func (e *MapEditor) dockToolbar() {
	e.panels.floats[popToolbar] = false
	e.relayout()
	e.savePanels()
}

// currentTool is the key of the left group's tool that is current: the
// draw-area tool while it is armed, the brush that owns the canvas, else
// select.
func (e *MapEditor) currentTool() string {
	switch {
	case e.pick == pickArea:
		return "drawarea"
	case e.pick == pickTrapArea:
		return "traparea"
	case e.brushActive():
		return brushMenuKeys[e.brush.mode]
	}
	return "select"
}

// toolbarEntries is the toolbar's buttons in table order, placed and with
// their states. Docked, the left group stands after the grip, the right
// group against the window's right edge and never over the left one;
// floating, each button stands at its cell of the window's grid. A row with
// no button, and every row while the toolbar is hidden, has no rectangle;
// its state is the menu row's.
func (e *MapEditor) toolbarEntries() []toolEntry {
	side := e.toolSizeNow().side
	out := make([]toolEntry, len(toolButtons))
	if s, ok := e.toolbarSpec(); ok {
		l := layoutPopup(s, e.font, e.bodyFont())
		origin := image.Pt(l.Rect.Min.X+popupPad, l.Title.Max.Y+popupPad)
		for i, b := range toolButtons {
			if b.onToolbar() {
				at := origin.Add(b.cell.Mul(side + toolGap))
				out[i].rect = image.Rectangle{Min: at, Max: at.Add(image.Pt(side, side))}
			}
		}
	} else if e.toolbarOn() {
		bar := e.toolbarRect()
		y := bar.Min.Y + toolbarPad
		place := func(group toolGroup, x int) int {
			for i, b := range toolButtons {
				if b.onToolbar() && b.group == group {
					out[i].rect = image.Rect(x, y, x+side, y+side)
					x += side + toolGap
				}
			}
			return x
		}
		right := 0
		for _, b := range toolButtons {
			if b.onToolbar() && b.group == toolRight {
				right += side + toolGap
			}
		}
		left := place(toolLeft, toolInset+toolGripW+toolGap) - toolGap + toolGroupGap
		place(toolRight, max(left, bar.Max.X-toolInset-right+toolGap))
	}
	tool := e.currentTool()
	for i, b := range toolButtons {
		t := &out[i]
		t.toolButton = b
		t.closed = b.off != nil && b.off(e)
		switch b.kind {
		case toolPick:
			t.shown = !t.closed && tool == b.key
		case toolSwitch:
			t.shown = b.on(e)
		}
	}
	return out
}

// ToolbarRect is the window rectangle of the toolbar button key ("select",
// "brush.terrain", "border", "grid" and the rest): a headless driver presses
// it. A menu-only row and a hidden toolbar have none.
func (e *MapEditor) ToolbarRect(key string) (image.Rectangle, bool) {
	for _, t := range e.toolbarEntries() {
		if t.key == key && !t.rect.Empty() {
			return t.rect, true
		}
	}
	return image.Rectangle{}, false
}

// ToolbarState reports whether the row key, a toolbar button or a menu-only
// row, shows on and whether it is disabled.
func (e *MapEditor) ToolbarState(key string) (on, disabled, ok bool) {
	for _, t := range e.toolbarEntries() {
		if t.key == key {
			return t.shown, t.closed, true
		}
	}
	return false, false, false
}

// toolbarPress is a press at p on the toolbar: the floating window's close
// button docks it, and a button that is not disabled performs its row key.
func (e *MapEditor) toolbarPress(p image.Point) {
	if s, ok := e.toolbarSpec(); ok {
		if part, _ := layoutPopup(s, e.font, e.bodyFont()).hit(p); part == popupClose {
			e.dockToolbar()
			return
		}
	}
	for _, t := range e.toolbarEntries() {
		if !p.In(t.rect) || t.closed {
			continue
		}
		e.menuAct(t.key)
		return
	}
}

// toolTip is the hover tip of the toolbar button under p, and the button's
// corner: its word, and the key that does the same, when it has one.
func (e *MapEditor) toolTip(p image.Point) (string, image.Point, bool) {
	for _, t := range e.toolbarEntries() {
		if !p.In(t.rect) {
			continue
		}
		return e.toolWord(t.toolButton), t.rect.Min, true
	}
	return "", image.Point{}, false
}

// toolWord is a row's word as its tip and its menu row read it: the word,
// then the key that does the same in brackets, when it has one.
func (e *MapEditor) toolWord(b toolButton) string {
	w := e.word(b.word)
	if r := editorKeyByName[b.keyRow]; r != nil && r.label != "" {
		w += " (" + r.label + ")"
	}
	return w
}

// drawToolbar draws the toolbar when floating says where it stands: the
// docked row's fill, gold rule and grip, or the floating window's chrome
// through the one popup builder; then each button through the one icon
// button painter. A hidden toolbar draws nothing.
func (e *MapEditor) drawToolbar(pic *image.RGBA, floating bool) {
	if floating != e.toolbarFloats() || !e.toolbarOn() {
		return
	}
	scale := e.toolSizeNow().scale
	if s, ok := e.toolbarSpec(); ok {
		e.drawWindow(pic, s)
	} else {
		drawFrame(pic, stripFrame(e.toolbarRect(), edgeBottomBit))
		drawIcon(pic, e.toolbarGrip(), &iconGrip, false, scale)
	}
	for _, t := range e.toolbarEntries() {
		if !t.onToolbar() {
			continue
		}
		drawIconButton(pic, iconButton{Rect: t.rect, Icon: t.icon, On: t.shown, Hover: e.pointerOK && e.pointer.In(t.rect), Disabled: t.closed, Scale: scale, Press: &e.buttonPress})
	}
}
