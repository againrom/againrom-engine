package mapedit_test

// The grid setters — the model's first mutation, and the first exercise of the
// carried-bytes invariant. Every diff is taken through fixture_test.go's
// comparator with the target region located by the frame walk, so no case ever
// asks the model where it wrote.

import (
	"bytes"
	"fmt"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// The values these cases write. Each is outside the rich fixture's declared
// table, so a setter that landed on the wrong cell cannot pass by writing a
// value that was already sitting there.
const (
	gridNewTile     = 0x1abc
	gridNewAltitude = 0x2f
	gridNewOverlay  = 0x3e
)

// gridLayers is the three layers as this story addresses them: the setter, the
// typeId of the record it writes into, that record's cell width, and the cell
// the shipped decoder reads back at a row-major index.
var gridLayers = []struct {
	name   string
	typeID uint32
	elem   int
	set    func(ed *mapedit.Editor, x, y int) error
	cell   func(m *alm.Map, idx int) uint64
	want   uint64
}{
	{
		"tile", 1, 2,
		func(ed *mapedit.Editor, x, y int) error { return ed.SetTile(x, y, gridNewTile) },
		func(m *alm.Map, idx int) uint64 { return uint64(m.Tiles[idx]) },
		gridNewTile,
	},
	{
		"altitude", 2, 1,
		func(ed *mapedit.Editor, x, y int) error { return ed.SetAltitude(x, y, gridNewAltitude) },
		func(m *alm.Map, idx int) uint64 { return uint64(m.Altitudes[idx]) },
		gridNewAltitude,
	},
	{
		"overlay", 3, 1,
		func(ed *mapedit.Editor, x, y int) error { return ed.SetOverlay(x, y, gridNewOverlay) },
		func(m *alm.Map, idx int) uint64 { return uint64(m.Overlay[idx]) },
		gridNewOverlay,
	},
}

// cellImage is the little-endian image the setter must leave in an elem-wide
// cell for value v. It is spelled out here rather than taken from the model: a
// witness that asked the model to encode the expectation would agree with a
// byte-swapped setter.
func cellImage(elem int, v uint64) []byte {
	img := make([]byte, elem)
	for i := 0; i < elem; i++ {
		img[i] = byte(v >> (8 * i))
	}
	return img
}

// TestGridSetterWritesItsCellAndCarriesEveryOtherByte is AC-2's grid half, each
// mutation on a fresh load: the diff is confined to the cell, the cell holds the
// value's own little-endian image, and the shipped decoder reads the new value
// back at that (x,y). The cells include (2,1) — the last one of a 3x2 grid — so
// a transposed index cannot land on a valid cell of the other axis.
func TestGridSetterWritesItsCellAndCarriesEveryOtherByte(t *testing.T) {
	cells := []struct{ x, y int }{{0, 0}, {1, 0}, {2, 1}}

	for _, v := range fixtureVariants() {
		for _, layer := range gridLayers {
			for _, c := range cells {
				t.Run(fmt.Sprintf("%s/%s/(%d,%d)", v.name, layer.name, c.x, c.y), func(t *testing.T) {
					before := richFixture(v.order, v.nUnits)
					ed := newEditor(t, before)

					if err := layer.set(ed, c.x, c.y); err != nil {
						t.Fatalf("set %s (%d,%d): %v", layer.name, c.x, c.y, err)
					}
					after := ed.Bytes()
					if len(after) != len(before) {
						t.Fatalf("a fixed-length edit changed the image size: %d -> %d", len(before), len(after))
					}

					// The target region, computed from the contract in both
					// images. A fixed-length edit relocates nothing, so the two
					// must agree — and that they do is asserted, not assumed.
					fsBefore, fsAfter := walkFrame(t, before), walkFrame(t, after)
					target := fsBefore.cell(t, layer.typeID, c.x, c.y, layer.elem)
					if got := fsAfter.cell(t, layer.typeID, c.x, c.y, layer.elem); got != target {
						t.Fatalf("the cell moved: %+v -> %+v", target, got)
					}

					if ok, why := carriedIdentical(before, after, []region{target}, []region{target}); !ok {
						t.Errorf("a byte outside the cell changed: %s", why)
					}

					want := cellImage(layer.elem, layer.want)
					if got := fsAfter.at(t, after, target); !bytes.Equal(got, want) {
						t.Errorf("the cell holds % x, want % x", got, want)
					}
					if bytes.Equal(fsBefore.at(t, before, target), want) {
						t.Fatal("the cell already held the value written, so this case witnesses no write")
					}

					m, err := ed.Map()
					if err != nil {
						t.Fatalf("the view failed after the edit: %v", err)
					}
					if got := layer.cell(m, c.y*fixW+c.x); got != layer.want {
						t.Errorf("the view's %s cell (%d,%d) = %#x, want %#x", layer.name, c.x, c.y, got, layer.want)
					}
				})
			}
		}
	}
}

// TestGridSetterRejectsACellOutsideTheGrid is AC-4's out-of-range-cell case,
// run against three history states rather than only against a fresh model.
// The undone state is the discriminating one: a setter that built and applied
// its edit before validating would truncate the redo history on its way to
// returning an error, and only a model with something to redo can see that.
func TestGridSetterRejectsACellOutsideTheGrid(t *testing.T) {
	outside := []struct{ x, y int }{
		{-1, 0}, {0, -1}, {-1, -1},
		{fixW, 0}, {0, fixH}, {fixW, fixH},
		{fixW - 1, fixH}, {fixW, fixH - 1},
		{1 << 20, 0}, {0, 1 << 20},
	}
	states := []struct {
		name  string
		setUp func(t *testing.T, ed *mapedit.Editor)
	}{
		{"a fresh model", func(*testing.T, *mapedit.Editor) {}},
		{"one accepted edit", func(t *testing.T, ed *mapedit.Editor) {
			if err := ed.SetOverlay(0, 0, gridNewOverlay); err != nil {
				t.Fatalf("the set-up edit failed: %v", err)
			}
		}},
		{"one edit, undone", func(t *testing.T, ed *mapedit.Editor) {
			if err := ed.SetOverlay(0, 0, gridNewOverlay); err != nil {
				t.Fatalf("the set-up edit failed: %v", err)
			}
			if !ed.Undo() {
				t.Fatal("the set-up Undo reported nothing to undo")
			}
		}},
	}

	for _, state := range states {
		for _, layer := range gridLayers {
			for _, c := range outside {
				t.Run(fmt.Sprintf("%s/%s/(%d,%d)", state.name, layer.name, c.x, c.y), func(t *testing.T) {
					ed := newEditor(t, richFixture(orderType6First, fixUnitsMax))
					state.setUp(t, ed)
					rejectionChangesNothing(t, ed, func() error { return layer.set(ed, c.x, c.y) })
				})
			}
		}
	}
}

// TestSetOverlayRejectsAMapWithNoOverlayRecord is AC-8's grid half — the branch
// that was declared unreachable when every accepted stream carried all ten
// records, and is reachable again for exactly one of the three layers.
//
// type-1 and type-2 are not among them: alm refuses a stream missing either, so
// a case for their absence would be a case no input can reach. type-3 is the
// asymmetry, and the view is why it needs saying at all — Overlay is a full W*H
// plane on these maps too, manufactured by the reader, so an editor that trusted
// the decoded view would paint a cell of a record the file does not have.
//
// The cells are the ones the accepting cases use, so the rejection is not a
// coordinate rejection wearing another name.
func TestSetOverlayRejectsAMapWithNoOverlayRecord(t *testing.T) {
	cells := []struct{ x, y int }{{0, 0}, {2, 1}}

	for _, r := range thinRosters() {
		if r.overlay {
			continue
		}
		for _, c := range cells {
			t.Run(fmt.Sprintf("%s/(%d,%d)", r.name, c.x, c.y), func(t *testing.T) {
				data := r.image()
				ed := newEditor(t, data)

				m, err := ed.Map()
				if err != nil {
					t.Fatalf("the view failed: %v", err)
				}
				if m.Present(3) {
					t.Fatal("the fixture carries a type-3 record, so this case witnesses no absence")
				}
				if len(m.Overlay) != fixW*fixH {
					t.Fatalf("the view's overlay is %d cells, want the manufactured %d: without it the rejection is unremarkable",
						len(m.Overlay), fixW*fixH)
				}

				rejectionChangesNothing(t, ed, func() error { return ed.SetOverlay(c.x, c.y, gridNewOverlay) })

				// The layers that ARE in the file are unaffected by the
				// refusal: this is a per-record answer, not a mode.
				if err := ed.SetTile(c.x, c.y, gridNewTile); err != nil {
					t.Errorf("SetTile was refused on the same map: %v", err)
				}
				if err := ed.SetAltitude(c.x, c.y, gridNewAltitude); err != nil {
					t.Errorf("SetAltitude was refused on the same map: %v", err)
				}
				reopensIdentically(t, "after the accepted edits", ed.Bytes())
			})
		}
	}
}
