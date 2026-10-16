package sim

import (
	"encoding/binary"
	"fmt"
)

const turnStateFormVersion byte = 115
const turnStateRecordLen = 9

func (w *World) appendTurnStates(b []byte) []byte {
	count := 0
	for _, e := range w.entities {
		if e.TurnState.Present {
			count++
		}
	}
	if count == 0 {
		return b
	}
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, e := range w.entities {
		s := e.TurnState
		if !s.Present {
			continue
		}
		flag := byte(0)
		if s.Active {
			flag = 1
		}
		if s.DrawComplete {
			flag |= 2
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
		b = append(b, flag, s.Counter, s.Drawn, s.DrawTarget, s.DrawRemaining)
	}
	return endWrapper(b, ws, turnStateFormVersion)
}

func (w *World) unmarshalTurnStates(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed turn state") }
	env, ok := readWrapper(data, turnStateFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 4+turnStateRecordLen {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || span != 4+turnStateRecordLen*count {
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
		at := start + 4 + turnStateRecordLen*n
		i := indexOfEntity(next.entities, EntityID(binary.LittleEndian.Uint32(data[at:])))
		if i <= last || data[at+4] > 3 {
			return fail()
		}
		e := &next.entities[i]
		e.TurnState = TurnState{Present: true, Active: data[at+4]&1 != 0, DrawComplete: data[at+4]&2 != 0,
			Counter: data[at+5], Drawn: data[at+6], DrawTarget: data[at+7], DrawRemaining: data[at+8]}
		if err := turnFault(*e); err != nil {
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
