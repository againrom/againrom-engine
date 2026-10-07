package sim

import (
	"bytes"
	"reflect"
	"testing"
)

func structureAreaWorld(t *testing.T, structures []Structure, ents ...Entity) *World {
	t.Helper()
	w, err := NewStructuredWorld(1086, Bounds{Width: 40, Height: 40}, ModeCanonical,
		Terrain{}, ents, nil, Relations{}, nil, nil, nil, GhostTemplate{}, structures)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestStructureCellRegistrationMaskAliasCollisionAndEdge(t *testing.T) {
	for _, tc := range []struct {
		name       string
		structures []Structure
		want       map[uint16]int
	}{
		{"mask holes", []Structure{{ID: 1, Col: 10, Row: 11, Width: 3, Height: 2, Attach: 0b100101}},
			map[uint16]int{0x0b0a: 0, 0x0b0c: 0, 0x0c0c: 0}},
		{"five-bit mask alias", []Structure{{ID: 1, Col: 10, Row: 11, Width: 17, Height: 2, Attach: 1}},
			map[uint16]int{0x0b0a: 0, 0x0c19: 0}},
		{"prefix retained and suffix abandoned", []Structure{
			{ID: 20, Col: 9, Row: 10, Width: 3, Height: 2, Attach: 63},
			{ID: 10, Col: 10, Row: 10, Width: 1, Height: 1, Attach: 1}},
			map[uint16]int{0x0a0a: 0, 0x0a09: 1}},
		{"no playable-map clip", []Structure{{ID: 1, Col: 39, Row: 39, Width: 2, Height: 2, Attach: 15}},
			map[uint16]int{0x2727: 0, 0x2728: 0, 0x2827: 0, 0x2828: 0}},
		{"low-word sum carries x and wraps y", []Structure{{ID: 1, Col: 255, Row: 255, Width: 2, Height: 2, Attach: 15}},
			map[uint16]int{0xffff: 0, 0x0000: 0, 0x00ff: 0, 0x0100: 0}},
		{"empty shapes", []Structure{{ID: 1}, {ID: 2, Width: 255, Height: 255}}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := structureAreaWorld(t, tc.structures)
			if !reflect.DeepEqual(w.structureSlots, tc.want) {
				t.Fatalf("slots = %v, want independently enumerated %v", w.structureSlots, tc.want)
			}
			form, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var back World
			if err := back.UnmarshalBinary(form); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(back.structureSlots, tc.want) {
				t.Fatal("native reconstruction changed registration")
			}
		})
	}
}

func TestStructureDirectAreaCallsUseReferencesNotRectangleOrObjectCount(t *testing.T) {
	for _, tc := range []struct {
		name  string
		spell uint16
		mode  uint8
		cells []uint16
		calls int
	}{
		{"blast partial square", 2, areaModeBlast, []uint16{0x0a0a, 0x0a0b, 0x0a0c, 0x0b0a, 0x0b0b}, 4},
		{"ring repeated reference", 9, areaModeRing, []uint16{0x0a0a, 0x0a0a, 0x0b0b}, 3},
		{"cloud excludes structures", 8, areaModeCloud, []uint16{0x0a0a, 0x0a0b}, 0},
		{"earth wall per-target early return", 19, areaModeRing, []uint16{0x0a0a}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := structureAreaWorld(t, []Structure{{ID: 1, Field42: 100, MaxHealth: 100, Col: 10, Row: 10, Width: 2, Height: 2, Attach: 15}})
			// The call count is enumerated independently above. Only primitive
			// draws supply the exact 5..6 damage values, not the area walker.
			r := w.rng
			want := uint16(100)
			for n := 0; n < tc.calls; n++ {
				want -= uint16(5 + r.uniform(1))
			}
			w.applyAreaCells(cellEffect{Spell: tc.spell, Mode: tc.mode}, SpellRule{ID: tc.spell, Damaging: true, DamageMin: 10, DamageMax: 11}, tc.cells)
			if w.structures[0].Field42 != want || w.rng != r {
				t.Fatalf("HP/RNG = %d/%v, want %d/%v", w.structures[0].Field42, w.rng, want, r)
			}
		})
	}
}

