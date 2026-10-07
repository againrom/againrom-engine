package sim

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

var (
	packMergeBow   = ItemInstance{Code: 0x8134, Kind: 2, Price: 267}
	packMergeCheap = ItemInstance{Code: 0x8114, Kind: 2, Price: 133}
)

// packMergeMagicBow is the bow with an effect, which is not stackable
// (ITEM-STACK-003).
func packMergeMagicBow() ItemInstance {
	magic := packMergeBow.Clone()
	magic.Price = 4276
	magic.Effects = []ItemEffect{{Kind: 12, Operand: 5}}
	return magic
}

// packMergeBuilt is the definition constructor the world declares for the
// bow's code, at code weight 3.
func packMergeBuilt() SourceEquipment {
	s := SourceEquipment{Class: SourceWeapon, DefinitionRow: 20, OwnKind: 4,
		Definition: SourceWeaponDefinition{Present: true, AttackType: 5, Hands: 2, Charge: 20, Relax: 5, Suitable: 1}}
	s.Attack[0], s.Attack[14], s.Attack[15] = 10, 3, 2
	return s
}

// packMergeBuiltBow is the bow that constructor writes, as a map archer
// wears it. With retained set it also holds Weapon bytes 22 and 23 no
// constructor writes, as a bow a SAV restores does.
func packMergeBuiltBow(retained bool) ItemInstance {
	bow := packMergeBow.Clone()
	bow.Weight, bow.WeightPresent, bow.SourceEquipment = 3, true, packMergeBuilt()
	if retained {
		bow.SourceEquipment.Attack[22], bow.SourceEquipment.Attack[23] = 15, 1
	}
	return bow
}

// packMergeBlockBow is that bow with Weapon byte 21 set, a block byte outside
// the two no claim reads, which keeps it apart from the constructor's bow.
func packMergeBlockBow() ItemInstance {
	bow := packMergeBuiltBow(false)
	bow.SourceEquipment.Attack[21] = 9
	return bow
}

// packMergeWorld is actor 7 holding held, actor 8 holding other and a map
// Sack at (3,1). With built set the world declares the bow's constructor, and
// with bound set actor 7's pack is saved objects, as mission entry binds a
// party pack.
func packMergeWorld(t *testing.T, built, bound bool, held, other []ItemStack, ground ...ItemInstance) *World {
	t.Helper()
	w := mustWorld(t, 25, Bounds{8, 8}, []Entity{{ID: 7, X: 1, Y: 1, HP: 20, MaxHP: 20}, {ID: 8, X: 2, Y: 1, HP: 20, MaxHP: 20}})
	if err := w.ReplaceGroundSacks([]Sack{makeSack(3, 1, 0, ground)}); err != nil {
		t.Fatal(err)
	}
	if built {
		if err := w.DeclareItemWeights([]ItemWeight{{Code: packMergeBow.Code, Weight: 3, Constructor: packMergeBuilt()}}); err != nil {
			t.Fatal(err)
		}
	}
	w.carried[0], w.carried[1] = held, other
	w.recomputeLoad(0)
	w.recomputeLoad(1)
	if bound {
		bindOperationsPack(t, w, 0)
	}
	return w
}

func packCells(stacks []ItemStack) string {
	var out []string
	for _, st := range stacks {
		out = append(out, fmt.Sprintf("%#04x x%d id%d weight %v/%d", st.Code, st.Count, st.ObjectID, st.WeightPresent, st.Weight))
	}
	return "[" + strings.Join(out, ", ") + "]"
}

