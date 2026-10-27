package ui

import (
	"fmt"
	"hash/fnv"
	"image"
	"os"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// The map editor window's behaviour as an author sees it: for each entry
// path into a mode, each key and pointer gesture, and what follows (the next
// keys, a canvas click, a typed number), the help lines, the message, the
// modal rows, the report, the list and the panel pixels. Nothing here reads
// the window's mode fields, so the same table runs against any layout of them.

type modesCalls struct{ lines []string }

func (c *modesCalls) add(format string, a ...any) {
	c.lines = append(c.lines, fmt.Sprintf(format, a...))
}

// modesBase is a host that edits: every edit returns a fresh document and is
// logged.
type modesBase struct {
	calls   *modesCalls
	dirty   bool
	doc     func(InspectionEdits) *InspectionDocument
	self    InspectionEdits
	players []string
}

func (p *modesBase) next(format string, a ...any) (*InspectionDocument, error) {
	p.calls.add(format, a...)
	p.dirty = true
	return p.doc(p.self), nil
}
func (p *modesBase) Dirty() bool   { return p.dirty }
func (p *modesBase) CanUndo() bool { return p.dirty }
func (p *modesBase) CanRedo() bool { return !p.dirty }
func (p *modesBase) Move(k string, i int, c image.Point) (*InspectionDocument, error) {
	return p.next("Move %s %d %v", k, i, c)
}
func (p *modesBase) Clone(k string, i int, c image.Point) (*InspectionDocument, error) {
	return p.next("Clone %s %d %v", k, i, c)
}
func (p *modesBase) Delete(k string, i int) (*InspectionDocument, error) {
	if k == "Player" {
		if i == 0 {
			return nil, fmt.Errorf("The first player cannot be deleted.")
		}
		p.players = append(p.players[:i], p.players[i+1:]...)
	}
	return p.next("Delete %s %d", k, i)
}
func (p *modesBase) Fields(k string, i int) []InspectionField {
	if k == "Unit" || k == "Sack" || k == "Structure" {
		return []InspectionField{{Key: "a", Label: "First", Value: 11}, {Key: "b", Label: "Second", Value: 22}}
	}
	if k == "Player" && i >= 0 && i < len(p.players) {
		return []InspectionField{
			{Key: "name", Label: "Name", Text: true, TextValue: p.players[i]},
			{Key: "colour", Label: "Colour (0..15)", Value: int64(i)},
			{Key: "control", Label: "Control (0 human, 1 computer)", Value: 1},
			{Key: "relation:0", Label: "Relation to Self", Value: 1},
		}
	}
	return nil
}
func (p *modesBase) SetField(k string, i int, key string, v int64) (*InspectionDocument, error) {
	return p.next("SetField %s %d %s %d", k, i, key, v)
}
func (p *modesBase) Undo() (*InspectionDocument, error) {
	p.calls.add("Undo")
	p.dirty = false
	return p.doc(p.self), nil
}
func (p *modesBase) Redo() (*InspectionDocument, error) {
	p.calls.add("Redo")
	p.dirty = true
	return p.doc(p.self), nil
}
func (p *modesBase) SaveAs(path string) (string, error) {
	p.calls.add("SaveAs %s", path)
	if path == "bad" {
		return "", &ReportRefusal{Findings: modesFindings}
	}
	if path == "plain" {
		return "", fmt.Errorf("the writer refused %s", path)
	}
	p.dirty = false
	return "saved/" + path, nil
}

var modesFindings = []InspectionFinding{
	{Error: true, Rule: "owner-slot", Kind: "Unit", Index: 0, Cell: image.Pt(10, 10), HasCell: true, Text: "owner slot 9"},
	{Rule: "drop-cell", Kind: "Action", Index: 0, Text: "the drop cell is blocked"},
	{Rule: "map-note", Text: "a note on the whole map"},
}

// modesFull adds the optional seams: script, brush, units and the report.
type modesFull struct{ modesBase }

func (p *modesFull) AddStock(unit int) (*InspectionDocument, error) {
	return p.next("AddStock %d", unit)
}
func (p *modesFull) AddItemRecipe(kind string, index int, item int64) (*InspectionDocument, error) {
	return p.next("AddItemRecipe %s %d %d", kind, index, item)
}

func (p *modesFull) Goals() []InspectionGoal {
	return []InspectionGoal{
		{Arm: "Win", Text: "kill unit", Pick: "Unit", Kind: "Action", Index: 0, Cell: image.Pt(10, 10), HasCell: true},
		{Arm: "Lose", Text: "lose structure", Pick: "Structure", Kind: "Action", Index: 1},
	}
}
func (p *modesFull) SetGoalTarget(g int, k string, i int) (*InspectionDocument, error) {
	return p.next("SetGoalTarget %d %s %d", g, k, i)
}
func (p *modesFull) SetDropCell(c image.Point) (*InspectionDocument, error) {
	return p.next("SetDropCell %v", c)
}
func (p *modesFull) BrushTerrains() []BrushTerrain {
	return []BrushTerrain{{Class: 2, Name: "Grass"}, {Class: 8, Name: "Mountain"}, {Class: 9, Name: "Water"}}
}
func (p *modesFull) BrushObjects() []BrushObject {
	return []BrushObject{{Code: 0, Name: "No object"}, {Code: 3, Name: "Oak"}}
}
func (p *modesFull) BrushMaxRadius() int                    { return 16 }
func (p *modesFull) BrushCells(s BrushStroke) []image.Point { return probeBrushCells(s, 64, 64) }
func (p *modesFull) Paint(s BrushStroke) (*InspectionDocument, error) {
	return p.next("Paint %v %d %d %d", s.Plane == BrushPlaneHeight, s.Value, s.Radius, len(s.Cells))
}

// NewRecordStep offers two choices for a choice step, "New" for a name and
// 1 for a number; the tier gets none, so its step is left out.
func (p *modesFull) NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep {
	switch key {
	case "tier":
		return InspectionNewStep{}
	case "name":
		return InspectionNewStep{Text: "New"}
	case "colour", "group":
		return InspectionNewStep{Value: 1}
	}
	return InspectionNewStep{Choices: []InspectionChoice{{Value: 1, Label: "1: One"}, {Value: 2, Label: "2: Two"}}, Value: 1}
}
func (p *modesFull) CheckNewRecord(InspectionNewRecord, string) error { return nil }
func (p *modesFull) NewRecordPreview(string, string, int64) *image.RGBA {
	return nil
}
func (p *modesFull) AddRecord(r InspectionNewRecord) (*InspectionDocument, error) {
	if r.Kind == "Player" {
		p.players = append(p.players, fmt.Sprintf("Player %d", len(p.players)+1))
	}
	return p.next("AddRecord %s %v %v %v", r.Kind, r.Cell, r.HasCell, r.Values)
}
func (p *modesFull) SetTextField(k string, i int, key, value string) (*InspectionDocument, error) {
	if k == "Player" && key == "name" {
		p.players[i] = value
	}
	return p.next("SetTextField %s %d %s %q", k, i, key, value)
}
func (p *modesFull) Report() []InspectionFinding { return modesFindings }

// modesClip is the full host's copied group.
type modesClip []RecordRef

func (c modesClip) Len() int { return len(c) }

func (p *modesFull) group(format string, a ...any) (SelectionEdit, error) {
	d, err := p.next(format, a...)
	return SelectionEdit{Doc: d, Note: "group edit"}, err
}
func (p *modesFull) MoveRecords(refs []RecordRef, offset image.Point) (SelectionEdit, error) {
	return p.group("MoveRecords %v %v", refs, offset)
}
func (p *modesFull) DeleteRecords(refs []RecordRef) (SelectionEdit, error) {
	return p.group("DeleteRecords %v", refs)
}
func (p *modesFull) CopyRecords(refs []RecordRef) (InspectionClip, string, error) {
	p.calls.add("CopyRecords %v", refs)
	return modesClip(refs), "copied", nil
}
func (p *modesFull) PasteRecords(c InspectionClip, cell image.Point) (SelectionEdit, error) {
	return p.group("PasteRecords %v %v", c, cell)
}

// modesMod is a full host that saves into a mod.
type modesMod struct{ modesFull }

func (p *modesMod) ModTarget() string { return "demo" }
func (p *modesMod) SaveIntoMod() (string, error) {
	p.calls.add("SaveIntoMod")
	p.dirty = false
	return "mods/demo/map.alm", nil
}

func newModesEditor(t *testing.T, host string) (*MapEditor, *modesCalls) {
	t.Helper()
	calls := &modesCalls{}
	var edits InspectionEdits
	var base *modesBase
	switch host {
	case "plain":
		p := &modesBase{}
		edits, base = p, p
	case "mod":
		p := &modesMod{}
		edits, base = p, &p.modesBase
	default:
		p := &modesFull{}
		edits, base = p, &p.modesBase
	}
	base.calls, base.self = calls, edits
	base.players = []string{"Self", "Enemy"}
	build := func(source string, edits InspectionEdits) *InspectionDocument {
		v, err := NewViewer("modes", terrain.Grid{Width: 50, Height: 50, Tiles: make([]uint16, 2500)}, &terrain.Tileset{})
		if err != nil {
			t.Fatal(err)
		}
		d := &InspectionDocument{Viewer: v, Source: source, Width: 50, Height: 50, SourceBytes: []byte("map"), Edits: edits, Drops: []image.Point{{X: 3, Y: 3}}, ScriptNotice: "no script"}
		d.Records = []InspectionRecord{
			{Kind: "Unit", Index: 0, Section: 6, Cell: image.Pt(10, 10), Spatial: true, Size: image.Pt(1, 1), Label: "first"},
			{Kind: "Unit", Index: 1, Section: 6, Cell: image.Pt(20, 10), Spatial: true, Size: image.Pt(1, 1), Label: "second"},
			{Kind: "Structure", Index: 0, Section: 4, Cell: image.Pt(30, 30), Spatial: true, Size: image.Pt(2, 2), Label: "hall"},
			{Kind: "Sack", Index: 0, Section: 5, Cell: image.Pt(10, 30), Spatial: true, Size: image.Pt(1, 1), Label: "bag"},
			{Kind: "Condition", Index: 2, Section: 7},
			{Kind: "Action", Index: 0, Section: 7},
			{Kind: "Trigger", Index: 0, Section: 7, Warning: "unreachable"},
		}
		for i := 0; i < 24; i++ {
			d.Records = append(d.Records, InspectionRecord{Kind: "Scenery", Index: i, Section: 3, Label: "tree"})
		}
		for i, name := range base.players {
			d.Records = append(d.Records, InspectionRecord{Kind: "Player", Index: i, Section: 5, Label: name,
				Details: []string{"Name: " + name, fmt.Sprintf("Colour: %d", i), "Control: computer", "Relation to Self: hostile"}})
		}
		return d
	}
	base.doc = func(e InspectionEdits) *InspectionDocument { return build("edited", e) }
	var maps []EditorMap
	for i := 0; i < 14; i++ {
		maps = append(maps, EditorMap{Key: fmt.Sprintf("map%d", i), Label: fmt.Sprintf("Map %d", i)})
	}
	e := NewMapEditor(panelFont(), maps, func(source string) (*InspectionDocument, error) {
		return build(source, edits), nil
	})
	e.Layout(1024, 768)
	if err := e.Open("map0"); err != nil {
		t.Fatal(err)
	}
	// The popups stand in the window's bottom-right corner, clear of the
	// cells the gestures aim at.
	e.listAt, e.listPlaced = image.Pt(e.window.X, e.window.Y), true
	e.objectAt, e.objPlaced = image.Pt(e.window.X, e.window.Y), true
	return e, calls
}

var modesKeys = []string{"close", "escape", "open", "catalog", "enter", "report", "terrain", "height", "objects", "passability", "radiusup", "radiusdown", "up", "down", "saveas", "new", "unit", "structure", "stock", "recipe", "clone", "delete", "field", "tab", "sack", "drop", "goals", "undo", "redo", "backspace", "fit", "triggers", "pageup", "pagedown", "home", "end", "players", "player", "testplay", "scripts", "help", "metadata", "pages", "package", "selectall", "newline", "drawarea", "bogus", "left", "right", "ctrl+left", "ctrl+right", "shift+left", "shift+right", "shift+home", "shift+end", "ctrl+shift+left", "ctrl+shift+right", "copy", "cut", "paste", "passview", "cardup", "carddown"}

var modesGestures = []string{"ui:select0", "ui:select2", "ui:click-cell", "ui:click-unit", "ui:text-7", "ui:text-bad", "ui:text-plain", "ui:drag-unit", "ui:press-unit", "ui:press-cell", "ui:menu-maps", "ui:menu-open", "ui:menu-fit", "ui:bar-undo", "ui:bar-redo", "ui:bar-save", "ui:bar-units", "ui:list-row0", "ui:wheel-list"}

func modesPress(e *MapEditor, p image.Point, down bool) {
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: down}, time.Unix(1, 0))
}

