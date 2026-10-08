package sim

import (
	"bytes"
	"math"
	"math/rand"
	"testing"
)

// The bearing code's sixteen sectors, each edge included (MOVE-097).
func TestPickerBBearingCodeFollowsTheSixteenSectors(t *testing.T) {
	cases := []struct {
		dx, dy int64
		want   int32
	}{
		{1, -3, 0}, {1, -2, 1}, {1, -1, 1}, {2, -1, 2}, {3, -1, 3}, {1, 0, 3},
		{3, 1, 4}, {2, 1, 5}, {1, 1, 6}, {1, 2, 6}, {1, 3, 7},
		{0, 1, 8}, {-1, 3, 8}, {-1, 2, 9}, {-1, 1, 9}, {-2, 1, 10}, {-3, 1, 11},
		{-1, 0, 12}, {-3, -1, 12}, {-2, -1, 13}, {-1, -1, 14}, {-1, -2, 14}, {-1, -3, 15}, {0, -1, 15},
		{0, 0, 14},
	}
	for _, c := range cases {
		if got := bearingCode(c.dx, c.dy); got != c.want {
			t.Errorf("bearing of (%d,%d) is %d, want %d", c.dx, c.dy, got, c.want)
		}
	}
}

// The crossing equals the double-precision evaluation over a single-precision
// slope, the arithmetic the original runs (MOVE-098).
func TestPickerBCrossingIsTheDoublePrecisionLineOverASingleSlope(t *testing.T) {
	ref := func(alongT, acrossT, alongM, acrossM int64, line int32) int32 {
		q := float64(acrossT - acrossM)
		if q == 0 {
			q = 1
		}
		slope := float64(float32(float64(alongT-alongM) / q))
		c := math.Trunc(float64(alongT) - float64(acrossT)*slope)
		return int32(int64(math.Trunc(float64(int64(line)*256+0x80)*slope+c)) >> 8)
	}
	r := rand.New(rand.NewSource(1))
	for n := 0; n < 20000; n++ {
		at, ac := r.Int63n(255*256)+128, r.Int63n(255*256)+128
		am, cm := r.Int63n(255*256)+128, r.Int63n(255*256)+128
		line := int32(r.Intn(256))
		if got, want := pickerCrossing(at, ac, am, cm, line), ref(at, ac, am, cm, line); got != want {
			t.Fatalf("crossing (%d,%d,%d,%d,%d) = %d, want %d", at, ac, am, cm, line, got, want)
		}
	}
	for _, v := range [][2]int64{{1, 3}, {2, 3}, {-7, 9}, {1 << 20, 3}, {5, 1 << 20}, {16777217, 1}, {3, 16777217}} {
		m, s := singleQuotient(v[0], v[1])
		got := float64(m.Int64()) * math.Pow(2, -float64(s))
		if want := float64(float32(float64(v[0]) / float64(v[1]))); got != want {
			t.Errorf("%d/%d rounds to %v, want %v", v[0], v[1], got, want)
		}
	}
}

// pickerWorld is an open 200x200 world with a side-1 victim at (100,100) and
// a pursuer of the given footprint, with every label cleared.
func pickerWorld(t *testing.T, mx, my int32, side uint8) (*World, *routeScratch) {
	t.Helper()
	m := cbEnt(1, mx, my)
	m.TokenSize = side
	m.AttackTarget, m.HasAttackTarget = 2, true
	w, err := NewWorld(1, Bounds{Width: 200, Height: 200}, ModeCanonical, nil, []Entity{m, cbEnt(2, 100, 100)})
	if err != nil {
		t.Fatal(err)
	}
	s := newRouteScratch(w)
	s.reset()
	return w, s
}

func setLabel(w *World, s *routeScratch, x, y int32, l uint64) {
	i, _ := w.cellIndex(x, y)
	s.plane[i] = l + 1
	s.touched = append(s.touched, i)
}

// A wide mover near NNE enters ring 1 east of the box: its first walker probes
// cells off the ring and its second leaves ring cells unprobed (MOVE-098,
// MOVE-099).
func TestPickerBWideMoverEntryLeavesTheRing(t *testing.T) {
	w, s := pickerWorld(t, 109, 88, 2)
	setLabel(w, s, 108, 98, 5)
	if got, ok := w.pickerB(s, 0, cell{x: 109, y: 88}); !ok || got != (cell{x: 108, y: 98}) {
		t.Fatalf("picker B returned %v %v, want the off-ring cell (108,98)", got, ok)
	}
	w, s = pickerWorld(t, 109, 88, 2)
	setLabel(w, s, 101, 100, 5)
	if got, ok := w.pickerB(s, 0, cell{x: 109, y: 88}); ok {
		t.Fatalf("picker B returned %v; the ring-1 cell (101,100) is never probed", got)
	}
}

