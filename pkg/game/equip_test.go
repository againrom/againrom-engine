package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/databin"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// eqScaleRow is a shape or material row carrying IDENTITY factors — 1 — at
// the three slots data.WeaponFromCode's own resolution reads (damage,
// to-hit, defence), so a weapon built through a real, parsed
// databin.Collection scales by exactly 1 and this file's expected numbers
// need no second arithmetic to derive. weaponRow's own reason
// (pkg/data/weapon_test.go), rebuilt here because that helper is unexported
// in another package.
func eqScaleRow(damage, toHit, defence float64) synth.DataBinRow {
	d := make([]float64, 9)
	d[4], d[5], d[6], d[7] = damage, toHit, defence, 1
	return synth.DataBinRow{Doubles: d}
}

func eqWeaponRowHands(name string, kind, min, max, toHit, defence, rng, charge, relax, hands int32) synth.DataBinRow {
	r := eqWeaponRow(name, kind, min, max, toHit, defence, rng, charge, relax)
	r.Params[14] = hands
	return r
}

func eqShieldRow(name string, defence, absorption int32) synth.DataBinRow {
	p := make([]int32, 16)
	for i := range p {
		p[i] = -1
	}
	p[9], p[10], p[data.SutableForColumn] = defence, absorption, eqSuitAny
	return synth.DataBinRow{Name: name, Params: p}
}

// eqWeaponRow is a Weapons row over the fourteen cells data.ResolveWeapon
// reads, laid out by RUNTIME column exactly as pkg/data/weapon.go's own slot
// constants name them (weaponAttackTypeSlot=5 .. weaponRelaxSlot=0xd).
// weaponRow's own layout (pkg/data/weapon_test.go), rebuilt here for the
// reason eqScaleRow states.
func eqWeaponRow(name string, kind, min, max, toHit, defence, rng, charge, relax int32) synth.DataBinRow {
	p := []int32{-1, -1, -1, -1, -1, kind, min, max, toHit, defence, -1, rng, charge, relax, -1, eqSuitAny}
	return synth.DataBinRow{Name: name, Params: p}
}

// eqSuitAny is the sutableFor cell (data.SutableForColumn) every row this
// package's fixtures write carries unless the test is about the wear rule
// itself: both bits set, usable by fighter and mage alike.
//
// IT IS STATED RATHER THAN LEFT OFF. A row shorter than the column reads as
// usable by NEITHER, which is the decoded answer for a row that has been read
// and says nothing there — so a fixture that omitted the cell would be
// modelling an item no character may wear, and every equip through it would be
// refused. Four shipped rows carry 3 (Ring, Amulet, BareHands, Plasma Sword),
// so the value is not invented for the fixtures.
const eqSuitAny = int32(3)

// eqSwordCode is the item code a Weapons entry named "Sword" resolves to and
// resolves back FROM (AC-2): field A the material index (0), field B the
// weapon class (1), field C the shape index (0), field D the row (1) — the
// shipped collection's own one-based numbering, entry 0 reserved and never
// written, "Sword" the first and only row this fixture writes.
const eqSwordCode = uint16(0)<<12 | uint16(1)<<8 | uint16(0)<<5 | uint16(1)

// eqMaceCode is eqSwordCode's sibling at row 2: the same material, class and
// shape, a different Weapons row, and — the whole point of it — a DIFFERENT
// ATTACK TYPE. "Sword" carries data.SkillBlade and "Mace" data.SkillBludgen,
// so a test that equips this code and reads the entity's credited slot
// discriminates a rearm that carries the skill from one that carries only the
// numbers a blow resolves against.
const eqMaceCode = uint16(0)<<12 | uint16(1)<<8 | uint16(0)<<5 | uint16(2)

const (
	eqBowCode    = uint16(0)<<12 | uint16(1)<<8 | uint16(0)<<5 | uint16(3)
	eqShieldCode = uint16(0)<<12 | uint16(2)<<8 | uint16(0)<<5 | uint16(1)
)

// eqNoRowCode names the weapon class (B=1, a real equipment slot) but a row
// data.ResolveWeapon's own search cannot match: D=0 is the reserved,
// never-written entry every one-based collection carries, whose name is the
// empty string — and findByName refuses an empty name outright (data/
// weapon.go).
const eqNoRowCode = uint16(1) << 8

const eqNoSlotCode = uint16(5)

// eqDefsTable is one *mapload.Table over a REAL parsed definition stream: a
// Shapes and a Materials collection of one identity row each, a Weapons
// collection of two written rows — "Sword" at eqSwordCode's own row and
// "Mace" at eqMaceCode's, differing in ATTACK TYPE (SkillBlade against
// SkillBludgen) and in cadence (a charge and relax of 1 apiece, so a blow
// lands every few ticks in the tests below that need many of them) — and,
// since 0136, an EMPTY Armors collection: a real, non-nil *databin.Collection
// of zero rows, present so EquipTarget's own four-field fence (rearm.go) does
// not refuse every weapon-only test in this file for a table this story
// widened out from under them, and empty because no test above this comment
// reads an armour code at all. wearAnArmourTable, below, is this same table
// with a written Armors row for the tests that do.
func eqDefsTable(t *testing.T) *mapload.Table {
	t.Helper()
	f, err := databin.Parse(synth.DataBin{
		Rows: [synth.DataBinCollections][]synth.DataBinRow{
			synth.DataBinShapes:    {eqScaleRow(1, 1, 1)},
			synth.DataBinMaterials: {eqScaleRow(1, 1, 1)},
			synth.DataBinWeapons: {
				eqWeaponRowHands("Sword", data.SkillBlade, 10, 20, 5, 3, 2, 7, 4, 0),
				eqWeaponRowHands("Mace", data.SkillBludgen, 10, 20, 500, 3, 2, 1, 1, 0),
				eqWeaponRowHands("Bow", 11, 6, 12, 8, 1, 6, 8, 5, 0),
			},
			synth.DataBinShields: {eqShieldRow("Ward", 7, 3)},
		},
	}.Bytes())
	if err != nil {
		t.Fatalf("databin.Parse: %v", err)
	}
	return &mapload.Table{
		Shapes: f.Collection(databin.Shapes), Materials: f.Collection(databin.Materials),
		Weapons: f.Collection(databin.Weapons), Shields: f.Collection(databin.Shields), Armors: f.Collection(databin.Armors),
	}
}

// eqSword is the *data.Weapon eqSwordCode resolves to against eqDefsTable —
// computed by calling data.WeaponFromCode itself, the same call
// production's enqueueEquip and rearm make, rather than hand-derived: this
// file is witnessing T4's WIRING (does the right resolved weapon reach
// Recompute and then SetCombat), not re-proving data.WeaponFromCode's own
// arithmetic, which pkg/data/weapon_test.go already does.
func eqSword(t *testing.T, table *mapload.Table) *data.Weapon {
	t.Helper()
	w, err := data.WeaponFromCode(data.ItemCode(eqSwordCode), table.Shapes, table.Materials, table.Weapons)
	if err != nil {
		t.Fatalf("setup: data.WeaponFromCode(eqSwordCode): %v", err)
	}
	return &w
}

// eqMace is eqSword's sibling at eqMaceCode, resolved the same way and for
// the same reason. The two exist together so a test can compare one NAME
// against another rather than against emptiness.
func eqMace(t *testing.T, table *mapload.Table) *data.Weapon {
	t.Helper()
	w, err := data.WeaponFromCode(data.ItemCode(eqMaceCode), table.Shapes, table.Materials, table.Weapons)
	if err != nil {
		t.Fatalf("setup: data.WeaponFromCode(eqMaceCode): %v", err)
	}
	return &w
}

// eqHero is one fixed hero every test below shares — the exact statistics
// are arbitrary, because every test compares the entity's own DamageBase
// against a value this same hero's own Hero.Recompute produces, never
// against a hand-computed number.
func eqHero() data.Hero { return data.Hero{Body: 30, Reaction: 20, Mind: 15, Spirit: 15} }

// equipMission opens a mission through openMission with entity as the sole
// party member and the sole start id — grabWorld's own shape (world_test.go)
// widened to state a hero, a fallback starting weapon and a definition
// table, the three T4 adds to invPartyGear (world.go).
func equipMission(t *testing.T, w *sim.World, entity sim.EntityID, hero data.Hero, startWeapon *data.Weapon, table *mapload.Table) *mapWorld {
	t.Helper()
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{{Hero: hero, Weapon: startWeapon}},
		Start: mapload.Start{IDs: []sim.EntityID{entity}}}
	return openMission(ms, table, nil, v, missionSource{}, nil, nil)
}

func TestEnqueueEquipAppendsACommandForAResolvableWeaponCode(t *testing.T) {
	table := eqDefsTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqNoSlotCode, eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, table)

	mw.enqueueEquip(1)

	if len(mw.pending) != 1 {
		t.Fatalf("pending = %v, want exactly one command", mw.pending)
	}
	got := mw.pending[0]
	want := sim.Command{Kind: sim.KindEquip, Entity: 7, X: 1, Y: 1}
	if got != want {
		t.Errorf("pending[0] = %+v, want %+v", got, want)
	}
}

