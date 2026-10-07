package terrain_test

import (
	"fmt"
	"image"
	"testing"

	"againrom/pkg/render/terrain"
)

// sackFrame is a frame of the given size, distinct in identity from any other
// call's — the placement arithmetic below reads only Width and Height, so the
// pixels are never populated.
func sackFrame(w, h int) *terrain.StaticFrame {
	return &terrain.StaticFrame{Width: w, Height: h}
}

// AC-5: the placed frame's OWN centre pixel — not merely what Ground() sums —
// lands on the cell's ground point in the flat geometry. The frame is given
// odd extents so the /2 truncation the ground point shares with every other
// cell-anchored placement is exercised rather than sidestepped.
func TestSackPlaceCentrePixelAtGroundPoint(t *testing.T) {
	f := sackFrame(41, 17)
	const col, row = 3, 5
	p, ok := terrain.SackPlace(col, row, f, 0, 0)
	if !ok {
		t.Fatalf("ok = false for a real frame")
	}
	if p.Cell != (image.Point{X: col, Y: row}) {
		t.Errorf("Cell = %v, want (%d,%d)", p.Cell, col, row)
	}
	if p.Frame != f {
		t.Errorf("Frame is not the pointer SackPlace was handed")
	}

	want := image.Point{X: col*structCellSize + structCellSize/2, Y: row*structCellSize + structCellSize/2}
	if g := p.Ground(); g != want {
		t.Errorf("Ground() = %v, want %v — the cell's ground point (AC-5)", g, want)
	}
	// The frame's own centre pixel, computed independently of Ground()'s own
	// TopLeft+Anchor sum, so a wrong Anchor that happened to still satisfy
	// Ground() could not hide behind that accessor alone.
	centre := p.TopLeft.Add(image.Point{X: f.Width / 2, Y: f.Height / 2})
	if centre != want {
		t.Errorf("frame centre = %v, want %v — the frame's centre pixel must BE the ground point (AC-5)", centre, want)
	}
}

func TestSackPlaceFlatVsDisplaced(t *testing.T) {
	f := sackFrame(32, 32)
	const col, row = 2, 9
	const lift, originY = 7, 3

	flat, ok1 := terrain.SackPlace(col, row, f, 0, 0)
	displaced, ok2 := terrain.SackPlace(col, row, f, lift, originY)
	if !ok1 || !ok2 {
		t.Fatalf("ok = false for a real frame")
	}
	if dx := displaced.TopLeft.X - flat.TopLeft.X; dx != 0 {
		t.Errorf("X differs by %d, want 0 (P-2)", dx)
	}
	if dy, want := displaced.TopLeft.Y-flat.TopLeft.Y, -(lift + originY); dy != want {
		t.Errorf("Y differs by %d, want %d = -(lift+originY) (AC-6, P-2)", dy, want)
	}
}

func TestSackPlaceNilFrame(t *testing.T) {
	p, ok := terrain.SackPlace(1, 1, nil, 0, 0)
	if ok {
		t.Errorf("ok = true for a nil frame")
	}
	if p != (terrain.StaticPlacement{}) {
		t.Errorf("placement = %+v, want the zero value", p)
	}
}

// A sack placement carries no class and no mirror: it neither animates nor
// reflects, so nothing downstream that reads Class to re-select a frame has
// anything to act on.
func TestSackPlaceCarriesNoClass(t *testing.T) {
	p, ok := terrain.SackPlace(0, 0, sackFrame(16, 16), 0, 0)
	if !ok {
		t.Fatalf("ok = false for a real frame")
	}
	if p.Class != nil {
		t.Errorf("Class = %v, want nil — a sack has none to carry", p.Class)
	}
	if p.Mirror {
		t.Errorf("Mirror = true, want false — a sack never reflects")
	}
}

// sackAt is a sack placement standing on a cell, built the way structObject
// builds an object or entity fixture: only Cell reaches the merge, which
// compares rectangle rows and nothing else.
func sackAt(col, row int) terrain.StaticPlacement {
	return terrain.StaticPlacement{Cell: image.Point{X: col, Y: row}}
}