// A bow picked up from a map Sack joins the equal saved-object bow, which
// keeps its identity and ORs in the pickup flag (ITEM-MERGE-129,
// ITEM-GROUNDMOVE-130). Another code and an enchanted bow stay apart.
func TestPickupJoinsTheEqualBoundBow(t *testing.T) {
	w := packMergeWorld(t, false, true, []ItemStack{StackItem(packMergeBow, 1)}, nil, packMergeBow, packMergeCheap, packMergeMagicBow())
	held := w.carried[0][0].ObjectID
	flags := w.savedObjects.item(held).Token.T08
	if err := w.TakeSack(7, 3, 1); err != nil {
		t.Fatal(err)
	}
	got := w.carried[0]
	if len(got) != 3 || got[0].ObjectID != held || got[0].Count != 2 ||
		got[1].Code != packMergeCheap.Code || got[1].Count != 1 || got[2].Code != packMergeBow.Code || got[2].Count != 1 || !got[2].Instance().HasEnchantment() {
		t.Fatalf("pack after the pickup: %s; want the held bow x2, the cheap bow x1 and the enchanted bow x1", packCells(got))
	}
	row := w.savedObjects.item(held)
	if row.Value.Count != 2 || row.Token.T08 != flags|1 {
		t.Fatalf("held object count %d flags %#x; want 2 and %#x", row.Value.Count, row.Token.T08, flags|1)
	}
	if c := w.savedObjects.container(w.savedPackOwner(0)); !reflect.DeepEqual(c.Items, []SavedObjectID{held, 0, 0}) || c.Accumulator != w.containerWeight(0) {
		t.Fatalf("saved pack %v weight %d; want [%d 0 0] and %d", c.Items, c.Accumulator, held, w.containerWeight(0))
	}
	cold := reloadOperations(t, w)
	if !reflect.DeepEqual(cold.carried[0], w.carried[0]) {
		t.Fatalf("native continuation pack %s, want %s", packCells(cold.carried[0]), packCells(w.carried[0]))
	}
}

// An enchanted bow picked up joins the held bow that carries the same ordered
// effects, and a bow with other effects or none stays apart: equally enchanted
// equipment stacks (DIV-1473).
func TestPickupJoinsTheEquallyEnchantedBow(t *testing.T) {
	other := packMergeMagicBow()
	other.Effects = []ItemEffect{{Kind: 12, Operand: 6}}
	for _, bound := range []bool{true, false} {
		w := packMergeWorld(t, false, bound, []ItemStack{StackItem(packMergeMagicBow(), 1)}, nil, packMergeMagicBow(), other, packMergeBow)
		held := w.carried[0][0].ObjectID
		if err := w.TakeSack(7, 3, 1); err != nil {
			t.Fatal(err)
		}
		got := w.carried[0]
		if len(got) != 3 || got[0].ObjectID != held || got[0].Count != 2 || !ItemEqual(got[0].Instance(), packMergeMagicBow()) ||
			!ItemEqual(got[1].Instance(), other) || got[1].Count != 1 || got[2].Instance().HasEnchantment() || got[2].Count != 1 {
			t.Fatalf("bound %v: pack after the pickup %s; want the enchanted bow x2, the other enchantment x1 and the plain bow x1", bound, packCells(got))
		}
		if cold := reloadOperations(t, w); !reflect.DeepEqual(cold.carried[0], w.carried[0]) {
			t.Fatalf("bound %v: native continuation pack %s, want %s", bound, packCells(cold.carried[0]), packCells(w.carried[0]))
		}
	}
}

// A bow moved from an unbound pack joins the receiving pack's equal
// saved-object bow.
func TestMovedBowJoinsTheEqualBoundBow(t *testing.T) {
	w := packMergeWorld(t, false, true, []ItemStack{StackItem(packMergeBow, 1)}, []ItemStack{StackItem(packMergeBow, 1), StackItem(packMergeCheap, 1)})
	held := w.carried[0][0].ObjectID
	if err := w.MoveCarried(8, 7, packMergeBow.Code, 1); err != nil {
		t.Fatal(err)
	}
	if err := w.MoveCarried(8, 7, packMergeCheap.Code, 1); err != nil {
		t.Fatal(err)
	}
	got := w.carried[0]
	if len(got) != 2 || got[0].ObjectID != held || got[0].Count != 2 || got[1].Code != packMergeCheap.Code || got[1].Count != 1 {
		t.Fatalf("receiving pack %s; want the held bow x2 and the cheap bow x1", packCells(got))
	}
	if len(w.carried[1]) != 0 || w.savedObjects.item(held).Value.Count != 2 {
		t.Fatalf("giving pack %s, held object count %d", packCells(w.carried[1]), w.savedObjects.item(held).Value.Count)
	}
	reloadOperations(t, w)
}

