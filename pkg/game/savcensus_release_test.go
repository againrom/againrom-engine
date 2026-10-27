package game

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// censusMissionFront loads a mission SAV the way a cold LOAD does and opens its
// mission, so the front end holds the loaded Document. A name "native:N" is
// the engine's own SAVE of a freshly started mission N, loaded back; any other
// name is a corpus file.
func censusMissionFront(t *testing.T, relative string) *FrontEnd {
	t.Helper()
	var raw []byte
	if n, ok := strings.CutPrefix(relative, "native:"); ok {
		mission, err := strconv.Atoi(n)
		if err != nil {
			t.Fatal(err)
		}
		raw = censusNativeSave(t, mission)
	} else {
		raw = censusCorpusFile(t, relative)
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	open, _, err := f.RestoreOriginal(raw)
	if err != nil || open == nil {
		t.Skipf("%s does not load as a mission here: open=%v err=%v", relative, open != nil, err)
	}
	app := f.App("census")
	app.Layout(1024, 768)
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	// Play on, so the live World no longer equals the loaded Document.
	if n, _ := strconv.Atoi(os.Getenv("AGAINROM_CENSUS_TICKS")); n > 0 {
		for i := 0; i < n; i++ {
			f.live.tick()
		}
	}
	return f
}

func censusCorpusFile(t *testing.T, relative string) []byte {
	t.Helper()
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Skip("no AGAINROM_SAVE_CORPUS: the writer census needs owner saves")
	}
	raw, err := os.ReadFile(filepath.Join(corpus, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// censusNativeSave starts mission n with the starting mage, plays a little and
// writes the engine's own mission SAVE.
func censusNativeSave(t *testing.T, mission int) []byte {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	app := f.App("census-native")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(mission, party)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 120; i++ {
		f.live.tick()
	}
	raw, err := censusMissionSave(t, f)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func censusMissionSave(t *testing.T, f *FrontEnd) ([]byte, error) {
	snap, label, err := f.Snapshot(true)
	if err != nil {
		return nil, err
	}
	return f.ExportCurrentSave(snap, label)
}