func TestEnqueueEquipResolvesTheIndexAgainstElementsNotFlatCodes(t *testing.T) {
	table := eqDefsTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqNoSlotCode, eqNoSlotCode, eqNoSlotCode, eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, table)

	stacks, ok := w.CarriedStacks(7)
	if !ok || len(stacks) != 2 || stacks[0].Code != eqNoSlotCode || stacks[0].Count != 3 ||
		stacks[1].Code != eqSwordCode || stacks[1].Count != 1 {
		t.Fatalf("setup: CarriedStacks(7) = %+v, want [{%#x 3} {%#x 1}]", stacks, eqNoSlotCode, eqSwordCode)
	}

	mw.enqueueEquip(1)

	if len(mw.pending) != 1 {
		t.Fatalf("pending = %v, want exactly one command — element 1 is the sword", mw.pending)
	}
	want := sim.Command{Kind: sim.KindEquip, Entity: 7, X: 1, Y: 1}
	if got := mw.pending[0]; got != want {
		t.Errorf("pending[0] = %+v, want %+v", got, want)
	}
}

// AC-9's first case: an index the container does not have — an empty pack
// cell, from the drawing tier's own numbering — leaves the world untouched
// and appends nothing.
func TestEnqueueEquipDoesNothingForAnOutOfRangeIndex(t *testing.T) {
	table := eqDefsTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, table)
	before := w.Hash()

	mw.enqueueEquip(5) // the container holds one code, at index 0

	if len(mw.pending) != 0 {
		t.Errorf("pending = %v, want none — index 5 is out of range", mw.pending)
	}
	if got := w.Hash(); got != before {
		t.Errorf("World.Hash() changed from %d to %d for an out-of-range index", before, got)
	}
}

// AC-9's second case: a code at the named index that names no equipment
// slot at all (field B of 0) — the class carried and never worn.
func TestEnqueueEquipDoesNothingForACodeNamingNoEquipmentSlot(t *testing.T) {
	table := eqDefsTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqNoSlotCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, table)
	before := w.Hash()

	mw.enqueueEquip(0)

	if len(mw.pending) != 0 {
		t.Errorf("pending = %v, want none — the code names no slot", mw.pending)
	}
	if got := w.Hash(); got != before {
		t.Errorf("World.Hash() changed from %d to %d for a code naming no slot", before, got)
	}
}

func TestEnqueueEquipDoesNothingForAWeaponSlotCodeThatDoesNotResolve(t *testing.T) {
	table := eqDefsTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqNoRowCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, table)
	before := w.Hash()

	mw.enqueueEquip(0)

	if len(mw.pending) != 0 {
		t.Errorf("pending = %v, want none — the code names the weapon slot but resolves to nothing", mw.pending)
	}
	if got := w.Hash(); got != before {
		t.Errorf("World.Hash() changed from %d to %d for an unresolvable weapon-slot code", before, got)
	}
}

func TestEnqueueEquipDoesNothingWithNoDefinitionTable(t *testing.T) {
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, nil) // no table
	before := w.Hash()

	mw.enqueueEquip(0)

	if len(mw.pending) != 0 {
		t.Errorf("pending = %v, want none — there is no table to resolve against", mw.pending)
	}
	if got := w.Hash(); got != before {
		t.Errorf("World.Hash() changed from %d to %d with no definition table", before, got)
	}
}

// AC-8, the number itself: enqueueEquip composes the command, tick applies
// it and rearm's own recompute follows in the same tick — the entity's
// DamageBase ends the tick at the value eqHero's own Hero.Recompute produces
// for the newly equipped sword, not at the arbitrary number it started at.
func TestRearmMovesTheEntitysDamageBaseToTheNewLoadout(t *testing.T) {
	const before = 999 // an arbitrary starting value no recompute below could coincide with
	table := eqDefsTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10, DamageBase: before}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	hero := eqHero()
	mw := equipMission(t, w, 7, hero, nil, table)

	mw.enqueueEquip(0)
	mw.tick()

	sword := eqSword(t, table)
	want := hero.Recompute(data.Profile{}, data.Loadout{Weapon: sword}).Combat.DamageBase
	if want == before {
		t.Fatalf("setup: the recomputed DamageBase (%d) coincides with the starting value; "+
			"choose a different fixture so the assertion below can discriminate", want)
	}

	e, ok := mw.entity(7)
	if !ok {
		t.Fatal("entity 7 is gone after one tick")
	}
	if e.DamageBase != want {
		t.Errorf("DamageBase = %d, want %d (Hero.Recompute for the equipped sword) — got the pre-equip value %d instead",
			e.DamageBase, want, before)
	}

	// AC-3, restated at this layer: the equip moved the code, it did not
	// duplicate it. The container this subject started with held one code,
	// which the command's own X=0 names, so the container is empty and slot
	// 1 carries it.
	if codes, _ := w.Carried(7); len(codes) != 0 {
		t.Errorf("Carried(7) = %v, want empty — the only code moved into the equipment slot", codes)
	}
	if slots, _ := w.Equipped(7); slots[0] != eqSwordCode {
		t.Errorf("Equipped(7)[0] = %#x, want %#x", slots[0], eqSwordCode)
	}
}

// eqBladeHero is a hero trained in ONE slot: Blade at level 10, every other
// slot at 0. Hero.Reward turns that into the six slot experiences a party
// mint seeds an entity with (mapload/start.go), so this fixture's own
// SkillXP[SkillBlade] is the number the character sheet shows for a default
// hero rather than a literal chosen here — asserted as such below. At 30
// the scale is exactly 150/120, which pays a raw of 1 exactly 1 — the
// per-blow step the crossing test below counts on.
func eqBladeHero() data.Hero {
	h := data.Hero{Body: 30, Reaction: 20, Mind: 30, Spirit: 15}
	h.Skill[data.SkillBlade] = 10
	return h
}

