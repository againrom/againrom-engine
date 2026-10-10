package game

// itemInfoLines, slotInfoLines and packInfoLines: the popup's own text
// (0151, defect 5).
//
// EVERY FIXTURE HERE IS SYNTHETIC (AGENTS.md rule 2) and reuses
// eqDefsTable/eqSword (equip_test.go) and gaTable/gaBootsCode (rearm_test.go)
// rather than building a third real, parsed *databin.Collection triple.

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A resolvable weapon states its name and both decoded characteristic
// lines, read off the SAME data.WeaponFromCode call itemName and rearm
// already make (eqSword, equip_test.go) — this test witnesses that
// itemInfoLines surfaces what that call answers, not a second arithmetic
// for the numbers themselves.
func TestItemInfoLinesStatesAResolvableWeaponsDamageAndCombat(t *testing.T) {
	table := eqDefsTable(t)
	w := eqSword(t, table)

	got := itemInfoLines(data.ItemCode(eqSwordCode), table)
	want := []string{
		"Sword",
		fmt.Sprintf("#Damage: %d-%d", w.DamageBase, w.DamageBase+w.DamageSpread),
		fmt.Sprintf("#To-hit: %d", w.ToHit),
		fmt.Sprintf("#Defence: %d", w.Defence),
	}
	if !slices.Equal(got, want) {
		t.Errorf("itemInfoLines(eqSwordCode) = %v, want %v", got, want)
	}
}

func TestItemFormatterUsesInstalledStatsLabelsAndOriginalLiteralForms(t *testing.T) {
	w := ui.AuthoredWords()
	w.ItemStats[2] = "Installed Body"
	w.ItemStats[43] = "Installed Damage"
	if got := itemEffectInfoLine(sim.ItemEffect{Kind: 2, Operand: 5}, nil, w); got != "#Installed Body +5" {
		t.Fatalf("signed form = %q", got)
	}
	if got := itemEffectInfoLine(sim.ItemEffect{Kind: 2, Operand: ^uint32(4)}, nil, w); got != "#Installed Body -5" {
		t.Fatalf("negative form = %q", got)
	}
	for _, tc := range []struct {
		effect sim.ItemEffect
		want   string
	}{
		{sim.ItemEffect{Kind: 2, Mode: 1, Operand: 5}, "#Installed Body: +5 (duration 0)"},
		{sim.ItemEffect{Kind: 2, Mode: 2, Operand: 5}, "#Installed Body: +5% (continuous 0)"},
		{sim.ItemEffect{Kind: 2, Mode: 4, Operand: 5}, "#Installed Body:   5.0 (charges 0)"},
		{sim.ItemEffect{Kind: 2, Mode: 8, Operand: 5}, "#Installed Body 5 (single use)"},
		{sim.ItemEffect{Kind: 43, Operand: 2 | 3<<8}, "#Installed Damage: 2-5"},
		{sim.ItemEffect{Kind: 41, Operand: 1 | 3<<16}, "#Casts spell 1"},
		{sim.ItemEffect{Kind: 42, Operand: 1}, " of spell 1"},
	} {
		if got := itemEffectInfoLine(tc.effect, nil, w); got != tc.want {
			t.Errorf("effect %+v = %q want %q", tc.effect, got, tc.want)
		}
	}
}

