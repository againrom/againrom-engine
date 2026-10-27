package game

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// withShopTriggers replaces the shop room's trigger table of the shared
// description for one test.
func withShopTriggers(t *testing.T, d *town.Description, triggers []town.TriggerSpec) {
	t.Helper()
	withRoomTriggers(t, d, "shop", triggers)
}

// withRoomTriggers replaces one room's trigger table of the shared
// description for one test.
func withRoomTriggers(t *testing.T, d *town.Description, room string, triggers []town.TriggerSpec) {
	t.Helper()
	for i := range d.Rooms {
		if d.Rooms[i].Name != room {
			continue
		}
		sounds := &d.Rooms[i].Scene.Sounds
		saved := sounds.Triggers
		sounds.Triggers = triggers
		t.Cleanup(func() { sounds.Triggers = saved })
		return
	}
	t.Fatalf("the description has no %s room", room)
}

// roomSoundRecorder keeps every request a room sends, with its parameters,
// and answers a voice that plays until the test ends it.
type roomSoundRecorder struct {
	requests []audio.Request
	voices   []*exteriorVoice
}

func (r *roomSoundRecorder) Play(audio.Sample, audio.Placement) {}
func (r *roomSoundRecorder) RequestSample(_ audio.Sample, q audio.Request) audio.Voice {
	v := &exteriorVoice{playing: true}
	r.requests = append(r.requests, q)
	r.voices = append(r.voices, v)
	return v
}

// keys names the requests from index from on, as source:key.
func (r *roomSoundRecorder) keys(from int) []string {
	out := []string{}
	for _, q := range r.requests[from:] {
		out = append(out, q.Source+":"+q.Selector)
	}
	return out
}

func (r *roomSoundRecorder) endAll() {
	for _, v := range r.voices {
		v.playing = false
	}
}

// roomSoundKeys are the samples the census names for the town rooms, spelled
// from the claims' paths (SHOP-SOUND-134, TAVERN-028, TOWN-551).
var roomSoundKeys = []string{
	"put_on.wav", "put_off.wav", "scroll.wav", "town/shop/depart.wav", "town/shop/enter.wav", "town/shop/inshop.wav",
	"town/shop/start.wav", "town/shop/nofit.wav", "town/buy.wav", "town/sell.wav", "undo.wav", "out.wav",
	"town/inn/helper.wav", "add.wav", "noadd.wav", "talk.wav", "town/school/enter.wav", "town/school/teach.wav",
}

func addRoomSounds(f *FrontEnd) *roomSoundRecorder {
	rec := &roomSoundRecorder{}
	f.SoundPlayer, f.speech.player = rec, rec
	if f.SoundBank == nil {
		f.SoundBank = &SoundBank{cache: map[int]soundCacheEntry{}}
	}
	if f.SoundBank.named == nil {
		f.SoundBank.named = map[string]soundCacheEntry{}
	}
	for _, k := range roomSoundKeys {
		f.SoundBank.named[k] = soundCacheEntry{sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{1}}, ok: true}
	}
	return rec
}

// shopSoundFixture enters the shop through the application with the
// recorder already attached, so the entry requests are recorded.
func shopSoundFixture(t *testing.T) (*FrontEnd, *ui.App, *townScreen, *roomSoundRecorder) {
	t.Helper()
	f := shellFrontEnd()
	rec := addRoomSounds(f)
	f.Shop = NewShop(1000)
	f.Shop.shelves[ShelfArmour] = []ShopItem{{Code: 0x0102, Count: 3, Price: 100}}
	f.Carried[0].CarriedItems = []sim.ItemInstance{sim.PlainItem(0x0606), sim.PlainItem(0x0606)}
	f.Carried[0].Carried = []uint16{0x0606, 0x0606}
	f.shopArtCache = resolved(shopInteriorTestArt(), nil)
	now := time.Unix(200, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.ShopRandom = func(int) int { return 99 }
	a := f.App("shop room sounds")
	a.Layout(640, 480)
	a.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town.ags", Label: "Town"}} }, func(string) (ui.MapOpener, bool, error) {
		f.townUI.resetForNewGame()
		return nil, true, nil
	})
	if err := a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"@first", "SHOP"} {
		if err := a.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	s := f.townUI
	if s.room != roomShop {
		t.Fatalf("shop entry = room %d", s.room)
	}
	closeShopTip(t, a, s)
	return f, a, s, rec
}

