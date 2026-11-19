package ui

import (
	"fmt"
	"image"
	"slices"
	"strconv"
	"strings"
)

// Every record is placed from its palette; the editor has no Add window.
// The Units, Structures, Sacks and Traps and spells palettes show a picture
// grid of the host's choices of one step of their kind (unit classes,
// structure kinds, items, spells), each with the host's own picture.
// Choosing one arms the cell pick; each click on the map then places one
// record, and a press held over the map places one on every cell it
// crosses. Each record is one history entry, assembled from the kind's
// steps (addKinds) at the palette's values: the player row's owner, the
// ticked items, the spell settings; every other step at the host's default.

// The object brush's one-kind mode shows the same grid over the structure
// kinds of the Objects palette, and a press on its cell places copies the
// same way.

// placeGrid is the tabs' grid state: its first shown row, and the object
// brush's, the block its switch shows, the pictures the host gave, by kind
// and value, and the choices of the document they were read for with the
// palette of each.
type placeGrid struct {
	top, brushTop int
	block         int
	ruined        bool
	icons         map[string]*image.RGBA
	doc           *InspectionDocument
	kind          RecordKind
	choices       []InspectionChoice
	groups        []InspectionChoiceGroup
	palettes      map[int64]Palette
	// owners is the player the Units tab places for, by map source.
	owners map[string]placeOwnerPick
	// ownerList is the player row's open list of players, nil while closed.
	ownerList *Picker
	// pictures is the host's picture offer of the grid's step, nil when it
	// offers none and NewRecordPreview draws the cells.
	pictures func([]InspectionChoice) map[int64]*image.RGBA
	// sack is the items the Sacks palette ticked: what the next sack holds.
	sack []int64
	// cast is the Traps and spells palette's settings.
	cast castPalette
}

// placeOwnerPick is a player the Units tab was set to: its player record's
// session key, which a rename keeps and a delete ends, and its owner value
// for a host that keys no record.
type placeOwnerPick struct {
	key   uint64
	value int64
}

// InspectionChoiceGroup places choice values under a block and row heading.
// Headings are in the font's bytes; values keep their field meanings.
type InspectionChoiceGroup struct {
	Block, Head string
	Values      []int64
}

var placeGridSides = map[RecordKind]int{KindStructure: 2*(gridCellSide-8) + 8}

// placeKey is the step each kind's palette grid chooses.
var placeKey = map[RecordKind]string{KindUnit: "identity", KindStructure: "kind", KindSack: "item", KindEnchantment: "spell"}

// placeGridShell is kind's grid with no cells: it shows every row the room
// holds.
func placeGridShell(kind RecordKind, top int) *popupGrid {
	return &popupGrid{Top: top, Side: placeGridSides[kind]}
}

// placeGridKind is the record kind and the Add step the shown tab's grid
// chooses: unit classes on the Units tab, structure kinds on the Structures
// tab.
func (e *MapEditor) placeGridKind() (kind RecordKind, key string, ok bool) {
	if _, host := e.paletteHost(); !host || e.listKind() != "records" {
		return "", "", false
	}
	for _, kind := range []RecordKind{KindUnit, KindStructure, KindSack, KindEnchantment} {
		if kind == KindEnchantment && !e.editable() {
			continue
		}
		if recordKinds[kind].filter == e.filter {
			return kind, placeKey[kind], true
		}
	}
	return "", "", false
}

// placeGridOf is the record kind, the Add step and the palette of the grid
// the list popup shows: on the Units tab every unit class, on the
// Structures tab the structure kinds of the Structures palette, and in the
// object brush's one-kind mode the structure kinds of the Objects palette.
func (e *MapEditor) placeGridOf() (kind RecordKind, key string, palette Palette, ok bool) {
	if kind, key, ok := e.placeGridKind(); ok {
		if kind == KindStructure {
			return kind, key, PaletteStructures, true
		}
		return kind, key, PaletteOwn, true
	}
	if _, host := e.addHost(); host && e.listKind() == "brush" && e.brush.mode == brushObjects && !e.brush.forest.on && !e.brush.forest.trees {
		return KindStructure, "kind", PaletteObjects, true
	}
	return "", "", PaletteOwn, false
}

