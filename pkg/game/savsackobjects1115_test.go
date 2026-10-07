package game

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Only the existing synthetic archive grammar is reused. The complete Token
// operands and stale empty-container bookkeeping below are independent inputs.
func sackObjectsLiteral1115(t *testing.T, f *FrontEnd) sav.DocumentData {
	t.Helper()
	doc, err := sav.DecodeDocumentData(crossingCellLiteral(t, f, true))
	if err != nil {
		t.Fatal(err)
	}
	r := &doc.Objects[doc.World.Sacks[0]-1]
	for _, field := range []struct {
		name  string
		value uint32
	}{
		{"RuntimeID", 0xabcdef01}, {"T0C", 0x83}, {"T0E", 0x4321}, {"T08", 0xfedcba98},
		{"T18", 0x9876}, {"T1C", 0x89ab0123}, {"Reference", 0x13572468},
		{"Contents1C", 0xfffeabcd}, {"Contents20", 0xfffffffe},
	} {
		newGroupSetValue1115(t, r, field.name, field.value)
	}
	p := crossingRawField(t, r, "Block12")
	p[6], p[7] = 0x72, 0x81
	for i := range doc.World.Blocks {
		if doc.World.Blocks[i].Cell == 0x100f {
			doc.World.Blocks[i].Static |= 0x20
			doc.World.Blocks[i].Dyn |= 0x20
		}
	}
	for i := range doc.Objects {
		actor := &doc.Objects[i]
		if actor.Class != "Unit" || actorProjectionValue(t, *actor, "Identity") != newGroupA {
			continue
		}
		newGroupSetValue1115(t, actor, "HasInventory", 1)
		actor.Values = append(actor.Values, sav.DocumentValueData{Name: "Inventory1C"}, sav.DocumentValueData{Name: "Inventory20"})
		actor.Counts = append(actor.Counts, sav.DocumentCountData{Name: "Inventory"})
		actor.RefSlots = append(actor.RefSlots, sav.DocumentRefsData{Name: "Inventory"})
		slices.SortFunc(actor.Values, func(a, b sav.DocumentValueData) int { return strings.Compare(a.Name, b.Name) })
		slices.SortFunc(actor.Counts, func(a, b sav.DocumentCountData) int { return strings.Compare(a.Name, b.Name) })
		slices.SortFunc(actor.RefSlots, func(a, b sav.DocumentRefsData) int { return strings.Compare(a.Name, b.Name) })
	}
	return doc
}

func sackObjectsOpen1115(t *testing.T, edit func(*sav.DocumentData)) (*FrontEnd, *ui.App, sav.DocumentData) {
	t.Helper()
	f := cellStateFront(t)
	doc := sackObjectsLiteral1115(t, f)
	if edit != nil {
		edit(&doc)
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("synthetic Sack original import", town, err)
	}
	app := f.App("synthetic exact Sack owner")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	clear(raw)
	return f, app, doc
}

func snapshotCurrentObjects(t *testing.T, f *FrontEnd) Snapshot {
	t.Helper()
	s, _, err := f.Snapshot(true)
	if err != nil || s.SavedDocument == nil || s.SavedDocument.Objects == nil {
		t.Fatal("Sack Snapshot lacks explicit object owner", err)
	}
	return s
}

