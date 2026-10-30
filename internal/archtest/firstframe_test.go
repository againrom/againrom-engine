package archtest

import "testing"

// A mission's first frame restores through the one restore; its two
// sources' readers are referenced only through it.
func TestFirstFramesRestoreThroughOneRestore(t *testing.T) {
	packages, defined := ownedProductionFunctions(t)
	checkOwnersDefined(t, firstFrameOwners, defined)
	for _, finding := range CheckFirstFrame(packages) {
		t.Error(finding)
	}
}
