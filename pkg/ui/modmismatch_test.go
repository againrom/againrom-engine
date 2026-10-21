package ui

import (
	"errors"
	"fmt"
	"image"
	"reflect"
	"strings"
	"testing"
)

// fixtureMismatch is a refusal over the mods as a LOAD seam returns it.
type fixtureMismatch struct{ m ModMismatch }

func (e fixtureMismatch) Error() string            { return "mod set mismatch: fixture prose" }
func (e fixtureMismatch) ModMismatch() ModMismatch { return e.m }

// everyMismatchKind is one row of each kind.
var everyMismatchKind = ModMismatch{Reason: ModMismatchSet, Rows: []ModMismatchRow{
	{Kind: ModMismatchBase, Saved: "rom1-en", Current: "rom1-ru"},
	{Kind: ModMismatchMissing, Mod: "alpha", Saved: "1.0"},
	{Kind: ModMismatchAdded, Mod: "beta", Current: "2.0"},
	{Kind: ModMismatchVersion, Mod: "gamma", Saved: "1.0", Current: "1.1"},
	{Kind: ModMismatchFiles, Mod: "delta", Saved: "3", Current: "3"},
	{Kind: ModMismatchSettingGone, Mod: "eps", Setting: "cap", Saved: "120"},
	{Kind: ModMismatchSetting, Mod: "eps", Setting: "speed", Saved: "2", Current: "3"},
	{Kind: ModMismatchSettingNew, Mod: "eps", Setting: "loud", Current: "true"},
	{Kind: ModMismatchOrder, Saved: "a, b", Current: "b, a"},
}}

var everyMismatchRow = []string{
	"Base game: saved [rom1-en], now [rom1-ru]",
	"Mod alpha 1.0: [not active]",
	"Mod beta 2.0: [active, the save did not use it]",
	"Mod gamma: saved [1.0], now [1.1]",
	"Mod delta 3: [its files differ from the ones the save was made with]",
	"Mod eps, setting cap = 120: [not declared now]",
	"Mod eps, setting speed: saved [2], now [3]",
	"Mod eps, setting loud = true: [not recorded in the save]",
	"Load order: saved [a, b], now [b, a]",
}

// mismatchApp is a Load page whose one save refuses with err, wrapped as a
// caller wraps a refusal.
func mismatchApp(t *testing.T, err error) *App {
	t.Helper()
	a := newTestApp(t, nil, nil)
	a.SetWords(AuthoredWords(), solidFont15(), nil)
	a.SetSaveSeams(nil, func() []SaveEntry {
		return []SaveEntry{{Name: "a.sav", Label: "First"}, {Name: "b.sav", Label: "Second"}}
	}, func(name string) (MapOpener, bool, error) {
		return nil, false, fmt.Errorf("load %s: %w", name, err)
	})
	a.SetCutsceneScrollArt(loadScrollTestFrames())
	if err := a.HeadlessKey("load"); err != nil || a.Screen() != ScreenLoad {
		t.Fatal("Load page", err, a.Screen())
	}
	return a
}

func mismatchKey(t *testing.T, a *App, key string) {
	t.Helper()
	if err := a.HeadlessKey(key); err != nil {
		t.Fatal(err)
	}
}

