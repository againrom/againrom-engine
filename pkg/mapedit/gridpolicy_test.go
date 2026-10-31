package mapedit_test

import (
	"bytes"
	"fmt"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// Each SetCell policy on the last cell of the 3x2 fixture grid leaves its own
// bytes: raw one cell; normalized on tiles also the save normalisation.
func TestGridPoliciesWriteTheirOwnBytes(t *testing.T) {
	const last = fixH*fixW - 1
	v := fixtureVariants()[0]
	before := richFixture(v.order, v.nUnits)

	view := func(t *testing.T, ed *mapedit.Editor) *alm.Map {
		t.Helper()
		m, err := ed.Map()
		if err != nil {
			t.Fatalf("view: %v", err)
		}
		return m
	}
	base := view(t, newEditor(t, before))

	t.Run("raw tile", func(t *testing.T) {
		ed := newEditor(t, before)
		if err := ed.SetCell(mapedit.GridRaw, mapedit.GridTile, fixW-1, fixH-1, gridNewTile); err != nil {
			t.Fatal(err)
		}
		m := view(t, ed)
		for i, w := range m.Tiles {
			want := base.Tiles[i]
			if i == last {
				want = gridNewTile
			}
			if w != want {
				t.Errorf("tile %d = %#x, want %#x", i, w, want)
			}
		}
		if m.Meta.Bitmask != base.Meta.Bitmask {
			t.Errorf("the raw write moved the tile-group mask: %#x -> %#x", base.Meta.Bitmask, m.Meta.Bitmask)
		}
	})

	t.Run("normalized tile", func(t *testing.T) {
		ed := newEditor(t, before)
		if err := ed.SetCell(mapedit.GridNormalized, mapedit.GridTile, fixW-1, fixH-1, gridNewTile); err != nil {
			t.Fatal(err)
		}
		m := view(t, ed)
		want := make([]uint16, len(base.Tiles))
		for i := range want {
			want[i] = gridNewTile&^0x2000 | base.Tiles[i]&0x2000
		}
		want[last] = gridNewTile
		for i, w := range m.Tiles {
			if w != want[i] {
				t.Errorf("tile %d = %#x, want %#x", i, w, want[i])
			}
		}
		if mask := alm.TileGroupMask(want); m.Meta.Bitmask != mask {
			t.Errorf("tile-group mask = %#x, want %#x", m.Meta.Bitmask, mask)
		}
	})

	for _, layer := range []struct {
		name  string
		which mapedit.GridLayer
		v     uint16
		cell  func(*alm.Map) []uint8
	}{
		{"altitude", mapedit.GridAltitude, gridNewAltitude, func(m *alm.Map) []uint8 { return m.Altitudes }},
		{"overlay", mapedit.GridOverlay, gridNewOverlay, func(m *alm.Map) []uint8 { return m.Overlay }},
	} {
		t.Run("byte layer "+layer.name, func(t *testing.T) {
			var got [2][]byte
			for p, policy := range []mapedit.GridPolicy{mapedit.GridRaw, mapedit.GridNormalized} {
				ed := newEditor(t, before)
				if err := ed.SetCell(policy, layer.which, fixW-1, fixH-1, layer.v); err != nil {
					t.Fatal(err)
				}
				got[p] = ed.Bytes()
				cells := layer.cell(view(t, ed))
				wantCells := append([]uint8(nil), layer.cell(base)...)
				wantCells[last] = uint8(layer.v)
				if !bytes.Equal(cells, wantCells) {
					t.Errorf("%s policy: cells % x, want % x", fmt.Sprint(policy), cells, wantCells)
				}
			}
			if !bytes.Equal(got[0], got[1]) {
				t.Error("the two policies left different bytes on a byte layer")
			}
		})
	}

	t.Run("an equal value", func(t *testing.T) {
		same := base.Tiles[last]
		raw, norm := newEditor(t, before), newEditor(t, before)
		if err := raw.SetCell(mapedit.GridRaw, mapedit.GridTile, fixW-1, fixH-1, same); err != nil {
			t.Fatal(err)
		}
		if err := norm.SetCell(mapedit.GridNormalized, mapedit.GridTile, fixW-1, fixH-1, same); err != nil {
			t.Fatal(err)
		}
		if !raw.CanUndo() {
			t.Error("the raw policy did not record an equal write")
		}
		if norm.CanUndo() {
			t.Error("the normalized policy recorded a write that changed nothing")
		}
	})

	t.Run("a value a byte cannot hold", func(t *testing.T) {
		ed := newEditor(t, before)
		for _, policy := range []mapedit.GridPolicy{mapedit.GridRaw, mapedit.GridNormalized} {
			if ed.SetCell(policy, mapedit.GridAltitude, 0, 0, 0x100) == nil || ed.CanUndo() {
				t.Errorf("policy %d accepted 0x100 for a byte layer", policy)
			}
		}
	})
}
