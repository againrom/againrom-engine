package sim

import (
	"cmp"
	"encoding/binary"
	"fmt"
	"slices"
)

const nativeScalarFormVersion byte = 114
const nativeScalarRecordLen = 105

func (w *World) appendNativeScalars(b []byte) []byte {
	rows := w.RemovedNativeActorBases()
	for _, e := range w.entities {
		rows = append(rows, NativeActorBasisRecord{ID: e.ID, Basis: w.nativeBasisNow(e)})
	}
	rows = slices.DeleteFunc(rows, func(r NativeActorBasisRecord) bool { return !r.Basis.hasScalarValues() })
	if len(rows) == 0 {
		return b
	}
	slices.SortFunc(rows, func(a, b NativeActorBasisRecord) int { return cmp.Compare(a.ID, b.ID) })
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(rows)))
	for _, row := range rows {
		v, flag := row.Basis, byte(0)
		if v.ScalarsPresent {
			flag |= 1
		}
		if v.BlockPresent {
			flag |= 2
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(row.ID))
		b = append(b, flag)
		b = binary.LittleEndian.AppendUint32(b, v.ScalarKnown)
		for _, scalar := range v.Scalars {
			b = binary.LittleEndian.AppendUint32(b, scalar)
		}
		b = binary.LittleEndian.AppendUint16(b, v.BlockKnown)
		b = append(b, v.Block[:]...)
	}
	return endWrapper(b, ws, nativeScalarFormVersion)
}

func (w *World) unmarshalNativeScalars(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed native scalar section") }
	env, ok := readWrapper(data, nativeScalarFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 4+nativeScalarRecordLen {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || count > 65535 || span != 4+nativeScalarRecordLen*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	if count > uint64(len(next.entities)+len(next.originalDead)+len(next.currentTerminalActors)) {
		return fail()
	}
	rows := make([]NativeActorBasisRecord, int(count))
	for n := range rows {
		at := start + 4 + nativeScalarRecordLen*n
		id, flag := EntityID(binary.LittleEndian.Uint32(data[at:])), data[at+4]
		if flag == 0 || flag & ^byte(3) != 0 {
			return fail()
		}
		var v NativeActorBasis
		if i := indexOfEntity(next.entities, id); i >= 0 {
			v = next.entities[i].NativeBasis
		} else {
			for _, row := range next.removedNativeBases {
				if row.ID == id {
					v = row.Basis
				}
			}
		}
		v.ScalarsPresent, v.BlockPresent = flag&1 != 0, flag&2 != 0
		v.ScalarKnown = binary.LittleEndian.Uint32(data[at+5:])
		for slot := range v.Scalars {
			v.Scalars[slot] = binary.LittleEndian.Uint32(data[at+9+4*slot:])
		}
		v.BlockKnown = binary.LittleEndian.Uint16(data[at+93:])
		copy(v.Block[:], data[at+95:at+105])
		rows[n] = NativeActorBasisRecord{ID: id, Basis: v}
	}
	if err := next.RestoreNativeActorBases(rows); err != nil {
		return fail()
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
