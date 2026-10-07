package game

import (
	"bytes"
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

func TestNativeFog1115CheckpointBetweenSamples(t *testing.T) {
	f, app, _ := sackObjectsOpen1115(t, nil)
	visible := append([]byte(nil), f.live.fog.visible...)
	explored := append([]byte(nil), f.live.fog.explored...)
	f.live.tick()
	if !bytes.Equal(visible, f.live.fog.visible) || !bytes.Equal(explored, f.live.fog.explored) {
		t.Fatal("fixture crossed the scheduled fog refresh instead of its interior")
	}
	newSample := newFogPlane(f.live.fog.cols, f.live.fog.rows)
	newSample.refresh(f.live.world, sim.SelfSlot)
	if bytes.Equal(newSample.visible, visible) {
		t.Fatal("movement did not discriminate saved visibility from a fresh sample")
	}
	before, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 {
		t.Fatal("ordinary SAVE", entries, err)
	}
	cold := cellStateFront(t)
	coldApp := cold.App("native visibility checkpoint")
	_, coldList, coldLoad := cold.SaveSeams(store, OriginalStore{}, nil)
	coldApp.SetSaveSeams(nil, coldList, coldLoad)
	groundAppLoad(t, coldApp, coldList, entries[0].Name)
	after, _, err := cold.Snapshot(true)
	if err != nil {
		t.Fatal("current SAV checkpoint", err)
	}
	itemMutationSame1115(t, before, after)
	if !bytes.Equal(visible, cold.live.fog.visible) || !bytes.Equal(explored, cold.live.fog.explored) {
		t.Fatal("native LOAD revealed constructor-only terrain")
	}
	refreshes := 0
	for range 64 {
		f.live.tick()
		cold.live.tick()
		if f.live.world.Hash() != cold.live.world.Hash() || !bytes.Equal(f.live.fog.visible, cold.live.fog.visible) || !bytes.Equal(f.live.fog.explored, cold.live.fog.explored) || !bytes.Equal(f.live.fog.project(), cold.live.fog.project()) {
			t.Fatal("native fog or simulation continuation differs", f.live.world.Tick())
		}
		if f.live.world.Tick()%fogPeriod == 0 {
			refreshes++
		}
	}
	if refreshes != 2 {
		t.Fatal("continuation did not cover two scheduled samples", refreshes)
	}
}

func TestNativeFog1115LegacyExplorationDoesNotGrowAtOpen(t *testing.T) {
	mw := &mapWorld{fog: newFogPlane(4, 2)}
	for i := range mw.fog.visible {
		mw.fog.visible[i], mw.fog.explored[i] = 1, 1
	}
	r := SnapshotResidue{FogCols: 4, FogRows: 2, FogExplored: []byte{1, 0, 0, 1, 0, 0, 0, 0}}
	if !mw.restoreNativeFog(r) || !bytes.Equal(mw.fog.explored, r.FogExplored) || !bytes.Equal(mw.fog.visible, r.FogExplored) {
		t.Fatal("legacy LOAD kept constructor exploration or revealed an unseen cell")
	}
	r.FogExplored[0] = 0
	if mw.fog.explored[0] != 1 {
		t.Fatal("restored exploration aliases the input")
	}
	current := mw.residue()
	cold := &mapWorld{fog: newFogPlane(4, 2)}
	cold.applyResidue(current)
	if !bytes.Equal(cold.fog.visible, mw.fog.visible) || !bytes.Equal(cold.fog.explored, mw.fog.explored) {
		t.Fatal("presentation-only residue did not restore the captured fog")
	}
	current.FogVisible[0], current.FogExplored[0] = 0, 0
	if mw.fog.visible[0] != 1 || mw.fog.explored[0] != 1 {
		t.Fatal("snapshot fog aliases live planes")
	}
	// Original import remains the separate OR-only law, not native replacement.
	if !mw.applyExplored(4, 2, []byte{0, 1, 0, 0, 0, 0, 0, 0}) || mw.fog.explored[0] != 1 || mw.fog.explored[1] != 1 {
		t.Fatal("original Fog import stopped accumulating its own saved bit")
	}
}

func TestNativeFog1115RejectsMalformedSampleBeforeAdoption(t *testing.T) {
	f, app, _ := sackObjectsOpen1115(t, nil)
	before, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		edit func(*SnapshotResidue)
	}{
		{"zero width", func(r *SnapshotResidue) { r.FogCols = 0 }},
		{"negative height", func(r *SnapshotResidue) { r.FogRows = -1 }},
		{"dimension overflow", func(r *SnapshotResidue) { r.FogCols, r.FogRows = int(^uint(0)>>1), 2 }},
		{"short visible", func(r *SnapshotResidue) { r.FogVisible = r.FogVisible[1:] }},
		{"long visible", func(r *SnapshotResidue) { r.FogVisible = append(r.FogVisible, 0) }},
		{"short explored", func(r *SnapshotResidue) { r.FogExplored = r.FogExplored[1:] }},
		{"nonbinary visible", func(r *SnapshotResidue) { r.FogVisible[0] = 2 }},
		{"nonbinary explored", func(r *SnapshotResidue) { r.FogExplored[0] = 2 }},
		{"visible unexplored", func(r *SnapshotResidue) { r.FogVisible[0], r.FogExplored[0] = 1, 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := before
			bad.Residue.FogExplored = bytes.Clone(before.Residue.FogExplored)
			bad.Residue.FogVisible = bytes.Clone(before.Residue.FogVisible)
			tc.edit(&bad.Residue)
			if _, err := EncodeSave(bad, "invalid visibility"); err == nil {
				t.Fatal("encoder accepted malformed visibility")
			}
			if _, _, err := DecodeSave(uncheckedDocumentEnvelope1115(t, bad)); err == nil {
				t.Fatal("checksum-valid hostile visibility envelope was accepted")
			}
			if _, _, err := f.Restore(bad); err == nil {
				t.Fatal("direct Restore accepted malformed visibility")
			}
			after, _, err := f.Snapshot(true)
			if err != nil || !reflect.DeepEqual(before, after) || app == nil {
				t.Fatal("refused visibility changed the active session", err)
			}
		})
	}
	// Same area and sound planes, but the saved extent names a different map.
	bad := before
	bad.Residue.FogCols /= 2
	bad.Residue.FogRows *= 2
	if err := validateNativeFogResidue(bad.Residue); err != nil {
		t.Fatal("extent control must pass shape validation", err)
	}
	if _, _, err := f.Restore(bad); err == nil {
		t.Fatal("LOAD accepted a complete sample for a different map extent")
	}
	after, _, err := f.Snapshot(true)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("late extent refusal changed the active session", err)
	}
}