// A weapon-spell tooltip is the live release interval, not the unused
// physical pair in the Weapons row. The fixture deliberately gives the staff
// a physical 0-0 so the owner's exact regression is visible if the resolver
// is removed.
func TestItemInfoLinesUsesTheLiveWeaponSpellInsteadOfStaffPhysicalDamage(t *testing.T) {
	weaponParams := chargenWeaponParams(data.SkillBlade)
	weaponParams[6], weaponParams[7] = 0, 0
	weapons := dbCollection{{}, {name: "Wood Staff", params: weaponParams}}
	spellParams := make([]int32, 19)
	spellParams[1], spellParams[2], spellParams[4], spellParams[6] = 3, 1, 1, 7
	spellParams[16], spellParams[17] = 15, 18
	table := &mapload.Table{
		Shapes: emptyScale{}, Materials: emptyScale{}, Weapons: weapons,
		Spells: dbCollection{{}, {name: "Fire Arrow", params: spellParams}},
	}
	staff, err := data.ResolveWeapon("Wood Staff {castSpell=Fire_Arrow:10}",
		table.Shapes, table.Materials, table.Weapons)
	if err != nil {
		t.Fatalf("resolve staff: %v", err)
	}
	table.Names = data.ItemNames{staff.Code: "Wood Staff"}
	spellID, ok := mapload.SpellIDByToken(table, staff.SpellName)
	if !ok {
		t.Fatal("fixture staff spell did not resolve")
	}
	w, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: 2, Height: 2}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{{ID: 7, MaxMana: 1, WeaponSpell: spellID,
			WeaponSpellLevel: staff.SpellPower}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Equipped: [sim.EquipSlots]uint16{uint16(staff.Code)}}},
		mapload.SpellRules(table))
	if err != nil {
		t.Fatalf("build spelled world: %v", err)
	}
	base, spread, ok := w.WeaponSpellDamage(7)
	if !ok {
		t.Fatal("live world refused the equipped staff spell")
	}

	got := itemInfoLinesWithWeaponDamage(staff.Code, table, func(code data.ItemCode) (weaponDamageInterval, bool) {
		return liveWeaponSpellDamage(w, 7, code)
	})
	want := []string{"Wood Staff", "MAGIC:", "Casts Fire Arrow",
		fmt.Sprintf("Damage %d-%d", base, base+spread), "Range 7"}
	if !slices.Equal(got, want) {
		t.Fatalf("staff tooltip = %v, want %v", got, want)
	}
	if slices.Contains(got, "#Damage: 0-0") {
		t.Fatalf("staff tooltip exposed unused physical damage: %v", got)
	}
}

func TestItemInstanceInfoLinesDescribesAnAttachedStaffCastOnce(t *testing.T) {
	weaponParams := chargenWeaponParams(data.SkillBlade)
	weaponParams[6], weaponParams[7] = 0, 0
	weapons := dbCollection{{}, {name: "Staff", params: weaponParams}}
	spellParams := make([]int32, 19)
	spellParams[1], spellParams[2], spellParams[4], spellParams[6] = 3, 1, 1, 5
	spellParams[16], spellParams[17] = 5, 10
	table := &mapload.Table{
		Shapes: emptyScale{}, Materials: emptyScale{}, Weapons: weapons,
		Spells: dbCollection{{}, {name: "Fire Arrow", params: spellParams}},
	}
	staff, err := data.ResolveWeapon("Staff", table.Shapes, table.Materials, table.Weapons)
	if err != nil {
		t.Fatalf("resolve staff: %v", err)
	}
	table.Names = data.ItemNames{staff.Code: "Staff"}
	item := sim.ItemInstance{Code: uint16(staff.Code), Effects: []sim.ItemEffect{{Kind: 41,
		Operand: uint32(1) | uint32(uint16(0))<<16}}}

	got := itemInstanceInfoLines(item, table)
	want := []string{"Staff", "MAGIC:", "Casts Fire Arrow", "Damage 5-10", "Range 5"}
	if !slices.Equal(got, want) {
		t.Fatalf("attached-cast staff tooltip = %v, want %v", got, want)
	}
	for _, line := range got {
		if line == "Cast Fire Arrow power 0" {
			t.Fatalf("attached-cast staff tooltip duplicated its raw effect line: %v", got)
		}
	}
}