// Ties go to the entry, then to the first walker; the first ring with a label
// ends the scan, and nothing labelled returns nothing (MOVE-099).
func TestPickerBKeepsTheFirstStrictlyLowestLabel(t *testing.T) {
	w, s := pickerWorld(t, 100, 90, 1)
	for x := int32(99); x <= 101; x++ {
		for y := int32(99); y <= 101; y++ {
			if x != 100 || y != 100 {
				setLabel(w, s, x, y, 7)
			}
		}
	}
	if got, _ := w.pickerB(s, 0, cell{x: 100, y: 90}); got != (cell{x: 100, y: 99}) {
		t.Errorf("equal labels on the ring return %v, want the entry (100,99)", got)
	}
	w, s = pickerWorld(t, 100, 90, 1)
	setLabel(w, s, 101, 99, 7)
	setLabel(w, s, 99, 99, 7)
	if got, _ := w.pickerB(s, 0, cell{x: 100, y: 90}); got != (cell{x: 101, y: 99}) {
		t.Errorf("equal labels beside the entry return %v, want the first walker's (101,99)", got)
	}
	w, s = pickerWorld(t, 100, 90, 1)
	setLabel(w, s, 99, 101, 9)
	setLabel(w, s, 100, 98, 1)
	if got, _ := w.pickerB(s, 0, cell{x: 100, y: 90}); got != (cell{x: 99, y: 101}) {
		t.Errorf("a ring-1 label 9 against a ring-2 label 1 returns %v, want the ring-1 cell", got)
	}
	w, s = pickerWorld(t, 100, 90, 1)
	if got, ok := w.pickerB(s, 0, cell{x: 100, y: 90}); ok {
		t.Errorf("no label returned %v", got)
	}
}

