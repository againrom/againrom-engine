package game

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"sort"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// campaignRecordBaseDigests are the bytes each route wrote before its
// campaign record went through the one projection: the SAV without its
// label, the snapshot's carried record and its offer labels. The second
// game's mission file SAVE also holds the loaded tail's words a town SAVE
// writes back (DIV-4306).
var campaignRecordBaseDigests = map[string]string{
	// The loaded computer Player's zero-length Diary is kept, not rebuilt
	// at the Units length.
	"mission progress/sav":               "d959651b195ec8172e290369",
	"mission progress/snapshot campaign": "d1b0dad36e60fe233e101fa6",
	"second mission file/resave":         "d6ef2e98d492c411ed85fa2e",
	"second mission file/sav":            "d6ef2e98d492c411ed85fa2e",
	"town chapter/sav":                   "e389207f3b64c63e1b653762",
	"town chapter/snapshot campaign":     "be2e971a485a96bdb461ac08",
	"town chapter/snapshot labels":       "0ca964b9fe0dc5fa909b358c",
	"town progress/sav":                  "f57fdfa564a2a91f25bcd80b",
	"town progress/snapshot campaign":    "787aea6f045c8cdcc9fcfa3c",
	"town progress/snapshot labels":      "355f8819b278efec7b034c20",
}

// campaignRecordSAVDigest is a SAV's digest with its label cleared, so a
// label that names the time does not enter it.
func campaignRecordSAVDigest(t *testing.T, raw []byte) string {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	doc.Label = nil
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(out)
	return hex.EncodeToString(sum[:12])
}

func campaignRecordValueDigest(v any) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%#v", v)))
	return hex.EncodeToString(sum[:12])
}

// Each route that writes a campaign record writes the bytes it wrote before:
// a town over its chapter, a town over loaded progress with a selected
// marker, a mission over loaded progress, and a second-game mission file's
// SAVE and its resave, whose offer labels keep chapter 10.
func TestCampaignRecordRoutesWriteTheBaseBytes(t *testing.T) {
	got := map[string]string{}
	exportTown := func(name string, f *FrontEnd) {
		s, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(name, err)
		}
		got[name+"/snapshot campaign"] = campaignRecordValueDigest([]any{s.CampaignState, s.Campaign, s.CampaignMarkerSelected})
		got[name+"/snapshot labels"] = campaignRecordValueDigest(s.OfferLabels)
		raw, err := f.ExportCurrentSave(s, label)
		if err != nil {
			t.Fatal(name, err)
		}
		got[name+"/sav"] = campaignRecordSAVDigest(t, raw)
	}

	chapter := currentTrainingCity(t)
	chapter.Campaign = resolved(saveCampaign(), nil)
	chapter.Town, chapter.Offered = saveTown(t), 20
	exportTown("town chapter", chapter)

	carried := restoredWorldMapFront(t, true, false)
	seed := currentTrainingCity(t)
	carried.Table, carried.Carried = seed.Table, mapload.CloneParty(seed.Carried)
	screen := carried.TownScreen().(*townScreen)
	screen.Choose(3)
	seedWorldMapMarker(carried)
	card := ui.WorldMapCardRect(0)
	screen.WorldMapClick(image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2))
	exportTown("town progress", carried)

	mission, _ := itemMutationOpen(t, 1, false)
	s := snapshotCurrentObjects(t, mission)
	got["mission progress/snapshot campaign"] = campaignRecordValueDigest([]any{s.CampaignState, s.Campaign})
	raw, err := mission.ExportCurrentSave(s, "campaign record")
	if err != nil {
		t.Fatal(err)
	}
	got["mission progress/sav"] = campaignRecordSAVDigest(t, raw)

	file := secondMissionFile(secondMissionFixtureHero(), secondMissionStore(4, 0, -1))
	f, _, dir := secondMissionLoadFile(t, file)
	name, first := worldCarrierSave(t, f, dir)
	cold, _ := secondMissionLoad(t, dir, name)
	_, second := worldCarrierSave(t, cold, t.TempDir())
	if a, b := savedOfferChapter(first), savedOfferChapter(second); a != "10" || b != "10" {
		t.Fatalf("the second game's mission file SAVE writes offer chapter %s and its resave %s, want 10 and 10", a, b)
	}
	got["second mission file/sav"] = campaignRecordSAVDigest(t, first)
	got["second mission file/resave"] = campaignRecordSAVDigest(t, second)

	names := make([]string, 0, len(got))
	for name := range got {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if want := campaignRecordBaseDigests[name]; got[name] != want {
			t.Errorf("%q: %q, want %q", name, got[name], want)
		}
	}
}
