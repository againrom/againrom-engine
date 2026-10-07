package sim

import (
	"reflect"
	"testing"
)

// runBounds is the extent every unit in this file walks inside. A unit never
// leaves its bounds, so every cell named below is one this extent contains: a
// schedule that ordered somebody off the map would be measuring the clamp, and
// nothing here is about the clamp.
var runBounds = Bounds{Width: 12, Height: 12}

// runEntities and runSchedule are this file's fixture, returned fresh each
// call so that no two runs share storage. The schedule is built so that a
// command dropped, added or reordered changes where somebody ends up: entity
// 1 is turned around twice mid-walk, tick 5 carries two orders for ONE
// entity pointing opposite ways, and several ticks carry nothing at all.
func runEntities() []Entity {
	return []Entity{{ID: 1, X: 0, Y: 0}, {ID: 4, X: 9, Y: 2}, {ID: 9, X: 0, Y: 6}}
}

func runSchedule() [][]Command {
	return [][]Command{
		{{Entity: 1, X: 8, Y: 8}, {Entity: 4, X: 0, Y: 0}},
		nil,
		{{Entity: 1, X: 1, Y: 1}},
		{}, // scheduled, and empty
		{{Entity: 9, X: 5, Y: 5}, {Entity: 1, X: 7, Y: 0}},
		{{Entity: 4, X: 11, Y: 11}, {Entity: 4, X: 0, Y: 0}},
		nil,
	}
}

// runTicks is longer than the schedule: the tail steps carry no commands, and
// still owe a frame each.
const runTicks = 10

// copyLog is the instrument the "Run keeps no reference" tests compare against.
// It deep-copies, including each frame's command array, and it builds the same
// never-nil empty list Run does so that reflect.DeepEqual can be used on the
// whole log.
func copyLog(l Log) Log {
	out := make(Log, len(l))
	for i, f := range l {
		out[i] = Frame{Tick: f.Tick, Commands: append(make([]Command, 0, len(f.Commands)), f.Commands...)}
	}
	return out
}

// TestCopyLogIsIndependentOfWhatItCopied checks the instrument, not the code
// under test. If copyLog shared a frame's command array, the comparisons below
// would be comparing a mutated log against itself and would pass whatever Run
// did with the schedule it was handed.
func TestCopyLogIsIndependentOfWhatItCopied(t *testing.T) {
	orig := Log{{Tick: 2, Commands: []Command{{Entity: 1, X: 3, Y: 4}}}}
	cp := copyLog(orig)
	if !reflect.DeepEqual(cp, orig) {
		t.Fatalf("copyLog returned %+v, want %+v", cp, orig)
	}

	orig[0].Commands[0].X = 99
	if cp[0].Commands[0].X != 3 {
		t.Error("copyLog shares the original's command array")
	}
	if reflect.DeepEqual(cp, orig) {
		t.Error("copyLog's result followed a write to the original")
	}
}

func TestRunRecordsOneFramePerAdvancedTick(t *testing.T) {
	w := mustWorld(t, 0x5eed, runBounds, runEntities())
	sched := runSchedule()
	log := Run(w, sched, runTicks)

	if len(log) != runTicks {
		t.Fatalf("a %d-tick run recorded %d frame(s)", runTicks, len(log))
	}
	if w.Tick() != uint64(runTicks) {
		t.Errorf("a %d-tick run left the world at tick %d", runTicks, w.Tick())
	}

	for i, f := range log {
		// This world started at tick 0, so the tick a step ran FROM is i. A frame
		// recording the tick the step produced reads i+1 here.
		if f.Tick != uint64(i) {
			t.Errorf("frame %d records tick %d, want %d", i, f.Tick, uint64(i))
		}
		if f.Commands == nil {
			t.Errorf("frame %d has a nil command list where an empty list of its own is meant", i)
			continue
		}
		var want []Command
		if i < len(sched) {
			want = sched[i]
		}
		if len(f.Commands) != len(want) {
			t.Errorf("frame %d carries %d command(s), the schedule named %d", i, len(f.Commands), len(want))
			continue
		}
		for j := range want {
			if f.Commands[j] != want[j] {
				t.Errorf("frame %d command %d is %+v, want %+v", i, j, f.Commands[j], want[j])
			}
		}
	}
}

