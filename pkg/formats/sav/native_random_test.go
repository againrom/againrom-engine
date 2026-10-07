package sav

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestNativeRandomLeafCurrentStateAndAbsence(t *testing.T) {
	var state DocumentStateData
	if _, present, err := NativeRandomState(state); err != nil || present {
		t.Fatalf("legacy absence = %v, %v", present, err)
	}
	for _, want := range []uint64{0, 0xffffffffffffffff, 0x87d4fba1d0c84897, 0x1020304050607080} {
		if err := SetNativeRandomState(&state, want); err != nil {
			t.Fatal(err)
		}
		got, present, err := NativeRandomState(state)
		if err != nil || !present || got != want || len(state.ValueRecords) != 1 {
			t.Fatalf("native stream = %#x, %v, %v", got, present, err)
		}
	}
}

func TestNativeRandomLeafOriginalShapeAndStrictAdmission(t *testing.T) {
	for _, tc := range []struct {
		name string
		leaf literalStateLeaf
		bad  bool
	}{
		{"current", literalStateLeaf{name: "AgainromRng", kind: 6, data: literalStateWords(1, 0xd0c84897, 0x87d4fba1)}, false},
		{"wrong type", literalStateLeaf{name: "AgainromRng", kind: 2, word: 1}, true},
		{"wrong version", literalStateLeaf{name: "AgainromRng", kind: 6, data: literalStateWords(2, 3, 4)}, true},
		{"short", literalStateLeaf{name: "AgainromRng", kind: 6, data: literalStateWords(1, 3)}, true},
		{"long", literalStateLeaf{name: "AgainromRng", kind: 6, data: literalStateWords(1, 3, 4, 5)}, true},
		{"case alias", literalStateLeaf{name: "againromrng", kind: 6, data: literalStateWords(1, 3, 4)}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dirs := literalWorldDirectories()
			dirs[1].leaves = append(dirs[1].leaves, tc.leaf)
			state, err := parseWorldState(literalStateStore(dirs))
			if tc.bad {
				if err == nil {
					t.Fatal("malformed native RNG accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			out, err := serializeWorldState(state)
			if err != nil {
				t.Fatal(err)
			}
			// Decode raw directory indices, independent of shape production.
			n := int(binary.LittleEndian.Uint32(out[8:]))
			for i := 0; i < n; i++ {
				r := out[24+32*i : 24+32*(i+1)]
				if string(bytes.TrimRight(r[16:], "\x00")) != "CurrentState" {
					continue
				}
				start := int(binary.LittleEndian.Uint32(r[4:]))
				first := out[24+32*start : 24+32*(start+1)]
				if binary.LittleEndian.Uint32(r[8:]) != 2 || string(bytes.TrimRight(first[16:], "\x00")) != "AgainromRng" || first[31] != 0 {
					t.Fatal("extension broke the name width or case-sensitive sorted order")
				}
			}
			if _, err := parseWorldState(out); err != nil {
				t.Fatal(err)
			}
			dirs[1].leaves = append(dirs[1].leaves, tc.leaf)
			if _, err := parseWorldState(literalStateStore(dirs)); err == nil {
				t.Fatal("duplicate native RNG accepted")
			}
		})
	}
}
