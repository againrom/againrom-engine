package mapload_test

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// 0159: a unit the mission script hands to the player joins the party and stays
// in it across the mission boundary, with his equipment.
//
// The fixtures reuse human_test.go's row, table and map shapes. Nothing here
// reads a game install.

// joinMap places two units: a person on key 7, which the humans band resolves,
// and a creature on a key above the class-key floor, which the units band
// resolves. Both stand under a slot that is not the player's, so a hand-over is
// what puts either under him and nothing else can.
func joinMap() *alm.Map {
	return &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{
		{X: 0x0C80, Y: 0x0C80, ClassID: 7, ClassSubID: 52, Flags: 1, UnitID: 11, Owner: 4},
		{X: 0x0D80, Y: 0x0C80, ClassID: joinCreatureKey, ClassSubID: 1, UnitID: 12, Owner: 4},
	}}
}

// joinCreatureKey is a key at or above the class-key floor, so the placement
// takes the units band. Its value is the fixture creature row's own type id.
const joinCreatureKey = 0x40

// joinPackItem is the item code the fixture script puts into the handed-over
// actor's container. Its value is arbitrary and names no shipped item; the
// container carries a code whether or not a table resolves it.
const joinPackItem uint16 = 0x1234

// joinTable resolves key 7 through one humans row and joinCreatureKey through
// one units row.
func joinTable() *mapload.Table {
	units := defRow(map[int]int32{
		slotUnitType: joinCreatureKey, slotUnitFace: 1, slotHealthMax: 40, slotSpeed: 10,
	})
	human := fullHumanRow()
	human[slotServerID] = 777
	npcBytes := synth.Reg(0x11, []synth.RegNode{{Name: "npc52", Kind: 0x01, Children: []synth.RegNode{
		{Name: "Flags", Kind: 0x00, Str: "Hero,Human"},
		{Name: "DataBinID", Kind: 0x02, Int: 777},
	}}})
	npcReg, err := reg.Parse(npcBytes)
	if err != nil {
		panic(err)
	}
	return &mapload.Table{
		Units: defCollection{{}, {name: "beast", params: units}},
		// THE PERSON IS ARMED. His worn set is what the story is about, so a
		// fixture wearing nothing would let AC-7's equipment clause pass
		// vacuously: an empty array equals an empty array.
		Humans: defCollection{{}, {name: "hero", params: human, strings: []string{"Sword"}}},
		NPC:    data.LoadNPCDefs(npcReg),
		Shapes: identityScale(), Materials: identityScale(), Weapons: swordAndBow(),
	}
}

// TestAHandedOverLowTypeHumanIsMissionOnly pins the mission-20 rule: a map
// Human may be controllable after a script transfer without becoming a
// persistent party member. The table TypeID, not the broad Humans arm, decides
// the boundary.
func TestAHandedOverLowTypeHumanIsMissionOnly(t *testing.T) {
	m := joinMap()
	m.Units[0].Flags = 0
	m.Units[0].ClassSubID = 0
	table := joinTable()
	humans := table.Humans.(defCollection)
	humans[1] = defEntry{name: "guard", params: fullHumanRow(), strings: []string{"Sword"}}
	table.Humans = humans
	w, st, err := mapload.StartMissionScripted(m, table, mapload.DifficultyNormal,
		joinParty(), handOverScript(t, 0))
	if err != nil {
		t.Fatalf("StartMissionScripted: %v", err)
	}
	for i := 0; i < 32; i++ {
		sim.Step(w, nil)
	}
	if got := joinEntity(t, w, 0); got.Owner != sim.SelfSlot || got.TypeID != 7 || got.GainsXP {
		t.Fatalf("transferred guard is owner/type/gains %d/%d/%v, want %d/7/false",
			got.Owner, got.TypeID, got.GainsXP, sim.SelfSlot)
	}
	out := mapload.CarryRoster(joinParty(), w, st.IDs, st.Roster)
	if len(out) != 1 || out[0].ID != "hero" {
		t.Fatalf("mission-only guard crossed the boundary: %#v", out)
	}
}

