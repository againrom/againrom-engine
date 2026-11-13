package sim

// secondGuardRoute is programme byte 18 (R2-ENGINE-507).
const secondGuardRoute uint8 = 0x12

// groupRouteProfile is a game's saved Group AI: the route of each programme
// byte +0x20, and the rules the Group handlers decide by game. The zero value
// is the first game's: every byte routes to itself.
type groupRouteProfile struct {
	// guardAt18: programme byte 18 takes the guard route (DIV-4285).
	guardAt18 bool
	// constructsForOlderRegistry: a non-native registry gets its orders built.
	constructsForOlderRegistry bool
	// noHumanHealArm: a human's unit with no target takes no heal arm; else
	// it does (AI-ACQUIRE-002, AI-349).
	noHumanHealArm bool
	recordsBlows   bool
	// stampWindow: guard B's member cases 1 and 2 run from the stamp a
	// failed approach route writes (R2-ENGINE-844, R2-ENGINE-873).
	stampWindow bool
	// recentBlow: guard B's member case 4 runs from the blow's clock
	// (R2-ENGINE-844, DIV-4503).
	recentBlow bool
	// endsTargetlessEngage (DIV-4691).
	endsTargetlessEngage bool
	// belowFloorAt18: the radius the mode-18 setter with no range argument
	// writes for a widest sum below MinimalGuardRange (R2-ENGINE-843); zero
	// raises the sum to MinimalGuardRange.
	belowFloorAt18 uint8
}

// spreadRadius is radiusSpread for a Group at programme byte b: the widest
// sum, or for a sum below MinimalGuardRange the floor, or belowFloorAt18 at
// programme 18.
func (p *groupRouteProfile) spreadRadius(sum, b uint8) uint8 {
	switch {
	case sum >= minimalGuardRange:
		return sum
	case b == secondGuardRoute && p.belowFloorAt18 != 0:
		return p.belowFloorAt18
	}
	return minimalGuardRange
}

// runsStampWindow reports whether a Group at programme byte b runs guard
// B's member cases 1 and 2 and its members' approaches stamp: only
// programme 18 does; guard A has cases 3 and 6 (R2-ENGINE-844,
// R2-ENGINE-873).
func (p *groupRouteProfile) runsStampWindow(b uint8) bool {
	return p.stampWindow && b == secondGuardRoute
}

// runsRecentBlow reports whether a Group at programme byte b runs guard B's
// member case 4.
func (p *groupRouteProfile) runsRecentBlow(b uint8) bool {
	return p.recentBlow && b == secondGuardRoute
}

func (p *groupRouteProfile) route(b uint8) uint8 {
	if b == secondGuardRoute && p.guardAt18 {
		return orderGuard
	}
	return b
}

// programme is the programme byte a Group running order holds: the second
// game holds its guard at 18, as every computer Group of its mission files
// does (DIV-4285); every other order is its own byte.
func (p *groupRouteProfile) programme(order uint8) uint8 {
	if order == orderGuard && p.guardAt18 {
		return secondGuardRoute
	}
	return order
}

var secondGameGroupRoute = groupRouteProfile{
	guardAt18:                  true,
	constructsForOlderRegistry: true,
	noHumanHealArm:             true,
	recordsBlows:               true,
	stampWindow:                true,
	recentBlow:                 false,
	endsTargetlessEngage:       true,
	belowFloorAt18:             20,
}

func (w *World) savedRoute(g *SavedGroup) uint8 {
	return w.game.groupRoute.route(g.AI[0x20])
}
