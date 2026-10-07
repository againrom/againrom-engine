package sim

import (
	"encoding/binary"
	"fmt"
)

type attackNoticeRecord struct {
	id     EntityID
	notice attackNotice
}

func (w *World) appendAttackNotices(b []byte) []byte {
	start := len(b)
	for _, e := range w.entities {
		n := e.attackNotice
		if n == (attackNotice{}) || w.savedOrder(e.ID) != nil {
			continue
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
		b = binary.LittleEndian.AppendUint16(b, n.Cell)
		b = append(b, n.Scans)
	}
	return binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
}

func splitAttackNotices(data []byte) ([]byte, []attackNoticeRecord, error) {
	if len(data) < headerLen+4 {
		return nil, nil, fmt.Errorf("sim: truncated attack notice footer")
	}
	end := len(data) - 4
	span := uint64(binary.LittleEndian.Uint32(data[end:]))
	if span%7 != 0 || span > uint64(end-headerLen) || span/7 > uint64(binary.LittleEndian.Uint32(data[25:])) {
		return nil, nil, fmt.Errorf("sim: attack notice span exceeds actors or payload")
	}
	start := end - int(span)
	var records []attackNoticeRecord
	for p := start; p < end; p += 7 {
		r := attackNoticeRecord{EntityID(binary.LittleEndian.Uint32(data[p:])), attackNotice{binary.LittleEndian.Uint16(data[p+4:]), data[p+6]}}
		if r.notice == (attackNotice{}) || len(records) != 0 && records[len(records)-1].id >= r.id {
			return nil, nil, fmt.Errorf("sim: attack notices are empty or not strictly ordered")
		}
		records = append(records, r)
	}
	return data[:start], records, nil
}

func (w *World) applyAttackNotices(records []attackNoticeRecord) error {
	for _, r := range records {
		i := indexOfEntity(w.entities, r.id)
		if i < 0 || w.savedOrder(r.id) != nil {
			return fmt.Errorf("sim: attack notice actor %d is absent or already owns an order", r.id)
		}
		w.entities[i].attackNotice = r.notice
	}
	return nil
}
