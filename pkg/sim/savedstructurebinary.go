package sim

import (
	"encoding/binary"
	"fmt"
)

// Form79 appends one bounded payload and its uint32 byte span after the complete
// form78 Group footer. Span zero is absent legacy mode; present empty is 8 bytes
// (two zero counts). The existing 22-byte live Structure record does not move.
const savedStructureRecordMin = 107
const savedStructureCellLen = 9

func (w *World) appendSavedStructureSection(b []byte) []byte {
	start := len(b)
	if w.hasSavedStructures {
		b = binary.LittleEndian.AppendUint32(b, uint32(len(w.savedStructures)))
		for _, s := range w.savedStructures {
			b = binary.LittleEndian.AppendUint32(b, uint32(s.ID))
			b = append(b, byte(s.Class))
			b = binary.LittleEndian.AppendUint32(b, s.SourceKey)
			b = binary.LittleEndian.AppendUint16(b, s.ArchiveIndex)
			b = binary.LittleEndian.AppendUint32(b, s.AuthoredID)
			b = binary.LittleEndian.AppendUint32(b, s.AuthoredIndex)
			flag := byte(0)
			if s.HasAuthored {
				flag = 1
			}
			b = append(b, flag)
			b = append(b, s.Position[:]...)
			b = binary.LittleEndian.AppendUint32(b, s.RuntimeID)
			b = append(b, s.Token0C)
			b = binary.LittleEndian.AppendUint16(b, s.Token0E)
			b = binary.LittleEndian.AppendUint16(b, s.Token18)
			b = binary.LittleEndian.AppendUint32(b, s.Token1C)
			b = binary.LittleEndian.AppendUint32(b, s.Reference)
			b = append(b, s.Base52[:]...)
			b = append(b, s.Kind)
			b = binary.LittleEndian.AppendUint16(b, s.Field46)
			b = append(b, s.Field48)
			b = binary.LittleEndian.AppendUint32(b, s.Blocking)
			b = binary.LittleEndian.AppendUint32(b, s.Tavern9C)
			b = binary.LittleEndian.AppendUint32(b, s.Shop70)
			for _, word := range s.OutpostWords {
				b = binary.LittleEndian.AppendUint32(b, word)
			}
			b = binary.LittleEndian.AppendUint32(b, uint32(len(s.OutpostRecords)))
			for _, row := range s.OutpostRecords {
				b = append(b, row[:]...)
			}
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(len(w.savedStructureCells)))
		for _, c := range w.savedStructureCells {
			b = binary.LittleEndian.AppendUint16(b, c.Cell)
			b = append(b, c.BaselineCost, c.BaselineStatic)
			b = binary.LittleEndian.AppendUint32(b, uint32(c.ID))
			flag := byte(0)
			if c.HasStructure {
				flag = 1
			}
			b = append(b, flag)
		}
	}
	return binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
}

func splitSavedStructureSection(data []byte) (body []byte, present bool, source []SavedStructure, cells []SavedStructureCell, err error) {
	if len(data) < headerLen+4 {
		return nil, false, nil, nil, fmt.Errorf("saved structures: truncated footer")
	}
	n := uint64(binary.LittleEndian.Uint32(data[len(data)-4:]))
	if n > uint64(len(data)-headerLen-4) {
		return nil, false, nil, nil, fmt.Errorf("saved structures: payload span overruns form")
	}
	end := len(data) - 4
	start := end - int(n)
	body = data[:start]
	if n == 0 {
		return body, false, nil, nil, nil
	}
	r := savedStructureReader{data: data[start:end]}
	count := r.u32()
	if uint64(count)*savedStructureRecordMin > uint64(len(r.data)) {
		return nil, false, nil, nil, fmt.Errorf("saved structures: source count overruns payload")
	}
	source = make([]SavedStructure, count)
	for i := range source {
		s := &source[i]
		s.ID = StructureID(r.u32())
		s.Class = SavedStructureClass(r.u8())
		s.SourceKey, s.ArchiveIndex = r.u32(), r.u16()
		s.AuthoredID, s.AuthoredIndex, s.HasAuthored = r.u32(), r.u32(), r.flag()
		copy(s.Position[:], r.take(12))
		s.RuntimeID, s.Token0C, s.Token0E, s.Token18 = r.u32(), r.u8(), r.u16(), r.u16()
		s.Token1C, s.Reference = r.u32(), r.u32()
		copy(s.Base52[:], r.take(22))
		s.Kind, s.Field46, s.Field48 = r.u8(), r.u16(), r.u8()
		s.Blocking, s.Tavern9C, s.Shop70 = r.u32(), r.u32(), r.u32()
		for j := range s.OutpostWords {
			s.OutpostWords[j] = r.u32()
		}
		rows := r.u32()
		if uint64(rows)*8 > uint64(len(r.data)) {
			return nil, false, nil, nil, fmt.Errorf("saved structures: Outpost count overruns payload")
		}
		if rows != 0 {
			s.OutpostRecords = make([][8]byte, rows)
			for j := range s.OutpostRecords {
				copy(s.OutpostRecords[j][:], r.take(8))
			}
		}
	}
	count = r.u32()
	if uint64(count)*savedStructureCellLen > uint64(len(r.data)) {
		return nil, false, nil, nil, fmt.Errorf("saved structures: cell count overruns payload")
	}
	cells = make([]SavedStructureCell, count)
	for i := range cells {
		cells[i] = SavedStructureCell{Cell: r.u16(), BaselineCost: r.u8(), BaselineStatic: r.u8(), ID: StructureID(r.u32()), HasStructure: r.flag()}
	}
	if r.err != nil {
		return nil, false, nil, nil, r.err
	}
	if len(r.data) != 0 {
		return nil, false, nil, nil, fmt.Errorf("saved structures: trailing payload bytes")
	}
	return body, true, source, cells, nil
}

type savedStructureReader struct {
	data []byte
	err  error
}

func (r *savedStructureReader) take(n int) []byte {
	if len(r.data) < n {
		r.err = fmt.Errorf("saved structures: truncated payload")
		return nil
	}
	b := r.data[:n]
	r.data = r.data[n:]
	return b
}
func (r *savedStructureReader) u8() byte {
	b := r.take(1)
	if len(b) == 0 {
		return 0
	}
	return b[0]
}
func (r *savedStructureReader) u16() uint16 {
	b := r.take(2)
	if len(b) == 0 {
		return 0
	}
	return binary.LittleEndian.Uint16(b)
}
func (r *savedStructureReader) u32() uint32 {
	b := r.take(4)
	if len(b) == 0 {
		return 0
	}
	return binary.LittleEndian.Uint32(b)
}
func (r *savedStructureReader) flag() bool {
	b := r.u8()
	if b > 1 {
		r.err = fmt.Errorf("saved structures: noncanonical presence byte")
	}
	return b == 1
}
