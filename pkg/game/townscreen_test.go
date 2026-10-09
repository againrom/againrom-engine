package game

import (
	"image"
	"math/rand"
	"reflect"
	"testing"
	"unsafe"

	"againrom/pkg/data"
	"againrom/pkg/formats/bmp"
	"againrom/pkg/render/text"
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
	ts := &townScreen{
		sess:                &CampaignSession{},
		in:                  &InstallResources{},
		pc:                  &PersistenceContext{},
		sound:               &fakeTownAudio{},
		draws:               fakeTownDraws{},
		art:                 fakeTownArt{},
		townProcess:         &town.Process{},
		openMission:         func(int) ui.MapOpener { return nil },
		room:                roomTalk,
		roomAudio:           &ui.AudioScope{},
		audioRoom:           roomSchool,
		dialogueBuilding:    TownSchool,
		npc:                 7,
		offer:               TownOffer{Index: 1, Mission: 2, NPC: 3},
		said:                4,
		dialogueRevision:    5,
		dialogueButtonState: ui.DialogueButtonState{Hover: true, Pressed: true, Inside: true, Disabled: true},
		dialoguePolicy:      ui.DialogueBackdrop{Clipped: true, Clip: image.Rect(1, 2, 3, 4)},
		dialogue:            newTownDialogue([]byte("previous payload")),
		mercenaryTalk:       6,
		talkDiagnostic:      "a previous game's own diagnostic",
		innQueue:            []int{3},
		speech:              townSpeech{key: "previous dialogue#1", voice: &tavernInteriorVoice{playing: true}, room: roomShop, pending: []string{"speech/shop/s01i02p2.wav"}},
		shopChosen:          2,
		shopShelf:           ShelfWeapons,
		shelfBase:           3,
		packBase:            4,
		shopMember:          5,
		tavernSelection:     tavernCandidateKey{kind: tavernCandidateMercenary, id: 6},
		tavernSlotRequests:  map[int]int{tavernSlotHireRefused: 2},
		schoolCell:          7,
		schoolSpent:         [schoolLatchCount]bool{3: true},
		schoolSounds:        playingSchoolSounds(),
		squareRandom:        [2]*rand.Rand{rand.New(rand.NewSource(1)), rand.New(rand.NewSource(2))},
		squareLoop:          &tavernInteriorVoice{},
		squareAction:        ui.TownAction{Msg: "pending"},
		tipRevision:         1,
		pageRandom:          map[string]*rand.Rand{"tender": rand.New(rand.NewSource(3))},
		townStats:           true,
		shopBook:            true,
		tavernDetailType:    6,
		tavernDetail:        ui.TownCharacterView{HasSubject: true},
		tavernDetailMask:    &ui.SlotMask{},
		tavernDetailInfo:    [12][]string{{"item"}},
		tavernDetailPixels:  &image.RGBA{},
		tavernDetailText:    []text.DrawCall{{}},
		shopFigures:         []*image.RGBA{{}},
		shopFigureMasks:     []*ui.SlotMask{{}},
		shopSuppressSlot:    8,
		shopSuppressMember:  9,
		shopSuppressFigure:  &image.RGBA{},
		shopSuppressMask:    &ui.SlotMask{},
		shopIconCache:       map[uint16]*image.RGBA{1: {}},
		shopSpellAtlasImg:   &bmp.Image{},
		shopSpellAtlasTried: true,
		shopSpellIcons:      map[uint16]*image.RGBA{1: {}},
		shopTip:             "a previous game's own tip",
		townTip:             "a previous game's own town tip",
		schoolTip:           "a previous game's own school tip",
		tavernTip:           "a previous game's own tavern tip",
		tipClosed:           [roomTalk + 1]bool{roomSquare: true},
		worldMap:            &worldMapState{},
		worldPosition:       image.Pt(10, 20),
		worldPositionSet:    true,
		worldSelectedOnce:   map[int]bool{5: true},
		resolver:            speakerResolver{npcFaces: map[int32]data.NPCFace{1: {}}},
	}
	ts.square = town.NewView(rom1Town, townSquareHost{ts}, ts.townProcess)
	ts.tavernPage().Enter()
	return ts
}

// Every field is classified as reset state or retained process presentation.
func TestResetForNewGameDropsExactlyTheGamePopulation(t *testing.T) {
	keepUnchanged := map[string]string{
		"sess":                "the campaign component the screen reads; does not change across a load",
		"in":                  "the install component the screen reads; does not change across a load",
		"sound":               "the audio service the screen plays through; does not change across a load",
		"draws":               "the presentation clock and draw service; does not change across a load",
		"art":                 "the first-use art service; does not change across a load",
		"townProcess":         "the town composer's process-scoped state; does not change across a load",
		"pc":                  "the persistence component the screen reads; does not change across a load",
		"openMission":         "the mission opener port; does not change across a load",
		"shopIconCache":       "item pictures keyed by item CODE, resolved from the install's own archives",
		"shopSpellAtlasImg":   "spell atlas resolved from the install's own archives",
		"shopSpellAtlasTried": "spell-atlas load attempt belongs to the install, not a game",
		"shopSpellIcons":      "spell pictures keyed by spell id, resolved from the install's own atlas",
		"shopTip":             "re-read from the install's own shop1.txt on every shop-room entry",

		"schoolTip":    "re-read from the install's own training.txt on every school entry",
		"tavernTip":    "re-read from the install's own inn.txt on every tavern entry",
		"resolver":     "rebuilt by composeShopFaces on every entry into a room that can show it",
		"square":       "the square's composer view; resetForNewGame resets it in place",
		"pages":        "the room pages the composer builds; resetForNewGame resets each in place",
		"pageRandom":   "process presentation fallback generators of the room pages; never game state",
		"squareRandom": "process presentation fallback generators of the square; never game state",
		"squareLoop":   "the square entry loop's voice, owned by the view's loop state",
		"squareAction": "set and cleared inside one square click",
	}

	ts := nonZeroTownScreen()
	rv := reflect.ValueOf(ts).Elem()
	typ := rv.Type()

	seeded := make(map[string]interface{}, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		fv := settable(rv.Field(i))
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

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		name := field.Name
		got := settable(rv.Field(i)).Interface()
		if reason, kept := keepUnchanged[name]; kept {
			if rv.Field(i).Kind() == reflect.Func {
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
		want := reflect.Zero(field.Type).Interface()
		switch name {
		case "dialogueRevision", "tipRevision":
			want = seeded[name].(uint64) + 1
		case "townTip":
			want = ""
		case "shopChosen":
			want = shopNoShelf
		case "schoolCell":
			want = schoolNoSelection
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("field %s = %#v after resetForNewGame, want %#v — classify this field in "+
				"resetForNewGame (reset it) or in this test's keepUnchanged (name why it is the "+
				"install's, not the game's) if it is new", name, got, want)
		}
	}
}
