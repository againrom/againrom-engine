package ui

import (
	"bytes"
	"fmt"
	"image"
	"strings"
	"testing"
)

func modTestScreens() []ModScreen {
	return []ModScreen{
		{Mod: "m", Key: "about", Kind: ModScreenInfo, Title: "About", MenuLabel: "About entry", Main: true, Game: true,
			Paragraphs: []string{"First paragraph.", "Second paragraph."}},
		{Mod: "m", Key: "perks", Kind: ModScreenList, Title: "Perks", MenuLabel: "Perks entry", Main: true,
			Items: []string{"One", "Two"}},
		{Mod: "m", Key: "stats", Kind: ModScreenTable, Title: "Stats", MenuLabel: "Stats entry", Game: true,
			Rows: [][2]string{{"Label", "Value"}}},
	}
}

func modTestApp(t *testing.T, screens []ModScreen) *App {
	t.Helper()
	a := newTestApp(t, appRows(2), okLoader(t))
	if err := a.SetModScreens(screens); err != nil {
		t.Fatal(err)
	}
	return a
}

func click(a *App, p image.Point) {
	a.step(appInput{CursorX: p.X, CursorY: p.Y}, atAt)
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true}, atAt)
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true}, atAt)
}

func midOf(r image.Rectangle) image.Point { return r.Min.Add(r.Max).Div(2) }

func TestScreenHandlersAgreeWithTheCensusRegistry(t *testing.T) {
	for _, e := range ScreenCensus() {
		h, ok := screenHandlers[e.Screen]
		switch {
		case e.Screen == ScreenMap || e.Screen == ScreenCutscene:
			// Composed by their own routes, not by composeScreen.
			if ok && h.Compose != nil {
				t.Errorf("%s has a Compose handler but the census says its frame is not composed by composeScreen", e.Name)
			}
		case e.Composer != "":
			if !ok || h.Compose == nil {
				t.Errorf("%s names a composer in the census but has no Compose handler", e.Name)
			}
		default:
			if ok && h.Compose != nil {
				t.Errorf("%s refuses composition in the census but has a Compose handler", e.Name)
			}
		}
		if e.Screen != ScreenMap && e.Screen != ScreenCutscene && (!ok || h.Draw == nil) {
			t.Errorf("%s has no Draw handler", e.Name)
		}
	}
	mod := screenHandlers[ScreenMod]
	if mod.Step == nil || mod.Compose == nil || mod.Draw == nil {
		t.Fatalf("the mod screen handler is incomplete: %+v", mod)
	}
	for _, e := range ScreenCensus() {
		if e.Screen == ScreenMod {
			t.Fatal("the shipped census lists the mod screen")
		}
	}
}

func TestUnmoddedMenuFrameIsTheBroochAndLabelOnly(t *testing.T) {
	a := newTestApp(t, appRows(2), okLoader(t))
	a.SetMenuLabel("label")
	got, err := a.composeScreen()
	if err != nil {
		t.Fatal(err)
	}
	want := a.assets.Compose(a.sel.State())
	a.drawMenuLabel(want)
	if !bytes.Equal(got.Pix, want.Pix) {
		t.Fatal("an unmodded menu frame differs from the brooch with its label")
	}
	// Loss control: registering a main menu entry changes the frame.
	if err := a.SetModScreens(modTestScreens()); err != nil {
		t.Fatal(err)
	}
	with, err := a.composeScreen()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(with.Pix, want.Pix) {
		t.Fatal("a registered main menu entry changes nothing in the frame")
	}
	// A screen placed only in the game menu leaves the main menu frame alone.
	if err := a.SetModScreens(modTestScreens()[2:]); err != nil {
		t.Fatal(err)
	}
	game, _ := a.composeScreen()
	if !bytes.Equal(game.Pix, want.Pix) {
		t.Fatal("a game menu entry changed the main menu frame")
	}
}

