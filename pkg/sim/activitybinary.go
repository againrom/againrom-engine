package sim

import (
	"encoding/binary"
	"fmt"
)

// Form86 appends an ID-ordered sparse list of known action clocks. Records are
// two dwords (ID, deadline); a final dword counts payload bytes. An absent list
// leaves clocks Unknown so old LOAD does not fabricate an elapsed idle period.
const actionClockSpanLen = 4

type actionClockRecord struct{ id, end uint32 }

func (w *World) appendActionClocks(b []byte) []byte {
	start := len(b)
	for _, e := range w.entities {
		if e.ActionClock.Known {
			b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
			b = binary.LittleEndian.AppendUint32(b, e.ActionClock.End)
		}
	}
	return binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
}

func splitActionClocks(data []byte) ([]byte, []actionClockRecord, error) {
	if len(data) < headerLen+actionClockSpanLen {
		return nil, nil, fmt.Errorf("sim: truncated action clock footer")
	}
	end := len(data) - actionClockSpanLen
	span := uint64(binary.LittleEndian.Uint32(data[end:]))
	if span%8 != 0 || span > uint64(end-headerLen) || span/8 > uint64(binary.LittleEndian.Uint32(data[25:29])) {
		return nil, nil, fmt.Errorf("sim: action clock span exceeds actors or payload")
	}
	start := end - int(span)
	var records []actionClockRecord
	for p := start; p < end; p += 8 {
		r := actionClockRecord{binary.LittleEndian.Uint32(data[p:]), binary.LittleEndian.Uint32(data[p+4:])}
		if len(records) != 0 && records[len(records)-1].id >= r.id {
			return nil, nil, fmt.Errorf("sim: action clock IDs are not strictly ordered")
		}
		records = append(records, r)
	}
	return data[:start], records, nil
}

func applyActionClocks(ents []Entity, records []actionClockRecord) error {
	for _, r := range records {
		i := indexOfEntity(ents, EntityID(r.id))
		if i < 0 {
			return fmt.Errorf("sim: action clock names missing actor %d", r.id)
		}
		ents[i].ActionClock = ActionClock{Known: true, End: r.end}
	}
	return nil
}
