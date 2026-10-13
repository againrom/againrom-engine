package sim

// castPhase is what an actor's book cast record is doing now.
type castPhase uint8

const (
	castNone castPhase = iota
	// castApproach walks a retained order into range (bookApproach).
	castApproach
	// castWaiting holds a retained order that waits for mana (bookPending).
	castWaiting
	castChargingUnpaid
	castChargingPaid
	// castRecovering is a retained record after its release: the recovery
	// count and the two boundary passes that end the cycle.
	castRecovering
)

// orderPhase is the one answer to "what is this actor's order doing now".
// The application of a cast happens inside the tick its wind-up reaches
// zero, so no tick ends in a releasing phase: a cast reads charging up to
// that tick and recovering after it.
type orderPhase uint8

const (
	orderPhaseNone orderPhase = iota
	orderPhaseStep
	orderPhaseQueued
	orderPhaseApproach
	orderPhaseWaiting
	orderPhaseChargingUnpaid
	orderPhaseChargingPaid
	orderPhaseRecovering
	orderPhaseScroll
	orderPhaseStriking
)

// orderState gathers every carrier of an actor's order in one value: the
// book cast record, the one-shot recovery CastWait, the scroll record, the
// player's cast row queued in PendingOrder, the step between two cells and
// the strike cycle. Every question about a cast in flight reads it.
type orderState struct {
	book   castPhase
	bookAt int
	wait   bool
	scroll uint8
	queued bool
	step   bool
	strike bool
}

const (
	scrollNone uint8 = iota
	scrollReserved
	scrollStarted
)

func (w *World) orderState(e *Entity) orderState {
	var s orderState
	if k, ok := w.bookCastIndex(e.ID); ok {
		s.bookAt = k
		switch c := w.bookCasts[k]; c.Phase {
		case bookApproach:
			s.book = castApproach
		case bookPending:
			s.book = castWaiting
		case bookCharging:
			s.book = castChargingUnpaid
			if c.Paid {
				s.book = castChargingPaid
			}
		default:
			s.book = castRecovering
		}
	}
	s.wait = e.CastWait != 0
	if k, ok := w.scrollIndex(e.ID); ok {
		s.scroll = scrollReserved
		if w.scrollCasts[k].Started {
			s.scroll = scrollStarted
		}
	}
	s.queued = e.PendingOrder.Kind == PendingActorCast || e.PendingOrder.Kind == PendingCellCast
	s.step = e.Transit != 0 || w.motionActive(e.ID)
	s.strike = e.AttackPhase != AttackReady || e.AttackCountdown != 0
	return s
}

func (w *World) orderStateAt(i int) orderState { return w.orderState(&w.entities[i]) }

// phase is the order's leading phase: the cast that owns the body first,
// then a scroll, a strike, a queued cast, a retained order and the step.
func (s orderState) phase() orderPhase {
	switch {
	case s.book == castChargingPaid:
		return orderPhaseChargingPaid
	case s.book == castChargingUnpaid:
		return orderPhaseChargingUnpaid
	case s.book == castRecovering || s.wait:
		return orderPhaseRecovering
	case s.scroll != scrollNone:
		return orderPhaseScroll
	case s.strike:
		return orderPhaseStriking
	case s.queued:
		return orderPhaseQueued
	case s.book == castApproach:
		return orderPhaseApproach
	case s.book == castWaiting:
		return orderPhaseWaiting
	case s.step:
		return orderPhaseStep
	}
	return orderPhaseNone
}

// booked reports a book cast record in any phase.
func (s orderState) booked() bool { return s.book != castNone }

// charging reports a book cast winding up, paid or not.
func (s orderState) charging() bool {
	return s.book == castChargingUnpaid || s.book == castChargingPaid
}

// paidWindUp reports a book cast winding up after its cost was spent. Once
// the mana is spent the cast completes: no order cancels or restarts it.
func (s orderState) paidWindUp() bool { return s.book == castChargingPaid }

// scrollPending reports a scroll cast from its reservation to its release.
func (s orderState) scrollPending() bool { return s.scroll != scrollNone }

// castBusy reports a cast that owns the actor's next action: a queued cast
// row, a scroll, a one-shot recovery or a book record past its approach.
func (s orderState) castBusy() bool {
	return s.queued || s.scroll != scrollNone || s.wait || s.book != castNone && s.book != castApproach
}

// ownsBody reports a cast that holds the body in place: the wind-up, the
// recovery after the release and a started scroll. The original keeps a
// cast's progress through its recovery, and a move order waits for its exit
// (MAGIC-311, AI-491).
func (s orderState) ownsBody() bool {
	return s.charging() || s.book == castRecovering || s.wait || s.scroll == scrollStarted
}