// placeChoices are the choices of the grid's step, as the host offers them
// for the open document.
func (e *MapEditor) placeChoices(kind RecordKind, key string) []InspectionChoice {
	host, ok := e.paletteHost()
	if !ok {
		return nil
	}
	if e.place.doc != e.doc || e.place.kind != kind {
		step := host.NewRecordStep(InspectionNewRecord{Kind: kind}, key)
		e.place.doc, e.place.kind = e.doc, kind
		e.place.choices, e.place.palettes = nil, step.Palettes
		e.place.groups, e.place.pictures = step.Groups, step.Pictures
		for _, c := range step.Choices {
			if !slices.Contains(step.Shut, c.Value) {
				e.place.choices = append(e.place.choices, c)
			}
		}
	}
	return e.place.choices
}

// placeShown are the choices the shown grid holds: those of its step that
// list in its palette, every one for a palette of PaletteOwn.
func (e *MapEditor) placeShown() []InspectionChoice {
	kind, key, palette, ok := e.placeGridOf()
	if !ok {
		return nil
	}
	return e.placeIn(kind, key, palette)
}

// placeIn are the choices of kind's step key that list in palette.
func (e *MapEditor) placeIn(kind RecordKind, key string, palette Palette) []InspectionChoice {
	all := e.placeChoices(kind, key)
	if palette == PaletteOwn {
		return all
	}
	var out []InspectionChoice
	for _, c := range all {
		if e.place.palettes[c.Value].in(PaletteStructures) == palette {
			out = append(out, c)
		}
	}
	return out
}

// placeTop is the first shown row of the shown grid.
func (e *MapEditor) placeTop() *int {
	if e.listKind() == "brush" {
		return &e.place.brushTop
	}
	return &e.place.top
}

// brushPlaceGridHeight is the room the brush window's list leaves the
// Objects palette's grid of structure kinds: its least rows and the hint.
func (e *MapEditor) brushPlaceGridHeight() int {
	if _, _, palette, ok := e.placeGridOf(); !ok || palette != PaletteObjects || len(e.placeShown()) == 0 {
		return 0
	}
	return (&popupGrid{Rows: gridLeastRows, Side: placeGridSides[KindStructure]}).height() + fontHeight(e.bodyFont(), 15) + popupRowExtra
}

// placeIcon is the host's picture of choice value of kind's step key, read
// once.
func (e *MapEditor) placeIcon(kind RecordKind, key string, value int64) *image.RGBA {
	host, ok := e.paletteHost()
	if !ok {
		return nil
	}
	k := fmt.Sprintf("%s/%s/%d", kind, key, value)
	if pic, ok := e.place.icons[k]; ok {
		return pic
	}
	if e.place.icons == nil {
		e.place.icons = map[string]*image.RGBA{}
	}
	if e.place.pictures != nil && e.place.kind == kind {
		pic := e.place.pictures([]InspectionChoice{{Value: value}})[value]
		e.place.icons[k] = pic
		return pic
	}
	previewKey := key
	if kind == KindStructure && key == "kind" {
		previewKey = "kind-grid"
	}
	if kind == KindUnit && key == "identity" {
		previewKey = "identity-grid"
	}
	pic := host.NewRecordPreview(kind, previewKey, value)
	e.place.icons[k] = pic
	return pic
}

// placeTipPicture is the picture the hover tip of a cell of the grid of kind
// and step key shows for choice value: the host's preview of the choice, the
// one picture the brush windows show beside their object list.
func (e *MapEditor) placeTipPicture(kind RecordKind, key string, value int64) *image.RGBA {
	host, ok := e.paletteHost()
	if !ok {
		return nil
	}
	k := fmt.Sprintf("tip/%s/%s/%d", kind, key, value)
	if pic, ok := e.place.icons[k]; ok {
		return pic
	}
	if e.place.icons == nil {
		e.place.icons = map[string]*image.RGBA{}
	}
	pic := host.NewRecordPreview(kind, key, value)
	e.place.icons[k] = pic
	return pic
}

