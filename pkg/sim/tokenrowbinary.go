package sim

import (
	"encoding/binary"
	"fmt"
)

// tokenRowFormVersion carries the high byte of each token row above 255,
// which the one row byte of a source binding or a saved structure does not
// hold. The second game's Humans and Buildings have rows past 255
// (R2-ENGINE-739, R2-ENGINE-209). A form without it reads every row as its
// byte.
const tokenRowFormVersion byte = 130

// tokenRowPairLen is one wide row: the uint32 index and the high byte.
const tokenRowPairLen = 5

// tokenRowStructure marks a pair's index as a saved structure's; an index
// without it is an entity's. Entity pairs come first, so indices ascend.
const tokenRowStructure = 1 << 31

// appendTokenRows writes one pair per entity, then per saved structure, whose
// token row is above 255; a World with none writes no form.
func (w *World) appendTokenRows(b []byte) []byte {
	if !w.hasWideTokenRows() {
		return b
	}
	ws := beginWrapper(b)
	pair := func(index uint32, row uint16) {
		b = binary.LittleEndian.AppendUint32(b, index)
		b = append(b, byte(row>>8))
	}
	for i := range w.entities {
		if row := w.entities[i].SourceBinding.TokenRow; row > 0xff {
			pair(uint32(i), row)
		}
	}
	for i := range w.savedStructures {
		if row := w.savedStructures[i].Token0C; row > 0xff {
			pair(tokenRowStructure|uint32(i), row)
		}
	}
	return endWrapper(b, ws, tokenRowFormVersion)
}

func (w *World) hasWideTokenRows() bool {
	for i := range w.entities {
		if w.entities[i].SourceBinding.TokenRow > 0xff {
			return true
		}
	}
	for i := range w.savedStructures {
		if w.savedStructures[i].Token0C > 0xff {
			return true
		}
	}
	return false
}

func (w *World) unmarshalTokenRows(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed token row section") }
	env, ok := readWrapper(data, tokenRowFormVersion)
	if !ok {
		return fail()
	}
	s := env.section()
	if len(s) == 0 || len(s)%tokenRowPairLen != 0 {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	entities := append([]Entity(nil), next.entities...)
	structures := cloneSavedStructures(next.savedStructures)
	prior := int64(-1)
	for at := 0; at < len(s); at += tokenRowPairLen {
		index, high := binary.LittleEndian.Uint32(s[at:]), s[at+4]
		if int64(index) <= prior || high == 0 {
			return fail()
		}
		prior = int64(index)
		if index&tokenRowStructure != 0 {
			i := index &^ tokenRowStructure
			if uint64(i) >= uint64(len(structures)) {
				return fail()
			}
			structures[i].Token0C |= uint16(high) << 8
			continue
		}
		if uint64(index) >= uint64(len(entities)) || entities[index].SourceBinding.Class == 0 {
			return fail()
		}
		entities[index].SourceBinding.TokenRow |= uint16(high) << 8
	}
	next.entities, next.savedStructures = entities, structures
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
