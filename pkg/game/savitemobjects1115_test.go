package game

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// These literal records use only the established detached archive grammar.
// They do not call the production Item/Effect/Spell projection builders.
func literalItemRecord1115(class string, code uint16, count uint32, key uint32, effects []uint16, spell uint16) sav.DocumentRecordData {
	r := sav.DocumentRecordData{Class: class, Raw: []sav.DocumentRawData{{Name: "Block12", Bytes: []byte{7, 8, 7, 8, 128, 128, 0x71, 0x83, 0, 0, 0, 0}}}, Values: []sav.DocumentValueData{
		{Name: "RuntimeID", Value: key + 1}, {Name: "T0C", Value: 0}, {Name: "T0E", Value: 0x1234}, {Name: "T08", Value: 8}, {Name: "T18", Value: 0xabcd}, {Name: "T1C", Value: 31}, {Name: "Identity", Value: key}, {Name: "Reference", Value: 0xfedcba98},
		{Name: "F40", Value: uint32(code)}, {Name: "F42", Value: count}, {Name: "F44", Value: 0}, {Name: "F45", Value: 0xa5}, {Name: "F46", Value: 0xb6}, {Name: "F47", Value: 0xc7}, {Name: "F48", Value: 0xd8e9}, {Name: "F4A", Value: 2},
	}, Counts: []sav.DocumentCountData{{Name: "Effects", Count: uint32(len(effects))}}, RefSlots: []sav.DocumentRefsData{{Name: "Effects", Objects: slices.Clone(effects)}}}
	if class == "Weapon" {
		r.Raw = append(r.Raw, sav.DocumentRawData{Name: "W52", Bytes: make([]byte, 24)}, sav.DocumentRawData{Name: "W6A", Bytes: make([]byte, 22)})
		r.Values = append(r.Values, sav.DocumentValueData{Name: "W50", Value: 1})
		r.RefSlots = append(r.RefSlots, sav.DocumentRefsData{Name: "WeaponSpell", Objects: []uint16{spell}})
	}
	slices.SortFunc(r.Values, func(a, b sav.DocumentValueData) int { return strings.Compare(a.Name, b.Name) })
	return r
}

func literalSavedEffectRecord(key uint32) sav.DocumentRecordData {
	r := literalItemRecord1115("Effect", 1, 1, key, nil, 0)
	r.Values = slices.DeleteFunc(r.Values, func(v sav.DocumentValueData) bool { return strings.HasPrefix(v.Name, "F") })
	r.Values = append(r.Values, sav.DocumentValueData{Name: "E0C"}, sav.DocumentValueData{Name: "E3C", Value: 8}, sav.DocumentValueData{Name: "E3D", Value: 1}, sav.DocumentValueData{Name: "E40", Value: 17})
	slices.SortFunc(r.Values, func(a, b sav.DocumentValueData) int { return strings.Compare(a.Name, b.Name) })
	r.RefSlots, r.Counts = nil, nil
	return r
}

func literalSavedObjectRefs(t *testing.T, r *sav.DocumentRecordData, name string, refs []uint16, counted bool) {
	t.Helper()
	found := false
	for i := range r.RefSlots {
		if r.RefSlots[i].Name == name {
			r.RefSlots[i].Objects = slices.Clone(refs)
			found = true
		}
	}
	if !found {
		t.Fatalf("fixture missing refs %s", name)
	}
	if counted {
		found = false
		for i := range r.Counts {
			if r.Counts[i].Name == name {
				r.Counts[i].Count = uint32(len(refs))
				found = true
			}
		}
		if !found {
			t.Fatalf("fixture missing count %s", name)
		}
	}
}

func itemObjectsOpen(t *testing.T, aliases bool, edits ...func(*sav.DocumentData)) *FrontEnd {
	t.Helper()
	f, _ := openCurrentItemFixtureApp(t, aliases, edits...)
	return f
}

