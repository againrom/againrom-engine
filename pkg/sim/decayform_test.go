package sim

// The decay ladder's half of the byte form (0089 AC-16): that the three fields
// reach the bytes and the digest at all, and the four shapes the decoder refuses.
//
// Where they SIT is the offset table's, in binary_test.go, and that this story
// wrote no byte outside them is TestThePinIsThePreStoryPinPlusTheDecayBlock's.

import (
	"bytes"
	"testing"
)

// dfWorld is two units on a 4x4 map, the second of them a body: the form is then
// 34 + 3*16 header and planes, so record 0 opens at 82 and record 1 at 357.
const (
	dfRecord0 = 34 + 3*16
	dfRecord1 = dfRecord0 + 492
	// The three offsets inside a record, written out from the contract rather
	// than read off the encoder.
	dfStageAt = 92
	dfDwellAt = 93
	dfDyingAt = 95
)

func dfWorld(t *testing.T) *World {
	t.Helper()
	return mustWorld(t, 3, Bounds{Width: 4, Height: 4}, []Entity{
		{ID: 1, X: 1, Y: 1, HP: 20, MaxHP: 20, DyingTime: 9},
		{ID: 2, X: 2, Y: 2, HP: -4, MaxHP: 20, DyingTime: 6},
	})
}

// TestTwoWorldsDifferingOnlyInADecayFieldHashDifferently is the assertion that
// does not care what any digest IS, and it is the one a version bump most needs:
// every pinned digest in this package moved with this story, and a digest that
// changed for the right reason looks exactly like one that changed for the wrong
// one. A field dropped between the constructor and the encoder would leave every
// behavioural test in the tree green.
func TestTwoWorldsDifferingOnlyInADecayFieldHashDifferently(t *testing.T) {
	t.Parallel()

	build := func(hp int32, dying int32, stage DecayStage) *World {
		return mustWorld(t, 7, Bounds{Width: 4, Height: 4}, []Entity{
			{ID: 1, X: 1, Y: 1, HP: hp, MaxHP: 20, DyingTime: dying, Decay: stage},
			{ID: 2, X: 2, Y: 2, HP: 5, MaxHP: 5},
		})
	}
	for _, tc := range []struct {
		name string
		a, b *World
	}{
		// The DYING TIME on a living unit, which nothing in a tick reads until
		// that unit falls — so this is the field a build could lose silently.
		{"the dying time", build(20, 3, DecayNone), build(20, 4, DecayNone)},
		// The STAGE, on two bodies alike in health.
		{"the stage", build(-30, 0, DecayBones), build(-30, 0, 3)},
		// And the DWELL, which differs here because the two dying times do.
		{"the dwell", build(-1, 3, DecayFallen), build(-1, 4, DecayFallen)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.a.Hash() == tc.b.Hash() {
				t.Errorf("two worlds differing only in %s hash alike (%#016x) — "+
					"the field does not reach the byte form", tc.name, tc.a.Hash())
			}
			if bytes.Equal(mustMarshal(t, tc.a), mustMarshal(t, tc.b)) {
				t.Errorf("two worlds differing only in %s marshal to the same bytes", tc.name)
			}
		})
	}
}

// TestADecodedWorldKeepsItsDecayState is the round trip, over every stage a
// stored world may carry and over a dwell that only the first stage owes.
func TestADecodedWorldKeepsItsDecayState(t *testing.T) {
	t.Parallel()

	for stage := DecayNone; stage <= decayLast; stage++ {
		hp := int32(20)
		if stage != DecayNone {
			hp = -50
		}
		w := mustWorld(t, 11, Bounds{Width: 4, Height: 4}, []Entity{
			{ID: 1, X: 1, Y: 1, HP: hp, MaxHP: 20, DyingTime: 300, Decay: stage},
		})
		form := mustMarshal(t, w)
		var back World
		if err := back.UnmarshalBinary(form); err != nil {
			t.Fatalf("stage %d: UnmarshalBinary: %v", stage, err)
		}
		if got, want := back.Entities(), w.Entities(); len(got) != 1 || got[0] != want[0] {
			t.Errorf("stage %d: decoded %+v, want %+v", stage, got, want)
		}
		if back.Hash() != w.Hash() {
			t.Errorf("stage %d: the decoded world hashes %#016x, the original %#016x",
				stage, back.Hash(), w.Hash())
		}
	}
}

// TestUnmarshalRefusesTheDecayShapesNoTickCanLeave is the decoder's half of the
// pairing rule the constructor normalises, plus the stage byte with no world to
// name. Each case differs from a valid form in exactly one field.
func TestUnmarshalRefusesTheDecayShapesNoTickCanLeave(t *testing.T) {
	t.Parallel()

	valid := mustMarshal(t, dfWorld(t))
	for _, tc := range []struct {
		name string
		data []byte
	}{
		// The byte with no world to name: the stage that means removal, and the
		// two above it.
		{"a stage of 5, which is removal and not a state", withByte(valid, dfRecord1+dfStageAt, 5)},
		{"a stage at the top of its range", withByte(valid, dfRecord1+dfStageAt, 0xff)},
		// The pairing rule, in both directions.
		{"a stage on a living unit", withByte(valid, dfRecord0+dfStageAt, 1)},
		{"no stage on a body", withByte(valid, dfRecord1+dfStageAt, 0)},
		// And a dwell at a stage that owes none.
		{"a dwell at the second stage",
			withU16(withByte(valid, dfRecord1+dfStageAt, 2), dfRecord1+dfDwellAt, 3)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var w World
			if err := w.UnmarshalBinary(tc.data); err == nil {
				t.Error("accepted")
			}
		})
	}
	// And the unspoiled form is accepted, so the cases above measure the decay
	// checks and not something else about this fixture.
	if err := (&World{}).UnmarshalBinary(valid); err != nil {
		t.Errorf("the unspoiled form was refused: %v", err)
	}
	// The dying time is carried WHOLE and refused nowhere, on the rule the health
	// pair and the seven combat numbers take: every int32 is a state the
	// constructor accepts.
	for _, v := range []uint32{0, 1, 0x7fffffff, 0xffffffff} {
		if err := (&World{}).UnmarshalBinary(withU32(valid, dfRecord0+dfDyingAt, v)); err != nil {
			t.Errorf("a dying time of %#08x was refused: %v", v, err)
		}
	}
}
