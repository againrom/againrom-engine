package data

import (
	"reflect"
	"slices"
	"testing"
)

// The animation descriptor over hand-written literals (0024 SC-2, AC-2).
//
// Anim() is a pure integer function of one resolved class, so the fixtures are
// class literals — no registry, no loader, no sheet, no IO. Every expected
// base, stride, total and track below is hand-computed from the sheet
// contract, never captured from Anim() itself: an expectation read back from
// the function under test would pin nothing.

// The spec's I/O example class: Flip 1, so (S, D) = (9, 5); MB=1, MV=2, AT=2,
// DY=2, BN=2, ID=0; move pair Time=[2,1], Frame=[0,1].
func specExampleClass() *UnitClass {
	return &UnitClass{
		Flip:            1,
		MoveBeginPhases: 1,
		MovePhases:      2,
		AttackPhases:    2,
		DyingPhases:     2,
		BonePhases:      2,
		MoveAnimTime:    []int32{2, 1},
		MoveAnimFrame:   []int32{0, 1},
	}
}

// The bases, strides and totals at both (S, D) pairs, every value a literal
// from the sheet-contract table. The example class has BN > ID and the Flip-0
// class ID > BN, so the predicted total's max(BN, ID) is witnessed from both
// sides — over the ONE TailBase that serves bone and idle alike.
func TestAnimBasesStridesAndTotals(t *testing.T) {
	// The spec's example, worked in its I/O section: bases 9/24/34/44,
	// total 54 = 9 + 5*(1+2+2+2+2), move track [0,0,1], period 3.
	a := specExampleClass().Anim()
	if a.S != 9 || a.D != 5 {
		t.Errorf("(S, D) = (%d, %d), want (9, 5) — resolved Flip is 1", a.S, a.D)
	}
	if a.MoveBase != 9 || a.AttackBase != 24 || a.DyingBase != 34 || a.TailBase != 44 {
		t.Errorf("bases = %d/%d/%d/%d, want 9/24/34/44",
			a.MoveBase, a.AttackBase, a.DyingBase, a.TailBase)
	}
	if a.MoveSlot != 3 || a.MoveWind != 1 || a.DyingSlot != 2 || a.IdleSlot != 0 {
		t.Errorf("MoveSlot/MoveWind/DyingSlot/IdleSlot = %d/%d/%d/%d, want 3/1/2/0",
			a.MoveSlot, a.MoveWind, a.DyingSlot, a.IdleSlot)
	}
	if a.Total != 54 {
		t.Errorf("Total = %d, want 54 — the predicted total 9 + 5*(1+2+2+2+2)", a.Total)
	}
	if want := []int{0, 0, 1}; !slices.Equal(a.MoveTrack, want) {
		t.Errorf("MoveTrack = %v, want %v — period 3", a.MoveTrack, want)
	}
	if !a.MoveOK || a.IdleOK {
		t.Errorf("MoveOK/IdleOK = %v/%v, want true/false — ID is 0 and no idle pair", a.MoveOK, a.IdleOK)
	}

	// The other layout: Flip 0, so (S, D) = (16, 8); MB=2, MV=3, AT=1, DY=4,
	// BN=0, ID=5. Bases 16/56/64/96; total 136 = 16 + 8*(2+3+1+4+5), the tail
	// counted at ID since ID > BN.
	b := (&UnitClass{
		MoveBeginPhases: 2,
		MovePhases:      3,
		AttackPhases:    1,
		DyingPhases:     4,
		IdlePhases:      5,
		MoveAnimTime:    []int32{1},
		MoveAnimFrame:   []int32{2},
		IdleAnimTime:    []int32{1, 1},
		IdleAnimFrame:   []int32{4, 0},
	}).Anim()
	if b.S != 16 || b.D != 8 {
		t.Errorf("(S, D) = (%d, %d), want (16, 8) — resolved Flip is 0", b.S, b.D)
	}
	if b.MoveBase != 16 || b.AttackBase != 56 || b.DyingBase != 64 || b.TailBase != 96 {
		t.Errorf("bases = %d/%d/%d/%d, want 16/56/64/96",
			b.MoveBase, b.AttackBase, b.DyingBase, b.TailBase)
	}
	if b.MoveSlot != 5 || b.MoveWind != 2 || b.DyingSlot != 4 || b.IdleSlot != 5 {
		t.Errorf("MoveSlot/MoveWind/DyingSlot/IdleSlot = %d/%d/%d/%d, want 5/2/4/5",
			b.MoveSlot, b.MoveWind, b.DyingSlot, b.IdleSlot)
	}
	if b.Total != 136 {
		t.Errorf("Total = %d, want 136 — the predicted total 16 + 8*(2+3+1+4+5)", b.Total)
	}
	if want := []int{2}; !slices.Equal(b.MoveTrack, want) {
		t.Errorf("MoveTrack = %v, want %v", b.MoveTrack, want)
	}
	if want := []int{4, 0}; !slices.Equal(b.IdleTrack, want) {
		t.Errorf("IdleTrack = %v, want %v", b.IdleTrack, want)
	}
	if !b.MoveOK || !b.IdleOK {
		t.Errorf("MoveOK/IdleOK = %v/%v, want true/true", b.MoveOK, b.IdleOK)
	}
}

