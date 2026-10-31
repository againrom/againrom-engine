package ui

import (
	"encoding/json"
	"fmt"
	"image"
	"strings"

	"againrom/pkg/editormcp"

	"github.com/hajimehoshi/ebiten/v2"
)

// The window's two side panels. Each is a row of sidePanels and one builder,
// sidePanelRect, places either. The left panel shows the tree of the map's
// objects and scenarios, the right one the selected record's window. A
// window in a panel is the same popup a floating window is: inPanel gives it
// the panel's rectangle as its place, its size and its bounds.

// popupPlace is where a popup stands: floating over the map or in a panel.
type popupPlace uint8

const (
	placeFloating popupPlace = iota
	placeLeft
	placeRight
)

// The side panels' rows in sidePanels.
const (
	panelLeft  = 0
	panelRight = 1
)

// sidePanel is one side panel as data: its place, its name in the profile
// file, its default, least and greatest widths, the word of its title, and
// its key row's name, help label and keyboard key.
type sidePanel struct {
	place              popupPlace
	name               string
	width, least, most int
	titleWord          string
	key, label         string
	hotkey             ebiten.Key
}

var sidePanels = [2]sidePanel{
	{place: placeLeft, name: "left", width: 260, least: 180, most: 420, titleWord: "editor.side.objects", key: "objectspanel", label: "F6", hotkey: ebiten.KeyF6},
	{place: placeRight, name: "right", width: 520, least: 360, most: 800, titleWord: "editor.side.properties", key: "propertiespanel", label: "F7", hotkey: ebiten.KeyF7},
}

// mapLeastWidth is the least width the map keeps beside the panels.
const mapLeastWidth = 480

// PanelStateFile is the editor profile file that keeps the panels' state.
const PanelStateFile = "mapedit-panels.json"

// WritePanelState writes the panel state to file through the writer of the
// profile's MCP record.
func WritePanelState(file string, b []byte) error {
	return editormcp.WriteProfileFile(file, ".mapedit-panels-*.tmp", "panel state", b)
}

// panelState is a panel's kept state: open or closed, and its width.
type panelState struct {
	Open  bool
	Width int
}

// editorPanels is the side panels' state. It stays outside editorChrome, so
// a cancelled window does not take a panel's size back.
type editorPanels struct {
	state [2]panelState
	save  func([]byte) error
	// hosted reports that the last object popup built stands in the right
	// panel; hostable that the one being built is a window the panel takes.
	hosted, hostable bool
	// detached is the record, plus one, whose window a double click on the
	// panel's title row took out to float.
	detached int
}

// clampPanel is width held between panel i's least and greatest widths; a
// width of zero is the panel's default.
func clampPanel(i, width int) int {
	p := sidePanels[i]
	if width <= 0 {
		return p.width
	}
	return max(p.least, min(p.most, width))
}

// panelsShown is each panel's shown width, zero for a panel that is closed
// or that the window has no room for. The right panel gives way first, then
// the left, so the map keeps mapLeastWidth.
func (e *MapEditor) panelsShown() [2]int {
	var w [2]int
	for i := range sidePanels {
		if s := e.panels.state[i]; s.Open {
			w[i] = clampPanel(i, s.Width)
		}
	}
	room := e.window.X - mapLeastWidth
	if w[panelLeft]+w[panelRight] > room {
		w[panelRight] = 0
	}
	if w[panelLeft] > room {
		w[panelLeft] = 0
	}
	return w
}

// sidePanelRect is panel i's window rectangle, between the bar and the
// status strip; empty while the panel is not shown.
func (e *MapEditor) sidePanelRect(i int) image.Rectangle {
	w := e.panelsShown()[i]
	if w == 0 {
		return image.Rectangle{}
	}
	top := e.barHeight()
	bottom := e.mapBottom(top)
	if sidePanels[i].place == placeLeft {
		return image.Rect(0, top, w, bottom)
	}
	return image.Rect(e.window.X-w, top, e.window.X, bottom)
}

// inPanel places popup s in panel i: the one popup builder then lays it out
// to fill the panel.
func (e *MapEditor) inPanel(s popupSpec, i int) popupSpec {
	r := e.sidePanelRect(i)
	s.At, s.Size, s.Bounds, s.MinWidth, s.Slack = r.Min, r.Size(), r, 1, false
	return s
}

// mapOrigin is the window position of the map frame's top left corner.
func (e *MapEditor) mapOrigin() (float64, float64) {
	ox, oy := e.doc.Viewer.place.Origin()
	return ox + float64(e.mapArea().Min.X), oy
}