func modesClick(e *MapEditor, p image.Point) {
	modesPress(e, p, true)
	modesPress(e, p, false)
}

// modesAct applies one key or gesture and reports the error text of a key.
func modesAct(e *MapEditor, name string) string {
	bar := func(key string) image.Point {
		for _, b := range e.barEntries() {
			if b.key == key {
				return center(b.rect)
			}
		}
		return image.Point{}
	}
	menu := func(key string, row int) {
		modesClick(e, bar(key))
		if b, ok := e.openMenuEntry(); ok {
			modesClick(e, center(e.menuDropdown(b, true).listBox(e.font).Row(row)))
		}
	}
	cell := func(x, y int) image.Point {
		e.Fit() // a selection may have scrolled the map; the gesture aims at the fitted view
		return e.CellPoint(image.Pt(x, y))
	}
	switch name {
	case "ui:select0":
		e.Select(0, false)
	case "ui:select2":
		e.Select(2, false)
	case "ui:group":
		e.Select(0, false)
		p := cell(20, 10)
		e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true, Shift: true}, time.Unix(1, 0))
		e.Pointer(Input{CursorX: p.X, CursorY: p.Y, Shift: true}, time.Unix(1, 0))
	case "ui:click-cell":
		modesClick(e, cell(25, 20))
	case "ui:click-unit":
		modesClick(e, cell(10, 10))
	case "ui:text-7":
		e.TextInput("7")
	case "ui:text-bad":
		e.TextInput("bad")
	case "ui:text-plain":
		e.TextInput("plain")
	case "ui:drag-unit":
		modesPress(e, cell(10, 10), true)
		modesPress(e, cell(14, 12), true)
		modesPress(e, cell(14, 12), false)
	case "ui:press-unit":
		modesPress(e, cell(10, 10), true)
	case "ui:press-cell":
		modesPress(e, cell(25, 20), true)
	case "ui:menu-maps":
		menu("map", 1)
	case "ui:menu-open":
		menu("map", 2)
	case "ui:menu-fit":
		n, _ := e.MenuRow("view", "fit")
		menu("view", n)
	case "ui:bar-undo":
		modesClick(e, bar("undo"))
	case "ui:bar-redo":
		modesClick(e, bar("redo"))
	case "ui:bar-save":
		modesClick(e, bar("save"))
	case "ui:bar-units":
		modesClick(e, bar("units"))
	case "ui:list-row0":
		if s, ok := e.listSpec(); ok {
			l := layoutPopup(s, e.font, e.bodyFont())
			modesClick(e, center(l.List.Row(0)))
		}
	case "ui:wheel-list":
		modesWheel(e)
	case "ui:esc-twice":
		_ = e.Key("escape")
		_ = e.Key("escape")
	case "ui:add-pick":
		if s, acts := e.addSpecOK(); acts != nil {
			for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
				if acts[rr.Index].button == "pick" {
					modesClick(e, center(rr.Button))
				}
			}
		}
	case "ui:pick-move":
		if s, _, ok := e.objectSpec(); ok && e.obj.cell >= 0 {
			for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
				if rr.Index == e.obj.cell {
					modesClick(e, center(rr.Button))
				}
			}
		}
	default:
		if err := e.Key(name); err != nil {
			return err.Error()
		}
	}
	return ""
}

