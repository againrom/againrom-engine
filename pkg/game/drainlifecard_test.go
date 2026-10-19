package game

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"

	"golang.org/x/text/encoding/charmap"
)

func TestDrainLifeItemCardsUseCurrentSpellAcrossProducers(t *testing.T) {
	for _, tc := range []struct {
		power int32
		line  string
	}{{0, "Damage: 3-5"}, {30, "Damage: 6-10"}} {
		table, weapon := staffTooltipTable(t, 11, "Drain Life", tc.power, 7, 3, 5, 0)
		item := sim.ItemInstance{Code: uint16(weapon.Code), Price: 123, Effects: []sim.ItemEffect{
			{Kind: 41, Operand: 11 | uint32(uint16(tc.power))<<16},
		}}
		beforeItem, err := json.Marshal(item)
		if err != nil {
			t.Fatal(err)
		}
		rules := mapload.SpellRules(table)
		world, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: 2, Height: 2}, sim.ModeCanonical,
			sim.Terrain{}, []sim.Entity{{ID: 7, MaxMana: 1, WeaponSpell: 11, WeaponSpellLevel: tc.power}},
			nil, sim.Relations{}, nil, []sim.Stock{{ID: 7,
				Equipped:      [sim.EquipSlots]uint16{uint16(weapon.Code)},
				EquippedItems: [sim.EquipSlots]sim.ItemInstance{item}}}, rules)
		if err != nil {
			t.Fatal(err)
		}
		beforeWorld, err := world.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		beforeHash := world.Hash()
		words := ui.AuthoredWords()
		words.ItemSpellNames[11] = "Localized Drain Life"
		want := []string{"Staff", "MAGIC:", "#Casts Localized Drain Life", tc.line, "Range: 5"}
		for _, route := range []struct {
			name    string
			resolve weaponCastResolver
		}{
			{"live", func(code data.ItemCode) (uint16, int32, bool) { return liveWeaponCast(world, 7, code) }},
			{"stored", func(code data.ItemCode) (uint16, int32, bool) { return storedWeaponCast(code, weapon, table) }},
			{"instance", nil},
		} {
			got := itemInstanceInfoLinesWithWeaponDamage(item, table, route.resolve, words)
			if !slices.Equal(got, want) {
				t.Errorf("synthetic power %d %s card = %q, want literal %q", tc.power, route.name, got, want)
			}
			if code := itemInfoLinesWithWeaponDamage(weapon.Code, table, route.resolve, words); route.resolve != nil && !slices.Equal(code, want) {
				t.Errorf("synthetic power %d %s code card = %q, want literal %q", tc.power, route.name, code, want)
			}
		}
		afterWorld, err := world.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		afterItem, err := json.Marshal(item)
		if err != nil {
			t.Fatal(err)
		}
		if beforeHash != world.Hash() || !bytes.Equal(beforeWorld, afterWorld) || !bytes.Equal(beforeItem, afterItem) {
			t.Fatal("card projection changed complete item or World bytes/hash")
		}
		if _, _, ok := liveWeaponCast(world, 7, weapon.Code+1); ok {
			t.Fatal("unequipped-code live control obtained a cast")
		}
		for _, rule := range world.Spells() {
			if rule.ID == 11 && rule.Damaging {
				t.Fatal("Drain Life gameplay flag changed")
			}
		}
	}
}

func TestDrainLifeItemAliasPreservesRawNamesAndSpellbookNames(t *testing.T) {
	for _, selector := range []int{0, 1} {
		rawName, want := "Drain Life", "Drain Life"
		if selector == 1 {
			var err error
			rawName, err = charmap.CodePage866.NewEncoder().String("Высасывание жизни")
			if err != nil {
				t.Fatal(err)
			}
			want, err = charmap.CodePage866.NewEncoder().String("Вампиризм")
			if err != nil {
				t.Fatal(err)
			}
		}
		names := textFile(28, map[int]string{5: "raw Heal", 10: rawName, 13: "raw Prismatic"})
		bookRows := make([]string, len(originalBookIDs))
		for i := range bookRows {
			bookRows[i] = "book name#book description"
		}
		bookNames := []byte(strings.Join(bookRows, "\r\n") + "\r\n")
		beforeNames, beforeBooks := bytes.Clone(names), bytes.Clone(bookNames)
		language := map[int]string{0: "english", 1: "russian"}[selector]
		words := &InstallWords{Selector: selector, Language: language, spellNames: SplitTextTable(names), spellBookNames: SplitTextTable(bookNames)}
		got := words.Words()
		if got.ItemSpellNames[11] != want || got.ItemSpellNames[6] != "raw Heal" || got.ItemSpellNames[14] != "raw Prismatic" || got.SpellBookNames[1] != "book name" || got.SpellBookNames[11] != "" {
			t.Fatalf("selector%d changed the alias boundary: drain=%x heal=%q prism=%q book1=%q book11=%q", selector, got.ItemSpellNames[11], got.ItemSpellNames[6], got.ItemSpellNames[14], got.SpellBookNames[1], got.SpellBookNames[11])
		}
		if raw, _ := words.spellNames.At(10); raw != rawName || !bytes.Equal(names, beforeNames) || !bytes.Equal(bookNames, beforeBooks) {
			t.Fatal("item alias changed raw name sources")
		}
	}
}

// TestUndescribedCastUsesInstalledNameWithoutDamage: a castSpell Weapon whose
// spell carries no damage flag still states the filled Spell's damage when its
// spread is nonzero, and the Weapon's range (ITEM-211, TEXT-129); a spell id
// without a Spells row states its number and the range; a teach-spell Effect
// keeps its fragment.
func TestUndescribedCastUsesInstalledNameWithoutDamage(t *testing.T) {
	table, weapon := staffTooltipTable(t, 6, "Heal", 30, 7, 3, 5, 0)
	words := ui.AuthoredWords()
	words.ItemSpellNames[6] = "Localized Heal"
	item := sim.ItemInstance{Code: uint16(weapon.Code), Effects: []sim.ItemEffect{{Kind: 41, Operand: 6 | 30<<16}}}
	before := item.Clone()
	got := itemInstanceInfoLines(item, table, words)
	if want := []string{"Staff", "MAGIC:", "#Casts Localized Heal", "Damage: 6-10", "Range: 5"}; !slices.Equal(got, want) {
		t.Fatalf("Heal staff = %q, want %q", got, want)
	}
	if !reflect.DeepEqual(item, before) {
		t.Fatal("the text changed the complete item")
	}
	unknown := item.Clone()
	unknown.Effects[0].Operand = 65535 | 30<<16
	if got, want := itemInstanceInfoLines(unknown, table, words), []string{"Staff", "MAGIC:", "#Casts spell 255", "Range: 5"}; !slices.Equal(got, want) {
		t.Fatalf("unknown spell = %q, want %q", got, want)
	}
	teach := item.Clone()
	teach.Effects[0] = sim.ItemEffect{Kind: 42, Operand: 6}
	if got := itemInstanceInfoLines(teach, table, words); got[len(got)-1] != "MAGIC: of Localized Heal" {
		t.Fatalf("kind42 fragment changed: %q", got)
	}
}