// relayout fits the map frame to the map area again after a panel changed.
func (e *MapEditor) relayout() {
	if e.window.X > 0 && e.window.Y > 0 {
		e.Layout(e.window.X, e.window.Y)
	}
}

// keyPanel opens or closes the panel whose key is key.
func (e *MapEditor) keyPanel(key string) error {
	for i, p := range sidePanels {
		if p.key == key {
			e.togglePanel(i)
		}
	}
	return nil
}

// togglePanel closes panel i when it is shown, else opens it; a panel the
// window has no room for stays closed and says so.
func (e *MapEditor) togglePanel(i int) {
	s := &e.panels.state[i]
	if e.panelsShown()[i] > 0 {
		s.Open = false
		if i == panelLeft {
			e.tree.typing = false
		}
	} else {
		s.Open, s.Width = true, clampPanel(i, s.Width)
		if e.panelsShown()[i] == 0 {
			s.Open = false
			e.message = e.word("editor.side.narrow")
			return
		}
	}
	e.relayout()
	e.savePanels()
}

// SetPanelProfile gives the editor the panel state its profile file holds,
// with the error reading it gave, and the writer that keeps a new state. A
// missing or unreadable state opens both panels at their default widths;
// a window too narrow for them shows the left one only, or none.
func (e *MapEditor) SetPanelProfile(data []byte, readErr error, save func([]byte) error) {
	e.panels.save = save
	e.panels.state = panelStateOf(data, readErr)
	e.relayout()
}

// panelStateOf reads the profile file's panel state; a side it does not
// hold, or a file that does not read, takes the defaults.
func panelStateOf(data []byte, readErr error) [2]panelState {
	var out [2]panelState
	for i, p := range sidePanels {
		out[i] = panelState{Open: true, Width: p.width}
	}
	if readErr != nil {
		return out
	}
	var kept map[string]struct {
		Open  *bool
		Width int
	}
	if json.Unmarshal(data, &kept) != nil {
		return out
	}
	for i, p := range sidePanels {
		if k, ok := kept[p.name]; ok && k.Open != nil {
			out[i] = panelState{Open: *k.Open, Width: clampPanel(i, k.Width)}
		}
	}
	return out
}

// panelStateBytes is the panel state as the profile file keeps it.
func (e *MapEditor) panelStateBytes() []byte {
	kept := map[string]map[string]any{}
	for i, p := range sidePanels {
		s := e.panels.state[i]
		kept[p.name] = map[string]any{"open": s.Open, "width": clampPanel(i, s.Width)}
	}
	b, _ := json.Marshal(kept)
	return append(b, '\n')
}

// savePanels writes the panel state through the profile writer.
func (e *MapEditor) savePanels() {
	if e.panels.save == nil {
		return
	}
	if err := e.panels.save(e.panelStateBytes()); err != nil {
		e.message = err.Error()
	}
}

// objectInPanel reports that the object popup being built stands in the
// right panel: the panel is shown, the window is one it takes, and a double
// click has not taken this record's window out.
func (e *MapEditor) objectInPanel() bool {
	return e.panels.hostable && e.panelsShown()[panelRight] > 0 && e.panels.detached != e.selected+1
}

// panelWide is the width a window in the right panel wraps its text to.
func (e *MapEditor) panelWide() int {
	return max(160, e.sidePanelRect(panelRight).Dx()-widgetSpriteSize-2)
}

// hostedObject is the object popup in the right panel, scrolled to fit it.
func (e *MapEditor) hostedObject(s popupSpec) popupSpec {
	s = e.inPanel(s, panelRight)
	if s.MaxRows == 0 {
		s.MaxRows = popupRowCount(s)
	}
	s.MaxRows = max(1, layoutPopup(s, e.font, e.bodyFont()).RowsFit)
	e.objectTop = max(0, min(e.objectTop, popupRowCount(s)-s.MaxRows))
	s.Top = e.objectTop
	return s
}

// propsSpec is the right panel while it hosts no window: one hint line.
func (e *MapEditor) propsSpec() (popupSpec, bool) {
	if e.panelsShown()[panelRight] == 0 {
		return popupSpec{}, false
	}
	hint := e.word("editor.side.none")
	if _, _, ok := e.objectSpec(); ok {
		if e.panels.hosted {
			return popupSpec{}, false
		}
		hint = e.word("editor.side.floating")
	}
	s := popupSpec{Title: e.word(sidePanels[panelRight].titleWord), NoteInk: editorPaper}
	for _, line := range editorWrap(e.bodyFont(), hint, e.sidePanelRect(panelRight).Dx()-2*popupPad-8) {
		s.Rows = append(s.Rows, popupRow{Kind: popupText, Label: line})
	}
	return e.inPanel(s, panelRight), true
}