// The run-length rule, through Anim() over the move pair: Frame[i] appended
// Time[i] times, a Time <= 0 entry appending nothing but consuming its round,
// order and duplicates preserved.
func TestAnimTrackExpansion(t *testing.T) {
	for _, tc := range []struct {
		label        string
		times, frame []int32
		want         []int
	}{
		{"the spec example: [2,1]x[0,1]", []int32{2, 1}, []int32{0, 1}, []int{0, 0, 1}},
		{"a ping-pong survives as written", []int32{1, 1, 1, 1}, []int32{0, 1, 2, 1}, []int{0, 1, 2, 1}},
		{"a zero-time entry adds no tick", []int32{2, 0, 1}, []int32{5, 6, 7}, []int{5, 5, 7}},
		{"an empty pair expands to nothing", nil, nil, nil},
	} {
		c := &UnitClass{MovePhases: 1, MoveAnimTime: tc.times, MoveAnimFrame: tc.frame}
		if got := c.Anim().MoveTrack; !slices.Equal(got, tc.want) {
			t.Errorf("%s: MoveTrack = %v, want %v", tc.label, got, tc.want)
		}
	}
}

// A Phases scalar is never a track length: a MovePhases exceeding its pair's
// expanded length leaves the period the TRACK's. The scalar sizes the sheet's
// slot; the track alone is the cycle.
func TestAnimPhasesScalarIsNeverALength(t *testing.T) {
	c := &UnitClass{
		MovePhases:    9,
		MoveAnimTime:  []int32{1, 1, 1},
		MoveAnimFrame: []int32{0, 1, 0},
	}
	a := c.Anim()
	if want := []int{0, 1, 0}; !slices.Equal(a.MoveTrack, want) {
		t.Errorf("MoveTrack = %v, want %v — period 3, not MovePhases' 9", a.MoveTrack, want)
	}
	if !a.MoveOK {
		t.Error("MoveOK = false, want true — MV > 0 and the track is non-empty")
	}
}

// Dying is never consulted: two classes differing only there derive one
// descriptor, field for field.
func TestAnimNeverConsultsDying(t *testing.T) {
	plain := specExampleClass()
	sentinel := specExampleClass()
	sentinel.Dying = 9999
	if a, b := plain.Anim(), sentinel.Anim(); !reflect.DeepEqual(a, b) {
		t.Errorf("descriptors differ on Dying alone:\n  %+v\n  %+v", a, b)
	}
}