// A bow looted from a map archer is the constructor's bow, and it joins the
// pack's plain bow as that plain bow, which constructs to the same value
// (ITEM-STACK-003, ITEM-MERGE-129). A looted bow that differs from it only in
// Weapon bytes 22 and 23 joins too; one that differs in another block byte
// stays apart: DIV-762 keeps that operand.
func TestLootedBowJoinsThePlainBow(t *testing.T) {
	for _, bound := range []bool{true, false} {
		w := packMergeWorld(t, true, bound, []ItemStack{StackItem(packMergeBow, 1)}, nil, packMergeBuiltBow(false), packMergeBuiltBow(true), packMergeBlockBow())
		held := w.carried[0][0].ObjectID
		if err := w.TakeSack(7, 3, 1); err != nil {
			t.Fatal(err)
		}
		got := w.carried[0]
		if len(got) != 2 || got[0].ObjectID != held || got[0].Count != 3 || got[0].WeightPresent ||
			!ItemEqual(got[1].Instance(), packMergeBlockBow()) || got[1].Count != 1 {
			t.Fatalf("bound %v: pack after the loot %s; want the plain bow x3 and the bow with another block byte x1", bound, packCells(got))
		}
		if bound {
			if w.savedObjects.item(held).Value.Count != 3 {
				t.Fatalf("held object count %d, want 3", w.savedObjects.item(held).Value.Count)
			}
			reloadOperations(t, w)
		}
	}
}

// packMergeTailBow is the constructor's bow holding Weapon bytes 22 and 23 as
// a SAV stores them, which differ from bow to bow.
func packMergeTailBow(a, b byte) ItemInstance {
	bow := packMergeBuiltBow(false)
	bow.SourceEquipment.Attack[22], bow.SourceEquipment.Attack[23] = a, b
	return bow
}

// A bow that differs from the held bow only in Weapon bytes 22 and 23 is one
// item to the player: it joins the held bow, which keeps its own two bytes,
// and the incoming Item's are dropped with it (ITEM-STACK-003,
// ITEM-MERGE-129, DIV-762). A bow that differs in another block byte stays
// apart, and every cell survives a cold reload.
func TestPickupJoinsTheBowDifferingInWeaponBytes22And23(t *testing.T) {
	for _, bound := range []bool{true, false} {
		held := packMergeTailBow(1, 2)
		w := packMergeWorld(t, true, bound, []ItemStack{StackItem(held, 2)}, nil, packMergeTailBow(3, 4), packMergeTailBow(0, 0), packMergeBlockBow())
		id := w.carried[0][0].ObjectID
		if err := w.TakeSack(7, 3, 1); err != nil {
			t.Fatal(err)
		}
		got := w.carried[0]
		if len(got) != 2 || got[0].ObjectID != id || got[0].Count != 4 || got[0].SourceEquipment != held.SourceEquipment ||
			got[1].Count != 1 || got[1].SourceEquipment.Attack[21] != 9 {
			t.Fatalf("bound %v: pack after the pickup %s; want the held bow x4 with its own bytes and the block-byte bow x1", bound, packCells(got))
		}
		if bound && w.savedObjects.item(id).Value.SourceEquipment != held.SourceEquipment {
			t.Fatalf("held object blocks %+v, want the destination's", w.savedObjects.item(id).Value.SourceEquipment)
		}
		if cold := reloadOperations(t, w); !reflect.DeepEqual(cold.carried[0], w.carried[0]) {
			t.Fatalf("bound %v: native continuation pack %s, want %s", bound, packCells(cold.carried[0]), packCells(w.carried[0]))
		}
	}
}

// A bow moved from another pack joins the receiving pack's bow that differs
// only in Weapon bytes 22 and 23, and the receiving pack's bytes stay.
func TestMovedBowJoinsTheBowDifferingInWeaponBytes22And23(t *testing.T) {
	for _, bound := range []bool{true, false} {
		held, moved := packMergeTailBow(1, 2), packMergeTailBow(9, 8)
		w := packMergeWorld(t, true, bound, []ItemStack{StackItem(held, 1)}, []ItemStack{StackItem(moved, 2)})
		if err := w.MoveCarried(8, 7, moved.Code, 2); err != nil {
			t.Fatal(err)
		}
		if got := w.carried[0]; len(got) != 1 || got[0].Count != 3 || got[0].SourceEquipment != held.SourceEquipment || len(w.carried[1]) != 0 {
			t.Fatalf("bound %v: receiving pack %s, giving pack %s; want one bow x3 with the held bytes", bound, packCells(got), packCells(w.carried[1]))
		}
		reloadOperations(t, w)
	}
}

