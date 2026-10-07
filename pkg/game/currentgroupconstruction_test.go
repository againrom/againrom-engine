package game

import (
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentMixedPlayerFixture(t *testing.T) (*FrontEnd, Snapshot) {
	t.Helper()
	f := groupDocumentFront1115(t)
	f.SetDeterministicFrames(true)
	raw := groupDocumentLiteral1115(t, f)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal(town, err)
	}
	if err := f.App("current Group construction").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	a := registryActors1111(t, f.live.world)[35]
	f.live.enqueue(uint32(a.ID), 20, 16)
	f.live.tick()
	s := groupDocumentSnapshot1115(t, f)
	return f, s
}

func TestCurrentGroupConstructionKeepsExistingPlayerPopulation(t *testing.T) {
	f, s := currentMixedPlayerFixture(t)
	players, present := f.live.world.SavedGroupPlayers()
	if !present || !reflect.DeepEqual(players, []sim.SavedGroupPlayer{{ID: 1, Slot: 1}}) {
		t.Fatal("fixture lost exact native Player", players, present)
	}
	hash := f.live.world.Hash()
	current, err := f.materializeCurrentWorld(s, f.live.world)
	if err != nil {
		t.Fatal(err)
	}
	if len(current.GroupBindings.Players) != 2 || len(nativeSavedPlayerBindings(current.GroupBindings)) != 1 || f.live.world.Hash() != hash {
		t.Fatal("constructor changed native Player population or missed an ordinary root")
	}
	for _, p := range current.GroupBindings.Players {
		slot, err := savedStructureValue(&current.Document.Objects[p.ObjectIndex-1], "Slot")
		if err != nil || p.Constructed != (slot == 0) || !p.Constructed && p.ID != players[0].ID {
			t.Fatal("mixed Player binding lost its exact presence", p, slot, err)
		}
	}
	if _, err := cloneSavedDocument(current); err != nil {
		t.Fatal("constructed bindings fail detached validation", err)
	}
}

func TestCurrentSavedOrderCursorOrdinaryEdit(t *testing.T) {
	f := actorRegistryFront1111(t)
	open, town, err := f.RestoreOriginal(savedGroupPayload1113())
	if err != nil || town {
		t.Fatal(town, err)
	}
	if err := f.App("current Order cursor").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	firstGroupPass1113(t, f.live.world)
	a := registryActors1111(t, f.live.world)[35]
	f.live.enqueue(uint32(a.ID), 20, 16)
	f.live.tick()
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "current cursor")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	var object uint16
	for _, binding := range leaf.Bindings {
		if !binding.Structure && binding.ID == a.ID {
			object = binding.Object
		}
	}
	if object == 0 {
		t.Fatal("current Order has no ordinary actor")
	}
	binary.LittleEndian.PutUint16(crossingRawField(t, &doc.Objects[object-1], "U158")[2:], 0x0705)
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	var previous *sim.World
	for cycle := range 2 {
		cold := actorRegistryFront1111(t)
		open, town, err := cold.RestoreOriginal(raw)
		if err != nil || town {
			t.Fatal(town, err)
		}
		if err := cold.App("edited current cursor").OpenMission(open); err != nil {
			t.Fatal(err)
		}
		if previous != nil && previous.Hash() != cold.live.world.Hash() {
			currentMenuWorldDiagnostics(t, previous, cold.live.world)
			t.Fatal("second SAV changed edited cursor or World")
		}
		_, orders, _ := cold.live.world.SavedGroups()
		found := false
		for _, order := range orders {
			if order.Entity == a.ID {
				found = true
				if binary.LittleEndian.Uint16(order.Raw[2:]) != 0x0705 || order.Raw[10] != 20 || order.Raw[11] != 16 {
					t.Fatal("mover replaced the ordinary cursor or its independent Move destination", order.Raw)
				}
			}
		}
		if !found {
			t.Fatal("edited current Order disappeared")
		}
		if previous != nil {
			for range 20 {
				sim.Step(previous, nil)
				sim.Step(cold.live.world, nil)
				if previous.Hash() != cold.live.world.Hash() {
					t.Fatal("edited cursor successor diverged")
				}
			}
		}
		s, _, err := cold.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err = cold.ExportCurrentSave(s, "second current cursor")
		if err != nil {
			t.Fatal(err)
		}
		previous = cold.live.world
		t.Logf("cycle %d preserves the independent ordinary cursor", cycle)
	}
}
