package ui

import (
	"errors"
	"fmt"
	"image"
	"strconv"
	"strings"
	"testing"
	"time"

	"againrom/pkg/words"
	"github.com/hajimehoshi/ebiten/v2"
)

type modesWorkspace struct {
	modesFull
	fields  []InspectionField
	maps    InspectionField // the mod's list of maps
	confirm *WorkspaceConfirmation
	leave   *WorkspaceLeave
	// dirtyOnEdit makes any field edit leave unsaved changes behind.
	dirtyOnEdit bool
	// overview is the mod's overview page; empty, W has none.
	overview ModOverviewPage
	// graph is the campaign graph; empty, W has no graph page.
	graph         CampaignGraphPage
	graphFields   []InspectionField
	handoutFields []InspectionField
	graphRefusal  error
	// inMod makes another map open beside the open one, as in an open mod.
	inMod       bool
	actions     []ModMapAction
	actionsOpen bool
}

func (p *modesWorkspace) InMod() bool                                    { return p.inMod }
func (p *modesWorkspace) MissionNumber() int                             { return 25 }
func (p *modesWorkspace) MapDirty() bool                                 { return p.dirty }
func (p *modesWorkspace) MapSource() string                              { return "" }
func (p *modesWorkspace) Shown(InspectionEdits, []InspectionEdits) error { return nil }

// modesOverview is the overview of a mod no mission of which opens.
var modesOverview = ModOverviewPage{First: true, Sections: []ModOverviewSection{
	{Heading: "Kinds the mod holds", Lines: []string{"terrain: data/terrain.toml"}},
	{Heading: "Report (againrom -check)", Lines: []string{"1 mod(s) active on rom1-en", "mod snow-field 1.0.0"}},
	{Heading: "Files (2)", Lines: []string{"data/terrain.toml", "mod.toml"}, Fold: true},
	{Heading: "Refusals", Lines: []string{"none"}},
}}

func (p *modesWorkspace) ModOverview() ModOverviewPage { return p.overview }

func (p *modesWorkspace) CampaignGraph() CampaignGraphPage { return p.graph }

func newModesWorkspace() *modesWorkspace {
	return &modesWorkspace{fields: []InspectionField{
		{Key: "folder", Label: "Folder", Text: true, TextValue: "working"},
		{Key: "title", Label: "Name", Text: true, TextValue: "River"},
		{Key: "after", Label: "After win", Value: 20, Column: 1, Choices: []InspectionChoice{{Value: 20, Label: "Campaign 20"}, {Value: 25, Label: "Crossing"}}},
		{Key: "source", Label: "Mission", Value: 1, Choices: []InspectionChoice{{Value: 1, Label: "Crossing (25)"}, {Value: 2, Label: "Watch (26)"}}},
		{Key: "party.0.character", Label: "Member / Leader", Value: 0, Column: 1, Choices: []InspectionChoice{{Value: 0, Label: "Aila"}, {Value: 1, Label: "Sellsword"}}},
		{Key: "party.0.level", Label: "Level", Value: 2, Column: 1, Parent: "party.0."},
		{Key: "add", Label: "Add open map", Action: true},
	}, maps: InspectionField{Key: "map", Label: "Maps of the mod", ReadOnly: true, Value: 1, Choices: []InspectionChoice{
		{Value: 1, Label: "crossing.alm: guest 25 after 20"}, {Value: 2, Label: "watch.alm: guest 26 after 25, replaces 3"},
		{Value: 3, Label: "7.alm: replaces 7 by name"}, {Value: 4, Label: "spare.alm: unused"}, {Value: 5, Label: "gone.alm: missing"}},
		ChoiceLines: [][]string{nil, {"watch.alm: guest 26 after 25,", "replaces 3"}}}}
}

// ModMaps is the package field "map": the mod's list of maps.
func (p *modesWorkspace) ModMaps() InspectionField { return p.maps }