func TestSackObjects1115ExactGoldOwnerPickupNativeContinuation(t *testing.T) {
	f, app, literal := sackObjectsOpen1115(t, nil)
	initial := snapshotCurrentObjects(t, f)
	metadata := initial.SavedDocument.Objects
	if metadata.Version != 2 || len(metadata.Sacks) != 1 || len(metadata.Unavailable) != 0 || metadata.Sacks[0].ID != 1 || metadata.Sacks[0].ObjectIndex <= 1 || metadata.Sacks[0].Unavailable != "" || len(metadata.Items)+len(metadata.Effects)+len(metadata.Spells) != 0 {
		t.Fatal("exact source Sack did not receive a separate monotonic native identity", metadata)
	}
	wantOwners := []SnapshotSAVContainerBinding{
		{Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerSack, Object: 1}},
		{Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: 0}},
		{Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: 2}},
		{Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: 3}},
	}
	if !reflect.DeepEqual(metadata.Containers, wantOwners) {
		t.Fatal("current original import lost exact Sack/actor container ownership", metadata.Containers)
	}
	registry := f.live.world.SavedObjects()
	wantToken := sim.SavedObjectToken{
		Position:  [12]byte{15, 16, 15, 16, 128, 128, 0x72, 0x81, 1, 12, 30, 122},
		RuntimeID: 0xabcdef01, T0C: 0x83, T0E: 0x4321, T08: 0xfedcba98,
		T18: 0x9876, T1C: 0x89ab0123, Identity: crossingSack1115, Reference: 0x13572468,
	}
	want := &sim.SavedObjects{Version: sim.SavedObjectsVersion, NextID: 2,
		Sacks:     []sim.SavedSackObject{{ID: 1, Origin: sim.SavedObjectOrigin{Kind: sim.SavedObjectOriginal}, Token: wantToken, Gold: 23}},
		SackRoots: []sim.SavedObjectID{1},
		Containers: []sim.SavedObjectContainer{
			{Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerSack, Object: 1}, Present: true, InsertIndex: 0xfffeabcd, Accumulator: -2},
			{Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: 0}, Present: true},
			{Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: 2}},
			{Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: 3}},
		},
	}
	if !reflect.DeepEqual(registry, want) || f.live.world.Sacks()[0].ObjectID != 1 {
		t.Fatalf("source Token or stale Contents state changed:\n got%+v\nwant%+v", registry, want)
	}
	if got := initial.SavedDocument.Document.Objects[metadata.Sacks[0].ObjectIndex-1]; !reflect.DeepEqual(got, literal.Objects[literal.World.Sacks[0]-1]) {
		t.Fatal("initial source Sack was reconstructed or normalized")
	}
	registry.Sacks[0].Gold = 999
	registry.Containers[0].Accumulator = 0
	if !reflect.DeepEqual(f.live.world.SavedObjects(), want) {
		t.Fatal("object getter aliases live owner")
	}
	f.live.tick() // move the actor out; only the Sack now retains cell100f
	e := newGroupActors1115(t, f.live.world)[newGroupA]
	purse := f.live.world.Purse(e.Owner)
	if err := f.live.world.TakeSack(e.ID, 15, 16); err != nil {
		t.Fatal(err)
	}
	if f.live.world.Purse(e.Owner) != purse+23 || len(f.live.world.Sacks()) != 0 || f.live.world.SavedSackCellKey(0x100f) != 0 {
		t.Fatal("actual pickup did not credit/remove the exact source Sack")
	}
	after := snapshotCurrentObjects(t, f)
	if after.SavedDocument.Objects.Sacks[0].ObjectIndex != 0 || len(after.SavedDocument.Document.World.Sacks) != 0 || len(after.SavedDocument.Document.Objects) != len(initial.SavedDocument.Document.Objects)-1 {
		t.Fatal("pickup did not explicitly retire source graph/binding")
	}
	got := f.live.world.SavedObjects()
	if got.NextID != 2 || len(got.Sacks) != 1 || !got.Sacks[0].Retired || got.Sacks[0].Gold != 0 || got.Sacks[0].Token != wantToken || len(got.SackRoots) != 0 || !reflect.DeepEqual(got.Containers, want.Containers[1:]) || !reflect.DeepEqual(after.SavedDocument.Objects.Containers, wantOwners[1:]) {
		t.Fatal("retirement discarded identity/Token or left live root/container", got)
	}
	if err := validateSavedGroupBindingWorld(after.SavedDocument, f.live.world); err != nil {
		t.Fatal("Sack retirement changed exact actor/Group/Player bindings", err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	beforeHash := f.live.world.Hash()
	name := crossingMenuSave(t, app, store)
	if f.live.world.Hash() != beforeHash {
		t.Fatal("ordinary SAVE mutated object owner")
	}
	fresh := cellStateFront(t)
	freshApp := fresh.App("fresh retired Sack native LOAD")
	save, list, load = fresh.SaveSeams(store, OriginalStore{}, nil)
	freshApp.SetSaveSeams(save, list, load)
	groundAppLoad(t, freshApp, list, name)
	reloaded := snapshotCurrentObjects(t, fresh)
	itemMutationSame1115(t, after, reloaded)
	for tick := 0; tick < 20; tick++ {
		f.live.tick()
		fresh.live.tick()
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("Sack native continuation differs", tick)
		}
	}
	if initial.SavedDocument.Objects.Sacks[0].ObjectIndex == 0 || len(initial.SavedDocument.Document.World.Sacks) != 1 {
		t.Fatal("retirement mutated detached earlier Snapshot")
	}
}

