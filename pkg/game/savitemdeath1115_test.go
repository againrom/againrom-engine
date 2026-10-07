package game

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// This is a composed native persistence witness, not original corpse-root
// constructor authority. The source Weapon/Effect/Spell archive is synthetic;
// an actual damage command and fall dwell reach the ordinary loot producer.
func TestItemDeath1115CurrentOwnersSurviveOrdinarySave(t *testing.T) {
	for _, existing := range []bool{true, false} {
		t.Run(fmt.Sprintf("existingSack=%t", existing), func(t *testing.T) {
			f, app := itemMutationOpen1115(t, 1, false)
			if !existing {
				f.live.tick() // the source crossing leaves the old gold Sack cell
			}
			before := snapshotCurrentObjects(t, f)
			registry := f.live.world.SavedObjects()
			weapon := itemObjectByKey(t, registry, itemMutationWeapon1115)
			actor := newGroupActors1115(t, f.live.world)[newGroupA]
			if (groundAt(f.live.world.Sacks(), actor.X, actor.Y) != nil) != existing {
				t.Fatal("independent death-cell fixture does not distinguish new/existing Sack")
			}
			f.live.pending = append(f.live.pending, sim.Command{Kind: sim.KindDamage, Entity: actor.ID, X: actor.HP + 10})
			for tick := 0; tick < 100; tick++ {
				f.live.tick()
				row := itemObjectByKey(t, f.live.world.SavedObjects(), itemMutationWeapon1115)
				if row.Owner.Kind == sim.SavedOwnerSack {
					break
				}
			}
			current := f.live.world.SavedObjects()
			row := itemObjectByKey(t, current, itemMutationWeapon1115)
			dead := newGroupActors1115(t, f.live.world)[newGroupA]
			if dead.Alive() || dead.Decay < sim.DecayBones || dead.Dwell != 0 || row.Owner.Kind != sim.SavedOwnerSack || row.ID != weapon.ID || row.Spell != 0 || row.Value.SourceEquipment.Spell.Present || !reflect.DeepEqual(current.Effects, registry.Effects) || !itemMutationSpell1115(t, current, weapon.Spell).Retired {
				t.Fatal("actual source death did not transfer the same Weapon and retire its Spell", dead, row)
			}
			after := snapshotCurrentObjects(t, f)
			if after.SavedDocument.GroupBindings.Unavailable != "" {
				t.Fatal("current corpse Group graph is unavailable", after.SavedDocument.GroupBindings.Unavailable)
			}
			itemBinding := itemMutationBinding1115(t, after.SavedDocument.Objects.Items, weapon.ID)
			sackBinding := itemMutationBinding1115(t, after.SavedDocument.Objects.Sacks, row.Owner.Object)
			if itemBinding.ObjectIndex == 0 || sackBinding.ObjectIndex == 0 || itemMutationBinding1115(t, after.SavedDocument.Objects.Spells, weapon.Spell).ObjectIndex != 0 {
				t.Fatal("death Item/Sack/retired Spell current graph missing")
			}
			sack := after.SavedDocument.Document.Objects[sackBinding.ObjectIndex-1]
			contents, _ := savedObjectRefs(&sack, "Contents")
			if !slices.Equal(contents, []uint16{itemBinding.ObjectIndex}) {
				t.Fatal("death Sack does not own the exact current Item edge", contents)
			}
			for _, binding := range after.SavedDocument.Actors {
				if binding.EntityID != actor.ID {
					continue
				}
				r := after.SavedDocument.Document.Objects[binding.ObjectIndex-1]
				held, _ := savedObjectRefs(&r, "HeldWeapon")
				pack, _ := savedObjectRefs(&r, "Inventory")
				if binding.Retired || !slices.Equal(held, []uint16{0}) || len(pack) != 0 || itemObjectValue1115(t, r, "Health") != uint32(uint16(dead.HP)) || itemObjectValue1115(t, r, "Inventory1C") != 10000 || itemObjectValue1115(t, r, "Inventory20") != 0 {
					t.Fatal("retained corpse actor projected stale health/holdings/container")
				}
			}
			wantObjects := len(before.SavedDocument.Document.Objects) - 1
			if !existing {
				wantObjects++ // one new Sack, one explicitly retired old Spell
				if sackBinding.Unavailable != "" {
					t.Fatal("new death Sack lacks current ordinary projection", sackBinding)
				}
			}
			if len(after.SavedDocument.Document.Objects) != wantObjects {
				t.Fatal("death projection changed unrelated archive object population")
			}
			fresh, freshApp := itemMutationCheckpoint(t, f, app, 1)
			fresh, _ = itemMutationCheckpoint(t, fresh, freshApp, 1)
			for range 20 {
				f.live.tick()
				fresh.live.tick()
				itemMutationSame1115(t, snapshotCurrentObjects(t, f), snapshotCurrentObjects(t, fresh))
			}
		})
	}
}

