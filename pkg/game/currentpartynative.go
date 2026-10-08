package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Known bytes live in the bound ordinary actor. The policy retains availability,
// unknown residue and observed absence for a member with no World row.
type currentPartyNativeHistory struct {
	Basis        sim.NativeActorBasis
	ClassPresent bool
}

func capturePartyNativeHistory(member mapload.PartyMember) (*currentPartyNativeHistory, error) {
	if member.Carry == nil || member.Carry.NativeHistory == nil {
		return nil, nil
	}
	h := member.Carry.NativeHistory
	if err := h.Basis.Validate(); err != nil {
		return nil, err
	}
	if !h.Class.Present && h.Class.Fighter || member.Carry.LiveLoad != nil && member.Carry.LiveLoad.Inventory.Source.Class != 0 {
		return nil, fmt.Errorf("party native history has conflicting source or class")
	}
	b := h.Basis
	for n := range b.Base {
		if b.BaseByteKnown(n) {
			b.Base[n] = 0
		}
	}
	for n := range b.Modifier {
		if b.ModifierByteKnown(n) {
			b.Modifier[n] = 0
		}
	}
	for n := range b.Attack {
		if b.AttackByteKnown(n) {
			b.Attack[n] = 0
		}
	}
	for n := range b.Defence {
		if b.DefenceByteKnown(n) {
			b.Defence[n] = 0
		}
	}
	if b.BodyKnown {
		b.Body = 0
	}
	return &currentPartyNativeHistory{Basis: b, ClassPresent: h.Class.Present}, nil
}

func (h *currentPartyNativeHistory) restore(r *sav.DocumentRecordData, source uint8, fighter bool) (*mapload.NativeCarryHistory, error) {
	if h == nil {
		return nil, nil
	}
	if source != 0 || r == nil || r.Class != "Human" && r.Class != "Humanoid" && r.Class != "Unit" || h.ClassPresent && r.Class == "Unit" {
		return nil, fmt.Errorf("native party history has no compatible ordinary actor")
	}
	b := h.Basis
	if err := b.Validate(); err != nil {
		return nil, err
	}
	for n := range b.Base {
		if b.BaseByteKnown(n) && b.Base[n] != 0 {
			return nil, fmt.Errorf("native party policy duplicates known Base")
		}
	}
	for n := range b.Modifier {
		if b.ModifierByteKnown(n) && b.Modifier[n] != 0 {
			return nil, fmt.Errorf("native party policy duplicates known Modifier")
		}
	}
	for n := range b.Attack {
		if b.AttackByteKnown(n) && b.Attack[n] != 0 {
			return nil, fmt.Errorf("native party policy duplicates known Attack")
		}
	}
	for n := range b.Defence {
		if b.DefenceByteKnown(n) && b.Defence[n] != 0 {
			return nil, fmt.Errorf("native party policy duplicates known Defence")
		}
	}
	if b.BodyKnown && b.Body != 0 {
		return nil, fmt.Errorf("native party policy duplicates known Body")
	}
	for _, block := range []struct {
		name string
		size int
		mask uint64
		dst  []byte
	}{
		{"U114", 24, uint64(b.BaseKnown), b.Base[:]}, {"UD4", 64, b.ModifierKnown, b.Modifier[:]},
		{"UA6", 24, uint64(b.AttackKnown), b.Attack[:]}, {"UBE", 22, uint64(b.DefenceKnown), b.Defence[:]},
	} {
		if block.mask == 0 {
			continue
		}
		raw, err := savedActorRaw(r, block.name, block.size)
		if err != nil {
			return nil, err
		}
		for n := range block.dst {
			if block.mask&(uint64(1)<<n) != 0 {
				block.dst[n] = raw[n]
			}
		}
	}
	if b.BodyKnown {
		body, err := savedStructureValue(r, "Body")
		if err != nil {
			return nil, err
		}
		b.Body = uint16(body)
	}
	class := sim.NativeClass{}
	if h.ClassPresent {
		class = sim.NativeClass{Present: true, Fighter: fighter}
	}
	return &mapload.NativeCarryHistory{Basis: b, Class: class}, nil
}
