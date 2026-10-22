package sim

import (
	"encoding/binary"
	"fmt"
)

const moverRadiusFormVersion byte = 126
const moverRadiusRecordLen = 5

// appendMoverRadii writes the search radius of each Mover record the World
// keeps whose radius is not the constructor's, one row per record; a World
// with none writes no form. A radius in an imported Mover travels with the
// imported motion.
func (w *World) appendMoverRadii(b []byte) []byte {
	ws := beginWrapper(b)
	for _, r := range w.movers {
		if r.Radius == ConstructorSearchRadius || indexOfEntity(w.entities, r.Entity) < 0 || w.motionFor(r.Entity) != nil {
			continue
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(r.Entity))
		b = append(b, r.Radius)
	}
	if len(b) == ws.at {
		return b
	}
	return endWrapper(b, ws, moverRadiusFormVersion)
}

func (w *World) unmarshalMoverRadii(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed mover radius section") }
	env, ok := readWrapper(data, moverRadiusFormVersion)
	span := uint64(len(env.section()))
	if !ok || span == 0 || span%moverRadiusRecordLen != 0 {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	if next.SecondGame() {
		return fail()
	}
	next.constructMovers()
	var previous EntityID
	for at := env.start; at < env.end; at += moverRadiusRecordLen {
		id := EntityID(binary.LittleEndian.Uint32(data[at:]))
		r := next.moverFor(id)
		if r == nil || at > env.start && id <= previous || data[at+4] == ConstructorSearchRadius {
			return fail()
		}
		held := *r
		held.Radius = data[at+4]
		next.setMover(held)
		previous = id
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
