package game

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/databin"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestSimDerivedBlockCarriesEveryResistanceByte is the game-to-simulation
// boundary used by both mission restore and live Rearm. Signed derived values
// narrow once modulo 256 and arrive in the combat block without a 0..100 clamp.
func TestSimDerivedBlockCarriesEveryResistanceByte(t *testing.T) {
	d := data.Derived{Resistance: [5]int32{0, 100, 255, 256, -1}}
	got := simDerivedBlock(d, 0, sim.WeaponSpellNone).Combat.Resistance
	want := [5]uint8{0, 100, 255, 0, 255}
	if got != want {
		t.Errorf("Resistance = %v, want %v", got, want)
	}
}

// gaIdentityRow is a shape or material row carrying IDENTITY factors — 1 —
// at every one of the four factor-ladder slots this file's fixtures read:
// 4 and 5 (a weapon's own damage and to-hit, unused by an armour but read
// when this file's tests also resolve "Sword"), 6 (a weapon's AND an
// armour's own defence slot — wear.go's own doc: the two share scaleDefenceSlot)
// and 7 (an armour's own absorption slot alone). eqScaleRow's own shape
// (equip_test.go), widened by one slot so a single row serves both classes.
func gaIdentityRow(name string) synth.DataBinRow {
	d := make([]float64, 9)
	d[4], d[5], d[6], d[7] = 1, 1, 1, 1
	return synth.DataBinRow{Name: name, Doubles: d}
}

// gaArmorRow is an Armors row over the three columns this story's own
// resolvers read: param 4 the Slot, param 9 the raw defence, param 10 the
// raw absorption — wearArmorRow's own layout (pkg/data/wear_test.go),
// rebuilt here because that helper is unexported in another package.
func gaArmorRow(name string, slot, defence, absorption int32) synth.DataBinRow {
	return gaArmorRowSuit(name, slot, defence, absorption, eqSuitAny)
}

// gaArmorRowSuit is gaArmorRow with the row's sutableFor cell stated
// (data.SutableForColumn). eqSuitAny is what gaArmorRow passes and why —
// see its own doc in equip_test.go.
func gaArmorRowSuit(name string, slot, defence, absorption, suit int32) synth.DataBinRow {
	p := []int32{-1, -1, -1, -1, slot, -1, -1, -1, -1, defence, absorption, -1, -1, -1, -1, suit}
	return synth.DataBinRow{Name: name, Params: p}
}

// gaBootsCode and gaHelmCode are the two Armors rows gaTable writes, coded
// BY HAND rather than through data.ArmorFromCode's own composition, so each
// can state its own field B independently of the row's own Slot column.
const gaBootsCode = uint16(0)<<12 | uint16(5)<<8 | uint16(0)<<5 | uint16(1)

// gaHelmCode is gaBootsCode's sibling at row 2, Slot 7 — this one composed
// with field B AGREEING with its row's Slot, the ordinary shape a real
// composition takes, so this file's fold tests are not built entirely from
// the deliberately-disagreeing case above.
const gaHelmCode = uint16(0)<<12 | uint16(7)<<8 | uint16(0)<<5 | uint16(2)

const gaNoArmorRowCode = uint16(9) << 8

// gaTable is one *mapload.Table over a real parsed definition stream,
// dedicated to this file: an identity Shapes and Materials row apiece, a
// Weapons collection carrying "Sword" at eqSwordCode's own row (equip_test.go
// — the two files share that one code because both name row 1 of an
// otherwise-empty Weapons collection), and an Armors collection of two
// written rows, gaBootsCode's and gaHelmCode's.
func gaTable(t *testing.T) *mapload.Table {
	t.Helper()
	f, err := databin.Parse(synth.DataBin{
		Rows: [synth.DataBinCollections][]synth.DataBinRow{
			synth.DataBinShapes:    {gaIdentityRow("Plain")},
			synth.DataBinMaterials: {gaIdentityRow("Steel")},
			synth.DataBinWeapons: {
				eqWeaponRow("Sword", data.SkillBlade, 10, 20, 5, 3, 2, 7, 4),
			},
			synth.DataBinArmors: {
				gaArmorRow("Boots", 12, 7, 3),
				gaArmorRow("Helm", 7, 2, 1),
			},
		},
	}.Bytes())
	if err != nil {
		t.Fatalf("databin.Parse: %v", err)
	}
	return &mapload.Table{
		Shapes: f.Collection(databin.Shapes), Materials: f.Collection(databin.Materials),
		Weapons: f.Collection(databin.Weapons), Armors: f.Collection(databin.Armors),
	}
}

