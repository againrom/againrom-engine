package sav

import (
	"bytes"
	"strings"
	"testing"
)

func TestCitySpellAliasUpdatesAreTransactionalAndIdentityBound(t *testing.T) {
	f, err := Open(cityTestSource(t))
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	// Independent literal Spell body. Both actors point to the same object,
	// not merely two instances with one ID.
	spell := &cityObject{sourceIndex: 200, class: "Spell", spell: &citySpell{fields: []byte{1, 99, 2, 255, 255, 0x30, 1, 0, 0}}}
	p.document.objects[200] = spell
	var updates []CityCharacterUpdate
	for _, c := range p.Roster() {
		u := p.document.objects[p.characterSourceIndex[c.Identity]].unit
		u.spellbookFlag, u.spellbookCount, u.spells = 1, 2, []*cityObject{spell}
		d, err := cityCharacterDescriptor(p.document.objects[p.characterSourceIndex[c.Identity]], c.Identity, c.Hero)
		if err != nil {
			t.Fatal(err)
		}
		d.Spells[0].Range = 6
		updates = append(updates, CityCharacterUpdate{Identity: c.Identity, Spells: &d.Spells})
	}
	before := append([]byte(nil), spell.spell.fields...)
	if err := p.ValidateSpellUpdates(updates[:1]); err == nil || !strings.Contains(err.Error(), "conflicting shared Spell") {
		t.Fatal("unmodified alias not checked", err)
	}
	(*updates[1].Spells)[0].Range = 7
	if err := p.ValidateSpellUpdates(updates); err == nil {
		t.Fatal("conflicting updates admitted")
	}
	(*updates[1].Spells)[0].Range = 6
	if err := p.ValidateSpellUpdates(updates); err != nil {
		t.Fatal("identical aliases refused", err)
	}
	values, err := p.spellUpdates(updates)
	if err != nil || values[200] != ([5]byte{1, 6, 2, 255, 255}) {
		t.Fatal(values, err)
	}
	(*updates[1].Spells)[0].Key++
	if err := p.ValidateSpellUpdates(updates); err == nil {
		t.Fatal("forged identity admitted")
	}
	if !bytes.Equal(before, spell.spell.fields) {
		t.Fatal("validation mutated provenance")
	}
}