func modesWheel(e *MapEditor) {
	s, ok := e.listSpec()
	if !ok {
		return
	}
	p := center(layoutPopup(s, e.font, e.bodyFont()).List.Row(0))
	for i := 0; i < 2; i++ {
		e.Pointer(Input{CursorX: p.X, CursorY: p.Y, WheelY: -0.1}, time.Unix(1, 0))
	}
}

// modesProbe is everything the author sees of the window, as text. pixels
// adds a hash of the composed panel.
func modesProbe(e *MapEditor, calls *modesCalls, errText string, pixels bool) string {
	var b strings.Builder
	rows, open := e.ReportOpen()
	fmt.Fprintf(&b, "err=%q msg=%q help=%q\n", errText, e.Message(), e.helpLines())
	list, cursor := e.listKind(), -1
	if s, ok := e.listSpec(); ok {
		cursor = s.List.Selection()
	}
	fmt.Fprintf(&b, "passview=%v\n", e.view.passability)
	fmt.Fprintf(&b, "report=%v/%d sel=%d brush=%v list=%q/%d selected=%d filter=%d dirty=%v\n", open, len(rows), e.ReportSelected(), e.brushActive(), list, cursor, e.Selected(), e.filter, e.Dirty())
	if s, ok := e.promptSpec(); ok {
		fmt.Fprintf(&b, "prompt %q", s.Title)
		for _, r := range s.Rows {
			fmt.Fprintf(&b, " %q", r.Value)
		}
		fmt.Fprintf(&b, " notes %q\n", s.Notes)
	}
	if e.doc != nil {
		fmt.Fprintf(&b, "source=%q\n", e.doc.Source)
	}
	fmt.Fprintf(&b, "calls=%q\n", calls.lines)
	if pixels {
		fmt.Fprintf(&b, "panel=%s\n", modesPanelHash(e))
	}
	return b.String()
}