// placing is the armed placing run and the value of its step key: the grid
// cell shown pressed.
func (e *MapEditor) placing(key string) (*addRun, int64, bool) {
	r, ok := e.addRunning()
	if !ok {
		return nil, 0, false
	}
	for i, st := range r.steps {
		if st.Key == key {
			v, err := strconv.ParseInt(strings.TrimSpace(r.texts[i]), 10, 64)
			return r, v, err == nil
		}
	}
	return r, 0, false
}

// placeGridSpec is the shown tab's grid as data, and the note naming the
// chosen cell, empty while none is.
func (e *MapEditor) placeGridSpec() (*popupGrid, string, bool) {
	kind, key, _, ok := e.placeGridOf()
	if !ok {
		return nil, "", false
	}
	choices := e.placeShown()
	if len(choices) == 0 {
		return nil, "", false
	}
	g := placeGridShell(kind, *e.placeTop())
	r, chosen, placing := e.placing(key)
	placing = placing && r.kind == kind
	// The note is a state: what is being placed. How placing works is the
	// title's (?), editor.place.help.
	note := ""
	for _, c := range choices {
		pressed := placing && c.Value == chosen
		if pressed {
			note = fmt.Sprintf(e.word("editor.place.placing"), c.Label)
			if r.note != "" {
				note = r.note
			}
		}
		cell := popupGridCell{Icon: e.placeIcon(kind, key, c.Value), Pressed: pressed, Tip: c.Label}
		switch {
		case kind == KindUnit && key == "identity":
			cell.TipPicture = e.placeTipPicture(kind, key, c.Value)
		case kind == KindSack:
			cell.Pressed, cell.Ticked = false, slices.Contains(e.place.sack, c.Value)
		case kind == KindEnchantment && cell.Icon == nil:
			// A spell the book has no icon for shows its number.
			cell.Label = strconv.FormatInt(c.Value, 10)
		}
		g.Cells = append(g.Cells, cell)
	}
	groups, blocks := e.placeGroups(choices)
	if kind == KindSack {
		if r != nil && r.kind == KindSack {
			note = r.note
		}
		// One long grid: a head row per folder, its items under it.
		for i := range groups {
			groups[i].Head, groups[i].Block = groups[i].Block, ""
		}
		g.Groups = groups
		return g, note, true
	}
	if len(blocks) < 2 {
		g.Groups = groups
		return g, note, true
	}
	shown := blocks[e.placeBlock(blocks)]
	for _, row := range groups {
		if row.Block == shown {
			row.Block = ""
			g.Groups = append(g.Groups, row)
		}
	}
	return g, note, true
}

// placeNoteRows are the note's lines in a window width wide, each wrapped
// to the window's text width; none for an empty note.
func (e *MapEditor) placeNoteRows(note string, width int) []popupRow {
	if note == "" {
		return nil
	}
	var rows []popupRow
	for _, line := range editorTextLines(e.bodyFont(), note, width-2*popupPad-8) {
		rows = append(rows, popupRow{Kind: popupText, Label: line})
	}
	return rows
}

// placeGroups are the grid's groups over choices, each with a cell, and the
// blocks they fall in, in their order.
func (e *MapEditor) placeGroups(choices []InspectionChoice) ([]popupGridGroup, []string) {
	if len(e.place.groups) == 0 {
		return nil, nil
	}
	indices := make(map[int64]int, len(choices))
	for i, c := range choices {
		indices[c.Value] = i
	}
	var groups []popupGridGroup
	var blocks []string
	for _, group := range e.place.groups {
		row := popupGridGroup{Block: group.Block, Head: group.Head}
		for _, value := range group.Values {
			if n, ok := indices[value]; ok {
				row.Cells = append(row.Cells, n)
			}
		}
		if len(row.Cells) > 0 {
			groups = append(groups, row)
			if !slices.Contains(blocks, row.Block) {
				blocks = append(blocks, row.Block)
			}
		}
	}
	return groups, blocks
}

