package game

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/mapload"
)

func TestReleaseShopClicksTransferOnTheReleaseFrame(t *testing.T) {
	path := os.Getenv("AGAINROM_SHOP_CLICK_INPUT")
	if path == "" {
		t.Skip("AGAINROM_SHOP_CLICK_INPUT is not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("source sha256=%x", sha256.Sum256(raw))
	f := shopOrderFront(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cityafter50.sav"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	app := openLocalTownSAV(t, f, dir, "cityafter50.sav")
	app.Layout(640, 480)
	t.Cleanup(app.StopAudio)
	store := SaveStore{Dir: dir}
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	s := shopOrderEnter(t, f, app)
	pointer := shopPointer(t, app)
	pointer("shelf_pick", 1, "press", "release")
	tap := func(surface string, index int) {
		t.Helper()
		x, y, err := app.HeadlessShopPoint(surface, index)
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		if err := app.HeadlessPointer("press", x, y); err != nil {
			t.Fatal(err)
		}
		press := time.Since(start)
		start = time.Now()
		if err := app.HeadlessPointer("release", x, y); err != nil {
			t.Fatal(err)
		}
		release := time.Since(start)
		t.Logf("%s%d press=%s release=%s", surface, index, press, release)
	}
	stock := f.Shop.Shelf(ShelfWeapons)
	if len(stock) == 0 || len(f.Shop.Table()) != 0 {
		t.Fatal("source lacks an empty table and weapon stock")
	}
	first := stock[0]
	tap("shelf", 0)
	if held := f.Shop.Table(); len(held) != 1 || held[0].Count != 1 || held[0].Code != first.Code || held[0].Mine || s.ShopScreen().Table[0].Count != 1 {
		t.Fatal("shelf release did not paint one merchant unit", held)
	}
	tap("table", 0)
	if len(f.Shop.Table()) != 0 || !shopOrderSame(f.Shop.Shelf(ShelfWeapons), stock) {
		t.Fatal("table release did not restore the merchant unit immediately")
	}
	tap("shelf", 0)
	tap("button", 1)
	if len(f.Shop.Table()) != 0 {
		t.Fatal("purchase did not clear the merchant table")
	}
	cell := -1
	for i, stack := range s.shopPackStacks() {
		if stack.Code == uint16(first.Code) {
			cell = i + 1 - s.packBase
			break
		}
	}
	if cell < 0 || cell >= len(s.ShopScreen().Pack) || s.ShopScreen().Pack[cell].UseItemKey == "" {
		t.Fatal("purchased equipment is not visible as a usable pack cell", cell)
	}
	member := s.shopPartyMember(s.shopMemberIndex())
	wantID, wantGold := member.ID, f.Town.Gold()
	want := mapload.MemberCarriedItems(*member, f.Table)
	tap("pack", cell)
	if held := f.Shop.Table(); len(held) != 1 || held[0].Count != 1 || held[0].Code != first.Code || !held[0].Mine || s.ShopScreen().Table[0].Count != 1 || len(mapload.MemberCarriedItems(*s.shopPartyMember(s.shopMemberIndex()), f.Table)) != len(want)-1 {
		t.Fatal("pack release deferred or lost the owned unit", held)
	}
	for range 28 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.Shop.Table()) != 1 || f.Shop.Table()[0].Count != 1 {
		t.Fatal("idle double-click expiry moved another unit")
	}
	saved := cityRosterF2Save(t, app, store, "responsive-shop")
	cold, coldApp, _ := cityPotionLoadApp(t, saved)
	coldApp.Layout(640, 480)
	if cold.Town.Gold() != wantGold || len(cold.Shop.Table()) != 0 {
		t.Fatal("cold SAVE/LOAD changed the purse or retained a transient trade")
	}
	found := false
	for _, m := range cold.Carried {
		if m.ID == wantID {
			found = true
			if !reflect.DeepEqual(mapload.MemberCarriedItems(m, cold.Table), want) {
				t.Fatal("cold SAVE/LOAD lost the immediately staged owned unit")
			}
		}
	}
	if !found {
		t.Fatal("cold SAVE/LOAD lost the member")
	}
	coldShop := shopOrderEnter(t, cold, coldApp)
	point := shopPointer(t, coldApp)
	point("pack", cell, "press", "release")
	if len(cold.Shop.Table()) != 1 || coldShop.ShopScreen().Table[0].Count != 1 {
		t.Fatal("the next cold App click was deferred")
	}
	if after, err := os.ReadFile(path); err != nil || !bytes.Equal(raw, after) {
		t.Fatal("owner source changed", err)
	}
}