func (p *modesWorkspace) WorkspaceFields(scope string) []InspectionField {
	if strings.HasPrefix(scope, "campaign.") && scope != "campaign.0" {
		return p.graphFields
	}
	if scope == graphPickScope {
		return p.handoutFields
	}
	if scope == "package" {
		return p.fields
	}
	return nil
}
func (p *modesWorkspace) SetWorkspaceField(scope, key, value string) (*InspectionDocument, error) {
	fields := p.fields
	if strings.HasPrefix(scope, "campaign.") {
		if p.graphRefusal != nil {
			return nil, p.graphRefusal
		}
		fields = p.graphFields
	}
	if scope == graphPickScope {
		fields = p.handoutFields
	}
	if key == "folder" && p.leave != nil {
		return nil, p.leave
	}
	if key == "publish" && p.confirm != nil {
		return nil, p.confirm
	}
	if n, err := strconv.ParseInt(value, 10, 64); key == "map" && err == nil {
		p.maps.Value = n
	}
	for i := range fields {
		if fields[i].Key == key {
			if fields[i].Text {
				fields[i].TextValue = value
			} else {
				n, e := strconv.ParseInt(value, 10, 64)
				if e != nil && !fields[i].Action {
					return nil, e
				}
				fields[i].Value = n
			}
		}
	}
	if p.dirtyOnEdit && key != "folder" {
		p.leave = &WorkspaceLeave{Folder: "other-mod"}
	}
	return p.next("Workspace %s %s %q", scope, key, value)
}

func TestModWindowUsesTheSharedSaveDiscardCancelQuestion(t *testing.T) {
	for _, choice := range []int{leaveSave, leaveGo, leaveStay} {
		e, calls := newModesEditor(t, "workspace")
		p := e.doc.Edits.(*modesWorkspace)
		p.leave = &WorkspaceLeave{Folder: "other-mod"}
		_ = e.Key("package")
		if !e.FocusModField("folder") {
			t.Fatal("missing folder field")
		}
		_ = e.Key("clear")
		e.TextInput("other-mod")
		_ = e.Key("enter")
		if !asked(e) || !e.prompt.then.workspace {
			t.Fatal("folder change did not reach the shared question")
		}
		leaveButton(t, e, choice)
		if choice == leaveSave {
			if e.prompt.kind != promptSave {
				t.Fatal("Save did not use Save As", e.prompt.kind)
			}
			_ = e.Key("clear")
			e.TextInput("copy.alm")
			if err := e.Key("enter"); err != nil {
				t.Fatal(err)
			}
		}
		opened := false
		for _, line := range calls.lines {
			opened = opened || line == "Workspace package folder-open \"other-mod\""
		}
		if opened != (choice != leaveStay) || e.ExitAsked() {
			t.Fatal("question chose the wrong continuation", choice, calls.lines)
		}
	}
}

func TestModWindowPublishConfirmationKeepsItsTargetAndShowsTheExplicitAction(t *testing.T) {
	e, calls := newModesEditor(t, "workspace")
	p := e.doc.Edits.(*modesWorkspace)
	p.confirm = &WorkspaceConfirmation{Action: "publish-replace", Message: "Replace this target?", Confirm: "Replace"}
	_ = e.Key("package")
	if !e.FocusModField("publish") {
		t.Fatal("missing Publish footer")
	}
	_ = e.Key("enter")
	s, _, _ := e.modSpec()
	if e.modConfirm != "publish-replace" || len(s.Buttons) != 2 || s.Buttons[0] != "Replace" {
		t.Fatal("missing explicit confirmation", s.Buttons)
	}
	before := len(calls.lines)
	for _, key := range []string{"cardup", "saveas", "testplay", "undo"} {
		_ = e.Key(key)
	}
	for range 5 {
		_ = e.Key("tab")
		if e.obj.focus < len(e.obj.fields) {
			t.Fatal("confirmation focus entered fields")
		}
	}
	if len(calls.lines) != before {
		t.Fatal("confirmation changed the target", calls.lines)
	}
	_ = e.Key("escape")
	if e.modConfirm != "" || len(calls.lines) != before {
		t.Fatal("Cancel performed an action")
	}
}

