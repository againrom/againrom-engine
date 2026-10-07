package sav

import "fmt"

// The explored-terrain record, in the uncompressed tail's state store
// (SAV-FOG-061).
//
// The section is `Fog` and it has two leaves. `FirstState` is an int and
// `Data` is an int32 array. `Data` is a run-length encoding of the map's
// explored bit over exactly width*height cells in linear order
// idx = col + row*width (TERR-EDGE-024). The first run carries the state
// `FirstState` records and the state flips at every run boundary.
//
// THE RECORD DOES NOT CARRY THE WIDTH. It carries a cell count, as the sum of
// its runs, and the linear order is only meaningful against a width the caller
// supplies. So this package decodes to a flat plane and reports the count; it
// divides by nothing. A consumer holding the map checks the product itself,
// which is what keeps this leaf free of a map.
//
// The between-mission save carries no `Fog` section at all, which is the store
// routine's own world-half guard. That is reported as absence and is not an
// error.

// fogSection and the two leaf names, spelled once.
const (
	fogSection    = "Fog"
	fogFirstState = "FirstState"
	fogData       = "Data"
)

// Fog is the decoded explored-terrain record.
type Fog struct {
	// FirstState is the leaf's own value, verbatim. The original writes the
	// masked bit rather than 0/1, so this is not a boolean and is not
	// narrowed to one; Explored below is what it means.
	FirstState int32

	// Runs is the run-length encoding, verbatim and in order.
	Runs []int32

	// Cells is one byte per cell, 1 where the cell is explored and 0 where
	// it is not, in the linear order idx = col + row*width. Its length is
	// the sum of Runs.
	Cells []byte

	// Set is how many of Cells are explored, counted during the decode so a
	// caller reporting it does not walk the plane again.
	Set int
}

// Fog decodes the explored-terrain record, and reports false where the save
// carries none.
//
// FALSE IS NOT AN ERROR. A save taken between missions has no `Fog` section —
// the store routine's world-half guard skips it — so "no record" is a lawful
// shape of the file and a caller distinguishes it from a malformed one by the
// error being nil.
//
// A NEGATIVE RUN IS REFUSED, naming the element. The leaf is a signed int32
// array and nothing in the format forbids one; a decoder that treated it as a
// length would size a plane from it.
func (f *File) Fog() (*Fog, bool, error) {
	r, ok := f.StateStore()
	if !ok {
		return nil, false, nil
	}
	data, ok := r.GetIntArray(fogSection, fogData)
	if !ok {
		return nil, false, nil
	}
	// FirstState is read after Data on purpose: a store carrying one leaf and
	// not the other is a store with no usable record, and Data is the leaf
	// that decides that. A missing FirstState is refused rather than defaulted
	// to zero, because zero is a POLARITY and supplying one would invert the
	// plane on a file that never said which way round it goes.
	first, ok := r.GetInt(fogSection, fogFirstState)
	if !ok {
		return nil, false, fmt.Errorf("sav: %s/%s is present and %s/%s is not",
			fogSection, fogData, fogSection, fogFirstState)
	}
	g := &Fog{FirstState: first, Runs: append([]int32(nil), data...)}
	total := 0
	for i, n := range data {
		if n < 0 {
			return nil, false, fmt.Errorf("sav: %s/%s element %d is %d, a negative run length",
				fogSection, fogData, i, n)
		}
		total += int(n)
	}
	g.Cells = make([]byte, total)
	// The state starts at FirstState and flips at every run boundary, which
	// is the load arm's own XOR. Non-zero is explored.
	at, state := 0, first
	for _, n := range data {
		if state != 0 {
			for i := 0; i < int(n); i++ {
				g.Cells[at+i] = 1
			}
			g.Set += int(n)
		}
		at += int(n)
		state ^= exploredBit
	}
	return g, true, nil
}

// exploredBit is the tile-plane bit the record encodes, and the value the
// original's own load arm holds in EDI while it walks the plane
// (TERR-FOG-145). It is here because FirstState carries the MASKED bit rather
// than a 0/1 flag, so the flip between runs has to XOR the same value the file
// was written with — flipping against 1 would read a FirstState of 0x8000 as
// explored on every run.
const exploredBit int32 = 0x8000
