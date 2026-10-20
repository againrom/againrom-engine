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

func TestScrollItemInfoUsesItsFixedSpellPowerAndRelevantFields(t *testing.T) {
	spells := make(dbCollection, 6)
	for id := 1; id < len(spells); id++ {
		spells[id].params = make([]int32, 22)
		spells[id].params[1], spells[id].params[2], spells[id].params[6], spells[id].params[8] = 19, 1, 7, 1
	}
	spells[1].name, spells[5].name = "Arrow", "Protection"
	spells[1].params[16], spells[1].params[17] = 4, 8
	spells[5].params[14] = 1
	table := &mapload.Table{Spells: spells}
	words := testSpellPopupWords()
	words.Hover[183] = "Resistance"
	for _, tc := range []struct {
		spell, power    uint32
		present, absent string
	}{
		{5, 60, "Resistance: +30", "Damage:"},
		{1, 60, "Damage: 12-24", "Resistance:"},
		{5, 6, "Resistance: +3", "Damage:"},
	} {
		item := sim.ItemInstance{Code: 0x0e0e, Kind: 4, Price: 99,
			Effects: []sim.ItemEffect{{Kind: 41, Operand: tc.spell | tc.power<<16}}}
		got := itemInstanceInfoLines(item, table, words)
		if !slices.Contains(got, tc.present) || !slices.Contains(got, fmt.Sprintf("Range: %d", 7+tc.power/30)) {
			t.Fatalf("spell%d power%d: %q", tc.spell, tc.power, got)
		}
		for _, line := range got {
			if strings.HasPrefix(line, tc.absent) || strings.HasPrefix(line, "Mana cost:") {
				t.Fatalf("irrelevant scroll line %q in %q", line, got)
			}
		}
		if tc.spell == 5 && !strings.Contains(strings.Join(got, "\n"), "Duration:") {
			t.Fatalf("protection scroll lacks duration: %q", got)
		}
		stacked := packInfoLinesWithWeaponDamage([]sim.ItemStack{sim.StackItem(item, 3)}, table, nil, words)
		if len(stacked) != 1 || !slices.Equal(stacked[0], got) {
			t.Fatalf("quantity changed scroll characteristics: %q/%q", got, stacked)
		}
	}
}

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
		fmt.Sprintf("#To-hit %+d", w.ToHit),
		fmt.Sprintf("#Defence %+d", w.Defence),
		fmt.Sprintf("#Item lore %d", w.Range),
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
// staffTooltipTable is a Weapons table of one Staff row, usable by a mage only
// (sutableFor 2) with range column 5, and a Spells table whose row spellID
// carries maxRange, duration and the damage pair; the returned weapon carries
// a castSpell Effect at power.
func staffTooltipTable(t *testing.T, spellID int, spellName string, power int32,
	maxRange, damageMin, damageMax, duration int32) (*mapload.Table, *data.Weapon) {
	t.Helper()
	return spellWeaponTable(t, "Staff", 5, 2, spellID, spellName, power, maxRange, damageMin, damageMax, duration)
}

func spellWeaponTable(t *testing.T, weaponName string, weaponRange, suitable int32, spellID int, spellName string,
	power, maxRange, damageMin, damageMax, duration int32) (*mapload.Table, *data.Weapon) {
	t.Helper()
	weaponParams := chargenWeaponParams(data.SkillBlade)
	weaponParams[6], weaponParams[7] = 0, 0
	weaponParams[11], weaponParams[15] = weaponRange, suitable
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
		Weapons: dbCollection{{}, {name: weaponName, params: weaponParams}},
		Spells:  spells,
	}
	weapon, err := data.ResolveWeapon(fmt.Sprintf("%s {castSpell=%s:%d}", weaponName,
		strings.ReplaceAll(spellName, " ", "_"), power), table.Shapes, table.Materials, table.Weapons)
	if err != nil {
		t.Fatalf("resolve %s %s: %v", spellName, weaponName, err)
	}
	table.Names = data.ItemNames{weapon.Code: weaponName}
	return table, &weapon
}

