package game

import (
	"os"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseVirtualDeadSourceHPStaysDistinctFromCurrentSave(t *testing.T) {
	for _, tc := range []struct {
		name, hash string
		hp         int16
	}{
		{"game0032.sav", "7375f0c08c8361b2fa32d20564802acba688d5e5a445ab5ca635c609632499ed", -58},
		{"game0033.sav", "b6ffd904f44717c76e22f12d8801a21b3dfca3dd8563b7e02aaecbf93a4c740a", -86},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, original := groundCorpusFile(t, "2027-09-07/"+tc.name, tc.hash)
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
				if err := front.App("virtual dead source health").OpenMission(mission); err != nil {
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
			if body := bodyFromWorld(first); body.Source.State.HP != tc.hp || body.Current.HP != tc.hp {
				t.Fatalf("original LOAD source/current HP = %d/%d, want %d/%d", body.Source.State.HP, body.Current.HP, tc.hp, tc.hp)
			}
			var before sim.OriginalDeadRecord
			for tick := 0; tick < 128; tick++ {
				first.LiveAdvance(1)
				before = bodyFromWorld(first)
				if before.Current.HP < tc.hp {
					break
				}
			}
			if before.Source.State.HP != tc.hp || before.Current.HP >= tc.hp {
				t.Fatalf("played source/current HP = %d/%d; expected distinct source %d", before.Source.State.HP, before.Current.HP, tc.hp)
			}
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
			if got := deadFromSAV(written, source.Identity); got.HP != before.Current.HP {
				t.Fatalf("written ordinary Health=%d, want current HP %d", got.HP, before.Current.HP)
			}
			cold := open(written)
			loaded := bodyFromWorld(cold)
			if loaded.Source.State.HP != before.Source.State.HP || loaded.Current.HP != before.Current.HP || cold.live.world.Hash() != beforeHash {
				t.Fatalf("cold source/current HP %d/%d hash %016x; want %d/%d hash %016x", loaded.Source.State.HP, loaded.Current.HP, cold.live.world.Hash(), before.Source.State.HP, before.Current.HP, beforeHash)
			}
			first.LiveAdvance(1)
			cold.LiveAdvance(1)
			if first.live.world.Hash() != cold.live.world.Hash() {
				t.Fatalf("next tick World %016x != cold %016x", first.live.world.Hash(), cold.live.world.Hash())
			}
		})
	}
}
