package game

import (
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
)

func TestReleaseMission111DormantEnemiesWakeAfterOriginalAndNativeLoad(t *testing.T) {
	path, _ := groundCorpusFile(t, "2026-09-09/game0076.sav", "ca6f2980986859fb19c7b602a00b92b0e3ae95b1d00adc6c757152d596ed556c")
	for _, tc := range []struct {
		name string
		id   sim.EntityID
		x, y int32
	}{{"hill orc", 77, 82, 32}, {"goblin camp", 13, 79, 46}} {
		t.Run(tc.name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			a := f.App("original enemy activation")
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
			a.SetSaveSeams(save, list, load)
			groundAppLoad(t, a, list, filepath.Base(path))
			w := f.live.world
			enemy := releaseEntityByID(t, w, tc.id)
			if !enemy.Alive() {
				t.Fatal("regression requires a living, dormant original enemy")
			}
			groupID := enemy.SourceBinding.GroupIndex
			groups, _, _ := w.SavedGroups()
			found := false
			for _, g := range groups {
				if g.ID == groupID {
					found = true
					if g.AI[0x45] != 0 {
						t.Fatal("original dormant byte was replaced during LOAD")
					}
				}
			}
			if !found {
				t.Fatal("missing original Group")
			}
			// These are the owner's previously reached attack positions, not
			// arbitrary hill cells that a normal movement command could reject.
			worldPlaceAndWalk(t, w, 85, tc.x, tc.y)
			fresh, _ := holdingsNativeFresh(t, f, a, store, nil)
			// 140 ticks: the hill orc's first blow needs 125 (its approach ends in
			// a crossing, and the charge waits for arrival, HERO-CROSSHOLD-146) and
			// the goblin camp's 37, so 15 ticks of margin over the longer.
			for tick := 0; tick < 140; tick++ {
				sim.Step(w, nil)
				sim.Step(fresh.live.world, nil)
				if w.Hash() != fresh.live.world.Hash() {
					t.Fatalf("enemy continuation diverged after native LOAD at tick%d", tick)
				}
				if tick == 7 {
					e := releaseEntityByID(t, w, tc.id)
					if !e.HasAttackTarget || e.AttackTarget != 85 {
						t.Fatalf("original enemy%d stayed dormant: target%v/%d", tc.id, e.HasAttackTarget, e.AttackTarget)
					}
				}
			}
			if hero := releaseEntityByID(t, w, 85); hero.HP >= 81 {
				t.Fatal("awakened enemies never delivered damage")
			}
		})
	}
}
