package sim

// InReach reports whether a stands close enough to t to strike it: the
// exported spelling of the very predicate resolveBlow consults before it
// resolves a blow (inReach, combat.go), over the same strike distance and the
// same Reach.
//
// IT EXISTS SO THERE IS ONE STRIKE-DISTANCE EXPRESSION IN THE TREE, and that
// is its whole justification as an exported name. The front end needs to ask
// "could this attacker actually reach what it is swinging at" — a sound
// question, not a simulation one — and the alternative was a second
// distance expression in pkg/game. strikeDistance uses both actors' token
// footprints, so the sound and the blow consult the same physical boundary.
//
// IT IS A PURE READ over two copies. Entities() hands out copies (world.go),
// which is what a front end holds, and this function writes nothing, draws
// nothing and steps nothing — so asking it cannot perturb the simulation or
// the generator's stream. That is what makes it safe to ask from outside the
// determinism wall.
//
// A CALLER OUTSIDE THIS PACKAGE MUST FIND THE TARGET ITSELF. There is no
// id-taking overload here on purpose: the attacker's own AttackTarget names
// an id the world may no longer hold, and what a front end should do about a
// vanished target is a front end's question — advanceAttack ends such an
// order at the attacker's own next turn, but that is a decision this
// predicate has no business making on a caller's behalf.
func InReach(a, t Entity) bool { return inReach(a, t) }
