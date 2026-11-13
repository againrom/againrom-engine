package game

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

// f5Inspector is the installed inspector with a recording test-play host.
func f5Inspector(t *testing.T) (*MapInspector, *[][]string) {
	t.Helper()
	x := installedInspector(t)
	game := filepath.Join(t.TempDir(), "againrom.exe")
	if err := os.WriteFile(game, []byte("program"), 0o644); err != nil {
		t.Fatal(err)
	}
	profile := t.TempDir()
	started := &[][]string{}
	x.SetTestPlay(TestPlayHost{
		Profile: func() (RuntimeProfile, error) {
			return ResolveRuntimeProfile(filepath.Join(profile, "saves"), "", "", x.archives.Root)
		},
		Game: game,
		Start: func(program string, args []string) error {
			*started = append(*started, append([]string{program}, args...))
			return nil
		},
	})
	return x, started
}

// TestReleaseTestPlayOfANewMapStartsAtOnce presses F5 on a map made with the
// new-map dialog's defaults, and again after one unit is placed: the window
// that asks who plays the map lists no warning, no notice opens and Start
// starts the game once each time, while the Map panel still says that nothing
// wins the map.
func TestReleaseTestPlayOfANewMapStartsAtOnce(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x, started := f5Inspector(t)
	lang := installLanguage(x)
	dir := os.Getenv("AGAINROM_EDITOR_FRAMES")
	c := &editorClicker{t: t, now: time.Unix(1, 0)}
	e := x.Editor()
	e.Layout(1280, 800)
	_ = e.Key("new")
	_ = e.Key("enter")
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 0})
	c.click(e, r, ok, "the mode button")
	endless := strings.Join(strings.Fields(x.inspectionUTF8(x.word("editor.panel.victory")+" "+x.word("editor.panel.victory.none"))), " ")
	check := func(when string) {
		t.Helper()
		_ = e.Key("testplay")
		if !e.TestPartyOpen() || len(e.PromptNotes()) != 0 || len(*started) != 0 {
			t.Fatalf("%s: party window %v, notes %q, launches %d", when, e.TestPartyOpen(), e.PromptNotes(), len(*started))
		}
		_ = e.Key("enter")
		if e.NoticeOpen() || e.TestPartyOpen() || len(*started) != 1 || !strings.HasPrefix(e.Message(), "Test play: ") {
			t.Fatalf("%s: notice %v, launches %d, message %q", when, e.NoticeOpen(), len(*started), e.Message())
		}
		if rows, open := e.ReportOpen(); open || len(rows) > 0 {
			t.Fatalf("%s: a report opened: %v", when, rows)
		}
		if !slices.Contains(e.PanelLines(), endless) {
			t.Fatalf("%s: the panel no longer says the map is endless: %q", when, e.PanelLines())
		}
		writeEditorFrame(t, e, filepath.Join(dir, "f5-newmap-"+when+"-"+lang+".png"), dir != "", panelArea)
		*started = nil
	}
	check("defaults")
	in := addInputsOf(t, x)
	s, isEdits := e.Document().Edits.(*mapInspectionEdits)
	if !isEdits {
		t.Fatal("the new map has no inspection edits")
	}
	if _, err := s.AddUnitIdentity(image.Pt(30, 30), in.identity); err != nil {
		t.Fatal(err)
	}
	check("unit")
}

// TestReleaseTestPlayPartyWindowFrames opens the window that asks who plays
// the map on an installed map: it offers the hero and the campaign's five
// companions under the install's names, each companion with a portrait, and
// the frames of the first choice and of a mixed one go to
// AGAINROM_EDITOR_FRAMES.
func TestReleaseTestPlayPartyWindowFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x, started := f5Inspector(t)
	lang := installLanguage(x)
	dir := os.Getenv("AGAINROM_EDITOR_FRAMES")
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/50.alm"); err != nil {
		t.Fatal(err)
	}
	cards := e.Document().Edits.(*mapInspectionEdits).TestPlayCards(0)
	if len(cards) != 6 || !cards[0].Hero {
		t.Fatalf("the window offers %d cards", len(cards))
	}
	for i, c := range cards[1:] {
		if c.Name == "" || c.Portrait == nil {
			t.Errorf("companion %d: name %q, portrait %v", i+1, c.Name, c.Portrait != nil)
		}
		t.Logf("card %d: %s", i+1, c.Name)
	}
	shot := func(name string) {
		t.Helper()
		cover, _ := e.PromptRect()
		writeEditorFrame(t, e, filepath.Join(dir, name+"-"+lang+".png"), dir != "", cover)
	}
	_ = e.Key("testplay")
	if !e.TestPartyOpen() {
		t.Fatal("F5 did not ask who plays the map")
	}
	shot("f5-party-first")
	for _, press := range [][2]int{{0, 1}, {2, 1}, {0, 3}, {1, 3}, {2, 3}, {0, 5}, {1, 0}} {
		partyPress(t, e, press[0], press[1])
	}
	shot("f5-party-chosen")
	if _ = e.Key("enter"); e.TestPartyOpen() || len(*started) != 1 {
		t.Fatalf("Start: window %v, launches %d, message %q", e.TestPartyOpen(), len(*started), e.Message())
	}
}

// panelArea is where the Map panel stands on a 1280x800 window; the glyph
// check leaves it out, since the Russian panel keeps three hidden glyphs after
// a test play on shipped maps as well.
var panelArea = image.Rect(0, 0, 460, 420)

// TestReleaseTestPlayOfAMapWithAnotherWarningStillAsks presses F5 on an
// installed map whose start point the checks find wanting: the window that
// asks who plays the map lists the warning, Cancel starts nothing and Start
// starts the game once.
func TestReleaseTestPlayOfAMapWithAnotherWarningStillAsks(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x, started := f5Inspector(t)
	c := &editorClicker{t: t, now: time.Unix(1, 0)}
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/50.alm"); err != nil {
		t.Fatal(err)
	}
	_ = e.Key("testplay")
	start := strings.Fields(x.inspectionUTF8(x.word("editor.panel.start")))[0]
	if !e.TestPartyOpen() || !strings.Contains(noticeWords(e.PromptNotes()), start) || len(*started) != 0 {
		t.Fatalf("party window %v, notes %q, launches %d", e.TestPartyOpen(), e.PromptNotes(), len(*started))
	}
	_ = e.Key("escape")
	if e.TestPartyOpen() || len(*started) != 0 {
		t.Fatal("Cancel left the window open or started the game")
	}
	_ = e.Key("testplay")
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 0})
	c.click(e, r, ok, "Start")
	if e.TestPartyOpen() || len(*started) != 1 {
		t.Fatalf("Start: window %v, launches %d, message %q", e.TestPartyOpen(), len(*started), e.Message())
	}
}

// noticeWords joins note lines into one string of single-spaced words, so a line
// the popup wrapped compares with the line it came from.
func noticeWords(lines []string) string {
	return strings.Join(strings.Fields(strings.Join(lines, " ")), " ")
}
