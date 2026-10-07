package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// creatureSpellFormVersion marks a form that appends the class spell slots of
// every creature that carries any. A world with none writes no trailer, so its
// bytes and digest are the ones the base form gives.
const creatureSpellFormVersion byte = 103

const creatureSpellRecordLen = 4 + CreatureSpellSlots*8

func (w *World) appendCreatureSpells(b []byte) []byte {
	count := 0
	for _, e := range w.entities {
		if e.hasCreatureSpells() {
			count++
		}
	}
	if count == 0 {
		return b
	}
	base, start := b[0], len(b)
	b[0] = creatureSpellFormVersion
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, e := range w.entities {
		if !e.hasCreatureSpells() {
			continue
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
		for _, s := range e.CreatureSpells {
			b = binary.LittleEndian.AppendUint32(b, s.ID)
			b = binary.LittleEndian.AppendUint32(b, s.Threshold)
		}
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'C', 'S', 'P', '1')
}

func (w *World) unmarshalCreatureSpells(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed creature spell section") }
	if len(data) < headerLen+9+4+creatureSpellRecordLen || !bytes.Equal(data[len(data)-4:], []byte("CSP1")) {
		return fail()
	}
	baseVersion := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if baseVersion >= creatureSpellFormVersion || span < 4+creatureSpellRecordLen || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || count > 65535 || span != 4+creatureSpellRecordLen*count {
		return fail()
	}
	base := bytes.Clone(data[:start])
	base[0] = baseVersion
	next := *w
	if err := next.UnmarshalBinary(base); err != nil {
		return err
	}
	var prior EntityID
	for n := uint64(0); n < count; n++ {
		o := start + 4 + creatureSpellRecordLen*int(n)
		id := EntityID(binary.LittleEndian.Uint32(data[o:]))
		i := indexOfEntity(next.entities, id)
		if i < 0 || n > 0 && id <= prior {
			return fail()
		}
		prior = id
		var slots [CreatureSpellSlots]CreatureSpell
		for k := range slots {
			at := o + 4 + 8*k
			slots[k] = CreatureSpell{ID: binary.LittleEndian.Uint32(data[at:]), Threshold: binary.LittleEndian.Uint32(data[at+4:])}
		}
		next.entities[i].CreatureSpells = slots
		if !next.entities[i].hasCreatureSpells() {
			return fail()
		}
	}
	next.fillOrderSlotWindows()
	canonical, err := next.MarshalBinary()
	if err != nil || !bytes.Equal(canonical, data) {
		return fail()
	}
	*w = next
	return nil
}