func TestItemInstanceInfoLinesUsesTheInstallWordsForAStaffSpell(t *testing.T) {
	weaponParams := chargenWeaponParams(data.SkillBlade)
	weaponParams[6], weaponParams[7] = 0, 0
	weapons := dbCollection{{}, {name: "Staff", params: weaponParams}}
	spellParams := make([]int32, 19)
	spellParams[1], spellParams[2], spellParams[4], spellParams[6] = 3, 1, 1, 5
	spellParams[16], spellParams[17] = 5, 10
	table := &mapload.Table{
		Shapes: emptyScale{}, Materials: emptyScale{}, Weapons: weapons,
		Spells: dbCollection{{}, {name: "Fire Arrow", params: spellParams}},
	}
	staff, err := data.ResolveWeapon("Staff", table.Shapes, table.Materials, table.Weapons)
	if err != nil {
		t.Fatalf("resolve staff: %v", err)
	}
	table.Names = data.ItemNames{staff.Code: "Staff"}
	item := sim.ItemInstance{Code: uint16(staff.Code), Effects: []sim.ItemEffect{{Kind: 41,
		Operand: uint32(1) | uint32(uint16(0))<<16}}}

	words := ui.AuthoredWords()
	words.ItemCasts, words.ItemDamage, words.ItemRange = "C", "D", "R"
	words.ItemSpellNames[1] = "Spell"
	got := itemInstanceInfoLines(item, table, words)
	want := []string{"Staff", "MAGIC:", "C Spell", "D 5-10", "R 5"}
	if !slices.Equal(got, want) {
		t.Fatalf("staff tooltip = %v, want %v", got, want)
	}
}
func staffTooltipTable(t *testing.T, spellID int, spellName string, power int32,
	maxRange, damageMin, damageMax, duration int32) (*mapload.Table, *data.Weapon) {
	t.Helper()
	weaponParams := chargenWeaponParams(data.SkillBlade)
	weaponParams[6], weaponParams[7] = 0, 0
	spells := make(dbCollection, 21)
	for i := 1; i < len(spells); i++ {
		spells[i] = dbEntry{name: fmt.Sprintf("Spell %d", i), params: make([]int32, 19)}
		spells[i].params[8] = 1
	}
	spells[spellID].name = spellName
	spells[spellID].params[4] = 1
	spells[spellID].params[6] = maxRange
	spells[spellID].params[14] = duration
	spells[spellID].params[16] = damageMin
	spells[spellID].params[17] = damageMax
	table := &mapload.Table{
		Shapes: emptyScale{}, Materials: emptyScale{},
		Weapons: dbCollection{{}, {name: "Staff", params: weaponParams}},
		Spells:  spells,
	}
	weapon, err := data.ResolveWeapon(fmt.Sprintf("Staff {castSpell=%s:%d}",
		strings.ReplaceAll(spellName, " ", "_"), power), table.Shapes, table.Materials, table.Weapons)
	if err != nil {
		t.Fatalf("resolve %s staff: %v", spellName, err)
	}
	table.Names = data.ItemNames{weapon.Code: "Staff"}
	return table, &weapon
}

func TestMagicStaffTooltipsDescribeStoneAndPrismaticAcrossEveryProducer(t *testing.T) {
	for _, tc := range []struct {
		name                           string
		spellID                        int
		power, maxRange                int32
		damageMin, damageMax, duration int32
		want                           []string
	}{
		{name: "Stone Curse", spellID: 20, power: 1, maxRange: 5, duration: 10,
			want: []string{"Staff", "MAGIC:", "Stone Curse 10.19 sec", "Range 5"}},
		{name: "Prismatic Spray", spellID: 14, power: 30, maxRange: 7, damageMin: 5, damageMax: 15,
			want: []string{"Staff", "MAGIC:", "Casts Prismatic Spray", "Damage 10-30", "Rays 3", "Range 7"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			table, weapon := staffTooltipTable(t, tc.spellID, tc.name, tc.power,
				tc.maxRange, tc.damageMin, tc.damageMax, tc.duration)
			spellID, ok := mapload.SpellIDByToken(table, weapon.SpellName)
			if !ok || int(spellID) != tc.spellID {
				t.Fatalf("spell token resolved as %d, %v", spellID, ok)
			}
			world, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: 2, Height: 2}, sim.ModeCanonical,
				sim.Terrain{}, []sim.Entity{{ID: 7, MaxMana: 1, WeaponSpell: spellID,
					WeaponSpellLevel: tc.power}}, nil, sim.Relations{}, nil,
				[]sim.Stock{{ID: 7, Equipped: [sim.EquipSlots]uint16{uint16(weapon.Code)}}},
				mapload.SpellRules(table))
			if err != nil {
				t.Fatalf("build spelled world: %v", err)
			}

			live, liveOK := liveWeaponSpellDamage(world, 7, weapon.Code)
			stored, storedOK := storedWeaponSpellDamage(weapon.Code, weapon, true, table)
			item := sim.ItemInstance{Code: uint16(weapon.Code), Price: 123,
				Effects: []sim.ItemEffect{{Kind: 41,
					Operand: uint32(spellID) | uint32(uint16(tc.power))<<16}}}
			attached, attachedOK := itemWeaponSpellDamage(item, table)
			if !liveOK || !storedOK || !attachedOK || live != stored || stored != attached {
				t.Fatalf("producer projections differ: live=%+v/%v stored=%+v/%v attached=%+v/%v",
					live, liveOK, stored, storedOK, attached, attachedOK)
			}

			for name, resolve := range map[string]weaponDamageResolver{
				"live": func(code data.ItemCode) (weaponDamageInterval, bool) {
					return liveWeaponSpellDamage(world, 7, code)
				},
				"stored": func(code data.ItemCode) (weaponDamageInterval, bool) {
					return storedWeaponSpellDamage(code, weapon, true, table)
				},
				"attached": nil,
			} {
				t.Run(name, func(t *testing.T) {
					got := itemInstanceInfoLinesWithWeaponDamage(item, table, resolve)
					if !slices.Equal(got, tc.want) {
						t.Fatalf("tooltip = %v, want %v", got, tc.want)
					}
					for _, line := range got {
						if strings.HasPrefix(line, "#") || strings.Contains(line, " power ") {
							t.Fatalf("tooltip retained a raw literal/power line: %v", got)
						}
					}
				})
			}
		})
	}
}

