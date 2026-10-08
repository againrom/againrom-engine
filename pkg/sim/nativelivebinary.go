package sim

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"fmt"
	"slices"
)

const nativeLiveFormVersion byte = 112
const nativeLiveRecordLen = 59

func (w *World) appendNativeLiveBlocks(b []byte) []byte {
	rows := slices.Clone(w.removedNativeBases)
	for _, e := range w.entities {
		rows = append(rows, NativeActorBasisRecord{ID: e.ID, Basis: e.NativeBasis})
	}
	rows = slices.DeleteFunc(rows, func(r NativeActorBasisRecord) bool { return !r.Basis.AttackPresent && !r.Basis.DefencePresent })
	if len(rows) == 0 {
		return b
	}
	slices.SortFunc(rows, func(a, b NativeActorBasisRecord) int { return cmp.Compare(a.ID, b.ID) })
	base, start := b[0], len(b)
	b[0] = nativeLiveFormVersion
	b = binary.LittleEndian.AppendUint32(b, uint32(len(rows)))
	for _, r := range rows {
		v, flag := r.Basis, byte(0)
		if v.AttackPresent {
			flag |= 1
		}
		if v.DefencePresent {
			flag |= 2
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(r.ID))
		b = append(b, flag)
		b = binary.LittleEndian.AppendUint32(b, v.AttackKnown)
		b = append(b, v.Attack[:]...)
		b = binary.LittleEndian.AppendUint32(b, v.DefenceKnown)
		b = append(b, v.Defence[:]...)
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'N', 'L', 'B', '1')
}

func (w *World) unmarshalNativeLiveBlocks(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed native live blocks") }
	if len(data) < headerLen+13+nativeLiveRecordLen || !bytes.Equal(data[len(data)-4:], []byte("NLB1")) {
		return fail()
	}
	baseVersion := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if baseVersion >= nativeLiveFormVersion || span < 4+nativeLiveRecordLen || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || count > 65535 || span != 4+nativeLiveRecordLen*count {
		return fail()
	}
	base := bytes.Clone(data[:start])
	base[0] = baseVersion
	next := *w
	if err := next.UnmarshalBinary(base); err != nil {
		return err
	}
	if count > uint64(len(next.entities)+len(next.originalDead)+len(next.currentTerminalActors)) {
		return fail()
	}
	rows := make([]NativeActorBasisRecord, int(count))
	for n := range rows {
		at := start + 4 + nativeLiveRecordLen*n
		id, flag := EntityID(binary.LittleEndian.Uint32(data[at:])), data[at+4]
		if flag == 0 || flag & ^byte(3) != 0 {
			return fail()
		}
		var v NativeActorBasis
		if i := indexOfEntity(next.entities, id); i >= 0 {
			v = next.entities[i].NativeBasis
		} else {
			for _, r := range next.removedNativeBases {
				if r.ID == id {
					v = r.Basis
				}
			}
		}
		v.AttackPresent, v.DefencePresent = flag&1 != 0, flag&2 != 0
		v.AttackKnown = binary.LittleEndian.Uint32(data[at+5:])
		copy(v.Attack[:], data[at+9:at+33])
		v.DefenceKnown = binary.LittleEndian.Uint32(data[at+33:])
		copy(v.Defence[:], data[at+37:at+59])
		rows[n] = NativeActorBasisRecord{ID: id, Basis: v}
	}
	if err := next.RestoreNativeActorBases(rows); err != nil {
		return fail()
	}
	canonical, err := next.MarshalBinary()
	if err != nil || !bytes.Equal(canonical, data) {
		return fail()
	}
	*w = next
	return nil
}
