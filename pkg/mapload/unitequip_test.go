package mapload_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The two class keys this file places a unit on. Distinct from every other
// _test.go file's own keys, because a table here carries several rows and
// each row needs its own.
const (
	equipMeleeKey  = 0x50
	equipRangedKey = 0x51
)

// equipUnitRow is a Units row at key/face carrying the row's own eight combat
// columns and the given trailing equipment strings. slotDamageArm is left
// empty, which the loader reads as arm 0 — the pair this tree models, no
// mark — exactly as combat_test.go's own armSet-false case does.
func equipUnitRow(key, face, dmgMin, dmgMax, toHit, defence, charge, relax int32, equipment []string) defEntry {
	return defEntry{
		name: "equip", strings: equipment,
		params: defRow(map[int]int32{
			slotUnitType: key, slotUnitFace: face, slotHealthMax: 30,
			slotDamageMin: dmgMin, slotDamageMax: dmgMax,
			slotToHit: toHit, slotDefence: defence,
			slotCharge: charge, slotRelax: relax,
		}),
	}
}

// equipWeapons is the weapon collection AC-3's two cases resolve against: a
// melee weapon and a ranged one (attack type 0xb, past meleeAttackTypes),
// each stating its own nonzero damage, to-hit, defence, cadence and range —
// so a fold that reads the wrong arm, or that assigns instead of adding,
// shows on every one of those fields rather than on one alone.
//
// A THIRD ROW, "MeleeNoRange", is a melee weapon whose own range cell is left
// empty, for the same reason reach_test.go's Dagger is: it proves the
// constructor's floor of 1 stands when a resolving weapon simply states none.
func equipWeapons() defCollection {
	melee := weaponRow(data.SkillBlade, 5, 9, 7, 2, 15, 9) // base 5, spread 4, toHit 7, defence 2
	melee[slotWeaponRange] = 6
	ranged := weaponRow(0xb, 20, 30, 50, 25, 12, 5) // third-component pair; own to-hit/defence are ignored
	ranged[slotWeaponRange] = 8
	meleeNoRange := weaponRow(data.SkillBlade, 1, 1, 0, 0, -1, -1)
	return defCollection{
		{},
		{name: "Melee", params: melee},
		{name: "Ranged", params: ranged},
		{name: "MeleeNoRange", params: meleeNoRange},
	}
}

// equipTable wraps one units row as the only entry of a table carrying the
// given item collections.
func equipTable(row defEntry, weapons data.Collection, shapes, materials data.ScaleTable) *mapload.Table {
	return &mapload.Table{
		Units:  defCollection{{}, row},
		Humans: defCollection{},
		Shapes: shapes, Materials: materials, Weapons: weapons,
	}
}

