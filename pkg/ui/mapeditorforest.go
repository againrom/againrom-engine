package ui

import (
	"fmt"
	"image"
	"strconv"
	"strings"
)

// The Forest mode of the object brush scatters the kinds ticked in the
// picture grid over the cells the brush covers, at the chosen density: a
// left stroke plants on open cells, a right stroke thins the ticked kinds.
// The host decides which cells are open; one stroke is one edit.

// BrushForest is the forest form of an object stroke: the ticked class codes,
// the share of covered cells a stroke acts on, in per cent, and Thin for a
// stroke that removes the ticked kinds instead of planting them.
type BrushForest struct {
	Kinds []int
	Share int
	Thin  bool
}

// InspectionBrushIcons is the optional seam of a host that has a picture of
// each object class for the forest grid.
type InspectionBrushIcons interface {
	BrushObjectIcon(code int) *image.RGBA
}

// forestShares are the density presets in per cent: sparse, medium, dense
// and solid. The default is medium.
var forestShares = [4]int{10, 25, 50, 100}

// forestGridRows is how many rows of the forest grid show.
const forestGridRows = 3

// forestState is the Forest mode's settings, kept for the session: whether
// it is on, the ticked class codes, the typed or preset share (0 is the
// default), the grid's first shown row, the pictures read so far, and the
// stroke in progress being a thinning one.
type forestState struct {
	on    bool
	ticks map[int]bool
	share int
	top   int
	icons map[int]*image.RGBA
	thin  bool
	right bool // the right button was held at the last pointer event
}

// forestOn reports whether the object brush is in its Forest mode.
func (e *MapEditor) forestOn() bool {
	return e.brush.mode == brushObjects && e.brush.forest.on
}

// forestShare is the share a forest stroke takes.
func (e *MapEditor) forestShare() int {
	if s := e.brush.forest.share; s >= 1 && s <= 100 {
		return s
	}
	return forestShares[1]
}

// forestKinds are the forest grid's classes: every offered class but the
// clear row, in the list's order. A host that has pictures offers only the
// classes whose picture shows something.
func (e *MapEditor) forestKinds() []BrushObject {
	h := e.brushHost()
	if h == nil {
		return nil
	}
	var out []BrushObject
	for _, o := range h.BrushObjects() {
		if o.Code != 0 && e.forestHasArt(o.Code) {
			out = append(out, o)
		}
	}
	return out
}

// forestHasArt reports whether class code may stand in the grid: always for a
// host without pictures, else only when its picture has a visible pixel.
func (e *MapEditor) forestHasArt(code int) bool {
	if _, ok := e.brushHost().(InspectionBrushIcons); !ok {
		return true
	}
	pic := e.forestIcon(code)
	if pic == nil || pic.Bounds().Empty() {
		return false
	}
	for i := 3; i < len(pic.Pix); i += 4 {
		if pic.Pix[i] != 0 {
			return true
		}
	}
	return false
}

// forestTicked are the ticked class codes in the list's order.
func (e *MapEditor) forestTicked() []int {
	var out []int
	for _, o := range e.forestKinds() {
		if e.brush.forest.ticks[o.Code] {
			out = append(out, o.Code)
		}
	}
	return out
}

// forestTick switches the tick of class code.
func (e *MapEditor) forestTick(code int) {
	if e.brush.forest.ticks == nil {
		e.brush.forest.ticks = map[int]bool{}
	}
	e.brush.forest.ticks[code] = !e.brush.forest.ticks[code]
	if !e.brush.forest.ticks[code] {
		delete(e.brush.forest.ticks, code)
	}
	e.message = ""
}

// forestIcon is the host's picture of class code, read once.
func (e *MapEditor) forestIcon(code int) *image.RGBA {
	if pic, ok := e.brush.forest.icons[code]; ok {
		return pic
	}
	h, ok := e.brushHost().(InspectionBrushIcons)
	if !ok {
		return nil
	}
	if e.brush.forest.icons == nil {
		e.brush.forest.icons = map[int]*image.RGBA{}
	}
	pic := h.BrushObjectIcon(code)
	e.brush.forest.icons[code] = pic
	return pic
}

// forestGridSpec is the forest grid as data: one cell per class, ticked
// where the class is in the mix.
func (e *MapEditor) forestGridSpec() (*popupGrid, bool) {
	if !e.forestOn() {
		return nil, false
	}
	kinds := e.forestKinds()
	if len(kinds) == 0 {
		return nil, false
	}
	g := &popupGrid{Rows: forestGridRows, Top: e.brush.forest.top}
	for _, o := range kinds {
		g.Cells = append(g.Cells, popupGridCell{Icon: e.forestIcon(o.Code), Ticked: e.brush.forest.ticks[o.Code], Tip: e.objectWord(o)})
	}
	return g, true
}

