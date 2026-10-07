package ui

import (
	"bytes"
	"image"
	"slices"
	"testing"
)

const panelCasterID, panelVictimID = 5, 9

func panelCasterApp(t *testing.T) (*App, *Viewer) {
	t.Helper()
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.SetEntities([]MapEntity{
		{ID: panelCasterID, Cell: image.Pt(3, 3), Life: LifeAlive, CastCapable: true},
		{ID: panelVictimID, Cell: image.Pt(7, 3), Life: LifeAlive},
	})
	v.Camera().X, v.Camera().Y = 0, 0
	v.Camera().Clamp()
	v.sel = selection{panelCasterID}
	v.SetSpellbook(panelCasterID, sbBook())
	v.SetInventorySubject(InventorySubject{ID: panelCasterID})
	for _, p := range []hudPanel{hudPanelPack, hudPanelBook} {
		if v.hudShown(p) {
			v.toggleHudPanel(p)
		}
	}
	return a, v
}

func pressPanelKey(t *testing.T, a *App, name string) {
	t.Helper()
	if err := a.HeadlessKey(name); err != nil {
		t.Fatal(err)
	}
}

func panelsOpen(v *Viewer) (pack, book bool) {
	return v.hudShown(hudPanelPack), v.hudShown(hudPanelBook)
}

func TestSpaceClosesEveryPanelWhenAnyIsOpenAndOpensAllOnlyFromClosed(t *testing.T) {
	a, v := panelCasterApp(t)
	pressPanelKey(t, a, "space")
	if pack, book := panelsOpen(v); !pack || !book {
		t.Fatalf("all closed then Space: pack=%v book=%v, want both open", pack, book)
	}
	pressPanelKey(t, a, "space")
	if pack, book := panelsOpen(v); pack || book {
		t.Fatalf("all open then Space: pack=%v book=%v, want both closed", pack, book)
	}
	for _, hand := range []string{"book", "inventory"} {
		pressPanelKey(t, a, hand)
		pack, book := panelsOpen(v)
		if pack == book {
			t.Fatalf("%s did not leave a mixed set", hand)
		}
		pressPanelKey(t, a, "space")
		if pack, book := panelsOpen(v); pack || book {
			t.Fatalf("mixed set (opened by %s) then Space: pack=%v book=%v, want all closed", hand, pack, book)
		}
		pressPanelKey(t, a, "space")
		if pack, book := panelsOpen(v); !pack || !book {
			t.Fatalf("after the closing press Space: pack=%v book=%v, want all open", pack, book)
		}
		pressPanelKey(t, a, "space")
	}
}

func TestDeselectKeepsOpenPanelsEmptyAndReselectFillsThem(t *testing.T) {
	a, v := panelCasterApp(t)
	v.SetWords(Words{NoHeroSelected: "NO HERO SELECTED", PauseNotice: "p"})
	pressPanelKey(t, a, "space")

	agree := func(when string, wantOpen bool) {
		t.Helper()
		pack, book := panelsOpen(v)
		_, _, packDrawn := v.packBarPresent()
		_, _, bookDrawn := v.spellbookPresent()
		view := v.characterPaneView(false)
		if pack != wantOpen || book != wantOpen || packDrawn != wantOpen || bookDrawn != wantOpen ||
			view.PackOpen != wantOpen || view.BookOpen != wantOpen {
			t.Fatalf("%s: switches %v/%v drawn %v/%v icons %v/%v, want all %v",
				when, pack, book, packDrawn, bookDrawn, view.PackOpen, view.BookOpen, wantOpen)
		}
	}
	agree("selected, opened", true)
	filled, _, _ := v.packBarPresent()

	x, y := cellPoint(v, 10, 8)
	rightUpAt(v, x, y)
	if len(v.sel) != 0 {
		t.Fatal("setup: the right click did not deselect")
	}
	v.ClearSpellbook()
	agree("deselected", true)
	empty, _, _ := v.packBarPresent()
	if empty == nil || filled == nil || empty.Bounds() != filled.Bounds() {
		t.Fatal("deselected pack bar is not the same grid")
	}
	withText, _, _ := v.spellbookPresent()
	v.SetWords(Words{})
	noText, _, _ := v.spellbookPresent()
	if bytes.Equal(withText.Pix, noText.Pix) {
		t.Fatal("no-hero line did not reach the empty spellbook strip")
	}
	if _, ok := v.spellbookEntryAt(x, y); ok {
		t.Fatal("empty strip names an entry")
	}

	sx, sy := cellPoint(v, 7, 3)
	tapAt(v, sx, sy)
	v.sel = selection{panelCasterID}
	v.SetSpellbook(panelCasterID, sbBook())
	agree("reselected", true)
	if _, _, ok := v.spellbookBar(); !ok || !v.spellbookHeld {
		t.Fatal("reselect did not fill the book")
	}
	pressPanelKey(t, a, "space")
	agree("closed after reselect", false)
}

