package archtest

import "testing"

// Every route that writes a campaign record into a SAV builds it through
// the one projection; its sources, its tail word and the campaign profiles
// are referenced only by the projection and the snapshot's capture.
func TestCampaignRecordsGoThroughOneProjection(t *testing.T) {
	packages, defined := ownedProductionFunctions(t)
	checkOwnersDefined(t, campaignRecordOwners, defined)
	for _, finding := range CheckCampaignRecord(packages) {
		t.Error(finding)
	}
}
