package sim

import (
	"reflect"
	"testing"
)

const (
	loadField     = 19
	capacityField = 20
)

func wideLoadValues(w *World, id EntityID, widths ...ActorNumericResidue) map[EntityID]ActorValues {
	values := make(map[EntityID]ActorValues)
	for _, e := range w.Entities() {
		v := e.Values()
		v.Widths = nil
		values[e.ID] = v
	}
	v := values[id]
	v.Widths = widths
	values[id] = v
	return values
}

// A wide Load or Capacity without an actor-load record round-trips.
func TestCurrentWideLoadAndCapacityRoundTripWithoutActorLoad(t *testing.T) {
	widths := []ActorNumericResidue{
		{Field: loadField, Wire: 7, Lift: 65536},
		{Field: capacityField, Wire: 9, Lift: -131072},
	}
	w := deadWorld(t)
	w.entities[1].Load, w.entities[1].Capacity = 7, 9
	if err := w.RestoreCurrentContinuation(nil, wideLoadValues(w, 2, widths...), w.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	e, _ := w.Entity(2)
	if e.Load != 65543 || e.Capacity != -131063 {
		t.Fatalf("wide load/capacity = %d/%d", e.Load, e.Capacity)
	}
	if got := e.Values().Widths; !reflect.DeepEqual(got, widths) {
		t.Fatalf("writer operands = %v, want %v", got, widths)
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	cold := deadWorld(t)
	cold.entities[1].Load, cold.entities[1].Capacity = 7, 9
	if err := cold.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.Entities(), cold.Entities()) || w.Hash() != cold.Hash() {
		t.Fatal("binary round trip changed a wide load or capacity")
	}

	// Loss control: without the operands the values read back narrow.
	narrow := deadWorld(t)
	narrow.entities[1].Load, narrow.entities[1].Capacity = 7, 9
	if err := narrow.RestoreCurrentContinuation(nil, wideLoadValues(narrow, 2), narrow.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	ne, _ := narrow.Entity(2)
	if ne.Load == e.Load || ne.Capacity == e.Capacity || reflect.DeepEqual(narrow.Entities(), w.Entities()) {
		t.Fatal("loss control did not distinguish a truncated load or capacity")
	}
}

// A wide Load or Capacity on an actor-load record is refused atomically.
func TestCurrentWideLoadWithActorLoadIsRefusedAtomically(t *testing.T) {
	for _, field := range []uint8{loadField, capacityField} {
		w := deadWorld(t)
		w.entities[1].Load, w.entities[1].Capacity = 7, 7
		w.entities[1].ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1}}
		before := w.Hash()
		values := wideLoadValues(w, 2, ActorNumericResidue{Field: field, Wire: 7, Lift: 65536})
		err := w.RestoreCurrentContinuation(nil, values, w.Actions(), nil)
		const want = "sim: restored current actions: sim: original actor load/capacity exceeds signed word"
		if err == nil || err.Error() != want || w.Hash() != before {
			t.Fatalf("field %d: wide operand on an actor-load record: err=%v, World changed=%t", field, err, w.Hash() != before)
		}
		// Control: without the operand the actor is admitted.
		if err := w.RestoreCurrentContinuation(nil, wideLoadValues(w, 2), w.Actions(), nil); err != nil {
			t.Fatalf("field %d: control without operand refused: %v", field, err)
		}
	}
}
