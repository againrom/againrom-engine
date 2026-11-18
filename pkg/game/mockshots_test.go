package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/internal/witness"
	"againrom/pkg/ui"
)

// mockShot is one row of the mockup screenshot table: the task page it
// belongs to, a name for the row, and a func that opens the window on an
// editor and calls frame for each picture it shows.
type mockShot struct {
	task, name string
	open       func(t *testing.T, s *mockShots)
}

// mockShots is what a row sees: the install's inspector and language, and the
// out dir the pictures go to.
type mockShots struct {
	x         *MapInspector
	out, lang string
	task      string
}

// frame draws the editor and writes its headless frame, cropped to the front
// window, as <out>/<task>/main/<name>-<lang>-1920x1080.png.
func (s *mockShots) frame(t *testing.T, e *ui.MapEditor, name string) {
	t.Helper()
	screen := ebiten.NewImage(1920, 1080)
	e.Draw(screen)
	screen.Dispose()
	pic, _, err := e.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	var shown image.Image = pic
	if r, ok := e.PopupRect(); ok {
		r = r.Inset(-12).Intersect(pic.Bounds())
		shown = pic.SubImage(r)
	}
	dir := filepath.Join(s.out, s.task, "main")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	witness.PNG(t, filepath.Join(dir, fmt.Sprintf("%s-%s-1920x1080.png", name, s.lang)), shown)
}

// editor opens a map in a fresh editor at 1920x1080 with the Map panel closed.
func (s *mockShots) editor(t *testing.T, mapName string) *ui.MapEditor {
	t.Helper()
	return mockShotEditor(t, s.x, mapName)
}

func mockShotEditor(t *testing.T, x *MapInspector, mapName string) *ui.MapEditor {
	t.Helper()
	e := x.Editor()
	e.SetPanelProfile([]byte(`{"right":{"minimized":true,"width":520,"view":"plain"}}`), nil, nil)
	e.Layout(1920, 1080)
	if err := e.Open(mapName); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	return e
}

func mockClicker(t *testing.T) *editorClicker {
	return &editorClicker{t: t, now: time.Unix(10, 0)}
}

func mockBarOpen(t *testing.T, e *ui.MapEditor, key string) {
	t.Helper()
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlBar, Key: key})
	mockClicker(t).click(e, r, ok, "the "+key+" entry")
}

func mockOpenGroup(t *testing.T, e *ui.MapEditor, player, group int) {
	t.Helper()
	showTree(e)
	mockBarOpen(t, e, "groups")
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlGroupsRow, N: player, K: group})
	mockClicker(t).click(e, r, ok, "the group row")
}

func mockOpenDialogFromAction(t *testing.T, e *ui.MapEditor, page int) {
	t.Helper()
	want := fmt.Sprintf("%d:", page)
	for i, r := range e.Document().Records {
		if string(r.Kind) != "Action" {
			continue
		}
		e.Select(i, false)
		for _, l := range e.PopupLines() {
			f := strings.Fields(l)
			if len(f) < 3 || f[1] != want {
				continue
			}
			for j, ln := range e.PopupLines() {
				// The Russian words are in the host's encoding.
				if strings.HasPrefix(ln, "Open Dialog") || strings.HasPrefix(ln, "\x8e\xe2\xaa\xe0\xeb\xe2\xec \x84\xa8\xa0\xab\xae\xa3") {
					rr, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupRow, N: j})
					mockClicker(t).click(e, rr, ok, "Open Dialog")
					return
				}
			}
		}
	}
	t.Fatalf("no action starts dialog %d", page)
}

func mockGroupUnits(e *ui.MapEditor, player, group int) []int {
	var units []int
	for _, id := range groupMembersOf(e.Document(), player, group) {
		units = append(units, e.Document().Records[id].Index)
	}
	return units
}

func mockToGroup(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlSelectionButton, N: 0})
	mockClicker(t).click(e, r, ok, "To group...")
}

