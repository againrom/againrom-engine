package archtest

// CommittedSpellDeletionSites is each production file's Spell deletion count
// outside the worn-change transaction: SAV projections that split a Spell into
// its own row, the registry's own row write, the native basis comparison copy
// and the source actor's removal virtual. A fall lowers its row in the same
// commit; a rise or a new file fails.
var CommittedSpellDeletionSites = map[string]int{
	"pkg/game/cityobjectprojection.go": 2,
	"pkg/game/currentobjects.go":       2,
	"pkg/sim/nativebasismutation.go":   2,
	"pkg/sim/savedobjectsmutation.go":  1,
	"pkg/sim/sourceequipmove.go":       1,
}
