package sim

import (
	"encoding/binary"
	"fmt"
)

const nativeItemFormVersion byte = 113
const nativeItemRecordLen = 47

func (w *World) eachNativeItemRecord(visit func(uint32, uint16, SavedObjectID, **NativeItemRecord)) {
	var ordinal uint32
	item := func(i *ItemInstance) { visit(ordinal, i.Code, i.ObjectID, &i.NativeRecord); ordinal++ }
	for si := range w.sacks {
		for ii := range w.sacks[si].ItemInstances {
			item(&w.sacks[si].ItemInstances[ii])
		}
	}
	for ei := range w.carried {
		for si := range w.carried[ei] {
			s := &w.carried[ei][si]
			visit(ordinal, s.Code, s.ObjectID, &s.NativeRecord)
			ordinal++
		}
	}
	for ei := range w.equipment {
		for si := range w.equipment[ei] {
			item(&w.equipment[ei][si])
		}
	}
	for ci := range w.scrollCasts {
		item(&w.scrollCasts[ci].Item)
	}
}

func (w *World) nativeItemsFault() (fault error) {
	w.eachNativeItemRecord(func(_ uint32, code uint16, id SavedObjectID, v **NativeItemRecord) {
		if *v != nil && (code == 0 || id != 0 || (*v).Class > SourceShield || (*v).Token.T1C != 0) {
			fault = fmt.Errorf("sim: native Item record has no unregistered item owner")
		}
	})
	return
}

func (w *World) appendNativeItemRecords(b []byte) []byte {
	ws := beginWrapper(b)
	w.eachNativeItemRecord(func(o uint32, _ uint16, _ SavedObjectID, v **NativeItemRecord) {
		if *v == nil {
			return
		}
		b = binary.LittleEndian.AppendUint32(b, o)
		b, _ = binary.Append(b, binary.LittleEndian, **v)
	})
	if len(b) == ws.at {
		return b
	}
	return endWrapper(b, ws, nativeItemFormVersion)
}

func (w *World) unmarshalNativeItemRecords(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed native Item record span") }
	env, ok := readWrapper(data, nativeItemFormVersion)
	span := uint64(len(env.section()))
	if !ok || span == 0 || span%nativeItemRecordLen != 0 {
		return fail()
	}
	start := env.start
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	for at := start + nativeItemRecordLen; at < env.end; at += nativeItemRecordLen {
		if binary.LittleEndian.Uint32(data[at:]) <= binary.LittleEndian.Uint32(data[at-nativeItemRecordLen:]) {
			return fail()
		}
	}
	at := start
	next.eachNativeItemRecord(func(o uint32, code uint16, id SavedObjectID, v **NativeItemRecord) {
		if at == env.end || binary.LittleEndian.Uint32(data[at:]) != o {
			return
		}
		if code == 0 || id != 0 {
			return
		}
		var value NativeItemRecord
		_, _ = binary.Decode(data[at+4:at+nativeItemRecordLen], binary.LittleEndian, &value)
		*v = &value
		at += nativeItemRecordLen
	})
	if at != env.end {
		return fail()
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
