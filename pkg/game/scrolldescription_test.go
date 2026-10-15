package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func scrollDescriptionExpectedDetails(id uint16, power int32, c sim.SpellCharacteristics, label func(int) string) []string {
	var lines []string
	if c.DamageMax-c.DamageMin != 0 {
		lines = append(lines, fmt.Sprintf("%s: %d-%d", label(118), c.DamageMin, c.DamageMax))
	}
	if c.Range != 0 {
		lines = append(lines, fmt.Sprintf("%s: %d", label(123), c.Range))
	}
	if c.Duration != 0 {
		lines = append(lines, fmt.Sprintf("%s: %5.1f", label(124), float64(c.Duration)/16))
	}
	caption := ""
	switch id {
	case 7, 28:
		caption = fmt.Sprintf("%s: %d", label(182), -(power/15 + 1))
	case 24:
		caption = fmt.Sprintf("%s: %d", label(182), power/15+1)
	case 5, 10, 16, 22:
		if power/2 != 0 {
			caption = fmt.Sprintf("%s: +%d", label(183), min(power/2, 100))
		}
	case 12, 17:
		caption = fmt.Sprintf("%s: %d", label(184), -(power/30 + 1))
	case 23:
		caption = fmt.Sprintf("%s: +%d%%", label(185), 4*power/5+20)
	case 27:
		caption = fmt.Sprintf("%s: +%d%%", label(187), 4*power/5+20)
	case 14:
		caption = fmt.Sprintf("%s: %d", label(186), min(power/20+2, 7))
	}
	if caption != "" {
		lines = append(lines, caption)
	}
	return lines
}

func scrollDescriptionFixture() (*mapload.Table, ui.Words) {
	spells := make(dbCollection, 29)
	words := captionWords()
	words.ItemSpellOf = "of"
	for id := 1; id < len(spells); id++ {
		spells[id] = dbEntry{name: fmt.Sprintf("raw spell %d", id), params: make([]int32, 22)}
		p := spells[id].params
		p[1], p[2], p[6], p[8], p[16], p[17] = 19, 1, 7, 1, 4, 8
		words.ItemSpellNames[id] = fmt.Sprintf("localized spell %d", id)
	}
	return &mapload.Table{Spells: spells, Names: data.ItemNames{0x0e10: "Scroll"}}, words
}

func TestScrollDescriptionFieldGrammarAndCaptionPrecedence(t *testing.T) {
	words := captionWords()
	for _, tc := range []struct {
		name string
		id   uint16
		c    sim.SpellCharacteristics
		want []string
	}{
		{"equal damage omits the line", 1, sim.SpellCharacteristics{ManaCost: 19, DamageMin: 7, DamageMax: 7, Range: 3, Duration: 16}, []string{"Range: 3", "Duration:   1.0"}},
		{"zero minimum keeps a nonzero spread", 1, sim.SpellCharacteristics{DamageMax: 5}, []string{"Damage: 0-5"}},
		{"zero maxima omit fields", 1, sim.SpellCharacteristics{ManaCost: 19}, nil},
		{"zero protection omits its caption", 5, sim.SpellCharacteristics{Power: 1}, nil},
		{"zero signed magnitude omits its caption", 28, sim.SpellCharacteristics{Power: 80, HasMagnitude: true}, nil},
		{"shield omits book absorption", 18, sim.SpellCharacteristics{Power: 80, HasMagnitude: true, Magnitude: 11}, nil},
		{"area scan has its signed caption", 17, sim.SpellCharacteristics{Power: 80}, []string{"C184: -3"}},
		{"skills has its own percent caption", 27, sim.SpellCharacteristics{Power: 80}, []string{"C187: +84%"}},
		{"slow is a valid descriptor", 28, sim.SpellCharacteristics{Power: 80}, []string{"C182: -6"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := spellDetailsLines([]spellKnower{{sim.SpellRule{ID: tc.id}, tc.c}}, nil, &words, scrollDetails)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("details = %q, want %q", got, tc.want)
			}
		})
	}
	ids := []uint16{24, 23, 27, 5, 14, 17}
	want := []string{"C182: 6", "C185: +84%", "C187: +84%", "C183: +40", "C186: 6", "C184: -3"}
	for end := 1; end <= len(ids); end++ {
		var knowers []spellKnower
		for _, id := range ids[:end] {
			knowers = append(knowers, knower(id, 80))
		}
		got := spellDetailsLines(knowers, nil, &words, scrollDetails)
		if !slices.Equal(got, []string{want[end-1]}) {
			t.Fatalf("caption prefix%d = %q, want only %q", end, got, want[end-1])
		}
	}
}

