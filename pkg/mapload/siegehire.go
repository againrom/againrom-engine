package mapload

import (
	"fmt"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// siegeDefinition is the definition a hired Catapult or Ballista (tavern type
// 1 or 2) is built from. The two types are Units-table creatures
// (MERC-LEVEL-005): the Units row supplies their combat, domain, footprint and
// equipment, so the block resolves through the same placement path and the
// same definition as a map creature.
func siegeDefinition(p PartyMember, t *Table) (sim.ActorDefinition, spawnBlock, error) {
	b, err := blockFor(alm.Unit{ClassID: int16(p.Class), ClassSubID: uint16(p.FigureFace)}, t, DifficultyNormal)
	if err != nil {
		return sim.ActorDefinition{}, spawnBlock{}, err
	}
	return b.actorDefinition(t), b, nil
}

// SiegeHire is a hired Catapult or Ballista as the constructor builds it: the
// actor basis (token row, face, type, tracked blocks) of the Units row and the
// worn set that row arms.
type SiegeHire struct {
	Basis sim.Entity
	Worn  [sim.EquipSlots]sim.ItemInstance
}

// SiegeHireActor constructs the hire's actor basis through the same
// constructor a map creature takes (ConstructActorBasis over the resolved
// Units row), then completes its load from the row's own worn set. A Unit
// carries own weight equal to its load.
func SiegeHireActor(p PartyMember, t *Table, key uint32) (SiegeHire, error) {
	if p.MercenaryType != 1 && p.MercenaryType != 2 {
		return SiegeHire{}, fmt.Errorf("mapload: tavern type %d is not a siege hire", p.MercenaryType)
	}
	def, b, err := siegeDefinition(p, t)
	if err != nil {
		return SiegeHire{}, err
	}
	e := sim.NewActor(def, sim.ActorPlacement{Owner: sim.SelfSlot})
	placement := alm.Unit{ClassID: int16(p.Class), ClassSubID: uint16(p.FigureFace)}
	// A generated binding needs a nonzero identity and runtime id; the SAVE
	// writer replaces both with the document's own.
	if key == 0 {
		key = 1
	}
	basis, _, err := ConstructActorBasis(e, p, &placement, t, key, 1, nil, 0)
	if err != nil {
		return SiegeHire{}, err
	}
	basis = ConstructActorLoad(basis, b.worn, nil)
	own := basis.ActorLoad.Source.Stats
	if own[5] == 0 {
		own[5] = own[6]
	}
	basis.ActorLoad.Source.Stats = own
	return SiegeHire{Basis: basis, Worn: cloneItemEquipment(b.worn)}, nil
}
