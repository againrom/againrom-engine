package game

import (
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentBoundHeldItemKeepsAbsentEquipmentThroughOrdinarySAV(t *testing.T) {
	f := itemObjectsOpen(t, false)
	doc, a := currentRootSAVDocument(t, f)
	var id sim.SavedObjectID
	for i := range a.Ownership {
		row := &a.Ownership[i]
		if row.Kind != 1 || row.Object == 0 {
			continue
		}
		key, _ := savedStructureValue(&doc.Objects[row.Object-1], "Identity")
		if key == 0x430001 {
			row.EquipmentKnown, row.DefinitionRow = false, nil
			row.Definition = sim.SourceWeaponDefinition{}
			if err := captureCurrentItemAbsence(&doc, row); err != nil {
				t.Fatal(err)
			}
			id = row.ID
		}
	}
	if id == 0 {
		t.Fatal("fixture has no bound held Weapon")
	}
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, raw); err != nil {
		t.Fatal(err)
	}
	f = loadCurrentRootSAV(t, doc)
	for cycle := 0; cycle < 2; cycle++ {
		registry := f.live.world.SavedObjects()
		item, ok := registry.Item(id)
		if !ok || item.Value.SourceEquipment != (sim.SourceEquipment{}) {
			t.Fatal("native fixture does not have absent equipment fields")
		}
		before := f.live.world.Hash()
		doc, a = currentRootSAVDocument(t, f)
		var wire uint16
		for _, row := range a.Ownership {
			if row.Kind == 1 && row.ID == id {
				wire = row.Object
				if row.EquipmentKnown {
					t.Fatal("ordinary subclass invented native equipment presence")
				}
			}
		}
		if wire == 0 || doc.Objects[wire-1].Class != "Weapon" || before != f.live.world.Hash() {
			t.Fatal("bound held Item lacks an ordinary Weapon or SAVE changed native state")
		}
		f = loadCurrentRootSAV(t, doc)
		if !reflect.DeepEqual(registry, f.live.world.SavedObjects()) {
			t.Fatal("ordinary held subclass changed the current registry")
		}
	}
}

func TestCurrentItemIdentityAbsenceKeepsOrdinaryValues(t *testing.T) {
	f := itemObjectsOpen(t, false)
	ms := f.live.mission.state
	g, rows, err := captureCurrentObjects(ms.savedDocument, ms.World)
	if err != nil {
		t.Fatal(err)
	}
	var chosen sim.SavedObjectID
	var count uint32
	for i := range rows {
		row := &rows[i]
		locations := ms.World.SavedObjects().Locations(row.ID)
		if row.Kind != 1 || len(locations) != 1 || locations[0].Owner.Kind != sim.SavedOwnerActorPack || row.Object == 0 {
			continue
		}
		no := false
		row.IdentityPresent = &no
		identity, err := savedStructureValue(&ms.savedDocument.Document.Objects[row.Object-1], "Identity")
		if err != nil {
			t.Fatal(err)
		}
		row.IdentityAnchor = &identity
		chosen = row.ID
		count, _ = savedStructureValue(&ms.savedDocument.Document.Objects[row.Object-1], "F42")
		break
	}
	if chosen == 0 {
		t.Fatal("no reachable item")
	}
	if _, err := restoreCurrentObjects(ms, g, rows, f.Table); err != nil {
		t.Fatal(err)
	}
	for _, v := range ms.World.SavedObjects().Items {
		if v.ID == chosen && (v.Token.Identity != 0 || v.Value.Count != count) {
			t.Fatal("constructor absence replaced ordinary stack or acquired an identity", v)
		}
		if v.ID != chosen && !v.Retired && v.Token.Identity == 0 {
			t.Fatal("absence policy leaked to another ordinary item")
		}
	}
	_, recaptured, err := captureCurrentObjects(ms.savedDocument, ms.World)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range recaptured {
		if row.ID == chosen && (row.IdentityPresent == nil || *row.IdentityPresent) {
			t.Fatal("second SAVE lost identity absence")
		}
	}
}

