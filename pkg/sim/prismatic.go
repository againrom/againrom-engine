package sim

const prismaticSpellID = 14

// prismaticBodyPrimary reports a body aimed as Prismatic Spray's primary.
// A body is never the primary (owner, DIV-072); a fan already paid at
// admission has passed this gate and keeps its release.
func prismaticBodyPrimary(target Entity, rule SpellRule) bool {
	return rule.ID == prismaticSpellID && target.HP < 1
}

// prismaticGroup is the group whose shared sight chooses the secondaries:
// the caster's saved Group, else its command or map group, else the caster
// alone. A player order has already built a one-member command group.
func (w *World) prismaticGroup(ci int) aiGroup {
	c := w.entities[ci]
	if g := w.savedGroupFor(c.ID); g != nil {
		if members, ok := w.savedLiving(g); ok && containsIndex(members, ci) {
			return aiGroup{owner: c.Owner, group: g.ID, members: members}
		}
	}
	members := w.groupLivingMembers(c.Owner, effectiveGroup(c))
	if !containsIndex(members, ci) {
		members = []int{ci}
	}
	return aiGroup{owner: c.Owner, group: effectiveGroup(c), members: members}
}

// footprintCoord is MAGIC-REACH-179's axis coordinate in 1/256 cell, u16 wide.
// Native actors are cell-centred, so the fraction byte is 128.
func footprintCoord(size uint8, at int32) int32 {
	return int32(uint16(((uint32(footprintSide(size)) + 2*uint32(at) + 511) << 7) + 128))
}

// edgeDistance is MAGIC-REACH-179's footprint-gap distance as a byte.
func edgeDistance(a, b Entity) uint8 {
	gap := func(ca, cb int32) int32 {
		d := ca - cb
		if d < 0 {
			d = -d
		}
		d -= (footprintSide(a.TokenSize) + footprintSide(b.TokenSize)) * 128
		return max(d, 0)
	}
	gx := gap(footprintCoord(a.TokenSize, a.X), footprintCoord(b.TokenSize, b.X))
	gy := gap(footprintCoord(a.TokenSize, a.Y), footprintCoord(b.TokenSize, b.Y))
	return uint8(1 + max(gx, gy)/256)
}

// prismaticTurnCost is the score's low byte (AI-COST-071, AI-361): the circular
// arc from facing to the heading toward the candidate (MAGIC-242).
func (w *World) prismaticTurnCost(facing uint8, caster, candidate Entity) uint32 {
	return uint32(facingArc(facing, w.headingBetween(caster, candidate)))
}

// prismaticVictims is the selector of MAGIC-SPRAY-134 and MAGIC-SPRAY-137 over
// the caster group's candidates (AI-SPRAY-266/267): the primary first, then
// the winners of cap strict-minimum scans, truncated to cap. List B is
// omitted: beside a living A its scores exceed the 65530 sentinel.
// A caster-less cast has no group and answers the primary alone. The primary's
// alarm and hostility flip come first (AI-DIPLO-084).
func (w *World) prismaticVictims(ci, ti int, rule SpellRule, power int32) []int {
	limit := rule.RayLimit(power)
	if ci < 0 || ci >= len(w.entities) {
		return []int{ti}
	}
	w.flipOnBlow(ci, ti)
	cands := w.candidates(w.prismaticGroup(ci))
	if len(cands) == 0 {
		return []int{ti}
	}
	caster := w.entities[ci]
	// The facing gate holds facing equal to the heading to the primary
	// (MAGIC-REACH-179); admission here precedes the turn, so the heading stands in.
	primary := w.entities[ti]
	facing := w.headingBetween(caster, primary)
	scores := make([]uint32, len(cands))
	for k, i := range cands {
		scores[k] = (uint32(edgeDistance(caster, w.entities[i]))<<8 + w.prismaticTurnCost(facing, caster, w.entities[i])) & 0xffff
	}
	type winner struct{ at, score uint32 }
	var winners []winner
	for scan := 0; scan < limit; scan++ {
		best := winner{at: 65530, score: 65530}
		for k := range cands {
			if scores[k] < best.score {
				best = winner{at: uint32(k), score: scores[k]}
			}
		}
		if best.at >= 60000 {
			break
		}
		scores[best.at] = 65500
		winners = append(winners, best)
	}
	out := []int{ti}
	for _, win := range winners {
		if win.at < 65000 && win.score < 65000 && cands[win.at] != ti {
			out = append(out, cands[win.at])
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
