package game

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/pkg/sim"
)

// adoptLoadedAllocator gives w the runtime-id bits no object carries that
// loaded holds, and returns w. A LOAD rebuilds those bits from the file it
// reads (SAV-1371), not from the World that wrote the file: a Sack
// constructor's bit and an Item's mark are the file's, so a World that picked
// up a Sack since its own LOAD holds a bit the next LOAD does not. Every other
// field of w stays, so a comparison against loaded still covers every carried
// id, and play continued from w allocates as play continued from loaded.
func adoptLoadedAllocator(t *testing.T, w, loaded *sim.World) *sim.World {
	t.Helper()
	marks := loaded.RuntimeIDMarks()
	records := make([]sim.RuntimeIDLoad, len(marks))
	for i, id := range marks {
		records[i] = sim.RuntimeIDLoad{Saved: uint32(id)}
	}
	w.RestoreRuntimeIDs(records)
	return w
}

// insertFixtureRuntimeIDs gives every on-map actor of a test-built World a
// runtime id in entity order, as mission entry gives a map's actors theirs.
// A body keeps its id until its last decay stage removes it (SAV-1370).
func insertFixtureRuntimeIDs(w *sim.World) *sim.World {
	var ids []sim.EntityID
	for _, e := range w.Entities() {
		ids = append(ids, e.ID)
	}
	w.InsertActorRuntimeIDs(ids)
	return w
}

// beforeRuntimeIDForm peels the outer runtime-id wrapper (version 128, tag
// RID1: section, uint32 span, base version, tag) off a World form.
func beforeRuntimeIDForm(t *testing.T, form []byte) []byte {
	t.Helper()
	if len(form) == 0 || form[0] != 128 {
		return form
	}
	end := len(form)
	if end < 47 || string(form[end-4:]) != "RID1" || form[end-5] >= 128 {
		t.Fatal("invalid runtime-id wrapper")
	}
	span := uint64(binary.LittleEndian.Uint32(form[end-9:]))
	if span < 8 || span > uint64(end-9-38) {
		t.Fatal("invalid runtime-id span", span)
	}
	base := form[end-5]
	form = bytes.Clone(form[:end-9-int(span)])
	form[0] = base
	return form
}

// formWithLoadedAllocator is adoptLoadedAllocator over two World byte forms.
func formWithLoadedAllocator(t *testing.T, form, loaded []byte) []byte {
	t.Helper()
	var w, l sim.World
	if err := w.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if err := l.UnmarshalBinary(loaded); err != nil {
		t.Fatal(err)
	}
	out, err := adoptLoadedAllocator(t, &w, &l).MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return out
}