func roomShopPointer(t *testing.T, a *ui.App, kind string, index int, edges ...string) {
	t.Helper()
	x, y, err := a.HeadlessShopPoint(kind, index)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range edges {
		if err := a.HeadlessPointer(e, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

// TestShopRoomRequestsTheCensusSounds drives the shop through the
// application's pointer: entry, the trade table's drop and pick, the
// buttons' presses and the trades, each request with its claim's
// parameters (SHOP-SOUND-135..137, VIDEO-SFX-099).
func TestShopRoomRequestsTheCensusSounds(t *testing.T) {
	f, a, s, rec := shopSoundFixture(t)
	entry := []string{"shop-interior:town/shop/depart.wav", "shop-interior:town/shop/enter.wav",
		"shop-interior:town/shop/inshop.wav", "shop-interior:town/shop/start.wav"}
	if got := rec.keys(0); len(got) < 4 || !reflect.DeepEqual(got[:4], entry) {
		t.Fatalf("entry requested %q, want %q first", got, entry)
	}
	for i, q := range rec.requests[:4] {
		wantGroup, wantLoop := audio.EffectsChannel, i == 2
		if i == 3 {
			wantGroup = audio.SpeechChannel
		}
		if q.Group != wantGroup || q.Repeat != wantLoop || q.Priority != 128 {
			t.Errorf("entry request %d: group %v repeat %v priority %d", i, q.Group, q.Repeat, q.Priority)
		}
	}
	rec.endAll()

	// A shelf item clicked onto the table: Put_On after it lands.
	base := len(rec.requests)
	roomShopPointer(t, a, "shelf", 0, "press", "release")
	if got := rec.keys(base); !reflect.DeepEqual(got, []string{"shop-interior:put_on.wav"}) {
		t.Fatalf("shelf to table requested %q, want Put_On", got)
	}
	// A second drop restarts the playing Put_On.
	first := rec.voices[len(rec.voices)-1]
	roomShopPointer(t, a, "shelf", 0, "press", "release")
	if first.stops != 1 || rec.keys(len(rec.requests) - 1)[0] != "shop-interior:put_on.wav" {
		t.Fatalf("a second drop stopped the first Put_On %d time(s)", first.stops)
	}
	// A pick from the table: Put_Off.
	base = len(rec.requests)
	roomShopPointer(t, a, "table", 0, "press", "release")
	if got := rec.keys(base); !reflect.DeepEqual(got, []string{"shop-interior:put_off.wav"}) {
		t.Fatalf("table pick requested %q, want Put_Off", got)
	}
	rec.endAll()

	// Button 0 presses Undo, and its clear picks each element.
	places := len(f.Shop.Table())
	base = len(rec.requests)
	roomShopPointer(t, a, "button", 0, "press")
	if got := rec.keys(base); !reflect.DeepEqual(got, []string{"shop-interior:undo.wav"}) {
		t.Fatalf("button 0 press requested %q, want Undo", got)
	}
	roomShopPointer(t, a, "button", 0, "release")
	if got := rec.keys(base + 1); len(got) != places || places == 0 || got[0] != "shop-interior:put_off.wav" {
		t.Fatalf("clearing %d place(s) requested %q", places, got)
	}
	rec.endAll()

	// Buy short of money: nofit; within the purse: buy.
	roomShopPointer(t, a, "shelf", 0, "press", "release")
	rec.endAll()
	f.Town.gold = 0
	base = len(rec.requests)
	roomShopPointer(t, a, "button", 1, "press", "release")
	if got := rec.keys(base); !reflect.DeepEqual(got, []string{"shop-interior:town/shop/nofit.wav"}) {
		t.Fatalf("short buy requested %q, want nofit", got)
	}
	f.Town.gold = 100000
	base = len(rec.requests)
	roomShopPointer(t, a, "button", 1, "press", "release")
	if got := rec.keys(base); !reflect.DeepEqual(got, []string{"shop-interior:town/buy.wav"}) {
		t.Fatalf("buy requested %q, want buy", got)
	}
	// An empty buy and an empty sale request nothing.
	base = len(rec.requests)
	roomShopPointer(t, a, "button", 1, "press", "release")
	roomShopPointer(t, a, "button", 2, "press", "release")
	if got := rec.keys(base); len(got) != 0 {
		t.Fatalf("an empty buy and sale requested %q", got)
	}
	// A pack item of no value put on the table: Put_On; its sale requests
	// nothing. The bought armour sold back: sell.
	base = len(rec.requests)
	roomShopPointer(t, a, "pack", 0, "press", "release")
	roomShopPointer(t, a, "button", 2, "press", "release")
	if got := rec.keys(base); !reflect.DeepEqual(got, []string{"shop-interior:put_on.wav"}) {
		t.Fatalf("a zero-value pack item and its sale requested %q, want Put_On only", got)
	}
	rec.endAll()
	roomShopPointer(t, a, "button", 0, "press", "release")
	rec.endAll()
	roomShopPointer(t, a, "pack", 1, "press", "release")
	if f.Shop.SellTotal() == 0 {
		t.Fatalf("the bought armour has no sale value; table %d place(s)", len(f.Shop.Table()))
	}
	rec.endAll()
	base = len(rec.requests)
	roomShopPointer(t, a, "button", 2, "press", "release")
	if got := rec.keys(base); !reflect.DeepEqual(got, []string{"shop-interior:town/sell.wav"}) {
		t.Fatalf("sale requested %q, want sell", got)
	}

	// Arrows scroll with a restarted Scroll; a shelf tab change departs.
	base = len(rec.requests)
	s.ShopPress(ui.ShopControl{Kind: ui.ShopControlArrowDown})
	s.ShopPress(ui.ShopControl{Kind: ui.ShopControlPackRight})
	roomShopPointer(t, a, "shelf_pick", 1, "press", "release")
	roomShopPointer(t, a, "shelf_pick", 1, "press", "release")
	want := []string{"shop-interior:scroll.wav", "shop-interior:scroll.wav", "shop-interior:town/shop/depart.wav"}
	if got := rec.keys(base); !reflect.DeepEqual(got, want) {
		t.Fatalf("arrows and tabs requested %q, want %q", got, want)
	}

	// Button 3 presses Out.
	rec.endAll()
	base = len(rec.requests)
	roomShopPointer(t, a, "button", 3, "press")
	if got := rec.keys(base); !reflect.DeepEqual(got, []string{"shop-interior:out.wav"}) {
		t.Fatalf("button 3 press requested %q, want Out", got)
	}
}

// TestShopRoomSoundRowsAreTheRequestSource removes the shop's sound table
// and shows the same gestures then request nothing: the table is the only
// source of these sounds.
func TestShopRoomSoundRowsAreTheRequestSource(t *testing.T) {
	_, a, s, rec := shopSoundFixture(t)
	withShopTriggers(t, s.townDescription(), nil)
	base := len(rec.requests)
	roomShopPointer(t, a, "shelf", 0, "press", "release")
	roomShopPointer(t, a, "table", 0, "press", "release")
	roomShopPointer(t, a, "button", 0, "press", "release")
	if got := rec.keys(base); len(got) != 0 {
		t.Fatalf("with no trigger rows the shop requested %q", got)
	}
}

// TestTavernAndSchoolRoomSounds: a roster cell press requests Helper, the
// leave press Out, a hire Add, a refused hire nofit, a dismissal NoAdd, a
// talk Talk; the school enters with Enter and leaves with Out (TAVERN-027,
// TAVERN-028, TOWN-551).
func TestTavernAndSchoolRoomSounds(t *testing.T) {
	f := shellFrontEnd()
	s := f.townUI
	rec := addRoomSounds(f)
	if s.room != roomTavern {
		t.Fatalf("shell room = %d, want the tavern", s.room)
	}
	cell := ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}
	s.TownSurfacePress(cell)
	s.TownSurfacePress(cell) // still playing: skipped
	if got := rec.keys(0); !reflect.DeepEqual(got, []string{"tavern-interior:town/inn/helper.wav"}) {
		t.Fatalf("roster press requested %q, want one Helper", got)
	}
	s.TownSurfaceClick(cell, false)
	rec.endAll()

	f.Town.mercPool[3], f.Town.gold = 2, 49
	base := len(rec.requests)
	s.townSurfaceButton(tavernButtonHire)
	f.Town.gold = 1000
	s.townSurfaceButton(tavernButtonHire)
	rec.endAll()
	s.townSurfaceButton(tavernButtonHire)
	want := []string{"tavern-interior:town/shop/nofit.wav", "tavern-interior:add.wav", "tavern-interior:noadd.wav"}
	if got := rec.keys(base); !reflect.DeepEqual(got, want) {
		t.Fatalf("hire, refusal and dismissal requested %q, want %q", got, want)
	}

	base = len(rec.requests)
	s.TownSurfacePress(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonHire})
	s.TownSurfacePress(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonExit})
	if got := rec.keys(base); !reflect.DeepEqual(got, []string{"tavern-interior:out.wav"}) {
		t.Fatalf("button presses requested %q, want Out on the leave button only", got)
	}
	rec.endAll()
	s.townSurfaceButton(tavernButtonTalk)
	if got := rec.keys(len(rec.requests) - 1); !strings.HasSuffix(got[0], ":talk.wav") {
		t.Fatalf("talk requested %q, want Talk", got)
	}

	// The school: the sound table's entry and leave press.
	base = len(rec.requests)
	s.schoolPage().Enter()
	s.room = roomSchool
	s.TownSurfacePress(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 1})
	s.roomSound("school", "teach")
	s.roomSound("school", "teach")
	want = []string{"school-training:town/school/enter.wav", "school-training:out.wav",
		"school-training:town/school/teach.wav", "school-training:town/school/teach.wav"}
	if got := rec.keys(base); !reflect.DeepEqual(got, want) {
		t.Fatalf("school requested %q, want %q", got, want)
	}
}

