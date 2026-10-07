package data

// The unit animation descriptor — the block arithmetic of a unit's .256 sheet,
// derived from one RESOLVED class by pure integer arithmetic. Nothing is read
// from any sheet: the block bases are functions of the class's own phase
// scalars, and the predicted total is a computed property, never an
// assumption — a sheet may hold any frame count, a mismatch is data, and every
// consumer bounds-guards a selected index against the sheet's own count.
//
// Write MB, MV, AT, DY, BN, ID for MoveBeginPhases, MovePhases, AttackPhases,
// DyingPhases, BonePhases, IdlePhases. (S, D) — standing frames stored,
// directions stored per animated block — is (16, 8) at resolved Flip 0 and
// (9, 5) otherwise. The blocks:
//
//	standing       base 0                    length S
//	move           base S                    length D*(MB+MV) — per direction
//	                                         slot: MB wind-up, then MV loop
//	attack         base S + D*(MB+MV)        length D*AT
//	dying          base S + D*(MB+MV+AT)     length D*DY
//	bone and idle  base S + D*(MB+MV+AT+DY)  length D*BN / D*ID — ONE base
//	                                         serves both
//
// EVERY phase scalar above enters that arithmetic CLAMPED AT ZERO: an absent
// phase contributes no block. See phaseCount.
//
// A track — never a Phases scalar — is a cycle: its length is the period in
// ticks, its values sub-frame indices within one direction's slot. The Dying
// KEY is never consulted, and neither the dying BLOCK nor the bone block has a
// track — one is indexed by a run's own clock and the other by a decay stage —
// so DY and BN both enter as slot lengths beside the bases as well as entering
// the predicted total.
//
// THE ATTACK BLOCK HAS BOTH, and it is the last of the three to get them: AT is
// a slot length like DY and the Attack key pair expands like Move's and Idle's.
// The base was already computed here — every block after it needed it — so what
// arrives with the attack timeline is the slot, the track and the gate, and no
// arithmetic that was not already in this expression.

// UnitAnim is one class's descriptor: plain integers and two expanded tracks,
// every field derived and nothing the class does not imply. The bases, the
// strides and the total are computed over CLAMPED scalars (phaseCount); the
// gates are computed over the resolved ones, and a zero or negative phase
// count fails them here as it always did.
type UnitAnim struct {
	// S and D are the layout pair resolved from Flip: standing frames stored,
	// and directions stored per animated block.
	S, D int

	// The animated blocks' bases, in sheet order. TailBase serves bone AND
	// idle: the two blocks share it, which is why there is one field.
	MoveBase, AttackBase, DyingBase, TailBase int

	// MoveSlot is one direction's slot length in the move block, MB+MV, and
	// MoveWind the wind-up offset MB into it, where the MV loop frames begin.
	// DyingSlot is one direction's slot length past DyingBase, DY. IdleSlot and
	// BoneSlot are the two slot lengths past TailBase, ID and BN — the block is
	// one base and two lengths, which is why there is one base field and two of
	// these. AttackSlot is one direction's slot length past AttackBase, AT.
	MoveSlot, MoveWind, DyingSlot, IdleSlot, BoneSlot, AttackSlot int

	// Total is the predicted sheet total S + D*(MB+MV+AT+DY+max(BN, ID)) — a
	// computed property of the class, not a fact about any sheet.
	Total int

	// MoveTrack, IdleTrack and AttackTrack are the run-length expansions of the
	// class's Move, Idle and Attack AnimTime/AnimFrame pairs, order and
	// duplicates preserved.
	MoveTrack, IdleTrack, AttackTrack []int

	// MoveOK, IdleOK and AttackOK are the gates: the phase scalar positive AND
	// the track non-empty. A failed gate sends the selector to its fallback.
	MoveOK, IdleOK, AttackOK bool
}

