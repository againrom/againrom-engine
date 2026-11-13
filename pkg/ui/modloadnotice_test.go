package ui

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

// changedModsNotice is what a town LOAD under changed mods answers: one row of
// each kind a load can take away.
var changedModsNotice = &ModLoadNotice{Changes: ModMismatch{Reason: ModMismatchChanged, Rows: []ModMismatchRow{
	{Kind: ModMismatchItemPack, Who: "Hero", Mod: "armory", What: "cloak"},
	{Kind: ModMismatchItemWorn, Who: "Ann", Mod: "armory", What: "helm"},
	{Kind: ModMismatchLayer, Who: "Ann", Mod: "armory", What: "veil"},
	{Kind: ModMismatchSkill, Who: "Hero", What: "blade", Saved: "120", Current: "100"},
	{Kind: ModMismatchGuest, Mod: "tales", What: "41"},
	{Kind: ModMismatchMission, What: "42"},
}}}

var changedModsRows = []string{
	"Hero: cloak of mod armory [is no longer defined and was removed from the pack]",
	"Ann: helm of mod armory [is no longer defined and was taken off and removed]",
	"Ann: clothing layer veil of mod armory [is no longer defined and was taken off]",
	"Hero, Blade: lowered from [120] to [100]",
	"Mod tales: the guest for mission 41 [no longer waits]",
	"Mission 42: [no longer declared, removed from the campaign]",
}

// TestATownLoadUnderChangedModsListsWhatWentThenEntersTheTown: the load is
// complete, the Load page's mod window lists what the mods took under its
// own title, and closing it enters the town.
func TestATownLoadUnderChangedModsListsWhatWentThenEntersTheTown(t *testing.T) {
	for _, closer := range []string{"escape", "enter"} {
		t.Run(closer, func(t *testing.T) {
			a := mismatchApp(t, nil)
			a.flow.town = &stubTown{rows: []TownRow{{Text: "gates", Choosable: true}}}
			a.flow.loadGame = func(string) (MapOpener, bool, error) { return nil, true, changedModsNotice }
			completed := a.flow.loadUI.completed
			mismatchKey(t, a, "enter")
			got, open := a.HeadlessModMismatch()
			if !open || a.Screen() != ScreenLoad {
				t.Fatalf("no window on the Load page: %v %v, message %q", open, a.Screen(), a.HeadlessMessage())
			}
			if !reflect.DeepEqual(got.Rows, changedModsRows) {
				t.Fatalf("rows\n%q\nwant\n%q", got.Rows, changedModsRows)
			}
			if got.Title != "The save was loaded under changed mods" || !strings.Contains(got.Hint, "took them away") {
				t.Fatalf("title %q hint %q", got.Title, got.Hint)
			}
			mismatchKey(t, a, closer)
			if _, open := a.HeadlessModMismatch(); open || a.Screen() != ScreenTown || a.flow.loadUI.completed != completed+1 {
				t.Fatalf("closing left window %v on %v", open, a.Screen())
			}
		})
	}
}

// TestATownLoadUnderChangedModsWithoutTheFontSaysItOnTheMessageLine: a build
// without the installed font enters the town at once and keeps the list on
// the message line.
func TestATownLoadUnderChangedModsWithoutTheFontSaysItOnTheMessageLine(t *testing.T) {
	f := newFlow(NewPicker(nil), nil)
	f.town = &stubTown{rows: []TownRow{{Text: "gates", Choosable: true}}}
	f.saveList = func() []SaveEntry { return []SaveEntry{{Name: "t.sav", Label: "town"}} }
	f.loadGame = func(string) (MapOpener, bool, error) { return nil, true, changedModsNotice }
	f.openLoad(ScreenMenu)
	f.loadList.Select(0)
	f.chooseLoad()
	if f.screen != ScreenTown || f.loadUI.mismatch != nil {
		t.Fatalf("landed on %v with window %v", f.screen, f.loadUI.mismatch != nil)
	}
	if !strings.Contains(f.msg, "removed from the pack") || !strings.Contains(f.msg, "lowered from 120 to 100") {
		t.Fatalf("message %q", f.msg)
	}
}

