package sim

import (
	"encoding/binary"
	"testing"
)

// sourceSightWorld is a world of either sight class holding entity 1 with a
// source basis whose whole sight is cells. Class 2 derives its sight from a
// base of cells plus the signed modifier word, as the human derive does.
func sourceSightWorld(t *testing.T, class uint8, cells uint16) *World {
	t.Helper()
	w := bookWorld(t, Entity{ID: 1, HP: 50, MaxHP: 100, Humanoid: class == 2, ScanRange: uint8(cells)}, PlainItem(0xe01))
	s := SourceActor{Class: class, Fighter: true, Stats: [14]uint16{10, 20, 10, 10, 18, 0, 0, 101, 50, 100, 100, 0, 0, 50}}
	if class == 1 {
		s.Sight = cells << 8
	}
	if err := w.RestoreActorLoad(1, ActorLoadSnapshot{Inventory: ActorLoad{Present: true, ContainerPresent: true, Source: s}, Capacity: 101, Speed: 18, Movement: HumanMovement{Present: true, RawSpeed: 18, NativeSpeed: 18, Capacity: 101}}); err != nil {
		t.Fatal(err)
	}
	w.entities[0].ScanRange = uint8(cells)
	w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) {
		if s.Class == 2 {
			s.Sight = cells<<8 + binary.LittleEndian.Uint16(s.Modifier[16:])
		}
		return s, nil
	})
	return w
}

// A Light or Darkness cut on a unit with a source basis stops at no sight,
// not at a wrapped sight, and the whole sight returns when the effect ends,
// the same as on a unit with no basis. Both games.
func TestSightCutOnASourceBasisStopsAtZeroAndReturns(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		for _, class := range []uint8{1, 2} {
			rule := darkSightRow(second, class == 2)
			w := sourceSightWorld(t, class, 3)
			// The unit takes the derive once, as a loaded unit has.
			if !w.attachEffect(1, 1, rule, EffectScanRange, 150, 4, EffectDuration) {
				t.Fatalf("second %v class %d: the cut did not attach", second, class)
			}
			if got := w.entities[0].ScanRange; got != 0 {
				t.Errorf("second %v class %d: sight under the cut %d, want 0", second, class, got)
			}
			if !w.attachEffect(1, 1, rule, EffectScanRange, 150, 4, EffectDuration) {
				t.Fatalf("second %v class %d: the renewal did not attach", second, class)
			}
			if got := w.entities[0].ScanRange; got != 0 {
				t.Errorf("second %v class %d: sight under the renewed cut %d, want 0", second, class, got)
			}
			for k := 0; k < 8 && len(w.attached) != 0; k++ {
				w.stepAttachedEffects()
			}
			if len(w.attached) != 0 {
				t.Fatalf("second %v class %d: the effect did not end", second, class)
			}
			if got := w.entities[0].ScanRange; got != 3 {
				t.Errorf("second %v class %d: sight after the effect %d, want 3", second, class, got)
			}
		}
	}
}
