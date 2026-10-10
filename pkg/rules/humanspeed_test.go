package rules

import "testing"

// The expected words are written out from SAV-1116's step order: the penalty
// and its floor act on the base, the modifier is added afterwards as a word,
// and a negative sum clears the modifier rather than the speed.
func TestHumanSpeedAppliesTheOverloadBeforeTheModifier(t *testing.T) {
	for _, tc := range []struct {
		name                           string
		base, modifier, load, capacity int16
		speed, kept                    int16
	}{
		{"zero load keeps base plus modifier", 20, 5, 0, 301, 25, 5},
		{"one below capacity is free", 20, 5, 300, 301, 25, 5},
		{"at capacity costs one before the modifier", 20, 5, 301, 301, 24, 5},
		{"the floor binds before a positive modifier", 20, 5, 6020, 301, 11, 5},
		{"the floor raises a slow base", 3, 2, 301, 301, 8, 2},
		{"a negative sum keeps the speed and clears the modifier", 20, -15, 6020, 301, -9, 0},
		{"a zero sum keeps the modifier", 20, -6, 6020, 301, 0, -6},
		{"a negative modifier above the floor stays", 20, -15, 0, 301, 5, -15},
		{"the word add wraps", 32767, 1, 0, 301, -32768, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			speed, kept, ok := HumanSpeed(tc.base, tc.modifier, tc.load, tc.capacity)
			if !ok || speed != tc.speed || kept != tc.kept {
				t.Fatalf("HumanSpeed(%d, %d, %d, %d) = %d, %d, %v; want %d, %d, true",
					tc.base, tc.modifier, tc.load, tc.capacity, speed, kept, ok, tc.speed, tc.kept)
			}
		})
	}
}

func TestHumanSpeedRefusesAnOverloadedZeroCapacity(t *testing.T) {
	if _, _, ok := HumanSpeed(20, 5, 0, 0); ok {
		t.Fatal("an overloaded zero capacity was divided")
	}
	if speed, kept, ok := HumanSpeed(20, 5, -1, 0); !ok || speed != 25 || kept != 5 {
		t.Fatal("a load below a zero capacity was refused", speed, kept, ok)
	}
}