func TestMainMenuEntryOpensTheScreenAndBackReturns(t *testing.T) {
	a := modTestApp(t, modTestScreens())
	rects := modMenuEntryRects(2)
	click(a, midOf(rects[1]))
	if a.Screen() != ScreenMod || a.flow.modUI.open != 1 {
		t.Fatalf("screen %v open %d after clicking the second entry", a.Screen(), a.flow.modUI.open)
	}
	if _, _, err := a.HeadlessFrame(); err != nil {
		t.Fatalf("the mod screen has no frame: %v", err)
	}
	click(a, midOf(modBackButton))
	if a.Screen() != ScreenMenu {
		t.Fatalf("Back returned to %v", a.Screen())
	}
	click(a, midOf(rects[0]))
	if a.Screen() != ScreenMod || a.flow.modUI.open != 0 {
		t.Fatalf("first entry opened %d on %v", a.flow.modUI.open, a.Screen())
	}
	a.step(appInput{Escape: true}, atAt)
	if a.Screen() != ScreenMenu {
		t.Fatalf("Escape returned to %v", a.Screen())
	}
	click(a, midOf(rects[0]))
	a.step(appInput{Enter: true}, atAt)
	if a.Screen() != ScreenMenu {
		t.Fatalf("Enter returned to %v", a.Screen())
	}
}

func TestPressOffTheEntryOpensNothing(t *testing.T) {
	a := modTestApp(t, modTestScreens())
	r := modMenuEntryRects(2)[0]
	p := midOf(r)
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true}, atAt)
	a.step(appInput{CursorX: 600, CursorY: 20, PrimaryReleased: true}, atAt)
	if a.Screen() != ScreenMenu {
		t.Fatalf("a press dragged off the entry opened %v", a.Screen())
	}
	// A release without a press on the entry opens nothing either.
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true}, atAt)
	if a.Screen() != ScreenMenu {
		t.Fatalf("a bare release opened %v", a.Screen())
	}
}

func TestMainMenuHoverRedrawsTheEntry(t *testing.T) {
	a := modTestApp(t, modTestScreens())
	// The entry is a push button: hover turns its caption gold (MENU-115).
	a.flow.menuFont = gameMenuTestFont()
	a.drawMenuForTest()
	idle, _ := a.composeScreen()
	p := midOf(modMenuEntryRects(2)[0])
	a.step(appInput{CursorX: p.X, CursorY: p.Y}, atAt)
	if a.hasMenu {
		t.Fatal("hover over an entry kept the cached menu frame")
	}
	hot, _ := a.composeScreen()
	if bytes.Equal(idle.Pix, hot.Pix) {
		t.Fatal("hover does not highlight the entry")
	}
}

// drawMenuForTest marks the cached menu frame composed, as a drawn frame does.
func (a *App) drawMenuForTest() { a.hasMenu = true }

func TestGameMenuRowsAppendTheModEntriesAndOpenThem(t *testing.T) {
	for _, surface := range []Screen{ScreenTown, ScreenMap} {
		a := modTestApp(t, modTestScreens())
		a.flow.openGameMenu(surface)
		rows := a.flow.menuRows()
		base := gameMenuBaseRows(a.flow.menuSurface)
		if len(rows) != base+2 {
			t.Fatalf("%v: %d rows, want %d", surface, len(rows), base+2)
		}
		for i, label := range []string{"About entry", "Stats entry"} {
			r := rows[base+i]
			if r.Action != gameMenuModScreen || r.Label != label || !r.Enabled || r.Mod == 0 {
				t.Fatalf("%v: row %d is %+v", surface, base+i, r)
			}
		}
		if got := a.flow.menuList.Rows(); len(got) != base+2 {
			t.Fatalf("%v: the list shows %d rows", surface, len(got))
		}
		if top, count := a.flow.menuList.Visible(); top != 0 || count != base+2 {
			t.Fatalf("%v: window shows %d rows from %d", surface, count, top)
		}
		// Choosing the second mod row opens the table screen and returns to it.
		a.flow.menuList.Select(base + 1)
		a.chooseGameMenu()
		if a.Screen() != ScreenMod || a.flow.modUI.open != 2 || a.flow.modUI.back != ScreenGameMenu {
			t.Fatalf("%v: screen %v open %d back %v", surface, a.Screen(), a.flow.modUI.open, a.flow.modUI.back)
		}
		a.step(appInput{Escape: true}, atAt)
		if a.Screen() != ScreenGameMenu || a.flow.menuList.Selection() != base+1 || a.flow.menuPage != gameMenuRoot {
			t.Fatalf("%v: back on %v selecting %d", surface, a.Screen(), a.flow.menuList.Selection())
		}
	}
}

