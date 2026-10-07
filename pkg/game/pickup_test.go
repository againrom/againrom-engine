package game

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A take posts a line for each stack it created or merged units into, in the
// container's order, and none for a stack it left alone: before A1 B2, after
// A1 C1 B3 reaches C1 and B3.
func TestReachedStacksAreTheCreatedAndGrownOnesInContainerOrder(t *testing.T) {
	before := []sim.ItemStack{sim.PlainStack(eqSwordCode, 1), sim.PlainStack(eqNoSlotCode, 2)}
	after := []sim.ItemStack{sim.PlainStack(eqSwordCode, 1), sim.PlainStack(eqMaceCode, 1), sim.PlainStack(eqNoSlotCode, 3)}
	want := []sim.ItemStack{after[1], after[2]}
	if got := reachedStacks(before, after); !reflect.DeepEqual(got, want) {
		t.Fatalf("reachedStacks = %+v, want %+v", got, want)
	}
	if got := reachedStacks(after, after); got != nil {
		t.Errorf("a container the take left alone reached %+v, want nil", got)
	}
}

// A stack bound to a saved object is the same stack by that object alone:
// another field changing reaches nothing, a larger count or a new object
// reaches the stack.
func TestReachedStacksKnowASavedItemByItsObject(t *testing.T) {
	held := sim.PlainStack(eqSwordCode, 1)
	held.ObjectID = 5
	repriced := held
	repriced.Price = 9
	grown := held
	grown.Count = 2
	fresh := held
	fresh.ObjectID = 6
	if got := reachedStacks([]sim.ItemStack{held}, []sim.ItemStack{repriced}); got != nil {
		t.Errorf("a repriced Item reached %+v, want nil", got)
	}
	if got := reachedStacks([]sim.ItemStack{held}, []sim.ItemStack{grown, fresh}); !reflect.DeepEqual(got, []sim.ItemStack{grown, fresh}) {
		t.Errorf("reachedStacks = %+v, want the grown Item and the new one", got)
	}
}

func TestPickupLinesForTakeOfAGoldOnlySackStatesTheGain(t *testing.T) {
	words := ui.AuthoredWords()
	got := pickupLinesForTake(nil, 500, nil, &words)
	if !reflect.DeepEqual(got, []string{"Picked up 500 gold"}) {
		t.Fatalf("pickupLinesForTake(gold 500) = %q, want the one gold line", got)
	}
}

func TestInventoryGoldIsAppendedAfterRealItems(t *testing.T) {
	subject := ui.InventorySubject{
		Pack:      []*image.RGBA{image.NewRGBA(image.Rect(0, 0, 1, 1))},
		PackCount: []uint32{2},
		PackInfo:  [][]string{{"Sword"}},
	}
	money := image.NewRGBA(image.Rect(0, 0, 80, 80))
	appendInventoryGold(&subject, 500, money)

	if len(subject.Pack) != 2 || subject.Pack[1] != money {
		t.Fatalf("gold pack = %+v, want a visible cell appended after the item", subject.Pack)
	}
	if !reflect.DeepEqual(subject.PackPurse, []bool{false, true}) {
		t.Fatalf("PackPurse = %v, want only the appended purse marked", subject.PackPurse)
	}
	if !reflect.DeepEqual(subject.PackCount, []uint32{2, 500}) {
		t.Fatalf("PackCount = %v, want [2 500]", subject.PackCount)
	}
	if len(subject.PackInfo) != 2 || len(subject.PackInfo[1]) == 0 || subject.PackInfo[1][0] != "Gold" {
		t.Fatalf("PackInfo = %v, want Gold information on the appended cell", subject.PackInfo)
	}
}

func TestPickupLinesForItemsWithNoItemsAnswersNil(t *testing.T) {
	words := ui.AuthoredWords()
	if got := pickupLinesForItems(nil, eqDefsTable(t), &words); got != nil {
		t.Errorf("pickupLinesForItems(nil, table) = %q, want nil", got)
	}
	if got := pickupLinesForItems(nil, nil, &words); got != nil {
		t.Errorf("pickupLinesForItems(nil, nil) = %q, want nil", got)
	}
}

func TestItemNameRecoversAResolvableWeaponsOwnName(t *testing.T) {
	table := eqDefsTable(t)
	if got := itemName(data.ItemCode(eqSwordCode), table); got != "Sword" {
		t.Errorf("itemName(eqSwordCode, table) = %q, want %q", got, "Sword")
	}
}

func TestItemNameFallsBackToTheSevenDigitNameWhenUnresolvable(t *testing.T) {
	table := eqDefsTable(t)
	code := data.ItemCode(eqNoSlotCode)
	want := code.Name()
	if got := itemName(code, table); got != want {
		t.Errorf("itemName(eqNoSlotCode, table) = %q, want the fallback %q", got, want)
	}
}

