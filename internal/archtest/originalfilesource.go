package archtest

// originalFileSourceOwners names, for each read of a state that comes from an
// original file, the production functions that may reference it. The source
// table is the one place that chooses a source and holds its readers; the
// read moment and each group source have one production site, its row. A read
// reached any other way is a second builder of the source.
var originalFileSourceOwners = map[string][]string{
	"againrom/pkg/game.readFileApplication":          {originalFileSourceTable},
	"againrom/pkg/game.applyOriginalBuildingCasters": {originalFileSourceTable},
	"againrom/pkg/game.cityGroupsFromLoaded":         {originalFileSourceTable},
	"againrom/pkg/game.cityGroupsFromGraph":          {originalFileSourceTable},
	"againrom/pkg/game.cityGroupsFromCurrent":        {originalFileSourceTable},
	"againrom/pkg/game.loadedPlayerGroups":           {"againrom/pkg/game.cityGroupsFromLoaded", "againrom/pkg/game.cityGroupsFromCurrent"},
	"againrom/pkg/game.originalFileSources": {
		"againrom/pkg/game.originalTownSource",
		"againrom/pkg/game.originalSource.readMissionApplication",
		"againrom/pkg/game.firstFrameSourceOf",
		"againrom/pkg/game.FrontEnd.captureApplicationState",
		"againrom/pkg/game.originalSource.adoptMissionState",
	},
	"againrom/pkg/game.originalTownSource":                    {"againrom/pkg/game.CampaignSession.adoptTownGroups"},
	"againrom/pkg/game.CampaignSession.adoptTownGroups":       {"againrom/pkg/game.CampaignSession.restoreOriginalTown"},
	"againrom/pkg/game.originalSource.readMissionApplication": {"againrom/pkg/game.originalSource.prepareMission"},
	"againrom/pkg/game.originalSource.adoptMissionState":      {"againrom/pkg/game.originalSource.prepareMission"},
	"againrom/pkg/game.readOriginalApplicationState": {
		"againrom/pkg/game.documentApplication",
		"againrom/pkg/game.validateOriginalApplicationSource",
		originalFileSourceTable,
		"againrom/pkg/game.projectApplicationState",
	},
	"againrom/pkg/game.originalApplicationStateOf": {
		"againrom/pkg/game.readOriginalApplicationState",
		"againrom/pkg/game.readFileApplication",
	},
	"againrom/pkg/game.applicationActorIDs":   {"againrom/pkg/game.documentApplication", "againrom/pkg/game.projectApplicationState"},
	"againrom/pkg/game.applicationCurrentRaw": {"againrom/pkg/game.projectCurrentSession", "againrom/pkg/game.projectCityApplication", "againrom/pkg/game.projectApplicationState"},
}

const originalFileSourceTable = "againrom/pkg/game.originalFileSources"

// CheckOriginalFileSources reports each production reference to a function of
// originalFileSourceOwners from a function the table does not name. Every use
// of an identifier counts, a function value and an import alias included.
func CheckOriginalFileSources(packages []*CheckedPackage) []string {
	return checkReferenceOwners(packages, originalFileSourceOwners)
}
