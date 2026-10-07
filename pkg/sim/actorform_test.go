package sim

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
)

// aftBounds names no in-bounds cell — the height is negative, offsetWorld's
// own trick (binary_test.go) — so the grid, the cost and the height planes
// are all empty and an encoded form's entity records begin immediately
// after the 34-byte header, at an offset this file need not compute from a
// grid.
var aftBounds = Bounds{Width: 5, Height: -1}

// aftEnts is two living units with a health system, far enough apart on the
// coordinate line that a ring built from one is never mistaken for the
// other's.
func aftEnts() []Entity {
	return []Entity{
		{ID: 1, X: 2, Y: 3, HP: 10, MaxHP: 10},
		{ID: 2, X: 40, Y: 41, HP: 20, MaxHP: 20},
	}
}

// aftDeadEnts is aftEnts with the first unit felled, for AC-5's fourth case:
// a patrol state needs a unit that is NOT ALIVE, and the constructor's own
// normalisation makes every entity it builds guard — so getting one to
// patrol while dead takes decoding, exactly as every other case here does,
// and this is the base that decode starts from.
func aftDeadEnts() []Entity {
	ents := aftEnts()
	ents[0].HP = -1
	return ents
}

// aftRecordOff is where entity i's (0-indexed, ascending id) record starts
// in a form over aftBounds: the header alone, then i whole records — no
// plane bytes come first, on aftBounds' own trick above.
func aftRecordOff(i int) int { return headerLen + entityLen*i }

// The record's new tail, offsets 100 through 117 (binary.go's own table),
// named once here rather than as six repeated literals below.
const (
	aftState = 100
	aftHeadX = 101
	aftHeadY = 105
	aftTailX = 109
	aftTailY = 113
	aftLeg   = 117
)

func aftBase(t *testing.T) []byte {
	t.Helper()
	b, err := mustWorld(t, 0xa17, aftBounds, aftEnts()).MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return b
}

// aftDeadBase is aftBase over aftDeadEnts: the same two records, except
// entity 0 is a corpse.
func aftDeadBase(t *testing.T) []byte {
	t.Helper()
	b, err := mustWorld(t, 0xa17, aftBounds, aftDeadEnts()).MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return b
}

func aftPatrol(t *testing.T) []byte {
	t.Helper()
	o := aftRecordOff(0)
	b := aftBase(t)
	b = withByte(b, o+aftState, actorStatePatrol)
	b = withU32(b, o+aftHeadX, uint32(int32(2)))
	b = withU32(b, o+aftHeadY, uint32(int32(3)))
	b = withU32(b, o+aftTailX, uint32(int32(4)))
	b = withU32(b, o+aftTailY, uint32(int32(1)))
	b = withByte(b, o+aftLeg, patrolLegTail)
	return b
}

// ---------------------------------------------------------------- AC-3

// TestAPatrollingActorRoundTrips is AC-3: a world holding a patrolling actor
// decodes and re-encodes to the same bytes, and hashes to the same digest
// the bytes it was decoded from would hash to.
func TestAPatrollingActorRoundTrips(t *testing.T) {
	form := aftPatrol(t)

	var w World
	if err := w.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	// The state, the ring and the leg actually arrived. A round trip that
	// passed on a decoder dropping the tail and an encoder writing zeros
	// for it would still balance, so the contract is checked field by field
	// first.
	got := w.Entities()[0]
	if got.ActorState != actorStatePatrol {
		t.Fatalf("entity %d decoded to actor state %d, want patrol (%d)", got.ID, got.ActorState, actorStatePatrol)
	}
	if got.PatrolHeadX != 2 || got.PatrolHeadY != 3 || got.PatrolTailX != 4 || got.PatrolTailY != 1 {
		t.Fatalf("entity %d decoded to ring (%d,%d)-(%d,%d), want (2,3)-(4,1)",
			got.ID, got.PatrolHeadX, got.PatrolHeadY, got.PatrolTailX, got.PatrolTailY)
	}
	if got.PatrolLeg != patrolLegTail {
		t.Fatalf("entity %d decoded to leg %d, want tail (%d)", got.ID, got.PatrolLeg, patrolLegTail)
	}

	again, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if !bytes.Equal(again, form) {
		t.Errorf("re-marshalling gave\n % x\nwant\n % x", again, form)
	}
	if got, want := w.Hash(), fnv1a(form); got != want {
		t.Errorf("the decoded world hashes %#016x, the form's own FNV-1a is %#016x", got, want)
	}
}

// ---------------------------------------------------------------- AC-4

// TestAVersion20FormIsRefused is AC-4. aftBase is a byte-for-byte valid
// current-version form (version 24 since 0106); changing only its version
// byte is what isolates the version check from everything else this task
// changed, and the message must name both the version that was refused and
// the one this build reads.
func TestAVersion20FormIsRefused(t *testing.T) {
	form := withByte(aftBase(t), 0, 20)

	var w World
	err := w.UnmarshalBinary(form)
	if err == nil {
		t.Fatalf("accepted a version-20 form")
	}
	if !strings.Contains(err.Error(), "20") || !strings.Contains(err.Error(), strconv.Itoa(int(formatVersion))) {
		t.Errorf("refusal %q does not name both versions", err.Error())
	}
}

// ---------------------------------------------------------------- AC-5

// TestFR13sFourRefusals is AC-5: each of patrolFault's four shapes, on a
// form built to carry exactly it and no other. Each case starts from a base
// this file already knows is otherwise valid and changes only the bytes its
// name says, so a case that fired on the wrong ground would be visible in
// the message it produced rather than only in a pass/fail.
func TestFR13sFourRefusals(t *testing.T) {
	o := aftRecordOff(0)

	cases := []struct {
		name string
		data []byte
		want string
	}{
		{
			name: "a state outside guard and patrol",
			data: withByte(aftBase(t), o+aftState, 5),
			want: "actor state is 5",
		},
		{
			// aftPatrol is a valid patroller; patching only the leg to a
			// value outside {0, 1} leaves the state at patrol, which is
			// what keeps the third shape from firing alongside it — that
			// one is gated on the state NOT being patrol.
			name: "a leg outside the ring's two cells",
			data: withByte(aftPatrol(t), o+aftLeg, 7),
			want: "patrol leg is 7",
		},
		{
			// aftBase's entity 0 stays at guard; only the leg (to a value
			// that is itself legal) and one ring coordinate are patched, so
			// this is a ring on a non-patrol entity and nothing else — not
			// an undefined state and not an undefined leg.
			name: "a ring on an entity not in the patrol state",
			data: withU32(withByte(aftBase(t), o+aftLeg, patrolLegTail), o+aftHeadX, uint32(int32(5))),
			want: "not patrol",
		},
		{
			// aftDeadBase's entity 0 is a corpse with an all-zero ring;
			// patching only the state to patrol leaves the leg and the
			// ring at their legal zero values, so this is a patrol state on
			// a unit that is not alive and nothing else.
			name: "a patrol state on an entity that is not alive",
			data: withByte(aftDeadBase(t), o+aftState, actorStatePatrol),
			want: "which is not alive",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var w World
			err := w.UnmarshalBinary(tc.data)
			if err == nil {
				t.Fatalf("accepted % x", tc.data)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("refusal %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}
