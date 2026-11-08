package game

import (
	"slices"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The second game's staff with Stone Curse states the spell's variable
// duration as 0.0 - floor(2.4 L)/16 after the casts line and the weapon's own
// range (R2-ENGINE-999). The first game's card is the loss control.
func TestSecondGameStoneCurseStaffCardStatesTheVariableDuration(t *testing.T) {
	second := base.GameROM2.Edition()
	for _, lang := range []struct {
		name                               string
		casts, rng, duration, curse, magic string
	}{
		{"EN", "Casts", "Range", "Duration", "Stone Curse", "Magic:"},
		{"RU", "с заклинанием", "Дистанция", "Длительность", "Каменное проклятие", "Магия:"},
	} {
		words := ui.AuthoredWords()
		words.ItemCasts, words.ItemMagic = lang.casts, lang.magic
		words.Hover[123], words.Hover[124] = lang.rng, lang.duration
		words.ItemSpellNames[18] = lang.curse
		for _, tc := range []struct {
			power int32
			want  string
		}{{10, "1.5"}, {50, "7.5"}, {100, "15.0"}} {
			table, weapon := staffTooltipTable(t, 18, "Stone Curse", tc.power, 6, 5, 15, 10)
			table.Edition, table.SpellArms = &second, mapload.SecondGameSpellArms
			item := sim.ItemInstance{Code: uint16(weapon.Code), Effects: []sim.ItemEffect{castSpellEffect(18, tc.power)}}
			got := itemInstanceInfoLines(item, table, words)
			want := []string{
				lang.casts + " " + lang.curse,
				lang.rng + " 5",
				lang.duration + " 0.0 - " + tc.want,
			}
			if len(got) < 3 || !slices.Equal(got[len(got)-3:], want) {
				t.Fatalf("%s power %d card = %q, want the tail %q", lang.name, tc.power, got, want)
			}
			for _, line := range got {
				if line == lang.curse+" 10 "+words.DurationUnit || line == lang.rng+" 6" {
					t.Fatalf("%s power %d card keeps the fixed-duration line %q", lang.name, tc.power, line)
				}
			}
		}
	}
}

func TestFirstGameStoneCurseStaffCardIsUnchanged(t *testing.T) {
	words := spellWeaponWords()
	table, weapon := staffTooltipTable(t, 18, "Stone Curse", 10, 6, 5, 15, 10)
	item := sim.ItemInstance{Code: uint16(weapon.Code), Effects: []sim.ItemEffect{castSpellEffect(18, 10)}}
	got := itemInstanceInfoLines(item, table, words)
	want := []string{"Staff", "Magic:", "#casts Stone Curse", "Damage: 6-20", "Range: 5", "Duration:  12.8"}
	if !slices.Equal(got, want) {
		t.Fatalf("first-game card = %q", got)
	}
}