// pursuitLine is an open strip with a reach-1 pursuer at (0,10) and a
// stationary victim d cells east of it, both moving a cell a tick.
func pursuitLine(t *testing.T, d int32) *World {
	t.Helper()
	w, err := NewWorld(1, Bounds{Width: 40, Height: 21}, ModeCanonical, nil, []Entity{puFighter(1, 0, 10), cbEnt(2, d, 10)})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// A fresh pursuit searches in full; passes above a third of the count plus one
// search in full again while the last count is above five, then rebuild the
// route end as one node (AI-413, AI-414, AI-415).
func TestAPursuitSearchesInFullThenRebuildsTheRouteEnd(t *testing.T) {
	w := pursuitLine(t, 8)
	var counts []uint16
	var xs []int32
	cmds := []Command{cbOrder(1, 2)}
	last := PursuitSearch{}
	for k := 0; k < 12; k++ {
		Step(w, cmds)
		cmds = nil
		a := cbAt(t, w, 1)
		p := a.Pursuit
		if p.Held && (!last.Held || p.Passes < last.Passes) {
			counts = append(counts, p.Count)
			xs = append(xs, a.X)
		}
		last = p
	}
	if len(counts) != 3 || counts[0] != 8 || counts[1] != 5 || counts[2] != 1 {
		t.Fatalf("searches left counts %v at x %v, want 8, then 5 after three steps, then the rebuild's 1", counts, xs)
	}
	if xs[1] != 4 {
		t.Errorf("the second full search ran at x %d, want the fourth pass (x 4 after its step)", xs[1])
	}
	if r := w.Route(1); len(r) > 1 || len(r) == 1 && r[0] != [2]int32{8, 10} {
		t.Errorf("the rebuilt route is %v, want the route end (8,10) alone", r)
	}
}

// A victim behind a band of four impassable rings: from Chebyshev 7 the first
// full search finds no ring within (7>>2)+3 and cancels at once; from 12 the
// pursuer walks to ring 5, its route end, and cancels there (AI-418).
func TestABandAroundTheVictimCancelsWhereTheFullSearchFindsNoRing(t *testing.T) {
	for _, c := range []struct {
		from, wantX int32
	}{{7, 93}, {12, 95}} {
		grid := make([]byte, 200*200)
		for y := int32(96); y <= 104; y++ {
			for x := int32(96); x <= 104; x++ {
				if x != 100 || y != 100 {
					grid[int(y)*200+int(x)] = blockGround
				}
			}
		}
		a := puFighter(1, 100-c.from, 100)
		a.Reach = 3
		w, err := NewWorld(1, Bounds{Width: 200, Height: 200}, ModeCanonical, grid, []Entity{a, cbEnt(2, 100, 100)})
		if err != nil {
			t.Fatal(err)
		}
		cmds := []Command{cbOrder(1, 2)}
		for k := 0; k < 40 && !cbAt(t, w, 1).PursuitIdle; k++ {
			Step(w, cmds)
			cmds = nil
		}
		if e := cbAt(t, w, 1); !e.PursuitIdle || e.X != c.wantX || e.Y != 100 {
			t.Errorf("from %d: idle %v at (%d,%d), want idle at (%d,100)", c.from, e.PursuitIdle, e.X, e.Y, c.wantX)
		}
	}
}

// ringWorld puts a victim at (10,10), bodies on the given ring-1 cells and a
// reach-1 pursuer at (12,10).
func ringWorld(t *testing.T, bodies []cell) *World {
	t.Helper()
	ents := []Entity{puFighter(1, 12, 10), cbEnt(2, 10, 10)}
	for i, c := range bodies {
		ents = append(ents, cbEnt(EntityID(10+i), c.x, c.y))
	}
	w, err := NewWorld(1, Bounds{Width: 24, Height: 24}, ModeCanonical, nil, ents)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func ringOne(skip cell) []cell {
	var out []cell
	for x := int32(9); x <= 11; x++ {
		for y := int32(9); y <= 11; y++ {
			if (x != 10 || y != 10) && (cell{x: x, y: y}) != skip {
				out = append(out, cell{x: x, y: y})
			}
		}
	}
	return out
}

// Occupied ring cells carry no label, so picker B passes them by and takes
// the free one; with the whole ring occupied it returns the mover's own start
// cell, the near search has no step, and the pursuit aimed at its route end is
// cancelled (MOVE-100, AI-373, AI-416).
func TestPickerBSkipsOccupiedCellsAndAFullRingCancels(t *testing.T) {
	free := cell{x: 10, y: 9}
	w := ringWorld(t, ringOne(free))
	cmds := []Command{cbOrder(1, 2)}
	for k := 0; k < 10; k++ {
		Step(w, cmds)
		cmds = nil
	}
	if e := cbAt(t, w, 1); e.X != free.x || e.Y != free.y || e.PursuitIdle {
		t.Errorf("with one free ring cell the pursuer stands at (%d,%d) idle %v, want (10,9)", e.X, e.Y, e.PursuitIdle)
	}
	w = ringWorld(t, ringOne(cell{}))
	Step(w, []Command{cbOrder(1, 2)})
	if e := cbAt(t, w, 1); !e.PursuitIdle || e.X != 12 || e.Y != 10 || e.Pursuit.Held {
		t.Errorf("with the ring full the pursuer is idle %v at (%d,%d) holding %+v, want idle at (12,10) with nothing held",
			e.PursuitIdle, e.X, e.Y, e.Pursuit)
	}
}

// A world mid-pursuit carries its search state through the byte form.
func TestAHeldPursuitSearchRoundTrips(t *testing.T) {
	w := pursuitLine(t, 12)
	Step(w, []Command{cbOrder(1, 2)})
	Step(w, nil)
	if !cbAt(t, w, 1).Pursuit.Held {
		t.Fatal("the fixture needs a held pursuit")
	}
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if b[0] != pursuitSearchFormVersion || b[len(b)-5] != formatVersion {
		t.Fatalf("the form reads %d over base %d", b[0], b[len(b)-5])
	}
	var got World
	if err := got.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	if again, _ := got.MarshalBinary(); !bytes.Equal(again, b) || got.Hash() != w.Hash() {
		t.Fatal("the held pursuit did not survive the round trip")
	}
	if cbAt(t, &got, 1).Pursuit != cbAt(t, w, 1).Pursuit {
		t.Fatalf("restored %+v, want %+v", cbAt(t, &got, 1).Pursuit, cbAt(t, w, 1).Pursuit)
	}
	for k := 0; k < 8; k++ {
		Step(w, nil)
		Step(&got, nil)
	}
	if got.Hash() != w.Hash() {
		t.Fatal("the restored pursuit continued differently")
	}
	b[len(b)-10] ^= 0xff
	var bad World
	if err := bad.UnmarshalBinary(b); err == nil {
		t.Fatal("a corrupted record was accepted")
	}
}

// An order rewritten at the victim the pursuit holds keeps its route and its
// counters; a different victim starts afresh (AI-413, AI-REISSUE-077).
func TestAReissueAtTheHeldVictimKeepsThePursuit(t *testing.T) {
	w := pursuitLine(t, 14)
	Step(w, []Command{cbOrder(1, 2)})
	Step(w, nil)
	before, route := cbAt(t, w, 1).Pursuit, w.Route(1)
	i := indexOfEntity(w.entities, 1)
	if !w.orderAttack(i, 2) {
		t.Fatal("reissue refused")
	}
	if p := cbAt(t, w, 1).Pursuit; p != before || len(w.Route(1)) != len(route) {
		t.Fatalf("reissue left %+v and %d cells, want %+v and %d", p, len(w.Route(1)), before, len(route))
	}
}
