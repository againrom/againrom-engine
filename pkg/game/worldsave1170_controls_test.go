package game

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Compare current values and the alias relation independently of newly
// allocated native object numbers. No World bytes or gameplay fields are
// normalized. Every nonempty item and every Sack must retain an exact,
// injective object binding after the cold import.
func world1170Holdings(t *testing.T, world *sim.World, actor sim.EntityID, pack []sim.ItemInstance, sacks []sim.Sack) {
	t.Helper()
	forward, reverse := map[sim.SavedObjectID]sim.SavedObjectID{}, map[sim.SavedObjectID]sim.SavedObjectID{}
	bind := func(want, got sim.SavedObjectID) {
		if want == 0 || got == 0 || forward[want] != 0 && forward[want] != got || reverse[got] != 0 && reverse[got] != want {
			t.Fatalf("current object relation lost: %d -> %d", want, got)
		}
		forward[want], reverse[got] = got, want
	}
	items := func(want, got []sim.ItemInstance) {
		if len(want) != len(got) {
			t.Fatal("current container length changed")
		}
		for i, a := range want {
			b := got[i]
			bind(a.ObjectID, b.ObjectID)
			b.ObjectID = a.ObjectID // The relation above owns this namespace.
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("current item fields changed at slot %d: %+v / %+v", i, a, b)
			}
		}
	}
	currentPack, ok := world.CarriedItems(actor)
	if !ok {
		t.Fatal("current actor pack disappeared")
	}
	items(pack, currentPack)
	currentSacks := world.Sacks()
	if len(sacks) != len(currentSacks) {
		t.Fatal("current Sack roots changed")
	}
	for i, a := range sacks {
		b := currentSacks[i]
		bind(a.ObjectID, b.ObjectID)
		if a.X != b.X || a.Y != b.Y || a.Gold != b.Gold || !slices.Equal(a.Items, b.Items) {
			t.Fatal("current Sack fields/order changed")
		}
		items(a.ItemInstances, b.ItemInstances)
	}
}

// This oracle starts with values sampled from the live World before SAVE. It
// reads named wire values and joins the cell's key back to its runtime actor;
// it never compares two projected Documents or two opaque digests.
func world1170WireDifference(raw []byte, runtime uint32, hp int32, pos [2]int32, sacks int, results [100]int32, deadline sim.ActionClock) string {
	d, err := sav.DecodeDocumentData(raw)
	if err != nil {
		return err.Error()
	}
	if d.World == nil || len(d.World.Sacks) != sacks || d.World.Session.Results != results {
		return "current roots or Results lost"
	}
	var actor *sav.DocumentRecordData
	for i := range d.Objects {
		r := &d.Objects[i]
		id, err := savedStructureValue(r, "RuntimeID")
		if err == nil && id == runtime && (r.Class == "Human" || r.Class == "Unit" || r.Class == "Humanoid") {
			if actor != nil {
				return "ambiguous runtime actor"
			}
			actor = r
		}
	}
	if actor == nil {
		return "runtime actor lost"
	}
	health, err := savedStructureValue(actor, "Health")
	if err != nil || int32(int16(health)) != hp {
		return "current health lost"
	}
	end, err := savedStructureValue(actor, "U138")
	if err != nil || !deadline.Known || end != deadline.End {
		return "current action deadline lost"
	}
	p, err := savedMotionRaw(actor, "Block12", 12)
	if err != nil {
		return err.Error()
	}
	cell := binary.LittleEndian.Uint16(p)
	if [2]int32{int32(cell&255)*256 + int32(p[4]), int32(cell>>8)*256 + int32(p[5])} != pos {
		return "current near-cell/fraction lost"
	}
	key, _ := savedStructureValue(actor, "Identity")
	linked := false
	for _, c := range d.World.Cells {
		if c.Cell == cell && (c.GroundActor == key || c.AirActor == key) {
			linked = true
		}
	}
	if !linked {
		return "position and current cell reference disagree"
	}
	return ""
}

