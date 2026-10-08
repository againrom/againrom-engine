//go:build sessioncorpusaudit

package game

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

func TestMilestone2NativeScalarMutation(t *testing.T) {
	matched := false
	milestone2Corpus(t, func(t *testing.T, input milestone2File, f *FrontEnd) {
		if input.rel != "2026-10-07/saveorcsdontgo.sav" {
			return
		}
		matched = true
		open, town, err := f.RestoreOriginal(input.raw)
		if err == nil && !town {
			err = f.App("native scalar mutation").OpenMission(open)
		}
		if err != nil || town {
			t.Fatal("corpus LOAD", town, err)
		}
		var rows []sim.NativeActorBasisRecord
		actions := f.live.world.Actions()
		for _, e := range f.live.world.Entities() {
			if e.ActorLoad.Source.Class != 0 {
				continue
			}
			b := e.NativeBasis
			if !b.ScalarsPresent || b.ScalarKnown != 1<<sim.ScalarCount-1 || b.AttackKnown != 0xffffff || b.DefenceKnown != 0x3fffff || b.BlockKnown != 0x3ff {
				t.Fatal("corpus native carrier incomplete", e.ID, b)
			}
			for _, slot := range []int{sim.ScalarT18, sim.ScalarU8E, sim.ScalarUA0, sim.ScalarU130, sim.ScalarU138} {
				b.Scalars[slot] ^= 1
			}
			if e.ActionClock.Known {
				for i := range actions.Actors {
					if actions.Actors[i].Entity == e.ID {
						actions.Actors[i].ActionClock.End ^= 1
						b.Scalars[sim.ScalarU138] = actions.Actors[i].ActionClock.End
					}
				}
			}
			b.Attack[22] ^= 1
			b.Defence[4] ^= 1
			b.Block[4] ^= 1
			rows = append(rows, sim.NativeActorBasisRecord{ID: e.ID, Basis: b})
		}
		if len(rows) != 35 {
			t.Fatalf("corpus native mutation population %d, want 35", len(rows))
		}
		if err := f.live.world.RestoreActions(actions, nil); err != nil {
			t.Fatal(err)
		}
		if err := f.live.world.RestoreNativeActorBases(rows); err != nil {
			t.Fatal(err)
		}
		written, _, _ := saveCurrentEffect(t, f)
		open, town, err = f.RestoreOriginal(written)
		if err == nil && !town {
			err = f.App("native scalar mutation cold LOAD").OpenMission(open)
		}
		if err != nil || town {
			t.Fatal("changed corpus cold LOAD", town, err)
		}
		for _, row := range rows {
			e, ok := f.live.world.Entity(row.ID)
			if !ok {
				t.Fatal("changed native actor disappeared", row.ID)
			}
			for _, slot := range []int{sim.ScalarT18, sim.ScalarU8E, sim.ScalarUA0, sim.ScalarU130, sim.ScalarU138} {
				if e.NativeBasis.Scalars[slot] != row.Basis.Scalars[slot] {
					t.Fatalf("actor %d field %s lost current mutation: got %#x want %#x", row.ID, nativeScalarFields[slot], e.NativeBasis.Scalars[slot], row.Basis.Scalars[slot])
				}
			}
			if e.NativeBasis.Attack[22] != row.Basis.Attack[22] || e.NativeBasis.Defence[4] != row.Basis.Defence[4] || e.NativeBasis.Block[4] != row.Basis.Block[4] {
				t.Fatal("changed corpus raw combat/tail lost", row.ID)
			}
		}
		for tick := range 3 {
			f.live.tick()
			before := map[sim.EntityID]sim.NativeActorBasis{}
			for _, e := range f.live.world.Entities() {
				if e.NativeBasis.ScalarsPresent {
					before[e.ID] = sim.NativeBasisNow(e)
				}
			}
			written, _, _ := saveCurrentEffect(t, f)
			open, town, err = f.RestoreOriginal(written)
			if err == nil && !town {
				err = f.App("native scalar next-tick cold LOAD").OpenMission(open)
			}
			if err != nil || town {
				t.Fatal("next-tick corpus cold LOAD", town, err)
			}
			for id, want := range before {
				e, ok := f.live.world.Entity(id)
				got := sim.NativeBasisNow(e)
				if !ok {
					t.Errorf("tick %d actor %d disappeared through SAVE/LOAD", tick, id)
					continue
				}
				if got != want {
					var fields []string
					for slot := range got.Scalars {
						if got.Scalars[slot] != want.Scalars[slot] {
							fields = append(fields, fmt.Sprintf("%s=%#x want %#x", nativeScalarFields[slot], got.Scalars[slot], want.Scalars[slot]))
						}
					}
					for n := range got.Block {
						if got.Block[n] != want.Block[n] {
							fields = append(fields, fmt.Sprintf("Block12[%d]=%d want %d", n+2, got.Block[n], want.Block[n]))
						}
					}
					if got.Base != want.Base || got.Modifier != want.Modifier || got.Attack != want.Attack || got.Defence != want.Defence || got.Body != want.Body {
						fields = append(fields, "combat bytes differ")
					}
					if len(fields) == 0 {
						fields = append(fields, "availability differs")
					}
					t.Errorf("tick %d actor %d current carrier changed through SAVE/LOAD: %s", tick, id, strings.Join(fields, "; "))
				}
			}
		}
		t.Logf("%d native actors: LOAD, current scalar/combat/tail mutation, SAVE, cold LOAD passed", len(rows))
	})
	if !matched {
		t.Fatal("native scalar regression input missing from external corpus")
	}
}
