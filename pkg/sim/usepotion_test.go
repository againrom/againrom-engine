package sim

import (
	"bytes"
	"math"
	"testing"
)

func testPotion(effects ...ItemEffect) ItemInstance {
	return ItemInstance{Code: 0x0e07, Kind: 3, Price: 500, Effects: effects}
}

func TestPotionTimedReplacementRetainsFirstKindAndNativeRemaining(t *testing.T) {
	for _, mage := range []bool{false, true} {
		for _, first := range []uint8{8, 11, 16} {
			e := Entity{ID: 7, HP: 10, MaxHP: 100, HealthRegeneration: 5, ManaRegeneration: 7, Absorption: 10}
			if mage {
				e.MaxMana = 100
			}
			p := testPotion(ItemEffect{Kind: first, Mode: 1, Operand: 100 | 960<<16})
			q := testPotion(ItemEffect{Kind: 11, Mode: 1, Operand: 250 | 1920<<16})
			w := bookWorld(t, e, p, q)
			if !w.usePotion(0, 0) {
				t.Fatal("first use refused")
			}
			for n := 0; n < 13; n++ {
				Step(w, nil)
			}
			if !w.usePotion(0, 0) {
				t.Fatal("repeat refused")
			}
			effects := w.ActiveEffects()
			wantKind := map[uint8]EffectKind{8: EffectHealthRegeneration, 11: EffectManaRegeneration, 16: EffectAbsorption}[first]
			if len(effects) != 1 || effects[0].Spell != 0 || effects[0].Kind != wantKind || effects[0].Remaining != 1920 {
				t.Fatalf("mage=%v kind=%d effects=%+v", mage, first, effects)
			}
			form, _ := w.MarshalBinary()
			var loaded World
			if err := loaded.UnmarshalBinary(form); err != nil {
				t.Fatal(err)
			}
			for n := 0; n < 1920; n++ {
				Step(w, nil)
				Step(&loaded, nil)
				if w.Hash() != loaded.Hash() {
					t.Fatalf("native differs at %d", n)
				}
			}
			got := w.Entities()[0]
			if len(w.ActiveEffects()) != 0 || got.HealthRegeneration != 5 || got.ManaRegeneration != 7 || got.Absorption != 10 {
				t.Fatalf("expiry did not restore base: %+v", got)
			}
		}
	}
}

func TestPotionPermanentCapConsumesWithoutChangingModifierOrLearning(t *testing.T) {
	e := Entity{ID: 7, HP: 10, MaxHP: 100, PotionHeadroom: [4]int32{1, 0, 2, 3}}
	p := testPotion(ItemEffect{Kind: 2, Operand: 1}, ItemEffect{Kind: 4, Operand: 1}, ItemEffect{Kind: 3, Operand: 1}, ItemEffect{Kind: 5, Operand: 1})
	w := bookWorld(t, e, p, p)
	if !w.usePotion(0, 0) || !w.usePotion(0, 0) {
		t.Fatal("use refused")
	}
	got := w.Entities()[0]
	if got.PotionStats != [4]int32{1, 0, 2, 2} || got.PotionHeadroom != [4]int32{0, 0, 0, 1} || got.KnownSpells != 0 {
		t.Fatalf("permanent use=%+v", got)
	}
	form, _ := w.MarshalBinary()
	var restored World
	if err := restored.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if w.Hash() != restored.Hash() {
		t.Fatal("permanent state was lost")
	}
}

func TestPotionInstantPoolsConsumeOneUnitAndNeverLearnASpell(t *testing.T) {
	for _, mage := range []bool{false, true} {
		e := Entity{ID: 7, X: 2, Y: 2, HP: 10, MaxHP: 100}
		if mage {
			e.Mana, e.MaxMana = 5, 80
		}
		potion := testPotion(ItemEffect{Kind: 6, Operand: 30}, ItemEffect{Kind: 9, Operand: 30})
		w := bookWorld(t, e, potion, potion)
		Step(w, []Command{{Kind: KindUsePotion, Entity: 7, X: 0}})
		got := bookEntity(t, w, 7)
		if got.HP != 40 || got.KnownSpells != 0 || mage && got.Mana != 35 || !mage && got.Mana != 0 {
			t.Fatalf("mage=%v: actor=%+v", mage, got)
		}
		stacks, _ := w.CarriedStacks(7)
		if len(stacks) != 1 || stacks[0].Count != 1 || !ItemEqual(stacks[0].Instance(), potion) {
			t.Fatalf("mage=%v: remaining=%+v", mage, stacks)
		}
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var restored World
		if err := restored.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
		for n := 0; n < 40; n++ {
			var cmd []Command
			if n == 0 {
				cmd = []Command{{Kind: KindUsePotion, Entity: 7, X: 0}}
			}
			Step(w, cmd)
			Step(&restored, cmd)
			if w.Hash() != restored.Hash() {
				t.Fatalf("native continuation diverged at %d", n)
			}
		}
		if stacks, _ := w.CarriedStacks(7); len(stacks) != 0 {
			t.Fatalf("final unit not consumed: %+v", stacks)
		}
	}
}

func TestPotionRefusalIsAtomicAcrossActorIndexAndPayload(t *testing.T) {
	valid := testPotion(ItemEffect{Kind: 6, Operand: 30})
	for _, tc := range []struct {
		name  string
		hp    int32
		off   bool
		index int
		item  ItemInstance
	}{
		{"downed", 0, false, 0, valid}, {"dead", -10, false, 0, valid},
		{"offmap", 10, true, 0, valid}, {"negative", 10, false, -1, valid},
		{"pastend", 10, false, 1, valid}, {"empty", 10, false, 0, testPotion()},
		{"mixed unknown", 10, false, 0, testPotion(ItemEffect{Kind: 6, Operand: 30}, ItemEffect{Kind: 255})},
		{"not potion", 10, false, 0, testSpellBook(6)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := bookWorld(t, Entity{ID: 7, HP: tc.hp, MaxHP: 100, OffMap: tc.off}, tc.item)
			before, _ := w.MarshalBinary()
			if w.usePotion(0, tc.index) {
				t.Fatal("admitted invalid use")
			}
			after, _ := w.MarshalBinary()
			if !bytes.Equal(before, after) {
				t.Fatal("refusal changed canonical state")
			}
		})
	}
}

func TestPotionPoolsClampWithoutOverflow(t *testing.T) {
	if got := potionPool(math.MaxInt32-2, math.MaxInt32, 100); got != math.MaxInt32 {
		t.Fatal(got)
	}
	if got := potionPool(math.MinInt32+2, 100, -100); got != math.MinInt32 {
		t.Fatal(got)
	}
	if got := potionPool(99, 100, 30); got != 100 {
		t.Fatal(got)
	}
}
