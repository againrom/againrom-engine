package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
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
	popArea // a trigger area's label on the map
)

// editorChrome is the chrome's state: the fonts, where each popup stands, the
// list popup's model, the open menu and a press in progress.
type editorChrome struct {
	body                   *text.Font
	cardFont               *text.Font
	book                   words.Book
	hasBook                bool
	listShown              bool
	objectHidden           bool
	listAt, objectAt       image.Point
	listPlaced, objPlaced  bool
	model                  *Picker
	modelKey               string
	modelSync              int
	menu                   string
	menuModel              *Picker
	drag                   editorPopupID
	dragFrom               image.Point
	uiDown                 bool
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
	obj                    objectEdit
	groupSel               editorGroup
	exitAsked              bool
	saveThen               *leaveStep // the step a Save As asked from the question runs once saved
	objectBack             []objectPlace
	messagePage            uint32
}

// editorView is the View menu's switches over the map's annotations.
type editorView struct {
	hideDrops, hideFootprints, hideIssues, grid, hideAreas bool
}

// SetBodyFont sets the plain font of rows, labels, values and report text.
// Titles, the top bar and headings keep the editor's first font.
func (e *MapEditor) SetBodyFont(f *text.Font) { e.body = f }

func (e *MapEditor) SetCardFont(f *text.Font) { e.cardFont = f }

// SetWords sets the word table the editor's own words are read from.
func (e *MapEditor) SetWords(b words.Book) { e.book, e.hasBook = b, true }

func (e *MapEditor) bodyFont() *text.Font {
	if e.body != nil {
		return e.body
	}
	return e.font
}

// word is the editor's word of id in the editor's language, in the font's
// encoding.
func (e *MapEditor) word(id string) string {
	b := e.book
	if !e.hasBook {
		b = words.For("")
	}
	return e.editorHostText(b.Text(id))
}

// barHeight is the top bar's height; the status strip has the same height, so
// the map frame is the window less both and sits at scale 1.
func (e *MapEditor) barHeight() int {
	return max(fontHeight(e.font, 15), fontHeight(e.bodyFont(), 15)) + 10
}

// mapArea is the window rectangle the map fills.
func (e *MapEditor) mapArea() image.Rectangle {
	h := e.barHeight()
	return image.Rect(0, h, e.window.X, max(h+1, e.window.Y-h))
}

func (e *MapEditor) statusRect() image.Rectangle {
	return image.Rect(0, e.mapArea().Max.Y, e.window.X, e.window.Y)
}

// barEntry is one entry of the top bar.
type barEntry struct {
	key, label string
	menu       bool
	disabled   bool
	rect       image.Rectangle
}

func (e *MapEditor) barEntries() []barEntry {
	entries := []barEntry{
		{key: "map", label: e.word("editor.bar.map"), menu: true},
		{key: "units", label: e.word("editor.bar.units")},
		{key: "structures", label: e.word("editor.bar.structures")},
		{key: "groups", label: e.word("editor.bar.groups")},
		{key: "sacks", label: e.word("editor.bar.sacks")},
		{key: "players", label: e.word("editor.bar.players")},
		{key: "scripts", label: e.word("editor.list.scripts")},
		{key: "view", label: e.word("editor.bar.view"), menu: true},
		{key: "undo", label: e.word("editor.bar.undo"), disabled: !e.editable() || !e.doc.Edits.CanUndo()},
		{key: "redo", label: e.word("editor.bar.redo"), disabled: !e.editable() || !e.doc.Edits.CanRedo()},
		{key: "save", label: e.word("editor.bar.save"), disabled: !e.editable()},
	}
	h := e.barHeight()
	x := 4
	for i := range entries {
		w := 16
		if e.font != nil {
			w += e.font.Advance(entries[i].label)
		}
		if entries[i].menu {
			w += h
		}
		if i > 0 && i < 7 && e.doc == nil || i >= 8 && e.doc == nil {
			entries[i].disabled = true
		}
		entries[i].rect = image.Rect(x, 3, x+w, h-3)
		x += w + 4
	}
	return entries
}

// BarRect is the window rectangle of the bar entry key ("map", "units",
// "view", "undo" and the rest): a headless driver presses it.
func (e *MapEditor) BarRect(key string) (image.Rectangle, bool) {
	for _, b := range e.barEntries() {
		if b.key == key {
			return b.rect, true
		}
	}
	return image.Rectangle{}, false
}

