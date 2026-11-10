package sim

type targetRanking uint8

const (
	targetGroupScore targetRanking = iota
	targetTurn
	targetIdleDistance
	targetCover
	targetIdleReserve
)

type orderProfile struct {
	idleRanking              targetRanking
	liveAdmission            bool
	mageAcquisitionException bool
	stopEmptyWalk            bool
	standGroundScore         bool
	lookAround               bool
	repickEmptyRoute         bool
	primaryCells             bool
}

// chooseTarget ranks an ordered candidate population without changing it.
func (w *World) chooseTarget(i int, candidates []int, order uint8, ranking targetRanking) int {
	winner, best := -1, scoreSeed
	for _, ci := range candidates {
		if ci == i {
			continue
		}
		cost := int32(0)
		switch ranking {
		case targetGroupScore:
			cost = w.candidateCost(i, ci, order)
		case targetTurn:
			cost = facingArc(w.entities[i].Facing, w.headingBetween(w.entities[i], w.entities[ci]))
		case targetCover:
			if w.targetVetoed(i, ci) {
				continue
			}
			cost = int32(cellOf(&w.entities[i]).chebyshevTo(cellOf(&w.entities[ci])))
			if winner >= 0 && coverPreferred(w.entities[winner]) != coverPreferred(w.entities[ci]) {
				if coverPreferred(w.entities[winner]) {
					continue
				}
				winner, best = ci, cost
				continue
			}
		case targetIdleDistance, targetIdleReserve:
			m, c := w.entities[i], w.entities[ci]
			if ranking == targetIdleDistance && w.candidateCost(i, ci, orderGuard) == scoreSeed {
				continue
			}
			d := int32(w.targetEdgeDistance(m, c))
			if ranking == targetIdleDistance && lawDomain(c.Domain) == 3 && lawDomain(m.Domain) != 3 {
				d++
			}
			if int64(d) > w.attackRange(i) {
				continue
			}
			cost = d<<8 + facingArc(m.Facing, w.headingBetween(m, c))
		}
		if cost < best || (ranking == targetTurn || ranking == targetIdleDistance || ranking == targetIdleReserve) && cost == best {
			winner, best = ci, cost
		}
	}
	return winner
}

// targetEdgeDistance uses the current position operands (R2-ENGINE-1106).
func (w *World) targetEdgeDistance(a, b Entity) uint8 {
	ax, ay := w.actorCentre(a)
	bx, by := w.actorCentre(b)
	return edgeDistanceAt(ax, ay, bx, by, footprintSide(a.TokenSize), footprintSide(b.TokenSize))
}
