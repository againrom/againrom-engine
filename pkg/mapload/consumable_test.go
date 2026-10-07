package mapload

import (
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func TestPotionDisplayProjectsAllCarriedModifiersWithoutMutation(t *testing.T) {
	for _, mage := range []bool{false, true} {
		for _, kind := range []sim.EffectKind{sim.EffectAbsorption, sim.EffectHealthRegeneration, sim.EffectManaRegeneration} {
			p := PartyMember{Hero: data.Hero{Body: 30, Reaction: 30, Mind: 30, Spirit: 30}, Profile: data.Profile{HealthColumn: true, ManaColumn: mage}, Mage: mage,
				PotionEffect: &sim.ActiveEffect{Kind: kind, Mode: sim.EffectDuration, Magnitude: 50, Remaining: 17}}
			before := clonePartyMember(p)
			base, hp, mana := PartySpawnWithTable(p, nil)
			want := base
			switch kind {
			case sim.EffectAbsorption:
				want.Combat.Absorption += 50
			case sim.EffectHealthRegeneration:
				want.HealthRegeneration += 50
			case sim.EffectManaRegeneration:
				if mage {
					want.ManaRegeneration += 50
				}
			}
			for n := 0; n < 3; n++ {
				got, gotHP, gotMana := PartyDisplayWithTable(p, nil)
				if got != want || gotHP != hp || gotMana != mana || !reflect.DeepEqual(p, before) {
					t.Fatalf("mage=%v kind=%d: projection mutated or differs", mage, kind)
				}
			}
			p.PotionEffect.Remaining = 0
			if got, _, _ := PartyDisplayWithTable(p, nil); got != base {
				t.Fatal("expired attachment was projected")
			}
		}
	}
}

type consumableRow struct {
	name, effects string
	price         int32
}
type consumableRows []consumableRow

func (r consumableRows) Len() int                    { return len(r) }
func (r consumableRows) EntryName(i int) string      { return r[i].name }
func (r consumableRows) EntryParams(i int) []int32   { return []int32{r[i].price, 1} }
func (r consumableRows) EntryStrings(i int) []string { return []string{r[i].effects} }

func TestPotionConstructionKeepsInstalledOrderedPayloadAndBaseValue(t *testing.T) {
	table := &Table{MagicItems: consumableRows{{}, {"Potion Custom", "health=+30, manaRegeneration=100: duration 60", 47}}}
	got := ItemInstanceFromCode(0x0e01, table)
	want := sim.ItemInstance{Code: 0x0e01, Kind: 3, Price: 47, Effects: []sim.ItemEffect{
		{Kind: 6, Operand: 30}, {Kind: 11, Mode: 1, Operand: 100 | 960<<16},
	}}
	if !reflect.DeepEqual(got, want) || RepriceItemInstance(got, table) != 47 {
		t.Fatalf("constructed=%+v repriced=%d want=%+v", got, RepriceItemInstance(got, table), want)
	}
	got.Effects[0].Operand = 1
	if again := ItemInstanceFromCode(0x0e01, table); !reflect.DeepEqual(again, want) {
		t.Fatalf("constructor shares mutable effect storage: %+v", again)
	}
}

func TestPotionMalformedEffectsNeverConstructAPartialUse(t *testing.T) {
	for _, text := range []string{"health=30,mana=not-a-number", "health=30,unknown=2", "health=2147483648", "health=3:duration x"} {
		t.Run(text, func(t *testing.T) {
			table := &Table{MagicItems: consumableRows{{}, {"Potion Custom", text, 10}}}
			item := ItemInstanceFromCode(0x0e01, table)
			if len(item.Effects) != 0 || item.Kind != 3 || item.Price != 10 {
				t.Fatalf("malformed payload became partially usable: %+v", item)
			}
		})
	}
}
