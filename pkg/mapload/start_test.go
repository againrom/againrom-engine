package mapload_test

import (
	"bytes"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// startMap is a map large enough to have open ground inside its border ring,
// carrying a script that authorises the cells given. Its tiles are all the
// passable default, so the only closed cells are the eight-cell ring.
func startMap(t *testing.T, w, h int, cells ...mapload.Cell) *alm.Map {
	t.Helper()
	nodes := make([][]byte, 0, len(cells))
	for _, c := range cells {
		nodes = append(nodes, dropNode(testDropOpcode, uint32(c.X), uint32(c.Y)))
	}
	n, body := payload(0, 0, nodes...)
	return &alm.Map{
		Width: w, Height: h,
		Tiles:    make([]uint16, w*h),
		Overlay:  make([]uint8, w*h),
		Triggers: alm.Triggers{EntryCount: n, Body: body},
	}
}

func party(n int) []mapload.PartyMember {
	p := make([]mapload.PartyMember, n)
	for i := range p {
		p[i] = mapload.PartyMember{Class: int32(100 + i)}
	}
	return p
}

func mustStart(t *testing.T, m *alm.Map, p []mapload.PartyMember) (*sim.World, mapload.Start) {
	t.Helper()
	w, st, err := mapload.StartMission(m, nil, mapload.DifficultyNormal, p)
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	return w, st
}

// The map's own cell is taken, the hero stands on it exactly, and the rest stand
// on open cells nobody else has.
func TestStartMissionPutsThePartyOnTheMapsOwnCell(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	w, st := mustStart(t, m, party(5))

	if st.Fallback || st.Authorised != 1 {
		t.Fatalf("start = %+v, want the map's one authorised cell", st)
	}
	if st.Drop != (mapload.Cell{X: 17, Y: 20}) || st.Cells[0] != st.Drop {
		t.Fatalf("drop %v, hero %v — the hero stands on the drop cell exactly", st.Drop, st.Cells[0])
	}
	if st.Crowded != 0 {
		t.Errorf("%d member(s) could not be placed on a 40x40 map", st.Crowded)
	}

	block := mapload.Passability(m)
	seen := make(map[mapload.Cell]bool)
	for i, c := range st.Cells {
		if seen[c] {
			t.Errorf("member %d shares cell %v", i, c)
		}
		seen[c] = true
		if i > 0 && block[c.Y*40+c.X]&1 != 0 {
			t.Errorf("member %d stands on a closed cell %v", i, c)
		}
	}

	// The party is appended, so every placement keeps its id and the members are
	// the last entities, in party order.
	ents := w.Entities()
	if len(ents) != len(m.Units)+5 {
		t.Fatalf("%d entities for %d placements and 5 members", len(ents), len(m.Units))
	}
	for i := 0; i < 5; i++ {
		e := ents[len(m.Units)+i]
		if e.Class != int32(100+i) || e.X != st.Cells[i].X || e.Y != st.Cells[i].Y {
			t.Errorf("member %d is entity %+v, want class %d at %v", i, e, 100+i, st.Cells[i])
		}
		if e.Domain != sim.DomainGround || e.HP != mapload.SpawnHP || e.MaxHP != mapload.SpawnHP {
			t.Errorf("member %d carries %+v, want the ground domain and the provisional pair", i, e)
		}
	}
}

// A script authorising nothing, and a cell with a zero coordinate, both fall
// back — and the fallback's range is closed at both ends on both axes.
// Quest documents are campaign state presented through the explicitly marked
// primary hero. A carried companion can arrive holding them, but the mission
// boundary consolidates only that code and leaves her ordinary items alone.
func TestStartMissionConsolidatesQuestDocumentsOntoThePrimaryHero(t *testing.T) {
	const ordinary = uint16(0x1111)
	p := []mapload.PartyMember{
		{Name: "Danath", Class: 100, PlayerCharacter: true, StartingHero: true,
			Carried: []uint16{uint16(data.QuestDocumentCode)}},
		{Name: "Reniesta", Class: 101, PlayerCharacter: true, CompanionNPC: 22,
			Carried: []uint16{ordinary, uint16(data.QuestDocumentCode), uint16(data.QuestDocumentCode)}},
	}
	w, st := mustStart(t, startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20}), p)
	primary, ok := w.Carried(st.IDs[0])
	if !ok || !reflect.DeepEqual(primary, []uint16{
		uint16(data.QuestDocumentCode), uint16(data.QuestDocumentCode), uint16(data.QuestDocumentCode),
	}) {
		t.Fatalf("primary carried = %#v, %v; want the three-document campaign stack", primary, ok)
	}
	companion, ok := w.Carried(st.IDs[1])
	if !ok || !reflect.DeepEqual(companion, []uint16{ordinary}) {
		t.Fatalf("companion carried = %#v, %v; want only ordinary item %#04x", companion, ok, ordinary)
	}
}