func TestAnimClampsAnAbsentPhaseToNoBlock(t *testing.T) {
	// DyingPhases absent, and BonePhases with it — the shape the corpus audit
	// found. Flip 1, so (S, D) = (9, 5); MB=1, MV=2, AT=2, DY=-1, BN=-1, ID=0.
	// Bases 9/24/34/34 — the dying block is EMPTY, so the tail sits where the
	// dying block starts — and total 34 = 9 + 5*(1+2+2+0+0). Unclamped this
	// class gave TailBase 29 and total 29, a whole direction block short.
	absentDying := &UnitClass{
		Flip:            1,
		MoveBeginPhases: 1,
		MovePhases:      2,
		AttackPhases:    2,
		DyingPhases:     -1,
		BonePhases:      -1,
		MoveAnimTime:    []int32{2, 1},
		MoveAnimFrame:   []int32{0, 1},
	}
	a := absentDying.Anim()
	if a.MoveBase != 9 || a.AttackBase != 24 || a.DyingBase != 34 || a.TailBase != 34 {
		t.Errorf("bases = %d/%d/%d/%d, want 9/24/34/34 — an absent dying block occupies nothing",
			a.MoveBase, a.AttackBase, a.DyingBase, a.TailBase)
	}
	if a.Total != 34 {
		t.Errorf("Total = %d, want 34 = 9 + 5*(1+2+2+0+0) — the absent phase adds no block "+
			"and subtracts none either", a.Total)
	}

	// The sentinel and a resolved 0 are the same class to this function.
	zeroed := *absentDying
	zeroed.DyingPhases, zeroed.BonePhases = 0, 0
	if b := zeroed.Anim(); !reflect.DeepEqual(a, b) {
		t.Errorf("a class resolving -1 and one resolving 0 derive different descriptors:\n  %+v\n  %+v", a, b)
	}

	// MoveBeginPhases absent: the clamp reaches the move SLOT and the wind-up
	// offset, which the selector reads on every walking frame. Flip 0, so
	// (S, D) = (16, 8); MB=-1, MV=3, AT=1, DY=4, BN=0, ID=5. Slot 3, wind-up 0,
	// bases 16/40/48/80, total 120 = 16 + 8*(0+3+1+4+5). Unclamped: slot 2,
	// wind-up -1, and an attack base of 32.
	c := (&UnitClass{
		MoveBeginPhases: -1,
		MovePhases:      3,
		AttackPhases:    1,
		DyingPhases:     4,
		IdlePhases:      5,
		MoveAnimTime:    []int32{1},
		MoveAnimFrame:   []int32{0},
		IdleAnimTime:    []int32{1},
		IdleAnimFrame:   []int32{0},
	}).Anim()
	if c.MoveSlot != 3 || c.MoveWind != 0 {
		t.Errorf("MoveSlot/MoveWind = %d/%d, want 3/0 — an absent wind-up is no frames, not one taken away",
			c.MoveSlot, c.MoveWind)
	}
	if c.MoveBase != 16 || c.AttackBase != 40 || c.DyingBase != 48 || c.TailBase != 80 {
		t.Errorf("bases = %d/%d/%d/%d, want 16/40/48/80", c.MoveBase, c.AttackBase, c.DyingBase, c.TailBase)
	}
	if c.Total != 120 {
		t.Errorf("Total = %d, want 120 = 16 + 8*(0+3+1+4+5)", c.Total)
	}

	// ...and the clamp does not open a gate: a -1 phase is still absent.
	if d := (&UnitClass{MovePhases: -1, IdlePhases: -1,
		MoveAnimTime: []int32{1}, MoveAnimFrame: []int32{0},
		IdleAnimTime: []int32{1}, IdleAnimFrame: []int32{0}}).Anim(); d.MoveOK || d.IdleOK {
		t.Errorf("MoveOK/IdleOK = %v/%v on -1 phases, want false/false — the clamp is the "+
			"arithmetic's, not the gates'", d.MoveOK, d.IdleOK)
	}
}

