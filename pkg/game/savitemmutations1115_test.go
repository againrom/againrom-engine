package game

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const itemMutationWeapon = uint32(0x730001)
const itemMutationWeapon1115 = itemMutationWeapon

// The source archive and the database are independent synthetic inputs.
// ITEM-SPELLMOVE-132 reads the first kind41 operand's low byte, then Spells
// parameters6/18/1. The old filled Spell deliberately disagrees with all four
// new values. Runtime pointer allocation remains unknown, not a copied This.
func itemMutationFront1115(t *testing.T, defensive int32) *FrontEnd {
	t.Helper()
	return itemMutationFront(t, defensive)
}

func itemMutationFront(t *testing.T, defensive int32) *FrontEnd {
	t.Helper()
	f := cellStateFront(t)
	f.SetDeterministicFrames(true)
	weapon := make([]int32, 16)
	weapon[5], weapon[12], weapon[13], weapon[14], weapon[15] = 2, 13, 17, 1, 1
	f.Table.Weapons = dbCollection{{}, {name: "literal Weapon row", params: weapon}}
	rows := dbCollection{{}}
	for i := 1; i <= 3; i++ {
		p := make([]int32, 19)
		p[1], p[6], p[18] = 3, 5, 0
		if i == 2 {
			p[1], p[6], p[18] = 0x12345, 0x127, defensive
		}
		rows = append(rows, dbEntry{name: fmt.Sprintf("literal Spell %d", i), params: p})
	}
	f.Table.Spells = rows
	return f
}

func itemMutationOpen1115(t *testing.T, defensive int32, shared bool) (*FrontEnd, *ui.App) {
	t.Helper()
	return itemMutationOpen(t, defensive, shared)
}

func itemMutationOpen(t *testing.T, defensive int32, shared bool) (*FrontEnd, *ui.App) {
	t.Helper()
	f := itemMutationFront1115(t, defensive)
	doc := sackObjectsLiteral1115(t, f)
	appendRecord := func(r sav.DocumentRecordData) uint16 {
		doc.Objects = append(doc.Objects, r)
		return uint16(len(doc.Objects))
	}
	var effects []uint16
	for i, operand := range []uint32{0x7abc0002, 3} {
		r := literalSavedEffectRecord(0x740001 + uint32(i))
		newGroupSetValue1115(t, &r, "E3C", 41)
		newGroupSetValue1115(t, &r, "E40", operand)
		effects = append(effects, appendRecord(r))
	}
	spell := appendRecord(sav.DocumentRecordData{Class: "Spell", Values: []sav.DocumentValueData{
		{Name: "S08", Value: 7}, {Name: "S09", Value: 11}, {Name: "S0A"},
		{Name: "S0C", Value: 13}, {Name: "This", Value: 0x750001},
	}})
	weapon := literalItemRecord1115("Weapon", 0x0101, 1, itemMutationWeapon1115, effects, spell)
	newGroupSetValue1115(t, &weapon, "T0C", 1)
	newGroupSetValue1115(t, &weapon, "F44", 2)
	weaponIndex := appendRecord(weapon)
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class == "Unit" && actorProjectionValue(t, *r, "Identity") == newGroupA {
			literalSavedObjectRefs(t, r, "HeldWeapon", []uint16{weaponIndex}, false)
		}
	}
	if shared {
		other := appendRecord(literalItemRecord1115("Weapon", 0x0102, 1, 0x730002, nil, spell))
		literalSavedObjectRefs(t, &doc.Objects[doc.World.Sacks[0]-1], "Contents", []uint16{other}, true)
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
		t.Fatal("literal equipped Weapon original LOAD", town, err)
	}
	app := f.App("synthetic owned Spell mutation")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f, app
}

