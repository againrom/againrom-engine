package sim

// refusedRoute names the walk a refused route ends: the order machine reads the
// actor's walk state when it consumes the route-failure flag, and each state
// has one row of refusedAnswers.
type refusedRoute uint8

const (
	refusedPursuit refusedRoute = iota
	refusedWalk
	refusedFlee
	refusedRetreatFailure
)

// refusedNone is what a refused route leaves when the reach pick names no
// victim, or names one for a suppressed acquirer (AI-GUARD-007).
type refusedNone uint8

const (
	// noneStays writes nothing: the actor stays where the walk ended.
	noneStays refusedNone = iota
	// noneIdle idles the pursuit and leaves the victim field as it was.
	noneIdle
	// noneClear clears the pending order and the victim.
	noneClear
)

// refusedAnswer is one row of the refused-route answer.
type refusedAnswer struct {
	// none is the outcome with no victim in reach.
	none refusedNone
	// failedIsNone takes the none outcome when the picked victim's order is
	// not written either.
	failedIsNone bool
	// progress restarts a Retreat actor's strike progress once it holds a
	// victim (AI-365: R0005 writes progress 1 and counter 0).
	progress bool
	// keepPhase restores the action phase and countdown the order rewrote.
	keepPhase bool
}

// refusedAnswers is the one table of refused-route answers. The pick is the
// hostile nearest within reach for every row (reacquisitionVictim, AI-327,
// AI-360, AI-361, AI-ROUTE-045); the rows differ in what the walk state leaves
// without one.
var refusedAnswers = [...]refusedAnswer{
	// The failure exit idles the order and never writes the victim (AI-328).
	refusedPursuit: {none: noneIdle, failedIsNone: true},
	// A walk with no victim stays with none in reach (AI-648, AI-649).
	refusedWalk: {none: noneStays},
	// A refused flee clears the order and the victim (AI-ROUTE-045, AI-328).
	refusedFlee: {none: noneClear, progress: true},
	// State 0x16's failure tail reinstalls strike progress, phase kept (AI-365).
	refusedRetreatFailure: {none: noneClear, progress: true, keepPhase: true},
}

// answerRefusedRoute is the order machine's answer to a refused route for
// actor i in walk state r: the reach pick becomes its acquisition order, and
// the row of refusedAnswers decides the rest.
func (w *World) answerRefusedRoute(i int, r refusedRoute) {
	a := refusedAnswers[r]
	e := &w.entities[i]
	phase, countdown := e.AttackPhase, e.AttackCountdown
	victim := w.reacquisitionVictim(i)
	if victim < 0 || w.suppressedAcquirer(*e) {
		w.answerRefusedNone(i, a.none)
		return
	}
	if !w.orderAcquire(i, w.entities[victim].ID) && a.failedIsNone {
		w.answerRefusedNone(i, a.none)
		return
	}
	if !e.HasAttackTarget {
		return
	}
	if a.keepPhase {
		e.AttackPhase, e.AttackCountdown = phase, countdown
	}
	if a.progress && e.ActorState == actorStateRetreat {
		e.Retreat.restartStrike()
	}
}

func (w *World) answerRefusedNone(i int, none refusedNone) {
	e := &w.entities[i]
	switch none {
	case noneIdle:
		e.PursuitIdle = true
	case noneClear:
		w.clearOrder(i)
		e.clearAttack()
	}
}

// restartStrike is the one writer of the retreat strike progress reset:
// progress 1, counter 0 and the completion latch cleared (AI-365).
func (r *RetreatContinuation) restartStrike() {
	r.Progress, r.Counter, r.Complete = 1, 0, false
}