// THE OWNER'S OWN REPORT, as close as a test can stand to it: take a mace,
// equip it, land blows, and watch which skill grows. He reported «опыт капает,
// но навык не растет» — experience accrues, the skill does not.
//
// It drives the equip through enqueueEquip, the command path a double-click
// raises (equipFromPack's one added statement is the pkg/ui hand-off, and it
// reaches this same function), then issues one attack order and ticks. It
// asserts VALUES, not movement:
//
// TO CONFIRM IT WITNESSES THE FIX, delete `e.XPSlot = c.XPSlot` from
// SetCombat (pkg/sim/rearm.go) — that is the tree exactly as it was before
// this hotfix. The assertion after the equip reddens with "the credited slot
// is 1, want 3": the entity fights with the mace and goes on paying every
// blow into the sword skill, which is the owner's report in one line.
func TestEquippingAMaceMovesTheCreditedSkillFromBladeToBludgen(t *testing.T) {
	const hero, victim = sim.EntityID(7), sim.EntityID(8)
	table := eqDefsTable(t)
	h := eqBladeHero()
	reward := h.Reward()
	if reward.SkillXP[data.SkillBlade] != 1593 {
		t.Fatalf("setup: a level-10 Blade seeds %d experience, want 1593 (S(10)) — "+
			"the curve moved and this test's numbers below are stale",
			reward.SkillXP[data.SkillBlade])
	}

	a := sim.Entity{ID: hero, X: 3, Y: 3, HP: 100, MaxHP: 100, Reach: 1,
		Owner: 2, GainsXP: true, TypeID: sim.HumanTypeID, Mind: reward.Mind, SkillXP: reward.SkillXP,
		XPSlot: uint8(data.SkillBlade)}
	a.Skill[data.SkillBlade] = h.Skill[data.SkillBlade]
	// A victim with health enough to survive every blow below — a dead
	// target pays nothing (payExperience's own aliveBefore refusal) — and a
	// small experience value, so the gain per blow is a few points and the
	// crossing of 100 can be caught rather than jumped over.
	v := sim.Entity{ID: victim, X: 4, Y: 3, HP: 1_000_000, MaxHP: 1_000_000,
		DyingTime: 200, Owner: 3, XPValue: 4}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, sim.Terrain{}, []sim.Entity{a, v}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: hero, Items: []uint16{eqMaceCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, hero, h, eqSword(t, table), table)

	at := func() sim.Entity {
		t.Helper()
		e, ok := mw.entity(hero)
		if !ok {
			t.Fatal("the world holds no hero")
		}
		return e
	}
	// level is the panel's own reading for a slot — entityDraws' overlay, the
	// surface the owner was actually looking at, not a second computation.
	level := func(slot int32) int {
		t.Helper()
		for _, d := range mw.entityDraws() {
			if d.ID == uint32(hero) {
				return d.Char.Skills[slot]
			}
		}
		t.Fatal("the hero crossed no entry")
		return 0
	}

	if got := at().XPSlot; got != uint8(data.SkillBlade) {
		t.Fatalf("before the equip the credited slot is %d, want %d (Blade) — "+
			"the mission opened crediting something else", got, data.SkillBlade)
	}
	if got := level(data.SkillBlade); got != 10 {
		t.Fatalf("before the equip the panel reads Blade at level %d, want 10", got)
	}

	mw.enqueueEquip(0)
	mw.tick()

	if got := at().XPSlot; got != uint8(data.SkillBludgen) {
		t.Fatalf("after equipping the mace the credited slot is %d, want %d (Bludgen) — "+
			"the entity fights with the mace and still trains the sword", got, data.SkillBludgen)
	}

	// BEFORE ANY BLOW HAS LANDED WITH THE MACE, the panel reads Bludgen at
	// its seeded level: 0. This is the state the first blow below moves it
	// away from.
	if got := level(data.SkillBludgen); got != 0 {
		t.Fatalf("before any blow lands the panel reads Bludgen at level %d, want 0", got)
	}

	// THE ORDER IS RE-ISSUED EVERY TICK, which is what a player holding an
	// enemy under attack does: a cycle that finishes its relax and finds
	// itself ready drops the target on the next advance, so one order lands
	// exactly one blow. Nothing here depends on that behaviour — it only
	// decides how the blows are fed.
	const crossing = 100 // S(1)
	firstBlowSeen, steadyLevel, crossed := false, -1, false
	for i := 0; i < 4000 && !crossed; i++ {
		xpBefore := at().SkillXP[data.SkillBludgen]
		mw.pending = append(mw.pending, sim.Command{Kind: sim.KindAttack, Entity: hero, X: int32(victim)})
		mw.tick()
		xpAfter := at().SkillXP[data.SkillBludgen]
		if !firstBlowSeen && xpBefore == 0 && xpAfter > xpBefore {
			firstBlowSeen = true
			if got := level(data.SkillBludgen); got != 1 {
				t.Fatalf("the tick of the first landed blow leaves the panel reading Bludgen at level %d, "+
					"want 1 — FR-3's first-award rise", got)
			}
		}
		if xpAfter > 0 && xpAfter < crossing {
			steadyLevel = level(data.SkillBludgen)
		}
		if xpAfter >= crossing {
			crossed = true
		}
	}
	if !crossed {
		t.Fatalf("Bludgen holds %d after 4000 ticks, want it to reach %d — "+
			"the blows are landing somewhere else", at().SkillXP[data.SkillBludgen], crossing)
	}
	if steadyLevel != 1 {
		t.Errorf("while Bludgen's experience stood below %d the panel read level %d, want 1", crossing, steadyLevel)
	}
	if got, xp := level(data.SkillBludgen), at().SkillXP[data.SkillBludgen]; got != 1 || xp != crossing {
		t.Errorf("with %d experience in Bludgen the panel reads level %d, want level 1 at exactly %d "+
			"experience — the entity's own stored level, raised once by the first blow and not again "+
			"until the experience stands strictly above S(1)", xp, got, crossing)
	}

	// AND THE SLOT HE WAS WATCHING NEVER MOVED. Blade is untouched by every
	// one of those blows: exactly S(10), exactly level 10, 260 short of the
	// S(11) = 1853 that would have shown him anything.
	if got := at().SkillXP[data.SkillBlade]; got != 1593 {
		t.Errorf("Blade holds %d experience, want exactly 1593 — no blow after the equip may be credited there", got)
	}
	if got := level(data.SkillBlade); got != 10 {
		t.Errorf("the panel reads Blade at level %d, want 10", got)
	}
}

// The other side of the same rule, and the case that would otherwise pass by
// coincidence: equipping a weapon of the skill the entity ALREADY credits
// leaves the credited slot exactly where it was, so the fix is "the slot
// follows the weapon" and not "the slot is rewritten to something".
func TestEquippingASwordLeavesABladeFighterCreditingBlade(t *testing.T) {
	table := eqDefsTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10, XPSlot: uint8(data.SkillBlade)}},
		nil, sim.Relations{}, nil, []sim.Stock{{ID: 7, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqBladeHero(), nil, table)

	mw.enqueueEquip(0)
	mw.tick()

	e, ok := mw.entity(7)
	if !ok {
		t.Fatal("entity 7 is gone after one tick")
	}
	if e.XPSlot != uint8(data.SkillBlade) {
		t.Errorf("credited slot = %d, want %d (Blade) — a sword trains the blade skill",
			e.XPSlot, data.SkillBlade)
	}
}

