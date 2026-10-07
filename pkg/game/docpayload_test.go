package game

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
)

const docPayloadColdEnv = "AGAINROM_DOCPAYLOAD_COLD_SAV"

func docPayloadFixtureBytes() []byte {
	out := make([]byte, 301)
	rand.New(rand.NewSource(11)).Read(out)
	return out
}

func docPayloadFixtureFront(t *testing.T) (*FrontEnd, func() *FrontEnd) {
	t.Helper()
	f, newFront := cityRosterHiredFixture(t, []int{3}, []int{1})
	for _, d := range []Document{{Value: 7001, Kind: DocumentText}, {Value: 7002, Kind: DocumentPicture}, {Value: 7003, Kind: DocumentText}} {
		f.Town.addDocument(d)
	}
	return f, newFront
}

func savCampaign(t *testing.T, raw []byte) sav.CampaignProjection {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	campaign, ok, err := file.Campaign()
	if err != nil || !ok {
		t.Fatalf("SAV campaign record: ok=%v err=%v", ok, err)
	}
	return campaign
}

// docPairRun finds the campaign document run of raw by its three vanilla
// pairs and returns its offset: the u32 count sits at the offset.
func docPairRun(t *testing.T, raw []byte) int {
	t.Helper()
	var run []byte
	for _, v := range [][2]uint32{{7001, 1}, {7002, 0}, {7003, 1}} {
		run = binary.LittleEndian.AppendUint32(run, v[0])
		run = binary.LittleEndian.AppendUint32(run, v[1])
	}
	at := bytes.Index(raw, run)
	if at < 4 || bytes.Contains(raw[at+1:], run) {
		t.Fatal("document run is absent or ambiguous in the SAV")
	}
	return at - 4
}

// withExtraPairs appends pairs after the document run of raw and raises the
// count, the way the original keeps carriers it did not write.
func withExtraPairs(t *testing.T, raw []byte, pairs [][2]uint32) []byte {
	t.Helper()
	at := docPairRun(t, raw)
	count := binary.LittleEndian.Uint32(raw[at:])
	end := at + 4 + 8*int(count)
	out := append([]byte(nil), raw[:end]...)
	for _, p := range pairs {
		out = binary.LittleEndian.AppendUint32(out, p[0])
		out = binary.LittleEndian.AppendUint32(out, p[1])
	}
	out = append(out, raw[end:]...)
	binary.LittleEndian.PutUint32(out[at:], count+uint32(len(pairs)))
	return out
}

func docPairCount(t *testing.T, raw []byte) uint32 {
	t.Helper()
	return binary.LittleEndian.Uint32(raw[docPairRun(t, raw):])
}

func requireVanillaDocuments(t *testing.T, f *FrontEnd) {
	t.Helper()
	want := []Document{{Value: 7001, Kind: DocumentText}, {Value: 7002, Kind: DocumentPicture}, {Value: 7003, Kind: DocumentText}}
	if got := f.Town.Documents(); !reflect.DeepEqual(got, want) {
		t.Fatalf("the panel's document list = %v, want %v", got, want)
	}
}