func TestModWindowTabAnswersAWaitingConfirmationAsCancel(t *testing.T) {
	for _, byKey := range []bool{false, true} {
		e, calls := newModesEditor(t, "workspace")
		p := e.doc.Edits.(*modesWorkspace)
		page := modesOverview
		page.First = false
		p.overview, p.graph = page, modesGraph()
		p.confirm = &WorkspaceConfirmation{Action: "publish-replace", Message: "Replace this target?", Confirm: "Replace"}
		_ = e.Key("package")
		if e.modPage != modPageFields {
			t.Fatal("the window opens on page", e.modPage)
		}
		if !e.FocusModField("publish") {
			t.Fatal("missing Publish footer")
		}
		_ = e.Key("enter")
		if e.modConfirm == "" {
			t.Fatal("no confirmation waits")
		}
		before := len(calls.lines)
		if byKey {
			_ = e.Key("package")
		} else {
			s, _, _ := e.modSpec()
			r := layoutPopup(s, e.font, e.bodyFont()).Tabs[1]
			e.modPress(r.Min.Add(r.Size().Div(2)))
		}
		if e.modPage != modPageOverview || e.modConfirm != "" || len(calls.lines) != before {
			t.Fatalf("byKey %v: page %d confirm %q calls %v", byKey, e.modPage, e.modConfirm, calls.lines[before:])
		}
	}
}

func TestModWindowMouseMissionSwitchAcceptsTheFocusedField(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	_ = e.Key("package")
	e.FocusModField("title")
	_ = e.Key("clear")
	e.TextInput("Accepted by click")
	s, _, _ := e.modSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	clicked := false
	for _, row := range l.Rows {
		if actions := e.obj.rowActs[row.Index]; len(actions) > 0 && actions[0] == "2" && len(row.Buttons) > 0 {
			modesClick(e, center(row.Buttons[0]))
			clicked = true
			break
		}
	}
	p := e.doc.Edits.(*modesWorkspace)
	if !clicked || p.fields[1].TextValue != "Accepted by click" || p.fields[3].Value != 2 {
		t.Fatal("mission switch discarded the focused field", p.fields)
	}
}

func TestModWindowMissionRowsHaveNoDropdownInBothLanguages(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, size := range []image.Point{{1024, 768}, {640, 480}} {
			t.Run(fmt.Sprintf("%s/%dx%d", lang, size.X, size.Y), func(t *testing.T) {
				e := modesWordsEditor(t, "workspace", lang, size)
				host := e.doc.Edits.(*modesWorkspace)
				host.overview, host.graph = modesOverview, modesGraph()
				if err := e.Key("package"); err != nil {
					t.Fatal(err)
				}
				for tab, page := range e.modPages() {
					e.turnModPage(tab)
					s, _, _ := e.modSpec()
					l := layoutPopup(s, e.font, e.bodyFont())
					rows := 0
					for _, rr := range l.Rows {
						if len(e.obj.rowActs[rr.Index]) == 0 {
							continue
						}
						rows++
						row := s.Rows[rr.Index]
						if !rr.Control.Empty() || row.Label == "" || len(rr.Buttons) != 1 || len(row.Buttons) != 1 || row.Buttons[0] != e.word("editor.mod.open") {
							t.Errorf("page %d mission %q: control %v, buttons %v / %q", page, row.Label, rr.Control, rr.Buttons, row.Buttons)
						}
					}
					want := 0
					if page == modPageFields {
						want = len(host.fields[3].Choices)
					}
					if rows != want {
						t.Errorf("page %d: laid out %d mission rows, want %d", page, rows, want)
					}
				}
			})
		}
	}
}

