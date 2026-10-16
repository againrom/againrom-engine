package sim

import (
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
	ws := beginWrapper(b)
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
	return endWrapper(b, ws, creatureSpellFormVersion)
}

func (w *World) unmarshalCreatureSpells(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed creature spell section") }
	env, ok := readWrapper(data, creatureSpellFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 4+creatureSpellRecordLen {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || count > 65535 || span != 4+creatureSpellRecordLen*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
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
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