// TestALoadRefusedOverTheModsOpensTheWindowNotTheMessageLine: a LOAD the mods
// refuse opens the window with one row per difference, the mismatched parts
// marked, and leaves the message line empty. The one-line message path fails
// here.
func TestALoadRefusedOverTheModsOpensTheWindowNotTheMessageLine(t *testing.T) {
	a := mismatchApp(t, fixtureMismatch{everyMismatchKind})
	mismatchKey(t, a, "down")
	before := loadScrollTestFrame(t, a)
	mismatchKey(t, a, "enter")
	got, open := a.HeadlessModMismatch()
	if !open {
		t.Fatalf("no window; the message line says %q", a.HeadlessMessage())
	}
	if a.HeadlessMessage() != "" {
		t.Fatalf("the refusal also went to the message line: %q", a.HeadlessMessage())
	}
	if !reflect.DeepEqual(got.Rows, everyMismatchRow) {
		t.Fatalf("rows\n%q\nwant\n%q", got.Rows, everyMismatchRow)
	}
	if got.Title != "The save does not match the active mods" || !strings.Contains(got.Hint, "mods and settings it was made with") {
		t.Fatalf("title %q hint %q", got.Title, got.Hint)
	}
	// Keys while the window is open do not reach the list.
	mismatchKey(t, a, "up")
	if a.flow.loadList.Selection() != 1 {
		t.Fatal("a key moved the list under the window")
	}
	mismatchKey(t, a, "escape")
	if _, open := a.HeadlessModMismatch(); open || a.Screen() != ScreenLoad {
		t.Fatalf("Escape left window %v on %s", open, a.Screen())
	}
	if after := loadScrollTestFrame(t, a); !reflect.DeepEqual(before.Pix, after.Pix) {
		t.Fatal("the Load page is not as it was after the window closed")
	}
	mismatchKey(t, a, "enter")
	mismatchKey(t, a, "enter")
	if _, open := a.HeadlessModMismatch(); open {
		t.Fatal("Enter did not close the window")
	}
	mismatchKey(t, a, "enter")
	ok := newNoticeWindowLayout(a.flow.menuFont, loadPanel).OK.Min.Add(image.Pt(4, 4))
	for _, action := range []string{"press", "release"} {
		if err := a.HeadlessPointer(action, ok.X, ok.Y); err != nil {
			t.Fatal(err)
		}
	}
	if _, open := a.HeadlessModMismatch(); open || a.Screen() != ScreenLoad {
		t.Fatal("OK did not close the window back to the Load page")
	}
}

// TestTheWindowNamesTheRefusalsWithoutADifferenceList: a save without a
// record lists the active mods under a lead row and names the unmarked flag;
// an unreadable record is its own row.
func TestTheWindowNamesTheRefusalsWithoutADifferenceList(t *testing.T) {
	unmarked := ModMismatch{Reason: ModMismatchUnmarked, Rows: []ModMismatchRow{{Kind: ModMismatchAdded, Mod: "alpha", Current: "1.0"}}}
	a := mismatchApp(t, fixtureMismatch{unmarked})
	mismatchKey(t, a, "enter")
	got, _ := a.HeadlessModMismatch()
	if want := []string{"[The save records no mods,] but these mods are active:", "Mod alpha 1.0: [active, the save did not use it]"}; !reflect.DeepEqual(got.Rows, want) {
		t.Fatalf("unmarked rows %q", got.Rows)
	}
	if !strings.Contains(got.Hint, "-mods-accept-unmarked") {
		t.Fatalf("unmarked hint %q", got.Hint)
	}
	b := mismatchApp(t, fixtureMismatch{ModMismatch{Reason: ModMismatchMark}})
	mismatchKey(t, b, "enter")
	got, _ = b.HeadlessModMismatch()
	if len(got.Rows) != 1 || !strings.HasPrefix(got.Rows[0], "[The save's record of its mods") || strings.Contains(got.Hint, "accept-unmarked") {
		t.Fatalf("unreadable mark %q / %q", got.Rows, got.Hint)
	}
	// Any other refusal keeps the message line.
	c := mismatchApp(t, errors.New("disk on fire"))
	mismatchKey(t, c, "enter")
	if _, open := c.HeadlessModMismatch(); open || !strings.Contains(c.HeadlessMessage(), "disk on fire") {
		t.Fatalf("an ordinary refusal: window %v, message %q", open, c.HeadlessMessage())
	}
}