// MenuRowRect is the window rectangle of row n of the open menu.
func (e *MapEditor) MenuRowRect(n int) (image.Rectangle, bool) {
	b, ok := e.openMenuEntry()
	if !ok || n < 0 || n >= len(e.menuItems(b.key)) {
		return image.Rectangle{}, false
	}
	d := e.menuDropdown(b, true)
	return d.listBox(e.font).Row(n), true
}

// ListRowRect is the window rectangle of row n of the open list popup, when
// that row is shown.
func (e *MapEditor) ListRowRect(n int) (image.Rectangle, bool) {
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

func (e *MapEditor) ListRowCount() int {
	if s, ok := e.listSpec(); ok && s.List != nil {
		return s.List.Len()
	}
	return 0
}

// FieldRect is the window rectangle of the control of field n of the object
// popup, when that row is shown.
func (e *MapEditor) FieldRect(n int) (image.Rectangle, bool) {
	s, _, ok := e.objectSpec()
	if !ok || n < 0 || n >= len(e.obj.fields) {
		return image.Rectangle{}, false
	}
	for _, r := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if r.Index == e.fieldRow(n) {
			return r.Control, true
		}
	}
	return image.Rectangle{}, false
}

// GroupMemberRect is the window rectangle of member row n of the group
// popup, when that row is shown.
func (e *MapEditor) GroupMemberRect(n int) (image.Rectangle, bool) {
	s, _, ok := e.groupSpec()
	if !ok || n < 0 {
		return image.Rectangle{}, false
	}
	for _, r := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if r.Index == 2+n {
			return r.Rect, true
		}
	}
	return image.Rectangle{}, false
}

// menuItem is one row of a bar menu.
type menuItem struct {
	key, label string
}

func (e *MapEditor) menuItems(menu string) []menuItem {
	mark := func(on bool, id string) string {
		if on {
			return "[x] " + id
		}
		return "[ ] " + id
	}
	switch menu {
	case "map":
		return []menuItem{
			{"new", e.word("editor.menu.new")},
			{"catalog", e.word("editor.menu.open")},
			{"open", e.word("editor.menu.file")},
			{"saveas", e.word("editor.menu.save")},
			{"saveto", e.word("editor.menu.saveas")},
			{"testplay", e.word("editor.menu.testplay")},
			{"panel", e.word("editor.menu.panel")},
			{"close", e.word("editor.menu.exit")},
		}
	case "view":
		return []menuItem{
			{"drops", mark(!e.view.hideDrops, e.word("editor.view.drops"))},
			{"footprints", mark(!e.view.hideFootprints, e.word("editor.view.footprints"))},
			{"issues", mark(!e.view.hideIssues, e.word("editor.view.issues"))},
			{"grid", mark(e.view.grid, e.word("editor.view.grid"))},
			{"areas", mark(!e.view.hideAreas, e.word("editor.view.areas"))},
			{"report", e.word("editor.view.report")},
			{"fit", e.word("editor.view.fit")},
			{"drawarea", e.word("editor.view.drawarea")},
			{"records", e.word("editor.view.records")},
		}
	}
	return nil
}

// menuDropdown is the open menu as a drop-down anchored on its bar entry.
func (e *MapEditor) menuDropdown(entry barEntry, open bool) dropdown {
	d := dropdown{Rect: entry.rect, Text: entry.label, Fill: editorPanel, Ink: editorPaper, ListWidth: 240, Disabled: entry.disabled}
	if open && e.menu == entry.key {
		items := e.menuItems(entry.key)
		rows := make([]PickerRow, len(items))
		for i, it := range items {
			rows[i] = PickerRow{Text: it.label, Choosable: true}
		}
		if e.menuModel == nil {
			e.menuModel = NewPicker(rows)
		}
		e.menuModel.SetRows(rows)
		e.menuModel.SetWindow(len(rows))
		d.Open, d.List, d.Rows = true, e.menuModel, len(rows)
	}
	return d
}

func (e *MapEditor) openMenuEntry() (barEntry, bool) {
	for _, b := range e.barEntries() {
		if b.menu && b.key == e.menu {
			return b, true
		}
	}
	return barEntry{}, false
}