// invPartyGear's own "MAY BE NIL": a nil table resolves nothing at all, so
// EVERY code — including one that would otherwise resolve — falls back to
// its own digits rather than reaching into a nil collection.
func TestItemNameFallsBackWithNoTableAtAll(t *testing.T) {
	code := data.ItemCode(eqSwordCode)
	want := code.Name()
	if got := itemName(code, nil); got != want {
		t.Errorf("itemName(eqSwordCode, nil) = %q, want the fallback %q", got, want)
	}
}

// 0151: the stored table is consulted BEFORE data.WeaponFromCode, not
// after it. A code that WOULD resolve through WeaponFromCode still reads
// the stored line when one is present, because research establishes the
// original names an item off the stored table and never off a recomposed
// weapon name (ITEM-DISPNAME-036).
func TestItemNameConsultsTheStoredTableBeforeWeaponFromCode(t *testing.T) {
	table := eqDefsTable(t)
	table.Names = data.ItemNames{data.ItemCode(eqSwordCode): "Blade of Renown"}
	if got := itemName(data.ItemCode(eqSwordCode), table); got != "Blade of Renown" {
		t.Errorf("itemName(eqSwordCode, table) = %q, want the stored name %q", got, "Blade of Renown")
	}
}

// 0151, defect 6: an armour code cannot resolve through data.WeaponFromCode
// at all (field B names an equipment slot, not the weapon class), so
// before this story it fell straight to its own seven digits. The stored
// table names it directly, with no weapon recovery in between.
func TestItemNameRecoversAnArmourCodeFromTheStoredTable(t *testing.T) {
	table := eqDefsTable(t)
	armourCode := data.ItemCode(uint16(0)<<12 | uint16(3)<<8 | uint16(0)<<5 | uint16(1))
	table.Names = data.ItemNames{armourCode: "Chain Mail"}
	if got := itemName(armourCode, table); got != "Chain Mail" {
		t.Errorf("itemName(armourCode, table) = %q, want %q", got, "Chain Mail")
	}
}

// R-4, exercised end to end through grab itself: TakeSack's own owner guard
// (an owner at or past relationSlots) refuses the transfer BEFORE anything
// moves (sim/carry.go), and grab's own gate on that error must leave the
// world exactly as it found it — the sack still standing, nothing appended
// to the subject's container — which is the same "posted only when the take
// actually succeeded" property TestMapWorldGrabIsInertWhenTheSubjectStandsOnNoSack
// (world_test.go) already proves for the OTHER refusal TakeSack can give.
func TestMapWorldGrabLeavesTheWorldUntouchedWhenTakeSackRefuses(t *testing.T) {
	const invalidOwner = 50 // relationSlots (sim/relations.go), unexported there
	w, err := sim.NewLootWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, Owner: invalidOwner}}, nil, sim.Relations{},
		[]sim.Sack{{X: 3, Y: 3, Gold: 5, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}
	before := w.Hash()

	mw := grabWorld(t, w, 7, missionSource{})
	mw.grab(0, 0, 0, false)

	if got := w.Hash(); got != before {
		t.Errorf("World.Hash() changed from %d to %d — an invalid owner must refuse the whole transfer",
			before, got)
	}
	if got := w.Sacks(); len(got) != 1 {
		t.Errorf("Sacks() = %v, want the one sack still standing — TakeSack must have refused it", got)
	}
	if codes, ok := w.Carried(7); !ok || len(codes) != 0 {
		t.Errorf("Carried(7) = %v, %v, want none, true — nothing may have moved into the container", codes, ok)
	}
}

// The message line names exactly the Items the container received: grab over
// an empty pack reaches every stack the take left in it, so the lines posted
// are those stacks' own lines, one per Item at its own count.
func TestMapWorldGrabsLinesMatchWhatTheContainerReceived(t *testing.T) {
	w, err := sim.NewLootWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3}}, nil, sim.Relations{},
		[]sim.Sack{{X: 3, Y: 3, Items: []uint16{eqSwordCode, eqNoSlotCode}}})
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}
	mw := grabWorld(t, w, 7, missionSource{})

	mw.grab(0, 0, 0, false)

	stacks, ok := w.CarriedStacks(7)
	if !ok || len(stacks) != 2 {
		t.Fatalf("CarriedStacks(7) = %v, %v, want the sack's two Items, true", stacks, ok)
	}
	words := mw.view.Words()
	want := announced(pickupLinesForItems(stacks, mw.invParty.table, &words)...)
	if rows := mw.view.MessageLines(); len(rows) != 2 || !reflect.DeepEqual(rows, want) {
		t.Errorf("grab posted %+v, want the lines of the two Items the container received %+v", rows, want)
	}
}