// gaBoots and gaHelm are the two *data.Armor gaBootsCode and gaHelmCode
// resolve to against gaTable — computed by calling data.ArmorFromCode
// itself, eqSword's own precedent (equip_test.go): this file witnesses
// rearm.go's WIRING, not data.ArmorFromCode's own arithmetic, which
// pkg/data/wear_test.go already covers.
func gaBoots(t *testing.T, table *mapload.Table) data.Armor {
	t.Helper()
	p, err := data.ArmorFromCode(data.ItemCode(gaBootsCode), table.Shapes, table.Materials, table.Armors)
	if err != nil {
		t.Fatalf("setup: data.ArmorFromCode(gaBootsCode): %v", err)
	}
	return p
}

func gaHelm(t *testing.T, table *mapload.Table) data.Armor {
	t.Helper()
	p, err := data.ArmorFromCode(data.ItemCode(gaHelmCode), table.Shapes, table.Materials, table.Armors)
	if err != nil {
		t.Fatalf("setup: data.ArmorFromCode(gaHelmCode): %v", err)
	}
	return p
}

// gaEntity finds id in w.Entities(), Fatal-ing the test if it is gone —
// sim.World carries no per-id accessor of its own (only the bulk Entities
// and the narrow Equipped/Carried), so this is the shortest read for a test
// calling Rearm directly rather than through a mapWorld's own mw.entity.
func gaEntity(t *testing.T, w *sim.World, id sim.EntityID) sim.Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("entity %d is gone", id)
	return sim.Entity{}
}

