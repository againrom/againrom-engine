package sim

import (
	"bytes"
	"testing"
)

func formationPlayerCommand(player uint32, value int32) Command {
	return Command{
		Kind:   KindPlayerParameter,
		Player: player,
		X:      int32(PlayerParameterFormation),
		Y:      value,
	}
}

func retreatPlayerCommand(player uint32, value int32) Command {
	return Command{
		Kind:   KindPlayerParameter,
		Player: player,
		X:      int32(PlayerParameterRetreat),
		Y:      value,
	}
}

func TestRetreatPlayerCommandUsesSelectorThreeAndScopesBothWritersToThePlayer(t *testing.T) {
	if PlayerParameterRetreat != 3 {
		t.Fatalf("retreat selector = %d, want decoded sub-code 3", PlayerParameterRetreat)
	}
	for _, tc := range []struct {
		name       string
		mode       int32
		wantFirst  int32
		wantSecond int32
	}{
		{"off", RetreatModeOff, 0, 0},
		{"low", RetreatModeLow, 7, 10},
		{"high", RetreatModeHigh, 23, 30},
	} {
		t.Run(tc.name, func(t *testing.T) {
			local := engFighter(10, SelfSlot, 5, 5)
			local.HP, local.MaxHP, local.Withdraw, local.Wimpy = 79, 79, 61, 17
			otherLocal := engFighter(11, SelfSlot, 6, 5)
			otherLocal.HP, otherLocal.MaxHP, otherLocal.Withdraw, otherLocal.Wimpy = 101, 101, 62, 18
			foreign := engFighter(12, SelfSlot+1, 7, 5)
			foreign.HP, foreign.MaxHP, foreign.Withdraw, foreign.Wimpy = 333, 333, 63, 19
			w := engWorld(t, engRel(t), local, otherLocal, foreign)

			Step(w, []Command{retreatPlayerCommand(SelfSlot, tc.mode)})
			gotLocal, gotOther, gotForeign := laEnt(t, w, 10), laEnt(t, w, 11), laEnt(t, w, 12)
			if got := [2]int32{gotLocal.Withdraw, gotLocal.Wimpy}; got != [2]int32{tc.wantFirst, 0} {
				t.Errorf("first local thresholds = %v, want [%d 0]", got, tc.wantFirst)
			}
			if got := [2]int32{gotOther.Withdraw, gotOther.Wimpy}; got != [2]int32{tc.wantSecond, 0} {
				t.Errorf("second local thresholds = %v, want [%d 0]", got, tc.wantSecond)
			}
			if got := [2]int32{gotForeign.Withdraw, gotForeign.Wimpy}; got != [2]int32{63, 19} {
				t.Errorf("foreign thresholds = %v, want authored [63 19] untouched", got)
			}
		})
	}
}

func TestRetreatPlayerCommandRefusesUnknownModeAndPlayerLikeAQuietTick(t *testing.T) {
	for _, tc := range []struct {
		name string
		cmd  Command
	}{
		{"negative mode", retreatPlayerCommand(SelfSlot, -1)},
		{"mode past High", retreatPlayerCommand(SelfSlot, RetreatModeHigh+1)},
		{"player outside roster", retreatPlayerCommand(relationSlots, RetreatModeLow)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := engFighter(10, SelfSlot, 5, 5)
			e.MaxHP, e.Withdraw, e.Wimpy = 79, 61, 17
			with := engWorld(t, engRel(t), e)
			quiet := engWorld(t, engRel(t), e)
			Step(with, []Command{tc.cmd})
			Step(quiet, nil)
			if got, want := laForm(t, with), laForm(t, quiet); !bytes.Equal(got, want) {
				t.Error("refused retreat command differs from a quiet tick")
			}
			if with.Hash() != quiet.Hash() {
				t.Errorf("refused retreat command hash %#x != quiet hash %#x", with.Hash(), quiet.Hash())
			}
		})
	}
}