func TestStartMissionFallsBack(t *testing.T) {
	for _, tc := range []struct {
		name string
		m    *alm.Map
		auth int
	}{
		{"no authorised cell", startMap(t, 128, 128), 0},
		{"a zero column", startMap(t, 128, 128, mapload.Cell{X: 0, Y: 40}), 1},
		{"a zero row", startMap(t, 128, 128, mapload.Cell{X: 40, Y: 0}), 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, st := mustStart(t, tc.m, party(1))
			if !st.Fallback || st.Authorised != tc.auth {
				t.Fatalf("start = %+v, want the fallback with %d authorised", st, tc.auth)
			}
			if st.Drop.X < 30 || st.Drop.X > 100 || st.Drop.Y < 30 || st.Drop.Y > 100 {
				t.Errorf("fallback cell %v is outside [30,100] on some axis", st.Drop)
			}
		})
	}
}

// Both ends of the fallback range are reachable and the two axes are drawn
// independently — asserted over the draw sequence itself, since one seed gives
// one cell and a single start cannot show a range.
func TestFallbackRangeIsClosedAndPerAxis(t *testing.T) {
	lowX, highX, lowY, highY, differ := false, false, false, false, false
	for seed := uint64(0); seed < 40000; seed++ {
		d := sim.NewDraws(seed)
		x, y := 30+d.Upto(70), 30+d.Upto(70)
		lowX, highX = lowX || x == 30, highX || x == 100
		lowY, highY = lowY || y == 30, highY || y == 100
		differ = differ || x != y
	}
	if !lowX || !highX || !lowY || !highY {
		t.Errorf("the range's ends were not all reached: %v %v %v %v", lowX, highX, lowY, highY)
	}
	if !differ {
		t.Error("the two axes never differed — they are not drawn independently")
	}
}

// A payload the framing does not tile is a map that authorises nothing, so the
// start falls back rather than reading a cell out of a drifted offset.
func TestStartMissionOverAnUntilingPayload(t *testing.T) {
	m := startMap(t, 128, 128, mapload.Cell{X: 40, Y: 40})
	m.Triggers.Body = append(m.Triggers.Body, 0)
	_, st := mustStart(t, m, party(1))
	if !st.Fallback || st.Authorised != 0 {
		t.Fatalf("start = %+v, want the fallback with nothing authorised", st)
	}
}

// One input, one world. This is what a start's whole determinism claim reduces
// to, and it is asserted on the digest rather than on the report alone.
func TestStartMissionIsDeterministic(t *testing.T) {
	build := func() (*sim.World, mapload.Start) {
		return mustStart(t, startMap(t, 128, 128), party(4))
	}
	w1, s1 := build()
	w2, s2 := build()
	if w1.Hash() != w2.Hash() {
		t.Errorf("two starts hash %#016x and %#016x", w1.Hash(), w2.Hash())
	}
	if s1.Drop != s2.Drop || s1.Fallback != s2.Fallback || len(s1.Cells) != len(s2.Cells) {
		t.Fatalf("reports differ: %+v vs %+v", s1, s2)
	}
	for i := range s1.Cells {
		if s1.Cells[i] != s2.Cells[i] {
			t.Errorf("member %d: %v vs %v", i, s1.Cells[i], s2.Cells[i])
		}
	}
}

