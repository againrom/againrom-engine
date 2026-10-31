package game

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// roomSoundRequest is one request a room scene makes: the event that raises
// it ("on <event>", "enter", "step" or "actor") and the key requested.
type roomSoundRequest struct{ event, key string }

// roomSoundRequests answers every request a room's scene names, each with
// the slot that serves it. The seat grid's selection slot serves the cell
// press, as the first game's cell-press trigger does.
func roomSoundRequests(t *testing.T, d *town.Description, room string) map[roomSoundRequest]town.SlotSpec {
	t.Helper()
	r := d.Room(room)
	if r == nil || r.Scene == nil {
		t.Fatalf("%s has no %s scene", d.Town, room)
	}
	slots := map[string]town.SlotSpec{}
	for _, sl := range r.Scene.Sounds.Slots {
		slots[sl.Name] = sl
	}
	out := map[roomSoundRequest]town.SlotSpec{}
	add := func(event, slot, key string) {
		sl, ok := slots[slot]
		if !ok {
			t.Fatalf("%s %s: %s requests slot %q, which it has not", d.Town, room, event, slot)
		}
		if key == "" {
			key = sl.Key
		}
		out[roomSoundRequest{event, key}] = sl
	}
	for _, tr := range r.Scene.Sounds.Triggers {
		add("on "+tr.On, tr.Slot, tr.Key)
	}
	if r.Seats != nil && r.Seats.Select != "" {
		add("on cell-press", r.Seats.Select, "")
	}
	for _, e := range r.Scene.Enter {
		if slot, ok := strings.CutPrefix(e, "sound "); ok {
			add("enter", slot, "")
		}
	}
	for _, g := range r.Scene.Steps {
		for _, e := range g.Run {
			if slot, ok := strings.CutPrefix(e, "sound "); ok {
				add("step", slot, "")
			}
		}
	}
	for _, slot := range actorSoundSlots(t, r.Scene.Actors) {
		add("actor", slot, "")
	}
	return out
}

// actorSoundSlots answers the slots the actors' programs name, in every
// field that names one: a state's start and end sounds, a frame cue's slot.
func actorSoundSlots(t *testing.T, actors []town.ActorSpec) []string {
	t.Helper()
	raw, err := json.Marshal(actors)
	if err != nil {
		t.Fatal(err)
	}
	var tree any
	if err := json.Unmarshal(raw, &tree); err != nil {
		t.Fatal(err)
	}
	var out []string
	var walk func(v any)
	walk = func(v any) {
		switch v := v.(type) {
		case []any:
			for _, e := range v {
				walk(e)
			}
		case map[string]any:
			for k, e := range v {
				switch s := e.(type) {
				case string:
					if s != "" && (k == "slot" || k == "first-sound" || k == "last-sound") {
						out = append(out, s)
					}
				case []any:
					if k == "start-sounds" || k == "end-sounds" {
						for _, n := range s {
							out = append(out, n.(string))
						}
						continue
					}
				}
				walk(e)
			}
		}
	}
	walk(tree)
	return out
}

// secondRoomSoundExceptions are the first game's room requests a second-game
// room makes no request for, by DIV-4372: the Kaarg inn's entry and water are
// Kin1 and Kvox5 and its taverner sounds its own (R2-ENGINE-687); the druid
// inn has no drink, steam or chair event.
var secondRoomSoundExceptions = map[string][]roomSoundRequest{
	"rom2-town2/inn": {
		{"enter", "town/inn/enter.wav"}, {"step", "town/inn/water.wav"}, {"step", "town/inn/steam.wav"},
		{"actor", "town/inn/drink.wav"}, {"actor", "town/inn/glotok.wav"}, {"actor", "town/inn/chair.wav"},
		{"actor", "town/shop/breath.wav"},
	},
	"rom2-town3/inn": {
		{"step", "town/inn/steam.wav"}, {"actor", "town/inn/drink.wav"}, {"actor", "town/inn/glotok.wav"},
		{"actor", "town/inn/chair.wav"}, {"actor", "town/shop/breath.wav"},
	},
}

// Each second-game shop and inn requests every sound the first game's shop
// and tavern request, on the same event, with the same key and a slot that
// plays it alike: loop, restart, tracking, priority and channel. A request
// it lacks is listed in DIV-4372's exceptions, and no exception is stale.
func TestSecondTownRoomsRequestTheFirstGameRoomSounds(t *testing.T) {
	first := ROM1TownDescription()
	for _, name := range []string{"rom2-town1", "rom2-town2", "rom2-town3"} {
		d := townDescriptions[name]
		if d == nil {
			t.Fatalf("no %s description", name)
		}
		for room, firstRoom := range map[string]string{"shop": "shop", "inn": "tavern"} {
			want, got := roomSoundRequests(t, first, firstRoom), roomSoundRequests(t, d, room)
			except := secondRoomSoundExceptions[name+"/"+room]
			for q, fs := range want {
				gs, ok := got[q]
				if slices.Contains(except, q) {
					if ok {
						t.Errorf("%s %s: the exception %v is stale, the room requests it", name, room, q)
					}
					continue
				}
				if !ok {
					t.Errorf("%s %s makes no request %v, which the first game's %s makes", name, room, q, firstRoom)
					continue
				}
				if gs.Loop != fs.Loop || gs.Restart != fs.Restart || gs.Untracked != fs.Untracked ||
					gs.Priority != fs.Priority || gs.Channel != fs.Channel {
					t.Errorf("%s %s %v plays as %+v, the first game's as %+v", name, room, q, gs, fs)
				}
			}
			for _, q := range except {
				if _, ok := want[q]; !ok {
					t.Errorf("%s %s: the exception %v is no first-game request", name, room, q)
				}
			}
		}
	}
}