// depthKinds renders a merged order as one readable string per ref, exactly as
// structures_test.go's own depthCells does, extended with the PlaneSack kind
// this story adds — kept apart from depthCells rather than widening it, since
// depthCells' every existing call site is frozen and must not change shape.
func depthKinds(order []terrain.DepthRef, structures []terrain.StructurePlacement,
	objects, sacks, entities []terrain.StaticPlacement) []string {
	out := make([]string, 0, len(order))
	for _, ref := range order {
		switch ref.Kind {
		case terrain.PlaneStructure:
			p := structures[ref.Index]
			out = append(out, fmt.Sprintf("S(%d,%d)", p.Cell.X, p.Cell.Y))
		case terrain.PlaneSack:
			p := sacks[ref.Index]
			out = append(out, fmt.Sprintf("K(%d,%d)", p.Cell.X, p.Cell.Y))
		case terrain.PlaneEntity:
			p := entities[ref.Index]
			out = append(out, fmt.Sprintf("E(%d,%d)", p.Cell.X, p.Cell.Y))
		default:
			p := objects[ref.Index]
			out = append(out, fmt.Sprintf("O(%d,%d)", p.Cell.X, p.Cell.Y))
		}
	}
	return out
}

// corpseAt and unitAt are entity placements carrying the two tiers an entity
// can stand in — sackAt's own shape with a DepthTie named, since only Cell and
// DepthTie reach the merge.
func corpseAt(col, row int) terrain.StaticPlacement {
	p := sackAt(col, row)
	p.DepthTie = terrain.TieCorpse
	return p
}

func unitAt(col, row int) terrain.StaticPlacement {
	p := sackAt(col, row)
	p.DepthTie = terrain.TieUnit
	return p
}

// depthRefs renders a merged order as kind AND INDEX, which depthKinds cannot:
// the cases below put a corpse and a living unit on ONE CELL, so both would
// print the same "E(x,y)" and the assertion could not tell which of them came
// out first — the entire question. It is the sequence the owner's rule is
// about, named exactly.
func depthRefs(order []terrain.DepthRef) []string {
	out := make([]string, 0, len(order))
	for _, ref := range order {
		switch ref.Kind {
		case terrain.PlaneStructure:
			out = append(out, fmt.Sprintf("S#%d", ref.Index))
		case terrain.PlaneSack:
			out = append(out, fmt.Sprintf("K#%d", ref.Index))
		case terrain.PlaneEntity:
			out = append(out, fmt.Sprintf("E#%d", ref.Index))
		default:
			out = append(out, fmt.Sprintf("O#%d", ref.Index))
		}
	}
	return out
}

