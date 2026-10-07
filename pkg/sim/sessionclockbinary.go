package sim

import (
	"encoding/binary"
	"fmt"
)

// Form77 appends this fixed suffix AFTER the complete form76 payload, including
// its relation. No entity, source binding or variable-section offset changes.
const sessionClockLen = 5

func (w *World) sessionClockFault() error {
	if !w.hasSessionClock && w.fullTick != 0 {
		return fmt.Errorf("sim: absent session clock carries a full counter")
	}
	if w.hasSessionClock && w.tick > 0xffffffff {
		return fmt.Errorf("sim: source session subtick exceeds uint32")
	}
	return nil
}

func (w *World) appendSessionClock(b []byte) []byte {
	o := len(b)
	b = append(b, make([]byte, sessionClockLen)...)
	if w.hasSessionClock {
		b[o] = 1
	}
	binary.LittleEndian.PutUint32(b[o+1:], w.fullTick)
	return b
}

func splitSessionClock(data []byte) (payload []byte, present bool, full uint32, err error) {
	if len(data) < headerLen+sessionClockLen {
		return nil, false, 0, fmt.Errorf("sim: truncated session clock suffix")
	}
	o := len(data) - sessionClockLen
	if data[o] > 1 {
		return nil, false, 0, fmt.Errorf("sim: session clock presence %d is not 0 or 1", data[o])
	}
	full = binary.LittleEndian.Uint32(data[o+1:])
	present = data[o] == 1
	candidate := World{tick: binary.LittleEndian.Uint64(data[1:9]), hasSessionClock: present, fullTick: full}
	if err := candidate.sessionClockFault(); err != nil {
		return nil, false, 0, err
	}
	return data[:o], present, full, nil
}