func TestEmptyOpenPackLeavesTheMapClickableWithNoSelection(t *testing.T) {
	a, v := panelCasterApp(t)
	pressPanelKey(t, a, "space")
	bar, _, ok := v.packBarShown()
	if !ok {
		t.Fatal("pack not shown")
	}
	v.sel = nil
	v.ClearSpellbook()
	if v.inventoryCaptures(bar.Min.X+10, bar.Min.Y+10) {
		t.Fatal("the empty pack took a press that belongs to the map")
	}
	if _, _, ok := v.packBar(); ok {
		t.Fatal("interactive pack answered with no subject")
	}
}

func selectSpellEntry(t *testing.T, v *Viewer) {
	t.Helper()
	book := sbBook()
	x, y := sbEntryPoint(t, v, book, 0)
	sbClick(v, x, y)
	if v.selectedSpell != book[0].ID {
		t.Fatalf("setup: spell %d selected, want %d", v.selectedSpell, book[0].ID)
	}
}

func TestManualBookGivesRepeatedCastsAndStaysOpen(t *testing.T) {
	a, v := panelCasterApp(t)
	pressPanelKey(t, a, "space")
	selectSpellEntry(t, v)
	for i := 0; i < 3; i++ {
		x, y := cellPoint(v, 7, 3)
		if got := v.gestureCursorAt(x, y); got != "cast" {
			t.Fatalf("cast %d: cursor %q before the click", i, got)
		}
		ords, ok := tapAt(v, x, y)
		if !ok || len(ords) != 1 || ords[0].kind != orderKindCast {
			t.Fatalf("cast %d: click produced %+v", i, ords)
		}
		if !v.hudShown(hudPanelBook) || !v.spellModeLive() {
			t.Fatalf("cast %d: book open=%v cast live=%v", i, v.hudShown(hudPanelBook), v.spellModeLive())
		}
		if got := v.gestureCursorAt(x, y); got != "cast" {
			t.Fatalf("cast %d: cursor %q after the click", i, got)
		}
	}
}

func TestCastKeyIsOneShotAndAgreesWithSpace(t *testing.T) {
	a, v := panelCasterApp(t)
	pressPanelKey(t, a, "space")
	selectSpellEntry(t, v)
	pressPanelKey(t, a, "space") // all closed; the book-chosen mode lapses
	if p, b := panelsOpen(v); p || b || v.spellModeLive() {
		t.Fatalf("setup: closed set %v/%v live %v", p, b, v.spellModeLive())
	}
	pressPanelKey(t, a, "c")
	x, y := cellPoint(v, 7, 3)
	if !v.hudShown(hudPanelBook) || !v.spellModeLive() || v.gestureCursorAt(x, y) != "cast" {
		t.Fatalf("C: book %v live %v cursor %q", v.hudShown(hudPanelBook), v.spellModeLive(), v.gestureCursorAt(x, y))
	}
	if v.hudShown(hudPanelPack) {
		t.Fatal("C opened the pack")
	}
	pressPanelKey(t, a, "space")
	if p, b := panelsOpen(v); p || b {
		t.Fatalf("Space after C: %v/%v, want all closed", p, b)
	}
	pressPanelKey(t, a, "space")
	if p, b := panelsOpen(v); !p || !b {
		t.Fatalf("second Space: %v/%v, want all open", p, b)
	}
	pressPanelKey(t, a, "space")
	pressPanelKey(t, a, "c")
	ords, ok := tapAt(v, x, y)
	if !ok || len(ords) != 1 || ords[0].kind != orderKindCast {
		t.Fatalf("one-shot click produced %+v", ords)
	}
	if v.hudShown(hudPanelBook) || v.spellModeLive() || v.selectedSpell != sbBook()[0].ID {
		t.Fatalf("after the cast: book %v live %v spell %d (the selection stays)", v.hudShown(hudPanelBook), v.spellModeLive(), v.selectedSpell)
	}
	if got := v.gestureCursorAt(x, y); got == "cast" {
		t.Fatal("cursor stayed the cast cursor")
	}
	if _, book := panelsOpen(v); book != v.characterPaneView(false).BookOpen {
		t.Fatal("book icon disagrees with the book")
	}
	pressPanelKey(t, a, "space")
	if p, b := panelsOpen(v); !p || !b {
		t.Fatalf("Space after the one-shot: %v/%v, want all open", p, b)
	}
}

