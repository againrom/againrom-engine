package game

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestObservedNativeItemAliasesKeepExactRecordAndOperands(t *testing.T) {
	item := sim.PlainItem(0x0201)
	item.WeightPresent, item.Weight = true, 7
	item.NativeRecord = &sim.NativeItemRecord{Token: sim.SavedObjectToken{Identity: 0x12345678, RuntimeID: 19, Reference: 23}, F45: 17, F48: 0xabcd}
	b := generatedDocumentBuilder{}
	first, err := b.item(item, 1, 23)
	if err != nil {
		t.Fatal(err)
	}
	alias, err := b.item(item.Clone(), 1, 23)
	if err != nil || alias != first || len(b.doc.Objects) != 1 {
		t.Fatal("exact observed alias reminted its physical record", alias, first, err)
	}
	for _, mutation := range []func(*sim.ItemInstance){
		func(v *sim.ItemInstance) { v.Code++ },
		func(v *sim.ItemInstance) { v.Price++ },
		func(v *sim.ItemInstance) { v.NativeRecord.F48++ },
	} {
		conflict := item.Clone()
		mutation(&conflict)
		if _, err := b.item(conflict, 1, 23); err == nil || len(b.doc.Objects) != 1 {
			t.Fatal("same identity with different current operands accepted")
		}
	}
	if _, err := b.item(item, 2, 23); err == nil {
		t.Fatal("different count accepted as the same observed alias")
	}
	plain := item.Clone()
	plain.NativeRecord = nil
	a, err := b.item(plain, 1, 23)
	if err != nil {
		t.Fatal(err)
	}
	z, err := b.item(plain, 1, 23)
	if err != nil || a == z {
		t.Fatal("equal unobserved items acquired an inferred alias", err)
	}
}

type actorItemModeScale string

func (actorItemModeScale) Len() int               { return 1 }
func (s actorItemModeScale) EntryName(int) string { return string(s) }
func (actorItemModeScale) EntryDoubles(int) []float64 {
	return []float64{1, 1, 1, 1, 1, 1, 1, 1, 1}
}

// OpenMission, a real Equip command, SAVE and cold LOAD establish these
// modes. No test edits a producer's presence flags or supplies expected DTOs.
func nativeActorNativeItemFixture(t *testing.T) ([]byte, *FrontEnd, *Mission, sim.EntityID) {
	t.Helper()
	f, _ := nativeSubjectFixture(t)
	defs := eqDefsTable(t)
	f.Table.Shapes, f.Table.Materials = actorItemModeScale("Common"), actorItemModeScale("Iron")
	f.Table.Weapons, f.Table.Shields, f.Table.Armors = defs.Weapons, defs.Shields, defs.Armors
	worn, plain := sim.PlainItem(eqShieldCode), sim.PlainItem(eqShieldCode)
	worn.Price, plain.Price = 17, 29
	plain.Effects = []sim.ItemEffect{{Kind: 12, Operand: 3}, {Kind: 44, Operand: 0x0907}}
	typed := mapload.SourceConstructedItem(sim.PlainItem(eqMaceCode), f.Table)
	typed.Price = 41
	typed.SourceEquipment.Spell = sim.SourceItemSpell{Present: true, ID: 7, Range: 11, Defensive: 1, ManaCost: 13}
	if typed.SourceEquipment.Class != sim.SourceWeapon || !typed.WeightPresent {
		t.Fatal("fixture lacks actual typed constructor")
	}
	e := f.live.world.Entities()[0]
	id := e.ID
	load := sim.ActorLoadSnapshot{Inventory: sim.ActorLoad{Present: true, ContainerPresent: true},
		Load: e.Load, Capacity: e.Capacity, Speed: e.Speed, HealthHundredths: e.HealthHundredths, ManaHundredths: e.ManaHundredths}
	if !f.live.world.ReplaceStock(sim.Stock{ID: id, ItemInstances: []sim.ItemInstance{worn, plain, typed}, LoadState: &load}) {
		t.Fatal("actual native stock constructor refused")
	}
	sim.Step(f.live.world, []sim.Command{sim.Equip(id, 0, 2)})
	pack, ok := f.live.world.CarriedStacks(id)
	equipment, equipped := f.live.world.EquippedItems(id)
	if !ok || !equipped || len(pack) != 2 || equipment[1].Code != eqShieldCode || equipment[1].ObjectID != 0 ||
		pack[0].Code != eqShieldCode || pack[0].ObjectID != 0 || pack[0].WeightPresent || pack[0].SourceEquipment.Class != 0 ||
		pack[1].Code != eqMaceCode || pack[1].ObjectID != 0 || !pack[1].WeightPresent || pack[1].SourceEquipment.Class != sim.SourceWeapon {
		t.Fatal("real producer did not establish unregistered/absent and known carrier modes", pack, equipment)
	}
	raw, _, _ := saveCurrentEffect(t, f)
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
	if err != nil {
		t.Fatal(err)
	}
	return raw, f, ms, id
}