// creditedSlot's own window, at its edges and outside them. THE -1 CASE IS
// NOT HYPOTHETICAL: the shipped Weapons table carries a row of attack type
// -1 on both lawful roots, and data.WeaponFromCode resolves it as a melee
// weapon, so activeSkill can hand this function a negative number. Narrowed
// to SkillGeneral it is a legal slot; converted straight to uint8 it would be
// 255, which SetCombat refuses and payExperience would have indexed with.
func TestCreditedSlotNarrowsToTheSixSlotsAnEntityHolds(t *testing.T) {
	for _, c := range []struct{ in, want int32 }{
		{-1, data.SkillGeneral}, {-1000, data.SkillGeneral},
		{data.SkillGeneral, data.SkillGeneral},
		{data.SkillBlade, data.SkillBlade},
		{data.SkillBludgen, data.SkillBludgen},
		{data.SkillSlots - 1, data.SkillSlots - 1},
		{data.SkillSlots, data.SkillGeneral},
		{9, data.SkillGeneral}, {256, data.SkillGeneral},
	} {
		if got := creditedSlot(c.in); got != c.want {
			t.Errorf("creditedSlot(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestRearmFallsBackToTheStartingWeaponWhenSlotOneIsEmpty(t *testing.T) {
	table := eqDefsTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqNoSlotCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	hero := eqHero()
	start := eqSword(t, table) // the party member's own starting weapon
	mw := equipMission(t, w, 7, hero, start, table)

	mw.pending = append(mw.pending, sim.Command{Kind: sim.KindEquip, Entity: 7, X: 0, Y: 2})
	mw.tick()

	want := hero.Recompute(data.Profile{}, data.Loadout{Weapon: start}).Combat.DamageBase
	e, ok := mw.entity(7)
	if !ok {
		t.Fatal("entity 7 is gone after one tick")
	}
	if e.DamageBase != want {
		t.Errorf("DamageBase = %d, want %d — the recompute for the starting weapon, slot 1 being empty",
			e.DamageBase, want)
	}
	if slots, _ := w.Equipped(7); slots[0] != 0 {
		t.Errorf("Equipped(7)[0] = %#x, want 0 — this test never equipped slot 1", slots[0])
	}
}

// THE OWNER'S OWN REPORT: equip a different weapon out of the pack and the
// info window's equipped-weapon row goes on naming the old one. The numbers
// move — that is TestRearmMovesTheEntitysDamageBaseToTheNewLoadout above — and
// the name does not.
//
// It reads the surface he was looking at: mw.entityDraws(), the readout the
// panel is built from, the same one the credited-slot test reads a skill level
// through. And it compares two DIFFERENT resolved names, before and after, so
// an implementation that merely wrote SOMETHING non-empty into that row would
// still redden — the assertion is a value, not "it is not blank".
//
// TO CONFIRM IT WITNESSES THE FIX, delete the mw.chars write at the end of
// rearm (pkg/game/world.go) — that is the tree exactly as it was before this
// hotfix — and rerun: the second assertion reddens with the sword's name,
// which is the defect, while every other test in this file stays green.
func TestRearmMovesThePanelsWeaponNameToTheNewLoadout(t *testing.T) {
	table := eqDefsTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqMaceCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	sword, mace := eqSword(t, table), eqMace(t, table)
	if sword.Name == mace.Name || sword.Name == "" {
		t.Fatalf("setup: the two fixture weapons must carry different non-empty names, got %q and %q — "+
			"the assertions below could not discriminate", sword.Name, mace.Name)
	}
	mw := equipMission(t, w, 7, eqHero(), sword, table)

	// The panel's own reading of the equipped-weapon row, through the readout
	// rather than through mw.chars: this is what pkg/ui is handed.
	named := func() string {
		t.Helper()
		for _, d := range mw.entityDraws() {
			if d.ID == 7 {
				return d.Char.Weapon
			}
		}
		t.Fatal("the hero crossed no entry")
		return ""
	}

	if got := named(); got != sword.Name {
		t.Fatalf("before the equip the panel names %q, want the starting weapon %q", got, sword.Name)
	}

	mw.enqueueEquip(0)
	mw.tick()

	if slots, _ := w.Equipped(7); slots[0] != eqMaceCode {
		t.Fatalf("setup: Equipped(7)[0] = %#x, want %#x — the equip did not land and the row below "+
			"would be asserting nothing", slots[0], eqMaceCode)
	}
	if got := named(); got != mace.Name {
		t.Errorf("the panel names %q after equipping the mace, want %q — the equipped weapon's numbers "+
			"moved and its name did not", got, mace.Name)
	}
}

// A BARE HERO READS AS BARE, not as whatever he last held. "No weapon" is a
// real state (buildInventorySubject, inventory.go) and rearm reaches it
// whenever slot 1 is empty and the party member was generated holding
// nothing — the same fallback arm
// TestRearmFallsBackToTheStartingWeaponWhenSlotOneIsEmpty drives, with a nil
// starting weapon instead of a sword, so w is nil at the write.
//
// THE STALE NAME IS SEEDED RATHER THAN REACHED, on purpose: this test's own
// question is the fallback arm (slot 2 changes, slot 1 stays empty), which
// TestRearmFallsBackToTheStartingWeaponWhenSlotOneIsEmpty already drives —
// a real unequip on slot 1 (0151, defect 4: KindUnequip, exercised in
// TestUnequippingASwordMovesTheEntitysDamageBaseBackToBare) would reach the
// same bare state by a different route and is not this fixture's own
// concern. The seed is the state the panel would be holding either way, and
// asserting against it is what makes this test discriminate between
// clearing the row and leaving it. Without the nil guard on w it does not
// merely redden, it panics.
func TestRearmNamesNoWeaponForABareHero(t *testing.T) {
	const stale = "Stale Blade" // a name no fixture in this file resolves to
	table := eqDefsTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqNoSlotCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, table) // no starting weapon: a bare hero

	char, ok := mw.chars[7]
	if !ok {
		t.Fatal("setup: the load knows no character for the sole party member")
	}
	if char.Weapon != "" {
		t.Fatalf("setup: a bare hero opens the mission named %q, want no weapon at all", char.Weapon)
	}
	char.Weapon = stale
	mw.chars[7] = char

	// Slot 2, so the equipment array changes and rearm's guard fires while
	// slot 1 stays at the zero code — the fallback test's own command.
	mw.pending = append(mw.pending, sim.Command{Kind: sim.KindEquip, Entity: 7, X: 0, Y: 2})
	mw.tick()

	if slots, _ := w.Equipped(7); slots[0] != 0 {
		t.Fatalf("setup: Equipped(7)[0] = %#x, want 0 — this test never equips slot 1", slots[0])
	}
	for _, d := range mw.entityDraws() {
		if d.ID != 7 {
			continue
		}
		if d.Char.Weapon != "" {
			t.Errorf("the panel names %q for a hero holding nothing, want no weapon — "+
				"a bare hero must not go on reading as the weapon he last held", d.Char.Weapon)
		}
		return
	}
	t.Fatal("the hero crossed no entry")
}

// ---------------------------------------------------------------------------
// D-14 (0139): a re-armed weapon's spell survives an equip exactly when the
// slot's own code is the one the start weapon composed, and is lost — the
// disclosed limit spec D-5 names — when it names something else.
// ---------------------------------------------------------------------------

// rearmStaffCode and rearmWandCode are two resolvable Weapons rows' own
// composed item codes, on eqSwordCode's own formula above (material 0,
// field B the weapon class 1, shape 0, and the row): row 1 for the staff
// this file's own start weapon is built from, row 2 for a second,
// unrelated weapon a rearm may equip instead.
const (
	rearmStaffCode = uint16(0)<<12 | uint16(1)<<8 | uint16(0)<<5 | uint16(1)
	rearmWandCode  = uint16(0)<<12 | uint16(1)<<8 | uint16(0)<<5 | uint16(2)
)

// rearmTable is D-14's own fixture: two resolvable weapons, "Staff" and
// "Wand", NEITHER named with a castSpell attachment — data.WeaponFromCode
// recomposes a bare NAME from a code's three collection indices and a code
// has no field an attachment could live in (spec D-5), so this table's own
// rows are exactly what a RE-RESOLUTION reaches. The spell rearmStaff
// (below) carries comes from being handed in as the party's own STARTING
// OBJECT, never from this table. Spells carries one row, "Fire Arrow", so
// mapload.SpellIDByToken has something to resolve the token against.
//
// Armors IS PRESENT AND EMPTY, and it has to be: EquipTarget (rearm.go)
// refuses outright — (0, false) — when a table is missing ANY of Shapes,
// Materials, Weapons or Armors, so a table without the fourth would make
// enqueueEquip append no command at all and both tests below would measure a
// rearm that never ran rather than the guard they are about. An EMPTY
// collection is the honest stand-in here, on emptyScale's own precedent
// above: neither of these codes names an armour (field B is the weapon
// class, which data.ArmorFromCode refuses on its first line), so no row of
// it is ever read.
func rearmTable() *mapload.Table {
	return &mapload.Table{
		Shapes: emptyScale{}, Materials: emptyScale{},
		Weapons: dbCollection{
			{},
			{name: "Staff", params: chargenWeaponParams(data.SkillBlade)},
			{name: "Wand", params: chargenWeaponParams(data.SkillBlade)},
		},
		Armors: dbCollection{},
		Spells: dbCollection{{}, {name: "Fire Arrow"}},
	}
}

// rearmStaff is the party's own starting weapon (0139 plan D-14): the SAME
// object data.ResolveWeapon would have built from `Staff
// {castSpell=Fire_Arrow:10}` against rearmTable, written out by hand on
// heroSword's own precedent (pkg/mapload/hero_test.go) rather than
// resolved, because this file's question is what REARM does with a
// member's already-resolved weapon, not ResolveWeapon's own arithmetic.
// Code matches rearmStaffCode exactly, which is the one fact this pair of
// tests turns on.
func rearmStaff() *data.Weapon {
	return &data.Weapon{Name: "Staff", Code: data.ItemCode(rearmStaffCode),
		AttackType: data.SkillBlade, ChargeTime: 6, RelaxTime: 4, Range: 1,
		SpellName: "Fire_Arrow", SpellPower: 10}
}

func rearmHero() data.Hero { return data.Hero{Body: 20, Reaction: 20, Mind: 20, Spirit: 20} }

func TestRearmReadsTheCurrentItemsSpellInsteadOfSameCodeFallback(t *testing.T) {
	cast := func(id uint16, power int16) sim.ItemInstance {
		item := sim.PlainItem(rearmStaffCode)
		item.Effects = []sim.ItemEffect{{Kind: 41, Operand: uint32(id) | uint32(uint16(power))<<16}}
		return item
	}
	explicitEmpty := sim.PlainItem(rearmStaffCode)
	explicitEmpty.WeightPresent = true
	for _, test := range []struct {
		name  string
		item  sim.ItemInstance
		spell uint16
		power int32
	}{
		{"code-only authored fallback", sim.PlainItem(rearmStaffCode), 1, 10},
		{"explicit absent attachment", explicitEmpty, 0, 0},
		{"explicit zero power", cast(1, 0), 1, 0},
		{"authored power as current effect", cast(1, 10), 1, 10},
		{"different current power", cast(1, 27), 1, 27},
		{"signed current power", cast(1, -2), 1, -2},
		{"explicit zero spell ID", cast(0, 7), 0, 7},
	} {
		t.Run(test.name, func(t *testing.T) {
			const id = sim.EntityID(7)
			table := rearmTable()
			w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
				[]sim.Entity{{ID: id, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
				[]sim.Stock{{ID: id, ItemInstances: []sim.ItemInstance{test.item}}})
			if err != nil {
				t.Fatalf("NewStockedWorld: %v", err)
			}
			mw := equipMission(t, w, id, rearmHero(), rearmStaff(), table)
			mw.enqueueEquip(0)
			mw.tick()
			e, ok := mw.entity(id)
			if !ok || e.WeaponSpell != test.spell || e.WeaponSpellLevel != test.power {
				t.Fatalf("current item spell = (%d,%d), want (%d,%d); actor present=%t", e.WeaponSpell, e.WeaponSpellLevel, test.spell, test.power, ok)
			}
			equipped, ok := w.EquippedItems(id)
			carried, held := w.CarriedStacks(id)
			if !ok || !held || len(carried) != 0 || equipped[0].Code != test.item.Code || !reflect.DeepEqual(equipped[0].Effects, test.item.Effects) {
				t.Fatal("equip lost or replaced current Item effects", equipped[0], carried)
			}
			mw.tick()
			e, ok = mw.entity(id)
			if !ok || e.WeaponSpell != test.spell || e.WeaponSpellLevel != test.power {
				t.Fatal("next tick replaced the current attachment", e.WeaponSpell, e.WeaponSpellLevel)
			}
		})
	}
}

// TestRearmLosesTheSpellWhenTheSlotNamesADifferentCode is spec D-5's own
// disclosed limit, witnessed from the other side of the same guard: a code
// the party's start weapon did NOT compose is re-resolved through
// data.WeaponFromCode, which recomposes a bare name — no attachment
// survives a code round trip — so the entity ends the tick carrying no
// spell at all, exactly as an equip off the ground or a corpse would leave
// it (spec D-5).
func TestRearmLosesTheSpellWhenTheSlotNamesADifferentCode(t *testing.T) {
	const id = sim.EntityID(7)
	table := rearmTable()
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: id, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: id, Items: []uint16{rearmWandCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, id, rearmHero(), rearmStaff(), table)

	mw.enqueueEquip(0)
	mw.tick()

	e, ok := mw.entity(id)
	if !ok {
		t.Fatal("entity is gone after one tick")
	}
	if e.WeaponSpell != 0 || e.WeaponSpellLevel != 0 {
		t.Errorf("weapon spell = (%d, %d) after equipping an unrelated code, want (0, 0) — "+
			"spec D-5's own disclosed limit", e.WeaponSpell, e.WeaponSpellLevel)
	}
}

func TestRearmRecomputesToHitWhenTheCreditedSkillRisesWithNoEquipThatTick(t *testing.T) {
	const hero, victim = sim.EntityID(7), sim.EntityID(8)
	table := eqDefsTable(t)
	h := eqBladeHero()
	reward := h.Reward()

	// THE BLOW'S OWN NUMBERS ARE ON THE FIXTURE, not folded in by rearm: this
	// test advances the world directly (below) and so never reaches the
	// per-frame recompute that would otherwise supply them.
	a := sim.Entity{ID: hero, X: 3, Y: 3, HP: 100, MaxHP: 100, Reach: 1,
		Owner: 2, GainsXP: true, TypeID: sim.HumanTypeID, Mind: reward.Mind, SkillXP: reward.SkillXP,
		XPSlot: uint8(data.SkillBlade), DamageBase: 5, AlwaysHits: true,
		AttackCharge: 1, AttackRelax: 1, Facing: 64, DesiredFacing: 64}
	a.Skill[data.SkillBlade] = h.Skill[data.SkillBlade]
	v := sim.Entity{ID: victim, X: 4, Y: 3, HP: 1_000_000, MaxHP: 1_000_000,
		DyingTime: 200, Owner: 3, XPValue: 4}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, sim.Terrain{}, []sim.Entity{a, v}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: hero, Items: []uint16{eqMaceCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, hero, h, eqSword(t, table), table)

	at := func() sim.Entity {
		t.Helper()
		e, ok := mw.entity(hero)
		if !ok {
			t.Fatal("the world holds no hero")
		}
		return e
	}

	mw.enqueueEquip(0)
	mw.tick() // the EQUIP tick: rearm's first trigger, Bludgen still level 0

	mace := eqMace(t, table)
	wantZero := h.Recompute(data.Profile{}, data.Loadout{Weapon: mace}).Combat.ToHit
	if got := at().ToHit; got != wantZero {
		t.Fatalf("setup: ToHit after the equip is %d, want %d (Hero.Recompute, Bludgen at level 0)", got, wantZero)
	}

	mw.pending = append(mw.pending, sim.Command{Kind: sim.KindAttack, Entity: hero, X: int32(victim)})
	mw.tick()

	if got := at().Skill[data.SkillBludgen]; got != 1 {
		t.Fatalf("setup: Bludgen reads level %d after the one blow, want 1 — FR-3's first-award rise", got)
	}

	levelOne := h
	levelOne.Skill[data.SkillBludgen] = 1
	wantOne := levelOne.Recompute(data.Profile{}, data.Loadout{Weapon: mace}).Combat.ToHit
	if wantOne == wantZero {
		t.Fatalf("setup: level 0 and level 1 both recompute ToHit to %d; "+
			"choose a different fixture so the assertion below can discriminate", wantZero)
	}
	if got := at().ToHit; got != wantOne {
		t.Errorf("ToHit = %d, want %d (Hero.Recompute for Bludgen at level 1) — "+
			"the tick that raised the level carried no equip command, so if this reads %d "+
			"the raise never reached rearm's second trigger", got, wantOne, wantZero)
	}
}

func TestLiveShieldEquipWithOneHandedWeaponFoldsBaseAndMagic(t *testing.T) {
	table := eqDefsTable(t)
	shield := sim.ItemInstance{Code: eqShieldCode, Kind: 1, Price: 701,
		Effects: []sim.ItemEffect{{Kind: 15, Operand: 4}}}
	sword := sim.ItemInstance{Code: eqSwordCode, Kind: 2, Price: 902,
		Effects: []sim.ItemEffect{{Kind: 17, Operand: 3}}}
	var worn [sim.EquipSlots]sim.ItemInstance
	worn[0] = sword
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 20, MaxHP: 20}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, ItemInstances: []sim.ItemInstance{shield}, EquippedItems: worn}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	hero := eqHero()
	mw := equipMission(t, w, 7, hero, nil, table)
	mw.enqueueEquip(0)
	if len(mw.pending) != 1 || mw.pending[0].Y != 2 || mw.pending[0].Spell != 0 {
		t.Fatalf("shield command = %+v, want slot 2 beside the one-handed weapon", mw.pending)
	}
	mw.tick()

	gotWorn, _ := w.EquippedItems(7)
	gotPack, _ := w.CarriedItems(7)
	if !sim.ItemEqual(gotWorn[0], sword) || !sim.ItemEqual(gotWorn[1], shield) || gotWorn[1].Price != shield.Price {
		t.Fatalf("worn = %+v, want complete sword and shield", gotWorn)
	}
	if len(gotPack) != 0 {
		t.Fatalf("pack = %+v, want empty after wearing the shield", gotPack)
	}
	piece, err := data.ShieldFromCode(data.ItemCode(eqShieldCode), table.Shapes, table.Materials, table.Shields)
	if err != nil {
		t.Fatalf("ShieldFromCode: %v", err)
	}
	want := hero.Recompute(data.Profile{}, data.Loadout{Weapon: eqSword(t, table), Mod: data.EquipMod{
		Defence: piece.Defence + 4, Absorption: piece.Absorption,
	}}).Combat
	entity, _ := mw.entity(7)
	if entity.Defence != want.Defence || entity.Absorption != want.Absorption {
		t.Fatalf("live Defence/Absorption = %d/%d, want base shield plus magic %d/%d",
			entity.Defence, entity.Absorption, want.Defence, want.Absorption)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back sim.World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if back.Hash() != w.Hash() {
		t.Fatalf("round-trip hash = %#x, want %#x", back.Hash(), w.Hash())
	}
	backWorn, _ := back.EquippedItems(7)
	backPack, _ := back.CarriedItems(7)
	if !sim.ItemEqual(backWorn[0], sword) || !sim.ItemEqual(backWorn[1], shield) || len(backPack) != 0 {
		t.Fatalf("round trip lost sword/shield instances: worn=%+v pack=%+v", backWorn, backPack)
	}
}

func TestLiveShieldEquipMaterializesCompatibleStartingWeaponFallback(t *testing.T) {
	table := eqDefsTable(t)
	shield := sim.ItemInstance{Code: eqShieldCode, Kind: 1, Price: 701,
		Effects: []sim.ItemEffect{{Kind: 15, Operand: 4}}}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 20, MaxHP: 20}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, ItemInstances: []sim.ItemInstance{shield}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	weapon := eqSword(t, table)
	mw := equipMission(t, w, 7, eqHero(), weapon, table)
	mw.enqueueEquip(0)
	if len(mw.pending) != 2 || mw.pending[0].Y != 1 || mw.pending[1].Y != 2 {
		t.Fatalf("fallback shield commands = %+v, want weapon then shield in one tick", mw.pending)
	}
	mw.tick()

	worn, _ := w.EquippedItems(7)
	pack, _ := w.CarriedItems(7)
	if worn[0].Code != uint16(weapon.Code) || !sim.ItemEqual(worn[1], shield) || worn[1].Price != shield.Price || len(pack) != 0 {
		t.Fatalf("fallback shield result worn=%+v pack=%+v, want materialized weapon and complete shield", worn, pack)
	}
}

func TestSameCodeShieldSwapRefreshesCombatPopupAndSave(t *testing.T) {
	for _, tc := range []struct {
		name       string
		oldEffects []sim.ItemEffect
		oldBonus   int32
	}{
		{name: "plain to enchanted"},
		{name: "enchanted to enchanted", oldEffects: []sim.ItemEffect{{Kind: 15, Operand: 4}}, oldBonus: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			table := eqDefsTable(t)
			table.Names = data.ItemNames{data.ItemCode(eqShieldCode): "Ward"}
			old := sim.ItemInstance{Code: eqShieldCode, Kind: 1, Price: 111, Effects: tc.oldEffects}
			incoming := sim.ItemInstance{Code: eqShieldCode, Kind: 1, Price: 222,
				Effects: []sim.ItemEffect{{Kind: 15, Operand: 9}}}
			var worn [sim.EquipSlots]sim.ItemInstance
			worn[0] = sim.ItemInstance{Code: eqSwordCode, Kind: 2}
			worn[1] = old
			w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
				[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 40, MaxHP: 40}}, nil, sim.Relations{}, nil,
				[]sim.Stock{{ID: 7, ItemInstances: []sim.ItemInstance{incoming}, EquippedItems: worn}})
			if err != nil {
				t.Fatalf("NewStockedWorld: %v", err)
			}
			hero := eqHero()
			if _, ok := Rearm(w, 7, hero, data.Profile{}, nil, true, table, 0); !ok {
				t.Fatal("initial Rearm refused the valid sword and shield")
			}
			mw := equipMission(t, w, 7, hero, nil, table)
			before, _ := mw.entity(7)

			mw.enqueueEquip(0)
			mw.tick()
			mw.refreshEquipment()

			after, _ := mw.entity(7)
			if got, want := after.Defence-before.Defence, int32(9)-tc.oldBonus; got != want {
				t.Fatalf("same-code shield swap moved Defence by %d, want %d", got, want)
			}
			wornAfter, _ := w.EquippedItems(7)
			packAfter, _ := w.CarriedItems(7)
			if !sim.ItemEqual(wornAfter[1], incoming) || wornAfter[1].Price != incoming.Price ||
				len(packAfter) != 1 || !sim.ItemEqual(packAfter[0], old) || packAfter[0].Price != old.Price {
				t.Fatalf("same-code swap lost an instance: worn=%+v pack=%+v", wornAfter, packAfter)
			}
			lines := mw.invSubject.SlotInfo[1]
			if !slices.Contains(lines, "#Defence +9") || slices.Contains(lines, "#Defence +4") ||
				slices.Contains(lines, "#Value 222") || slices.Contains(lines, "#Value 111") {
				t.Fatalf("worn shield popup = %v, want the incoming instance only and no price line", lines)
			}

			form, err := w.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary: %v", err)
			}
			var back sim.World
			if err := back.UnmarshalBinary(form); err != nil {
				t.Fatalf("UnmarshalBinary: %v", err)
			}
			if back.Hash() != w.Hash() {
				t.Fatalf("round-trip hash = %#x, want %#x", back.Hash(), w.Hash())
			}
			backWorn, _ := back.EquippedItems(7)
			backPack, _ := back.CarriedItems(7)
			if backWorn[1].Code != incoming.Code || backWorn[1].Kind != incoming.Kind ||
				backWorn[1].Price != incoming.Price || !reflect.DeepEqual(backWorn[1].Effects, incoming.Effects) ||
				len(backPack) != 1 || backPack[0].Code != old.Code || backPack[0].Kind != old.Kind ||
				backPack[0].Price != old.Price || !reflect.DeepEqual(backPack[0].Effects, old.Effects) {
				t.Fatalf("round trip lost same-code instances: worn=%+v pack=%+v", backWorn, backPack)
			}
		})
	}
}