func TestJoinedRosterDeduplicatesTheRuntimeActorAndCullsDeadCandidates(t *testing.T) {
	member := mapload.PartyMember{ID: "join:1", CompanionNPC: 25, PlayerCharacter: true}
	world, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
		[]sim.Entity{
			{ID: 1, HP: 10, MaxHP: 10, Owner: sim.SelfSlot, TypeID: sim.HumanTypeID},
			{ID: 3, HP: 0, MaxHP: 10, Owner: sim.SelfSlot, TypeID: sim.HumanTypeID},
		})
	if err != nil {
		t.Fatal(err)
	}
	party, ids := mapload.CarryRosterIDs([]mapload.PartyMember{member}, world, []sim.EntityID{1},
		map[sim.EntityID]mapload.PartyMember{
			3: {ID: "join:3", CompanionNPC: 26, PlayerCharacter: true},
		})
	if len(party) != 1 || len(ids) != 1 || ids[0] != 1 {
		t.Fatalf("dedup/death carry = %#v ids %v, want the existing Brian alone", party, ids)
	}
}

// handOverScript is a script whose one trigger fires on the first pass and
// hands unit to the player's own slot. Its condition compares a constant
// against itself.
//
// unit is an ENTITY id and not a map record's own unit id: a compiled script
// carries the entity id the binder resolved, and these fixtures are built
// directly rather than bound from a map section.
func handOverScript(t *testing.T, unit sim.EntityID) *sim.Script {
	t.Helper()
	s, err := sim.NewScript(
		[]sim.ScriptCheck{
			{Op: sim.ScriptCheckConstant, Register: 0, Args: [10]int32{1}},
			{Op: sim.ScriptCheckConstant, Register: 1, Args: [10]int32{1}},
		},
		[]sim.ScriptInstant{
			{Op: sim.ScriptInstantGiveUnit,
				Unit: unit, HasUnit: true, Player: sim.SelfSlot, HasPlayer: true},
			// ONE ITEM INTO HIS PACK, so the pack clause of AC-7 is measurable:
			// the fixture row arms him but fills no container, and an empty
			// pack equalling an empty pack would pass whatever the boundary
			// did with it.
			{Op: sim.ScriptInstantAddItem, Unit: unit, HasUnit: true, Item: joinPackItem, HasItem: true},
		},
		[]sim.ScriptTrigger{{
			Pairs:    [3]sim.ScriptPair{{Left: 0, Right: 1, Cmp: sim.ScriptCmpEQ, Used: true}},
			Instants: [4]int32{0, 1, sim.ScriptNone, sim.ScriptNone},
			Once:     true, Latch: 0,
		}})
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	return s
}

// joinParty is the one member who walks in. He is the starting hero, so the
// companion appended beside him is measurably not.
func joinParty() []mapload.PartyMember {
	return []mapload.PartyMember{{ID: "hero", StartingHero: true, PlayerCharacter: true,
		Hero: data.Hero{Body: 20, Reaction: 20, Mind: 20, Spirit: 20}}}
}

// runJoin starts the fixture mission with the hand-over script, drives it far
// enough for the trigger to fire, and returns the world and the start.
func runJoin(t *testing.T, unit sim.EntityID) (*sim.World, mapload.Start) {
	t.Helper()
	w, st, err := mapload.StartMissionScripted(joinMap(), joinTable(), mapload.DifficultyNormal,
		joinParty(), handOverScript(t, unit))
	if err != nil {
		t.Fatalf("StartMissionScripted: %v", err)
	}
	for i := 0; i < 32; i++ {
		sim.Step(w, nil)
	}
	if !w.ScriptLatched(0) {
		t.Fatal("the hand-over trigger never fired; every case here measures what happens when it does")
	}
	return w, st
}

// joinEntity is one entity of the world, by id, failing where the world does
// not hold it.
func joinEntity(t *testing.T, w *sim.World, id sim.EntityID) sim.Entity {
	t.Helper()
	e, ok := entityByID(w, id)
	if !ok {
		t.Fatalf("the world holds no entity %d", id)
	}
	return e
}

