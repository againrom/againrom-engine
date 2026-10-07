package sim

import "fmt"

// RestoreCurrentContinuation validates policy, actor values and actions as one
// detached LOAD transaction. A failed field or binding leaves the World intact.
func (w *World) RestoreCurrentContinuation(policy *CurrentWorldPolicy, values map[EntityID]ActorValues, actions ActionContinuations, objects map[SavedObjectID]SavedObjectID, terminalMotions ...CurrentTerminalMotion) error {
	next := *w
	if policy != nil {
		if err := next.restoreCurrentPolicy(*policy); err != nil {
			return err
		}
	}
	if values != nil {
		if err := next.restoreActorValues(values); err != nil {
			return err
		}
		if err := restoreAttachedEffectWidths(&next, values); err != nil {
			return err
		}
	}
	if err := next.RestoreActions(actions, objects); err != nil {
		return err
	}
	if err := next.restoreCurrentTerminalMotions(terminalMotions); err != nil {
		return err
	}
	if policy != nil && policy.EntityIDFloor != nil {
		if *policy.EntityIDFloor > entityIDLimit {
			return fmt.Errorf("sim: current actor identity floor exceeds namespace")
		}
		next.entityIDFloor = max(next.entityIDFloor, *policy.EntityIDFloor)
		next.reserveAbsentEntityIDs(next.ActorIdentityReferences())
	}
	*w = next
	return nil
}
