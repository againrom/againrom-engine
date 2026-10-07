package game

// THE DIALOGUE SPEAKER WEARS HIS OWN EQUIPMENT (0160).
//
// The witness throughout is the COMPOSITION INPUT, read off the composed-figure
// cache key: the key carries the figure and the worn set the composer was
// called with, so "this speaker was composed from these item codes" is a fact a
// test can read without a single decoded pixel. That is deliberate. The defect
// this story fixes was invisible in the return value, which was a perfectly
// good picture of a naked man.

import (
	"image"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The fixture's two dressed rows. The plate helm's own Armors row states slot
// 6, the head slot (`DLG-FIGURE-021`), and the robe's states slot 7 — so the
// two rows below are told apart by WHERE their piece lands, which is the row's
// own Slot column and never a cell index.
const (
	dressHelmSlot = 6
	dressRobeSlot = 7

	// The paladin's server id, the value mission 40's `npc25` states as its
	// `DataBinID` (`DLG-DRESS-024`), and the server id of a row that states no
	// equipment cell at all.
	dressPaladinServerID = 42
	dressBareServerID    = 43

	// The row's drawn class id and face column: 3 is the swordsman appearance
	// name chain's own id, outside the mage band, so the row draws a man
	// fighter.
	dressPaladinType = 3
	dressPaladinFace = 1
)

// dressArmors is the fixture's Armors collection: one piece per slot the tests
// below read, each stating its slot in its own row exactly as a shipped row
// does (armorSlotColumn = 4).
func dressArmors() dbCollection {
	return dbCollection{
		{}, // 0: the reserved entry
		{name: "PlateHelm", params: []int32{-1, -1, -1, -1, dressHelmSlot}},
		{name: "Robe", params: []int32{-1, -1, -1, -1, dressRobeSlot}},
	}
}

// dressHumansRow is one Humans row wearing the named armour in its first armour
// cell, with a server id and the three art columns FigureFor reads.
func dressHumansRow(name, armour string, serverID int32) dbEntry {
	p := humansParams(50, 0, dressPaladinFace)
	p[16] = dressPaladinType
	p[18] = 0 // the gender cell: a man
	p[24] = serverID
	cells := make([]string, 10)
	cells[2] = armour
	return dbEntry{name: name, params: p, strings: cells}
}

// dressTable is the definition table the dressing tests resolve against: an
// Armors collection, a Humans collection holding the paladin at his own server
// id and the four archetype rows data.ChargenBase falls back to, and an NPC
// lookup naming the paladin's row from subscript 25.
//
// THE ARCHETYPE ROWS ARE DRESSED TOO, which is the whole of the fallback arm:
// a section naming no definition id — 82 of the 105 shipped — is dressed from
// the row its own `Mage` and `Female` flags choose.
func dressTable() *mapload.Table {
	humans := dbCollection{
		{}, // 0: the reserved entry
		dressHumansRow("PC_Danath", "Robe", 1),
		dressHumansRow("PC_Naira", "Robe", 2),
		dressHumansRow("PC_Fergard", "Robe", 3),
		dressHumansRow("PC_Reniesta", "Robe", 4),
		dressHumansRow("PC_Paladin", "PlateHelm", dressPaladinServerID),
		// A named row wearing NOTHING — seven shipped rows are exactly this,
		// `M10_Witch` and `M71_PoorGuy` among them.
		dressHumansRow("M10_Witch", "", dressBareServerID),
	}
	return &mapload.Table{
		Humans: humans, Armors: dressArmors(),
		Shapes: emptyScale{}, Materials: emptyScale{},
		NPC: data.LoadNPCDefs(dressNPCReg()),
	}
}

// dressNPCReg is the scenario NPC registry the fixture resolves against: one
// section naming the paladin's own definition id, exactly as mission 40's
// `npc25` does (`DLG-DRESS-024`). A registry this build cannot parse is not a
// case this fixture can reach, so the parse failure panics rather than being
// carried into a test as a silent empty lookup.
func dressNPCReg() *reg.Reg {
	r, err := reg.Parse(synth.Reg(kindRoot, []synth.RegNode{
		{Name: "npc25", Kind: kindDir, Children: []synth.RegNode{
			{Name: "Flags", Kind: 0x00, Str: "Hero,Face,!Female,!Mage"},
			{Name: "DataBinID", Kind: kindInt, Int: dressPaladinServerID},
		}},
		{Name: "npc51", Kind: kindDir, Children: []synth.RegNode{
			{Name: "DataBinID", Kind: kindInt, Int: dressBareServerID},
		}},
	}))
	if err != nil {
		panic("speakerdress fixture: " + err.Error())
	}
	return r
}

// dressArmorCode is the code the fixture's own resolution gives one armour
// name. It is asked of data.ResolveArmor rather than written down, so the test
// compares against the resolution the loader itself performs.
func dressArmorCode(t *testing.T, name string) uint16 {
	t.Helper()
	a, err := data.ResolveArmor(name, emptyScale{}, emptyScale{}, dressArmors())
	if err != nil {
		t.Fatalf("setup: ResolveArmor(%q): %v", name, err)
	}
	return uint16(a.Code)
}

// dressWorld is one entity standing on a small map wearing worn.
func dressWorld(t *testing.T, id sim.EntityID, worn [sim.EquipSlots]uint16) *sim.World {
	t.Helper()
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: id, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: id, Equipped: worn}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	return w
}

