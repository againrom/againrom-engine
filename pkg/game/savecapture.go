package game

import "fmt"

// saveCaptureMode is what a SAVE capture may do to the live game.
type saveCaptureMode uint8

const (
	// captureTentative captures for a file that may not be written: the
	// save dialog before the player has confirmed an overwrite.
	captureTentative saveCaptureMode = iota + 1
	// captureCommitted captures for a file the player SAVEs: the menu SAVE,
	// the quicksave and the save dialog once its file is written.
	captureCommitted
	// captureTimer captures for a file no player asked for: the autosaves,
	// the mod route's stash and the witnesses.
	captureTimer
)

// saveCaptureTransform is one mode's row. pruneGroups removes from the live
// lists the Groups the original's SAVE removes (SAV-1251); restore puts them
// back once the build returns.
type saveCaptureTransform struct {
	pruneGroups bool
	restore     bool
}

// saveCaptureTransforms is the one table of which mode applies which
// transform. A timer must not change hashed state, so like a tentative
// capture it builds from the pruned lists and leaves the live game as it
// was.
var saveCaptureTransforms = map[saveCaptureMode]saveCaptureTransform{
	captureTentative: {pruneGroups: true, restore: true},
	captureCommitted: {pruneGroups: true},
	captureTimer:     {pruneGroups: true, restore: true},
}

// captureSave is the one SAVE capture: every route that captures the running
// game calls it with its mode. It applies the mode's transforms, captures,
// and calls build with the snapshot and its label while they hold. A nil
// build applies the transforms and captures nothing: the save dialog
// commits the transforms of the file it has written.
func (f *FrontEnd) captureSave(mode saveCaptureMode, onMap bool, build func(Snapshot, string) error) error {
	row, ok := saveCaptureTransforms[mode]
	if !ok {
		return fmt.Errorf("save capture mode %d", mode)
	}
	if row.pruneGroups {
		undo := pruneGroupsForSave(f, onMap)
		if row.restore {
			defer undo()
		}
	}
	if build == nil {
		return nil
	}
	s, label, err := f.Snapshot(onMap)
	if err != nil {
		return err
	}
	return build(s, label)
}

// pruneGroupsForSave removes the Groups the original's SAVE removes from the
// live lists a LOAD set, the mission World's or the town's, and returns the
// function that restores them.
func pruneGroupsForSave(f *FrontEnd, onMap bool) func() {
	if onMap && f.live != nil && f.live.world != nil && f.liveMission != 0 {
		_, undo := f.live.world.PruneGroupsForSaveUndo()
		return undo
	}
	if !onMap && f.Town != nil {
		before := f.Town.cityGroups
		f.Town.pruneCityGroupsForSave(f.Carried)
		return func() { f.Town.cityGroups = before }
	}
	return func() {}
}
