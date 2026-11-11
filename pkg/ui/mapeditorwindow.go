package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"slices"
	"strings"
	"time"

	"againrom/pkg/pathdialog"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textfit"
	"againrom/pkg/render/textsmooth"
	"againrom/pkg/words"

	"github.com/hajimehoshi/ebiten/v2"
)

// The editor window's chrome: a top bar, a one-line status strip and the
// popups over the map. The map fills the window between the bar and the
// strip. Every popup is built by layoutPopup and drawn by drawPopup; the bar's
// two menus are drop-downs over the shared list model.

// editorPopupID names a popup of the window.
type editorPopupID uint8

const (
	popNone editorPopupID = iota
	popList
	popObject
	popPrompt
	popMenu
	popBar
	popStatus
	popArea    // a trigger area's label on the map
	popTree    // the left side panel's tree of the map
	popProps   // the right side panel while it hosts no window
	popReport  // the bottom panel while it shows no report, or its strip
	popToolbar // the toolbar over the map
	popCount
)

// editorChrome is the chrome's state: the fonts, where each popup stands, the
// list popup's model, the open menu and a press in progress.
type editorChrome struct {
	// triggerDetails is the trigger, plus one, whose popup shows the pair
	// view instead of the wizard; 0 shows the wizard.
	triggerDetails int
	// fresh is the one-click trigger whose first Apply joins its add's
	// history entry, while its wizard session lasts and no save stands
	// between.
	fresh freshTrigger
	// formAgreed is set once the author agreed to extend the open map's
	// script; the question is asked once per opened map.
	formAgreed bool
	// dropSel is the drop zone, plus one, the drop tool has selected.
	dropSel                int
	body                   *text.Font
	cardFont               *text.Font
	book                   words.Book
	hasBook                bool
	commandWords           [8]string
	listShown              bool
	objectHidden           bool
	places                 [popCount]popupStanding // where each movable popup stands, its size and its policy: the placement store
	model                  *Picker
	modelKey               string
	modelSync              int
	menu                   string
	menuModel              *Picker
	drag                   editorPopupID
	dragFrom               image.Point
	cursorShape            ebiten.CursorShapeType // the pointer shape last set on the window
	dragEdge               popupEdge              // the border a resize drag holds; zero is a move
	dragCard               int                    // the card row, plus one, a drag holds by its grip; zero is none
	uiDown                 bool
	buttonPress            buttonPress
	scroll                 editorScroll
	objectTop              int
	objectWheel, listWheel float64
	clicks                 doubleClick
	lastRow                int
	view                   editorView
	pointer                image.Point
	pointerOK              bool
	hoverCell              image.Point
	hoverOK                bool
	chromeImage            *ebiten.Image
	passInk                *ebiten.Image // the passability fills' one white pixel
	obj                    objectEdit
	groupSel               editorGroup
	exitAsked              bool
	saveNote, saveNoteFor  string           // the status line's account of the last save, and the source it is for
	saveStatus             saveNotice       // the status line's notice of a save
	saveClock              func() time.Time // the time the written notice is stamped and read by; time.Now when nil
	saveDeferred           func() error     // the write a window frame waits on, so the writing notice is drawn first
	deferSaves             bool             // set while the window's own update runs: a save waits one frame
	saveErrors             int              // the errors a confirmed save is writing
	saveThen               *leaveStep       // the step a Save As asked from the question runs once saved
	objectBack             []objectPlace
	messagePage            uint32
	messageNew             InspectionMessageOwner // where a new message is named, while messagePage is NewMessagePage
	// messageFor is the trigger action the dialog editor was opened for,
	// nil for any other route; messageFrom is the page that action names,
	// NewMessagePage for none.
	messageFor  *InspectionMessageOwner
	messageFrom uint32
	// tip is the hover tooltip over a palette's grid cells.
	tip editorTip
}

// editorView is the Layers menu's and the grid button's switches over the
// map's annotations.
type editorView struct {
	hideDrops, hideFootprints, hideIssues, grid, hideAreas bool
	passability, hideBorder                                bool
	heights                                                bool // the height numbers at the cell corners
	playerColours                                          bool // each owned unit and structure under its player's colour
}

// SetBodyFont sets the plain font of rows, labels, values and report text.
// Titles, the top bar and headings keep the editor's first font.
func (e *MapEditor) SetBodyFont(f *text.Font) { e.body = f }

func (e *MapEditor) SetCardFont(f *text.Font) { e.cardFont = f }

// SetWords sets the word table the editor's own words are read from.
func (e *MapEditor) SetWords(b words.Book) { e.book, e.hasBook = b, true }

func (e *MapEditor) SetCommandWords(w [8]string) { e.commandWords = w }

func (e *MapEditor) bodyFont() *text.Font {
	if e.body != nil {
		return e.body
	}
	return e.font
}

// word is the editor's word of id in the editor's language, in the font's
// encoding.
func (e *MapEditor) word(id string) string {
	return e.editorHostText(e.plainWord(id))
}

// plainWord is the editor's word of id as UTF-8 text, for text the map keeps.
func (e *MapEditor) plainWord(id string) string {
	b := e.book
	if !e.hasBook {
		b = words.For("")
	}
	return b.Text(id)
}

// barRowHeight is the height of one row of the top bar and of the status
// strip.
func (e *MapEditor) barRowHeight() int {
	return max(fontHeight(e.font, 15), fontHeight(e.bodyFont(), 15)) + 10
}

// barHeight is the top bar's height: one row, and more only in a window
// narrower than the bar, where barRows wraps it. The map frame is the
// window less the bar and the one-row status strip and sits at scale 1.
func (e *MapEditor) barHeight() int {
	entries := e.barRows()
	return entries[len(entries)-1].rect.Max.Y + 3
}

// mapArea is the window rectangle the map fills: the window between the
// toolbar and the status strip, less the side panels and the bottom panel.
func (e *MapEditor) mapArea() image.Rectangle {
	top := e.toolbarRect().Max.Y
	shown := e.panelsShown()
	return image.Rect(shown[panelLeft], top, e.window.X-shown[panelRight], max(top+1, e.mapBottom(top)-shown[panelBottom]))
}

// MapRect is the window rectangle the map fills, between the toolbar, the
// panels and the status strip: a headless driver aims at cells inside it.
func (e *MapEditor) MapRect() image.Rectangle { return e.mapArea() }

// mapBottom is the map's and the side panels' bottom edge: the status
// strip's top.
func (e *MapEditor) mapBottom(top int) int {
	return max(top+1, e.window.Y-e.barRowHeight())
}

func (e *MapEditor) statusRect() image.Rectangle {
	return image.Rect(0, e.mapBottom(e.toolbarRect().Max.Y), e.window.X, e.window.Y)
}

// barEntry is one entry of the top bar.
type barEntry struct {
	key, label string
	menu       bool
	field      bool
	disabled   bool
	on         bool
	rect       image.Rectangle
}

// barEntries is the top bar's entries in their rows, each with its state.
func (e *MapEditor) barEntries() []barEntry {
	entries := e.barRows()
	mode := e.barWindow()
	for i := range entries {
		if entries[i].menu {
			continue
		}
		entries[i].on = entries[i].key == mode
		switch entries[i].key {
		case "objects", "trees":
			entries[i].disabled = e.brushHost() == nil
		case "panel":
			_, ok := e.mapPanel()
			entries[i].disabled = !ok
		default:
			entries[i].disabled = e.doc == nil
		}
	}
	return entries
}

// barKey is one top bar entry as data: its key, its word, whether it opens
// a menu and whether it belongs against the right edge.
type barKey struct {
	key, labelWord string
	menu, right    bool
}

// barKeys places File and the panels at the left, the three utility menus
// at the right. Every menu reads the command table.
var barKeys = barKeyRows()

func barKeyRows() []barKey {
	rows := []barKey{
		{key: "file", labelWord: "editor.bar.file", menu: true},
		{key: "panel", labelWord: "editor.bar.map"},
	}
	for _, c := range editorCategories {
		rows = append(rows, barKey{key: c.key, labelWord: c.word})
	}
	return append(rows,
		barKey{key: "objects", labelWord: "editor.bar.objects"},
		barKey{key: "trees", labelWord: "editor.bar.trees"},
		barKey{key: "tools", labelWord: "editor.bar.tools", menu: true, right: true},
		barKey{key: "view", labelWord: "editor.bar.view", menu: true, right: true},
		barKey{key: "window", labelWord: "editor.bar.window", menu: true, right: true},
	)
}

// barRows places the top bar's entries, without their states, which the
// window's layout does not read.
func (e *MapEditor) barRows() []barEntry {
	entries := make([]barEntry, len(barKeys))
	for i, k := range barKeys {
		entries[i] = barEntry{key: k.key, label: e.word(k.labelWord), menu: k.menu}
	}
	h := e.barRowHeight()
	widths := make([]int, len(entries))
	rightWidth := 0
	for i := range entries {
		w := 16
		if e.font != nil {
			w += e.font.Advance(entries[i].label)
		}
		if entries[i].menu && !entries[i].field {
			w += h
		}
		widths[i] = w
		if barKeys[i].right {
			rightWidth += w + 4
		}
	}
	x, top := 4, 0
	for i := range entries {
		w := widths[i]
		if barKeys[i].right && (i == 0 || !barKeys[i-1].right) {
			x = max(x, e.window.X-rightWidth)
		}
		// An entry that would pass the window's right edge starts the next
		// row, so a window narrower than the bar still shows every entry.
		if x > 4 && e.window.X > 0 && x+w > e.window.X-4 {
			x, top = 4, top+h
		}
		entries[i].rect = image.Rect(x, top+3, x+w, top+h-3)
		x += w + 4
	}
	return entries
}

// controlBarRect is the window rectangle of a bar entry, such as file or panel.
func (e *MapEditor) controlBarRect(key string) (image.Rectangle, bool) {
	for _, b := range e.barEntries() {
		if b.key == key {
			return b.rect, true
		}
	}
	return image.Rectangle{}, false
}

// controlMenuRowRect is the window rectangle of row n of the open menu.
func (e *MapEditor) controlMenuRowRect(n int) (image.Rectangle, bool) {
	b, ok := e.openMenuEntry()
	if !ok || n < 0 || n >= len(e.menuItems(b.key)) {
		return image.Rectangle{}, false
	}
	d := e.menuDropdown(b, true)
	return d.listBox(e.font).Row(n), true
}

// controlListRowRect is the window rectangle of row n of the open list popup, when
// that row is shown.
func (e *MapEditor) controlListRowRect(n int) (image.Rectangle, bool) {
	s, ok := e.listSpec()
	if !ok || e.model == nil || s.List == nil {
		return image.Rectangle{}, false
	}
	top, count := e.model.Visible()
	if n < top || n >= top+count {
		return image.Rectangle{}, false
	}
	return layoutPopup(s, e.font, e.bodyFont()).List.Row(n - top), true
}

// controlListButtonRect is the window rectangle of button k of row n above the open
// list popup's list: the brush list's size and shape buttons.
func (e *MapEditor) controlListButtonRect(n, k int) (image.Rectangle, bool) {
	s, ok := e.listSpec()
	if !ok {
		return image.Rectangle{}, false
	}
	for _, r := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if r.Index == n && k >= 0 && k < len(r.Buttons) {
			return r.Buttons[k], true
		}
	}
	return image.Rectangle{}, false
}