func TestSackObjects1115ProductionGrabAndUnboundPour(t *testing.T) {
	t.Run("production grab", func(t *testing.T) {
		f, _, _ := sackObjectsOpen1115(t, nil)
		e := newGroupActors1115(t, f.live.world)[newGroupA]
		purse := f.live.world.Purse(e.Owner)
		f.live.grab(uint32(e.ID), 15, 16, true)
		for tick := 0; tick < 100 && len(f.live.world.Sacks()) != 0; tick++ {
			f.live.tick()
		}
		if len(f.live.world.Sacks()) != 0 || f.live.world.Purse(e.Owner) != purse+23 || !f.live.world.SavedObjects().Sacks[0].Retired {
			t.Fatal("production aimed grab did not retire/credit the exact bound Sack")
		}
		if s := snapshotCurrentObjects(t, f); s.SavedDocument.Objects.Sacks[0].ObjectIndex != 0 || len(s.SavedDocument.Document.World.Sacks) != 0 {
			t.Fatal("production grab Snapshot retained the old source Sack")
		}
	})
	t.Run("native unbound item pour", func(t *testing.T) {
		f, _, _ := sackObjectsOpen1115(t, nil)
		before := snapshotCurrentObjects(t, f)
		e := newGroupActors1115(t, f.live.world)[newGroupA]
		// Restore only the fixture's genuinely unbound native donor. The
		// public transfer admits its ID0 item into the bound actor's pack;
		// bulk restore must not bypass that actor's current ownership guard.
		var donor sim.EntityID
		found := false
		for _, candidate := range f.live.world.Entities() {
			if candidate.SourceBinding.Class == 0 && candidate.Alive() {
				donor, found = candidate.ID, true
				break
			}
		}
		if !found || !f.live.world.ReplaceStock(sim.Stock{ID: donor, Items: []uint16{0x1234}, OrderedStacks: []sim.ItemStack{{Code: 0x1234, Count: 1}}}) {
			t.Fatal("unbound native donor setup failed")
		}
		if err := f.live.world.MoveCarried(donor, e.ID, 0x1234, 1); err != nil {
			t.Fatal("ordinary unbound-item ingress", err)
		}
		if pack, ok := f.live.world.CarriedStacks(e.ID); !ok || len(pack) != 1 || pack[0].ObjectID != 0 || pack[0].Code != 0x1234 {
			t.Fatal("ordinary transfer did not retain the native ID0 item", pack)
		}
		sim.Step(f.live.world, []sim.Command{{Kind: sim.KindDropCarried, Entity: e.ID, X: 15, Y: 16, Spell: 0}})
		sacks := f.live.world.Sacks()
		if len(sacks) != 1 || sacks[0].ObjectID != 1 || !slices.Equal(sacks[0].Items, []uint16{0x1234}) || len(sacks[0].ItemInstances) != 0 {
			t.Fatal("ordinary drop did not pour an unbound item into the exact Sack", sacks)
		}
		after := snapshotCurrentObjects(t, f)
		binding := after.SavedDocument.Objects.Sacks[0]
		if binding.ID != 1 || binding.Unavailable != "" || len(after.SavedDocument.Document.Objects) != len(before.SavedDocument.Document.Objects)+1 {
			t.Fatal("current pour lost its Sack identity or ordinary child", binding)
		}
		refs, ok := savedObjectRefs(&after.SavedDocument.Document.Objects[binding.ObjectIndex-1], "Contents")
		if !ok || len(refs) != 1 || refs[0] == 0 {
			t.Fatal("current Sack omitted the exact unbound Contents position", refs)
		}
		item, err := savedItemRecord(after.SavedDocument.Document, refs[0])
		if err != nil || item.Value.Code != 0x1234 || item.Value.Count != 1 {
			t.Fatal("ordinary Item confused current code and count", item, err)
		}
		_, policies, err := captureCurrentObjects(after.SavedDocument, f.live.world)
		if err != nil {
			t.Fatal(err)
		}
		found = false
		for _, row := range policies {
			if row.Kind == 1 && row.Object == refs[0] {
				found = row.ID == 0 && !row.WeightKnown && !row.EquipmentKnown && row.Item == nil
			}
		}
		if !found {
			t.Error("bound Sack omitted its unbound child presence policy")
		}
		t.Run("ordinary item edits", func(t *testing.T) {
			doc, a := currentRootSAVDocument(t, f)
			var object uint16
			for _, row := range a.Ownership {
				if row.Kind == 4 && row.ID == binding.ID {
					object = row.Object
				}
			}
			if object == 0 {
				t.Fatal("poured Sack lost its ordinary binding")
			}
			contents, ok := savedObjectRefs(&doc.Objects[object-1], "Contents")
			if !ok || len(contents) != 1 {
				t.Fatal("poured Item lost its exact ordinary address")
			}
			_, _, wantContainer, err := savedSackRecord(&doc.Objects[object-1])
			if err != nil {
				t.Fatal(err)
			}
			record := &doc.Objects[contents[0]-1]
			savedObjectSetValue(record, "F40", 0x1235)
			savedObjectSetValue(record, "F42", 3)
			savedObjectSetValue(record, "F4A", 7)
			savedObjectSetValue(record, "T1C", 19)
			edited := currentCursorCold(t, doc, f)
			for range 2 {
				ground := edited.live.world.Sacks()
				if len(ground) != 1 || ground[0].ObjectID != binding.ID || len(ground[0].ItemInstances) != 3 {
					t.Fatal("ordinary count edit lost current Sack units", ground)
				}
				for _, value := range ground[0].ItemInstances {
					if value.ObjectID != 0 || value.Code != 0x1235 || !value.WeightPresent || value.Weight != 7 || value.Price != 19 {
						t.Fatal("native absence policy replaced ordinary Item edits", value)
					}
				}
				container, ok := savedSackContainer(edited.live.world.SavedObjects(), binding.ID)
				if !ok || !slices.Equal(container.Items, []sim.SavedObjectID{0, 0, 0}) || container.InsertIndex != wantContainer.InsertIndex || container.Accumulator != wantContainer.Accumulator {
					t.Fatal("ordinary count edit changed Sack root order or bookkeeping", container)
				}
				edited = currentCursorRoundTrip(t, edited)
			}
		})
		fresh := f
		for cycle := 0; cycle < 2; cycle++ {
			hash := fresh.live.world.Hash()
			doc, _ := currentRootSAVDocument(t, fresh)
			if fresh.live.world.Hash() != hash {
				t.Fatal("current SAV changed the poured Sack")
			}
			fresh = currentCursorCold(t, doc, fresh)
			if fresh.live.world.Hash() != hash || !reflect.DeepEqual(fresh.live.world.Sacks(), sacks) {
				currentMenuWorldDiagnostics(t, f.live.world, fresh.live.world)
				t.Fatal("current SAV lost the unbound item or Sack identity", cycle)
			}
		}
		if err := fresh.live.world.TakeSack(e.ID, 15, 16); err != nil {
			t.Fatal(err)
		}
		if err := f.live.world.TakeSack(e.ID, 15, 16); err != nil {
			t.Fatal(err)
		}
		pack, ok := fresh.live.world.CarriedStacks(e.ID)
		if !ok || len(pack) != 1 || !sim.StackStateEqual(pack[0], sim.ItemStack{Code: 0x1234, Count: 1}) || fresh.live.world.Hash() != f.live.world.Hash() {
			t.Fatal("pickup changed the native plain Item or continuation", pack)
		}
		if s := snapshotCurrentObjects(t, fresh); s.SavedDocument.Objects.Sacks[0].ObjectIndex != 0 || s.SavedDocument.Objects.Sacks[0].Unavailable != "" {
			t.Fatal("pickup of an unbound pour did not retire the current Sack")
		}
		for range 20 {
			f.live.tick()
			fresh.live.tick()
			if f.live.world.Hash() != fresh.live.world.Hash() {
				t.Fatal("poured Sack successor differs after current SAV")
			}
		}
		if refs, _ := savedObjectRefs(&before.SavedDocument.Document.Objects[before.SavedDocument.Objects.Sacks[0].ObjectIndex-1], "Contents"); len(refs) != 0 {
			t.Fatal("pour or retirement changed the detached earlier Snapshot")
		}
	})
}

