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
// new-map dialog's defaults, and again after one unit is placed: no notice
// opens and the game starts once each time, while the Map panel still says
// that nothing wins the map.
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
	r, ok := e.PromptButtonRect(0)
	c.click(e, r, ok, "the mode button")
	endless := strings.Join(strings.Fields(x.inspectionUTF8(x.word("editor.panel.victory")+" "+x.word("editor.panel.victory.none"))), " ")
	check := func(when string) {
		t.Helper()
		_ = e.Key("testplay")
		if e.NoticeOpen() || len(*started) != 1 || !strings.HasPrefix(e.Message(), "Test play: ") {
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
	if _, err := s.AddUnitTier(image.Pt(30, 30), int64(in.class), int64(in.tier)); err != nil {
		t.Fatal(err)
	}
	check("unit")
}

// panelArea is where the Map panel stands on a 1280x800 window; the glyph
// check leaves it out, since the Russian panel keeps three hidden glyphs after
// a test play on shipped maps as well.
var panelArea = image.Rect(0, 0, 460, 420)

// TestReleaseTestPlayOfAMapWithAnotherWarningStillAsks presses F5 on an
// installed map whose start point the checks find wanting: the notice opens
// and lists the warning, Cancel starts nothing and Continue starts the game
// once.
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
	if !e.NoticeOpen() || !strings.Contains(noticeWords(e.PromptNotes()), start) || len(*started) != 0 {
		t.Fatalf("notice %v, notes %q, launches %d", e.NoticeOpen(), e.PromptNotes(), len(*started))
	}
	_ = e.Key("escape")
	if e.NoticeOpen() || len(*started) != 0 {
		t.Fatal("Cancel left the notice open or started the game")
	}
	_ = e.Key("testplay")
	r, ok := e.PromptButtonRect(0)
	c.click(e, r, ok, "Continue")
	if e.NoticeOpen() || len(*started) != 1 {
		t.Fatalf("Continue: notice %v, launches %d, message %q", e.NoticeOpen(), len(*started), e.Message())
	}
}

// noticeWords joins note lines into one string of single-spaced words, so a line
// the popup wrapped compares with the line it came from.
func noticeWords(lines []string) string {
	return strings.Join(strings.Fields(strings.Join(lines, " ")), " ")
}
