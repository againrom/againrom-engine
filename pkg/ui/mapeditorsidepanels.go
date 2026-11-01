package ui

import (
	"encoding/json"
	"fmt"
	"image"
	"strings"

	"againrom/pkg/editormcp"

	"github.com/hajimehoshi/ebiten/v2"
)

// The window's panels: two side panels and the bottom panel. Each is a row of
// sidePanels and one builder, sidePanelRect, places any of them. The left
// panel shows the tree of the map's objects and scenarios, the right one the
// selected record's window, the bottom one the report. A window in a panel
// is the same popup a floating window is: inPanel gives it the panel's
// rectangle as its place, its size and its bounds. A panel does not close: it
// minimizes to a strip at its window edge that holds one restore button.

// popupPlace is where a popup stands: floating over the map or in a panel.
type popupPlace uint8

const (
	placeFloating popupPlace = iota
	placeLeft
	placeRight
	placeBottom
)

// The panels' rows in sidePanels.
const (
	panelLeft   = 0
	panelRight  = 1
	panelBottom = 2
)

// sidePanel is one panel as data: its place, its name in the profile file,
// its default, least and greatest widths (heights for the bottom panel), the
// word of its title, its key row's name, help label and keyboard key, the
// words of its title row's minimize button and its strip's restore button,
// the word that says the window has no room for it, whether a fresh profile
// keeps it minimized and whether its strip shows the title.
type sidePanel struct {
	place                     popupPlace
	name                      string
	width, least, most        int
	titleWord                 string
	key, label                string
	hotkey                    ebiten.Key
	minimizeWord, restoreWord string
	narrowWord                string
	startMinimized            bool
	stripTitled               bool
}

// The left panel's least width lets the tree show at the least window the
// editor admits, 640 wide, beside the map's least width and the right
// panel's strip. The bottom panel spans the map's width under it.
var sidePanels = [3]sidePanel{
	{place: placeLeft, name: "left", width: 260, least: 130, most: 420, titleWord: "editor.side.objects", key: "objectspanel", label: "F6", hotkey: ebiten.KeyF6, minimizeWord: "editor.side.toleft", restoreWord: "editor.side.toright", narrowWord: "editor.side.narrow"},
	{place: placeRight, name: "right", width: 520, least: 360, most: 800, titleWord: "editor.side.properties", key: "propertiespanel", label: "F7", hotkey: ebiten.KeyF7, minimizeWord: "editor.side.toright", restoreWord: "editor.side.toleft", narrowWord: "editor.side.narrow"},
	{place: placeBottom, name: "bottom", width: 180, least: 96, most: 480, titleWord: "editor.side.report", key: "reportpanel", label: "F8", hotkey: ebiten.KeyF8, minimizeWord: "editor.side.down", restoreWord: "editor.side.up", narrowWord: "editor.side.short", startMinimized: true, stripTitled: true},
}

// mapLeastWidth is the least width the map keeps beside the side panels,
// mapLeastHeight the least height it keeps above the bottom panel.
const (
	mapLeastWidth  = 480
	mapLeastHeight = 160
)

// PanelStateFile is the editor profile file that keeps the panels' state.
const PanelStateFile = "mapedit-panels.json"

// WritePanelState writes the panel state to file through the writer of the
// profile's MCP record.
func WritePanelState(file string, b []byte) error {
	return editormcp.WriteProfileFile(file, ".mapedit-panels-*.tmp", "panel state", b)
}

// panelState is a panel's kept state: restored or minimized, and its width.
// The zero state is minimized at the default width.
type panelState struct {
	Restored bool
	Width    int
}

