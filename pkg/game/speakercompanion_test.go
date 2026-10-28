package game

import (
	"image"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// companionHumansRow is a man fighter's Humans row at serverID with face.
func companionHumansRow(name string, serverID, face int32) dbEntry {
	p := humansParams(50, 0, face)
	p[16] = dressPaladinType
	p[18] = 0
	p[24] = serverID
	return dbEntry{name: name, params: p, strings: make([]string, 10)}
}

// companionSpeakerTable is a second-game table holding the hero's starting
// row, the row identity 22 is built from (serverID 10220, face 49) and an
// inn patron at his own serverID 2108 (face 46).
func companionSpeakerTable() *mapload.Table {
	humans := dbCollection{
		{},
		companionHumansRow("Start_MF", 10210, 32),
		companionHumansRow("Fighter_Hero", 10220, 49),
		companionHumansRow("2U_PeasantMale", 2108, 46),
	}
	return readUnder(base.GameROM2, &mapload.Table{
		Humans: humans, Armors: dressArmors(), Shapes: emptyScale{}, Materials: emptyScale{},
		NPCCatalog: &data.NPCCatalog{},
	})
}

// companionParty is the hero (face 1) and, with iglez, identity 22 (face 49).
func companionParty(iglez bool) []mapload.PartyMember {
	party := []mapload.PartyMember{{ID: "hero", PlayerCharacter: true, StartingHero: true, Class: 3,
		FigureDir: string(data.FigureDirManFighter), FigureFace: 1}}
	if iglez {
		party = append(party, mapload.PartyMember{ID: "npc:22", CompanionNPC: 22, PlayerCharacter: true, Class: 3,
			FigureDir: string(data.FigureDirManFighter), FigureFace: 49})
	}
	return party
}

func companionSheets() missionSource {
	return missionSource{
		"graphics/equipment/mfighter/1.256":  invBaseSheet(),
		"graphics/equipment/mfighter/46.256": invBaseSheet(),
		"graphics/equipment/mfighter/49.256": invBaseSheet(),
		"graphics/equipment/mmage/1.256":     invBaseSheet(),
		"graphics/equipment/fmage/20.256":    invBaseSheet(),
	}
}

// dinaSpeakerTable is companionSpeakerTable with the rows of key 23: the man
// mage whose own server id is 23 and Mage_Hero (server id 10230, face 20), the
// row identity 23 is built from.
func dinaSpeakerTable() *mapload.Table {
	t := companionSpeakerTable()
	mage := func(name string, serverID, face, gender int32) dbEntry {
		e := companionHumansRow(name, serverID, face)
		e.params[16], e.params[18] = 23, gender
		return e
	}
	t.Humans = append(t.Humans.(dbCollection),
		mage("ManMage_Unarmed", 23, 1, 0), mage("Mage_Hero", 10230, 20, 1))
	return t
}

// dinaParty is the hero, with iglez identity 22 (face 49) and with dina
// identity 23 (a woman mage, face 20), in the order the producer adds them.
func dinaParty(iglez, dina bool) []mapload.PartyMember {
	party := companionParty(iglez)
	if dina {
		party = append(party, mapload.PartyMember{ID: "npc:23", CompanionNPC: 23, PlayerCharacter: true, Class: 23,
			FigureDir: string(data.FigureDirWomanMage), FigureFace: 20})
	}
	return party
}

// dinaFigures is what each speaker of the second game draws: Dina is a hero
// woman mage, the table's row at key 23 a man mage who is not a hero.
var (
	dinaFigure      = figureID{Dir: data.FigureDirWomanMage, Face: 20, Hero: true}
	manMageFigure   = figureID{Dir: data.FigureDirManMage, Face: 1}
	iglezFigure     = figureID{Dir: data.FigureDirManFighter, Face: 49, Hero: true}
	heroFigure      = figureID{Dir: data.FigureDirManFighter, Face: 1, Hero: true}
	innPatronFigure = figureID{Dir: data.FigureDirManFighter, Face: 46}
)

// companionFace runs one SpeakerFace on a fresh cache and returns the figure
// it composed.
func companionFace(t *testing.T, r speakerResolver, speaker int) figureID {
	t.Helper()
	r.figurePics = map[figureCacheKey]*image.RGBA{}
	pic, _, ok := r.SpeakerFace(speaker)
	if !ok || pic == nil || len(r.figurePics) != 1 {
		t.Fatalf("speaker %d: ok %v, picture %v, %d figures composed; want his figure", speaker, ok, pic != nil, len(r.figurePics))
	}
	for key := range r.figurePics {
		return key.fig
	}
	return figureID{}
}

// In a second-game town talk a line of NPC 22 draws Iglez, the carried
// member of identity 22, and before he joins the figure of the Humans row he
// is built from; the hero's lines and an inn patron's keep their faces.
func TestSecondGameTownTalkDrawsTheCompanionIdentity(t *testing.T) {
	faces := speakerFaces(companionSpeakerTable())
	iglez := figureID{Dir: data.FigureDirManFighter, Face: 49, Hero: true}
	for _, tc := range []struct {
		name    string
		joined  bool
		speaker int
		want    figureID
	}{
		{"Iglez in the party", true, 22, iglez},
		{"Iglez before he joins", false, 22, iglez},
		{"the hero", true, 21, figureID{Dir: data.FigureDirManFighter, Face: 1, Hero: true}},
		{"the inn patron", true, 2108, figureID{Dir: data.FigureDirManFighter, Face: 46}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := speakerResolver{src: companionSheets(), npcFaces: faces, cast: newSpeakerCast(companionParty(tc.joined), nil)}
			if got := companionFace(t, r, tc.speaker); got != tc.want {
				t.Fatalf("speaker %d drew %+v, want %+v", tc.speaker, got, tc.want)
			}
		})
	}
}

