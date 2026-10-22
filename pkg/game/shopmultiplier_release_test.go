package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

func TestReleaseShopStockMultiplierDefaultMatchesMain(t *testing.T) {
	f := shopOrderFront(t)
	h := sha256.New()
	for _, ceiling := range shelfTestCeilings(t, f) {
		for _, seed := range []int64{1, 7, 42} {
			s := NewShop(ceiling)
			s.Generate(f.Table, seed)
			explicit := NewShop(ceiling)
			explicit.stockMultiplier = 1
			explicit.Generate(f.Table, seed)
			assertShopStockMultiple(t, s, explicit, 1)
			if err := json.NewEncoder(h).Encode(s.shelves); err != nil {
				t.Fatal(err)
			}
		}
	}
	const mainDigest = "ea11aac9e58853f0e35aa680320977ed82e5e58d271e6e47382dffc8677c0496"
	if got := fmt.Sprintf("%x", h.Sum(nil)); got != mainDigest {
		t.Fatalf("default stock digest %s, want main %s", got, mainDigest)
	}
	t.Logf("main stock digest %s; %d ceilings, three seeds, four shelves", mainDigest, len(shelfTestCeilings(t, f)))
}

func TestReleaseShopStockMultiplierEveryShelf(t *testing.T) {
	f := shopOrderFront(t)
	for _, ceiling := range shelfTestCeilings(t, f) {
		for _, seed := range []int64{1, 7, 42} {
			base := NewShop(ceiling)
			base.Generate(f.Table, seed)
			for _, n := range []int32{5, MaxShopStockMultiplier} {
				got := NewShop(ceiling)
				got.stockMultiplier = n
				got.Generate(f.Table, seed)
				assertShopStockMultiple(t, base, got, n)
				for _, shelf := range got.shelves {
					for _, item := range shelf {
						if width := f.tipFont().Advance(ui.GroupDigits(int64(item.Count))) + 1; width > 69 {
							t.Fatalf("quantity %d needs %d pixels; cell allows 69", item.Count, width)
						}
					}
				}
			}
		}
	}
}

func TestReleaseShopStockMultiplierBuysEveryItem(t *testing.T) {
	f := currentTown(t, nil, nil)
	f.SoundPlayer, f.speech.player, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer = nil, nil, nil, nil, nil
	f.SetDeterministicFrames(true)
	base := cloneShopMutation(f.Shop)
	raw := currentTownSave(t, f)
	if err := f.SetShopStockMultiplier(1); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, currentTownSave(t, f)) {
		t.Fatal("explicit default changed SAV bytes")
	}
	if err := f.SetShopStockMultiplier(5); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, currentTownSave(t, f)) {
		t.Fatal("launch option entered SAV")
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err := store.WriteOriginal("", raw); err != nil {
		t.Fatal(err)
	}
	app := f.App("shop stock multiplier")
	app.Layout(640, 480)
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{}, nil))
	for _, action := range []string{"load game", "@first"} {
		if err := app.HeadlessActivate(action); err != nil {
			t.Fatal(err)
		}
	}
	assertShopStockMultiple(t, base, f.Shop, 5)
	s := shopOrderEnter(t, f, app)
	stock := shopOrderShelves(t, app, s, "multiplied stock")
	pointer := shopPointer(t, app)
	for room, rack := range shopRoomShelves {
		pointer("shelf_pick", room, "press", "release")
		var units int32
		for _, want := range stock[rack.shelf] {
			count := func() int {
				n := 0
				for _, item := range s.shopPackItemInstances() {
					if item.Code == uint16(want.Code) && item.Kind == want.Kind && item.Price == want.Price && reflect.DeepEqual(item.Effects, want.Effects) {
						n++
					}
				}
				return n
			}
			before := count()
			f.Town.gold = 1 << 30
			pointer("shelf", 0, "shift-press", "shift-release")
			if held := f.Shop.Table(); len(held) != 1 || !shopItemStateEqual(held[0].ShopItem, want) {
				t.Fatal("screen did not stage the complete stack", held, want)
			}
			pointer("button", 1, "press", "release")
			if len(f.Shop.Table()) != 0 || count()-before != int(want.Count) || f.Town.Gold() != (1<<30)-int(want.Price)*int(want.Count) {
				t.Fatal("screen did not buy the complete stack at its unit price", count()-before, want)
			}
			units += want.Count
		}
		if len(f.Shop.Shelf(rack.shelf)) != 0 {
			t.Fatal("the screen left items unreachable", rack.shelf)
		}
		t.Logf("shelf %s: N=5 bought all %d units in %d cells through App pointer controls", rack.shelf, units, len(stock[rack.shelf]))
	}
}