// panelOf is the panel popup id stands in, if it stands in one.
func (e *MapEditor) panelOf(id editorPopupID) (int, bool) {
	shown := e.panelsShown()
	switch id {
	case popTree:
		return panelLeft, shown[panelLeft] > 0
	case popProps:
		_, ok := e.propsSpec()
		return panelRight, ok
	case popObject:
		_, _, ok := e.objectSpec()
		return panelRight, ok && e.panels.hosted
	}
	return 0, false
}

// panelSpecOf is the spec of the popup id that stands in a panel.
func (e *MapEditor) panelSpecOf(id editorPopupID) popupSpec {
	switch id {
	case popTree:
		s, _ := e.treeSpec()
		return s
	case popProps:
		s, _ := e.propsSpec()
		return s
	}
	s, _, _ := e.objectSpec()
	return s
}

// panelEdgeAt is the inner border of the panel popup id stands in, when a
// press at p would grab it: the right border of the left panel, the left
// border of the right one.
func (e *MapEditor) panelEdgeAt(id editorPopupID, p image.Point) popupEdge {
	i, ok := e.panelOf(id)
	if !ok || !popupResizeEnabled {
		return 0
	}
	inner := edgeLeft
	if sidePanels[i].place == placeLeft {
		inner = edgeRight
	}
	return layoutPopup(e.panelSpecOf(id), e.font, e.bodyFont()).edgeAt(p) & inner
}

// resizePanel moves panel i's inner border by d, between the panel's least
// and greatest widths and no further than the map's least width allows.
func (e *MapEditor) resizePanel(i int, d image.Point) {
	shown := e.panelsShown()
	w := shown[i]
	if sidePanels[i].place == placeLeft {
		w += d.X
	} else {
		w -= d.X
	}
	room := e.window.X - mapLeastWidth - shown[1-i]
	e.panels.state[i].Width = max(sidePanels[i].least, min(sidePanels[i].most, room, w))
	e.relayout()
}

// panelTitlePress takes a press on the title row of a popup in a panel: a
// panel does not move, and a double click on the right panel's window takes
// it out to float.
func (e *MapEditor) panelTitlePress(id editorPopupID, p image.Point, double bool) bool {
	if _, ok := e.panelOf(id); !ok {
		return false
	}
	if part, _ := layoutPopup(e.panelSpecOf(id), e.font, e.bodyFont()).hit(p); part != popupTitle {
		return false
	}
	if double && id == popObject {
		e.panels.detached = e.selected + 1
	}
	return true
}

// drawSidePanels draws the shown panels: the tree, and the right panel's
// window or its hint line.
func (e *MapEditor) drawSidePanels(pic *image.RGBA) {
	if s, ok := e.treeSpec(); ok {
		drawPopup(pic, e.font, e.bodyFont(), nil, s, layoutPopup(s, e.font, e.bodyFont()), editorPopupInk, e.pointer, e.pointerOK)
	}
	if s, _, ok := e.objectSpec(); ok && e.panels.hosted {
		drawPopup(pic, e.font, e.bodyFont(), nil, s, layoutPopup(s, e.font, e.bodyFont()), editorPopupInk, e.pointer, e.pointerOK)
		e.drawObjectPreview(pic, s)
	} else if s, ok := e.propsSpec(); ok {
		drawPopup(pic, e.font, e.bodyFont(), nil, s, layoutPopup(s, e.font, e.bodyFont()), editorPopupInk, e.pointer, e.pointerOK)
	}
}

// panelRects are the shown panels' window rectangles.
func (e *MapEditor) panelRects() []image.Rectangle {
	var out []image.Rectangle
	for i := range sidePanels {
		if r := e.sidePanelRect(i); !r.Empty() {
			out = append(out, r)
		}
	}
	return out
}

// editorTree is the left panel's tree: its list model, what each row stands
// for, the closed branches, the search text and whether it is typed into.
type editorTree struct {
	model  *Picker
	rows   []treeRow
	closed map[string]bool
	search string
	typing bool
	sync   int
	window int
	// followed is the selection the tree last opened a branch for.
	followed string
}

// treeRow is what a tree row stands for: a branch, a record or a group.
type treeRow struct {
	branch string
	record int
	group  editorGroup
}

// treeMember is one member row: its text, the name and ID a search reads,
// and what it stands for.
type treeMember struct {
	text, key string
	row       treeRow
}