// The dying block's own per-direction slot length (SC-1, AC-1).
//
// It is the one quantity the block arithmetic already computed with and never
// named: the tail base is the dying base plus D whole dying slots, so the two
// cannot be checked against each other unless the length is carried. Every
// expectation below is hand-computed from the sheet-contract table, and the
// identity is asserted from the OTHER SIDE — the tail base this function
// derives independently — so a slot length taken from the wrong scalar moves
// one side of the comparison and not the other.
func TestAnimDyingSlotIsTheDyingBlocksOwnLength(t *testing.T) {
	for _, tc := range []struct {
		label string
		class *UnitClass
		want  int
	}{
		{"the spec example, DY 2 at (9, 5)", specExampleClass(), 2},
		{"DY 7 at (16, 8), with blocks either side of it", &UnitClass{
			MoveBeginPhases: 2, MovePhases: 3, AttackPhases: 1,
			DyingPhases: 7, BonePhases: 3, IdlePhases: 1}, 7},
		{"DY 1 — a one-frame fall", &UnitClass{DyingPhases: 1}, 1},
		{"DY absent: no block, not a block of -1", &UnitClass{
			Flip: 1, MovePhases: 2, DyingPhases: -1, BonePhases: -1}, 0},
		{"DY resolved 0, the twin of the sentinel", &UnitClass{
			Flip: 1, MovePhases: 2, DyingPhases: 0, BonePhases: 0}, 0},
	} {
		a := tc.class.Anim()
		if a.DyingSlot != tc.want {
			t.Errorf("%s: DyingSlot = %d, want %d", tc.label, a.DyingSlot, tc.want)
		}
		// The dying block runs from its base to the base of the block after
		// it, D slots of DyingSlot frames each and nothing over.
		if got := a.DyingBase + a.D*a.DyingSlot; got != a.TailBase {
			t.Errorf("%s: DyingBase + D*DyingSlot = %d, but TailBase is %d — the dying "+
				"block does not tile the space between the two bases", tc.label, got, a.TailBase)
		}
	}
}

// The gates: the phase scalar must be positive AND the track non-empty, each
// failing alone. -1 is now the absent-everywhere default of every phase
// scalar's kin, so the negative case is a resolved value, not a contrivance.
func TestAnimGates(t *testing.T) {
	pair := func(c *UnitClass) *UnitClass { // a one-entry move AND idle pair
		c.MoveAnimTime, c.MoveAnimFrame = []int32{1}, []int32{0}
		c.IdleAnimTime, c.IdleAnimFrame = []int32{1}, []int32{0}
		return c
	}

	for _, tc := range []struct {
		label          string
		c              *UnitClass
		moveOK, idleOK bool
	}{
		{"positive scalars and non-empty tracks pass",
			pair(&UnitClass{MovePhases: 1, IdlePhases: 1}), true, true},
		{"a zero scalar fails its gate with the track non-empty",
			pair(&UnitClass{MovePhases: 0, IdlePhases: 0}), false, false},
		{"a negative scalar fails the same way",
			pair(&UnitClass{MovePhases: -1, IdlePhases: -1}), false, false},
		{"an empty pair fails with the scalar positive",
			&UnitClass{MovePhases: 1, IdlePhases: 1}, false, false},
		{"an all-zero Time expands empty and fails too",
			&UnitClass{MovePhases: 1, IdlePhases: 1,
				MoveAnimTime: []int32{0, 0}, MoveAnimFrame: []int32{3, 4},
				IdleAnimTime: []int32{0}, IdleAnimFrame: []int32{3}}, false, false},
	} {
		a := tc.c.Anim()
		if a.MoveOK != tc.moveOK || a.IdleOK != tc.idleOK {
			t.Errorf("%s: MoveOK/IdleOK = %v/%v, want %v/%v",
				tc.label, a.MoveOK, a.IdleOK, tc.moveOK, tc.idleOK)
		}
	}
}

