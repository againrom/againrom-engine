package game

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type creaturePortraitProfile struct {
	bindingRow bool
}

type creaturePortraitTier struct {
	value int
	found bool
}

type creaturePortraitInput struct {
	existing, saved, placed creaturePortraitTier
	placement               *alm.Unit
	binding                 sim.SourceBinding
	typeID, partyClass      int32
	partyFace               int
	humanoid                bool
	keepSourceZero          bool
}

// resolveCreaturePortrait owns face precedence; its inputs keep absent and zero
// tiers distinct. Resolving a portrait never changes its actor's saved binding.
func resolveCreaturePortrait(in creaturePortraitInput, t *mapload.Table) (int, bool) {
	if in.existing.found {
		return in.existing.value, true
	}
	if in.saved.found && in.saved.value != 0 && !in.humanoid && in.binding.ActorClass() != 1 {
		if in.placement != nil && in.saved.value == int(uint8(in.placement.ClassSubID)) && in.placed.found && in.placed.value > 0 {
			return in.placed.value, true
		}
		return in.saved.value, true
	}
	if in.binding.ActorClass() == 1 {
		face := int(in.binding.Face)
		if face == 0 && tableFiles(t).creaturePortraits.bindingRow && int32(in.binding.TypeID) == in.typeID {
			row := int(in.binding.DefinitionRow())
			if tier, ok := unitRowTier(row, t, in.typeID); row > 0 && ok {
				face = tier
			}
		}
		if face != 0 || in.keepSourceZero {
			return face, true
		}
	}
	if in.placed.found && !in.humanoid {
		return in.placed.value, true
	}
	if !data.ComposesFigure(in.partyClass) && in.partyFace > 0 {
		return in.partyFace, true
	}
	return 0, false
}
