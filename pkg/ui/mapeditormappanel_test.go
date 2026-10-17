package ui

import (
	"fmt"
	"image"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// mapPanelEdits is a host with the Map panel's facts, a report, the metadata
// fields, the script, test play and a mod save.
type mapPanelEdits struct {
	noUnitAuthoring
	facts    InspectionMapFacts
	rows     []InspectionFinding
	plays    int
	modSaves int
	mode     int64
}

func (p *mapPanelEdits) Dirty() bool                                                { return false }
func (p *mapPanelEdits) CanUndo() bool                                              { return false }
func (p *mapPanelEdits) CanRedo() bool                                              { return false }
func (p *mapPanelEdits) Undo() (*InspectionDocument, error)                         { return nil, nil }
func (p *mapPanelEdits) Redo() (*InspectionDocument, error)                         { return nil, nil }
func (p *mapPanelEdits) SaveAs(path string) (string, error)                         { return path, nil }
func (p *mapPanelEdits) Move(string, int, image.Point) (*InspectionDocument, error) { return nil, nil }
func (p *mapPanelEdits) Report() []InspectionFinding                                { return p.rows }
func (p *mapPanelEdits) MapFacts() InspectionMapFacts                               { return p.facts }
func (p *mapPanelEdits) TestPlay() (string, error) {
	p.plays++
	return "playing", nil
}
func (p *mapPanelEdits) ModTarget() string { return "demo" }
func (p *mapPanelEdits) SaveIntoMod() (string, error) {
	p.modSaves++
	return "mods/demo/map.alm", nil
}
func (p *mapPanelEdits) Goals() []InspectionGoal {
	return []InspectionGoal{{Arm: "Win", Text: "add: the death of a unit wins", Pick: "Unit", Index: -1}}
}
func (p *mapPanelEdits) SetGoalTarget(int, string, int) (*InspectionDocument, error) {
	return nil, nil
}
func (p *mapPanelEdits) SetDropCell(image.Point) (*InspectionDocument, error) { return nil, nil }
func (p *mapPanelEdits) WorkspaceFields(scope string) []InspectionField {
	if scope != "metadata" {
		return nil
	}
	return []InspectionField{
		{Key: "name", Label: "Name", Text: true, TextValue: p.facts.Name},
		{Key: "description", Label: "Description", Text: true, TextValue: p.facts.Description},
		{Key: "angle", Label: "Raw angle", Value: 1},
		{Key: ModeFieldKey, Label: "Mode", Value: p.mode, Choices: []InspectionChoice{{Value: 1, Label: "Single-player"}, {Value: 4, Label: "Multiplayer"}}},
	}
}
func (p *mapPanelEdits) SetWorkspaceField(_, key, value string) (*InspectionDocument, error) {
	if key == ModeFieldKey {
		p.mode, _ = strconv.ParseInt(value, 10, 64)
	}
	return nil, nil
}

// playable is the facts of a map with a start point and a win.
func playable() InspectionMapFacts {
	return InspectionMapFacts{Name: "Bridge", Description: "Hold the bridge.", Mode: MapModeSingle, Players: 2, Sides: 2,
		Drops: []image.Point{{X: 12, Y: 14}}, Wins: []string{"Bridge held"}, Losses: []string{"Bridge lost"}, Protects: 1}
}

// endless is the facts and findings of a map with no start point and no win.
func endless() (InspectionMapFacts, []InspectionFinding) {
	f := InspectionMapFacts{Mode: MapModeMultiplayer, Players: 2, Sides: 1}
	return f, []InspectionFinding{
		{Rule: RuleDropCell, Text: "the map has no start point"},
		{Rule: RuleVictory, Text: "nothing wins this map"},
	}
}

func mapPanelEditor(t *testing.T, p *mapPanelEdits, maps []EditorMap) (*MapEditor, *[]string) {
	t.Helper()
	opened := &[]string{}
	open := func(source string) (*InspectionDocument, error) {
		*opened = append(*opened, source)
		v, err := NewViewer("editor", terrain.Grid{Width: 50, Height: 50, Tiles: make([]uint16, 2500)}, &terrain.Tileset{})
		if err != nil {
			return nil, err
		}
		return &InspectionDocument{Viewer: v, Source: source, Width: 50, Height: 50, SourceBytes: []byte("map"), Edits: p}, nil
	}
	e := NewMapEditor(panelFont(), maps, open)
	e.Layout(1024, 768)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	return e, opened
}

func clickPanelRow(t *testing.T, e *MapEditor, n int) {
	t.Helper()
	r, ok := e.PanelRowRect(n)
	if !ok {
		t.Fatalf("panel row %d is not shown", n)
	}
	p := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Unix(1, 0))
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
}

