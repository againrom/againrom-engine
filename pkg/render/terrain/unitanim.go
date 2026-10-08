package terrain

// UnitAnim is one class's animation descriptor as this layer selects from it:
// plain integers, two expanded tracks and two gates, filled by the loader
// from pkg/data's derivation. The values are carried, never validated — a
// negative or inconsistent field selects an index the guard refuses, which is
// the spec's answer to bad data (a mismatch is data, never an error).
type UnitAnim struct {
	// S and D are the layout pair resolved from Flip: standing frames
	// stored, and directions stored per animated block — (16, 8) at
	// resolved Flip 0 and (9, 5) otherwise.
	S, D int

	// The animated blocks' bases, in sheet order. TailBase serves bone AND
	// idle — the two blocks share it, which is why there is one field. The
	// live selection reads MoveBase and TailBase, the death selection
	// DyingBase, and the swing selection AttackBase — which rode along
	// unread until there was a swing to draw from it.
	MoveBase, AttackBase, DyingBase, TailBase int

	// MoveSlot is one direction's slot length in the move block (MB+MV),
	// MoveWind the wind-up offset (MB) into it, where the MV loop frames
	// begin. DyingSlot is one direction's slot length past DyingBase (DY),
	// IdleSlot one direction's slot length past TailBase (ID), and
	// AttackSlot one direction's slot length past AttackBase (AT).
	//
	// BoneSlot is the other slot length past TailBase (BN) — the block is one
	// base and two lengths, which is why there is one base field and two of
	// these.
	//
	// Neither DyingSlot nor BoneSlot carries a gate beside it. MoveOK, IdleOK and
	// AttackOK exist because each combines a positive phase count with a
	// non-empty track, and neither of these blocks has a track: one is indexed by
	// a run's own clock and the other by a decay stage. A bool equal to
	// BoneSlot > 0 would be a second copy of one comparison, free to disagree
	// with it, so each selection tests the length itself.
	MoveSlot, MoveWind, DyingSlot, IdleSlot, BoneSlot, AttackSlot int

	// Total is the predicted sheet total — a computed property of the
	// class, not a fact about any sheet. The selector never reads it: the
	// guard measures against the sheet's OWN count, handed in per call.
	Total int

	// MoveTrack, IdleTrack and AttackTrack are the run-length expanded
	// cycles: length is the period in ticks, values are sub-frame indices
	// within one direction's slot.
	MoveTrack, IdleTrack, AttackTrack []int

	// MoveOK, IdleOK and AttackOK are the gates the descriptor derived: the
	// phase scalar positive AND the track non-empty.
	MoveOK, IdleOK, AttackOK bool
}

func SelectUnitFrame(a UnitAnim, frameCount int, moving bool, oct, tick, odo int) (frame int, mirror bool) {
	switch {
	case moving && a.MoveOK && len(a.MoveTrack) > 0:
		slot, m := unitSlot(a.D, oct)
		frame = a.MoveBase + slot*a.MoveSlot + a.MoveWind + a.MoveTrack[unitStep(WalkPhase(odo), len(a.MoveTrack))]
		mirror = m
	case a.IdleOK && len(a.IdleTrack) > 0:
		slot, m := unitSlot(a.D, oct)
		frame = a.TailBase + slot*a.IdleSlot + a.IdleTrack[unitStep(tick, len(a.IdleTrack))]
		mirror = m
	default:
		// Standing is 16-way over the eight octants, g = 2*oct. The S 9
		// layout stores frames 0..8 and mirrors the rest as 16-g; every
		// other layout draws g plain — at S 16 that is the whole block,
		// and any other S yields an index for the guard to judge.
		g := 2 * oct
		if a.S == 9 && g > 8 {
			frame, mirror = 16-g, true
		} else {
			frame, mirror = g, false
		}
	}
	if frame < 0 || frame >= frameCount {
		return 0, false
	}
	return frame, mirror
}

func SelectStandingFrame(a UnitAnim, frameCount, facing int) (frame int, mirror bool) {
	frame = facing
	if a.S == 9 && facing > 8 {
		frame, mirror = 16-facing, true
	}
	if frame < 0 || frame >= frameCount {
		return 0, false
	}
	return frame, mirror
}

// deathPhaseTicks is how long ONE dying frame is held: two ticks
// (ANIM-DEATH-007, TERR-SPR-047).
//
// It is DECODED AND NOT TUNED. The engine sizes a fall at twice the dying
// phase count in ticks and draws that run's own clock halved, which is exactly
// what plays the block once, and the halving is a named instruction. So this
// is a number read off the thing being reconstructed, not a constant chosen to
// look right — there is nothing here to turn.
const deathPhaseTicks = 2

