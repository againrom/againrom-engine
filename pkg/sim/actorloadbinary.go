package sim

import (
	"encoding/binary"
	"fmt"
)

const sourceActorLen = 209
const actorLoadRecordLen = 15 + sourceActorLen

// Form 75's span high bit marks an optional source-equipment prefix: u32
// byte count, then ascending ordinal + 58-byte operand records. The remaining
// span holds 220-byte actor records. No item prefix is written when empty,
// so every older native upgrade still supplies only an empty four-byte span.

func (w *World) actorLoadSectionLen() int {
	n := 4 + w.sourceEquipmentSectionLen()
	for i := range w.entities {
		if w.entities[i].ActorLoad.Present {
			n += actorLoadRecordLen
		}
	}
	return n
}

func (w *World) encodeActorLoads(dst []byte, off int) int {
	start := off
	itemsLen := w.sourceEquipmentSectionLen()
	if itemsLen != 0 {
		binary.LittleEndian.PutUint32(dst[off:], uint32(itemsLen-4))
		off = w.encodeSourceEquipment(dst, off+4)
	}
	for i := range w.entities {
		e := &w.entities[i]
		a := &e.ActorLoad
		if !a.Present {
			continue
		}
		binary.LittleEndian.PutUint32(dst[off:], uint32(e.ID))
		binary.LittleEndian.PutUint16(dst[off+4:], uint16(a.OwnWeight))
		if a.ContainerPresent {
			dst[off+6] = 1
		}
		binary.LittleEndian.PutUint32(dst[off+7:], a.InsertIndex)
		binary.LittleEndian.PutUint32(dst[off+11:], uint32(a.Accumulator))
		putSourceActor(dst[off+15:off+actorLoadRecordLen], &a.Source)
		off += actorLoadRecordLen
	}
	span := uint32(off - start)
	if itemsLen != 0 {
		span |= 1 << 31
	}
	binary.LittleEndian.PutUint32(dst[off:], span)
	return off + 4
}

func decodeActorLoads(data []byte, entities []Entity) error {
	if len(data)%actorLoadRecordLen != 0 || len(data)/actorLoadRecordLen > len(entities) {
		return fmt.Errorf("sim: invalid actor-load record span")
	}
	var previous EntityID
	for off := 0; off < len(data); off += actorLoadRecordLen {
		id := EntityID(binary.LittleEndian.Uint32(data[off:]))
		i := indexOfEntity(entities, id)
		if i < 0 || off != 0 && id <= previous || data[off+6] > 1 {
			return fmt.Errorf("sim: invalid actor-load identity/presence")
		}
		previous = id
		for _, flag := range []byte{data[off+15+198], data[off+15+199], data[off+15+200], data[off+15+208]} {
			if flag > 1 {
				return fmt.Errorf("sim: noncanonical source actor boolean")
			}
		}
		entities[i].ActorLoad = ActorLoad{Present: true, OwnWeight: int16(binary.LittleEndian.Uint16(data[off+4:])),
			ContainerPresent: data[off+6] != 0, InsertIndex: binary.LittleEndian.Uint32(data[off+7:]), Accumulator: int32(binary.LittleEndian.Uint32(data[off+11:]))}
		getSourceActor(data[off+15:off+actorLoadRecordLen], &entities[i].ActorLoad.Source)
		if err := entities[i].CurrentActorLoad().Validate(); err != nil {
			return err
		}
	}
	return nil
}
