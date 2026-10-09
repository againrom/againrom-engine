package sim

import (
	"bytes"
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
	base, start := b[0], len(b)
	b[0] = turnStateFormVersion
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
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'T', 'R', 'N', '1')
}

func (w *World) unmarshalTurnStates(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed turn state") }
	if len(data) < headerLen+13+turnStateRecordLen || !bytes.Equal(data[len(data)-4:], []byte("TRN1")) {
		return fail()
	}
	version := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if version >= turnStateFormVersion || span < 4+turnStateRecordLen || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || span != 4+turnStateRecordLen*count {
		return fail()
	}
	base := bytes.Clone(data[:start])
	base[0] = version
	next := *w
	if err := next.UnmarshalBinary(base); err != nil {
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
	canonical, err := next.MarshalBinary()
	if err != nil || !bytes.Equal(canonical, data) {
		return fail()
	}
	*w = next
	return nil
}