// This packet owns item rows. Other comparator families remain failures in
// the full instrument and retain the controls from the preceding packet.
func nativeActorOnlyItemDifferences(all []string) []string {
	return slices.DeleteFunc(slices.Clone(all), func(s string) bool {
		return !strings.Contains(s, "item") && !strings.Contains(s, "worn slot") &&
			!strings.Contains(s, "pack length") && !strings.Contains(s, "pack absent") && !strings.Contains(s, "equipment absent")
	})
}

func nativeActorItemCheck(t *testing.T, raw []byte, ms *Mission) []string {
	t.Helper()
	return nativeActorOnlyItemDifferences(nativeActorModeCheck(t, raw, ms, ms.World.Entities()))
}

func nativeActorNativeItemRecord(t *testing.T, doc *sav.DocumentData, id sim.EntityID, slot int) *sav.DocumentRecordData {
	t.Helper()
	a, err := readCurrentActions(doc)
	if err != nil || a == nil {
		t.Fatal("input correspondence", err)
	}
	for _, b := range a.Bindings {
		if b.ID != id || b.Structure || b.Missing {
			continue
		}
		refs, present := savedObjectRefs(&doc.Objects[b.Object-1], "Inventory")
		if !present || slot >= len(refs) || refs[slot] == 0 {
			t.Fatal("exact raw pack ordinal unavailable")
		}
		return &doc.Objects[refs[slot]-1]
	}
	t.Fatal("exact input actor unavailable")
	return nil
}

