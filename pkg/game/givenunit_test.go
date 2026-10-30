package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// givenUnitMission opens a mission whose script gives the companion (id 2) to
// player 8 when the hero (id 1) stands in cells (4,4)..(7,7).
func givenUnitMission(t *testing.T) (*mapWorld, *ui.Viewer) {
	t.Helper()
	const hero, companion sim.EntityID = 1, 2
	script, err := sim.NewScript(
		[]sim.ScriptCheck{
			{Op: sim.ScriptCheckInBox, Register: 0, Args: [10]int32{4, 4, 7, 7}, Unit: hero, HasUnit: true},
			{Op: sim.ScriptCheckConstant, Register: 1, Args: [10]int32{1}},
		},
		[]sim.ScriptInstant{{Op: sim.ScriptInstantGiveUnit, Unit: companion, HasUnit: true, Player: 8, HasPlayer: true}},
		[]sim.ScriptTrigger{{Pairs: [3]sim.ScriptPair{{Left: 0, Right: 1, Cmp: sim.ScriptCmpEQ, Used: true}},
			Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true}},
	)
	if err != nil {
		t.Fatal(err)
	}
	world, err := sim.NewScriptedWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil, []sim.Entity{
		{ID: hero, X: 1, Y: 1, HP: 40, MaxHP: 40, Owner: sim.SelfSlot, TypeID: sim.HumanTypeID, KnownSpells: 1 << 6},
		{ID: companion, X: 2, Y: 2, HP: 30, MaxHP: 30, Owner: sim.SelfSlot, TypeID: sim.HumanTypeID, KnownSpells: 1 << 6},
	}, script)
	if err != nil {
		t.Fatal(err)
	}
	party := []mapload.PartyMember{
		{ID: "hero", Name: "Hero", StartingHero: true, PlayerCharacter: true},
		{ID: "join:2", Name: "Companion", PlayerCharacter: true, CompanionNPC: 22},
	}
	mission := &Mission{Number: 130, World: world, Party: party, Start: mapload.Start{IDs: []sim.EntityID{hero, companion}}}
	v, err := ui.NewViewer("given", terrain.Grid{Width: 8, Height: 8, Tiles: make([]uint16, 64)}, &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	mw := openMission(mission, nil, nil, v, nil, nil, nil)
	mw.tick()
	return mw, v
}

// Every route that opens or changes a party member's own state refuses the
// companion once the script gave it to another player, and the spell book of
// a unit the player does not own shows the no-hero state unless the map is
// revealed (DIV-4313).
func TestAGivenCompanionLeavesEveryManageRoute(t *testing.T) {
	mw, v := givenUnitMission(t)
	if !v.SelectUnit(2) {
		t.Fatal("the companion is not selectable before the handover")
	}
	mw.refreshInventorySelection()
	mw.pushSpellbook()
	if owner, held := v.SpellbookOwner(); mw.invSubject.ID != 2 || mw.managedMember(2) != 1 || !held || owner != 2 {
		t.Fatalf("before the handover: subject %d member %d book %d/%v", mw.invSubject.ID, mw.managedMember(2), owner, held)
	}
	if err := mw.world.HeadlessPlace(1, 5, 5); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 400; i++ {
		if e, _ := mw.entity(2); e.Owner == 8 {
			break
		}
		mw.tick()
	}
	if e, _ := mw.entity(2); e.Owner != 8 {
		h, _ := mw.entity(1)
		t.Fatal("the script did not give the companion away", h.X, h.Y, mw.world.Tick())
	}
	if mw.managedMember(2) != -1 || mw.managedMember(1) != 0 {
		t.Fatal("the managed party still holds the given companion")
	}
	if _, ok := mw.swapMember(2); ok {
		t.Fatal("the weapon swap still acts on the given companion")
	}
	mw.observeShopUses([]sim.StructureUse{{Entity: 2}})
	if mw.shop.waiting {
		t.Fatal("the given companion still opens the Shop")
	}
	mw.refreshInventorySelection()
	if mw.invSubject.ID != 1 {
		t.Fatalf("inventory subject = %d, want the hero after the handover", mw.invSubject.ID)
	}
	mw.switchInventorySubject(2)
	if mw.invSubject.ID != 1 {
		t.Fatal("the inventory switched to the given companion")
	}
	mw.pushSpellbook()
	if _, held := v.SpellbookOwner(); held {
		t.Fatal("the book of the given companion is shown")
	}
	v.SetFogReveal(true)
	mw.pushSpellbook()
	if owner, held := v.SpellbookOwner(); !held || owner != 2 {
		t.Fatalf("revealed map book = %d/%v, want the companion's", owner, held)
	}
	v.SetFogReveal(false)
	if !v.SelectUnit(1) {
		t.Fatal("the hero is not selectable")
	}
	mw.pushSpellbook()
	if owner, held := v.SpellbookOwner(); !held || owner != 1 {
		t.Fatalf("own unit book = %d/%v, want the hero's", owner, held)
	}
}
