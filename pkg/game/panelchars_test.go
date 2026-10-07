package game

// The driver's own join of the two placement bands and the party into one
// character map, and the live overlay's guard against moving a creature's
// columns.
//
// It is an internal test for sheet_test.go's own reason: what it drives is
// unexported end to end — sheetCharacter and placedCharacters and
// missionCharacters (panelchars.go), and entityDraws' own readout loop
// (world.go) — and none of it has an exported spelling by design.

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// sheetPersonKey is the humans-by-type search's own key column (slot 0x10,
// data.FindHumanByType) this file's person placement carries, kept below
// unitsKeyFloor so Resolve reaches ArmHumansByType rather than ArmUnits —
// mapload/spawn.go's own band split, exercised here the way tableFixtureMap
// (world_test.go) already exercises unitsArmKey for the units arm.
const sheetPersonKey = 7

// personDBRow is a Humans row of the streamed width (data.MinHumanRow),
// every cell empty (-1) but the four statistics, the six skill levels and
// the type-search key — unitsRow's own reason (world_test.go), applied to
// the humans collection that helper does not build.
func personDBRow(name string, body, reaction, mind, spirit int32, skill [data.SkillSlots]int32, typeID int32) dbEntry {
	p := make([]int32, data.MinHumanRow)
	for i := range p {
		p[i] = -1
	}
	p[0], p[1], p[2], p[3] = body, reaction, mind, spirit
	for i, v := range skill {
		p[10+i] = v
	}
	p[0x10] = typeID
	return dbEntry{name: name, params: p}
}

// sheetFixtureMap is worldFixtureMap with two more placements appended: a
// units-arm creature at the first fixture unit's own anchor and a
// humans-arm person at the second's — tableFixtureMap's own trick
// (world_test.go), applied twice, since this file asks neither placement to
// move and only needs each to resolve and cross the driver's join.
func sheetFixtureMap() *alm.Map {
	m := worldFixtureMap()
	m.Units = append(m.Units,
		alm.Unit{X: 0x1580, Y: 0x1700, ClassID: unitsArmKey},
		alm.Unit{X: 0x1900, Y: 0x1480, ClassID: sheetPersonKey})
	return m
}

// sheetFixtureCreatureElemental and sheetFixtureCreatureWeaponKind are the
// creature row's own two families in sheetFixtureTable, TEN DISTINCT
// numbers between them (mapload/sheet_test.go's own R-1 mitigation, carried
// up to this package's conversion so a transposition IN sheetCharacter, not
// just in PlacedSheets one tier down, would show as a wrong number).
var (
	sheetFixtureCreatureElemental  = [5]int32{101, 102, 103, 104, 105}
	sheetFixtureCreatureWeaponKind = [5]int32{111, 112, 113, 114, 115}
)

// sheetFixtureTable is the table sheetFixtureMap resolves its two appended
// placements against.
func sheetFixtureTable() *mapload.Table {
	slots := map[int]int32{0: 44, 1: 33, 2: 22, 3: 11}
	for i, v := range sheetFixtureCreatureElemental {
		slots[19+i] = v
	}
	for i, v := range sheetFixtureCreatureWeaponKind {
		slots[24+i] = v
	}
	creature := unitsRow("Creature", slots)

	personSkill := [data.SkillSlots]int32{9, 1, 2, 3, 4, 5}
	person := personDBRow("Person", 30, 20, 18, 17, personSkill, sheetPersonKey)

	return &mapload.Table{
		Units:  dbCollection{{}, creature},
		Humans: dbCollection{{}, person},
	}
}

