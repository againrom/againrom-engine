package game

// The affordance's data half, tested at the two seams T6 adds (AC-10): the
// dispatcher that turns a selected spell into a cast command and an
// unselected one into the attack the seam always issued, and the pure
// function that turns a mask and a table into the book a unit is offered.

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestASpellSelectedRoutesToCastAndZeroRoutesToTheAttackTheSeamAlwaysIssued
// is AC-10's own wording: "with a spell the seam appends the cast; with 0
// the attack it always did" — attackOrCast is the ui.MapAttack the loader
// hands the front-end, measured against strike's own queue the way
// attack_test.go's TestAnAttackOrderBecomesACommandInTheQueueTheOrdersUse
// measures strike itself.
func TestASpellSelectedRoutesToCastAndZeroRoutesToTheAttackTheSeamAlwaysIssued(t *testing.T) {
	b := sim.Bounds{Width: 12, Height: 12}
	w, err := sim.NewWorld(1, b, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 0, X: 1, Y: 1, HP: mapload.SpawnHP, MaxHP: mapload.SpawnHP},
		{ID: 1, X: 3, Y: 1, HP: mapload.SpawnHP, MaxHP: mapload.SpawnHP},
		{ID: 2, X: 5, Y: 1, HP: mapload.SpawnHP, MaxHP: mapload.SpawnHP},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	mw := &mapWorld{world: w, swing: make(map[sim.EntityID]int),
		castRun: make(map[sim.EntityID]castRun), commanded: make(map[sim.EntityID]bool)}

	before := w.Hash()
	mw.attackOrCast(0, 2, 5, 0, 0, false) // a spell selected: a cast, naming spell 5
	mw.attackOrCast(1, 2, 0, 0, 0, false) // none selected: the attack it always did

	want := []sim.Command{
		{Kind: sim.KindCast, Entity: 0, X: 2, Y: 5},
		{Kind: sim.KindAttack, Entity: 1, X: 2},
	}
	if !reflect.DeepEqual(mw.pending, want) {
		t.Errorf("the queue holds\n %+v\nwant\n %+v", mw.pending, want)
	}
	// The mechanism strike's own test already pins: issuing with no advance
	// behind it moves no world field and no digest.
	if got := w.Hash(); got != before {
		t.Errorf("the world hashes %#016x after two orders and %#016x before them", got, before)
	}
	// castAt marks exactly as strike does (spec: "marking the entity
	// commanded exactly as strike does").
	for _, id := range []sim.EntityID{0, 1} {
		if !mw.commanded[id] {
			t.Errorf("entity %d is not marked commanded after an order through attackOrCast", id)
		}
	}
}

// TestTheBookOfferedForAUnitIsItsMaskIntersectedWithTheTableInTableOrder is
// AC-10's other half: "the book offered for a unit is exactly its mask
// intersected with the table, in table order" — including a unit whose mask
// names a spell the table does not hold, which must not appear, and a unit
// with an empty mask, from which nothing appears.
func TestTheBookOfferedForAUnitIsItsMaskIntersectedWithTheTableInTableOrder(t *testing.T) {
	// The table's own order is deliberately not ascending by id, so an
	// implementation reading id order rather than table order would be
	// caught: 12 before 6.
	table := []sim.SpellRule{
		{ID: 1, Damaging: true, TargetsUnit: true},
		{ID: 12, Damaging: true, TargetsUnit: true},
		{ID: 6, Damaging: true, TargetsUnit: true},
		{ID: 18, Damaging: true, TargetsUnit: true},
	}
	names := map[uint16]string{1: "Fire Arrow", 6: "Heal", 12: "Ice Bolt", 18: "Cure"}
	w := testSpellPopupWords()

	t.Run("the mask intersected with the table, in table order", func(t *testing.T) {
		// 266306 is the shipped ManMage_Staff value (spec AC-5a): spells 1,
		// 6, 12 and 18.
		e := sim.Entity{KnownSpells: 266306}
		got := spellbookOf(sim.Rules{}, e, table, names, w, nil)
		want := []ui.SpellEntry{
			{ID: 1, Name: "Fire Arrow", Info: []string{"Fire Arrow", "Mana cost: 0"}},
			{ID: 12, Name: "Ice Bolt", Info: []string{"Ice Bolt", "Mana cost: 0"}},
			{ID: 6, Name: "Heal", Info: []string{"Heal", "Mana cost: 0"}},
			{ID: 18, Name: "Cure", Info: []string{"Cure", "Mana cost: 0"}},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("book = %+v, want %+v", got, want)
		}
	})

	t.Run("a mask bit naming a spell the table does not hold does not appear", func(t *testing.T) {
		// Bit 1 (spell 1, in the table) and bit 2 (spell 2, not in the table).
		e := sim.Entity{KnownSpells: (1 << 1) | (1 << 2)}
		got := spellbookOf(sim.Rules{}, e, table, names, w, nil)
		want := []ui.SpellEntry{{ID: 1, Name: "Fire Arrow",
			Info: []string{"Fire Arrow", "Mana cost: 0"}}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("book = %+v, want %+v — spell 2 names no row of the table", got, want)
		}
	})

	t.Run("an empty mask offers nothing", func(t *testing.T) {
		got := spellbookOf(sim.Rules{}, sim.Entity{}, table, names, w, nil)
		if len(got) != 0 {
			t.Errorf("book = %+v, want none", got)
		}
	})
}

