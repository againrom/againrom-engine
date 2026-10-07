package game

// What the readout's PUSH costs a frame (0060 plan SC-7, R-1).
//
// The digest is the expensive half and the only reason this benchmark exists:
// it is a full encode of the world's byte form, allocated, and then hashed. Its
// cost is linear in the entity count, so it is measured at three of them; the
// hidden case is measured beside it, because that is what the gate on the push
// is worth.

import (
	"fmt"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func benchViewer(b *testing.B) *ui.Viewer {
	b.Helper()
	v, err := ui.NewViewer("bench", terrain.Grid{
		Width: readoutW, Height: readoutH, Tiles: make([]uint16, readoutW*readoutH),
	}, &terrain.Tileset{})
	if err != nil {
		b.Fatalf("NewViewer: %v", err)
	}
	return v
}

func benchWorld(b *testing.B, n int) (*mapWorld, *sim.World) {
	b.Helper()
	ents := make([]sim.Entity, n)
	for i := range ents {
		ents[i] = sim.Entity{ID: sim.EntityID(i + 1), X: int32(i % readoutW), Y: int32(i / readoutW),
			HP: 10, MaxHP: 10, Speed: 30}
	}
	w, err := sim.NewWorld(1, sim.Bounds{Width: readoutW, Height: readoutH},
		sim.ModeCanonical, make([]byte, readoutW*readoutH), ents)
	if err != nil {
		b.Fatalf("NewWorld: %v", err)
	}
	v := benchViewer(b)
	return newMapWorld(w, nil, nil, v), w
}

// BenchmarkPushReadout is one frame's push, digest and all.
func BenchmarkPushReadout(b *testing.B) {
	for _, n := range []int{1, 64, 256} {
		b.Run(fmt.Sprintf("entities=%d", n), func(b *testing.B) {
			mw, _ := benchWorld(b, n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				mw.pushReadout()
			}
		})
	}
}

// BenchmarkPushReadoutHidden is the same call with the box hidden: one method
// call and a return, which is what the gate buys.
func BenchmarkPushReadoutHidden(b *testing.B) {
	mw, _ := benchWorld(b, 256)
	mw.view.ShowReadout(false)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mw.pushReadout()
	}
}

// BenchmarkWorldHash isolates the digest itself out of the push above.
func BenchmarkWorldHash(b *testing.B) {
	_, w := benchWorld(b, 256)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = w.Hash()
	}
}