// treeBranches are the tree's branches in order, each with its title word.
var treeBranches = []struct{ key, word string }{
	{"players", "editor.bar.players"},
	{"triggers", "editor.scripts.triggers"},
	{"units", "editor.bar.units"},
	{"structures", "editor.bar.structures"},
	{"groups", "editor.bar.groups"},
}

// treeKey is a search key from parts in the font's bytes: decoded,
// lower-cased and kept apart, so a search matches within one part.
func (e *MapEditor) treeKey(parts ...string) string {
	for i, part := range parts {
		parts[i] = strings.ToLower(e.editorSearchText(part))
	}
	return strings.Join(parts, "\n")
}

// treeMembers are branch's member rows, in the order the editor lists them.
func (e *MapEditor) treeMembers(branch string) []treeMember {
	var out []treeMember
	record := func(kind string) {
		for i, r := range e.doc.Records {
			if r.Kind == kind {
				out = append(out, treeMember{text: e.recordSummary(i), key: e.treeKey(fmt.Sprintf("%s #%d", r.Kind, r.Index), r.Label), row: treeRow{record: i}})
			}
		}
	}
	switch branch {
	case "players":
		record("Player")
	case "units":
		record("Unit")
	case "structures":
		record("Structure")
	case "triggers":
		for _, id := range e.scriptSections()[0] {
			key := e.treeKey(e.recordSummary(id))
			if v, ok := e.scriptViewOf(id); ok && v.trigger != nil {
				key = e.treeKey(fmt.Sprintf(e.word("editor.trigger.title"), v.index+1), e.editorHostText(v.trigger.Name))
			}
			out = append(out, treeMember{text: e.scriptRowText(id, 0), key: key, row: treeRow{record: id}})
		}
	case "groups":
		for _, g := range e.groups() {
			out = append(out, treeMember{text: e.groupLabel(g), key: e.treeKey(fmt.Sprintf("%s %d", e.word("editor.group.group"), g.group)), row: treeRow{record: -1, group: g}})
		}
	}
	return out
}

// treeChosen reports that row stands for the editor's selection.
func (e *MapEditor) treeChosen(row treeRow) bool {
	if row.group.set {
		return e.groupSel.set && e.groupSel.player == row.group.player && e.groupSel.group == row.group.group
	}
	return !e.groupSel.set && row.branch == "" && row.record >= 0 && row.record == e.selected
}

// treeRows are the tree as list rows, what each stands for and the row of
// the selection, -1 for none. A search keeps the members whose name or ID
// holds it, under their branches, open; a branch then counts its matches.
func (e *MapEditor) treeRows() ([]PickerRow, []treeRow, int) {
	if e.doc == nil {
		return nil, nil, -1
	}
	query := strings.ToLower(strings.TrimSpace(e.tree.search))
	var rows []PickerRow
	var kinds []treeRow
	sel := -1
	for _, b := range treeBranches {
		var shown []treeMember
		for _, m := range e.treeMembers(b.key) {
			if query == "" || strings.Contains(m.key, query) {
				shown = append(shown, m)
			}
		}
		if query != "" && len(shown) == 0 {
			continue
		}
		open := query != "" || !e.tree.closed[b.key]
		fold := FoldOpen
		if !open {
			fold = FoldClosed
		}
		rows = append(rows, PickerRow{Text: fmt.Sprintf("%s (%d)", e.word(b.word), len(shown)), Choosable: true, Heading: true, Fold: fold})
		kinds = append(kinds, treeRow{branch: b.key, record: -1})
		if !open {
			continue
		}
		for _, m := range shown {
			if e.treeChosen(m.row) {
				sel = len(rows)
			}
			rows = append(rows, PickerRow{Text: m.text, Choosable: true, Indent: 1})
			kinds = append(kinds, m.row)
		}
	}
	return rows, kinds, sel
}

// treeFollow opens the branch that holds a selection made since the tree
// last looked, so the selection's row shows.
func (e *MapEditor) treeFollow() {
	now := fmt.Sprint(e.selected, e.groupSel.set, e.groupSel.player, e.groupSel.group)
	if now == e.tree.followed || e.doc == nil {
		return
	}
	e.tree.followed = now
	for _, b := range treeBranches {
		for _, m := range e.treeMembers(b.key) {
			if e.treeChosen(m.row) {
				delete(e.tree.closed, b.key)
				return
			}
		}
	}
}

// treeTyping reports that the search field takes the typed text.
func (e *MapEditor) treeTyping() bool {
	return e.tree.typing && e.panelsShown()[panelLeft] > 0 && !e.prompt.open() && !e.objectTyping() && e.pathDialog.pending == nil
}

