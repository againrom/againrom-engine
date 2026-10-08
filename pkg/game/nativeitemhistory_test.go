package game

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestNativeItemHistoryRearmOrdinarySaveAndByteEdits(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	defs := eqDefsTable(t)
	f.Table.Shapes, f.Table.Materials = defs.Shapes, defs.Materials
	f.Table.Weapons, f.Table.Shields, f.Table.Armors = defs.Weapons, defs.Shields, defs.Armors
	item := mapload.SourceConstructedItem(sim.PlainItem(eqShieldCode), f.Table)
	item.Effects = []sim.ItemEffect{{Kind: 12, Operand: 3}, {Kind: 44, Operand: 0x0907}, {Kind: 8, Operand: 100}}
	party := f.NextParty()
	if len(party) == 0 || item.SourceEquipment.Class != sim.SourceShield {
		t.Fatal("fixture lacks a fresh party and typed Shield")
	}
	party[0].Carry = nil
	party[0].CarriedItems = []sim.ItemInstance{item}
	if err := f.App("native item history").OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	if len(f.live.mission.state.Start.IDs) == 0 {
		t.Fatal("fixture has no native party subject")
	}
	id := f.live.mission.state.Start.IDs[0]
	e, ok := f.live.world.Entity(id)
	if !ok || !e.Humanoid || e.ActorLoad.Source.Class != 0 || !f.live.invSubjectSet || sim.EntityID(f.live.invSubject.ID) != id {
		t.Fatal("fixture lacks actual native party inventory subject")
	}
	var modifier [64]byte
	binary.LittleEndian.PutUint16(modifier[18:], 0x1234)
	binary.LittleEndian.PutUint16(modifier[10:], 33)
	modifier[40] = 0x99
	if err := f.live.world.RestoreNativeActorBases([]sim.NativeActorBasisRecord{{ID: id, Basis: e.NativeBasis.WithModifier(modifier)}}); err != nil {
		t.Fatal(err)
	}
	sim.Step(f.live.world, []sim.Command{sim.Equip(id, 0, 2)})
	f.live.rearm()
	e, _ = f.live.world.Entity(id)
	basis := e.NativeBasis
	if binary.LittleEndian.Uint16(basis.Modifier[18:]) != 0x1237 || basis.Modifier[37] != 7 || basis.Modifier[38] != 9 || basis.Modifier[39] != 1 || basis.Modifier[40] != 0x99 ||
		binary.LittleEndian.Uint16(basis.Modifier[10:]) != uint16(e.HealthRegeneration) {
		t.Fatal("actual equipment/rearm lost local raw history or doubled regeneration", basis, e.HealthRegeneration)
	}
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	var captured sim.World
	if err := captured.UnmarshalBinary(snapshot.World); err != nil {
		t.Fatal(err)
	}
	snapshot.SavedDocument, err = f.materializeCurrentWorld(snapshot, &captured)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range snapshot.SavedDocument.Actors {
		if row.EntityID == id {
			block, err := savedActorRaw(&snapshot.SavedDocument.Document.Objects[row.ObjectIndex-1], "UD4", 64)
			if err != nil {
				t.Fatal(err)
			}
			for i := range block {
				block[i] = 0xcc
			}
		}
	}
	raw, err := f.ExportCurrentSave(snapshot, "stale item history Document")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	record := castOrderRecord(t, doc, actions, id)
	block, err := savedActorRaw(&record, "UD4", 64)
	if err != nil || !bytes.Equal(block, basis.Modifier[:]) {
		t.Fatal("known current bytes did not replace stale ordinary UD4", block, err)
	}
	cold := openCurrentEffectSave(t, f, raw)
	got, ok := cold.live.world.Entity(id)
	if !ok || got.ActorLoad.Source.Class != 0 || got.NativeBasis != basis {
		t.Fatal("cold ordinary LOAD lost current native item history", got.NativeBasis)
	}
	for _, row := range actions.Bindings {
		if row.ID == id && !row.Structure && !row.Missing {
			changed, _ := savedActorRaw(&doc.Objects[row.Object-1], "UD4", 64)
			changed[39] = 5
		}
	}
	edited, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	changed := openCurrentEffectSave(t, f, edited)
	got, _ = changed.live.world.Entity(id)
	if got.NativeBasis.Modifier[39] != 5 || !got.NativeBasis.ModifierByteKnown(39) {
		t.Fatal("ordinary selector edit lost to unchanged supplemental history")
	}
	changed.live.tick()
	_, next, nextActions := saveCurrentEffect(t, changed)
	record = castOrderRecord(t, next, nextActions, id)
	block, _ = savedActorRaw(&record, "UD4", 64)
	if block[39] != 5 {
		t.Fatal("next SAVE masked the ordinary selector edit")
	}
	sim.Step(cold.live.world, []sim.Command{sim.Unequip(id, 2)})
	cold.live.rearm()
	got, _ = cold.live.world.Entity(id)
	if binary.LittleEndian.Uint16(got.NativeBasis.Modifier[18:]) != 0x1234 || [3]byte{got.NativeBasis.Modifier[37], got.NativeBasis.Modifier[38], got.NativeBasis.Modifier[39]} != [3]byte{7, 9, 1} {
		t.Fatal("actual next removal lost effect subtraction/assignment order", got.NativeBasis)
	}
	for range 3 {
		cold.live.tick()
	}
	nextRaw, next, nextActions := saveCurrentEffect(t, cold)
	record = castOrderRecord(t, next, nextActions, id)
	block, _ = savedActorRaw(&record, "UD4", 64)
	if !bytes.Equal(block, got.NativeBasis.Modifier[:]) {
		t.Fatal("next SAVE lost post-removal current raw bytes")
	}
	last := openCurrentEffectSave(t, cold, nextRaw)
	lastEntity, _ := last.live.world.Entity(id)
	if lastEntity.NativeBasis != got.NativeBasis {
		t.Fatal("second cold LOAD lost post-removal current history")
	}
}