// panelRow is the index of the panel row whose text starts with label.
func panelRow(t *testing.T, e *MapEditor, label string) int {
	t.Helper()
	for n, l := range e.PanelLines() {
		if strings.HasPrefix(l, label+" ") {
			return n
		}
	}
	t.Fatalf("no panel line %q in %q", label, e.PanelLines())
	return -1
}

// Each line of the Map panel states the map's value in player words and
// opens the place that sets it. A removed line fails its own subtest.
func TestMapPanelLinesStateTheMapAndOpenTheirPlace(t *testing.T) {
	lines := []struct {
		name, line string
		opens      func(e *MapEditor, p *mapPanelEdits) bool
	}{
		{"size", "Size 50 x 50", nil},
		{"name", "Name Bridge", func(e *MapEditor, _ *mapPanelEdits) bool {
			return e.prompt.kind == promptWorkspace && e.prompt.scope == "metadata" && e.prompt.field == 0 && e.prompt.path == "Bridge"
		}},
		{"description", "Description Hold the bridge.", func(e *MapEditor, _ *mapPanelEdits) bool {
			return e.prompt.kind == promptWorkspace && e.prompt.field == 1
		}},
		{"players", "Players 2 players, 2 sides", func(e *MapEditor, _ *mapPanelEdits) bool {
			return e.listKind() == "records" && e.filter == editorPlayersFilter
		}},
		{"mode", "Mode Single-player", func(e *MapEditor, _ *mapPanelEdits) bool {
			return e.prompt.kind == promptWorkspace && e.prompt.field == 3 && e.prompt.path == "1"
		}},
		{"start", "Start point cell 12,14", func(e *MapEditor, _ *mapPanelEdits) bool { return e.pick == pickDrop }},
		{"victory", "Victory the trigger 'Bridge held' wins the map", func(e *MapEditor, _ *mapPanelEdits) bool { return len(e.report.goals) > 0 }},
		{"defeat", "Defeat 2 rules lose the map; the hero's fall loses it too", func(e *MapEditor, _ *mapPanelEdits) bool { return len(e.report.goals) > 0 }},
		{"report", "Report 0 errors, 0 warnings", func(e *MapEditor, _ *mapPanelEdits) bool {
			return e.report.open && len(e.report.goals) == 0
		}},
		{"test play", "Test play F5: play the map now", func(e *MapEditor, p *mapPanelEdits) bool { return p.plays == 1 }},
	}
	for _, l := range lines {
		t.Run(l.name, func(t *testing.T) {
			p := &mapPanelEdits{facts: playable(), mode: 1}
			e, _ := mapPanelEditor(t, p, nil)
			if !e.PanelOpen() {
				t.Fatal("the Map panel did not open with the map")
			}
			n := slices.Index(e.PanelLines(), l.line)
			if n < 0 {
				t.Fatalf("no line %q in %q", l.line, e.PanelLines())
			}
			if l.opens == nil {
				return
			}
			clickPanelRow(t, e, n)
			if !l.opens(e, p) {
				t.Fatalf("the line %q did not open its place: prompt %+v pick %q list %q", l.line, e.prompt, e.pick, e.listKind())
			}
		})
	}
}

// The two checks mark their lines with the findings of their rules; without
// the finding the line is not marked.
func TestMapPanelChecksMarkTheStartAndVictoryLines(t *testing.T) {
	f, rows := endless()
	p := &mapPanelEdits{facts: f, rows: rows}
	e, _ := mapPanelEditor(t, p, nil)
	s, _ := e.panelSpec()
	ink := func(label string) bool {
		return s.Rows[panelLineRow(s, panelRow(t, e, label))].Ink == editorWarning
	}
	for _, want := range []string{
		"Start point none: the party lands at a random cell",
		"Victory nothing wins this map: it is endless",
		"Mode Multiplayer",
		"Defeat only the hero's fall loses the map",
		"Report 0 errors, 2 warnings",
	} {
		if !slices.Contains(e.PanelLines(), want) {
			t.Errorf("no line %q in %q", want, e.PanelLines())
		}
	}
	if !ink("Start point") || !ink("Victory") || ink("Defeat") || ink("Mode") {
		t.Fatal("the checks marked the wrong lines")
	}
	// Loss control: without the findings neither line is marked.
	p.rows = nil
	p.facts = playable()
	s, _ = e.panelSpec()
	if ink("Start point") || ink("Victory") {
		t.Fatal("a line is marked without a finding")
	}
	// A drop cell the rule names is stated as one the party cannot land on.
	p.rows = []InspectionFinding{{Rule: RuleDropCell, Cell: image.Pt(12, 14), HasCell: true, Text: "blocked"}}
	if !slices.Contains(e.PanelLines(), "Start point cell 12,14: the party cannot land there") {
		t.Fatalf("blocked drop: %q", e.PanelLines())
	}
}