// An empty party builds exactly the world a plain load builds — the one place a
// start could have changed a world nobody asked it to change.
func TestStartMissionWithNoPartyBuildsThePlainWorld(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	got, st := mustStart(t, m, nil)
	want, err := mapload.FromALMWith(m, nil, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	if got.Hash() != want.Hash() {
		t.Errorf("a start with no party hashes %#016x, want %#016x", got.Hash(), want.Hash())
	}
	if st.Drop != (mapload.Cell{X: 17, Y: 20}) || len(st.Cells) != 0 {
		t.Errorf("start = %+v, want the map's cell and no member cells", st)
	}
}

// A party larger than the open ground around it is counted, not hidden: the map
// is a ring of border with one open cell at its centre.
func TestStartMissionCountsACrowdedParty(t *testing.T) {
	// 17x17 leaves exactly one cell outside the eight-cell border ring.
	m := startMap(t, 17, 17, mapload.Cell{X: 8, Y: 8})
	_, st := mustStart(t, m, party(3))
	if st.Crowded != 2 {
		t.Fatalf("Crowded = %d, want 2 — one open cell and three members", st.Crowded)
	}
	for i, c := range st.Cells {
		if c != (mapload.Cell{X: 8, Y: 8}) {
			t.Errorf("member %d stands at %v, want the drop cell", i, c)
		}
	}
}

// TestAStartedPartysStockComesFromHisWornAndCarried is 0134's own widening
// of the starting-equipment hotfix: a member's sim.Stock entry is built from
// his OWN Worn array and Carried slice, not from his Weapon pointer —
// which the fold still reads, for his combat numbers alone (spec Out of
// scope) — and a member carrying neither produces NO entry at all, exactly
// as every party member in this tree did before this story.
//
// THREE MEMBERS, EACH A DIFFERENT ARM: the first carries neither Worn nor
// Carried and costs nothing; the second carries both and is asked back both
// ways; the third holds a Weapon alone, with no Worn set — the control that
// says the old automatic derivation is gone, not merely untested.
func TestAStartedPartysStockComesFromHisWornAndCarried(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})

	var worn [sim.EquipSlots]uint16
	worn[0], worn[4] = 0x0101, 0x0505
	p := []mapload.PartyMember{
		{Class: 100},
		{Class: 101, Worn: worn, Carried: []uint16{0x0e01}},
		{Class: 102, Weapon: &heroSword},
	}
	w, st := mustStart(t, m, p)

	if got := w.Stock(); len(got) != 1 {
		t.Fatalf("Stock() = %+v, want exactly one entry — members 0 and 2 cost nothing", got)
	}

	if eq, _ := w.Equipped(st.IDs[0]); eq != ([sim.EquipSlots]uint16{}) {
		t.Errorf("member 0 (neither Worn nor Carried) is equipped %v, want every slot empty", eq)
	}

	eq, ok := w.Equipped(st.IDs[1])
	if !ok || eq != worn {
		t.Errorf("member 1 equipped %v (present=%v), want %v", eq, ok, worn)
	}
	carried, ok := w.Carried(st.IDs[1])
	if !ok || len(carried) != 1 || carried[0] != 0x0e01 {
		t.Errorf("member 1 carries %v (present=%v), want [0x0e01]", carried, ok)
	}

	if eq, _ := w.Equipped(st.IDs[2]); eq != ([sim.EquipSlots]uint16{}) {
		t.Errorf("member 2 (a Weapon alone, no Worn) is equipped %v, want every slot empty — "+
			"the mint no longer derives an equip array from Weapon", eq)
	}
}

// An undefined difficulty is refused by the one function that owns that
// refusal, and the start reports nothing at all.
func TestStartMissionRefusesAnUndefinedDifficulty(t *testing.T) {
	w, st, err := mapload.StartMission(startMap(t, 40, 40), nil, mapload.Difficulty(9), party(1))
	if err == nil {
		t.Fatal("difficulty 9 was accepted")
	}
	if w != nil || st.Cells != nil {
		t.Errorf("a refused start still answered %v / %+v", w, st)
	}
}

// ---------------------------------------------------------------------------
// 0066 T7 — a started mission RUNS its script (AC-25, AC-26).
// ---------------------------------------------------------------------------

// alwaysScript is a program whose one trigger always holds: two constant checks
// preset to the same value, compared for equality, running the instants given.
//
// The checks are built here rather than compiled from a map, because what is
// under test is the JOIN — that the world a start returns evaluates a program at
// all — and not the binder, which pkg/mapload's own script tests already pin.
func alwaysScript(t *testing.T, latch int32, acts ...int32) *sim.Script {
	t.Helper()
	var trg sim.ScriptTrigger
	trg.Pairs[0] = sim.ScriptPair{Left: 0, Right: 1, Cmp: sim.ScriptCmpEQ, Used: true}
	trg.Once, trg.Latch = true, latch
	for i := range trg.Instants {
		trg.Instants[i] = sim.ScriptNone
	}
	for i, a := range acts {
		trg.Instants[i] = a
	}
	s, err := sim.NewScript(
		[]sim.ScriptCheck{
			{Op: sim.ScriptCheckConstant, Register: 0, Args: [10]int32{1}},
			{Op: sim.ScriptCheckConstant, Register: 1, Args: [10]int32{1}},
		},
		[]sim.ScriptInstant{{Op: sim.ScriptInstantWin}},
		[]sim.ScriptTrigger{trg})
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	return s
}

// AC-25. The world a scripted start returns evaluates the program: the trigger
// holds on the first pass, its latch is set, and its win instant decides the
// mission. Before T7 the world ran no script and every one of these stayed put
// however long it was stepped.
func TestStartMissionScriptedRunsTheScript(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	w, st, err := mapload.StartMissionScripted(m, nil, mapload.DifficultyNormal,
		party(1), alwaysScript(t, 3, 0))
	if err != nil {
		t.Fatalf("StartMissionScripted: %v", err)
	}
	if w.Script() == nil {
		t.Fatal("the world runs no script — the join is not made")
	}
	if st.Drop != (mapload.Cell{X: 17, Y: 20}) {
		t.Errorf("drop = %v, want the map's own cell — the start's answer must not move", st.Drop)
	}
	if w.ScriptLatched(3) || w.Outcome() != sim.OutcomeUndecided {
		t.Fatalf("before any step: latched=%v outcome=%v, want an untouched world",
			w.ScriptLatched(3), w.Outcome())
	}
	// One whole script cycle is 16 ticks, and the pass sits on a fixed phase of
	// it, so a cycle is what guarantees exactly one pass has run.
	for i := 0; i < 16; i++ {
		sim.Step(w, nil)
	}
	if !w.ScriptLatched(3) {
		t.Errorf("latch 3 is clear after a whole script cycle — no trigger was evaluated")
	}
	if got := w.Outcome(); got != sim.OutcomeWon {
		t.Errorf("outcome = %v, want %v — the trigger's win instant never ran",
			got, sim.OutcomeWon)
	}
}

