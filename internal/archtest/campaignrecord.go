package archtest

// campaignRecordOwners names, for each source, overlay and profile of the
// campaign record a SAV carries, the production functions that may
// reference it. Every route builds the record through the one projection per
// document kind, so a direct call of a source or of the tail word elsewhere
// is a second order of calls.
var campaignRecordOwners = map[string][]string{
	"againrom/pkg/game.nativeCampaignProjection":           {campaignRecordRoute},
	"againrom/pkg/game.generatedChapterProjection":         {campaignRecordRoute},
	"againrom/pkg/game.nativeCampaignProjectionForChapter": {"againrom/pkg/game.nativeCampaignProjection", "againrom/pkg/game.generatedChapterProjection"},
	"againrom/pkg/game.FrontEnd.campaignTail":              {"againrom/pkg/game.FrontEnd.overlayCampaignRecord"},
	"againrom/pkg/game.FrontEnd.overlayCampaignRecord":     {campaignRecordProjection},
	"againrom/pkg/game.campaignRecordRouteOf":              {campaignRecordProjection},
	"againrom/pkg/game.campaignService.recordProfile":      {campaignRecordProjection, "againrom/pkg/game.snapshotTown"},
	"againrom/pkg/game.firstCampaignRules.recordProfile":   {campaignRecordProjection, "againrom/pkg/game.snapshotTown"},
	"againrom/pkg/game.secondCampaignRules.recordProfile":  {campaignRecordProjection, "againrom/pkg/game.snapshotTown"},
	"againrom/pkg/game.secondStatedRecord":                 {"againrom/pkg/game.secondCampaignRules.recordProfile"},
	"againrom/pkg/game.captureCampaignRecord":              {"againrom/pkg/game.snapshotTown"},
	"againrom/pkg/game.capturedOfferLabels":                {"againrom/pkg/game.captureCampaignRecord"},
	"againrom/pkg/game.progressCampaignRecord":             {"againrom/pkg/game.captureCampaignRecord"},
	"againrom/pkg/game.campaignProgress.projection":        {"againrom/pkg/game.progressCampaignRecord"},
	"againrom/pkg/game.FrontEnd.seatCarriedCampaignRecord": {"againrom/pkg/game.FrontEnd.ExportCurrentSave"},
	campaignRecordProjection:                               {"againrom/pkg/game.FrontEnd.currentMissionDocument", "againrom/pkg/game.FrontEnd.currentCityDocument", "againrom/pkg/game.FrontEnd.Snapshot"},
}

const (
	campaignRecordProjection = "againrom/pkg/game.FrontEnd.campaignRecord"
	campaignRecordRoute      = "againrom/pkg/game.campaignRecordRouteOf"
)

// CheckCampaignRecord reports each production reference to a function of
// campaignRecordOwners from a function the table does not name.
func CheckCampaignRecord(packages []*CheckedPackage) []string {
	return checkReferenceOwners(packages, campaignRecordOwners)
}