func TestModWindowUsesSharedRowsAndFocusedFieldsBlockCanvasInBothLanguages(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e, calls := newModesEditor(t, "workspace")
		e.SetWords(words.For(lang))
		e.Layout(640, 480)
		if err := e.Key("package"); err != nil {
			t.Fatal(err)
		}
		s, _, ok := e.objectSpec()
		l := layoutPopup(s, e.font, e.bodyFont())
		if !ok || s.Columns != 2 || !l.Rect.In(image.Rect(0, e.barHeight(), 640, 480)) || len(l.Buttons) != len(modButtons) {
			t.Fatalf("%s mod window layout: %v", lang, l.Rect)
		}
		if !e.FocusModField("folder") {
			t.Fatal("missing folder field")
		}
		before := len(calls.lines)
		for _, key := range []string{"unit", "clone", "delete", "drop", "field", "new"} {
			_ = e.Key(key)
		}
		modesClick(e, image.Pt(400, 400))
		if len(calls.lines) != before {
			t.Fatal("mod window leaked a canvas command", calls.lines)
		}
		if !e.FocusModField("after") {
			t.Fatal("missing shared choice field")
		}
		_ = e.Key("enter")
		e.TextInput("cross")
		_ = e.Key("enter")
		if e.obj.fields[e.obj.focus].Value != 25 {
			t.Fatal("named choice did not commit", calls.lines)
		}
		if !e.FocusModField("title") {
			t.Fatal("missing title")
		}
		_ = e.Key("clear")
		e.TextInput("Changed")
		_ = e.Key("enter")
		if !e.ModWindowOpen() {
			t.Fatal("accepting a value closed W")
		}
		_ = e.Key("escape")
		if e.ModWindowOpen() {
			t.Fatal("Escape did not close W")
		}
	}
}

func TestPopupColumnScrollingKeepsTheOtherColumnFixed(t *testing.T) {
	s := popupSpec{Columns: 2, Width: 620, MaxRows: 5, Top: 7, ColumnTops: []int{0, 7}}
	for col := 0; col < 2; col++ {
		for n := 0; n < 14; n++ {
			s.Rows = append(s.Rows, popupRow{Kind: popupText, Column: col, Label: fmt.Sprint(col, n)})
		}
	}
	l := layoutPopup(s, nil, nil)
	if len(l.Rows) != 10 {
		t.Fatal("independent columns", len(l.Rows))
	}
	if l.Rows[0].Index != 0 || l.Rows[5].Index != 21 {
		t.Fatal("column scroll moved identity", l.Rows)
	}
}

func TestModWindowNativeKeysReachSaveHistoryAndReverseFocus(t *testing.T) {
	e, calls := newModesEditor(t, "workspace")
	_ = e.Key("package")
	e.FocusModField("title")
	e.TextInput("x")
	before := len(calls.lines)
	updateEditorKeys(t, e, ebiten.KeyControl, ebiten.KeyZ)
	if len(calls.lines) != before {
		t.Fatal("text undo reached project history")
	}
	_ = e.Key("clear")
	e.TextInput("Accepted")
	_ = e.Key("enter")
	updateEditorKeys(t, e, ebiten.KeyControl, ebiten.KeyS)
	if len(calls.lines) == 0 || calls.lines[len(calls.lines)-1] != "Workspace package save \"\"" {
		t.Fatal("Ctrl+S did not save the mod", calls.lines)
	}
	e.FocusModField("title")
	updateEditorKeys(t, e, ebiten.KeyShift, ebiten.KeyTab)
	if e.obj.fields[e.obj.focus].Key != "folder" {
		t.Fatal("Shift+Tab did not move backward")
	}
}

