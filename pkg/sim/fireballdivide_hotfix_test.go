package sim

import (
	"bytes"
	"reflect"
	"testing"
)

func TestFireballDividesEachByteComponent(t *testing.T) {
	full := []uint16{0x0a0a, 0x0b0a, 0x0a0b, 0x0b0b}
	for _, tc := range []struct {
		name      string
		seed      uint64
		side      uint8
		min, max  int32
		power     uint16
		cells     []uint16
		wantHP    int32
		wantDraws int
	}{
		{"review regression", 20, 2, 11, 17, 0, full, 992, 4},
		{"nonzero spread rolls", 0, 2, 11, 17, 0, full, 990, 4},
		{"half coverage", 20, 2, 11, 17, 0, []uint16{0x0a09, 0x0a0a, 0x0b0a}, 996, 2},
		{"quarter coverage", 20, 2, 11, 17, 0, []uint16{0x0909, 0x0a0a}, 998, 1},
		{"outside footprint", 20, 2, 11, 17, 0, []uint16{0x090a, 0x0a0c}, 1000, 0},
		{"remainders do not carry", 0, 2, 3, 4, 0, full, 1000, 4},
		{"exact component division", 20, 2, 8, 16, 0, full, 991, 4},
		{"three cell side", 0, 3, 17, 27, 0, full, 994, 4},
		{"one cell side", 20, 1, 11, 17, 0, full[:1], 988, 1},
		{"legacy zero side", 20, 0, 11, 17, 0, full[:1], 988, 1},
		{"base narrowed before divide", 20, 2, 267, 273, 0, full, 992, 4},
		{"spread narrowed before divide", 20, 2, 11, 273, 0, full, 992, 4},
		{"power before divide", 20, 2, 11, 17, 30, full, 978, 4},
		{"power before byte narrowing", 20, 2, 100, 200, 60, full, 947, 4},
		{"footprint exceeds each byte", 0, 16, 255, 510, 0, full, 1000, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := Entity{ID: 1, X: 10, Y: 10, HP: 1000, MaxHP: 1000, TokenSize: tc.side}
			w, err := NewSpelledWorld(tc.seed, Bounds{Width: 40, Height: 40}, ModeCanonical, nil, []Entity{target}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			// MAGIC-FIREDIV-047 divides the copied base/spread bytes separately.
			// Literal HP expectations use independently reduced SplitMix words,
			// not spellDamage, the area walker, or the production RNG helper.
			// At seed 20 the first four rolls with spread 1 are all zero;
			// at seed 0 they are 1, 0, 0, 1. Zero spread still takes one draw.
			w.applyAreaCells(cellEffect{Mode: areaModeBlast, Power: tc.power},
				SpellRule{ID: 2, Damaging: true, DamageMin: tc.min, DamageMax: tc.max}, tc.cells)
			if got := w.entities[0].HP; got != tc.wantHP {
				t.Errorf("Fire Ball HP = %d, want %d", got, tc.wantHP)
			}
			wantState := tc.seed + uint64(tc.wantDraws)*uint64(0x9e3779b97f4a7c15)
			if w.rng.state != wantState {
				t.Errorf("RNG state = %x, want %x (%d draws)", w.rng.state, wantState, tc.wantDraws)
			}
		})
	}
}

func TestFireballComponentDivisionNativeCastContinuation(t *testing.T) {
	caster := effectMage(1, 6, 10, 1<<2)
	target := Entity{ID: 2, X: 10, Y: 10, HP: 1000, MaxHP: 1000, TokenSize: 2}
	rule := SpellRule{ID: 2, Area: true, Radius: 1, School: 1, MaxRange: 6,
		ManaCost: 1, Damaging: true, DamageMin: 11, DamageMax: 17}
	w := hlWorld(t, 20, Relations{}, []SpellRule{rule}, caster, target)
	var restored []*World
	checkpoint := func() {
		t.Helper()
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		back := new(World)
		if err := back.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
		if back.Hash() != w.Hash() {
			t.Fatal("native checkpoint changed hash")
		}
		restored = append(restored, back)
	}
	checkpoint() // Before admission.
	releasedAt := -1
	for tick := 0; tick < 256; tick++ {
		var commands []Command
		if tick == 0 {
			commands = []Command{{Kind: KindCastAt, Entity: 1, X: 10, Y: 10, Spell: 2}}
		}
		want := StepReported(w, commands)
		for i, back := range restored {
			if got := StepReported(back, commands); !reflect.DeepEqual(got, want) || back.Hash() != w.Hash() {
				t.Fatalf("checkpoint %d diverged at tick %d", i, tick)
			}
		}
		if tick == 0 {
			if len(w.bookCasts) != 1 || w.entities[1].HP != 1000 {
				t.Fatal("cast did not retain a pre-damage windup")
			}
			checkpoint() // Pending cast must use the corrected rule after loading.
		}
		if releasedAt < 0 && w.entities[1].HP != 1000 {
			if got := w.entities[1].HP; got != 992 {
				t.Fatalf("book-cast Fire Ball HP = %d, want 992", got)
			}
			releasedAt = tick
			checkpoint() // Corrected HP and advanced RNG must persist together.
		}
		if releasedAt >= 0 && tick == releasedAt+20 {
			break
		}
	}
	if releasedAt < 0 || len(restored) != 3 {
		t.Fatal("cast did not reach all three native checkpoints")
	}
	want, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	for i, back := range restored {
		got, err := back.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("checkpoint %d final native bytes differ", i)
		}
	}
}
