package game

import (
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

// These tests decode a snapshot into a restore candidate from plain values.
// No FrontEnd, screen or sound device is built.

func TestDecodeRestoreTownSnapshotBuildsATownCandidate(t *testing.T) {
	in := townInstall{campaign: Campaign{}}
	c, s, err := decodeRestore(Snapshot{Offered: 5, Party: []mapload.PartyMember{{Name: "Danath"}}}, in)
	if err != nil {
		t.Fatalf("decodeRestore: %v", err)
	}
	if !c.townOnly || c.offered != 5 || c.town == nil || len(c.carried) != 1 || c.carried[0].ID == "" {
		t.Fatalf("candidate = %+v, want a town-only candidate with a named party", c)
	}
	if len(s.Party) != 1 {
		t.Fatalf("validated snapshot party = %d", len(s.Party))
	}
}

func TestDecodeRestoreRefusesAMissionWithNoWorld(t *testing.T) {
	_, _, err := decodeRestore(Snapshot{Mission: 10}, townInstall{campaign: Campaign{}})
	if err == nil || !strings.Contains(err.Error(), "carries no world") {
		t.Fatalf("decodeRestore error = %v, want a refusal naming the missing world", err)
	}
}

func TestDecodeRestoreRefusesAMalformedCampaignRecord(t *testing.T) {
	c, _ := restoredTownFixture(t)
	_, _, err := decodeRestore(Snapshot{CampaignState: true}, townInstall{campaign: c})
	if err == nil || !strings.Contains(err.Error(), "main mission") {
		t.Fatalf("decodeRestore error = %v, want the campaign refusal", err)
	}
}

func TestDecodeRestoreRefusesATownOnlySaveBeforeTheCampaignTown(t *testing.T) {
	c := townCampaign(t)
	_, _, err := decodeRestore(Snapshot{CampaignState: true, Campaign: campaignProjectionAt(c, 20)}, townInstall{campaign: c})
	if err == nil || !strings.Contains(err.Error(), "before campaign town begins") {
		t.Fatalf("decodeRestore error = %v, want the pre-town refusal", err)
	}
}

// An original SAV's campaign tail decodes into the town it describes, over the
// install's campaign alone.
func TestDecodeOriginalCampaignRestoresTheTownFromTheCampaignTail(t *testing.T) {
	c, projection := restoredCampaignFixture()
	saved := originalSaveWithCampaign(t, 0, projection)
	sf, err := sav.Open(saved)
	if err != nil {
		t.Fatal(err)
	}
	d, err := decodeOriginalCampaign(sf, saved, c, [4]uint32{})
	if err != nil {
		t.Fatalf("decodeOriginalCampaign: %v", err)
	}
	if d.progress == nil || d.town == nil || d.mission != 0 {
		t.Fatalf("decode = %+v, want a restored town and its progress", d)
	}
	if d.town.lastMap != sf.Head.MapName {
		t.Fatalf("town-only save kept map %q, want the saved %q", d.town.lastMap, sf.Head.MapName)
	}
}

func TestDecodeOriginalCampaignRefusesATownOnlyFileBeforeTheCampaignTown(t *testing.T) {
	c := townCampaign(t)
	saved := originalSaveWithCampaign(t, 0, campaignProjectionAt(c, 20))
	sf, err := sav.Open(saved)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeOriginalCampaign(sf, saved, c, [4]uint32{}); err == nil || !strings.Contains(err.Error(), "before campaign town begins") {
		t.Fatalf("decodeOriginalCampaign error = %v, want the pre-town refusal", err)
	}
}