func TestRearmCarriesTheLastSecondaryDamageEffectAsOneTriple(t *testing.T) {
	table := gaTable(t)
	const id = sim.EntityID(7)
	var equipped [sim.EquipSlots]sim.ItemInstance
	equipped[11] = sim.ItemInstance{Code: gaBootsCode, Kind: 1, Effects: []sim.ItemEffect{
		{Kind: 44, Operand: 5 | 7<<8},
		{Kind: 45, Operand: 9 | 2<<8},
	}}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{{ID: id, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil,
		sim.Relations{}, nil, []sim.Stock{{ID: id, EquippedItems: equipped}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	if _, wrote := Rearm(w, id, eqHero(), data.Profile{}, nil, false, table, 0); !wrote {
		t.Fatal("Rearm refused an enchanted armour loadout")
	}
	want := sim.SecondaryDamage{Base: 9, Spread: 2, Selector: 1}
	if got := gaEntity(t, w, id).SecondaryDamage; got != want {
		t.Fatalf("Rearm secondary damage = %+v, want %+v", got, want)
	}
}

// TestEquipTargetAnswersTheArmoursOwnRowSlot is AC-7, read directly off the
// exported gate rather than through enqueueEquip: gaBootsCode's own field B
// is 5, and EquipTarget answers 12 — the ROW's Slot column, never the pack
// index (there is none here to even supply) and never the code's own class
// field.
func TestEquipTargetAnswersTheArmoursOwnRowSlot(t *testing.T) {
	table := gaTable(t)

	slot, ok := EquipTarget(data.ItemCode(gaBootsCode), table)
	if !ok {
		t.Fatal("EquipTarget refused a code that resolves through data.ArmorFromCode")
	}
	if slot != 12 {
		t.Errorf("slot = %d, want 12 (the row's own Slot column, not field B's 5)", slot)
	}
}

func TestEquipTargetRefusesWhenTheTableIsMissingAnyOfTheFourCollections(t *testing.T) {
	full := gaTable(t)

	if slot, ok := EquipTarget(data.ItemCode(eqSwordCode), nil); ok {
		t.Errorf("a nil table: slot = %d, ok = true, want (0, false)", slot)
	}

	// A table that COULD have resolved eqSwordCode as a weapon (Shapes,
	// Materials and Weapons all present) but is missing Armors — refused
	// all the same, the fence's own "including" clause.
	noArmors := &mapload.Table{Shapes: full.Shapes, Materials: full.Materials, Weapons: full.Weapons}
	if slot, ok := EquipTarget(data.ItemCode(eqSwordCode), noArmors); ok {
		t.Errorf("a table missing Armors, weapon code: slot = %d, ok = true, want (0, false)", slot)
	}

	// And the symmetric case: a table missing Weapons refuses an armour
	// code too, though ArmorFromCode alone would not have needed Weapons at
	// all — the four ride together as one fence, not two independent ones.
	noWeapons := &mapload.Table{Shapes: full.Shapes, Materials: full.Materials, Armors: full.Armors}
	if slot, ok := EquipTarget(data.ItemCode(gaBootsCode), noWeapons); ok {
		t.Errorf("a table missing Weapons, armour code: slot = %d, ok = true, want (0, false)", slot)
	}
}

// TestEquipTargetRefusesACodeThatResolvesToNeitherClass is the gate's
// remaining totality: a code that fails both data.WeaponFromCode (wrong
// class) and data.ArmorFromCode (no written row) is refused rather than
// answering either function's partial state.
func TestEquipTargetRefusesACodeThatResolvesToNeitherClass(t *testing.T) {
	table := gaTable(t)

	if slot, ok := EquipTarget(data.ItemCode(gaNoArmorRowCode), table); ok {
		t.Errorf("slot = %d, ok = true, want (0, false) — the code names no written Armors row", slot)
	}
}

// TestEnqueueEquipAppendsAnArmourCommandAtTheRowsOwnSlotNotThePackIndex is
// AC-7 restated at the front end, TestEnqueueEquipAppendsACommandForA-
// ResolvableWeaponCode's own shape (equip_test.go) with an armour in place
// of a weapon: gaBootsCode sits at pack index 1, and the command's Y is 12 —
// neither 1 (the index) nor 5 (the code's own field B).
func TestEnqueueEquipAppendsAnArmourCommandAtTheRowsOwnSlotNotThePackIndex(t *testing.T) {
	table := gaTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqNoSlotCode, gaBootsCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, table)

	mw.enqueueEquip(1)

	if len(mw.pending) != 1 {
		t.Fatalf("pending = %v, want exactly one command", mw.pending)
	}
	got := mw.pending[0]
	want := sim.Command{Kind: sim.KindEquip, Entity: 7, X: 1, Y: 12}
	if got != want {
		t.Errorf("pending[0] = %+v, want %+v", got, want)
	}
}

// TestEquippingAnArmourRaisesDefenceAndAbsorptionWhileLeavingTheWeaponsNumbersUntouched
// is T2's own required witness: an armour equipped through the ORDINARY
// COMMAND PATH — enqueueEquip, then a tick — moves the subject's Defence and
// Absorption by exactly the equipped piece's own two values, and moves
// nothing else (0136 spec AC-6, AC-7, AC-10; plan SC-4).
//
// THE SUBJECT ALREADY HOLDS A SWORD when the armour goes on — AC-10's own
// wording, "a subject wearing a weapon, who equips an armour" — so this test
// also witnesses FR-6a's "and nothing else": DamageBase, DamageSpread,
// ToHit, AttackCharge, AttackRelax, AlwaysHits, Reach and XPSlot are read
// off the entity BEFORE the armour goes on and compared for EXACT equality
// after, not merely "still nonzero".
func TestEquippingAnArmourRaisesDefenceAndAbsorptionWhileLeavingTheWeaponsNumbersUntouched(t *testing.T) {
	table := gaTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqSwordCode, gaBootsCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, table)

	mw.enqueueEquip(0) // the sword, index 0
	mw.tick()
	before := gaEntity(t, w, 7)

	mw.enqueueEquip(0) // the sword is gone; the boots shifted down to index 0
	mw.tick()
	after := gaEntity(t, w, 7)

	boots := gaBoots(t, table)
	if got, want := after.Defence, before.Defence+boots.Defence; got != want {
		t.Errorf("Defence = %d, want %d (before %d + the boots' own %d)", got, want, before.Defence, boots.Defence)
	}
	if got, want := after.Absorption, before.Absorption+boots.Absorption; got != want {
		t.Errorf("Absorption = %d, want %d (before %d + the boots' own %d)", got, want, before.Absorption, boots.Absorption)
	}
	if boots.Defence == 0 && boots.Absorption == 0 {
		t.Fatal("setup: the boots contribute nothing, so the two assertions above prove nothing")
	}

	// Every number the WEAPON set: unmoved, exactly.
	for name, pair := range map[string][2]int32{
		"DamageBase":   {before.DamageBase, after.DamageBase},
		"DamageSpread": {before.DamageSpread, after.DamageSpread},
		"ToHit":        {before.ToHit, after.ToHit},
		"AttackCharge": {before.AttackCharge, after.AttackCharge},
		"AttackRelax":  {before.AttackRelax, after.AttackRelax},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s moved from %d to %d — an armour must move only Defence and Absorption", name, pair[0], pair[1])
		}
	}
	if before.AlwaysHits != after.AlwaysHits {
		t.Errorf("AlwaysHits moved from %v to %v", before.AlwaysHits, after.AlwaysHits)
	}
	if before.Reach != after.Reach {
		t.Errorf("Reach moved from %d to %d", before.Reach, after.Reach)
	}
	if before.XPSlot != after.XPSlot {
		t.Errorf("XPSlot moved from %d to %d — a sword's own credited skill must not move for an armour", before.XPSlot, after.XPSlot)
	}

	if codes, _ := w.Carried(7); len(codes) != 0 {
		t.Errorf("Carried(7) = %v, want empty — both codes moved into equipment", codes)
	}
	slots, _ := w.Equipped(7)
	if slots[0] != eqSwordCode {
		t.Errorf("Equipped(7)[0] = %#x, want %#x (the sword)", slots[0], eqSwordCode)
	}
	if slots[11] != gaBootsCode {
		t.Errorf("Equipped(7)[11] = %#x, want %#x (the boots, slot 12) — a piece landing in a slot "+
			"other than the first (FR-10)", slots[11], gaBootsCode)
	}
}