// AC-26. With no program the scripted start builds exactly the world the plain
// start builds — the same seed, mode, bounds and plane — so attaching a script is
// the ONLY difference between the two entry points.
func TestStartMissionScriptedWithNoScriptBuildsThePlainWorld(t *testing.T) {
	for _, n := range []int{0, 1, 5} {
		m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
		want, wantSt := mustStart(t, m, party(n))
		got, gotSt, err := mapload.StartMissionScripted(m, nil, mapload.DifficultyNormal, party(n), nil)
		if err != nil {
			t.Fatalf("party of %d: StartMissionScripted: %v", n, err)
		}
		if got.Hash() != want.Hash() {
			t.Errorf("party of %d hashes %#016x, want the plain start's %#016x",
				n, got.Hash(), want.Hash())
		}
		if got.Script() != nil {
			t.Errorf("party of %d: a nil program produced a world running a script", n)
		}
		if !reflect.DeepEqual(gotSt, wantSt) {
			t.Errorf("party of %d reports %+v, want the plain start's %+v", n, gotSt, wantSt)
		}
	}
}

// An undefined difficulty is refused by the SAME function that refuses it for the
// plain start, before a script can be attached to anything.
func TestStartMissionScriptedRefusesAnUndefinedDifficulty(t *testing.T) {
	w, st, err := mapload.StartMissionScripted(startMap(t, 40, 40), nil,
		mapload.Difficulty(9), party(1), alwaysScript(t, 3, 0))
	if err == nil {
		t.Fatal("difficulty 9 was accepted")
	}
	if w != nil || st.Cells != nil {
		t.Errorf("a refused start still answered %v / %+v", w, st)
	}
}

// TestAPartyMemberCarriesTheUnresolvedEight is AC-6 and SC-5.
//
// THE COMPARISON IS AGAINST A PLACEMENT IN THE SAME WORLD, never against a
// literal 8 and 4. That is the whole point of the row: a change that moved the
// constructor's cadence would move both sides together and a test written
// against constants would still pass while the two populations stayed equal —
// but a change that moved the placement fallback and NOT the party's, or the
// reverse, is exactly the drift this asserts against, and only a comparison
// inside one world can see it.
func TestAPartyMemberCarriesTheUnresolvedEight(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	// One placement, on a key at or above the class-key floor. The start takes
	// NO TABLE, so it resolves to no definition — which is what makes it the
	// population a party member has to match.
	m.Units = []alm.Unit{{X: 0x0C80, Y: 0x0C80, ClassID: 0x40, ClassSubID: 1}}

	const members = 4
	w, _ := mustStart(t, m, party(members))

	ents := w.Entities()
	if len(ents) != 1+members {
		t.Fatalf("the world holds %d entities, want the placement and %d party members", len(ents), members)
	}
	placed := eightOf(ents[0])
	if (placed == eight{}) {
		t.Fatal("the placement carries eight zeros, so this comparison would hold for a party " +
			"that carried nothing either")
	}
	for i, e := range ents[1:] {
		if got := eightOf(e); got != placed {
			t.Errorf("party member %d carries %+v and the placement beside it carries %+v",
				i, got, placed)
		}
	}
}

func TestAStartedPartyMemberCarriesTheConstructorsPeriodsAndNoPool(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	w, _ := mustStart(t, m, party(3))

	ents := w.Entities()
	for i, e := range ents[len(m.Units):] {
		if e.HealthRegenPeriod != 100 || e.ManaRegenPeriod != 50 {
			t.Errorf("member %d carries periods %d/%d, want the constructor's 100/50",
				i, e.HealthRegenPeriod, e.ManaRegenPeriod)
		}
		if e.Mana != 0 || e.MaxMana != 0 {
			t.Errorf("member %d carries mana %d/%d, want 0/0 — DD-8 mints no pool", i, e.Mana, e.MaxMana)
		}
	}
}

