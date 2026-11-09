package ui

import (
	"image"
	"image/color"
	"testing"
)

type bookScrollShot struct {
	owner, victim uint32
	index         int
	key           string
}

// bookScrollFighter is a selected fighter (no mana, no known spell) whose book
// shows a scroll of spell 24 at the fifth cell of an otherwise covered catalog.
func bookScrollFighter(t *testing.T) (*Viewer, *[]bookScrollShot) {
	t.Helper()
	_, v := panelCasterApp(t)
	v.SetEntities([]MapEntity{
		{ID: panelCasterID, Owner: 1, Cell: image.Pt(3, 3), Life: LifeAlive, SpellStateKnown: true},
		{ID: panelVictimID, Owner: 2, Cell: image.Pt(7, 3), Life: LifeAlive},
	})
	shots := new([]bookScrollShot)
	v.SetItemCastSink(func(owner uint32, index int, key string, victim uint32, x, y int, cell bool) {
		*shots = append(*shots, bookScrollShot{owner, victim, index, key})
	})
	entries := make([]SpellEntry, 24)
	for i := range entries {
		entries[i] = SpellEntry{ID: uint32(i + 1), Unavailable: true}
	}
	entries[4] = SpellEntry{ID: 24, Unavailable: true, ItemCast: &BookItemCast{Index: 2, Key: "scroll"}}
	v.SetSpellbookCatalog(panelCasterID, entries)
	if !v.hudShown(hudPanelBook) {
		v.toggleHudPanel(hudPanelBook)
	}
	return v, shots
}

func bookCellCentre(t *testing.T, v *Viewer, slot int) (int, int) {
	t.Helper()
	bar, cols, ok := v.spellbookBar()
	if !ok {
		t.Fatal("no book bar")
	}
	r := bookCellRects(bar, cols)[slot]
	return r.Min.X + r.Dx()/2, r.Min.Y + r.Dy()/2
}

// A press on the cell a scroll fills arms the pack's item cast; the release on
// the fighter's own figure reaches the item-cast sink with the scroll's pack
// cell and identity, aimed at himself (DIV-4732).
func TestABookScrollCellArmsTheItemCastAndReachesHisOwnFigure(t *testing.T) {
	v, shots := bookScrollFighter(t)
	x, y := bookCellCentre(t, v, 4)
	sbClick(v, x, y)
	if c := v.itemCast; c == nil || c.owner != panelCasterID || c.index != 2 || c.spell != 24 || c.key != "scroll" {
		t.Fatalf("item cast after the book press: %+v", c)
	}
	if v.selectedSpell != 0 || v.bookLitSpell() != 24 {
		t.Fatalf("selected spell %d lit %d, want no book spell and the scroll's cell lit", v.selectedSpell, v.bookLitSpell())
	}
	fx, fy := cellPoint(v, 3, 3)
	tapAt(v, fx, fy)
	want := bookScrollShot{panelCasterID, panelCasterID, 2, "scroll"}
	if len(*shots) != 1 || (*shots)[0] != want || v.itemCast != nil {
		t.Fatalf("sink %+v item cast %+v, want one %+v", *shots, v.itemCast, want)
	}
}

// A second press on the armed cell clears the arm, as on a spell cell, and a
// press on a cell no scroll fills arms nothing.
func TestABookScrollCellSecondPressClearsAndAnEmptyCellArmsNothing(t *testing.T) {
	v, shots := bookScrollFighter(t)
	x, y := bookCellCentre(t, v, 4)
	sbClick(v, x, y)
	sbClick(v, x, y)
	if v.itemCast != nil || v.bookLitSpell() != 0 {
		t.Fatalf("second press left %+v lit %d", v.itemCast, v.bookLitSpell())
	}
	x, y = bookCellCentre(t, v, 3)
	sbClick(v, x, y)
	if v.itemCast != nil || v.selectedSpell != 0 || v.spellArmed || len(*shots) != 0 {
		t.Fatalf("a covered cell armed: cast %+v spell %d armed %v shots %v", v.itemCast, v.selectedSpell, v.spellArmed, *shots)
	}
}

// The scroll cell stays unavailable to the book's own spell tests: it is not an
// available spell, autocast refuses it and a group selection arms nothing.
func TestABookScrollCellIsNoSpellOfHisOwn(t *testing.T) {
	v, _ := bookScrollFighter(t)
	var autocast []uint32
	v.SetAutocastSink(func(entity, spell uint32) { autocast = append(autocast, spell) })
	if v.spellAvailable(24) || v.bookOffers(24) {
		t.Fatal("the scroll cell reads as an available spell")
	}
	x, y := bookCellCentre(t, v, 4)
	v.command(appInput{SecondaryPressed: true, CursorX: x, CursorY: y})
	v.command(appInput{SecondaryReleased: true, CursorX: x, CursorY: y})
	if len(autocast) != 0 {
		t.Fatalf("autocast sink got %v", autocast)
	}
	v.sel = selection{panelCasterID, panelVictimID}
	sbClick(v, x, y)
	if v.itemCast != nil {
		t.Fatalf("a group selection armed %+v", v.itemCast)
	}
}

// The original-art bar shows a scroll cell's installed picture instead of the
// cover, lit while armed; a cell with no scroll stays covered.
func TestABookScrollCellIsDrawnAndLit(t *testing.T) {
	green := color.RGBA{0, 200, 0, 255}
	hidden := color.RGBA{9, 9, 9, 255}
	art := &BottomHUDArt{Book: solidPic(480, 85, green), UnknownSpell: solidPic(36, 36, hidden)}
	bar, cols, ok := bookBarRect(image.Pt(640, 480))
	if !ok {
		t.Fatal("no bar")
	}
	entries := []SpellEntry{{ID: 1, Unavailable: true}, {ID: 24, Unavailable: true, ItemCast: &BookItemCast{Key: "s"}}}
	cells := bookCellRects(bar, cols)
	at := func(img *image.RGBA, slot int) color.RGBA {
		c := cells[slot].Sub(bar.Min).Min
		return img.RGBAAt(c.X+1, c.Y+1)
	}
	plain := composeOriginalSpellBar(art, entries, 0, bar, 0)
	if got := at(plain, 0); got != hidden {
		t.Fatalf("no-scroll cell drew %v, want the cover", got)
	}
	shown := at(plain, 1)
	if shown == hidden {
		t.Fatal("the scroll cell is covered")
	}
	if lit := at(composeOriginalSpellBar(art, entries, 24, bar, 0), 1); lit == shown || lit.R <= shown.R {
		t.Fatalf("armed cell %v, unlit %v", lit, shown)
	}
}
