package sim

// standingRule is when a step ends and the unit counts as standing.
type standingRule uint8

const (
	standingByArrival     standingRule = iota
	standingAtCentreBytes              // R2-ENGINE-1967
)

type standingBody struct {
	endsStill func(e *Entity) bool
	noMotion  func(rate int32, diagonal bool) bool
	arrived   func(next SavedActorPosition, paid bool) bool
}

var standingBodies = [...]standingBody{
	standingByArrival: {
		endsStill: func(*Entity) bool { return false },
		noMotion:  func(int32, bool) bool { return false },
		arrived:   func(_ SavedActorPosition, paid bool) bool { return paid },
	},
	standingAtCentreBytes: {
		endsStill: func(e *Entity) bool { return e.Stride.Present && e.Stride.StepX == 0 && e.Stride.StepY == 0 },
		noMotion:  func(rate int32, diagonal bool) bool { return strideAxis(rate, diagonal) == 0 },
		arrived:   func(next SavedActorPosition, _ bool) bool { return next.centred() },
	},
}