// mockShotTable is every named shot of every task. A new window is one row.
var mockShotTable = []mockShot{
	{"t1", "structure", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/150.alm")
		e.Select(218, true)
		s.frame(t, e, "structure")
	}},
	{"t1", "sack", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/10.alm")
		e.Select(53, true)
		s.frame(t, e, "sack")
	}},
	{"t1", "record", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/10.alm")
		e.Select(58, true)
		s.frame(t, e, "record")
	}},
	{"t1", "group", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/40.alm")
		mockOpenGroup(t, e, 3, 1)
		s.frame(t, e, "group")
	}},
	{"t1", "usedby", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/10.alm")
		mockOpenGroup(t, e, 2, 2)
		s.frame(t, e, "usedby")
	}},
	{"t1", "objectnav", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/150.alm")
		e.Select(2637, false)
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupRow, N: 5})
		mockClicker(t).click(e, r, ok, "the Open Structure row")
		s.frame(t, e, "objectnav")
	}},

	{"t2", "palette", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/90.alm")
		mockBarOpen(t, e, "units")
		s.frame(t, e, "palette")
	}},
	{"t2", "players", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/90.alm")
		mockBarOpen(t, e, "players")
		s.frame(t, e, "players")
	}},
	{"t2", "player", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/90.alm")
		e.Select(recordOf(e.Document(), "Player", 1), false)
		s.frame(t, e, "player")
	}},
	{"t2", "playerdelete", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/90.alm")
		e.Select(recordOf(e.Document(), "Player", 1), false)
		if err := e.Key("delete"); err != nil {
			t.Fatal(err)
		}
		s.frame(t, e, "playerdelete")
	}},

	{"t4", "message-dialog", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/151.alm")
		mockOpenDialogFromAction(t, e, 4)
		s.frame(t, e, "message-dialog")
	}},
	{"t4", "message-list", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/70.alm")
		ui.ProbeOpenMessage(e, 2)
		s.frame(t, e, "message-list")
	}},
	{"t4", "message-class", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/20.alm")
		mockOpenDialogFromAction(t, e, 3)
		s.frame(t, e, "message-class")
	}},
	{"t4", "message-never", func(t *testing.T, s *mockShots) {
		m := "scenario/101.alm"
		if s.lang == "en" {
			m = "scenario/81.alm"
		}
		e := s.editor(t, m)
		mockOpenDialogFromAction(t, e, 6)
		s.frame(t, e, "message-never")
	}},
	{"t4", "message-new", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/151.alm")
		key := ""
	find:
		for _, tr := range e.Document().Script.Triggers {
			e.Select(tr.Record, false)
			for action := range 4 {
				if e.WizardField(fmt.Sprintf("a%d.p0", action)) < 0 {
					continue
				}
				for _, n := range e.Document().Script.Actions {
					if n.ID == tr.Acts[action] && n.Opcode == 2 && n.Types[0] == 1 {
						key = fmt.Sprintf("a%d.item", action)
						break find
					}
				}
			}
		}
		if key == "" {
			t.Fatal("no trigger shows a dialog action")
		}
		configure(t, e, key)
		messageField(t, e, e.DialogChoiceField())
		messageChoiceClick(t, e, s.x.inspectionUTF8(s.x.word("editor.message.newdialog")))
		s.frame(t, e, "message-new")
	}},
	{"t4", "variable", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/151.alm")
		e.Select(-2-2, false)
		s.frame(t, e, "variable")
	}},
	{"t4", "variable-shared", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/60.alm")
		e.Select(-2-0, false)
		s.frame(t, e, "variable-shared")
	}},

	{"t13", "newmode", func(t *testing.T, s *mockShots) {
		e := s.x.Editor()
		e.Layout(1920, 1080)
		_ = e.Key("new")
		_ = e.Key("enter")
		s.frame(t, e, "newmode")
	}},
	{"t13", "leave", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/81.alm")
		e.Select(0, false)
		_ = e.Key("delete")
		_ = e.Key("close")
		s.frame(t, e, "leave")
	}},
	{"t13", "notice", func(t *testing.T, s *mockShots) {
		fx, _ := f5Inspector(t)
		e := mockShotEditor(t, fx, "scenario/50.alm")
		_ = e.Key("testplay")
		if e.NoticeOpen() {
			s.frame(t, e, "notice")
			_ = e.Key("enter")
		}
		if !e.TestPartyOpen() {
			t.Fatalf("no party window; windows %q, message %q", e.OpenWindows(), e.Message())
		}
		s.frame(t, e, "testparty")
	}},
	{"t13", "areatrigger", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/81.alm")
		areaDraw(t, e, image.Pt(28, 23), image.Pt(32, 27))
		s.frame(t, e, "areatrigger")
	}},
	{"t13", "areashared", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/81.alm")
		for _, tn := range []int{4, 11, 12, 13} {
			areaSelect(t, e, tn)
			for _, a := range e.TriggerAreas(tn) {
				areaPress(e, e.AreaCornerPoint(a.Rect.Max), e.CellPoint(a.Rect.Max.Add(image.Pt(2, 0))), true)
				if e.PromptTitle() != "" {
					s.frame(t, e, "areashared")
					return
				}
				_ = e.Key("escape")
			}
		}
		t.Fatal("no shared area asked")
	}},
	{"t13", "togroup", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/81.alm")
		var u []int
		for g := 1; g < 30 && len(u) < 3; g++ {
			u = mockGroupUnits(e, 4, g)
		}
		if len(u) < 3 {
			t.Fatal("player 4 has no group of three")
		}
		selectUnits(t, e, u[:2]...)
		e.Layout(1920, 1080)
		mockToGroup(t, e)
		s.frame(t, e, "togroup")
	}},
	{"t13", "groupwarn", func(t *testing.T, s *mockShots) {
		for player := 1; player <= 9; player++ {
			for group := 1; group < 40; group++ {
				e := s.editor(t, "scenario/81.alm")
				u := mockGroupUnits(e, player, group)
				if len(u) < 2 || len(u) > 40 || !trySelectUnits(t, e, u...) {
					continue
				}
				e.Layout(1920, 1080)
				c := mockClicker(t)
				r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlSelectionButton, N: 0})
				if !ok {
					continue
				}
				c.click(e, r, ok, "To group...")
				r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 0})
				if !ok {
					continue
				}
				c.click(e, r, ok, "OK")
				if len(e.PromptNotes()) > 0 && slices.Contains(e.OpenWindows(), "prompt") {
					s.frame(t, e, "groupwarn")
					return
				}
			}
		}
		t.Skip("no group warning found")
	}},
	{"t13", "scriptform", func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/81.alm")
		e.Select(trigRecord(t, e, "Trigger", 2), true)
		if !e.WizardShown() {
			t.Fatal("no wizard")
		}
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: -1, K: 1, Flag: false})
		trigClick(t, e, r, ok, "Add OR group")
		trigChoose(t, e, wizField(t, e, "c3.item"), s.x.inspectionUTF8(s.x.word("editor.wizard.c.always")))
		r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 0})
		trigClick(t, e, r, ok, "Apply")
		s.frame(t, e, "scriptform")
	}},

	mockT12("structures"), mockT12("sacks"), mockT12("objects"),
	mockT12("trees-forest"), mockT12("trees-one"),
	mockT12("height"), mockT12("terrain"), mockT12("bridge"),
}