func openCurrentItemFixtureApp(t *testing.T, aliases bool, edits ...func(*sav.DocumentData)) (*FrontEnd, *ui.App) {
	t.Helper()
	f := cellStateFront(t)
	doc := sackObjectsLiteral1115(t, f)
	actorIndex := 0
	for i, r := range doc.Objects {
		if r.Class == "Unit" && actorProjectionValue(t, r, "Identity") == newGroupA {
			actorIndex = i
		}
	}
	appendRecord := func(r sav.DocumentRecordData) uint16 {
		doc.Objects = append(doc.Objects, r)
		return uint16(len(doc.Objects))
	}
	var ground []uint16
	if aliases {
		effect := appendRecord(literalSavedEffectRecord(0x510001))
		spell := appendRecord(sav.DocumentRecordData{Class: "Spell", Values: []sav.DocumentValueData{{Name: "S08", Value: 7}, {Name: "S09", Value: 11}, {Name: "S0A", Value: 1}, {Name: "S0C", Value: 13}, {Name: "This", Value: 0x610001}}})
		ground = append(ground, appendRecord(literalItemRecord1115("Weapon", 0x0101, 1, 0x410001, []uint16{effect, effect}, spell)))
		ground = append(ground, appendRecord(literalItemRecord1115("Weapon", 0x0102, 1, 0x410002, []uint16{effect}, spell)))
	} else {
		ground = append(ground, appendRecord(literalItemRecord1115("Item", 0x0e06, 2, 0x410001, nil, 0)))
		ground = append(ground, appendRecord(literalItemRecord1115("Item", 0x0e06, 3, 0x410002, nil, 0)))
	}
	p1 := appendRecord(literalItemRecord1115("Item", 0x0e06, 4, 0x420001, nil, 0))
	p2 := appendRecord(literalItemRecord1115("Item", 0x0e06, 5, 0x420002, nil, 0))
	worn := appendRecord(literalItemRecord1115("Weapon", 0x0101, 1, 0x430001, nil, 0))
	literalSavedObjectRefs(t, &doc.Objects[actorIndex], "Inventory", []uint16{p1, p2}, true)
	literalSavedObjectRefs(t, &doc.Objects[actorIndex], "HeldWeapon", []uint16{worn}, false)
	literalSavedObjectRefs(t, &doc.Objects[doc.World.Sacks[0]-1], "Contents", ground, true)
	for _, edit := range edits {
		edit(&doc)
	}
	var err error
	doc, _, err = sav.ReindexDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("source item graph import", town, err)
	}
	app := f.App("source Item identity")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	clear(raw)
	return f, app
}

func itemObjectByKey(t *testing.T, r *sim.SavedObjects, key uint32) sim.SavedItemObject {
	t.Helper()
	for _, row := range r.Items {
		if row.Token.Identity == key {
			locations := r.Locations(row.ID)
			if len(locations) > 1 {
				t.Fatal("exclusive-owner fixture has multiple roots")
			}
			if len(locations) == 1 {
				row.Owner = locations[0].Owner
			} else if row.Retired {
				row.Owner = sim.SavedObjectOwner{Kind: sim.SavedOwnerRetired}
			} else if row.InFlight != 0 {
				row.Owner = sim.SavedObjectOwner{Kind: sim.SavedOwnerInFlight}
			}
			return row
		}
	}
	t.Fatalf("missing literal Item key %x", key)
	return sim.SavedItemObject{}
}