// Anim derives the descriptor from the resolved class. It performs no IO,
// reads no sheet and names no float: integer arithmetic over c's fields, and
// nothing else.
func (c *UnitClass) Anim() UnitAnim {
	s, d := 16, 8
	if c.Flip != 0 {
		s, d = 9, 5
	}
	mb, mv := phaseCount(c.MoveBeginPhases), phaseCount(c.MovePhases)
	at, dy := phaseCount(c.AttackPhases), phaseCount(c.DyingPhases)
	bn, id := phaseCount(c.BonePhases), phaseCount(c.IdlePhases)

	a := UnitAnim{
		S:          s,
		D:          d,
		MoveBase:   s,
		AttackBase: s + d*(mb+mv),
		DyingBase:  s + d*(mb+mv+at),
		TailBase:   s + d*(mb+mv+at+dy),
		MoveSlot:   mb + mv,
		MoveWind:   mb,
		DyingSlot:  dy,
		IdleSlot:   id,
		// The bone block's own slot length. It shares TailBase with the idle
		// block and is indexed by a decay stage rather than by a track, so it
		// arrives as a slot length beside DyingSlot and takes no gate: a bool
		// equal to BoneSlot > 0 would be a second copy of one comparison.
		BoneSlot:   bn,
		AttackSlot: at,
		Total:      s + d*(mb+mv+at+dy+max(bn, id)),
		MoveTrack:  expandTrack(c.MoveAnimTime, c.MoveAnimFrame),
		IdleTrack:  expandTrack(c.IdleAnimTime, c.IdleAnimFrame),
		// The ATTACK BLOCK, derived by the same three rules the move and idle
		// blocks already take and over keys this loader already resolves: the
		// slot is the clamped phase count the base beside it is computed from,
		// the track the same run-length expansion, the gate the same
		// conjunction. Nothing new is learned about a sheet here.
		AttackTrack: expandTrack(c.AttackAnimTime, c.AttackAnimFrame),
	}
	// The gates test the RESOLVED scalar, and the clamp cannot reach them: it
	// maps every non-positive value to 0, which fails "> 0" exactly as the
	// value it replaced did. Reading mb..id here is therefore the same
	// predicate written over fewer names, not a second convention.
	a.MoveOK = mv > 0 && len(a.MoveTrack) > 0
	a.IdleOK = id > 0 && len(a.IdleTrack) > 0
	a.AttackOK = at > 0 && len(a.AttackTrack) > 0
	return a
}

// Timeline is one OBJECT class's animation cycle, expanded: the same
// run-length rule the unit tracks take, over the resolved AnimationTime /
// AnimationFrame pair.
//
// It is a DERIVATION and not a field, exactly as UnitAnim is: the class stores
// two arrays and the cycle is what walking them produces, so a stored timeline
// would be a second copy of the pair, free to disagree with it. The values are
// the registry's own — offsets a consumer adds to the class's Index — and
// nothing here learns what a frame is or where one is drawn.
//
// ITS LENGTH IS THE PERIOD, and an empty expansion means the class has NO
// CYCLE, whatever its Phases scalar says. Phases is set on classes carrying no
// arrays at all, so it does not answer the question and is not read here.
//
// The rule is expandTrack's, unchanged and shared: value i repeated Time[i]
// times, a non-positive time contributing nothing while still consuming its
// round, the walk ending when either array runs out. The loader validates the
// pair to one resolved length, but the rule is total over any two slices and
// nothing here assumes the equality.
func (c *ObjectClass) Timeline() []int { return expandTrack(c.AnimationTime, c.AnimationFrame) }

// Timeline is one STRUCTURE class's animation cycle, expanded: the same
// run-length rule the unit tracks and the object cycle take, over the
// resolved AnimTime / AnimFrame pair.
//
// It is a DERIVATION and not a field, for ObjectClass.Timeline's reason, and it
// is the SAME rule spelled over this registry's own two key names — which are
// AnimTime/AnimFrame here and AnimationTime/AnimationFrame there. Sharing
// expandTrack is what keeps the two registries from coming to expand a cycle
// differently; nothing else about the two paths is shared, and nothing here
// learns what a frame is or where one is drawn.
//
// ITS LENGTH IS THE PERIOD, and an empty expansion means the class has NO CYCLE.
// Phases is NOT consulted: ten structure classes spell Phases > 1 and then spell
// no timeline at all (SPR256-STR-041), so the scalar alone does not mean
// animated. The consumer conjoins this expansion with the Phases gate and with
// the mask's own length; none of those three tests belongs to a derivation over
// two arrays.
func (c *StructureClass) Timeline() []int { return expandTrack(c.AnimTime, c.AnimFrame) }

// phaseCount is a resolved phase scalar as the block arithmetic consumes it:
// clamped at zero, because an ABSENT PHASE CONTRIBUTES NO BLOCK.
//
// It is applied to every phase scalar rather than to the one that was caught:
// -1 is the default of MoveBeginPhases, MovePhases, AttackPhases, DyingPhases
// and BonePhases alike, and MB in particular reaches the move slot and the
// wind-up offset, which the selector reads on every walking frame.
func phaseCount(v int32) int { return max(int(v), 0) }

// expandTrack is the run-length rule: Frame[i] appended Time[i] times — an
// entry whose Time <= 0 appends nothing but still consumes its round — both
// heads dropped each round, ending when either side empties. Order and
// duplicates are preserved, so [0 1 2 1] survives as written. The loader
// validates a pair to one resolved length, but the rule is total over any two
// slices and nothing here assumes the equality.
func expandTrack(times, frames []int32) []int {
	var track []int
	for i := 0; i < min(len(times), len(frames)); i++ {
		for t := int32(0); t < times[i]; t++ {
			track = append(track, int(frames[i]))
		}
	}
	return track
}
