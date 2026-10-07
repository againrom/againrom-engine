package sav

import "fmt"

// ActorBasis retains the complete reached derive input on the owner graph.
// Reading it performs no derive and does not infer a template from a type ID.
type ActorBasis struct {
	Class                            string
	Stats                            [UnitStatWords]uint16
	SkillXP                          [6]uint32
	Experience                       uint32
	Human                            CityHuman
	Reach, AttackCharge, AttackRelax uint8

	// Token18 is the Token head's own +0x18 word (Unit+0x18 = Token+0x18,
	// SAV-636), a 16-bit recipient publication mask the original's located
	// emission paths set and clear (SAV-678, High for the mechanism, Medium
	// for the breadth of its senders/clearers). SAV-797's corpus census finds
	// exactly two values, 2 on 1 404 of 1 420 records and 0 on the other 16,
	// with no established rule for which records carry which (Medium). This
	// package carries it decoded and NAMES NO CONSUMER: no recipient-mask or
	// fog-of-war mechanism exists anywhere in pkg/sim or pkg/game to wire it
	// into (DIV-969).
	Token18 uint16
}

func actorBasis(r *Record, players []*Record) (*ActorBasis, error) {
	b, err := actorBasisFields(r)
	if err != nil {
		return nil, err
	}
	h := &b.Human
	if owner := r.value("Reference"); owner != 0 {
		for _, p := range players {
			if p.value("This") == owner {
				h.HasOwner = true
				h.ManaReservePercent = p.value("F58")
				break
			}
		}
		if !h.HasOwner {
			return nil, fmt.Errorf("sav: actor %d owner %#x does not resolve", r.Off, owner)
		}
	}
	return b, nil
}

func actorBasisFields(r *Record) (*ActorBasis, error) {
	if len(r.Raw["UA6"]) != 24 || len(r.Raw["UBE"]) != 22 || len(r.Raw["U114"]) != 24 || len(r.Raw["UD4"]) != 64 || len(r.Raw["U154"]) != 180 {
		return nil, fmt.Errorf("sav: actor %d has incomplete derive basis", r.Off)
	}
	b := &ActorBasis{Class: r.Class, Experience: r.value("U130"), Token18: uint16(r.value("T18"))}
	b.Reach, b.AttackCharge, b.AttackRelax = uint8(r.value("U12C")), uint8(r.value("U134")), uint8(r.value("U135"))
	for i, n := range statNames {
		b.Stats[i] = uint16(r.value(n))
	}
	h := &b.Human
	h.Fighter, h.HasSpellbook, h.TypeID = r.value("U4C")&4 == 0, r.value("HasSpellbook") != 0, uint16(r.value("T0E"))
	h.InventoryWeight = int32(r.value("Inventory20"))
	copy(h.Fields.Attack[:], r.Raw["UA6"])
	copy(h.Fields.Base[:], r.Raw["U114"])
	copy(h.Fields.Defence[:], r.Raw["UBE"])
	copy(h.Fields.Modifier[:], r.Raw["UD4"])
	// Sight (+0xa4) is a plain 16-bit read, not a derive: SAV-795/SAV-796 find
	// its low byte always zero on a Unit record and its high byte the whole
	// scan-range cell count a consumer already reads with `>> 8`
	// (originalactorregistry.go), so nothing here may round or rescale it.
	h.Fields.ManaFloor, h.Fields.Sight, h.Fields.MoverSpeed = uint16(r.value("UA0")), uint16(r.value("UA4")), r.Raw["U154"][10]
	if r.Class != "Unit" {
		if len(r.Raw["H1CC"]) != 24 {
			return nil, fmt.Errorf("sav: actor %d lacks Humanoid XP", r.Off)
		}
		for i := range b.SkillXP {
			b.SkillXP[i] = u32(r.Raw["H1CC"], 4*i)
		}
	}
	return b, nil
}