// placeBlock is the index of the block the grid's switch shows among
// blocks: the one last chosen while the editor runs, the first at the
// start.
func (e *MapEditor) placeBlock(blocks []string) int {
	return max(0, min(e.place.block, len(blocks)-1))
}

// placeSwitch is the shown grid's switch on popup s: a tab for each block
// of a grid whose groups fall in two or more, the shown block pressed. The
// grid then shows that block alone, without its block heading.
func (e *MapEditor) placeSwitch(s *popupSpec) {
	if kind, _, _, _ := e.placeGridOf(); kind != KindUnit {
		return
	}
	if _, blocks := e.placeGroups(e.placeShown()); len(blocks) > 1 {
		s.Tabs, s.ActiveTab, s.TabSwitch = blocks, e.placeBlock(blocks), true
		// A caption wider than its share takes a line of its own.
		s.Fit = true
	}
}

// placeSwitchTo is a press on the grid switch's tab n: the grid shows that
// block from its top.
func (e *MapEditor) placeSwitchTo(n int) {
	if kind, _, _, _ := e.placeGridOf(); kind != KindUnit {
		return
	}
	_, blocks := e.placeGroups(e.placeShown())
	if len(blocks) < 2 || n < 0 || n >= len(blocks) || n == e.placeBlock(blocks) {
		return
	}
	e.place.block = n
	*e.placeTop() = 0
}

// placeChoose is a press on grid cell n: it arms the placing of that
// choice, and each click on the map then places one record of it. A press
// on the chosen cell ends the placing. On the Sacks palette the press ticks
// or unticks the item instead, and the placing holds the ticked items. Its
// palette stays open.
func (e *MapEditor) placeChoose(n int) {
	kind, key, _, ok := e.placeGridOf()
	choices := e.placeShown()
	if !ok || n < 0 || n >= len(choices) || !e.editable() {
		return
	}
	v := choices[n].Value
	if kind == KindSack {
		e.sackTick(v)
		return
	}
	if r, at, ok := e.placing(key); ok && r.kind == kind && at == v {
		e.leave(leavePick | leavePrompt)
		return
	}
	if kind == KindEnchantment {
		e.place.cast.set(key, v)
	}
	e.placeStart(kind, map[string]int64{key: v})
}

// placeStart arms the placing of kind: no window, the cell pick armed, and
// each click on the map adds one record through the kind's steps
// (addKinds), each step at the value given or the palette's, else at the
// host's default (addAccepted). It replaces any placing armed before.
func (e *MapEditor) placeStart(kind RecordKind, given map[string]int64) {
	row, ok := addKindOf(kind)
	if _, host := e.addHost(); !ok || !host {
		return
	}
	fromBrush := e.listKind() == "brush"
	x := leaveDrag | leavePick | leavePrompt
	if !fromBrush {
		x |= leaveBrush
	}
	e.leave(x)
	r := &addRun{kind: kind, brush: fromBrush}
	r.setSteps(row.Steps)
	for i, st := range r.steps {
		if v, ok := given[st.Key]; ok {
			r.texts[i] = strconv.FormatInt(v, 10)
		}
	}
	e.openPrompt(editorPrompt{kind: promptPlace, add: r})
	e.armPick(pickPlace)
}

// placeOwners are the players a unit the Units tab places may belong to,
// as the host offers the owner step, and the value that step starts
// at. They are read from the open map each time, so a renamed or deleted
// player shows at once.
func (e *MapEditor) placeOwners() ([]InspectionChoice, int64) {
	host, ok := e.addHost()
	if !ok {
		return nil, 0
	}
	offer, _ := e.addOffer(host, &addRun{kind: KindUnit, steps: []addStep{addOwner}}, 0)
	return offer.Choices, offer.Value
}