// dressSheets is a figure base sheet for each directory the tests below name,
// so the arm that ran also returns a picture and not only a cache key.
func dressSheets() missionSource {
	return missionSource{
		"graphics/equipment/mfighter/1.256": invBaseSheet(),
		"graphics/equipment/mmage/1.256":    invBaseSheet(),
		"graphics/equipment/ffighter/1.256": invBaseSheet(),
	}
}

// composeKey runs one SpeakerFace and returns the key it composed under. The
// cache is emptied first, so exactly one key can be attributed to the call.
func composeKey(t *testing.T, mw *mapWorld, speaker int) figureCacheKey {
	t.Helper()
	mw.figurePics = map[figureCacheKey]*image.RGBA{}
	mw.SpeakerFace(speaker)
	if len(mw.figurePics) != 1 {
		t.Fatalf("SpeakerFace(%d) composed %d figures, want exactly 1", speaker, len(mw.figurePics))
	}
	for key := range mw.figurePics {
		return key
	}
	return figureCacheKey{}
}

// npcTokenSet is the test's own spelling of a record's token set.
func npcTokenSet(list ...data.NPCToken) data.NPCTokens {
	var out data.NPCTokens
	for _, tok := range list {
		out |= data.NPCTokens(tok)
	}
	return out
}

func TestDialogueSpeakerComposesTheLiveActorsWornSet(t *testing.T) {
	helm := dressArmorCode(t, "PlateHelm")
	var worn [sim.EquipSlots]uint16
	worn[dressHelmSlot-1] = helm

	mw := &mapWorld{
		world:   dressWorld(t, 0, worn),
		mission: &missionNotices{src: dressSheets()},
		npcFaces: map[int32]data.NPCFace{
			25: {Kind: data.NPCFigure, Dir: data.FigureDirManFighter, Face: 1,
				Tokens: npcTokenSet(data.NPCTokenHero)},
		},
		speakerActors: []speakerActor{
			{id: 0, hero: true, fig: figureID{Dir: data.FigureDirManFighter, Face: 1},
				face: dressPaladinFace, typeID: dressPaladinType},
		},
	}

	key := composeKey(t, mw, 25)
	if occupied, _ := key.eq.Occupied(dressHelmSlot); !occupied {
		t.Errorf("the speaker was composed from %+v, head slot empty — "+
			"`DLG-FIGURE-021`: no layer is excluded at the dialogue site", key.eq)
	}
	if code, _ := key.eq.Code(dressHelmSlot); code != data.ItemCode(helm) {
		t.Errorf("head slot composed with code %#x, want the helm %#x", code, helm)
	}

	// AC-8: the same speaker, now bare-headed, is a different composition.
	mw.world = dressWorld(t, 0, [sim.EquipSlots]uint16{})
	next := composeKey(t, mw, 25)
	if next == key {
		t.Error("removing the helm produced the same cache key; the worn set is not reaching the composer")
	}
	if occupied, _ := next.eq.Occupied(dressHelmSlot); occupied {
		t.Errorf("the head slot is still composed after the helm came off: %+v", next.eq)
	}
}