func TestStructureAreaDamagePrefixAndHPZeroRetainSlots(t *testing.T) {
	w := structureAreaWorld(t, []Structure{
		{ID: 1, Field42: 100, MaxHealth: 100, Col: 11, Row: 10, Width: 1, Height: 1, Attach: 1},
		{ID: 2, Field42: 1, MaxHealth: 100, Col: 10, Row: 10, Width: 3, Height: 1, Attach: 7},
	})
	r := w.rng
	first := 5 + r.uniform(1)
	r.uniform(1)
	r.uniform(1)
	w.applyAreaCells(cellEffect{Mode: areaModeBlast}, SpellRule{ID: 2, Damaging: true, DamageMin: 10, DamageMax: 11}, []uint16{0x0a0b, 0x0a0a, 0x0a0a, 0x0a0c})
	if w.structures[0].Field42 != uint16(100-first) || w.structures[1].Field42 != 0 || w.rng != r {
		t.Fatalf("prefix/collision/lethal calls = %+v, rng %v want %v", w.structures, w.rng, r)
	}
	if len(w.structureSlots) != 2 || w.structureSlots[0x0a0b] != 0 {
		t.Fatal("lethal prefix call detached or cleared another object's slot")
	}
}

func TestStructureDamageBytePairAndSignedWordClamp(t *testing.T) {
	for _, tc := range []struct {
		name      string
		hp, max   uint16
		base, top int32
		want      uint16
		draws     int
	}{
		{"zero spread ignores base", 100, 100, 20, 20, 100, 0},
		{"zero max", 100, 0, 20, 21, 100, 0},
		{"nonpositive result", 100, 100, 1, 2, 100, 1},
		{"byte spread wraps to zero", 100, 100, 10, 266, 100, 0},
		{"negative signed HP clamps", 0xffff, 100, 0, 0, 0, 0},
		{"zero HP stays zero after damage", 0, 100, 10, 11, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := structureAreaWorld(t, []Structure{{ID: 1, Field42: tc.hp, MaxHealth: tc.max, Width: 1, Height: 1, Attach: 1}})
			r := w.rng
			for n := 0; n < tc.draws; n++ {
				r.next()
			}
			w.applyStructureSpellAt(0, 0, SpellRule{DamageMin: tc.base, DamageMax: tc.top}, 0)
			if w.structures[0].Field42 != tc.want || w.rng != r {
				t.Fatalf("HP/RNG=%d/%v want %d/%v", w.structures[0].Field42, w.rng, tc.want, r)
			}
		})
	}
}

func TestBlastVisitOrderAndRingInsetAreDistinct(t *testing.T) {
	want := []uint16{0xffff, 0x00ff, 0x01ff, 0xff00, 0x0000, 0x0100, 0xff01, 0x0001, 0x0101}
	if got := blastCells(0, 0, 1); !reflect.DeepEqual(got, want) {
		t.Fatalf("blast = %x want %x", got, want)
	}
	w := structureAreaWorld(t, nil)
	if got := w.ringStageCells(cellEffect{Key: 0, Spell: 4}, 0); len(got) != 0 {
		t.Fatalf("edge ring admitted %x", got)
	}
	want = []uint16{0x0908, 0x0909, 0x0809}
	if got := w.ringStageCells(cellEffect{Key: 0x0808, Spell: 4}, 0); !reflect.DeepEqual(got, want) {
		t.Fatalf("inset ring=%x want %x", got, want)
	}
}

func TestAreaDirectDamageAndOrdinaryAttachmentStayDistinct(t *testing.T) {
	cells := []uint16{0x0a0a, 0x0a0b, 0x0b0a, 0x0b0b}
	big := Entity{ID: 1, X: 10, Y: 10, HP: 100, MaxHP: 100, TokenSize: 2}
	w := structureAreaWorld(t, nil, big)
	w.applyAreaCells(cellEffect{Mode: areaModeRing}, SpellRule{ID: 9, Damaging: true, DamageMin: 10, DamageMax: 10}, cells)
	if got := w.entities[0].HP; got != 60 {
		t.Fatalf("direct ring HP=%d want 60 (four direct calls)", got)
	}
	w = structureAreaWorld(t, nil, big)
	w.applyAreaCells(cellEffect{Mode: areaModeRing, Power: 14}, SpellRule{ID: 16, EffectKind: EffectProtectionAir, EffectMode: EffectDuration, SpellDuration: 20, EffectMagnitude: 7}, cells)
	if len(w.attached) != 1 {
		t.Fatalf("ordinary effect count=%d want one attachment", len(w.attached))
	}
	if got := w.entities[0].Protection[2]; got != 7 {
		t.Fatalf("protection=%d", got)
	}
	// Untimed ordinary effects are not stored. Repeated reference visits must
	// still reach that effect's own application rather than a universal seen set.
	w = structureAreaWorld(t, nil, big)
	w.applyAreaCells(cellEffect{Mode: areaModeRing}, SpellRule{ID: 30, EffectKind: EffectAbsorption, EffectMagnitude: 2}, cells)
	if len(w.attached) != 0 || w.entities[0].Absorption != 8 {
		t.Fatalf("untimed ordinary: attached=%d absorption=%d", len(w.attached), w.entities[0].Absorption)
	}
}

