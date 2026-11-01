package game

import (
	"bytes"
	"cmp"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func patrolRowFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	if err := f.App("patrol row").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	actors := []sim.Entity{
		{ID: 1, Owner: sim.SelfSlot, Group: 1, X: 10, Y: 10, HP: 100, MaxHP: 100, Reach: 1, DyingTime: 60},
		{ID: 2, Owner: 3, Group: 2, X: 40, Y: 40, HP: 20, MaxHP: 20, DyingTime: 60},
	}
	for i := range actors {
		actors[i].TokenSize, actors[i].TypeID, actors[i].Speed = 1, 1, 10
		actors[i].ScanRange = 5
		actors[i].HealthRegenPeriod, actors[i].ManaRegenPeriod, actors[i].Capacity = 1, 1, data.UnitCapacity()
	}
	w, err := sim.NewRelatedWorld(1, f.live.world.Bounds(), sim.ModeCanonical, sim.Terrain{}, actors, nil, sim.Relations{})
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = insertFixtureRuntimeIDs(w), w
	return f
}

func patrolRowOrder(w *sim.World, id sim.EntityID) (sim.SavedActorOrder, bool) {
	_, orders, _ := w.SavedGroups()
	for _, o := range orders {
		if o.Entity == id {
			return o, true
		}
	}
	return sim.SavedActorOrder{}, false
}

func patrolRowField(r sav.DocumentRecordData, name string) []byte {
	for _, x := range r.Raw {
		if x.Name == name {
			return x.Bytes
		}
	}
	return nil
}

// A live patroller whose action row holds no order keeps its patrol through
// LOAD: a row an engine wrote while the order was missing, or a row of a World
// without a Group registry, says nothing of the patrol's end. The next SAVE
// writes the ring, U158 and U54 the source holds, and the patroller walks as
// the source World does. A row that holds the order restores it as before.
func TestPatrolRowWithoutOrderKeepsThePatrolThroughLOADAndSAVE(t *testing.T) {
	f := patrolRowFront(t)
	w := f.live.world
	sim.Step(w, []sim.Command{sim.GroupPatrolTo(1, sim.CellPoint{X: 16, Y: 10}, 1)})
	for range 24 {
		sim.Step(w, nil)
	}
	held, ok := patrolRowOrder(w, 1)
	if e, _ := w.Entity(1); !ok || held.State != 10 || !held.Authored || len(held.Patrol) != 2 || !e.HasTarget {
		t.Fatalf("fixture: patrol order %+v held %v", held, ok)
	}
	raw, doc, a := saveCurrentEffect(t, f)
	source := castOrderRecord(t, doc, a, 1)
	if len(patrolRowField(source, "U158_90")) != 4 {
		t.Fatal("fixture: source SAV holds no two-cell ring")
	}
	native, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	for _, variant := range []string{"order row", "row without order", "World without a Group registry"} {
		t.Run(variant, func(t *testing.T) {
			input := raw
			if variant != "order row" {
				edited, err := sav.DecodeDocumentData(raw)
				if err != nil {
					t.Fatal(err)
				}
				rows, err := json.Marshal(a)
				if err != nil {
					t.Fatal(err)
				}
				var b currentActionData
				if err := json.Unmarshal(rows, &b); err != nil {
					t.Fatal(err)
				}
				for i := range b.Actions.Actors {
					if b.Actions.Actors[i].Entity == 1 || variant == "World without a Group registry" {
						b.Actions.Actors[i].Order = nil
					}
				}
				// The writer numbers the orders the World holds from zero.
				var stated []*uint32
				for i := range b.Actions.Actors {
					if o := b.Actions.Actors[i].Order; o != nil && o.Ordinal != nil {
						stated = append(stated, o.Ordinal)
					}
				}
				slices.SortFunc(stated, func(x, y *uint32) int { return cmp.Compare(*x, *y) })
				for i, o := range stated {
					*o = uint32(i)
				}
				if variant == "World without a Group registry" {
					if b.Policy == nil {
						t.Fatal("fixture: SAV holds no current policy")
					}
					b.Policy.GroupCarrier = false
				}
				leaf, err := json.Marshal(b)
				if err != nil {
					t.Fatal(err)
				}
				if err := sav.SetNativeActions(&edited.State, leaf); err != nil {
					t.Fatal(err)
				}
				if input, err = sav.EncodeDocumentData(edited); err != nil {
					t.Fatal(err)
				}
			}
			cold := openCurrentEffectSave(t, f, input)
			got, ok := patrolRowOrder(cold.live.world, 1)
			if !ok || got.State != held.State || !got.Authored || !reflect.DeepEqual(got.Patrol, held.Patrol) || got.Raw != held.Raw {
				t.Fatalf("LOAD order %+v held %v, want %+v", got, ok, held)
			}
			_, second, b := saveCurrentEffect(t, cold)
			again := castOrderRecord(t, second, b, 1)
			for _, name := range []string{"U158", "U158_90", "U54", "U50"} {
				if !bytes.Equal(patrolRowField(again, name), patrolRowField(source, name)) {
					t.Fatalf("SAVE %s = %x, source %x", name, patrolRowField(again, name), patrolRowField(source, name))
				}
			}
			var walked sim.World
			if err := walked.UnmarshalBinary(native); err != nil {
				t.Fatal(err)
			}
			start, _ := walked.Entity(1)
			cells := map[[2]int32]bool{}
			for tick := range 96 {
				sim.Step(&walked, nil)
				sim.Step(cold.live.world, nil)
				want, _ := walked.Entity(1)
				e, _ := cold.live.world.Entity(1)
				if e.X != want.X || e.Y != want.Y || e.HasTarget != want.HasTarget || e.TargetX != want.TargetX || e.TargetY != want.TargetY || e.PatrolLeg != want.PatrolLeg {
					t.Fatalf("tick %d: patroller at %d,%d to %d,%d leg %d, source World at %d,%d to %d,%d leg %d",
						tick, e.X, e.Y, e.TargetX, e.TargetY, e.PatrolLeg, want.X, want.Y, want.TargetX, want.TargetY, want.PatrolLeg)
				}
				cells[[2]int32{e.X, e.Y}] = true
			}
			if end, _ := walked.Entity(1); len(cells) < 3 || end.PatrolLeg == start.PatrolLeg && end.X == start.X {
				t.Fatalf("fixture: the source patroller did not walk its ring (%d cells)", len(cells))
			}
		})
	}
}
