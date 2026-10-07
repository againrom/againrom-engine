package game

import (
	"encoding/binary"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func currentScriptOrderFront(t *testing.T, command int32) (*FrontEnd, sim.EntityID) {
	t.Helper()
	front := groupDocumentFront1115(t)
	doc, err := sav.DecodeDocumentData(groupDocumentLiteral1115(t, front))
	if err != nil {
		t.Fatal(err)
	}
	player := &doc.Objects[doc.Players[0]-1]
	group := &player.Groups[0]
	// Null membership is tested elsewhere; this real script needs a fully
	// materialized group admitted by the ordinary current Group dispatcher.
	for i := range group.RefSlots {
		if group.RefSlots[i].Name == "Actors" {
			group.RefSlots[i].Objects = slices.DeleteFunc(group.RefSlots[i].Objects, func(v uint16) bool { return v == 0 })
		}
	}
	for i := range group.Counts {
		if group.Counts[i].Name == "Actors" {
			group.Counts[i].Count--
		}
	}
	for i := range player.Counts {
		if player.Counts[i].Name == "Actors" {
			player.Counts[i].Count--
		}
	}
	for i := range group.Raw {
		if group.Raw[i].Name == "G3C" {
			group.Raw[i].Bytes[0x45] = 1
		}
	}
	for i := range doc.Objects {
		for j := range doc.Objects[i].Raw {
			field := &doc.Objects[i].Raw[j]
			if field.Name == "U158" {
				binary.LittleEndian.PutUint32(field.Bytes[0x10:], 0xdecafbad)
				binary.LittleEndian.PutUint32(field.Bytes[0x18:], 0xfeedface)
			}
		}
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	open, town, err := front.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("prepare controlled source", town, err)
	}
	clear(raw)
	if err := front.App("current order document").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	actors := registryActors1111(t, front.live.world)
	target := actors[33].ID
	groups, incoming, _ := front.live.world.SavedGroups()
	for _, order := range incoming {
		if binary.LittleEndian.Uint32(order.Raw[0x10:]) != 0xdecafbad || binary.LittleEndian.Uint32(order.Raw[0x18:]) != 0xfeedface {
			t.Fatal("source fixture lost its distinct pre-command operands")
		}
	}
	args := [10]int32{command, 1}
	if command == 14 {
		args[1], args[2] = 24, 24
	}
	script, err := sim.NewScript(nil, []sim.ScriptInstant{{Op: sim.ScriptInstantGroupOrder, HasGroup: true, Group: groups[0].Selector, HasUnit: command != 14, Unit: target, Args: args}},
		[]sim.ScriptTrigger{{Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true}})
	if err != nil {
		t.Fatal(err)
	}
	controlled, err := sim.NewControlledScriptWorld(front.live.world, script)
	if err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(controlled)
	*front.live.world = *controlled
	for range 20 {
		sim.Step(front.live.world, nil)
		_, orders, _ := front.live.world.SavedGroups()
		if len(orders) != 0 && orders[0].Authored {
			return front, target
		}
	}
	t.Fatal("ordinary script never issued the current order")
	return nil, 0
}

func TestSavedCurrentOrder1115ScriptToDocumentAndNativeContinuation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		command int32
		state   uint32
	}{{"patrol", 14, 0xa}, {"defend", 11, 8}, {"follow", 15, 0x11}} {
		t.Run(tc.name, func(t *testing.T) {
			front, target := currentScriptOrderFront(t, tc.command)
			var current Snapshot
			var currentOrders []sim.SavedActorOrder
			// Include a following decision pass, not just setter fields.
			for range 20 {
				sim.Step(front.live.world, nil)
				_, currentOrders, _ = front.live.world.SavedGroups()
				if tc.command == 14 || currentOrders[0].Raw[8] == 4 {
					break
				}
			}
			beforeSave := front.live.world.Hash()
			current = groupDocumentSnapshot1115(t, front)
			if current.SavedDocument.GroupBindings.Unavailable != "" {
				t.Fatal("known current order was not projected", current.SavedDocument.GroupBindings.Unavailable)
			}
			savBytes, err := sav.EncodeDocumentData(*current.SavedDocument.Document)
			if err != nil {
				t.Fatal(err)
			}
			produced, err := front.ExportCurrentSave(current, "current script order")
			if err != nil {
				t.Fatal(err)
			}
			if front.live.world.Hash() != beforeSave {
				t.Fatal("Snapshot or SAVE mutated the current World")
			}
			file, err := sav.Open(produced)
			if err != nil {
				t.Fatal(err)
			}
			graph, err := file.ActorGraph()
			if err != nil {
				t.Fatal(err)
			}
			byKey := make(map[uint32]sav.ActorRecord)
			for _, actor := range graph.Actors {
				byKey[actor.Identity] = actor
			}
			checked := 0
			for _, binding := range current.SavedDocument.Actors {
				key := actorProjectionValue(t, current.SavedDocument.Document.Objects[binding.ObjectIndex-1], "Identity")
				got := byKey[key]
				var want sim.SavedActorOrder
				for _, order := range currentOrders {
					if order.Entity == binding.EntityID {
						want = order
						break
					}
				}
				if !want.Authored {
					t.Fatal("test skipped actual native setter")
				}
				if tc.command != 14 && binding.EntityID == target {
					continue
				}
				checked++
				if got.ActorState != tc.state {
					t.Fatal("current state missing", got.ActorState, tc.state)
				}
				if !slices.Equal(got.Order[:144], want.Raw[:]) {
					t.Fatal("ordinary SAVE changed current Order144", got.Order[:144], want.Raw)
				}
				if tc.command == 14 {
					if !slices.Equal(got.Patrol, want.Patrol) || binary.LittleEndian.Uint16(got.Order[2:]) != binary.LittleEndian.Uint16(want.Raw[2:]) || binary.LittleEndian.Uint32(got.Order[4:]) != binary.LittleEndian.Uint32(want.Raw[4:]) {
						t.Fatal("current patrol ring/cursor/latch lost")
					}
				} else {
					if got.Order[8] != 4 || binary.LittleEndian.Uint32(got.Order[0x10:]) != 1007 || binary.LittleEndian.Uint32(got.Order[0x18:]) != 1007 || got.Order[0x70] != 1 || got.Order[0x14] != 1 {
						t.Fatalf("typed target/closing/range projection lost: %x", got.Order[:0x20])
					}
					if binary.LittleEndian.Uint32(want.Raw[0x10:]) != 1007 || binary.LittleEndian.Uint32(want.Raw[0x18:]) != 1007 {
						t.Fatal("actual script did not synchronize current escort keys before SAVE")
					}
				}
			}
			if checked < 2 {
				t.Fatal("missing current actor population")
			}
			// Exercise the produced SAV through the production original loader,
			// not only the format reader. Authored provenance becomes imported
			// provenance, so compare the actual target/ring meaning, not that bit.
			fromSAV := groupDocumentFront1115(t)
			savOpen, savTown, err := fromSAV.RestoreOriginal(savBytes)
			if err != nil || savTown {
				t.Fatal("current SAV prepare", savTown, err)
			}
			if err := fromSAV.App("current SAV reload").OpenMission(savOpen); err != nil {
				t.Fatal(err)
			}
			clear(savBytes)
			// Unlike the initial-import helper, this is a changed-world reload:
			// facing and other current values may differ from literal defaults.
			savActors := make(map[uint16]sim.Entity)
			for _, actor := range fromSAV.live.world.Entities() {
				if actor.SourceBinding.Class != 0 {
					savActors[actor.SourceBinding.TypeID] = actor
				}
			}
			if len(savActors) != 3 {
				t.Fatal("produced SAV lost source actor identities")
			}
			_, loadedOrders, _ := fromSAV.live.world.SavedGroups()
			for _, order := range loadedOrders {
				if order.Authored {
					t.Fatal("SAV reader invented native-authored provenance")
				}
				if tc.command == 14 {
					if order.State != 0xa || len(order.Patrol) != 2 || !slices.Contains(order.Patrol, binary.LittleEndian.Uint16(order.Raw[2:])) {
						t.Fatal("produced SAV lost current patrol", order)
					}
				} else if order.Entity != savActors[33].ID && (order.State != tc.state || !order.EscortBound || order.EscortTarget != savActors[33].ID || order.Raw[0x70] != 1) {
					t.Fatal("produced SAV did not resolve current typed escort", order)
				}
			}
			for range 20 {
				fromSAV.live.tick()
			}
			if issues := fromSAV.live.world.SavedGroupIssues(); len(issues) != 0 {
				t.Fatal("produced SAV cannot continue known current order", issues)
			}
			native, err := EncodeSave(current, "current script order")
			if err != nil {
				t.Fatal(err)
			}
			decoded, _, err := DecodeSave(native)
			if err != nil {
				t.Fatal(err)
			}
			fresh := groupDocumentFront1115(t)
			open, town, err := fresh.Restore(decoded)
			if err != nil || town {
				t.Fatal("fresh current native", town, err)
			}
			if err := fresh.App("fresh current order").OpenMission(open); err != nil {
				t.Fatal(err)
			}
			again := groupDocumentSnapshot1115(t, fresh)
			if !reflect.DeepEqual(current.SavedDocument, again.SavedDocument) {
				t.Fatal("fresh native LOAD lost current document")
			}
			for range 20 {
				front.live.tick()
				fresh.live.tick()
				if front.live.world.Hash() != fresh.live.world.Hash() {
					t.Fatal("current order continuation changed across native SAVE")
				}
			}
		})
	}
}