func TestLiveRangedWeaponEquipDisplacesShieldButOneHandedWeaponCoexists(t *testing.T) {
	table := eqDefsTable(t)
	shield := sim.ItemInstance{Code: eqShieldCode, Kind: 1, Price: 701,
		Effects: []sim.ItemEffect{{Kind: 15, Operand: 4}}}
	for _, tc := range []struct {
		name       string
		weaponCode uint16
		wantMove   uint16
	}{
		{"ranged weapon", eqBowCode, 2},
		{"one-handed weapon", eqSwordCode, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			weapon := sim.ItemInstance{Code: tc.weaponCode, Kind: 2, Price: 902,
				Effects: []sim.ItemEffect{{Kind: 12, Operand: 3}}}
			oldSword := sim.ItemInstance{Code: eqSwordCode, Kind: 2, Price: 411}
			var worn [sim.EquipSlots]sim.ItemInstance
			worn[0] = oldSword
			worn[1] = shield
			w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
				[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 20, MaxHP: 20}}, nil, sim.Relations{}, nil,
				[]sim.Stock{{ID: 7, ItemInstances: []sim.ItemInstance{weapon}, EquippedItems: worn}})
			if err != nil {
				t.Fatalf("NewStockedWorld: %v", err)
			}
			mw := equipMission(t, w, 7, eqHero(), nil, table)
			mw.enqueueEquip(0)
			if len(mw.pending) != 1 || mw.pending[0].Spell != tc.wantMove {
				t.Fatalf("weapon command = %+v, want displacement slot %d", mw.pending, tc.wantMove)
			}
			mw.tick()
			gotWorn, _ := w.EquippedItems(7)
			gotPack, _ := w.CarriedItems(7)
			if !sim.ItemEqual(gotWorn[0], weapon) || gotWorn[0].Price != weapon.Price {
				t.Fatalf("slot 1 = %+v, want complete weapon %+v", gotWorn[0], weapon)
			}
			if tc.wantMove != 0 {
				if !gotWorn[1].Empty() || len(gotPack) != 2 ||
					!sim.ItemEqual(gotPack[0], oldSword) || !sim.ItemEqual(gotPack[1], shield) || gotPack[1].Price != shield.Price {
					t.Fatalf("ranged result worn=%+v pack=%+v, want displaced complete shield", gotWorn, gotPack)
				}
			} else if !sim.ItemEqual(gotWorn[1], shield) || len(gotPack) != 1 || !sim.ItemEqual(gotPack[0], oldSword) {
				t.Fatalf("one-handed result worn=%+v pack=%+v, want weapon and shield together", gotWorn, gotPack)
			}
		})
	}
}