// TestDriverStatesACharacterForEachPlacedBandAndForTheParty is AC-1..AC-4,
// AC-7: the driver's merged map (missionCharacters, reached here through
// openMapWorld's own placedCharacters half) states a character for a placed
// creature and a placed person, each converted field by field from the
// map-loading tier's own Sheet.
func TestDriverStatesACharacterForEachPlacedBandAndForTheParty(t *testing.T) {
	m := sheetFixtureMap()
	tbl := sheetFixtureTable()

	mw := mustOpenMapWorld(t, m, tbl, nil, worldFixtureViewer(t, m))

	// THE CREATURE (placement index 4): its four statistics and both families
	// are the row's own columns, unchanged; its skill positions 1..5 are a copy
	// of the weapon-kind family and position 0 is left unstated at zero; its
	// experience is unstated.
	creature, ok := mw.chars[4]
	if !ok || !creature.Known {
		t.Fatalf("the creature placement states no character: %+v", creature)
	}
	if creature.Band != ui.CharacterBandCreature {
		t.Errorf("creature Band = %v, want CharacterBandCreature", creature.Band)
	}
	if creature.Body != 44 || creature.Reaction != 33 || creature.Mind != 22 || creature.Spirit != 11 {
		t.Errorf("creature stats = {%d %d %d %d}, want the row's own {44 33 22 11}",
			creature.Body, creature.Reaction, creature.Mind, creature.Spirit)
	}
	wantElemental, wantWeaponKind := [5]int{101, 102, 103, 104, 105}, [5]int{111, 112, 113, 114, 115}
	if creature.Protection != wantElemental {
		t.Errorf("creature Protection = %v, want %v (the row's elemental columns)", creature.Protection, wantElemental)
	}
	if creature.Resistance != wantWeaponKind {
		t.Errorf("creature Resistance = %v, want %v (the row's weapon-kind columns)", creature.Resistance, wantWeaponKind)
	}
	wantSkill := [ui.PanelSkillSlots]int{0, 111, 112, 113, 114, 115}
	if creature.Skills != wantSkill {
		t.Errorf("creature Skills = %v, want %v (FR-4: a copy of the weapon-kind family, General unstated)", creature.Skills, wantSkill)
	}
	if creature.Experience != 0 {
		t.Errorf("creature Experience = %d, want 0 (FR-3: unstated)", creature.Experience)
	}

	// THE PERSON (placement index 5): capped statistics and the derived
	// families, off the SAME row taken through NewHumanDef and Recompute
	// directly with an empty Loadout — the reference this does not
	// re-derive (that is personSheet's own test, 0137 T1), only wires up.
	person, ok := mw.chars[5]
	if !ok || !person.Known {
		t.Fatalf("the person placement states no character: %+v", person)
	}
	if person.Band != ui.CharacterBandPerson {
		t.Errorf("person Band = %v, want CharacterBandPerson", person.Band)
	}
	d, err := data.NewHumanDef("Person", tbl.Humans.EntryParams(1))
	if err != nil {
		t.Fatalf("data.NewHumanDef: %v", err)
	}
	want := d.Hero().Recompute(d.Profile(), data.Loadout{})
	if person.Body != int(want.Body) || person.Reaction != int(want.Reaction) ||
		person.Mind != int(want.Mind) || person.Spirit != int(want.Spirit) {
		t.Errorf("person stats = {%d %d %d %d}, want the capped {%d %d %d %d}",
			person.Body, person.Reaction, person.Mind, person.Spirit,
			want.Body, want.Reaction, want.Mind, want.Spirit)
	}
	wantPersonSkill := [ui.PanelSkillSlots]int{9, 1, 2, 3, 4, 5}
	if person.Skills != wantPersonSkill {
		t.Errorf("person Skills = %v, want the row's own %v (FR-5, plan DD-7: not the graph's restored levels)", person.Skills, wantPersonSkill)
	}
	if person.Experience != int(want.Experience) {
		t.Errorf("person Experience = %d, want the graph's own %d", person.Experience, want.Experience)
	}
}

func TestMissionCharactersMergesPlacementsThenParty(t *testing.T) {
	m := sheetFixtureMap()
	tbl := sheetFixtureTable()

	hero := data.NewHero(data.Spread{Body: 30, Reaction: 20, Mind: 18, Spirit: 17}, data.SkillAxe)
	ms := &Mission{
		Map:   m,
		Party: []mapload.PartyMember{{Hero: hero}},
		// Start.IDs names entity 4 ON PURPOSE — the creature placement's own
		// id — so a merge that ran the two passes in the wrong order would
		// still leave a person character at it, and the mistake would read
		// as a person rather than as a missing overwrite.
		Start: mapload.Start{IDs: []sim.EntityID{4}},
	}

	got := missionCharacters(m, tbl, ms)

	person, ok := got[5]
	if !ok || person.Band != ui.CharacterBandPerson {
		t.Fatalf("placement 5 states %+v after the merge, want an untouched known person", person)
	}

	want := partyCharacters(ms)[4]
	if got[4] != want {
		t.Errorf("id 4 states %+v after the merge, want the party's own %+v unchanged", got[4], want)
	}
	if got[4].Band != ui.CharacterBandPerson {
		t.Errorf("id 4 Band = %v after the merge, want CharacterBandPerson — the party's character, not the creature's", got[4].Band)
	}
}