func TestRearmSumsTwoWornArmourPiecesWithEachSumOnItsOwnStatistic(t *testing.T) {
	table := gaTable(t)
	const id = sim.EntityID(7)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{gaBootsCode, gaHelmCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	sim.Step(w, []sim.Command{
		{Kind: sim.KindEquip, Entity: id, X: 0, Y: 12}, // boots, straight into slot 12
		{Kind: sim.KindEquip, Entity: id, X: 0, Y: 7},  // the helm, now at index 0, into slot 7
	})

	boots, helm := gaBoots(t, table), gaHelm(t, table)
	if boots.Defence+helm.Defence == boots.Absorption+helm.Absorption {
		t.Fatalf("setup: the two sums coincide (%d), so this test cannot discriminate a swap",
			boots.Defence+helm.Defence)
	}

	hero := eqHero()
	weapon, wrote := Rearm(w, id, hero, data.Profile{}, nil, false, table, 0)
	if !wrote {
		t.Fatal("Rearm refused to write, want it to succeed against a full table and two resolvable pieces")
	}
	if weapon != nil {
		t.Errorf("weapon = %+v, want nil — neither slot in this fixture holds one", weapon)
	}

	// bare is hero's own recompute over an EMPTY loadout — his Reaction
	// alone already gives him a nonzero base Defence (recompute.go step 8),
	// so the two pieces' sums land ON TOP of that rather than replacing it.
	bare := hero.Recompute(data.Profile{}, data.Loadout{}).Combat
	e := gaEntity(t, w, id)
	if want := bare.Defence + boots.Defence + helm.Defence; e.Defence != want {
		t.Errorf("Defence = %d, want %d (bare %d + boots %d + helm %d)",
			e.Defence, want, bare.Defence, boots.Defence, helm.Defence)
	}
	if want := bare.Absorption + boots.Absorption + helm.Absorption; e.Absorption != want {
		t.Errorf("Absorption = %d, want %d (bare %d + boots %d + helm %d)",
			e.Absorption, want, bare.Absorption, boots.Absorption, helm.Absorption)
	}
}

// This is the whole production lifetime of an opening loadout: mission mint,
// a presentation-only first tick, ordinary unequip, campaign carry into a new
// map, ordinary equip, and another no-op tick. It exists beside the focused
// Rearm tests because the owner-visible failure was at the seam between them:
// the worn codes survived while the mission constructor derived a bare body.
// The two fixture pieces deliberately contribute non-equal, nonzero Defence
// and Absorption so either a dropped fold or a field swap is visible.
func TestOpeningArmourSurvivesUnequipEquipAndMapTransitionWithoutDoubleApplication(t *testing.T) {
	table := gaTable(t)
	hero := eqHero()
	boots, helm := gaBoots(t, table), gaHelm(t, table)
	if boots.Absorption == 0 || helm.Absorption == 0 {
		t.Fatalf("setup: lawful synthetic armour needs nonzero absorption, got boots=%+v helm=%+v", boots, helm)
	}
	member := mapload.PartyMember{ID: "mage", Name: "Mage", Hero: hero, Mage: true}
	member.Worn[11], member.Worn[6] = gaBootsCode, gaHelmCode

	missionMap := func(name string) *alm.Map {
		const width, height = 128, 128
		return &alm.Map{Name: name, Width: width, Height: height,
			Tiles: make([]uint16, width*height), Overlay: make([]uint8, width*height)}
	}
	world, start, err := mapload.StartMission(missionMap("first"), table, mapload.DifficultyNormal,
		[]mapload.PartyMember{member})
	if err != nil {
		t.Fatalf("StartMission(first): %v", err)
	}
	id := start.IDs[0]
	bare := hero.Recompute(data.Profile{}, data.Loadout{}).Combat
	wantFullDefence := bare.Defence + boots.Defence + helm.Defence
	wantFullAbsorption := bare.Absorption + boots.Absorption + helm.Absorption
	assertCombat := func(label string, w *sim.World, entity sim.EntityID, wantDefence, wantAbsorption int32) {
		t.Helper()
		got := gaEntity(t, w, entity)
		if got.Defence != wantDefence || got.Absorption != wantAbsorption {
			t.Fatalf("%s defence/absorption = %d/%d, want %d/%d",
				label, got.Defence, got.Absorption, wantDefence, wantAbsorption)
		}
	}
	assertCombat("opening mission", world, id, wantFullDefence, wantFullAbsorption)

	first := equipMission(t, world, id, hero, nil, table)
	first.tick()
	assertCombat("presentation-only first tick", world, id, wantFullDefence, wantFullAbsorption)

	first.enqueueUnequip(11)
	first.tick()
	wantHelmDefence := bare.Defence + helm.Defence
	wantHelmAbsorption := bare.Absorption + helm.Absorption
	assertCombat("after ordinary unequip", world, id, wantHelmDefence, wantHelmAbsorption)

	carried := mapload.CarryParty([]mapload.PartyMember{member}, world, start.IDs)
	if carried[0].Carry == nil || carried[0].Carry.Equipped[6] != gaHelmCode ||
		carried[0].Carry.Equipped[11] != 0 {
		t.Fatalf("carried equipment = %+v, want helm worn and boots removed", carried[0].Carry)
	}
	secondWorld, secondStart, err := mapload.StartMission(missionMap("second"), table,
		mapload.DifficultyNormal, carried)
	if err != nil {
		t.Fatalf("StartMission(second): %v", err)
	}
	secondID := secondStart.IDs[0]
	assertCombat("after map transition", secondWorld, secondID, wantHelmDefence, wantHelmAbsorption)

	second := equipMission(t, secondWorld, secondID, hero, nil, table)
	second.enqueueEquip(0)
	second.tick()
	assertCombat("after ordinary re-equip", secondWorld, secondID, wantFullDefence, wantFullAbsorption)
	second.tick()
	assertCombat("after unchanged follow-up tick", secondWorld, secondID, wantFullDefence, wantFullAbsorption)
}

// TestRearmReproducesTodaysRefusalForANilTableWithSlotOneOccupied and
// TestRearmReproducesTodaysRefusalForASlotOneCodeThatDoesNotResolve are the
// two refusals Rearm's own doc names, called directly so each is witnessed
// without the front-end gate (which already prevents both from arising
// through enqueueEquip) standing in the way. Each must write NOTHING: the
// entity's own starting DamageBase — an arbitrary sentinel no recompute
// below could coincide with — must survive untouched.
func TestRearmReproducesTodaysRefusalForANilTableWithSlotOneOccupied(t *testing.T) {
	const before = 999
	const id = sim.EntityID(7)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10, DamageBase: before}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	sim.Step(w, []sim.Command{{Kind: sim.KindEquip, Entity: id, X: 0, Y: 1}}) // slot 1 now holds the sword's code

	weapon, wrote := Rearm(w, id, eqHero(), data.Profile{}, nil, false, nil, 0) // no table at all
	if wrote {
		t.Error("wrote = true, want false — there is no table to resolve slot 1's code against")
	}
	if weapon != nil {
		t.Errorf("weapon = %+v, want nil", weapon)
	}
	if e := gaEntity(t, w, id); e.DamageBase != before {
		t.Errorf("DamageBase = %d, want %d (untouched) — Rearm must write nothing on this refusal", e.DamageBase, before)
	}
}

