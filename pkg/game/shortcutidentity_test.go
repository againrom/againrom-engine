package game

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The key is the item's description: a world's object identity and token are
// not part of it, so a sale that renumbers the pack leaves a bound key waiting
// on the same item.
func TestShortcutItemKeyIgnoresWorldAllocatedFields(t *testing.T) {
	plain := sim.ItemInstance{Code: 0xe1c, Kind: 4}
	moved := plain
	moved.ObjectID = 7
	if itemCastKey(plain) != itemCastKey(moved) {
		t.Fatal("object identity changes the key")
	}
	native := plain
	native.NativeRecord = &sim.NativeItemRecord{Class: 2, F45: 1}
	renumbered := native
	renumbered.ObjectID = 9
	renumbered.NativeRecord = &sim.NativeItemRecord{Class: 2, F45: 1, Token: sim.SavedObjectToken{RuntimeID: 33, Identity: 4}}
	if itemCastKey(native) != itemCastKey(renumbered) {
		t.Fatal("native token changes the key")
	}
	renumbered.NativeRecord.F45 = 2
	if itemCastKey(native) == itemCastKey(renumbered) {
		t.Fatal("a different description shares the key")
	}
	sold := []sim.ItemStack{sim.PlainStack(0xe10, 1), sim.StackItem(moved, 1)}
	after := []sim.ItemStack{sim.StackItem(plain, 1)}
	if inventoryItemKeys(sold)[1] != inventoryItemKeys(after)[0] {
		t.Fatal("a sale in town lost the bound item's key")
	}
}

// A first-game town visit gives the next mission's world fresh object identities.
func TestShortcutItemBindingSurvivesFirstGameTownVisit(t *testing.T) {
	f := currentArchiveFixture(t)
	stacks, _ := f.live.world.CarriedStacks(f.live.mission.ids[0])
	if len(stacks) == 0 {
		t.Fatal("fixture has no item")
	}
	key := inventoryItemKeys(stacks)[0]
	ms := f.live.mission
	if refusal := f.CampaignSession.carryMissionHome(f.townInstall(), 10, ms.party, f.live.world, ms.ids, nil); refusal != "" {
		t.Fatal(refusal)
	}
	if err := f.App("town visit").OpenMission(f.MissionOpener(10)); err != nil || f.live == nil {
		t.Fatal("reopen", err)
	}
	next, _ := f.live.world.CarriedStacks(f.live.mission.ids[0])
	if !slices.Contains(inventoryItemKeys(next), key) {
		t.Fatal("the bound item no longer carries its key", key, inventoryItemKeys(next))
	}
}

// Selecting a companion and then the hero again leaves the hero's pack keys.
func TestShortcutPackKeysSurviveSubjectSwitch(t *testing.T) {
	f, app, screen, _ := secondBagFront(t, secondBagSample())
	secondBagEnter(t, f, app, screen)
	mw := f.live
	if len(mw.mission.ids) < 2 {
		t.Skip("fixture has one managed member")
	}
	hero, other := mw.mission.ids[0], mw.mission.ids[1]
	mw.switchInventorySubject(uint32(hero))
	keys := slices.Clone(mw.invSubject.PackKeys)
	if len(keys) == 0 || keys[0] == "" {
		t.Fatal("hero has no pack keys", keys)
	}
	f.quickSpells[0] = ui.Shortcut{Item: keys[0]}
	mw.switchInventorySubject(uint32(other))
	mw.switchInventorySubject(uint32(hero))
	if !slices.Equal(mw.invSubject.PackKeys, keys) {
		t.Fatal("pack keys lost after a subject switch", mw.invSubject.PackKeys, keys)
	}
	if err := app.HeadlessKey("f4"); err != nil {
		t.Fatal(err)
	}
	if i, ok := mw.view.TakeInventoryEquip(); !ok || i != 0 {
		t.Fatal("the bound key does not fire after the switch", i, ok)
	}
}

// A first-game play whose engine rows all fit the original four cells writes
// the rows the original layout wrote: no layout field, numbered from F5. The
// expected bytes come from a struct built with the original row's two fields.
func TestShortcutFirstGameCustomSpellKeepsOriginalRows(t *testing.T) {
	f, _ := originalCityRouteFixture(t)
	f.quickSpells = spellSlots(0, 999, 0, 0, uint32(originalBookIDs[3]))
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	doc := engineSaveDocument(t, raw)
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil || a.Session == nil {
		t.Fatal("session record", err)
	}
	type originalRow struct {
		Slot uint8
		ID   uint32
	}
	want, _ := json.Marshal([]originalRow{{0, 999}})
	got, _ := json.Marshal(a.Session.QuickSpells)
	if a.Session.ShortcutLayout != 0 || !bytes.Equal(got, want) {
		t.Fatalf("layout %d rows %s, want layout 0 rows %s", a.Session.ShortcutLayout, got, want)
	}
	cold := &FrontEnd{InstallResources: f.InstallResources}
	if _, _, err := cold.RestoreOriginal(raw); err != nil || cold.quickSpells != f.quickSpells {
		t.Fatal("round trip", cold.quickSpells, err)
	}
}
