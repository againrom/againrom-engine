package sim

import (
	"encoding/binary"
	"fmt"
)

// entityIDLimit is also the exhausted floor. The extra bit keeps max uint32
// distinct from an empty namespace without wrapping an EntityID.
const entityIDLimit uint64 = 1 << 32
const entityIDFloorLen = 8

// ReserveEntityIDs protects observable identity bindings, including absent
// actors retained by an older save's mission envelope. Repeated or reordered
// reservations are equivalent. This is native identity, not ROM1 provenance.
func (w *World) ReserveEntityIDs(ids []EntityID) {
	for _, id := range ids {
		w.entityIDFloor = max(w.entityIDFloor, uint64(id)+1)
	}
}

func (w *World) reserveObservableEntityIDs() {
	for _, e := range w.entities {
		w.entityIDFloor = max(w.entityIDFloor, uint64(e.ID)+1)
	}
	for _, d := range w.originalDead {
		w.entityIDFloor = max(w.entityIDFloor, uint64(d.ID)+1)
	}
	w.reserveScriptEntityIDs()
}

func (w *World) reserveScriptEntityIDs() {
	if w.script == nil {
		return
	}
	reserve := func(id EntityID, present bool) {
		if present {
			w.entityIDFloor = max(w.entityIDFloor, uint64(id)+1)
		}
	}
	for _, c := range w.script.checks {
		reserve(c.Unit, c.HasUnit)
		reserve(c.Unit2, c.HasUnit2)
	}
	for _, in := range w.script.instants {
		reserve(in.Unit, in.HasUnit)
		reserve(in.Unit2, in.HasUnit2)
	}
}

func (w *World) appendEntityIDFloor(data []byte) []byte {
	return binary.LittleEndian.AppendUint64(data, w.entityIDFloor)
}

func splitEntityIDFloor(data []byte) ([]byte, uint64, error) {
	if len(data) < headerLen+entityIDFloorLen {
		return nil, 0, fmt.Errorf("sim: truncated entity identity floor")
	}
	at := len(data) - entityIDFloorLen
	floor := binary.LittleEndian.Uint64(data[at:])
	if floor > entityIDLimit {
		return nil, 0, fmt.Errorf("sim: entity identity floor %d exceeds namespace", floor)
	}
	return data[:at], floor, nil
}
