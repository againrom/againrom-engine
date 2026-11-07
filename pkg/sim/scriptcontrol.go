package sim

import "fmt"

// NewControlledScriptWorld copies a normally constructed mission world and
// replaces its program and execution state with fresh state for that dialect.
// The byte form copies the mission, including existing non-program effects;
// ordinary stepping executes the replacement without modifying the source.
func NewControlledScriptWorld(base *World, script *Script) (*World, error) {
	if base == nil {
		return nil, fmt.Errorf("sim: no mission world for controlled script")
	}
	if script == nil {
		return nil, fmt.Errorf("sim: no replacement program for controlled script")
	}
	form, err := base.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("sim: copy controlled script world: %w", err)
	}
	out := *base
	if err := out.UnmarshalBinary(form); err != nil {
		return nil, fmt.Errorf("sim: restore controlled script world: %w", err)
	}
	// The trigger binder reads the new script (AI-452), and a copy into the
	// other game takes that game's constructor. A registry play has changed
	// is kept whole: a copy into the other game asks it of the copied
	// World's own game and script, a copy in the same game of the new one.
	held := out.savedGroups.fresh
	rebuild := held && out.script.Dialect() != script.Dialect() && (out.savedGroups.native || out.groupsAsConstructed())
	out.script = script
	out.initializeROM2Script()
	native := false
	if rebuild || held && (out.savedGroups.native || out.groupsAsConstructed()) {
		native, out.savedGroups = out.savedGroups.native, nil
	}
	out.constructCarriers(native)
	out.reserveScriptEntityIDs()
	out.registers = [scriptRegisters]int32{}
	out.latches = [scriptLatches]byte{}
	out.scriptVars, out.scriptResults, out.latchTail = nil, nil, nil
	out.fitScriptState()
	out.won, out.lost, out.outcome = 0, 0, OutcomeUndecided
	out.casts = nil
	out.presetRegisters()
	return &out, nil
}

// RestoreScriptProgram replaces only the compiled mission program and keeps
// the running program state (registers, latches, counters and outcome). A
// current SAV may learn a bounded objective after its party has been restored;
// that objective changes the program shape while the execution state remains
// the saved world's state. An extended program's own state is fitted to the
// new program, keeping what fits.
func (w *World) RestoreScriptProgram(program *Script) error {
	if w == nil || program == nil {
		return fmt.Errorf("sim: cannot restore an absent script program")
	}
	if w.script.Dialect() != program.Dialect() {
		return fmt.Errorf("sim: cannot change restored script dialect")
	}
	restored, err := newScriptOn(program.Dialect(), program.machine, program.Checks(), program.Instants(), program.Triggers())
	if err != nil {
		return err
	}
	w.script = restored
	w.fitScriptState()
	w.reserveScriptEntityIDs()
	return nil
}
