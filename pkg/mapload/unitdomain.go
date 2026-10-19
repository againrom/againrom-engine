package mapload

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// UnitDomain is the movement domain the map load gives placement u: the
// domain of the block it builds for u, so a flier the map places is a flier
// here. A nil table answers ground.
func UnitDomain(u alm.Unit, t *Table) (sim.Domain, error) {
	if t == nil {
		return sim.DomainGround, nil
	}
	b, err := blockFor(u, t, DifficultyNormal)
	if err != nil {
		return sim.DomainGround, err
	}
	return b.domain, nil
}