func currentItemRoundTrip(t *testing.T, f *FrontEnd) *FrontEnd {
	t.Helper()
	s := snapshotCurrentObjects(t, f).SavedDocument
	before := f.live.world.Hash()
	if err := projectItems(s, f.live.world, f.Table); err != nil {
		t.Fatal(err)
	}
	doc, _, err := sav.ReindexDocumentData(*s.Document)
	if err != nil {
		t.Fatal(err)
	}
	doc, err = sav.RemintDocumentKeys(doc)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil || before != f.live.world.Hash() {
		t.Fatal("projection mutated current state", err)
	}
	cold := cellStateFront(t)
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("current Item graph LOAD", town, err)
	}
	if err := cold.App("current Item SAV").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return cold
}

func TestCurrentItemGraphWideStackKeepsCountsAndNextTransfer(t *testing.T) {
	f := itemObjectsOpen(t, false, func(doc *sav.DocumentData) {
		for i := range doc.Objects {
			r := &doc.Objects[i]
			if !savedItemClass(r.Class) {
				continue
			}
			newGroupSetValue1115(t, r, "F4A", 0)
			switch actorProjectionValue(t, *r, "Identity") {
			case 0x420001:
				newGroupSetValue1115(t, r, "F42", 65535)
			case 0x410001:
				newGroupSetValue1115(t, r, "F42", 1)
			case 0x410002:
				newGroupSetValue1115(t, r, "F40", 0x0e07)
				newGroupSetValue1115(t, r, "F42", 1)
			}
		}
	})
	item := itemObjectByKey(t, f.live.world.SavedObjects(), 0x420001)
	locations := f.live.world.SavedObjects().Locations(item.ID)
	if len(locations) != 1 || locations[0].Owner.Kind != sim.SavedOwnerActorPack {
		t.Fatal("wide fixture lacks one exact pack root", locations)
	}
	actor := locations[0].Owner.Entity
	if err := f.live.world.TakeSack(actor, 15, 16); err != nil {
		t.Fatal(err)
	}
	if got := itemObjectByKey(t, f.live.world.SavedObjects(), 0x420001); got.Value.Count != 65536 {
		t.Fatal("pickup did not create a valid wide native stack", got)
	}
	cold := currentCursorRoundTrip(t, f)
	var owner sim.Entity
	for _, a := range cold.live.world.Entities() {
		if a.ID == actor {
			owner = a
		}
	}
	pack, _ := cold.live.world.CarriedStacks(actor)
	slot, matches := -1, 0
	for i, value := range pack {
		if value.ObjectID == item.ID {
			slot, matches = i, matches+1
			if value.Count != 65536 || value.Code != item.Value.Code || value.Price != item.Value.Price {
				t.Fatal("wide current stack lost its count or values", value)
			}
		}
	}
	if slot < 0 || matches != 1 {
		t.Fatal("wide stack identity or pack position changed", pack)
	}
	cold.live.pending = append(cold.live.pending, sim.DropCarried(owner.ID, sim.ItemSlot(slot), sim.CellPoint{X: owner.X, Y: owner.Y}))
	cold.live.tick()
	f.live.pending = append(f.live.pending, sim.DropCarried(owner.ID, sim.ItemSlot(slot), sim.CellPoint{X: owner.X, Y: owner.Y}))
	f.live.tick()
	if f.live.world.Hash() != cold.live.world.Hash() {
		t.Fatal("next wide drop changed across SAV")
	}
	if len(cold.live.world.Sacks()) != 1 {
		t.Fatal("next drop failed")
	}
	dropped, _ := currentRootSAVDocument(t, cold)
	second := currentCursorCold(t, dropped, cold)
	if cold.live.world.Hash() != second.live.world.Hash() {
		currentMenuWorldDiagnostics(t, cold.live.world, second.live.world)
		t.Fatal("current Sack SAVE changed exact World state")
	}
	second = currentCursorRoundTrip(t, second)
	for _, row := range second.live.world.SavedObjects().Sacks {
		if !row.Retired && row.Token.Identity != 0 {
			t.Fatal("new Sack acquired an opaque native key")
		}
	}
	if err := second.live.world.TakeSack(owner.ID, owner.X, owner.Y); err != nil {
		t.Fatal(err)
	}
	if err := cold.live.world.TakeSack(owner.ID, owner.X, owner.Y); err != nil {
		t.Fatal(err)
	}
	if cold.live.world.Hash() != second.live.world.Hash() {
		currentMenuWorldDiagnostics(t, cold.live.world, second.live.world)
		t.Fatal("next pickup differs after current Sack LOAD")
	}
	if len(second.live.world.Sacks()) != 0 || len(second.live.world.SavedObjects().SackRoots) != 0 {
		t.Fatal("pickup did not retire the Sack root")
	}
	second = currentCursorRoundTrip(t, second)
	second = currentCursorRoundTrip(t, second)
	for range 20 {
		cold.live.tick()
		second.live.tick()
		if cold.live.world.Hash() != second.live.world.Hash() {
			t.Fatal("successor tick differs after Sack retirement")
		}
	}
	for _, target := range []sim.CellPoint{{X: owner.X + 1, Y: owner.Y}, {X: owner.X, Y: owner.Y}} {
		cold.live.pending = append(cold.live.pending, sim.MoveTo(owner.ID, target))
		second.live.pending = append(second.live.pending, sim.MoveTo(owner.ID, target))
		reached := false
		for range 64 {
			cold.live.tick()
			second.live.tick()
			if cold.live.world.Hash() != second.live.world.Hash() {
				currentMenuWorldDiagnostics(t, cold.live.world, second.live.world)
				t.Fatal("crossing the retired Sack cell differs after LOAD")
			}
			for _, actor := range second.live.world.Entities() {
				if actor.ID == owner.ID && actor.X == target.X && actor.Y == target.Y {
					reached = true
				}
			}
			if reached {
				break
			}
		}
		if !reached {
			t.Fatal("real move did not cross the Sack cell", target)
		}
	}
	second = currentCursorRoundTrip(t, second)
	var total uint64
	for _, a := range second.live.world.Entities() {
		pack, _ := second.live.world.CarriedStacks(a.ID)
		for _, item := range pack {
			if item.Code == 0x0e06 {
				total += uint64(item.Count)
			}
		}
	}
	if total != 65541 {
		t.Fatal("wide total lost after cold transfer and second SAV", total)
	}
}