func TestStoneCurseStaffDurationUsesInstallLanguage(t *testing.T) {
	table, weapon := staffTooltipTable(t, 20, "Stone Curse", 1, 5, 0, 0, 10)
	russian := LoadInstallWords(installFixture{LanguagePath: []byte("russian 1")})
	russian.Language = "russian"
	w := russian.Words()
	got := itemInfoLinesWithWeaponDamage(weapon.Code, table,
		func(code data.ItemCode) (weaponDamageInterval, bool) {
			return storedWeaponSpellDamage(code, weapon, true, table)
		}, w)
	if len(got) < 3 || got[2] != "Stone Curse 10.19 \xe1\xa5\xaa" {
		t.Fatalf("RU Stone Curse duration = %q", got)
	}
}

// An armour code cannot resolve through data.WeaponFromCode at all (field B
// names an equipment slot, not the weapon class), so its NAME comes from
// the stored table (T4, world.go's itemName) and its characteristics line
// from data.ArmorFromCode — the two halves of defect 5 and defect 6, read
// together here at the popup layer.
func TestItemInfoLinesStatesAResolvableArmoursDefenceAndAbsorption(t *testing.T) {
	table := gaTable(t)
	table.Names = data.ItemNames{data.ItemCode(gaBootsCode): "Boots"}
	a, err := data.ArmorFromCode(data.ItemCode(gaBootsCode), table.Shapes, table.Materials, table.Armors)
	if err != nil {
		t.Fatalf("setup: data.ArmorFromCode(gaBootsCode): %v", err)
	}

	got := itemInfoLines(data.ItemCode(gaBootsCode), table)
	want := wornCardLines("Boots", a.Defence, a.Absorption)
	if !slices.Equal(got, want) {
		t.Errorf("itemInfoLines(gaBootsCode) = %v, want %v", got, want)
	}
}

func TestItemInfoLinesStatesShieldBaseProtectionAndInstanceMagic(t *testing.T) {
	table := eqDefsTable(t)
	table.Names = data.ItemNames{data.ItemCode(eqShieldCode): "Ward"}
	shield, err := data.ShieldFromCode(data.ItemCode(eqShieldCode), table.Shapes, table.Materials, table.Shields)
	if err != nil {
		t.Fatalf("ShieldFromCode: %v", err)
	}
	item := sim.ItemInstance{Code: eqShieldCode, Price: 321,
		Effects: []sim.ItemEffect{{Kind: 15, Operand: 4}}}
	got := itemInstanceInfoLines(item, table)
	want := append(wornCardLines("Ward", shield.Defence, shield.Absorption), "MAGIC:", "#Defence +4")
	if !slices.Equal(got, want) {
		t.Fatalf("shield tooltip = %v, want %v", got, want)
	}
}