// OpenWindows names the windows open over the map, in the order a press
// tests them: "menu" (a bar menu's drop-down), "prompt", "object" (the
// record popup) and "list" (the list popup), for a headless driver.
func (e *MapEditor) OpenWindows() []string {
	var out []string
	if _, ok := e.openMenuEntry(); ok {
		out = append(out, "menu")
	}
	if _, ok := e.promptSpec(); ok {
		out = append(out, "prompt")
	}
	if _, _, ok := e.objectSpec(); ok {
		out = append(out, "object")
	}
	if _, ok := e.listSpec(); ok {
		out = append(out, "list")
	}
	return out
}

func (e *MapEditor) ListRowCount() int {
	if s, ok := e.listSpec(); ok && s.List != nil {
		return s.List.Len()
	}
	return 0
}

// controlFieldRect is the window rectangle of the control of field n of the object
// popup, when that row is shown.
func (e *MapEditor) controlFieldRect(n int) (image.Rectangle, bool) {
	return e.fieldControlRect(n, -1)
}

func (e *MapEditor) controlFieldChoiceRect(n, choice int) (image.Rectangle, bool) {
	return e.fieldControlRect(n, choice)
}

func (e *MapEditor) fieldControlRect(n, choice int) (image.Rectangle, bool) {
	s, _, ok := e.objectSpec()
	if !ok || n < 0 || n >= len(e.obj.fields) {
		return image.Rectangle{}, false
	}
	for _, r := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if r.Index == e.fieldRow(n) {
			if choice >= 0 {
				if s.Rows[r.Index].Kind != popupDirection || choice >= len(r.Buttons) {
					return image.Rectangle{}, false
				}
				return r.Buttons[choice], true
			}
			return r.Control, true
		}
	}
	return image.Rectangle{}, false
}

func (e *MapEditor) controlObjectSectionRect(section string) (image.Rectangle, bool) {
	s, _, ok := e.objectSpec()
	i := map[string]int{"identity": 0, "equipment": 1, "inventory": 2, "stats": 3}[section]
	if !ok || len(s.Tabs) != 4 {
		return image.Rectangle{}, false
	}
	return layoutPopup(s, e.font, e.bodyFont()).Tabs[i], true
}

// controlGroupMemberRect is the window rectangle of member row n of the group
// popup, when that row is shown.
func (e *MapEditor) controlGroupMemberRect(n int) (image.Rectangle, bool) {
	s, links, ok := e.groupSpec()
	if !ok || n < 0 || n >= len(e.groupSel.members) {
		return image.Rectangle{}, false
	}
	row := slices.IndexFunc(links, func(l int) bool { return l < 0 }) + n
	if row < n || row >= len(links) || links[row] != -(e.groupSel.members[n]+1) {
		return image.Rectangle{}, false
	}
	for _, r := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if r.Index == row {
			return r.Rect, true
		}
	}
	return image.Rectangle{}, false
}

// menuItem is one row of a bar menu; a row with no key is a heading. A
// disabled row does nothing.
type menuItem struct {
	key, label string
	off        bool
}

// markWord is a switch's caption: its CheckText mark, then id.
func markWord(on bool, id string) string {
	return CheckSpec{On: on, Style: CheckText}.Mark() + id
}

// menuItems is the rows of bar menu menu, every menu taking them from the
// toolbar's table in table order, with a heading row over a row that names
// one and a rule row over a row that asks for one: a row reads as its
// button's tip, a tool or switch row carries the mark while its button shows
// on, and a row is disabled while its action would do nothing.
func (e *MapEditor) menuItems(menu string) []menuItem {
	if menu == searchKey {
		return e.searchItems()
	}
	var out []menuItem
	for _, t := range e.toolbarEntries() {
		if t.menu != menu {
			continue
		}
		if t.ruleBefore {
			out = append(out, menuItem{})
		}
		if t.headingWord != "" {
			out = append(out, menuItem{label: e.word(t.headingWord)})
		}
		label := e.toolWord(t.toolButton)
		if t.kind != toolAction {
			label = markWord(t.shown, label)
		}
		out = append(out, menuItem{key: t.key, label: label, off: t.closed})
	}
	return out
}

// brushMenuKeys is the row key of each brush mode: its toolbar button's.
var brushMenuKeys = map[brushMode]string{brushTerrain: "brush.terrain", brushHeight: "brush.height", brushObjects: "brush.objects", brushPassability: "brush.passability", brushBridge: "brush.bridge"}

// menuDropdown is the open menu as a drop-down anchored on its bar entry,
// whose list ends inside the window and is wide enough for its longest row;
// a disabled row draws in the hint ink and cannot be chosen.
func (e *MapEditor) menuDropdown(entry barEntry, open bool) dropdown {
	d := dropdown{Rect: entry.rect, Text: entry.label, Fill: editorPanel, Ink: editorPaper, ListWidth: 240, Disabled: entry.disabled, ListRight: e.window.X, Press: &e.buttonPress, Button: !entry.field, ButtonRows: true}
	if entry.field {
		row := e.searchRow()
		d.Edit, d.EditFont, d.Ink = &row, e.bodyFont(), editorLight
	}
	if open && e.menu == entry.key {
		items := e.menuItems(entry.key)
		rows := make([]PickerRow, len(items))
		for i, it := range items {
			rows[i] = PickerRow{Text: it.label, Choosable: it.key != "" && !it.off}
			switch {
			case it.key == "":
				rows[i].Heading, rows[i].Ink = true, editorGold
			case it.off:
				rows[i].Ink = popupHintInk
			}
			if e.font != nil {
				d.ListWidth = max(d.ListWidth, e.font.Advance(it.label)+2*listTextX+widgetSpriteSize+8)
			}
		}
		if e.menuModel == nil {
			e.menuModel = NewPicker(rows)
		}
		shown := len(rows)
		if entry.field {
			// A map's hits run to hundreds: the list scrolls.
			shown = min(shown, searchRows)
		}
		e.menuModel.SetRows(rows)
		e.menuModel.SetWindow(shown)
		d.Open, d.List, d.Rows = shown > 0, e.menuModel, shown
	}
	return d
}

// openMenuEntry is the bar entry of the open menu.
func (e *MapEditor) openMenuEntry() (barEntry, bool) {
	if e.menu == "" {
		return barEntry{}, false
	}
	if e.menu == searchKey && e.searchTyping() {
		return e.searchEntry(), true
	}
	for _, b := range e.barEntries() {
		if b.menu && b.key == e.menu {
			return b, true
		}
	}
	return barEntry{}, false
}

// toggleMenu opens menu key, or closes it when it is open.
func (e *MapEditor) toggleMenu(key string) {
	e.endFieldTyping()
	if e.menu == key {
		e.menu = ""
		return
	}
	e.menu = key
	e.menuModel = nil
}

// barAct is what a bar entry does.
func (e *MapEditor) barAct(key string) {
	if key == searchKey {
		_ = e.keySearch(key)
		return
	}
	e.endFieldTyping()
	for _, k := range barKeys {
		if k.key == key && k.menu {
			e.toggleMenu(key)
			return
		}
	}
	if e.barWindow() == key {
		e.closeBarWindow()
		return
	}
	e.closeBarWindow()
	set, remembered := e.barSets[key]
	delete(e.barSets, key)
	switch key {
	case "objects", "trees":
		e.objectBrush(key == "trees")
	case "panel":
		e.menuAct(key)
	default:
		if c, ok := categoryByKey(key); ok {
			e.showRecords(c.filter)
		}
	}
	if remembered {
		e.restoreBarWindows(set)
	}
}

// menuAct is what a menu row does.
func (e *MapEditor) menuAct(key string) {
	e.menu = ""
	e.endFieldTyping()
	switch key {
	case "records":
		e.showRecords(0)
	case "panel":
		if _, ok := e.mapPanel(); ok {
			e.leave(leaveDrag | leavePick)
			e.openPanel()
		}
	case "catalog":
		e.leave(leaveDrag | leaveReport)
		e.enterCatalog()
	case "saveto":
		if e.editable() {
			e.askPath(pathdialog.SaveFile)
		}
	case "drops":
		e.view.hideDrops = !e.view.hideDrops
	case "footprints":
		e.view.hideFootprints = !e.view.hideFootprints
	case "issues":
		e.view.hideIssues = !e.view.hideIssues
	case "grid":
		e.view.grid = !e.view.grid
	case "passview":
		e.view.passability = !e.view.passability
	case "border":
		e.view.hideBorder = !e.view.hideBorder
	case "brush.terrain", "brush.height", "brush.objects", "brush.passability", "brush.bridge":
		for mode, k := range brushMenuKeys {
			if k == key {
				e.chooseBrush(mode)
			}
		}
	case "areas":
		e.view.hideAreas = !e.view.hideAreas
	case "toolbar":
		e.toggleToolbar()
	case "select":
		if e.pick == pickArea || e.pick == pickTrapArea {
			e.leave(leavePick)
		}
		e.chooseBrush(brushOff)
	case "newtrigger":
		e.addNow(KindTrigger)
	default:
		if e.Key(key) == ebiten.Termination {
			e.exitAsked = true
		}
	}
}

// ExitAsked reports that a menu row asked to close the editor; Update then
// ends the run.
func (e *MapEditor) ExitAsked() bool { return e.exitAsked }

// showRecords opens the list popup on the records filter shows. A record
// category's filter opens its palette by every route, its bar entry, its key
// and an MCP call, and leaves the side panels and the tree's branches as
// they are, also when the Players entry selects a record or the Scenario
// entry clears one.
func (e *MapEditor) showRecords(filter int) {
	if e.doc == nil {
		return
	}
	if c, ok := categoryOf(filter); ok {
		e.openPalette(c)
		e.chooseFirstOf(filter)
		return
	}
	e.openRecords(filter)
}

// openPalette shows category c's palette in the list popup; a category
// whose palette holds nothing closes a shown palette instead. The selection
// stays as it is.
func (e *MapEditor) openPalette(c editorCategory) {
	e.openRecords(c.filter)
	if _, ok := e.listSpecBase(); !ok {
		e.listShown, e.filter = false, 0
	}
}

// openRecords opens the list popup on the records filter shows.
func (e *MapEditor) openRecords(filter int) {
	c, category := categoryOf(filter)
	same := category && e.barWindow() == c.key
	if !same {
		e.closeBarWindow()
	}
	e.rememberListCancel("records")
	if !same {
		e.leave(leaveDrag | leavePick | leaveReport | leaveCatalog | leaveBrush)
	}
	e.message = ""
	e.filter, e.listShown = filter, true
	if category {
		e.barOwner = c.key
		delete(e.barSets, c.key)
	}
}

// chooseFirstOf selects the first player record when the Players entry
// opens and the selection is none of them. The Scenario entry clears the
// selected record, a trigger's included, and opens none: a trigger opens
// only from its own row. A pasted group stays selected, so Del still
// deletes it while the Scenario list shows.
func (e *MapEditor) chooseFirstOf(filter int) {
	switch {
	case filter == editorScriptsFilter:
		e.selected = -1
	case filter == editorPlayersFilter:
		if rows := e.recordsOf(filter); len(rows) > 0 && !e.selectedIn(rows) {
			e.selectUnfollowed(rows[0])
		}
	}
}