// TestTheWindowDrawsTheMismatchedPartsInTheAlertInk: alert runs reach the
// frame in the alert ink and plain runs in the page's ink, inside the well.
func TestTheWindowDrawsTheMismatchedPartsInTheAlertInk(t *testing.T) {
	a := mismatchApp(t, fixtureMismatch{everyMismatchKind})
	mismatchKey(t, a, "enter")
	pix := loadScrollTestFrame(t, a)
	l := newNoticeWindowLayout(a.flow.menuFont, loadPanel)
	count := func(r image.Rectangle, c [4]uint8) int {
		n := 0
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				p := pix.RGBAAt(x, y)
				if [4]uint8{p.R, p.G, p.B, p.A} == c {
					n++
				}
			}
		}
		return n
	}
	alert := [4]uint8{noticeAlertInk.R, noticeAlertInk.G, noticeAlertInk.B, noticeAlertInk.A}
	plain := [4]uint8{listRowInk.R, listRowInk.G, listRowInk.B, listRowInk.A}
	// Each visible line holds the inks of its runs.
	lines := a.flow.loadUI.mismatch.noticeLines(a.flow.menuFont, l, a.modMismatchDisplay)
	for i := 0; i < min(l.List.Rows, len(lines)); i++ {
		var hasAlert, hasPlain bool
		for _, r := range lines[i].runs {
			hasAlert, hasPlain = hasAlert || r.Alert, hasPlain || !r.Alert
		}
		row := l.List.Row(i)
		if (count(row, alert) > 0) != hasAlert || (count(row, plain) > 0) != hasPlain {
			t.Errorf("line %d: alert %d plain %d pixels, runs %+v", i, count(row, alert), count(row, plain), lines[i].runs)
		}
	}
	if n := count(image.Rect(0, 0, 640, 480), alert) - count(l.List.Rect, alert); n != 0 {
		t.Errorf("%d alert pixels outside the list", n)
	}
}

// TestALongListScrollsInsideTheWindow: rows beyond the well scroll by keys,
// the wheel and the bar, and nothing is drawn outside the window.
func TestALongListScrollsInsideTheWindow(t *testing.T) {
	long := ModMismatch{Reason: ModMismatchSet}
	for i := 0; i < 40; i++ {
		long.Rows = append(long.Rows, ModMismatchRow{Kind: ModMismatchVersion, Mod: fmt.Sprintf("mod-%02d-with-a-rather-long-identifier", i), Saved: "1.0.0", Current: "1.0.1"})
	}
	a := mismatchApp(t, fixtureMismatch{long})
	page := loadScrollTestFrame(t, a)
	mismatchKey(t, a, "enter")
	got, _ := a.HeadlessModMismatch()
	rows := got.Visible
	if len(got.Rows) != 40 || got.Lines < 40 || got.Lines <= rows || rows != newNoticeWindowLayout(a.flow.menuFont, loadPanel).List.Rows {
		t.Fatalf("%d rows in %d lines, %d visible", len(got.Rows), got.Lines, rows)
	}
	maxTop := got.Lines - rows
	steps := []struct {
		key string
		top int
	}{{"down", 1}, {"pagedown", 1 + rows}, {"end", maxTop}, {"down", maxTop}, {"pageup", maxTop - rows}, {"home", 0}, {"up", 0}}
	for _, s := range steps {
		mismatchKey(t, a, s.key)
		if got, _ := a.HeadlessModMismatch(); got.Top != s.top {
			t.Fatalf("%s: top %d, want %d", s.key, got.Top, s.top)
		}
	}
	l := newNoticeWindowLayout(a.flow.menuFont, loadPanel)
	bottom := l.List.Bar().Max.Sub(image.Pt(4, 4))
	for _, action := range []string{"press", "release"} {
		if err := a.HeadlessPointer(action, bottom.X, bottom.Y); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := a.HeadlessModMismatch(); got.Top != 1 {
		t.Fatalf("the bar's bottom arrow left top %d", got.Top)
	}
	mismatchKey(t, a, "end")
	pix := loadScrollTestFrame(t, a)
	window := windowFrame(l.Body, a.flow.menuArt).Rect
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			if !image.Pt(x, y).In(window) && pix.RGBAAt(x, y) != page.RGBAAt(x, y) {
				t.Fatalf("pixel %d,%d outside the window %v changed", x, y, window)
			}
		}
	}
	width := l.List.Rect.Dx() - 2*listTextX
	for _, ln := range a.flow.loadUI.mismatch.noticeLines(a.flow.menuFont, l, a.modMismatchDisplay) {
		w := ln.indent
		for _, r := range ln.runs {
			w += a.flow.menuFont.Advance(r.Text)
		}
		if w > width {
			t.Fatalf("a line %d wide runs past the well's %d", w, width)
		}
	}
}