func TestCastKeyBookOwnershipAndCancel(t *testing.T) {
	a, v := panelCasterApp(t)
	pressPanelKey(t, a, "space")
	selectSpellEntry(t, v)
	pressPanelKey(t, a, "c")
	x, y := cellPoint(v, 7, 3)
	if ords, _ := tapAt(v, x, y); len(ords) != 1 || !v.hudShown(hudPanelBook) {
		t.Fatalf("a hand-opened book closed after a C cast: %+v", ords)
	}
	pressPanelKey(t, a, "space")
	pressPanelKey(t, a, "c")
	if !v.hudShown(hudPanelBook) {
		t.Fatal("C did not open the book")
	}
	rightUpAt(v, x, y)
	if v.hudShown(hudPanelBook) || v.spellArmed || !slices.Equal(v.sel, selection{panelCasterID}) {
		t.Fatalf("cancel: book %v armed %v sel %v", v.hudShown(hudPanelBook), v.spellArmed, v.sel)
	}
}

func TestCastCellOpensTheBookLikeC(t *testing.T) {
	_, v := panelCasterApp(t)
	v.selectedSpell = sbBook()[0].ID
	v.pressCommandPanelCell(commandCellCast, true)
	if !v.hudShown(hudPanelBook) || !v.spellArmed || !v.castOnce {
		t.Fatalf("Cast cell: book %v armed %v once %v", v.hudShown(hudPanelBook), v.spellArmed, v.castOnce)
	}
}

func TestCastKeyWithNoSpellSelectedIsANoOp(t *testing.T) {
	a, v := panelCasterApp(t)
	pressPanelKey(t, a, "c")
	if v.hudShown(hudPanelBook) || v.spellArmed || v.castOnce || v.missionMode() == modeCast {
		t.Fatalf("C without a spell: book %v armed %v mode %d", v.hudShown(hudPanelBook), v.spellArmed, v.missionMode())
	}
}

func TestCastKeyHookSurvivesClosingTheBook(t *testing.T) {
	a, v := panelCasterApp(t)
	pressPanelKey(t, a, "space")
	selectSpellEntry(t, v)
	pressPanelKey(t, a, "space")
	pressPanelKey(t, a, "c")
	pressPanelKey(t, a, "book")
	if v.hudShown(hudPanelBook) || !v.spellModeLive() {
		t.Fatalf("closing the book: book %v live %v, want closed and still armed", v.hudShown(hudPanelBook), v.spellModeLive())
	}
	x, y := cellPoint(v, 7, 3)
	if got := v.gestureCursorAt(x, y); got != "cast" {
		t.Fatalf("cursor %q with the book closed after C, want cast", got)
	}
	if ords, _ := tapAt(v, x, y); len(ords) != 1 || ords[0].kind != orderKindCast {
		t.Fatalf("armed hook click produced %+v", ords)
	}
	if v.spellModeLive() || v.hudShown(hudPanelBook) || v.selectedSpell == 0 {
		t.Fatalf("after the cast: live %v book %v spell %d", v.spellModeLive(), v.hudShown(hudPanelBook), v.selectedSpell)
	}
}