// SelectDeathFrame is the selection for an entity that is not alive: a pure
// integer function of the CORPSE class's descriptor, that class's own frame
// count, the facing octant and how many ticks have passed since death,
// returning the sheet frame to draw, whether it draws reflected, and whether
// there is a frame at all.
//
// The whole of it is one clamped expression:
//
//	slot, mirror = the animated blocks' direction rule, at the CORPSE class's D
//	phase        = min(elapsed / deathPhaseTicks, DyingSlot - 1)
//	frame        = DyingBase + slot*DyingSlot + phase
//
// and that ONE clamp covers both halves of what the engine does. Inside the
// run — the first 2*DyingSlot ticks — the division is the engine's own frame
// per two ticks. Past it the clamp holds the block's last frame, which is what
// the engine's corpse fork freezes on at the first decay stage. Written as two
// branches they would be two places for the boundary frame to be off by one.
//
// The phase is clamped BELOW as well, so a negative elapsed — which no caller
// can produce, the count being a scene clock less a stamp taken from it — is
// the first frame rather than an index before the block. Go's division
// truncates toward zero, so the clamp and not the division is what makes that
// total.
//
// The direction rule and the mirror are unitSlot's, SHARED with the live
// selection rather than copied: every block but the standing one halves the
// facing the same way, and the layout switch mirrors the same half. It is read
// off the corpse class's own D, because the substitution replaces the layout
// switch along with the sheet.
//
// THE THIRD RETURN IS THE POINT OF A SEPARATE FUNCTION. SelectUnitFrame
// answers a refused index as sheet frame 0 unmirrored — the picture drawn
// before it existed, which is the right total answer there. Here a refusal
// must reach the caller's fallback, so it is reported: false for a class whose
// dying block is absent (DyingSlot <= 0, which an absent phase clamps to) and
// for any index outside [0, frameCount). No input panics, no division by zero
// occurs, and equal inputs give equal answers.
func SelectDeathFrame(a UnitAnim, frameCount, oct, elapsed int) (frame int, mirror bool, ok bool) {
	if a.DyingSlot <= 0 {
		return 0, false, false
	}
	phase := elapsed / deathPhaseTicks
	if phase < 0 {
		phase = 0
	}
	if phase > a.DyingSlot-1 {
		phase = a.DyingSlot - 1
	}
	slot, m := unitSlot(a.D, oct)
	frame = a.DyingBase + slot*a.DyingSlot + phase
	if frame < 0 || frame >= frameCount {
		return 0, false, false
	}
	return frame, m, true
}

// boneStageFloor is the first decay stage drawn from the bone block, and the
// stage the bone index is measured from: at it the block's first frame is drawn,
// one rung deeper its second, and so on.
//
// It is a NUMBER FROM THE SIMULATION and this tier does not interpret it — the
// stage crosses the seam as a plain integer and nothing here asks what a body
// is — so the one thing this constant fixes is where the drawn block starts,
// which is this file's own question.
const boneStageFloor = 2

// SelectBoneFrame is the selection for a body past its fall: a pure integer
// function of the CORPSE class's descriptor, that class's own frame count,
// the facing octant and the decay stage, returning the sheet frame to draw,
// whether it draws reflected, and whether there is a frame at all.
//
// The whole of it is one expression:
//
//	slot, mirror = the animated blocks' direction rule, at the CORPSE class's D
//	frame        = TailBase + slot*BoneSlot + (stage - boneStageFloor)
//
// THE DIRECTION TERM IS THE POINT, and it is the one thing about this rule a
// consumer gets wrong. Stated at its shortest the rule reads "a body at stage
// two and beyond draws bone frame stage minus two", which is an ABSOLUTE index
// into the block and carries no facing at all — and the instruction it is
// derived from shows none either, the direction having been folded into the
// running base before it. Written that way every body's bones come out of
// direction slot zero: right for a body that fell facing north, wrong for the
// other seven octants, and at the five-way layout wrong a second time, since a
// fold that never happens never mirrors.
//
// What settles it is a class whose bone phase count is the loader's ABSENT
// value: its bone frame is documented as the base MINUS the direction plus the
// stage offset, and that is this expression at a slot length of minus one and at
// no other value. A formula that degenerates to a negative direction because a
// phase count is negative must carry the direction times that count in general.
//
// The direction rule and the mirror are unitSlot's, SHARED with the three
// selections above rather than copied, and read off the corpse class's own D
// because the substitution replaces the layout switch along with the sheet.
//
// THE THIRD RETURN IS THE POINT OF A SEPARATE FUNCTION, exactly as it is for
// SelectDeathFrame: a refusal must reach the caller's fallback rather than
// arrive as a frame it cannot tell from a real one. It is false for a class
// whose bone block is absent (BoneSlot <= 0, which an absent phase count clamps
// to), for a stage that is not drawn from bones at all, and for any index
// outside [0, frameCount). No input panics, no division occurs, and equal inputs
// give equal answers.
//
// It is a FOURTH selection rather than an argument on the third. The two answer
// different blocks off different clocks — one a run clock, one a stage — so a
// merged function would take an argument that means nothing on one of its two
// arms, which is what the one-selection-per-drawn-state rule above exists to
// prevent.
func SelectBoneFrame(a UnitAnim, frameCount, oct, stage int) (frame int, mirror bool, ok bool) {
	if a.BoneSlot <= 0 || stage < boneStageFloor {
		return 0, false, false
	}
	slot, m := unitSlot(a.D, oct)
	frame = a.TailBase + slot*a.BoneSlot + (stage - boneStageFloor)
	if frame < 0 || frame >= frameCount {
		return 0, false, false
	}
	return frame, m, true
}