func TestRearmReproducesTodaysRefusalForASlotOneCodeThatDoesNotResolve(t *testing.T) {
	const before = 999
	const id = sim.EntityID(7)
	table := gaTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10, DamageBase: before}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqNoRowCode}}}) // names the weapon class, no row a resolve can match
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	sim.Step(w, []sim.Command{{Kind: sim.KindEquip, Entity: id, X: 0, Y: 1}})

	weapon, wrote := Rearm(w, id, eqHero(), data.Profile{}, nil, false, table, 0)
	if wrote {
		t.Error("wrote = true, want false — slot 1's code resolves to nothing")
	}
	if weapon != nil {
		t.Errorf("weapon = %+v, want nil", weapon)
	}
	if e := gaEntity(t, w, id); e.DamageBase != before {
		t.Errorf("DamageBase = %d, want %d (untouched) — Rearm must write nothing on this refusal", e.DamageBase, before)
	}
}

// TestRearmFoldsNothingFromATableMissingArmoursAloneButStillResolvesTheWeapon
// documents Rearm's own leniency, mapload.Table's armorItems() precedent
// (spec.go's own doc on Table): a table present for Shapes, Materials and
// Weapons but missing Armors still resolves slot 1's weapon exactly as it
// always has — nothing about the WEAPON half of a loadout is held hostage
// to whether the same table can say what an armour is worth. This table
// could never have reached this call through enqueueEquip's own gate (which
// refuses on the same missing collection, rearm.go's own doc), but Rearm
// itself is called directly here to prove the leniency exists rather than
// merely appears to, by never being tested against its absence.
func TestRearmFoldsNothingFromATableMissingArmoursAloneButStillResolvesTheWeapon(t *testing.T) {
	const id = sim.EntityID(7)
	full := gaTable(t)
	partial := &mapload.Table{Shapes: full.Shapes, Materials: full.Materials, Weapons: full.Weapons}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	sim.Step(w, []sim.Command{{Kind: sim.KindEquip, Entity: id, X: 0, Y: 1}})

	hero := eqHero()
	weapon, wrote := Rearm(w, id, hero, data.Profile{}, nil, false, partial, 0)
	if !wrote {
		t.Fatal("wrote = false, want true — a table missing only Armors must still resolve the weapon")
	}
	sword := eqSword(t, full)
	want := hero.Recompute(data.Profile{}, data.Loadout{Weapon: sword}).Combat.DamageBase
	if e := gaEntity(t, w, id); e.DamageBase != want {
		t.Errorf("DamageBase = %d, want %d (the sword alone, no armour folded)", e.DamageBase, want)
	}
	if weapon == nil || weapon.Name != sword.Name {
		t.Errorf("weapon = %+v, want the resolved sword %+v", weapon, sword)
	}
}