// TestRunAdvancesExactlyTicksTimes: the run's length is its ticks argument and
// not the schedule's length, in either direction.
func TestRunAdvancesExactlyTicksTimes(t *testing.T) {
	sched := runSchedule()
	for _, ticks := range []int{0, 1, 2, len(sched) - 1, len(sched), runTicks} {
		w := mustWorld(t, 1, runBounds, runEntities())
		log := Run(w, sched, ticks)
		if len(log) != ticks {
			t.Errorf("Run(..., %d) recorded %d frame(s)", ticks, len(log))
		}
		if w.Tick() != uint64(ticks) {
			t.Errorf("Run(..., %d) left the world at tick %d", ticks, w.Tick())
		}
	}
}

func TestRunFramesCarryTheWorldsTickNotTheScheduleIndex(t *testing.T) {
	const first, second = 4, 3
	w := mustWorld(t, 1, runBounds, runEntities())
	Run(w, runSchedule(), first)

	order := Command{Entity: 1, X: 2, Y: 2}
	log := Run(w, [][]Command{{order}}, second)
	if len(log) != second {
		t.Fatalf("the continuation recorded %d frame(s), want %d", len(log), second)
	}
	for i, f := range log {
		if want := uint64(first + i); f.Tick != want {
			t.Errorf("frame %d of the continuation records tick %d, want %d", i, f.Tick, want)
		}
	}
	if len(log[0].Commands) != 1 || log[0].Commands[0] != order {
		t.Errorf("the continuation's first frame carries %+v, want the schedule's one entry %+v "+
			"(empty would mean the schedule was indexed by absolute tick)", log[0].Commands, order)
	}
	for i, f := range log[1:] {
		if len(f.Commands) != 0 {
			t.Errorf("frame %d of the continuation carries %+v, but the schedule ran out at its "+
				"first step", i+1, f.Commands)
		}
	}
}

// TestRunKeepsNoReferenceToTheSchedule: a frame holds a copy, so what the log
// says happened cannot be edited afterwards through the schedule it was read
// from — nor the schedule through the log.
func TestRunKeepsNoReferenceToTheSchedule(t *testing.T) {
	w := mustWorld(t, 1, runBounds, runEntities())
	sched := runSchedule()
	log := Run(w, sched, runTicks)
	before := copyLog(log)

	for i := range sched {
		for j := range sched[i] {
			sched[i][j] = Command{Entity: 99, X: -999, Y: -999}
		}
		sched[i] = []Command{{Entity: 1, X: -999, Y: -999}}
	}
	if !reflect.DeepEqual(log, before) {
		t.Errorf("mutating the schedule after Run reached the log:\n got  %+v\n want %+v", log, before)
	}

	// And the other direction, over a schedule nothing has written to yet.
	fresh := runSchedule()
	w2 := mustWorld(t, 1, runBounds, runEntities())
	log2 := Run(w2, fresh, runTicks)
	poison := Command{Entity: 77, X: 77, Y: 77}
	log2[0].Commands[0] = poison
	if fresh[0][0] == poison {
		t.Error("writing through a frame's command list reached the schedule")
	}
}

// TestTwoRunsOfOneScheduleAgree is AC-4: a run is a function of the initial
// world and the schedule alone.
func TestTwoRunsOfOneScheduleAgree(t *testing.T) {
	const seed = 0x0b5e55ed
	one := mustWorld(t, seed, runBounds, runEntities())
	two := mustWorld(t, seed, runBounds, runEntities())

	logOne := Run(one, runSchedule(), runTicks)
	logTwo := Run(two, runSchedule(), runTicks)

	if one.Hash() != two.Hash() {
		t.Errorf("two runs of one schedule hash %#016x and %#016x", one.Hash(), two.Hash())
	}
	if !reflect.DeepEqual(logOne, logTwo) {
		t.Errorf("two runs of one schedule logged differently:\n %+v\n %+v", logOne, logTwo)
	}
	// Agreement is only worth something if the run went somewhere.
	if untouched := mustWorld(t, seed, runBounds, runEntities()); one.Hash() == untouched.Hash() {
		t.Error("the run left the world exactly where it started — the fixture proves nothing")
	}
}

func TestReplayReproducesTheRunsFinalDigest(t *testing.T) {
	const seed = 0xA11D05
	run := mustWorld(t, seed, runBounds, runEntities())
	log := Run(run, runSchedule(), runTicks)

	replayed := mustWorld(t, seed, runBounds, runEntities())
	if err := Replay(replayed, log); err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if len(log) != int(run.Tick()) {
		t.Errorf("the run advanced %d tick(s) and logged %d frame(s)", run.Tick(), len(log))
	}
	if replayed.Tick() != run.Tick() {
		t.Errorf("the replay is at tick %d, the run at %d", replayed.Tick(), run.Tick())
	}
	if replayed.Hash() != run.Hash() {
		t.Errorf("the replay hashes %#016x, the run %#016x", replayed.Hash(), run.Hash())
	}
}