func TestMultiCellRingNativeContinuationBeforePendingAndAfter(t *testing.T) {
	w := structureAreaWorld(t, []Structure{{ID: 1, Field42: 2000, MaxHealth: 2000, Col: 8, Row: 8, Width: 5, Height: 5, Attach: (1 << 25) - 1}})
	rule := SpellRule{ID: 4, Area: true, Damaging: true, Distribution: 5, DamageMin: 10, DamageMax: 11}
	w.spells = []SpellRule{rule}
	check := func(label string) {
		t.Helper()
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var back World
		if err := back.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
		if back.Hash() != w.Hash() {
			t.Fatalf("%s initial hash", label)
		}
		for tick := 0; tick < 5; tick++ {
			a, b := StepReported(w, nil), StepReported(&back, nil)
			if !reflect.DeepEqual(a, b) || w.Hash() != back.Hash() {
				t.Fatalf("%s tick%d continuation mismatch", label, tick)
			}
		}
		got, err := back.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		want, _ := w.MarshalBinary()
		if !bytes.Equal(got, want) {
			t.Fatalf("%s bytes", label)
		}
		// Rewind only the test fixture after comparing the never-loaded live
		// world with its loaded continuation. This preserves each checkpoint.
		if err := w.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
	}
	check("before")
	if !w.landArea(rule, 0, 0, false, 0, 0, 10, 10, nil) {
		t.Fatal("ring refused")
	}
	if hp := w.structures[0].Field42; hp < 1952 || hp > 1960 {
		t.Fatalf("first shell HP=%d want eight calls (1952..1960)", hp)
	}
	check("pending second stage")
	for n := 0; n < 4; n++ {
		Step(w, nil)
	}
	if len(w.effects) != 0 {
		t.Fatalf("terminal ring retained %+v", w.effects)
	}
	if hp := w.structures[0].Field42; hp < 1880 || hp > 1900 {
		t.Fatalf("two shells HP=%d want twenty calls (1880..1900)", hp)
	}
	check("after")
}

func TestConstructedAreaDamagePairIsNotPowerScaledAgain(t *testing.T) {
	w := structureAreaWorld(t, []Structure{{ID: 1, Field42: 100, MaxHealth: 100, Col: 10, Row: 10, Width: 1, Height: 1, Attach: 1}})
	r := w.rng
	want := uint16(100 - 5 - r.uniform(1))
	w.applyAreaCells(cellEffect{Spell: 4, Mode: areaModeRing, Power: 100, DamageMin: 10, DamageMax: 11}, SpellRule{ID: 4}, []uint16{0x0a0a})
	if w.structures[0].Field42 != want || w.rng != r {
		t.Fatalf("constructed pair HP=%d want%d", w.structures[0].Field42, want)
	}
}

func TestMultiCellAreaRuinCancelsPhysicalPursuitAcrossNativeSave(t *testing.T) {
	a := structureCombatActor()
	a.ID, a.Owner, a.X, a.Y, a.Speed = 4, SelfSlot, 2, 10, 20
	w := structureAreaWorld(t, []Structure{{ID: 4, Field42: 100, MaxHealth: 100, Col: 10, Row: 10, Width: 3, Height: 3, Attach: 511}}, a)
	Step(w, []Command{{Kind: KindAttackStructure, Entity: 4, X: 4}})
	before := w.entities[0]
	if !before.HasAttackTarget || !before.HasTarget || before.Transit == 0 {
		t.Fatalf("missing pursuit: %+v", before)
	}
	w.applyAreaCells(cellEffect{Mode: areaModeRing}, SpellRule{ID: 9, Damaging: true, DamageMin: 150, DamageMax: 151}, []uint16{0x0a0b, 0x0b0b})
	if w.structures[0].Field42 != 0 {
		t.Fatal("multi-cell reference did not ruin target")
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 50; tick++ {
		a, b := StepReported(w, nil), StepReported(&back, nil)
		if !reflect.DeepEqual(a, b) || w.Hash() != back.Hash() {
			t.Fatalf("ruin/pursuit save tick%d", tick)
		}
		actor := w.entities[0]
		if actor.HasAttackTarget || actor.HasTarget || len(w.routes[0]) != 0 || actor.X != before.X || actor.Y != before.Y {
			t.Fatalf("ruin retained pursuit at tick%d: %+v", tick, actor)
		}
		if tick == 0 && actor.Transit != before.Transit-1 {
			t.Fatal("ruin discarded current crossing payment")
		}
	}
}