func TestItemObjects1115ExactSourceOrderCountAndWorn(t *testing.T) {
	f := itemObjectsOpen(t, false)
	w := f.live.world
	r := w.SavedObjects()
	state := snapshotCurrentObjects(t, f).SavedDocument
	if state.Objects.Version != 2 || len(r.Items) != 5 || len(r.Sacks) != 1 || len(r.Effects) != 0 || len(r.Spells) != 0 || len(state.Objects.Items) != 5 {
		t.Fatal("complete source ownership missing", r, state.Objects)
	}
	a, b := itemObjectByKey(t, r, 0x420001), itemObjectByKey(t, r, 0x420002)
	if a.ID == b.ID || a.Value.Count != 4 || b.Value.Count != 5 || a.Owner != b.Owner || a.Owner.Kind != sim.SavedOwnerActorPack {
		t.Fatal("equal source values folded into one identity")
	}
	pack, ok := w.CarriedStacks(a.Owner.Entity)
	if !ok || len(pack) != 2 || pack[0].ObjectID != a.ID || pack[1].ObjectID != b.ID || pack[0].Count != 4 || pack[1].Count != 5 {
		t.Fatal("source pack ordinal/count lost", pack)
	}
	worn := itemObjectByKey(t, r, 0x430001)
	equipped, _ := w.EquippedItems(a.Owner.Entity)
	if worn.Owner != (sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: a.Owner.Entity, Slot: 1}) || equipped[0].ObjectID != worn.ID {
		t.Fatal("exact held source owner missing")
	}
	ground := w.Sacks()[0].ItemInstances
	x, y := itemObjectByKey(t, r, 0x410001), itemObjectByKey(t, r, 0x410002)
	if len(ground) != 5 || ground[0].ObjectID != x.ID || ground[1].ObjectID != x.ID || ground[2].ObjectID != y.ID || ground[4].ObjectID != y.ID {
		t.Fatal("source Sack run identities lost", ground)
	}
	if x.F45 != 0xa5 || x.F46 != 0xb6 || x.F47 != 0xc7 || x.F48 != 0xd8e9 || x.Token.T0E != 0x1234 || x.Token.Reference != 0xfedcba98 {
		t.Fatal("non-value source operands were inferred or discarded")
	}
	if err := validateSavedObjectBindingWorld(state, w); err != nil {
		t.Fatal(err)
	}
}

func TestItemObjects1115ChildAliasesSurviveWholePickup(t *testing.T) {
	f := itemObjectsOpen(t, true)
	w := f.live.world
	before := w.SavedObjects()
	a := itemObjectByKey(t, before, 0x410001)
	b := itemObjectByKey(t, before, 0x410002)
	if len(before.Effects) != 1 || len(before.Spells) != 1 || len(a.Effects) != 2 || a.Effects[0] != a.Effects[1] || a.Effects[0] != b.Effects[0] || a.Spell != b.Spell || before.Effects[0].ExternalReferences != 0 || before.Spells[0].ExternalReferences != 0 {
		t.Fatal("archive child aliases were copied or flattened")
	}
	actor := itemObjectByKey(t, before, 0x420001).Owner.Entity
	if err := w.TakeSack(actor, 15, 16); err != nil {
		t.Fatal(err)
	}
	after := snapshotCurrentObjects(t, f)
	r := w.SavedObjects()
	x := itemObjectByKey(t, r, 0x410001)
	y := itemObjectByKey(t, r, 0x410002)
	if x.ID != a.ID || y.ID != b.ID || x.Owner.Kind != sim.SavedOwnerActorPack || y.Owner.Kind != sim.SavedOwnerActorPack || x.Token.T08 != 0 || y.Token.T08 != 0 || !slices.Equal(x.Effects, a.Effects) || x.Spell != a.Spell || len(r.Effects) != 1 || len(r.Spells) != 1 {
		t.Fatal("whole pickup did not retain exact Item/child identities")
	}
	if after.SavedDocument.Objects.Sacks[0].ObjectIndex != 0 || len(after.SavedDocument.Document.World.Sacks) != 0 {
		t.Fatal("Sack was not retired after current destination refs")
	}
	if err := validateSavedObjectBindingWorld(after.SavedDocument, w); err != nil {
		t.Fatalf("%v: native containers=%+v document containers=%+v", err, r.Containers, after.SavedDocument.Objects.Containers)
	}
}