// modesPanelHash hashes the composed panel. A panel that cannot be composed
// is recorded as such: the window draws a brush palette row beyond the
// palette when a report lists more rows than the palette has.
func modesPanelHash(e *MapEditor) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = "panic"
		}
	}()
	h := fnv.New32a()
	h.Write(e.ChromeImage().Pix)
	return fmt.Sprintf("%08x", h.Sum32())
}

type modesCont struct {
	name  string
	steps []string
	pre   bool // wheel twice before the action
}

var modesConts = []modesCont{
	{name: "now"},
	{name: "escapes", steps: []string{"escape", "escape", "escape", "escape"}},
	{name: "click-cell", steps: []string{"ui:click-cell"}},
	{name: "click-unit", steps: []string{"ui:click-unit"}},
	{name: "number", steps: []string{"ui:text-7", "enter"}},
	{name: "wheel", pre: true, steps: []string{"ui:wheel-rail"}},
}

func modesFullProbe(t *testing.T, host string, path []string, action string) string {
	saved := textClipboard
	textClipboard = newTextClipboard(true)
	defer func() { textClipboard = saved }()
	var out strings.Builder
	for _, c := range modesConts {
		e, calls := newModesEditor(t, host)
		for _, s := range path {
			modesAct(e, s)
		}
		if c.pre {
			modesWheel(e)
		}
		errText := modesAct(e, action)
		out.WriteString("== " + c.name + "\n" + modesProbe(e, calls, errText, c.name == "now"))
		for i, s := range c.steps {
			errText = modesAct(e, s)
			if c.name == "escapes" || i == len(c.steps)-1 {
				out.WriteString(fmt.Sprintf("-- %s\n", s) + modesProbe(e, calls, errText, false))
			}
		}
		if c.pre {
			modesWheel(e)
			out.WriteString(modesProbe(e, calls, "", false))
		}
	}
	return out.String()
}

