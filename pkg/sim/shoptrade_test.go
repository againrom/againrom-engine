package sim

import (
	"reflect"
	"slices"
	"testing"
)

func init() {
	worldMethods = append(worldMethods, "TradeCarried")
	slices.Sort(worldMethods)
	worldWriters = append(worldWriters, "TradeCarried")
}

// shopUseReport runs one structure use to its end and returns every Shop use
// the step reports named.
func shopUseReport(t *testing.T, w *World) []StructureUse {
	t.Helper()
	var uses []StructureUse
	uses = append(uses, StepReported(w, []Command{UseStructure(0, 0)}).ShopUses...)
	for range 400 {
		if len(w.StructureUses()) == 0 {
			return uses
		}
		uses = append(uses, StepReported(w, nil).ShopUses...)
	}
	t.Fatal("interaction did not finish")
	return nil
}

func TestShopStructureUseReportsTheUserAndChangesNoStructure(t *testing.T) {
	for _, kind := range []uint16{34, 35} {
		w := structureUseWorld(t, kind)
		uses := shopUseReport(t, w)
		if !reflect.DeepEqual(uses, []StructureUse{{Entity: 0, Structure: 0}}) {
			t.Fatalf("kind %d reported %+v", kind, uses)
		}
		if s, e := w.structures[0], w.entities[0]; s.Field42 != 3 || e.HP != 10 || e.Mana != 20 {
			t.Fatalf("kind %d use changed the structure or the user: charge %d pools %d/%d", kind, s.Field42, e.HP, e.Mana)
		}
	}
	for _, kind := range []uint16{15, 16, 28, 29} {
		if uses := shopUseReport(t, structureUseWorld(t, kind)); len(uses) != 0 {
			t.Fatalf("kind %d reported a Shop use %+v", kind, uses)
		}
	}
}

func TestShopStructureUseIsReportOnly(t *testing.T) {
	quiet, told := structureUseWorld(t, 35), structureUseWorld(t, 35)
	cmds, uses := []Command{UseStructure(0, 0)}, 0
	for range 400 {
		Step(quiet, cmds)
		uses += len(StepReported(told, cmds).ShopUses)
		cmds = nil
	}
	if uses != 1 || len(told.StructureUses()) != 0 || told.Hash() != quiet.Hash() {
		t.Fatal("the Shop use report changed canonical state", uses)
	}
}

func TestTradeCarriedBuysAndSellsAtomically(t *testing.T) {
	w := cheatWorld(t)
	if !w.CheatAddItem(1, StackItem(PlainItem(0xe03), 2)) || !w.CheatAddGold(SelfSlot, 50) {
		t.Fatal("setup refused")
	}
	tick := w.Tick()
	if !w.TradeCarried(1, []int{0}, []ItemStack{StackItem(PlainItem(0xe05), 1)}, 30) {
		t.Fatal("deal refused")
	}
	carried, _ := w.Carried(1)
	if w.Purse(SelfSlot) != 30 || !reflect.DeepEqual(carried, []uint16{0xe03, 0xe05}) || w.Tick() != tick {
		t.Fatalf("deal result: purse %d carry %x tick %d", w.Purse(SelfSlot), carried, w.Tick())
	}
	hash := w.Hash()
	for _, refused := range []func() bool{
		func() bool { return w.TradeCarried(1, []int{5}, nil, 99) },
		func() bool { return w.TradeCarried(1, []int{0, 0}, nil, 99) },
		func() bool { return w.TradeCarried(1, nil, []ItemStack{{}}, 99) },
		func() bool { return w.TradeCarried(99, nil, nil, 99) },
	} {
		if refused() || w.Hash() != hash {
			t.Fatal("a refused deal changed state")
		}
	}
	cheatCold(t, w)
}

func TestTradeCarriedKeepsSavedObjectsAndReloads(t *testing.T) {
	w := mustWorld(t, 7, Bounds{8, 8}, []Entity{{ID: 7, X: 1, Y: 1, HP: 20, MaxHP: 20}, {ID: 8, X: 2, Y: 1, HP: 20, MaxHP: 20}})
	w.carried[0] = []ItemStack{{Code: 0x204, Kind: 1, Price: 31, Count: 3, WeightPresent: true, Weight: 2}}
	w.recomputeLoad(0)
	bindOperationsPack(t, w, 0)
	if !w.TradeCarried(7, []int{0}, []ItemStack{StackItem(PlainItem(0xe05), 1)}, 12) {
		t.Fatal("deal refused on a saved-object pack")
	}
	held, _ := w.CarriedStacks(7)
	if len(held) != 2 || held[0].Code != 0x204 || held[0].Count != 2 || held[1].Code != 0xe05 || w.Purse(0) != 12 {
		t.Fatalf("saved-object deal result %+v purse %d", held, w.Purse(0))
	}
	reloadOperations(t, w)
}