func nativeActorNativeItemRawRecord(t *testing.T, raw []byte, id sim.EntityID, slot int) (*sackByteReader, *sackByteRecord) {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	roots, reader, _, err := readActorRoots(f, raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range roots {
		if a.current == nil || a.current.ID != id {
			continue
		}
		refs := a.refs["Inventory"]
		if slot >= len(refs) || reader.source.rows[refs[slot]] == nil {
			t.Fatal("independent raw slot unavailable")
		}
		return reader, reader.source.rows[refs[slot]]
	}
	t.Fatal("independent raw actor subject unavailable")
	return nil, nil
}

func nativeActorNativeItemRawValue(t *testing.T, raw []byte, id sim.EntityID, slot int) sim.ItemStack {
	t.Helper()
	reader, row := nativeActorNativeItemRawRecord(t, raw, id, slot)
	return reader.source.item(row, 0)
}

func nativeActorItemTokenNextSAVEControl(t *testing.T, edited, next []byte, id, nextID sim.EntityID, slot int, name string) {
	t.Helper()
	width := map[string]uint32{
		"Identity": 4, "RuntimeID": 4, "Reference": 4,
		"T0C": 1, "T0E": 2, "T08": 4, "T18": 2,
		"F45": 1, "F46": 1, "F47": 1, "F48": 2,
	}[name]
	if name != "Block12" && width == 0 {
		return
	}
	_, before := nativeActorNativeItemRawRecord(t, edited, id, slot)
	_, after := nativeActorNativeItemRawRecord(t, next, nextID, slot)
	if name == "Block12" {
		if len(before.raw[name]) != 12 || len(after.raw[name]) != 12 || !bytes.Equal(before.raw[name], after.raw[name]) {
			t.Fatal("next SAVE lost edited Position bytes or unchanged bytes of Block12", before.raw[name], after.raw[name])
		}
		return
	}
	want, sourcePresent := before.values[name]
	got, outputPresent := after.values[name]
	mask := ^uint32(0)
	if width < 4 {
		mask = (uint32(1) << (8 * width)) - 1
	}
	if !sourcePresent || !outputPresent || want&^mask != 0 || got&^mask != 0 || got != want {
		t.Fatal("next SAVE lost exact edited ordinary token/service word", name, width, got, want)
	}
}

func TestActorRootsNativeItemPresenceAndIdentityModes(t *testing.T) {
	raw, _, ms, id := nativeActorNativeItemFixture(t)
	before := ms.World.Hash()
	if differences := nativeActorItemCheck(t, raw, ms); len(differences) != 0 {
		t.Fatal("native item modes did not retain actual fields and exact holder edges", differences)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil || actions == nil {
		t.Fatal("exact holder-edge input unavailable", err)
	}
	changed := false
	for _, b := range actions.Bindings {
		if b.ID != id || b.Structure || b.Missing {
			continue
		}
		actor := &doc.Objects[b.Object-1]
		refs, ok := savedObjectRefs(actor, "Inventory")
		if !ok || len(refs) != 2 {
			t.Fatal("exact source holder edges unavailable")
		}
		refs = slices.Clone(refs)
		refs[0], refs[1] = refs[1], refs[0]
		savedObjectSetRefs(actor, "Inventory", refs, true)
		changed = true
	}
	if !changed {
		t.Fatal("raw holder permutation was not applied")
	}
	doc, _, err = sav.ReindexDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	swapped, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(nativeActorItemCheck(t, swapped, ms)) == 0 {
		t.Fatal("accepted raw holder ordinal/class permutation against stale World")
	}
	sim.Step(ms.World, []sim.Command{sim.Unequip(id, 2)})
	if len(nativeActorItemCheck(t, raw, ms)) == 0 {
		t.Fatal("accepted an actual holder move against the old raw edges")
	}
	if ms.World.Hash() == before {
		t.Fatal("real Unequip did not change current holder")
	}
}

func TestActorRootsNativeItemOrdinaryByteLossControls(t *testing.T) {
	raw, source, ms, id := nativeActorNativeItemFixture(t)
	for _, name := range []string{"F40", "F42", "F44", "T1C", "F4A", "S50", "W52", "W6A", "W50", "S09", "S0A", "S0C", "E3C", "E3D", "E40", "Identity", "RuntimeID", "Reference", "Block12", "T0C", "T0E", "T08", "T18", "F45", "F46", "F47", "F48"} {
		t.Run(name, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			leaf, _, _ := sav.NativeActions(doc.State)
			slot := 0
			if strings.HasPrefix(name, "W") || strings.HasPrefix(name, "S0") {
				slot = 1
			}
			r := nativeActorNativeItemRecord(t, &doc, id, slot)
			switch {
			case name == "S50" || name == "W52" || name == "W6A" || name == "Block12":
				size := map[string]int{"S50": 22, "W52": 24, "W6A": 22, "Block12": 12}[name]
				b, err := savedObjectRaw(r, name, size)
				if err != nil {
					t.Fatal(err)
				}
				b[0] ^= 1
			case strings.HasPrefix(name, "S0"):
				refs, ok := savedObjectRefs(r, "WeaponSpell")
				if !ok || len(refs) != 1 || refs[0] == 0 {
					t.Fatal("exact owned Spell unavailable")
				}
				child := &doc.Objects[refs[0]-1]
				old, err := savedStructureValue(child, name)
				if err != nil {
					t.Fatal(err)
				}
				mustSetValue(child, name, old+1)
			case strings.HasPrefix(name, "E"):
				refs, ok := savedObjectRefs(r, "Effects")
				if !ok || len(refs) != 2 {
					t.Fatal("exact ordered Effect child unavailable")
				}
				child := &doc.Objects[refs[0]-1]
				old, err := savedStructureValue(child, name)
				if err != nil {
					t.Fatal(err)
				}
				mustSetValue(child, name, old+1)
			default:
				old, err := savedStructureValue(r, name)
				if err != nil {
					t.Fatal(err)
				}
				changed := old + 1
				if name == "F40" {
					changed = old ^ 0x1000
				}
				mustSetValue(r, name, changed)
			}
			back, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := sav.DecodeDocumentData(back)
			if err != nil {
				t.Fatal(err)
			}
			unchanged, _, _ := sav.NativeActions(decoded.State)
			if !bytes.Equal(leaf, unchanged) {
				t.Fatal("ordinary edit changed native mode authority")
			}
			if len(nativeActorItemCheck(t, back, ms)) == 0 {
				t.Fatal("accepted ordinary raw edit against stale World/retained carrier", name)
			}
			cold, _, err := ResumeOriginalSave(source.Archives.Containers, back, source.Table, source.Difficulty, nil, source.Bodies)
			if err != nil {
				t.Fatal(err)
			}
			if differences := nativeActorItemCheck(t, back, cold); len(differences) != 0 {
				t.Fatal("cold LOAD lost an independently edited raw field or exact mode", name, differences)
			}
			front := openCurrentEffectSave(t, source, back)
			nextRaw, _, _ := saveCurrentEffect(t, front)
			nextID := id
			if !sim.StackStateEqual(nativeActorNativeItemRawValue(t, back, id, slot), nativeActorNativeItemRawValue(t, nextRaw, nextID, slot)) {
				if name == "F4A" {
					t.Logf("raw before=%+v after=%+v", nativeActorNativeItemRawValue(t, back, id, slot), nativeActorNativeItemRawValue(t, nextRaw, nextID, slot))
				}
				t.Fatal("next SAVE lost ordinary known item/ordered child values", name)
			}
			nativeActorItemTokenNextSAVEControl(t, back, nextRaw, id, nextID, slot, name)
		})
	}
}

func TestActorRootsItemExactTypedBindingControls(t *testing.T) {
	f := currentSharedItemFront(t)
	raw, _, _ := saveCurrentEffect(t, f)
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
	if err != nil {
		t.Fatal(err)
	}
	if differences := nativeActorItemCheck(t, raw, ms); len(differences) != 0 {
		t.Fatal("bound/aliased actual item baseline", differences)
	}
	original := ms.savedDocument.Objects
	if original == nil || len(original.Items) < 2 {
		t.Fatal("typed binding controls lack actual items")
	}
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	roots, _, origins, err := readActorRoots(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	target := uint16(0)
	for _, a := range roots {
		for _, ref := range a.refs["Inventory"] {
			if ref != 0 {
				target = origins[ref]
				break
			}
		}
		if target != 0 {
			break
		}
	}
	at := slices.IndexFunc(original.Items, func(b SnapshotSAVObjectBinding) bool { return b.ObjectIndex == target })
	if at < 0 || len(original.Effects) == 0 || len(original.Spells) == 0 {
		t.Fatal("exact live alias fixture unavailable")
	}
	other := (at + 1) % len(original.Items)
	for _, name := range []string{"missing", "wrong ID", "wrong object", "duplicate", "Effect binding", "Spell binding"} {
		t.Run(name, func(t *testing.T) {
			copy := *original
			copy.Items = slices.Clone(original.Items)
			switch name {
			case "missing":
				copy.Items = slices.Delete(copy.Items, at, at+1)
			case "wrong ID":
				copy.Items[at].ID = copy.Items[other].ID
			case "wrong object":
				copy.Items[at].ObjectIndex = copy.Items[other].ObjectIndex
			case "duplicate":
				copy.Items = append(copy.Items, copy.Items[at])
			case "Effect binding":
				copy.Effects = slices.Clone(original.Effects)
				copy.Effects[0].ObjectIndex = 0
			case "Spell binding":
				copy.Spells = slices.Clone(original.Spells)
				copy.Spells[0].ObjectIndex = 0
			}
			ms.savedDocument.Objects = &copy
			defer func() { ms.savedDocument.Objects = original }()
			if len(nativeActorItemCheck(t, raw, ms)) == 0 {
				t.Fatal("accepted conflicting/absent current typed binding", name)
			}
		})
	}
}

// A retained token is lawful Document fallback for an ID0 holder. Its next
// SAVE must preserve the changed key; reconstructing the item is real loss.
func TestActorRootsUnregisteredItemEditedIdentitySurvivesNextSAVE(t *testing.T) {
	raw, source, ms, id := nativeActorNativeItemFixture(t)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	r := nativeActorNativeItemRecord(t, &doc, id, 0)
	old, err := savedStructureValue(r, "Identity")
	if err != nil {
		t.Fatal(err)
	}
	want := old ^ 0x01000000
	mustSetValue(r, "Identity", want)
	edited, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(nativeActorItemCheck(t, edited, ms)) == 0 {
		t.Fatal("accepted stale retained token identity")
	}
	cold := openCurrentEffectSave(t, source, edited)
	_, next, _ := saveCurrentEffect(t, cold)
	got, err := savedStructureValue(nativeActorNativeItemRecord(t, &next, id, 0), "Identity")
	if err != nil || got != want {
		t.Fatal("known unregistered item token identity lost on next ordinary SAVE", got, want, err)
	}
}

func TestNativeItemRecordFollowsActualEquipAcrossColdSAV(t *testing.T) {
	raw, source, _, id := nativeActorNativeItemFixture(t)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	r := nativeActorNativeItemRecord(t, &doc, id, 0)
	mustSetValue(r, "F45", 73)
	mustSetValue(r, "F47", 91)
	mustSetValue(r, "F48", 0x9876)
	position, err := savedObjectRaw(r, "Block12", 12)
	if err != nil {
		t.Fatal(err)
	}
	position[0], position[11] = 0xa5, 0x5a
	edited, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	front := openCurrentEffectSave(t, source, edited)
	pack, _ := front.live.world.CarriedStacks(id)
	if len(pack) != 2 || pack[0].NativeRecord == nil {
		t.Fatal("ordinary edit lacks current unregistered history")
	}
	want := *pack[0].NativeRecord
	sim.Step(front.live.world, []sim.Command{sim.Equip(id, 0, 2)})
	worn, _ := front.live.world.EquippedItems(id)
	if worn[1].NativeRecord == nil || *worn[1].NativeRecord != want {
		t.Fatal("real Equip lost current token history")
	}
	for cycle := 0; cycle < 2; cycle++ {
		before := front.live.world.Hash()
		next, _, _ := saveCurrentEffect(t, front)
		front = openCurrentEffectSave(t, source, next)
		if front.live.world.Hash() != before {
			t.Fatal("current Item history changed cold World", cycle)
		}
		worn, _ = front.live.world.EquippedItems(id)
		if worn[1].NativeRecord == nil || *worn[1].NativeRecord != want {
			t.Fatal("cold SAV lost full token/service history", cycle)
		}
		sim.Step(front.live.world, nil)
	}
	sim.Step(front.live.world, []sim.Command{sim.Unequip(id, 2)})
	pack, _ = front.live.world.CarriedStacks(id)
	found := false
	for _, v := range pack {
		if v.NativeRecord != nil && *v.NativeRecord == want {
			found = true
		}
		if v.ObjectID != 0 {
			if registry := front.live.world.SavedObjects(); registry != nil {
				if row, ok := registry.Item(v.ObjectID); ok && row.Token.T1C == uint32(v.Price) {
					token := row.Token
					token.T1C = 0
					if token == want.Token && row.F45 == want.F45 && row.F46 == want.F46 && row.F47 == want.F47 && row.F48 == want.F48 {
						found = true
					}
				}
			}
		}
	}
	if !found {
		t.Fatal("real Unequip lost current item history")
	}
	before := front.live.world.Hash()
	next, _, _ := saveCurrentEffect(t, front)
	again := openCurrentEffectSave(t, source, next)
	if again.live.world.Hash() != before {
		t.Fatal("registered acquisition after real Unequip lost cold current state")
	}
}

func nativeActorItemLeafEdit(t *testing.T, raw []byte, edit func(map[string]any)) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	leaf, present, err := sav.NativeActions(doc.State)
	if err != nil || !present {
		t.Fatal("fixture native item modes unavailable", err)
	}
	var input map[string]any
	decoder := json.NewDecoder(bytes.NewReader(leaf))
	decoder.UseNumber()
	if err := decoder.Decode(&input); err != nil {
		t.Fatal(err)
	}
	edit(input)
	leaf, err = json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
	back, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return back
}

func nativeActorLegacyItemLeaf(t *testing.T, raw []byte, version *uint32) []byte {
	t.Helper()
	return nativeActorItemLeafEdit(t, raw, func(input map[string]any) {
		inventory := input["Inventory"].(map[string]any)
		delete(inventory, "Version")
		if version != nil {
			inventory["Version"] = *version
		}
		delete(inventory, "ItemRoots")
		delete(inventory, "BookActors")
		items := 0
		for _, value := range input["Ownership"].([]any) {
			row := value.(map[string]any)
			if row["Kind"] != json.Number("1") {
				continue
			}
			if row["ID"] != json.Number("0") || row["Object"] == json.Number("0") {
				t.Fatal("legacy fixture lacks an ordinary ID0 item")
			}
			items++
			delete(row, "NativeRecordKnown")
			delete(row, "NativeRecordAnchor")
			delete(row, "NativeRecordAnchorVersion")
		}
		if items != 3 {
			t.Fatal("legacy fixture ordinary item population", items)
		}
	})
}

func TestActorRootsLegacyNativeItemIdentityModes(t *testing.T) {
	raw, source, _, id := nativeActorNativeItemFixture(t)
	zero, one := uint32(0), uint32(1)
	for _, test := range []struct {
		name    string
		version *uint32
	}{{"omitted", nil}, {"version0", &zero}, {"version1", &one}} {
		t.Run(test.name, func(t *testing.T) {
			legacy := nativeActorLegacyItemLeaf(t, raw, test.version)
			cold, _, err := ResumeOriginalSave(source.Archives.Containers, legacy, source.Table, source.Difficulty, nil, source.Bodies)
			if err != nil {
				t.Fatal(err)
			}
			pack, ok := cold.World.CarriedStacks(id)
			worn, equipped := cold.World.EquippedItems(id)
			if !ok || !equipped || len(pack) != 2 || pack[0].ObjectID != 0 || pack[1].ObjectID != 0 || worn[1].ObjectID != 0 ||
				pack[0].NativeRecord == nil || pack[1].NativeRecord == nil || worn[1].NativeRecord == nil {
				t.Fatal("legacy ordinary ID0/history semantics lost", pack, worn)
			}
			if cold.savedDocument.Objects != nil && len(cold.savedDocument.Objects.Items) != 0 {
				t.Fatal("legacy ID0 acquired typed item bindings")
			}
			before := cold.World.Hash()
			if differences := nativeActorItemCheck(t, legacy, cold); len(differences) != 0 {
				t.Fatal("legacy current items differ from independent ordinary bytes", differences)
			}
			if cold.World.Hash() != before {
				t.Fatal("legacy item comparison changed World")
			}
		})
	}
}

func TestActorRootsLegacyNativeItemModeCorruptionControls(t *testing.T) {
	raw, _, _, _ := nativeActorNativeItemFixture(t)
	legacy := nativeActorLegacyItemLeaf(t, raw, nil)
	for _, name := range []string{"future version", "missing inventory", "legacy ItemRoots", "legacy BookActors", "duplicate object", "duplicate ID", "retired subject", "record anchor version", "presence anchor"} {
		t.Run(name, func(t *testing.T) {
			changed := nativeActorItemLeafEdit(t, legacy, func(input map[string]any) {
				inventory := input["Inventory"].(map[string]any)
				rows := input["Ownership"].([]any)
				var items []map[string]any
				for _, value := range rows {
					row := value.(map[string]any)
					if row["Kind"] == json.Number("1") {
						items = append(items, row)
					}
				}
				if len(items) != 3 {
					t.Fatal("corruption fixture item population", len(items))
				}
				switch name {
				case "future version":
					inventory["Version"] = uint32(3)
				case "missing inventory":
					delete(input, "Inventory")
				case "legacy ItemRoots":
					inventory["ItemRoots"] = []any{map[string]any{"ID": uint32(1)}}
				case "legacy BookActors":
					inventory["Version"] = uint32(1)
					inventory["BookActors"] = []any{uint32(1)}
				case "duplicate object":
					input["Ownership"] = append(rows, items[0])
				case "duplicate ID":
					items[0]["ID"], items[1]["ID"] = uint32(123), uint32(123)
				case "retired subject":
					items[0]["Retired"] = true
				case "record anchor version":
					items[0]["NativeRecordAnchorVersion"] = uint8(2)
				case "presence anchor":
					items[0]["WeightKnown"], items[0]["WeightAnchor"] = true, [32]byte{1}
				}
			})
			file, err := sav.Open(changed)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := sav.DecodeDocumentData(changed)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := nativeActorReadItemModes(&doc, file); err == nil {
				t.Fatal("accepted corrupt legacy identity mode", name)
			}
		})
	}
}

func TestActorRootsLegacyNativeItemOrdinaryLossControls(t *testing.T) {
	raw, source, _, id := nativeActorNativeItemFixture(t)
	legacy := nativeActorLegacyItemLeaf(t, raw, nil)
	cold, _, err := ResumeOriginalSave(source.Archives.Containers, legacy, source.Table, source.Difficulty, nil, source.Bodies)
	if err != nil {
		t.Fatal(err)
	}
	if differences := nativeActorItemCheck(t, legacy, cold); len(differences) != 0 {
		t.Fatal("legacy loss-control baseline", differences)
	}
	for _, name := range []string{"T1C", "Identity", "F48", "E0C", "E40", "S0C"} {
		t.Run(name, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(legacy)
			if err != nil {
				t.Fatal(err)
			}
			leaf, _, _ := sav.NativeActions(doc.State)
			slot := 0
			if name == "S0C" {
				slot = 1
			}
			r := nativeActorNativeItemRecord(t, &doc, id, slot)
			if name == "E0C" || name == "E40" || name == "S0C" {
				field := "Effects"
				if name == "S0C" {
					field = "WeaponSpell"
				}
				refs, present := savedObjectRefs(r, field)
				if !present || len(refs) == 0 || refs[0] == 0 {
					t.Fatal("legacy raw child unavailable", name)
				}
				r = &doc.Objects[refs[0]-1]
			}
			old, err := savedStructureValue(r, name)
			if err != nil {
				t.Fatal(err)
			}
			mustSetValue(r, name, old+1)
			changed, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := sav.DecodeDocumentData(changed)
			if err != nil {
				t.Fatal(err)
			}
			unchanged, _, _ := sav.NativeActions(decoded.State)
			if !bytes.Equal(leaf, unchanged) {
				t.Fatal("ordinary legacy edit changed native mode authority")
			}
			if len(nativeActorItemCheck(t, changed, cold)) == 0 {
				t.Fatal("accepted legacy ordinary item/child loss against stale World", name)
			}
			reloaded, _, err := ResumeOriginalSave(source.Archives.Containers, changed, source.Table, source.Difficulty, nil, source.Bodies)
			if name == "E0C" {
				if err == nil || !strings.Contains(err.Error(), "Effect state") {
					t.Fatal("invalid owned Effect lifetime was not refused", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if differences := nativeActorItemCheck(t, changed, reloaded); len(differences) != 0 {
				t.Fatal("legacy LOAD lost changed raw item/child field", name, differences)
			}
		})
	}
}
