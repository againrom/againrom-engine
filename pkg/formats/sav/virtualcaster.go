package sav

import "fmt"

// VirtualCaster is one caster of the second game's list after the Sacks
// (R2-ENGINE-1011): the Token head's location and owner reference, the reach
// byte and the spell, power and cell of the six-byte block. OwnerKnown says the
// reference names a Player of this file; a caster whose reference names none
// carries no owner slot here.
type VirtualCaster struct {
	Reach, Spell, Power, X, Y uint8
	Cell                      uint16
	OwnerSlot                 uint32
	OwnerKnown                bool
}

// VirtualCasters reads the caster records of the list after the Sacks, in file
// order. A record of another class in that list has no reader and is left out
// (DIV-4172). A no-world file and a first-game file hold none.
func (f *File) VirtualCasters() ([]VirtualCaster, error) {
	doc, present, err := f.exactDocument()
	if err != nil || !present {
		return nil, err
	}
	var out []VirtualCaster
	for i, record := range doc.extra {
		if record.Class != "VirtualCaster" {
			continue
		}
		block, position := record.Raw["VC40"], record.Raw["Block12"]
		if len(block) != 6 || len(position) != 12 {
			return nil, fmt.Errorf("sav: VirtualCaster %d has no complete block or Position", i)
		}
		c := VirtualCaster{Reach: uint8(record.value("VC3C")), Spell: block[0], Power: block[1], X: block[2], Y: block[3], Cell: u16(position, 2)}
		reference := record.value("Reference")
		for _, player := range doc.players {
			if reference != 0 && player.value("This") == reference {
				c.OwnerSlot, c.OwnerKnown = player.value("Slot"), true
				break
			}
		}
		out = append(out, c)
	}
	return out, nil
}
