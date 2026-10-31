package rules

import "testing"

func TestSightWordInReportsRange(t *testing.T) {
	cases := []struct {
		name                 string
		mind, reaction, cell int32
		word                 uint16
		ok                   bool
		clamped              uint16
	}{
		{"fraction carried, cells replaced", 37, 26, 3, 901, true, 901},
		{"whole cells only", 0, 0, 5, 1280, true, 1280},
		{"one cell, no stats", 0, 0, 1, 256, true, 256},
		{"top of range", 0, 0, 255, 65280, true, 65280},
		{"zero word", 0, 0, 0, 0, false, 0},
		{"negative cells", 0, 0, -1, 0, false, 0},
		{"cells past a byte", 0, 0, 300, 0, false, 255 << 8},
		{"very negative cells", 37, 26, -100, 0, false, 0},
		{"fraction at the top byte", 37, 26, 255, 65413, true, 65413},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			word, ok := SightWordIn(tc.mind, tc.reaction, tc.cell)
			if word != tc.word || ok != tc.ok {
				t.Fatalf("SightWordIn(%d, %d, %d) = %d, %v; want %d, %v",
					tc.mind, tc.reaction, tc.cell, word, ok, tc.word, tc.ok)
			}
			if got := SightWord(tc.mind, tc.reaction, tc.cell); got != tc.clamped {
				t.Fatalf("SightWord(%d, %d, %d) = %d, want %d", tc.mind, tc.reaction, tc.cell, got, tc.clamped)
			}
		})
	}
}
