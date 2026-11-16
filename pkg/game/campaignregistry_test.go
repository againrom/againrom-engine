package game

import (
	"testing"
)

// installedCampaign is the installed scenario registry as production reads it
// (LoadCampaign over ReadCampaign).
func installedCampaign(t *testing.T, f *FrontEnd) Campaign {
	t.Helper()
	c, err := LoadCampaign(f.Archives.Containers)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