func TestTheAutocastSeamAppendsOneCommandToTheQueueTheOrdersUse(t *testing.T) {
	b := sim.Bounds{Width: 12, Height: 12}
	w, err := sim.NewWorld(1, b, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 1, X: 1, Y: 1, HP: mapload.SpawnHP, MaxHP: mapload.SpawnHP},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	mw := &mapWorld{world: w, swing: make(map[sim.EntityID]int),
		castRun: make(map[sim.EntityID]castRun), commanded: make(map[sim.EntityID]bool)}

	before := w.Hash()
	mw.setAutocast(1, 6)
	mw.setAutocast(1, 0)

	want := []sim.Command{
		{Kind: sim.KindAutocast, Entity: 1, X: 6},
		{Kind: sim.KindAutocast, Entity: 1, X: 0},
	}
	if !reflect.DeepEqual(mw.pending, want) {
		t.Errorf("the queue holds\n %+v\nwant\n %+v", mw.pending, want)
	}
	if got := w.Hash(); got != before {
		t.Errorf("the world hashes %#016x after two toggles and %#016x before them", got, before)
	}
	// AND IT MARKS NOTHING COMMANDED. It did, and mw.commanded is permanent:
	// commands() drops the scripted command track for every entity in it, so
	// arming an autocast took the unit off the mission script.
	if mw.commanded[1] {
		t.Error("a toggle through setAutocast marked entity 1 commanded")
	}
}

