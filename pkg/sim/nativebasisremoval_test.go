package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func TestNativeBasisSurvivesActualTerminalCompaction(t *testing.T) {
	for _, id := range []EntityID{0, 3} {
		corpse := spEnt(id, 10, 11)
		corpse.HP, corpse.Decay = decayGoneHP-1, decayLast
		corpse.NativeBasis = NativeActorBasis{BasePresent: true, BaseKnown: 1 << 22, ModifierPresent: true, ModifierKnown: 1 << 40, BodyPresent: true, BodyKnown: true}
		corpse.NativeBasis.Base[22], corpse.NativeBasis.Modifier[40] = 96, 100
		w := spWorld(t, 9, nil, corpse, spEnt(8, 1, 1))
		Step(w, nil)
		if _, exists := w.Entity(id); exists || len(w.CurrentTerminalActors()) != 1 {
			t.Fatal("fixture did not actually remove its native corpse")
		}
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(form[len(form)-4:], []byte("NAB1")) {
			t.Fatal("actual removal lost known native basis bytes and masks")
		}
		span := int(binary.LittleEndian.Uint32(form[len(form)-9:]))
		at := len(form) - 9 - span + 4
		if binary.LittleEndian.Uint32(form[at:]) != uint32(id) || binary.LittleEndian.Uint32(form[at+5:]) != corpse.NativeBasis.BaseKnown || form[at+31] != 96 || binary.LittleEndian.Uint64(form[at+33:]) != corpse.NativeBasis.ModifierKnown || form[at+81] != 100 {
			t.Fatal("terminal basis identity, current bytes or knowledge changed")
		}
		var cold World
		if err := cold.UnmarshalBinary(form); err != nil || cold.Hash() != w.Hash() {
			t.Fatal("cold LOAD lost removed native basis", err)
		}
		Step(&cold, nil)
		if _, err := cold.MarshalBinary(); err != nil {
			t.Fatal("next tick cannot save removed native basis", err)
		}
	}
}

func TestRemovedNativeBasisRetainedDeadAndConstructionBoundaries(t *testing.T) {
	basis := (NativeActorBasis{}).WithBody(0).WithBase([24]byte{3}).WithModifier([64]byte{7})
	w := deadWorld(t)
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{deadInput(1, 3, -40)}); err != nil {
		t.Fatal(err)
	}
	if err := w.RestoreNativeActorBases([]NativeActorBasisRecord{{ID: 1, Basis: basis}}); err != nil {
		t.Fatal(err)
	}
	w.entities[0].HP, w.entities[0].Decay = decayGoneHP-1, decayLast
	Step(w, nil)
	want := []NativeActorBasisRecord{{ID: 1, Basis: basis}}
	if _, exists := w.Entity(1); exists || !reflect.DeepEqual(w.RemovedNativeActorBases(), want) || len(w.CurrentTerminalActors()) != 0 {
		t.Fatal("retained OriginalDead early return lost native history")
	}
	before := w.Hash()
	rows := w.RemovedNativeActorBases()
	rows[0].Basis.Body = 77
	if w.Hash() != before {
		t.Fatal("removed basis getter aliases current history")
	}
	if err := w.RestoreNativeActorBases([]NativeActorBasisRecord{{ID: 1, Basis: basis.WithBody(42)}, {ID: 99, Basis: basis}}); err == nil || w.Hash() != before {
		t.Fatal("dangling removed owner changed a partial batch")
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil || cold.Hash() != before {
		t.Fatal("cold retained-dead basis", err)
	}
	span := int(binary.LittleEndian.Uint32(form[len(form)-9:]))
	wrong := bytes.Clone(form)
	binary.LittleEndian.PutUint32(wrong[len(form)-9-span+4:], 99)
	if err := cold.UnmarshalBinary(wrong); err == nil || cold.Hash() != before {
		t.Fatal("dangling NAB1 owner accepted or changed state")
	}
	ids := map[EntityID]EntityID{}
	for _, id := range cold.ActorIdentityReferences() {
		ids[id] = id + 100
	}
	if err := cold.RestoreActorIdentities(ids, nil); err != nil {
		t.Fatal(err)
	}
	if got := cold.RemovedNativeActorBases(); len(got) != 1 || got[0].ID != 101 || got[0].Basis != basis {
		t.Fatal("removed basis identity did not follow its owner", got)
	}
	legacy := deadWorld(t)
	old, err := legacy.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if err := cold.UnmarshalBinary(old); err != nil || len(cold.RemovedNativeActorBases()) != 0 {
		t.Fatal("old form retained removed history", err)
	}
	for _, restore := range []bool{false, true} {
		constructor := spEnt(3, 10, 11)
		constructor.NativeBasis = basis
		if restore {
			constructor.HP, constructor.Decay = decayGoneHP-1, decayLast
		}
		fresh := spWorld(t, 9, nil, constructor)
		if restore {
			err = fresh.RestoreCurrentTerminalActors([]CurrentTerminalActor{{ID: 3, Cell: 0x0b0a, HP: decayGoneHP - 1, Stage: uint8(decayLast)}})
		} else {
			err = fresh.RetireUnboundConstructors([]EntityID{3})
		}
		if err != nil || len(fresh.RemovedNativeActorBases()) != 0 {
			t.Fatal("LOAD-discarded constructor created current history", restore, err)
		}
	}
}
