package sav

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// The explored-terrain record and the tail split it depends on.
//
// EVERY FIXTURE IS BUILT HERE FROM THE DOCUMENTED FRAMING (golden rule 2). The
// store is a registry in the .reg format's framing — a 0x18-byte header, a node
// table of 0x20-byte records, a u32 pool length, then the pool — and the
// builders below write exactly that. Nothing in this file reads a game install.

// regNode writes one 0x20-byte node record: reserved, data, size, kind, name.
func regNode(data, size, kind uint32, name string) []byte {
	rec := make([]byte, 0x20)
	binary.LittleEndian.PutUint32(rec[0x04:], data)
	binary.LittleEndian.PutUint32(rec[0x08:], size)
	binary.LittleEndian.PutUint32(rec[0x0c:], kind)
	copy(rec[0x10:0x20], name)
	return rec
}

// fogStore builds a state store holding one section `Fog` with the two leaves,
// and returns it followed by trail — which is the three-region shape a real
// save's uncompressed tail has (SAV-TAILEXT-062).
//
// The node table is: 0 the Fog directory, 1 FirstState, 2 Data. The root's own
// child range is [0,1), so the directory is the one top-level entry.
func fogStore(firstState int32, runs []int32, trail []byte) []byte {
	pool := make([]byte, 4*len(runs))
	for i, r := range runs {
		binary.LittleEndian.PutUint32(pool[4*i:], uint32(r))
	}
	head := make([]byte, 0x18)
	binary.LittleEndian.PutUint32(head[0x00:], 0x31415926) // signature
	binary.LittleEndian.PutUint32(head[0x04:], 0)          // root's first child
	binary.LittleEndian.PutUint32(head[0x08:], 1)          // root's child count
	binary.LittleEndian.PutUint32(head[0x0c:], 0)          // root flags
	binary.LittleEndian.PutUint32(head[0x10:], 3)          // node count
	out := append([]byte(nil), head...)
	out = append(out, regNode(1, 2, 0x01, "Fog")...)                         // dir, children [1,3)
	out = append(out, regNode(uint32(firstState), 0, 0x02, "FirstState")...) // int
	out = append(out, regNode(0, uint32(len(pool)), 0x06, "Data")...)        // int32 array at pool 0
	var n [4]byte
	binary.LittleEndian.PutUint32(n[:], uint32(len(pool)))
	out = append(out, n[:]...)
	out = append(out, pool...)
	return append(out, trail...)
}

func TestFogDecodesRunsIntoCells(t *testing.T) {
	for _, c := range []struct {
		name  string
		first int32
		runs  []int32
		want  []byte
	}{
		// FirstState 0: the first run is UNEXPLORED. This is the polarity
		// every preserved save carries.
		{"first run clear", 0, []int32{3, 2, 1}, []byte{0, 0, 0, 1, 1, 0}},
		// FirstState carries the MASKED bit, not 1, so the flip has to XOR
		// the same value: a decoder flipping against 1 would read every run
		// as explored here.
		{"first run set", 0x8000, []int32{3, 2, 1}, []byte{1, 1, 1, 0, 0, 1}},
		{"one run, nothing explored", 0, []int32{6}, []byte{0, 0, 0, 0, 0, 0}},
		{"one run, all explored", 0x8000, []int32{6}, []byte{1, 1, 1, 1, 1, 1}},
		// A zero-length run still flips the state, which is what keeps the
		// run list and the cell order in step.
		{"zero-length run flips", 0, []int32{0, 2, 0, 2}, []byte{1, 1, 1, 1}},
		{"no runs at all", 0, []int32{}, []byte{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := &File{}
			f.splitTail(fogStore(c.first, c.runs, nil))
			g, ok, err := f.Fog()
			if err != nil || !ok {
				t.Fatalf("Fog() = %v, %v, %v", g, ok, err)
			}
			if !bytes.Equal(g.Cells, c.want) {
				t.Fatalf("cells %v, want %v", g.Cells, c.want)
			}
			if g.FirstState != c.first {
				t.Fatalf("FirstState %d, want %d", g.FirstState, c.first)
			}
			set := 0
			for _, v := range c.want {
				if v != 0 {
					set++
				}
			}
			if g.Set != set {
				t.Fatalf("Set %d, want %d", g.Set, set)
			}
			if len(g.Runs) != len(c.runs) {
				t.Fatalf("Runs %v, want %v", g.Runs, c.runs)
			}
		})
	}
}