// TestReplayReproducesOnlyItsOwnRunsInitialWorld is the other half of AC-10's
// "from that run's initial world". Replay constructs no world, so a log applied
// to some other world of the same shape reaches somewhere else — which is what
// makes the test above an assertion and not a tautology.
func TestReplayReproducesOnlyItsOwnRunsInitialWorld(t *testing.T) {
	const seed = 0xA11D05
	run := mustWorld(t, seed, runBounds, runEntities())
	log := Run(run, runSchedule(), runTicks)

	// One unit, two cells off, everything else identical — and the one whose
	// last order it cannot finish inside the run, so the difference is still
	// there at the end instead of being walked off by an arrival.
	other := runEntities()
	other[1].X += 2
	elsewhere := mustWorld(t, seed, runBounds, other)
	if err := Replay(elsewhere, log); err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if elsewhere.Hash() == run.Hash() {
		t.Error("a log replayed into a different initial world reached the run's digest")
	}
}

// TestReplayOfAnEmptyLogChangesNothing: no frames, no steps, no error.
func TestReplayOfAnEmptyLogChangesNothing(t *testing.T) {
	w := mustWorld(t, 3, runBounds, runEntities())
	Run(w, runSchedule(), 2)
	before := snap(w)

	for _, log := range []Log{nil, {}} {
		if err := Replay(w, log); err != nil {
			t.Errorf("Replay of an empty log: %v", err)
		}
		if got := snap(w); !equalState(got, before) {
			t.Errorf("Replay of an empty log changed the world:\n before %+v\n after  %+v", before, got)
		}
	}
}

// TestReplayRefusesALogThatIsNotThisWorlds is AC-11 over a whole log, in both
// directions: a world already past the log's first frame, and a log that starts
// past the world. Either way the first frame is the mismatched one, so NOTHING
// is applied.
func TestReplayRefusesALogThatIsNotThisWorlds(t *testing.T) {
	const seed = 0xFEEDFACE
	recorded := mustWorld(t, seed, runBounds, runEntities())
	log := Run(recorded, runSchedule(), runTicks)

	cases := []struct {
		name    string
		advance int
		log     Log
	}{
		{"the world is ahead of the log's first frame", 3, log},
		{"the log starts past the world", 0, log[2:]},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := mustWorld(t, seed, runBounds, runEntities())
			Run(w, runSchedule(), tc.advance)
			before := snap(w)

			if err := Replay(w, tc.log); err == nil {
				t.Fatal("Replay accepted a log whose first frame is not this world's tick")
			}
			if got := snap(w); !equalState(got, before) {
				t.Errorf("the refused replay changed the world:\n before %+v\n after  %+v", before, got)
			}
		})
	}
}

func TestReplayLeavesTheMismatchedFrameAndEveryLaterOneUnapplied(t *testing.T) {
	const seed = 0xBADF00D
	const k = 3

	recorded := mustWorld(t, seed, runBounds, runEntities())
	log := Run(recorded, runSchedule(), runTicks)

	prefix := mustWorld(t, seed, runBounds, runEntities())
	if err := Replay(prefix, log[:k]); err != nil {
		t.Fatalf("Replay of the good prefix: %v", err)
	}

	for _, tc := range []struct {
		name string
		skew uint64
	}{
		{"the frame is ahead of the world", 1},
		{"the frame is behind the world", ^uint64(0)}, // -1
	} {
		t.Run(tc.name, func(t *testing.T) {
			broken := copyLog(log)
			broken[k].Tick += tc.skew // one tick out; every later frame is untouched

			w := mustWorld(t, seed, runBounds, runEntities())
			if err := Replay(w, broken); err == nil {
				t.Fatal("Replay accepted a frame whose tick is not the world's")
			}
			if w.Tick() != uint64(k) {
				t.Errorf("after the refusal the world is at tick %d, want %d", w.Tick(), k)
			}
			if w.Hash() != prefix.Hash() {
				t.Errorf("after the refusal the world hashes %#016x, the good prefix %#016x",
					w.Hash(), prefix.Hash())
			}
			if w.Hash() == recorded.Hash() {
				t.Error("Replay carried on past the mismatch and finished the log")
			}
		})
	}
}

func TestALogHasNoByteFormOfItsOwn(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(Log{}), reflect.TypeOf(&Log{}),
		reflect.TypeOf(Frame{}), reflect.TypeOf(&Frame{}),
	} {
		for i := 0; i < typ.NumMethod(); i++ {
			t.Errorf("%s has exported method %s", typ, typ.Method(i).Name)
		}
	}
}