// barAct is what a bar entry does.
func (e *MapEditor) barAct(key string) {
	e.endFieldTyping()
	switch key {
	case "map", "view":
		if e.menu == key {
			e.menu = ""
		} else {
			e.menu = key
			e.menuModel = nil
		}
	case "units":
		e.showRecords(1)
	case "structures":
		e.showRecords(2)
	case "groups":
		e.showRecords(editorGroupsFilter)
	case "sacks":
		e.showRecords(3)
	case "players":
		e.showRecords(editorPlayersFilter)
	case "scripts":
		e.showRecords(editorScriptsFilter)
	case "undo", "redo":
		_ = e.Key(key)
	case "save":
		_ = e.Key("saveas")
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
	case "areas":
		e.view.hideAreas = !e.view.hideAreas
	default:
		if e.Key(key) == ebiten.Termination {
			e.exitAsked = true
		}
	}
}

// ExitAsked reports that a menu row asked to close the editor; Update then
// ends the run.
func (e *MapEditor) ExitAsked() bool { return e.exitAsked }

// showRecords opens the list popup on the records filter shows.
func (e *MapEditor) showRecords(filter int) {
	if e.doc == nil {
		return
	}
	e.rememberListCancel("records")
	e.leave(leaveDrag | leavePick | leaveReport | leaveCatalog | leaveBrush)
	e.message = ""
	e.filter, e.listShown = filter, true
	if filter == editorPlayersFilter || filter == editorScriptsFilter {
		if rows := e.rows(); len(rows) > 0 && !e.selectedIn(rows) {
			e.Select(rows[0], false)
		} else if len(rows) == 0 && filter == editorScriptsFilter {
			e.selected = -1
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
		e.listShown, e.filter = false, 0
	}
}

// listKind is what the list popup shows: the map picker, the report or
// script panel, the brush palette, a records list, or nothing.
func (e *MapEditor) listKind() string {
	switch {
	case e.catalog:
		return "catalog"
	case e.report.open:
		return "report"
	case e.brushActive():
		return "brush"
	case e.listShown && e.doc != nil:
		return "records"
	case e.panel && e.doc != nil:
		return "panel"
	}
	return ""
}

// catalogRows are the map picker's rows: a heading, the new-map sizes, a
// heading, the installed maps. The second result maps a row to its map, -1
// for a heading.
func (e *MapEditor) catalogRows() ([]PickerRow, []int) {
	var rows []PickerRow
	var maps []int
	add := func(heading string, fresh bool) {
		first := true
		for i, m := range e.maps {
			if strings.HasPrefix(m.Key, "new/") != fresh {
				continue
			}
			if first {
				rows, maps = append(rows, PickerRow{Text: heading}), append(maps, -1)
				first = false
			}
			label := m.Label
			if fresh { // under its heading a new map is its size: "32 x 32"
				label = strings.Replace(strings.TrimPrefix(m.Key, "new/"), "x", " x ", 1)
			}
			rows, maps = append(rows, PickerRow{Text: "  " + label, Choosable: true}), append(maps, i)
		}
	}
	add(e.word("editor.picker.new"), true)
	add(e.word("editor.picker.installed"), false)
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
		for _, s := range e.brushRows() {
			rows = append(rows, PickerRow{Text: s, Choosable: true})
		}
		sel = e.brush.sel[e.brush.mode]
	case "records":
		if e.filter == editorScriptsFilter {
			headings := []string{e.word("editor.scripts.triggers"), e.word("editor.scripts.variables"), e.word("editor.scripts.conditions"), e.word("editor.scripts.actions")}
			for section, ids := range e.scriptSections() {
				text := headings[section]
				if len(ids) == 0 {
					text = fmt.Sprintf(e.word("editor.scripts.emptyheading"), text)
				}
				rows = append(rows, PickerRow{Text: text})
				for _, id := range ids {
					if id == e.selected {
						sel = len(rows)
					}
					rows = append(rows, PickerRow{Text: e.scriptRowText(id, section), Choosable: true})
				}
			}
			break
		}
		if e.filter == editorGroupsFilter {
			for n, g := range e.groups() {
				rows = append(rows, PickerRow{Text: e.groupLabel(g), Choosable: true})
				if g.player == e.groupSel.player && g.group == e.groupSel.group {
					sel = n
				}
			}
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
// cell.
func (e *MapEditor) recordSummary(id int) string {
	r := e.doc.Records[id]
	s := fmt.Sprintf("%s #%d  %s", r.Kind, r.Index, r.Label)
	if r.Player > 0 {
		s += fmt.Sprintf("  P%d", r.Player)
	}
	if r.Spatial {
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
	pitch := fontHeight(e.bodyFont(), 15) + listPitchExtra
	avail := e.mapArea().Dy() - 4*(fontHeight(e.bodyFont(), 15)+popupRowExtra) - 40
	return max(3, min(18, avail/pitch))
}

// visibleRows is the list popup's window in rows.
func (e *MapEditor) visibleRows() int { return e.listWindow() }

func (e *MapEditor) popupBounds() image.Rectangle {
	return image.Rect(0, e.barHeight(), e.window.X, e.window.Y)
}

// listSpec is the list popup as data.
func (e *MapEditor) listSpec() (popupSpec, bool) {
	kind := e.listKind()
	if kind == "" {
		return popupSpec{}, false
	}
	if kind == "panel" {
		s, _ := e.panelSpec()
		return s, true
	}
	m := e.syncList(kind, e.listWindow())
	if !e.listPlaced {
		e.listAt, e.listPlaced = image.Pt(8, e.barHeight()+8), true
	}
	s := popupSpec{Title: e.listTitle(kind), At: e.listAt, Width: 400, Bounds: e.popupBounds(), List: m, ListRows: e.listWindow(), NoteInk: editorPaper}
	s.ListText = func(row, width int) string {
		rows := m.Rows()
		if row < 0 || row >= len(rows) {
			return ""
		}
		return editorClip(e.bodyFont(), rows[row].Text, width)
	}
	switch kind {
	case "report":
		if len(e.report.goals) > 0 {
			s.Width, s.ListAside = 560, 84
			s.Notes = []string{e.word("editor.arms.hint")}
		}
	case "catalog":
		if e.doc == nil {
			s.Notes = []string{e.word("editor.picker.hint")}
		}
		s.Buttons = []string{e.word("editor.button.open"), e.word("editor.button.file")}
	case "records":
		if e.filter == editorScriptsFilter {
			s.Notes = []string{e.word("editor.scripts.hint")}
			if len(e.rows()) == 0 && e.doc.ScriptNotice != "" {
				s.Notes = append(s.Notes, e.doc.ScriptNotice)
			}
		}
		if _, ok := e.listAdd(); ok {
			s.Buttons = []string{e.word("editor.button.add")}
		}
	}
	if m.Len() > 0 {
		top, n := m.Visible()
		s.Notes = append(s.Notes, fmt.Sprintf("%d-%d / %d", top+1, top+n, m.Len()))
	}
	return s, true
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
		if key := e.maps[maps[n]].Key; open && strings.HasPrefix(key, "new/") {
			e.askNewMode(key)
		} else if open {
			_ = e.Open(key)
		}
	case "report":
		e.selectFinding(n)
	case "brush":
		e.brushRowClick(n)
	case "records":
		if e.filter == editorScriptsFilter {
			row := 0
			for _, ids := range e.scriptSections() {
				row++
				for _, id := range ids {
					if row == n {
						e.Select(id, true)
						return
					}
					row++
				}
			}
			return
		}
		if e.filter == editorGroupsFilter {
			if gs := e.groups(); n >= 0 && n < len(gs) {
				e.chooseGroup(gs[n])
			}
			return
		}
		if rows := e.rows(); n >= 0 && n < len(rows) {
			e.Select(rows[n], true)
		}
	}
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

// objectSpec is the selected record's popup as data, and the reference each
// link row opens.
func (e *MapEditor) objectSpec() (popupSpec, []int, bool) {
	if e.messagePage != 0 && !e.catalog && !e.objectHidden && e.pick == pickNone && e.prompt.kind != promptAdd {
		return e.messageSpec()
	}
	if v, ok := e.scriptVariableRow(e.selected); ok && !e.catalog && !e.objectHidden && e.pick == pickNone && e.prompt.kind != promptAdd {
		s, links, ok := e.variableSpec(v)
		e.objectBackButton(&s)
		return s, links, ok
	}
	if s, links, ok := e.groupSpec(); ok {
		e.objectBackButton(&s)
		return s, links, true
	}
	// A cell pick moves the object popup aside: the pick is a click on the map.
	// The Add window stands in its place until it closes.
	if e.doc == nil || e.catalog || e.objectHidden || e.pick != pickNone || e.prompt.kind == promptAdd || e.selected < 0 || e.selected >= len(e.doc.Records) {
		return popupSpec{}, nil, false
	}
	if v, ok := e.scriptPopup(); ok {
		s, links := e.scriptSpec(v)
		e.obj.data = -1
		e.objectBackButton(&s)
		return s, links, true
	}
	if !e.objPlaced {
		e.objectAt, e.objPlaced = image.Pt(e.window.X-8-420, e.barHeight()+8), true
	}
	r := e.doc.Records[e.selected]
	e.loadObject()
	if e.obj.move != nil {
		r.Cell = *e.obj.move
	}
	s := popupSpec{Title: fmt.Sprintf("%s #%d", r.Kind, r.Index), At: e.objectAt, Width: 420, Bounds: e.popupBounds(), NoteInk: editorWarning}
	var links []int
	add := func(row popupRow, link int) {
		s.Rows, links = append(s.Rows, row), append(links, link)
	}
	if r.Spatial {
		add(popupRow{Kind: popupText, Label: e.word("editor.object.cell"), Value: fmt.Sprintf("%d,%d  %dx%d", r.Cell.X, r.Cell.Y, r.Size.X, r.Size.Y)}, 0)
	}
	e.objectFieldRows(&s, &links)
	if e.hasUsedBy() {
		e.scriptLinkSection(&s, &links, e.word("editor.usedby.title"), e.usedByRows())
	}
	e.obj.data = len(s.Rows)
	add(popupRow{Kind: popupHeading, Label: e.word("editor.object.data")}, 0)
	width := 420 - 2*popupPad - 4
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
		for _, part := range editorWrap(e.bodyFont(), l.Text, width) {
			add(popupRow{Kind: kind, Label: part, Ink: inspectionRoleColor(l.Role)}, l.Reference)
		}
	}
	e.objectBackButton(&s)
	if s.Card != nil {
		s.Width += compactPanelW + popupPad
	}
	s.MaxRows = e.objectRowsShown(len(s.Buttons) > 0 || e.objectHasFields())
	e.objectTop = max(0, min(e.objectTop, len(s.Rows)-s.MaxRows))
	s.Top = e.objectTop
	return s, links, true
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
func (e *MapEditor) promptSpec() (popupSpec, bool) {
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
	}
	w := min(560, e.window.X-16)
	s := popupSpec{Width: w, Bounds: e.popupBounds(), NoteInk: editorWarning, LabelWidth: 1}
	text := w - 2*popupPad - 60
	if e.prompt.open() {
		s.Title = e.promptTitle()
		var paths []string
		for _, line := range strings.Split(strings.ReplaceAll(e.prompt.path, "\r\n", "\n"), "\n") {
			paths = append(paths, editorWrap(e.bodyFont(), e.editorHostText(line), text)...)
		}
		if len(paths) > 6 {
			paths = paths[len(paths)-6:]
			paths[0] = textfit.Mark + paths[0]
			if f := e.bodyFont(); f != nil {
				paths[0] = textfit.Spec{Measure: f.Advance, Unit: textfit.Byte, Side: textfit.Start}.Continued(paths[0][len(textfit.Mark):], text)
			}
		}
		for i, p := range paths {
			row := popupRow{Kind: popupEdit, Value: p}
			if i == len(paths)-1 {
				row.Focus = true
				if f := e.bodyFont(); f != nil {
					row.Caret = f.Advance(p)
				}
			}
			s.Rows = append(s.Rows, row)
		}
		s.Buttons = []string{e.word("editor.button.ok"), e.word("editor.button.cancel")}
		if fields := e.promptFields(); e.prompt.fields() && e.prompt.field < len(fields) && fields[e.prompt.field].Meaning != "" {
			s.Notes = append(s.Notes, editorWrap(e.bodyFont(), e.editorHostText(fields[e.prompt.field].Meaning), text)...)
		}
	} else {
		s.Title = e.word("editor.message.title")
		s.Buttons = []string{e.word("editor.button.ok")}
	}
	if e.message != "" {
		errors := editorWrap(e.bodyFont(), e.editorHostText(e.message), text)
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
	s.At = image.Pt((e.window.X-l.Rect.Dx())/2, max(e.barHeight()+8, (e.window.Y-l.Rect.Dy())/3))
	return s, true
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
	}
	return e.word("editor.prompt.open")
}

var editorPopupInk = popupInk{Ground: editorPanel, Edge: editorGold, TitleGround: editorInk, Title: editorGold, Label: editorPaper, Value: editorLight}

// ChromeImage composes the window's chrome over a transparent window-size
// picture: the bar, the status strip, the list and object popups, the prompt
// and, last, an open drop-down.
func (e *MapEditor) ChromeImage() *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, e.window.X, e.window.Y))
	bar := image.Rect(0, 0, e.window.X, e.barHeight())
	draw.Draw(pic, bar, image.NewUniform(editorPanel), image.Point{}, draw.Src)
	draw.Draw(pic, image.Rect(0, bar.Max.Y-1, e.window.X, bar.Max.Y), image.NewUniform(editorGold), image.Point{}, draw.Src)
	for _, b := range e.barEntries() {
		if b.menu {
			drawDropdown(pic, e.font, nil, e.menuDropdown(b, false), nil, e.pointer, e.pointerOK)
			continue
		}
		drawPushButton(pic, e.font, pushButton{Rect: b.rect, Label: b.label, Literal: true, LabelRaise: labelRaise(e.font), Disabled: b.disabled, Hover: e.pointerOK && e.pointer.In(b.rect)})
	}
	status := e.statusRect()
	draw.Draw(pic, status, image.NewUniform(editorPanel), image.Point{}, draw.Src)
	draw.Draw(pic, image.Rect(0, status.Min.Y, e.window.X, status.Min.Y+1), image.NewUniform(editorGold), image.Point{}, draw.Src)
	if f := e.bodyFont(); f != nil {
		f.Draw(pic.SubImage(status).(*image.RGBA), editorClip(f, e.statusText(), status.Dx()-16), 8, status.Min.Y+5, editorPaper)
	}
	e.drawAreaLabels(pic)
	if s, ok := e.listSpec(); ok {
		drawPopup(pic, e.font, e.bodyFont(), nil, s, layoutPopup(s, e.font, e.bodyFont()), editorPopupInk, e.pointer, e.pointerOK)
		e.drawGoalLinks(pic, s)
	}
	if s, _, ok := e.objectSpec(); ok {
		drawPopup(pic, e.font, e.bodyFont(), nil, s, layoutPopup(s, e.font, e.bodyFont()), editorPopupInk, e.pointer, e.pointerOK)
		e.drawObjectPreview(pic, s)
	}
	if s, ok := e.promptSpec(); ok {
		drawPopup(pic, e.font, e.bodyFont(), nil, s, layoutPopup(s, e.font, e.bodyFont()), editorPopupInk, e.pointer, e.pointerOK)
		e.drawAddPreview(pic, s)
	}
	e.drawOpenDropdown(pic)
	e.drawAreaDropdown(pic)
	if b, ok := e.openMenuEntry(); ok {
		d := e.menuDropdown(b, true)
		drawDropdown(pic, e.font, nil, d, func(row, width int) string { return editorClip(e.font, d.List.Rows()[row].Text, width) }, e.pointer, e.pointerOK)
	}
	return pic
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
	var parts []string
	if e.hoverOK {
		parts = append(parts, fmt.Sprintf("%d,%d", e.hoverCell.X, e.hoverCell.Y))
		if h := e.hoverRecord(); h != "" {
			parts = append(parts, h)
		}
	}
	if e.doc != nil {
		mark := ""
		if e.Dirty() {
			mark = "* "
		}
		parts = append(parts, mark+e.editorHostText(e.doc.Source))
	}
	if e.pick == pickScriptCell {
		parts = append(parts, e.word("editor.trigger.pickcell"))
	}
	if e.pick == pickArea {
		parts = append(parts, e.word("editor.area.draw"))
	}
	parts = append(parts, strings.Join(e.helpLines(), "  /  "))
	return strings.Join(parts, "   |   ")
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
	e.chromeImage.WritePixels(blit.Pix)
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
	calls = text.Record(func() { pic = e.ChromeImage() })
	blit = image.NewRGBA(pic.Bounds())
	copy(blit.Pix, pic.Pix)
	kept = textsmooth.Settle(blit, calls)
	return pic, blit, calls, text.MarkErased(kept)
}