// SelectAttackFrame is the selection for a living entity mid-swing: a pure
// integer function of the descriptor, the sheet's own frame count, the
// facing octant and the swing clock, returning the sheet frame to draw,
// whether it draws reflected, and whether there is a frame at all.
//
// The whole of it is one expression:
//
//	slot, mirror = the animated blocks' direction rule, at this class's D
//	step         = swing reduced euclidean mod len(AttackTrack)
//	frame        = AttackBase + slot*AttackSlot + AttackTrack[step]
//
// It is SelectUnitFrame's moving arm over the attack block rather than the move
// one, and the direction rule and the mirror are unitSlot's, SHARED with both
// selections rather than copied: every block but the standing one halves the
// facing the same way.
//
// THE THIRD RETURN IS THE POINT OF A SEPARATE FUNCTION, exactly as it is for
// SelectDeathFrame. SelectUnitFrame answers a refused index as sheet frame 0
// unmirrored — the right total answer there, because there is nothing further to
// fall through to. Here a refusal must reach the caller's fallback, so it is
// reported: false for a class whose attack block is absent (AttackSlot <= 0,
// which an absent phase count clamps to), false for a failed gate or an empty
// track, and false for any index outside [0, frameCount).
//
// THE RUN IS SIZED FROM THE ART AND IT PLAYS ONCE (ANIM-RUN-004,
// ANIM-PHASE-003, ANIM-STATE-023). The attack arm's clock counts one per tick
// from zero and indexes the expanded track with NO MODULO — the only arm of the
// engine's own switch that takes one is the move arm — and the run lasts exactly
// len(AttackAnimTime expansion) ticks, which is this track's length. So no index
// can fall outside the run, and when the run counter reaches zero the drawn
// state is forced back to 0 and the unit stands.
//
// This function is that ending: a clock at or past the track's length is a run
// that is OVER, and it is REFUSED so the caller falls through to the live
// selection — which is what "the state is forced to 0" looks like from here. A
// modulus in its place would loop the swing forever, which is the arm's
// behaviour nowhere.
//
// THE CLOCK IS NOT THE ATTACK CYCLE'S, and that is decoded too rather than
// conceded: the swing frame, the swing sound and the damage are scheduled from
// three different numbers — the art's length, AttackDelay and attackChargeTime —
// and binding any two of them together reproduces the original on at most 2 of
// the 157 shipped pairs that answer (ANIM-CLOCK-024). So a swing that landed on
// the frame the damage does would be less faithful here, not more.
//
// A NEGATIVE clock is refused with the rest: it is out of the run. No input
// panics, no division occurs at all, and equal inputs give equal answers — the
// function reads nothing but its arguments and writes nothing.
func SelectAttackFrame(a UnitAnim, frameCount, oct, swing int) (frame int, mirror bool, ok bool) {
	if a.AttackSlot <= 0 || !a.AttackOK || len(a.AttackTrack) == 0 {
		return 0, false, false
	}
	if swing < 0 || swing >= len(a.AttackTrack) {
		return 0, false, false
	}
	slot, m := unitSlot(a.D, oct)
	frame = a.AttackBase + slot*a.AttackSlot + a.AttackTrack[swing]
	if frame < 0 || frame >= frameCount {
		return 0, false, false
	}
	return frame, m, true
}

// unitSlot is the direction rule for the animated blocks: the direction slot
// an octant selects, and whether it draws mirrored. At D 5 octants 5..7 fold
// onto slots 3..1 about the vertical axis; every other D takes the octant as
// the slot, plain — at D 8 that is the whole rule, and any other D or an
// out-of-range octant yields a slot whose index the guard refuses.
func unitSlot(d, oct int) (slot int, mirror bool) {
	if d == 5 && oct > 4 {
		return 8 - oct, true
	}
	return oct, false
}

// unitStep is the euclidean reduction of a tick to a track step: the
// remainder in [0, period) at ANY int tick. Go's % truncates toward zero, so
// a negative tick's remainder lands in (-period, 0] and one addition folds it
// back. The caller guarantees period > 0 — both call sites gate on a
// non-empty track first.
func unitStep(tick, period int) int {
	s := tick % period
	if s < 0 {
		s += period
	}
	return s
}