func TestDocPayloadSurvivesSaveAndColdLoad(t *testing.T) {
	if path := os.Getenv(docPayloadColdEnv); path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		_, newFront := docPayloadFixtureFront(t)
		cold := cityProjectionLoad(t, raw, newFront)
		got := cold.Town.DocPayload()
		if got == nil || got.Version != sav.PayloadVersion || !bytes.Equal(got.Data, docPayloadFixtureBytes()) {
			t.Fatalf("cold LOAD in a fresh process lost the payload: %v", got)
		}
		requireVanillaDocuments(t, cold)
		return
	}
	f, newFront := docPayloadFixtureFront(t)
	want := &sav.DocPayload{Version: sav.PayloadVersion, Data: docPayloadFixtureBytes()}
	f.Town.SetDocPayload(want)

	raw := currentTownSave(t, f)
	campaign := savCampaign(t, raw)
	if !reflect.DeepEqual(campaign.Payload, want) || campaign.PayloadError != "" || len(campaign.Documents) != 3 {
		t.Fatalf("SAV carries payload %v (%q) and %d vanilla documents", campaign.Payload, campaign.PayloadError, len(campaign.Documents))
	}
	carriers := docPairCount(t, raw) - 3
	if carriers < 2 {
		t.Fatalf("the written SAV holds %d carrier pairs", carriers)
	}

	// A second SAVE of a loaded session writes the block once, not twice.
	cold := cityProjectionLoad(t, raw, newFront)
	if got := cold.Town.DocPayload(); !reflect.DeepEqual(got, want) {
		t.Fatalf("LOAD holds payload %v", got)
	}
	requireVanillaDocuments(t, cold)
	cur := cold
	for cycle := 0; cycle < 3; cycle++ {
		again := currentTownSave(t, cur)
		if n := docPairCount(t, again) - 3; n != carriers {
			t.Fatalf("SAVE %d writes %d carrier pairs, want %d", cycle, n, carriers)
		}
		if !reflect.DeepEqual(savCampaign(t, again).Payload, want) {
			t.Fatalf("SAVE %d changed the payload", cycle)
		}
		cur = cityProjectionLoad(t, again, newFront)
		requireVanillaDocuments(t, cur)
	}

	// Loss control: a SAVE that drops the payload leaves a SAV the checks
	// above would refuse.
	f.Town.SetDocPayload(nil)
	dropped := currentTownSave(t, f)
	if savCampaign(t, dropped).Payload != nil || docPairCount(t, dropped) != 3 {
		t.Fatal("a session without a payload still wrote carriers")
	}
	if cityProjectionLoad(t, dropped, newFront).Town.DocPayload() != nil {
		t.Fatal("a SAV without carriers produced a payload")
	}

	path := filepath.Join(t.TempDir(), "carrier.sav")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestDocPayloadSurvivesSaveAndColdLoad$", "-test.v")
	cmd.Env = append(os.Environ(), docPayloadColdEnv+"="+path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cold LOAD in a separate process: %v\n%s", err, out)
	}
}

func TestDocPayloadStrayAndUnknownVersionCarriers(t *testing.T) {
	f, newFront := docPayloadFixtureFront(t)
	base := currentTownSave(t, f)

	stray := withExtraPairs(t, base, [][2]uint32{{1, 0x80000000 | 0x1234567}, {1, 0x80000000 | 0x7654321}})
	cold := cityProjectionLoad(t, stray, newFront)
	if cold.Town.DocPayload() != nil || cold.Town.DocPayloadError() == "" {
		t.Fatalf("stray carriers loaded as %v, diagnostic %q", cold.Town.DocPayload(), cold.Town.DocPayloadError())
	}
	requireVanillaDocuments(t, cold)
	if n := docPairCount(t, currentTownSave(t, cold)); n != 3 {
		t.Fatalf("a SAVE wrote %d document pairs, want the 3 vanilla ones", n)
	}

	later := &sav.DocPayload{Version: 7, Data: []byte("layout from a later version")}
	f.Town.SetDocPayload(later)
	unknown := currentTownSave(t, f)
	cold = cityProjectionLoad(t, unknown, newFront)
	if got := cold.Town.DocPayload(); !reflect.DeepEqual(got, later) || cold.Town.DocPayloadError() != "" {
		t.Fatalf("unknown version loaded as %v (%q)", got, cold.Town.DocPayloadError())
	}
	again := currentTownSave(t, cold)
	if !bytes.Equal(again[docPairRun(t, again):docPairRun(t, again)+4+8*int(docPairCount(t, again))],
		unknown[docPairRun(t, unknown):docPairRun(t, unknown)+4+8*int(docPairCount(t, unknown))]) {
		t.Fatal("an unknown-version payload was not written back verbatim")
	}
}

