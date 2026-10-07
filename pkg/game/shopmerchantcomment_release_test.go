package game

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/ui"
)

// releaseQuietShopApp loads a saved first-town game through App and enters
// the installed shop. Speech reaches a recorder; the sound, music, ambient and
// movie devices are removed before App is built, so nothing plays aloud.
func releaseQuietShopApp(t *testing.T) (*FrontEnd, *ui.App, *townScreen, *tavernInteriorRecorder) {
	t.Helper()
	f := releaseFront(t)
	r := &tavernInteriorRecorder{}
	f.SpeechPlayer = r
	f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer = nil, nil, nil, nil
	f.Carried = f.NextParty()
	f.arriveInTown()
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodeSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err := store.Write(time.Unix(200, 0), payload); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(200, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.ShopRandom = func(int) int { return 0 }
	app := f.App("release shop merchant comments")
	app.Layout(640, 480)
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
	for _, target := range []string{"load game", "@first", "SHOP"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	s := f.TownScreen().(*townScreen)
	for n := 0; s.room == roomTalk && n < 32; n++ {
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if s.room != roomShop || !s.shopInterior.ready {
		t.Fatalf("App entered room %d, interior ready %v; want the installed shop", s.room, s.shopInterior.ready)
	}
	closeShopTip(t, app, s)
	return f, app, s, r
}

// The installed shop through App input: the player's own goods (the pack,
// after a doll tap has added the hero's weapon) are hovered, clicked and
// dragged to the table, and the merchant is pressed with only those goods on
// the table; he starts no recording and arms no reaction. A click on a voiced
// armour stack starts that stack's installed recording; a later press on him
// starts and stops nothing (TOWN-478); a changed rack still arms Yes
// (SHOP-ANIMATION-084).
func TestReleaseShopMerchantDescribesOnlyHisSaleStock(t *testing.T) {
	f, app, s, r := releaseQuietShopApp(t)
	point := shopPointer(t, app)
	base := len(r.samples)
	if s.shopInterior.merchantModes != 0 {
		t.Fatalf("shop entry armed merchant modes %#x", s.shopInterior.merchantModes)
	}
	recorded := func(action string, want int) {
		t.Helper()
		if got := len(r.samples) - base; got != want {
			t.Fatalf("%s: merchant started %d recording(s) in the shop, want %d", action, got, want)
		}
		if s.shopInterior.merchantModes != 0 {
			t.Fatalf("%s armed merchant modes %#x", action, s.shopInterior.merchantModes)
		}
	}
	places := func(action string, want ...bool) {
		t.Helper()
		table := f.Shop.Table()
		if len(table) != len(want) {
			t.Fatalf("%s: table holds %d place(s), want %d", action, len(table), len(want))
		}
		for i, mine := range want {
			if table[i].Mine != mine {
				t.Fatalf("%s: table place %d Mine=%v, want %v", action, i, table[i].Mine, mine)
			}
		}
	}
	recording := func(code data.ItemCode) (string, audio.Sample, bool) {
		name := fmt.Sprintf("speech/shop/s%02di%02dp1.wav", code.B(), code.D())
		sample, ok := f.SpeechBank.Sample(name)
		return name, sample, ok
	}

	point("doll", 1, "press", "release")
	recorded("doll tap", 0)
	stacks := s.shopPackStacks()
	if len(stacks) == 0 {
		t.Fatal("the pack holds nothing after the doll tap")
	}
	goods := data.ItemCode(stacks[len(stacks)-1].Code)
	for _, st := range stacks {
		if _, _, ok := recording(data.ItemCode(st.Code)); ok {
			goods = data.ItemCode(st.Code)
			break
		}
	}
	goodsName, _, goodsVoiced := recording(goods)
	t.Logf("pack stacks %d; player's goods %#04x, %s installed: %v", len(stacks), uint16(goods), goodsName, goodsVoiced)
	packCell := func() int {
		t.Helper()
		for i, st := range s.shopPackStacks() {
			if data.ItemCode(st.Code) == goods {
				return i + 1 - s.packBase
			}
		}
		t.Fatalf("the pack no longer holds %#04x", uint16(goods))
		return -1
	}
	clickPack := func() {
		t.Helper()
		point("pack", packCell(), "press", "release")
	}

	point("shelf", 0, "hover")
	point("pack", packCell(), "hover")
	recorded("hover", 0)

	clickPack()
	places("pack click", true)
	recorded("pack click", 0)

	point("table", 0, "press", "release")
	places("table click")
	recorded("table click", 0)

	point("pack", packCell(), "press")
	point("table", 0, "move", "release")
	places("pack drag", true)
	recorded("pack drag", 0)

	line := app.HeadlessMessage()
	point("merchant", 0, "press", "release")
	recorded("merchant press over player goods", 0)
	if got := app.HeadlessMessage(); got != line {
		t.Fatalf("merchant press over player goods changed the line from %q to %q", line, got)
	}
	point("table", 0, "press", "release")
	places("table click back")

	shelf := f.Shop.Shelf(ShelfArmour)
	cell, name, want := -1, "", audio.Sample{}
	for i := range shelf {
		r := ui.ShopShelfCellRect(i)
		if c, ok := ui.ShopControlAt(r.Min.Add(r.Size().Div(2))); !ok || c.Kind != ui.ShopControlShelfCell || c.Index != i {
			break
		}
		if n, sample, ok := recording(shelf[i].Code); ok {
			cell, name, want = i, n, sample
			break
		}
	}
	if cell < 0 {
		t.Fatalf("no visible armour stack of %d has an installed description", len(shelf))
	}
	point("shelf", cell, "press", "release")
	places("shelf click", false)
	recorded("shelf click", 1)
	if got := r.samples[base]; !slices.Equal(got.PCM, want.PCM) || !r.voices[base].Playing() {
		t.Fatalf("shelf click did not play %s", name)
	}

	clickPack()
	places("pack click after sale stock", false, true)
	recorded("pack click after sale stock", 1)
	if !r.voices[base].Playing() {
		t.Fatal("pack click stopped the merchant's description")
	}

	point("merchant", 0, "press", "release")
	recorded("merchant press", 1)
	if !r.voices[base].Playing() {
		t.Fatalf("merchant press stopped %s", name)
	}

	point("shelf_pick", roomWeapons, "press", "release")
	if got := len(r.samples) - base; got != 1 || s.shopInterior.merchantModes != shopMerchantYes {
		t.Fatalf("changed rack: recordings %d, modes %#x; want 1 and Yes", got, s.shopInterior.merchantModes)
	}
	t.Logf("sale stack %s at armour shelf cell %d", name, cell)
}