func itemMutationSame1115(t *testing.T, want, got Snapshot) {
	t.Helper()
	// The retained input Document is transport provenance, not the live item
	// graph. Current SAV rebuilds ordinary actor roots and adds continuation
	// data. The registry/World and each current frontend value stay exact.
	want.SavedDocument, got.SavedDocument = nil, nil
	var world sim.World
	if err := world.UnmarshalBinary(want.World); err != nil {
		t.Fatal(err)
	}
	want.CurrentRoster = maps.Clone(want.CurrentRoster)
	for _, e := range world.Entities() {
		member, ok := want.CurrentRoster[e.ID]
		if ok && member.Book.State == sim.BookLegacy && !member.SpellbookRestored && e.Book.State != sim.BookLegacy {
			member.Book, member.SpellbookPresent, member.SpellbookRestored = e.Book, e.Book.HasInstances(), true
			want.CurrentRoster[e.ID] = member
		}
	}
	if reflect.DeepEqual(want, got) {
		return
	}
	currentItemWorldDiagnostics(t, want.World, got.World)
	a, b := reflect.ValueOf(want), reflect.ValueOf(got)
	for i := 0; i < a.NumField(); i++ {
		if !a.Field(i).CanInterface() {
			continue
		}
		if !reflect.DeepEqual(a.Field(i).Interface(), b.Field(i).Interface()) {
			t.Log("changed Snapshot field", a.Type().Field(i).Name)
			if a.Type().Field(i).Name == "ApplicationState" {
				currentItemFieldDiagnostics(t, "Application", a.Field(i), b.Field(i))
			}
			if a.Type().Field(i).Name == "CurrentRoster" {
				for id, member := range want.CurrentRoster {
					currentItemFieldDiagnostics(t, fmt.Sprintf("roster[%d]", id), reflect.ValueOf(member), reflect.ValueOf(got.CurrentRoster[id]))
				}
			}
			if a.Type().Field(i).Name == "Residue" {
				x, y := a.Field(i), b.Field(i)
				for j := 0; j < x.NumField(); j++ {
					if x.Field(j).CanInterface() && !reflect.DeepEqual(x.Field(j).Interface(), y.Field(j).Interface()) {
						t.Log("changed residue", x.Type().Field(j).Name, x.Field(j).Interface(), y.Field(j).Interface())
					}
				}
			}
		}
	}
	t.Fatal("current World/document/metadata or session residue changed")
}

