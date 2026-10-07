package sim

import (
	"reflect"
	"testing"
)

// Instant opcode 2, the broadcast message.
//
// The opcode is a SUPPORTED arm that writes nothing. Both halves need their own
// witness, because each is invisible in the other's test: an opcode dropped
// from the gap list with no arm passes the first test and an opcode that
// quietly wrote a register passes neither on its own.

// messageInstant is one instant-2 node raising event number e.
func messageInstant(e int32) ScriptInstant {
	return ScriptInstant{Op: ScriptInstantMessage, Args: [scriptParams]int32{e}}
}

func TestMessageInstantIsSupportedAndReportsNoGap(t *testing.T) {
	t.Parallel()

	if !scriptInstantSupported(messageInstant(11)) {
		t.Error("scriptInstantSupported(instant 2) = false, want true")
	}
	s := mustScript(t, []ScriptCheck{constCheck(0, 1)},
		[]ScriptInstant{messageInstant(11)},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 0, ScriptCmpEQ)}, Instants: acts(0), Latch: 0}})
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Fatalf("Unsupported() = %+v, want none", gaps)
	}
}

// messageWorld is a world whose one trigger fires and runs an instant-2 node
// raising e, stepped far enough for the script pass to have run several times.
func messageWorld(t *testing.T, e int32) *World {
	t.Helper()
	s := mustScript(t, []ScriptCheck{constCheck(0, 1)},
		[]ScriptInstant{messageInstant(e)},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 0, ScriptCmpEQ)}, Instants: acts(0), Latch: 4}})
	w := scriptWorld(t, s, []Entity{{ID: 1, X: 10, Y: 10, HP: 30, MaxHP: 30, Owner: 1}})
	scriptTicks(w, 64, nil)
	return w
}

func TestMessageInstantRunsAndWritesNothing(t *testing.T) {
	t.Parallel()

	w := messageWorld(t, 25)
	if !w.ScriptLatched(4) {
		t.Fatal("the trigger holding the instant-2 node never fired")
	}
	for i := int32(0); i < scriptRegisters; i++ {
		want := int32(0)
		if i == 0 {
			want = 1 // the build-time constant this trigger reads
		}
		if i == 93 {
			want = 4 // the script clock advances before each of the four passes
		}
		if got := w.registerAt(i); got != want {
			t.Errorf("register %d = %d, want %d", i, got, want)
		}
	}
	if w.Outcome() != OutcomeUndecided {
		t.Errorf("outcome = %v, want undecided", w.Outcome())
	}
	ents := w.Entities()
	if len(ents) != 1 {
		t.Fatalf("%d entities, want 1", len(ents))
	}
	if e := ents[0]; e.X != 10 || e.Y != 10 || e.HP != 30 || e.Owner != 1 {
		t.Errorf("entity = %+v, want it exactly as it was placed", e)
	}
	for i := int32(0); i < scriptLatches; i++ {
		if i == 4 {
			continue
		}
		if w.ScriptLatched(i) {
			t.Errorf("latch %d is set; only the firing trigger's own may be", i)
		}
	}
}

func TestMessageInstantIgnoresItsEventNumber(t *testing.T) {
	t.Parallel()

	low, high := messageWorld(t, 0), messageWorld(t, 254)
	for i := int32(0); i < scriptRegisters; i++ {
		if a, b := low.registerAt(i), high.registerAt(i); a != b {
			t.Errorf("register %d = %d raising 0, %d raising 254", i, a, b)
		}
	}
	if a, b := low.Outcome(), high.Outcome(); a != b {
		t.Errorf("outcome = %v raising 0, %v raising 254", a, b)
	}
	if a, b := low.Entities(), high.Entities(); !reflect.DeepEqual(a, b) {
		t.Errorf("entities = %+v raising 0, %+v raising 254", a, b)
	}
	if a, b := low.Tick(), high.Tick(); a != b {
		t.Errorf("tick = %d raising 0, %d raising 254", a, b)
	}
}
