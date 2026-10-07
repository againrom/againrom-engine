package game_test

// Tests for deriving announcements from a world's latch array.
//
// These drive a REAL world through pkg/sim rather than stubbing a latch array,
// because the whole claim is about what the script pass writes and when. A stub
// would pin our reading of that pass instead of the pass itself.
//
// One convention throughout: the mission variable lives at register 10, above
// every check these scripts compile, because a variable and a check result share
// one array and a variable index below the check count is a cell a check
// overwrites every pass.

import (
	"reflect"
	"testing"

	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

const annVar int32 = 10

// annChecks is the condition vocabulary these scripts share: the mission
// variable in register 0, the constant 0 in register 1, the constant 1 in
// register 2.
func annChecks() []sim.ScriptCheck {
	return []sim.ScriptCheck{
		{Op: sim.ScriptCheckVariable, Register: 0, Args: [10]int32{annVar}},
		{Op: sim.ScriptCheckConstant, Register: 1, Args: [10]int32{0}},
		{Op: sim.ScriptCheckConstant, Register: 2, Args: [10]int32{1}},
	}
}

// annTrigger is one trigger comparing register 0 against register right, with
// its latch at the given map position and up to four instant subscripts.
func annTrigger(latch int32, right int32, once bool, acts ...int32) sim.ScriptTrigger {
	var t sim.ScriptTrigger
	t.Pairs[0] = sim.ScriptPair{Left: 0, Right: right, Cmp: sim.ScriptCmpEQ, Used: true}
	t.Once, t.Latch = once, latch
	for i := range t.Instants {
		t.Instants[i] = sim.ScriptNone
	}
	for i, a := range acts {
		t.Instants[i] = a
	}
	return t
}

func annWorld(t *testing.T, instants []sim.ScriptInstant, triggers []sim.ScriptTrigger) *sim.World {
	t.Helper()
	s, err := sim.NewScript(annChecks(), instants, triggers)
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: 4, Height: 4},
		sim.ModeCanonical, make([]byte, 16), nil, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	return w
}

func stepPasses(w *sim.World, a *game.Announcer, cycles int) []int32 {
	var got []int32
	for i := 0; i < cycles*16; i++ {
		sim.Step(w, nil)
		got = append(got, a.Sample(w)...)
	}
	return got
}

func TestAnnouncerRaisesAOneShotExactlyOnce(t *testing.T) {
	// The variable starts 0, so the trigger holds on every pass — but being a
	// one-shot it is skipped whole after the first, and its latch stays set.
	w := annWorld(t,
		[]sim.ScriptInstant{{Op: 2, Args: [10]int32{7}}},
		[]sim.ScriptTrigger{annTrigger(3, 1, true, 0)})
	a := game.NewAnnouncer(w, []mapload.ScriptRaise{{Latch: 3, Event: 7}})

	if got, want := stepPasses(w, a, 5), []int32{7}; !reflect.DeepEqual(got, want) {
		t.Fatalf("raises over 5 passes = %v, want %v — a one-shot announces once", got, want)
	}
}

func TestAnnouncerRaisesARepeatingTriggerOnEveryPassItHolds(t *testing.T) {
	w := annWorld(t,
		[]sim.ScriptInstant{{Op: 2, Args: [10]int32{4}}},
		[]sim.ScriptTrigger{annTrigger(3, 1, false, 0)})
	a := game.NewAnnouncer(w, []mapload.ScriptRaise{{Latch: 3, Event: 4}})

	if got, want := stepPasses(w, a, 6), []int32{4, 4, 4, 4, 4, 4}; !reflect.DeepEqual(got, want) {
		t.Fatalf("six adjacent firings = %v, want %v — the original broadcasts once per pass "+
			"and the panel's discard rule, not the announcer, hides the repeats", got, want)
	}
}

// A repeating trigger loaded already latched announces on its first pass.
func TestAnnouncerRaisesAHeldRepeatingTriggerAfterALoad(t *testing.T) {
	w := annWorld(t,
		[]sim.ScriptInstant{{Op: 2, Args: [10]int32{4}}},
		[]sim.ScriptTrigger{annTrigger(3, 1, false, 0)})
	for i := 0; i < 16; i++ {
		sim.Step(w, nil)
	}
	if !w.ScriptLatched(3) {
		t.Fatal("the trigger did not latch")
	}
	a := game.NewAnnouncer(w, []mapload.ScriptRaise{{Latch: 3, Event: 4}})
	if got, want := stepPasses(w, a, 2), []int32{4, 4}; !reflect.DeepEqual(got, want) {
		t.Fatalf("raises after a load = %v, want %v", got, want)
	}
}

