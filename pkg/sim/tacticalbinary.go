package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
)

const tacticalFormVersion byte = 101

func (w *World) defaultActorTraversal() []EntityID {
	var ids []EntityID
	for _, e := range w.entities {
		if !e.OffMap {
			ids = append(ids, e.ID)
		}
	}
	return ids
}

func (w *World) tacticalFault() error {
	if w.actorTraversal != nil {
		check := *w
		if err := check.RestoreActorTraversal(w.actorTraversal); err != nil {
			return err
		}
	}
	for _, e := range w.entities {
		r := e.Retreat
		if r == (RetreatContinuation{}) {
			continue
		}
		if !r.Known || r.Progress > 4 || e.ActorState != actorStateRetreat && e.PendingOrder.Kind == PendingNone ||
			r.Pending && (r.X < 0 || r.Y < 0 || r.X >= w.bounds.Width || r.Y >= w.bounds.Height) {
			return fmt.Errorf("sim: invalid Retreat continuation for actor %d", e.ID)
		}
	}
	return nil
}

func (w *World) appendTactical(b []byte) []byte {
	ids := w.actorTraversalIDs()
	traversal := !slices.Equal(ids, w.defaultActorTraversal())
	count := 0
	for _, e := range w.entities {
		if e.Retreat != (RetreatContinuation{}) {
			count++
		}
	}
	if !traversal && count == 0 {
		return b
	}
	base, start := b[0], len(b)
	b[0] = tacticalFormVersion
	if traversal {
		b = append(b, 1)
	} else {
		b = append(b, 0)
		ids = nil
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(ids)))
	for _, id := range ids {
		b = binary.LittleEndian.AppendUint32(b, uint32(id))
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, e := range w.entities {
		r := e.Retreat
		if r == (RetreatContinuation{}) {
			continue
		}
		flags := byte(1)
		if r.Pending {
			flags |= 2
		}
		if r.Failure {
			flags |= 4
		}
		if r.Complete {
			flags |= 8
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
		b = append(b, flags, r.Progress, r.Counter, 0)
		b = binary.LittleEndian.AppendUint32(b, uint32(r.X))
		b = binary.LittleEndian.AppendUint32(b, uint32(r.Y))
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'T', 'A', 'C', '1')
}

func (w *World) unmarshalTactical(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed tactical continuation") }
	if len(data) < headerLen+18 || !bytes.Equal(data[len(data)-4:], []byte("TAC1")) {
		return fail()
	}
	baseVersion := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if baseVersion == tacticalFormVersion || span < 9 || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	payload := data[start : len(data)-9]
	if payload[0] > 1 {
		return fail()
	}
	count := uint64(binary.LittleEndian.Uint32(payload[1:]))
	if count > 65535 || 9+4*count > span || payload[0] == 0 && count != 0 {
		return fail()
	}
	retreatAt := 5 + 4*int(count)
	retreats := uint64(binary.LittleEndian.Uint32(payload[retreatAt:]))
	if retreats > 65535 || 9+4*count+16*retreats != span {
		return fail()
	}
	base := bytes.Clone(data[:start])
	base[0] = baseVersion
	next := *w
	if err := next.UnmarshalBinary(base); err != nil {
		return err
	}
	if payload[0] == 1 {
		ids := make([]EntityID, count)
		for i := range ids {
			ids[i] = EntityID(binary.LittleEndian.Uint32(payload[5+4*i:]))
		}
		if err := next.RestoreActorTraversal(ids); err != nil {
			return fail()
		}
	}
	var prior EntityID
	for i := uint64(0); i < retreats; i++ {
		o := retreatAt + 4 + 16*int(i)
		id := EntityID(binary.LittleEndian.Uint32(payload[o:]))
		flags := payload[o+4]
		index := indexOfEntity(next.entities, id)
		if index < 0 || i > 0 && id <= prior || flags&1 == 0 || flags > 15 || payload[o+7] != 0 {
			return fail()
		}
		prior = id
		next.entities[index].Retreat = RetreatContinuation{Known: true, Pending: flags&2 != 0, Failure: flags&4 != 0, Complete: flags&8 != 0,
			Progress: payload[o+5], Counter: payload[o+6], X: int32(binary.LittleEndian.Uint32(payload[o+8:])), Y: int32(binary.LittleEndian.Uint32(payload[o+12:]))}
	}
	if err := next.tacticalFault(); err != nil {
		return fail()
	}
	canonical, err := next.MarshalBinary()
	if err != nil || !bytes.Equal(canonical, data) {
		return fail()
	}
	*w = next
	return nil
}