// editorPanels is the side panels' state. It stays outside editorChrome, so
// a cancelled window does not take a panel's size back.
type editorPanels struct {
	state [len(sidePanels)]panelState
	save  func([]byte) error
	// reportRows is the list rows the report fits in the bottom panel.
	reportRows int
	// hosted reports that the last object popup built stands in the right
	// panel; hostable that the one being built is a window the panel takes.
	hosted, hostable bool
	// detached names the selection whose window a double click on the
	// panel's title row took out to float; empty for none.
	detached string
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

// panelStrip is minimized panel i's width: the title row's one button with
// two pixels of frame on each side. The bottom strip is as high as the one
// popup builder lays out a title row alone, so it never draws over the
// status strip.
func (e *MapEditor) panelStrip(i int) int {
	side := fontHeight(e.font, 15) + popupRowExtra + 2
	if sidePanels[i].place != placeBottom {
		return side
	}
	s := popupSpec{Title: " "}
	dockSpec(&s, image.Rect(0, 0, 4*side, side))
	return layoutPopup(s, e.font, e.bodyFont()).Rect.Dy()
}

// panelsLaid is each panel's shown width and whether it shows restored. A
// minimized panel shows its strip. So that the map keeps mapLeastWidth
// beside them, the right panel gives way first, then the left: a restored
// panel narrows from its remembered width down to its least width, and
// minimizes only when even that does not fit. The remembered width stays
// as it was, so a wider window shows it again. The bottom panel gives way
// the same way to the map's least height.
func (e *MapEditor) panelsLaid() (w [len(sidePanels)]int, open [len(sidePanels)]bool) {
	if e.window.X <= 0 {
		return w, open
	}
	for i := range sidePanels {
		w[i] = e.panelStrip(i)
		if s := e.panels.state[i]; s.Restored {
			w[i], open[i] = clampPanel(i, s.Width), true
		}
	}
	room := e.window.X - mapLeastWidth
	for _, i := range []int{panelRight, panelLeft} {
		if !open[i] || w[panelLeft]+w[panelRight] <= room {
			continue
		}
		w[i] = max(sidePanels[i].least, room-w[1-i])
		if w[panelLeft]+w[panelRight] > room {
			w[i], open[i] = e.panelStrip(i), false
		}
	}
	top := e.toolbarRect().Max.Y
	if high := e.mapBottom(top) - top - mapLeastHeight; open[panelBottom] && w[panelBottom] > high {
		w[panelBottom] = max(sidePanels[panelBottom].least, high)
		if w[panelBottom] > high {
			w[panelBottom], open[panelBottom] = e.panelStrip(panelBottom), false
		}
	}
	return w, open
}

// panelsShown is each panel's shown width, or height for the bottom panel:
// its width while restored, its strip while minimized.
func (e *MapEditor) panelsShown() [len(sidePanels)]int {
	w, _ := e.panelsLaid()
	return w
}

// panelOpen reports that panel i shows restored.
func (e *MapEditor) panelOpen(i int) bool {
	_, open := e.panelsLaid()
	return open[i]
}

// sidePanelRect is panel i's window rectangle: a side panel between the
// toolbar and the status strip, the bottom panel between the side panels
// above the status strip. It is the panel, or its strip while minimized.
func (e *MapEditor) sidePanelRect(i int) image.Rectangle {
	shown := e.panelsShown()
	w := shown[i]
	if w == 0 {
		return image.Rectangle{}
	}
	top := e.toolbarRect().Max.Y
	bottom := e.mapBottom(top)
	switch sidePanels[i].place {
	case placeLeft:
		return image.Rect(0, top, w, bottom)
	case placeBottom:
		return image.Rect(shown[panelLeft], max(top, bottom-w), e.window.X-shown[panelRight], bottom)
	}
	return image.Rect(e.window.X-w, top, e.window.X, bottom)
}

// inPanel places popup s in panel i: the one popup builder then lays it out
// to fill the panel, fitting its tabs, buttons and labels to the panel's
// width (popupFlowOf). Its title row's button minimizes the panel.
func (e *MapEditor) inPanel(s popupSpec, i int) popupSpec {
	dockSpec(&s, e.sidePanelRect(i))
	s.CloseLabel = e.word(sidePanels[i].minimizeWord)
	return s
}

// stripSpec is minimized panel i: the popup frame with no title, whose title
// row button restores the panel.
func (e *MapEditor) stripSpec(i int) (popupSpec, bool) {
	if e.window.X <= 0 || e.panelOpen(i) {
		return popupSpec{}, false
	}
	s := e.inPanel(popupSpec{NoteInk: editorPaper}, i)
	if sidePanels[i].stripTitled {
		s.Title = e.word(sidePanels[i].titleWord)
	}
	s.CloseLabel = e.word(sidePanels[i].restoreWord)
	return s, true
}

// PanelButtonRect is the window rectangle of panel i's title row button:
// minimize while the panel shows restored, restore on its strip.
func (e *MapEditor) PanelButtonRect(i int) (image.Rectangle, bool) {
	s, ok := e.stripSpec(i)
	switch {
	case ok:
	case i == panelLeft:
		s, ok = e.treeSpec()
	case i == panelBottom:
		s, ok = e.listSpec()
		if !ok || !e.reportDocked() {
			s, ok = e.reportHintSpec()
		}
	default:
		s, ok = e.propsSpec()
		if !ok {
			s, _, ok = e.objectSpec()
			ok = ok && e.panels.hosted
		}
	}
	if !ok {
		return image.Rectangle{}, false
	}
	return layoutPopup(s, e.font, e.bodyFont()).Close, true
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

// keyPanel minimizes or restores the panel whose key is key.
func (e *MapEditor) keyPanel(key string) error {
	for i, p := range sidePanels {
		if p.key == key {
			e.togglePanel(i)
		}
	}
	return nil
}

// togglePanel minimizes panel i when it shows restored, else restores it; a
// panel the window has no room for stays minimized and says so.
func (e *MapEditor) togglePanel(i int) {
	if !e.showPanel(i, !e.panelOpen(i)) {
		e.message = e.word(sidePanels[i].narrowWord)
	}
}

// showPanel restores panel i when show is set and minimizes it when not,
// and reports whether the panel now shows as asked: a panel the window has
// no room for stays minimized.
func (e *MapEditor) showPanel(i int, show bool) bool {
	if e.panelOpen(i) == show {
		return true
	}
	s := &e.panels.state[i]
	if !show {
		// The report minimized with its panel closes rather than float.
		if i == panelBottom && e.reportDocked() {
			e.leave(leaveReport)
		}
		s.Restored = false
		if i == panelLeft {
			e.tree.typing = false
		}
	} else {
		was := s.Restored
		s.Restored, s.Width = true, clampPanel(i, s.Width)
		if !e.panelOpen(i) {
			s.Restored = was
			return false
		}
	}
	e.relayout()
	e.savePanels()
	return true
}

// SetPanelProfile gives the editor the panel state its profile file holds,
// with the error reading it gave, and the writer that keeps a new state. A
// missing or unreadable state restores both side panels at their default
// widths and keeps the bottom panel minimized; a window too narrow for the
// side panels shows the left one only, or neither.
func (e *MapEditor) SetPanelProfile(data []byte, readErr error, save func([]byte) error) {
	e.panels.save = save
	e.panels.state = panelStateOf(data, readErr)
	e.props.plain = plainViewOf(data, readErr)
	e.kept = keptSizesOf(data, readErr)
	e.relayout()
}

// profileEntries is the profile file's top-level entries, each still
// encoded, so that one entry that does not read costs no other.
func profileEntries(data []byte, readErr error) map[string]json.RawMessage {
	var all map[string]json.RawMessage
	if readErr != nil || json.Unmarshal(data, &all) != nil {
		return nil
	}
	return all
}

// keptWindowsEntry is the profile file's entry for the dragged sizes of the
// windows that keep them.
const keptWindowsEntry = "windows"

// keptSizesOf reads the dragged window sizes from the profile file: one
// width and height for each window that keeps its size. A missing, broken or
// empty entry gives no size, and the window its default.
func keptSizesOf(data []byte, readErr error) map[string]image.Point {
	var windows map[string]json.RawMessage
	if json.Unmarshal(profileEntries(data, readErr)[keptWindowsEntry], &windows) != nil {
		return nil
	}
	var out map[string]image.Point
	for _, name := range keptWindows {
		var k struct{ Width, Height int }
		if json.Unmarshal(windows[name], &k) != nil || k.Width <= 0 || k.Height <= 0 || k.Width > keptSizeMost || k.Height > keptSizeMost {
			continue
		}
		if out == nil {
			out = map[string]image.Point{}
		}
		out[name] = image.Pt(k.Width, k.Height)
	}
	return out
}

// plainViewOf reads the object window's view from the profile file: the
// right panel's "view", plain unless it names the technical view. A
// missing or broken file gives the plain view.
func plainViewOf(data []byte, readErr error) bool {
	var kept struct{ View string }
	if json.Unmarshal(profileEntries(data, readErr)[sidePanels[panelRight].name], &kept) != nil {
		return true
	}
	return kept.View != viewTechnical
}

// The words the profile file keeps the object window's view as.
const (
	viewPlain     = "plain"
	viewTechnical = "technical"
)

// panelStateOf reads the profile file's panel state; a panel it does not
// hold, or a file that does not read, takes the defaults. A side kept as
// "open" by an earlier build reads as restored or minimized.
func panelStateOf(data []byte, readErr error) [len(sidePanels)]panelState {
	var out [len(sidePanels)]panelState
	for i, p := range sidePanels {
		out[i] = panelState{Restored: !p.startMinimized, Width: p.width}
	}
	entries := profileEntries(data, readErr)
	if entries == nil {
		return out
	}
	kept := map[string]struct {
		Minimized *bool
		Open      *bool
		Width     int
	}{}
	for _, p := range sidePanels {
		raw, ok := entries[p.name]
		if !ok {
			continue
		}
		var k struct {
			Minimized *bool
			Open      *bool
			Width     int
		}
		if json.Unmarshal(raw, &k) != nil {
			return out
		}
		kept[p.name] = k
	}
	for i, p := range sidePanels {
		k, ok := kept[p.name]
		switch {
		case !ok:
		case k.Minimized != nil:
			out[i] = panelState{Restored: !*k.Minimized, Width: clampPanel(i, k.Width)}
		case k.Open != nil:
			out[i] = panelState{Restored: *k.Open, Width: clampPanel(i, k.Width)}
		}
	}
	return out
}

// panelStateBytes is the panel state as the profile file keeps it.
func (e *MapEditor) panelStateBytes() []byte {
	kept := map[string]map[string]any{}
	for i, p := range sidePanels {
		s := e.panels.state[i]
		kept[p.name] = map[string]any{"minimized": !s.Restored, "width": clampPanel(i, s.Width)}
	}
	view := viewTechnical
	if e.props.plain {
		view = viewPlain
	}
	kept[sidePanels[panelRight].name]["view"] = view
	if len(e.kept) > 0 {
		windows := map[string]any{}
		for name, sz := range e.kept {
			windows[name] = map[string]int{"width": sz.X, "height": sz.Y}
		}
		kept[keptWindowsEntry] = windows
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

// detachKey names the selection whose window stands in the right panel: a
// record, a script variable or a chosen group.
func (e *MapEditor) detachKey() string {
	if e.groupSel.set && e.selected < 0 {
		return fmt.Sprintf("group %d/%d", e.groupSel.player, e.groupSel.group)
	}
	return fmt.Sprintf("record %d", e.selected)
}

// objectInPanel reports that the object popup being built stands in the
// right panel: the panel shows restored, the window is one it takes, and a
// double click has not taken this selection's window out.
func (e *MapEditor) objectInPanel() bool {
	return e.panels.hostable && e.panelOpen(panelRight) && e.panels.detached != e.detachKey()
}

// panelWide is the width a window in the right panel wraps its text to.
func (e *MapEditor) panelWide() int {
	return max(160, e.sidePanelRect(panelRight).Dx()-widgetSpriteSize-2)
}

// hostedAside is the width the card's column of object popup s takes beside
// its rows: all of it while the popup floats, none while the right panel
// stands the card above the rows (cardOnTop).
func (e *MapEditor) hostedAside(s popupSpec) int {
	aside := cardAside(s)
	if aside > 0 && e.objectInPanel() && cardOnTop(e.sidePanelRect(panelRight).Dx(), aside) {
		return 0
	}
	return aside
}

// hostedWrap is the width object popup s's builder wraps a line of text to:
// natural while the popup floats, and in the right panel no wider than the
// rows there, beside the scroll bar and the card's column.
func (e *MapEditor) hostedWrap(s popupSpec, natural int) int {
	if !e.objectInPanel() {
		return natural
	}
	w := e.panelWide() - 2*popupPad - 4
	if aside := e.hostedAside(s); aside > 0 {
		w -= aside + popupPad
	}
	return max(80, min(natural, w))
}

// hostedObject is the object popup in the right panel, scrolled to fit it.
func (e *MapEditor) hostedObject(s popupSpec) popupSpec {
	s = e.inPanel(s, panelRight)
	if s.MaxRows == 0 {
		s.MaxRows = popupRowCount(s)
	}
	// The rows that fit are counted from the first row shown; where they
	// reach the last row, the first row shown is the least that still
	// shows the last, so every row is in reach and the last page is full.
	count, all := popupRowCount(s), s.MaxRows
	fit := func(top int) int {
		s.Top, s.MaxRows = top, all
		return max(1, layoutPopup(s, e.font, e.bodyFont()).RowsFit)
	}
	top := max(0, min(e.objectTop, count-1))
	if n := fit(top); top+n < count {
		e.objectTop, s.Top, s.MaxRows = top, top, n
		return s
	}
	lo := 0
	for hi := top; lo < hi; {
		if mid := (lo + hi) / 2; mid+fit(mid) >= count {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	n := fit(lo)
	e.objectTop, s.Top, s.MaxRows = lo, lo, n
	return s
}

// propsSpec is the restored right panel while it hosts no window: one hint
// line.
func (e *MapEditor) propsSpec() (popupSpec, bool) {
	if !e.panelOpen(panelRight) {
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
	for _, line := range editorTextLines(e.bodyFont(), hint, e.sidePanelRect(panelRight).Dx()-2*popupPad-8) {
		s.Rows = append(s.Rows, popupRow{Kind: popupText, Label: line})
	}
	return e.inPanel(s, panelRight), true
}

// panelOf is the panel popup id stands in, if it stands in one.
func (e *MapEditor) panelOf(id editorPopupID) (int, bool) {
	switch id {
	case popTree:
		return panelLeft, e.window.X > 0
	case popProps:
		if _, ok := e.stripSpec(panelRight); ok {
			return panelRight, true
		}
		_, ok := e.propsSpec()
		return panelRight, ok
	case popObject:
		_, _, ok := e.objectSpec()
		return panelRight, ok && e.panels.hosted
	case popReport:
		return panelBottom, e.window.X > 0
	case popList:
		return panelBottom, e.reportDocked()
	}
	return 0, false
}

// reportDocked reports that the list popup shows the report in the restored
// bottom panel. The script panel's arms keep their floating window.
func (e *MapEditor) reportDocked() bool {
	return e.listKind() == "report" && len(e.report.goals) == 0 && e.panelOpen(panelBottom)
}

// reportHintSpec is the restored bottom panel while it shows no report: its
// title and one hint line.
func (e *MapEditor) reportHintSpec() (popupSpec, bool) {
	if !e.panelOpen(panelBottom) || e.reportDocked() {
		return popupSpec{}, false
	}
	s := popupSpec{Title: e.word(sidePanels[panelBottom].titleWord), NoteInk: editorPaper}
	for _, line := range editorTextLines(e.bodyFont(), e.word("editor.side.noreport"), e.sidePanelRect(panelBottom).Dx()-2*popupPad-8) {
		s.Rows = append(s.Rows, popupRow{Kind: popupText, Label: line})
	}
	return e.inPanel(s, panelBottom), true
}

// dockReport is the report's list popup s in the bottom panel, its list
// fitted to the panel's height.
func (e *MapEditor) dockReport(s popupSpec) popupSpec {
	s = e.inPanel(s, panelBottom)
	if s.List == nil {
		return s
	}
	if n := layoutPopup(s, e.font, e.bodyFont()).ListRows; n != e.panels.reportRows {
		// The rows the panel fits are the list's window: build the popup
		// again at that window, so its rows and its count line agree.
		e.panels.reportRows = n
		s, _ = e.listSpecBase()
		s = e.inPanel(s, panelBottom)
	}
	s.ListRows = e.panels.reportRows
	return s
}

// panelSpecOf is the spec of the popup id that stands in a panel: the
// panel's strip while it is minimized.
func (e *MapEditor) panelSpecOf(id editorPopupID) popupSpec {
	switch id {
	case popTree:
		if s, ok := e.stripSpec(panelLeft); ok {
			return s
		}
		s, _ := e.treeSpec()
		return s
	case popProps:
		if s, ok := e.stripSpec(panelRight); ok {
			return s
		}
		s, _ := e.propsSpec()
		return s
	case popReport:
		if s, ok := e.stripSpec(panelBottom); ok {
			return s
		}
		s, _ := e.reportHintSpec()
		return s
	case popList:
		s, _ := e.listSpec()
		return s
	}
	s, _, _ := e.objectSpec()
	return s
}

// panelEdgeAt is the inner border of the restored panel popup id stands in,
// when a press at p would grab it: the right border of the left panel, the
// left border of the right one, the top border of the bottom one.
func (e *MapEditor) panelEdgeAt(id editorPopupID, p image.Point) popupEdge {
	i, ok := e.panelOf(id)
	if !ok || !popupResizeEnabled || !e.panelOpen(i) {
		return 0
	}
	inner := edgeLeft
	switch sidePanels[i].place {
	case placeLeft:
		inner = edgeRight
	case placeBottom:
		inner = edgeTop
	}
	return layoutPopup(e.panelSpecOf(id), e.font, e.bodyFont()).edgeAt(p) & inner
}

// resizePanel moves panel i's inner border by d, between the panel's least
// and greatest widths and no further than the map's least width, or for the
// bottom panel the map's least height, allows.
func (e *MapEditor) resizePanel(i int, d image.Point) {
	shown := e.panelsShown()
	w := shown[i]
	var room int
	switch sidePanels[i].place {
	case placeLeft:
		w += d.X
	case placeBottom:
		w -= d.Y
	default:
		w -= d.X
	}
	if sidePanels[i].place == placeBottom {
		top := e.toolbarRect().Max.Y
		room = e.mapBottom(top) - top - mapLeastHeight
	} else {
		room = e.window.X - mapLeastWidth - shown[1-i]
	}
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
		e.panels.detached = e.detachKey()
	}
	return true
}

// drawSidePanels draws the panels: the tree or its strip, the right panel's
// window, its hint line or its strip, and the bottom panel's hint line or
// its strip; the report in the bottom panel draws as the list popup.
func (e *MapEditor) drawSidePanels(pic *image.RGBA) {
	draw := func(s popupSpec) {
		drawPopup(pic, e.font, e.bodyFont(), nil, s, layoutPopup(s, e.font, e.bodyFont()), editorPopupInk, e.pointer, e.pointerOK)
	}
	if s, ok := e.stripSpec(panelLeft); ok {
		draw(s)
	} else if s, ok := e.treeSpec(); ok {
		draw(s)
	}
	switch s, ok := e.stripSpec(panelRight); {
	case ok:
		draw(s)
	default:
		if s, _, ok := e.objectSpec(); ok && e.panels.hosted {
			draw(s)
			e.drawObjectPreview(pic, s)
		} else if s, ok := e.propsSpec(); ok {
			draw(s)
		}
	}
	if s, ok := e.stripSpec(panelBottom); ok {
		draw(s)
	} else if s, ok := e.reportHintSpec(); ok {
		draw(s)
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

// panelButtonPress takes a press on panel i's title row button: it
// minimizes a restored panel and restores a minimized one.
func (e *MapEditor) panelButtonPress(i int, p image.Point) bool {
	if r, ok := e.PanelButtonRect(i); ok && p.In(r) {
		e.togglePanel(i)
		return true
	}
	return false
}

// editorCategory is one of the editor's record categories: the key its bar
// entry and its tree branch share, its word, the records list filter that
// holds its members and how the tree reads them. The bar's record entries
// and the tree's branches are this table in its order; each category's
// floating window is its palette, the records popup without its list.
type editorCategory struct {
	key, word string
	filter    int
	members   categoryMembers
}

// categoryMembers is how the tree reads a category's members.
type categoryMembers uint8

const (
	membersRecords categoryMembers = iota // the records of the category's filter
	membersScripts                        // the scripts list's four sections
	membersGroups                         // the map's unit groups
)

var editorCategories = []editorCategory{
	{key: "scripts", word: "editor.list.scripts", filter: editorScriptsFilter, members: membersScripts},
	{key: "players", word: "editor.bar.players", filter: editorPlayersFilter},
	{key: "groups", word: "editor.bar.groups", filter: editorGroupsFilter, members: membersGroups},
	{key: "units", word: "editor.bar.units", filter: 1},
	{key: "sacks", word: "editor.bar.sacks", filter: 3},
	{key: "structures", word: "editor.bar.structures", filter: 2},
}

// categoryByKey is the category whose key is key.
func categoryByKey(key string) (editorCategory, bool) {
	for _, c := range editorCategories {
		if c.key == key {
			return c, true
		}
	}
	return editorCategory{}, false
}

// categoryOf is the category whose records list filter is filter.
func categoryOf(filter int) (editorCategory, bool) {
	for _, c := range editorCategories {
		if c.filter == filter {
			return c, true
		}
	}
	return editorCategory{}, false
}

// paletteShown reports that the list popup shows a category's palette.
func (e *MapEditor) paletteShown() bool {
	_, ok := categoryOf(e.filter)
	return ok && e.listKind() == "records"
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
	// reveal is the branch the tree scrolls to its title row next; last the
	// branch of the last member row chosen.
	reveal, last string
}

// treeRow is what a tree row stands for: a branch, a member (a record, a
// script variable or a group) or, with neither, a section heading.
type treeRow struct {
	branch string
	of     string
	member bool
	record int
	group  editorGroup
}

// treeMember is one member row: its text, the name and ID a search reads,
// what it stands for, and the heading of the section it lists under.
type treeMember struct {
	text, key, section string
	row                treeRow
}

// treeKey is a search key from parts in the font's bytes: decoded,
// lower-cased and kept apart, so a search matches within one part.
func (e *MapEditor) treeKey(parts ...string) string {
	for i, part := range parts {
		parts[i] = strings.ToLower(e.editorSearchText(part))
	}
	return strings.Join(parts, "\n")
}

// treeMembers are category c's member rows, in the order its records list
// listed them.
func (e *MapEditor) treeMembers(c editorCategory) []treeMember {
	var out []treeMember
	switch c.members {
	case membersRecords:
		for _, id := range e.recordsOf(c.filter) {
			r := e.doc.Records[id]
			out = append(out, treeMember{text: e.recordSummary(id), key: e.treeKey(fmt.Sprintf("%s #%d", r.Kind, r.Index), r.Label), row: treeRow{of: c.key, member: true, record: id}})
		}
	case membersScripts:
		headings := []string{e.word("editor.scripts.triggers"), e.word("editor.scripts.variables"), e.word("editor.scripts.conditions"), e.word("editor.scripts.actions")}
		for section, ids := range e.scriptSections() {
			if len(ids) == 0 {
				// An empty section shows its heading only, as the scripts
				// list showed it.
				out = append(out, treeMember{section: fmt.Sprintf(e.word("editor.scripts.emptyheading"), headings[section]), row: treeRow{record: -1}})
			}
			for _, id := range ids {
				text := e.scriptRowText(id, section)
				key := e.treeKey(text)
				if v, ok := e.scriptViewOf(id); ok && v.trigger != nil {
					key = e.treeKey(fmt.Sprintf(e.word("editor.trigger.title"), v.index+1), e.editorHostText(v.trigger.Name))
				}
				out = append(out, treeMember{text: text, key: key, section: headings[section], row: treeRow{of: c.key, member: true, record: id}})
			}
		}
	case membersGroups:
		for _, g := range e.groups() {
			out = append(out, treeMember{text: e.groupLabel(g), key: e.treeKey(fmt.Sprintf("%s %d", e.word("editor.group.group"), g.group)), row: treeRow{of: c.key, member: true, record: -1, group: g}})
		}
	}
	return out
}

// treeChosen reports that row stands for the editor's selection.
func (e *MapEditor) treeChosen(row treeRow) bool {
	if !row.member {
		return false
	}
	if row.group.set {
		return e.groupSel.set && e.groupSel.player == row.group.player && e.groupSel.group == row.group.group
	}
	return !e.groupSel.set && row.record == e.selected
}

// treeRows are the tree as list rows, what each stands for and the row of
// the selection, -1 for none. The branches are the record categories in the
// bar's order. A search keeps the members whose name or ID holds it, under
// their branches, open; a branch then counts its matches.
func (e *MapEditor) treeRows() ([]PickerRow, []treeRow, int) {
	if e.doc == nil {
		return nil, nil, -1
	}
	query := strings.ToLower(strings.TrimSpace(e.tree.search))
	var rows []PickerRow
	var kinds []treeRow
	sel := -1
	for _, c := range editorCategories {
		var shown []treeMember
		count := 0
		for _, m := range e.treeMembers(c) {
			if query == "" || strings.Contains(m.key, query) {
				shown = append(shown, m)
				if m.row.member {
					count++
				}
			}
		}
		if query != "" && len(shown) == 0 {
			continue
		}
		open := query != "" || !e.tree.closed[c.key]
		fold := FoldOpen
		if !open {
			fold = FoldClosed
		}
		rows = append(rows, PickerRow{Text: fmt.Sprintf("%s (%d)", e.word(c.word), count), Choosable: true, Heading: true, Fold: fold})
		kinds = append(kinds, treeRow{branch: c.key, record: -1})
		if !open {
			continue
		}
		section := ""
		for _, m := range shown {
			indent := 1
			if m.section != "" {
				indent = 2
				if m.section != section {
					section = m.section
					rows = append(rows, PickerRow{Text: section, Heading: true, Indent: 1})
					kinds = append(kinds, treeRow{record: -1})
				}
			}
			if !m.row.member {
				continue
			}
			if e.treeChosen(m.row) {
				sel = len(rows)
			}
			rows = append(rows, PickerRow{Text: m.text, Choosable: true, Indent: indent})
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
	for _, c := range editorCategories {
		for _, m := range e.treeMembers(c) {
			if e.treeChosen(m.row) {
				delete(e.tree.closed, c.key)
				return
			}
		}
	}
}

// treeReveal opens category c's branch and has the tree scroll to its
// title row, restoring the left panel when it is minimized. A search that
// leaves the branch out is cleared.
func (e *MapEditor) treeReveal(c editorCategory) {
	delete(e.tree.closed, c.key)
	e.tree.reveal = c.key
	if q := strings.ToLower(strings.TrimSpace(e.tree.search)); q != "" {
		hit := false
		for _, m := range e.treeMembers(c) {
			hit = hit || strings.Contains(m.key, q)
		}
		if !hit {
			e.tree.search, e.tree.typing = "", false
		}
	}
	if s := &e.panels.state[panelLeft]; !s.Restored {
		s.Restored, s.Width = true, clampPanel(panelLeft, s.Width)
		e.relayout()
		e.savePanels()
	}
}

// treeTyping reports that the search field takes the typed text.
func (e *MapEditor) treeTyping() bool {
	return e.tree.typing && e.panelOpen(panelLeft) && !e.prompt.open() && !e.objectTyping() && e.pathDialog.pending == nil
}

// treeSpec is the restored left panel: the search field over the tree list.
func (e *MapEditor) treeSpec() (popupSpec, bool) {
	if !e.panelOpen(panelLeft) {
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
	if e.tree.reveal != "" {
		for n, k := range kinds {
			if k.branch == e.tree.reveal {
				top, _ := m.Visible()
				m.Scroll(n - top)
			}
		}
		e.tree.reveal = ""
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

// treeRowsNow are what the tree's rows stand for as it shows now; none while
// the left panel is minimized.
func (e *MapEditor) treeRowsNow() []treeRow {
	if _, ok := e.treeSpec(); !ok {
		return nil
	}
	return e.tree.rows
}

// TreeRowRect is the window rectangle of tree row n, the tree scrolled to
// show it: a headless driver presses it.
func (e *MapEditor) TreeRowRect(n int) (image.Rectangle, bool) {
	s, ok := e.treeSpec()
	if !ok || n < 0 || n >= s.List.Len() {
		return image.Rectangle{}, false
	}
	if top, count := s.List.Visible(); n < top {
		s.List.Scroll(n - top)
	} else if n >= top+count {
		s.List.Scroll(n - top - count + 1)
	}
	top, _ := s.List.Visible()
	return layoutPopup(s, e.font, e.bodyFont()).List.Row(n - top), true
}

// TreeMemberRow is the tree row of member n of category key's branch, -1
// when it does not show.
func (e *MapEditor) TreeMemberRow(key string, n int) int {
	for row, r := range e.treeRowsNow() {
		if r.member && r.of == key {
			if n == 0 {
				return row
			}
			n--
		}
	}
	return -1
}

// TreeBranchRow is the tree row of category key's branch, -1 when it does
// not show.
func (e *MapEditor) TreeBranchRow(key string) int {
	for row, r := range e.treeRowsNow() {
		if r.branch == key {
			return row
		}
	}
	return -1
}

// TreeLines are the tree's rows as text, indented two spaces a step, in the
// font's bytes, for a headless driver.
func (e *MapEditor) TreeLines() []string {
	s, ok := e.treeSpec()
	if !ok {
		return nil
	}
	var out []string
	for _, r := range s.List.Rows() {
		mark := ""
		if r.Fold != FoldNone {
			mark = rowFoldMarks[r.Fold] + " "
		}
		out = append(out, strings.Repeat("  ", r.Indent)+mark+r.Text)
	}
	return out
}

// GroupDeleteRect is the window rectangle of the chosen group's Delete group
// button.
func (e *MapEditor) GroupDeleteRect() (image.Rectangle, bool) {
	s, _, ok := e.objectSpec()
	if !ok || !e.groupSel.set || e.selected >= 0 {
		return image.Rectangle{}, false
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	if len(l.Buttons) == 0 {
		return image.Rectangle{}, false
	}
	return l.Buttons[0], true
}

// treeChoose is a press on tree row n. A branch opens or closes and shows
// its palette, as its bar entry does. A member is chosen as the records
// list's row chose it (chooseListed, or chooseGroup for a group), and a
// member of another branch than the last one chosen shows its branch's
// palette.
func (e *MapEditor) treeChoose(n int) {
	if n < 0 || n >= len(e.tree.rows) || e.doc == nil {
		return
	}
	r := e.tree.rows[n]
	if r.branch == "" && !r.member {
		return
	}
	e.tree.model.Select(n)
	switch {
	case r.branch != "":
		if strings.TrimSpace(e.tree.search) == "" {
			if e.tree.closed == nil {
				e.tree.closed = map[string]bool{}
			}
			e.tree.closed[r.branch] = !e.tree.closed[r.branch]
		}
		if c, ok := categoryByKey(r.branch); ok {
			e.openPalette(c)
		}
		return
	}
	if c, ok := categoryByKey(r.of); ok && r.of != e.tree.last {
		if shown, ok := categoryOf(e.filter); !e.paletteShown() || !ok || shown.key != c.key {
			e.openPalette(c)
		}
	}
	e.tree.last = r.of
	if r.group.set {
		e.chooseGroup(r.group, true)
		return
	}
	e.chooseListed(r.record)
}

// treePress is a press on the left panel or its strip.
func (e *MapEditor) treePress(p image.Point) {
	if e.panelButtonPress(panelLeft, p) {
		return
	}
	s, ok := e.treeSpec()
	if !ok {
		return
	}
	if part, n := layoutPopup(s, e.font, e.bodyFont()).hit(p); part == popupListRow {
		if top, count := s.List.Visible(); n < count {
			e.treeChoose(top + n)
		}
	}
}

// propsPress is a press on the right panel's hint or its strip: the title
// row button minimizes or restores the panel.
func (e *MapEditor) propsPress(p image.Point) {
	e.panelButtonPress(panelRight, p)
}

// reportPress is a press on the bottom panel's hint or its strip: the title
// row button minimizes or restores the panel.
func (e *MapEditor) reportPress(p image.Point) {
	e.panelButtonPress(panelBottom, p)
}

// treeSearchBox is the search field's box, empty while the panel is
// minimized.
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
// the field take the typed text; while it does, a drag keeps it. A floating
// window over the field takes the press instead.
func (e *MapEditor) treeInputAt(p image.Point, press bool) (*editorTextView, image.Rectangle, bool) {
	box := e.treeSearchBox()
	if box.Empty() {
		return nil, image.Rectangle{}, false
	}
	if press {
		if !p.In(box) || e.chromeHit(p) != popTree {
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
