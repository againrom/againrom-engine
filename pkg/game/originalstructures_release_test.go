package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The original produced three zero-HP Buildings. Independent pinned savfull
// archive events locate the bodies at 46677, 46756, 46835, each 77 bytes.
// Positive unequal and signed-negative variants change only literal health
// words in memory; they are NOT observations of original-produced damage.
func TestReleaseOriginalStructures1098HealthRuinAndNativeSave(t *testing.T) {
	for _, tc := range []struct {
		name    string
		hp, max uint16
	}{{"original ruins", 0, 2000}, {"synthetic wounded", 7, 31}, {"synthetic signed ruin", 0xffff, 73}} {
		t.Run(tc.name, func(t *testing.T) {
			f := releaseFront(t)
			_, payload := groundCorpusFile(t, "2026-08-15/game0017.sav", "eafce5d6575d54fdddc7a35f57531cd3df9317006c80f7c4085866c1b02b4fe0")
			source, err := sav.Open(payload)
			if err != nil {
				t.Fatal(err)
			}
			for i, off := range []int{46677, 46756, 46835} {
				if binary.LittleEndian.Uint32(source.Body[off+19:]) != uint32(7+i) ||
					binary.LittleEndian.Uint16(source.Body[off+60:]) != 0 ||
					binary.LittleEndian.Uint16(source.Body[off+62:]) != 2000 {
					t.Fatalf("independent original Building %d anchor changed", i)
				}
			}
			if tc.hp != 0 {
				binary.LittleEndian.PutUint16(source.Body[46677+60:], tc.hp)
				binary.LittleEndian.PutUint16(source.Body[46677+62:], tc.max)
				payload = source.Marshal()
			}
			fresh, err := StartMission(f.Archives.Containers, 40, f.Table, mapload.DifficultyNormal, nil)
			if err != nil {
				t.Fatal(err)
			}
			var ids []sim.StructureID
			for _, authored := range []uint16{7, 8, 9} {
				found := 0
				for i, object := range fresh.Map.Objects {
					if object.Field12 == authored {
						if object.Field0C != 0 {
							t.Fatalf("authored %d no longer starts as a ruin: placement word=%#04x", authored, object.Field0C)
						}
						ids = append(ids, sim.StructureID(i))
						found++
					}
				}
				if found != 1 {
					t.Fatalf("authored %d has %d placements", authored, found)
				}
			}
			for _, id := range ids {
				s := fresh.World.Structures()[id]
				if s.Field42 != 0 || s.MaxHealth != 2000 {
					t.Fatalf("fresh structure %d changed: %+v", id, s)
				}
			}
			originals := t.TempDir()
			if err := os.WriteFile(filepath.Join(originals, "game9999.sav"), payload, 0o600); err != nil {
				t.Fatal(err)
			}
			f.SetDeterministicFrames(true)
			app := f.App("1098 health/ruin")
			app.Layout(1024, 768)
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: originals}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, "game9999.sav")
			if err := app.HeadlessKey("0"); err != nil {
				t.Fatal(err)
			}
			if f.live.world.Tick() != rawSavedSubTick1112(t, payload) {
				t.Fatal("LOAD ticked before inspection")
			}
			hash := f.live.world.Hash()
			check := func() []byte {
				live := f.live
				// Native LOAD does not persist pause. Keep this inspection-only
				// witness stopped at any saved phase, not only the former zero.
				if !live.stopped {
					if err := app.HeadlessKey("0"); err != nil {
						t.Fatal(err)
					}
				}
				for i := range live.fog.visible {
					live.fog.visible[i], live.fog.explored[i] = 1, 1
				}
				live.push()
				for i, id := range ids {
					hp, max := uint16(0), uint16(2000)
					if i == 0 {
						hp, max = tc.hp, tc.max
					}
					s := live.world.Structures()[id]
					if s.Field42 != hp || s.MaxHealth != max {
						t.Fatalf("imported structure %d=%d/%d", id, s.Field42, s.MaxHealth)
					}
					inspectionCentre(live, int(s.Col), int(s.Row))
					releaseHoverInspection(t, app, live, ui.InspectionSubject{Kind: ui.InspectionStructure, ID: uint32(id)})
					panel, ok := live.view.InspectionPanel()
					if !ok || panel.HP != int(int16(hp)) || panel.MaxHP != int(max) {
						t.Fatalf("panel %+v", panel)
					}
					entries, ruins := live.view.StructureRuinFrames(uint32(id))
					wantRuins := 0
					if int16(hp) <= 0 {
						wantRuins = entries
					}
					if entries == 0 || ruins != wantRuins {
						t.Fatalf("structure %d hp%d ruin frames %d/%d", id, int16(hp), ruins, entries)
					}
				}
				// Render the changed first structure's actual stats card.
				s := live.world.Structures()[ids[0]]
				inspectionCentre(live, int(s.Col), int(s.Row))
				releaseHoverInspection(t, app, live, ui.InspectionSubject{Kind: ui.InspectionStructure, ID: uint32(ids[0])})
				card, err := app.HeadlessMissionCard()
				if err != nil || len(card.Pix) == 0 {
					t.Fatalf("card: %v", err)
				}
				return append([]byte(nil), card.Pix...)
			}
			pixels := check()
			if f.live.world.Hash() != hash {
				t.Fatal("inspection mutated imported world")
			}
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal(err)
			}
			entries, err := listAGS(store)
			if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".ags" {
				t.Fatalf("SAVE %+v %v", entries, err)
			}
			groundAppLoad(t, app, list, entries[0].Name)
			if f.live.world.Hash() != hash || !bytes.Equal(pixels, check()) {
				t.Fatal("native SAVE/LOAD changed world or rendered health card")
			}
			ms, report, err := loadOriginalMission(f, payload)
			if err != nil || report.Structures != (originalStructureCounts{Restored: 11}) {
				t.Fatalf("diagnostic %+v %v", report.Structures, err)
			}
			if s := ms.World.Structures()[ids[0]]; s.Field42 != tc.hp || s.MaxHealth != tc.max {
				t.Fatalf("diagnostic health %+v", s)
			}
			t.Logf("authored7/8/9 simIDs=%v fresh0/2000 -> %d/%d,0/2000,0/2000; saved tick hover+ruins+card stable across native SAVE/LOAD; hash=%016x", ids, int16(tc.hp), tc.max, hash)
		})
	}
}
