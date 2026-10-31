package game

import (
	"testing"

	"againrom/pkg/data"
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

// installedNPCDefs is the installed npc.reg as production reads it.
func installedNPCDefs(t *testing.T, f *FrontEnd) *data.NPCDefs {
	t.Helper()
	catalog, err := LoadNPCCatalog(f.Archives.Containers)
	if err != nil {
		t.Fatal(err)
	}
	return catalog.Defs
}
