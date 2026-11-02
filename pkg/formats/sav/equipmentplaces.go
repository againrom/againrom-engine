package sav

// An actor record keeps its equipment in three fields: HeldWeapon and
// HeldShield hold one ref each, and Worn holds one ref per slot. Each slot has
// one home in equipmentPlaces. A Worn position whose slot lives in a held
// field is an overlap position: a writer leaves it empty, and a reader that
// admits no overlap refuses an item there. A Unit record has no Worn. Every
// reader and writer of these fields routes a slot through this table.
const (
	HeldWeaponField = "HeldWeapon"
	HeldShieldField = "HeldShield"
	WornField       = "Worn"
)

// EquipmentSlots is the number of equipment slots and of Worn positions.
const EquipmentSlots = 12

// EquipmentPlace is one slot's home: the field and the position in it that
// hold the slot's ref, and the item record class the slot takes.
type EquipmentPlace struct {
	Field     string
	Ordinal   int
	ItemClass string
}

var equipmentPlaces = [EquipmentSlots]EquipmentPlace{
	{HeldWeaponField, 0, "Weapon"}, {HeldShieldField, 0, "Shield"},
	{WornField, 2, "Armor"}, {WornField, 3, "Armor"}, {WornField, 4, "Armor"}, {WornField, 5, "Armor"},
	{WornField, 6, "Armor"}, {WornField, 7, "Armor"}, {WornField, 8, "Armor"}, {WornField, 9, "Armor"},
	{WornField, 10, "Armor"}, {WornField, 11, "Armor"},
}

// EquipmentFields is the equipment fields in record order.
func EquipmentFields() [3]string { return [3]string{HeldWeaponField, HeldShieldField, WornField} }

// EquipmentPlaces is every slot's home, in slot order.
func EquipmentPlaces() [EquipmentSlots]EquipmentPlace { return equipmentPlaces }

// EquipmentSlotPlace is one slot's home.
func EquipmentSlotPlace(slot int) EquipmentPlace { return equipmentPlaces[slot] }

// EquipmentFieldSlot is the slot one position of an equipment field stands
// for: a held field's own slot, or the Worn position's slot.
func EquipmentFieldSlot(field string, ordinal int) (int, bool) {
	if field == WornField {
		return ordinal, true
	}
	for slot, place := range equipmentPlaces {
		if place.Field == field {
			return slot, true
		}
	}
	return 0, false
}

// WornOverlap reports whether a Worn position stands for a slot whose home
// is a held field.
func WornOverlap(ordinal int) bool {
	return ordinal >= 0 && ordinal < EquipmentSlots && equipmentPlaces[ordinal].Field != WornField
}

// CarriesWorn reports whether records of a class have the Worn field.
func CarriesWorn(class string) bool { return class != "Unit" }

// HeldPiece is the piece a holdings record keeps in a held field.
func (a *ActorHoldings) HeldPiece(field string) *Piece {
	switch field {
	case HeldWeaponField:
		return a.HeldWeapon
	case HeldShieldField:
		return a.HeldShield
	}
	return nil
}

func (a *ActorHoldings) setHeldPiece(field string, p *Piece) {
	switch field {
	case HeldWeaponField:
		a.HeldWeapon = p
	case HeldShieldField:
		a.HeldShield = p
	}
}