func currentDroppedSackFront(t *testing.T) (*FrontEnd, sim.SavedObjectID) {
	t.Helper()
	f := itemObjectsOpen(t, false)
	item := itemObjectByKey(t, f.live.world.SavedObjects(), 0x420001)
	locations := f.live.world.SavedObjects().Locations(item.ID)
	if len(locations) != 1 || locations[0].Owner.Kind != sim.SavedOwnerActorPack {
		t.Fatal("fixture lacks an exact pack root")
	}
	id := locations[0].Owner.Entity
	if err := f.live.world.TakeSack(id, 15, 16); err != nil {
		t.Fatal(err)
	}
	f = currentCursorRoundTrip(t, f)
	var actor sim.Entity
	for _, row := range f.live.world.Entities() {
		if row.ID == id {
			actor = row
		}
	}
	f.live.pending = append(f.live.pending, sim.DropCarried(id, 0, sim.CellPoint{X: actor.X, Y: actor.Y}))
	f.live.tick()
	sacks := f.live.world.Sacks()
	if len(sacks) != 1 || sacks[0].ObjectID == 0 {
		t.Fatal("real Drop did not create a bound native Sack")
	}
	return f, sacks[0].ObjectID
}

func TestCurrentSackOrdinaryOperandsRemainAuthority(t *testing.T) {
	for _, editIdentity := range []bool{false, true} {
		t.Run(map[bool]string{false: "absent_key", true: "edited_key"}[editIdentity], func(t *testing.T) {
			f, id := currentDroppedSackFront(t)
			doc, a := currentRootSAVDocument(t, f)
			var index uint16
			for _, row := range a.Ownership {
				if row.Kind == 4 && row.ID == id {
					index = row.Object
				}
			}
			if index == 0 {
				t.Fatal("new Sack lacks an exact ordinary binding")
			}
			record := &doc.Objects[index-1]
			oldKey, _ := savedStructureValue(record, "Identity")
			for name, value := range map[string]uint32{"RuntimeID": 0x765432, "T0C": 9, "T0E": 0x3456, "T08": 0xaabbcc, "T18": 0xab12, "T1C": 901, "Reference": 0x76543210, "S3C": 81, "Contents1C": ^uint32(0), "Contents20": ^uint32(0) - 18} {
				savedObjectSetValue(record, name, value)
			}
			position, err := savedObjectRaw(record, "Block12", 12)
			if err != nil {
				t.Fatal(err)
			}
			position[0], position[1], position[4], position[5], position[6], position[7] = 13, 14, 37, 211, 5, 6
			if editIdentity {
				savedObjectSetValue(record, "Identity", 0x765401)
				for i := range doc.World.Cells {
					if doc.World.Cells[i].Sack == oldKey {
						doc.World.Cells[i].Sack = 0x765401
					}
				}
			}
			doc.World.Sacks = append(doc.World.Sacks, index)
			want, gold, container, err := savedSackRecord(record)
			if err != nil {
				t.Fatal(err)
			}
			if !editIdentity {
				want.Identity = 0
			}
			f = currentCursorCold(t, doc, f)
			for range 2 {
				registry := f.live.world.SavedObjects()
				var got sim.SavedSackObject
				for _, row := range registry.Sacks {
					if row.ID == id {
						got = row
					}
				}
				c, ok := savedSackContainer(registry, id)
				if got.Token != want || got.Gold != gold || !ok || c.InsertIndex != container.InsertIndex || c.Accumulator != container.Accumulator || len(c.Items) != 1 || !reflect.DeepEqual(registry.SackRoots, []sim.SavedObjectID{id, id}) {
					t.Fatal("ordinary Sack edit lost current operands", got, c)
				}
				f = currentCursorRoundTrip(t, f)
			}
		})
	}
}

