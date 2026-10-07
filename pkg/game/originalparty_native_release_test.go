package game

import (
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

// The literals were measured before native SAVE on two independent natural
// mission40 inputs. A World hash alone missed this defect: the live party
// driver pointed at the next actor, then at an absent actor, after native LOAD.
func TestReleaseOriginalPartyNativeLoadKeepsActorBindings(t *testing.T) {
	for _, tc := range []struct {
		path, sha string
		party     [3]string
		mapIDs    [3]uint16
		hp        [3]int32
	}{
		{"2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b",
			[3]string{"hero", "npc:25", "npc:22"}, [3]uint16{0, 6, 0}, [3]int32{131, 146, 22}},
		{"2026-08-15/game0019.sav", "7f4d7b736c3011df1c8e41b6b16ccc087f9ced07309f6fc9f503be4ca72529b8",
			[3]string{"hero", "npc:22", "npc:25"}, [3]uint16{0, 0, 6}, [3]int32{133, 23, 157}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			f := releaseFront(t)
			path, _ := groundCorpusFile(t, tc.path, tc.sha)
			f.SetDeterministicFrames(true)
			app := f.App("original-party-native-bindings")
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, filepath.Base(path))
			check := func(front *FrontEnd) {
				t.Helper()
				mn := front.live.mission
				if len(mn.party) != 3 || !reflect.DeepEqual(mn.ids, []sim.EntityID{46, 47, 48}) {
					t.Fatalf("party=%d entity IDs=%v, want three members bound to46,47,48", len(mn.party), mn.ids)
				}
				for i, p := range mn.party {
					if p.ID != tc.party[i] || p.StartingHero != (i == 0) {
						t.Fatalf("member%d: %s primary=%t", i, p.ID, p.StartingHero)
					}
					found := false
					for _, e := range front.live.world.Entities() {
						if e.ID != mn.ids[i] {
							continue
						}
						found = true
						if e.MapUnitID != tc.mapIDs[i] || e.HP != tc.hp[i] || e.Owner != sim.SelfSlot {
							t.Fatalf("member%s -> entity%d map%d HP%d owner%d", p.ID, e.ID, e.MapUnitID, e.HP, e.Owner)
						}
					}
					if !found {
						t.Fatalf("member%s names absent entity%d", p.ID, mn.ids[i])
					}
				}
			}
			check(f)
			fresh, freshApp := holdingsNativeFresh(t, f, app, store, nil)
			check(fresh)
			for _, id := range fresh.live.mission.ids {
				if err := freshApp.HeadlessSelectEntity(uint32(id)); err != nil {
					t.Fatalf("select saved party%d: %v", id, err)
				}
			}
			// Use a separate store so the second ordinary SAVE is independently
			// listed. This proves the repair is stable, not a one-load offset.
			store2 := SaveStore{Dir: t.TempDir()}
			save2, list2, load2 := fresh.SaveSeams(store2, OriginalStore{}, nil)
			freshApp.SetSaveSeams(save2, list2, load2)
			again, _ := holdingsNativeFresh(t, fresh, freshApp, store2, nil)
			check(again)
			t.Logf("source=%s sha=%s: ordinary App SAVE/fresh LOAD twice retains all three literal party bindings", tc.path, tc.sha)
		})
	}
}