type modesPath struct {
	name  string
	steps []string
}

var modesPaths = []modesPath{
	{"list", nil},
	{"selected", []string{"ui:select0"}},
	{"structure", []string{"ui:select2"}},
	{"catalog", []string{"catalog"}},
	{"catalog-open", []string{"catalog", "open"}},
	{"new", []string{"new"}},
	{"open", []string{"open"}},
	{"open-text", []string{"open", "ui:text-7"}},
	{"open-refused", []string{"open", "enter"}},
	{"saveas", []string{"saveas"}},
	{"saveas-bad", []string{"saveas", "ui:text-bad", "enter"}},
	{"saveas-plain", []string{"saveas", "ui:text-plain", "enter"}},
	{"saveas-ok", []string{"saveas", "ui:text-7", "enter"}},
	{"field", []string{"ui:select0", "field"}},
	{"field-tab", []string{"ui:select0", "field", "tab"}},
	{"field-none", []string{"field"}},
	{"clone", []string{"ui:select0", "clone"}},
	{"clone-none", []string{"clone"}},
	{"delete", []string{"ui:select0", "delete"}},
	{"sack", []string{"sack"}},
	{"sack-prompt", []string{"sack", "ui:click-cell"}},
	{"sack-prompt-tab", []string{"sack", "ui:click-cell", "tab"}},
	{"unit", []string{"unit"}},
	{"unit-prompt", []string{"unit", "ui:click-cell"}},
	{"unit-owner", []string{"unit", "enter"}},
	{"unit-cell", []string{"unit", "enter", "enter"}},
	{"unit-pick", []string{"unit", "enter", "enter", "ui:add-pick"}},
	{"unit-picked", []string{"unit", "enter", "enter", "ui:add-pick", "ui:click-cell"}},
	{"add-player", []string{"players", "player"}},
	{"add-script", []string{"scripts", "player"}},
	{"new-structure", []string{"structure"}},
	{"new-structure-prompt", []string{"structure", "ui:click-cell"}},
	{"drop", []string{"drop"}},
	{"goals", []string{"goals"}},
	{"goals-pick", []string{"goals", "enter"}},
	{"goals-pick-miss", []string{"goals", "enter", "ui:click-cell"}},
	{"goals-second", []string{"goals", "down"}},
	{"report", []string{"report"}},
	{"report-clone", []string{"report", "clone"}},
	{"report-field", []string{"report", "field"}},
	{"report-sack", []string{"report", "sack"}},
	{"report-unit", []string{"report", "unit"}},
	{"terrain", []string{"terrain"}},
	{"terrain-down", []string{"terrain", "down", "radiusup"}},
	{"height", []string{"height"}},
	{"objects", []string{"objects"}},
	{"objects-stroke", []string{"objects", "down", "ui:press-cell"}},
	{"objects-report", []string{"objects", "report"}},
	{"passability", []string{"passability"}},
	{"passability-stroke", []string{"passability", "down", "ui:press-cell"}},
	{"terrain-open", []string{"terrain", "open"}},
	{"terrain-clone", []string{"ui:select0", "terrain", "clone"}},
	{"terrain-report", []string{"terrain", "report"}},
	{"terrain-goals", []string{"terrain", "goals"}},
	{"terrain-catalog", []string{"terrain", "catalog"}},
	{"terrain-stroke", []string{"terrain", "ui:press-cell"}},
	{"terrain-message", []string{"terrain", "delete"}},
	{"drag", []string{"ui:press-unit"}},
	{"message", []string{"delete"}},
	{"dirty", []string{"ui:select0", "clone", "ui:click-cell"}},
	{"dirty-catalog", []string{"ui:select0", "clone", "ui:click-cell", "catalog"}},
	{"triggers", []string{"triggers"}},
	{"scripts", []string{"scripts"}},
	{"paged", []string{"pagedown"}},
	{"players", []string{"players"}},
	{"player-name", []string{"players", "down", "field"}},
	{"player-number", []string{"players", "down", "field", "tab"}},
}

