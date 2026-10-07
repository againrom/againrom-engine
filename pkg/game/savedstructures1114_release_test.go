package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The three first Building offsets are independent published savfull anchors
// already guarded by release1098. Only synthetic in-memory mutations are made:
// move authored7, make authored8 source-only, and give each positive HP. No
// owner save or install is written. Installed art and the App LOAD/SAVE path run.
func TestReleaseSavedStructures1114MovedSourceOnlyAndNoGhost(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	_, payload := groundCorpusFile(t, "2026-08-15/game0017.sav", "eafce5d6575d54fdddc7a35f57531cd3df9317006c80f7c4085866c1b02b4fe0")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	for i, off := range []int{46677, 46756} {
		if binary.LittleEndian.Uint32(source.Body[off+19:]) != uint32(7+i) {
			t.Fatal("independent Building anchor changed")
		}
		binary.LittleEndian.PutUint16(source.Body[off+60:], uint16(71+i))
	}
	x, y := source.Body[46677]+1, source.Body[46677+1]+1
	source.Body[46677], source.Body[46677+1] = x, y
	binary.LittleEndian.PutUint32(source.Body[46756+19:], 0)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game9999.sav"), source.Marshal(), 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	app := f.App("1114 restored Building roster")
	app.Layout(1024, 768)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game9999.sav")
	check := func() {
		live := f.live
		if !live.stopped {
			if err := app.HeadlessKey("0"); err != nil {
				t.Fatal(err)
			}
		}
		for i := range live.fog.visible {
			live.fog.visible[i], live.fog.explored[i] = 1, 1
		}
		live.push()
		meta, _, present := live.world.SavedStructures()
		if !present || len(meta) != 11 || len(live.world.Structures()) != 11 {
			t.Fatalf("saved roster %+v", meta)
		}
		var oldID uint32
		foundOld := false
		for i, o := range live.mission.state.Map.Objects {
			if o.Field12 == 8 {
				oldID = uint32(i)
				foundOld = true
			}
		}
		if !foundOld {
			t.Fatal("missing original authored8 placement")
		}
		seen := 0
		for i, s := range meta {
			st := live.world.Structures()[i]
			if uint32(st.ID) == oldID {
				t.Fatal("ALM authored8 ghost resurrected")
			}
			if s.AuthoredID != 7 && s.AuthoredID != 0 {
				continue
			}
			seen++
			if s.AuthoredID == 7 && (st.Col != int32(x) || st.Row != int32(y) || st.Field42 != 71) {
				t.Fatal("moved saved geometry", st)
			}
			if s.AuthoredID == 0 && (s.HasAuthored || st.Field42 != 72) {
				t.Fatal("source-only binding", s, st)
			}
			inspectionCentre(live, int(st.Col), int(st.Row))
			releaseHoverInspection(t, app, live, ui.InspectionSubject{Kind: ui.InspectionStructure, ID: uint32(st.ID)})
			panel, ok := live.view.InspectionPanel()
			if !ok || panel.HP != int(st.Field42) {
				t.Fatal("saved structure panel", panel)
			}
			if entries, ruins := live.view.StructureRuinFrames(uint32(st.ID)); entries == 0 || ruins != 0 {
				t.Fatal("saved installed art absent or wrong ruin", entries, ruins)
			}
			if card, err := app.HeadlessMissionCard(); err != nil || len(card.Pix) == 0 {
				t.Fatal("saved installed card", err)
			}
		}
		if seen != 2 {
			t.Fatalf("moved/source-only witnesses%d", seen)
		}
	}
	check()
	canonicalSources := func(rows []sim.SavedStructure, doc *sav.DocumentData) []sim.SavedStructure {
		keys := map[uint32]uint32{0: 0, doc.World.TerrainIdentity: uint32(len(doc.Objects) + 1)}
		for i, object := range doc.Objects {
			for _, value := range object.Values {
				if value.Value != 0 && (value.Name == "Identity" || value.Name == "This") {
					keys[value.Value] = uint32(i + 1)
				}
			}
		}
		canonical := func(key uint32) uint32 {
			value, ok := keys[key]
			if !ok {
				t.Fatalf("unbound saved structure key %#x", key)
			}
			return value
		}
		for i := range rows {
			rows[i].SourceKey = canonical(rows[i].SourceKey)
			rows[i].Reference = canonical(rows[i].Reference)
			binary.LittleEndian.PutUint32(rows[i].Position[8:], canonical(binary.LittleEndian.Uint32(rows[i].Position[8:])))
		}
		return rows
	}
	structures := f.live.world.Structures()
	metadata, cells, present := f.live.world.SavedStructures()
	metadata = canonicalSources(metadata, f.live.mission.state.savedDocument.Document)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 {
		t.Fatalf("ordinary SAVE %v %v", entries, err)
	}
	if !IsOriginal(entries[0].Name) {
		t.Fatalf("ordinary SAVE did not write SAV: %s", entries[0].Name)
	}
	groundAppLoad(t, app, list, localOriginalSaveToken(entries[0].Name))
	check()
	gotSource, gotCells, gotPresent := f.live.world.SavedStructures()
	gotSource = canonicalSources(gotSource, f.live.mission.state.savedDocument.Document)
	if !reflect.DeepEqual(f.live.world.Structures(), structures) || !reflect.DeepEqual(gotSource, metadata) || !reflect.DeepEqual(gotCells, cells) || gotPresent != present {
		t.Fatal("SAVE/LOAD changed saved structures, source roster or cell links")
	}
	t.Logf("installed first-frame moved authored7 at%d,%d, source-only former8, no ALM8 ghost; art/hover/card and SAVE/LOAD PASS", x, y)
}
