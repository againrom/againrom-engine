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
	// strikeCases: guard B's member cases 1, 2 and 4 run from the blow's
	// record and the pursuit stamp (R2-ENGINE-844, DIV-4503).
	strikeCases bool
}

// runsStrikeCases reports whether a Group at programme byte b runs guard B's
// member cases 1, 2 and 4 and the pursuit stamp: only programme 18 does;
// guard A has cases 3 and 6 (R2-ENGINE-844, R2-ENGINE-846).
func (p *groupRouteProfile) runsStrikeCases(b uint8) bool {
	return p.strikeCases && b == secondGuardRoute
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
	strikeCases:                false,
}

func (w *World) savedRoute(g *SavedGroup) uint8 {
	return w.game.groupRoute.route(g.AI[0x20])
}