func TestSackObjects1115HostilePairsRefuseBeforeAdoption(t *testing.T) {
	f, _, _ := sackObjectsOpen1115(t, nil)
	before := snapshotCurrentObjects(t, f)
	for _, tc := range []struct {
		name string
		edit func(*Snapshot)
	}{
		{"nil metadata", func(s *Snapshot) { s.SavedDocument.Objects = nil }},
		{"version", func(s *Snapshot) { s.SavedDocument.Objects.Version = 0 }},
		{"zero ID", func(s *Snapshot) { s.SavedDocument.Objects.Sacks[0].ID = 0 }},
		{"wrong ID", func(s *Snapshot) { s.SavedDocument.Objects.Sacks[0].ID++ }},
		{"duplicate binding", func(s *Snapshot) { m := s.SavedDocument.Objects; m.Sacks = append(m.Sacks, m.Sacks[0]) }},
		{"forged retirement", func(s *Snapshot) { s.SavedDocument.Objects.Sacks[0].ObjectIndex = 0 }},
		{"forged Contents gap", func(s *Snapshot) { s.SavedDocument.Objects.Sacks[0].Unavailable = savedSackContentsUnavailable }},
		{"forged source gap", func(s *Snapshot) {
			m := s.SavedDocument.Objects
			m.Unavailable = []SnapshotSAVObjectCoverage{{ObjectIndex: m.Sacks[0].ObjectIndex, Reason: savedSackItemsUnavailable}}
			m.Sacks = nil
		}},
		{"gold", func(s *Snapshot) {
			r := &s.SavedDocument.Document.Objects[s.SavedDocument.Objects.Sacks[0].ObjectIndex-1]
			newGroupSetValue1115(t, r, "S3C", 24)
		}},
		{"Token scalar", func(s *Snapshot) {
			r := &s.SavedDocument.Document.Objects[s.SavedDocument.Objects.Sacks[0].ObjectIndex-1]
			newGroupSetValue1115(t, r, "T08", 1)
		}},
		{"Token residue", func(s *Snapshot) {
			r := &s.SavedDocument.Document.Objects[s.SavedDocument.Objects.Sacks[0].ObjectIndex-1]
			crossingRawField(t, r, "Block12")[6] ^= 1
		}},
		{"empty cursor", func(s *Snapshot) {
			r := &s.SavedDocument.Document.Objects[s.SavedDocument.Objects.Sacks[0].ObjectIndex-1]
			newGroupSetValue1115(t, r, "Contents1C", 0)
		}},
		{"empty accumulator", func(s *Snapshot) {
			r := &s.SavedDocument.Document.Objects[s.SavedDocument.Objects.Sacks[0].ObjectIndex-1]
			newGroupSetValue1115(t, r, "Contents20", 0)
		}},
		{"added root alias", func(s *Snapshot) {
			d := s.SavedDocument.Document
			d.World.Sacks = append(d.World.Sacks, d.World.Sacks[0])
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := before
			var err error
			bad.SavedDocument, err = cloneSavedDocument(before.SavedDocument)
			if err != nil {
				t.Fatal(err)
			}
			tc.edit(&bad)
			if decoded, _, err := DecodeSave(uncheckedDocumentEnvelope1115(t, bad)); err == nil || !reflect.DeepEqual(decoded, Snapshot{}) {
				t.Fatal("checksum-valid disagreeing Sack pair accepted", err)
			}
			live, hash := f.live, f.live.world.Hash()
			if open, town, err := f.Restore(bad); err == nil || open != nil || town {
				t.Fatal("hostile Sack pair reached frontend adoption", err)
			}
			if f.live != live || f.live.world.Hash() != hash || !reflect.DeepEqual(snapshotCurrentObjects(t, f), before) {
				t.Fatal("rejected Sack pair changed active session")
			}
		})
	}
}