func TestScrollDescriptionsCoverEveryDescriptorAndBytePower(t *testing.T) {
	table, words := scrollDescriptionFixture()
	rules := mapload.SpellRules(table)
	for id := uint16(1); id <= 28; id++ {
		for _, stored := range []int32{0, 1, 30, 60, 100, 101, 255, 256, -1} {
			power := int32(uint8(stored))
			item := sim.ItemInstance{Code: 0x0e10, Kind: 4, Price: 137,
				Effects: []sim.ItemEffect{{Kind: 41, Operand: uint32(id) | uint32(uint16(stored))<<16}}}
			c := sim.SpellCharacteristicsAtPower(mapload.TableRules(table), rules[id-1], power)
			want := []string{fmt.Sprintf("Scroll of localized spell %d", id)}
			want = append(want, scrollDescriptionExpectedDetails(id, power, c, func(i int) string { return words.Hover[i] })...)
			got := itemInstanceInfoLines(item, table, words)
			if !slices.Equal(got, want) {
				t.Fatalf("spell%d stored%d byte%d = %q, want %q", id, stored, power, got, want)
			}
			for _, quantity := range []uint32{1, 3} {
				pack := packInfoLinesWithWeaponDamage([]sim.ItemStack{sim.StackItem(item, quantity)}, table, nil, words)
				if len(pack) != 1 || !slices.Equal(pack[0], want) {
					t.Fatalf("spell%d stored%d quantity%d = %q, want %q", id, stored, quantity, pack, want)
				}
			}
		}
	}
}

func TestScrollDescriptionsPreserveFormulaTablesAndWideDamage(t *testing.T) {
	table, words := scrollDescriptionFixture()
	rows := table.Spells.(dbCollection)
	rows[1].params[16], rows[1].params[17] = 300, 600
	rows[20].params[14] = 1
	var set sim.SpellFormulaSet
	set.Spell[sim.FormulaDamage][1] = []int32{60}
	set.Spell[sim.FormulaRange][1] = []int32{6}
	set.Spell[sim.FormulaPower][1] = []int32{0}
	set.Spell[sim.FormulaDuration][20] = []int32{2000}
	set.Spell[sim.FormulaMagnitude][17] = []int32{9}
	set.Spell[sim.FormulaMagnitude][27] = []int32{55}
	var err error
	table.Rules, err = sim.Rules{}.WithSpellFormulas(set)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id   uint16
		want []string
	}{
		{1, []string{"Scroll of localized spell 1", "Damage: 600-1200", "Range: 13"}},
		{20, []string{"Scroll of localized spell 20", "Damage: 17-34", "Range: 10", "Duration:   2.0"}},
		{17, []string{"Scroll of localized spell 17", "Damage: 17-34", "Range: 10", "C184: -9"}},
		{27, []string{"Scroll of localized spell 27", "Damage: 17-34", "Range: 10", "C187: +55%"}},
	} {
		item := sim.ItemInstance{Code: 0x0e10, Kind: 4, Price: 137,
			Effects: []sim.ItemEffect{{Kind: 41, Operand: uint32(tc.id) | 100<<16}}}
		if got := itemInstanceInfoLines(item, table, words); !slices.Equal(got, tc.want) {
			t.Fatalf("spell%d formula projection = %q, want %q", tc.id, got, tc.want)
		}
	}
}

func TestScrollDescriptionsKeepByteTransportAndZeroSpreadOmission(t *testing.T) {
	table, words := scrollDescriptionFixture()
	rows := table.Spells.(dbCollection)
	rows[1].params[6], rows[1].params[16], rows[1].params[17] = 254, 7, 7
	item := sim.ItemInstance{Code: 0x0e10, Kind: 4, Price: 137,
		Effects: []sim.ItemEffect{{Kind: 41, Operand: 0xab01 | 0x013c<<16}}}
	if got := itemInstanceInfoLines(item, table, words); !slices.Equal(got, []string{"Scroll of localized spell 1"}) {
		t.Fatalf("low-byte identity/power, wrapped range and zero spread = %q", got)
	}
	rows[26].params[6], rows[26].params[16], rows[26].params[17] = 0, 0, 0
	item.Effects[0].Operand = 26 | 6<<16
	if got := itemInstanceInfoLines(item, table, words); !slices.Equal(got, []string{"Scroll of localized spell 26", "Range: 2"}) {
		t.Fatalf("zero-base Teleport record range = %q", got)
	}
}

func TestScrollDescriptionChangesPreserveTheSpellbookCaptionPopulation(t *testing.T) {
	words := captionWords()
	captions := map[uint16]string{
		5: "C183: +40", 7: "C182: -6", 10: "C183: +40", 12: "C184: -3",
		14: "C186: 6", 16: "C183: +40", 18: "C217: 11", 22: "C183: +40",
		23: "C185: +84%", 24: "C182: 6",
	}
	for id := uint16(1); id <= 28; id++ {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			name := fmt.Sprintf("spell %d", id)
			want := []string{name, "Mana cost: 9", "Damage: 2-5", "Range: 8", "Duration:   1.0"}
			if caption := captions[id]; caption != "" {
				want = append(want, caption)
			}
			got := spellInfoLines(sim.SpellRule{ID: id}, sim.SpellCharacteristics{
				Power: 80, ManaCost: 9, DamageMin: 2, DamageMax: 5, Range: 8, Duration: 16,
			}, name, &words)
			if !slices.Equal(got, want) {
				t.Fatalf("book popup = %q, want %q", got, want)
			}
		})
	}
}
