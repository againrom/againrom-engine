package game

import "againrom/pkg/sim"

// applyOriginalDying finishes the detached actor construction, after stock,
// profile, pools, groups and document bindings. Both LOAD doors call it before
// adopting the candidate. It never adds Player records to the dead manager.
func applyOriginalDying(ms *Mission, report *OriginalSaveResume) error {
	var batch []sim.OriginalDyingActor
	for _, binding := range ms.actorRegistry.actors {
		a := binding.Source
		// A current actor whose width operand restores positive health is
		// living; its terminal wire stage cannot name it.
		if h, current := ms.actorRegistry.health[binding.ID]; a.Dying() && (!current || h.HP <= 0) {
			batch = append(batch, sim.OriginalDyingActor{ID: binding.ID, HP: a.HP, Timer: a.DyingTimer})
		}
	}
	if err := ms.World.ImportOriginalDyingActors(batch); err != nil {
		return err
	}
	report.DyingRestored = len(batch)
	return importSavedActorActions(ms, ms.savedDocument)
}