// spellWeaponWords are words whose main.txt captions name themselves, so a
// line shows which caption the composition took.
func spellWeaponWords() ui.Words {
	words := ui.AuthoredWords()
	for i, label := range map[int]string{189: "Magic:", 92: "casts", 118: "Damage", 123: "Range",
		124: "Duration", 186: "Rays"} {
		words.Hover[i] = label
	}
	return words
}

// spellWeaponProducers composes one castSpell Weapon's text on every route
// that describes it: worn with its Spell, worn without it, a shop shelf, a
// pack cell, and the code-only worn and pack routes that answer the cast from
// the live World and from the member's stored weapon.
func spellWeaponProducers(t *testing.T, table *mapload.Table, weapon *data.Weapon, words ui.Words) map[string][]string {
	t.Helper()
	spellID, ok := mapload.SpellIDByToken(table, weapon.SpellName)
	if !ok {
		t.Fatal("fixture spell did not resolve")
	}
	item := sim.ItemInstance{Code: uint16(weapon.Code), Kind: 2, Price: 50, Effects: []sim.ItemEffect{
		{Kind: 41, Operand: uint32(spellID) | uint32(uint16(int16(weapon.SpellPower)))<<16}}}
	built := item.Clone()
	built.SourceEquipment = sim.SourceEquipment{Class: sim.SourceWeapon,
		Spell: sim.SourceItemSpell{Present: true, ID: uint8(spellID), Range: 9, ManaCost: 7}}
	world, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: 2, Height: 2}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{{ID: 7, MaxMana: 1, WeaponSpell: spellID, WeaponSpellLevel: weapon.SpellPower}},
		nil, sim.Relations{}, nil, []sim.Stock{{ID: 7, Equipped: [sim.EquipSlots]uint16{uint16(weapon.Code)}}},
		mapload.SpellRules(table))
	if err != nil {
		t.Fatal(err)
	}
	live := func(code data.ItemCode) (uint16, int32, bool) { return liveWeaponCast(world, 7, code) }
	stored := func(code data.ItemCode) (uint16, int32, bool) { return storedWeaponCast(code, weapon, table) }
	var worn, wornGap [sim.EquipSlots]sim.ItemInstance
	worn[0], wornGap[0] = built, item
	return map[string][]string{
		"worn with its Spell":    slotItemInfoLinesWithWeaponDamage(worn, table, live, words)[0],
		"worn without its Spell": slotItemInfoLinesWithWeaponDamage(wornGap, table, live, words)[0],
		"shop shelf":             itemInstanceInfoLines(item, table, words),
		"pack":                   packInfoLinesWithWeaponDamage([]sim.ItemStack{sim.StackItem(item, 1)}, table, nil, words)[0],
		"worn code, live":        itemInfoLinesWithWeaponDamage(weapon.Code, table, live, words),
		"pack code, stored":      itemInfoLinesWithWeaponDamage(weapon.Code, table, stored, words),
	}
}

// TestSpellWeaponTextFollowsTheWeaponStream: a castSpell Weapon's text is the
// stream ITEM-211 composes, the same on every route, worn or not, with or
// without its Spell. The range line is the Weapon's range column (Staff 5,
// Shaman Staff 4), not the spell's Max Range 7; a Weapon usable by fighters
// (sutableFor bit 0) first states its own damage and its range above 1.
// Damage is the Spells row filled at the Effect's power (TEXT-096): at power
// 30 the factor is 2, so 5 and 10 give 10 and 20.
func TestSpellWeaponTextFollowsTheWeaponStream(t *testing.T) {
	words := spellWeaponWords()
	for _, tc := range []struct {
		name               string
		weaponRange, suits int32
		want               []string
	}{
		{"Staff", 5, 2, []string{"Staff", "Magic:", "#casts Fire Arrow", "Damage: 10-20", "Range: 5"}},
		{"Shaman Staff", 4, 2, []string{"Shaman Staff", "Magic:", "#casts Fire Arrow", "Damage: 10-20", "Range: 4"}},
		{"Crossbow", 6, 1, []string{"Crossbow", "#" + words.ItemStats[43] + ": 0-0", "#" + words.ItemStats[38] + " 6",
			"Magic:", "#casts Fire Arrow", "Damage: 10-20", "Range: 6"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			table, weapon := spellWeaponTable(t, tc.name, tc.weaponRange, tc.suits, 1, "Fire Arrow", 30, 7, 5, 10, 0)
			for route, got := range spellWeaponProducers(t, table, weapon, words) {
				if !slices.Equal(got, tc.want) {
					t.Errorf("%s: %q, want %q", route, got, tc.want)
				}
				for _, line := range got {
					if strings.HasSuffix(line, ": 7") || strings.HasSuffix(line, ": 8") {
						t.Errorf("%s states the spell's Max Range: %q", route, got)
					}
				}
			}
		})
	}
}