func TestDialogueSpeakerPredicateNarrowsByToken(t *testing.T) {
	actors := []speakerActor{
		{id: 0, hero: true, fig: figureID{Dir: data.FigureDirManFighter, Face: 1}, face: 1, typeID: 3},
		{id: 1, fig: figureID{Dir: data.FigureDirManMage, Face: 1}, face: 4, typeID: 24},
		{id: 2, fig: figureID{Dir: data.FigureDirWomanFighter, Face: 1}, face: 8, typeID: 14},
		{id: 3, me: true, hero: true, fig: figureID{Dir: data.FigureDirManFighter, Face: 5}, face: 5, typeID: 3},
	}
	cast := speakerCast{actors: actors, alive: func(sim.EntityID) bool { return true },
		playerDir: data.FigureDirManFighter, hasPlayer: true}

	for _, tc := range []struct {
		name string
		rec  data.NPCFace
		want sim.EntityID
		none bool
	}{
		{name: "no token at all takes the first candidate", want: 0},
		{name: "Hero takes the first, every candidate being a placed person",
			rec: data.NPCFace{Tokens: npcTokenSet(data.NPCTokenHero)}, want: 0},
		{name: "!Hero skips the hero candidate",
			rec: data.NPCFace{Tokens: npcTokenSet(data.NPCTokenNotHero)}, want: 1},
		{name: "Mage skips the fighter",
			rec: data.NPCFace{Tokens: npcTokenSet(data.NPCTokenMage)}, want: 1},
		{name: "Female skips both men",
			rec: data.NPCFace{Tokens: npcTokenSet(data.NPCTokenFemale)}, want: 2},
		{name: "!Mage with Female is the woman fighter",
			rec: data.NPCFace{Tokens: npcTokenSet(data.NPCTokenNotMage, data.NPCTokenFemale)}, want: 2},
		{name: "Face compares the record's own Face against the row's",
			rec: data.NPCFace{Face: 8, Tokens: npcTokenSet(data.NPCTokenFace)}, want: 2},
		{name: "Face naming nobody's face resolves no speaker",
			rec: data.NPCFace{Face: 9, Tokens: npcTokenSet(data.NPCTokenFace)}, none: true},
		{name: "Picture compares against the row's own type id",
			rec: data.NPCFace{Class: 24, HasClass: true, Tokens: npcTokenSet(data.NPCTokenPicture)}, want: 1},
		{name: "Picture with no key at all resolves nobody",
			rec: data.NPCFace{Tokens: npcTokenSet(data.NPCTokenPicture)}, none: true},
		{name: "MySex over a male player skips the woman",
			rec: data.NPCFace{Tokens: npcTokenSet(data.NPCTokenMySex, data.NPCTokenMage)}, want: 1},
		{name: "!MyClass over a fighter player takes the mage",
			rec: data.NPCFace{Tokens: npcTokenSet(data.NPCTokenNotMyClass)}, want: 1},
		{name: "Me selects the primary party actor",
			rec: data.NPCFace{Tokens: npcTokenSet(data.NPCTokenMe)}, want: 3},
		{name: "!Me excludes the primary party actor",
			rec: data.NPCFace{Face: 5, Tokens: npcTokenSet(data.NPCTokenFace, data.NPCTokenNotMe)}, none: true},
		{name: "!Human matches no placement",
			rec: data.NPCFace{Tokens: npcTokenSet(data.NPCTokenNotHuman)}, none: true},
		{name: "Platoon narrows nothing",
			rec: data.NPCFace{Tokens: npcTokenSet(data.NPCTokenPlatoon)}, want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := cast.resolve(tc.rec)
			if tc.none {
				if ok {
					t.Fatalf("resolved entity %d, want no speaker at all", got.id)
				}
				return
			}
			if !ok {
				t.Fatalf("resolved no speaker, want entity %d", tc.want)
			}
			if got.id != tc.want {
				t.Errorf("resolved entity %d, want %d", got.id, tc.want)
			}
		})
	}

	// MySex and MyClass over a driver holding no player character reject
	// rather than guess.
	blind := speakerCast{actors: actors, alive: func(sim.EntityID) bool { return true }}
	if _, ok := blind.resolve(data.NPCFace{Tokens: npcTokenSet(data.NPCTokenMySex)}); ok {
		t.Error("MySex resolved a speaker with no player character to compare against")
	}
}

