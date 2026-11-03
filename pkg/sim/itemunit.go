package sim

// UnitRule is a row of the item identity rule: the fields a crossing of an
// item between a shop, a bag, a party pack, the city holdings or a saved
// form compares to call two items one unit. SameUnit answers every row.
type UnitRule uint8

const (
	// UnitStack keeps the whole persisted cell: saved object, code, kind,
	// price, count, weight, equipment operands, effects and native record.
	// Two instances compare as cells of one unit each.
	UnitStack UnitRule = iota + 1
	// UnitDeal keeps the kind and the item (ItemEqual); the saved object and
	// the price are left out.
	UnitDeal
	// UnitItem keeps the item alone (ItemEqual): the saved object, and the
	// price and kind outside a potion, are left out. A shelf compares
	// without price (DIV-322).
	UnitItem
	// UnitPriced keeps the item and its price; the saved object is left out.
	UnitPriced
	// UnitPricedObject keeps the saved object, the price and the item.
	// Distinct saved objects stay apart even when all their values agree.
	UnitPricedObject
	// UnitObject keeps the saved object alone.
	UnitObject
	// UnitMerge keeps what a container merge retains: the item, its price,
	// its kind and its native record apart from the record's identity and
	// runtime words.
	UnitMerge
	// UnitWorn keeps the item across worn changes, a Weapon's owned Spell
	// left out: TakeOffWorn deletes it and WearWorn builds it again, so the
	// Weapon taken off is the one worn before.
	UnitWorn
	// UnitForm keeps the fields a mission form carries: code, kind, price
	// and effects in order. Weight, equipment operands, saved object and
	// native record are left out.
	UnitForm
)

// ItemUnit is a value a crossing compares: one instance or one stack cell.
type ItemUnit interface{ ItemInstance | ItemStack }

// SameUnit reports whether a and b are one item unit under rule.
func SameUnit[T ItemUnit](a, b T, rule UnitRule) bool {
	x, y := unitCell(a), unitCell(b)
	switch rule {
	case UnitStack:
		return StackStateEqual(x, y)
	case UnitObject:
		return x.ObjectID == y.ObjectID
	case UnitForm:
		return x.Code == y.Code && x.Kind == y.Kind && x.Price == y.Price && itemEffectsEqual(x.Effects, y.Effects)
	}
	i, j := x.view(), y.view()
	switch rule {
	case UnitDeal:
		i.ObjectID, i.Price, j.ObjectID, j.Price = 0, 0, 0, 0
		return i.Kind == j.Kind && ItemEqual(i, j)
	case UnitItem:
		return ItemEqual(i, j)
	case UnitPriced:
		return i.Price == j.Price && ItemEqual(i, j)
	case UnitPricedObject:
		return i.ObjectID == j.ObjectID && i.Price == j.Price && ItemEqual(i, j)
	case UnitMerge:
		return ItemEqual(i, j) && i.Price == j.Price && i.Kind == j.Kind && nativeItemMergeEqual(i.NativeRecord, j.NativeRecord)
	case UnitWorn:
		i.SourceEquipment.Spell, j.SourceEquipment.Spell = SourceItemSpell{}, SourceItemSpell{}
		return ItemEqual(i, j)
	}
	panic("sim: unknown item unit rule")
}

// Empty reports a cell that holds nothing: no item value and no count, the
// null position of a saved pack.
func (s ItemStack) Empty() bool { return s.Count == 0 && s.view().Empty() }

// unitCell is v as a cell; an instance is a cell of no count. It shares v's
// slices and record: SameUnit reads them and writes neither.
func unitCell[T ItemUnit](v T) ItemStack {
	switch v := any(v).(type) {
	case ItemStack:
		return v
	case ItemInstance:
		return ItemStack{ObjectID: v.ObjectID, Code: v.Code, Kind: v.Kind, Effects: v.Effects, Price: v.Price, Weight: v.Weight, WeightPresent: v.WeightPresent, SourceEquipment: v.SourceEquipment, NativeRecord: v.NativeRecord}
	}
	panic("sim: unknown item unit")
}

// view is the cell's item sharing its slices and record, for reading only.
func (s ItemStack) view() ItemInstance {
	return ItemInstance{ObjectID: s.ObjectID, Code: s.Code, Kind: s.Kind, Effects: s.Effects, Price: s.Price, Weight: s.Weight, WeightPresent: s.WeightPresent, SourceEquipment: s.SourceEquipment, NativeRecord: s.NativeRecord}
}
