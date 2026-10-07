package sim

import "encoding/binary"

// OriginalDeadWeapon is nonplayable provenance for the bounded terminal Unit
// held-Weapon arm. It is never placed in carried/equipped stock or interpreted
// as a combat rule. Effects and outgoing object references are absent by this
// arm's contract. Named W52/W6A members remain uninterpreted and byte-exact.
type OriginalDeadWeapon struct {
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

// The 98-byte payload follows the archive index with the complete 37-byte
// Token, 12-byte Item scalar body and 47-byte Weapon member body. Empty Effects
// and null WeaponSpell are implicit in this bounded variant, not live defaults.
func putDeadWeapon(b []byte, w OriginalDeadWeapon) {
	binary.LittleEndian.PutUint16(b, w.ArchiveIndex)
	binary.LittleEndian.PutUint16(b[2:], w.Cell)
	binary.LittleEndian.PutUint16(b[4:], w.PackedCell)
	b[6], b[7] = w.FineX, w.FineY
	binary.LittleEndian.PutUint16(b[8:], w.PositionU06)
	binary.LittleEndian.PutUint32(b[10:], w.TerrainKey)
	binary.LittleEndian.PutUint32(b[14:], w.RuntimeID)
	b[18] = w.T0C
	binary.LittleEndian.PutUint16(b[19:], w.T0E)
	binary.LittleEndian.PutUint32(b[21:], w.T08)
	binary.LittleEndian.PutUint16(b[25:], w.T18)
	binary.LittleEndian.PutUint32(b[27:], w.T1C)
	binary.LittleEndian.PutUint32(b[31:], w.Identity)
	binary.LittleEndian.PutUint32(b[35:], w.Reference)
	binary.LittleEndian.PutUint16(b[39:], w.F40)
	binary.LittleEndian.PutUint16(b[41:], w.F42)
	b[43], b[44], b[45] = w.F44, w.F45, w.F46
	binary.LittleEndian.PutUint16(b[46:], w.F48)
	binary.LittleEndian.PutUint16(b[48:], w.F4A)
	b[50] = w.F47
	copy(b[51:75], w.W52[:])
	copy(b[75:97], w.W6A[:])
	b[97] = w.W50
}

func readDeadWeapon(b []byte) OriginalDeadWeapon {
	w := OriginalDeadWeapon{
		ArchiveIndex: binary.LittleEndian.Uint16(b),
		Cell:         binary.LittleEndian.Uint16(b[2:]), PackedCell: binary.LittleEndian.Uint16(b[4:]),
		FineX: b[6], FineY: b[7], PositionU06: binary.LittleEndian.Uint16(b[8:]),
		TerrainKey: binary.LittleEndian.Uint32(b[10:]), RuntimeID: binary.LittleEndian.Uint32(b[14:]),
		T0C: b[18], T0E: binary.LittleEndian.Uint16(b[19:]), T08: binary.LittleEndian.Uint32(b[21:]),
		T18: binary.LittleEndian.Uint16(b[25:]), T1C: binary.LittleEndian.Uint32(b[27:]),
		Identity: binary.LittleEndian.Uint32(b[31:]), Reference: binary.LittleEndian.Uint32(b[35:]),
		F40: binary.LittleEndian.Uint16(b[39:]), F42: binary.LittleEndian.Uint16(b[41:]),
		F44: b[43], F45: b[44], F46: b[45], F48: binary.LittleEndian.Uint16(b[46:]),
		F4A: binary.LittleEndian.Uint16(b[48:]), F47: b[50], W50: b[97],
	}
	copy(w.W52[:], b[51:75])
	copy(w.W6A[:], b[75:97])
	return w
}
