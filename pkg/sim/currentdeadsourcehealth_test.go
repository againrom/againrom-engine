package sim

import "testing"

func TestCurrentVirtualDeadSourceHealthUsesOrdinaryAnchor(t *testing.T) {
	for _, edit := range []bool{false, true} {
		ordinary, wantedSource := int16(-15), int16(-14)
		if edit {
			ordinary, wantedSource = -16, -16
		}
		cold := currentRetainedRuntimeWorld(t, 71, false, false)
		cold.originalDead[0].Source.State.HP = ordinary
		cold.originalDead[0].terminal.HP = ordinary
		want := currentRetainedRuntimeWorld(t, 71, false, false)
		want.originalDead[0].Source.State.HP = wantedSource
		want.originalDead[0].terminal.HP = ordinary
		values := make(map[EntityID]ActorValues)
		for _, e := range cold.Entities() {
			values[e.ID] = e.Values()
		}
		values[cold.originalDead[0].ID] = ActorValues{DeadSourceHealth: &ActorDeadSourceHealth{Wire: -15, Value: -14}}
		if err := cold.RestoreCurrentContinuation(nil, values, cold.Actions(), nil); err != nil {
			t.Fatal("restore", edit, err)
		}
		if cold.Hash() != want.Hash() {
			t.Fatalf("ordinary edit %t changed Source/Current HP: %x/%x", edit, cold.Hash(), want.Hash())
		}
		for tick := 0; tick < 33; tick++ {
			Step(cold, nil)
			Step(want, nil)
			if cold.Hash() != want.Hash() {
				t.Fatalf("ordinary edit %t changed next death tick %d", edit, tick)
			}
		}
	}
}

func TestCurrentVirtualDeadSourceHealthRejectsInvalidBinding(t *testing.T) {
	for _, change := range []string{"unknown body", "live actor", "extra operand", "reversed health", "zero health"} {
		t.Run(change, func(t *testing.T) {
			w := currentRetainedRuntimeWorld(t, 71, false, false)
			w.originalDead[0].Source.State.HP = -15
			w.originalDead[0].terminal.HP = -15
			values := make(map[EntityID]ActorValues)
			for _, e := range w.Entities() {
				values[e.ID] = e.Values()
			}
			id := w.originalDead[0].ID
			v := ActorValues{DeadSourceHealth: &ActorDeadSourceHealth{Wire: -15, Value: -14}}
			switch change {
			case "unknown body":
				id = 999
			case "live actor":
				id = 1
			case "extra operand":
				v.AlwaysHits = true
			case "reversed health":
				v.DeadSourceHealth.Value = -16
			case "zero health":
				v.DeadSourceHealth.Value = 0
			}
			values[id] = v
			before := w.Hash()
			if err := w.RestoreCurrentContinuation(nil, values, w.Actions(), nil); err == nil || w.Hash() != before {
				t.Fatalf("invalid continuation changed World: %v", err)
			}
		})
	}
}
