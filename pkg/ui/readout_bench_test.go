package ui

// What the readout costs a frame (0060 plan SC-7, R-1, R-2).
//
// THE FONT IS THE SUITE'S SYNTHETIC ONE — 224 records of 5x6 — not the game's,
// because a benchmark that needed a lawful install could not run here at all.
// The composition's cost is dominated by the box's AREA (the frame fill writes
// every pixel) and by the glyph count, and the synthetic font's cell is within a
// pixel or two of the shipped one on both axes, so these numbers are the right
// order and are not claimed to be exact for the shipped font.

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
)

func benchReadoutViewer(b *testing.B) *Viewer {
	b.Helper()
	v, err := NewViewer("bench", grid(60, 60), &terrain.Tileset{})
	if err != nil {
		b.Fatalf("NewViewer: %v", err)
	}
	v.SetFont(panelFont())
	v.cam.ViewW, v.cam.ViewH = 1920, 1080
	v.hasCursor, v.cursorX, v.cursorY = true, 400, 300
	v.entities = make([]MapEntity, 200)
	for i := range v.entities {
		v.entities[i] = MapEntity{ID: uint32(i + 1), Cell: image.Pt(i%60, i/60),
			Speed: 30, GroupSpeed: 12, TransitSpan: 5, HP: 10, MaxHP: 10}
	}
	v.sel = selection{1, 2, 3, 4, 5}
	v.SetReadout(Readout{PeriodUS: 62_500, Tick: 1000, Digest: 0x0123456789abcdef})
	return v
}

// BenchmarkReadoutCompose is the worst case: every frame states a new tick, so
// the key never matches and the box is composed from scratch. This is the cost
// at the top of the rate ladder, where a tick fires every frame.
func BenchmarkReadoutCompose(b *testing.B) {
	v := benchReadoutViewer(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.SetReadout(Readout{PeriodUS: 62_500, Tick: uint64(i), Digest: uint64(i)})
		v.readoutPresent(60)
	}
}

// BenchmarkReadoutCached is the ordinary frame: nothing stated has changed, so
// the picture already held is re-presented. At the map-load cadence this is 44
// frames in 60.
func BenchmarkReadoutCached(b *testing.B) {
	v := benchReadoutViewer(b)
	v.readoutPresent(60)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.readoutPresent(60)
	}
}

// BenchmarkReadoutHidden is what a hidden box costs: the first statement of the
// path returns, so no subject is built and no cell resolved.
func BenchmarkReadoutHidden(b *testing.B) {
	v := benchReadoutViewer(b)
	v.ShowReadout(false)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.readoutPresent(60)
	}
}

// BenchmarkReadoutSubject is the per-frame work the cache does NOT save: the
// ground pick, the selection walk and the entity count, which run on every frame
// whether the picture is rebuilt or not.
func BenchmarkReadoutSubject(b *testing.B) {
	v := benchReadoutViewer(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.readoutSubjectOf(60)
	}
}
