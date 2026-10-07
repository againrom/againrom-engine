package sim

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"testing"
)

// Independent outer-format adapters. Older literal byte fixtures and their
// hashes stay unchanged; only the new version byte and empty span are added.
func widenedNativeStridePin(old []byte) []byte {
	out := append(append([]byte(nil), old...), 0, 0, 0, 0)
	out[0] = 81
	return widenedSavedMotionPin(out)
}

func strippedNativeStridePin(form []byte) []byte {
	out := strippedSavedMotionPin(form)
	if len(out) != 0 && out[0] >= 81 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-span]
		out[0] = 80
	}
	return out
}

func nativeStrideActor1115() Entity {
	return nativeStrideActorForTest()
}

func nativeStrideActorForTest() Entity {
	return Entity{ID: 7, X: 5, Y: 4, HP: 100, MaxHP: 100, Speed: 16,
		Facing: 64, DesiredFacing: 64, Transit: 15, TransitTotal: 16,
		Stride: NativeStride{Present: true, FromX: 4, FromY: 4, ToX: 5, ToY: 4,
			Rate: 16, StepX: 16, Direction: 2}}
}

func TestNativeStride1115EveryDirectionCapturesActualRateAndLastPayment(t *testing.T) {
	// Literal compass deltas and the independent worked rate16 values:
	// straight axis16/total16, diagonal axis11/total24.
	for direction, delta := range [][2]int32{{0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}} {
		for _, speed := range []int32{1, 16, 1000} {
			w := mustWorld(t, 1115, Bounds{20, 20}, []Entity{{ID: 7, X: 8, Y: 8, HP: 100, MaxHP: 100, Speed: speed}})
			Step(w, []Command{{Entity: 7, X: 8 + delta[0], Y: 8 + delta[1]}})
			rate, axis, total := uint8(16), int8(16), uint16(16)
			if speed == 1 {
				rate, axis, total = 1, 1, 256
			} else if speed == 1000 {
				rate, axis, total = 63, 63, 5
			}
			if delta[0] != 0 && delta[1] != 0 && speed != 1 {
				axis, total = 11, 24
				if speed == 1000 {
					axis, total = 44, 6
				}
			}
			want := NativeStride{Present: true, FromX: 8, FromY: 8, ToX: 8 + delta[0], ToY: 8 + delta[1],
				Rate: rate, StepX: int8(delta[0]) * axis, StepY: int8(delta[1]) * axis, Direction: uint8(direction)}
			for paid := uint16(1); paid <= total; paid++ {
				e := w.Entities()[0]
				if e.Stride != want || e.X != want.ToX || e.Y != want.ToY || e.TransitTotal != total || e.Transit != total-paid || e.HasTarget {
					t.Fatalf("direction%d speed%d paid%d: stride=%+v transit%d/%d target%t", direction, speed, paid, e.Stride, e.Transit, e.TransitTotal, e.HasTarget)
				}
				var cold World
				if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil || cold.Hash() != w.Hash() {
					t.Fatalf("payment%d native roundtrip: %v", paid, err)
				}
				if paid != total {
					Step(w, nil)
				}
			}
			Step(w, nil)
			if w.Entities()[0].Stride != want {
				t.Fatal("settled tick erased the last rated stride")
			}
		}
	}
}

func TestNativeStride1115IndependentWireAndHistoricalPin(t *testing.T) {
	w := mustWorld(t, 1115, Bounds{8, 8}, []Entity{nativeStrideActor1115()})
	form := strippedSavedMotionPin(mustMarshal(t, w))
	want := []byte{
		1, 0, 0, 0, // sparse record count
		7, 0, 0, 0, // actor ID
		4, 0, 0, 0, 4, 0, 0, 0, // accepted origin
		5, 0, 0, 0, 4, 0, 0, 0, // accepted destination
		16, 16, 0, 2, // rate, signed axis steps, native octant
		28, 0, 0, 0, // payload span, excluding footer
	}
	if form[0] != 81 || !bytes.Equal(form[len(form)-len(want):], want) {
		t.Fatal("native stride is not the independently transcribed wire record")
	}
	for _, tc := range []struct {
		form []byte
		hash uint64
	}{{pinBytes, 0x299d72ec65c850c3}, {rtfBytes, 0xe8f1f827f4cec622}} {
		if fnv1a(strippedNativeStridePin(tc.form)) != tc.hash {
			t.Fatal("form81 changed the immutable form80 historical pin")
		}
	}
}

