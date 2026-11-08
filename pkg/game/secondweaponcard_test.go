package game

import (
	"slices"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestSecondGameFighterWeaponWithSpellStatesItsOwnDamage(t *testing.T) {
	second := base.GameROM2.Edition()
	words := spellWeaponWords()
	build := func(name string, suits int32, spellID int, spell string, power int32) ([]string, []string) {
		table, weapon := spellWeaponTable(t, name, 4, suits, spellID, spell, power, 7, 5, 10, 0)
		params := table.Weapons.(dbCollection)[1].params
		params[6], params[7], params[8], params[9] = 12, 17, 3, 2
		table.Edition, table.SpellArms = &second, mapload.SecondGameSpellArms
		item := sim.ItemInstance{Code: uint16(weapon.Code), Effects: []sim.ItemEffect{castSpellEffect(uint16(spellID), power)}}
		return itemInstanceInfoLines(item, table, words), itemBaseInfoLines(weapon.Code, table, &words)
	}
	baseLines := []string{"#" + words.ItemStats[43] + ": 12-17", "#" + words.ItemStats[12] + " +3",
		"#" + words.ItemStats[15] + " +2", "#" + words.ItemStats[38] + " 4"}
	got, _ := build("Sword", 1, 1, "Fire Arrow", 30)
	want := append(append([]string{"Sword"}, baseLines...), words.ItemMagic)
	if len(got) < len(want) || !slices.Equal(got[:len(want)], want) {
		t.Fatalf("fighter weapon card = %q, want the head %q", got, want)
	}
	if !slices.Contains(got, words.ItemCasts+" Fire Arrow") && len(got) == len(want) {
		t.Fatalf("fighter weapon card lost its spell lines: %q", got)
	}
	if staff, _ := build("Staff", 2, 1, "Fire Arrow", 30); staff[1] != words.ItemMagic {
		t.Fatalf("staff card gained base lines: %q", staff)
	}
	if curse, _ := build("Sword", 1, 18, "Stone Curse", 10); !slices.Equal(curse[:6], want) {
		t.Fatalf("fighter weapon with Stone Curse = %q", curse)
	}
	if staff, _ := build("Staff", 2, 18, "Stone Curse", 10); staff[1] != words.ItemMagic {
		t.Fatalf("Stone Curse staff card gained base lines: %q", staff)
	}
	_, plain := build("Sword", 1, 1, "Fire Arrow", 30)
	if !slices.Equal(plain, append([]string{"Sword"}, baseLines...)) {
		t.Fatalf("weapon without a spell = %q", plain)
	}
}