func TestTheBookCarriesTheAutocastFlagAndTheSpellsOwnLines(t *testing.T) {
	table := []sim.SpellRule{
		{ID: 1, ManaCost: 3, School: 1, MaxRange: 7, DamageMin: 4, DamageMax: 8,
			TargetsUnit: true, Damaging: true},
		{ID: 6, ManaCost: 10, School: 5, MaxRange: 6, DamageMin: 10, DamageMax: 20,
			TargetsUnit: true, Restorative: true},
	}
	names := map[uint16]string{1: "Fire Arrow", 6: "Heal"}
	// Mind 30 and school skill 0 put the record at power 0 (TEXT-097).
	e := sim.Entity{KnownSpells: (1 << 1) | (1 << 6), AutoSpell: 6, Mind: 30}

	got := spellbookOf(sim.Rules{}, e, table, names, testSpellPopupWords(), nil)
	want := []ui.SpellEntry{
		{ID: 1, Name: "Fire Arrow",
			Info: []string{"Fire Arrow", "Mana cost: 3", "Damage: 4-8", "Range: 7"}},
		// Heal carries damage columns, and the record fills its damage bytes
		// from them whatever the effect arm (TEXT-096).
		{ID: 6, Name: "Heal", Autocast: true,
			Info: []string{"Heal", "Mana cost: 10", "Damage: 10-20", "Range: 6"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("book =\n %+v\nwant\n %+v", got, want)
	}
}

func TestCustomSpellTableKeepsItsCollectionNames(t *testing.T) {
	src := installFixture{SpellBookNamesTextPath: textFile(24, map[int]string{
		0: "installed first#detail", 5: "installed sixth#detail",
	})}
	bookNames := LoadInstallWords(src).Words().SpellBookNames
	if bookNames[23] != "installed sixth" {
		t.Fatalf("cell mapping was not exercised: %q", bookNames[23])
	}
	spells := make(shopBookSpells, 4)
	spells[1].name, spells[2].name, spells[3].name = "custom first", "custom second", "custom third"
	names := spellNamesFrom(&mapload.Table{Spells: spells}, bookNames)
	for id, want := range map[uint16]string{1: "custom first", 2: "custom second", 3: "custom third"} {
		if names[id] != want {
			t.Fatalf("custom spell %d name = %q, want %q", id, names[id], want)
		}
	}
	partial := LoadInstallWords(installFixture{SpellBookNamesTextPath: textFile(23, map[int]string{0: "partial"})}).Words()
	if partial.SpellBookNames != ([29]string{}) {
		t.Fatalf("incomplete book table imposed fixed cell names: %q", partial.SpellBookNames)
	}
}

// TestTheSameOpenSpellbookEntryUsesTheActorsCurrentCharacteristics exercises
// the actual projection pushSpellbook feeds to the popup. The book is rebuilt
// twice for the same owner and same installed row; only the actor's live school
// and Mind move. A cache keyed by spell id or book owner would return the first
// lines on the second push, while recomputing row-independent formulas in this
// package would duplicate the simulation owner this test deliberately calls.
func TestTheSameOpenSpellbookEntryUsesTheActorsCurrentCharacteristics(t *testing.T) {
	rule := sim.SpellRule{ID: 6, ManaCost: 10, School: 2, MaxRange: 6,
		DamageMin: 10, DamageMax: 20, TargetsUnit: true, Restorative: true}
	names := map[uint16]string{6: "Heal"}
	w := testSpellPopupWords()
	e := sim.Entity{ID: 7, KnownSpells: 1 << 6, Mind: 24}
	e.Skill[2], e.SkillXP[2] = 4, 90
	before := spellbookOf(sim.Rules{}, e, []sim.SpellRule{rule}, names, w, nil)

	// The same entity, book and popup remain selected. This is the canonical
	// state the next per-frame push observes after training/recomputation.
	e.Mind, e.Skill[2], e.SkillXP[2] = 60, 12, 240
	after := spellbookOf(sim.Rules{}, e, []sim.SpellRule{rule}, names, w, nil)
	if len(before) != 1 || len(after) != 1 {
		t.Fatalf("book lengths before=%d after=%d, want one entry", len(before), len(after))
	}
	if reflect.DeepEqual(before[0].Info, after[0].Info) {
		t.Fatalf("the same popup lines remained stale after actor growth: %q", after[0].Info)
	}
	wantAfter := spellInfoLines(rule, sim.SpellCharacteristicsFor(sim.Rules{}, e, rule), "Heal", &w)
	if !reflect.DeepEqual(after[0].Info, wantAfter) {
		t.Errorf("refreshed popup lines = %q, canonical current projection = %q", after[0].Info, wantAfter)
	}
	if before[0].Info[1] != "Mana cost: 10" || after[0].Info[1] != "Mana cost: 10" {
		t.Errorf("flat mana cost moved: before %q after %q", before[0].Info[1], after[0].Info[1])
	}
}

func TestASpellTheTableCouldNotNameStillStatesItsNumbers(t *testing.T) {
	rule := sim.SpellRule{ID: 1, ManaCost: 3, School: 1, MaxRange: 7,
		DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}
	w := testSpellPopupWords()
	got := spellInfoLines(rule, sim.SpellCharacteristicsFor(sim.Rules{}, sim.Entity{Mind: 30}, rule), "", &w)
	want := []string{"spell", "Mana cost: 3", "Damage: 4-8", "Range: 7"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %q, want %q", got, want)
	}
}

func TestSpellDurationIsTicksTimesOneSixteenthWithOneDecimal(t *testing.T) {
	w := testSpellPopupWords()
	rule := sim.SpellRule{ID: 20, TargetsUnit: true}
	for _, tc := range []struct {
		ticks uint16
		want  string
	}{{1, "Duration:   0.1"}, {16, "Duration:   1.0"}, {163, "Duration:  10.2"}} {
		got := spellInfoLines(rule, sim.SpellCharacteristics{Duration: tc.ticks}, "Stone Curse", &w)
		if got[len(got)-1] != tc.want {
			t.Errorf("%d ticks display = %q, want %q", tc.ticks, got[len(got)-1], tc.want)
		}
	}
	ru := LoadInstallWords(installFixture{
		LanguagePath: []byte("russian 1"),
		MainTextPath: textFile(125, map[int]string{spellLabelDuration: "\x84"}),
	}).Words()
	got := spellInfoLines(rule, sim.SpellCharacteristics{Duration: 163}, "Stone Curse", &ru)
	if want := "\x84:  10.2"; got[len(got)-1] != want {
		t.Errorf("RU duration = %q, want installed caption and one decimal %q", got[len(got)-1], want)
	}
}

// testSpellPopupWords is TEXT-HOVERTEXT-052's four bound captions, in words
// distinct from the labels a defect would leak in from — mana cost, damage,
// range and duration — so a test failure names which one moved.
func testSpellPopupWords() ui.Words {
	var w ui.Words
	w.Hover[spellLabelManaCost] = "Mana cost"
	w.Hover[spellLabelDamage] = "Damage"
	w.Hover[spellLabelRange] = "Range"
	w.Hover[spellLabelDuration] = "Duration"
	return w
}

func TestTheMarkingSpellsSchoolIsResolvedOffTheWorldsOwnTable(t *testing.T) {
	b := sim.Bounds{Width: 12, Height: 12}
	var entities []sim.Entity
	for i, spell := range []uint8{0, 1, 6, 27} {
		entities = append(entities, sim.Entity{ID: sim.EntityID(i + 1), X: int32(i + 1), Y: 1,
			HP: mapload.SpawnHP, MaxHP: mapload.SpawnHP, SpellFX: 8, SpellFXSpell: spell})
	}
	w, err := sim.NewSpelledWorld(1, b, sim.ModeCanonical, nil, entities, nil,
		[]sim.SpellRule{
			{ID: 1, School: 1, MaxRange: 7, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true},
			{ID: 6, School: 5, MaxRange: 6, DamageMin: 10, DamageMax: 20, TargetsUnit: true, Restorative: true},
		})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	mw := &mapWorld{world: w}

	for _, tc := range []struct {
		id   uint16
		want int
	}{
		{0, 0},  // no mark
		{1, 1},  // fire
		{6, 5},  // life
		{27, 0}, // an id this table holds no row for
	} {
		if got := spellSchool(tc.id, w.Spells()); got != tc.want {
			t.Errorf("spellSchool(%d) = %d, want %d", tc.id, got, tc.want)
		}
	}
	before := w.Hash()
	draws := mw.entityDraws()
	if len(draws) != len(entities) {
		t.Fatalf("draw count = %d, want %d", len(draws), len(entities))
	}
	for i, want := range []int{0, 1, 5, 0} {
		if draws[i].SpellFX != 8 || draws[i].SpellFXSchool != want {
			t.Errorf("actor %d mark = %d/%d, want 8/%d", draws[i].ID, draws[i].SpellFX, draws[i].SpellFXSchool, want)
		}
	}
	if w.Hash() != before {
		t.Fatal("effect school projection changed hashed state")
	}
}

// The popup folds over every selected knower: the mana value is the last
// knower's and range is the minimum and maximum (TEXT-080, TEXT-081).
func TestSelectedSpellbookFoldsEveryKnowerAndKeepsPrimaryUnavailableEntries(t *testing.T) {
	table := []sim.SpellRule{
		{ID: 26, ManaCost: 30, MaxRange: 9},
		{ID: 6, ManaCost: 10, MaxRange: 6, TargetsUnit: true, Restorative: true},
		{ID: 89, ManaCost: 2, MaxRange: 3},
		{ID: 1, ManaCost: 3, MaxRange: 7, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true},
	}
	names := map[uint16]string{26: "Teleport", 6: "Heal", 89: "Custom", 1: "Fire Arrow"}
	primary := sim.Entity{KnownSpells: 1 << 1, AutoSpell: 26, Mind: 30}
	knower := sim.Entity{KnownSpells: 1<<26 | 1<<6, AutoSpell: 6, Mind: 30, Book: sim.Spellbook{State: sim.BookPresent}}
	knower.Book.Slots[25] = sim.BookSpell{Range: 13, ManaCost: 7}
	knower.Book.Slots[5] = sim.BookSpell{Range: 8, ManaCost: 4}
	later := knower
	later.Book.Slots[25] = sim.BookSpell{Range: 90, ManaCost: 90}
	later.Book.Slots[5] = sim.BookSpell{Range: 91, ManaCost: 91}
	selected := []sim.Entity{primary, knower, later}
	pic := &image.RGBA{}
	var icons []uint16
	icon := func(id uint16) *image.RGBA {
		icons = append(icons, id)
		return pic
	}
	want := []ui.SpellEntry{
		{ID: 26, Name: "Teleport", PointTarget: true, Autocast: true, Icon: pic,
			Info: []string{"Teleport", "Mana cost: 90", "Range: 13-90"}},
		{ID: 6, Name: "Heal", Icon: pic, Info: []string{"Heal", "Mana cost: 91", "Range: 8-91"}},
		{ID: 89, Name: "Custom", Unavailable: true, Icon: pic, Info: []string{"Custom", "Mana cost: 2", "Range: 3"}},
		{ID: 1, Name: "Fire Arrow", Icon: pic, Info: []string{"Fire Arrow", "Mana cost: 3", "Damage: 4-8", "Range: 7"}},
	}
	got, fixed := selectedSpellbook(sim.Rules{}, selected, table, names, testSpellPopupWords(), icon)
	if fixed || !reflect.DeepEqual(got, want) {
		t.Fatalf("union book = %+v fixed=%v, want %+v custom", got, fixed, want)
	}
	if !reflect.DeepEqual(icons, []uint16{26, 6, 89, 1}) {
		t.Fatalf("icon lookups = %v, want one per catalog entry", icons)
	}
	got, fixed = selectedSpellbook(sim.Rules{}, selected[:1], table, names, testSpellPopupWords(), nil)
	want[0] = ui.SpellEntry{ID: 26, Name: "Teleport", Unavailable: true,
		Info: []string{"Teleport", "Mana cost: 30", "Range: 9"}}
	want[1] = ui.SpellEntry{ID: 6, Name: "Heal", Unavailable: true,
		Info: []string{"Heal", "Mana cost: 10", "Range: 6"}}
	want[2].Icon, want[3].Icon = nil, nil
	if fixed || !reflect.DeepEqual(got, want) {
		t.Fatalf("primary book = %+v fixed=%v, want %+v custom", got, fixed, want)
	}
	selected[0].AutoSpell = 6
	selected[1].Book.Slots[5] = sim.BookSpell{Range: 11, ManaCost: 5}
	got, _ = selectedSpellbook(sim.Rules{}, selected, table, names, testSpellPopupWords(), nil)
	want[0] = ui.SpellEntry{ID: 26, Name: "Teleport", PointTarget: true,
		Info: []string{"Teleport", "Mana cost: 90", "Range: 13-90"}}
	want[1] = ui.SpellEntry{ID: 6, Name: "Heal", Autocast: true,
		Info: []string{"Heal", "Mana cost: 91", "Range: 11-91"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("changed book = %+v, want %+v", got, want)
	}
}

func spellClassProjectionFixture(t *testing.T) *mapWorld {
	t.Helper()
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: 12, Height: 12}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10,
			TypeID: sim.HumanTypeID, Class: 97, KnownSpells: 1 << 1}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{2}}})
	if err != nil {
		t.Fatal(err)
	}
	state := &Mission{}
	state.Start.Roster = map[sim.EntityID]mapload.PartyMember{7: {ID: "roster", Mage: true, Class: 77}}
	return &mapWorld{world: w, mission: &missionNotices{
		list: data.BodyList{"unarmed", "swordsman"}, state: state,
	}}
}

