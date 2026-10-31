package mapedit

import (
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
)

// The random terrain command asks its source column by column, x outer and y
// inner, one answer per cell that has a look-alike. A map whose cells need
// different numbers of answers shows the order: the sequence of the sizes the
// source is asked for is that of the cells walked x outer.
func TestRandomTerrainWalksTheMapXOuterYInner(t *testing.T) {
	const w, h = 32, 32
	m, err := NewMap(alm.NewMapSpec{Width: w, Height: h, Tile: 0x11, Altitude: 64})
	if err != nil {
		t.Fatal(err)
	}
	e, err := New(m.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	// Cells of rows 8 and below, each given a word whose look-alike count
	// follows a pattern that differs between the two walks.
	var pool []uint16
	seen := map[int]bool{}
	for i := range uint16(0x400) {
		if ws := lookalikes(i); ws != nil && !seen[len(ws)] {
			seen[len(ws)] = true
			pool = append(pool, i)
		}
	}
	if len(pool) < 3 {
		t.Fatalf("only %d look-alike sizes", len(pool))
	}
	f := e.locate()
	span := f.required(alm.TypeTiles)
	old := e.tileWords(span, w*h)
	next := slices.Clone(old)
	for i := range next {
		next[i] = pool[(i*i+i/3)%len(pool)]
	}
	e.apply(e.tileEdit(f, w, old, next, nil))
	now := e.tileWords(span, w*h)

	var want []int
	for x := range w {
		for y := range h {
			if ws := lookalikes(now[y*w+x]); ws != nil {
				want = append(want, len(ws))
			}
		}
	}
	var other []int
	for y := range h {
		for x := range w {
			if ws := lookalikes(now[y*w+x]); ws != nil {
				other = append(other, len(ws))
			}
		}
	}
	if slices.Equal(want, other) {
		t.Fatal("the fixture cannot tell the two walks apart")
	}
	var got []int
	if _, err := e.RandomizeTerrain(func(n int) int { got = append(got, n); return 0 }); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("source asked for sizes %v, want the x-outer walk %v", got, want)
	}
}