func itemMutationCheckpoint(t *testing.T, f *FrontEnd, app *ui.App, defensive int32) (*FrontEnd, *ui.App) {
	t.Helper()
	before := snapshotCurrentObjects(t, f)
	if _, err := f.ExportCurrentSave(before, "current item continuation"); err != nil {
		t.Fatal("current item SAV projection", err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	name := crossingMenuSave(t, app, store)
	if !reflect.DeepEqual(before, snapshotCurrentObjects(t, f)) {
		t.Fatal("SAVE changed the detached live snapshot")
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sav.DecodeDocumentData(raw); err != nil {
		t.Fatal("menu did not write ordinary SAV", err)
	}
	fresh := itemMutationFront1115(t, defensive)
	freshApp := fresh.App("fresh synthetic owned Spell LOAD")
	save, list, load = fresh.SaveSeams(store, OriginalStore{}, nil)
	freshApp.SetSaveSeams(save, list, load)
	groundAppLoad(t, freshApp, list, localOriginalSaveToken(name))
	itemMutationSame1115(t, before, snapshotCurrentObjects(t, fresh))
	if f.live.world.Hash() != fresh.live.world.Hash() {
		t.Fatal("fresh native world hash differs")
	}
	return fresh, freshApp
}

func itemMutationSpell1115(t *testing.T, r *sim.SavedObjects, id sim.SavedObjectID) sim.SavedSpellObject {
	t.Helper()
	for _, s := range r.Spells {
		if s.ID == id {
			return s
		}
	}
	t.Fatal("missing native Spell identity", id)
	return sim.SavedSpellObject{}
}

func itemMutationBinding1115(t *testing.T, bindings []SnapshotSAVObjectBinding, id sim.SavedObjectID) SnapshotSAVObjectBinding {
	t.Helper()
	for _, b := range bindings {
		if b.ID == id {
			return b
		}
	}
	t.Fatal("missing exact archive binding", id)
	return SnapshotSAVObjectBinding{}
}

func itemMutationEdge1115(t *testing.T, s Snapshot, item sim.SavedItemObject, oldSpell sim.SavedObjectID, initial Snapshot) {
	t.Helper()
	state := s.SavedDocument
	ib := itemMutationBinding1115(t, state.Objects.Items, item.ID)
	if ib.ObjectIndex == 0 || ib.Unavailable != "" {
		t.Fatal("same original Weapon became absent or uncovered", ib)
	}
	want := itemObjectRecord1115(t, initial, item.ID)
	want.RefSlots = slices.Clone(want.RefSlots)
	spellIndex := uint16(0)
	if item.Spell != 0 {
		sb := itemMutationBinding1115(t, state.Objects.Spells, item.Spell)
		spellIndex = sb.ObjectIndex
		if spellIndex == 0 {
			t.Fatal("new Spell lacks its exact ordinary binding", sb)
		}
		v := item.Value.SourceEquipment.Spell
		wantSpell := sav.DocumentRecordData{Class: "Spell", Values: []sav.DocumentValueData{
			{Name: "S08", Value: uint32(v.ID)}, {Name: "S09", Value: uint32(v.Range)},
			{Name: "S0A", Value: uint32(v.Defensive)}, {Name: "S0C", Value: uint32(v.ManaCost)}, {Name: "This"},
		}}
		if !reflect.DeepEqual(state.Document.Objects[spellIndex-1], wantSpell) {
			t.Fatal("new archive Spell reused the old filled values or invented This")
		}
	}
	literalSavedObjectRefs(t, &want, "WeaponSpell", []uint16{spellIndex}, false)
	// Child archive indices may change at retirement/reindex; native Effect IDs
	// and complete source Effect records are checked independently below.
	var effectIndices []uint16
	for _, id := range item.Effects {
		current := itemMutationBinding1115(t, state.Objects.Effects, id)
		source := itemMutationBinding1115(t, initial.SavedDocument.Objects.Effects, id)
		if current.ObjectIndex == 0 || current.Unavailable != "" || !reflect.DeepEqual(state.Document.Objects[current.ObjectIndex-1], initial.SavedDocument.Document.Objects[source.ObjectIndex-1]) {
			t.Fatal("retained Effect source scalar/raw payload or coverage changed", id)
		}
		effectIndices = append(effectIndices, current.ObjectIndex)
	}
	literalSavedObjectRefs(t, &want, "Effects", effectIndices, true)
	if !reflect.DeepEqual(state.Document.Objects[ib.ObjectIndex-1], want) {
		t.Fatal("Weapon scalar/raw/source payload changed outside its child edge")
	}
	if itemMutationBinding1115(t, state.Objects.Spells, oldSpell).ObjectIndex != 0 {
		t.Fatal("old Spell archive record was not explicitly retired")
	}
	for _, actor := range state.Actors {
		if actor.EntityID != item.Owner.Entity {
			continue
		}
		r := state.Document.Objects[actor.ObjectIndex-1]
		held, _ := savedObjectRefs(&r, "HeldWeapon")
		pack, _ := savedObjectRefs(&r, "Inventory")
		if item.Owner.Kind == sim.SavedOwnerActorWorn {
			if !slices.Equal(held, []uint16{ib.ObjectIndex}) || len(pack) != 0 {
				t.Fatal("current worn Weapon archive owner differs", held, pack)
			}
		} else if !slices.Equal(held, []uint16{0}) || !slices.Equal(pack, []uint16{ib.ObjectIndex}) {
			t.Fatal("current carried Weapon archive owner differs", held, pack)
		}
		return
	}
	t.Fatal("source actor binding disappeared")
}

func TestItemMutations1115OwnedSpellUnequipEquipNativeContinuation(t *testing.T) {
	for _, defensive := range []int32{1, 2} {
		t.Run(fmt.Sprintf("parameter18=%d", defensive), func(t *testing.T) {
			f, app := itemMutationOpen1115(t, defensive, false)
			initial := snapshotCurrentObjects(t, f)
			immutable, err := cloneSavedDocument(initial.SavedDocument)
			if err != nil {
				t.Fatal(err)
			}
			registry := f.live.world.SavedObjects()
			old := itemObjectByKey(t, registry, itemMutationWeapon1115)
			oldSpell := itemMutationSpell1115(t, registry, old.Spell)
			if len(registry.Items) != 1 || len(registry.Effects) != 2 || len(registry.Spells) != 1 || old.Owner.Kind != sim.SavedOwnerActorWorn || old.Value.Count != 1 || old.Value.SourceEquipment.DefinitionRow != 1 || !old.Value.SourceEquipment.Definition.Present || oldSpell.Value != (sim.SourceItemSpell{Present: true, ID: 7, Range: 11, ManaCost: 13}) || oldSpell.This != 0x750001 {
				t.Fatal("literal source setup is not an exclusively owned filled Weapon", old, oldSpell)
			}
			if rule, ok := f.live.world.Spell(2); !ok || rule.MaxRange != 0x27 || rule.ManaCost != 0x12345 || rule.Defensive != (defensive == 1) {
				t.Fatal("independent synthetic Spells row did not reach the world", rule)
			}
			f.live.pending = append(f.live.pending, sim.Command{Kind: sim.KindUnequip, Entity: old.Owner.Entity, X: 1})
			f.live.tick()
			afterUnequip := snapshotCurrentObjects(t, f)
			unequipped := itemObjectByKey(t, f.live.world.SavedObjects(), itemMutationWeapon1115)
			want := old
			want.Owner = sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: old.Owner.Entity}
			want.Spell, want.Value.SourceEquipment.Spell = 0, sim.SourceItemSpell{}
			retired := oldSpell
			retired.Retired = true
			if !reflect.DeepEqual(unequipped, want) || itemMutationSpell1115(t, f.live.world.SavedObjects(), old.Spell) != retired || len(afterUnequip.SavedDocument.Document.Objects) != len(initial.SavedDocument.Document.Objects)-1 {
				t.Fatal("ordinary Unequip changed Item identity or failed to retire exactly the old Spell", unequipped, want)
			}
			itemMutationEdge1115(t, afterUnequip, unequipped, old.Spell, initial)
			fresh, freshApp := itemMutationCheckpoint(t, f, app, defensive)
			for range 20 {
				f.live.tick()
				fresh.live.tick()
				itemMutationSame1115(t, snapshotCurrentObjects(t, f), snapshotCurrentObjects(t, fresh))
			}
			f, app = fresh, freshApp
			preEquip, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			f.live.pending = append(f.live.pending, sim.Command{Kind: sim.KindEquip, Entity: old.Owner.Entity, X: 0, Y: 1})
			f.live.tick()
			afterEquip := snapshotCurrentObjects(t, f)
			current := f.live.world.SavedObjects()
			equipped := itemObjectByKey(t, current, itemMutationWeapon1115)
			newValue := sim.SourceItemSpell{Present: true, ID: 2, Range: 0x27, ManaCost: 0x2345}
			if defensive == 1 {
				newValue.Defensive = 1
			}
			want = old
			want.Spell, want.Value.SourceEquipment.Spell = registry.NextID, newValue
			newSpell := sim.SavedSpellObject{ID: registry.NextID, Origin: sim.SavedObjectOrigin{Kind: sim.SavedObjectGenerated}, Value: newValue, Coverage: sim.SavedObjectCoverage{Unknown: sim.SavedUnknownIdentity}}
			if !reflect.DeepEqual(equipped, want) || equipped.Spell == old.Spell || current.NextID != registry.NextID+1 || len(current.Items) != 1 || len(current.Spells) != 2 || itemMutationSpell1115(t, current, equipped.Spell) != newSpell || itemMutationSpell1115(t, current, old.Spell) != retired || !reflect.DeepEqual(current.Effects, registry.Effects) || len(afterEquip.SavedDocument.Document.Objects) != len(preEquip.SavedDocument.Document.Objects)+1 {
				t.Fatal("ordinary Equip did not retain Item/create a distinct known-rule Spell", equipped, want, current.Spells)
			}
			itemMutationEdge1115(t, afterEquip, equipped, old.Spell, initial)
			fresh, _ = itemMutationCheckpoint(t, f, app, defensive)
			for range 20 {
				f.live.tick()
				fresh.live.tick()
				itemMutationSame1115(t, snapshotCurrentObjects(t, f), snapshotCurrentObjects(t, fresh))
			}
			if !reflect.DeepEqual(initial.SavedDocument, immutable) {
				t.Fatal("later mutations or SAVE changed the detached source snapshot")
			}
		})
	}
}

func TestSharedWeaponSpellRemovalKeepsOtherReference(t *testing.T) {
	f, app := itemMutationOpen1115(t, 1, true)
	registry := f.live.world.SavedObjects()
	item := itemObjectByKey(t, registry, itemMutationWeapon1115)
	other := itemObjectByKey(t, registry, 0x730002)
	if item.Spell == 0 || item.Spell != other.Spell || len(registry.Spells) != 1 {
		t.Fatal("shared source Spell control did not preserve the archive alias")
	}
	if _, ok := f.live.world.UnequipSource(item.Owner.Entity, 1, true); !ok {
		t.Fatal("shared Spell blocked removal of one weapon edge")
	}
	for cycle := 0; cycle < 3; cycle++ {
		current := f.live.world.SavedObjects()
		removed, _ := current.Item(item.ID)
		retained, _ := current.Item(other.ID)
		if removed.Spell != 0 || removed.Value.SourceEquipment.Spell.Present || !current.HasRoot(item.ID, sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: item.Owner.Entity}) || retained.Spell != item.Spell || !reflect.DeepEqual(retained.Value, other.Value) || itemMutationSpell1115(t, current, item.Spell).Retired || !reflect.DeepEqual(current.Effects, registry.Effects) {
			t.Fatal("weapon removal changed another Spell reference or child values", cycle)
		}
		if cycle < 2 {
			f, app = itemMutationCheckpoint(t, f, app, 1)
		}
	}
}
