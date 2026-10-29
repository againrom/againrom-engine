package mod

type EquipmentSlot struct {
	Number          int
	Name            string
	ModItem, Anchor bool
}

func EquipmentSlots() []EquipmentSlot {
	return []EquipmentSlot{
		{1, "weapon", false, true}, {2, "shield", true, true}, {3, "unused", false, false},
		{4, "ring", true, true}, {5, "amulet", true, true}, {6, "head", true, true},
		{7, "body", true, true}, {8, "outer", true, true}, {9, "bracers", true, true},
		{10, "gloves", true, true}, {11, "legs", false, true}, {12, "boots", true, true},
	}
}
