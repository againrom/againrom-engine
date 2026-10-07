package spr16

import (
	"encoding/binary"
	"strings"
	"testing"
)

// sidecar builds an advance-table stream from the values it should hold.
func sidecar(values ...uint32) []byte {
	b := make([]byte, len(values)*4)
	for i, v := range values {
		binary.LittleEndian.PutUint32(b[i*4:], v)
	}
	return b
}

// AC-1: a well-formed table yields exactly its entries, in file order, at every
// size a shipped font uses — the empty one, a single entry, and font3's 64.
func TestAdvancesWellFormed(t *testing.T) {
	if got, err := Advances(nil); err != nil || len(got) != 0 {
		t.Fatalf("Advances(nil) = %v, %v; want 0 entries, no error", got, err)
	}
	if got, err := Advances([]byte{}); err != nil || len(got) != 0 {
		t.Fatalf("Advances(empty) = %v, %v; want 0 entries, no error", got, err)
	}

	one, err := Advances(sidecar(7))
	if err != nil {
		t.Fatalf("Advances(one entry): %v", err)
	}
	if len(one) != 1 || one[0] != 7 {
		t.Fatalf("Advances(one entry) = %v; want [7]", one)
	}

	// 64 entries, each distinct, so a decoder that reversed or repeated the
	// stream could not pass.
	vals := make([]uint32, 64)
	for i := range vals {
		vals[i] = uint32(i * 3)
	}
	got, err := Advances(sidecar(vals...))
	if err != nil {
		t.Fatalf("Advances(64 entries): %v", err)
	}
	if len(got) != 64 {
		t.Fatalf("Advances(64 entries) yielded %d entries, want 64", len(got))
	}
	for i, v := range vals {
		if got[i] != int(v) {
			t.Fatalf("entry %d = %d, want %d", i, got[i], v)
		}
	}
}

// AC-1: a length that is not a whole number of entries is refused, with no
// partial result — the shape of a truncated read, which must not be reported as
// a shorter font.
func TestAdvancesRaggedLength(t *testing.T) {
	for _, n := range []int{1, 2, 3, 5, 7, 9} {
		got, err := Advances(make([]byte, n))
		if err == nil {
			t.Fatalf("Advances(%d bytes) = %v; want an error", n, got)
		}
		if got != nil {
			t.Fatalf("Advances(%d bytes) returned %d entries beside its error; want none", n, len(got))
		}
		if !strings.Contains(err.Error(), "whole number") {
			t.Fatalf("Advances(%d bytes) error %q does not name the quantity it refused", n, err)
		}
	}
}

// AC-2: the two caps, each refused and each naming its own quantity. Both are
// the container's own numbers, so a table this package accepts can always belong
// to an atlas this package accepts.
func TestAdvancesCaps(t *testing.T) {
	tooMany := make([]byte, (maxFrameCount+1)*4)
	got, err := Advances(tooMany)
	if err == nil {
		t.Fatalf("Advances(%d entries) = %v; want an error", maxFrameCount+1, got)
	}
	if got != nil {
		t.Fatalf("over-count table returned %d entries beside its error", len(got))
	}
	if !strings.Contains(err.Error(), "advance table") {
		t.Fatalf("over-count error %q does not name the table", err)
	}

	// Exactly at the cap is accepted: the refusal is > and not >=.
	if _, err := Advances(make([]byte, maxFrameCount*4)); err != nil {
		t.Fatalf("Advances(%d entries) refused at exactly the cap: %v", maxFrameCount, err)
	}

	// An entry over the dimension cap, in the middle of an otherwise sound
	// table, so the check cannot be passing by only looking at entry 0.
	got, err = Advances(sidecar(3, 4, maxDimension+1, 5))
	if err == nil {
		t.Fatalf("Advances(over-cap entry) = %v; want an error", got)
	}
	if got != nil {
		t.Fatalf("over-cap entry returned %d entries beside its error", len(got))
	}
	if !strings.Contains(err.Error(), "advance") || !strings.Contains(err.Error(), "entry 2") {
		t.Fatalf("over-cap error %q names neither the advance nor which entry", err)
	}

	// Exactly at the dimension cap is accepted, for the same reason.
	if _, err := Advances(sidecar(maxDimension)); err != nil {
		t.Fatalf("Advances(entry at exactly the cap) refused: %v", err)
	}
}

// A sidecar with the shape the shipped fonts have: 224 entries, 896 bytes, every
// value small. The figure is the one a loader checks against the atlas's record
// count, so it is pinned here rather than left to the loader's own arithmetic.
func TestAdvancesShippedShape(t *testing.T) {
	const records = 224
	got, err := Advances(make([]byte, records*4))
	if err != nil {
		t.Fatalf("Advances(896 bytes): %v", err)
	}
	if len(got) != records {
		t.Fatalf("896 bytes yielded %d entries, want %d", len(got), records)
	}
}