// AC-9, at the layer pickup_test.go's own header says this file must test
// it at: skillRiseRows, skillRises' pure row-building half and the sibling of
// pickupLinesForItems, is what a test can assert precisely against.
//
// A WARRIOR'S ROW IS THE WEAPON SKILL'S RAISE LINE and carries the slot's NEW
// level — not the delta, not the old one.
func TestSkillRiseRowsNamesTheWarriorSkillAndItsNewLevel(t *testing.T) {
	var last, current [data.SkillSlots]int32
	last[data.SkillBludgen] = 10
	current[data.SkillBludgen] = 11

	words := ui.AuthoredWords()
	got := skillRiseRows(&words, false, last, current)
	if len(got) != 1 {
		t.Fatalf("skillRiseRows = %+v, want exactly one row", got)
	}
	if want := "Bludgeon skill improved: 11"; got[0] != want {
		t.Errorf("row 0 = %q, want %q", got[0], want)
	}
}

func TestSkillRiseRowsNamesTheMageSkillForACarrier(t *testing.T) {
	var last, current [data.SkillSlots]int32
	last[data.SkillBludgen] = 10
	current[data.SkillBludgen] = 11

	words := ui.AuthoredWords()
	got := skillRiseRows(&words, true, last, current)
	if len(got) != 1 {
		t.Fatalf("skillRiseRows = %+v, want exactly one row", got)
	}
	if want := "Air skill improved: 11"; got[0] != want {
		t.Errorf("row 0 = %q, want %q — Bludgen's mage name is Air", got[0], want)
	}
}

// TWO SLOTS RISING IN ONE COMPARE POST TWO ROWS, in slot order — the shape a
// tick that raises a slot AND crosses a threshold for another in the same
// award pipeline could produce, though no single award does both today.
func TestSkillRiseRowsPostsOneRowPerSlotThatRose(t *testing.T) {
	last := [data.SkillSlots]int32{0, 10, 0, 0, 0, 0}
	current := [data.SkillSlots]int32{0, 10, 5, 0, 0, 1}

	words := ui.AuthoredWords()
	got := skillRiseRows(&words, false, last, current)
	if len(got) != 2 {
		t.Fatalf("skillRiseRows = %+v, want exactly two rows", got)
	}
	if got[0] != "Axe skill improved: 5" {
		t.Errorf("row 0 = %q, want \"Axe skill improved: 5\" — slot 2 rose and comes first in slot order", got[0])
	}
	if got[1] != "Shooting skill improved: 1" {
		t.Errorf("row 1 = %q, want \"Shooting skill improved: 1\" — slot 5 rose and comes second", got[1])
	}
}

// AC-9's OTHER HALF: nothing crossed, so nothing posts. The two arrays are
// identical — the exact shape skillRises hands this function on a tick that
// raised no level anywhere — and equal, unmoved, or fallen (a slot reading
// LOWER than last time, no feed in this tree produces but the compare must
// still refuse) all answer nil alike, since the test is current > last and
// nothing else.
func TestSkillRiseRowsWithNothingRisenAnswersNil(t *testing.T) {
	same := [data.SkillSlots]int32{0, 10, 0, 3, 0, 0}
	words := ui.AuthoredWords()
	if got := skillRiseRows(&words, false, same, same); got != nil {
		t.Errorf("skillRiseRows(same, same) = %+v, want nil", got)
	}

	last := [data.SkillSlots]int32{0, 10, 0, 0, 0, 0}
	fallen := [data.SkillSlots]int32{0, 9, 0, 0, 0, 0}
	if got := skillRiseRows(&words, false, last, fallen); got != nil {
		t.Errorf("skillRiseRows(last, fallen) = %+v, want nil — a drop is not a rise", got)
	}
}