func TestSavedCurrentOrder1115TargetAndStateGapsAreAtomic(t *testing.T) {
	front, target := currentScriptOrderFront(t, 15)
	state := groupDocumentSnapshot1115(t, front).SavedDocument
	_, orders, _ := front.live.world.SavedGroups()
	var sourceOrder sim.SavedActorOrder
	for _, order := range orders {
		if order.State == 0x11 {
			sourceOrder = order
			break
		}
	}
	var object uint16
	for _, actor := range state.Actors {
		if actor.EntityID == sourceOrder.Entity {
			object = actor.ObjectIndex
		}
	}
	if object == 0 || !sourceOrder.Authored {
		t.Fatal("missing current escort control")
	}
	for _, name := range []string{"missing target", "self target", "absent actor", "missing binding", "zero key", "colliding key", "repair stage", "actor stage", "inner cast", "native attack", "unsupported state", "empty patrol", "missing cursor"} {
		t.Run(name, func(t *testing.T) {
			candidate, err := cloneSavedDocument(state)
			if err != nil {
				t.Fatal(err)
			}
			order := sourceOrder
			entities := front.live.world.Entities()
			var actor *sim.Entity
			for i := range entities {
				if entities[i].ID == order.Entity {
					actor = &entities[i]
					break
				}
			}
			setKey := func(id sim.EntityID, key uint32) {
				t.Helper()
				for _, a := range candidate.Actors {
					if a.EntityID != id {
						continue
					}
					for i := range candidate.Document.Objects[a.ObjectIndex-1].Values {
						value := &candidate.Document.Objects[a.ObjectIndex-1].Values[i]
						if value.Name == "Identity" {
							value.Value = key
							return
						}
					}
				}
				t.Fatal("missing target key fixture")
			}
			switch name {
			case "missing target":
				actor.HasEscortTarget = false
			case "self target":
				actor.EscortTarget = actor.ID
			case "absent actor":
				entities = slices.DeleteFunc(entities, func(e sim.Entity) bool { return e.ID == order.Entity })
			case "missing binding":
				candidate.Actors = slices.DeleteFunc(candidate.Actors, func(a SnapshotSAVActor) bool { return a.EntityID == target })
				candidate.GroupBindings.Members = slices.DeleteFunc(candidate.GroupBindings.Members, func(m SnapshotSAVGroupMemberBinding) bool { return m.Bound && m.EntityID == target })
			case "zero key":
				setKey(target, 0)
			case "colliding key":
				setKey(order.Entity, 1007)
			case "repair stage":
				order.RepairStage = 1
			case "actor stage":
				for i := range candidate.Document.Objects[object-1].Values {
					v := &candidate.Document.Objects[object-1].Values[i]
					if v.Name == "Stage" {
						v.Value = 1
					}
				}
			case "inner cast":
				order.Raw[8] = 8
			case "native attack":
				actor.HasAttackTarget = true
			case "unsupported state":
				order.State = 0x16
			case "empty patrol":
				order.State, order.Patrol = 0xa, nil
			case "missing cursor":
				order.State, order.Patrol = 0xa, []uint16{0x1234}
				binary.LittleEndian.PutUint16(order.Raw[2:], 0x5678)
			}
			before, err := cloneSavedDocument(candidate)
			if err != nil {
				t.Fatal(err)
			}
			gap, err := projectSavedCurrentOrder(candidate, entities, object, order)
			if err != nil || gap == "" {
				t.Fatal("expected explicit field coverage gap", gap, err)
			}
			if !reflect.DeepEqual(candidate, before) {
				t.Fatal("refused target/state published partial document")
			}
		})
	}
}