func TestAHiredMemberIsMintedOutsideThePersistBand(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20}, mapload.Cell{X: 18, Y: 20})
	hero := data.NewCampaignHero(data.SkillBlade)
	party := []mapload.PartyMember{
		{Name: "Danath", Class: 3, Hero: hero, Weapon: &heroSword},
		// NPC03_1's own row values: type id 24, the "Human Mage" class record.
		{Name: "NPC03_1", Class: 24, Temporary: true, MercenaryType: 3, Hero: hero},
	}
	w, st := mustStart(t, m, party)

	byID := map[sim.EntityID]sim.Entity{}
	for _, e := range w.Entities() {
		byID[e.ID] = e
	}
	for i, want := range []struct {
		typeID  int32
		gainsXP bool
	}{{sim.HumanTypeID, true}, {24, false}} {
		e, ok := byID[st.IDs[i]]
		if !ok {
			t.Fatalf("member %d (%q) minted no entity", i, party[i].Name)
		}
		if e.TypeID != want.typeID {
			t.Errorf("%q minted with type id %d, want %d (in band: %v, want %v)",
				party[i].Name, e.TypeID, want.typeID,
				sim.InPersistBand(e.TypeID), sim.InPersistBand(want.typeID))
		}
		if e.GainsXP != want.gainsXP {
			t.Errorf("%q minted with GainsXP %v, want %v", party[i].Name, e.GainsXP, want.gainsXP)
		}
	}
}

func TestAStartedPartyMembersExperienceEqualsWhatHisLevelsAccountFor(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	hero := data.NewCampaignHero(data.SkillBlade)
	w, _ := mustStart(t, m, []mapload.PartyMember{{Class: 100, Hero: hero, Weapon: &heroSword}})

	e := lastEntity(t, w)
	want := hero.Reward()
	if e.SkillXP != want.SkillXP {
		t.Errorf("member minted with SkillXP %v, want his levels' own %v", e.SkillXP, want.SkillXP)
	}
	if e.Mind != want.Mind {
		t.Errorf("member minted with Mind %d, want his own capped %d", e.Mind, want.Mind)
	}
	if !e.GainsXP {
		t.Error("a party member does not gain experience; FR-3 says every member must")
	}
	// A hero holding no trained slot accounts for zero experience in every
	// one of them, so a wiring bug that dropped Reward() and left SkillXP at
	// its zero value would still pass every check above with a hero this
	// bare. The fixture hero trains SkillBlade at chargen's own level, and
	// this guard is what makes sure that training actually reached want.
	sum := int32(0)
	for _, xp := range want.SkillXP {
		sum += xp
	}
	if sum == 0 {
		t.Fatal("the fixture hero's own Reward() accounts for zero experience in every slot; " +
			"this comparison would hold even if the mint dropped Reward() entirely")
	}
}

func TestAStartedPartyMemberCreditsTheWeaponsSlotOrSkillGeneralBare(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})

	armed, _ := mustStart(t, m, []mapload.PartyMember{{
		Class: 100, Hero: data.NewCampaignHero(data.SkillBlade), Weapon: &heroSword,
	}})
	if e := lastEntity(t, armed); e.XPSlot != data.SkillBlade {
		t.Errorf("armed member credits slot %d, want the weapon's own Blade (%d)", e.XPSlot, data.SkillBlade)
	}

	bare, _ := mustStart(t, m, []mapload.PartyMember{{
		Class: 101, Hero: data.NewCampaignHero(data.SkillAxe),
	}})
	if e := lastEntity(t, bare); e.XPSlot != data.SkillGeneral {
		t.Errorf("bare member credits slot %d, want SkillGeneral (%d)", e.XPSlot, data.SkillGeneral)
	}
}

// ---------------------------------------------------------------------------
// 0086 T5 / AC-15 — the relation survives the start path's two rebuilds.
//
// Both entry points here BUILD A WORLD AND THEN BUILD ANOTHER ONE out of its
// parts, and a rebuild carries exactly what it names. The relation is the part
// whose loss is silent: the world still builds, still hashes, still ticks, and
// the only symptom is that nothing in it ever starts a fight — which is what
// every world looked like before the relation existed, so no older test in this
// tree can see it go. These are the tests that can.
// ---------------------------------------------------------------------------

// hostileStartMap is startMap with a two-record roster: slot 2 is hostile to
// slot 1 and slot 1 is hostile to nobody, which is a row no symmetric store
// could produce.
func hostileStartMap(t *testing.T, w, h int, cells ...mapload.Cell) *alm.Map {
	t.Helper()
	m := startMap(t, w, h, cells...)
	var self, monsters [16]uint16
	monsters[0] = 1
	m.Groups = []alm.Group{{Name: "Self", Relation: self}, {Name: "Monsters", Relation: monsters}}
	return m
}

// wantRelation is the store hostileStartMap's roster authors, as the byte form
// carries it.
func wantRelation() []byte {
	want := make([]byte, relBlockLen)
	want[1*relSlots+1] = 2 // slot 1's forced diagonal
	want[2*relSlots+1] = 1 // slot 2 -> slot 1, hostile
	want[2*relSlots+2] = 2 // slot 2's forced diagonal
	return want
}