func requireSpellClientClass(t *testing.T, mw *mapWorld, want int32) {
	t.Helper()
	before := mw.world.Hash()
	if got := mw.spellClientClass(7, 97); got != want {
		t.Fatalf("client class = %d, want %d", got, want)
	}
	draws := mw.entityDraws()
	if len(draws) != 1 || draws[0].CastCapable != (want == 23 || want == 24) || !draws[0].SpellStateKnown {
		t.Fatalf("class %d draw = %+v", want, draws)
	}
	if mw.world.Hash() != before {
		t.Fatal("client class or entity projection changed World hash")
	}
}

func TestSpellClientClassReadsCurrentRosterEquipment(t *testing.T) {
	mw := spellClassProjectionFixture(t)
	requireSpellClientClass(t, mw, 23)
	sim.Step(mw.world, []sim.Command{sim.Equip(7, 0, 1)})
	requireSpellClientClass(t, mw, 3)
	sim.Step(mw.world, []sim.Command{sim.Unequip(7, 1)})
	requireSpellClientClass(t, mw, 23)
}

func TestSpellClientClassReadsStartingWeaponMaterialization(t *testing.T) {
	for _, inParty := range []bool{false, true} {
		t.Run(map[bool]string{false: "roster", true: "party"}[inParty], func(t *testing.T) {
			mw := spellClassProjectionFixture(t)
			member := mw.mission.state.Start.Roster[7]
			member.Weapon = &data.Weapon{Code: 2}
			if inParty {
				mw.mission.ids = []sim.EntityID{7}
				mw.mission.party = []mapload.PartyMember{member}
			} else {
				mw.mission.state.Start.Roster[7] = member
			}
			requireSpellClientClass(t, mw, 3)
			if inParty {
				mw.mission.party[0].WeaponMaterialized = true
			} else {
				member.WeaponMaterialized = true
				mw.mission.state.Start.Roster[7] = member
			}
			requireSpellClientClass(t, mw, 23)
		})
	}
}