func TestNativeStride1115FaultsAndIntegerBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*Entity)
	}{
		{"absent-residue", func(e *Entity) { e.Stride.Present = false }},
		{"position", func(e *Entity) { e.X++ }},
		{"same-cell", func(e *Entity) { e.Stride.FromX = e.Stride.ToX }},
		{"distant", func(e *Entity) { e.Stride.FromX-- }},
		{"wrapped-delta", func(e *Entity) { e.X, e.Stride.ToX, e.Stride.FromX = math.MinInt32, math.MinInt32, math.MaxInt32 }},
		{"direction-range", func(e *Entity) { e.Stride.Direction = 8 }},
		{"direction-mismatch", func(e *Entity) { e.Stride.Direction = 3 }},
		{"zero-rate", func(e *Entity) { e.Stride.Rate = 0 }},
		{"large-rate", func(e *Entity) { e.Stride.Rate = 64 }},
		{"axis-sign", func(e *Entity) { e.Stride.StepX = -16 }},
		{"axis-magnitude", func(e *Entity) { e.Stride.StepX = 15 }},
		{"axis-extra", func(e *Entity) { e.Stride.StepY = 1 }},
		{"no-total", func(e *Entity) { e.Transit, e.TransitTotal = 0, 0 }},
		{"different-total", func(e *Entity) { e.TransitTotal = 17 }},
		{"equal-remainder", func(e *Entity) { e.Transit = 16 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := nativeStrideActor1115()
			tc.edit(&e)
			if _, err := NewWorld(1115, Bounds{8, 8}, ModeCanonical, nil, []Entity{e}); err == nil {
				t.Fatal("constructor accepted malformed stride")
			}
			w := mustWorld(t, 1115, Bounds{8, 8}, []Entity{nativeStrideActor1115()})
			tc.edit(&w.entities[0])
			if _, err := w.MarshalBinary(); err == nil {
				t.Fatal("encoder accepted malformed stride")
			}
		})
	}
	for _, x := range []int32{math.MinInt32, math.MaxInt32 - 1} {
		e := nativeStrideActor1115()
		e.X, e.Stride.FromX, e.Stride.ToX = x+1, x, x+1
		w := mustWorld(t, 1115, Bounds{8, 8}, []Entity{e})
		var cold World
		if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil || cold.Entities()[0].Stride != e.Stride {
			t.Fatal("native signed-coordinate boundary was narrowed", err)
		}
	}
}

