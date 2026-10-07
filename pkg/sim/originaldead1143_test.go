package sim

import (
	"bytes"
	"reflect"
	"testing"
)

func TestOriginalDead1143TwoUnboundActorsKeepStateAndLeaveGameplayAlone(t *testing.T) {
	w, control := deadWorld(t), deadWorld(t)
	batch := []OriginalDeadActor{deadInput(10, 2, -14), deadInput(11, 4, -58)}
	for i := range batch {
		batch[i].Source.MapUnitID = 0
	}
	if err := w.ImportOriginalDeadActors(batch); err != nil {
		t.Fatal(err)
	}
	want := []OriginalDeadRecord{{OriginalDeadActor: batch[0], Current: batch[0].Source.State},
		{OriginalDeadActor: batch[1], Current: batch[1].Source.State}}
	if !reflect.DeepEqual(w.OriginalDeadActors(), want) {
		t.Fatal("unbound records were merged, rebound or normalized")
	}
	if next, ok := w.NextEntityID(); !ok || next != 12 {
		t.Fatalf("unbound identities can be reused: next=%d available=%t", next, ok)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(form, back.encode()) {
		t.Fatal("native LOAD changed unbound state")
	}
	for tick := range 64 {
		var commands []Command
		if tick == 0 {
			// Both retained records name this otherwise empty destination.
			commands = []Command{{Kind: KindMoveTo, Entity: 2, X: 4, Y: 3}}
		}
		Step(w, commands)
		Step(&back, commands)
		Step(control, commands)
		if w.Hash() != back.Hash() {
			t.Fatalf("unbound cold continuation differs at tick %d", tick)
		}
		for i, record := range w.OriginalDeadActors() {
			if record.Source != batch[i].Source {
				t.Fatal("dead-list continuation rewrote source provenance")
			}
		}
		if !reflect.DeepEqual(w.Entities(), control.Entities()) || !reflect.DeepEqual(w.Stock(), control.Stock()) ||
			w.rng.state != control.rng.state || w.Purse(1) != control.Purse(1) || len(w.sacks) != 0 {
			t.Fatalf("unbound records changed live population, movement or rewards at tick %d", tick)
		}
	}
	for i, record := range w.OriginalDeadActors() {
		if record.Current.HP != batch[i].Source.State.HP-2 || record.Current.Stage != batch[i].Source.State.Stage {
			t.Fatalf("64 legacy ticks must age each corpse twice: %+v", record)
		}
	}
	if e := w.entities[1]; e.X != 4 || e.Y != 3 {
		t.Fatal("retained unbound records obstructed the destination")
	}
}

func TestOriginalDead1143StageTwoKeepsItsSourceThroughNativeContinuation(t *testing.T) {
	w := deadWorld(t)
	d := deadInput(1, 2, -14)
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{d}); err != nil {
		t.Fatal(err)
	}
	if e := w.entities[0]; e.HP != -14 || e.Decay != 2 || e.Dwell != 0 || e.OrdinaryTargetable() {
		t.Fatalf("stage two was resurrected or normalized: %+v", e)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	for tick := range 64 {
		Step(w, nil)
		Step(&back, nil)
		if w.Hash() != back.Hash() {
			t.Fatalf("stage-two cold continuation differs at tick %d", tick)
		}
	}
	r := w.OriginalDeadActors()[0]
	if r.Source.State.HP != d.Source.State.HP-2 || r.Source.State.Stage != d.Source.State.Stage || r.Current.HP >= -14 || r.Current.Stage < 2 ||
		w.Purse(1) != 12345 || w.rng.state != 33 || len(w.sacks) != 0 {
		t.Fatalf("source health mirror or native decay/reward policy changed: %+v", r)
	}
}

func TestOriginalDead1143UnboundIdentityErrorsRejectTheWholeBatch(t *testing.T) {
	batch := []OriginalDeadActor{deadInput(10, 2, -14), deadInput(11, 4, -58)}
	for i := range batch {
		batch[i].Source.MapUnitID = 0
	}
	for name, mutate := range map[string]func(*OriginalDeadActor){
		"live entity":       func(d *OriginalDeadActor) { d.ID = 2 },
		"same entity ID":    func(d *OriginalDeadActor) { d.ID = 10 },
		"same source key":   func(d *OriginalDeadActor) { d.Source.Identity = batch[0].Source.Identity },
		"same archive slot": func(d *OriginalDeadActor) { d.Source.ArchiveIndex = batch[0].Source.ArchiveIndex },
		"missing runtime":   func(d *OriginalDeadActor) { d.Source.State.RuntimeID = 0 },
		"early stage":       func(d *OriginalDeadActor) { d.Source.State.Stage = 1 },
	} {
		t.Run(name, func(t *testing.T) {
			w := deadWorld(t)
			before := w.encode()
			bad := append([]OriginalDeadActor(nil), batch...)
			mutate(&bad[1])
			if err := w.ImportOriginalDeadActors(bad); err == nil {
				t.Fatal("accepted ambiguous or invalid unbound record")
			}
			if !bytes.Equal(before, w.encode()) {
				t.Fatal("failed batch imported its first record")
			}
		})
	}
}