func TestSackObjects1115RemapAndLateRetirementFailureAreAtomic(t *testing.T) {
	bindings := &SnapshotSAVObjectBindings{Version: 1, Sacks: []SnapshotSAVObjectBinding{{ID: 1, ObjectIndex: 2}}, Unavailable: []SnapshotSAVObjectCoverage{{ObjectIndex: 3, Reason: savedSackItemsUnavailable}}}
	before := &SnapshotSAVObjectBindings{Version: 1, Sacks: slices.Clone(bindings.Sacks), Unavailable: slices.Clone(bindings.Unavailable)}
	if err := remapSavedObjectBindings(bindings, []uint16{0, 2, 1, 0}); err == nil || !reflect.DeepEqual(bindings, before) {
		t.Fatal("late uncovered-object remap failure partially published")
	}
	f, _, _ := sackObjectsOpen1115(t, nil)
	source := snapshotCurrentObjects(t, f).SavedDocument
	ms := &Mission{World: f.live.world, savedDocument: source}
	index := source.Document.World.Sacks[0]
	// A second, explicitly uncovered Sack retains an incoming archive edge.
	// Removing only the bound Sack must fail; the graph helper must not GC it.
	sackObjectsAddIncoming1115(t, source.Document, index)
	source.Objects.Unavailable = []SnapshotSAVObjectCoverage{{ObjectIndex: uint16(len(source.Document.Objects)), Reason: savedSackItemsUnavailable}}
	// Use a real pickup, which retires the exact native owner independently of
	// whether a later original-document producer can retire its incoming edges.
	actor := newGroupActors1115(t, ms.World)[newGroupA]
	if err := ms.World.TakeSack(actor.ID, 15, 16); err != nil {
		t.Fatal(err)
	}
	state, err := cloneSavedDocument(ms.savedDocument)
	if err != nil {
		t.Fatal(err)
	}
	immutable, err := cloneSavedDocument(state)
	if err != nil {
		t.Fatal(err)
	}
	hash := ms.World.Hash()
	if err := projectSavedSackObjects(state, ms.World); err == nil || !strings.Contains(err.Error(), "retired Sack") {
		t.Fatal("retirement did not reject the surviving incoming reference", err)
	}
	if !reflect.DeepEqual(state, immutable) || ms.World.Hash() != hash {
		t.Fatal("late retirement failure mutated document, bindings or native owner")
	}
}

