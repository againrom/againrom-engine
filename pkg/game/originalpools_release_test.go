package game

import (
	"encoding/binary"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

// One original observation, independently decoded by research/tools/savfull:
// Player slot 4 (Beasts), Unit index 82 at decoded body[25050,25671), map ID 39.
// Literal word offsets below are not obtained from the reader under test.
// The same source is imported through EN and RU, not claimed as two recordings.
func TestReleaseOriginalPools1094WoundedNonPartyAppLoadAndNativeRoundtrip(t *testing.T) {
	f := releaseFront(t)
	path, payload := groundCorpusFile(t, "2026-08-14/game0013.sav", "b211b9ad621a2cec38ff5a1d1f3f632542a58e1e377c8f562b25ba48ca2aa5ea")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []struct {
		off   int
		value uint16
	}{{25599, 4}, {25601, 10}, {25605, 0}, {25607, 0}, {25069, 39}} {
		if got := binary.LittleEndian.Uint16(source.Body[field.off:]); got != field.value {
			t.Fatalf("independent source word at %d = %d, want %d", field.off, got, field.value)
		}
	}
	if source.Body[25629] != 0 {
		t.Fatal("independent source death stage changed")
	}
	fresh, err := StartMission(f.Archives.Containers, 10, f.Table, mapload.DifficultyNormal, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertPools(t, poolEntity(t, fresh.World, 39), [4]int32{10, 10, 0, 0})
	f.SetDeterministicFrames(true)
	app := f.App("1094-original-nonparty-pools")
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, filepath.Base(path))
	w := f.live.world
	actor := poolEntity(t, w, 39)
	assertPools(t, actor, [4]int32{4, 10, 0, 0})
	if actor.Owner != 4 || w.Tick() != rawSavedSubTick1112(t, payload) || originalPartyCarriesMapUnit(f.liveParty, 39) {
		t.Fatalf("wrong owner/tick/party for imported entity: %+v tick=%d", actor, w.Tick())
	}
	// Diagnostic resume must make the same four assignments after rearming.
	ms, report, err := ResumeOriginalSave(f.Archives.Containers, payload, f.Table, f.Difficulty, nil, f.Bodies)
	if err != nil || report.PoolsRestored == 0 {
		t.Fatalf("diagnostic import failed: %v report=%+v", err, report)
	}
	assertPools(t, poolEntity(t, ms.World, 39), [4]int32{4, 10, 0, 0})
	hash := w.Hash()
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := listAGS(store)
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".ags" {
		t.Fatalf("native mission SAVE = %+v %v", entries, err)
	}
	groundAppLoad(t, app, list, entries[0].Name)
	assertPools(t, poolEntity(t, f.live.world, 39), [4]int32{4, 10, 0, 0})
	if f.live.world.Hash() != hash {
		t.Fatalf("native world hash %016x, want %016x", f.live.world.Hash(), hash)
	}
	t.Logf("source=%s owner=4 mapUnitID=39 fresh=10/10,0/0 imported=4/10,0/0 restored=%d excluded=%d unmatched=%d native=%s hash=%016x",
		path, report.PoolsRestored, report.PoolsExcluded, report.PoolsUnmatched, entries[0].Name, hash)
}