func TestModMismatchRowSplitsTheBracketedRuns(t *testing.T) {
	got := modMismatchRow("Mod {mod}: saved [{saved}], now [{current}]", ModMismatchRow{Mod: "m", Saved: "1", Current: "2"})
	want := noticeRow{{"Mod m: saved ", false}, {"1", true}, {", now ", false}, {"2", true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v", got)
	}
}

// TestClosingTheWindowLeavesTheLoadPagesMessageLineAsItWas: a Load page that
// shows an earlier read error shows the same line and the same frame after
// the mismatch window opened over it and was closed by each of its controls.
func TestClosingTheWindowLeavesTheLoadPagesMessageLineAsItWas(t *testing.T) {
	for _, closer := range []string{"escape", "enter", "ok"} {
		t.Run(closer, func(t *testing.T) {
			a := mismatchApp(t, fixtureMismatch{everyMismatchKind})
			a.flow.loadGame = func(name string) (MapOpener, bool, error) {
				if name == "a.sav" {
					return nil, false, errors.New("cannot read save")
				}
				return nil, false, fixtureMismatch{everyMismatchKind}
			}
			mismatchKey(t, a, "enter")
			mismatchKey(t, a, "down")
			message := a.HeadlessMessage()
			if message == "" {
				t.Fatal("the first save left no message")
			}
			before := loadScrollTestFrame(t, a)
			mismatchKey(t, a, "enter")
			if _, open := a.HeadlessModMismatch(); !open {
				t.Fatal("no window")
			}
			if closer == "ok" {
				p := newNoticeWindowLayout(a.flow.menuFont, loadPanel).OK.Min.Add(image.Pt(4, 4))
				for _, action := range []string{"press", "release"} {
					if err := a.HeadlessPointer(action, p.X, p.Y); err != nil {
						t.Fatal(err)
					}
				}
			} else {
				mismatchKey(t, a, closer)
			}
			if _, open := a.HeadlessModMismatch(); open {
				t.Fatal("the window stayed open")
			}
			if got := a.HeadlessMessage(); got != message {
				t.Fatalf("message was %q, now %q", message, got)
			}
			if after := loadScrollTestFrame(t, a); !reflect.DeepEqual(before.Pix, after.Pix) {
				t.Fatal("the Load page is not as it was")
			}
		})
	}
}

// TestALongVersionWrapsInsideTheRowWithEveryCharacter: in the Load panel's
// own geometry, a 120-character version breaks into lines none wider than the
// row's text width, indent included, and keeps every character.
func TestALongVersionWrapsInsideTheRowWithEveryCharacter(t *testing.T) {
	a := mismatchApp(t, fixtureMismatch{everyMismatchKind})
	font := a.flow.menuFont
	l := newNoticeWindowLayout(font, loadPanel)
	width := l.List.Rect.Dx() - 2*listTextX
	version := strings.Repeat("W", 120)
	row := modMismatchRow(a.flow.modMismatchWord(ModMismatchVersion), ModMismatchRow{Mod: "alpha", Saved: version, Current: "2"})
	var joined strings.Builder
	for i, line := range wrapNoticeRow(font, row, width, a.modMismatchDisplay) {
		used := line.indent
		for _, run := range line.runs {
			used += font.Advance(run.Text)
			joined.WriteString(run.Text)
		}
		if used > width {
			t.Errorf("line %d is %d pixels wide in a %d-pixel row", i, used, width)
		}
	}
	if !strings.Contains(joined.String(), version) {
		t.Errorf("the version is not whole in %q", joined.String())
	}
}
