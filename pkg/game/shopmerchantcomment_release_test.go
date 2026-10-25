package game

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"againrom/internal/firstgame"
	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// releaseQuietShopApp loads a saved first-town game through App and enters
// the installed shop. Speech reaches a recorder; the sound, music, ambient and
// movie devices are removed before App is built, so nothing plays aloud.
func releaseQuietShopApp(t *testing.T) (*FrontEnd, *ui.App, *townScreen, *tavernInteriorRecorder) {
	t.Helper()
	f := releaseFront(t)
	r := &tavernInteriorRecorder{}
	f.speech.player = r
	f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer = nil, nil, nil, nil
	f.Carried = f.NextParty()
	arriveInTownGrantTaken(f)
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := f.ExportCurrentSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err := store.WriteOriginal("", payload); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(200, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.ShopRandom = func(int) int { return 0 }
	app := f.App("release shop merchant comments")
	app.Layout(640, 480)
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{}, nil))
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
	if s.room != roomShop || !s.shopPage().Ready() {
		t.Fatalf("App entered room %d, interior ready %v; want the installed shop", s.room, s.shopPage().Ready())
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
	firstgame.SkipOnSecondGame(t)
	f, app, s, r := releaseQuietShopApp(t)
	point := shopPointer(t, app)
	base := len(r.samples)
	if shopModes(s) != 0 {
		t.Fatalf("shop entry armed merchant modes %#x", shopModes(s))
	}
	recorded := func(action string, want int) {
		t.Helper()
		if got := len(r.samples) - base; got != want {
			t.Fatalf("%s: merchant started %d recording(s) in the shop, want %d", action, got, want)
		}
		if shopModes(s) != 0 {
			t.Fatalf("%s armed merchant modes %#x", action, shopModes(s))
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
		present := func(n string) bool { _, ok := f.SpeechBank.Sample(n); return ok }
		if n, ok := shopItemSpeech(shelf[i], present, func(int) int { return 0 }); ok {
			sample, _ := f.SpeechBank.Sample(n)
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
	if got := len(r.samples) - base; got != 1 || shopModes(s) != shopMerchantYes {
		t.Fatalf("changed rack: recordings %d, modes %#x; want 1 and Yes", got, shopModes(s))
	}
	t.Logf("sale stack %s at armour shelf cell %d", name, cell)
}

// shopSpeechDrain completes every recorded voice and steps App until no new
// recording starts, and returns how many started since base.
func shopSpeechDrain(t *testing.T, app *ui.App, r *tavernInteriorRecorder, base int) int {
	t.Helper()
	for n := 0; n < 8; n++ {
		for _, v := range r.voices {
			v.playing = false
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	return len(r.samples) - base
}

// Every installed item recording kind in the entered shop: each class/row
// key with parts, each spell description and each effect description is one
// selection, and each selection plays exactly one recording, the one
// shopItemSpeech names. Loss control: the request a selection made before,
// every class/row part queued at once, plays more than one recording for
// every key with two or more parts.
func TestReleaseShopMerchantSpeaksOneLinePerItem(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f, app, s, r := releaseQuietShopApp(t)
	shopSpeechDrain(t, app, r, 0)
	present := func(n string) bool { _, ok := f.SpeechBank.Sample(n); return ok }
	var items []ShopItem
	multi := 0
	for b := 1; b < 14; b++ {
		for d := 0; d < 100; d++ {
			code := data.ItemCode(b<<8 | d)
			var parts []string
			for part := 1; part <= 3; part++ {
				if n := fmt.Sprintf("speech/shop/s%02di%02dp%d.wav", b, d, part); present(n) {
					parts = append(parts, n)
				}
			}
			if len(parts) == 0 {
				continue
			}
			items = append(items, ShopItem{Code: code, Count: 1})
			if len(parts) > 1 {
				multi++
				base := len(r.samples)
				s.startTownResponse(parts)
				if got := shopSpeechDrain(t, app, r, base); got != len(parts) {
					t.Fatalf("loss control %#04x: the queued %d parts played %d", uint16(code), len(parts), got)
				}
			}
		}
	}
	books, effects := 0, 0
	for spell := 1; spell < 32; spell++ {
		if present(fmt.Sprintf("speech/shop/books/%02d.wav", spell)) {
			books++
			items = append(items, ShopItem{Code: 0x0e01, Kind: 5, Count: 1, Effects: []sim.ItemEffect{{Kind: 42, Operand: uint32(spell)}}})
		}
	}
	for kind := 1; kind < 41; kind++ {
		if present(fmt.Sprintf("speech/shop/effects/%02d.wav", kind)) {
			effects++
			items = append(items, ShopItem{Code: 0x0e01, Count: 1, Effects: []sim.ItemEffect{{Kind: uint8(kind), Operand: 1}}})
		}
	}
	keys := len(items) - books - effects
	if keys == 0 || books == 0 || effects == 0 || multi == 0 {
		t.Fatalf("installed shop recordings: %d class/row keys (%d with several parts), %d spells, %d effects", keys, multi, books, effects)
	}
	for draw := 0; draw < 3; draw++ {
		f.ShopRandom = func(int) int { return draw }
		for _, item := range items {
			want, ok := shopItemSpeech(item, present, f.ShopRandom)
			if !ok {
				t.Fatalf("%#04x names no recording", uint16(item.Code))
			}
			sample, _ := f.SpeechBank.Sample(want)
			base := len(r.samples)
			s.shopItemResponse(item)
			if got := shopSpeechDrain(t, app, r, base); got != 1 || !slices.Equal(r.samples[base].PCM, sample.PCM) {
				t.Fatalf("draw %d, %#04x: one selection played %d recording(s); want one, %s", draw, uint16(item.Code), got, want)
			}
		}
	}
	t.Logf("%d class/row keys (%d with several parts), %d spells, %d effects: one recording per selection under three draws", keys, multi, books, effects)
}

// 300 seeded placements of installed items speak exactly when the draw is below 30.
func TestReleaseShopMerchantSpeaksOnThirtyPercentDraw(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f, app, s, r := releaseQuietShopApp(t)
	shopSpeechDrain(t, app, r, 0)
	present := func(n string) bool { _, ok := f.SpeechBank.Sample(n); return ok }
	var items []ShopItem
	for b := 1; b < 14; b++ {
		for d := 0; d < 100; d++ {
			if present(fmt.Sprintf("speech/shop/s%02di%02dp1.wav", b, d)) {
				items = append(items, ShopItem{Code: data.ItemCode(b<<8 | d), Count: 1})
			}
		}
	}
	if len(items) == 0 {
		t.Fatal("the install holds no class/row recording")
	}
	state := uint32(12345)
	next := func() int {
		state = state*1103515245 + 12345
		return int(state>>16) & 0x7fff
	}
	var gate int
	f.ShopRandom = func(n int) int {
		v := next()
		if n == 100 {
			gate = v * 100 / 32767 % 100
			return gate
		}
		return v % n
	}
	asked, spoke := 0, 0
	for i := 0; i < 300; i++ {
		item := items[i%len(items)]
		base := len(r.samples)
		s.shopItemResponse(item)
		got := shopSpeechDrain(t, app, r, base)
		if (gate < 30) != (got == 1) || got > 1 {
			t.Fatalf("placement %d: draw %d, %d recording(s)", i, gate, got)
		}
		asked++
		if got == 1 {
			spoke++
		}
	}
	if spoke == 0 || spoke == asked {
		t.Fatalf("%d of %d placements spoke", spoke, asked)
	}
	t.Logf("%d items with recordings; %d of %d placements requested a recording", len(items), spoke, asked)
}