func TestItemObjects1115StackPickupMergeCurrentGraphAndFreshLoad(t *testing.T) {
	f, app := openCurrentItemFixtureApp(t, false)
	w := f.live.world
	before := snapshotCurrentObjects(t, f)
	registry := w.SavedObjects()
	dst := itemObjectByKey(t, registry, 0x420001)
	if err := w.TakeSack(dst.Owner.Entity, 15, 16); err != nil {
		t.Fatal(err)
	}
	after := snapshotCurrentObjects(t, f)
	r := w.SavedObjects()
	retained := itemObjectByKey(t, r, 0x420001)
	other := itemObjectByKey(t, r, 0x420002)
	if retained.ID != dst.ID || retained.Value.Count != 9 || retained.Token.T08 != 0 || other.Value.Count != 5 {
		t.Fatal("literal merge destination/count/flags differ", retained, other)
	}
	if len(r.Items) != 8 || len(after.SavedDocument.Objects.Items) != 8 || len(after.SavedDocument.Document.Objects) != len(before.SavedDocument.Document.Objects)-3 {
		t.Fatalf("split/retired graph: native items=%d bound items=%d doc objects=%d before=%d", len(r.Items), len(after.SavedDocument.Objects.Items), len(after.SavedDocument.Document.Objects), len(before.SavedDocument.Document.Objects))
	}
	for _, key := range []uint32{0x410001, 0x410002} {
		if itemObjectByKey(t, r, key).Owner.Kind != sim.SavedOwnerRetired {
			t.Fatal("incoming source identity remains live")
		}
	}
	fresh, freshApp := currentItemMenuCheckpoint(t, f, app, cellStateFront)
	fresh, _ = currentItemMenuCheckpoint(t, fresh, freshApp, cellStateFront)
	itemMutationSame1115(t, after, snapshotCurrentObjects(t, fresh))
	for range 5 {
		sim.Step(w, nil)
		sim.Step(fresh.live.world, nil)
		if w.Hash() != fresh.live.world.Hash() {
			t.Fatal("native continuation differs")
		}
	}
}

func TestItemObjects1115HostilePairsAndRemapAreAtomic(t *testing.T) {
	f := itemObjectsOpen(t, true)
	before := snapshotCurrentObjects(t, f)
	for _, tc := range []struct {
		name string
		edit func(*SnapshotSAVDocument)
	}{
		{"missing item metadata", func(s *SnapshotSAVDocument) { s.Objects.Items = s.Objects.Items[1:] }},
		{"wrong child class", func(s *SnapshotSAVDocument) { s.Objects.Effects[0].ObjectIndex = s.Objects.Items[0].ObjectIndex }},
		{"unknown child identity", func(s *SnapshotSAVDocument) { s.Objects.Effects[0].ID++ }},
		{"forged retirement", func(s *SnapshotSAVDocument) { s.Objects.Items[0].ObjectIndex = 0 }},
		{"forged constructor coverage", func(s *SnapshotSAVDocument) { s.Objects.Items[0].Unavailable = "unknown=1; " }},
		{"missing container", func(s *SnapshotSAVDocument) { s.Objects.Containers = s.Objects.Containers[1:] }},
		{"owner differs", func(s *SnapshotSAVDocument) { s.Objects.Containers[0].Owner.Object++ }},
		{"source count differs", func(s *SnapshotSAVDocument) {
			newGroupSetValue1115(t, &s.Document.Objects[s.Objects.Items[0].ObjectIndex-1], "F42", 7)
		}},
		{"effect operand differs", func(s *SnapshotSAVDocument) {
			newGroupSetValue1115(t, &s.Document.Objects[s.Objects.Effects[0].ObjectIndex-1], "E40", 18)
		}},
		{"spell identity differs", func(s *SnapshotSAVDocument) {
			newGroupSetValue1115(t, &s.Document.Objects[s.Objects.Spells[0].ObjectIndex-1], "This", 0x610002)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := before
			var err error
			bad.SavedDocument, err = cloneSavedDocument(before.SavedDocument)
			if err != nil {
				t.Fatal(err)
			}
			tc.edit(bad.SavedDocument)
			hash := f.live.world.Hash()
			if _, err := savedDocumentFromSnapshot(bad); err == nil {
				t.Fatal("hostile pair admitted")
			}
			if f.live.world.Hash() != hash || !reflect.DeepEqual(snapshotCurrentObjects(t, f), before) {
				t.Fatal("hostile pair changed active world")
			}
		})
	}
	m := before.SavedDocument.Objects
	copy, err := cloneSavedObjectBindings(m, before.SavedDocument.Document)
	if err != nil {
		t.Fatal(err)
	}
	permutation := make([]uint16, len(before.SavedDocument.Document.Objects)+1)
	for i := range permutation {
		permutation[i] = uint16(i)
	}
	// Late item bounds failure follows valid Sack remaps; no prefix publishes.
	copy.Items[len(copy.Items)-1].ObjectIndex = 65535
	immutable := *copy
	immutable.Sacks = slices.Clone(copy.Sacks)
	immutable.Items = slices.Clone(copy.Items)
	if err := remapSavedObjectBindings(copy, permutation); err == nil || !reflect.DeepEqual(*copy, immutable) {
		t.Fatal("late child remap partially committed")
	}
}

