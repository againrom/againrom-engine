package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// captionInstallWords are two installs' words: the English labels and a
// Russian set, each with distinct damage, range, duration and stat labels.
func captionInstallWords() map[string]ui.Words {
	en := scrollDescriptionWords()
	ru := scrollDescriptionWords()
	ru.Hover[118], ru.Hover[123], ru.Hover[124] = "Урон", "Дистанция", "Длительность"
	for _, n := range []int{182, 183, 184, 185, 186, 187} {
		ru.Hover[n] = fmt.Sprintf("Свойство %d", n)
	}
	ru.ItemStats[45] = "Огонь"
	en.ItemStats[45] = "Fire"
	return map[string]ui.Words{"EN": en, "RU": ru}
}

func scrollDescriptionWords() ui.Words {
	_, words := scrollDescriptionFixture()
	return words
}

func scrollCaptionTable(second bool) *mapload.Table {
	table, _ := scrollDescriptionFixture()
	if second {
		edition := base.GameROM2.Edition()
		table.Edition, table.SpellArms = &edition, mapload.SecondGameSpellArms
	}
	return table
}

// The second game's scroll caption names the spell, then the Damage, Range
// and Duration lines of the claim, then the one stat line of the getter that
// answers the spell id; a spell no getter answers, Lightning among them, has
// none. The first game's caption of the same scrolls does not move.
func TestSecondGameScrollCaptionStatLine(t *testing.T) {
	const lightning = 10
	getter := map[uint16]int{21: 182, 20: 185, 28: 187, 4: 183, 11: 186, 14: 184, lightning: 0}
	for install, words := range captionInstallWords() {
		for id, label := range getter {
			item := sim.ItemInstance{Code: 0x0e10, Kind: 4, Price: 137,
				Effects: []sim.ItemEffect{{Kind: 41, Operand: uint32(id) | 100<<16}}}
			first := scrollCaptionTable(false)
			second := scrollCaptionTable(true)
			rules := mapload.SpellRules(second)
			c := sim.SpellCharacteristicsAtPower(mapload.TableRules(second), rules[id-1], 100)
			head := []string{fmt.Sprintf("Scroll of localized spell %d", id)}
			if c.DamageMax != c.DamageMin {
				head = append(head, fmt.Sprintf("%s: %d-%d", words.Hover[118], c.DamageMin, c.DamageMax))
			}
			if c.Range != 0 {
				head = append(head, fmt.Sprintf("%s: %d", words.Hover[123], c.Range))
			}
			if c.Duration != 0 {
				head = append(head, fmt.Sprintf("%s: %5.1f", words.Hover[124], float64(c.Duration)/16))
			}
			got := itemInstanceInfoLines(item, second, words)
			if len(got) < len(head) || !slices.Equal(got[:len(head)], head) {
				t.Errorf("%s spell %d: head %q, want %q", install, id, got, head)
			}
			stat := got[len(head):]
			if label == 0 {
				if len(stat) != 0 {
					t.Errorf("%s spell %d has no getter, caption %q", install, id, got)
				}
			} else if len(stat) != 1 || len(stat[0]) < len(words.Hover[label])+1 || stat[0][:len(words.Hover[label])+1] != words.Hover[label]+":" {
				t.Errorf("%s spell %d: stat line %q, want label %q", install, id, stat, words.Hover[label])
			}
			// The first game's caption of the same scroll is the arm's own.
			c1 := sim.SpellCharacteristicsAtPower(mapload.TableRules(first), mapload.SpellRules(first)[id-1], 100)
			want := append([]string{head[0]}, scrollDescriptionExpectedDetails(uint16(id), 100, c1, func(i int) string { return words.Hover[i] })...)
			if got := itemInstanceInfoLines(item, first, words); !slices.Equal(got, want) {
				t.Errorf("%s spell %d: first game caption %q, want %q", install, id, got, want)
			}
		}
	}
}

// An elemental damage effect reads `#name v-w` in both games, with no colon.
func TestElementalDamageLineHasNoColon(t *testing.T) {
	for install, words := range captionInstallWords() {
		item := sim.ItemInstance{Code: 0x0e10, Kind: 4, Effects: []sim.ItemEffect{{Kind: 45, Operand: 5 | 3<<8}}}
		name := "Scroll"
		want := "#" + words.ItemStats[45] + " 5-8"
		for _, table := range []bool{true, false} {
			got := itemInstanceInfoLines(item, scrollCaptionTable(table), words)
			if len(got) < 1 || got[len(got)-1] != want {
				t.Errorf("%s %s second=%v: caption %q, want last line %q", install, name, table, got, want)
			}
		}
	}
}