func TestRetreatPlayerCommandUsesExistingThresholdBytesAndRoundTrips(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode int32
		want int32
	}{
		{"off", RetreatModeOff, 0},
		{"low", RetreatModeLow, 10},
		{"high", RetreatModeHigh, 30},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := engFighter(10, SelfSlot, 5, 5)
			e.HP, e.MaxHP, e.Withdraw, e.Wimpy = 101, 101, 77, 66
			with := engWorld(t, engRel(t), e)
			quiet := engWorld(t, engRel(t), e)
			Step(with, []Command{retreatPlayerCommand(SelfSlot, tc.mode)})
			Step(quiet, nil)

			if got := laEnt(t, with, 10); got.Withdraw != tc.want || got.Wimpy != 0 {
				t.Fatalf("command thresholds = %d/%d, want %d/0", got.Withdraw, got.Wimpy, tc.want)
			}
			form, quietForm := laForm(t, with), laForm(t, quiet)
			if len(form) != len(quietForm) || form[0] != formatVersion || quietForm[0] != formatVersion {
				t.Fatalf("command changed form shape/version: len %d/%d version %d/%d want %d",
					len(form), len(quietForm), form[0], quietForm[0], formatVersion)
			}
			if bytes.Equal(form, quietForm) || with.Hash() == quiet.Hash() {
				t.Fatal("retreat command moved neither existing byte form nor digest")
			}

			var back World
			if err := back.UnmarshalBinary(form); err != nil {
				t.Fatalf("UnmarshalBinary: %v", err)
			}
			if got := laEnt(t, &back, 10); got.Withdraw != tc.want || got.Wimpy != 0 {
				t.Errorf("loaded thresholds = %d/%d, want %d/0", got.Withdraw, got.Wimpy, tc.want)
			}
			if back.Hash() != with.Hash() || !bytes.Equal(laForm(t, &back), form) {
				t.Error("loaded world did not preserve the exact command result")
			}
		})
	}
}

// TestFormationPlayerCommandUsesTheDecodedOpcodeSelectorAndWholeRemap is the
// exact opcode 0x46 / selector 2 table in `AI-FORM-037`. The two default cases
// make the switch total rather than accepting only the three values this UI
// emits.
func TestFormationPlayerCommandUsesTheDecodedOpcodeSelectorAndWholeRemap(t *testing.T) {
	if KindPlayerParameter != 0x46 {
		t.Fatalf("player-parameter command kind = %#x, want decoded opcode 0x46", KindPlayerParameter)
	}
	if PlayerParameterFormation != 2 {
		t.Fatalf("formation selector = %d, want decoded sub-code 2", PlayerParameterFormation)
	}

	for _, tc := range []struct {
		name  string
		value int32
		want  uint8
	}{
		{"off", 0, 0},
		{"auto", 1, 2},
		{"on", 2, 1},
		{"negative takes default", -1, 2},
		{"positive outside table takes default", 3, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := engWorld(t, engRel(t), engFighter(10, 3, 5, 5))
			// No entity has id 2. The command must still reach PLAYER 2.
			Step(w, []Command{formationPlayerCommand(2, tc.value)})
			if got := w.FormationMode(2); got != tc.want {
				t.Errorf("authored value %d stored mode %d, want %d", tc.value, got, tc.want)
			}
			if got := w.FormationMode(3); got != formationDefault {
				t.Errorf("player 3 moved to mode %d, want unchanged default %d", got, formationDefault)
			}
		})
	}
}