func TestCurrentSackBindingsMalformedLoadIsAtomic(t *testing.T) {
	for _, mode := range []string{"duplicate", "wrong_class", "missing_owner", "mixed_record", "empty_anchor", "motion_duplicate", "motion_anchor", "motion_cost_conflict", "motion_sack_conflict"} {
		t.Run(mode, func(t *testing.T) {
			f, id := currentDroppedSackFront(t)
			doc, a := currentRootSAVDocument(t, f)
			var row *currentOwnedObject
			for i := range a.Ownership {
				if a.Ownership[i].Kind == 4 && a.Ownership[i].ID == id {
					row = &a.Ownership[i]
				}
			}
			if row == nil || row.Object == 0 {
				t.Fatal("new Sack binding missing")
			}
			switch mode {
			case "duplicate":
				a.Ownership = append(a.Ownership, *row)
			case "wrong_class":
				row.Object = a.Bindings[0].Object
			case "missing_owner":
				for i := range a.Inventory.Containers {
					if a.Inventory.Containers[i].Owner.Object == id {
						a.Inventory.Containers[i].Owner.Object = a.Inventory.NextID
					}
				}
			case "mixed_record":
				row.Sack = &sim.SavedSackObject{ID: id}
			case "empty_anchor":
				zero := uint32(0)
				row.IdentityAnchor = &zero
			case "motion_duplicate":
				a.AbsentMotionCells = append(a.AbsentMotionCells, a.AbsentMotionCells[0])
			case "motion_anchor":
				a.AbsentMotionCells[0].Anchor = [32]byte{}
			case "motion_cost_conflict":
				a.CellCosts = append(a.CellCosts, sim.CurrentCellCost{Cell: a.AbsentMotionCells[0].Cell})
			case "motion_sack_conflict":
				a.AbsentCellSacks = append(a.AbsentCellSacks, currentAbsentCellSack{Cell: a.AbsentMotionCells[0].Cell, Wire: *row.IdentityAnchor, Carriers: sim.CurrentSackKeyMotion})
			}
			raw, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, raw); err != nil {
				t.Fatal(err)
			}
			raw, err = sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			before, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := f.RestoreOriginal(raw); err == nil {
				t.Fatal("malformed new Sack binding was accepted")
			}
			after, _, err := f.Snapshot(true)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("malformed Sack LOAD changed current state", err)
			}
		})
	}
}

