package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// LegacyPartyRepairs reports compatibility changes made at a save ownership
// boundary. ItemEffects counts repaired scalar effects; HiredRotationSpeeds
// counts hired members whose exact saved definition-row name restored the
// row-local turn rate introduced after their save was written.
type LegacyPartyRepairs struct {
	ItemEffects         int
	HiredRotationSpeeds int
}

// RepairLegacyParty repairs complete item state and hired definition metadata
// in an already-owned party. Both canonical item populations are rewritten
// together with their code projections. A second call is inert.
func RepairLegacyParty(party []PartyMember, t *Table) LegacyPartyRepairs {
	var out LegacyPartyRepairs
	for i := range party {
		equipped := MemberItemEquipment(party[i], t)
		carried := MemberCarriedItems(party[i], t)
		changed := false
		repair := func(item *sim.ItemInstance) {
			before := append([]sim.ItemEffect(nil), item.Effects...)
			if sim.RepairLegacyLinkedItemInstance(item, func(repaired sim.ItemInstance) int32 {
				return RepriceItemInstance(repaired, t)
			}) {
				for effect := range before {
					if before[effect] != item.Effects[effect] {
						out.ItemEffects++
					}
				}
				changed = true
			}
		}
		for slot := range equipped {
			repair(&equipped[slot])
		}
		for position := range carried {
			repair(&carried[position])
		}
		if changed {
			writePartyStock(&party[i], equipped, carried)
		}

		if party[i].MercenaryType == 0 || party[i].HiredRotationSpeed != 0 {
			continue
		}
		if speed := legacyHiredRotationSpeed(party[i], t); speed > 0 {
			party[i].HiredRotationSpeed = speed
			out.HiredRotationSpeeds++
		}
	}
	return out
}

func legacyHiredRotationSpeed(p PartyMember, t *Table) int32 {
	if t == nil || p.Name == "" {
		return 0
	}
	if p.MercenaryType <= 2 {
		for i := 1; t.Units != nil && i < t.Units.Len(); i++ {
			if t.Units.EntryName(i) != p.Name {
				continue
			}
			if def, err := data.NewUnitDef(p.Name, t.Units.EntryParams(i)); err == nil {
				return def.RotationSpeed
			}
			return 0
		}
		return 0
	}
	i := data.FindHumanByName(t.Humans, p.Name)
	if i == data.NotFound {
		return 0
	}
	def, err := data.NewHumanDef(p.Name, t.Humans.EntryParams(i))
	if err != nil {
		return 0
	}
	return def.RotationSpeed
}