func TestARaiseEarnsOneRowAndATickWithoutOneEarnsNothing(t *testing.T) {
	const hero, victim = sim.EntityID(7), sim.EntityID(8)
	table := eqDefsTable(t)
	h := eqBladeHero()
	reward := h.Reward()

	// THE BLOW'S OWN NUMBERS ARE ON THE FIXTURE, not folded in by rearm: this
	// test advances the world directly (below) and so never reaches the
	// per-frame recompute that would otherwise supply them.
	a := sim.Entity{ID: hero, X: 3, Y: 3, HP: 100, MaxHP: 100, Reach: 1,
		Owner: 2, GainsXP: true, TypeID: sim.HumanTypeID, Mind: reward.Mind, SkillXP: reward.SkillXP,
		XPSlot: uint8(data.SkillBlade), DamageBase: 5, AlwaysHits: true,
		AttackCharge: 1, AttackRelax: 1}
	a.Skill[data.SkillBlade] = h.Skill[data.SkillBlade]
	v := sim.Entity{ID: victim, X: 4, Y: 3, HP: 1_000_000, MaxHP: 1_000_000,
		DyingTime: 200, Owner: 3, XPValue: 4}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, sim.Terrain{}, []sim.Entity{a, v}, nil, sim.Relations{}, nil, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, hero, h, eqSword(t, table), table)

	if rows := mw.skillRises(); rows != nil {
		t.Fatalf("the baseline tick earned %v, want nothing — opening levels are not a raise", rows)
	}
	if rows := mw.skillRises(); rows != nil {
		t.Fatalf("a tick on which nothing moved earned %v, want nothing", rows)
	}

	// The world is advanced here rather than through mw.tick, because tick
	// calls skillRises itself and would consume the very comparison this
	// test is reading. The attack order is the one mw.commands would have
	// built from mw.pending.
	//
	// Blade stands at exactly S(10); the first landed blow of one point or
	// more therefore carries it strictly past that boundary and raises it.
	var rows []string
	for i := 0; i < 4000 && rows == nil; i++ {
		sim.Step(mw.world, []sim.Command{{Kind: sim.KindAttack, Entity: hero, X: int32(victim)}})
		rows = mw.skillRises()
	}
	want := []string{"Blade skill improved: 11"}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("the raise earned %v, want %v", rows, want)
	}
	if rows := mw.skillRises(); rows != nil {
		t.Errorf("the tick after the raise earned %v, want nothing — a raise announces once", rows)
	}
}

// ---------------------------------------------------------------------------
// 0151, defect 4: unequip. enqueueUnequip's own applicability question — is
// there a subject, is the slot in range — mirrors enqueueEquip's shape one
// tier down (world.go's own doc). The combat-block re-derivation after a
// removal is TestUnequippingAnArmourPieceDropsItsDefenceAndAbsorption
// (rearm_test.go), which needs gaTable's own armour fixture and stands
// there.
// ---------------------------------------------------------------------------

// TestEnqueueUnequipAppendsACommandForTheNamedSlot mirrors
// TestEnqueueEquipAppendsACommandForAResolvableWeaponCode (above), over the
// inverse move: a subject wearing a sword in slot 1, unequipFromWorn's own
// zero-based cell index 0 turns into KindUnequip{X: 1} — the one-based slot
// number pkg/sim reads (equip.go).
func TestEnqueueUnequipAppendsACommandForTheNamedSlot(t *testing.T) {
	table := eqDefsTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	sim.Step(w, []sim.Command{{Kind: sim.KindEquip, Entity: 7, X: 0, Y: 1}}) // slot 1 now holds the sword
	mw := equipMission(t, w, 7, eqHero(), nil, table)

	mw.enqueueUnequip(0) // worn cell 0 (Slots' own zero-based index for slot 1)

	if len(mw.pending) != 1 {
		t.Fatalf("pending = %v, want exactly one command", mw.pending)
	}
	want := sim.Command{Kind: sim.KindUnequip, Entity: 7, X: 1}
	if got := mw.pending[0]; got != want {
		t.Errorf("pending[0] = %+v, want %+v", got, want)
	}
}

// TestEnqueueUnequipDoesNothingWithNoSubject is enqueueEquip's own first
// guard (world.go's doc), restated for the inverse: a mapWorld with no
// inventory subject appends nothing, whatever slot is named.
func TestEnqueueUnequipDoesNothingWithNoSubject(t *testing.T) {
	table := eqDefsTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	sim.Step(w, []sim.Command{{Kind: sim.KindEquip, Entity: 7, X: 0, Y: 1}})
	mw := equipMission(t, w, 7, eqHero(), nil, table)
	mw.invSubjectSet = false

	mw.enqueueUnequip(0)

	if len(mw.pending) != 0 {
		t.Errorf("pending = %v, want none — no subject is set", mw.pending)
	}
}

// TestEnqueueUnequipDoesNothingForAnOutOfRangeSlot is the defensive bound
// enqueueUnequip's own doc names: every idx TakeInventoryUnequip can hand
// back already names one of the twelve worn cells, so this is a fence
// rather than a reachable case, witnessed the same way regardless.
func TestEnqueueUnequipDoesNothingForAnOutOfRangeSlot(t *testing.T) {
	table := eqDefsTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, table)

	for _, idx := range []int{-1, sim.EquipSlots} {
		mw.enqueueUnequip(idx)
	}

	if len(mw.pending) != 0 {
		t.Errorf("pending = %v, want none — both slots are out of range", mw.pending)
	}
}

// TestUnequippingASwordMovesTheEntitysDamageBaseBackToBare is
// TestRearmMovesTheEntitysDamageBaseToTheNewLoadout's own inverse: equip a
// sword (DamageBase moves off its starting value), then unequip it through
// the SAME command-and-tick path, and DamageBase returns to the bare hero's
// own recompute — the fixture carries no starting weapon, so "bare" is
// hero.Recompute over an empty Loadout, the fallback TestRearmFallsBackTo-
// TheStartingWeaponWhenSlotOneIsEmpty already reads for a nil startWeapon.
//
// TO CONFIRM THIS TEST WITNESSES THE RE-DERIVATION, comment out the
// mw.rearm() call in tick (world.go) or the KindUnequip arm in stepWorld's
// switch (pkg/sim/step.go): the code leaves slot 1 (the sim.Equipped
// answer moves) while DamageBase keeps the sword's own value, and the
// assertion below reddens on the number — a removed weapon that goes on
// hitting as hard as it did worn.
func TestUnequippingASwordMovesTheEntitysDamageBaseBackToBare(t *testing.T) {
	table := eqDefsTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	hero := eqHero()
	mw := equipMission(t, w, 7, hero, nil, table) // no starting weapon: a bare hero
	at := func() sim.Entity {
		t.Helper()
		e, ok := mw.entity(7)
		if !ok {
			t.Fatal("entity 7 is gone")
		}
		return e
	}

	mw.enqueueEquip(0)
	mw.tick()
	sword := eqSword(t, table)
	wantSword := hero.Recompute(data.Profile{}, data.Loadout{Weapon: sword}).Combat.DamageBase
	if got := at().DamageBase; got != wantSword {
		t.Fatalf("setup: DamageBase after the equip is %d, want %d (the sword) — the assertion "+
			"below could not discriminate", got, wantSword)
	}
	if slots, _ := w.Equipped(7); slots[0] != eqSwordCode {
		t.Fatalf("setup: Equipped(7)[0] = %#x, want %#x", slots[0], eqSwordCode)
	}

	mw.enqueueUnequip(0)
	mw.tick()

	wantBare := hero.Recompute(data.Profile{}, data.Loadout{}).Combat.DamageBase
	if wantBare == wantSword {
		t.Fatalf("setup: bare (%d) and sworded (%d) DamageBase coincide; choose a different "+
			"fixture so the assertion below can discriminate", wantBare, wantSword)
	}
	if got := at().DamageBase; got != wantBare {
		t.Errorf("DamageBase = %d, want %d (bare hero, no weapon) — the sword's own value %d "+
			"survived the unequip", got, wantBare, wantSword)
	}
	if slots, _ := w.Equipped(7); slots[0] != 0 {
		t.Errorf("Equipped(7)[0] = %#x, want 0 (empty)", slots[0])
	}
	if codes, _ := w.Carried(7); !equalUint16(codes, []uint16{eqSwordCode}) {
		t.Errorf("Carried(7) = %#x, want [%#x] — the sword returned to the pack", codes, eqSwordCode)
	}
}