func world1170LossControls(t *testing.T, s Snapshot, original []byte, world *sim.World, actor sim.Entity) {
	t.Helper()
	pos := world1170Position(world, actor)
	oracle := func(raw []byte) string {
		return world1170WireDifference(raw, actor.SourceBinding.RuntimeID, actor.HP, pos, len(world.Sacks()), world.ScriptRegisters(), actor.ActionClock)
	}
	doc, err := currentWorldDocument(s, false)
	if err != nil {
		t.Fatal(err)
	}
	// Constructor keys also share the final allocator's exclusion set. A raw
	// missing order reference must not accidentally bind a newly created Sack.
	reservation, err := cloneSavedDocument(s.SavedDocument)
	if err != nil {
		t.Fatal(err)
	}
	var orderIndex uint16
	for _, binding := range reservation.Actors {
		if binding.EntityID == actor.ID {
			orderIndex = binding.ObjectIndex
		}
	}
	if orderIndex == 0 {
		t.Fatal("reservation control lacks actor binding")
	}
	order, err := savedMotionRaw(&reservation.Document.Objects[orderIndex-1], "U158", 148)
	if err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint32(order[0xc:], 0x70000000)
	binary.LittleEndian.PutUint32(order[0x10:], 0x01000000)
	if err := projectItems(reservation, world); err != nil {
		t.Fatal(err)
	}
	reserved, err := sav.RemintDocumentKeys(*reservation.Document)
	if err != nil {
		t.Fatal(err)
	}
	order, err = savedMotionRaw(&reserved.Objects[orderIndex-1], "U158", 148)
	if err != nil || binary.LittleEndian.Uint32(order[0xc:]) != 0x70000000 || binary.LittleEndian.Uint32(order[0x10:]) != 0x01000000 {
		t.Fatal("temporary Sack key rebound a missing raw order reference", err)
	}
	if why := oracle(original); why == "" {
		t.Fatal("unchanged old Body/Document passed current-state oracle")
	}
	for _, loss := range []string{"current position", "reference permutation", "new root omission", "Results omission", "action deadline omission"} {
		t.Run(loss, func(t *testing.T) {
			d, err := sav.CloneDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			var target *sav.DocumentRecordData
			for i := range d.Objects {
				r := &d.Objects[i]
				id, _ := savedStructureValue(r, "RuntimeID")
				if id == actor.SourceBinding.RuntimeID && (r.Class == "Human" || r.Class == "Unit") {
					target = r
				}
			}
			if target == nil {
				t.Fatal("probe target absent")
			}
			switch loss {
			case "action deadline omission":
				savedObjectSetValue(target, "U138", actor.ActionClock.End-81)
			case "Results omission":
				d.World.Session.Results = [100]int32{}
			case "current position":
				p, _ := savedMotionRaw(target, "Block12", 12)
				p[0]++ // Keep the complete current cells while making Position stale.
			case "reference permutation":
				key, _ := savedStructureValue(target, "Identity")
				for i := range d.Objects {
					r := &d.Objects[i]
					if r != target && (r.Class == "Human" || r.Class == "Unit") {
						other, _ := savedStructureValue(r, "Identity")
						savedObjectSetValue(r, "Identity", key)
						savedObjectSetValue(target, "Identity", other)
						break
					}
				}
			case "new root omission":
				root := d.World.Sacks[len(d.World.Sacks)-1]
				key, _ := savedStructureValue(&d.Objects[root-1], "Identity")
				d.World.Sacks = d.World.Sacks[:len(d.World.Sacks)-1]
				for i := range d.World.Cells {
					if d.World.Cells[i].Sack == key {
						d.World.Cells[i].Sack = 0
					}
				}
				incoming := savedDocumentIncoming(&d)
				retired := []uint16{root}
				for at := 0; at < len(retired); at++ {
					for _, refs := range d.Objects[retired[at]-1].RefSlots {
						for _, id := range refs.Objects {
							if id != 0 && incoming[id] == 1 && !slices.Contains(retired, id) {
								retired = append(retired, id)
							}
						}
					}
				}
				d, _, err = sav.RetireDocumentData(d, retired)
				if err != nil {
					if !strings.Contains(err.Error(), "current ownership node was retired") {
						t.Fatal("single-cause omission produced an unrelated graph error", err)
					}
					t.Skipf("not an independent wire-oracle control: the current ownership node could not be retired: %v", err)
				}
			}
			wire, err := sav.EncodeDocumentData(d)
			if err != nil {
				t.Fatal(err)
			}
			if why := oracle(wire); why == "" {
				t.Fatalf("%s loss passed current-state oracle", loss)
			}
		})
	}
	// Persisted metadata is authoritative, so a bijective actor-index swap is
	// rejected even when the records have the same class and equal field values.
	broken, err := cloneSavedDocument(s.SavedDocument)
	if err != nil {
		t.Fatal(err)
	}
	if len(broken.Actors) < 2 {
		t.Fatal("binding control lacks population")
	}
	broken.Actors[0].ObjectIndex, broken.Actors[1].ObjectIndex = broken.Actors[1].ObjectIndex, broken.Actors[0].ObjectIndex
	s.SavedDocument = broken
	if _, err := currentWorldDocument(s, false); err == nil {
		t.Fatal(fmt.Errorf("permuted exact bindings were silently rebound"))
	}
}