// TestSpellWeaponTextStatesDurationAndCaption: Stone Curse states no damage
// (its spread is zero) and the duration line at %5.1f; Prismatic Spray states
// its ray caption, min(power/20+2, 7) (TEXT-129, TEXT-133).
func TestSpellWeaponTextStatesDurationAndCaption(t *testing.T) {
	words := spellWeaponWords()
	for _, tc := range []struct {
		name                                  string
		spellID                               int
		power, maxRange, dmin, dmax, duration int32
		want                                  []string
	}{
		{"Stone Curse", 20, 0, 5, 0, 0, 10, []string{"Staff", "Magic:", "#casts Stone Curse", "Range: 5", "Duration:  10.0"}},
		{"Prismatic Spray", 14, 30, 7, 5, 15, 0, []string{"Staff", "Magic:", "#casts Prismatic Spray", "Damage: 10-30", "Range: 5", "Rays: 3"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			table, weapon := staffTooltipTable(t, tc.spellID, tc.name, tc.power, tc.maxRange, tc.dmin, tc.dmax, tc.duration)
			for route, got := range spellWeaponProducers(t, table, weapon, words) {
				if !slices.Equal(got, tc.want) {
					t.Errorf("%s: %q, want %q", route, got, tc.want)
				}
			}
		})
	}
}

// TestSpellWeaponTextUsesTheInstallCaptions: an install's main.txt captions
// 189, 92, 93, 118 and 123 label the lines; without them the authored words
// stand in, and an absent duration caption drops its line.
func TestSpellWeaponTextUsesTheInstallCaptions(t *testing.T) {
	table, weapon := staffTooltipTable(t, 1, "Fire Arrow", 0, 7, 5, 10, 0)
	item := sim.ItemInstance{Code: uint16(weapon.Code), Effects: []sim.ItemEffect{{Kind: 41, Operand: 1}}}
	words := ui.AuthoredWords()
	words.Hover[189], words.Hover[92], words.Hover[93] = "M:", "c", "!"
	words.Hover[118], words.Hover[123] = "D", "R"
	words.ItemSpellNames[1] = "Spell"
	if got, want := itemInstanceInfoLines(item, table, words), []string{"Staff", "M:", "#c Spell!", "D: 5-10", "R: 5"}; !slices.Equal(got, want) {
		t.Fatalf("installed captions: %q, want %q", got, want)
	}
	authored := ui.AuthoredWords()
	if got, want := itemInstanceInfoLines(item, table), []string{"Staff", authored.ItemMagic, "#" + authored.ItemCasts + " Fire Arrow",
		authored.ItemDamage + ": 5-10", authored.ItemRange + ": 5"}; !slices.Equal(got, want) {
		t.Fatalf("authored fallback: %q, want %q", got, want)
	}
}

