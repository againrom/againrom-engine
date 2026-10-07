package game

import (
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The explored map coming back out of an ORIGINAL save.
//
// NOTHING HERE READS A GAME INSTALL (golden rule 2). The store is built from
// the documented framing, and what it is checked against is the plane a map
// screen would draw from. Evidence against the preserved saves is in
// verification.md and comes from cmd/savtool and cmd/savecheck.

func TestFogForMapSizesTheRecordAgainstTheMap(t *testing.T) {
	// 4x2 = 8 cells, explored at indices 3 and 4.
	g := &sav.Fog{Cells: []byte{0, 0, 0, 1, 1, 0, 0, 0}, Set: 2}
	var r OriginalSaveResume
	got := fogForMap(&alm.Map{Width: 4, Height: 2}, g, &r)
	if got == nil {
		t.Fatal("a record whose cell count is the map's extent was refused")
	}
	if got.cols != 4 || got.rows != 2 {
		t.Fatalf("sized %dx%d, want 4x2", got.cols, got.rows)
	}
	if r.FogCells != 8 || r.FogSet != 2 {
		t.Fatalf("reported %d of %d, want 2 of 8", r.FogSet, r.FogCells)
	}
	mw := &mapWorld{fog: newFogPlane(4, 2)}
	if !mw.applyExplored(got.cols, got.rows, got.cells) {
		t.Fatal("the sized plane was refused by the map screen")
	}
	if n := countState(mw.fog.explored, 1); n != 2 {
		t.Fatalf("%d cells explored, want 2", n)
	}

	// A record for another map is refused, and the counts are still reported
	// so the divergence is visible rather than silent.
	var r2 OriginalSaveResume
	if fogForMap(&alm.Map{Width: 8, Height: 8}, g, &r2) != nil {
		t.Fatal("an 8-cell record was carried for a 64-cell map")
	}
	if r2.FogCells != 8 {
		t.Fatalf("a refused record reported %d cells, want it still counted", r2.FogCells)
	}
}

// TestApplyExploredOnlyEverORs is TERR-FOG-145's own rule, and AC-7: the
// original's load arm ORs the bit into each tile word and clears nothing, so a
// cell this build has already lit must survive a record that does not carry it.
func TestApplyExploredOnlyEverORs(t *testing.T) {
	mw := &mapWorld{fog: newFogPlane(4, 2)}
	mw.fog.explored[0] = 1 // already seen before the restore
	if !mw.applyExplored(4, 2, []byte{0, 0, 0, 1, 1, 0, 0, 0}) {
		t.Fatal("the plane was refused")
	}
	if mw.fog.explored[0] != 1 {
		t.Error("a cell the plane already held was cleared by a record that does not carry it")
	}
	if mw.fog.explored[3] != 1 || mw.fog.explored[4] != 1 {
		t.Error("the record's own cells did not go in")
	}
}

func TestApplyExploredRefusesAnotherMapExtent(t *testing.T) {
	mw := &mapWorld{fog: newFogPlane(4, 2)}
	if mw.applyExplored(8, 8, make([]byte, 64)) {
		t.Fatal("a 64-cell plane was applied to an 8-cell map")
	}
	for i, b := range mw.fog.explored {
		if b != 0 {
			t.Fatalf("cell %d was painted from a plane of another size", i)
		}
	}
}

func TestARestoredExploredPlaneSurvivesOurOwnSave(t *testing.T) {
	from := &mapWorld{fog: newFogPlane(4, 2)}
	if !from.applyExplored(4, 2, []byte{0, 0, 0, 1, 1, 0, 0, 0}) {
		t.Fatal("the plane was refused")
	}
	s := Snapshot{Mission: 10, World: []byte{1}, Residue: from.residue()}
	b, err := EncodeSave(s, "explored")
	if err != nil {
		t.Fatalf("EncodeSave: %v", err)
	}
	back, _, err := DecodeSave(b)
	if err != nil {
		t.Fatalf("DecodeSave: %v", err)
	}
	if back.Residue.FogCols != 4 || back.Residue.FogRows != 2 {
		t.Fatalf("dimensions came back %dx%d", back.Residue.FogCols, back.Residue.FogRows)
	}
	to := &mapWorld{
		commanded: map[sim.EntityID]bool{},
		swing:     map[sim.EntityID]int{},
		phase:     map[sim.EntityID]sim.AttackPhase{},
		fog:       newFogPlane(4, 2),
	}
	to.applyResidue(back.Residue)
	if to.fog.explored[3] != 1 || to.fog.explored[4] != 1 {
		t.Fatalf("the explored plane did not survive our own save: %v", to.fog.explored)
	}
	if n := countState(to.fog.explored, 1); n != 2 {
		t.Fatalf("%d cells came back, want 2", n)
	}
}

func TestTheFogLineNamesWhichOfTheThreeCasesHolds(t *testing.T) {
	for _, c := range []struct {
		name string
		r    OriginalSaveResume
		want string
	}{
		{"no record", OriginalSaveResume{}, "NO RECORD"},
		{"not applied", OriginalSaveResume{FogCells: 100, FogSet: 20}, "NOT APPLIED"},
		{"restored", OriginalSaveResume{FogCells: 100, FogSet: 20, FogApplied: true}, "RESTORED"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := c.r.fogLine()
			if !strings.Contains(got, c.want) {
				t.Fatalf("fogLine() = %q, want it to name %q", got, c.want)
			}
		})
	}
	// The percentage is of the RECORDED grid, stated so the number in the
	// report can be checked against savtool's own.
	if got := (OriginalSaveResume{FogCells: 20736, FogSet: 3475, FogApplied: true}).fogLine(); !strings.Contains(got, "16.8%") {
		t.Fatalf("fogLine() = %q, want it to carry 16.8%%", got)
	}
}
