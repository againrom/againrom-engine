package game

import (
	"image"
	"reflect"
	"testing"
	"unsafe"

	"againrom/pkg/data"
	"againrom/pkg/formats/bmp"
	"againrom/pkg/render/text"
	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// This file is the guard round 2 of 1013's own adversarial review asked for
// (P finding): round 1 closed the world map's own two fields at every
// genuinely-different-game install site; round 2 found the SAME shape
// unclosed in room, conversation, shop presentation and the party-keyed
// composition caches, at the same three call sites the world-map-only reset
// did not reach on its own (that narrower reset was folded into
// resetForNewGame at the landing, so only one reset is now callable). TestResetForNewGameDropsExactlyTheGamePopulation
// enumerates townScreen's whole field set with reflect, so a field added to
// the struct later fails this test until it is classified as game state
// (reset) or install state (kept, with a reason) — the class cannot be
// reopened by a fourth field pair nobody thought to check by hand.

// settable returns an addressable, interfaceable reflect.Value for a struct
// field reflect would otherwise refuse to read or write, because reflect
// marks a field read-only whenever it is unexported — regardless of which
// package is doing the reflecting. Every field of townScreen is unexported,
// so this is required for both halves of the walk below (reading the seeded
// value back, and reading the post-reset value); it is the standard escape
// hatch for exactly this case and is used nowhere else in this package.
func settable(v reflect.Value) reflect.Value {
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

// nonZeroTownScreen returns a townScreen with every field set to a value
// that differs from that field's own Go zero value, built by direct field
// assignment (this file is package game, so every field is reachable by
// name) rather than through reflection — speakerResolver's own src field is
// an interface (entrySource), which reflect cannot synthesize generically,
// and does not need to: this test only requires resolver to come back
// UNCHANGED, not to hold any particular value.
func nonZeroTownScreen() *townScreen {
	ts := &townScreen{shopController: shopController{sess: &CampaignSession{}, in: &InstallResources{}, dealOf: func(*CampaignSession) *Shop { return nil }, bag: func(*CampaignSession) *[]sim.ItemInstance { return nil }, bagUnit: func(sim.ItemInstance) sim.ItemInstance { return sim.ItemInstance{} }, shop: shopHost{shown: func() bool { return true }}, racks: func() [4]shopRack { return shopRoomShelves }, packUnits: 9, value: func(sim.ItemInstance) int32 { return 0 }, label: func(sim.ItemInstance) []string { return nil }, shopChosen: 2, shopShelf: ShelfWeapons, shelfBase: 3, packBase: 4, tableBase: 6, shopMember: 5, shopBook: true, shopFigures: []*image.RGBA{{}}, shopFigureMasks: []*ui.SlotMask{{}}, shopSuppressSlot: 8, shopSuppressMember: 9, shopSuppressFigure: &image.RGBA{}, shopSuppressMask: &ui.SlotMask{}, shopIconCache: map[uint16]*image.RGBA{1: {}}, shopSpellAtlasImg: &bmp.Image{}, shopSpellAtlasTried: true, shopSpellIcons: map[uint16]*image.RGBA{1: {}}}, pc: &PersistenceContext{}, sound: &fakeTownAudio{}, draws: fakeTownDraws{}, art: fakeTownArt{}, townProcess: &town.Process{}, openMission: func(int) ui.MapOpener { return nil }, room: roomTalk, roomAudio: &ui.AudioScope{}, audioRoom: "school", dialogueBuilding: TownSchool, npc: 7, offer: TownOffer{Index: 1, Mission: 2, NPC: 3}, talkPage: talkCursor{payload: []byte("previous payload"), part: 4}, dialogue: newTownDialogue([]byte("previous payload")), mercenaryTalk: 6, talkDiagnostic: "a previous game's own diagnostic", innQueue: []int{3}, speech: townSpeech{key: "previous dialogue#1", voice: &tavernInteriorVoice{playing: true}, room: roomShop, pending: []string{"speech/shop/s01i02p2.wav"}}, tavernSelection: tavernCandidateKey{kind: tavernCandidateMercenary, id: 6}, tavernSlotRequests: map[int]int{tavernSlotHireRefused: 2}, schoolCell: 7, schoolHover: 2, schoolSpent: [schoolLatchCount]bool{3: true}, schoolSounds: playingSchoolSounds(), squareLoop: &tavernInteriorVoice{}, squareAction: ui.TownAction{Msg: "pending"}, tipRevision: 1, townStats: true, tavernDetailType: 6, tavernDetail: ui.TownCharacterView{HasSubject: true}, tavernDetailMask: &ui.SlotMask{}, tavernDetailInfo: [12][]string{{"item"}}, tavernDetailPixels: &image.RGBA{}, tavernDetailText: []text.DrawCall{{}}, shopTip: "a previous game's own tip", shopTipSecond: true, townTip: "a previous game's own town tip", schoolTip: "a previous game's own school tip", tavernTip: "a previous game's own tavern tip", tipClosed: [roomTalk + 1]bool{roomSquare: true}, worldMap: &worldMapState{}, worldPosition: image.Pt(10, 20), worldPositionSet: true, worldFlag1Frame: 4, worldSelectedOnce: map[int]bool{5: true}, worldSelectedOrder: []int{5}, resolver: speakerResolver{npcFaces: map[int32]data.NPCFace{1: {}}}}
	for range 5 {
		ts.talkWindow.Open(true)
	}
	ts.adapter = chapterTownCampaign{ts}
	ts.defaults = ROM1TownDescription()
	ts.composed = newTownSceneState(nil)
	ts.mapRoute = &chapterTownMap{ts}
	ts.mapArt = func() *worldMapAssets { return nil }
	ts.worldMapOf = newSecondCampaign()
	ts.worldMapAt = &secondMapOpening{from: partyPlace{kind: placeTown, id: 1}}
	ts.start = func() error { return nil }
	ts.frame = func() *ui.DialogFrame { return nil }
	ts.tags = firstEventTags{}
	ts.tipArt = func() *ui.TipPanelArt { return nil }
	ts.tip = secondRoomTip{room: "inn", text: "old", revision: 1}
	ts.talkFaces = speakerResolver{npcFaces: map[int32]data.NPCFace{1: {}}}
	ts.paneArt = func() (TownCharacterPaneArt, *ui.CharacterPaneCornerArt) { return TownCharacterPaneArt{}, nil }
	ts.talkSource = &chapterTownTalk{ts}
	ts.talkWindow.SetVisual(ui.DialogueButtonState{Hover: true, Pressed: true, Inside: true, Disabled: true}, ui.DialogueBackdrop{Clipped: true, Clip: image.Rect(1, 2, 3, 4)})
	ts.square = newSquareScene(ts, ROM1TownDescription())
	ts.tavernPage().Enter()
	return ts
}

// townScreenFields is the screen's fields, the embedded page's own fields in
// place of the page.
func townScreenFields(rv reflect.Value) (names []string, values []reflect.Value) {
	for i := 0; i < rv.NumField(); i++ {
		f := rv.Type().Field(i)
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			n, v := townScreenFields(rv.Field(i))
			names, values = append(names, n...), append(values, v...)
			continue
		}
		names, values = append(names, f.Name), append(values, rv.Field(i))
	}
	return names, values
}

// Every field is classified as reset state or retained process presentation.
func TestResetForNewGameDropsExactlyTheGamePopulation(t *testing.T) {
	keepUnchanged := map[string]string{
		"adapter":             "campaign state accessor reads the current session",
		"defaults":            "profile description remains process data",
		"composed":            "scene and art caches retain their process state",
		"mapRoute":            "map page reads the current host and campaign",
		"mapArt":              "install map art callback",
		"start":               "campaign start callback",
		"frame":               "install dialog art callback",
		"tags":                "campaign tag vocabulary",
		"tipArt":              "install tip art callback",
		"paneArt":             "install pane art callback",
		"sess":                "the campaign component the screen reads; does not change across a load",
		"in":                  "the install component the screen reads; does not change across a load",
		"sound":               "the audio service the screen plays through; does not change across a load",
		"draws":               "the presentation clock and draw service; does not change across a load",
		"art":                 "the first-use art service; does not change across a load",
		"townProcess":         "the town composer's process-scoped state; does not change across a load",
		"pc":                  "the persistence component the screen reads; does not change across a load",
		"openMission":         "the mission opener port; does not change across a load",
		"dealOf":              "the page's deal selector, bound with the screen",
		"bag":                 "the page's bag selector, bound with the screen",
		"bagUnit":             "the page's bag unit form, bound with the screen",
		"shop":                "the page's room hooks, bound with the screen",
		"racks":               "the page's racks, bound with the screen",
		"value":               "the page's pricer, bound with the screen",
		"packUnits":           "the page's pack bound, bound with the screen",
		"label":               "the page's item names, bound with the screen",
		"shopIconCache":       "item pictures keyed by item CODE, resolved from the install's own archives",
		"shopSpellAtlasImg":   "spell atlas resolved from the install's own archives",
		"shopSpellAtlasTried": "spell-atlas load attempt belongs to the install, not a game",
		"shopSpellIcons":      "spell pictures keyed by spell id, resolved from the install's own atlas",
		"shopTip":             "re-read from the install's own shop1.txt on every shop-room entry",
		"shopTipSecond":       "cleared on every shop-room entry with the popup it belongs to",

		"schoolTip":    "re-read from the install's own training.txt on every school entry",
		"tavernTip":    "re-read from the install's own inn.txt on every tavern entry",
		"talkSource":   "the screen itself, bound to its own talk window",
		"resolver":     "rebuilt by composeShopFaces on every entry into a room that can show it",
		"square":       "the square's composer view; resetForNewGame resets it in place",
		"pages":        "the room pages the composer builds; resetForNewGame resets each in place",
		"squareLoop":   "the square entry loop's voice, owned by the view's loop state",
		"squareAction": "set and cleared inside one square click",
	}

	ts := nonZeroTownScreen()
	names, fields := townScreenFields(reflect.ValueOf(ts).Elem())

	seeded := make(map[string]interface{}, len(names))
	for i, name := range names {
		fv := settable(fields[i])
		val := fv.Interface()
		seeded[name] = val
		if reflect.DeepEqual(val, reflect.Zero(fv.Type()).Interface()) {
			t.Fatalf("fixture field %s is still its own Go zero value; nonZeroTownScreen must "+
				"seed it non-zero, or this test cannot tell a real reset from a field it never "+
				"touched", name)
		}
	}

	voice := ts.speech.voice.(*tavernInteriorVoice)
	ts.resetForNewGame()
	if voice.playing || voice.stops != 1 {
		t.Fatal("new game retained the previous dialogue voice")
	}

	for i, name := range names {
		got := settable(fields[i]).Interface()
		if reason, kept := keepUnchanged[name]; kept {
			if host, ok := got.(shopHost); ok {
				if host.shown == nil {
					t.Errorf("field %s changed by resetForNewGame, but is kept (%s)", name, reason)
				}
				continue
			}
			if fields[i].Kind() == reflect.Func {
				if reflect.ValueOf(got).Pointer() != reflect.ValueOf(seeded[name]).Pointer() {
					t.Errorf("field %s changed by resetForNewGame, but is kept (%s)", name, reason)
				}
				continue
			}
			if !reflect.DeepEqual(got, seeded[name]) {
				t.Errorf("field %s changed by resetForNewGame, but is kept (%s): got %#v, want unchanged %#v",
					name, reason, got, seeded[name])
			}
			continue
		}
		// The two fields whose reset is a sentinel rather than Go's zero
		// value, because 0 is a real shelf and a real skill (see
		// resetForNewGame's own comment).
		want := reflect.Zero(fields[i].Type()).Interface()
		switch name {
		case "tipRevision":
			want = seeded[name].(uint64) + 1
		case "talkWindow":
			// The revision keeps counting; the shows and the pointer state go.
			before, after := seeded[name].(ui.TownTalk), got.(ui.TownTalk)
			if after.Revision() != before.Revision()+1 || after.ButtonState() != (ui.DialogueButtonState{}) ||
				after.Backdrop() != (ui.DialogueBackdrop{}) || after.Shows(ui.TownTalkView{Profile: firstTownTalkProfile, Open: true}) != 0 {
				t.Errorf("talkWindow after resetForNewGame: revision %d (was %d), button %+v, backdrop %+v", after.Revision(), before.Revision(), after.ButtonState(), after.Backdrop())
			}
			continue
		case "townTip":
			want = ""
		case "shopChosen":
			want = shopNoShelf
		case "schoolCell", "schoolHover":
			want = schoolNoSelection
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("field %s = %#v after resetForNewGame, want %#v — classify this field in "+
				"resetForNewGame (reset it) or in this test's keepUnchanged (name why it is the "+
				"install's, not the game's) if it is new", name, got, want)
		}
	}
}