// TestAQuickLoadUnderChangedModsNamesWhatWent: a quick load from the town
// paints the whole list in the Load page's mod window, every row in its
// alert ink inside the well, and closing it enters the town. A build without
// the font says it on the message line under the word book's prefix.
func TestAQuickLoadUnderChangedModsNamesWhatWent(t *testing.T) {
	a := mismatchApp(t, nil)
	a.flow.town = &stubTown{rows: []TownRow{{Text: "gates", Choosable: true}}}
	if err := a.flow.loadPrepared(nil, true); err != nil || a.Screen() != ScreenTown {
		t.Fatal("town", err, a.Screen())
	}
	a.flow.loadUI.quick.Load = func() (MapOpener, bool, error) { return nil, true, changedModsNotice }
	completed := a.flow.loadUI.completed
	a.stepQuickSave(false)
	got, open := a.HeadlessModMismatch()
	if !open || !reflect.DeepEqual(got.Rows, changedModsRows) || got.Title != "The save was loaded under changed mods" {
		t.Fatalf("window %v on %v: %q %q, message %q", open, a.Screen(), got.Title, got.Rows, a.flow.msg)
	}
	pix := loadScrollTestFrame(t, a)
	l := newNoticeWindowLayout(a.flow.menuFont, loadPanel, 1)
	lines := a.flow.loadUI.mismatch.noticeLines(a.flow.menuFont, l, a.modMismatchDisplay)
	if len(lines) < len(changedModsRows) || len(lines) > l.List.Rows {
		t.Fatalf("%d lines in a well of %d", len(lines), l.List.Rows)
	}
	for i := range lines {
		if !slices.ContainsFunc(lines[i].runs, func(r inkRun) bool { return r.Alert }) {
			continue
		}
		r, alert := l.List.Row(i), 0
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if pix.RGBAAt(x, y) == noticeAlertInk {
					alert++
				}
			}
		}
		if alert == 0 {
			t.Errorf("row %d %q painted no alert ink", i, changedModsRows[i])
		}
	}
	mismatchKey(t, a, "enter")
	if _, open := a.HeadlessModMismatch(); open || a.Screen() != ScreenTown || a.flow.loadUI.completed != completed+1 {
		t.Fatalf("closing left window %v on %v", open, a.Screen())
	}

	f := newFlow(NewPicker(nil), nil)
	f.town = &stubTown{rows: []TownRow{{Text: "gates", Choosable: true}}}
	b := &App{flow: f}
	f.loadUI.quick.Load = a.flow.loadUI.quick.Load
	b.stepQuickSave(false)
	if f.screen != ScreenTown || !strings.HasPrefix(f.msg, "Quickload complete under changed mods: Hero: ") || !strings.Contains(f.msg, "Mission 42") {
		t.Fatalf("without the font: %v %q", f.screen, f.msg)
	}
}

// TestASaveWrittenBeforeItemKeysNamesWhyItIsRefused: the refusal of a save
// whose mod items carry no keys leads its differences with its own row.
func TestASaveWrittenBeforeItemKeysNamesWhyItIsRefused(t *testing.T) {
	keyless := ModMismatch{Reason: ModMismatchKeyless, Rows: []ModMismatchRow{{Kind: ModMismatchFiles, Mod: "armory", Saved: "1.0"}}}
	a := mismatchApp(t, fixtureMismatch{keyless})
	mismatchKey(t, a, "enter")
	got, open := a.HeadlessModMismatch()
	want := []string{
		"[The save was written before saves stored the names of mod items,] so they cannot be matched under the changed mods:",
		"Mod armory 1.0: [its files differ from the ones the save was made with]",
	}
	if !open || !reflect.DeepEqual(got.Rows, want) || got.Title != "The save does not match the active mods" {
		t.Fatalf("window %v: %q %q", open, got.Title, got.Rows)
	}
}

// TestALoadWithoutANoticeOpensNoWindow: a load that lost nothing answers no
// notice and shows no list.
func TestALoadWithoutANoticeOpensNoWindow(t *testing.T) {
	if m, err := loadNoticeOf(nil); m != nil || err != nil {
		t.Fatal(m, err)
	}
	a := mismatchApp(t, nil)
	a.flow.town = &stubTown{rows: []TownRow{{Text: "gates", Choosable: true}}}
	a.flow.loadGame = func(string) (MapOpener, bool, error) { return nil, true, nil }
	mismatchKey(t, a, "enter")
	if _, open := a.HeadlessModMismatch(); open || a.Screen() != ScreenTown {
		t.Fatalf("window %v on %v", open, a.Screen())
	}
}
