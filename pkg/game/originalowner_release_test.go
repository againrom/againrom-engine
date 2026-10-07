package game

import (
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Independently decoded natural Token owner keys point to Player slot1 for
// these temporary allies. The original ALM slots are5 (M20) and2 (M10).
// Both installs consume each same owner save; this is not a ROM1 runtime test.
func TestReleaseOriginalSavedAlliesKeepPlayerControl(t *testing.T) {
	for _, tc := range []struct {
		path, sha string
		mapIDs    []uint16
	}{
		{"2026-08-02/game0001.sav", "c5010cfbbe1f59018452813e7eda192dc5f3ce9834c9246bad23287e168899dd", []uint16{136, 137, 138, 139}},
		{"2026-08-02/game0002.sav", "b1ce079cc2b3f1bc101862c1dcf2afd8237421458b3e2474c4447efe5df2e761", []uint16{21}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			f := releaseFront(t)
			path, _ := groundCorpusFile(t, tc.path, tc.sha)
			f.SetDeterministicFrames(true)
			app := f.App("original-saved-owner")
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, filepath.Base(path))
			check := func(front *FrontEnd, a *ui.App) {
				t.Helper()
				if err := a.HeadlessKey("e"); err != nil {
					t.Fatal(err)
				}
				selected := a.HeadlessSelection()
				for _, mapID := range tc.mapIDs {
					found := 0
					for _, e := range front.live.world.Entities() {
						if e.MapUnitID != mapID {
							continue
						}
						found++
						if e.Owner != sim.SelfSlot {
							t.Fatalf("map%d owner%d, want1", mapID, e.Owner)
						}
						controlled := false
						for _, id := range selected {
							controlled = controlled || id == uint32(e.ID)
						}
						if !controlled {
							t.Fatalf("map%d entity%d not selected by E: %v", mapID, e.ID, selected)
						}
					}
					if found != 1 {
						t.Fatalf("map%d matches%d", mapID, found)
					}
				}
				// Selecting every local actor must not include an opponent.
				for _, id := range selected {
					for _, e := range front.live.world.Entities() {
						if uint32(e.ID) == id && e.Owner != sim.SelfSlot {
							t.Fatalf("E selected foreign entity%d", id)
						}
					}
				}
			}
			check(f, app)
			fresh, freshApp := holdingsNativeFresh(t, f, app, store, nil)
			check(fresh, freshApp)
			store2 := SaveStore{Dir: t.TempDir()}
			s2, l2, load2 := fresh.SaveSeams(store2, OriginalStore{}, nil)
			freshApp.SetSaveSeams(s2, l2, load2)
			again, againApp := holdingsNativeFresh(t, fresh, freshApp, store2, nil)
			check(again, againApp)
			// E must reach actual movement dispatch, not just a highlighted
			// foreign unit. Observe the production queue after a left tap.
			before := len(again.live.pending)
			x, y, err := againApp.HeadlessGroundPoint()
			if err != nil {
				t.Fatal(err)
			}
			if err := againApp.HeadlessPointer("press", x, y); err != nil {
				t.Fatal(err)
			}
			if err := againApp.HeadlessPointer("release", x, y); err != nil {
				t.Fatal(err)
			}
			for _, mapID := range tc.mapIDs {
				var id sim.EntityID
				for _, e := range again.live.world.Entities() {
					if e.MapUnitID == mapID {
						id = e.ID
					}
				}
				ordered := false
				for _, cmd := range again.live.pending[before:] {
					ordered = ordered || cmd.Entity == id && cmd.Kind == sim.KindGroupMoveTo
				}
				if !ordered {
					t.Fatalf("map%d entity%d received no movement command: %v", mapID, id, again.live.pending[before:])
				}
			}
			t.Logf("source=%s sha=%s allies=%v: Player1 ownership and E selection survive two ordinary SAVE/fresh LOAD cycles", tc.path, tc.sha, tc.mapIDs)
		})
	}
}