// modesSmallPaths are the paths also run against the hosts without the
// optional seams and against the mod host.
var modesSmallPaths = map[string]bool{"list": true, "selected": true, "catalog": true, "saveas": true, "saveas-bad": true, "field": true, "clone": true, "sack": true, "sack-prompt": true, "unit": true, "terrain": true, "drop": true, "goals": true, "report": true, "dirty": true, "open": true, "players": true, "player-name": true}

func modesGoldenLines(t *testing.T) []string {
	actions := append(append([]string(nil), modesKeys...), modesGestures...)
	var lines []string
	for _, host := range []string{"full", "plain", "mod"} {
		for _, p := range modesPaths {
			if host != "full" && !modesSmallPaths[p.name] {
				continue
			}
			var hashes []string
			for _, a := range actions {
				h := fnv.New32a()
				h.Write([]byte(modesFullProbe(t, host, p.steps, a)))
				hashes = append(hashes, fmt.Sprintf("%08x", h.Sum32()))
			}
			lines = append(lines, host+"/"+p.name+" "+strings.Join(hashes, " "))
		}
	}
	return lines
}

const modesGoldenPath = "testdata/mapeditor_modes.golden"

// TestMapEditorModesHoldEveryKeyAndGesture compares the window's behaviour in
// every entry path with the recorded table. A difference names the path and
// the action columns that changed.
func TestMapEditorModesHoldEveryKeyAndGesture(t *testing.T) {
	got := modesGoldenLines(t)
	if os.Getenv("MAPEDITOR_MODES_WRITE") != "" {
		if err := os.WriteFile(modesGoldenPath, []byte(strings.Join(got, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(modesGoldenPath)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Split(strings.TrimRight(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n"), "\n")
	if len(got) != len(want) {
		t.Fatalf("%d rows, recorded %d", len(got), len(want))
	}
	actions := append(append([]string(nil), modesKeys...), modesGestures...)
	for i := range got {
		if got[i] == want[i] {
			continue
		}
		g, w := strings.Fields(got[i]), strings.Fields(want[i])
		var diff []string
		for c := 1; c < len(g) && c < len(w); c++ {
			if g[c] != w[c] {
				diff = append(diff, actions[c-1])
			}
		}
		t.Errorf("%s differs on %v", g[0], diff)
	}
}

// modesRow is one explicit case: the entry path, an action, and the help
// line, message and prompt title that follow.
type modesRow struct {
	host, path string
	action     string
	help       string // first help line
	message    string
	title      string // gold modal title, if a prompt is open
	report     bool
	brush      bool
}

func modesTitle(e *MapEditor) string {
	if !e.prompt.open() {
		return ""
	}
	return editorClip(e.font, e.promptTitle(), 300)
}

func modesPathSteps(name string) []string {
	for _, p := range modesPaths {
		if p.name == name {
			return p.steps
		}
	}
	panic("no path " + name)
}

// TestMapEditorModesReadableTable states the main transitions in words.
func TestMapEditorModesReadableTable(t *testing.T) {
	keys := "N: new map / U: unit"
	prompt := "Enter: set / Tab: next / Esc: back"
	players := "P: records / A: add / Del: delete"
	add := "Enter: next / Esc: cancel"
	rows := []modesRow{
		{"full", "list", "clone", keys, "Select a unit, structure or sack first.", "", false, false},
		{"full", "selected", "clone", "Clone: click a cell", "", "", false, false},
		{"full", "selected", "ui:click-cell", keys, "", "", false, false},
		{"full", "clone", "escape", keys, "", "", false, false},
		{"full", "sack", "escape", add, "", "Add sack", false, false},
		{"full", "list", "sack", "Cell: click a cell", "", "Add sack", false, false},
		{"full", "list", "unit", "Cell: click a cell", "", "Add unit", false, false},
		{"full", "unit-cell", "ui:add-pick", "Cell: click a cell", "", "Add unit", false, false},
		{"full", "unit-pick", "ui:click-cell", keys, "", "", false, false},
		{"full", "unit-pick", "escape", add, "", "Add unit", false, false},
		{"plain", "list", "unit", keys, "", "", false, false},
		{"full", "list", "drop", "Drop zones: click / drag / Del", "", "", false, false},
		{"full", "list", "goals", "Enter: pick a target", "", "", true, false},
		{"full", "goals", "enter", "Target: click a unit or", "", "", true, false},
		{"full", "goals", "goals", keys, "", "", false, false},
		{"full", "goals-pick", "ui:click-unit", "Enter: pick a target", "", "", true, false},
		{"full", "goals-pick", "ui:click-cell", "Enter: pick a target", "Click a unit on the map.", "", true, false},
		{"full", "list", "report", keys, "", "", true, false},
		{"full", "report", "report", keys, "", "", false, false},
		{"full", "list", "terrain", "T: off / Drag: paint / [ ]: size", "", "", false, true},
		{"full", "terrain", "terrain", keys, "", "", false, false},
		{"full", "terrain", "height", "H: off / Drag: shape / [ ]: size", "", "", false, true},
		{"plain", "list", "terrain", keys, "", "", false, false},
		{"full", "terrain", "clone", "T: off / Drag: paint / [ ]: size", "Select a unit, structure or sack first.", "", false, true},
		{"full", "terrain", "open", prompt, "", "ALM path / Enter: open", false, false},
		{"full", "terrain-open", "escape", "T: off / Drag: paint / [ ]: size", "", "", false, true},
		{"full", "list", "open", prompt, "", "ALM path / Enter: open", false, false},
		{"full", "list", "saveas", prompt, "", "New .alm path / Enter: Save As", false, false},
		{"mod", "list", "saveas", keys, "", "", true, false},
		{"full", "selected", "field", prompt, "", "First / Tab: next / Enter: set", false, false},
		{"full", "field", "tab", prompt, "", "Second / Tab: next / Enter: set", false, false},
		{"full", "list", "field", keys, "Select a unit, structure or sack first.", "", false, false},
		{"full", "sack", "ui:click-cell", keys, "", "", false, false},
		{"full", "unit", "ui:click-cell", keys, "", "", false, false},
		{"full", "sack-prompt", "escape", keys, "", "", false, false},
		{"full", "list", "catalog", keys, "", "", false, false},
		{"full", "catalog-open", "escape", keys, "", "", false, false},
		{"full", "catalog", "escape", keys, "", "", false, false},
		{"full", "saveas-bad", "escape", keys, "", "", false, false},
		{"full", "saveas-ok", "escape", keys, "", "", false, false},
		{"full", "report-clone", "escape", keys, "", "", true, false},
		{"full", "list", "new", prompt, "", "New map: width x height / Enter", false, false},
		{"full", "report", "new", prompt, "", "New map: width x height / Enter", false, false},
		{"full", "list", "players", players, "", "", false, false},
		{"full", "players", "field", prompt, "", "Name / Tab: next / Enter: set", false, false},
		{"full", "player-name", "tab", prompt, "", "Colour (0..15) / Tab: next / Enter: set", false, false},
		{"full", "players", "player", add, "", "Add player", false, false},
		{"plain", "players", "player", players, "", "", false, false},
		{"full", "players", "delete", players, "The first player cannot be deleted.", "", false, false},
		{"full", "players", "escape", players, "", "", false, false}, // Esc closes the player's popup first
		{"full", "players", "ui:esc-twice", keys, "", "", false, false},
	}
	for _, r := range rows {
		e, _ := newModesEditor(t, r.host)
		for _, s := range modesPathSteps(r.path) {
			modesAct(e, s)
		}
		modesAct(e, r.action)
		_, report := e.ReportOpen()
		// A brush screen leads with the size line all brushes share; the
		// brush's own line follows it.
		lines := e.helpLines()
		if e.brushActive() && len(lines) > 1 {
			lines = lines[1:]
		}
		if help := lines[0]; help != r.help || e.Message() != r.message || modesTitle(e) != r.title || report != r.report || e.brushActive() != r.brush {
			t.Errorf("%s %s then %s: help %q message %q title %q report %v brush %v; recorded %q %q %q %v %v", r.host, r.path, r.action, help, e.Message(), modesTitle(e), report, e.brushActive(), r.help, r.message, r.title, r.report, r.brush)
		}
	}
}