// TestACreatureNamingAMeleeWeaponArrivesWithTheSumOfBoth is AC-3's melee
// case and R-3: the row's own damage base, damage spread, to-hit and
// defence are chosen so that the row's own value, the weapon's own value and
// their sum are three distinct numbers on every one of the four — so a fold
// that assigns instead of adding, on any one field, fails this test rather
// than passing it by coincidence. The cadence pair and the reach are the
// weapon's alone, never the row's, on the melee arm exactly as on the
// ranged one.
func TestACreatureNamingAMeleeWeaponArrivesWithTheSumOfBoth(t *testing.T) {
	t.Parallel()

	// row: base 3, spread 6, toHit 10, defence 4, cadence 99/77 (must not
	// survive the fold). weapon (equipWeapons' "Melee"): base 5, spread 4,
	// toHit 7, defence 2, cadence 15/9, range 6.
	row := equipUnitRow(equipMeleeKey, 1, 3, 9, 10, 4, 99, 77, []string{"Melee"})
	for i, v := range [5]int32{11, 22, 33, 44, 255} {
		row.params[24+i] = v
	}
	row.params[34], row.params[35] = 123, 45
	tbl := equipTable(row, equipWeapons(), identityScale(), identityScale())

	e := srcLoad(t, srcMap(equipMeleeKey), tbl)

	for _, c := range []struct {
		field     string
		got, want int32
	}{
		{"DamageBase", e.DamageBase, 3 + 5},
		{"DamageSpread", e.DamageSpread, 6 + 4},
		{"ToHit", e.ToHit, 10 + 7},
		{"Defence", e.Defence, 4 + 2},
		{"AttackCharge", e.AttackCharge, 15},
		{"AttackRelax", e.AttackRelax, 9},
		{"Reach", int32(e.Reach), 6},
	} {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.field, c.got, c.want)
		}
	}
	if e.XPSlot != uint8(data.SkillBlade) {
		t.Errorf("XPSlot = %d, want the resolved melee kind %d", e.XPSlot, data.SkillBlade)
	}
	if e.Resistance != ([5]uint8{11, 22, 33, 44, 255}) {
		t.Errorf("Resistance = %v, want the Units row's five bytes", e.Resistance)
	}
	if e.Withdraw != 123 || e.Wimpy != 45 {
		t.Errorf("Withdraw/Wimpy = %d/%d, want the Units row's 123/45", e.Withdraw, e.Wimpy)
	}
}

// TestACreatureNamingARangedWeaponUsesItsGeneralAndFireComponent is the
// placed-Units route. The authored ToHit is also the loader's General source,
// so it is added once more; attack type 11 contributes the weapon byte pair
// as Fire while leaving the physical pair and Defence untouched.
func TestACreatureNamingARangedWeaponUsesItsGeneralAndFireComponent(t *testing.T) {
	t.Parallel()

	row := equipUnitRow(equipRangedKey, 1, 3, 9, 10, 4, 99, 77, []string{"Ranged"})
	for i, v := range [5]int32{1, 2, 3, 4, 5} {
		row.params[24+i] = v
	}
	tbl := equipTable(row, equipWeapons(), identityScale(), identityScale())

	e := srcLoad(t, srcMap(equipRangedKey), tbl)

	for _, c := range []struct {
		field     string
		got, want int32
	}{
		{"DamageBase", e.DamageBase, 3},
		{"DamageSpread", e.DamageSpread, 6},
		{"ToHit", e.ToHit, 10 + 10},
		{"Defence", e.Defence, 4},
		{"AttackCharge", e.AttackCharge, 12},
		{"AttackRelax", e.AttackRelax, 5},
		{"Reach", int32(e.Reach), 8},
	} {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.field, c.got, c.want)
		}
	}
	if e.XPSlot != 0 {
		t.Errorf("XPSlot = %d, want 0 for the ranged arm", e.XPSlot)
	}
	if want := (sim.SecondaryDamage{Base: 20, Spread: 10, Selector: 0}); e.SecondaryDamage != want {
		t.Errorf("SecondaryDamage = %+v, want Flame Thrower Fire component %+v", e.SecondaryDamage, want)
	}
	if e.Resistance != ([5]uint8{1, 2, 3, 4, 5}) {
		t.Errorf("Resistance = %v, want the Units row's five bytes", e.Resistance)
	}
}

// TestRangedGeneralSnapshotPrecedesTheHardPlacementBonus pins the two-stage
// order. The Units stream first copies authored ToHit 10 into General; ranged
// equip therefore reaches 20. Hard difficulty then adds one +50 to live ToHit
// only, so the finished value is 70 rather than the 120 produced by copying
// the already-adjusted value into General.
func TestRangedGeneralSnapshotPrecedesTheHardPlacementBonus(t *testing.T) {
	row := equipUnitRow(equipRangedKey, 1, 3, 9, 10, 4, 99, 77, []string{"Ranged"})
	tbl := equipTable(row, equipWeapons(), identityScale(), identityScale())
	load := func(diff mapload.Difficulty) sim.Entity {
		t.Helper()
		w, err := mapload.FromALMWith(srcMap(equipRangedKey), tbl, diff)
		if err != nil {
			t.Fatalf("FromALMWith(%d): %v", diff, err)
		}
		return w.Entities()[0]
	}

	normal := load(mapload.DifficultyNormal)
	hard := load(mapload.DifficultyHard)
	if normal.ToHit != 20 || hard.ToHit != 70 || hard.ToHit-normal.ToHit != 50 {
		t.Errorf("normal/hard ToHit = %d/%d, want 20/70 and one +50 placement delta", normal.ToHit, hard.ToHit)
	}
	if hard.SecondaryDamage != normal.SecondaryDamage {
		t.Errorf("Hard moved ranged third component from %+v to %+v", normal.SecondaryDamage, hard.SecondaryDamage)
	}
}