// TestReleaseDocPayloadOwnerResaves reads the original's own resaves of carrier-laden
// SAVs through the production reader. Their carriers are raw sequence values,
// not an envelope, so they take the stray path.
func TestReleaseDocPayloadOwnerResaves(t *testing.T) {
	dir := os.Getenv("AGAINROM_DOCS_CARRIER_KIT")
	if dir == "" {
		t.Skip("AGAINROM_DOCS_CARRIER_KIT is not set; it names the owner docs carrier kit directory")
	}
	for _, c := range []struct {
		file    string
		carrier int
	}{{"game0002-resave.sav", 4}, {"game0003-resave-9501.sav", 1000}, {"game0004-resave-9502.sav", 26426}} {
		raw, err := os.ReadFile(filepath.Join(dir, "candidates", c.file))
		if err != nil {
			t.Fatal(err)
		}
		campaign := savCampaign(t, raw)
		want := []sav.CampaignDocument{{Value: 1, Kind: 1}, {Value: 2, Kind: 1}, {Value: 3, Kind: 1}}
		if !reflect.DeepEqual(campaign.Documents, want) || campaign.Payload != nil || campaign.PayloadError == "" {
			t.Fatalf("%s: documents %v payload %v diagnostic %q", c.file, campaign.Documents, campaign.Payload, campaign.PayloadError)
		}
		if campaign.CarrierRecords != c.carrier {
			t.Fatalf("%s: %d carrier records, want %d", c.file, campaign.CarrierRecords, c.carrier)
		}
	}
}

func TestDocPayloadBoundAccountsForVanillaPairs(t *testing.T) {
	const listBound = 65536
	f, newFront := docPayloadFixtureFront(t)
	edge := func(vanilla int) int { return (31*(listBound-vanilla))/8 - 14 }
	fits := &sav.DocPayload{Version: sav.PayloadVersion, Data: bytes.Repeat([]byte{0x5a}, edge(3))}
	if err := f.Town.SetDocPayload(&sav.DocPayload{Version: sav.PayloadVersion, Data: bytes.Repeat([]byte{1}, edge(3)+1)}); err == nil {
		t.Fatal("SetDocPayload took a payload one byte past the edge for 3 documents")
	}
	if f.Town.DocPayload() != nil {
		t.Fatal("a refused SetDocPayload changed the held payload")
	}
	if err := f.Town.SetDocPayload(fits); err != nil {
		t.Fatalf("SetDocPayload at the edge for 3 documents: %v", err)
	}
	raw := currentTownSave(t, f)
	if n := docPairCount(t, raw); n != listBound {
		t.Fatalf("edge SAVE wrote %d document pairs, want %d", n, listBound)
	}
	if got := f.Town.DocPayload(); !reflect.DeepEqual(got, fits) || f.Town.DocPayloadError() != "" {
		t.Fatalf("edge SAVE changed the payload or set %q", f.Town.DocPayloadError())
	}
	cold := cityProjectionLoad(t, raw, newFront)
	if got := cold.Town.DocPayload(); !reflect.DeepEqual(got, fits) {
		t.Fatal("edge payload did not survive the cold load")
	}

	f.Town.addDocument(Document{Value: 7004, Kind: DocumentText})
	raw = currentTownSave(t, f)
	if n := docPairCount(t, raw); n != 4 {
		t.Fatalf("SAVE beside an unfit payload wrote %d document pairs, want the 4 vanilla ones", n)
	}
	if f.Town.DocPayload() != nil || f.Town.DocPayloadError() == "" {
		t.Fatalf("unfit payload kept as %v, diagnostic %q", f.Town.DocPayload() != nil, f.Town.DocPayloadError())
	}
	if cold := cityProjectionLoad(t, raw, newFront); cold.Town.DocPayload() != nil || len(cold.Town.Documents()) != 4 {
		t.Fatal("the unfit-payload SAVE did not load as the vanilla state")
	}
}
