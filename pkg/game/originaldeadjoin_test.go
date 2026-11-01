package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
)

// applyOriginalDead is the dead body row's archive decode over source with at
// most one current document and no object origins, for the tests that drive
// the join alone.
func applyOriginalDead(ms *Mission, source []sav.DeadActor, report *OriginalSaveResume, states ...*SnapshotSAVDocument) error {
	var state *SnapshotSAVDocument
	if len(states) > 1 {
		return fmt.Errorf("original dead actors: repeated current document")
	}
	if len(states) == 1 {
		state = states[0]
	}
	return applyOriginalDeadWithOrigins(ms, source, report, state, nil)
}
