package sim

// secondGuardRoute is programme byte 18 (R2-ENGINE-507).
const secondGuardRoute uint8 = 0x12

// groupRouteProfile is a game's saved Group AI: the route of each programme
// byte +0x20, and the rules the Group handlers decide by game. The zero value
// is the first game's: every byte routes to itself.
type groupRouteProfile struct {
	// guardAt18: programme byte 18 takes the guard route (DIV-4285).
	guardAt18 bool
	// guardNeedsMembers: the guard order, as every other, decides only over
	// living members; else it runs over dying members too.
	guardNeedsMembers bool
	// constructsForOlderRegistry: a non-native registry gets its orders built.
	constructsForOlderRegistry bool
	// noHumanHealArm: a human's unit with no target takes no heal arm; else
	// it does (AI-ACQUIRE-002, AI-349).
	noHumanHealArm bool
	// engagedFoesUnclipped: all engaged foes stay (DIV-1523); else they must
	// lie in the clipped list (AI-GRPGUARD-074, AI-REISSUE-077).
	engagedFoesUnclipped bool
	// heldCircle: a registry Group's circle is the held one, not the stored
	// one (AI-740, AI-741).
	heldCircle bool
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
	guardNeedsMembers:          true,
	constructsForOlderRegistry: true,
	noHumanHealArm:             true,
	engagedFoesUnclipped:       true,
	heldCircle:                 true,
}

func (w *World) savedRoute(g *SavedGroup) uint8 {
	return w.game.groupRoute.route(g.AI[0x20])
}