// The live client list does not end at the map's placements. A companion that
// crossed a mission boundary is represented by the party and its start id; Me
// is the explicit StartingHero, not whichever member happens to be first after
// a transition.
func TestMissionSpeakersIncludesThePartyWithStableIdentity(t *testing.T) {
	party := []mapload.PartyMember{
		{ID: "npc:25", PlayerCharacter: true, Class: 42,
			FigureDir: string(data.FigureDirManFighter), FigureFace: 1},
		{ID: "hero", PlayerCharacter: true, StartingHero: true, Class: 3,
			FigureDir: string(data.FigureDirManFighter), FigureFace: 5},
	}
	actors := missionSpeakers(nil, nil, nil, party, []sim.EntityID{40, 41}, nil)
	if len(actors) != 2 {
		t.Fatalf("mission speaker count = %d, want two party actors", len(actors))
	}
	if actors[0].id != 40 || actors[0].me || actors[0].face != 1 || actors[0].typeID != 42 {
		t.Errorf("companion candidate = %+v, want id40/non-Me/face1/type42", actors[0])
	}
	if actors[1].id != 41 || !actors[1].me || actors[1].face != 5 || actors[1].typeID != 3 {
		t.Errorf("primary candidate = %+v, want id41/Me/face5/type3", actors[1])
	}

	cast := speakerCast{actors: actors, alive: func(sim.EntityID) bool { return true }}
	if got, ok := cast.resolve(data.NPCFace{Tokens: npcTokenSet(data.NPCTokenMe)}); !ok || got.id != 41 {
		t.Fatalf("Me resolved %+v/%v, want primary id41", got, ok)
	}
	if got, ok := cast.resolve(data.NPCFace{Face: 1,
		Tokens: npcTokenSet(data.NPCTokenFace, data.NPCTokenNotMe)}); !ok || got.id != 40 {
		t.Fatalf("carried companion resolved %+v/%v, want id40", got, ok)
	}
}

// AC-5: a dead candidate is not the speaker, and a live one later in id order
// is. `DLG-SPEAKER-023`'s state gate, in the terms this tree has.
func TestDialogueSpeakerSkipsTheDead(t *testing.T) {
	actors := []speakerActor{
		{id: 0, hero: true, fig: figureID{Dir: data.FigureDirManFighter, Face: 1}, face: 1},
		{id: 1, hero: true, fig: figureID{Dir: data.FigureDirManFighter, Face: 3}, face: 1},
	}
	cast := speakerCast{actors: actors, alive: func(id sim.EntityID) bool { return id != 0 }}
	got, ok := cast.resolve(data.NPCFace{Tokens: npcTokenSet(data.NPCTokenHero)})
	if !ok || got.id != 1 {
		t.Fatalf("resolved %d/%v, want the living entity 1", got.id, ok)
	}

	// With no liveness test there is no world, and so no live speaker.
	bare := speakerCast{actors: actors}
	if _, ok := bare.resolve(data.NPCFace{}); ok {
		t.Error("a cast holding no world resolved a live speaker")
	}
}