func (e *MapEditor) selectedIn(rows []int) bool {
	for _, r := range rows {
		if r == e.selected {
			return true
		}
	}
	return false
}

// closeList closes what the list popup shows.
func (e *MapEditor) closeList() {
	switch e.listKind() {
	case "catalog":
		if e.doc != nil {
			e.leave(leaveCatalog)
		}
	case "report":
		e.leave(leaveReport)
	case "brush":
		e.leave(leaveBrush)
	case "panel":
		e.leave(leavePanel)
	default:
		e.leave(leaveRecords)
	}
	e.releaseBarWindow()
}

// listKind is what the list popup shows: the map picker, the report or
// script panel, the brush palette, the variables table, a records list,
// or nothing.
func (e *MapEditor) listKind() string {
	switch {
	case e.catalog:
		return "catalog"
	case e.report.open:
		return "report"
	case e.brushActive() || e.prompt.kind == promptAdd && e.prompt.add != nil && e.prompt.add.brush && e.brush.mode == brushObjects:
		return "brush"
	case e.listShown && e.doc != nil && e.filter == editorVariablesFilter:
		return "variables"
	case e.listShown && e.doc != nil:
		return "records"
	case e.panel && e.doc != nil:
		return "panel"
	}
	return ""
}

// catalogRows are the map picker's rows: under three headings the new-map
// sizes, the single-player maps and the multiplayer maps. The second result
// maps a row to its map, -1 for a heading.
func (e *MapEditor) catalogRows() ([]PickerRow, []int) {
	var rows []PickerRow
	var maps []int
	const fresh, single, multi = 0, 1, 2
	group := func(m EditorMap) int {
		switch {
		case ParseMapSource(m.Key).Kind == SourceNew:
			return fresh
		case m.Multiplayer:
			return multi
		}
		return single
	}
	add := func(heading string, g int) {
		first := true
		for i, m := range e.maps {
			if group(m) != g {
				continue
			}
			if first {
				rows, maps = append(rows, PickerRow{Text: heading}), append(maps, -1)
				first = false
			}
			label := m.Label
			if g == fresh { // under its heading a new map is its size: "32 x 32"
				label = strings.Replace(ParseMapSource(m.Key).Path, "x", " x ", 1)
			}
			rows, maps = append(rows, PickerRow{Text: "  " + label, Choosable: true}), append(maps, i)
		}
	}
	add(e.word("editor.picker.new"), fresh)
	add(e.word("editor.picker.single"), single)
	add(e.word("editor.picker.multi"), multi)
	return rows, maps
}

// listRows are the list popup's rows and the state's selected row, -1 when
// the model owns its selection.
func (e *MapEditor) listRows(kind string) ([]PickerRow, int) {
	var rows []PickerRow
	sel := -1
	switch kind {
	case "catalog":
		rows, _ = e.catalogRows()
	case "report":
		for n, f := range e.report.rows {
			label := reportLabel(f)
			if len(e.report.goals) > 0 {
				label = e.goalText(e.report.goals[n])
			}
			rows = append(rows, PickerRow{Text: label, Choosable: true})
		}
		sel = e.report.sel
	case "brush":
		var terrains []BrushTerrain
		if h := e.brushHost(); h != nil && e.brush.mode == brushTerrain {
			terrains = h.BrushTerrains()
		}
		for n, s := range e.brushRows() {
			row := PickerRow{Text: s, Choosable: true}
			if n < len(terrains) {
				row.Icon = terrains[n].Swatch
			}
			rows = append(rows, row)
		}
		sel = e.brush.sel[e.brush.mode]
	case "variables":
		for _, r := range e.variableTableRows() {
			rows = append(rows, PickerRow{Text: strings.Join(r.cells[:], "\t"), Choosable: true})
		}
	case "records":
		if e.filter == editorScriptsFilter {
			rows, _, sel = e.scenarioRows()
			break
		}
		for n, id := range e.rows() {
			text := e.recordSummary(id)
			if v, ok := e.scriptViewOf(id); ok && v.trigger != nil {
				text = fmt.Sprintf(e.word("editor.trigger.title"), v.index+1) + "  " + e.editorHostText(v.trigger.Name)
			}
			rows = append(rows, PickerRow{Text: text, Choosable: true})
			if id == e.selected {
				sel = n
			}
		}
	}
	return rows, sel
}

// recordSummary is a record's list row: kind and index, name, player and
// cell; a placed unit's name, which holds its cell, and player.
func (e *MapEditor) recordSummary(id int) string {
	r := e.doc.Records[id]
	s := fmt.Sprintf("%s #%d  %s", r.Kind, r.Index, r.Label)
	if r.Name != "" {
		s = r.Name
	}
	if r.Player > 0 {
		s += fmt.Sprintf("  P%d", r.Player)
	}
	if r.Spatial && r.Name == "" {
		s += fmt.Sprintf("  %d,%d", r.Cell.X, r.Cell.Y)
	}
	if r.Warning != "" {
		s = "! " + s
	}
	return s
}

func (e *MapEditor) listTitle(kind string) string {
	switch kind {
	case "catalog":
		return e.word("editor.picker.title")
	case "report":
		return e.report.note
	case "brush":
		return e.brushStatus()
	case "variables":
		return e.word("editor.variables.title")
	}
	switch e.filter {
	case 1:
		return e.word("editor.bar.units")
	case 2:
		return e.word("editor.bar.structures")
	case 3:
		return e.word("editor.bar.sacks")
	case editorPlayersFilter:
		return e.word("editor.bar.players")
	case editorScriptsFilter:
		return e.word("editor.list.scripts")
	case editorGroupsFilter:
		return e.word("editor.bar.groups")
	case editorCellSpellsFilter:
		return e.word("editor.cast.palette")
	}
	return e.word("editor.list.records")
}

// syncList brings the list model to the shown content: a new content gets a
// new model, a changed selection in the state moves the model's.
func (e *MapEditor) syncList(kind string, window int) *Picker {
	rows, sel := e.listRows(kind)
	key := fmt.Sprintf("%s/%d/%t", kind, e.filter, e.doc == nil)
	if e.model == nil || key != e.modelKey {
		e.model, e.modelKey, e.modelSync = NewPicker(rows), key, -2
		if kind == "catalog" && len(rows) > 1 && !rows[0].Choosable {
			e.model.Select(1)
		}
	} else {
		e.model.SetRows(rows)
	}
	e.model.SetWindow(window)
	if sel != e.modelSync {
		e.modelSync = sel
		if sel >= 0 {
			e.model.Select(sel)
		}
	}
	return e.model
}

// listWindow is how many rows the list popup shows at this window size.
func (e *MapEditor) listWindow() int {
	if e.reportDocked() && e.panels.reportRows > 0 {
		return e.panels.reportRows
	}
	if p := e.places[popList]; p.size != (image.Point{}) && p.fit > 0 {
		return p.fit
	}
	pitch := fontHeight(e.bodyFont(), 15) + listPitchExtra
	avail := e.mapArea().Dy() - 4*(fontHeight(e.bodyFont(), 15)+popupRowExtra) - 40 - e.forestGridHeight() - e.heightModeGridHeight() - e.brushPlaceGridHeight()
	return max(3, min(18, avail/pitch))
}

// visibleRows is the list popup's window in rows.
func (e *MapEditor) visibleRows() int { return e.listWindow() }

// popupBounds is where a floating window may stand: the window below the
// toolbar.
func (e *MapEditor) popupBounds() image.Rectangle {
	return image.Rect(0, e.toolbarRect().Max.Y, e.window.X, e.window.Y)
}

// popupRoom is the height a tall window sizes itself to: popupBounds above
// the status strip, so the person and mod windows at their places leave the
// status line in view at every window size.
func (e *MapEditor) popupRoom() image.Rectangle {
	b := e.popupBounds()
	b.Max.Y = max(b.Min.Y+1, e.mapArea().Max.Y)
	return b
}

// listSpec is the list popup as data.
func (e *MapEditor) listSpecBase() (popupSpec, bool) {
	kind := e.listKind()
	if kind == "" {
		return popupSpec{}, false
	}
	if kind == "panel" {
		s, _ := e.panelSpec()
		return s, true
	}
	if e.groupsShown() {
		return e.groupsSpec(), true
	}
	// A record category's window is its palette: the tree lists the
	// category's records, so the popup keeps only what is not that list.
	// The Scenario palette lists its records too, as the tree does, under
	// New trigger.
	palette := e.paletteShown()
	var m *Picker
	if !palette || e.paletteListsRecords() {
		m = e.syncList(kind, e.listWindow())
	}
	s := popupSpec{Title: e.listTitle(kind), At: e.placedCorner(popList, homeListMap), Width: e.popupWide(popList, 400), Bounds: e.popupBounds(), Room: e.mapArea(), List: m, ListRows: e.listWindow(), NoteInk: editorPaper}
	if m != nil {
		s.ListText = func(row, width int) string {
			rows := m.Rows()
			if row < 0 || row >= len(rows) {
				return ""
			}
			return editorClip(e.bodyFont(), rows[row].Text, width)
		}
	}
	switch kind {
	case "report":
		if len(e.report.goals) > 0 {
			s.Width, s.ListAside = e.popupWide(popList, 560), 84
			s.Notes = []string{e.word("editor.arms.hint")}
		} else {
			s.Notes = e.findingNotes(e.noteWidth(s.Width))
			if e.report.hint != "" {
				s.Notes = append(s.Notes, e.report.hint)
			}
		}
	case "brush":
		s.Rows = e.brushPopupRows()
		s.Width, s.LabelWidth, s.ListAside = e.popupWide(popList, 520), brushLabelWidth, e.brushAside()
		if g, ok := e.forestGridSpec(); ok {
			s.Grid = g
		} else if g, ok := e.heightModeGridSpec(); ok {
			s.Grid = g
		} else if g, note, ok := e.placeGridSpec(); ok {
			s.Grid = g
			s.Rows = append(s.Rows, popupRow{Kind: popupText, Label: editorClip(e.bodyFont(), note, s.Width-2*popupPad-8)})
		}
	case "catalog":
		if e.doc == nil {
			s.Notes = []string{e.word("editor.picker.hint")}
		}
		s.Buttons = []string{e.word("editor.button.open"), e.word("editor.button.file")}
	case "variables":
		e.variableTableSpec(&s, m)
	case "records":
		if e.filter == editorScriptsFilter {
			s.Notes = []string{e.word("editor.scripts.hint")}
			if len(e.rows()) == 0 && e.doc.ScriptNotice != "" {
				s.Notes = append(s.Notes, e.doc.ScriptNotice)
			}
		}
		s.ButtonFit = e.filter == editorScriptsFilter
		for _, b := range e.listButtons() {
			s.Buttons, s.Disabled = append(s.Buttons, b.label), append(s.Disabled, b.off)
			// A switch's caption carries its mark beside its word.
			s.ButtonFit = s.ButtonFit || b.key != "add"
		}
		if e.filter == editorScriptsFilter {
			// The palette is as wide as its three buttons need.
			s.Width = e.popupWide(popList, max(400, buttonRowWidth(s, e.bodyFont())))
		}
		if g, note, ok := e.placeGridSpec(); ok {
			s.Grid = g
			s.Rows = append(s.Rows, popupRow{Kind: popupText, Label: editorClip(e.bodyFont(), note, s.Width-2*popupPad-8)})
			e.placeSwitch(&s)
		}
		if g, rows, ok := e.relationGridSpec(); ok {
			s.Grid, s.Rows = g, append(s.Rows, rows...)
			s.Width = e.popupWide(popList, max(400, g.HeadW+g.Cols*g.side()+2*popupPad+widgetSpriteSize))
			s.LabelWidth = s.Width - 2*popupPad - 3*relationSide
		}
	}
	for _, note := range e.emptyPaletteNote() {
		s.Rows = append(s.Rows, popupRow{Kind: popupText, Label: editorClip(e.bodyFont(), note, s.Width-2*popupPad-8)})
	}
	counter := m != nil && m.Len() > 0
	if counter {
		s.Notes = append(s.Notes, "")
	}
	if p := e.places[popList]; kind == "brush" && m != nil && p.size == (image.Point{}) && !p.slack {
		// A brush window nobody sized or dragged shows the list rows that keep
		// it above the map's bottom edge, never fewer than listWindow's least:
		// its button rows and grid are more than listWindow allows for.
		for s.ListRows > 3 && layoutPopup(s, e.font, e.bodyFont()).Rect.Max.Y > s.Room.Max.Y {
			s.ListRows--
		}
		m.SetWindow(s.ListRows)
	}
	if counter {
		top, n := m.Visible()
		s.Notes[len(s.Notes)-1] = fmt.Sprintf("%d-%d / %d", top+1, top+n, m.Len())
	}
	if palette && len(s.Rows) == 0 && len(s.Notes) == 0 && len(s.Buttons) == 0 && s.Grid == nil {
		return popupSpec{}, false
	}
	return s, true
}

