package game

import (
	"slices"

	"againrom/pkg/sim"
)

// flightStep runs one world step around the record memories the map world
// keeps: the pre-move points before it and the trails and looks after it.
func flightStep(mw *mapWorld) {
	mw.noteShotPreMoves()
	sim.StepReported(mw.world, nil)
	mw.advanceShotTrails()
}

// flightRecords is the World's armed records in id order.
func flightRecords(mw *mapWorld) []sim.SavedProjectile {
	records, _ := mw.armedRecords()
	slices.SortFunc(records, func(a, b sim.SavedProjectile) int { return int(a.ID) - int(b.ID) })
	return records
}
