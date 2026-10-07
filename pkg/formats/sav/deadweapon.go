package sav

// DeadWeapon retains one terminal Unit's inert held Weapon, not an equipment
// projection. SAV-TOKEN-034/-TOKENPOS-074 and the unaffected Item/Weapon
// programmes of SAV-MEMBER-036 fix every member below. Effects and WeaponSpell
// must be absent. W52/W6A remain named fixed-width members of unknown meaning,
// not a whole object/body blob or guessed combat values.
type DeadWeapon struct {
	Present                  bool
	ArchiveIndex             uint16
	Cell, PackedCell         uint16
	FineX, FineY             uint8
	PositionU06              uint16
	TerrainKey, RuntimeID    uint32
	T0C                      uint8
	T0E                      uint16
	T08                      uint32
	T18                      uint16
	T1C, Identity, Reference uint32
	F40, F42                 uint16
	F44, F45, F46, F47       uint8
	F48, F4A                 uint16
	W52                      [24]byte
	W6A                      [22]byte
	W50                      uint8
}

func projectDeadWeapon(r *Record) (DeadWeapon, bool) {
	if r == nil || r.Class != "Weapon" || r.Index == 0 || r.value("Identity") == 0 ||
		len(r.Value) != 17 || len(r.Raw) != 3 || len(r.Counts) != 1 ||
		r.Counts["Effects"] != 0 || len(r.Refs) != 0 || len(r.Text) != 0 || len(r.SpellSlots) != 0 {
		return DeadWeapon{}, false
	}
	// Fail closed if a later walker adds a member this bounded DTO cannot hold.
	for _, name := range []string{"RuntimeID", "T0C", "T0E", "T08", "T18", "T1C", "Identity", "Reference", "F40", "F42", "F44", "F45", "F46", "F47", "F48", "F4A", "W50"} {
		if _, ok := r.Value[name]; !ok {
			return DeadWeapon{}, false
		}
	}
	if _, ok := r.Counts["Effects"]; !ok || len(r.Raw["Block12"]) != 12 || len(r.Raw["W52"]) != 24 || len(r.Raw["W6A"]) != 22 {
		return DeadWeapon{}, false
	}
	p := r.Raw["Block12"]
	w := DeadWeapon{
		Present: true, ArchiveIndex: r.Index,
		Cell: u16(p, 0), PackedCell: u16(p, 2), FineX: p[4], FineY: p[5], PositionU06: u16(p, 6), TerrainKey: u32(p, 8),
		RuntimeID: r.value("RuntimeID"), T0C: uint8(r.value("T0C")), T0E: uint16(r.value("T0E")),
		T08: r.value("T08"), T18: uint16(r.value("T18")), T1C: r.value("T1C"), Identity: r.value("Identity"), Reference: r.value("Reference"),
		F40: uint16(r.value("F40")), F42: uint16(r.value("F42")), F44: uint8(r.value("F44")),
		F45: uint8(r.value("F45")), F46: uint8(r.value("F46")), F47: uint8(r.value("F47")), F48: uint16(r.value("F48")), F4A: uint16(r.value("F4A")), W50: uint8(r.value("W50")),
	}
	copy(w.W52[:], r.Raw["W52"])
	copy(w.W6A[:], r.Raw["W6A"])
	return w, true
}