func TestSpellClientClassKeepsHiredAndFallbackClasses(t *testing.T) {
	t.Run("hired", func(t *testing.T) {
		mw := spellClassProjectionFixture(t)
		mw.mission.state.Start.Roster[7] = mapload.PartyMember{ID: "hire", MercenaryType: 1, Class: 24}
		requireSpellClientClass(t, mw, 24)
		sim.Step(mw.world, []sim.Command{sim.Equip(7, 0, 1)})
		requireSpellClientClass(t, mw, 24)
		sim.Step(mw.world, []sim.Command{sim.Unequip(7, 1)})
		requireSpellClientClass(t, mw, 24)
		mw.mission.state.Start.Roster[7] = mapload.PartyMember{ID: "hire", MercenaryType: 1, Class: 31}
		requireSpellClientClass(t, mw, 31)
	})
	t.Run("missing roster", func(t *testing.T) {
		mw := spellClassProjectionFixture(t)
		delete(mw.mission.state.Start.Roster, 7)
		requireSpellClientClass(t, mw, 97)
	})
	t.Run("missing mission", func(t *testing.T) {
		mw := spellClassProjectionFixture(t)
		mw.mission = nil
		requireSpellClientClass(t, mw, 97)
	})
	t.Run("unresolved body list", func(t *testing.T) {
		mw := spellClassProjectionFixture(t)
		mw.mission.list = nil
		requireSpellClientClass(t, mw, 77)
		sim.Step(mw.world, []sim.Command{sim.Equip(7, 0, 1)})
		requireSpellClientClass(t, mw, 77)
	})
}

