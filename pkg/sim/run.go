package sim

import "fmt"

// Frame is one advanced tick of a run: the commands applied at it, and the tick
// they were applied at.
//
// Tick is the tick the world held BEFORE the step this frame records — not the
// tick that step produced. That is what makes a frame's tick and a world's tick
// one quantity, and so what lets a replay compare them at all; recording the
// tick after the step would compare a frame against a tick the world no longer
// has.
//
// Commands is the frame's own list, empty where the run's schedule named none
// and never nil. It is never the caller's storage: a frame says what happened,
// and what happened must not change afterwards because somebody still holds the
// schedule it was read from.
type Frame struct {
	Tick     uint64
	Commands []Command
}

// Log is a run's frames in the order they were advanced, one per advanced tick —
// so a log's length IS the number of ticks the run advanced, with no separate
// count to keep true.
//
// A log has no byte form of its own. The versioned, self-contained encoding this
// package defines is the world's; a log lives in memory, is checked by replaying
// it into a world and comparing digests, and gains a wire shape when there is a
// wire.
type Log []Frame

// Run advances w by ticks steps against schedule and returns the frames it
// recorded, one per advanced tick.
//
// Step i takes schedule[i] where i < len(schedule) and no commands at all
// otherwise, so a schedule may be shorter than the run. schedule is indexed FROM
// THE RUN'S FIRST STEP and not by absolute tick: for a world at tick 0 — every
// world the loader builds — the two coincide, and continuing an interrupted run
// is expressed by slicing the schedule rather than by padding its front.
//
// One frame is recorded per advanced tick, the ones carrying no command
// included, so len(log) == ticks whenever ticks is positive and zero otherwise;
// a ticks of zero or less advances nothing. Each frame takes a copy of that
// step's commands, so mutating schedule after Run returns cannot change what the
// log says happened — and the copy is what is handed to Step, so the log records
// the very slice the world was advanced by.
//
// The world is advanced in place, by Step and by nothing else, so a run sets no
// field of a world that a step would not. Nothing here reads a clock, opens a
// file or touches a device: the same world and the same schedule are the same
// run wherever and whenever it is made.
func Run(w *World, schedule [][]Command, ticks int) Log {
	if ticks < 0 {
		ticks = 0
	}
	log := make(Log, 0, ticks)
	for i := 0; i < ticks; i++ {
		var cmds []Command
		if i < len(schedule) {
			cmds = schedule[i]
		}
		f := Frame{Tick: w.Tick(), Commands: append(make([]Command, 0, len(cmds)), cmds...)}
		log = append(log, f)
		Step(w, f.Commands)
	}
	return log
}

// Replay applies log's frames to w in order, checking each frame's tick against
// w's own tick before that frame is applied.
//
// It CONSTRUCTS NO WORLD. The world it advances is the one it was handed, which
// is what leaves "a log reproduces its run's final digest exactly when w is that
// run's initial world" a property that can be tested rather than one built in.
//
// A frame whose tick is not w's current tick is a log that does not belong to
// this world at this point in its life, and Replay refuses it: the error is
// returned BEFORE that frame is stepped, so the mismatched frame AND every frame
// after it leave w untouched. Frames already applied stay applied — a replay is
// not a transaction, and w is left at the tick the last accepted frame produced.
func Replay(w *World, log Log) error {
	for i, f := range log {
		if f.Tick != w.Tick() {
			return fmt.Errorf("sim: frame %d records tick %d but the world is at tick %d",
				i, f.Tick, w.Tick())
		}
		Step(w, f.Commands)
	}
	return nil
}