// TestALongReachMeleeShootingWeaponKeepsKindFive distinguishes reach from the
// ranged arm. AttackType 5 is a supported melee kind even when its range is
// eight cells, so its physical blows consult Shooting resistance.
func TestALongReachMeleeShootingWeaponKeepsKindFive(t *testing.T) {
	row := equipUnitRow(equipMeleeKey, 1, 3, 9, 10, 4, 99, 77, []string{"LongMelee"})
	weapon := weaponRow(data.SkillShoot, 5, 9, 7, 2, 15, 9)
	weapon[slotWeaponRange] = 8
	tbl := equipTable(row, defCollection{{}, {name: "LongMelee", params: weapon}}, identityScale(), identityScale())

	e := srcLoad(t, srcMap(equipMeleeKey), tbl)
	if e.Reach != 8 || e.XPSlot != uint8(data.SkillShoot) {
		t.Errorf("long melee weapon produced Reach/XPSlot %d/%d, want 8/%d", e.Reach, e.XPSlot, data.SkillShoot)
	}
}

func TestAPlacedCreaturesWeaponSpellReachesHisEntity(t *testing.T) {
	t.Parallel()

	catapult := weaponRow(data.SkillBlade, 5, 9, 7, 2, 15, 9)
	for len(catapult) <= iwCarrySlot {
		catapult = append(catapult, -1)
	}
	catapult[iwCarrySlot] = 0
	weapons := defCollection{
		{},
		{name: "Catapult", params: catapult},
	}
	spells := defCollection{
		{}, // 0: reserved
		{name: "Fire Ball", params: spellRow(5, 1, 1, 10, 6, 12, 0)},
	}

	row := equipUnitRow(equipMeleeKey, 1, 3, 9, 10, 4, 99, 77, []string{"Catapult{castSpell=Fire_Ball:20}"})
	tbl := equipTable(row, weapons, identityScale(), identityScale())
	tbl.Spells = spells

	e := srcLoad(t, srcMap(equipMeleeKey), tbl)
	if e.WeaponSpell != 1 {
		t.Errorf("WeaponSpell = %d, want 1 (the resolved Fire Ball id)", e.WeaponSpell)
	}
	if e.WeaponSpellLevel != 20 {
		t.Errorf("WeaponSpellLevel = %d, want 20 (the attachment's own level)", e.WeaponSpellLevel)
	}
	if e.WeaponSpellSource != sim.WeaponSpellItem {
		t.Errorf("WeaponSpellSource = %d, want Item for the held cast weapon", e.WeaponSpellSource)
	}
	world, err := mapload.FromALMWith(srcMap(equipMeleeKey), tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	if worn, _ := world.Equipped(0); worn[0] == 0 {
		t.Error("non-carriable cast weapon lost its live held slot")
	}

	// The same row's weapon, named without the attachment, carries neither.
	bareRow := equipUnitRow(equipRangedKey, 1, 3, 9, 10, 4, 99, 77, []string{"Catapult"})
	bareTbl := equipTable(bareRow, weapons, identityScale(), identityScale())
	bareTbl.Spells = spells
	if e := srcLoad(t, srcMap(equipRangedKey), bareTbl); e.WeaponSpell != 0 || e.WeaponSpellLevel != 0 || e.WeaponSpellSource != sim.WeaponSpellNone {
		t.Errorf("weapon spell = (%d, %d) for the same weapon named bare, want (0, 0)",
			e.WeaponSpell, e.WeaponSpellLevel)
	}
}

// TestNoShippedReachMoves is AC-4: over a table shaped like the shipped
// one — several rows naming a melee weapon, a ranged one, no equipment at
// all, and a name that resolves to no row — every reach is exactly what
// unitReach alone would have produced. The values below are worked out by
// hand from equipWeapons' own range column and defaultRange (1), never by
// calling anything this file is testing: a melee or ranged name that
// resolves answers its weapon's own range; a name that does not, or none at
// all, answers the constructor's floor.
func TestNoShippedReachMoves(t *testing.T) {
	t.Parallel()

	weapons := equipWeapons()
	for i, tc := range []struct {
		name      string
		equipment []string
		want      int32
	}{
		{"a melee weapon that states a range", []string{"Melee"}, 6},
		{"a ranged weapon", []string{"Ranged"}, 8},
		{"a melee weapon whose own range cell is empty", []string{"MeleeNoRange"}, 1},
		{"no equipment string at all", nil, 1},
		{"a name that resolves to no row", []string{"Nonexistent"}, 1},
		{"armour first, then a weapon that resolves", []string{"", "Ranged"}, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key := int32(0x60 + i) // one class key per row, so one table holds them all
			row := equipUnitRow(key, 1, 3, 9, 10, 4, 8, 4, tc.equipment)
			tbl := equipTable(row, weapons, identityScale(), identityScale())

			got := srcLoad(t, srcMap(int16(key)), tbl).Reach
			if int32(got) != tc.want {
				t.Errorf("reach = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestACreatureWithNoItemCollectionsKeepsItsOwnNumbers(t *testing.T) {
	t.Parallel()

	row := equipUnitRow(0x70, 1, 3, 9, 10, 4, 8, 4, []string{"Melee"})
	tbl := &mapload.Table{
		Units: defCollection{{}, row}, Humans: defCollection{},
		// No Shapes, no Materials, no Weapons.
	}

	e := srcLoad(t, srcMap(0x70), tbl)
	for _, c := range []struct {
		field     string
		got, want int32
	}{
		{"DamageBase", e.DamageBase, 3},
		{"DamageSpread", e.DamageSpread, 6},
		{"ToHit", e.ToHit, 10},
		{"Defence", e.Defence, 4},
		{"AttackCharge", e.AttackCharge, 8},
		{"AttackRelax", e.AttackRelax, 4},
		{"Reach", int32(e.Reach), 1},
	} {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.field, c.got, c.want)
		}
	}
}

// TestACreatureNamingNothingKeepsItsOwnRowUntouched is the fold's identity
// case: a row that names no equipment at all folds nothing, so its own
// damage, to-hit, defence and cadence stand exactly as its own columns state
// them and reach stays at the constructor's floor.
func TestACreatureNamingNothingKeepsItsOwnRowUntouched(t *testing.T) {
	t.Parallel()

	row := equipUnitRow(0x71, 1, 3, 9, 10, 4, 8, 4, nil)
	tbl := equipTable(row, equipWeapons(), identityScale(), identityScale())

	e := srcLoad(t, srcMap(0x71), tbl)
	for _, c := range []struct {
		field     string
		got, want int32
	}{
		{"DamageBase", e.DamageBase, 3},
		{"DamageSpread", e.DamageSpread, 6},
		{"ToHit", e.ToHit, 10},
		{"Defence", e.Defence, 4},
		{"AttackCharge", e.AttackCharge, 8},
		{"AttackRelax", e.AttackRelax, 4},
		{"Reach", int32(e.Reach), 1},
	} {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.field, c.got, c.want)
		}
	}
}
