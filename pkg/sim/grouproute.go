package sim

// secondGuardRoute is programme byte 18 (R2-ENGINE-507).
const secondGuardRoute uint8 = 0x12

// groupRouteProfile is a game's saved Group AI: the route of each programme
// byte +0x20, and the rules the Group handlers decide by game. The zero value
// is the first game's: every byte routes to itself.
type groupRouteProfile struct {
	// guardRoute: whether programme byte 18 takes the guard route
	// (guardRouteRules, DIV-4285).
	guardRoute guardRouteRule
	// constructsForOlderRegistry: a non-native registry gets its orders built.
	constructsForOlderRegistry bool
	// humanArm: a human's unit with no target takes no heal arm; else
	// it does (AI-ACQUIRE-002, AI-349).
	humanArm     humanArmRule
	recordsBlows bool
	// stampWindow: guard B's member cases 1 and 2 run from the stamp a
	// failed approach route writes (R2-ENGINE-844, R2-ENGINE-873).
	stampWindow bool
	// recentBlow: guard B's member case 4 runs from the blow's clock
	// (R2-ENGINE-844, DIV-4503).
	recentBlow bool
	// targetless: whether an engage with no target left ends (DIV-4691).
	targetless targetlessRule
	// belowFloorAt18: the radius the mode-18 setter with a zero range
	// argument writes for a widest sum below MinimalGuardRange
	// (R2-ENGINE-1253); zero raises the sum to MinimalGuardRange.
	belowFloorAt18 uint8
}

// spreadRadius is radiusSpread at programme b and range n: a nonzero n raises
// the sum to n (AI-793, R2-ENGINE-1253); else the sum, or below the floor
// MinimalGuardRange or belowFloorAt18 at programme 18.
func (p *groupRouteProfile) spreadRadius(sum, b, n uint8) uint8 {
	switch {
	case n != 0:
		return max(sum, n)
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
	return guardRouteRules[p.guardRoute].route(b)
}

// guardRouteRule selects the byte that programme 18 and the guard order map to.
type guardRouteRule uint8

const (
	guardRouteOwnByte guardRouteRule = iota
	guardRouteAt18
)

// guardRouteRow holds the two mappings of a guardRouteRule.
type guardRouteRow struct {
	route     func(b uint8) uint8
	programme func(order uint8) uint8
}

var guardRouteRules = [...]guardRouteRow{
	guardRouteOwnByte: {
		route:     func(b uint8) uint8 { return b },
		programme: func(order uint8) uint8 { return order },
	},
	guardRouteAt18: {
		route: func(b uint8) uint8 {
			if b == secondGuardRoute {
				return orderGuard
			}
			return b
		},
		programme: func(order uint8) uint8 {
			if order == orderGuard {
				return secondGuardRoute
			}
			return order
		},
	},
}

// targetlessRule selects whether an engage with no target left ends.
type targetlessRule uint8

const (
	targetlessContinues targetlessRule = iota
	targetlessEnds
)

// targetlessRow holds the end test of a targetlessRule.
type targetlessRow struct {
	ends func(w *World, i int) bool
}

var targetlessRules = [...]targetlessRow{
	targetlessContinues: {ends: func(*World, int) bool { return false }},
	targetlessEnds:      {ends: func(w *World, i int) bool { return w.engageHasNothingLeft(i) }},
}

// humanArmRule selects whether a human's unit with no target sets its order
// byte to 0 when it is player controlled.
type humanArmRule uint8

const (
	humanArmPlayerZeroes humanArmRule = iota
	humanArmNone
)

// humanArmRow holds the test of a humanArmRule.
type humanArmRow struct {
	zeroes func(w *World, e Entity) bool
}

var humanArmRules = [...]humanArmRow{
	humanArmPlayerZeroes: {zeroes: func(w *World, e Entity) bool { return w.playerControlledUnit(e) }},
	humanArmNone:         {zeroes: func(*World, Entity) bool { return false }},
}

// programme is the programme byte a Group running order holds: the second
// game holds its guard at 18, as every computer Group of its mission files
// does (DIV-4285); every other order is its own byte.
func (p *groupRouteProfile) programme(order uint8) uint8 {
	return guardRouteRules[p.guardRoute].programme(order)
}

var secondGameGroupRoute = groupRouteProfile{
	guardRoute:                 guardRouteAt18,
	constructsForOlderRegistry: true,
	humanArm:                   humanArmNone,
	recordsBlows:               true,
	stampWindow:                true,
	recentBlow:                 true,
	targetless:                 targetlessEnds,
	belowFloorAt18:             20,
}

func (w *World) savedRoute(g *SavedGroup) uint8 {
	return w.game.groupRoute.route(g.AI[0x20])
}