// placeOwner is the player the Units tab places for: the one last chosen
// on this map while the map still has it, at the slot it holds now, else the
// owner step's start, else the first player.
func (e *MapEditor) placeOwner() (int64, bool) {
	choices, start := e.placeOwners()
	if len(choices) == 0 {
		return 0, false
	}
	listed := func(v int64) bool {
		return slices.ContainsFunc(choices, func(c InspectionChoice) bool { return c.Value == v })
	}
	if pick, ok := e.place.owners[e.doc.Source]; ok {
		v := pick.value
		if pick.key != 0 {
			v = 0
			for _, r := range e.doc.Records {
				if r.Kind == KindPlayer && r.Key == pick.key {
					v = int64(r.Index) + 1
				}
			}
		}
		if listed(v) {
			return v, true
		}
	}
	if listed(start) {
		return start, true
	}
	return choices[0].Value, true
}

// placeOwnerRow is the player row of the Units, Structures and Objects
// palettes: one choice showing the chosen player, whose list of players
// opens on its row. ok is false on any other palette.
func (e *MapEditor) placeOwnerRow() (popupRow, bool) {
	if kind, _, _, ok := e.placeGridOf(); !ok || kind != KindUnit && kind != KindStructure {
		return popupRow{}, false
	}
	choices, _ := e.placeOwners()
	owner, ok := e.placeOwner()
	if !ok {
		return popupRow{}, false
	}
	row := popupRow{Kind: popupChoice, Label: e.word("editor.place.player")}
	for _, c := range choices {
		if c.Value == owner {
			row.Value = c.Label
		}
	}
	return row, true
}

// placeOwnerAt is the index of the player row among s's rows, -1 for none.
func (e *MapEditor) placeOwnerAt(s popupSpec) int {
	return slices.IndexFunc(s.Rows, func(r popupRow) bool {
		return r.Kind == popupChoice && r.HeadOf == 0 && r.Label == e.word("editor.place.player")
	})
}

// placeOwnerOpen is a press on the player row's box: it opens the list of
// players on the chosen one, or closes the open list.
func (e *MapEditor) placeOwnerOpen() {
	if e.place.ownerList != nil {
		e.place.ownerList = nil
		return
	}
	choices, _ := e.placeOwners()
	owner, ok := e.placeOwner()
	if !ok {
		return
	}
	rows, sel := make([]PickerRow, len(choices)), 0
	for k, c := range choices {
		rows[k] = PickerRow{Text: c.Label, Choosable: true}
		if c.Value == owner {
			sel = k
		}
	}
	list := NewPicker(rows)
	list.SetWindow(min(10, len(rows)))
	list.Select(sel)
	e.place.ownerList = list
}

// placeOwnerDropdown is the player row's drop-down while its list is open:
// the editor's one row drop-down, on the row's box.
func (e *MapEditor) placeOwnerDropdown() (dropdown, bool) {
	list := e.place.ownerList
	if list == nil {
		return dropdown{}, false
	}
	s, ok := e.listSpec()
	if !ok {
		return dropdown{}, false
	}
	at := e.placeOwnerAt(s)
	for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if rr.Index == at && at >= 0 {
			return e.rowDropdown(rr.Control, s.Rows[at].Value, list, min(10, list.Len()), true, false), true
		}
	}
	return dropdown{}, false
}

// placeOwnerDropdownHit reports that p lies on the player row's open
// drop-down: its box or its list.
func (e *MapEditor) placeOwnerDropdownHit(p image.Point) bool {
	d, ok := e.placeOwnerDropdown()
	return ok && d.Covers(e.bodyFont(), p)
}

