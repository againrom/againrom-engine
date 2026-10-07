package game

import (
	"bytes"
	"image"
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

func TestWallFireAmbientCellsReadsOnlyLiveSpellThreeCoverage(t *testing.T) {
	effects := []sim.CellEffect{
		{Spell: 3, Cells: [][2]int32{{4, 5}, {6, 7}}},
		{Spell: 12, Cells: [][2]int32{{8, 9}}},
		{Spell: 3, Cells: [][2]int32{{4, 5}}},
	}
	want := []image.Point{image.Pt(4, 5), image.Pt(6, 7), image.Pt(4, 5)}
	if got := wallFireAmbientCells(effects); !reflect.DeepEqual(got, want) {
		t.Fatalf("wallFireAmbientCells = %v, want %v", got, want)
	}
	if effects[0].Cells[0] != [2]int32{4, 5} {
		t.Fatal("presentation projection mutated the canonical effect")
	}
}

func TestAmbientProjectionPushDoesNotChangeCanonicalBytesOrHash(t *testing.T) {
	mw, _ := readoutWorld(t, sim.Entity{ID: 1, X: 4, Y: 5})
	beforeBytes := marshalWorld(t, mw.world)
	beforeHash := mw.world.Hash()
	mw.push()
	if got := mw.world.Hash(); got != beforeHash {
		t.Fatalf("ambient presentation push changed hash from %#x to %#x", beforeHash, got)
	}
	if got := marshalWorld(t, mw.world); !bytes.Equal(got, beforeBytes) {
		t.Fatal("ambient presentation push changed canonical bytes")
	}
}
