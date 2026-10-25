package game

import (
	"encoding/binary"
	"fmt"
	"os"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// orderResidueWords reads, by actor identity, the completion byte U136 and
// countdown byte U6C of every living actor in sub-phase 0 that holds either
// nonzero, through the raw-offset oracle of unit1156Read.
func orderResidueWords(t *testing.T, raw []byte) map[uint32][2]uint32 {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	set, err := unitScalarExpected(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	out := map[uint32][2]uint32{}
	for _, r := range set.records {
		v := r.values
		if v["Stage"] == 0 && binary.LittleEndian.Uint32(r.raw["U58"]) == 0 && (v["U136"] != 0 || v["U6C"] != 0) {
			out[v["Identity"]] = [2]uint32{v["U136"], v["U6C"]}
		}
	}
	return out
}

func orderResidueLosses(want map[uint32][2]uint32, got func(uint32) ([2]uint32, bool)) []string {
	var lost []string
	for identity, words := range want {
		if written, ok := got(identity); !ok || written != words {
			lost = append(lost, fmt.Sprintf("%#x: U136/U6C %v written %v", identity, words, written))
		}
	}
	return lost
}

// The first SAVE after an original LOAD keeps an idle actor's completion and
// countdown bytes, and a cold LOAD of that SAVE restores the same World.
func TestReleaseFirstSaveKeepsIdleActorOrderResidue(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	_, original := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	want := orderResidueWords(t, original)
	complete, countdown := 0, 0
	for _, words := range want {
		if words[0] != 0 {
			complete++
		}
		if words[1] != 0 {
			countdown++
		}
	}
	if complete == 0 || countdown == 0 {
		t.Fatalf("game0021 holds %d completion and %d countdown residues; the witness needs both", complete, countdown)
	}
	open := func(raw []byte) *FrontEnd {
		t.Helper()
		front := releaseFront(t)
		front.SetDeterministicFrames(true)
		mission, town, err := front.RestoreOriginal(raw)
		if err != nil || town {
			t.Fatalf("mission LOAD: town=%t error=%v", town, err)
		}
		if err := front.App("order residue").OpenMission(mission); err != nil {
			t.Fatal(err)
		}
		return front
	}
	first := open(original)
	hash := first.live.world.Hash()
	dir := t.TempDir()
	prepared, err := first.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{}).Prepare(ui.SaveRequest{
		OnMap: true, Directory: dir, Name: "order residue", Format: ui.SaveSAV,
	})
	if err != nil {
		t.Fatal(err)
	}
	paths, err := prepared.Commit(true)
	if err != nil || len(paths) != 1 {
		t.Fatalf("SAVE: paths=%v error=%v", paths, err)
	}
	written, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(written)
	if err != nil {
		t.Fatal(err)
	}
	set, err := unitScalarExpected(file, written)
	if err != nil {
		t.Fatal(err)
	}
	got := map[uint32][2]uint32{}
	for _, r := range set.records {
		got[r.values["Identity"]] = [2]uint32{r.values["U136"], r.values["U6C"]}
	}
	read := func(identity uint32) ([2]uint32, bool) { words, ok := got[identity]; return words, ok }
	if lost := orderResidueLosses(want, read); len(lost) != 0 {
		t.Fatalf("first SAVE lost %d of %d idle residues: %v", len(lost), len(want), lost)
	}
	// Control: the comparison reports a residue written as the World's idle zero.
	zeroed := func(identity uint32) ([2]uint32, bool) { return [2]uint32{}, true }
	if lost := orderResidueLosses(want, zeroed); len(lost) != len(want) {
		t.Fatal("the residue comparison misses a zero write")
	}
	cold := open(written)
	if cold.live.world.Hash() != hash {
		t.Fatalf("cold LOAD World %016x differs from the saving World %016x", cold.live.world.Hash(), hash)
	}
	if !sameResave(t, first, cold) {
		t.Fatal("the cold LOAD's next SAVE writes another document")
	}
	t.Logf("order residues: %d actors, %d completion bytes and %d countdown bytes kept through SAVE and cold LOAD", len(want), complete, countdown)
}
