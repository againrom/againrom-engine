package game

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// savFogCells decodes the explored bit-15 plane of one SAV document.
func savFogCells(t *testing.T, raw []byte) []byte {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	fog, ok, err := file.Fog()
	if err != nil || !ok {
		t.Fatalf("Fog record: present=%t %v", ok, err)
	}
	return fog.Cells
}

// fogDelta compares a resave's explored plane with the loaded input and with
// the original's resave of the same input: kept input cells, gained cells the
// original also gained, gained cells the original lacks, and cells the original
// gained that the resave does not hold.
type fogDelta struct {
	inputSet, kept, wantGained, gained int
	missing, extra                     []string
}

func compareFogResave(input, got, want []byte, cols int) fogDelta {
	var d fogDelta
	for i := range input {
		cell := fmt.Sprintf("(%d,%d)", i%cols, i/cols)
		if input[i] != 0 {
			d.inputSet++
			if got[i] != 0 {
				d.kept++
			}
			continue
		}
		if want[i] != 0 {
			d.wantGained++
		}
		switch {
		case want[i] != 0 && got[i] != 0:
			d.gained++
		case want[i] != 0:
			d.missing = append(d.missing, cell)
		case got[i] != 0:
			d.extra = append(d.extra, cell)
		}
	}
	return d
}

// TestReleaseFogStampAfterColdLoad repeats the owner's mission-150 run
// (TERR-FOG-265, SAV-1308): a cold LOAD of an engine-written document, a SAVE
// before any world tick and a SAVE after 96 world ticks, each compared with the
// original's at-once resave of the same input. The owner files are read from
// AGAINROM_FOG_STAMP_DIR/<en|ru>/ and never committed; a language with no
// files there is skipped.
func TestReleaseFogStampAfterColdLoad(t *testing.T) {
	root, dir := os.Getenv("AGAINROM_ASSETS"), os.Getenv("AGAINROM_FOG_STAMP_DIR")
	if root == "" || dir == "" {
		t.Skip("AGAINROM_ASSETS and AGAINROM_FOG_STAMP_DIR are required")
	}
	lang := map[string]string{"english": "en", "russian": "ru"}[InspectInstall(root).Language]
	inputPath := filepath.Join(dir, lang, "input-9704-engine-written.sav")
	atOncePath := filepath.Join(dir, lang, "original-cold-load-at-once.sav")
	if _, err := os.Stat(inputPath); lang == "" || err != nil {
		t.Skip("no owner-run document for this install's language")
	}
	raw, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	atOnce, err := os.ReadFile(atOncePath)
	if err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("cold LOAD: town=%t %v", town, err)
	}
	if err := f.App("fog stamp after LOAD").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	for _, e := range mw.world.Entities() {
		if e.Owner == sim.SelfSlot && e.HP > 0 {
			t.Logf("party actor %d at (%d,%d) sight word %d", e.ID, e.X, e.Y, e.SightWord(e.Humanoid))
		}
	}
	save := func(label string) []byte {
		t.Helper()
		snapshot, name, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(label, err)
		}
		out, err := f.ExportCurrentWorldSave(snapshot, name)
		if err != nil {
			t.Fatal(label, err)
		}
		return savFogCells(t, out)
	}
	input, want := savFogCells(t, raw), savFogCells(t, atOnce)
	cols := mw.fog.cols
	first := save("at once")
	hash := mw.world.Hash()
	for n := 0; n < 96; n++ {
		mw.tick()
	}
	later := save("after 96 ticks")
	for _, run := range []struct {
		label string
		cells []byte
	}{{"at once", first}, {"after 96 ticks", later}} {
		d := compareFogResave(input, run.cells, want, cols)
		t.Logf("%s: kept %d of %d, gained %d of %d, missing %v, extra %v",
			run.label, d.kept, d.inputSet, d.gained, d.wantGained, d.missing, d.extra)
		if d.kept != d.inputSet {
			t.Errorf("%s: LOAD lost %d explored cells", run.label, d.inputSet-d.kept)
		}
		// The original's resave holds exactly the input plus 34 rim cells; the
		// engine's stamp at the loaded cells and sight words reproduces it.
		if d.gained != d.wantGained || len(d.extra) != 0 {
			t.Errorf("%s: gained %d of %d, missing %v, extra %v", run.label, d.gained, d.wantGained, d.missing, d.extra)
		}
	}
	for i := range first {
		if first[i] != later[i] {
			t.Fatalf("cell (%d,%d) differs between the at-once SAVE and the SAVE 96 ticks later (world hash %x)", i%cols, i/cols, hash)
		}
	}
}