// mockT12 is the row of one task-12 palette or mode window on map 10.
func mockT12(name string) mockShot {
	return mockShot{"t12", name, func(t *testing.T, s *mockShots) {
		e := s.editor(t, "scenario/10.alm")
		switch name {
		case "structures", "sacks", "objects":
			mockBarOpen(t, e, name)
		case "trees-forest", "trees-one":
			mockBarOpen(t, e, "trees")
			if name == "trees-one" {
				r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlListButton, N: 0, K: 0})
				mockClicker(t).click(e, r, ok, "the One kind button")
			}
		default:
			if err := e.Key(name); err != nil {
				t.Fatal(err)
			}
		}
		s.frame(t, e, name)
	}}
}

// TestMockShots writes every row of the table for the install AGAINROM_ASSETS
// names, in that install's language, under the directory AGAINROM_MOCKSHOTS
// names. scripts/mockshots.sh runs it once per install.
func TestMockShots(t *testing.T) {
	out := os.Getenv("AGAINROM_MOCKSHOTS")
	if out == "" {
		t.Skip("AGAINROM_MOCKSHOTS is not set")
	}
	x := installedInspector(t)
	lang := installLanguage(x)
	for _, row := range mockShotTable {
		t.Run(row.task+"/"+row.name, func(t *testing.T) {
			row.open(t, &mockShots{x: x, out: out, lang: lang, task: row.task})
		})
	}
}