// TestUnequippingTheStartingWeaponMovesDamageBaseToBareNotToTheStartingWeapon
// is the owner's report (hotfix b51b439+1): "when you take a hero's weapon
// off, his damage does not become bare-handed".
//
// TestUnequippingASwordMovesTheEntitysDamageBaseBackToBare above does not
// witness this: its fixture passes a nil startWeapon to equipMission, so
// ResolveEquipmentLoadout's fallback was already nil before this hotfix and
// the unequip could not have exercised it either way. This test's hero
// starts the mission already wearing the sword in slot 1, with the sword
// also standing as his invParty.startWeapon — the shape assembleParty
// (hero.go) builds for a generated or restored party member since
// 0134-what-he-wears, where Worn[0] carries the starting weapon's own code
// from the first tick rather than the zero value the other test's fixture
// leaves it at.
//
// TO CONFIRM IT WITNESSES THE FIX, pass mw.invParty.startWeapon to Rearm
// unconditionally in rearm (pkg/game/world.go) — the tree exactly as it was
// before this hotfix — and rerun: DamageBase after the unequip reads the
// sword's own value, not the bare hero's, and every other test in this file
// stays green.
func TestUnequippingTheStartingWeaponMovesDamageBaseToBareNotToTheStartingWeapon(t *testing.T) {
	table := eqDefsTable(t)
	sword := eqSword(t, table)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Equipped: [sim.EquipSlots]uint16{0: eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	hero := eqHero()

	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{{Hero: hero, Weapon: sword, Worn: [sim.EquipSlots]uint16{0: eqSwordCode}}},
		Start: mapload.Start{IDs: []sim.EntityID{7}}}
	mw := openMission(ms, table, nil, v, missionSource{}, nil, nil)

	if slots, _ := w.Equipped(7); slots[0] != eqSwordCode {
		t.Fatalf("setup: Equipped(7)[0] = %#x, want %#x (the sword, worn from the open)", slots[0], eqSwordCode)
	}

	mw.enqueueUnequip(0)
	mw.tick()

	wantSword := hero.Recompute(data.Profile{}, data.Loadout{Weapon: sword}).Combat.DamageBase
	wantBare := hero.Recompute(data.Profile{}, data.Loadout{}).Combat.DamageBase
	if wantBare == wantSword {
		t.Fatalf("setup: bare (%d) and sworded (%d) DamageBase coincide; choose a different "+
			"fixture so the assertion below can discriminate", wantBare, wantSword)
	}
	e, ok := mw.entity(7)
	if !ok {
		t.Fatal("entity 7 is gone after the unequip")
	}
	if e.DamageBase != wantBare {
		t.Errorf("DamageBase = %d, want %d (bare hero, no weapon) — the starting weapon's own "+
			"value %d survived the unequip", e.DamageBase, wantBare, wantSword)
	}
	if slots, _ := w.Equipped(7); slots[0] != 0 {
		t.Errorf("Equipped(7)[0] = %#x, want 0 (empty)", slots[0])
	}
	if codes, _ := w.Carried(7); !equalUint16(codes, []uint16{eqSwordCode}) {
		t.Errorf("Carried(7) = %#x, want [%#x] — the sword returned to the pack", codes, eqSwordCode)
	}
}

// TestRaisingASkillAfterUnequippingTheStartingWeaponKeepsTheHeroBare is
// DIV-067's own witness for recomputeRaisedSkills (rearm.go): a hero takes
// his starting weapon off, then his level rises with no equip command on
// that tick, and the recompute the raise triggers must read his empty slot
// 1 as taken off (bare) rather than never armed (the starting weapon) —
// the same case TestUnequippingTheStartingWeaponMovesDamageBaseToBareNot-
// ToTheStartingWeapon (above) proves for the EQUIP-triggered recompute,
// proved here for the RAISE-triggered one instead.
//
// THE CAST IS DRIVEN THROUGH sim.Step DIRECTLY AND NOT mw.tick(), on the
// tick the level actually rises. mw.tick() runs two separate recomputes on
// that tick, mw.recomputeRaisedSkills() first and mw.rearm() second
// (world.go's own tick), and mw.rearm() re-derives the SAME subject
// correctly regardless of what recomputeRaisedSkills wrote, because it
// reads mw.invWeaponEverEquipped directly rather than through a
// subject-scoped expression — there is only one subject for it to be
// scoped to. Going through mw.tick() would let rearm's own correct pass
// overwrite whatever recomputeRaisedSkills wrote on the very same tick, so
// the entity's end-of-tick state would read correctly whether or not
// recomputeRaisedSkills' own value was right. Calling
// mw.recomputeRaisedSkills() directly, once, after the world has already
// advanced past the raise, isolates its own contribution from rearm's.
//
// THE SKILL RISES THROUGH A CAST AND NOT A BLOW because the hero is BARE at
// the moment it rises: a bare wielder's own credited slot is SkillGeneral
// (data.activeSkill, pkg/data/hero.go), and awardSkill (pkg/sim/skill.go)
// refuses a non-mage award naming slot 0 outright — a bare hero cannot
// land a blow that pays himself anything. A mage's own school credit
// (awardSkill's other arm) is gated on isMage (MaxMana > 0) alone and does
// not read XPSlot at all, so it is the one XP source this fixture's bare
// hero can still earn from. Defensive and self-targeting, on
// TestTheSheetsProtectionRowStatesTheLiveValueWhileAnEffectStands' own
// spell rule (spellsheet1001_test.go), because a cast at oneself is not
// refused as "the caster itself" the way an offensive cast at a victim
// would be, and this test needs no victim at all.
//
// TO CONFIRM THIS TEST WITNESSES THE SUBJECT SCOPING, pass false
// unconditionally for everEquipped at recomputeRaisedSkills' own call to
// mapload.ResolveEquipmentLoadout (rearm.go) and rerun: DamageBase after
// the raise reads the sword's own value and the assertion below reddens on
// the number.
func TestRaisingASkillAfterUnequippingTheStartingWeaponKeepsTheHeroBare(t *testing.T) {
	table := eqDefsTable(t)
	sword := eqSword(t, table)
	const hero = sim.EntityID(7)
	const school = uint8(1)

	rule := sim.SpellRule{ID: 5, ManaCost: 1, School: school, MaxRange: 6, TargetsUnit: true,
		Defensive: true, SpellDuration: 4, EffectKind: sim.EffectProtectionFire,
		EffectMode: sim.EffectDuration}
	caster := sim.Entity{ID: hero, X: 3, Y: 3, HP: 100, MaxHP: 100,
		Mana: 100, MaxMana: 100, Mind: 60, KnownSpells: 1 << 5, TokenSize: 1,
		GainsXP: true, TypeID: sim.HumanTypeID}
	w, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, sim.Terrain{}, []sim.Entity{caster}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: hero, Equipped: [sim.EquipSlots]uint16{0: eqSwordCode}}},
		[]sim.SpellRule{rule})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	hero7 := eqHero()
	hero7.Mind = 60

	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{{Hero: hero7, Profile: data.Profile{ManaColumn: true}, Weapon: sword, Worn: [sim.EquipSlots]uint16{0: eqSwordCode}}},
		Start: mapload.Start{IDs: []sim.EntityID{hero}}}
	mw := openMission(ms, table, nil, v, missionSource{}, nil, nil)

	if slots, _ := w.Equipped(hero); slots[0] != eqSwordCode {
		t.Fatalf("setup: Equipped(%d)[0] = %#x, want %#x (the sword, worn from the open)", hero, slots[0], eqSwordCode)
	}

	mw.enqueueUnequip(0)
	mw.tick()

	if slots, _ := w.Equipped(hero); slots[0] != 0 {
		t.Fatalf("setup: Equipped(%d)[0] = %#x after the unequip, want 0 (empty)", hero, slots[0])
	}
	if !mw.invWeaponEverEquipped {
		t.Fatal("setup: invWeaponEverEquipped is false after the unequip — the fixture cannot discriminate")
	}
	sim.Step(w, []sim.Command{{Kind: sim.KindCast, Entity: hero, X: int32(hero), Y: int32(rule.ID)}})
	var raised bool
	for i := 0; i < 64 && !raised; i++ {
		sim.Step(w, nil)
		for _, e := range w.Entities() {
			if e.ID == hero && e.Skill[school] > 0 {
				raised = true
			}
		}
	}
	if !raised {
		t.Fatal("setup: the caster's own school skill never rose after the cast — the fixture cannot exercise a raise")
	}

	// The recompute under test runs ALONE here, never through mw.tick() —
	// see the doc above for why going through the tick would mask it.
	mw.recomputeRaisedSkills()

	raisedHero := hero7
	for _, e := range w.Entities() {
		if e.ID == hero {
			raisedHero.Skill = e.Skill
		}
	}
	wantBare := raisedHero.Recompute(data.Profile{}, data.Loadout{}).Combat.DamageBase
	wantSword := raisedHero.Recompute(data.Profile{}, data.Loadout{Weapon: sword}).Combat.DamageBase
	if wantBare == wantSword {
		t.Fatalf("setup: bare (%d) and sworded (%d) DamageBase coincide at the raised level; "+
			"choose a different fixture so the assertion below can discriminate", wantBare, wantSword)
	}

	e, ok := mw.entity(hero)
	if !ok {
		t.Fatal("entity is gone after recomputeRaisedSkills")
	}
	if e.DamageBase != wantBare {
		t.Errorf("DamageBase = %d, want %d (bare hero, no weapon) — the starting weapon's own value %d "+
			"reappeared on the raise, after the weapon had already been taken off", e.DamageBase, wantBare, wantSword)
	}
	if slots, _ := w.Equipped(hero); slots[0] != 0 {
		t.Errorf("Equipped(%d)[0] = %#x, want 0 (empty) — recomputeRaisedSkills must not have re-armed the slot itself", hero, slots[0])
	}
}

// equalUint16 is a plain slice-equality helper for the []uint16 Carried
// answers this file's own unequip tests compare — pkg/sim's own equalCodes
// (equip_test.go there) is unexported outside that package.
func equalUint16(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
