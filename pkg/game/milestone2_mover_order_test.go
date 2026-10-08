package game

import (
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

// The expected order comes from the current World, not another projected
// Document or projectSavedCurrentOrder. Only the explicit Entity-to-DTO binding
// is shared. Order144 is current state; bytes144..147 remain Document-only
// transport residue and are checked by mover1160ContinuationDifferences.
func mover1160CurrentOrderDifferences(state *SnapshotSAVDocument, w *sim.World) []string {
	retained, err := mover1160Retained(state)
	if err != nil {
		return []string{err.Error()}
	}
	if w == nil {
		return []string{"current orders lack World"}
	}
	_, orders, present := w.SavedGroups()
	if !present || len(orders) == 0 {
		return []string{"current order witness lacks a populated World registry"}
	}
	if state.GroupBindings == nil {
		return []string{"current order projection lacks Group bindings"}
	}
	if state.GroupBindings.Unavailable != "" {
		return []string{"current order projection unavailable: " + state.GroupBindings.Unavailable}
	}
	var differences []string
	byEntity := map[sim.EntityID]uint16{}
	byObject := map[uint16]sim.EntityID{}
	bind := func(entity sim.EntityID, index uint16, repeatedMember bool) {
		if prior, found := byEntity[entity]; found {
			if !repeatedMember || prior != index {
				differences = append(differences, fmt.Sprintf("current order%d has duplicate/conflicting DTO binding", entity))
			}
			return
		}
		if other, found := byObject[index]; found {
			differences = append(differences, fmt.Sprintf("current orders%d/%d alias DTO%d", other, entity, index))
		}
		byEntity[entity], byObject[index] = index, entity
	}
	for _, a := range state.Actors {
		if !a.Retired {
			bind(a.EntityID, a.ObjectIndex, false)
		}
	}
	for _, m := range state.GroupBindings.Members {
		if m.Bound {
			bind(m.EntityID, m.ObjectIndex, true)
		}
	}
	entities := map[sim.EntityID]sim.Entity{}
	for _, e := range w.Entities() {
		entities[e.ID] = e
	}
	seen := map[sim.EntityID]bool{}
	for _, order := range orders {
		index := byEntity[order.Entity]
		prefix := fmt.Sprintf("current order%d DTO%d", order.Entity, index)
		if seen[order.Entity] {
			differences = append(differences, prefix+" duplicate World order")
		}
		seen[order.Entity] = true
		r, found := retained[index]
		if !found {
			differences = append(differences, prefix+" lacks Unit-family DTO")
			continue
		}
		want, err := mover1160CurrentOrderBytes(order, entities)
		if err != nil {
			differences = append(differences, prefix+" outside current order witness: "+err.Error())
			continue
		}
		for i := range want {
			if r.order[i] != want[i] {
				differences = append(differences, fmt.Sprintf("%s Order144 byte%d Document=%02x World=%02x", prefix, i, r.order[i], want[i]))
			}
		}
		if !slices.Equal(r.routes[2], order.Patrol) {
			differences = append(differences, fmt.Sprintf("%s U158_90 differs: Document=%v World=%v", prefix, r.routes[2], order.Patrol))
		}
	}
	return differences
}

// Active attack and escort endpoints replace their retained address words.
// Nonzero SourceBinding.Identity is the independent World key oracle;
// no Document key or production projection helper supplies the expectation.
func mover1160CurrentOrderBytes(order sim.SavedActorOrder, entities map[sim.EntityID]sim.Entity) ([144]byte, error) {
	want := order.Raw
	actor, found := entities[order.Entity]
	if !found {
		return want, fmt.Errorf("typed order actor is absent")
	}
	attackOrder := order.State == 3 || order.Raw[8] == 2 || order.Raw[8] == 5 || order.Raw[8] == 6
	if attackOrder && actor.HasAttackTarget {
		target, present := entities[actor.AttackTarget]
		if actor.AttackTargetKind != sim.AttackTargetUnit || !present || target.SourceBinding.Class == 0 || target.SourceBinding.Identity == 0 {
			return want, fmt.Errorf("attack target lacks a retained World key")
		}
		binary.LittleEndian.PutUint32(want[0x0c:], target.SourceBinding.Identity)
	}
	escort := order.State == 8 || order.State == 0x11
	if !order.Authored && !escort {
		return want, nil
	}
	if order.Authored && order.State != 0xa && order.State != 0xb && order.State != 0xc && !escort {
		return want, fmt.Errorf("authored state%x has no supported current producer", order.State)
	}
	if order.Raw[8] != 0 && order.Raw[8] != 1 && order.Raw[8] != 0xb && !(escort && order.Raw[8] == 4) {
		return want, fmt.Errorf("active attack/inner order%d needs another typed producer", order.Raw[8])
	}
	if order.State == 0xa && !slices.Contains(order.Patrol, binary.LittleEndian.Uint16(order.Raw[2:])) {
		return want, fmt.Errorf("patrol cursor is absent from current ring")
	}
	if !escort {
		return want, nil
	}
	if order.RepairStage != 0 {
		return want, fmt.Errorf("escort stage%d skips key repair", order.RepairStage)
	}
	target, bound := order.EscortTarget, order.EscortBound
	rangeByte := order.Raw[0x70]
	if order.Authored {
		target, bound, rangeByte = actor.EscortTarget, actor.HasEscortTarget, actor.EscortRange
		want[0x70] = rangeByte
	}
	e, found := entities[target]
	if !bound || !found || target == actor.ID || e.SourceBinding.Class == 0 || e.SourceBinding.Identity == 0 {
		return want, fmt.Errorf("escort lacks a distinct live target with a retained World key")
	}
	binary.LittleEndian.PutUint32(want[0x10:], e.SourceBinding.Identity)
	if order.Raw[8] == 4 {
		binary.LittleEndian.PutUint32(want[0x18:], e.SourceBinding.Identity)
		want[0x14] = rangeByte
		if rangeByte == 0 {
			want[0x14] = actor.ScanRange
		}
	}
	return want, nil
}

func mover1160OrderControl(t *testing.T) (*FrontEnd, Snapshot, uint16) {
	t.Helper()
	f := unit1158FixtureFront(t)
	open, town, err := f.RestoreOriginal(mover1160AppFixture(t))
	if err != nil || town {
		t.Fatal(town, err)
	}
	if err := f.App("current order controls").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	s := mover1160Snapshot(t, f)
	_, orders, present := f.live.world.SavedGroups()
	if !present || len(orders) != 2 || orders[0].Authored || orders[0].State == 8 || orders[0].State == 0x11 || len(orders[0].Patrol) != 3 {
		t.Fatal("control needs two retained orders and a nonempty distinct order path", orders)
	}
	for _, a := range s.SavedDocument.Actors {
		if a.EntityID == orders[0].Entity {
			return f, s, a.ObjectIndex
		}
	}
	t.Fatal("control order lacks an explicit actor binding")
	return nil, Snapshot{}, 0
}

func TestMover1160CurrentOrderLossControls(t *testing.T) {
	f, s, index := mover1160OrderControl(t)
	for i := range 144 {
		t.Run(fmt.Sprintf("Order144-byte%d", i), func(t *testing.T) {
			bad := cloneSavedDocumentFixture(t, s.SavedDocument)
			for _, field := range bad.Document.Objects[index-1].Raw {
				if field.Name == "U158" {
					field.Bytes[i] ^= 0x80
				}
			}
			if diff, debt := mover1160ContinuationDifferences(bad, bad, f.live.world); len(diff)+len(debt) != 0 {
				t.Fatal("paired identical corruption must agree", diff, debt)
			}
			if diff := mover1160CurrentOrderDifferences(bad, f.live.world); len(diff) == 0 {
				t.Fatal("same wrong order in both Documents escaped current World")
			}
		})
	}
	for i := 144; i < 148; i++ {
		t.Run(fmt.Sprintf("transport-byte%d", i), func(t *testing.T) {
			bad := cloneSavedDocumentFixture(t, s.SavedDocument)
			for _, field := range bad.Document.Objects[index-1].Raw {
				if field.Name == "U158" {
					field.Bytes[i] ^= 0x80
				}
			}
			if diff := mover1160CurrentOrderDifferences(bad, f.live.world); len(diff) != 0 {
				t.Fatal("transport tail incorrectly treated as current World", diff)
			}
			if diff, _ := mover1160ContinuationDifferences(s.SavedDocument, bad, f.live.world); len(diff) == 0 {
				t.Fatal("retained Document comparison lost transport tail")
			}
		})
	}
}

func TestMover1160CurrentPathLossControls(t *testing.T) {
	f, s, index := mover1160OrderControl(t)
	for _, name := range mover1160Lists {
		for _, loss := range []string{"drop", "truncate", "append", "low-byte", "high-byte"} {
			t.Run(name+"-"+loss, func(t *testing.T) {
				bad := cloneSavedDocumentFixture(t, s.SavedDocument)
				r := &bad.Document.Objects[index-1]
				for j := range r.Raw {
					field := &r.Raw[j]
					if field.Name != name {
						continue
					}
					if len(field.Bytes) == 0 {
						t.Fatal("each path must have a nonempty control subject")
					}
					switch loss {
					case "drop":
						field.Bytes = nil
					case "truncate":
						field.Bytes = field.Bytes[:len(field.Bytes)-2]
					case "append":
						field.Bytes = append(field.Bytes, 0x43, 0x21)
					case "low-byte":
						field.Bytes[0] ^= 0x80
					case "high-byte":
						field.Bytes[1] ^= 0x80
					}
					for k := range r.Counts {
						if r.Counts[k].Name == name {
							r.Counts[k].Count = uint32(len(field.Bytes) / 2)
						}
					}
				}
				if diff, debt := mover1160ContinuationDifferences(bad, bad, f.live.world); len(diff)+len(debt) != 0 {
					t.Fatal("paired valid matching list corruption must agree", diff, debt)
				}
				diff := mover1160CurrentMotionDifferences(bad, f.live.world)
				diff = append(diff, mover1160CurrentOrderDifferences(bad, f.live.world)...)
				if len(diff) == 0 {
					t.Fatal("matching projected list loss escaped current World")
				}
			})
		}
	}
}

func TestMover1160CurrentTypedOrderTransforms(t *testing.T) {
	for _, tc := range []struct {
		name    string
		command int32
	}{{"patrol", 14}, {"defend", 11}, {"follow", 15}} {
		t.Run(tc.name, func(t *testing.T) {
			f, _ := currentScriptOrderFront(t, tc.command)
			for range 20 {
				sim.Step(f.live.world, nil)
				_, orders, _ := f.live.world.SavedGroups()
				if tc.command == 14 || orders[0].Raw[8] == 4 {
					break
				}
			}
			_, orders, _ := f.live.world.SavedGroups()
			beforeSave := f.live.world.Hash()
			s := mover1160Snapshot(t, f)
			if f.live.world.Hash() != beforeSave {
				t.Fatal("Snapshot changed the current World")
			}
			var subject sim.SavedActorOrder
			var index uint16
			for _, o := range orders {
				if o.Authored && (tc.command == 14 && o.State == 0xa || tc.command != 14 && (o.State == 8 || o.State == 0x11) && o.Raw[8] == 4) {
					subject = o
					for _, a := range s.SavedDocument.Actors {
						if a.EntityID == o.Entity {
							index = a.ObjectIndex
						}
					}
					break
				}
			}
			if index == 0 {
				t.Fatal("typed control never reached its named current order")
			}
			if tc.command != 14 {
				entities := map[sim.EntityID]sim.Entity{}
				for _, e := range f.live.world.Entities() {
					entities[e.ID] = e
				}
				want, err := mover1160CurrentOrderBytes(subject, entities)
				if err != nil || subject.Raw != want {
					t.Fatal("runtime did not synchronize the exact typed escort before Snapshot", err)
				}
				stale := cloneSavedDocumentFixture(t, s.SavedDocument)
				for _, field := range stale.Document.Objects[index-1].Raw {
					if field.Name == "U158" {
						binary.LittleEndian.PutUint32(field.Bytes[0x10:], 0xdecafbad)
						binary.LittleEndian.PutUint32(field.Bytes[0x18:], 0xfeedface)
					}
				}
				if diff := mover1160CurrentOrderDifferences(stale, f.live.world); len(diff) == 0 {
					t.Fatal("stale pre-command escort keys escaped the typed current witness")
				}
			}
			for i := range 144 {
				bad := cloneSavedDocumentFixture(t, s.SavedDocument)
				for _, field := range bad.Document.Objects[index-1].Raw {
					if field.Name == "U158" {
						field.Bytes[i] ^= 0x80
					}
				}
				if diff := mover1160CurrentOrderDifferences(bad, f.live.world); len(diff) == 0 {
					t.Fatal("accepted typed Order144 byte loss", i)
				}
			}
			t.Log("144 current order bytes checked, including transformed escort operands where present")
		})
	}
}

func TestMover1160CurrentOrderBindingControls(t *testing.T) {
	f, s, index := mover1160OrderControl(t)
	entity := s.SavedDocument.Actors[0].EntityID
	if s.SavedDocument.Actors[0].ObjectIndex != index {
		t.Fatal("binding control fixture order changed")
	}
	for _, control := range []string{"missing", "duplicate", "alias", "absent-groups", "unavailable"} {
		t.Run(control, func(t *testing.T) {
			bad := cloneSavedDocumentFixture(t, s.SavedDocument)
			switch control {
			case "missing":
				bad.Actors = bad.Actors[1:]
				bad.GroupBindings.Members = slices.DeleteFunc(bad.GroupBindings.Members, func(m SnapshotSAVGroupMemberBinding) bool { return m.Bound && m.EntityID == entity })
			case "duplicate":
				bad.Actors = append(bad.Actors, bad.Actors[0])
			case "alias":
				other := bad.Actors[1].ObjectIndex
				bad.Actors[0].ObjectIndex = other
				for i := range bad.GroupBindings.Members {
					if bad.GroupBindings.Members[i].EntityID == entity {
						bad.GroupBindings.Members[i].ObjectIndex = other
					}
				}
			case "absent-groups":
				bad.GroupBindings = nil
			case "unavailable":
				bad.GroupBindings.Unavailable = "unsupported controlled order arm"
			}
			if diff := mover1160CurrentOrderDifferences(bad, f.live.world); len(diff) == 0 {
				t.Fatal("accepted missing or ambiguous current order authority")
			}
		})
	}
	if diff := mover1160CurrentOrderDifferences(s.SavedDocument, &sim.World{}); len(diff) == 0 {
		t.Fatal("empty World order registry made the current oracle vacuous")
	}
}

func TestMover1160UnsupportedTypedOrderNamed(t *testing.T) {
	for _, tc := range []struct {
		name, reason string
		change       func(*sim.SavedActorOrder, map[sim.EntityID]sim.Entity)
	}{
		{"authored-state", "authored state", func(o *sim.SavedActorOrder, _ map[sim.EntityID]sim.Entity) { o.State = 0x16 }},
		{"inner-order", "another typed producer", func(o *sim.SavedActorOrder, _ map[sim.EntityID]sim.Entity) { o.Raw[8] = 2 }},
		{"repair-stage", "skips key repair", func(o *sim.SavedActorOrder, _ map[sim.EntityID]sim.Entity) { o.RepairStage = 1 }},
		{"target-key", "retained World key", func(_ *sim.SavedActorOrder, entities map[sim.EntityID]sim.Entity) {
			e := entities[7]
			e.SourceBinding = sim.SourceBinding{}
			entities[7] = e
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := sim.SavedActorOrder{Entity: 3, Authored: true, State: 8}
			entities := map[sim.EntityID]sim.Entity{
				3: {ID: 3, HasEscortTarget: true, EscortTarget: 7, EscortRange: 2},
				7: {ID: 7, SourceBinding: sim.SourceBinding{Class: 1, Identity: 0x12345678}},
			}
			if _, err := mover1160CurrentOrderBytes(o, entities); err != nil {
				t.Fatal("supported typed baseline", err)
			}
			tc.change(&o, entities)
			if _, err := mover1160CurrentOrderBytes(o, entities); err == nil || !strings.Contains(err.Error(), tc.reason) {
				t.Fatal("unsupported typed boundary not named", err)
			}
		})
	}
}