// In a second-game mission a line of NPC 22 draws the live Iglez of the
// party, not the hero; the hero's own line still draws the hero.
func TestSecondGameMissionLineDrawsTheCompanionIdentity(t *testing.T) {
	party := companionParty(true)
	ids := []sim.EntityID{124, 125}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 124, X: 3, Y: 3, HP: 10, MaxHP: 10}, {ID: 125, X: 5, Y: 3, HP: 10, MaxHP: 10}},
		nil, sim.Relations{}, nil, []sim.Stock{{ID: 124}, {ID: 125}})
	if err != nil {
		t.Fatal(err)
	}
	mw := &mapWorld{
		world:         w,
		mission:       &missionNotices{src: companionSheets(), party: party},
		npcFaces:      speakerFaces(companionSpeakerTable()),
		speakerActors: missionSpeakers(nil, nil, nil, party, ids, nil),
	}
	for speaker, want := range map[int]figureID{
		22: {Dir: data.FigureDirManFighter, Face: 49, Hero: true},
		21: {Dir: data.FigureDirManFighter, Face: 1, Hero: true},
	} {
		mw.figurePics = map[figureCacheKey]*image.RGBA{}
		if _, _, ok := mw.SpeakerFace(speaker); !ok || len(mw.figurePics) != 1 {
			t.Fatalf("speaker %d: ok %v, %d figures composed", speaker, ok, len(mw.figurePics))
		}
		for key := range mw.figurePics {
			if key.fig != want {
				t.Errorf("speaker %d drew %+v, want %+v", speaker, key.fig, want)
			}
		}
	}
	// The first game's speaker table over the same rows is its registry's.
	first := companionSpeakerTable()
	edition := base.GameROM1.Edition()
	first.Edition = &edition
	if rec, held := speakerFaces(first)[22]; held {
		t.Fatalf("first game record 22 = %+v, want none", rec)
	}
}

type dinaCase struct {
	name    string
	iglez   bool
	dina    bool
	speaker int
	want    figureID
}

var dinaCases = []dinaCase{
	{"Dina in the party", true, true, 23, dinaFigure},
	{"Dina and no Iglez", false, true, 23, dinaFigure},
	{"no member of identity 23", true, false, 23, manMageFigure},
	{"the hero alone", false, false, 23, manMageFigure},
	{"Iglez with Dina", true, true, 22, iglezFigure},
	{"the hero with Dina", true, true, 21, heroFigure},
	{"the inn patron with Dina", true, true, 2108, innPatronFigure},
}

// In a second-game town talk a line of NPC 23 draws the carried member of
// identity 23, Dina, once she is in the party; with no such member it draws
// the Humans row whose own server id is 23, as before. The other speakers keep
// their faces.
func TestSecondGameTownTalkDrawsDina(t *testing.T) {
	faces := speakerFaces(dinaSpeakerTable())
	for _, tc := range dinaCases {
		t.Run(tc.name, func(t *testing.T) {
			r := speakerResolver{src: companionSheets(), npcFaces: faces, cast: newSpeakerCast(dinaParty(tc.iglez, tc.dina), nil)}
			if got := companionFace(t, r, tc.speaker); got != tc.want {
				t.Fatalf("speaker %d drew %+v, want %+v", tc.speaker, got, tc.want)
			}
		})
	}
}

// In a second-game mission the same lines draw from the mission's own cast.
func TestSecondGameMissionLineDrawsDina(t *testing.T) {
	faces := speakerFaces(dinaSpeakerTable())
	for _, tc := range dinaCases {
		t.Run(tc.name, func(t *testing.T) {
			party := dinaParty(tc.iglez, tc.dina)
			var ids []sim.EntityID
			var entities []sim.Entity
			var stocks []sim.Stock
			for i := range party {
				id := sim.EntityID(124 + i)
				ids = append(ids, id)
				entities = append(entities, sim.Entity{ID: id, X: int32(3 + 2*i), Y: 3, HP: 10, MaxHP: 10})
				stocks = append(stocks, sim.Stock{ID: id})
			}
			w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
				sim.ModeCanonical, sim.Terrain{}, entities, nil, sim.Relations{}, nil, stocks)
			if err != nil {
				t.Fatal(err)
			}
			mw := &mapWorld{
				world:         w,
				mission:       &missionNotices{src: companionSheets(), party: party},
				npcFaces:      faces,
				speakerActors: missionSpeakers(nil, nil, nil, party, ids, nil),
			}
			mw.figurePics = map[figureCacheKey]*image.RGBA{}
			if _, _, ok := mw.SpeakerFace(tc.speaker); !ok || len(mw.figurePics) != 1 {
				t.Fatalf("speaker %d: ok %v, %d figures composed", tc.speaker, ok, len(mw.figurePics))
			}
			for key := range mw.figurePics {
				if key.fig != tc.want {
					t.Errorf("speaker %d drew %+v, want %+v", tc.speaker, key.fig, tc.want)
				}
			}
		})
	}
}