// The Kaarg shop keeps its keeper's Kman4 cues and the Kaarg inn its Kin1
// entry beside the first game's sounds (R2-ENGINE-279, R2-ENGINE-687).
func TestSecondTownKaargKeepsItsOwnRoomSounds(t *testing.T) {
	d := townDescriptions["rom2-town2"]
	shop, inn := roomSoundRequests(t, d, "shop"), roomSoundRequests(t, d, "inn")
	if _, ok := shop[roomSoundRequest{"actor", "town_kaarg/shop/kman4.wav"}]; !ok {
		t.Error("the Kaarg shop lost its keeper's Kman4")
	}
	if _, ok := inn[roomSoundRequest{"enter", "town_kaarg/inn/kin1.wav"}]; !ok {
		t.Error("the Kaarg inn lost its Kin1 entry")
	}
}

// addSceneSounds gives the recorder's bank every key a room scene's slots
// name, so each request reaches the recorder.
func addSceneSounds(f *FrontEnd, d *town.Description, room string) {
	for _, sl := range d.Scene(room).Sounds.Slots {
		f.SoundBank.named[sl.Key] = soundCacheEntry{sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{1}}, ok: true}
	}
}

// requested answers whether the recorder holds a request for key from
// index from on, and ends every voice so the next request is not refused.
func requested(rec *roomSoundRecorder, from int, key string) bool {
	defer rec.endAll()
	for _, k := range rec.keys(from) {
		if strings.HasSuffix(k, ":"+key) {
			return true
		}
	}
	return false
}

// The second game's shop raises its controller's events on its own scene:
// a shelf pick on the table requests Put_On, the Undo press undo.wav and its
// clear Put_Off, an arrow press Scroll, a buy town/buy.wav, a sale
// town/sell.wav and the exit press out.wav, each from the shop's slots.
func TestSecondShopRequestsItsRoomSounds(t *testing.T) {
	f, _, screen := secondCampaignFixtureFiles(t, false)
	rec := addRoomSounds(f)
	c := secondShopSample()
	c.current, c.available, c.room = partyPlace{2, 1}, []partyPlace{{2, 1}}, secondTownShop
	f.Town.second, f.Town.gold = c, 300
	screen.bindBankShop()
	screen.shop.shown = nil
	screen.composed = newTownSceneState(nil)
	d, ok := screen.composedTown()
	if !ok || d.Town != "rom2-town1" {
		t.Fatal("no composed town 1")
	}
	addSceneSounds(f, d, "shop")
	screen.composed.art[d.Town+"/shop"] = &townRoomArt{scene: &town.Art{}}
	for _, step := range []struct {
		press bool
		kind  ui.ShopControlKind
		index int
		key   string
	}{
		{false, ui.ShopControlShelfCell, 0, "put_on.wav"},
		{true, ui.ShopControlButton, 0, "undo.wav"},
		{false, ui.ShopControlButton, 0, "put_off.wav"},
		{true, ui.ShopControlArrowDown, 0, "scroll.wav"},
		{true, ui.ShopControlPackRight, 0, "scroll.wav"},
		{false, ui.ShopControlShelfCell, 0, "put_on.wav"},
		{false, ui.ShopControlButton, 1, "town/buy.wav"},
		{false, ui.ShopControlPackCell, 1, "put_on.wav"},
		{false, ui.ShopControlButton, 2, "town/sell.wav"},
		{true, ui.ShopControlButton, 3, "out.wav"},
	} {
		from, c := len(rec.requests), ui.ShopControl{Kind: step.kind, Index: step.index}
		if step.press {
			screen.ShopPress(c)
		} else {
			screen.ShopClick(c)
		}
		if !requested(rec, from, step.key) {
			t.Errorf("press %v %v %d requested %v, want %s", step.press, step.kind, step.index, rec.keys(from), step.key)
		}
	}
}

// The second game's inn raises its room events on its own scene: Hire
// requests add.wav, Fire noadd.wav, a refused hire town/shop/nofit.wav, TALK
// talk.wav and the exit out.wav.
func TestSecondInnRequestsItsRoomSounds(t *testing.T) {
	f, app, screen := secondCampaignFixtureFiles(t, false)
	rec := addRoomSounds(f)
	c := innAt(2, 30, map[int]int32{532: 1, 552: 2, 533: 2, 553: 2, 513: 1})
	c.room = secondTownInn
	c.enterInn()
	f.Town.second = c
	app.SetTown(screen)
	d, ok := screen.composedTown()
	if !ok || d.Town != "rom2-town2" {
		t.Fatal("no composed town 2")
	}
	addSceneSounds(f, d, "inn")
	screen.composed.art[d.Town+"/inn"] = &townRoomArt{scene: kaargInnArt()}
	for _, step := range []struct {
		seat int
		key  string
	}{{0, "add.wav"}, {0, "noadd.wav"}, {1, "noadd.wav"}, {1, "town/shop/nofit.wav"}} {
		from := len(rec.requests)
		screen.innAct(c, step.seat)
		if !requested(rec, from, step.key) {
			t.Errorf("seat %d requested %v, want %s", step.seat, rec.keys(from), step.key)
		}
	}
	from := len(rec.requests)
	screen.Choose(len(screen.Rows()) - 1)
	if !requested(rec, from, "out.wav") {
		t.Errorf("the exit requested %v, want out.wav", rec.keys(from))
	}
	c.room = secondTownInn
	from = len(rec.requests)
	screen.innAct(c, 2)
	if !requested(rec, from, "talk.wav") {
		t.Errorf("TALK requested %v, want talk.wav", rec.keys(from))
	}
}