func TestFogRefusesANegativeRun(t *testing.T) {
	f := &File{}
	f.splitTail(fogStore(0, []int32{3, -1, 2}, nil))
	if _, _, err := f.Fog(); err == nil {
		t.Fatal("a negative run decoded")
	}
}

func TestNoFogSectionIsNotAnError(t *testing.T) {
	// A store with one section named something else.
	head := make([]byte, 0x18)
	binary.LittleEndian.PutUint32(head[0x00:], 0x31415926)
	binary.LittleEndian.PutUint32(head[0x08:], 1)
	binary.LittleEndian.PutUint32(head[0x10:], 1)
	store := append(append([]byte(nil), head...), regNode(0, 0, 0x01, "View")...)
	store = append(store, 0, 0, 0, 0)
	f := &File{}
	f.splitTail(store)
	g, ok, err := f.Fog()
	if err != nil {
		t.Fatalf("a store with no Fog section errored: %v", err)
	}
	if ok || g != nil {
		t.Fatalf("Fog() = %v, %v; want no record", g, ok)
	}
}

func TestTheTailSplitsAtTheStoreOwnExtent(t *testing.T) {
	trail := make([]byte, 268)
	binary.LittleEndian.PutUint32(trail, 20)
	tail := fogStore(0, []int32{4, 4}, trail)
	f := &File{}
	f.splitTail(tail)
	if len(f.Store)+len(f.TailRest) != len(tail) {
		t.Fatalf("store %d + rest %d != tail %d", len(f.Store), len(f.TailRest), len(tail))
	}
	if len(f.TailRest) != len(trail) {
		t.Fatalf("trailing region is %d bytes, want %d", len(f.TailRest), len(trail))
	}
	if !bytes.Equal(f.TailRest, trail) {
		t.Fatal("the trailing region is not the bytes that followed the store")
	}
	if _, ok := f.StateStore(); !ok {
		t.Fatal("the store did not parse out of a tail with a region behind it")
	}
	// The record decodes to the store's own cell count and not to anything
	// the trailing region contributes.
	g, ok, err := f.Fog()
	if err != nil || !ok || len(g.Cells) != 8 {
		t.Fatalf("Fog() = %v, %v, %v", g, ok, err)
	}
}

func TestAnUnframedTailIsCarriedWhole(t *testing.T) {
	for _, c := range []struct {
		name string
		tail []byte
	}{
		{"empty", nil},
		{"shorter than a header", []byte("&YA1")},
		{"wrong signature", append([]byte{1, 2, 3, 4}, make([]byte, 64)...)},
		{"heap past the end", func() []byte {
			b := make([]byte, 0x1c)
			binary.LittleEndian.PutUint32(b[0x00:], 0x31415926)
			binary.LittleEndian.PutUint32(b[0x18:], 9999) // heap length
			return b
		}()},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := &File{}
			f.splitTail(c.tail)
			if f.Store != nil {
				t.Fatalf("Store is %d bytes, want none", len(f.Store))
			}
			if _, ok := f.StateStore(); ok {
				t.Fatal("a store was reported for an unframed tail")
			}
			if !bytes.Equal(f.TailRest, c.tail) && len(c.tail) != 0 {
				t.Fatalf("rest %q, want %q", f.TailRest, c.tail)
			}
			g, ok, err := f.Fog()
			if err != nil || ok || g != nil {
				t.Fatalf("Fog() = %v, %v, %v; want no record and no error", g, ok, err)
			}
		})
	}
}

func TestMarshalPutsTheTailBackWhole(t *testing.T) {
	trail := make([]byte, 268)
	binary.LittleEndian.PutUint32(trail, 10)
	tail := fogStore(0x8000, []int32{2, 3, 4}, trail)
	fx := standard()
	fx.tail = tail
	f := open(t, fx)
	out := f.Marshal()
	if !bytes.HasSuffix(out, tail) {
		t.Fatal("the re-emitted file does not end in the tail it was given")
	}
	back, err := Open(out)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	if !bytes.Equal(back.Store, f.Store) || !bytes.Equal(back.TailRest, f.TailRest) {
		t.Fatal("a round trip moved the boundary between the store and the region behind it")
	}
	g, ok, err := back.Fog()
	if err != nil || !ok || g.Set != 2+4 {
		t.Fatalf("Fog() after a round trip = %v, %v, %v", g, ok, err)
	}
}
