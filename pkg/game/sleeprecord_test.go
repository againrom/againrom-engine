package game

import (
	"crypto/sha256"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/random"
	"againrom/pkg/ui"
)

// The Sleep record: each game's town on synthetic fixtures after zero, one,
// two and three Sleeps through the production controls, and a Sleep with no
// shop. Each line holds the Sleep control as shown, the shelves with their
// prices, the restock count, the shop seed, the next draw of the shop-stock
// stream and the digest of the town's SAV bytes. The record is compared with
// testdata/sleep_record.txt; AGAINROM_WRITE_SLEEP_RECORD=1 rewrites it.
func TestSleepRecord(t *testing.T) {
	var b strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	for n := 0; n <= 3; n++ {
		recordFirstSleeps(t, line, n, true)
	}
	recordFirstSleeps(t, line, 1, false)
	for _, id := range []int{1, 2, 3} {
		for n := 0; n <= 3; n++ {
			recordSecondSleeps(t, line, id, n, true)
		}
		recordSecondSleeps(t, line, id, 1, false)
	}
	compareSleepRecord(t, filepath.Join("testdata", "sleep_record.txt"), b.String())
}

func compareSleepRecord(t *testing.T, golden, got string) {
	t.Helper()
	if os.Getenv("AGAINROM_WRITE_SLEEP_RECORD") == "1" {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) == got {
		return
	}
	w, g := strings.Split(string(want), "\n"), strings.Split(got, "\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		var a, c string
		if i < len(w) {
			a = w[i]
		}
		if i < len(g) {
			c = g[i]
		}
		if a != c {
			t.Fatalf("Sleep record line %d:\n got %q\nwant %q", i+1, c, a)
		}
	}
}

// sleepSAV is the digest of the session's town SAV bytes, or the refusal.
func sleepSAV(f *FrontEnd) string {
	snap, _, err := f.Snapshot(false)
	if err != nil {
		return "snapshot refused: " + err.Error()
	}
	raw, err := f.ExportCurrentSave(snap, "Sleep")
	if err != nil {
		return "export refused: " + err.Error()
	}
	return fmt.Sprintf("%d bytes %x", len(raw), sha256.Sum256(raw))
}

func recordShelves(line func(string, ...any), key string, shelves [][]ShopItem) {
	for k, shelf := range shelves {
		for i, it := range shelf {
			line("%s shelf %d #%d of %d: code %#04x price %d units %d kind %d", key, k, i, len(shelf), uint16(it.Code), it.Price, it.Count, it.Kind)
		}
	}
}

// recordFirstSleeps is the first game's tavern after n presses of its
// Sleep button; without a shop the session holds none.
func recordFirstSleeps(t *testing.T, line func(string, ...any), n int, shop bool) {
	f := shellFrontEnd()
	table := shopTable()
	table.Humans, table.NPC = f.Table.Humans, f.Table.NPC
	f.Table = table
	f.Carried[0].ID, f.Carried[0].StartingHero = "hero", true
	key := fmt.Sprintf("first sleeps %d", n)
	if shop {
		f.Shop = NewShop(1000)
		f.Shop.Generate(f.Table, shopSeed(f.Town.Chapter(), f.Town.finishedCount(), f.Shop.Ceiling()))
	} else {
		f.Shop = nil
		key = "first no shop"
	}
	s := f.townUI
	b := s.TownSurface().Buttons[0]
	line("%s: button %q enabled %v rect %v", key, b.Label, b.Enabled, TownRoomButtonRect(ui.TownSurfaceTavern, 0))
	for i := 0; i < n; i++ {
		if act := s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 0}, false); act.Msg != "" || act.Open != nil || act.Info {
			line("%s: press %d acted %+v", key, i+1, act)
		}
	}
	if f.Shop != nil {
		line("%s: restocks %d seed %d table %d", key, f.Shop.restocks, f.Shop.drawSeed, len(f.Shop.table))
		var shelves [][]ShopItem
		for k := ShopShelf(0); k < shopShelfCount; k++ {
			shelves = append(shelves, f.Shop.Shelf(k))
		}
		recordShelves(line, key, shelves)
	} else {
		line("%s: shop absent", key)
	}
	line("%s: gold %d room %d sav %s", key, f.Town.Gold(), s.room, sleepSAV(f))
	line("%s: next stock draw %d", key, f.randomService().Stream(random.ShopStock).Raw())
}

// recordSecondSleeps is the second game's inn of town id after n choices
// of its Sleep row; without a shop the install has no item tables.
func recordSecondSleeps(t *testing.T, line func(string, ...any), id, n int, shop bool) {
	f, s := sleepTown(id)
	key := fmt.Sprintf("second town %d sleeps %d", id, n)
	if !shop {
		f.Table = nil
		key = fmt.Sprintf("second town %d no shop", id)
	}
	c := f.Town.second
	d := s.townDescription()
	if r := d.Room("inn"); r != nil && len(r.Buttons) > 0 {
		b := r.Buttons[0]
		s.composed = &townSceneState{seat: -1}
		at := b.Rect.Rectangle().Min.Add(image.Pt(5, 5))
		ctl, ok := (roomPageView{t: s, d: d, room: "inn"}).ControlAt(at)
		line("%s: plaque %q word %q rect %v click %v kind %d door %d", key, b.Name, b.Word, b.Rect.Rectangle(), ok, ctl.Kind, ctl.Door)
	}
	rows := s.Rows()
	line("%s: row 0 %q choosable %v of %d", key, rows[0].Text, rows[0].Choosable, len(rows))
	for i := 0; i < n; i++ {
		chooseSleep(t, s)
	}
	line("%s: filled %d table %d category %d", key, c.shop.filled, len(c.shop.table), c.shop.category)
	recordShelves(line, key, c.shop.shelves[:])
	line("%s: gold %d room %d sav %s", key, f.Town.gold, c.room, sleepSAV(f))
	line("%s: next stock draw %d", key, f.randomService().Stream(random.ShopStock).Raw())
}
