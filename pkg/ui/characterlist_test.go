package ui

import (
	"image"
	"testing"
	"time"
)

// multiplayerApps are a first-game and a second-game application on
// synthetic menu art, each with one save the Load page lists.
func multiplayerApps(t *testing.T) map[string]*App {
	t.Helper()
	second := NewApp("t", secondAppAssets(t), appRows(1), okLoader(t))
	second.Layout(640, 480)
	out := map[string]*App{"first": newTestApp(t, appRows(1), okLoader(t)), "second": second}
	for _, a := range out {
		a.SetSaveSeams(nil, func() []SaveEntry { return []SaveEntry{{Name: "game0001.sav", Label: "saved"}} }, nil)
		a.SetSaveFiles(func() bool { return true })
	}
	return out
}

// clickPageButton presses and releases the open page's button i at its
// centre.
func clickPageButton(t *testing.T, a *App, i int, now time.Time) {
	t.Helper()
	r := a.flow.loadPage().Buttons[i].Rect
	x, y, ok := a.nativeFrameToWindow(r.Min.Add(r.Size().Div(2)))
	if !ok {
		t.Fatalf("page button %d at %v is not in the window", i, r)
	}
	a.step(appInput{CursorX: x, CursorY: y}, now)
	clickAt(a, now, image.Pt(x, y))
}

// The Multiplayer button opens the list page's character role in both games:
// the load page's window and Cancel, its own title and Ok, and an empty list.
// Ok on the empty list stays; Cancel returns to the menu, and Load then opens
// the load page as before.
func TestMultiplayerOpensTheCharacterListInBothGames(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	apps := multiplayerApps(t)
	for _, name := range []string{"first", "second"} {
		t.Run(name, func(t *testing.T) { checkCharacterList(t, apps[name], now) })
	}
}

func checkCharacterList(t *testing.T, a *App, now time.Time) {
	t.Helper()
	load := a.flow.loadPage()
	if err := a.HeadlessActivate("multi player"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenLoad {
		t.Fatalf("Multiplayer opens %v, want the list page", a.Screen())
	}
	page := a.flow.loadPage()
	if len(page.Texts) != 1 || page.Texts[0].Text != "Characters" || page.Texts[0].Box != load.Texts[0].Box {
		t.Errorf("character page texts %+v, want the title Characters in the load title's box", page.Texts)
	}
	if page.Panel != load.Panel || page.List != load.List || page.Buttons[0].Caption != "Accept" ||
		page.Buttons[0].Rect != load.Buttons[0].Rect || page.Buttons[2] != load.Buttons[2] {
		t.Errorf("character page %+v, want the load window with Accept and its Cancel", page)
	}
	if rows := a.HeadlessRows(); len(rows) != 0 || a.HeadlessMessage() != "" {
		t.Errorf("character list rows %v message %q, want none", rows, a.HeadlessMessage())
	}
	clickPageButton(t, a, 0, now)
	if a.Screen() != ScreenLoad || len(a.HeadlessRows()) != 0 {
		t.Errorf("Ok on the empty list left for %v", a.Screen())
	}
	clickPageButton(t, a, 2, now)
	if a.Screen() != ScreenMenu {
		t.Fatalf("Cancel left the character list for %v, want the menu", a.Screen())
	}
	if err := a.HeadlessActivate("load game"); err != nil {
		t.Fatal(err)
	}
	again := a.flow.loadPage()
	if a.Screen() != ScreenLoad || len(a.HeadlessRows()) != 1 || again.Texts[0].Text != load.Texts[0].Text ||
		len(again.Texts) != len(load.Texts) || again.Buttons != load.Buttons {
		t.Errorf("Load after the character list opens %v with %d rows and texts %+v", a.Screen(),
			len(a.HeadlessRows()), again.Texts)
	}
}