func TestItemObjects1115HumanoidArmorSlotsExcludeHeldRefs(t *testing.T) {
	state := &SnapshotSAVDocument{Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{{Class: "Human"}}}, Actors: []SnapshotSAVActor{{EntityID: 7, ObjectIndex: 1}}}
	registry := &sim.SavedObjects{ItemRoots: []sim.SavedItemRoot{
		{ID: 10, Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: 7, Slot: 1}},
		{ID: 11, Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: 7, Slot: 2}},
		{ID: 12, Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: 7, Slot: 3}},
	}}
	if err := savedCurrentActorItems(state, registry, map[sim.SavedObjectID]uint16{10: 4, 11: 5, 12: 6}); err != nil {
		t.Fatal(err)
	}
	r := &state.Document.Objects[0]
	weapon, _ := savedObjectRefs(r, "HeldWeapon")
	shield, _ := savedObjectRefs(r, "HeldShield")
	armor, _ := savedObjectRefs(r, "Worn")
	if !slices.Equal(weapon, []uint16{4}) || !slices.Equal(shield, []uint16{5}) || !slices.Equal(armor, []uint16{0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0}) {
		t.Fatal("held refs leaked into independent armor slots", weapon, shield, armor)
	}
}

func TestItemObjects1115HeldWeaponU68AliasIsNotAmbiguousOwnership(t *testing.T) {
	f := itemObjectsOpen(t, false, func(doc *sav.DocumentData) {
		for i := range doc.Objects {
			r := &doc.Objects[i]
			if r.Class != "Unit" {
				continue
			}
			held, ok := savedObjectRefs(r, "HeldWeapon")
			if !ok || len(held) != 1 || held[0] == 0 {
				continue
			}
			literalSavedObjectRefs(t, r, "U68", []uint16{held[0]}, false)
		}
	})
	w := f.live.world
	r := w.SavedObjects()
	worn := itemObjectByKey(t, r, 0x430001)
	equipped, ok := w.EquippedItems(worn.Owner.Entity)
	if !ok || equipped[0].ObjectID != worn.ID || worn.Owner.Kind != sim.SavedOwnerActorWorn || worn.Owner.Slot != 1 {
		t.Fatal("a held weapon aliased by its own actor's U68 item-cast slot was refused instead of bound", worn)
	}
	state := snapshotCurrentObjects(t, f).SavedDocument
	if err := validateSavedObjectBindingWorld(state, w); err != nil {
		t.Fatal(err)
	}
}

