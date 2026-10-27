package game

import (
	"os"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A virtual body's record holds one state copy through LOAD, play, SAVE and
// cold LOAD: the dead-list clock publishes each decay to the source state.
// game0024's body is loaded at stage 4 and played to its terminal stage.
func TestReleaseVirtualDeadSourceStateFollowsCurrentSave(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	for _, tc := range []struct {
		name, hash string
		hp         int16
		terminal   bool
	}{
		{"2027-09-07/game0032.sav", "7375f0c08c8361b2fa32d20564802acba688d5e5a445ab5ca635c609632499ed", -58, false},
		{"2027-09-07/game0033.sav", "b6ffd904f44717c76e22f12d8801a21b3dfca3dd8563b7e02aaecbf93a4c740a", -86, false},
		{"2026-10-02/projectiles-original-en/game0024.sav", "0de5da52950f124ca2d8c492be3a3dc82e30c603e1e55ed79b6ddddac09cb934", -140, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, original := groundCorpusFile(t, tc.name, tc.hash)
			deadFromSAV := func(raw []byte, key uint32) sav.DeadActor {
				t.Helper()
				file, err := sav.Open(raw)
				if err != nil {
					t.Fatal(err)
				}
				dead, err := file.DeadActors()
				if err != nil {
					t.Fatal(err)
				}
				for _, body := range dead {
					if key != 0 && body.Identity == key || key == 0 && body.MapUnitID == 0 && body.HP == tc.hp {
						return body
					}
				}
				t.Fatalf("dead SAV has no virtual body with identity %#x and HP %d", key, tc.hp)
				return sav.DeadActor{}
			}
			source := deadFromSAV(original, 0)
			if source.MapUnitID != 0 || source.Stage < 2 || source.Stage >= 5 || source.HP != tc.hp {
				t.Fatalf("source dead actor tuple: %+v", source)
			}
			open := func(raw []byte) *FrontEnd {
				t.Helper()
				front := releaseFront(t)
				front.SetDeterministicFrames(true)
				mission, town, err := front.RestoreOriginal(raw)
				if err != nil || town {
					t.Fatalf("mission LOAD: town=%t error=%v", town, err)
				}
				if err := front.App("virtual dead source state").OpenMission(mission); err != nil {
					t.Fatal(err)
				}
				return front
			}
			bodyFromWorld := func(front *FrontEnd) sim.OriginalDeadRecord {
				t.Helper()
				for _, body := range front.live.world.OriginalDeadActors() {
					if body.Source.Identity == source.Identity {
						return body
					}
				}
				t.Fatalf("World has no source dead identity %#x", source.Identity)
				return sim.OriginalDeadRecord{}
			}
			first := open(original)
			loadedBody := bodyFromWorld(first)
			if loadedBody.Source.State.HP != tc.hp || loadedBody.Current.HP != tc.hp {
				t.Fatalf("original LOAD source/current HP = %d/%d, want %d/%d", loadedBody.Source.State.HP, loadedBody.Current.HP, tc.hp, tc.hp)
			}
			var before sim.OriginalDeadRecord
			for tick := 0; tick < 40000; tick++ {
				first.LiveAdvance(1)
				before = bodyFromWorld(first)
				if before.Source.State != before.Current {
					t.Fatalf("tick %d source state %+v differs from current %+v", tick, before.Source.State, before.Current)
				}
				if !tc.terminal && before.Current.HP < tc.hp || tc.terminal && before.Current.Stage == 5 {
					break
				}
			}
			if tc.terminal && (before.Current.Stage != 5 || before.Current.HP != -10001 || before.Current.RuntimeID != 0) ||
				!tc.terminal && before.Current.HP >= tc.hp {
				t.Fatalf("played body %+v did not reach the wanted state", before.Current)
			}
			// Control: the played record is not the loaded one, so an equal cold
			// record below is not a comparison of two unplayed copies.
			if before == loadedBody {
				t.Fatal("play did not change the body record")
			}
			first.live.world.PruneGroupsForSave() // SAVE prunes first; the rest of it changes nothing
			beforeHash := first.live.world.Hash()
			dir := t.TempDir()
			prepared, err := first.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{}).Prepare(ui.SaveRequest{
				OnMap: true, Directory: dir, Name: "virtual dead", Format: ui.SaveSAV,
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
			if first.live.world.Hash() != beforeHash {
				t.Fatal("SAVE changed the live World")
			}
			if got := deadFromSAV(written, source.Identity); got.HP != before.Current.HP || got.Stage != before.Current.Stage {
				t.Fatalf("written ordinary Health/Stage=%d/%d, want current %d/%d", got.HP, got.Stage, before.Current.HP, before.Current.Stage)
			}
			cold := open(written)
			beforeHash = adoptLoadedAllocator(t, first.live.world, cold.live.world).Hash()
			if loaded := bodyFromWorld(cold); loaded != before || cold.live.world.Hash() != beforeHash {
				t.Fatalf("cold record %+v hash %016x; want %+v hash %016x", loaded, cold.live.world.Hash(), before, beforeHash)
			}
			first.LiveAdvance(1)
			cold.LiveAdvance(1)
			if first.live.world.Hash() != cold.live.world.Hash() {
				t.Fatalf("next tick World %016x != cold %016x", first.live.world.Hash(), cold.live.world.Hash())
			}
		})
	}
}