// A flat list of units, as a mission entry or an older SAV holds a party
// pack, folds bows that differ only in Weapon bytes 22 and 23 into the first
// one's cell and keeps its bytes (FoldItems, ITEM-MERGE-129).
func TestFoldItemsJoinsBowsDifferingInWeaponBytes22And23(t *testing.T) {
	first := packMergeTailBow(1, 2)
	got := FoldItems([]ItemInstance{first, packMergeTailBow(3, 4), packMergeBlockBow(), packMergeTailBow(5, 6), packMergeCheap})
	if len(got) != 3 || got[0].Count != 3 || got[0].SourceEquipment != first.SourceEquipment || got[1].Count != 1 || got[1].SourceEquipment.Attack[21] != 9 ||
		got[2].Code != packMergeCheap.Code {
		t.Fatalf("folded pack %s; want the first bow x3 with its bytes, the block-byte bow x1 and the cheap bow", packCells(got))
	}
}

// A plain bow picked up joins a pack bow whose blocks a SAV restored, a byte
// no constructor writes included. The plain bow holds no saved operand, and
// the held bow keeps its blocks (ITEM-MERGE-129).
func TestPlainBowJoinsTheRestoredBow(t *testing.T) {
	w := packMergeWorld(t, true, true, []ItemStack{StackItem(packMergeBuiltBow(true), 1)}, nil, packMergeBow, packMergeMagicBow())
	held := w.carried[0][0].ObjectID
	if err := w.TakeSack(7, 3, 1); err != nil {
		t.Fatal(err)
	}
	got := w.carried[0]
	if len(got) != 2 || got[0].ObjectID != held || got[0].Count != 2 || !ItemEqual(got[0].Instance(), packMergeBuiltBow(true)) ||
		!got[1].Instance().HasEnchantment() || got[1].Count != 1 {
		t.Fatalf("pack after the pickup %s; want the restored bow x2 and the enchanted bow x1", packCells(got))
	}
	if row := w.savedObjects.item(held); row.Value.Count != 2 || row.Value.SourceEquipment != packMergeBuiltBow(true).SourceEquipment {
		t.Fatalf("held object %+v", row.Value)
	}
	if c := w.savedObjects.container(w.savedPackOwner(0)); w.containerWeight(0) != 3*3 || c.Accumulator != w.containerWeight(0) {
		t.Fatalf("pack weight %d, saved accumulator %d; want 9", w.containerWeight(0), c.Accumulator)
	}
	reloadOperations(t, w)
}

// A plain potion picked up joins the pack potion a SAV restored with the
// per-unit weight 1 (ITEM-STACK-003), which the world's tables do not give a
// potion. The joined unit weighs what the held potion weighs
// (ITEM-MERGE-129).
func TestPlainPotionJoinsTheRestoredPotion(t *testing.T) {
	potion := ItemInstance{Code: 0x0e06, Kind: 3, Price: 50, Effects: []ItemEffect{{Kind: 8, Mode: 1, Operand: 5}}}
	restored := potion.Clone()
	restored.Weight, restored.WeightPresent = 1, true
	w := packMergeWorld(t, false, true, []ItemStack{StackItem(restored, 1)}, nil, potion)
	held := w.carried[0][0].ObjectID
	if err := w.TakeSack(7, 3, 1); err != nil {
		t.Fatal(err)
	}
	got := w.carried[0]
	if len(got) != 1 || got[0].ObjectID != held || got[0].Count != 2 || !got[0].WeightPresent || got[0].Weight != 1 {
		t.Fatalf("pack after the pickup %s; want the restored potion x2 at weight 1", packCells(got))
	}
	if c := w.savedObjects.container(w.savedPackOwner(0)); w.containerWeight(0) != 2 || c.Accumulator != 2 {
		t.Fatalf("pack weight %d, saved accumulator %d; want 2", w.containerWeight(0), c.Accumulator)
	}
	reloadOperations(t, w)
}