// treeSpec is the left panel: the search field over the tree list.
func (e *MapEditor) treeSpec() (popupSpec, bool) {
	if e.panelsShown()[panelLeft] == 0 {
		return popupSpec{}, false
	}
	e.treeFollow()
	rows, kinds, sel := e.treeRows()
	if e.tree.model == nil {
		e.tree.model, e.tree.sync, e.tree.window = NewPicker(rows), -2, 0
	} else {
		e.tree.model.SetRows(rows)
	}
	m := e.tree.model
	e.tree.rows = kinds
	label := e.word("editor.side.search")
	search := popupRow{Kind: popupEdit, Label: label, Value: e.editorHostText(e.tree.search), Hint: e.word("editor.side.searchhint")}
	if e.treeTyping() {
		search.Focus = true
		search.Edit = e.inputView(editorTextSpan{0, len([]rune(e.tree.search))}, false)
	}
	s := popupSpec{Title: e.word(sidePanels[panelLeft].titleWord), Rows: []popupRow{search}, List: m, ListRows: 1, NoteInk: editorPaper}
	if f := e.bodyFont(); f != nil {
		s.LabelWidth = f.Advance(label) + 2*popupPad
	}
	s = e.inPanel(s, panelLeft)
	if n := layoutPopup(s, e.font, e.bodyFont()).ListRows; n != e.tree.window {
		e.tree.window = n
		m.SetRowLimit(n).SetWindow(n)
	}
	if sel != e.tree.sync {
		e.tree.sync = sel
		if sel >= 0 {
			m.Select(sel)
		}
	}
	s.ListRows = e.tree.window
	s.ListText = func(row, width int) string {
		rows := m.Rows()
		if row < 0 || row >= len(rows) {
			return ""
		}
		return editorClip(e.bodyFont(), rows[row].Text, width)
	}
	return s, true
}

// treeChoose is a press on tree row n: a branch opens or closes, a member is
// selected as a click on it on the map selects it, and a unit or structure
// comes into view.
func (e *MapEditor) treeChoose(n int) {
	if n < 0 || n >= len(e.tree.rows) || e.doc == nil {
		return
	}
	e.tree.model.Select(n)
	r := e.tree.rows[n]
	switch {
	case r.branch != "":
		if strings.TrimSpace(e.tree.search) == "" {
			if e.tree.closed == nil {
				e.tree.closed = map[string]bool{}
			}
			e.tree.closed[r.branch] = !e.tree.closed[r.branch]
		}
	case r.group.set:
		e.chooseGroup(r.group, true)
	case r.record >= 0 && r.record < len(e.doc.Records):
		kind := e.doc.Records[r.record].Kind
		e.Select(r.record, kind == "Unit" || kind == "Structure")
	}
}

// treePress is a press on the left panel.
func (e *MapEditor) treePress(p image.Point) {
	s, ok := e.treeSpec()
	if !ok {
		return
	}
	switch part, n := layoutPopup(s, e.font, e.bodyFont()).hit(p); part {
	case popupClose:
		e.togglePanel(panelLeft)
	case popupListRow:
		if top, count := s.List.Visible(); n < count {
			e.treeChoose(top + n)
		}
	}
}

// propsPress is a press on the right panel's hint: its close button closes
// the panel.
func (e *MapEditor) propsPress(p image.Point) {
	if s, ok := e.propsSpec(); ok {
		if part, _ := layoutPopup(s, e.font, e.bodyFont()).hit(p); part == popupClose {
			e.togglePanel(panelRight)
		}
	}
}

// treeSearchBox is the search field's box, empty while the panel is closed.
func (e *MapEditor) treeSearchBox() image.Rectangle {
	s, ok := e.treeSpec()
	if !ok {
		return image.Rectangle{}
	}
	for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if rr.Index == 0 {
			return rr.Control
		}
	}
	return image.Rectangle{}
}

// treeInputAt is the search field's text view under p: a press there makes
// the field take the typed text; while it does, a drag keeps it.
func (e *MapEditor) treeInputAt(p image.Point, press bool) (*editorTextView, image.Rectangle, bool) {
	box := e.treeSearchBox()
	if box.Empty() {
		return nil, image.Rectangle{}, false
	}
	if press {
		if !p.In(box) {
			return nil, image.Rectangle{}, false
		}
		e.endFieldTyping()
		e.tree.typing = true
	} else if !e.treeTyping() {
		return nil, image.Rectangle{}, false
	}
	v := e.inputView(editorTextSpan{0, len([]rune(e.tree.search))}, false)
	return v, box, v != nil
}