func TestGameMenuModRowIsClickableWhereItIsDrawn(t *testing.T) {
	a := modTestApp(t, modTestScreens())
	a.flow.openGameMenu(ScreenTown)
	base := gameMenuBaseRows(gameMenuTown)
	r := gameMenuRowRect(gameMenuTown, base)
	if p := gameMenuPanelRectFor(gameMenuTown, a.flow.menuRows()); !r.In(p) || p.Dy() <= gameMenuPanelRect(gameMenuTown).Dy() {
		t.Fatalf("mod row %v is not inside the grown panel %v", r, p)
	}
	click(a, midOf(r))
	if a.Screen() != ScreenMod || a.flow.modUI.open != 0 {
		t.Fatalf("clicking the mod row opened %d on %v", a.flow.modUI.open, a.Screen())
	}
}

func TestGameMenuPanelIsDecodedWithoutModRows(t *testing.T) {
	for _, s := range []gameMenuSurface{gameMenuMission, gameMenuTown} {
		rows := make([]gameMenuRow, gameMenuBaseRows(s))
		if got := gameMenuPanelRectFor(s, rows); got != gameMenuPanelRect(s) {
			t.Errorf("surface %d: panel %v, decoded %v", s, got, gameMenuPanelRect(s))
		}
		// Loss control: one mod row grows the town panel and keeps the mission one.
		rows = append(rows, gameMenuRow{Mod: 1})
		got := gameMenuPanelRectFor(s, rows)
		if s == gameMenuTown && got.Dy() <= gameMenuPanelRect(s).Dy() {
			t.Errorf("a mod row did not grow the town panel")
		}
		if s == gameMenuMission && got != gameMenuPanelRect(s) {
			t.Errorf("one mod row moved the mission panel: %v", got)
		}
	}
}

func TestUnmoddedGameMenuListIsUnchanged(t *testing.T) {
	a := newTestApp(t, appRows(2), okLoader(t))
	a.flow.openGameMenu(ScreenTown)
	rows := a.flow.menuRows()
	want := townGameMenuRows(a.flow.words)
	if len(rows) != len(want) {
		t.Fatalf("%d rows, want %d", len(rows), len(want))
	}
	for i := range rows {
		if rows[i].Label != want[i].Label || rows[i].Action != want[i].Action || rows[i].Mod != 0 {
			t.Fatalf("row %d is %+v, want %+v", i, rows[i], want[i])
		}
	}
	if _, count := a.flow.menuList.Visible(); count != len(want) {
		t.Fatalf("window shows %d rows", count)
	}
}

func TestModScreenBodyScrolls(t *testing.T) {
	long := strings.Repeat("word ", 400)
	a := modTestApp(t, []ModScreen{{Mod: "m", Key: "k", Kind: ModScreenInfo, Title: "T", MenuLabel: "L", Main: true, Paragraphs: []string{long}}})
	click(a, midOf(modMenuEntryRects(1)[0]))
	if a.Screen() != ScreenMod {
		t.Fatalf("screen %v", a.Screen())
	}
	first, err := a.composeModScreen()
	if err != nil {
		t.Fatal(err)
	}
	if a.modMaxTop() == 0 {
		t.Fatal("a 2000-character paragraph fits one page")
	}
	a.step(appInput{Down: true}, atAt)
	if a.flow.modUI.top != 1 {
		t.Fatalf("Down left the offset at %d", a.flow.modUI.top)
	}
	second, _ := a.composeModScreen()
	if bytes.Equal(first.Pix, second.Pix) {
		t.Fatal("scrolling did not change the page")
	}
	a.step(appInput{WheelY: -1}, atAt)
	if a.flow.modUI.top != 1+modWheelLines {
		t.Fatalf("wheel left the offset at %d", a.flow.modUI.top)
	}
	for i := 0; i < 1000; i++ {
		a.step(appInput{Down: true}, atAt)
	}
	if a.flow.modUI.top != a.modMaxTop() {
		t.Fatalf("offset %d runs past the last line %d", a.flow.modUI.top, a.modMaxTop())
	}
	a.step(appInput{WheelY: 1}, atAt)
	if a.flow.modUI.top >= a.modMaxTop() {
		t.Fatal("wheel up did not scroll back")
	}
}

