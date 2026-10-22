package sim

func (w *World) unsafeUnorderedArea(ci, ti int, rule SpellRule) bool {
	if !rule.Area || !isMage(w.entities[ci]) || !w.healArmRules(w.entities[ci]) || holdsOrderedVictim(w.entities[ci]) {
		return false
	}
	caster, target := w.entities[ci], w.entities[ti]
	e := cellEffect{Key: cellKey(target.X, target.Y), Spell: rule.ID, Mode: areaModeFor(rule), Direction: uint8(FacingDir(w.headingBetween(caster, target)))}
	var cells []uint16
	switch e.Mode {
	case areaModeBlast:
		cells = w.blastCellsInMap(target.X, target.Y, int32(rule.Radius))
	case areaModeRing:
		if ringStageDraws(rule.arm()) {
			cells = w.ringCellsAtOffsets(e, meteorStormFootprint())
		} else {
			for stage := 0; stage < ringStageCount(rule.arm()); stage++ {
				cells = append(cells, w.ringStageCells(e, stage)...)
			}
		}
	default:
		paint := w.diamondCells(target.X, target.Y, int32(rule.Radius))
		if rule.Distribution == distributionWall {
			paint = w.wallCells(target.X, target.Y, e.Direction)
			if areaArmBlocks(rule.arm()) {
				paint = w.skipOccupiedGround(paint)
			}
		}
		cells = w.cloudPulseCellsWithPaint(e.Key, rule.arm(), int32(rule.Radius), paint)
	}
	for _, k := range cells {
		x, y := keyCell(k)
		ground, air := w.cellLayerOccupants(x, y)
		groups := [][]int{ground}
		if e.Mode != areaModeCloud {
			groups = append(groups, air)
		}
		for _, group := range groups {
			for _, i := range group {
				unit := w.entities[i]
				if unit.ID != caster.ID && unit.Owner != caster.Owner && !w.relations.Locked(caster.Owner, unit.Owner) {
					continue
				}
				if !areaArmBlocks(rule.arm()) && spellTargetable(unit, rule) && w.areaHitAllowed(ci, rule, unit) {
					return true
				}
			}
		}
	}
	return false
}