// placeOwnerPress is a press while the player list is open: a row chooses
// that player and closes the list, the box closes it, the bar scrolls it,
// and a press elsewhere closes it and goes on to what lies under it.
func (e *MapEditor) placeOwnerPress(p image.Point) bool {
	d, ok := e.placeOwnerDropdown()
	if !ok {
		return false
	}
	switch part, row := d.hit(e.bodyFont(), p); part {
	case dropdownRow:
		e.placeOwnerTo(row)
		e.place.ownerList = nil
		return true
	case dropdownToggle:
		e.place.ownerList = nil
		return true
	case dropdownBar:
		return true
	}
	e.place.ownerList = nil
	return false
}

// placeOwnerPressOff closes the player list on a press that reaches
// neither the list nor its window.
func (e *MapEditor) placeOwnerPressOff(p image.Point) {
	if e.place.ownerList != nil && e.chromeHit(p) != popList {
		e.place.ownerList = nil
	}
}

// placeOwnerTo chooses player k of the player list: the Units tab places
// for that player from then on.
func (e *MapEditor) placeOwnerTo(k int) {
	choices, _ := e.placeOwners()
	if k < 0 || k >= len(choices) || e.doc == nil {
		return
	}
	if e.place.owners == nil {
		e.place.owners = map[string]placeOwnerPick{}
	}
	pick := placeOwnerPick{value: choices[k].Value}
	for _, r := range e.doc.Records {
		if r.Kind == KindPlayer && int64(r.Index)+1 == pick.value {
			pick.key = r.Key
		}
	}
	e.place.owners[e.doc.Source] = pick
}

// controlPlaceGridCellRect is the window rectangle of cell n of the list popup's
// grid, the tab's or the forest's, when it is shown: a headless driver
// presses it.
func (e *MapEditor) controlPlaceGridCellRect(n int) (image.Rectangle, bool) {
	s, ok := e.listSpec()
	if !ok || s.Grid == nil {
		return image.Rectangle{}, false
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	for i, rect := range l.GridCells {
		if l.gridIndex(i) == n {
			return rect, true
		}
	}
	return image.Rectangle{}, false
}

// controlPlaceGridSwitchRect is the window rectangle of tab n of the list popup
// grid's switch, when the grid has one: a headless driver presses it.
func (e *MapEditor) controlPlaceGridSwitchRect(n int) (image.Rectangle, bool) {
	s, ok := e.listSpec()
	if !ok || !s.TabSwitch || n < 0 || n >= len(s.Tabs) {
		return image.Rectangle{}, false
	}
	return layoutPopup(s, e.font, e.bodyFont()).Tabs[n], true
}

// PlaceGridLen is how many cells the shown tab's grid has.
func (e *MapEditor) PlaceGridLen() int {
	if g, _, ok := e.placeGridSpec(); ok {
		return len(g.Cells)
	}
	return 0
}

// placeStroke is a press held over the map while copies are placed: the
// cells it has placed on.
type placeStroke struct{ cells map[image.Point]bool }

// placePointer drives the map while copies are placed: a press places one
// copy on the cell under the pointer, and while the button is held each
// further cell the pointer enters gets one. It reports whether it took the
// input, so the map does not pan under it.
func (e *MapEditor) placePointer(in Input, p image.Point, rail int) bool {
	r, ok := e.addRunning()
	if !ok {
		return false
	}
	if r.stroke == nil {
		if e.pick != pickPlace || !in.PrimaryDown || e.down || e.recordDrag != nil || e.cancelledDrag {
			return false
		}
		if !p.In(image.Rect(0, 0, rail, e.doc.Viewer.cam.ViewH)) {
			return false
		}
		r.stroke = &placeStroke{cells: map[image.Point]bool{}}
		e.doc.Viewer.dragging = false
	}
	e.down = in.PrimaryDown
	if x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y)); ok && e.pick == pickPlace && p.In(image.Rect(0, 0, rail, e.doc.Viewer.cam.ViewH)) {
		if cell := image.Pt(x, y); cell.In(image.Rect(0, 0, e.doc.Width, e.doc.Height)) && !r.stroke.cells[cell] {
			r.stroke.cells[cell] = true
			e.pickAt(p)
		}
	}
	if !in.PrimaryDown {
		r.stroke = nil
	}
	return true
}

