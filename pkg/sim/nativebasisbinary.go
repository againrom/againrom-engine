package sim

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"fmt"
	"slices"
)

const nativeBasisFormVersion byte = 109
const nativeBasisRecordLen = 107

func (w *World) appendNativeActorBases(b []byte) []byte {
	rows := slices.Clone(w.removedNativeBases)
	rows = slices.DeleteFunc(rows, func(row NativeActorBasisRecord) bool { return !row.Basis.hasCoreValues() })
	for _, e := range w.entities {
		if e.NativeBasis.hasCoreValues() {
			rows = append(rows, NativeActorBasisRecord{ID: e.ID, Basis: e.NativeBasis})
		}
	}
	if len(rows) == 0 {
		return b
	}
	slices.SortFunc(rows, func(a, b NativeActorBasisRecord) int { return cmp.Compare(a.ID, b.ID) })
	base, start := b[0], len(b)
	b[0] = nativeBasisFormVersion
	b = binary.LittleEndian.AppendUint32(b, uint32(len(rows)))
	for _, row := range rows {
		v := row.Basis
		flag := byte(0)
		if v.BasePresent {
			flag |= 1
		}
		if v.ModifierPresent {
			flag |= 2
		}
		if v.BodyPresent {
			flag |= 4
		}
		if v.BodyKnown {
			flag |= 8
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(row.ID))
		b = append(b, flag)
		b = binary.LittleEndian.AppendUint32(b, v.BaseKnown)
		b = append(b, v.Base[:]...)
		b = binary.LittleEndian.AppendUint64(b, v.ModifierKnown)
		b = append(b, v.Modifier[:]...)
		b = binary.LittleEndian.AppendUint16(b, v.Body)
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'N', 'A', 'B', '1')
}

func (w *World) unmarshalNativeActorBases(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed native actor basis") }
	if len(data) < headerLen+13+nativeBasisRecordLen || !bytes.Equal(data[len(data)-4:], []byte("NAB1")) {
		return fail()
	}
	baseVersion := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if baseVersion >= nativeBasisFormVersion || span < 4+nativeBasisRecordLen || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || count > 65535 || span != 4+nativeBasisRecordLen*count {
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
		at := start + 4 + nativeBasisRecordLen*n
		flag := data[at+4]
		if flag == 0 || flag & ^byte(15) != 0 {
			return fail()
		}
		v := NativeActorBasis{BasePresent: flag&1 != 0, ModifierPresent: flag&2 != 0, BodyPresent: flag&4 != 0, BodyKnown: flag&8 != 0,
			BaseKnown: binary.LittleEndian.Uint32(data[at+5:]), ModifierKnown: binary.LittleEndian.Uint64(data[at+33:]),
			Body: binary.LittleEndian.Uint16(data[at+105:])}
		copy(v.Base[:], data[at+9:at+33])
		copy(v.Modifier[:], data[at+41:at+105])
		rows[n] = NativeActorBasisRecord{EntityID(binary.LittleEndian.Uint32(data[at:])), v}
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