func TestAStepDoesNotMoveACreaturesSkillPositions(t *testing.T) {
	const creditedSlot = 3
	const creatureAttacker, creatureTarget sim.EntityID = 1, 2
	const personAttacker, personTarget sim.EntityID = 3, 4

	attacker := func(id sim.EntityID, x, y int32, owner uint32) sim.Entity {
		return sim.Entity{ID: id, X: x, Y: y, HP: 100, MaxHP: 100, DyingTime: 200,
			AttackCharge: 1, AttackRelax: 50, DamageBase: 10, AlwaysHits: true,
			Owner: owner, GainsXP: true, Mind: 60, XPSlot: creditedSlot}
	}
	target := func(id sim.EntityID, x, y int32, owner uint32) sim.Entity {
		return sim.Entity{ID: id, X: x, Y: y, HP: 100, MaxHP: 100, DyingTime: 200,
			Owner: owner, XPValue: 10}
	}

	creature := attacker(creatureAttacker, 0, 0, 2)
	person := attacker(personAttacker, 8, 8, 4)
	person.TypeID = sim.HumanTypeID
	w, err := sim.NewWorld(200, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil, []sim.Entity{
		creature, target(creatureTarget, 1, 0, 3), person, target(personTarget, 9, 8, 5),
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}

	// THE CREATURE'S SKILLS ARE ITS SHEET'S OWN COLUMNS, arbitrary and
	// distinct from anything SkillLevelFor could plausibly answer for the
	// XP one landed blow credits, so a guard that failed to hold them would
	// show as a wrong number rather than a coincidence.
	creatureSkill := [ui.PanelSkillSlots]int{0, 11, 12, 13, 14, 15}
	mw := &mapWorld{world: w, chars: map[sim.EntityID]ui.UnitCharacter{
		creatureAttacker: {Known: true, Band: ui.CharacterBandCreature, Skills: creatureSkill},
		personAttacker:   {Known: true, Band: ui.CharacterBandPerson},
	}}

	readChar := func(id sim.EntityID) ui.UnitCharacter {
		t.Helper()
		for _, d := range mw.entityDraws() {
			if d.ID == uint32(id) {
				return d.Char
			}
		}
		t.Fatalf("entity %d crossed no entry", id)
		return ui.UnitCharacter{}
	}

	sim.Step(w, []sim.Command{
		{Kind: sim.KindAttack, Entity: creatureAttacker, X: int32(creatureTarget)},
		{Kind: sim.KindAttack, Entity: personAttacker, X: int32(personTarget)},
	})

	creatureReadout := readChar(creatureAttacker)
	if creatureReadout.Skills != creatureSkill {
		t.Errorf("creature Skills = %v after a landed blow, want the unmoved columns %v", creatureReadout.Skills, creatureSkill)
	}
	if creatureReadout.Experience != 0 {
		t.Errorf("low-TypeID creature Experience = %d after a landed blow, want 0", creatureReadout.Experience)
	}

	// THE PERSON: Skills[creditedSlot] IS what the overlay read off the
	// entity after the blow just landed, and it is checked against the
	// entity's own reading rather than pasted, so a wrong read could not
	// pass alongside a right one.
	personReadout := readChar(personAttacker)
	if personReadout.Experience <= 0 {
		t.Fatalf("person Experience = %d after a landed blow, want more than 0", personReadout.Experience)
	}
	var stored int32
	for _, e := range w.Entities() {
		if e.ID == personAttacker {
			stored = e.Skill[creditedSlot]
		}
	}
	if stored == 0 {
		t.Fatalf("the person's stored level at slot %d is still 0 after a landed blow — "+
			"this test can say nothing about the overlay if nothing moved", creditedSlot)
	}
	if personReadout.Skills[creditedSlot] != int(stored) {
		t.Errorf("person Skills[%d] = %d, want the entity's own stored level %d",
			creditedSlot, personReadout.Skills[creditedSlot], stored)
	}
}