// The information lines state no price (ITEM-PRICETAG-144). Whatever price is
// stored, -1 (the installed access item's own value), 0, a positive price or
// one below -1, the lines are those of the same item at price 0. Each branch
// that once appended a price line is covered: a name-only carried item, an
// enchanted shield, a staff whose cast is described and a spell book. A pack
// stack keeps its stored price.
func TestItemInstanceInfoLinesComposeNoPriceLine(t *testing.T) {
	table := eqDefsTable(t)
	table.Names = data.ItemNames{data.ItemCode(eqShieldCode): "Ward"}
	staffTable, staff := staffTooltipTable(t, 20, "Stone Curse", 1, 5, 0, 0, 10)
	spellID, ok := mapload.SpellIDByToken(staffTable, staff.SpellName)
	if !ok {
		t.Fatal("fixture staff spell did not resolve")
	}
	subjects := []struct {
		name  string
		item  sim.ItemInstance
		table *mapload.Table
	}{
		{"name only", sim.ItemInstance{Code: eqNoSlotCode}, table},
		{"enchanted shield", sim.ItemInstance{Code: eqShieldCode, Kind: 1,
			Effects: []sim.ItemEffect{{Kind: 15, Operand: 4}}}, table},
		{"described staff", sim.ItemInstance{Code: uint16(staff.Code), Kind: 2,
			Effects: []sim.ItemEffect{{Kind: 41, Operand: uint32(spellID) | 1<<16}}}, staffTable},
		{"spell book", sim.ItemInstance{Code: eqNoSlotCode, Kind: 5,
			Effects: []sim.ItemEffect{{Kind: 42, Operand: 1}}}, table},
	}
	for _, subject := range subjects {
		unpriced := itemInstanceInfoLines(subject.item, subject.table)
		if len(unpriced) == 0 {
			t.Fatalf("%s: no lines at price 0", subject.name)
		}
		for _, price := range []int32{math.MinInt32, -2, -1, 0, 1, 321, 9_999_999} {
			item := subject.item.Clone()
			item.Price = price
			got := itemInstanceInfoLines(item, subject.table)
			if !slices.Equal(got, unpriced) {
				t.Errorf("%s priced %d: lines = %q, want the unpriced %q", subject.name, price, got, unpriced)
			}
			for _, line := range got {
				if strings.Contains(line, "Value") {
					t.Errorf("%s priced %d: line %q states the price", subject.name, price, line)
				}
			}
		}
	}
	name := data.ItemCode(eqNoSlotCode).Name()
	if got, want := itemInstanceInfoLines(sim.ItemInstance{Code: eqNoSlotCode, Price: 321}, table),
		[]string{name}; !slices.Equal(got, want) {
		t.Errorf("priced name-only item: lines = %q, want %q", got, want)
	}
	for _, price := range []int32{-1, 321} {
		stacks := []sim.ItemStack{{Code: eqNoSlotCode, Price: price, Count: 1}}
		got := packInfoLines(stacks, table)
		if len(got) != 1 || !slices.Equal(got[0], []string{name}) {
			t.Errorf("pack stack priced %d: lines = %q, want [%q]", price, got, name)
		}
		if stacks[0].Price != price {
			t.Errorf("pack stack price = %d after the information lines, want %d", stacks[0].Price, price)
		}
	}
}

// THE SAME ARMOUR CODE WITH NO STORED NAME states its characteristics line
// alone below its digit fallback — an armour never resolves a name through
// data.ArmorFromCode (0136's own "the result carries no Name"), so a
// popup's name line is only ever as good as the stored table itemName
// reads first.
func TestItemInfoLinesStatesAnArmoursCharacteristicsEvenWithNoStoredName(t *testing.T) {
	table := gaTable(t)
	code := data.ItemCode(gaBootsCode)
	a, err := data.ArmorFromCode(code, table.Shapes, table.Materials, table.Armors)
	if err != nil {
		t.Fatalf("setup: data.ArmorFromCode(gaBootsCode): %v", err)
	}

	got := itemInfoLines(code, table)
	want := wornCardLines(code.Name(), a.Defence, a.Absorption)
	if !slices.Equal(got, want) {
		t.Errorf("itemInfoLines(gaBootsCode, no Names) = %v, want %v", got, want)
	}
}

// A code that resolves through neither table states its name alone — the
// same fallback itemName already gives it, with no characteristics line
// appended over nothing to append.
func TestItemInfoLinesStatesTheNameAloneWhenNeitherTableResolves(t *testing.T) {
	table := eqDefsTable(t)
	code := data.ItemCode(eqNoSlotCode)
	got := itemInfoLines(code, table)
	want := []string{code.Name()}
	if !slices.Equal(got, want) {
		t.Errorf("itemInfoLines(eqNoSlotCode) = %v, want %v", got, want)
	}
}

func TestItemInfoLinesWithNoTableStatesTheNameAlone(t *testing.T) {
	code := data.ItemCode(eqSwordCode)
	got := itemInfoLines(code, nil)
	want := []string{code.Name()}
	if !slices.Equal(got, want) {
		t.Errorf("itemInfoLines(nil table) = %v, want %v", got, want)
	}
}