// placedNote is the palette's note after a record is placed.
func (e *MapEditor) placedNote(r *addRun) {
	r.placed++
	r.note = fmt.Sprintf(e.word("editor.place.placed"), r.placed)
}

// placeValues sets the steps of run, a click's record, to the palette's
// values: the player row's owner, the ticked items of the next sack and the
// Traps and spells palette's settings.
func (e *MapEditor) placeValues(run *addRun) {
	owner, hasOwner := e.placeOwner()
	for i, st := range run.steps {
		switch {
		case st.Key == addOwner.Key && hasOwner:
			run.texts[i] = strconv.FormatInt(owner, 10)
		case run.kind == KindSack && st.Control == addMany:
			run.many[i] = slices.Clone(e.place.sack)
		case run.kind == KindEnchantment && st.Key == "template":
			run.texts[i] = strconv.FormatInt(e.castTemplate(), 10)
		case run.kind == KindEnchantment:
			if v, ok := e.place.cast.values[st.Key]; ok {
				run.texts[i] = strconv.FormatInt(v, 10)
			}
		}
	}
}

// sackTick ticks item v for the next sack, or unticks a ticked one. The
// placing of sacks is armed while an item is ticked.
func (e *MapEditor) sackTick(v int64) {
	if i := slices.Index(e.place.sack, v); i >= 0 {
		e.place.sack = slices.Delete(e.place.sack, i, i+1)
	} else {
		e.place.sack = append(e.place.sack, v)
	}
	r, placing := e.addRunning()
	switch {
	case len(e.place.sack) == 0 && placing:
		e.leave(leavePick | leavePrompt)
	case len(e.place.sack) > 0 && (!placing || r.kind != KindSack):
		e.placeStart(KindSack, nil)
	}
}

// sackRows are the Sacks palette's rows above its grid: the Next sack head
// and one row per ticked item, its picture, number and name, with a Remove
// button.
func (e *MapEditor) sackRows() []popupRow {
	kind, key, _, ok := e.placeGridOf()
	if !ok || kind != KindSack {
		return nil
	}
	rows := []popupRow{{Kind: popupText, Label: e.word("editor.sacks.next")}}
	for _, v := range e.place.sack {
		label := strconv.FormatInt(v, 10)
		for _, c := range e.placeChoices(kind, key) {
			if c.Value == v {
				label = c.Label
			}
		}
		row := popupRow{Kind: popupText, Label: label, Buttons: []string{e.word("editor.button.remove")}}
		if pic := e.placeIcon(kind, key, v); pic != nil {
			row.Icons = []*image.RGBA{pic}
		}
		rows = append(rows, row)
	}
	return rows
}

// sackRowPress is a press on row n of s, the Sacks palette: a ticked item's
// Remove unticks it. It reports whether the row is one of the item rows.
func (e *MapEditor) sackRowPress(s popupSpec, n int) bool {
	head := slices.IndexFunc(s.Rows, func(r popupRow) bool {
		return r.Kind == popupText && r.Label == e.word("editor.sacks.next")
	})
	if head < 0 || n <= head || n-head-1 >= len(e.place.sack) {
		return false
	}
	e.sackTick(e.place.sack[n-head-1])
	return true
}

// castPalette is the Traps and spells palette's settings, by step key: the
// template its Where row or a record chosen in its list gives, and the
// spell, power, trap kind and scan range each placed record takes.
type castPalette struct {
	values      map[string]int64
	source      uint64
	constructor int64
	doc         *InspectionDocument
	offer       InspectionNewStep
}

// set gives step key value v.
func (c *castPalette) set(key string, v int64) {
	if c.values == nil {
		c.values = map[string]int64{}
	}
	c.values[key] = v
}