func TestAnimAttackSlotTrackAndGate(t *testing.T) {
	for _, tc := range []struct {
		label     string
		c         *UnitClass
		wantSlot  int
		wantTrack []int
		wantOK    bool
	}{
		{"a phase count and a pair",
			&UnitClass{AttackPhases: 4,
				AttackAnimTime:  []int32{2, 1, 3},
				AttackAnimFrame: []int32{0, 2, 1}},
			4, []int{0, 0, 2, 1, 1, 1}, true},
		{"an ABSENT phase clamps to no block, and the gate fails with it",
			&UnitClass{AttackPhases: -1,
				AttackAnimTime:  []int32{1},
				AttackAnimFrame: []int32{0}},
			0, []int{0}, false},
		{"a zero phase count fails the gate with the track non-empty",
			&UnitClass{AttackPhases: 0,
				AttackAnimTime:  []int32{1},
				AttackAnimFrame: []int32{0}},
			0, []int{0}, false},
		{"a positive phase count fails the gate with no pair at all",
			&UnitClass{AttackPhases: 3}, 3, nil, false},
		{"an all-zero Time expands empty and fails too",
			&UnitClass{AttackPhases: 3,
				AttackAnimTime:  []int32{0, 0},
				AttackAnimFrame: []int32{3, 4}},
			3, nil, false},
		{"the two arrays run to the shorter of them",
			&UnitClass{AttackPhases: 2,
				AttackAnimTime:  []int32{1, 1, 1},
				AttackAnimFrame: []int32{5}},
			2, []int{5}, true},
	} {
		a := tc.c.Anim()
		if a.AttackSlot != tc.wantSlot {
			t.Errorf("%s: AttackSlot = %d, want %d", tc.label, a.AttackSlot, tc.wantSlot)
		}
		if len(a.AttackTrack) != len(tc.wantTrack) {
			t.Errorf("%s: AttackTrack = %v, want %v", tc.label, a.AttackTrack, tc.wantTrack)
		} else {
			for i := range tc.wantTrack {
				if a.AttackTrack[i] != tc.wantTrack[i] {
					t.Errorf("%s: AttackTrack = %v, want %v", tc.label, a.AttackTrack, tc.wantTrack)
					break
				}
			}
		}
		if a.AttackOK != tc.wantOK {
			t.Errorf("%s: AttackOK = %v, want %v", tc.label, a.AttackOK, tc.wantOK)
		}
	}
}

// TestAnimAttackSlotIsTheBlockItsOwnBaseMeasures is AC-7's structural half: the
// slot the descriptor now carries is the same AT the bases beside it are already
// computed from, so the attack block's own extent is (AttackBase, AttackBase +
// D*AttackSlot) and the dying block begins exactly where it ends.
//
// Written as the relation between the fields rather than as three numbers,
// because what would break is the slot being taken from a different scalar than
// the base — which any pair of literals would also satisfy by coincidence.
func TestAnimAttackSlotIsTheBlockItsOwnBaseMeasures(t *testing.T) {
	for _, c := range []*UnitClass{
		{MoveBeginPhases: 2, MovePhases: 3, AttackPhases: 4, DyingPhases: 5, IdlePhases: 6},
		{Flip: 1, MoveBeginPhases: 1, MovePhases: 1, AttackPhases: 9, DyingPhases: 2},
		{AttackPhases: -1, DyingPhases: 3},
		{AttackPhases: 0, DyingPhases: 3},
	} {
		a := c.Anim()
		if got, want := a.AttackBase+a.D*a.AttackSlot, a.DyingBase; got != want {
			t.Errorf("AttackPhases %d: the attack block ends at %d and the dying block begins at %d",
				c.AttackPhases, got, want)
		}
	}
}

func TestStructureTimelineIsTheSameExpansion(t *testing.T) {
	for _, c := range []struct {
		name          string
		times, frames []int32
		want          []int
	}{
		{"the ordinary pair", []int32{2, 1}, []int32{0, 1}, []int{0, 0, 1}},
		{"a non-positive time appends nothing but consumes its round",
			[]int32{0, 2}, []int32{7, 9}, []int{9, 9}},
		{"a negative time is the other half of non-positive",
			[]int32{-3, 1}, []int32{7, 9}, []int{9}},
		{"the walk ends when either array runs out",
			[]int32{1, 1, 1}, []int32{4, 5}, []int{4, 5}},
		{"no pair at all", nil, nil, nil},
		{"order and duplicates are preserved",
			[]int32{1, 1, 1, 1}, []int32{0, 1, 2, 1}, []int{0, 1, 2, 1}},
	} {
		t.Run(c.name, func(t *testing.T) {
			sc := &StructureClass{AnimTime: c.times, AnimFrame: c.frames}
			if got := sc.Timeline(); !slices.Equal(got, c.want) {
				t.Errorf("Timeline() = %v, want %v", got, c.want)
			}
			// The identical pair through the object class's own method: one rule,
			// two key spellings.
			oc := &ObjectClass{AnimationTime: c.times, AnimationFrame: c.frames}
			if got, want := sc.Timeline(), oc.Timeline(); !slices.Equal(got, want) {
				t.Errorf("the structure expansion is %v and the object one %v", got, want)
			}
		})
	}
}

