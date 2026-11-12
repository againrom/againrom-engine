package game

import (
	"maps"
	"slices"
)

// winMissionsBelow records a win for every campaign mission below target, in
// ascending order: an auto-advance mission won after a later one sets main
// back, so a walk in map order prepares a different chapter on each run.
func winMissionsBelow(town *Town, c Campaign, target int) {
	for _, mission := range slices.Sorted(maps.Keys(c.Chapters)) {
		if mission < target {
			town.Won(mission)
		}
	}
}
