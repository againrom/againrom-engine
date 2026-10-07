package game

// The dialogue speaker's face: three kinds of answer over one table (0141).
//
// THE BINDING BEHIND IT WAS MEASURED, NOT ASSUMED, and the measurement lives in
// speakers.go's own header — 337 of 337 shipped `npc=` tags name a section of
// the scenario registry, against 53 of 337 for the placement reading the first
// draft of this file used. What is driven here is what this build DOES with a
// record once it has one.

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

func TestSpeakerFaceAnswersTheThreeKinds(t *testing.T) {
	// A figure base sheet and a flat portrait, both synthetic: synthBMP is
	// portrait_test.go's and invBaseSheet-style .256 fixtures are built the way
	// inventory_test.go builds them, through internal/synth.
	src := missionSource{
		"graphics/infowindow/Goblin.bmp":    synthBMP(4, 4, color.RGBA{G: 0xff, A: 0xff}),
		"graphics/equipment/ffighter/8.256": invBaseSheet(),
		"graphics/equipment/mmage/6.256":    invBaseSheet(),
	}

	mw := &mapWorld{
		mission: &missionNotices{src: src},
		units: &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{
			64: {Portrait: "Goblin"},
		}},
		portraits:  map[string]*image.RGBA{},
		figurePics: map[figureCacheKey]*image.RGBA{},
		npcFaces: map[int32]data.NPCFace{
			51:  {Kind: data.NPCFigure, Dir: data.FigureDirWomanFighter, Face: 8},
			52:  {Kind: data.NPCFigure, Dir: data.FigureDirManMage, Face: 6},
			103: {Kind: data.NPCPortrait, Class: 64},
			21:  {Kind: data.NPCNoPicture},
			99:  {Kind: data.NPCFigure, Dir: data.FigureDirManFighter, Face: 3}, // no sheet
		},
	}

	for _, tc := range []struct {
		name    string
		speaker int
		want    bool
	}{
		{"a woman fighter composes her own figure", 51, true},
		{"a man mage composes his", 52, true},
		{"a monster loads its class's portrait", 103, true},
		{"a figure whose sheet the install lacks answers none", 99, false},
		{"a number naming no record answers none", 4242, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pic, _, ok := mw.SpeakerFace(tc.speaker)
			if ok != tc.want {
				t.Fatalf("SpeakerFace(%d) ok = %v, want %v", tc.speaker, ok, tc.want)
			}
			if ok != (pic != nil) {
				t.Errorf("SpeakerFace(%d) answered ok=%v with picture=%v", tc.speaker, ok, pic != nil)
			}
		})
	}

	// The two figures are DIFFERENT pictures: one cache key per (figure,
	// equipment) pair, so a second speaker does not get the first one's doll.
	a, _, _ := mw.SpeakerFace(51)
	b, _, _ := mw.SpeakerFace(52)
	if a == b {
		t.Error("two speakers of different figures share one picture")
	}
}

// THE PLAYER'S OWN CHARACTER is the record carrying neither key, and it accounts
// for half the campaign's dialogue. Its picture is the one the inventory window
// already composed — worn layers and all — so the hero speaks wearing what he
// is wearing.
func TestSpeakerFaceAnswersTheHeroFromTheInventorySubject(t *testing.T) {
	hero := image.NewRGBA(image.Rect(0, 0, 4, 4))
	mw := &mapWorld{
		npcFaces:   map[int32]data.NPCFace{21: {Kind: data.NPCNoPicture}},
		figurePics: map[figureCacheKey]*image.RGBA{},
	}

	// With no party there is no such picture, and the pane stays empty.
	if pic, _, ok := mw.SpeakerFace(21); ok || pic != nil {
		t.Error("a mission with no subject answered a hero's face")
	}

	mw.invSubjectSet = true
	mw.invSubject = ui.InventorySubject{ID: 5, Figure: hero}
	pic, _, ok := mw.SpeakerFace(21)
	if !ok || pic != hero {
		t.Errorf("SpeakerFace answered %v (ok=%v), want the subject's own figure", pic, ok)
	}
}

func TestSpeakerResolverUsesTheAddedCompanionsOwnFigure(t *testing.T) {
	danath := image.NewRGBA(image.Rect(0, 0, 4, 4))
	reniesta := image.NewRGBA(image.Rect(0, 0, 5, 5))
	r := speakerResolver{
		npcFaces: map[int32]data.NPCFace{
			21: {Kind: data.NPCNoPicture},
			22: {Kind: data.NPCNoPicture},
		},
		playerFigure:  danath,
		playerFigures: map[int]*image.RGBA{22: reniesta},
	}
	if pic, _, ok := r.SpeakerFace(21); !ok || pic != danath {
		t.Fatalf("npc21 face = %v/%v, want Danath", pic, ok)
	}
	if pic, _, ok := r.SpeakerFace(22); !ok || pic != reniesta {
		t.Fatalf("npc22 face = %v/%v, want Reniesta's own figure", pic, ok)
	}
}