// The actor search is outside the registry's picture-kind switch. All three
// kinds therefore use a matching live actor and its current equipment; the
// kind chooses only the fallback used when this population is empty.
func TestLiveSpeakerPrecedesEverySyntheticPictureKind(t *testing.T) {
	helm := dressArmorCode(t, "PlateHelm")
	var worn [sim.EquipSlots]uint16
	worn[dressHelmSlot-1] = helm
	mw := &mapWorld{
		world:   dressWorld(t, 7, worn),
		mission: &missionNotices{src: dressSheets()},
		npcFaces: map[int32]data.NPCFace{
			21:  {Kind: data.NPCNoPicture, Tokens: npcTokenSet(data.NPCTokenHero)},
			25:  {Kind: data.NPCFigure, Dir: data.FigureDirManMage, Face: 1, Tokens: npcTokenSet(data.NPCTokenHero)},
			103: {Kind: data.NPCPortrait, Class: 64, Tokens: npcTokenSet(data.NPCTokenHero)},
		},
		speakerActors: []speakerActor{{id: 7, hero: true,
			fig: figureID{Dir: data.FigureDirManFighter, Face: 1}, face: 1, typeID: 3}},
		invSubjectSet: true,
		invSubject:    ui.InventorySubject{Figure: image.NewRGBA(image.Rect(0, 0, 1, 1))},
	}

	for _, speaker := range []int{21, 25, 103} {
		key := composeKey(t, mw, speaker)
		want := mw.speakerActors[0].fig
		want.Hero = true
		if key.fig != want {
			t.Errorf("speaker %d composed figure %+v, want live hero %+v", speaker, key.fig, want)
		}
		if code, _ := key.eq.Code(dressHelmSlot); code != data.ItemCode(helm) {
			t.Errorf("speaker %d composed head %#x, want live helm %#x", speaker, code, helm)
		}
	}
}

// A speaker no live actor answers for uses the bare face sheet. This is the
// complete synthesised population, including the town's tavern and school
// quest portraits: a Humans row may state a starting outfit, but a portrait is
// not a hero instance and must not materialise it.
func TestSynthesisedSpeakerUsesTheBareFaceSheet(t *testing.T) {
	table := dressTable()
	faces := map[int32]data.NPCFace{
		// npc25: mission 40's own shape, naming Humans row 42 through its
		// DataBinID.
		25: {Kind: data.NPCFigure, Dir: data.FigureDirManFighter, Face: 1,
			Tokens: npcTokenSet(data.NPCTokenHero, data.NPCTokenFace,
				data.NPCTokenNotFemale, data.NPCTokenNotMage)},
		// npc60: no DataBinID at all — 82 of the 105 shipped sections.
		60: {Kind: data.NPCFigure, Dir: data.FigureDirManMage, Face: 1,
			Tokens: npcTokenSet(data.NPCTokenHuman, data.NPCTokenMage)},
		// npc51: a DataBinID naming a row whose ten equipment cells are all
		// empty — seven shipped records are this.
		51: {Kind: data.NPCFigure, Dir: data.FigureDirWomanFighter, Face: 8,
			Tokens: npcTokenSet(data.NPCTokenHuman, data.NPCTokenFemale)},
	}

	// NO CANDIDATE AT ALL — the town screen's own state, and a mission's
	// whenever the section names nobody standing on the map.
	mw := &mapWorld{mission: &missionNotices{src: dressSheets()}, npcFaces: faces}

	for _, tc := range []struct {
		name    string
		speaker int32
	}{
		{"a tavern-style man ignores his Humans-row armour", 25},
		{"a school-style mage ignores the archetype robe", 60},
		{"an outfit-less named row also stays bare", 51},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := faces[tc.speaker]
			worn, _, ok := mapload.SpeakerOutfit(tc.speaker, rec.Dir.Mage(), rec.Dir.Female(), table)
			if !ok || worn == ([sim.EquipSlots]uint16{}) {
				t.Fatal("fixture offers no starting outfit, so it cannot discriminate accidental dressing")
			}
			key := composeKey(t, mw, int(tc.speaker))
			if key.eq != (data.Equipment{}) {
				t.Errorf("synthesised speaker composed with %+v, want twelve empty slots", key.eq)
			}
		})
	}
}

