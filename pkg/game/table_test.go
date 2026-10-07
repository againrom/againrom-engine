package game_test

// Tests for the definition-table load. Every fixture is a synthetic archive
// written to a temp dir; no game install is read.

import (
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
	"againrom/pkg/vfs"
)

// worldFS opens a container filesystem over one world archive built from the
// given entries, so a case can plant a table that is absent, unreadable, or
// whatever else it needs.
//
// It lays a scenario archive beside it carrying an npc.reg, because the table
// load reads that file too and every case here is about the world archive's
// half. A case about the NPC half plants its own scenario archive.
func worldFS(t *testing.T, files []synth.File) *vfs.FS {
	t.Helper()
	return worldAndScenarioFS(t, files, []synth.File{{Path: "npc.reg", Data: synth.NPCReg(nil)}})
}

func worldAndScenarioFS(t *testing.T, world, scenario []synth.File) *vfs.FS {
	t.Helper()
	dir := installDir(t, map[string][]byte{
		game.WorldArchive:    synth.Archive(world),
		game.ScenarioArchive: synth.Archive(scenario),
	})
	fsys, err := vfs.Open([]string{dir + "/" + game.WorldArchive, dir + "/" + game.ScenarioArchive}, nil)
	if err != nil {
		t.Fatalf("vfs.Open: %v", err)
	}
	return fsys
}

// TestLoadTableReadsTheTwoSearchedCollections is the load's happy path: the
// collections a placement resolves against arrive with the rows the archive
// holds.
func TestLoadTableReadsTheTwoSearchedCollections(t *testing.T) {
	units := []synth.DataBinRow{
		{Name: "one", Params: make([]int32, 38)},
		{Name: "two", Params: make([]int32, 38)},
	}
	humans := []synth.DataBinRow{{Name: "a-human", Params: make([]int32, 38)}}

	tbl, err := game.LoadTable(worldFS(t, []synth.File{
		{Path: "data/data.bin", Data: synth.DataBinUnitsTable(units, humans)},
	}))
	if err != nil {
		t.Fatalf("LoadTable: %v", err)
	}

	// Three slots: the reserved entry 0 and the two written rows.
	if got := tbl.Units.Len(); got != 3 {
		t.Errorf("Units holds %d entries, want 3", got)
	}
	if got := tbl.Units.EntryName(1); got != "one" {
		t.Errorf("the first written units entry is %q, want %q", got, "one")
	}
	if got := tbl.Humans.Len(); got != 2 {
		t.Errorf("Humans holds %d entries, want 2", got)
	}

	// THE THREE ITEM COLLECTIONS ARRIVE TOO, and they are asserted PRESENT
	// rather than by their contents: what a name is worth is the definition
	// tier's question, and this one is only whether the table a world builder
	// receives can answer it at all. A table missing any of the three arms every
	// person on every map, silently and with every test still green, which is
	// exactly the failure this line exists to catch.
	if tbl.Shapes == nil || tbl.Materials == nil || tbl.Weapons == nil {
		t.Errorf("the table carries shapes=%v materials=%v weapons=%v; a person's equipment "+
			"cannot be resolved without all three", tbl.Shapes != nil, tbl.Materials != nil,
			tbl.Weapons != nil)
	}

	// THE SPELLS COLLECTION ARRIVES TOO, asserted present on the three item
	// collections' own reason above: a table missing it casts nothing for
	// anyone, silently and with every other test still green.
	if tbl.Spells == nil {
		t.Error("the table carries no Spells collection; a mission world's cast table cannot be built")
	}
}

func TestEveryWayTheTableCanFailIsAFailure(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files []synth.File
	}{
		{"the archive holds no table at all", []synth.File{
			{Path: "data/other.bin", Data: []byte("not the table")},
		}},
		{"the table is bytes the walk refuses", []synth.File{
			{Path: "data/data.bin", Data: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x01}},
		}},
		{"the table is empty", []synth.File{
			{Path: "data/data.bin", Data: nil},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tbl, err := game.LoadTable(worldFS(t, tc.files))
			if err == nil {
				t.Fatalf("LoadTable accepted it and returned %+v", tbl)
			}
			if tbl != nil {
				t.Errorf("LoadTable returned a non-nil table alongside an error")
			}
			if !strings.Contains(err.Error(), "data.bin") {
				t.Errorf("error %q does not name what failed to read", err)
			}
		})
	}
}