// TestSpellWeaponTextKeepsTheEffectOrder: the heading precedes every Effect
// and a castSpell Effect's lines stand where the Effect stands in the list;
// an unknown spell id still states the Weapon's range.
func TestSpellWeaponTextKeepsTheEffectOrder(t *testing.T) {
	table, weapon := staffTooltipTable(t, 1, "Fire Arrow", 0, 7, 5, 10, 0)
	words := spellWeaponWords()
	item := sim.ItemInstance{Code: uint16(weapon.Code), Effects: []sim.ItemEffect{
		{Kind: 12, Operand: 3}, {Kind: 41, Operand: 1}}}
	got := itemInstanceInfoLines(item, table, words)
	want := []string{"Staff", "Magic:", itemEffectInfoLine(item.Effects[0], table, words), "#casts Fire Arrow", "Damage: 5-10", "Range: 5"}
	if !slices.Equal(got, want) {
		t.Fatalf("effect order: %q, want %q", got, want)
	}
	unknown := sim.ItemInstance{Code: uint16(weapon.Code), Effects: []sim.ItemEffect{{Kind: 41, Operand: 19}}}
	if got, want := itemInstanceInfoLines(unknown, table, words), []string{"Staff", "Magic:", "#casts Spell 19", "Range: 5"}; !slices.Equal(got, want) {
		t.Fatalf("unknown spell: %q, want %q", got, want)
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

// A weapon's base pairs follow the formatter's stream order (ITEM-211): damage
// as [base, base+spread], to-hit and defence only when positive, the range
// byte whenever it exceeds 1 and whatever the attack type, all only when
// sutableFor has bit 0. The range pair uses the stats label of tag 38.
func TestItemInfoLinesBasePairsFollowTheWeaponStream(t *testing.T) {
	row := func(kind, toHit, defence, rng, suit int32) []int32 {
		return []int32{-1, -1, -1, -1, -1, kind, 10, 20, toHit, defence, -1, rng, 6, 4, -1, suit}
	}
	table := &mapload.Table{
		Shapes: emptyScale{}, Materials: emptyScale{},
		Weapons: dbCollection{{},
			{name: "Bow", params: row(data.SkillShoot, 0, 0, 6, 1)},
			{name: "Sword", params: row(data.SkillBlade, 5, 3, 1, 3)},
			{name: "Spear", params: row(data.SkillPike, 0, 0, 2, 3)},
			{name: "Wand", params: row(data.SkillBlade, 5, 3, 5, 2)},
			{name: "Flamer", params: row(11, 0, 0, 8, 1)},
			{name: "Beam", params: row(data.SkillShoot, 0, 0, 4, 0)},
			{name: "Mace", params: row(data.SkillBlade, 0, 0, 3, 0)},
		},
	}
	for _, tc := range []struct {
		row  int
		want []string
	}{
		{1, []string{"Bow", "#Damage: 10-20", "#Item lore 6"}},
		{2, []string{"Sword", "#Damage: 10-20", "#To-hit +5", "#Defence +3"}},
		{3, []string{"Spear", "#Damage: 10-20", "#Item lore 2"}},
		{4, []string{"Wand"}},
		{5, []string{"Flamer", "#Damage: 10-20", "#Item lore 8"}},
		{6, []string{"Beam", "#Item lore 4"}},
		{7, []string{"Mace"}},
	} {
		code := data.ComposeItemCode(0, 1, 0, tc.row)
		if got := itemInfoLines(code, table); !slices.Equal(got, tc.want) {
			t.Errorf("row %d lines = %q, want %q", tc.row, got, tc.want)
		}
		if got := itemInstanceInfoLines(sim.ItemInstance{Code: uint16(code)}, table); !slices.Equal(got, tc.want) {
			t.Errorf("row %d instance lines = %q, want %q", tc.row, got, tc.want)
		}
	}
	words := ui.AuthoredWords()
	words.ItemStats[38] = "Installed Range"
	code := data.ComposeItemCode(0, 1, 0, 1)
	if got := itemInstanceInfoLines(sim.ItemInstance{Code: uint16(code)}, table, words); got[len(got)-1] != "#Installed Range 6" {
		t.Errorf("installed caption lines = %q", got)
	}
	// Effects follow the base pairs under the magic heading.
	with := itemInstanceInfoLines(sim.ItemInstance{Code: uint16(code), Effects: []sim.ItemEffect{{Kind: 2, Operand: 3}}}, table)
	if len(with) != 5 || with[2] != "#Item lore 6" || with[3] != words.ItemMagic || !strings.HasPrefix(with[4], "#") {
		t.Errorf("effect lines = %q", with)
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
