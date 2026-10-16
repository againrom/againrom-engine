package sim

import (
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
	ws := beginWrapper(b)
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
	return endWrapper(b, ws, pendingOrderFormVersion)
}

func (w *World) unmarshalPendingOrders(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed pending order continuation") }
	env, ok := readWrapper(data, pendingOrderFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 24 {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || count > 65535 || span != 4+20*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
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
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