// TestTavernAndSchoolRoomSoundRowsAreTheRequestSource removes the tavern's
// and the school's trigger rows; the roster press, a hire and the leave
// presses then request nothing.
func TestTavernAndSchoolRoomSoundRowsAreTheRequestSource(t *testing.T) {
	f := shellFrontEnd()
	s := f.townUI
	rec := addRoomSounds(f)
	withRoomTriggers(t, s.townDescription(), "tavern", nil)
	withRoomTriggers(t, s.townDescription(), "school", nil)
	s.TownSurfacePress(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0})
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}, false)
	f.Town.mercPool[3], f.Town.gold = 2, 1000
	s.townSurfaceButton(tavernButtonHire)
	s.TownSurfacePress(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonExit})
	s.room = roomSchool
	s.TownSurfacePress(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 1})
	if got := rec.keys(0); len(got) != 0 {
		t.Fatalf("with no trigger rows the tavern and school requested %q", got)
	}
}

// TestRoomSoundTriggersValidate refuses a trigger on a slot the table does
// not hold.
func TestRoomSoundTriggersValidate(t *testing.T) {
	d := TownDescription(shellFrontEnd().townUI.profile())
	if err := d.Validate(); err != nil {
		t.Fatalf("the shipped description fails validation: %v", err)
	}
	withShopTriggers(t, d, []town.TriggerSpec{{On: "x", Slot: "missing"}})
	if err := d.Validate(); err == nil || !strings.Contains(err.Error(), "names no slot") {
		t.Fatalf("validate = %v, want a refusal of the missing slot", err)
	}
}

