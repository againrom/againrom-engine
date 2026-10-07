package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const pendingOrderFormVersion byte = 102

func (w *World) appendPendingOrders(b []byte) []byte {
	count := 0
	for _, e := range w.entities {
		if e.PendingOrder.Kind != PendingNone {
			count++
		}
	}
	if count == 0 {
		return b
	}
	base, start := b[0], len(b)
	b[0] = pendingOrderFormVersion
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, e := range w.entities {
		p := e.PendingOrder
		if p.Kind == PendingNone {
			continue
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
		state := byte(0)
		if e.Retreat.Known && e.ActorState != actorStateRetreat {
			state = e.ActorState + 1
		}
		if p.RowAdmitted {
			state |= 0x80
		}
		b = append(b, p.Kind, state)
		b = binary.LittleEndian.AppendUint16(b, p.Spell)
		b = binary.LittleEndian.AppendUint32(b, uint32(p.Target))
		b = binary.LittleEndian.AppendUint32(b, uint32(p.X))
		b = binary.LittleEndian.AppendUint32(b, uint32(p.Y))
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'O', 'R', 'D', '1')
}

func (w *World) unmarshalPendingOrders(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed pending order continuation") }
	if len(data) < headerLen+33 || !bytes.Equal(data[len(data)-4:], []byte("ORD1")) {
		return fail()
	}
	baseVersion := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if baseVersion >= pendingOrderFormVersion || span < 24 || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || count > 65535 || span != 4+20*count {
		return fail()
	}
	base := bytes.Clone(data[:start])
	base[0] = baseVersion
	next := *w
	if err := next.UnmarshalBinary(base); err != nil {
		return err
	}
	var prior EntityID
	for n := uint64(0); n < count; n++ {
		o := start + 4 + 20*int(n)
		id := EntityID(binary.LittleEndian.Uint32(data[o:]))
		i := indexOfEntity(next.entities, id)
		if i < 0 || n > 0 && id <= prior || data[o+4] == PendingNone {
			return fail()
		}
		prior = id
		p := PendingOrder{Kind: data[o+4], RowAdmitted: data[o+5]&0x80 != 0, Spell: binary.LittleEndian.Uint16(data[o+6:]), Target: EntityID(binary.LittleEndian.Uint32(data[o+8:])), X: int32(binary.LittleEndian.Uint32(data[o+12:])), Y: int32(binary.LittleEndian.Uint32(data[o+16:]))}
		next.entities[i].PendingOrder = p
		if state := data[o+5] & 0x7f; state != 0 {
			if !next.entities[i].Retreat.Known || !actorStateDefined(state-1) {
				return fail()
			}
			next.entities[i].ActorState = state - 1
		}
		if p.Kind == PendingPickupComplete {
			next.entities[i].ActorState = actorStatePickupComplete
		}
	}
	if err := next.pendingOrderFault(); err != nil {
		return fail()
	}
	canonical, err := next.MarshalBinary()
	if err != nil || !bytes.Equal(canonical, data) {
		return fail()
	}
	*w = next
	return nil
}