// Phases is NOT consulted: ten shipped structure classes spell Phases > 1 and
// then spell no timeline at all, so the scalar alone does not say a class
// animates and this derivation must not read it (SPR256-STR-041).
func TestStructureTimelineIgnoresPhases(t *testing.T) {
	loud := &StructureClass{Phases: 17}
	if got := loud.Timeline(); len(got) != 0 {
		t.Errorf("Timeline() = %v over Phases 17 and no arrays, want empty", got)
	}
	quiet := &StructureClass{Phases: 0, AnimTime: []int32{2}, AnimFrame: []int32{3}}
	if got, want := quiet.Timeline(), []int{3, 3}; !slices.Equal(got, want) {
		t.Errorf("Timeline() = %v over Phases 0 and a usable pair, want %v", got, want)
	}
}

// TestAnimBoneSlotIsTheBoneBlocksOwnLength is 0089 AC-11 on this tier: the bone
// block's slot length is the class's own BonePhases clamped at zero, it sits
// past the ONE TailBase the bone and idle blocks share, and admitting it moves
// no base and no total.
//
// The last clause is the one worth asserting rather than assuming. The predicted
// total already counted this block through max(BN, ID), so a derivation that
// gained a length and also gained a term would be counting it twice, and every
// class whose bone block is the longer of the two would come out over.
func TestAnimBoneSlotIsTheBoneBlocksOwnLength(t *testing.T) {
	for _, tc := range []struct {
		label string
		class *UnitClass
		want  int
	}{
		{"the spec example, BN 2 at (9, 5)", specExampleClass(), 2},
		{"BN 3 at (16, 8), a block either side of it", &UnitClass{
			MoveBeginPhases: 2, MovePhases: 3, AttackPhases: 1,
			DyingPhases: 7, BonePhases: 3, IdlePhases: 1}, 3},
		{"BN longer than ID, so the tail is the bone block's", &UnitClass{
			DyingPhases: 2, BonePhases: 5, IdlePhases: 1}, 5},
		{"BN absent: no block, not a block of -1", &UnitClass{
			Flip: 1, MovePhases: 2, DyingPhases: 2, BonePhases: -1}, 0},
		{"BN resolved 0, the twin of the sentinel", &UnitClass{
			Flip: 1, MovePhases: 2, DyingPhases: 2, BonePhases: 0}, 0},
	} {
		a := tc.class.Anim()
		if a.BoneSlot != tc.want {
			t.Errorf("%s: BoneSlot = %d, want %d", tc.label, a.BoneSlot, tc.want)
		}
		// The bone block starts where the idle block does and ends inside the
		// space the total already reserved for the longer of the two.
		if got := a.TailBase + a.D*a.BoneSlot; got > a.Total {
			t.Errorf("%s: TailBase + D*BoneSlot = %d, past the predicted total %d",
				tc.label, got, a.Total)
		}
	}
}

// TestAnimBoneSlotMovesNoBaseAndNoTotal pins the addition as ADDITIVE against a
// derivation computed with the same inputs: every other field of the descriptor
// is what it was, so a reader of an older pinned descriptor is not reading a
// different sheet.
func TestAnimBoneSlotMovesNoBaseAndNoTotal(t *testing.T) {
	c := &UnitClass{MoveBeginPhases: 2, MovePhases: 3, AttackPhases: 1,
		DyingPhases: 7, BonePhases: 3, IdlePhases: 1}
	a := c.Anim()
	if a.MoveBase != 16 || a.AttackBase != 56 || a.DyingBase != 64 || a.TailBase != 120 {
		t.Errorf("bases = %d/%d/%d/%d, want 16/56/64/120",
			a.MoveBase, a.AttackBase, a.DyingBase, a.TailBase)
	}
	if want := 16 + 8*(2+3+1+7+3); a.Total != want {
		t.Errorf("Total = %d, want %d", a.Total, want)
	}
}
