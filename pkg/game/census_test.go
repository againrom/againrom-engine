package game_test

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
)

// The class ids the fixture set answers, deliberately sparse. censusFramedNeg
// is negative on purpose: an entity's class id arrives sign-extended, so a
// framed entry at a negative key is counted a sprite only by a census that
// looks up the SIGNED value — one that zero-extended would miss it and call
// it no-class.
const (
	censusFramed    = 3  // an entry with a frame
	censusFramedNeg = -5 // a framed entry at a negative key
	censusExcluded  = 7  // a frameless entry — an excluded class
	censusNilEntry  = 9  // an entry present but nil
	censusUnknown   = 4  // no entry at all
	censusUnknownN  = -6 // no entry, negative
)

// censusSet is the hand-assembled bundle.
func censusSet() *terrain.UnitSet {
	sheet := []*terrain.StaticFrame{{Width: 1, Height: 1}}
	return &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{
		censusFramed:    {Width: 16, Height: 16, CenterX: 8, CenterY: 14, Frames: sheet},
		censusFramedNeg: {Width: 12, Height: 12, CenterX: 6, CenterY: 10, Frames: sheet},
		censusExcluded:  {Width: 16, Height: 16, CenterX: 8, CenterY: 14},
		censusNilEntry:  nil,
	}}
}

// censusMap is the decoded map itself, seven placed units: three whose keys
// resolve to framed entries (two of one class, one at the negative key), two
// naming frameless entries (the excluded class and the nil one), two naming
// no entry at all (one negative). The stored anchors carry non-zero low
// bytes so the world's >>8 cell truncation is genuinely crossed.
func censusMap() *alm.Map {
	return &alm.Map{
		Width:  4,
		Height: 3,
		Units: []alm.Unit{
			{X: 1<<8 | 0x40, Y: 1 << 8, ClassID: censusFramed},
			{X: 2 << 8, Y: 1<<8 | 0x80, ClassID: censusFramed},
			{X: 3 << 8, Y: 1 << 8, ClassID: censusFramedNeg},
			{X: 1 << 8, Y: 2 << 8, ClassID: censusExcluded},
			{X: 2 << 8, Y: 2 << 8, ClassID: censusNilEntry},
			{X: 3 << 8, Y: 2 << 8, ClassID: censusUnknown},
			{X: 3<<8 | 0x7f, Y: 2<<8 | 0x01, ClassID: censusUnknownN},
		},
	}
}

func TestUnitCensus(t *testing.T) {
	t.Run("the three answers are counted apart", func(t *testing.T) {
		m := censusMap()
		got := game.UnitCensus(m, censusSet())
		want := game.UnitCounts{Sprites: 3, NoFrame: 2, NoClass: 2}
		if got != want {
			t.Errorf("UnitCensus = %+v, want %+v", got, want)
		}
		// The buckets PARTITION the entities: FromALM builds one entity per
		// placed unit and every entity lands in exactly one count.
		if total := got.Sprites + got.NoClass + got.NoFrame; total != len(m.Units) {
			t.Errorf("the counts sum to %d, want the map's %d units", total, len(m.Units))
		}
	})

	t.Run("with no bundle every id is no-class", func(t *testing.T) {
		m := censusMap()
		for _, tc := range []struct {
			label string
			set   *terrain.UnitSet
		}{
			{"a nil set", nil},
			{"a zero set", &terrain.UnitSet{}},
		} {
			want := game.UnitCounts{NoClass: len(m.Units)}
			if got := game.UnitCensus(m, tc.set); got != want {
				t.Errorf("%s: UnitCensus = %+v, want %+v", tc.label, got, want)
			}
		}
	})

	t.Run("a map with no units counts nothing", func(t *testing.T) {
		m := &alm.Map{Width: 2, Height: 2}
		if got := game.UnitCensus(m, censusSet()); got != (game.UnitCounts{}) {
			t.Errorf("UnitCensus = %+v, want all zeroes", got)
		}
	})
}