// AC-2, SC-6: mission 40's decoded join, driven end to end over a synthetic
// fixture built from the format contracts (`DLG-DRESS-024`).
//
// The fixture reproduces what the claim reads off the shipped files: a
// placement below the units floor taking the npc arm and naming subscript 25;
// section `npc25` carrying `DataBinID = 42`; and a `Humans` row whose server id
// is 42, named `PC_Paladin`, wearing a plate helm in slot 6. It reads no
// install (golden rule 2).
func TestMission40SpeakerIsThePlacedPaladinInHisHelm(t *testing.T) {
	table := dressTable()
	m := &alm.Map{Units: []alm.Unit{
		// A creature first, so the candidate list is not simply "entity 0".
		{ClassID: 0x20, ClassSubID: 1},
		// The paladin: below the units floor, npc bit set, subscript 25.
		{ClassID: 1, ClassSubID: 25, Flags: 1},
	}}

	actors := entitySpeakers(m, table)
	if len(actors) != 1 {
		t.Fatalf("the map resolved %d candidates, want the one placed person", len(actors))
	}
	if actors[0].id != 1 {
		t.Fatalf("the paladin is entity %d, want 1 — a candidate's id is the placement's own index", actors[0].id)
	}
	if actors[0].face != dressPaladinFace || actors[0].typeID != dressPaladinType {
		t.Fatalf("the candidate carries face %d / typeID %d, want the row's own %d / %d",
			actors[0].face, actors[0].typeID, dressPaladinFace, dressPaladinType)
	}

	// The placement's own spawn dresses him; this is that worn set.
	worn, row, ok := mapload.SpeakerOutfit(25, false, false, table)
	if !ok {
		t.Fatal("the table dressed the paladin in nothing at all")
	}
	if row != "PC_Paladin" {
		t.Fatalf("npc25 dressed from row %q, want PC_Paladin — the row its own DataBinID names", row)
	}
	if got, want := worn[dressHelmSlot-1], dressArmorCode(t, "PlateHelm"); got != want {
		t.Fatalf("slot %d holds %#x, want the plate helm %#x", dressHelmSlot, got, want)
	}

	faces := map[int32]data.NPCFace{
		25: {Kind: data.NPCFigure, Dir: data.FigureDirManFighter, Face: dressPaladinFace,
			Tokens: npcTokenSet(data.NPCTokenHero, data.NPCTokenFace,
				data.NPCTokenNotFemale, data.NPCTokenNotMage)},
	}
	mw := &mapWorld{
		world:    dressWorld(t, 1, worn),
		mission:  &missionNotices{src: dressSheets()},
		npcFaces: faces, speakerActors: actors,
	}

	key := composeKey(t, mw, 25)
	want := actors[0].fig
	want.Hero = true
	if key.fig != want {
		t.Errorf("npc25 composed figure %+v, want the placed paladin's %+v", key.fig, want)
	}
	if code, _ := key.eq.Code(dressHelmSlot); code != data.ItemCode(dressArmorCode(t, "PlateHelm")) {
		t.Errorf("npc25 composed slot %d with %#x, want the plate helm; whole set %+v",
			dressHelmSlot, code, key.eq)
	}
}