func TestRearmPreservesAuthoritativeWeaponSpellSources(t *testing.T) {
	const id = sim.EntityID(7)
	table := gaTable(t)
	for _, source := range []sim.WeaponSpellSource{sim.WeaponSpellInnate, sim.WeaponSpellLegacy} {
		t.Run(sourceName(source), func(t *testing.T) {
			w, err := sim.NewStockedWorld(1,
				sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
				[]sim.Entity{{ID: id, X: 3, Y: 3, HP: 10, MaxHP: 10,
					WeaponSpell: 14, WeaponSpellLevel: 99, WeaponSpellSource: source}},
				nil, sim.Relations{}, nil, nil)
			if err != nil {
				t.Fatalf("NewStockedWorld: %v", err)
			}
			if _, wrote := Rearm(w, id, eqHero(), data.Profile{}, nil, true, table, 0); !wrote {
				t.Fatal("Rearm refused an empty loadout")
			}
			e := gaEntity(t, w, id)
			if e.WeaponSpellSource != source || e.WeaponSpell != 14 || e.WeaponSpellLevel != 99 {
				t.Fatalf("Rearm changed authoritative source %d to (%d,%d,%d)", source,
					e.WeaponSpellSource, e.WeaponSpell, e.WeaponSpellLevel)
			}
		})
	}

	cast := sim.ItemInstance{Code: eqSwordCode, Kind: 2,
		Effects: []sim.ItemEffect{{Kind: 41, Operand: uint32(7) | uint32(uint16(15))<<16}}}
	w, err := sim.NewStockedWorld(1,
		sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: id, X: 3, Y: 3, HP: 10, MaxHP: 10,
			WeaponSpell: 7, WeaponSpellLevel: 15, WeaponSpellSource: sim.WeaponSpellItem}},
		nil, sim.Relations{}, nil, []sim.Stock{{ID: id,
			EquippedItems: [sim.EquipSlots]sim.ItemInstance{cast}}})
	if err != nil {
		t.Fatalf("NewStockedWorld(Item): %v", err)
	}
	if _, wrote := Rearm(w, id, eqHero(), data.Profile{}, nil, true, table, 0); !wrote {
		t.Fatal("Rearm refused the item-cast loadout")
	}
	e := gaEntity(t, w, id)
	if e.WeaponSpellSource != sim.WeaponSpellItem || e.WeaponSpell != 7 || e.WeaponSpellLevel != 15 {
		t.Fatalf("item source after Rearm = (%d,%d,%d)", e.WeaponSpellSource, e.WeaponSpell, e.WeaponSpellLevel)
	}
}