// The Mode line names the two values a new map is given and any other value
// as itself with the mode it gives.
func TestMapPanelModeWords(t *testing.T) {
	e, _ := mapPanelEditor(t, &mapPanelEdits{facts: playable()}, nil)
	for v, want := range map[uint32]string{1: "Single-player", 4: "Multiplayer", 16: "16 (multiplayer)", 0: "0 (single-player)", 2: "2 (multiplayer)"} {
		if got := e.modeWord(v); got != want {
			t.Errorf("mode %d: %q, want %q", v, got, want)
		}
	}
}

// The panel opens first with each map, Esc and its close button close it, and
// the Map menu brings it back over a records list.
func TestMapPanelOpensFirstAndFromTheMapMenu(t *testing.T) {
	e, _ := mapPanelEditor(t, &mapPanelEdits{facts: playable()}, nil)
	_ = e.Key("escape")
	if e.PanelOpen() {
		t.Fatal("Esc did not close the panel")
	}
	e.showRecords(1)
	e.menuAct("panel")
	if !e.PanelOpen() || e.listShown {
		t.Fatalf("the Map menu did not bring the panel back: %q", e.listKind())
	}
	items := e.menuItems("map")
	if !slices.Contains(items, menuItem{"panel", "Map panel"}) {
		t.Fatalf("map menu %v", items)
	}
	s, _ := e.panelSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	e.Pointer(Input{CursorX: l.Close.Min.X + 2, CursorY: l.Close.Min.Y + 2, PrimaryDown: true}, time.Unix(1, 0))
	e.Pointer(Input{CursorX: l.Close.Min.X + 2, CursorY: l.Close.Min.Y + 2}, time.Unix(1, 0))
	if e.PanelOpen() {
		t.Fatal("the close button did not close the panel")
	}
	if err := e.Open("scenario/11.alm"); err != nil || !e.PanelOpen() {
		t.Fatalf("the panel did not open with the next map: %v", err)
	}
	// A host without the panel's facts opens no panel.
	r := reportEditor(t, &reportEdits{})
	if r.PanelOpen() || r.panel {
		t.Fatal("a host without facts opened the panel")
	}
}

// Before test play and before a save into the mod, a map the checks find
// wanting gets one notice the author can continue past or leave.
func TestMapPanelNoticeBeforeTestPlayAndModSave(t *testing.T) {
	for _, step := range []struct {
		key   string
		title string
		count func(p *mapPanelEdits) int
	}{
		{"testplay", "Before test play", func(p *mapPanelEdits) int { return p.plays }},
		{"saveas", "Before saving into the mod", func(p *mapPanelEdits) int { return p.modSaves }},
	} {
		t.Run(step.key, func(t *testing.T) {
			f, rows := endless()
			p := &mapPanelEdits{facts: f, rows: rows}
			e, _ := mapPanelEditor(t, p, nil)
			_ = e.Key(step.key)
			if e.prompt.kind != promptNotice || step.count(p) != 0 {
				t.Fatalf("no notice: prompt %+v, %d runs", e.prompt, step.count(p))
			}
			s := e.noticeSpec()
			notes := strings.Join(s.Notes, " / ")
			if s.Title != step.title || !strings.Contains(notes, "Start point: none: the party lands at a random cell") ||
				!strings.Contains(notes, "Victory: nothing wins this map: it is endless") || s.Buttons[0] != "Continue" {
				t.Fatalf("notice %q: %q %v", s.Title, notes, s.Buttons)
			}
			_ = e.Key("escape")
			if e.prompt.open() || step.count(p) != 0 {
				t.Fatal("Esc did not leave the notice without the step")
			}
			_ = e.Key(step.key)
			_ = e.Key("enter")
			if e.prompt.kind == promptNotice || step.count(p) != 1 {
				t.Fatalf("Enter did not continue: %d runs", step.count(p))
			}
			// The Continue button does the same.
			_ = e.Key(step.key)
			l := layoutPopup(e.noticeSpec(), e.font, e.bodyFont())
			b := l.Buttons[noticeGo]
			e.Pointer(Input{CursorX: b.Min.X + 2, CursorY: b.Min.Y + 2, PrimaryDown: true}, time.Unix(1, 0))
			e.Pointer(Input{CursorX: b.Min.X + 2, CursorY: b.Min.Y + 2}, time.Unix(1, 0))
			if step.count(p) != 2 {
				t.Fatalf("Continue did not run the step: %d runs", step.count(p))
			}
			// Loss controls: a playable map, and a map whose report holds an
			// error, which refuses the step itself, get no notice.
			p.facts, p.rows = playable(), nil
			_ = e.Key(step.key)
			if e.prompt.kind == promptNotice || step.count(p) != 3 {
				t.Fatal("a playable map got the notice")
			}
			p.facts, p.rows = f, append(slices.Clone(rows), InspectionFinding{Error: true, Rule: "owner-slot", Text: "x"})
			_ = e.Key(step.key)
			if e.prompt.kind == promptNotice {
				t.Fatal("a map with an error got the notice")
			}
		})
	}
}

