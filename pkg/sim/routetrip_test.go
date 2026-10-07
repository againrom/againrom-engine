package sim

// A corpus of worlds walked to arrival, each one cut open at every tick.
//
// AC-7 is the criterion a stored route has to earn its place in the byte form
// by: a world marshalled at ANY tick and decoded into another must advance
// identically to it at every later tick, digest for digest. The cut is taken at
// every tick rather than at one, because a route only exists between an order
// and an arrival — a run cut before the order and after the arrival would carry
// no route across the boundary at all and would pass with the routes deleted
// from the form.
//
// The second world of every pair is built ONLY by decoding the first's bytes.
// A constructed twin would agree by having been given the same inputs, which is
// a different and much weaker fact.
//
// The terrains hold obstacles that force a detour, since a straight walk cannot
// show a route diverging: two searches from the same cell to the same goal over
// open ground agree whatever either of them stores.

import (
	"bytes"
	"testing"
)

// rtpSide is the corpus's map size, and rtpTicks how far each run is advanced.
// Long enough for the group to cross the obstacles and for some of it to
// arrive, short enough that every cut of every run is affordable.
const rtpSide int32 = 20
const rtpTicks = 18

// rtpGrid draws one terrain: blobs of blocked cells laid out from the seeded
// generator, with a clear border so that no unit is walled in where it stands.
func rtpGrid(r *rng) []byte {
	g := make([]byte, rtpSide*rtpSide)
	blobs := 4 + int(r.next()%4)
	for k := 0; k < blobs; k++ {
		x0 := 3 + int32(r.next()%uint64(rtpSide-7))
		y0 := 3 + int32(r.next()%uint64(rtpSide-7))
		w := 1 + int32(r.next()%4)
		h := 1 + int32(r.next()%4)
		for y := y0; y < y0+h && y < rtpSide-1; y++ {
			for x := x0; x < x0+w && x < rtpSide-1; x++ {
				g[y*rtpSide+x] = blockGround
			}
		}
	}
	return g
}

// rtpRun is one scenario: a grid, the units on it, and the orders they open
// with. Units start on the edges and are sent across, so their routes cross each
// other and the obstacles alike.
type rtpRun struct {
	grid []byte
	ents []Entity
	cmds []Command
}

func rtpDraw(t *testing.T, r *rng) rtpRun {
	t.Helper()
	g := rtpGrid(r)
	free := func() (int32, int32) {
		for {
			x := int32(r.next() % uint64(rtpSide))
			y := int32(r.next() % uint64(rtpSide))
			if g[y*rtpSide+x]&blockGround == 0 {
				return x, y
			}
		}
	}

	out := rtpRun{grid: g}
	taken := map[cell]bool{}
	for k := 0; k < 5; k++ {
		var x, y int32
		for {
			x, y = free()
			if !taken[cell{x, y}] {
				break
			}
		}
		taken[cell{x, y}] = true
		id := EntityID(1 + 2*k)
		out.ents = append(out.ents, Entity{ID: id, X: x, Y: y})
		tx, ty := free()
		out.cmds = append(out.cmds, Command{Entity: id, X: tx, Y: ty})
	}
	return out
}

// rtpStepAll advances a world one tick with cmds and returns its form and
// digest, so a run is recorded in the two things a comparison needs.
func rtpForm(t *testing.T, w *World) ([]byte, uint64) {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return b, w.Hash()
}

