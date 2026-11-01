package ui

import (
	"image"
	"image/draw"
)

// The toolbar: one row across the window between the top bar and the map.
// toolButtons is its one table and toolbarEntries its one builder: the left
// group holds the tools, one of them current, the right group the layer
// switches, the other switches and the actions. A button performs its row key
// through menuAct, the one path a menu row takes.

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

// toolButton is one toolbar button as data: the menu row key it performs, its icon, its tip's word, the key row whose label the
// tip names, its group and kind, a switch's state and when it is disabled.
type toolButton struct {
	key    string
	icon   *editorIcon
	word   string
	keyRow string
	group  toolGroup
	kind   toolKind
	on     func(e *MapEditor) bool
	off    func(e *MapEditor) bool
}

// The toolbar's geometry: a button's side, the gaps between buttons and
// between the groups, and the row's padding above and below its buttons.
const (
	toolSide     = 22
	toolGap      = 2
	toolGroupGap = 8
	toolbarPad   = 3
	toolbarHigh  = toolSide + 2*toolbarPad
)

var toolButtons []toolButton

func init() {
	noMap := func(e *MapEditor) bool { return e.doc == nil }
	noBrush := func(e *MapEditor) bool { return e.brushHost() == nil || e.catalog }
	toolButtons = []toolButton{
		{key: "select", icon: &iconSelect, word: "editor.tool.select", kind: toolPick, off: noMap},
		{key: brushMenuKeys[brushTerrain], icon: &iconTerrain, word: "editor.tool.terrain", keyRow: "terrain", kind: toolPick, off: noBrush},
		{key: brushMenuKeys[brushHeight], icon: &iconHeight, word: "editor.tool.height", keyRow: "height", kind: toolPick, off: noBrush},
		{key: brushMenuKeys[brushObjects], icon: &iconObjects, word: "editor.tool.objects", keyRow: "objects", kind: toolPick, off: noBrush},
		{key: brushMenuKeys[brushPassability], icon: &iconPassability, word: "editor.tool.passability", keyRow: "passability", kind: toolPick, off: noBrush},
		{key: "drawarea", icon: &iconDrawArea, word: "editor.tool.drawarea", keyRow: "drawarea", kind: toolPick, off: func(e *MapEditor) bool { return !e.editable() }},
		{key: "border", icon: &iconBorder, word: "editor.tool.border", group: toolRight, kind: toolSwitch, on: func(e *MapEditor) bool { return !e.view.hideBorder }, off: noMap},
		{key: "areas", icon: &iconAreas, word: "editor.tool.areas", group: toolRight, kind: toolSwitch, on: func(e *MapEditor) bool { return !e.view.hideAreas }, off: noMap},
		{key: "drops", icon: &iconDrops, word: "editor.tool.drops", group: toolRight, kind: toolSwitch, on: func(e *MapEditor) bool { return !e.view.hideDrops }, off: noMap},
		{key: "footprints", icon: &iconFootprints, word: "editor.tool.footprints", group: toolRight, kind: toolSwitch, on: func(e *MapEditor) bool { return !e.view.hideFootprints }, off: noMap},
		{key: "issues", icon: &iconIssues, word: "editor.tool.issues", group: toolRight, kind: toolSwitch, on: func(e *MapEditor) bool { return !e.view.hideIssues }, off: noMap},
		{key: "heightlabels", icon: &iconHeightLabels, word: "editor.tool.heightlabels", keyRow: "heightlabels", group: toolRight, kind: toolSwitch, on: func(e *MapEditor) bool { return e.view.heights }, off: noMap},
		{key: "passview", icon: &iconPassView, word: "editor.tool.passview", keyRow: "passview", group: toolRight, kind: toolSwitch, on: func(e *MapEditor) bool { return e.view.passability }, off: noMap},
		{key: "playertint", icon: &iconPlayerTint, word: "editor.tool.playertint", keyRow: "playertint", group: toolRight, kind: toolSwitch, on: func(e *MapEditor) bool { return e.view.playerColours }, off: noMap},
		{key: "grid", icon: &iconGrid, word: "editor.tool.grid", group: toolRight, kind: toolSwitch, on: func(e *MapEditor) bool { return e.view.grid }, off: noMap},
		{key: "fit", icon: &iconFit, word: "editor.tool.fit", keyRow: "fit", group: toolRight, kind: toolAction, off: noMap},
		{key: "report", icon: &iconReport, word: "editor.tool.report", keyRow: "report", group: toolRight, kind: toolSwitch,
			on: (*MapEditor).checkShown,
			off: func(e *MapEditor) bool {
				_, ok := e.checker()
				return !ok && !e.checkShown()
			}},
		{key: sidePanels[panelLeft].key, icon: &iconObjectsPanel, word: "editor.tool.objectspanel", keyRow: sidePanels[panelLeft].key, group: toolRight, kind: toolSwitch, on: func(e *MapEditor) bool { return e.panelOpen(panelLeft) }},
		{key: sidePanels[panelRight].key, icon: &iconPropertiesPanel, word: "editor.tool.propertiespanel", keyRow: sidePanels[panelRight].key, group: toolRight, kind: toolSwitch, on: func(e *MapEditor) bool { return e.panelOpen(panelRight) }},
	}
}

