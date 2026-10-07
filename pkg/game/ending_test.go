package game

import (
	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
	"testing"
)

func TestEndingRequiresInstalledFlagAndMainMission(t *testing.T) {
	raw := synth.Reg(kindRoot, []synth.RegNode{
		{Name: "Mission20", Kind: kindDir},
		{Name: "Mission30", Kind: kindDir, Children: []synth.RegNode{{Name: "LastMission", Kind: kindInt, Int: -1}}},
		{Name: "Mission31", Kind: kindDir, Children: []synth.RegNode{{Name: "LastMission", Kind: kindInt, Int: 1}}},
	})
	r, err := reg.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	c := ReadCampaign(r)
	if !c.Last[31] || c.TerminalMission(31) || c.TerminalMission(20) || !c.TerminalMission(30) || c.TerminalMission(150) {
		t.Fatal("ending predicate", c.Last)
	}
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	f.Town.Won(20)
	if f.completedCampaign() {
		t.Fatal("no-successor alone ended campaign")
	}
	f.Town.Won(31)
	if f.completedCampaign() {
		t.Fatal("side flag ended campaign")
	}
	f.Town.Won(30)
	if !f.completedCampaign() {
		t.Fatal("terminal win lost")
	}
}