func TestItemDeath1115RemovedActorRetiresUndroppableWeaponBeforeSave(t *testing.T) {
	f := itemMutationFront1115(t, 1)
	// ITEM-DEATH-012: independently authored source parameter15 zero leaves this Weapon
	// on the corpse until the actual actor-removal producer disposes it.
	f.Table.Weapons.EntryParams(1)[15] = 0
	doc := sackObjectsLiteral1115(t, f)
	weapon := literalItemRecord1115("Weapon", 0x0101, 1, itemMutationWeapon1115, nil, 0)
	newGroupSetValue1115(t, &weapon, "T0C", 1)
	newGroupSetValue1115(t, &weapon, "F44", 2)
	doc.Objects = append(doc.Objects, weapon)
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class == "Unit" && actorProjectionValue(t, *r, "Identity") == newGroupA {
			literalSavedObjectRefs(t, r, "HeldWeapon", []uint16{uint16(len(doc.Objects))}, false)
		}
	}
	doc, _, err := sav.ReindexDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("undroppable source Weapon import", town, err)
	}
	app := f.App("source actor removal")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	before := snapshotCurrentObjects(t, f)
	item := itemObjectByKey(t, f.live.world.SavedObjects(), itemMutationWeapon1115)
	actor := newGroupActors1115(t, f.live.world)[newGroupA]
	if item.Value.SourceEquipment.Definition.Suitable != 0 {
		t.Fatal("literal undroppable row was not bound")
	}
	f.live.pending = append(f.live.pending, sim.Command{Kind: sim.KindDamage, Entity: actor.ID, X: actor.HP + 1000})
	for tick := 0; tick < 100 && slices.ContainsFunc(f.live.world.Entities(), func(e sim.Entity) bool { return e.ID == actor.ID }); tick++ {
		f.live.tick()
	}
	if slices.ContainsFunc(f.live.world.Entities(), func(e sim.Entity) bool { return e.ID == actor.ID }) || itemObjectByKey(t, f.live.world.SavedObjects(), itemMutationWeapon1115).Owner.Kind != sim.SavedOwnerRetired {
		t.Fatal("actual terminal overshoot did not remove the actor and dispose its undroppable Weapon")
	}
	after := snapshotCurrentObjects(t, f)
	if itemMutationBinding1115(t, after.SavedDocument.Objects.Items, item.ID).ObjectIndex != 0 || len(after.SavedDocument.Document.Objects) != len(before.SavedDocument.Document.Objects)-1 {
		for _, stage := range []struct {
			name     string
			snapshot Snapshot
		}{{"before", before}, {"after", after}} {
			for i, record := range stage.snapshot.SavedDocument.Document.Objects {
				var identity uint32
				for _, value := range record.Values {
					if value.Name == "Identity" || value.Name == "This" {
						identity = value.Value
					}
				}
				t.Logf("%s object%d %s key=%x", stage.name, i+1, record.Class, identity)
			}
		}
		t.Fatalf("removed actor weapon binding=%+v object count=%d before=%d", itemMutationBinding1115(t, after.SavedDocument.Objects.Items, item.ID), len(after.SavedDocument.Document.Objects), len(before.SavedDocument.Document.Objects))
	}
	for _, record := range after.SavedDocument.Document.Objects {
		if record.Class == "Weapon" && itemObjectValue1115(t, record, "Identity") == item.Token.Identity {
			t.Fatal("retired weapon identity still has an ordinary record")
		}
	}
	found := false
	for _, binding := range after.SavedDocument.Actors {
		if binding.EntityID != actor.ID {
			continue
		}
		found = true
		r := after.SavedDocument.Document.Objects[binding.ObjectIndex-1]
		held, _ := savedObjectRefs(&r, "HeldWeapon")
		pack, _ := savedObjectRefs(&r, "Inventory")
		if !binding.Retired || !slices.Equal(held, []uint16{0}) || len(pack) != 0 {
			t.Fatal("retired actor retained an inbound Item edge")
		}
	}
	if !found {
		t.Fatal("actor retirement metadata disappeared")
	}
	fresh, _ := itemMutationCheckpoint(t, f, app, 1)
	for range 20 {
		f.live.tick()
		fresh.live.tick()
		itemMutationSame1115(t, snapshotCurrentObjects(t, f), snapshotCurrentObjects(t, fresh))
	}
}