// chromeHit is the foremost chrome part under p.
func (e *MapEditor) chromeHit(p image.Point) editorPopupID {
	if b, ok := e.openMenuEntry(); ok && p.In(e.menuDropdown(b, true).Bounds(e.font)) {
		return popMenu
	}
	if s, ok := e.promptSpec(); ok && (p.In(layoutPopup(s, e.font, e.bodyFont()).Rect) || e.areaDropdownHit(p)) {
		return popPrompt
	}
	if s, _, ok := e.objectSpec(); ok && (p.In(layoutPopup(s, e.font, e.bodyFont()).Rect) || e.openDropdownHit(p)) {
		return popObject
	}
	if s, ok := e.listSpec(); ok && p.In(layoutPopup(s, e.font, e.bodyFont()).Rect) {
		return popList
	}
	if _, ok := e.areaLabelAt(p); ok && e.pick == pickNone && !e.blocksCanvas() {
		return popArea
	}
	if p.Y < e.barHeight() {
		return popBar
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
	pressed := in.PrimaryDown && !e.down
	if e.scrollPointer(in, p, pressed) {
		return true
	}
	if e.drag != popNone {
		if in.PrimaryDown {
			d := p.Sub(e.dragFrom)
			e.dragFrom = p
			if e.drag == popList {
				e.listAt = e.listAt.Add(d)
			} else {
				e.objectAt = e.objectAt.Add(d)
			}
		} else {
			e.drag = popNone
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
	if target == popNone && !modal {
		return false
	}
	if in.WheelY != 0 {
		e.chromeWheel(target, in.WheelY)
	}
	if pressed {
		e.down, e.uiDown = true, true
		double := e.clicks.press(p, now)
		if e.menu != "" && target != popMenu {
			e.menu = ""
			if target != popBar {
				return true
			}
		}
		if modal && target != popPrompt && target != popMenu {
			return true
		}
		e.chromePress(target, p, double)
	}
	return true
}

// chromeWheel scrolls the popup under the pointer.
func (e *MapEditor) chromeWheel(target editorPopupID, wheel float64) {
	switch target {
	case popList:
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
	case popPrompt:
		if !e.areaWheel(wheel) {
			e.addWheel(wheel)
		}
	}
}

// chromePress is a press on a chrome part.
func (e *MapEditor) chromePress(target editorPopupID, p image.Point, double bool) {
	switch target {
	case popMenu:
		b, _ := e.openMenuEntry()
		d := e.menuDropdown(b, true)
		switch part, row := d.hit(e.font, p); part {
		case dropdownToggle:
			e.menu = ""
		case dropdownRow:
			items := e.menuItems(e.menu)
			if row < len(items) {
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
		case popupClose:
			e.cancelPopup(popList)
		case popupTitle:
			e.drag, e.dragFrom = popList, p
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
			} else if row, ok := e.listAdd(); ok {
				e.openAdd(row.Kind)
			}
		}
	}
}

// objectPress is a press on the object popup.
func (e *MapEditor) objectPress(p image.Point) {
	if e.openDropdownPress(p) {
		return
	}
	s, links, _ := e.objectSpec()
	switch part, n := layoutPopup(s, e.font, e.bodyFont()).hit(p); part {
	case popupClose:
		e.cancelPopup()
	case popupTitle:
		e.drag, e.dragFrom = popObject, p
	case popupRowButton:
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
			if e.hasUsedBy() && scriptRecord(e.doc.Records[-links[n]-1].Kind) {
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
	case popupButton:
		if len(e.objectBack) > 0 && n == len(s.Buttons)-1 {
			e.backObject()
			return
		}
		if _, ok := e.scriptVariableRow(e.selected); ok {
			e.cancelPopup()
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

// gridLines are the View menu's grid over the map as editor lines.
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