// A new map asks single-player or multiplayer, single by default, and opens
// with the mode in its source; Cancel opens nothing.
func TestNewMapAsksSingleOrMultiplayer(t *testing.T) {
	maps := []EditorMap{{Key: "new/32x32", Label: "New map 32 x 32"}, {Key: "scenario/10.alm", Label: "10"}}
	press := func(e *MapEditor, n int) {
		l := layoutPopup(e.newModeSpec(), e.font, e.bodyFont())
		b := l.Buttons[n]
		e.Pointer(Input{CursorX: b.Min.X + 2, CursorY: b.Min.Y + 2, PrimaryDown: true}, time.Unix(1, 0))
		e.Pointer(Input{CursorX: b.Min.X + 2, CursorY: b.Min.Y + 2}, time.Unix(1, 0))
	}
	for _, c := range []struct {
		name string
		act  func(e *MapEditor)
		want string
	}{
		{"enter", func(e *MapEditor) { _ = e.Key("enter") }, "new/32x32"},
		{"single", func(e *MapEditor) { press(e, newModeSingle) }, "new/32x32"},
		{"multiplayer", func(e *MapEditor) { press(e, newModeMulti) }, "new/32x32" + NewMapMultiplayer},
		{"cancel", func(e *MapEditor) { press(e, newModeStay) }, ""},
	} {
		for _, door := range []string{"catalog", "size"} {
			t.Run(c.name+"/"+door, func(t *testing.T) {
				e, opened := mapPanelEditor(t, &mapPanelEdits{facts: playable()}, maps)
				*opened = nil
				if door == "catalog" {
					_ = e.Key("catalog")
					e.listSpec() // the picker's model, as a drawn frame makes it
					e.listChoose(1, true)
				} else {
					_ = e.Key("new")
					_ = e.Key("clear")
					e.TextInput("32x32")
					_ = e.Key("enter")
				}
				if e.prompt.kind != promptNewMode || len(*opened) != 0 {
					t.Fatalf("no question: prompt %+v opened %v", e.prompt, *opened)
				}
				s := e.newModeSpec()
				if s.Title != "New map: single-player or multiplayer?" || fmt.Sprint(s.Buttons) != "[Single-player Multiplayer Cancel]" {
					t.Fatalf("question %q %v", s.Title, s.Buttons)
				}
				c.act(e)
				got := ""
				if len(*opened) > 0 {
					got = (*opened)[len(*opened)-1]
				}
				if got != c.want || e.prompt.kind == promptNewMode {
					t.Fatalf("opened %q, want %q; prompt %+v", got, c.want, e.prompt)
				}
			})
		}
	}
}

// The panel's lines use the words of the editor's language.
func TestMapPanelWordsAreTheBooks(t *testing.T) {
	e, _ := mapPanelEditor(t, &mapPanelEdits{facts: playable()}, nil)
	for _, id := range []string{"editor.panel.title", "editor.panel.start.none", "editor.panel.victory.none", "editor.mode.single", "editor.mode.multi",
		"editor.notice.testplay", "editor.notice.modsave", "editor.notice.continue", "editor.newmap.title", "editor.menu.panel"} {
		if w := e.word(id); w == "" || w == id {
			t.Errorf("%s has no English word: %q", id, w)
		}
	}
}