// TestEveryCutOfEveryRunDecodesAndWalksOnIdentically is AC-7, and SC-2. Each run
// is recorded tick by tick; then, from every one of those ticks, a world is
// decoded out of the recorded bytes alone and stepped to the end of the run
// beside the record — comparing the whole byte form at every tick and not the
// digest alone, so a divergence says which bytes moved.
func TestEveryCutOfEveryRunDecodesAndWalksOnIdentically(t *testing.T) {
	const runs = 6
	r := &rng{state: 0xc07a1}

	routed, arrivals := 0, 0
	for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
		for k := 0; k < runs; k++ {
			run := rtpDraw(t, r)
			w := mustWorldGrid(t, r.next(), Bounds{Width: rtpSide, Height: rtpSide}, mode, run.grid, run.ents)

			// The record: the world's form and digest at every tick, index 0 being
			// the world before anything is stepped.
			forms := make([][]byte, 0, rtpTicks+1)
			digests := make([]uint64, 0, rtpTicks+1)
			f, d := rtpForm(t, w)
			forms, digests = append(forms, f), append(digests, d)
			for tick := 1; tick <= rtpTicks; tick++ {
				if tick == 1 {
					Step(w, run.cmds)
				} else {
					Step(w, nil)
				}
				for i := range w.routes {
					if len(w.routes[i]) != 0 {
						routed++
					}
				}
				for _, e := range w.Entities() {
					if !e.HasTarget {
						arrivals++
					}
				}
				f, d = rtpForm(t, w)
				forms, digests = append(forms, f), append(digests, d)
			}

			for cut := 0; cut < len(forms); cut++ {
				twin := mustWorldGrid(t, 99, Bounds{Width: 1, Height: 1}, ModeCanonical, nil, nil)
				if err := twin.UnmarshalBinary(forms[cut]); err != nil {
					t.Fatalf("mode %d run %d: the world at tick %d does not decode: %v", mode, k, cut, err)
				}
				if got := twin.Hash(); got != digests[cut] {
					t.Fatalf("mode %d run %d: the world decoded at tick %d hashes %#016x, the record %#016x",
						mode, k, cut, got, digests[cut])
				}

				for tick := cut + 1; tick <= rtpTicks; tick++ {
					var cmds []Command
					if tick == 1 {
						cmds = run.cmds
					}
					Step(twin, cmds)
					got, gotDigest := rtpForm(t, twin)
					if gotDigest != digests[tick] {
						t.Fatalf("mode %d run %d: cut at tick %d, then tick %d: the decoded world hashes "+
							"%#016x and the record %#016x", mode, k, cut, tick, gotDigest, digests[tick])
					}
					if !bytes.Equal(got, forms[tick]) {
						t.Fatalf("mode %d run %d: cut at tick %d, then tick %d: the two forms differ\n % x\n % x",
							mode, k, cut, tick, got, forms[tick])
					}
				}
			}
		}
	}

	// The corpus's own condition. A run whose units never hold a route, or never
	// finish an order, would cross the boundary with nothing this criterion is
	// about and would agree for reasons that have nothing to do with the routes.
	if routed == 0 || arrivals == 0 {
		t.Errorf("the corpus carried %d stored route(s) across its ticks and ended %d order(s) — "+
			"a run holding neither witnesses nothing here", routed, arrivals)
	}
}

// TestTheCorpusForcesDetours is the other condition, and it is separate because
// it is about the TERRAIN rather than about the run: a route that is a straight
// line has tails that are straight lines, and a world of open ground would carry
// the criterion above with the route section deleted from the form.
//
// A detour is measured as a walk longer than the Chebyshev distance the order
// spans — a unit that took more ticks to arrive than a clear line would have
// cost. It is asserted over the corpus rather than per run, since which grid
// happens to force one is the generator's business.
func TestTheCorpusForcesDetours(t *testing.T) {
	r := &rng{state: 0xc07a1} // the same corpus, drawn again
	detours := 0

	for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
		for k := 0; k < 6; k++ {
			run := rtpDraw(t, r)
			w := mustWorldGrid(t, r.next(), Bounds{Width: rtpSide, Height: rtpSide}, mode, run.grid, run.ents)

			start := w.Entities()
			straight := make([]int32, len(start))
			for i, e := range start {
				dx, dy := run.cmds[i].X-e.X, run.cmds[i].Y-e.Y
				if dx < 0 {
					dx = -dx
				}
				if dy < 0 {
					dy = -dy
				}
				straight[i] = dx
				if dy > dx {
					straight[i] = dy
				}
			}

			for tick := 1; tick <= rtpTicks; tick++ {
				if tick == 1 {
					Step(w, run.cmds)
				} else {
					Step(w, nil)
				}
				for i, e := range w.Entities() {
					if !e.HasTarget && e.X == run.cmds[i].X && e.Y == run.cmds[i].Y && int32(tick) > straight[i] {
						detours++
						straight[i] = 1 << 30 // counted once
					}
				}
			}
		}
	}

	if detours == 0 {
		t.Error("no unit in the corpus arrived later than a clear line would have taken, so every " +
			"route in it is a straight walk and the criterion above is not measuring a detour")
	}
	t.Logf("the corpus produced %d arrival(s) that took longer than the straight line", detours)
}
