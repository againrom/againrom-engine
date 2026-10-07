package sim

import (
	"encoding/binary"
	"fmt"
)

// Form81 appends a sparse, ID-ordered stride section after complete form80.
// Zero span means every actor lacks provenance. A populated payload is a u32
// count followed by 24-byte records: ID, four signed coordinates, rate, signed
// X/Y axis steps and octant. No count allocates before its byte span is checked.
const nativeStrideSpanLen = 4
const nativeStrideRecordLen = 24

type nativeStrideRecord struct {
	id EntityID
	s  NativeStride
}

func (w *World) appendNativeStrides(b []byte) []byte {
	start := len(b)
	var count uint32
	for _, e := range w.entities {
		if e.Stride.Present {
			count++
		}
	}
	if count != 0 {
		b = binary.LittleEndian.AppendUint32(b, count)
		for _, e := range w.entities {
			s := e.Stride
			if !s.Present {
				continue
			}
			b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
			b = binary.LittleEndian.AppendUint32(b, uint32(s.FromX))
			b = binary.LittleEndian.AppendUint32(b, uint32(s.FromY))
			b = binary.LittleEndian.AppendUint32(b, uint32(s.ToX))
			b = binary.LittleEndian.AppendUint32(b, uint32(s.ToY))
			b = append(b, s.Rate, byte(s.StepX), byte(s.StepY), s.Direction)
		}
	}
	return binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
}

func splitNativeStrides(data []byte) ([]byte, []nativeStrideRecord, error) {
	if len(data) < headerLen+nativeStrideSpanLen {
		return nil, nil, fmt.Errorf("sim: truncated native stride footer")
	}
	end := len(data) - nativeStrideSpanLen
	span := uint64(binary.LittleEndian.Uint32(data[end:]))
	if span > uint64(end-headerLen) {
		return nil, nil, fmt.Errorf("sim: native stride span exceeds payload")
	}
	if span == 0 {
		return data[:end], nil, nil
	}
	start := end - int(span)
	payload := data[start:end]
	if len(payload) < 4 {
		return nil, nil, fmt.Errorf("sim: truncated native stride count")
	}
	count := binary.LittleEndian.Uint32(payload)
	if count == 0 || count > binary.LittleEndian.Uint32(data[25:29]) || uint64(count)*nativeStrideRecordLen+4 != span {
		return nil, nil, fmt.Errorf("sim: native stride count does not match actor count or payload")
	}
	out := make([]nativeStrideRecord, count)
	payload = payload[4:]
	for i := range out {
		r := nativeStrideRecord{id: EntityID(binary.LittleEndian.Uint32(payload)), s: NativeStride{
			Present: true, FromX: int32(binary.LittleEndian.Uint32(payload[4:])),
			FromY: int32(binary.LittleEndian.Uint32(payload[8:])), ToX: int32(binary.LittleEndian.Uint32(payload[12:])),
			ToY: int32(binary.LittleEndian.Uint32(payload[16:])), Rate: payload[20],
			StepX: int8(payload[21]), StepY: int8(payload[22]), Direction: payload[23],
		}}
		if i > 0 && out[i-1].id >= r.id {
			return nil, nil, fmt.Errorf("sim: native stride actor IDs are not strictly ordered")
		}
		out[i] = r
		payload = payload[nativeStrideRecordLen:]
	}
	return data[:start], out, nil
}

func applyNativeStrides(ents []Entity, records []nativeStrideRecord) error {
	for _, r := range records {
		i := indexOfEntity(ents, r.id)
		if i < 0 {
			return fmt.Errorf("sim: native stride names missing actor %d", r.id)
		}
		ents[i].Stride = r.s
		if err := strideFault(ents[i]); err != nil {
			return fmt.Errorf("sim: actor %d: %w", r.id, err)
		}
	}
	return nil
}