func TestShopRoomRequestsInShopAgainAfterEachPause(t *testing.T) {
	_, a, s, rec := shopSoundFixture(t)
	step := func(n int) {
		for i := 0; i < n; i++ {
			if err := a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
			if _, _, err := a.HeadlessFrame(); err != nil {
				t.Fatal(err)
			}
		}
	}
	inshop := func() (requests int, playing bool) {
		for i, q := range rec.requests {
			if q.Selector == "town/shop/inshop.wav" {
				requests++
				playing = rec.voices[i].playing
			}
		}
		return
	}
	step(5)
	if n, on := inshop(); n != 1 || !on {
		t.Fatalf("after entry InShop requests %d playing %v, want 1 and playing", n, on)
	}
	if err := a.HeadlessFocus(false); err != nil {
		t.Fatal(err)
	}
	step(3)
	if n, on := inshop(); n != 1 || on {
		t.Fatalf("while unfocused InShop requests %d playing %v, want 1 and stopped", n, on)
	}
	if err := a.HeadlessFocus(true); err != nil {
		t.Fatal(err)
	}
	step(20)
	if n, on := inshop(); n != 2 || !on {
		t.Fatalf("after focus returns InShop requests %d playing %v, want 2 and playing", n, on)
	}
	if err := a.HeadlessKey("f2"); err != nil {
		t.Fatal(err)
	}
	step(2)
	if n, on := inshop(); n != 2 || on {
		t.Fatalf("in the game menu InShop requests %d playing %v, want 2 and stopped", n, on)
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	step(5)
	if n, on := inshop(); n != 3 || !on || s.room != roomShop {
		t.Fatalf("after the game menu InShop requests %d playing %v room %d, want 3, playing, the shop", n, on, s.room)
	}
	step(50)
	if n, _ := inshop(); n != 3 {
		t.Fatalf("steady play made %d InShop requests, want 3", n)
	}
}
