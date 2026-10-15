package game

import (
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
)

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