// findingNotes is the selected finding's text, wrapped to width and cut at
// five lines; a key list row is its own text and gives none.
func (e *MapEditor) findingNotes(width int) []string {
	f, ok := e.selectedFinding()
	if !ok || f.Rule == keyListRule || f.Text == "" {
		return nil
	}
	lines := editorTextLines(e.bodyFont(), e.editorHostText(f.Text), width)
	if len(lines) > 5 {
		lines = append(lines[:4], e.continuedLine(lines[4], width))
	}
	return lines
}

// continuedLine is line with the mark that more text follows, fitted to
// width.
func (e *MapEditor) continuedLine(line string, width int) string {
	if font := e.bodyFont(); font != nil {
		return editorFit(font).Continued(line, width)
	}
	return line + textfit.Mark
}

// noteWidth is the width the list popup's note lines wrap at: the popup's,
// or while the report stands in the bottom panel the panel's less the room
// of the mark a scrolled line may carry (fitDocked), so the mark cuts none.
func (e *MapEditor) noteWidth(popupWidth int) int {
	if !e.reportDocked() {
		return popupWidth - 2*popupPad - 8
	}
	mark := len(textfit.Mark)
	if font := e.bodyFont(); font != nil {
		mark = font.Advance(textfit.Mark)
	}
	return e.dockedNoteWidth() - mark
}

// dockedNoteWidth is the width of a note line in the bottom panel.
func (e *MapEditor) dockedNoteWidth() int {
	return e.sidePanelRect(panelBottom).Dx() - 2*popupPad - 8
}

// listButtons are the keys and captions of the records list's buttons: on
// the Scenario list the drop zone tool's switch, then New trigger; Add where another list adds.
func (e *MapEditor) listButtons() []menuItem {
	var out []menuItem
	if e.filter == editorCellSpellsFilter {
		out = append(out, menuItem{key: "traparea", label: e.word("editor.cast.area")})
		if e.selected >= 0 && e.selected < len(e.doc.Records) && e.doc.Records[e.selected].SetSize > 0 {
			out = append(out, menuItem{key: "delete", label: e.word("editor.cast.removeset")})
		}
	}
	if _, ok := e.script(); ok && e.listKind() == "records" && e.filter == editorScriptsFilter {
		out = append(out, menuItem{key: "drop", label: markWord(e.pick == pickDrop, e.word("editor.tools.dropzones"))})
	}
	if e.listKind() == "records" && e.filter == editorScriptsFilter {
		for _, kind := range scenarioNew {
			out = append(out, menuItem{key: kind.key, label: e.word(kind.word), off: !e.addNowOpen(kind.kind)})
		}
	}
	if _, ok := e.listAdd(); ok {
		out = append(out, menuItem{key: "add", label: e.word("editor.button.add")})
	}
	if kind, _, ok := e.placeGridKind(); ok && kind == KindStructure {
		out = append(out, menuItem{key: "ruined", label: markWord(e.place.ruined, e.word("editor.place.ruined"))})
	}
	return out
}

// scenarioNew are the Scenario palette's one-click add buttons: the key and
// caption word of each and the record kind it adds.
var scenarioNew = []struct {
	key, word string
	kind      RecordKind
}{
	{"newtrigger", "editor.tool.newtrigger", KindTrigger},
}