func TestItemObjects1115LiveSplitProjectsNewChildrenAndUnknowns(t *testing.T) {
	f, app := openCurrentItemFixtureApp(t, true, func(doc *sav.DocumentData) {
		for i := range doc.Objects {
			r := &doc.Objects[i]
			if r.Class == "Weapon" && actorProjectionValue(t, *r, "Identity") == 0x410001 {
				newGroupSetValue1115(t, r, "F42", 2)
			}
		}
	})
	w := f.live.world
	before := w.SavedObjects()
	parent := itemObjectByKey(t, before, 0x410001)
	actor := itemObjectByKey(t, before, 0x420001).Owner.Entity
	if err := w.TakeSack(actor, 15, 16); err != nil {
		t.Fatal(err)
	}
	state := snapshotCurrentObjects(t, f).SavedDocument
	r := w.SavedObjects()
	if len(r.Items) != 6 || len(r.Effects) != 3 || len(r.Spells) != 2 {
		t.Fatal("split did not allocate one Item/two independent Effects/one Spell", r)
	}
	var split sim.SavedItemObject
	for _, item := range r.Items {
		if item.Origin == (sim.SavedObjectOrigin{Kind: sim.SavedObjectSplit, Parent: parent.ID}) {
			split = item
		}
	}
	locations := r.Locations(split.ID)
	if split.ID == 0 || len(locations) != 1 || locations[0].Owner.Kind != sim.SavedOwnerActorPack || split.Value.Count != 1 || split.Effects[0] == split.Effects[1] || split.Effects[0] == parent.Effects[0] || split.Spell == parent.Spell || split.Coverage.Unknown&sim.SavedUnknownF47 == 0 {
		t.Fatal("split identity/alias or explicit constructor Unknown lost", split)
	}
	for _, binding := range state.Objects.Items {
		if binding.ID == split.ID && (binding.ObjectIndex == 0 || binding.Unavailable != "") {
			t.Fatal("live split graph lacks current record or coverage", binding)
		}
	}
	fresh, freshApp := currentItemMenuCheckpoint(t, f, app, cellStateFront)
	fresh, _ = currentItemMenuCheckpoint(t, fresh, freshApp, cellStateFront)
	if !reflect.DeepEqual(r, fresh.live.world.SavedObjects()) {
		t.Fatal("SAV lost split child identities, values or constructor debt")
	}
}

func TestItemObjects1115GeneratedSackKeepsExplicitConstructorDebt(t *testing.T) {
	f, app := openCurrentItemFixtureApp(t, false)
	w := f.live.world
	row := itemObjectByKey(t, w.SavedObjects(), 0x420001)
	var actor sim.Entity
	for _, e := range w.Entities() {
		if e.ID == row.Owner.Entity {
			actor = e
		}
	}
	x, y := actor.X+1, actor.Y
	sim.Step(w, []sim.Command{{Kind: sim.KindDropCarried, Entity: actor.ID, X: x, Y: y, Spell: 0}})
	r := w.SavedObjects()
	if len(r.Sacks) != 2 || r.Sacks[1].Origin.Kind != sim.SavedObjectGenerated || r.Sacks[1].Coverage.Unknown&sim.SavedUnknownToken != sim.SavedUnknownToken {
		t.Fatal("actual drop did not retain a generated Sack identity/Unknown", r)
	}
	after := snapshotCurrentObjects(t, f)
	bindings := after.SavedDocument.Objects.Sacks
	if len(bindings) != 2 || bindings[1].ID != r.Sacks[1].ID || bindings[1].ObjectIndex == 0 || bindings[1].Unavailable != "" || len(after.SavedDocument.Document.World.Sacks) != 2 {
		t.Fatal("generated Sack lost current graph or claimed known constructor", bindings)
	}
	fresh, freshApp := currentItemMenuCheckpoint(t, f, app, cellStateFront)
	fresh, _ = currentItemMenuCheckpoint(t, fresh, freshApp, cellStateFront)
	if !reflect.DeepEqual(r, fresh.live.world.SavedObjects()) {
		t.Fatal("SAV lost generated Sack identities, values or constructor debt")
	}
	if err := w.TakeSack(actor.ID, x, y); err != nil {
		t.Fatal(err)
	}
	if err := fresh.live.world.TakeSack(actor.ID, x, y); err != nil {
		t.Fatal(err)
	}
	itemMutationSame1115(t, snapshotCurrentObjects(t, f), snapshotCurrentObjects(t, fresh))
	final := snapshotCurrentObjects(t, f)
	if final.SavedDocument.Objects.Sacks[1].ObjectIndex != 0 || len(final.SavedDocument.Document.World.Sacks) != 1 {
		t.Fatal("generated Sack retirement retained stale graph")
	}
}