func sourceName(source sim.WeaponSpellSource) string {
	if source == sim.WeaponSpellInnate {
		return "Innate"
	}
	return "Legacy"
}

func TestEquippingAMetalPieceIsAcceptedForAMagesOwnCharacter(t *testing.T) {
	table := gaTable(t)
	mage := data.Hero{Body: 8, Reaction: 10, Mind: 40, Spirit: 20}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{gaBootsCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, mage, nil, table)

	mw.enqueueEquip(0)
	if len(mw.pending) != 1 {
		t.Fatalf("pending = %v, want exactly one command — the gate must not refuse a metal "+
			"piece on account of who would wear it", mw.pending)
	}
	mw.tick()

	boots := gaBoots(t, table)
	want := mage.Recompute(data.Profile{}, data.Loadout{Mod: data.EquipMod{Defence: boots.Defence, Absorption: boots.Absorption}}).Combat.Defence
	e := gaEntity(t, w, 7)
	if e.Defence != want {
		t.Errorf("Defence = %d, want %d — the equip was accepted but the recompute did not follow", e.Defence, want)
	}
}

// TestUnequippingAnArmourPieceDropsItsDefenceAndAbsorption is 0151 defect
// 4's own required witness on the trap named for this task: an armour
// piece equipped through the ordinary command path, then taken off through
// the SAME path (enqueueUnequip, then a tick), must leave the subject's
// Defence and Absorption exactly where they stood before the piece went
// on — not still carrying it. TestEquippingAnArmourRaisesDefenceAnd-
// AbsorptionWhileLeavingTheWeaponsNumbersUntouched (above) is this test's
// own first half, reused rather than re-derived: boots go on over a sword
// already worn, and this test continues from there by taking the boots
// back off.
//
// TO CONFIRM THIS TEST WITNESSES THE RE-DERIVATION, comment out the
// mw.rearm() call in tick (world.go) or the KindUnequip arm in stepWorld's
// switch (pkg/sim/step.go): sim.Equipped(7) loses the boots' code while
// Defence and Absorption keep the values they raised to, and the two
// assertions below redden — a removed boot that goes on protecting.
func TestUnequippingAnArmourPieceDropsItsDefenceAndAbsorption(t *testing.T) {
	table := gaTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqSwordCode, gaBootsCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, table)

	mw.enqueueEquip(0) // the sword, index 0
	mw.tick()
	mw.enqueueEquip(0) // the boots, now at index 0
	mw.tick()
	worn := gaEntity(t, w, 7)

	boots := gaBoots(t, table)
	if boots.Defence == 0 && boots.Absorption == 0 {
		t.Fatal("setup: the boots contribute nothing, so this test cannot discriminate their removal")
	}
	if slots, _ := w.Equipped(7); slots[11] != gaBootsCode {
		t.Fatalf("setup: Equipped(7)[11] = %#x, want %#x (the boots, slot 12)", slots[11], gaBootsCode)
	}

	mw.enqueueUnequip(11) // worn cell 11 (Slots' own zero-based index for slot 12)
	mw.tick()
	bare := gaEntity(t, w, 7)

	if want := worn.Defence - boots.Defence; bare.Defence != want {
		t.Errorf("Defence = %d, want %d (worn %d − the boots' own %d) — the removed boots still "+
			"protect", bare.Defence, want, worn.Defence, boots.Defence)
	}
	if want := worn.Absorption - boots.Absorption; bare.Absorption != want {
		t.Errorf("Absorption = %d, want %d (worn %d − the boots' own %d) — the removed boots still "+
			"protect", bare.Absorption, want, worn.Absorption, boots.Absorption)
	}
	// The sword's own numbers are untouched: this move named slot 12 alone.
	if bare.DamageBase != worn.DamageBase {
		t.Errorf("DamageBase moved from %d to %d — unequipping the boots must not touch the sword", worn.DamageBase, bare.DamageBase)
	}

	if slots, _ := w.Equipped(7); slots[11] != 0 {
		t.Errorf("Equipped(7)[11] = %#x, want 0 (empty)", slots[11])
	}
	if codes, _ := w.Carried(7); len(codes) != 1 || codes[0] != gaBootsCode {
		t.Errorf("Carried(7) = %#x, want [%#x] — the boots returned to the pack", codes, gaBootsCode)
	}
}