// The case that must be EXACT: a repeating trigger whose conditions stop holding
// between firings. The script authors its own flip-flop — one trigger sets the
// variable when it is 0, another clears it when it is 1, and because every check
// is evaluated before any trigger runs, the two alternate pass by pass.
func TestAnnouncerSeesEveryNonAdjacentFiring(t *testing.T) {
	w := annWorld(t,
		[]sim.ScriptInstant{
			{Op: 2, Args: [10]int32{9}},                                    // 0: the announcement
			{Op: sim.ScriptInstantSetVariable, Args: [10]int32{annVar, 1}}, // 1: raise the flag
			{Op: sim.ScriptInstantSetVariable, Args: [10]int32{annVar, 0}}, // 2: lower it
		},
		[]sim.ScriptTrigger{
			annTrigger(3, 1, false, 0), // variable == 0: announce
			annTrigger(5, 1, false, 1), // variable == 0: set it to 1
			annTrigger(7, 2, false, 2), // variable == 1: set it back to 0
		})
	a := game.NewAnnouncer(w, []mapload.ScriptRaise{{Latch: 3, Event: 9}})

	// Six passes, alternating, so the announcing trigger fires on three of them.
	got := stepPasses(w, a, 6)
	if want := []int32{9, 9, 9}; !reflect.DeepEqual(got, want) {
		t.Fatalf("non-adjacent firings = %v, want %v — each firing must be seen", got, want)
	}
}

func TestAnnouncerRaisesEveryMessageSlotInOrder(t *testing.T) {
	w := annWorld(t,
		[]sim.ScriptInstant{
			{Op: 2, Args: [10]int32{11}},
			{Op: sim.ScriptInstantIncVariable, Args: [10]int32{annVar}},
			{Op: 2, Args: [10]int32{12}},
		},
		[]sim.ScriptTrigger{annTrigger(3, 1, true, 0, 1, 2)})
	// The compile reports the two message slots in slot order and skips the
	// increment between them.
	a := game.NewAnnouncer(w, []mapload.ScriptRaise{{Latch: 3, Event: 11}, {Latch: 3, Event: 12}})

	if got, want := stepPasses(w, a, 3), []int32{11, 12}; !reflect.DeepEqual(got, want) {
		t.Fatalf("raises = %v, want %v in slot order", got, want)
	}
}

func TestAnnouncerRaisesNothingForATriggerThatCannotFire(t *testing.T) {
	// The variable is 0 and the trigger tests it against the constant 1, so the
	// pair never compares equal and the latch is never set.
	w := annWorld(t,
		[]sim.ScriptInstant{{Op: 2, Args: [10]int32{3}}},
		[]sim.ScriptTrigger{annTrigger(3, 2, false, 0)})
	a := game.NewAnnouncer(w, []mapload.ScriptRaise{{Latch: 3, Event: 3}})

	if got := stepPasses(w, a, 8); got != nil {
		t.Fatalf("a trigger whose conditions never hold raised %v, want nothing", got)
	}
}

func TestAnnouncerIsSilentWithNoRaiseList(t *testing.T) {
	w := annWorld(t,
		[]sim.ScriptInstant{{Op: sim.ScriptInstantWin}},
		[]sim.ScriptTrigger{annTrigger(3, 1, true, 0)})

	if got := stepPasses(w, game.NewAnnouncer(w, nil), 4); got != nil {
		t.Fatalf("an announcer with no raise list produced %v", got)
	}
	// A nil announcer is the answer for a map with no script at all.
	var none *game.Announcer
	if got := none.Sample(w); got != nil {
		t.Fatalf("a nil announcer produced %v", got)
	}
}

// A world arriving with a one-shot already latched — one resumed from its bytes
// part-way through a mission — must not re-announce what fired before the
// announcer existed.
func TestAnnouncerDoesNotReannounceAnAlreadyLatchedTrigger(t *testing.T) {
	w := annWorld(t,
		[]sim.ScriptInstant{{Op: 2, Args: [10]int32{6}}},
		[]sim.ScriptTrigger{annTrigger(3, 1, true, 0)})
	raises := []mapload.ScriptRaise{{Latch: 3, Event: 6}}

	if got := stepPasses(w, game.NewAnnouncer(w, raises), 2); !reflect.DeepEqual(got, []int32{6}) {
		t.Fatalf("setup: the first announcer got %v, want [6]", got)
	}
	if got := stepPasses(w, game.NewAnnouncer(w, raises), 3); got != nil {
		t.Fatalf("a fresh announcer over an already-latched world raised %v, want nothing", got)
	}
}

func TestSamplingDoesNotReachTheWorld(t *testing.T) {
	mk := func() *sim.World {
		return annWorld(t,
			[]sim.ScriptInstant{{Op: 2, Args: [10]int32{1}}},
			[]sim.ScriptTrigger{annTrigger(3, 1, false, 0)})
	}
	sampled, plain := mk(), mk()
	a := game.NewAnnouncer(sampled, []mapload.ScriptRaise{{Latch: 3, Event: 1}})

	for i := 0; i < 64; i++ {
		sim.Step(sampled, nil)
		a.Sample(sampled)
		sim.Step(plain, nil)
		if sampled.Tick() != plain.Tick() {
			t.Fatalf("step %d: sampled world at tick %d, plain at %d", i, sampled.Tick(), plain.Tick())
		}
		if sampled.Hash() != plain.Hash() {
			t.Fatalf("step %d: sampling changed the world's digest", i)
		}
	}
}