// THE OWNER'S RULING, top to bottom on one cell: «на клетке
// стоит герой или юнит — он выше всех. если
// есть мешок, то он под героем. если есть
// труп, то он под мешочком.» A living unit above
// everything, the sack under it, the corpse under the sack — and the list
// is painted in order, so first out is underneath.
//
// THE THREE-WAY CASE IS ASSERTED DIRECTLY and not left to compose out of the
// two pairs, because a two-way fix passes the pairs by accident and only a
// three-tier one passes all three. And every case runs TWICE, once on each
// side of an art ref: the per-ref flush and the tail drain are two copies of
// this comparison, and 0111's bug was in both of them.
//
// TO CONFIRM THESE WITNESS THE FIX, put sackFirst back to `sRow <= eRow`
// (structures.go): "corpse then sack" and the three-way case redden with the
// sack ahead of the corpse, while "sack then living unit" stays green — which
// is the shape of the defect, one half of the old tie having always been right.
func TestDepthOrderPutsTheSackAboveACorpseAndUnderALivingUnit(t *testing.T) {
	// One object on row 4, so a trio on row 1 goes through the per-ref flush
	// and a trio on row 8 goes through the tail drain. Both must agree.
	objects := []terrain.StaticPlacement{structObject(0, 4)}
	order := terrain.PlaneOrder(nil, objects)

	for _, side := range []struct {
		name string
		row  int
		lead []string // what precedes the trio in the emitted order
	}{
		{"through the per-ref flush", 1, nil},
		{"through the tail drain", 8, []string{"O#0"}},
	} {
		t.Run(side.name, func(t *testing.T) {
			for _, c := range []struct {
				name     string
				entities []terrain.StaticPlacement
				sacks    []terrain.StaticPlacement
				trio     []string
			}{
				{
					"a corpse and a sack on one cell: the corpse first, so the sack is painted over it",
					[]terrain.StaticPlacement{corpseAt(3, side.row)},
					[]terrain.StaticPlacement{sackAt(3, side.row)},
					[]string{"E#0", "K#0"},
				},
				{
					"a sack and a living unit on one cell: the sack first, so the unit stands over it",
					[]terrain.StaticPlacement{unitAt(3, side.row)},
					[]terrain.StaticPlacement{sackAt(3, side.row)},
					[]string{"K#0", "E#0"},
				},
				{
					"a corpse, a sack and a living unit on one cell: all three, in that order",
					// The living unit is FIRST in the caller's slice, so the
					// tie sort has to move it behind the corpse — a stable
					// sort on row alone would leave it in front and the sack
					// could then only land on one side of the pair.
					[]terrain.StaticPlacement{unitAt(3, side.row), corpseAt(3, side.row)},
					[]terrain.StaticPlacement{sackAt(3, side.row)},
					[]string{"E#1", "K#0", "E#0"},
				},
			} {
				t.Run(c.name, func(t *testing.T) {
					got := depthRefs(terrain.DepthOrder(nil, order, nil, objects, c.entities, c.sacks))
					want := append(append([]string{}, side.lead...), c.trio...)
					if side.row == 1 {
						want = append(want, "O#0")
					}
					if len(got) != len(want) {
						t.Fatalf("order = %v, want %v", got, want)
					}
					for i := range want {
						if got[i] != want[i] {
							t.Fatalf("order = %v, want %v", got, want)
						}
					}
				})
			}
		})
	}
}