func sackObjectsAddIncoming1115(t *testing.T, doc *sav.DocumentData, index uint16) {
	t.Helper()
	r := doc.Objects[index-1]
	r.Values, r.Counts, r.RefSlots = slices.Clone(r.Values), slices.Clone(r.Counts), slices.Clone(r.RefSlots)
	newGroupSetValue1115(t, &r, "Identity", 0x12345678)
	position := bytes.Clone(crossingRawField(t, &r, "Block12"))
	binary.LittleEndian.PutUint16(position[2:], 0x100e)
	r.Raw = slices.Clone(r.Raw)
	for i := range r.Raw {
		if r.Raw[i].Name == "Block12" {
			r.Raw[i].Bytes = position
		}
	}
	for i := range r.Counts {
		if r.Counts[i].Name == "Contents" {
			r.Counts[i].Count = 1
		}
	}
	for i := range r.RefSlots {
		if r.RefSlots[i].Name == "Contents" {
			r.RefSlots[i].Objects = []uint16{index}
		}
	}
	doc.Objects = append(doc.Objects, r)
	doc.World.Sacks = append(doc.World.Sacks, uint16(len(doc.Objects)))
}

func TestSackObjects1115AmbiguousOriginalSacksKeepNativeSave(t *testing.T) {
	f, _, _ := sackObjectsOpen1115(t, func(doc *sav.DocumentData) {
		r := doc.Objects[doc.World.Sacks[0]-1]
		r.Values = slices.Clone(r.Values)
		newGroupSetValue1115(t, &r, "Identity", 0x12345678)
		newGroupSetValue1115(t, &r, "S3C", 31)
		doc.Objects = append(doc.Objects, r)
		doc.World.Sacks = append(doc.World.Sacks, uint16(len(doc.Objects)))
	})
	s := snapshotCurrentObjects(t, f)
	if len(s.SavedDocument.Objects.Sacks) != 0 || len(s.SavedDocument.Objects.Unavailable) != 2 || len(f.live.world.Sacks()) != 1 || f.live.world.Sacks()[0].Gold != 54 || f.live.world.Sacks()[0].ObjectID != 0 {
		t.Fatal("ordinary original LOAD guessed an owner for merged same-cell Sacks")
	}
	for _, row := range s.SavedDocument.Objects.Unavailable {
		if row.Reason != savedSackCellAmbiguous {
			t.Fatal("ambiguous source coverage changed", row)
		}
	}
	// This remains an explicit predecessor-codec input control. Current SAV
	// export of unbound Sacks is covered separately by current-unbound tests.
	s.ghost, s.terrainBase = nil, nil
	raw, err := EncodeSave(s, "explicitly uncovered merged Sacks")
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(raw)
	if err != nil || !reflect.DeepEqual(decoded, s) {
		t.Fatal("uncovered original Sacks broke ordinary native SAVE", err)
	}
	fresh := cellStateFront(t)
	open, town, err := fresh.Restore(decoded)
	if err != nil || town {
		t.Fatal("uncovered Sack native LOAD", town, err)
	}
	if err := fresh.App("uncovered native Sacks").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if fresh.live.world.Hash() != f.live.world.Hash() || !reflect.DeepEqual(snapshotCurrentObjects(t, fresh).SavedDocument, s.SavedDocument) {
		t.Fatal("native LOAD inferred an owner or changed uncovered source records")
	}
}
