package sav

import "encoding/binary"

// Payload returns the complete fixed-width SAV-CELLLOAD-111 image, excluding
// its independently serialized cell key. Unknown bytes are retained literally.
func (c DocumentCellData) Payload() [52]byte {
	var p [52]byte
	p[0], p[1], p[2], p[3] = c.Cost, c.Static, c.LayerCount, c.Residue03
	keys := [10]uint32{c.GroundActor, c.AirActor, c.Building, c.Sack}
	copy(keys[4:], c.Layers[:])
	for i, key := range keys {
		binary.LittleEndian.PutUint32(p[4+4*i:], key)
	}
	p[44], p[45], p[46], p[47] = c.Operation, c.Power, c.SourceX, c.SourceY
	p[48], p[49] = c.TargetX, c.TargetY
	binary.LittleEndian.PutUint16(p[50:], c.Residue32)
	return p
}

// DocumentCellFromPayload separates a cell key from its literal 52-byte image.
// This is a wire conversion, not a constructor or occupancy reconstruction.
func DocumentCellFromPayload(cell uint16, p [52]byte) DocumentCellData {
	c := DocumentCellData{Cell: cell, Cost: p[0], Static: p[1], LayerCount: p[2], Residue03: p[3],
		GroundActor: binary.LittleEndian.Uint32(p[4:]), AirActor: binary.LittleEndian.Uint32(p[8:]),
		Building: binary.LittleEndian.Uint32(p[12:]), Sack: binary.LittleEndian.Uint32(p[16:]),
		Operation: p[44], Power: p[45], SourceX: p[46], SourceY: p[47],
		TargetX: p[48], TargetY: p[49], Residue32: binary.LittleEndian.Uint16(p[50:])}
	for i := range c.Layers {
		c.Layers[i] = binary.LittleEndian.Uint32(p[20+4*i:])
	}
	return c
}