// A CARRIED COMPANION WHOSE RECORD IS AN `NPCFigure` IS DRAWN DRESSED
// (owner). npc25 is Brian's own join record and its shipped Kind is
// NPCFigure, so before this fix the town answered a figure composed with
// every equipment slot empty while the dressed figure for that same member
// was already sitting in playerFigures under that same key.
//
// THE TWO PICTURES ARE INDEPENDENT VALUES, not two calls to one composer: the
// bare one is what speakerFigure returns for the record's own Dir/Face with an
// empty equipment set, and the dressed one is handed in. The assertion is that
// the answer is the second and not the first.
func TestSpeakerFaceDrawsACarriedCompanionDressed(t *testing.T) {
	dressed := image.NewRGBA(image.Rect(0, 0, 7, 7))
	src := missionSource{"graphics/equipment/mfighter/1.256": invBaseSheet()}
	rec := data.NPCFace{Kind: data.NPCFigure, Dir: data.FigureDirManFighter, Face: 1}

	// The bare picture the record alone produces, composed through the same
	// resolver arm the fix bypasses.
	bareOnly := speakerResolver{src: src, npcFaces: map[int32]data.NPCFace{25: rec},
		figurePics: map[figureCacheKey]*image.RGBA{}}
	bare, _, ok := bareOnly.SpeakerFace(25)
	if !ok || bare == nil {
		t.Fatal("the record alone composed no figure; the fixture cannot tell the two arms apart")
	}

	r := speakerResolver{src: src, npcFaces: map[int32]data.NPCFace{25: rec},
		figurePics:    map[figureCacheKey]*image.RGBA{},
		playerFigures: map[int]*image.RGBA{25: dressed}}
	pic, _, ok := r.SpeakerFace(25)
	if !ok || pic != dressed {
		t.Errorf("npc25 face = %v (ok=%v), want the companion's own dressed figure %v", pic, ok, dressed)
	}
	if pic == bare {
		t.Error("npc25 answered the bare synthesised figure; the companion is talking naked")
	}

	// A speaker the party is NOT carrying still takes the record's bare arm.
	r2 := speakerResolver{src: src, npcFaces: map[int32]data.NPCFace{25: rec},
		figurePics:    map[figureCacheKey]*image.RGBA{},
		playerFigures: map[int]*image.RGBA{22: dressed}}
	if pic, _, ok := r2.SpeakerFace(25); !ok || pic == dressed {
		t.Errorf("a speaker no member carries answered %v (ok=%v), want the record's own bare figure", pic, ok)
	}
}

// THE RECORD'S OWN WINDOW AND ITS OWN TIER DIGIT BOTH REACH THE SEAM
// (`REG-NPC-088`, `REG-NPC-089`): the pane cuts a 72x96 rectangle at the
// speaker's `PortraitX1`/`PortraitY1`, and a creature's picture is its class's
// `InfoPicture` with the record's `Face` appended as the tier.
func TestSpeakerFaceCarriesTheWindowAndTheTier(t *testing.T) {
	green := color.RGBA{G: 0xff, A: 0xff}
	blue := color.RGBA{B: 0xff, A: 0xff}
	src := missionSource{
		// The class's first picture and its third: the address the record's own
		// Face builds, and the one a hardcoded tier 1 built instead.
		"graphics/infowindow/Goblin.bmp":    synthBMP(4, 4, green),
		"graphics/infowindow/Goblin3.bmp":   synthBMP(4, 4, blue),
		"graphics/equipment/ffighter/8.256": invBaseSheet(),
	}
	mw := &mapWorld{
		mission: &missionNotices{src: src},
		units: &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{
			64: {Portrait: "Goblin"},
		}},
		portraits:  map[string]*image.RGBA{},
		figurePics: map[figureCacheKey]*image.RGBA{},
		npcFaces: map[int32]data.NPCFace{
			// npc51's own shipped shape: a woman fighter stating a window.
			51: {Kind: data.NPCFigure, Dir: data.FigureDirWomanFighter, Face: 8,
				WindowX: 40, WindowY: 10, HasWindow: true},
			// npc102's: the third goblin, stating its class's own window.
			102: {Kind: data.NPCPortrait, Class: 64, Face: 3,
				WindowX: 30, WindowY: 72, HasWindow: true},
			// A record stating none — the ordinary case, 57 of the 105.
			56: {Kind: data.NPCFigure, Dir: data.FigureDirWomanFighter, Face: 8},
		},
	}

	if _, win, ok := mw.SpeakerFace(51); !ok || win != ui.NoticeFaceWindow(40, 10) {
		t.Errorf("npc51 answered window %v (ok=%v), want the record's own", win, ok)
	}
	if _, win, ok := mw.SpeakerFace(56); !ok || !win.Empty() {
		t.Errorf("a record stating no window answered %v (ok=%v), want the zero rectangle", win, ok)
	}

	// THE TIER: the third goblin's own picture, not the first. Both files are
	// in the fixture and they differ in one colour, so a reader still passing 1
	// answers green here.
	pic, win, ok := mw.SpeakerFace(102)
	if !ok || pic == nil {
		t.Fatal("npc102 answered no picture")
	}
	if got := pic.RGBAAt(pic.Bounds().Min.X, pic.Bounds().Min.Y); got != blue {
		t.Errorf("npc102 drew %v, want the third goblin %v — the tier digit is the record's own Face", got, blue)
	}
	if win != ui.NoticeFaceWindow(30, 72) {
		t.Errorf("npc102 answered window %v, want the record's own", win)
	}
}

// A driver with no table at all — every map opened before this story, and every
// one opened from the picker, which has no campaign registry behind it.
func TestSpeakerFaceIsTotalWithNoTable(t *testing.T) {
	mw := &mapWorld{}
	for _, n := range []int{0, 21, 51, 103, -1} {
		if pic, _, ok := mw.SpeakerFace(n); ok || pic != nil {
			t.Errorf("SpeakerFace(%d) answered a picture with no table", n)
		}
	}
}
