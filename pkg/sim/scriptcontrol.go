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
	out.script = script
	out.rom2 = nil
	out.initializeROM2Script()
	out.reserveScriptEntityIDs()
	out.registers = [scriptRegisters]int32{}
	out.latches = [scriptLatches]byte{}
	out.won, out.lost, out.outcome = 0, 0, OutcomeUndecided
	out.casts = nil
	out.presetRegisters()
	return &out, nil
}

// RestoreScriptProgram replaces only the compiled mission program and keeps
// the running program state (registers, latches, counters and outcome). A
// current SAV may learn a bounded objective after its party has been restored;
// that objective changes the program shape while the execution state remains
// the saved world's state.
func (w *World) RestoreScriptProgram(program *Script) error {
	if w == nil || program == nil {
		return fmt.Errorf("sim: cannot restore an absent script program")
	}
	if w.script.Dialect() != program.Dialect() {
		return fmt.Errorf("sim: cannot change restored script dialect")
	}
	restored, err := newScript(program.Dialect(), program.Checks(), program.Instants(), program.Triggers())
	if err != nil {
		return err
	}
	w.script = restored
	w.reserveScriptEntityIDs()
	return nil
}
