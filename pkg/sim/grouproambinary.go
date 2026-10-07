package sim

import (
	"encoding/binary"
	"fmt"
)

type groupRoamRecord struct {
	owner, group uint32
	counter      uint8
}

func (w *World) appendGroupRoam(b []byte) []byte {
	start := len(b)
	for _, g := range w.groups {
		if g.roamCounter == 0 {
			continue
		}
		b = binary.LittleEndian.AppendUint32(b, g.owner)
		b = binary.LittleEndian.AppendUint32(b, g.group)
		b = append(b, g.roamCounter)
	}
	return binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
}

func splitGroupRoam(data []byte) ([]byte, []groupRoamRecord, error) {
	if len(data) < headerLen+4 {
		return nil, nil, fmt.Errorf("sim: truncated Group counter footer")
	}
	end := len(data) - 4
	span := uint64(binary.LittleEndian.Uint32(data[end:]))
	if span%9 != 0 || span > uint64(end-headerLen) {
		return nil, nil, fmt.Errorf("sim: invalid Group counter span")
	}
	start := end - int(span)
	var records []groupRoamRecord
	for at := start; at < end; at += 9 {
		r := groupRoamRecord{binary.LittleEndian.Uint32(data[at:]), binary.LittleEndian.Uint32(data[at+4:]), data[at+8]}
		if r.counter == 0 {
			return nil, nil, fmt.Errorf("sim: redundant zero Group counter")
		}
		if len(records) > 0 {
			last := records[len(records)-1]
			if last.owner > r.owner || last.owner == r.owner && last.group >= r.group {
				return nil, nil, fmt.Errorf("sim: Group counters are not strictly ordered")
			}
		}
		records = append(records, r)
	}
	return data[:start], records, nil
}

func applyGroupRoam(groups []groupAI, records []groupRoamRecord) error {
	at := 0
	for _, r := range records {
		for at < len(groups) && (groups[at].owner < r.owner || groups[at].owner == r.owner && groups[at].group < r.group) {
			at++
		}
		if at == len(groups) || groups[at].owner != r.owner || groups[at].group != r.group {
			return fmt.Errorf("sim: counter names missing Group %d/%d", r.owner, r.group)
		}
		groups[at].roamCounter = r.counter
	}
	return nil
}
