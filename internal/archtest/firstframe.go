package archtest

// firstFrameOwners names, for each reader of a mission's first-frame
// application state, the production functions that may reference it. The
// first frame restores through the one restore, whose source is the save's
// decoded document or its mission file's state store; a direct read of
// either elsewhere is a second restore.
var firstFrameOwners = map[string][]string{
	"againrom/pkg/game.readFileApplication":              {"againrom/pkg/game.originalSource.prepareMission"},
	"againrom/pkg/game.documentApplication":              {"againrom/pkg/game.documentFirstFrame"},
	"againrom/pkg/game.documentFirstFrame":               {"againrom/pkg/game.firstFrameSourceOf"},
	"againrom/pkg/game.fileStoreFirstFrame":              {"againrom/pkg/game.firstFrameSourceOf"},
	"againrom/pkg/game.firstFrameSourceOf":               {firstFrameRestore},
	"againrom/pkg/game.firstFrameSource.firstFrame":      {firstFrameRestore},
	"againrom/pkg/game.fileApplication.view":             {"againrom/pkg/game.firstFrameSource.firstFrame"},
	"againrom/pkg/game.applyCurrentView":                 {"againrom/pkg/game.firstFrameSource.firstFrame"},
	"againrom/pkg/game.originalFogPlane":                 {"againrom/pkg/game.documentFirstFrame", "againrom/pkg/game.validateOriginalApplicationSource"},
	"againrom/pkg/game.mapWorld.restoreApplicationState": {"againrom/pkg/game.settleMission"},
}

const firstFrameRestore = "againrom/pkg/game.mapWorld.restoreApplicationState"

// CheckFirstFrame reports each production reference to a function of
// firstFrameOwners from a function the table does not name.
func CheckFirstFrame(packages []*CheckedPackage) []string {
	return checkReferenceOwners(packages, firstFrameOwners)
}