func TestNativeStride1115MalformedWireIsAtomic(t *testing.T) {
	a, b := nativeStrideActor1115(), nativeStrideActor1115()
	b.ID, b.Y, b.Stride.FromY, b.Stride.ToY = 17, 6, 6, 6
	w := mustWorld(t, 1115, Bounds{8, 8}, []Entity{a, b})
	valid := strippedSavedMotionPin(mustMarshal(t, w))
	start := len(valid) - 56 // count4, two literal24-byte records, footer4
	record := start + 4
	// a's id is 7, b's is 17 (above); no originalDead and no script, so
	// bounded reconstruction's floor is the higher one plus one — the value
	// widenedSavedMotionPin's own chain does not reach, stopping at form93 to
	// keep areaHeaderDigest1164's frozen digest untouched.
	current := func(b []byte) []byte { return widenedEntityIDFloorPin(widenedSavedMotionPin(b), 18) }
	for _, tc := range []struct {
		name string
		edit func([]byte) []byte
	}{
		{"huge-span", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[len(b)-4:], math.MaxUint32); return b }},
		{"huge-count", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start:], math.MaxUint32); return b }},
		{"zero-count", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start:], 0); return b }},
		{"count-over-actors", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start:], 3); return b }},
		{"count-under-span", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start:], 1); return b }},
		{"duplicate-ID", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[record+24:], 7); return b }},
		{"descending-ID", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[record+24:], 1); return b }},
		{"missing-ID", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[record+24:], 99); return b }},
		{"late-rate", func(b []byte) []byte { b[record+24+20] = 0; return b }},
		{"late-direction", func(b []byte) []byte { b[record+24+23] = 8; return b }},
		{"late-axis", func(b []byte) []byte { b[record+24+21] = 255; return b }},
		{"late-position", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[record+24+12:], 6); return b }},
		{"truncated-footer", func(b []byte) []byte { return b[:len(b)-1] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := tc.edit(bytes.Clone(valid))
			before := mustMarshal(t, w)
			if err := w.UnmarshalBinary(current(bad)); err == nil || !bytes.Equal(before, mustMarshal(t, w)) {
				t.Fatal("malformed stride partially adopted", err)
			}
		})
	}
	for n := 1; n < 52; n++ {
		bad := append(bytes.Clone(valid[:start]), valid[start:start+n]...)
		bad = binary.LittleEndian.AppendUint32(bad, uint32(n))
		if err := w.UnmarshalBinary(current(bad)); err == nil || !bytes.Equal(current(valid), mustMarshal(t, w)) {
			t.Fatalf("payload truncation%d partially adopted: %v", n, err)
		}
	}
	var cold World
	if err := cold.UnmarshalBinary(current(valid)); err != nil {
		t.Fatal(err)
	}
	clear(valid)
	if !reflect.DeepEqual(cold.Entities(), w.Entities()) {
		t.Fatal("decoded stride aliases input bytes")
	}
}

func TestNativeStride1115OrderAndRateChangesPreserveUntilNextAcceptedStep(t *testing.T) {
	w := mustWorld(t, 1115, Bounds{20, 20}, []Entity{{ID: 7, X: 4, Y: 4, HP: 100, MaxHP: 100, Speed: 16}})
	Step(w, []Command{{Entity: 7, X: 12, Y: 4}})
	want := NativeStride{Present: true, FromX: 4, FromY: 4, ToX: 5, ToY: 4, Rate: 16, StepX: 16, Direction: 2}
	if w.Entities()[0].Stride != want {
		t.Fatal("fixture did not capture the independent east stride")
	}
	if _, ok := w.applyEffectDelta(0, EffectSpeed, 30); !ok {
		t.Fatal("speed mutation did not apply")
	}
	Step(w, []Command{{Entity: 7, X: 5, Y: 1}})
	if e := w.Entities()[0]; e.Stride != want || e.Transit != 14 || e.TargetX != 5 || e.TargetY != 1 {
		t.Fatal("replacement command/rate changed the accepted crossing", e.Stride)
	}
	for w.Entities()[0].Transit != 0 {
		Step(w, nil)
	}
	if w.Entities()[0].Stride != want {
		t.Fatal("last payment erased accepted rate")
	}
	Step(w, nil)
	next := NativeStride{Present: true, FromX: 5, FromY: 4, ToX: 5, ToY: 3, Rate: 46, StepY: -46, Direction: 0}
	if w.Entities()[0].Stride != next {
		t.Fatal("next accepted stride retained previous inputs", w.Entities()[0].Stride)
	}
	// A later unrated step keeps the historical coarse total but cannot claim
	// that old total/rate describes its own instant step.
	for w.Entities()[0].Transit != 0 {
		Step(w, nil)
	}
	w.entities[0].Speed = 0
	previousTotal := w.entities[0].TransitTotal
	Step(w, []Command{{Entity: 7, X: 6, Y: 3}})
	if e := w.Entities()[0]; e.Stride != (NativeStride{}) || e.X != 6 || e.Transit != 0 || e.TransitTotal != previousTotal {
		t.Fatal("unrated step changed coarse behavior or retained false provenance", e.Stride, e.Transit, e.TransitTotal)
	}
}