// TestFormationPlayerCommandRefusesUnknownScopeLikeAQuietTick compares each
// refusal against an otherwise identical tick. The world tick itself advances,
// so comparing against the pre-step bytes would mistake time for an effect.
func TestFormationPlayerCommandRefusesUnknownScopeLikeAQuietTick(t *testing.T) {
	for _, tc := range []struct {
		name string
		cmd  Command
	}{
		{"unknown selector", Command{Kind: KindPlayerParameter, Player: 2, X: 99, Y: 1}},
		{"player outside roster", formationPlayerCommand(relationSlots, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			with := engWorld(t, engRel(t), engFighter(10, 3, 5, 5))
			quiet := engWorld(t, engRel(t), engFighter(10, 3, 5, 5))
			Step(with, []Command{tc.cmd})
			Step(quiet, nil)
			if got, want := laForm(t, with), laForm(t, quiet); !bytes.Equal(got, want) {
				t.Error("refused player command differs from a quiet tick")
			}
			if with.Hash() != quiet.Hash() {
				t.Errorf("refused command hash %#x != quiet hash %#x", with.Hash(), quiet.Hash())
			}
		})
	}
}

// TestFormationPlayerCommandUsesTheExistingCanonicalByte proves the command
// changes the existing fixed-width formation section. There is no byte-form
// version or length change, and the changed byte survives a read/write round
// trip with the same digest.
func TestFormationPlayerCommandUsesTheExistingCanonicalByte(t *testing.T) {
	w := engWorld(t, engRel(t), engFighter(10, 3, 5, 5))
	before := laForm(t, w)
	beforeHash := w.Hash()

	Step(w, []Command{formationPlayerCommand(2, 2)}) // authored On -> stored 1
	after := laForm(t, w)
	if len(after) != len(before) {
		t.Fatalf("byte-form length changed from %d to %d", len(before), len(after))
	}
	if before[0] != formatVersion || after[0] != formatVersion {
		t.Fatalf("byte-form version moved: before %d after %d want %d", before[0], after[0], formatVersion)
	}
	if bytes.Equal(after, before) || w.Hash() == beforeHash {
		t.Fatal("the formation command moved neither byte form nor digest")
	}

	var back World
	if err := back.UnmarshalBinary(after); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got := back.FormationMode(2); got != 1 {
		t.Errorf("loaded formation mode = %d, want 1", got)
	}
	if got := laForm(t, &back); !bytes.Equal(got, after) {
		t.Error("loaded world did not write the exact command result back")
	}
	if back.Hash() != w.Hash() {
		t.Errorf("loaded hash %#x != command result %#x", back.Hash(), w.Hash())
	}
}

// TestFormationPlayerCommandChangesTheGroupMoveThatFollowsIt proves the command
// is consumed by actual group movement, not merely stored. The members are too
// far apart for Auto (stored 2), while On (stored 1) forces the formation arm.
func TestFormationPlayerCommandChangesTheGroupMoveThatFollowsIt(t *testing.T) {
	build := func(t *testing.T, authored int32) *World {
		t.Helper()
		w := engWorld(t, engRel(t),
			laFighter(10, 2, 7, 5, 5), laFighter(11, 2, 7, 30, 30))
		if inFormation(w.entities, []int{0, 1}, 17, 17) {
			t.Fatal("fixture spread unexpectedly passes")
		}
		Step(w, []Command{
			formationPlayerCommand(2, authored),
			{Kind: KindGroupMoveTo, Entity: 10, X: 20, Y: 20, Group: 1},
			{Kind: KindGroupMoveTo, Entity: 11, X: 20, Y: 20, Group: 1},
		})
		return w
	}

	auto := build(t, 1) // stored mode 2: spread-gated, refused here
	on := build(t, 2)   // stored mode 1: unconditional formation
	autoMember, onMember := laEnt(t, auto, 10), laEnt(t, on, 10)
	if autoMember.TargetX != 20 || autoMember.TargetY != 20 || autoMember.GroupSpeed != 0 {
		t.Fatalf("Auto member = target (%d,%d), term %d; want ordered cell and no term",
			autoMember.TargetX, autoMember.TargetY, autoMember.GroupSpeed)
	}
	if onMember.TargetX == 20 && onMember.TargetY == 20 {
		t.Fatalf("On member still walks to the undistributed ordered cell: %+v", onMember)
	}
	if onMember.GroupSpeed == 0 {
		t.Fatal("On member carries no group-rate term; movement ignored the changed mode")
	}
}