// forestGridHeight is the room the forest grid takes in the list popup.
func (e *MapEditor) forestGridHeight() int {
	if !e.forestOn() {
		return 0
	}
	return (&popupGrid{Rows: forestGridRows}).height() + fontHeight(e.bodyFont(), 15) + popupRowExtra
}

// forestGridWheel scrolls the forest grid by the wheel's rows.
func (e *MapEditor) forestGridWheel(l popupLayout, rows int) {
	if l.GridCols <= 0 {
		return
	}
	total := (len(e.forestKinds()) + l.GridCols - 1) / l.GridCols
	e.brush.forest.top = max(0, min(e.brush.forest.top+rows, total-forestGridRows))
}

// forestGridPress is a press on forest grid cell n: it ticks or unticks the
// class.
func (e *MapEditor) forestGridPress(n int) {
	if kinds := e.forestKinds(); n >= 0 && n < len(kinds) {
		e.forestTick(kinds[n].Code)
	}
}

// forestShareWord names the share: the preset's word and its per cent, or
// the per cent alone for a typed share.
func (e *MapEditor) forestShareWord() string {
	s := e.forestShare()
	for i, p := range forestShares {
		if p == s {
			return fmt.Sprintf("%s %d%%", e.forestPresetWord(i), s)
		}
	}
	return fmt.Sprintf("%d%%", s)
}

// forestPresetWord is the word of density preset i.
func (e *MapEditor) forestPresetWord(i int) string {
	switch i {
	case 0:
		return e.word("editor.brush.density.sparse")
	case 1:
		return e.word("editor.brush.density.medium")
	case 2:
		return e.word("editor.brush.density.dense")
	}
	return e.word("editor.brush.density.solid")
}

// forestPopupRows are the object brush's rows below the shape: the tool
// with its One kind and Forest buttons and, in Forest mode, the density with
// the button that asks for a typed per cent, a row of the button that clears
// the marks, off while none is marked, and the preset buttons, and the hint.
func (e *MapEditor) forestPopupRows() []popupRow {
	tool := e.word("editor.brush.tool.one")
	if e.brush.forest.on {
		tool = e.word("editor.brush.tool.forest")
	}
	rows := []popupRow{{Kind: popupText, Label: e.word("editor.brush.tool"), Value: tool,
		Buttons: []string{e.word("editor.brush.tool.one"), e.word("editor.brush.tool.forest")}}}
	if !e.brush.forest.on {
		return rows
	}
	buttons := make([]string, 0, len(forestShares)+1)
	buttons = append(buttons, e.word("editor.brush.forest.clear"))
	for i := range forestShares {
		buttons = append(buttons, e.forestPresetWord(i))
	}
	rows = append(rows,
		popupRow{Kind: popupText, Label: e.word("editor.brush.density"), Value: e.forestShareWord(), Buttons: []string{"%"}},
		popupRow{Kind: popupText, Buttons: buttons, Disabled: []bool{len(e.forestTicked()) == 0}},
		popupRow{Kind: popupText, Label: editorClip(e.bodyFont(), e.word("editor.brush.forest.hint"), 480)})
	return rows
}

// forestRowButton is a press on button k of the tool row (row 2), the
// density row's per cent button (row 3) or the presets row (row 4), whose
// first button unticks every kind in the grid and the list at once.
func (e *MapEditor) forestRowButton(row, k int) {
	switch {
	case row == 2:
		e.brush.forest.on = k == 1
	case row == 3:
		e.openPrompt(editorPrompt{kind: promptForestShare, path: strconv.Itoa(e.forestShare())})
	case row == 4 && k == 0:
		e.brush.forest.ticks, e.message = nil, ""
	case row == 4 && k <= len(forestShares):
		e.brush.forest.share = forestShares[k-1]
	}
}

// setForestShare takes the typed per cent; anything but a whole number from
// 1 to 100 leaves the share as it was and says why.
func (e *MapEditor) setForestShare(text string) {
	n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(text), "%"))
	if err != nil || n < 1 || n > 100 {
		e.leave(leavePrompt)
		e.message = fmt.Sprintf(e.word("editor.brush.share.bad"), strings.TrimSpace(text))
		return
	}
	e.brush.forest.share = n
	e.leave(leavePrompt)
	e.message = ""
}

// forestStroke is the stroke's forest settings, or false with the reason in
// the message when no kind is ticked.
func (e *MapEditor) forestStroke(thin bool) (BrushForest, bool) {
	kinds := e.forestTicked()
	if len(kinds) == 0 {
		e.message = e.word("editor.brush.forest.none")
		return BrushForest{}, false
	}
	return BrushForest{Kinds: kinds, Share: e.forestShare(), Thin: thin}, true
}
