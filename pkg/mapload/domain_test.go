package mapload_test

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The band map's shape. The extent is large enough that the eight-cell border
// ring leaves an interior on both sides of the band, and the band itself is
// three cells of water laid across the whole width — wider than one step, so no
// mover crosses it by accident, and spanning the map, so there is no way round
// it inside the interior.
const (
	bandW, bandH = 40, 40
	bandTop      = 20
	bandRows     = 3
	southRow     = 12 // where the movers start
	northRow     = 28 // where they are sent
)

// waterTile is a tile word whose index falls in the water range the derivation
// classifies, so the band's cells close to a ground mover and to no other.
//
// It is written as the contract's own range rather than taken from the
// derivation: read from the classifier, this constant would agree with whatever
// the classifier said.
const waterTile = 600

// bandMap is a map whose derived plane carries a band of ground-blocking cells
// straight across it, with the given placements in the southern interior.
//
// Every cell outside the band and the border ring is open to every domain, so a
// mover that fails to arrive failed at the band.
func bandMap(units []alm.Unit) *alm.Map {
	tiles := make([]uint16, bandW*bandH)
	for y := bandTop; y < bandTop+bandRows; y++ {
		for x := 0; x < bandW; x++ {
			tiles[y*bandW+x] = waterTile
		}
	}
	return &alm.Map{Width: bandW, Height: bandH, Tiles: tiles, Units: units}
}

// at places one unit at the centre of cell (x, y) carrying the given class key.
func at(x, y int32, class int16) alm.Unit {
	return alm.Unit{X: uint32(x)<<8 | 0x80, Y: uint32(y)<<8 | 0x80, ClassID: class}
}

// bandTable keys one row per domain on the three class keys the fixtures use.
func bandTable() *mapload.Table {
	return &mapload.Table{Units: defCollection{
		{},
		{name: "walker", params: domainRow(0x40, 1)},
		{name: "ghost", params: domainRow(0x41, 2)},
		{name: "flyer", params: domainRow(0x42, 3)},
	}}
}

// stepTo advances w for n ticks, issuing cmds on the first.
func stepTo(w *sim.World, n int, cmds []sim.Command) {
	sim.Step(w, cmds)
	for i := 1; i < n; i++ {
		sim.Step(w, nil)
	}
}

// TestTheBandStopsAWalkerAndNeitherOtherDomain is AC-4, SC-4 and the whole of
// what the owner is meant to see: a flyer crosses what a walker cannot.
//
// The ground mover is checked at EVERY tick and not only at the end. Ending on
// the near side is consistent with two different stories — refused outright, or
// walked across and back — and only the per-tick check tells them apart.
func TestTheBandStopsAWalkerAndNeitherOtherDomain(t *testing.T) {
	m := bandMap([]alm.Unit{
		at(15, southRow, 0x40), // walker
		at(17, southRow, 0x41), // ghost
		at(19, southRow, 0x42), // flyer
	})
	w, err := mapload.FromALMWith(m, bandTable(), mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}

	// The domains the fixture rests on. Asserted before the movement, so a
	// crossing that happened for the wrong reason cannot read as this test
	// passing.
	for i, want := range []sim.Domain{sim.DomainGround, sim.DomainGhost, sim.DomainAir} {
		if got := w.Entities()[i].Domain; got != want {
			t.Fatalf("entity %d is in domain %d, want %d", i, got, want)
		}
	}

	cmds := []sim.Command{
		{Entity: 0, X: 15, Y: northRow},
		{Entity: 1, X: 17, Y: northRow},
		{Entity: 2, X: 19, Y: northRow},
	}
	// Long enough that every mover has crossed the band with room to spare. A
	// unit these fixtures place resolves to no definition and so takes
	// DefaultSpeed, whose rate costs 26 ticks a straight cell, so the count is
	// scaled by that: a bare 300 is now eleven cells and this walk is longer.
	const ticks = 300 * 26
	sim.Step(w, cmds)
	for tick := 1; tick < ticks; tick++ {
		sim.Step(w, nil)
		if e := w.Entities()[0]; e.Y >= bandTop {
			t.Fatalf("at tick %d the ground mover stands at (%d,%d), on or past the band", tick, e.X, e.Y)
		}
	}

	for _, tc := range []struct {
		id   int
		name string
	}{{1, "the ghost"}, {2, "the flyer"}} {
		e := w.Entities()[tc.id]
		if e.Y != northRow {
			t.Errorf("%s stands at (%d,%d); it was sent to row %d and did not arrive", tc.name, e.X, e.Y, northRow)
		}
	}
	if e := w.Entities()[0]; e.Y == northRow {
		t.Errorf("the ground mover reached row %d across a band that closes to it", northRow)
	}
}

// TestAGhostContendsWithAWalkerAndAFlyerDoesNot is AC-5 and SC-5.
//
// The two halves are the two sides of the layer split: a ghost shares the ground
// mover's plane and so cannot come to rest on its cell, while flyers are on a
// plane of their own and pass through each other while moving.
func TestAGhostContendsWithAWalkerAndAFlyerDoesNot(t *testing.T) {
	t.Run("a ghost and a walker end on distinct cells", func(t *testing.T) {
		m := bandMap([]alm.Unit{
			at(15, southRow, 0x40), // walker
			at(18, southRow, 0x41), // ghost
		})
		w, err := mapload.FromALMWith(m, bandTable(), mapload.DifficultyNormal)
		if err != nil {
			t.Fatalf("FromALMWith: %v", err)
		}
		stepTo(w, 200, []sim.Command{
			{Entity: 0, X: 16, Y: southRow + 2},
			{Entity: 1, X: 16, Y: southRow + 2},
		})

		a, b := w.Entities()[0], w.Entities()[1]
		if a.X == b.X && a.Y == b.Y {
			t.Errorf("the ghost and the walker came to rest on one cell, (%d,%d)", a.X, a.Y)
		}
	})

	t.Run("two flyers pass through each other", func(t *testing.T) {
		m := bandMap([]alm.Unit{
			at(14, southRow, 0x42),
			at(20, southRow, 0x42),
		})
		w, err := mapload.FromALMWith(m, bandTable(), mapload.DifficultyNormal)
		if err != nil {
			t.Fatalf("FromALMWith: %v", err)
		}
		for i := 0; i < 2; i++ {
			if got := w.Entities()[i].Domain; got != sim.DomainAir {
				t.Fatalf("entity %d is in domain %d, want air", i, got)
			}
		}

		// Each is sent to the other's cell, so their straight lines cross.
		cmds := []sim.Command{
			{Entity: 0, X: 20, Y: southRow},
			{Entity: 1, X: 14, Y: southRow},
		}
		shared := false
		sim.Step(w, cmds)
		for tick := 1; tick < 200; tick++ {
			sim.Step(w, nil)
			a, b := w.Entities()[0], w.Entities()[1]
			if a.X == b.X && a.Y == b.Y {
				shared = true
			}
		}
		if !shared {
			t.Error("the two flyers never occupied one cell; they routed around each other")
		}
		for i, want := range []int32{20, 14} {
			if e := w.Entities()[i]; e.X != want || e.Y != southRow {
				t.Errorf("flyer %d ended at (%d,%d), want (%d,%d)", i, e.X, e.Y, want, southRow)
			}
		}
	})
}
