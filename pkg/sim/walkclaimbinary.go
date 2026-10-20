package sim

import (
	"encoding/binary"
	"fmt"
)

const walkClaimFormVersion byte = 125
const walkClaimRecordLen = 9

// appendWalkClaims writes the walk claims of the Mover records the World
// keeps, one row per record whose claim is not zero; a World with none writes
// no form. A claim in an imported Mover travels with the imported motion.
func (w *World) appendWalkClaims(b []byte) []byte {
	ws := beginWrapper(b)
	for _, r := range w.movers {
		c := r.Claim
		if c == (walkClaim{}) || indexOfEntity(w.entities, r.Entity) < 0 || w.motionFor(r.Entity) != nil {
			continue
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(r.Entity))
		b = binary.LittleEndian.AppendUint16(b, c.Target)
		b = binary.LittleEndian.AppendUint16(b, c.Claim)
		b = append(b, layoutBool(c.Arrived))
	}
	if len(b) == ws.at {
		return b
	}
	return endWrapper(b, ws, walkClaimFormVersion)
}

func (w *World) unmarshalWalkClaims(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed walk claim section") }
	env, ok := readWrapper(data, walkClaimFormVersion)
	span := uint64(len(env.section()))
	if !ok || span == 0 || span%walkClaimRecordLen != 0 {
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
	for at := env.start; at < env.end; at += walkClaimRecordLen {
		id := EntityID(binary.LittleEndian.Uint32(data[at:]))
		r := next.moverFor(id)
		if r == nil || at > env.start && id <= previous || data[at+8] > 1 {
			return fail()
		}
		c := walkClaim{Target: binary.LittleEndian.Uint16(data[at+4:]), Claim: binary.LittleEndian.Uint16(data[at+6:]), Arrived: data[at+8] == 1}
		if c == (walkClaim{}) {
			return fail()
		}
		held := *r
		held.Claim = c
		next.setMover(held)
		previous = id
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