// A caller that names NO tie at all gets exactly the order 0111 gave it: every
// placement is TieGround, the full tie favours the sack, and the sack goes
// first. It is what keeps every frozen `want` in this file and in
// structures_test.go meaningful — those fixtures name no tie, so they pin the
// OLD behaviour and would have to be rewritten if the tie were not inert for
// them. Asserted here rather than inferred from their passing.
func TestDepthOrderWithNoTiesNamedKeepsTheSackFirst(t *testing.T) {
	objects := []terrain.StaticPlacement{structObject(0, 4)}
	entities := []terrain.StaticPlacement{sackAt(3, 1)}
	sacks := []terrain.StaticPlacement{sackAt(3, 1)}

	got := depthRefs(terrain.DepthOrder(nil, terrain.PlaneOrder(nil, objects), nil, objects, entities, sacks))
	want := []string{"K#0", "E#0", "O#0"}
	if len(got) != len(want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestDepthOrderMergesSacksByRow(t *testing.T) {
	tall := structClass(1, 1, 3)
	flat := structClass(1, 1, 1)
	flat.Flat = true
	set := structSet(map[byte]*terrain.StructureClass{1: tall, 2: flat})

	g := structGrid(10, 10,
		structAt(4, 2, 1), // the tall one, row 2
		structAt(4, 7, 2), // the flat one, on the last row of the map
	)
	structures, _, _ := terrain.StructurePlacements(g, set, nil, 0)
	objects := []terrain.StaticPlacement{structObject(1, 0), structObject(0, 5)}

	entities := []terrain.StaticPlacement{
		sackAt(3, 5), // shares the object's row
		sackAt(2, 2), // the structure's own row, alongside a sack there
		sackAt(7, 1), // between the two objects
		sackAt(6, 6), // beyond every ref: the tail drain
	}
	sacks := []terrain.StaticPlacement{
		sackAt(5, 2), // the structure's own row, alongside the entity there
		sackAt(9, 6), // tail drain, row 6 — shares a row with the tail entity
		sackAt(1, 6), //
		sackAt(0, 3), // a row with no entity and no ref of its own: a lone flush
	}

	order := terrain.DepthOrder(nil, terrain.PlaneOrder(structures, objects), structures, objects, entities, sacks)
	got := depthKinds(order, structures, objects, sacks, entities)
	want := []string{
		"S(4,7)", // the flat pass, first whatever its row
		"O(1,0)",
		"E(7,1)",
		"S(4,2)", "S(4,2)", "S(4,2)", // the tall structure's three strip entries
		"K(5,2)", // the structure's own row: the sack, after both art planes
		"E(2,2)", // the same row: the entity, after the sack
		"K(0,3)", // a row of its own, between the structure's row and the object's
		"O(0,5)",
		"E(3,5)", // the object's own row: in front of it
		"K(9,6)", // the tail drain: earlier list-order sack of row 6 first
		"K(1,6)", // the tail drain: later list-order sack of the SAME row
		"E(6,6)", // the tail drain: row 6's entity, after both its sacks
	}
	if len(got) != len(want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}

	t.Run("every drawable appears exactly once", func(t *testing.T) {
		seen := map[terrain.DepthRef]bool{}
		for _, ref := range order {
			if seen[ref] {
				t.Fatalf("%+v appears twice", ref)
			}
			seen[ref] = true
		}
		if len(seen) != len(structures)+len(objects)+len(entities)+len(sacks) {
			t.Fatalf("%d refs, want one per drawable", len(seen))
		}
	})

	t.Run("the caller's sack slice is not reordered", func(t *testing.T) {
		want := []image.Point{{X: 5, Y: 2}, {X: 9, Y: 6}, {X: 1, Y: 6}, {X: 0, Y: 3}}
		for i, w := range want {
			if sacks[i].Cell != w {
				t.Fatalf("sack %d is now at %v, want the caller's own %v — the sort runs over an "+
					"index permutation and must not write through the caller's slice", i, sacks[i].Cell, w)
			}
		}
	})
}

// SC-4's other half: an EMPTY sack stream — nil or a zero-length slice — must
// not perturb the merge at all, so a caller that supplies no sacks (0111 T1's
// own pkg/ui call site) reproduces exactly the order it produced before this
// story. structures_test.go's own three DepthOrder tests already pin that with
// a frozen `want`; this pins the general property — nil and `{}` agree with
// each other and with a plain PlaneOrder + entities merge, over a fixture with
// entities on the very rows a stray sack contribution would be visible on.
func TestDepthOrderEmptySackStreamChangesNothing(t *testing.T) {
	set := structSet(map[byte]*terrain.StructureClass{1: structClass(1, 1, 3)})
	structures, _, _ := terrain.StructurePlacements(structGrid(8, 8, structAt(4, 2, 1)), set, nil, 0)
	objects := []terrain.StaticPlacement{structObject(1, 0), structObject(0, 5)}
	entities := []terrain.StaticPlacement{sackAt(3, 5), sackAt(2, 2), sackAt(7, 1)}
	order := terrain.PlaneOrder(structures, objects)

	baseline := depthKinds(terrain.DepthOrder(nil, order, structures, objects, entities, nil), structures, objects, nil, entities)
	for _, sacks := range [][]terrain.StaticPlacement{nil, {}} {
		got := depthKinds(terrain.DepthOrder(nil, order, structures, objects, entities, sacks), structures, objects, sacks, entities)
		if len(got) != len(baseline) {
			t.Fatalf("sacks=%#v: order = %v, want %v", sacks, got, baseline)
		}
		for i := range baseline {
			if got[i] != baseline[i] {
				t.Fatalf("sacks=%#v: order = %v, want %v", sacks, got, baseline)
			}
		}
	}
}
