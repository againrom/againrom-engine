package game

import (
	"strings"

	"againrom/pkg/sim"
)

// cityExportNotice returns a player-facing disclosure naming which
// DIVERGENCES.md approximation a city SAV export actually used (DIV-1319,
// DIV-1320, DIV-1321, DIV-1322), or "" when none applied. It only reads state
// a writer already decided a few lines earlier (fameForOriginal,
// ExportNativeCitySave's DIV-1320 fallback, citySaveAdvisory); it refuses
// nothing itself. A mission-path export (world != nil) gets no notice of its
// own here; playerMissionSave folds that empty result into its own DIV-1334
// disclosure instead.
//
// rawOffered must be the session's own advisory value captured before
// citySnapshotFromMission normalizes city.Offered, or the DIV-1322 check
// below compares an already-normalized value against itself and never fires.
func cityExportNotice(n *FrontEnd, city Snapshot, rawOffered int, world *sim.World) string {
	if n == nil || n.Town == nil || world != nil {
		return ""
	}
	var notes []string
	if city.Fame != nil && !city.Fame.Known {
		notes = append(notes, "campaign score history predates this project's counters; wrote our own value in its place, not the original's (DIV-1319)")
	}
	if n.originalCity == nil && n.Town.progress != nil {
		notes = append(notes, "this city's original import record was lost; reconstructed it from current state instead (DIV-1320, DIV-1321)")
	}
	if rawOffered != n.Town.Chapter() {
		notes = append(notes, "the settled-town advisory banner does not survive this export and resets on load (DIV-1322)")
	}
	if len(notes) == 0 {
		return ""
	}
	return "Approximated for SAV: " + strings.Join(notes, "; ") + "."
}