// Reintroduce just the retired zero-countdown root and its exclusive child.
// The current actors, clocks, application, layers and other roots are kept.
// The encoder must accept the complete graph, while the independently sampled
// empty current root population must reject this resurrection.
func world1170ExpiredRootControl(t *testing.T, aliveWire, expiredWire []byte) {
	t.Helper()
	alive, err := sav.DecodeDocumentData(aliveWire)
	if err != nil || len(alive.World.Effects) != 1 {
		t.Fatal("live root control", err)
	}
	expired, err := sav.DecodeDocumentData(expiredWire)
	if err != nil || len(expired.World.Effects) != 0 {
		t.Fatal("current empty root oracle", err)
	}
	root := alive.Objects[alive.World.Effects[0]-1]
	var children []uint16
	for _, slot := range root.RefSlots {
		for _, child := range slot.Objects {
			if child != 0 {
				children = append(children, child)
			}
		}
	}
	if len(children) != 1 {
		t.Fatal("resurrection control requires one exclusive child")
	}
	child := alive.Objects[children[0]-1]
	if len(child.RefSlots) != 0 {
		t.Fatal("resurrection child has another graph")
	}
	rootIndex, childIndex := uint16(len(expired.Objects)+1), uint16(len(expired.Objects)+2)
	for i := range root.RefSlots {
		for j, ref := range root.RefSlots[i].Objects {
			if ref != 0 {
				root.RefSlots[i].Objects[j] = childIndex
			}
		}
	}
	used := map[uint32]bool{}
	for _, r := range expired.Objects {
		for _, v := range r.Values {
			used[v.Value] = true
		}
	}
	key := uint32(0x79000000)
	for _, r := range []*sav.DocumentRecordData{&root, &child} {
		for used[key] {
			key++
		}
		savedObjectSetValue(r, "Identity", key)
		used[key] = true
	}
	expired.Objects = append(expired.Objects, root, child)
	expired.World.Effects = append(expired.World.Effects, rootIndex)
	mutant, _, err := sav.ReindexDocumentData(expired)
	if err != nil {
		t.Fatal("resurrection must remain a complete graph", err)
	}
	wire, err := sav.EncodeDocumentData(mutant)
	if err != nil {
		t.Fatal("valid resurrection output", err)
	}
	read, err := sav.DecodeDocumentData(wire)
	if err != nil || len(read.World.Effects) == 0 {
		t.Fatal("single-cause expired-root control did not discriminate", err)
	}
}
