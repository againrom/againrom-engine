package sim

import (
	"bytes"
	"testing"
)

func currentRetainedRuntimeWorld(t *testing.T, runtime uint32, bound, sourceBound bool) *World {
	t.Helper()
	w := deadWorld(t)
	d := deadInput(1, 2, -14)
	if !bound {
		d.ID, d.Source.MapUnitID = 4, 0
	}
	d.Source.State.RuntimeID = runtime
	if sourceBound {
		e := &w.entities[0]
		e.SourceBinding = SourceBinding{Class: 1, ArchiveIndex: d.Source.ArchiveIndex, Identity: d.Source.Identity, RuntimeID: runtime}
		e.ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1}}
	}
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{d}); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestCurrentRetainedRuntimeUsesOneAnchorForAllHolders(t *testing.T) {
	for _, mode := range []struct {
		name               string
		bound, sourceBound bool
	}{{"held", false, false}, {"bound absent source", true, false}, {"bound source", true, true}} {
		t.Run(mode.name, func(t *testing.T) {
			for _, edit := range []bool{false, true} {
				wire, native := uint32(71), uint32(0xabcdef77)
				ordinary, wanted := wire, native
				if edit {
					ordinary, wanted = 60123, 60123
				}
				got := currentRetainedRuntimeWorld(t, ordinary, mode.bound, mode.sourceBound)
				want := currentRetainedRuntimeWorld(t, wanted, mode.bound, mode.sourceBound)
				id := got.originalDead[0].ID
				values := make(map[EntityID]ActorValues)
				for _, e := range got.Entities() {
					values[e.ID] = e.Values()
				}
				v := values[id]
				v.RuntimeID = &ActorRuntimeCoordinate{Wire: wire, Value: native}
				values[id] = v
				if err := got.RestoreCurrentContinuation(nil, values, got.Actions(), nil); err != nil {
					t.Fatal("restore", edit, err)
				}
				if got.Hash() != want.Hash() {
					t.Fatalf("ordinary edit %t changed complete World: %x/%x", edit, got.Hash(), want.Hash())
				}
				for tick := 0; tick < 33; tick++ {
					Step(got, nil)
					Step(want, nil)
					if got.Hash() != want.Hash() {
						t.Fatal("next death tick changed", edit, tick)
					}
				}
			}
		})
	}
}

func TestCurrentRetainedRuntimeTransactionRejectsUnboundOrExtraOperands(t *testing.T) {
	for _, change := range []string{"unknown retained ID", "actor payload", "wide anchor", "redundant", "zero value", "later action failure", "missing live actor"} {
		t.Run(change, func(t *testing.T) {
			w := currentRetainedRuntimeWorld(t, 71, false, false)
			values := make(map[EntityID]ActorValues)
			for _, e := range w.Entities() {
				values[e.ID] = e.Values()
			}
			id := w.originalDead[0].ID
			v := ActorValues{RuntimeID: &ActorRuntimeCoordinate{Wire: 71, Value: 0xabcdef77}}
			actions := w.Actions()
			switch change {
			case "unknown retained ID":
				id = 999
			case "actor payload":
				v.AlwaysHits = true
			case "wide anchor":
				v.RuntimeID.Wire = 65536
			case "redundant":
				v.RuntimeID.Value = 71
			case "zero value":
				v.RuntimeID.Value = 0
			case "later action failure":
				actions.Actors = append(actions.Actors, actions.Actors[0])
			case "missing live actor":
				delete(values, 1)
			}
			values[id] = v
			before, _ := w.MarshalBinary()
			if err := w.RestoreCurrentContinuation(nil, values, actions, nil); err == nil {
				t.Fatal("invalid continuation accepted")
			}
			after, err := w.MarshalBinary()
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("failed continuation changed retained holder", err)
			}
		})
	}
}
