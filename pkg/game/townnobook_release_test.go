package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/ui"
)

// The school and the tavern have no spellbook: with the book switched on it
// is neither drawn nor opened there, and the book control does nothing. The
// shop keeps its book.
func TestReleaseSchoolAndTavernHaveNoSpellbook(t *testing.T) {
	f := releaseFront(t)
	f.Town = NewTown(f.Campaign.Value())
	for _, mission := range f.Campaign.Value().Main {
		if mission < 30 {
			f.Town.Won(mission)
		}
	}
	f.Town.Arrive()
	out := os.Getenv("AGAINROM_HOVER_PANEL_DIR")
	if out != "" {
		if err := os.MkdirAll(out, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	root := filepath.Base(os.Getenv("AGAINROM_ASSETS"))
	town := f.TownScreen().(*townScreen)
	for _, room := range []struct {
		name string
		room townRoom
	}{{"school", roomSchool}, {"tavern", roomTavern}} {
		town.room = room.room
		town.composeShopFaces()
		frame := func(book bool) []byte {
			town.shopBook = book
			pic, err := ui.ComposeTownScreen(town, "")
			if err != nil {
				t.Fatal(err)
			}
			if out != "" && book {
				if err := writeMediaFrame(out, room.name+"-book-on-"+root, pic); err != nil {
					t.Fatal(err)
				}
			}
			return pic.Pix
		}
		if !bytes.Equal(frame(false), frame(true)) {
			t.Errorf("%s: the switched-on book changed the frame", room.name)
		}
		town.shopBook = true
		v := town.TownSurface()
		if v.BookView != nil || !v.Hero.NoBook || v.Hero.BookOpen {
			t.Errorf("%s: book view %v, NoBook %v, BookOpen %v", room.name, v.BookView != nil, v.Hero.NoBook, v.Hero.BookOpen)
		}
		town.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlBook}, false)
		if !town.shopBook {
			t.Errorf("%s: the book control toggled the book", room.name)
		}
	}
	town.room, town.shopBook = roomShop, true
	town.composeShopFaces()
	if !town.ShopScreen().Book {
		t.Error("shop: the book is not shown with the book switched on")
	}
}
