package archtest

import "testing"

// An original file's sources are chosen and read in one table.
func TestOriginalFileSourcesAreOneBoundary(t *testing.T) {
	packages, defined := ownedProductionFunctions(t)
	checkOwnersDefined(t, originalFileSourceOwners, defined)
	for _, finding := range CheckOriginalFileSources(packages) {
		t.Error(finding)
	}
}
