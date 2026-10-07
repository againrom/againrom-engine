package sim

import (
	"encoding/binary"
	"testing"
)

func requireHumanMovementLegacyDigest(t *testing.T, w *World, want uint64) {
	t.Helper()
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	old := strippedWorldOfHumanMovement(form, w)
	if got := fnv1a(old); got != want {
		t.Fatalf("version68 digest changed: %016x want %016x", got, want)
	}
}

func TestRetainedHumanMovementRateAndInvalidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  int16
		want int32
	}{
		{"load before capacity modifier", 20, 20}, {"negative signed word", -12, 1},
		{"zero still rated", 0, 1}, {"high word clamps", 32767, 63}, {"wrapped word", -32768, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := srWorld(t, nil, nil, []Entity{{ID: 1, X: 1, Y: 1, Speed: 21, Capacity: 1411}})
			if !w.SetHumanMovement(1, tc.raw, 490) {
				t.Fatal("projection refused")
			}
			rate, _, _, ok := w.StepRate(1, 2, 1)
			if !ok || rate != tc.want {
				t.Fatalf("rate=%d ok=%v want %d", rate, ok, tc.want)
			}
			if !rated(w.entities[0]) {
				t.Fatal("signed source became unpaced")
			}
			w.entities[0].GroupSpeed = 18
			if raw, ok := w.entities[0].RetainedHumanSpeed(); !ok || raw != tc.raw {
				t.Fatal("group override changed the own Human statistic")
			}
			if moverSpeed(w.entities[0]) != 18 {
				t.Fatal("group override bypassed")
			}
			w.entities[0].clearGroupSpeed()
			if moverSpeed(w.entities[0]) != int32(tc.raw) {
				t.Fatal("group exit lost retained speed")
			}
			w.recomputeLoad(0)
			if _, ok := w.entities[0].RetainedHumanSpeed(); ok {
				t.Fatal("load mutation kept the old own-stat projection")
			}
			if w.entities[0].HumanMovement.Present || moverSpeed(w.entities[0]) != 21 {
				t.Fatal("load mutation kept stale context")
			}
			w.SetHumanMovement(1, tc.raw, 490)
			w.SetDerived(1, DerivedBlock{Speed: 30, Capacity: 1000})
			if w.entities[0].HumanMovement.Present {
				t.Fatal("rearm kept context")
			}
			w.SetHumanMovement(1, tc.raw, 490)
			e, _, _ := effectLanding(w.entities[0], EffectSpeed, 2)
			if e.HumanMovement.Present {
				t.Fatal("speed effect kept context")
			}
		})
	}
}

func TestRetainedHumanMovementMidMoveForm(t *testing.T) {
	w := srWorld(t, nil, nil, []Entity{{ID: 1, X: 1, Y: 1, Speed: 21, Capacity: 1411}})
	w.SetHumanMovement(1, -12, 490)
	Step(w, []Command{{Entity: 1, X: 3, Y: 1}})
	if w.entities[0].TransitTotal == 0 {
		t.Fatal("negative source did not enter rated transit")
	}
	form, _ := w.MarshalBinary()
	var fresh World
	if err := fresh.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 80; i++ {
		Step(w, nil)
		Step(&fresh, nil)
		if w.Hash() != fresh.Hash() {
			t.Fatalf("diverged at step %d", i)
		}
	}
	// Independent wire layout and injective absence. Context differences must
	// reach the hash even when the terrain clamp gives them the same rate.
	base := 34 + 3*int(gridCells(w.bounds))
	form, _ = w.MarshalBinary()
	if form[base+326] != 1 || int16(binary.LittleEndian.Uint16(form[base+327:])) != -12 {
		t.Fatal("missing signed wire context")
	}
	other := append([]byte(nil), form...)
	binary.LittleEndian.PutUint16(other[base+327:], 65525)
	var changed World
	if err := changed.UnmarshalBinary(other); err != nil {
		t.Fatal(err)
	}
	if changed.Hash() == w.Hash() {
		t.Fatal("raw source absent from digest")
	}
	for _, at := range []int{326, 329, 333, 337} {
		bad := append([]byte(nil), form...)
		bad[base+at] ^= 2
		before := fresh.Hash()
		if err := fresh.UnmarshalBinary(bad); err == nil || fresh.Hash() != before {
			t.Fatalf("invalid context +%d was not transactional", at)
		}
	}
}