// TestAStartCarriesTheMapsRelationThroughEveryRebuild is the fence on both
// rebuilds, at every shape that reaches them: with a party and without,
// scripted and not.
//
// The no-party StartMission case returns the loaded world untouched and would
// pass with both rebuilds broken; it is asked anyway, because what the other
// five measure is that they agree with it.
func TestAStartCarriesTheMapsRelationThroughEveryRebuild(t *testing.T) {
	m := hostileStartMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	want := wantRelation()

	scripted := func(p []mapload.PartyMember, s *sim.Script) *sim.World {
		t.Helper()
		w, _, err := mapload.StartMissionScripted(m, nil, mapload.DifficultyNormal, p, s)
		if err != nil {
			t.Fatalf("StartMissionScripted: %v", err)
		}
		return w
	}
	plain, err := mapload.FromALMWith(m, nil, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	withParty, _ := mustStart(t, m, party(2))
	noParty, _ := mustStart(t, m, nil)

	cases := []struct {
		what string
		w    *sim.World
	}{
		{"the plain load", plain},
		{"a start with no party", noParty},
		{"a start with a party", withParty},
		{"a scripted start with no party and no script", scripted(nil, nil)},
		{"a scripted start with a party and no script", scripted(party(2), nil)},
		{"a scripted start with a party and a script", scripted(party(2), alwaysScript(t, 3, 0))},
	}
	for _, c := range cases {
		if got := relBlock(t, c.w); !bytes.Equal(got, want) {
			i := firstDifference(got, want)
			t.Errorf("%s lost the map's relation at byte %d ([%d][%d]): got %d, want %d",
				c.what, i, i/relSlots, i%relSlots, got[i], want[i])
		}
	}
}

// TestAStartedMissionFightsWithoutBeingTold is the end-to-end statement on the
// path a mission actually takes: pkg/game starts every mission through
// StartMissionScripted, so this is the shape that decides whether a shipped map
// opens with its monsters moving or standing still.
//
// SINCE 0094 THIS WORLD HOLDS TWO CANDIDATES AND NOT ONE, and saying so is what
// keeps the test discriminating. The party stands on slot 1 as well, five cells
// from the hostile against the placement's two, so the hostile takes the nearer
// on cost alone — and a version that had lost the party from the matrix entirely
// would go on passing the first assertion unchanged. The third names the party
// and is what tells the two apart (0094 AC-8a).
//
// SINCE 0108 THE ONE-WAYNESS IS READ OFF THE MATRIX AND NOT OFF WHO ACQUIRED.
// A connecting blow flips both of a struck pair's cells, so the placement
// nobody was hostile to turns hostile the moment it is hit and does acquire
// by tick 64. The store is therefore checked directly, before a tick — a
// stricter reading of the same property, since a symmetric store now fails on
// tick zero rather than surviving to tick 64 — and the retaliation is asserted
// instead of forbidden.
func TestAStartedMissionFightsWithoutBeingTold(t *testing.T) {
	m := hostileStartMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	m.Units = []alm.Unit{
		{X: 20 << 8, Y: 20 << 8, ClassID: 7, UnitID: 1, GroupID: 1, Owner: 1},
		{X: 22 << 8, Y: 20 << 8, ClassID: 7, UnitID: 2, GroupID: 2, Owner: 2},
	}
	w, _, err := mapload.StartMissionScripted(m, nil, mapload.DifficultyNormal, party(1), nil)
	if err != nil {
		t.Fatalf("StartMissionScripted: %v", err)
	}
	// The store the start rebuilt, before a tick: one way and one way only.
	if rel := w.Relations(); rel.Byte(2, 1) != 1 || rel.Byte(1, 2) != 0 {
		t.Fatalf("the started store is [2][1]=%d [1][2]=%d, want 1 and 0 — a started "+
			"mission carries the map's relation and this one is symmetric",
			rel.Byte(2, 1), rel.Byte(1, 2))
	}
	for i := 0; i < 96; i++ {
		sim.Step(w, nil)
	}
	got := w.Entities()
	if !got[1].HasAttackTarget || got[1].AttackTarget != got[0].ID {
		t.Errorf("the hostile placement holds target %v/%v, want entity %d — a started "+
			"mission carries no relation and nothing acquires",
			got[1].AttackTarget, got[1].HasAttackTarget, got[0].ID)
	}
	// And the struck side fought back (0108): the blow flipped its blank cell.
	// And the direction already hostile is byte-identical — a blow into a cell
	// that already carries bit 0 writes nothing.
	if rel := w.Relations(); rel.Byte(1, 2) != 1 || rel.Byte(2, 1) != 1 {
		t.Errorf("the pair ends [1][2]=%d [2][1]=%d, want 1 and 1 — the struck cell gains "+
			"bit 0 and the one already hostile is left alone",
			rel.Byte(1, 2), rel.Byte(2, 1))
	}
	if !got[0].HasAttackTarget {
		t.Errorf("the struck placement acquired nothing — it was hit, so it is hostile now")
	}
	if p := got[2]; p.Owner != sim.SelfSlot {
		t.Fatalf("the party member owns slot %d, want %d — the cost ordering above "+
			"decides nothing unless it is choosing between two candidates",
			p.Owner, uint32(sim.SelfSlot))
	}
	// The hostile is at (22,20), the placement at (20,20) and the party at
	// (17,20): 2 against 5, same preference cell, so the nearer wins on distance.
	if got[1].AttackTarget == got[2].ID {
		t.Errorf("the hostile took the party at Chebyshev 5 over the placement at 2")
	}
}

// ---------------------------------------------------------------------------
// The ground-sack list survives every rebuild a mission start makes
// (hotfix).
// ---------------------------------------------------------------------------

// lootStartMap is startMap with a two-record loot section: one ground record
// and one stock record, so a rebuild that placed the wrong kind is as visible
// as one that placed none.
func lootStartMap(t *testing.T, w, h int, cells ...mapload.Cell) *alm.Map {
	t.Helper()
	m := startMap(t, w, h, cells...)
	m.FormatVersion = 990
	m.Meta = alm.Meta{Word2C: 2}
	m.LootSection = alm.LootSection{Body: llJoin(
		llHead(1, 0, llCell(5), llCell(6), 100),
		llElement(0x0205, 3, 0),
		llHead(0, 7, llCell(8), llCell(9), 999),
	)}
	return m
}

// TestAStartCarriesTheMapsGroundSacksThroughEveryRebuild is the fence beside
// TestAStartCarriesTheMapsRelationThroughEveryRebuild, at the same six shapes,
// and it exists for the same reason stated one function up: a rebuild names
// what it carries, so what a later story adds to a world is lost unless every
// rebuild is told about it. The relation was that lesson once. The ground-sack
// list, added by 0103 after both rebuilds were written, was the same lesson
// again — StartMission's party branch and StartMissionScripted both rebuilt
// through sim.NewRelatedWorld, which names no sacks, so EVERY SHIPPED MISSION
// opened with an empty list while a plain load carried the map's own records.
//
// It was silent in the worst way, exactly as the relation would have been.
//
// THE NO-PARTY StartMission CASE RETURNS THE LOADED WORLD UNTOUCHED and would
// pass with both rebuilds broken. It is asked anyway, for its neighbour's
// reason: what the other five measure is that they agree with it.
func TestAStartCarriesTheMapsGroundSacksThroughEveryRebuild(t *testing.T) {
	m := lootStartMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})

	scripted := func(p []mapload.PartyMember, s *sim.Script) *sim.World {
		t.Helper()
		w, _, err := mapload.StartMissionScripted(m, nil, mapload.DifficultyNormal, p, s)
		if err != nil {
			t.Fatalf("StartMissionScripted: %v", err)
		}
		return w
	}
	plain, err := mapload.FromALMWith(m, nil, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	withParty, _ := mustStart(t, m, party(2))
	noParty, _ := mustStart(t, m, nil)

	// The plain load's answer is the oracle rather than a literal, so this test
	// states "every rebuild agrees with the loader" and cannot drift from what
	// the loader means by a ground record.
	want := plain.Sacks()
	if len(want) != 1 || want[0].X != 5 || want[0].Y != 6 || want[0].Gold != 100 {
		t.Fatalf("the fixture's plain load is %+v, want one ground sack at (5,6) with "+
			"100 gold — the rest of this test measures nothing otherwise", want)
	}

	cases := []struct {
		what string
		w    *sim.World
	}{
		{"a start with no party", noParty},
		{"a start with a party", withParty},
		{"a scripted start with no party and no script", scripted(nil, nil)},
		{"a scripted start with a party and no script", scripted(party(2), nil)},
		{"a scripted start with a party and a script", scripted(party(2), alwaysScript(t, 3, 0))},
	}
	for _, c := range cases {
		if got := c.w.Sacks(); !reflect.DeepEqual(got, want) {
			t.Errorf("%s carries %+v, want the loader's own %+v", c.what, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// The map's own STOCK survives every rebuild a mission start makes (defect
// found at the orchestrator seat before landing).
// ---------------------------------------------------------------------------

// stockStartMap is startMap with two placed units and a three-record loot
// section: one ground record, one stock record naming the second unit and
// carrying two elements, and one stock record naming nobody. So a rebuild
// that carried the wrong record, or carried one onto the wrong entity, is as
// visible as one that carried none.
//
// It is its own fixture rather than a widening of lootStartMap, because that
// one's stock record carries no element and names no unit — everything this
// test measures would be zero against zero there.
func stockStartMap(t *testing.T, w, h int, cells ...mapload.Cell) *alm.Map {
	t.Helper()
	m := startMap(t, w, h, cells...)
	m.Units = []alm.Unit{
		{X: 10<<8 | 0x80, Y: 10<<8 | 0x80, ClassID: 3, UnitID: 4},
		{X: 12<<8 | 0x80, Y: 12<<8 | 0x80, ClassID: 3, UnitID: 7},
	}
	m.FormatVersion = 990
	m.Meta = alm.Meta{Word2C: 3}
	m.LootSection = alm.LootSection{Body: llJoin(
		llHead(1, 0, llCell(5), llCell(6), 100),
		llElement(0x0205, 0, 0),
		llHead(2, 7, llCell(8), llCell(9), 999),
		llElement(0x0102, 0, 0),
		llElement(0x0e06, 0, 0),
		llHead(1, 99, llCell(3), llCell(3), 5),
		llElement(0x0203, 0, 0),
	)}
	return m
}

// TestAStartCarriesTheMapsStockThroughEveryRebuild is the fence beside
// TestAStartCarriesTheMapsGroundSacksThroughEveryRebuild, at the same shapes,
// and it is the THIRD time the lesson stated in that function has had to be
// written down: A REBUILD CARRIES ONLY WHAT IT NAMES. The relation was the
// lesson once, the ground-sack list a second time, and 0112's containers a
// third — added after both rebuilds were written, so both rebuilds dropped
// them, and every shipped mission opened with 43 stocked people carrying
// nothing while a plain load carried the map's own records.
//
// IT WAS SILENT IN THE SAME WAY BOTH TIMES, which is why the class keeps
// recurring rather than being learnt once. Nothing in this build DRAWS a
// stocked actor's container, so "nobody carries anything" is indistinguishable
// from "this map stocks nobody" — exactly as an empty sack list was
// indistinguishable from a map that authored no loot. The measurement that
// found it was made through StartMission on the shipped mission 10, whose one
// stock record carries three items for a person at (36,51): a plain load put
// them on him and the mission path did not.
//
// THE PURSES ARE COMPARED TOO, and that half currently passes trivially: no
// load path can write a purse, so both sides are zero. It is asked anyway,
// because it is the ONE assertion here that would catch the FOURTH instance —
// the day a story authors gold at load and a rebuild does not name it, this
// goes red instead of a mission shipping wrong.
//
// THE NO-PARTY StartMission CASE RETURNS THE LOADED WORLD UNTOUCHED and would
// pass with both rebuilds broken. It is asked for its neighbour's reason: what
// the other four measure is that they agree with it.
func TestAStartCarriesTheMapsStockThroughEveryRebuild(t *testing.T) {
	m := stockStartMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})

	scripted := func(p []mapload.PartyMember, s *sim.Script) *sim.World {
		t.Helper()
		w, _, err := mapload.StartMissionScripted(m, nil, mapload.DifficultyNormal, p, s)
		if err != nil {
			t.Fatalf("StartMissionScripted: %v", err)
		}
		return w
	}
	plain, err := mapload.FromALMWith(m, nil, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	withParty, _ := mustStart(t, m, party(2))
	noParty, _ := mustStart(t, m, nil)

	// The plain load's answer is the oracle rather than a literal, so this test
	// states "every rebuild agrees with the loader" and cannot drift from what
	// the loader means by a stock record.
	want := plain.Stock()
	if len(want) != 1 || want[0].ID != 1 || len(want[0].Items) != 2 ||
		want[0].Items[0] != 0x0102 || want[0].Items[1] != 0x0e06 {
		t.Fatalf("the fixture's plain load stocks %+v, want entity 1 carrying "+
			"[0x0102 0x0e06] — the rest of this test measures nothing otherwise", want)
	}

	cases := []struct {
		what string
		w    *sim.World
	}{
		{"a start with no party", noParty},
		{"a start with a party", withParty},
		{"a scripted start with no party and no script", scripted(nil, nil)},
		{"a scripted start with a party and no script", scripted(party(2), nil)},
		{"a scripted start with a party and a script", scripted(party(2), alwaysScript(t, 3, 0))},
	}
	for _, c := range cases {
		if got := c.w.Stock(); !reflect.DeepEqual(got, want) {
			t.Errorf("%s stocks %+v, want the loader's own %+v", c.what, got, want)
		}
		// Read back the other way too, through the per-entity door a consumer
		// actually uses: a rebuild that produced the right list against the
		// wrong entity would satisfy neither, but only this says which.
		if got, ok := c.w.Carried(1); !ok || !reflect.DeepEqual(got, want[0].Items) {
			t.Errorf("%s: entity 1 carries %v (present=%v), want %v",
				c.what, got, ok, want[0].Items)
		}
		// The party member the rebuild ADDS carries nothing: a start mints him,
		// no record names him, and a rebuild that spread the map's stock over
		// the wrong index would show up here first.
		if got, ok := c.w.Carried(2); ok && len(got) != 0 {
			t.Errorf("%s: entity 2 carries %v, want nothing — no record names it", c.what, got)
		}
		for slot := uint32(0); slot < 4; slot++ {
			if got, wantP := c.w.Purse(slot), plain.Purse(slot); got != wantP {
				t.Errorf("%s: purse %d is %d, want the loader's own %d", c.what, slot, got, wantP)
			}
		}
	}
}
