package sim

import (
	"encoding/binary"
	"fmt"
)

// pursuitSearchFormVersion wraps any earlier form with the held pursuit
// search records. A world holding none keeps its earlier bytes.
const pursuitSearchFormVersion byte = 117
const pursuitSearchRecordLen = 27

func (w *World) appendPursuitSearches(b []byte) []byte {
	count := 0
	for _, e := range w.entities {
		if e.Pursuit.Held {
			count++
		}
	}
	if count == 0 {
		return b
	}
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, e := range w.entities {
		p := e.Pursuit
		if !p.Held {
			continue
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
		b = binary.LittleEndian.AppendUint32(b, uint32(p.Victim))
		b = binary.LittleEndian.AppendUint16(b, p.Count)
		b = append(b, p.Passes)
		for _, v := range [4]int32{p.EndX, p.EndY, p.AimX, p.AimY} {
			b = binary.LittleEndian.AppendUint32(b, uint32(v))
		}
	}
	return endWrapper(b, ws, pursuitSearchFormVersion)
}

func (w *World) unmarshalPursuitSearches(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed pursuit search section") }
	env, ok := readWrapper(data, pursuitSearchFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 4+pursuitSearchRecordLen {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || span != 4+pursuitSearchRecordLen*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	if count > uint64(len(next.entities)) {
		return fail()
	}
	for n := 0; n < int(count); n++ {
		at := start + 4 + pursuitSearchRecordLen*n
		i := indexOfEntity(next.entities, EntityID(binary.LittleEndian.Uint32(data[at:])))
		if i < 0 || !next.entities[i].HasAttackTarget || next.entities[i].AttackTargetKind != AttackTargetUnit {
			return fail()
		}
		p := PursuitSearch{
			Held:   true,
			Victim: EntityID(binary.LittleEndian.Uint32(data[at+4:])),
			Count:  binary.LittleEndian.Uint16(data[at+8:]),
			Passes: data[at+10],
		}
		var v [4]int32
		for k := range v {
			v[k] = int32(binary.LittleEndian.Uint32(data[at+11+4*k:]))
		}
		p.EndX, p.EndY, p.AimX, p.AimY = v[0], v[1], v[2], v[3]
		if _, ok := cellIndexIn(next.bounds, p.EndX, p.EndY); !ok {
			return fail()
		}
		if _, ok := cellIndexIn(next.bounds, p.AimX, p.AimY); !ok {
			return fail()
		}
		next.entities[i].Pursuit = p
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