func TestCurrentSackMotionAbsenceKeepsOrdinaryCellEdits(t *testing.T) {
	for _, mode := range []string{"unchanged", "payload", "duplicate", "new_cell", "block"} {
		t.Run(mode, func(t *testing.T) {
			f, id := currentDroppedSackFront(t)
			doc, a := currentRootSAVDocument(t, f)
			if len(a.AbsentMotionCells) != 1 {
				t.Fatal("new Sack lacks its one absent motion-cell anchor", a.AbsentMotionCells)
			}
			key := a.AbsentMotionCells[0].Cell
			var cell sav.DocumentCellData
			for i := range doc.World.Cells {
				if doc.World.Cells[i].Cell != key {
					continue
				}
				if mode == "payload" {
					doc.World.Cells[i].Residue03 = 137
				}
				cell = doc.World.Cells[i]
			}
			if mode == "duplicate" {
				doc.World.Cells = append(doc.World.Cells, cell)
			}
			if mode == "new_cell" {
				doc.World.Cells = append(doc.World.Cells, sav.DocumentCellData{Cell: 0x0b0b, Cost: 73, Static: 4, Residue03: 21})
			}
			if mode == "block" {
				for i := range doc.World.Blocks {
					if doc.World.Blocks[i].Cell == key {
						doc.World.Blocks[i].Dyn, doc.World.Blocks[i].Static = 80, 16
					}
				}
			}
			cold := currentCursorCold(t, doc, f)
			if mode == "unchanged" && cold.live.world.Hash() != f.live.world.Hash() {
				currentMenuWorldDiagnostics(t, f.live.world, cold.live.world)
				t.Fatal("unchanged Sack acquired a native cell carrier")
			}
			for range 2 {
				_, cells, _, _ := cold.live.world.SavedActorMotions()
				present, added := false, false
				for _, row := range cells {
					if row.Cell == key {
						present = true
						if row.Payload != cell.Payload() {
							t.Fatal("ordinary cell payload lost", row)
						}
					}
					if row.Cell == 0x0b0b {
						added = row.Payload[0] == 73 && row.Payload[1] == 4 && row.Payload[3] == 21
					}
				}
				if present != (mode == "payload" || mode == "duplicate") || added != (mode == "new_cell") {
					t.Fatal("ordinary cell presence edit was suppressed", present, added)
				}
				for _, row := range cold.live.world.SavedObjects().Sacks {
					if row.ID == id && row.Token.Identity != 0 {
						t.Fatal("cell edit fabricated native Sack identity")
					}
				}
				if mode == "block" {
					planes, _ := cold.live.world.SavedCellPlanes()
					if planes.Dynamic[key] != 80 || planes.Static[key] != 16 {
						t.Fatal("ordinary Block edit was replaced", planes.Dynamic[key], planes.Static[key])
					}
				}
				cold = currentCursorRoundTrip(t, cold)
			}
		})
	}
}

func TestCurrentItemGraphConstructsUnboundSlots(t *testing.T) {
	code := uint16(0x777)
	f := itemObjectsOpen(t, false)
	actor := itemObjectByKey(t, f.live.world.SavedObjects(), 0x420001).Owner.Entity
	castSpellWitness(t, f, sim.ScriptInstant{Op: sim.ScriptInstantAddItem, Unit: actor, HasUnit: true, Item: code, HasItem: true}, func() bool { return true })
	pack, _ := f.live.world.CarriedStacks(actor)
	if len(pack) != 3 || pack[0].Code != code || pack[0].ObjectID != 0 {
		t.Fatal("unbound slot acquired a forged source binding")
	}
	cold := currentItemRoundTrip(t, f)
	found := false
	for _, a := range cold.live.world.Entities() {
		pack, _ := cold.live.world.CarriedStacks(a.ID)
		if len(pack) == 3 && pack[0].Code == code {
			found = pack[0].Count == 1 && pack[1].Count == 4 && pack[2].Count == 5 && pack[0].ObjectID != 0
		}
	}
	if !found {
		t.Fatal("unbound slot lost order, values or count")
	}
	currentItemRoundTrip(t, cold)
}

