package sim

// secondGuardRoute is programme byte 18 (R2-ENGINE-507).
const secondGuardRoute uint8 = 0x12

// groupRouteProfile is a game's saved Group AI: the route of each programme
// byte +0x20, and the rules the Group handlers decide by game.
type groupRouteProfile struct {
	routes [256]uint8
	// guardOverMembers: the guard order runs over dying members too.
	guardOverMembers bool
	// constructsForOlderRegistry: a non-native registry gets its orders built.
	constructsForOlderRegistry bool
	// humanHealArm: a human's unit with no target takes the heal arm
	// (AI-ACQUIRE-002, AI-349).
	humanHealArm bool
	// engagedFoesClipped: engaged foes must lie in the clipped list
	// (AI-GRPGUARD-074, AI-REISSUE-077); else all stay (DIV-1523).
	engagedFoesClipped bool
	// heldCircle: a registry Group's circle is the held one, not the stored
	// one (AI-740, AI-741).
	heldCircle bool
}

func groupRoutes(at18 uint8) (routes [256]uint8) {
	for i := range routes {
		routes[i] = uint8(i)
	}
	routes[secondGuardRoute] = at18
	return routes
}

var firstGameGroupRoute = groupRouteProfile{
	routes:             groupRoutes(secondGuardRoute),
	guardOverMembers:   true,
	humanHealArm:       true,
	engagedFoesClipped: true,
}

// The second game's byte 18 takes the guard route (DIV-4285).
var secondGameGroupRoute = groupRouteProfile{
	routes:                     groupRoutes(orderGuard),
	constructsForOlderRegistry: true,
	heldCircle:                 true,
}

func (w *World) groupRoute() *groupRouteProfile {
	if w.SecondGame() {
		return &secondGameGroupRoute
	}
	return &firstGameGroupRoute
}

func (w *World) savedRoute(g *SavedGroup) uint8 {
	return w.groupRoute().routes[g.AI[0x20]]
}