// W opens a mod whose host says so on its overview page: its sections are
// the window's rows, the Missions tab turns to the mod's fields and the
// window's key turns back. With no overview W keeps its fields alone.
func TestModWindowOpensOnTheOverviewAndItsTabsTurnPages(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	e.doc.Edits.(*modesWorkspace).overview = modesOverview
	_ = e.Key("package")
	s, _, ok := e.modSpec()
	if !ok || e.modPage != 1 || len(s.Tabs) != 2 || s.ActiveTab != 1 {
		t.Fatalf("page %d tabs %v active %d", e.modPage, s.Tabs, s.ActiveTab)
	}
	lines := strings.Join(e.PopupLines(), "\n")
	for _, want := range []string{"Kinds the mod holds", "terrain: data/terrain.toml", "1 mod(s) active on rom1-en", "Files (2)"} {
		if !strings.Contains(lines, want) {
			t.Fatalf("overview lacks %q:\n%s", want, lines)
		}
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	modesClick(e, center(l.Tabs[0]))
	if e.modPage != 0 || !e.modWindow {
		t.Fatalf("the Missions tab left page %d", e.modPage)
	}
	if !strings.Contains(strings.Join(e.PopupLines(), "\n"), "River") {
		t.Fatal("the fields page lacks the mod's name")
	}
	_ = e.Key("package")
	if e.modPage != 1 || !e.modWindow {
		t.Fatalf("the window's key left page %d, open %v", e.modPage, e.modWindow)
	}
	plain, _ := newModesEditor(t, "workspace")
	_ = plain.Key("package")
	if s, _, _ := plain.modSpec(); plain.modPage != 0 || len(s.Tabs) != 0 {
		t.Fatalf("without an overview: page %d tabs %v", plain.modPage, s.Tabs)
	}
}

// noMapHost is a mod host that changes nothing with no map open.
type noMapHost struct{ *modesWorkspace }

func (noMapHost) SetWorkspaceField(scope, key, value string) (*InspectionDocument, error) {
	return nil, errors.New("open a map first")
}

// With no map open W opens the mod window over the map list through the
// mod host, on the mod's overview; its tabs and buttons answer, and what
// edits a map's own data keeps its message.
func TestModWindowOpensWithNoMapOpen(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	host := e.doc.Edits.(*modesWorkspace)
	host.overview = modesOverview
	e.doc, e.catalog = nil, true
	e.SetModHost(func() ModWindowHost { return noMapHost{host} })
	if err := e.Key("pages"); err != nil || e.modWindow || e.message == "" {
		t.Fatalf("pages with no map: %v window %v message %q", err, e.modWindow, e.message)
	}
	if err := e.Key("package"); err != nil || !e.modWindow {
		t.Fatalf("W with no map: %v window %v", err, e.modWindow)
	}
	if field, _, _ := e.FieldFocus(); field != -1 || e.obj.focus != -1 {
		t.Fatalf("W with no map: field %d, control %d", field, e.obj.focus)
	}
	s, _, ok := e.modSpec()
	if !ok || e.modPage != 1 || len(s.Tabs) != 2 {
		t.Fatalf("page %d tabs %v", e.modPage, s.Tabs)
	}
	if lines := strings.Join(e.PopupLines(), "\n"); !strings.Contains(lines, "terrain: data/terrain.toml") {
		t.Fatalf("overview lacks its lines:\n%s", lines)
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	_ = e.ChromeImage()
	for _, p := range []image.Point{center(l.Tabs[0]), {X: 2, Y: 2}, {X: 900, Y: 500}} {
		e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
		_ = e.ChromeImage()
	}
	modesClick(e, center(l.Tabs[0]))
	if e.modPage != 0 || !e.modWindow {
		t.Fatalf("the Missions tab left page %d", e.modPage)
	}
	for _, key := range []string{"tab", "backtab", "package", "package"} {
		if err := e.Key(key); err != nil {
			t.Fatalf("%s: %v", key, err)
		}
	}
	s, _, _ = e.modSpec()
	l = layoutPopup(s, e.font, e.bodyFont())
	for n := range modButtonIndex("close") {
		modesClick(e, center(l.Buttons[n]))
	}
	if !e.modWindow || e.doc != nil {
		t.Fatalf("a button left window %v map %v", e.modWindow, e.doc != nil)
	}
	modesClick(e, center(l.Buttons[modButtonIndex("close")]))
	if e.modWindow || !e.catalog {
		t.Fatalf("Close left window %v catalog %v", e.modWindow, e.catalog)
	}
}

func (p *modesWorkspace) ModMapActions() ([]ModMapAction, bool) { return p.actions, p.actionsOpen }