func TestCurrentItemRecordPreservesValueAndConcreteClass(t *testing.T) {
	for _, tc := range []struct {
		name  string
		item  sim.ItemInstance
		held  bool
		class string
	}{
		{"held weapon with known weight", sim.ItemInstance{Code: 0x810d, WeightPresent: true, Weight: 19, Price: 731, Kind: 0, Effects: []sim.ItemEffect{{Kind: 41, Operand: 0x70002}}}, true, "Weapon"},
		{"explicit pack BaseItem", sim.ItemInstance{Code: 0x810d, WeightPresent: true, Weight: 19, Price: 731, Kind: 0}, false, "Item"},
		{"plain potion", sim.PlainItem(0x0d0d), false, "Item"},
		{"plain scroll", sim.PlainItem(0x0e0d), false, "Item"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := currentItemRecordValue(tc.item, tc.held, nil, nil)
			if err := value.SourceEquipment.Validate(); err != nil {
				t.Fatal(err)
			}
			if value.Code != tc.item.Code || value.Price != tc.item.Price || value.Kind != tc.item.Kind || !reflect.DeepEqual(value.Effects, tc.item.Effects) || tc.item.WeightPresent && value.Weight != tc.item.Weight {
				t.Fatal("current known item values were replaced by constructor operands")
			}
			b := generatedDocumentBuilder{nextKey: 1}
			index, err := b.item(value, 7, 0)
			if err != nil {
				t.Fatal(err)
			}
			if b.doc.Objects[index-1].Class != tc.class {
				t.Fatal("wrong ordinary item class", b.doc.Objects[index-1].Class)
			}
		})
	}
}

func TestCurrentItemGraphPreservesSharedChildren(t *testing.T) {
	f := itemObjectsOpen(t, true)
	cold := currentItemRoundTrip(t, f)
	r := cold.live.world.SavedObjects()
	var weapons []sim.SavedItemObject
	for _, row := range r.Items {
		locations := r.Locations(row.ID)
		if len(locations) == 1 && locations[0].Owner.Kind == sim.SavedOwnerSack {
			weapons = append(weapons, row)
		}
	}
	if len(weapons) != 2 || !reflect.DeepEqual(weapons[0].Effects, []sim.SavedObjectID{weapons[1].Effects[0], weapons[1].Effects[0]}) || weapons[0].Spell != weapons[1].Spell {
		t.Fatal("shared child relation lost", weapons)
	}
}

func TestCurrentItemSplitPreservesIndependentChildren(t *testing.T) {
	f := itemObjectsOpen(t, true, func(doc *sav.DocumentData) {
		for i := range doc.Objects {
			r := &doc.Objects[i]
			if r.Class == "Weapon" && actorProjectionValue(t, *r, "Identity") == 0x410001 {
				newGroupSetValue1115(t, r, "F42", 2)
			}
		}
	})
	owner := itemObjectByKey(t, f.live.world.SavedObjects(), 0x420001).Owner.Entity
	if err := f.live.world.TakeSack(owner, 15, 16); err != nil {
		t.Fatal(err)
	}
	cold := currentItemRoundTrip(t, f)
	var split sim.SavedItemObject
	for _, item := range cold.live.world.SavedObjects().Items {
		if len(item.Effects) == 2 && item.Effects[0] != item.Effects[1] {
			split = item
		}
	}
	if split.ID == 0 || split.Spell == 0 || split.Value.Count != 1 {
		t.Fatal("split child identities lost", split)
	}
	currentItemRoundTrip(t, cold)
}
