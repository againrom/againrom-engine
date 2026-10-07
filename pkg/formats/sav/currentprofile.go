package sav

import "fmt"

// ActorCurrent is a narrow current-value view, not a Human rebuild basis.
// The live attack/defence blocks have the same Unit layout on all three
// classes (SAV-UNITPROG-156, SAV-HUMRUN-444). Unknown tails, maintained bases,
// equipment, skill membership and book membership are deliberately absent.
type ActorCurrent struct {
	ActorPools
	Class                                         string
	Reaction, Mind, Spirit                        int16
	ToHit, Defence, Absorption                    int16
	DamageBase, DamageSpread, Active              uint8
	SecondBase, SecondSpread                      uint8
	ElementalBase, ElementalSpread, ElementalKind uint8
	Protection                                    [5]int16
	Resistance                                    [5]uint8
	HealthPeriod, ManaPeriod                      int16
	HealthRegeneration, ManaRegeneration          int16
	HealthHundredths, ManaHundredths              uint8
}

// ActorCurrentProfiles traverses the complete, exact shared archive, including
// every Player and repeated/null group references. Values are detached. A late
// malformed object returns no partial projection (SAV-HUMLOAD-445).
func (f *File) ActorCurrentProfiles(current ...uint16) ([]ActorCurrent, error) {
	actors, err := f.playerActors(current)
	if err != nil {
		return nil, err
	}
	var out []ActorCurrent
	for _, actor := range actors {
		p, err := actorPool(actor)
		if err != nil {
			return nil, err
		}
		a, d, m := actor.Raw["UA6"], actor.Raw["UBE"], actor.Raw["UD4"]
		if len(a) != 24 || len(d) != 22 || len(m) != 64 {
			return nil, fmt.Errorf("sav: actor at %d has incomplete current blocks", actor.Off)
		}
		c := ActorCurrent{ActorPools: p, Class: actor.Class,
			Reaction: int16(actor.value("Reaction")), Mind: int16(actor.value("Mind")), Spirit: int16(actor.value("Spirit")),
			ToHit: int16(u16(a, 0)), Defence: int16(u16(d, 0)), Absorption: int16(u16(d, 2)),
			DamageBase: a[14], DamageSpread: a[15], Active: a[16], SecondBase: a[17], SecondSpread: a[18],
			ElementalBase: a[19], ElementalSpread: a[20], ElementalKind: a[21],
			HealthPeriod: int16(actor.value("HealthRegen")), ManaPeriod: int16(actor.value("ManaRegen")),
			HealthRegeneration: int16(u16(m, 10)), ManaRegeneration: int16(u16(m, 14)),
			HealthHundredths: uint8(actor.value("UA2")), ManaHundredths: uint8(actor.value("UA3")),
		}
		for i := range c.Protection {
			c.Protection[i] = int16(u16(d, 6+2*i))
			c.Resistance[i] = d[17+i]
		}
		out = append(out, c)
	}
	return out, nil
}

func (f *File) playerActors(current []uint16) ([]*Record, error) {
	if f == nil {
		return nil, fmt.Errorf("sav: actors require a save")
	}
	doc, _, err := f.exactDocument()
	if err != nil {
		return nil, err
	}
	seen := make(map[*Record]bool)
	var out []*Record
	for _, actor := range currentActorRecords(doc.players, current, doc.dead) {
		if actor == nil {
			continue
		}
		if !groundClass(actor.Class, "Unit") {
			return nil, fmt.Errorf("sav: Player actor at %d is %s, not a Unit", actor.Off, actor.Class)
		}
		if !seen[actor] {
			seen[actor] = true
			out = append(out, actor)
		}
	}
	return out, nil
}

func actorPool(actor *Record) (ActorPools, error) {
	p := actor.Raw["Block12"]
	if len(p) != 12 {
		return ActorPools{}, fmt.Errorf("sav: actor at %d has no complete Position", actor.Off)
	}
	return ActorPools{Off: actor.Off, MapUnitID: uint16(actor.value("T08")), Cell: u16(p, 0), Stage: uint8(actor.value("Stage")),
		HP: uint16(actor.value("Health")), MaxHP: uint16(actor.value("HealthMax")), Mana: uint16(actor.value("Mana")), MaxMana: uint16(actor.value("ManaMax"))}, nil
}