// TestAPlacedPersonAndAPlacedCreatureCarryDifferentTypeIDs is 0159 AC-1's
// loader half: the person's is inside the band and the creature's is its own
// row's key, outside it.
func TestAPlacedPersonAndAPlacedCreatureCarryDifferentTypeIDs(t *testing.T) {
	t.Parallel()

	w, err := mapload.FromALMWith(joinMap(), joinTable(), mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	person, creature := joinEntity(t, w, 0), joinEntity(t, w, 1)
	if !sim.InPersistBand(person.TypeID) {
		t.Errorf("the placed person's type id is %#x, outside the band", person.TypeID)
	}
	if sim.InPersistBand(creature.TypeID) {
		t.Errorf("the placed creature's type id is %#x, inside the band", creature.TypeID)
	}
	if creature.TypeID != joinCreatureKey {
		t.Errorf("the placed creature's type id is %d, want its row's own %d",
			creature.TypeID, joinCreatureKey)
	}
}

// TestAMintedPartyMemberCarriesABandTypeID is 0159 AC-2's loader half.
func TestAMintedPartyMemberCarriesABandTypeID(t *testing.T) {
	t.Parallel()

	w, st, err := mapload.StartMission(joinMap(), joinTable(), mapload.DifficultyNormal, joinParty())
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	e := joinEntity(t, w, st.IDs[0])
	if !sim.InPersistBand(e.TypeID) {
		t.Errorf("the minted member's type id is %#x, outside the band", e.TypeID)
	}
	// And the boundary therefore keeps him, which is the pre-existing behaviour
	// this story must not have broken: a party member has always crossed.
	got := w.BoundarySurvivors(sim.SelfSlot)
	if len(got) != 1 || got[0] != st.IDs[0] {
		t.Errorf("the boundary keeps %v, want just the member's own %d", got, st.IDs[0])
	}
}

// TestAHandedOverPersonEntersThePartyWithWhatHeCarried is 0159 AC-7.
func TestAHandedOverPersonEntersThePartyWithWhatHeCarried(t *testing.T) {
	t.Parallel()

	w, st := runJoin(t, 0)
	before := joinEntity(t, w, 0)
	if before.Owner != sim.SelfSlot {
		t.Fatalf("the hand-over left the person under slot %d", before.Owner)
	}
	worn, _ := w.Equipped(0)
	items, _ := w.Carried(0)

	out := mapload.CarryRoster(joinParty(), w, st.IDs, st.Roster)
	if len(out) != 2 {
		t.Fatalf("the party crossed with %d members, want the hero and the joiner", len(out))
	}
	joiner := out[1]
	if joiner.Carry == nil {
		t.Fatal("the joiner crossed with no carry at all")
	}
	if worn == ([sim.EquipSlots]uint16{}) {
		t.Fatal("the fixture person wears nothing; this case cannot measure that he kept it")
	}
	if joiner.Carry.Equipped != worn {
		t.Errorf("the joiner crossed wearing %v, want the entity's own %v",
			joiner.Carry.Equipped, worn)
	}
	if len(items) == 0 {
		t.Fatal("the fixture person holds nothing; this case cannot measure that he kept it")
	}
	if len(joiner.Carry.Items) != len(items) || joiner.Carry.Items[0] != items[0] {
		t.Errorf("the joiner crossed holding %v, want the entity's own %v",
			joiner.Carry.Items, items)
	}
	if joiner.Carry.SkillXP != before.SkillXP {
		t.Errorf("the joiner crossed with experience %v, want the entity's own %v",
			joiner.Carry.SkillXP, before.SkillXP)
	}
	// The hero who walked in is still first and is still himself.
	if out[0].ID != "hero" {
		t.Errorf("the party's first member is %q, want the hero who walked in", out[0].ID)
	}
}

func TestAHandedOverPersonsLearnedSpellsEnterThePartyAndNextMission(t *testing.T) {
	w, st := runJoin(t, 0)
	ents := w.Entities()
	const learned = uint32(1) << 26
	found := false
	for i := range ents {
		if ents[i].ID == 0 {
			ents[i].KnownSpells |= learned
			found = true
		}
	}
	if !found {
		t.Fatal("fixture world has no handed-over person 0")
	}
	finished, err := sim.NewStockedWorld(1051, w.Bounds(), sim.ModeCanonical, sim.Terrain{},
		ents, nil, w.Relations(), w.Sacks(), w.Stock())
	if err != nil {
		t.Fatalf("rebuild finished world: %v", err)
	}

	party := mapload.CarryRoster(joinParty(), finished, st.IDs, st.Roster)
	if len(party) != 2 || party[1].KnownSpells&learned == 0 {
		t.Fatalf("carried party = %#v, want joiner with Teleport learned", party)
	}
	next, nst, err := mapload.StartMission(joinMap(), joinTable(), mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatalf("StartMission on next map: %v", err)
	}
	if len(nst.IDs) != 2 || joinEntity(t, next, nst.IDs[1]).KnownSpells&learned == 0 {
		t.Fatalf("next mission joiner did not retain Teleport")
	}
}

// TestTheJoinerIsMintedFromHisOwnRowOnTheNextMap is 0159 AC-8.
//
// His maximum health on the next map is the person row's own derived maximum. A
// joiner rebuilt from his entity alone would carry no Body, Reaction or Spirit
// and would fold a different number here, which is what this measures.
func TestTheJoinerIsMintedFromHisOwnRowOnTheNextMap(t *testing.T) {
	t.Parallel()

	w, st := runJoin(t, 0)
	out := mapload.CarryRoster(joinParty(), w, st.IDs, st.Roster)

	next, nst, err := mapload.StartMission(joinMap(), joinTable(), mapload.DifficultyNormal, out)
	if err != nil {
		t.Fatalf("StartMission on the next map: %v", err)
	}
	if len(nst.IDs) != 2 {
		t.Fatalf("the next map minted %d members, want 2", len(nst.IDs))
	}
	e := joinEntity(t, next, nst.IDs[1])
	if want := wantHumanHealth(t); e.MaxHP != want {
		t.Errorf("the joiner's maximum health on the next map is %d, want his row's own %d",
			e.MaxHP, want)
	}
	// And he arrives whole, which is what the original's own boundary does to a
	// kept actor by restoring each pool from its maximum.
	if e.HP != e.MaxHP {
		t.Errorf("the joiner arrived at %d/%d health, want full", e.HP, e.MaxHP)
	}
}

// TestTheJoinerIsAPersistentMemberAndNotAPrimaryCharacter is 0159 AC-9.
func TestTheJoinerIsAPersistentMemberAndNotAPrimaryCharacter(t *testing.T) {
	t.Parallel()

	w, st := runJoin(t, 0)
	joiner := mapload.CarryRoster(joinParty(), w, st.IDs, st.Roster)[1]
	switch {
	case joiner.StartingHero:
		t.Error("the joiner is the starting hero; the hand-over writes no primary-character pointer")
	case joiner.Temporary:
		t.Error("the joiner is temporary; he is a persistent roster member")
	case joiner.MercenaryType != 0:
		t.Errorf("the joiner carries mercenary type %d", joiner.MercenaryType)
	case !joiner.PlayerCharacter:
		t.Error("the joiner is not a player character")
	case joiner.ID != "join:0":
		t.Errorf("the joiner's identity is %q, want the runtime id he joined under", joiner.ID)
	}
}

func TestTheJoinerSurvivesASecondBoundaryExactlyOnce(t *testing.T) {
	t.Parallel()

	w, st := runJoin(t, 0)
	first := mapload.CarryRoster(joinParty(), w, st.IDs, st.Roster)

	// The second mission is the same map with NO hand-over: the joiner is a
	// member there like any other, and what carries him out is the ordinary
	// per-member arm rather than a second join.
	next, nst, err := mapload.StartMission(joinMap(), joinTable(), mapload.DifficultyNormal, first)
	if err != nil {
		t.Fatalf("StartMission on the next map: %v", err)
	}
	second := mapload.CarryRoster(first, next, nst.IDs, nst.Roster)

	if len(second) != 2 {
		t.Fatalf("the party crossed the second boundary with %d members, want 2", len(second))
	}
	if second[1].ID != first[1].ID {
		t.Errorf("the joiner's identity changed from %q to %q across the second boundary",
			first[1].ID, second[1].ID)
	}
	// The map's own person placement is still standing under its own slot on
	// the second map and is NOT appended a second time, which is what "exactly
	// once" measures: nothing handed it over there.
	for _, p := range second {
		if p.ID != first[1].ID {
			continue
		}
		if p.Carry == nil {
			t.Error("the joiner lost his carry across the second boundary")
		}
	}
}

// TestAHandedOverCreatureDoesNotEnterTheParty is 0159 AC-11's own clause and
// PARTY-BAND-027's consequence: a script handover of a creature is mission-only,
// with no flag, group or script difference from the person's.
func TestAHandedOverCreatureDoesNotEnterTheParty(t *testing.T) {
	t.Parallel()

	w, st := runJoin(t, 1)
	if e := joinEntity(t, w, 1); e.Owner != sim.SelfSlot {
		t.Fatalf("the hand-over left the creature under slot %d", e.Owner)
	}
	out := mapload.CarryRoster(joinParty(), w, st.IDs, st.Roster)
	if len(out) != 1 {
		t.Errorf("the party crossed with %d members, want only the hero: a creature is mission-only", len(out))
	}
}

func TestCarryRosterWithNoRosterIsCarryParty(t *testing.T) {
	t.Parallel()

	w, st := runJoin(t, 0)
	plain := mapload.CarryParty(joinParty(), w, st.IDs)
	none := mapload.CarryRoster(joinParty(), w, st.IDs, nil)
	if len(none) != len(plain) {
		t.Fatalf("with no roster the party crossed with %d members, want CarryParty's %d",
			len(none), len(plain))
	}
	for i := range plain {
		if none[i].ID != plain[i].ID {
			t.Errorf("member %d is %q with no roster and %q through CarryParty",
				i, none[i].ID, plain[i].ID)
		}
	}
}

// RosterJoinPending gates the per-tick roster clone, so it must answer
// exactly whether CarryRosterIDs grows the party.
func TestRosterJoinPendingMatchesCarryRosterIDs(t *testing.T) {
	world, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
		[]sim.Entity{
			{ID: 1, HP: 10, MaxHP: 10, Owner: sim.SelfSlot, TypeID: sim.HumanTypeID},
			{ID: 2, HP: 10, MaxHP: 10, Owner: sim.SelfSlot, TypeID: sim.HumanTypeID},
			{ID: 3, HP: 0, MaxHP: 10, Owner: sim.SelfSlot, TypeID: sim.HumanTypeID},
			{ID: 4, HP: 10, MaxHP: 10, Owner: sim.SelfSlot + 1, TypeID: sim.HumanTypeID},
		})
	if err != nil {
		t.Fatal(err)
	}
	party := []mapload.PartyMember{{ID: "join:1", PlayerCharacter: true}}
	joins := 0
	for _, roster := range []sim.EntityID{0, 1, 2, 3, 4} {
		r := map[sim.EntityID]mapload.PartyMember{}
		if roster != 0 {
			r[roster] = mapload.PartyMember{ID: "join:roster", PlayerCharacter: true}
		}
		out, _ := mapload.CarryRosterIDs(party, world, []sim.EntityID{1}, r)
		if got, want := mapload.RosterJoinPending(world, []sim.EntityID{1}, r), len(out) > len(party); got != want {
			t.Errorf("roster %d: RosterJoinPending %v, CarryRosterIDs grew %v", roster, got, want)
		} else if got {
			joins++
		}
	}
	if joins != 1 {
		t.Errorf("%d roster case(s) joined, want only the living self-owned actor 2", joins)
	}
	if mapload.RosterJoinPending(nil, nil, map[sim.EntityID]mapload.PartyMember{2: {}}) {
		t.Error("RosterJoinPending without a world")
	}
}
