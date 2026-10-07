package sim

import "testing"

func TestSavedObjectIdentity1115ValueConversionsAreNotAllocation(t *testing.T) {
	item := ItemInstance{ObjectID: 0x123456789abcdef0, Code: 17, Kind: 3,
		Price: -29, Effects: []ItemEffect{{Kind: 6, Operand: 31}}}
	stack := StackItem(item, 7)
	copy := stack.Clone()
	if stack.ObjectID != item.ObjectID || copy.ObjectID != item.ObjectID || copy.Instance().ObjectID != item.ObjectID || !StackStateEqual(stack, copy) {
		t.Fatal("value conversion minted or lost native object identity")
	}
	copy.Effects[0].Operand++
	if item.Effects[0].Operand != 31 || stack.Effects[0].Operand != 31 {
		t.Fatal("identity handle made mutable effect values alias")
	}
	other := stack.Clone()
	other.ObjectID++
	if StackStateEqual(stack, other) || !ItemEqual(stack.Instance(), other.Instance()) {
		t.Fatal("object identity must differ from semantic merge equality")
	}
	residue := ItemInstance{ObjectID: 9}
	if residue.Empty() || residue.ValidateWeight() == nil || !itemHasMetadata(residue) {
		t.Fatal("identity was hidden in the empty sentinel")
	}
	if PlainItem(17).ObjectID != 0 || PlainStack(17, 2).ObjectID != 0 {
		t.Fatal("legacy constructor invented object identity")
	}
}
