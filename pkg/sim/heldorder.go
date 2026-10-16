package sim

import (
	"encoding/binary"
	"fmt"
)

// HeldOrder kinds.
const (
	HeldOrderNone uint8 = iota
	// HeldOrderFrozen is a dead actor's last attack phase word and complete
	// flag: no order machine runs on it, so the words stay as loaded.
	HeldOrderFrozen
	// HeldOrderBody is a live actor's attack order on a body below the
	// targetable floor. The World drops that target, so the order words stay
	// as loaded until the World advances.
	HeldOrderBody
)

// HeldOrder is an original actor's order words the World carries but does
// not run: the phase word, target, countdown and complete flag of an actor
// order (HERO-CADENCE-023/112/114).
type HeldOrder struct {
	Kind                       uint8
	Target                     EntityID
	Phase, Countdown, Complete uint32
}

// HeldOrderRecord binds one held order to its actor.
type HeldOrderRecord struct {
	ID    EntityID
	Order HeldOrder
}

// heldOrder is the order the byte form carries: a frozen order ends when its
// actor lives again.
func heldOrder(e Entity) HeldOrder {
	if e.HeldOrder.Kind == HeldOrderFrozen && (e.Alive() || e.Dying()) {
		return HeldOrder{}
	}
	return e.HeldOrder
}

func heldOrderFault(e Entity) error {
	o := e.HeldOrder
	switch {
	case o.Kind == HeldOrderNone && o != (HeldOrder{}),
		o.Kind == HeldOrderFrozen && (o.Target != 0 || o.Countdown != 0 || o.Phase == 0 && o.Complete == 0 || e.Alive() || e.Dying()),
		o.Kind == HeldOrderBody && (o.Phase != 5 && o.Phase != 7 || !e.Alive() && !e.Dying()),
		o.Kind > HeldOrderBody:
		return fmt.Errorf("sim: invalid held order for actor %d", e.ID)
	}
	return nil
}

// RestoreHeldOrders sets the held orders of existing actors.
func (w *World) RestoreHeldOrders(rows []HeldOrderRecord) error {
	next := make([]Entity, len(w.entities))
	copy(next, w.entities)
	for _, row := range rows {
		i := indexOfEntity(next, row.ID)
		if i < 0 || row.Order.Kind == HeldOrderNone {
			return fmt.Errorf("sim: held order names no actor")
		}
		next[i].HeldOrder = row.Order
		if err := heldOrderFault(next[i]); err != nil {
			return err
		}
	}
	w.entities = next
	return nil
}

// releaseHeldOrders ends every held order the advancing World now runs: a
// body order at once, a frozen order when its actor lives again.
func (w *World) releaseHeldOrders() {
	for i := range w.entities {
		e := &w.entities[i]
		if e.HeldOrder.Kind == HeldOrderBody || e.HeldOrder.Kind == HeldOrderFrozen && (e.Alive() || e.Dying()) {
			e.HeldOrder = HeldOrder{}
		}
	}
}

const heldOrderFormVersion byte = 118
const heldOrderRecordLen = 21

func (w *World) appendHeldOrders(b []byte) []byte {
	count := 0
	for _, e := range w.entities {
		if heldOrder(e).Kind != HeldOrderNone {
			count++
		}
	}
	if count == 0 {
		return b
	}
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, e := range w.entities {
		o := heldOrder(e)
		if o.Kind == HeldOrderNone {
			continue
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
		b = append(b, o.Kind)
		b = binary.LittleEndian.AppendUint32(b, uint32(o.Target))
		b = binary.LittleEndian.AppendUint32(b, o.Phase)
		b = binary.LittleEndian.AppendUint32(b, o.Countdown)
		b = binary.LittleEndian.AppendUint32(b, o.Complete)
	}
	return endWrapper(b, ws, heldOrderFormVersion)
}

func (w *World) unmarshalHeldOrders(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed held order") }
	env, ok := readWrapper(data, heldOrderFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 4+heldOrderRecordLen {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || span != 4+heldOrderRecordLen*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	if count > uint64(len(next.entities)) {
		return fail()
	}
	last := -1
	for n := 0; n < int(count); n++ {
		at := start + 4 + heldOrderRecordLen*n
		i := indexOfEntity(next.entities, EntityID(binary.LittleEndian.Uint32(data[at:])))
		if i <= last || data[at+4] == HeldOrderNone {
			return fail()
		}
		e := &next.entities[i]
		e.HeldOrder = HeldOrder{Kind: data[at+4], Target: EntityID(binary.LittleEndian.Uint32(data[at+5:])),
			Phase: binary.LittleEndian.Uint32(data[at+9:]), Countdown: binary.LittleEndian.Uint32(data[at+13:]),
			Complete: binary.LittleEndian.Uint32(data[at+17:])}
		if err := heldOrderFault(*e); err != nil {
			return fail()
		}
		last = i
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
