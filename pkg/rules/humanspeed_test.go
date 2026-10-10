package rules

import "testing"

// The expected words are written out from SAV-1116's step order: the penalty
// and its floor act on the base, the modifier is added afterwards as a word,
// and a negative sum clears the modifier rather than the speed.
func TestHumanSpeedAppliesTheOverloadBeforeTheModifier(t *testing.T) {
	for _, tc := range []struct {
		name           string
		base, modifier int16
		load, capacity int32
		speed, kept    int16
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
		{"a load above a word is compared whole", 20, 5, 40000, 301, 11, 5},
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

func TestNativeHumanSpeedReadsTheUnencumberedSum(t *testing.T) {
	for _, tc := range []struct {
		name                            string
		speed, modifier, load, capacity int32
		word, kept                      int32
	}{
		{"overload under haste", 19, 4, 4680, 261, 10, 4},
		{"a negative sum clears the modifier", 6, -9, 4680, 261, -3, 0},
		{"no rate at zero speed", 0, 3, 4680, 261, 0, 3},
		{"no capacity stated", 15, 2, 4680, 0, 15, 2},
	} {
		if word, kept := NativeHumanSpeed(tc.speed, tc.modifier, tc.load, tc.capacity); word != tc.word || kept != tc.kept {
			t.Errorf("%s: NativeHumanSpeed = %d, %d; want %d, %d", tc.name, word, kept, tc.word, tc.kept)
		}
	}
}