// slotInfoLines walks every one of the twelve slots and fills only the
// occupied one — one entry per code actually there, over a fixed-width
// array.
func TestSlotInfoLinesFillsOnlyOccupiedSlotsInSlotOrder(t *testing.T) {
	table := eqDefsTable(t)
	var eq data.Equipment
	eq.SetCode(1, data.ItemCode(eqSwordCode))

	got := slotInfoLines(eq, table)
	if len(got[0]) == 0 || got[0][0] != "Sword" {
		t.Errorf("slotInfoLines[0] = %v, want a line starting %q", got[0], "Sword")
	}
	for n := 2; n <= data.EquipSlots; n++ {
		if len(got[n-1]) != 0 {
			t.Errorf("slotInfoLines[%d] = %v, want nil — no code was ever set there", n-1, got[n-1])
		}
	}
}

// packInfoLines follows stacks' own order and is parallel to it, index for
// index — Pack's and PackCount's own shape (pkg/ui's InventorySubject doc),
// restated for the popup's own text.
func TestPackInfoLinesFollowsTheStacksOwnOrder(t *testing.T) {
	table := eqDefsTable(t)
	stacks := []sim.ItemStack{{Code: eqSwordCode, Count: 1}, {Code: eqNoSlotCode, Count: 3}}

	got := packInfoLines(stacks, table)
	if len(got) != 2 {
		t.Fatalf("packInfoLines returned %d entr(y/ies), want 2", len(got))
	}
	if got[0][0] != "Sword" {
		t.Errorf("packInfoLines[0][0] = %q, want %q", got[0][0], "Sword")
	}
	if want := data.ItemCode(eqNoSlotCode).Name(); got[1][0] != want {
		t.Errorf("packInfoLines[1][0] = %q, want %q", got[1][0], want)
	}
}

func TestPackInfoLinesWithNoStacksAnswersNil(t *testing.T) {
	if got := packInfoLines(nil, eqDefsTable(t)); got != nil {
		t.Errorf("packInfoLines(nil, table) = %v, want nil", got)
	}
}

// A ranged weapon's description ends with its range column; a melee weapon of
// the same table states none. The caption is the installed range caption when
// the words carry one.
func TestItemInfoLinesStatesARangedWeaponsRange(t *testing.T) {
	table := eqDefsTable(t)
	bow, err := data.WeaponFromCode(data.ItemCode(eqBowCode), table.Shapes, table.Materials, table.Weapons)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	got := itemInfoLines(data.ItemCode(eqBowCode), table)
	want := fmt.Sprintf("Range: %d", bow.Range)
	if bow.Range != 6 || got[len(got)-1] != want {
		t.Errorf("bow lines = %v, want last %q", got, want)
	}
	words := ui.AuthoredWords()
	words.Hover[123] = "Installed Range"
	got = itemInstanceInfoLines(sim.ItemInstance{Code: eqBowCode}, table, words)
	if want = "Installed Range: 6"; got[len(got)-1] != want {
		t.Errorf("installed caption lines = %v, want last %q", got, want)
	}
	for _, line := range itemInfoLines(data.ItemCode(eqSwordCode), table) {
		if strings.Contains(line, "Range") {
			t.Errorf("melee weapon states a range line %q", line)
		}
	}
}

// A shooting-skill weapon takes the melee equip arm yet is ranged: its range
// column is stated, and a melee weapon with the same column is not.
func TestItemInfoLinesStatesAShootingSkillWeaponsRange(t *testing.T) {
	rowParams := func(kind int32) []int32 {
		return []int32{-1, -1, -1, -1, -1, kind, 10, 20, 5, 3, -1, 6, 6, 4, -1, eqSuitAny}
	}
	table := &mapload.Table{
		Shapes: emptyScale{}, Materials: emptyScale{},
		Weapons: dbCollection{{}, {name: "Crossbow", params: rowParams(data.SkillShoot)}, {name: "Spear", params: rowParams(data.SkillPike)}},
	}
	got := itemInfoLines(data.ComposeItemCode(0, 1, 0, 1), table)
	if want := "Range: 6"; got[len(got)-1] != want {
		t.Errorf("crossbow lines = %v, want last %q", got, want)
	}
	for _, line := range itemInfoLines(data.ComposeItemCode(0, 1, 0, 2), table) {
		if strings.Contains(line, "Range") {
			t.Errorf("melee weapon states a range line %q", line)
		}
	}
}

// wornCardLines is the name and base protection lines of an Armor or Shield as
// the item formatter prints them: defence always, absorption only above zero.
func wornCardLines(name string, defence, absorption int32) []string {
	lines := []string{name, fmt.Sprintf("#Defence %+d", defence)}
	if absorption > 0 {
		lines = append(lines, fmt.Sprintf("#Absorption %+d", absorption))
	}
	return lines
}