func TestSpellClientClassPrefersPartyBeforeRoster(t *testing.T) {
	mw := spellClassProjectionFixture(t)
	mw.mission.ids = []sim.EntityID{7}
	mw.mission.party = []mapload.PartyMember{{ID: "party", Mage: true, Class: 75}}
	mw.mission.state.Start.Roster[7] = mapload.PartyMember{ID: "roster", MercenaryType: 1, Class: 31}
	requireSpellClientClass(t, mw, 23)
	sim.Step(mw.world, []sim.Command{sim.Equip(7, 0, 1)})
	requireSpellClientClass(t, mw, 3)
	sim.Step(mw.world, []sim.Command{sim.Unequip(7, 1)})
	requireSpellClientClass(t, mw, 23)
	mw.mission.party[0].MercenaryType, mw.mission.party[0].Class = 2, 24
	requireSpellClientClass(t, mw, 24)
}

// The popup states the spell record's values at the record power, which
// follows the cast power up to 255. An actor whose skill plus Mind is below 30
// keeps the original byte fold and reads power 100 (TEXT-097).
func TestThePopupStatesTheRecordAtTheCastPower(t *testing.T) {
	rule := sim.SpellRule{ID: 1, ManaCost: 3, School: 1, MaxRange: 7, DamageMin: 4, DamageMax: 8,
		TargetsUnit: true, Damaging: true}
	w := testSpellPopupWords()
	for _, tc := range []struct {
		name       string
		skill, mnd int32
		damage     string
		rng        string
	}{
		{"below thirty", 5, 10, "Damage: 17-34", "Range: 10"},
		{"thirty", 0, 30, "Damage: 4-8", "Range: 7"},
		{"above one hundred", 100, 100, "Damage: 26-53", "Range: 12"},
		{"two hundred eighty-six", 186, 100, "Damage: 38-76", "Range: 15"},
	} {
		e := sim.Entity{Mind: tc.mnd}
		e.Skill[1] = tc.skill
		got := spellInfoLines(rule, sim.SpellCharacteristicsFor(sim.Rules{}, e, rule), "Fire Arrow", &w)
		want := []string{"Fire Arrow", "Mana cost: 3", tc.damage, tc.rng}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: lines = %q, want %q", tc.name, got, want)
		}
	}
}
