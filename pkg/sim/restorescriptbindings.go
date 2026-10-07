package sim

import "slices"

// RestoreScriptBindings fills missing unit references only when the saved
// program exactly matches legacy and current changes no other program fields.
// It returns whether bindings changed and preserves all execution state.
func (w *World) RestoreScriptBindings(legacy, current *Script) bool {
	if w == nil || w.script == nil || legacy == nil || current == nil ||
		legacy.Dialect() != current.Dialect() ||
		!sameScriptProgram(w.script, legacy) ||
		len(legacy.checks) != len(current.checks) ||
		len(legacy.instants) != len(current.instants) ||
		!slices.Equal(legacy.triggers, current.triggers) {
		return false
	}

	changed := false
	for i, before := range legacy.checks {
		after := current.checks[i]
		if !before.HasUnit && after.HasUnit {
			before.Unit, before.HasUnit = after.Unit, true
			changed = true
		}
		if !before.HasUnit2 && after.HasUnit2 {
			before.Unit2, before.HasUnit2 = after.Unit2, true
			changed = true
		}
		if before != after {
			return false
		}
	}
	for i, before := range legacy.instants {
		after := current.instants[i]
		if !before.HasUnit && after.HasUnit {
			before.Unit, before.HasUnit = after.Unit, true
			changed = true
		}
		if !before.HasUnit2 && after.HasUnit2 {
			before.Unit2, before.HasUnit2 = after.Unit2, true
			changed = true
		}
		if before != after {
			return false
		}
	}
	if !changed {
		return false
	}

	restored, err := newScript(current.Dialect(), current.checks, current.instants, current.triggers)
	if err != nil {
		return false
	}
	w.script = restored
	w.reserveScriptEntityIDs()
	return true
}

func sameScriptProgram(a, b *Script) bool {
	return a.Dialect() == b.Dialect() && slices.Equal(a.checks, b.checks) &&
		slices.Equal(a.instants, b.instants) &&
		slices.Equal(a.triggers, b.triggers)
}