// toolEntry is one toolbar button as the window shows it: its row, its
// rectangle, and whether it shows on and disabled.
type toolEntry struct {
	toolButton
	rect          image.Rectangle
	shown, closed bool
}

// toolbarRect is the toolbar's row: the window's width under the top bar.
func (e *MapEditor) toolbarRect() image.Rectangle {
	top := e.barHeight()
	return image.Rect(0, top, e.window.X, top+toolbarHigh)
}

// currentTool is the key of the left group's tool that is current: the
// draw-area tool while it is armed, the brush that owns the canvas, else
// select.
func (e *MapEditor) currentTool() string {
	switch {
	case e.pick == pickArea:
		return "drawarea"
	case e.brushActive():
		return brushMenuKeys[e.brush.mode]
	}
	return "select"
}

// toolbarEntries is the toolbar's buttons in table order, placed and with
// their states: the left group from the window's left edge, the right group
// against its right edge and never over the left one.
func (e *MapEditor) toolbarEntries() []toolEntry {
	bar := e.toolbarRect()
	y := bar.Min.Y + toolbarPad
	out := make([]toolEntry, len(toolButtons))
	place := func(group toolGroup, x int) int {
		for i, b := range toolButtons {
			if b.group == group {
				out[i].rect = image.Rect(x, y, x+toolSide, y+toolSide)
				x += toolSide + toolGap
			}
		}
		return x
	}
	right := 0
	for _, b := range toolButtons {
		if b.group == toolRight {
			right += toolSide + toolGap
		}
	}
	left := place(toolLeft, 4) - toolGap + toolGroupGap
	place(toolRight, max(left, bar.Max.X-4-right+toolGap))
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
// it.
func (e *MapEditor) ToolbarRect(key string) (image.Rectangle, bool) {
	for _, t := range e.toolbarEntries() {
		if t.key == key {
			return t.rect, true
		}
	}
	return image.Rectangle{}, false
}

// ToolbarState reports whether toolbar button key shows on and whether it is
// disabled.
func (e *MapEditor) ToolbarState(key string) (on, disabled, ok bool) {
	for _, t := range e.toolbarEntries() {
		if t.key == key {
			return t.shown, t.closed, true
		}
	}
	return false, false, false
}

// toolbarPress is a press at p on the toolbar: a button that is not disabled
// performs its row key.
func (e *MapEditor) toolbarPress(p image.Point) {
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
		tip := e.word(t.word)
		if r := editorKeyByName[t.keyRow]; r != nil && r.label != "" {
			tip += " (" + r.label + ")"
		}
		return tip, t.rect.Min, true
	}
	return "", image.Point{}, false
}

// drawToolbar draws the toolbar row: the bar's fill and gold rule, each
// button through the one icon button painter.
func (e *MapEditor) drawToolbar(pic *image.RGBA) {
	bar := e.toolbarRect()
	draw.Draw(pic, bar, image.NewUniform(editorPanel), image.Point{}, draw.Src)
	draw.Draw(pic, image.Rect(bar.Min.X, bar.Max.Y-1, bar.Max.X, bar.Max.Y), image.NewUniform(editorGold), image.Point{}, draw.Src)
	for _, t := range e.toolbarEntries() {
		drawIconButton(pic, iconButton{Rect: t.rect, Icon: t.icon, On: t.shown, Hover: e.pointerOK && e.pointer.In(t.rect), Disabled: t.closed})
	}
}