// controlListButtonKeyRect is the window rectangle of the records list's button
// key ("drop", "add"): a headless driver presses it.
func (e *MapEditor) controlListButtonKeyRect(key string) (image.Rectangle, bool) {
	s, ok := e.listSpec()
	if !ok {
		return image.Rectangle{}, false
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	for n, b := range e.listButtons() {
		if b.key == key && n < len(l.Buttons) {
			return l.Buttons[n], true
		}
	}
	return image.Rectangle{}, false
}

// listAdd is the kind the shown records list's Add button adds.
func (e *MapEditor) listAdd() (addKindRow, bool) {
	if _, ok := e.addHost(); !ok || e.listKind() != "records" {
		return addKindRow{}, false
	}
	return addKindFor(e.filter)
}

// listChoose acts on list row n: open a map, pick a finding or a brush row,
// or select a record and open its popup.
func (e *MapEditor) listChoose(n int, open bool) {
	switch e.listKind() {
	case "catalog":
		_, maps := e.catalogRows()
		if n < 0 || n >= len(maps) || maps[n] < 0 {
			return
		}
		e.model.Select(n)
		if key := e.maps[maps[n]].Key; open && ParseMapSource(key).Kind == SourceNew {
			e.askNewMode(key)
		} else if open {
			_ = e.Open(key)
		}
	case "report":
		e.selectFinding(n)
	case "brush":
		e.brushRowClick(n)
	case "variables":
		e.variablesChoose(n)
	case "records":
		if e.filter == editorScriptsFilter {
			e.scenarioChoose(n)
			return
		}
		if rows := e.rows(); n >= 0 && n < len(rows) {
			e.chooseListed(rows[n])
		}
	}
}

// scenarioChoose is a press on Scenario palette row n: a record row chooses
// its record as its tree row does.
func (e *MapEditor) scenarioChoose(n int) {
	rows, kinds, _ := e.scenarioRows()
	switch {
	case n < 0 || n >= len(rows) || !rows[n].Choosable:
	case kinds[n].member:
		e.model.Select(n)
		e.memberChoose(kinds[n])
	}
}

// chooseListed is the choice of record id on a list row, the records list's
// or the tree's: it selects the record and brings it into view, centring
// the map on a unit, structure or other placed record and on a trigger's
// area.
func (e *MapEditor) chooseListed(id int) {
	e.Select(id, true)
}

// catalogStep moves the picker's highlighted map, skipping headings.
func (e *MapEditor) catalogStep(delta int) {
	if e.model == nil || e.listKind() != "catalog" || delta == 0 {
		return
	}
	rows := e.model.Rows()
	step := 1
	if delta < 0 {
		step = -1
	}
	start := clampIndex(e.model.Selection()+delta, len(rows))
	for i := start; i >= 0 && i < len(rows); i += step {
		if rows[i].Choosable {
			e.model.Select(i)
			return
		}
	}
	for i := start; i >= 0 && i < len(rows); i -= step {
		if rows[i].Choosable {
			e.model.Select(i)
			return
		}
	}
}

// catalogOpen opens the picker's highlighted map.
func (e *MapEditor) catalogOpen() {
	if e.model != nil && e.listKind() == "catalog" {
		e.listChoose(e.model.Selection(), true)
	}
}

// objectSpecBase is the selected record's popup as data, and the reference
// each link row opens, with the button that armed the map pick drawn armed.
func (e *MapEditor) objectSpecBase() (popupSpec, []int, bool) {
	s, links, ok := e.objectSpecRows()
	if ok {
		e.markArmedPick(&s)
	}
	return s, links, ok
}

// markArmedPick marks the button of the field whose map pick is armed, so the
// popup row painter draws it pressed in: the message card, the node popup and
// the trigger wizard share it.
func (e *MapEditor) markArmedPick(s *popupSpec) {
	if (e.pick != pickParamUnit && e.pick != pickParamArea) || e.pickFieldTarget() == targetNone {
		return
	}
	row := e.fieldRow(e.obj.pickField)
	if row < 0 || row >= len(s.Rows) {
		return
	}
	for k, act := range e.obj.rowActs[row] {
		if act == actPickParam || act == actPickArea {
			s.Rows[row].Armed = make([]bool, max(k+1, len(s.Rows[row].Buttons)))
			s.Rows[row].Armed[k] = true
		}
	}
}

func (e *MapEditor) objectSpecRows() (popupSpec, []int, bool) {
	e.panels.hostable = false
	e.obj.viewOffer = false
	if e.modWindow && !e.prompt.open() && (!e.catalog || e.modOverCatalog()) {
		return e.modSpec()
	}
	if e.messagePage != 0 && (!e.objectBlocked() || !e.catalog && !e.objectHidden && e.speakerPicking()) {
		return e.messageSpec()
	}
	if v, ok := e.scriptVariableRow(e.selected); ok && !e.objectBlocked() {
		s, links, ok := e.variableSpec(v)
		e.objectBackButton(&s)
		return s, links, ok
	}
	if s, links, ok := e.groupSpec(); ok {
		e.objectBackButton(&s)
		e.panels.hostable = true
		return s, links, true
	}
	if s, ok := e.selectionSpec(); ok {
		return s, nil, true
	}
	// A record's cell pick moves its popup aside for the map click.
	if e.doc == nil || e.objectBlocked() || e.selected < 0 || e.selected >= len(e.doc.Records) {
		return popupSpec{}, nil, false
	}
	// The record's own window and a script record's window stand in the
	// right panel while it is shown; the windows above float.
	e.panels.hostable = true
	if v, ok := e.scriptPopup(); ok {
		s, links := e.scriptSpec(v)
		e.obj.data = -1
		e.objectBackButton(&s)
		return s, links, true
	}
	e.placedCorner(popObject, homeObjectMap)
	r := e.doc.Records[e.selected]
	e.loadObject()
	if e.obj.move != nil {
		r.Cell = *e.obj.move
	}
	wide := e.popupWide(popObject, 420)
	title := fmt.Sprintf("%s #%d", r.Kind, r.Index)
	if r.Name != "" {
		title = r.Name
	}
	s := popupSpec{Title: title, At: e.placedCorner(popObject, homeObjectMap), Width: wide, Bounds: e.popupBounds(), NoteInk: editorWarning}
	var links []int
	add := func(row popupRow, link int) {
		s.Rows, links = append(s.Rows, row), append(links, link)
	}
	// The person window moves its unit by a drag on the map; it shows no
	// cell row.
	if r.Spatial && !e.nativeUnitWindow() {
		add(popupRow{Kind: popupText, Label: e.word("editor.object.cell"), Value: fmt.Sprintf("%d,%d  %dx%d", r.Cell.X, r.Cell.Y, r.Size.X, r.Size.Y)}, 0)
	}
	e.objectFieldRows(&s, &links)
	if e.nativeUnitWindow() {
		// The person window lists the triggers that use the unit under its
		// fields on every section; a unit no trigger uses shows no list.
		if e.hasUsedBy() && !e.obj.usesShown {
			if rows := e.usedByRows(); len(rows) > 0 {
				e.scriptLinkSection(&s, &links, e.word("editor.usedby.title"), rows)
			}
		}
		e.obj.data = -1
		e.viewSwitchButton(&s)
		if aside := e.hostedAside(s); aside > 0 {
			s.Width += aside + popupPad
		}
		s.MaxRows = max(3, e.objectRowsShown(true)-3-len(s.Notes))
		height := max(450, layoutPopup(s, e.font, e.bodyFont()).Rect.Dy())
		// The basic view's larger card under the rows takes the height it
		// needs; the person window's other views keep their cap.
		most := 620
		if s.CardBelow {
			most = height
		}
		room := e.popupRoom().Dy() - 16
		naturalSize(&s, image.Pt(s.Width, min(most, height, room)))
		// The column under the card grows to hold the figure at the game's
		// size while the room allows.
		if l := layoutPopup(s, e.font, e.bodyFont()); s.Card != nil && l.CardRest.Min.Y >= l.Card.Max.Y {
			if short := e.unitFigureSize().Y - l.CardRest.Dy(); short > 0 {
				naturalSize(&s, image.Pt(s.Width, min(l.Rect.Dy()+short, room)))
			}
		}
		s.MaxRows = layoutPopup(s, e.font, e.bodyFont()).RowsFit
		e.clampObjectTop(&s, popupRowCount(s))
		return s, links, true
	}
	if e.hasUsedBy() && !e.obj.usesShown {
		e.scriptLinkSection(&s, &links, e.word("editor.usedby.title"), e.usedByRows())
	}
	e.obj.data = len(s.Rows)
	// The plain view keeps the record's data lines in a block that opens
	// and closes, closed at first.
	closed := false
	if data := (plainBlock{id: "data", fold: true}); e.obj.rowBlock != nil {
		e.obj.rowBlock[len(s.Rows)] = data.id
		add(blockRow(e.word("editor.object.data"), e.blockOpen(data)), 0)
		if closed = !e.blockOpen(data); closed {
			e.obj.data = -1
		}
	} else {
		add(popupRow{Kind: popupHeading, Label: e.word("editor.object.data")}, 0)
	}
	width := e.hostedWrap(s, wide-2*popupPad-4)
	if r.Preview != nil {
		width -= previewW + popupPad // the picture's column
	}
	var lines []InspectionLine
	if r.Warning != "" && len(r.Lines) == 0 {
		lines = append(lines, InspectionLine{Text: "! " + r.Warning, Role: "warning"})
	}
	lines = append(lines, InspectionLine{Text: r.Label})
	for _, d := range r.Details {
		lines = append(lines, InspectionLine{Text: d})
	}
	lines = append(lines, r.Lines...)
	if closed {
		lines = nil
	}
	for _, l := range lines {
		if l.Role == "heading" {
			add(popupRow{Kind: popupHeading, Label: l.Text}, 0)
			continue
		}
		kind := popupText
		if l.Reference > 0 {
			kind = popupLink
			l.Text = "> " + l.Text
		}
		for _, part := range editorTextLines(e.bodyFont(), l.Text, width) {
			add(popupRow{Kind: kind, Label: part, Ink: inspectionRoleColor(l.Role)}, l.Reference)
		}
	}
	e.objectBackButton(&s)
	e.viewSwitchButton(&s)
	if aside := e.hostedAside(s); aside > 0 {
		s.Width += aside + popupPad
	}
	s.MaxRows = e.objectRowsShown(len(s.Buttons) > 0 || e.objectHasFields())
	if !e.objectInPanel() {
		// A label over its field or a face takes more lines: the window
		// shows fewer rows and keeps inside the editor.
		e.clampObjectTop(&s, len(s.Rows))
		for s.MaxRows > popupMinRows && layoutPopup(s, e.font, e.bodyFont()).Rect.Dy() > e.popupBounds().Dy() {
			s.MaxRows--
		}
	}
	e.clampObjectTop(&s, len(s.Rows))
	return s, links, true
}

// clampObjectTop holds the object window's first shown row where its last
// rows still fill the floating window, count rows in all. A window in the
// right panel fits its own rows: hostedObject holds it there, so the tail
// the panel's height shows stays in reach.
func (e *MapEditor) clampObjectTop(s *popupSpec, count int) {
	if !e.objectInPanel() {
		e.objectTop = max(0, min(e.objectTop, count-s.MaxRows))
	}
	s.Top = e.objectTop
}

// ScrollDetails moves the object popup's rows down by steps wheel notches.
func (e *MapEditor) ScrollDetails(steps int) {
	e.objectTop = max(0, e.objectTop+3*steps)
}

// objectRowsShown is how many rows the object popup shows at this size.
func (e *MapEditor) objectRowsShown(buttons bool) int {
	pitch := fontHeight(e.bodyFont(), 15) + popupRowExtra
	avail := e.popupBounds().Dy() - (fontHeight(e.font, 15) + popupRowExtra) - 3*popupPad - 2*pitch
	if buttons {
		avail -= pitch + 4 + popupPad
	}
	return max(3, avail/pitch)
}

// promptSpec is the prompt and the message as one popup.
func (e *MapEditor) promptSpecBase() (popupSpec, bool) {
	if !e.prompt.open() && e.message == "" {
		return popupSpec{}, false
	}
	if e.prompt.kind == promptAdd {
		s, _ := e.addSpec()
		return s, true
	}
	switch e.prompt.kind {
	case promptLeave:
		return e.leaveSpec(), true
	case promptNotice:
		return e.noticeSpec(), true
	case promptNewMode:
		return e.newModeSpec(), true
	case promptAreaTrigger:
		return e.areaTriggerSpec(), true
	case promptAreaShared:
		return e.areaSharedSpec(), true
	case promptToGroup:
		return e.toGroupSpec(), true
	case promptGroupWarn:
		return e.groupWarnSpec(), true
	case promptScriptForm:
		return e.scriptFormSpec(), true
	}
	w := e.popupWide(popPrompt, min(560, e.window.X-16))
	s := popupSpec{Width: w, Bounds: e.popupBounds(), NoteInk: editorWarning, LabelWidth: 1}
	text := w - 2*popupPad - 60
	if e.prompt.open() {
		s.Title = e.promptTitle()
		s.Rows = e.promptInputRows(text)
		s.Buttons = []string{e.word("editor.button.ok"), e.word("editor.button.cancel")}
		if fields := e.promptFields(); e.prompt.fields() && e.prompt.field < len(fields) && fields[e.prompt.field].Meaning != "" {
			s.Notes = append(s.Notes, editorTextLines(e.bodyFont(), e.editorHostText(fields[e.prompt.field].Meaning), text)...)
		}
	} else {
		s.Title = e.word("editor.message.title")
		s.Buttons = []string{e.word("editor.button.ok")}
	}
	if e.message != "" {
		errors := editorTextLines(e.bodyFont(), e.editorHostText(e.message), text)
		if len(errors) > 6 {
			more := errors[5] + textfit.Mark
			if f := e.bodyFont(); f != nil {
				more = editorFit(f).Continued(errors[5], text)
			}
			errors = append(errors[:5], more)
		}
		s.Notes = append(s.Notes, errors...)
		s.Notes = append(s.Notes, e.word("editor.message.kept"))
	} else if e.multilinePrompt() {
		s.Notes = append(s.Notes, e.word("editor.prompt.multiline"))
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	s.At = e.promptCorner(centreQuestion, l.Rect.Size())
	return s, true
}

// movableSpec is the popup id as drawn: the list, the object or the prompt
// popup, or a floating side panel's or the toolbar's window (a docked panel
// does not move).
func (e *MapEditor) movableSpec(id editorPopupID) (popupSpec, bool) {
	switch id {
	case popList:
		return e.listSpec()
	case popObject:
		s, _, ok := e.objectSpec()
		return s, ok
	case popPrompt:
		return e.promptSpec()
	case popTree:
		return e.treeSpec()
	case popProps:
		return e.propsSpec()
	case popToolbar:
		return e.toolbarSpec()
	}
	return popupSpec{}, false
}

// controlTitleRect is the title bar of the front popup: the prompt, the object or
// the list popup.
func (e *MapEditor) controlTitleRect() (image.Rectangle, bool) {
	for _, id := range []editorPopupID{popPrompt, popObject, popList} {
		if s, ok := e.movableSpec(id); ok {
			return layoutPopup(s, e.font, e.bodyFont()).Title, true
		}
	}
	return image.Rectangle{}, false
}

// titlePress starts the editor's one popup drag when p lies on the title bar
// of popup id; an open drop-down over the bar takes the press instead.
func (e *MapEditor) titlePress(id editorPopupID, p image.Point) bool {
	if id == popObject && e.openDropdownHit(p) || id == popPrompt && e.areaDropdownHit(p) {
		return false
	}
	s, ok := e.movableSpec(id)
	if !ok {
		return false
	}
	if part, _ := layoutPopup(s, e.font, e.bodyFont()).hit(p); part != popupTitle {
		return false
	}
	e.drag, e.dragFrom, e.dragEdge = id, p, 0
	return true
}

func (e *MapEditor) promptTitle() string {
	if r, ok := e.addRunning(); ok {
		return r.title
	}
	switch {
	case e.prompt.fields():
		return e.fieldTitle()
	case e.prompt.kind == promptItemRecipe:
		return e.word("editor.prompt.recipe")
	case e.prompt.kind == promptSave:
		return e.word("editor.prompt.save")
	case e.prompt.kind == promptRectangle:
		return e.word("editor.prompt.newmap")
	case e.prompt.kind == promptSpeaker:
		return e.word("editor.speaker.add.name")
	case e.prompt.kind == promptForestShare:
		return e.word("editor.prompt.share")
	case e.prompt.kind == promptSeed:
		return e.seedTitle()
	}
	return e.word("editor.prompt.open")
}

var editorPopupInk = popupInk{Ground: editorPanel, Edge: editorGold, TitleGround: editorInk, Title: editorGold, Label: editorPaper, Value: editorLight}

// ChromeImage composes the window's chrome over a transparent window-size
// picture: the bar, the status strip, the list and object popups, the prompt
// and, last, an open drop-down. While a map drag has moved, the windows that
// float over the map are drawn faded (chromeFaded).
func (e *MapEditor) ChromeImage() *image.RGBA {
	pic := e.chromeCompose(e.dragFades())
	if layer := e.chromeFaded(); layer != nil {
		draw.Draw(pic, pic.Bounds(), layer, image.Point{}, draw.Over)
	}
	return pic
}

// chromeCompose draws the chrome in its order. With fade set the windows
// that float over the map are left out: the docked panels, the bar and the
// strip are drawn as they are, and chromeFaded draws the floating windows.
func (e *MapEditor) chromeCompose(fade bool) *image.RGBA {
	e.chromeWindows = e.chromeWindows[:0]
	pic := e.chromeBlank()
	bar := image.Rect(0, 0, e.window.X, e.barHeight())
	drawFrame(pic, stripFrame(bar, edgeBottomBit))
	for _, b := range e.barEntries() {
		if b.menu {
			drawDropdown(pic, e.font, nil, e.menuDropdown(b, false), nil, e.pointer, e.pointerOK)
			continue
		}
		drawPushButton(pic, e.font, pushButton{Rect: b.rect, Label: b.label, Literal: true, LabelRaise: labelRaise(e.font), Disabled: b.disabled, Pressed: b.on, Inside: b.on, Hover: e.pointerOK && e.pointer.In(b.rect), Press: &e.buttonPress})
	}
	e.drawToolbar(pic, false)
	status := e.statusRect()
	drawFrame(pic, stripFrame(status, edgeTopBit))
	if f := e.bodyFont(); f != nil {
		f.Draw(pic.SubImage(status).(*image.RGBA), editorClip(f, e.statusFitted(f, status.Dx()-16), status.Dx()-16), 8, status.Min.Y+5, editorPaper)
	}
	e.drawMapLabels(pic)
	e.drawPlayerLegend(pic)
	if fade {
		e.drawSidePanels(pic, panelsDocked)
		if s, ok := e.listSpec(); ok && e.reportDocked() {
			e.drawListWindow(pic, s)
		}
	} else {
		e.drawSidePanels(pic, panelsAll)
		e.drawFloating(pic, false)
	}
	if b, ok := e.openMenuEntry(); ok {
		d := e.menuDropdown(b, true)
		e.noteDropdown(d, e.font)
		drawDropdown(pic, e.font, nil, d, func(row, width int) string { return editorClip(e.font, d.List.Rows()[row].Text, width) }, e.pointer, e.pointerOK)
	}
	e.drawTip(pic)
	e.buttonPress.finish(pic)
	return pic
}

// drawFloating draws the windows that float over the map: with only set the
// floating panels first, then the floating toolbar, the list, the object
// window, the prompt and the open drop-downs.
func (e *MapEditor) drawFloating(pic *image.RGBA, only bool) {
	if only {
		e.drawSidePanels(pic, panelsFloating)
	}
	e.drawToolbar(pic, true)
	if s, ok := e.listSpec(); ok && !(only && e.reportDocked()) {
		e.drawListWindow(pic, s)
	}
	if s, _, ok := e.objectSpec(); ok && !e.panels.hosted {
		e.drawWindow(pic, s)
		e.drawObjectPreview(pic, s)
	}
	if s, ok := e.promptSpec(); ok {
		e.drawWindow(pic, s)
		e.drawAddPreview(pic, s)
	}
	e.drawOpenDropdown(pic)
	e.drawAreaDropdown(pic)
}

// ChromeWindow is one panel, floating window or open drop-down list the
// chrome drew: its window rectangle, and Text, the count of the frame's
// captured text calls drawn before it. A glyph whose TextFates index is below
// Text lies under the window.
type ChromeWindow struct {
	Rect image.Rectangle
	Text int
}

// ChromeWindows are the panels, floating windows and open drop-down lists the
// last drawn frame's chrome drew, in draw order. They are noted only while the
// chrome's text is captured; drawing never calls this.
func (e *MapEditor) ChromeWindows() []ChromeWindow { return slices.Clone(e.chromeWindows) }

// drawWindow draws popup s as one window of the chrome.
func (e *MapEditor) drawWindow(pic *image.RGBA, s popupSpec) {
	l := layoutPopup(s, e.font, e.bodyFont())
	e.noteWindow(l.Rect)
	drawPopup(pic, e.font, e.bodyFont(), nil, s, l, editorPopupInk, e.pointer, e.pointerOK, &e.buttonPress)
}

// noteDropdown notes the open list of drop-down d, laid out in f, as the
// frame drawDropdown draws around it.
func (e *MapEditor) noteDropdown(d dropdown, f *text.Font) {
	if d.Open && d.List != nil {
		l := d.listBox(f)
		e.noteWindow(l.Rect.Union(l.Bar()).Inset(-2))
	}
}

func (e *MapEditor) noteWindow(r image.Rectangle) {
	if text.Capturing() {
		e.chromeWindows = append(e.chromeWindows, ChromeWindow{r, text.CapturedLen()})
	}
}

// drawListWindow draws the list popup s with the links and the brush picture
// that belong to it.
func (e *MapEditor) drawListWindow(pic *image.RGBA, s popupSpec) {
	e.drawWindow(pic, s)
	e.drawGoalLinks(pic, s)
	e.drawBrushPreview(pic, s)
}

// previewW and previewH bound the record's picture in the object popup.
const previewW, previewH = 96, 64

// drawObjectPreview draws the selected record's picture at the right of its
// data, from the data heading down, when that heading is shown.
func (e *MapEditor) drawObjectPreview(pic *image.RGBA, s popupSpec) {
	if e.selected < 0 || e.selected >= len(e.doc.Records) {
		return // the group popup has no one record
	}
	r := e.doc.Records[e.selected]
	if r.Kind == KindUnit && s.Card != nil && s.Card.Char.Band == CharacterBandPerson {
		// The figure stands at the right of the room the layout leaves
		// beside the card, at the game's size where the room holds it.
		rest := layoutPopup(s, e.font, e.bodyFont()).CardRest
		left := max(rest.Min.X, rest.Max.X-max(compactPanelW, e.unitFigureSize().X))
		box := image.Rect(left, rest.Min.Y, rest.Max.X, rest.Max.Y).Intersect(pic.Bounds())
		if box.Empty() {
			return
		}
		var preview *image.RGBA
		if pictures := e.obj.unit.picture; pictures != nil {
			preview = pictures.Doll
		}
		if preview == nil || preview.Bounds().Empty() {
			if f := e.bodyFont(); f != nil {
				f.Draw(pic.SubImage(box).(*image.RGBA), e.word("editor.unit.look.missing"), box.Min.X, box.Min.Y, editorGold)
			}
			return
		}
		editorBlit(pic.SubImage(box).(*image.RGBA), preview, fitRect(preview.Bounds().Size(), box), false)
		return
	}
	if r.Preview == nil {
		return
	}
	preview := r.Preview.RGBA()
	if preview == nil || preview.Bounds().Empty() {
		return
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	for _, rr := range l.Rows {
		if rr.Index != e.obj.data {
			continue
		}
		box := image.Rect(rr.Rect.Max.X-previewW, rr.Rect.Min.Y, rr.Rect.Max.X, rr.Rect.Min.Y+previewH)
		last := l.Rows[len(l.Rows)-1].Rect.Max.Y
		clip := pic.SubImage(image.Rect(box.Min.X, box.Min.Y, box.Max.X, min(box.Max.Y, last))).(*image.RGBA)
		editorBlit(clip, preview, fitRect(preview.Bounds().Size(), box), false)
	}
}

// fitRect scales size into box, never up, anchored at its top right.
func fitRect(size image.Point, box image.Rectangle) screenRect {
	scale := min(1, min(float64(box.Dx())/float64(size.X), float64(box.Dy())/float64(size.Y)))
	w, h := float64(size.X)*scale, float64(size.Y)*scale
	return screenRect{X: float64(box.Max.X) - w, Y: float64(box.Min.Y), W: w, H: h}
}

// statusText is the strip's line: the cell under the pointer, the map's
// saved state and the mode's keys.
func (e *MapEditor) statusText() string {
	parts, help := e.statusParts(e.wholeSelectionNote())
	return statusJoin(parts, help)
}

// wholeSelectionNote is the group edit's note as the whole line names it.
func (e *MapEditor) wholeSelectionNote() string {
	return e.selectionNote(e.selection.refusals[:min(selectionShownRefusals, len(e.selection.refusals))])
}

// statusJoin is the strip's line of the lead parts and the help lines.
func statusJoin(parts, help []string) string {
	if len(help) == 0 {
		return strings.Join(parts, "   |   ")
	}
	return strings.Join(append(parts, strings.Join(help, "  /  ")), "   |   ")
}

// statusFitted is the line the strip draws in f within width: while the
// whole line is wider, the help loses its first line, so the last, which
// names F1 for every key, stays in view. A group edit's note that still does
// not fit names its first refusal alone, cut to the room left, and counts the
// rest; the last help line stays while that refusal keeps a part beside the
// cut mark, and goes when it does not.
func (e *MapEditor) statusFitted(f *text.Font, width int) string {
	parts, help := e.statusParts(e.wholeSelectionNote())
	line := statusJoin(parts, help)
	for f != nil && len(help) > 1 && f.Advance(line) > width {
		help = help[1:]
		line = statusJoin(parts, help)
	}
	if f == nil || f.Advance(line) <= width || len(e.selection.refusals) == 0 {
		return line
	}
	first := e.selection.refusals[0]
	for _, h := range [][]string{help, nil} {
		withFirst := func(r string) string {
			p, _ := e.statusParts(e.selectionNote([]string{r}))
			return statusJoin(p, h)
		}
		if l := withFirst(first); f.Advance(l) <= width {
			return l
		}
		room := width - f.Advance(withFirst(""))
		if cut := editorFit(f).Fit(first, room); strings.HasSuffix(cut, textfit.Mark) && len(cut) > len(textfit.Mark) {
			if l := withFirst(cut); f.Advance(l) <= width {
				return l
			}
		}
	}
	p, _ := e.statusParts(e.selectionNote(nil))
	return statusJoin(p, nil)
}

// StatusFit is the status line as the strip draws it at the window's size,
// its width and the whole line's width in the body font, and the strip's
// width, for a headless driver.
func (e *MapEditor) StatusFit() (line string, width, whole, limit int) {
	f := e.bodyFont()
	limit = e.statusRect().Dx() - 16
	line = e.statusFitted(f, limit)
	if f == nil {
		return line, 0, 0, limit
	}
	return line, f.Advance(line), f.Advance(e.statusText()), limit
}

// statusParts is the strip's lead parts, ending in the group's part with
// note, and the mode's help lines.
func (e *MapEditor) statusParts(note string) ([]string, []string) {
	var parts []string
	if e.mcp.last != "" {
		parts = append(parts, e.mcp.last)
	}
	if e.mcp.connection != "" {
		parts = append(parts, e.mcp.connection)
	}
	if e.hoverOK {
		parts = append(parts, fmt.Sprintf("%d,%d", e.hoverCell.X, e.hoverCell.Y))
		if h := e.hoverRecord(); h != "" {
			parts = append(parts, h)
		}
		if h, ok := e.hoverHeight(); ok {
			parts = append(parts, h)
		}
		if p, ok := e.hoverPassability(); ok {
			parts = append(parts, p)
		}
	}
	if e.doc != nil {
		mark := ""
		if e.Dirty() {
			mark = "* "
		}
		parts = append(parts, mark+e.editorHostText(e.doc.Source))
		if line := e.saveNoticeLine(); line != "" {
			parts = append(parts, line)
		}
		if e.saveNote != "" && e.saveNoteFor == e.doc.Source {
			parts = append(parts, e.editorHostText(e.saveNote))
		}
		if e.doc.AgainromOnly {
			parts = append(parts, e.word("editor.againrom.map"))
		}
	}
	if e.pick == pickScriptCell {
		parts = append(parts, e.word("editor.trigger.pickcell"))
	}
	switch {
	case e.pick == pickDrop:
		parts = append(parts, e.dropToolStatus())
	case e.pick == pickParamUnit || e.pick == pickParamArea:
		parts = append(parts, e.paramPickStatus()...)
	case e.pick == pickArea && e.obj.wiz != nil && e.obj.wiz.draw >= 0 && e.obj.wiz.square:
		parts = append(parts, e.word("editor.wizard.drawsquare"))
	case e.pick == pickArea && e.obj.wiz != nil && e.obj.wiz.draw >= 0:
		parts = append(parts, e.word("editor.wizard.drawbox"))
	case e.pick == pickArea:
		parts = append(parts, e.word("editor.area.draw"))
	case e.pick == pickTrapArea:
		parts = append(parts, e.word("editor.cast.areahint"))
	case e.pick == pickPaste:
		parts = append(parts, e.word("editor.selection.armed"))
	}
	parts = append(parts, e.selectionStatus(note)...)
	return parts, e.helpLines()
}

// drawChrome uploads the chrome over screen and reports the text calls the
// smoothed overlay draws.
func (e *MapEditor) drawChrome(screen *ebiten.Image) []text.DrawCall {
	pic, blit, calls, kept := e.chromePicture()
	log := &pixelLog{}
	log.reset(screen.Bounds())
	log.over(pic, image.Point{})
	log.end()
	e.textLayers = append(e.textLayers, textLayer{calls, screen.Bounds(), log})
	e.textCaptured, e.textKept = len(calls), len(kept)
	if e.chromeImage == nil || e.chromeImage.Bounds().Size() != pic.Bounds().Size() {
		if e.chromeImage != nil {
			e.chromeImage.Dispose()
		}
		e.chromeImage = ebiten.NewImage(pic.Bounds().Dx(), pic.Bounds().Dy())
	}
	if e.chromeUploaded != blit || e.chromeUploadedTo != e.chromeImage {
		e.chromeImage.WritePixels(blit.Pix)
		e.chromeUploaded, e.chromeUploadedTo = blit, e.chromeImage
	}
	screen.DrawImage(e.chromeImage, &ebiten.DrawImageOptions{})
	return kept
}

// chromePicture is the chrome as composed, the picture uploaded after the
// smoothed text's cells are settled, every captured text call and the calls
// the overlay draws.
func (e *MapEditor) chromePicture() (pic, blit *image.RGBA, calls, kept []text.DrawCall) {
	if !e.textSmoothingEnabled {
		pic = e.ChromeImage()
		return pic, pic, nil, nil
	}
	fade := e.dragFades()
	calls = text.Record(func() { pic = e.chromeCompose(fade) })
	if fade {
		e.chromeSettled = chromeSettled{}
	} else if c := &e.chromeSettled; c.pic != nil && c.pic.Rect == pic.Rect && bytes.Equal(c.pic.Pix, pic.Pix) && sameCalls(c.calls, calls, true) {
		// The composition is the one already settled: its settled upload and
		// kept text are the answer, and this picture is free for the next frame.
		e.chromeSpare = pic
		return c.pic, c.blit, c.calls, c.kept
	}
	blit = image.NewRGBA(pic.Bounds())
	copy(blit.Pix, pic.Pix)
	kept = textsmooth.Settle(blit, calls)
	composed := len(e.chromeWindows)
	if layer := e.chromeFaded(); layer != nil {
		// The faded windows lie over every call the chrome captured.
		for i := composed; i < len(e.chromeWindows); i++ {
			e.chromeWindows[i].Text = len(calls)
		}
		// The faded windows keep their text as plain pixels, so it fades
		// with them.
		draw.Draw(pic, pic.Bounds(), layer, image.Point{}, draw.Over)
		draw.Draw(blit, blit.Bounds(), layer, image.Point{}, draw.Over)
	}
	kept = text.MarkErased(kept)
	if !fade {
		e.chromeSettled = chromeSettled{pic: pic, blit: blit, calls: calls, kept: kept}
	}
	return pic, blit, calls, kept
}

// chromeSettled is the last composition chromePicture settled: the picture
// and the text calls it was composed from, the upload and the kept text it
// settled to. A composition that equals it is not settled again.
type chromeSettled struct {
	pic, blit *image.RGBA
	calls     []text.DrawCall
	kept      []text.DrawCall
}

// chromeBlank is a transparent window-size picture to compose the chrome on:
// the picture of an unchanged frame, which nothing else holds, when there is
// one.
func (e *MapEditor) chromeBlank() *image.RGBA {
	r := image.Rect(0, 0, e.window.X, e.window.Y)
	if spare := e.chromeSpare; spare != nil && spare.Rect == r {
		e.chromeSpare = nil
		clear(spare.Pix)
		return spare
	}
	e.chromeSpare = nil
	return image.NewRGBA(r)
}

// chromeHit is the foremost chrome part under p. Within a hover sample (chromePointer)
// it is built once: the popup specs under it cannot change before the sample
// ends.
func (e *MapEditor) chromeHit(p image.Point) editorPopupID {
	if e.hitKeep.on && e.hitKeep.held && e.hitKeep.at == p {
		return e.hitKeep.id
	}
	id := e.chromeHitBuilt(p)
	if e.hitKeep.on {
		e.hitKeep.at, e.hitKeep.id, e.hitKeep.held = p, id, true
	}
	return id
}

func (e *MapEditor) chromeHitBuilt(p image.Point) editorPopupID {
	if b, ok := e.openMenuEntry(); ok && e.menuDropdown(b, true).Covers(e.font, p) {
		return popMenu
	}
	if s, ok := e.promptSpec(); ok && (p.In(layoutPopup(s, e.font, e.bodyFont()).Rect) || e.areaDropdownHit(p)) {
		return popPrompt
	}
	object, _, objectOK := e.objectSpec()
	hosted := e.panels.hosted
	if objectOK && (!hosted && p.In(layoutPopup(object, e.font, e.bodyFont()).Rect) || e.openDropdownHit(p)) {
		return popObject
	}
	if s, ok := e.listSpec(); ok && p.In(layoutPopup(s, e.font, e.bodyFont()).Rect) {
		return popList
	}
	if s, ok := e.toolbarSpec(); ok && p.In(layoutPopup(s, e.font, e.bodyFont()).Rect) {
		return popToolbar
	}
	for _, id := range []editorPopupID{popProps, popTree} {
		if i := panelOfRow(id); e.panelFloats(i) {
			if s, ok := e.movableSpec(id); ok && p.In(layoutPopup(s, e.font, e.bodyFont()).Rect) {
				return id
			}
		}
	}
	// A hosted window takes a press on what it draws, the rectangle its
	// layout fills, as a floating one does.
	if objectOK && hosted && p.In(layoutPopup(object, e.font, e.bodyFont()).Rect) {
		return popObject
	}
	if p.In(e.sidePanelRect(panelRight)) {
		return popProps
	}
	if p.In(e.sidePanelRect(panelLeft)) {
		return popTree
	}
	if p.In(e.sidePanelRect(panelBottom)) {
		return popReport
	}
	if _, ok := e.areaLabelAt(p); ok && e.pick == pickNone && !e.blocksCanvas() {
		return popArea
	}
	if p.Y < e.barHeight() {
		return popBar
	}
	if p.In(e.toolbarRect()) {
		return popToolbar
	}
	if p.Y >= e.statusRect().Min.Y {
		return popStatus
	}
	return popNone
}

// chromePointer routes a pointer sample to the chrome and reports whether
// the chrome took it. A gesture that started on the map stays with the map;
// one that started on the chrome stays with the chrome.
func (e *MapEditor) chromePointer(in Input, now time.Time) bool {
	p := image.Pt(in.CursorX, in.CursorY)
	e.pointer, e.pointerOK = p, true
	e.buttonPress.pointer = p
	if !in.PrimaryDown {
		e.buttonPress.latch.Clear()
	}
	e.hitKeep = chromeHitKeep{on: !in.PrimaryDown && !in.SecondaryDown && !wheeled(in) && e.drag == popNone && !e.uiDown && !e.down}
	defer func() { e.hitKeep = chromeHitKeep{} }()
	e.observeTip(p, now, in.PrimaryDown)
	pressed := in.PrimaryDown && !e.down
	if pressed && e.tree.typing && !p.In(e.treeSearchBox()) {
		e.tree.typing = false
	}
	if pressed {
		e.searchPressOff(p)
	}
	if e.drag == popNone && e.inputPointer(in) {
		return true
	}
	if e.scrollPointer(in, p, pressed) {
		return true
	}
	if e.drag != popNone {
		if in.PrimaryDown {
			d := p.Sub(e.dragFrom)
			e.dragFrom = p
			switch {
			case e.dragCard != 0:
				if wheeled(in) {
					e.chromeWheel(popObject, in)
				}
			case e.dragEdge != 0:
				e.resizePopup(d)
			case e.pullPanel(d):
			default:
				e.dragPopup(d)
			}
		} else {
			if e.dragCard != 0 {
				e.dropCard()
			}
			if _, ok := e.panelOf(e.drag); (ok || e.keptWindow(e.drag) != "") && e.dragEdge != 0 || e.dragsFloatingPanel() {
				e.savePanels()
			}
			if e.panels.pull.on {
				e.clicks.release()
			}
			e.panels.pull = pullState{}
			e.drag, e.dragEdge, e.dragCard = popNone, 0, 0
			e.uiDown = false
		}
		e.down = in.PrimaryDown
		return true
	}
	if e.uiDown {
		if !in.PrimaryDown {
			e.uiDown = false
			e.clicks.release()
		}
		e.down = in.PrimaryDown
		return true
	}
	if e.down {
		return false
	}
	target := e.chromeHit(p)
	modal := e.prompt.open() && !e.addPicking() || e.message != ""
	barWindow := false
	if target == popBar && e.barWindow() != "" {
		for _, b := range e.barEntries() {
			if !b.menu && !b.disabled && p.In(b.rect) {
				barWindow = true
				break
			}
		}
	}
	if target == popNone && !modal {
		return false
	}
	if wheeled(in) {
		e.chromeWheel(target, in)
	}
	if pressed {
		if !modal || target == popPrompt || target == popMenu || barWindow {
			e.buttonPress = buttonPress{pointer: p, capturing: true}
			pic := e.ChromeImage()
			e.buttonPress.begin(pic)
		}
		e.down, e.uiDown = true, true
		double := e.clicks.press(p, now)
		if e.menu != "" && target != popMenu {
			e.menu = ""
			if target != popBar && target != popToolbar {
				return true
			}
		}
		if modal && target != popPrompt && target != popMenu && !barWindow {
			return true
		}
		e.chromePress(target, p, double)
	}
	return true
}

// chromeWheel scrolls the popup under the pointer.
func (e *MapEditor) chromeWheel(target editorPopupID, wheel wheelSource) {
	switch target {
	case popList:
		if s, ok := e.listSpec(); ok && e.notesWheel(layoutPopup(s, e.font, e.bodyFont()), wheel) {
			return
		} else if ok && s.Grid != nil {
			if l := layoutPopup(s, e.font, e.bodyFont()); e.pointer.In(l.Grid) || e.pointer.In(l.GridBar) {
				if rows := editorWheelRows(&e.listWheel, wheel); e.listKind() == "brush" {
					if e.forestOn() {
						e.forestGridWheel(l, rows)
					} else if e.brush.mode == brushObjects {
						e.placeGridWheel(l, rows)
					}
				} else if e.filter != editorPlayersFilter {
					e.placeGridWheel(l, rows)
				}
				return
			}
		}
		if e.model != nil && e.listKind() != "panel" {
			rows := editorWheelRows(&e.listWheel, wheel)
			if e.listKind() == "catalog" {
				e.catalogStep(rows)
			} else {
				e.model.Move(rows)
			}
		}
	case popObject:
		if !e.objectWheelDropdown(wheel) {
			e.objectTop = max(0, e.objectTop+editorWheelRows(&e.objectWheel, wheel))
		}
	case popMenu:
		if e.menuModel != nil {
			e.menuModel.Move(editorWheelRows(&e.listWheel, wheel))
		}
	case popTree:
		if e.tree.model != nil {
			e.tree.model.Scroll(editorWheelRows(&e.listWheel, wheel))
		}
	case popPrompt:
		if !e.areaWheel(wheel) {
			e.addWheel(wheel)
		}
	}
}

// chromePress is a press on a chrome part.
func (e *MapEditor) chromePress(target editorPopupID, p image.Point, double bool) {
	if e.resizePress(target, p) || e.panelTitlePress(target, p, double) || e.titlePress(target, p) {
		return
	}
	switch target {
	case popTree:
		e.treePress(p)
	case popProps:
		e.propsPress(p)
	case popReport:
		e.reportPress(p)
	case popToolbar:
		e.toolbarPress(p)
	case popMenu:
		b, _ := e.openMenuEntry()
		d := e.menuDropdown(b, true)
		switch part, row := d.hit(e.font, p); part {
		case dropdownToggle:
			if e.menu != searchKey {
				e.menu = ""
			}
		case dropdownRow:
			if e.menu == searchKey {
				e.chooseHit(row)
				return
			}
			items := e.menuItems(e.menu)
			if row < len(items) && items[row].key != "" && !items[row].off {
				e.menuAct(items[row].key)
			}
		}
	case popBar:
		for _, b := range e.barEntries() {
			if p.In(b.rect) && !b.disabled {
				e.barAct(b.key)
			}
		}
	case popPrompt:
		if e.prompt.kind == promptAdd {
			e.addPress(p, double)
			return
		}
		switch e.prompt.kind {
		case promptLeave:
			e.leavePress(p)
			return
		case promptNotice:
			e.noticePress(p)
			return
		case promptNewMode:
			e.newModePress(p)
			return
		case promptAreaTrigger:
			e.areaTriggerPress(p)
			return
		case promptAreaShared:
			e.areaSharedPress(p)
			return
		case promptToGroup:
			e.toGroupPress(p, double)
			return
		case promptGroupWarn:
			e.groupWarnPress(p)
			return
		case promptScriptForm:
			e.scriptFormPress(p)
			return
		}
		s, _ := e.promptSpec()
		part, n := layoutPopup(s, e.font, e.bodyFont()).hit(p)
		switch {
		case part == popupClose || part == popupButton && n == 1:
			e.cancelPopup()
		case part == popupButton && n == 0:
			if e.prompt.open() {
				_ = e.Key("enter")
			} else {
				e.cancelPopup(popPrompt)
			}
		}
	case popObject:
		e.objectPress(p)
	case popArea:
		if t, ok := e.areaLabelAt(p); ok {
			e.selectTrigger(t, false)
		}
	case popList:
		if e.listKind() == "panel" {
			e.panelPress(p)
			return
		}
		if e.groupsShown() {
			e.groupsPress(p)
			return
		}
		s, _ := e.listSpec()
		l := layoutPopup(s, e.font, e.bodyFont())
		if len(e.report.goals) > 0 {
			for row := range e.report.goals {
				if r := e.goalButton(l, row); !r.Empty() && p.In(r) {
					e.openGoal(row)
					return
				}
			}
		}
		switch part, n := l.hit(p); part {
		case popupTab:
			e.placeSwitchTo(n)
		case popupClose:
			if e.reportDocked() {
				e.togglePanel(panelBottom)
			} else {
				e.cancelPopup(popList)
			}
		case popupRowButton:
			if row, k, ok := l.rowButtonAt(p); ok && e.listKind() == "brush" {
				e.brushRowButton(row, k)
			}
		case popupGridPart:
			switch {
			case e.listKind() == "brush" && e.brush.mode == brushHeight:
				e.heightModePress(n)
			case e.listKind() == "brush" && e.forestOn():
				e.forestGridPress(n)
			case e.listKind() == "brush":
				e.placeChoose(n)
			case e.filter == editorPlayersFilter:
				e.relationPress(n)
			default:
				e.placeChoose(n)
			}
		case popupListRow:
			top, count := e.model.Visible()
			if n >= count {
				return
			}
			row := top + n
			double = double && row == e.lastRow
			e.lastRow = row
			e.listChoose(row, e.listKind() != "catalog" || double)
		case popupButton:
			if e.listKind() == "catalog" {
				if n == 0 {
					e.catalogOpen()
				} else {
					_ = e.Key("open")
				}
			} else if bs := e.listButtons(); n < len(bs) {
				switch bs[n].key {
				case "drop":
					e.scriptKey("drop")
				case "traparea":
					_ = e.Key("traparea")
				case "delete":
					_ = e.Key("delete")
				case "add":
					row, _ := e.listAdd()
					e.openAdd(row.Kind)
				case scenarioNew[0].key:
					for _, kind := range scenarioNew {
						if kind.key == bs[n].key {
							e.addNow(kind.kind)
						}
					}
				case "ruined":
					e.place.ruined = !e.place.ruined
				}
			}
		}
	}
}

// objectPress is a press on the object popup.
func (e *MapEditor) objectPress(p image.Point) {
	if e.openDropdownPress(p) {
		return
	}
	if e.modPress(p) {
		return
	}
	if _, ok := e.selectionSpec(); ok {
		e.selectionPress(p)
		return
	}
	s, links, _ := e.objectSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	switch part, n := l.hit(p); part {
	case popupTab:
		e.obj.section, e.obj.open, e.obj.focus, e.objectTop = n, -1, -1, 0
	case popupClose:
		if e.panels.hosted || e.panelFloats(panelRight) && e.panels.hostable {
			e.togglePanel(panelRight)
			return
		}
		e.cancelPopup()
	case popupRowGrip:
		if e.messagePage != 0 {
			e.obj.open = -1
			e.drag, e.dragFrom, e.dragEdge, e.dragCard = popObject, p, 0, n+1
		}
	case popupRowButton:
		if s.Rows[n].Kind == popupDirection {
			if _, k, ok := l.rowButtonAt(p); ok {
				e.objectRowPress(part, n)
				e.chooseObjectChoice(k)
			}
			return
		}
		if e.messagePage != 0 {
			if row, k, ok := layoutPopup(s, e.font, e.bodyFont()).rowButtonAt(p); ok {
				e.messageRowButton(row, k)
			}
			return
		}
		if _, ok := e.scriptVariableRow(e.selected); ok {
			return
		}
		if _, ok := e.scriptPopup(); ok {
			if row, k, ok := layoutPopup(s, e.font, e.bodyFont()).rowButtonAt(p); ok {
				e.scriptRowButton(row, k)
			}
			return
		}
		e.objectRowPress(part, n)
	case popupRowLine, popupRowControl:
		if f, ok := e.obj.rowOpen[n]; ok && s.Rows[n].Kind == popupLink {
			e.openParameter(f)
			return
		}
		if n < len(links) && links[n] > 0 {
			e.followReference(links[n])
			return
		}
		if n < len(links) && links[n] < 0 {
			if e.hasUsedBy() && e.doc.Records[-links[n]-1].Kind.Scripted() {
				e.openObjectUse(-links[n] - 1)
			} else {
				e.openObjectRecord(-links[n]-1, true)
			}
			return
		}
		if _, ok := e.scriptVariableRow(e.selected); ok {
			return
		}
		e.objectRowPress(part, n)
	case popupSwitchButton:
		e.switchView()
	case popupButton:
		if len(e.objectBack) > 0 && n == len(s.Buttons)-1 {
			e.backObject()
			return
		}
		if v, ok := e.scriptVariableRow(e.selected); ok {
			e.variableButton(v, n)
			return
		}
		if e.groupSel.set && e.selected < 0 {
			if n == 0 {
				e.removeGroupOf(e.groupSel)
			}
			return
		}
		if v, ok := e.scriptPopup(); ok {
			e.scriptButton(v, n)
			return
		}
		e.objectButton(n)
	}
}

// hoverRecord is the hover line of the first record on the hovered cell that
// carries one.
func (e *MapEditor) hoverRecord() string {
	if e.doc == nil {
		return ""
	}
	for _, r := range e.doc.Records {
		if r.Spatial && r.Hover != "" && r.Cell == e.hoverCell {
			return r.Hover
		}
	}
	return ""
}

// hoverAt records the cell under the frame position p for the status strip.
func (e *MapEditor) hoverAt(p image.Point) {
	e.hoverOK = false
	if e.doc == nil || p.X < 0 || p.Y < 0 || p.X >= e.doc.Viewer.cam.ViewW || p.Y >= e.doc.Viewer.cam.ViewH {
		return
	}
	if x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y)); ok && x >= 0 && y >= 0 && x < e.doc.Width && y < e.doc.Height {
		e.hoverCell, e.hoverOK = image.Pt(x, y), true
	}
}

// gridLines are the grid button's grid over the map as editor lines.
func (e *MapEditor) gridLines() []editorLine {
	if !e.view.grid {
		return nil
	}
	v := e.doc.Viewer
	was := v.showGrid
	v.showGrid = true
	segs := v.gridScreenSegments()
	v.showGrid = was
	out := make([]editorLine, 0, len(segs))
	for _, s := range segs {
		out = append(out, editorLine{float32(s.X0), float32(s.Y0), float32(s.X1), float32(s.Y1), color.RGBA{0x60, 0x58, 0x48, 255}})
	}
	return out
}

// chromeHitKeep holds the chrome hit of one point for the length of a hover
// sample, a pointer sample with no button, wheel or drag, which changes no
// popup.
type chromeHitKeep struct {
	on, held bool
	at       image.Point
	id       editorPopupID
}