func TestModLinesLayOutEachKind(t *testing.T) {
	a := modTestApp(t, modTestScreens())
	s := modTestScreens()
	info := a.modLines(s[0])
	if len(info) != 3 || info[1] != (modLine{}) || info[0].left != "First paragraph." {
		t.Fatalf("info lines %+v", info)
	}
	list := a.modLines(s[1])
	if len(list) != 2 || list[0].left != "-" || list[0].right != "One" {
		t.Fatalf("list lines %+v", list)
	}
	table := a.modLines(s[2])
	if len(table) != 1 || table[0].left != "Label" || table[0].right != "Value" {
		t.Fatalf("table lines %+v", table)
	}
	// A value wider than its column wraps without losing a word.
	wide := ModScreen{Kind: ModScreenTable, Rows: [][2]string{{"L", strings.Repeat("alpha ", 20)}}}
	lines := a.modLines(wide)
	if len(lines) < 2 {
		t.Fatalf("a long value is one line: %+v", lines)
	}
	var joined []string
	for _, l := range lines {
		joined = append(joined, l.right)
	}
	if got := strings.Fields(strings.Join(joined, " ")); len(got) != 20 {
		t.Fatalf("wrapping kept %d of 20 words", len(got))
	}
}

func TestModTextWrapCutsAWordWiderThanTheLine(t *testing.T) {
	a := modTestApp(t, nil)
	lines := a.modWrap(strings.Repeat("x", 200), 60)
	if len(lines) < 3 {
		t.Fatalf("%d lines", len(lines))
	}
	total := 0
	for _, l := range lines {
		if a.modMeasure(l) > 60 {
			t.Fatalf("line %q is wider than the column", l)
		}
		total += len(l)
	}
	if total != 200 {
		t.Fatalf("the cut lost characters: %d", total)
	}
}

func TestSetModScreensRefusals(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	ok := ModScreen{Mod: "m", Key: "k", Kind: ModScreenInfo, Title: "T", MenuLabel: "L", Main: true}
	many := func(n int, place func(*ModScreen)) []ModScreen {
		var out []ModScreen
		for i := 0; i < n; i++ {
			s := ok
			s.Main = false
			s.Key = fmt.Sprintf("k%d", i)
			place(&s)
			out = append(out, s)
		}
		return out
	}
	cases := map[string][]ModScreen{
		"unknown kind":   {{Mod: "m", Key: "k", Kind: 9, Title: "T", MenuLabel: "L", Main: true}},
		"no menu label":  {{Mod: "m", Key: "k", Kind: ModScreenInfo, Title: "T", Main: true}},
		"no placement":   {{Mod: "m", Key: "k", Kind: ModScreenInfo, Title: "T", MenuLabel: "L"}},
		"too many main":  many(MaxModMainEntries+1, func(s *ModScreen) { s.Main = true }),
		"too many games": many(MaxModGameEntries+1, func(s *ModScreen) { s.Game = true }),
	}
	for name, screens := range cases {
		if err := a.SetModScreens(screens); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if err := a.SetModScreens(many(MaxModMainEntries, func(s *ModScreen) { s.Main = true })); err != nil {
		t.Errorf("a full main menu was refused: %v", err)
	}
}

func TestModScreenOpenedFromTheMissionMenuReportsTheMapBehindIt(t *testing.T) {
	a := modTestApp(t, modTestScreens())
	a.flow.openGameMenu(ScreenMap)
	base := gameMenuBaseRows(a.flow.menuSurface)
	a.flow.menuList.Select(base)
	a.chooseGameMenu()
	if a.Screen() != ScreenMod || a.HeadlessGameplayScreen() != ScreenMap {
		t.Fatalf("screen %v, gameplay %v", a.Screen(), a.HeadlessGameplayScreen())
	}
}
