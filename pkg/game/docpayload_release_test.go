package game

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
)

const docPayloadKitLabel = "9503 docs ARSV payload"

// TestReleaseDocPayloadMissionSAVRoundTrip loads an original mission SAV with
// three vanilla documents, gives the session a payload, and checks that the
// production SAVE writes it after the vanilla pairs, that a cold LOAD returns
// it, and that no document the panel lists changed. With
// AGAINROM_DOCS_CARRIER_OUT set it also writes the owner kit candidate there.
func TestReleaseDocPayloadMissionSAVRoundTrip(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-10-06/game0001-from-9281-original-resave.sav",
		"c75b8b19c12de96372243d2076aef975f383b704ce7a75e8e016d4ca635b99b5")
	vanilla := []sav.CampaignDocument{{Value: 1, Kind: 1}, {Value: 2, Kind: 1}, {Value: 3, Kind: 1}}
	if base := savCampaign(t, raw); !reflect.DeepEqual(base.Documents, vanilla) || base.CarrierRecords != 0 {
		t.Fatalf("base SAV documents = %v with %d carriers", base.Documents, base.CarrierRecords)
	}
	payload := &sav.DocPayload{Version: sav.PayloadVersion, Data: []byte("Againrom payload round 3: 9503 carrier test.\n")}

	f := releaseFront(t)
	open, _, err := f.RestoreOriginal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.App("document payload SAV").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	wantDocs := f.Town.Documents()
	if len(wantDocs) != 3 || f.Town.DocPayload() != nil {
		t.Fatalf("loaded town holds %v and payload %v", wantDocs, f.Town.DocPayload())
	}
	f.Town.SetDocPayload(payload)
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	written, err := f.ExportCurrentSave(snapshot, docPayloadKitLabel)
	if err != nil {
		t.Fatal(err)
	}
	campaign := savCampaign(t, written)
	if !reflect.DeepEqual(campaign.Documents, vanilla) || !reflect.DeepEqual(campaign.Payload, payload) || campaign.PayloadError != "" {
		t.Fatalf("written SAV: documents %v payload %v (%q)", campaign.Documents, campaign.Payload, campaign.PayloadError)
	}
	chunks, err := sav.EncodeDocPayload(payload)
	if err != nil || campaign.CarrierRecords != len(chunks) || docPairCountAfter(t, written, vanilla) != 3+uint32(len(chunks)) {
		t.Fatalf("written SAV holds %d carrier records, want %d (%v)", campaign.CarrierRecords, len(chunks), err)
	}

	cold := releaseFront(t)
	open, _, err = cold.RestoreOriginal(written)
	if err != nil {
		t.Fatal(err)
	}
	if err := cold.App("document payload cold LOAD").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if got := cold.Town.DocPayload(); !reflect.DeepEqual(got, payload) || !reflect.DeepEqual(cold.Town.Documents(), wantDocs) {
		t.Fatalf("cold LOAD holds payload %v and documents %v", got, cold.Town.Documents())
	}
	// A second SAVE of the loaded session keeps one block.
	snapshot, _, err = cold.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	second, err := cold.ExportCurrentSave(snapshot, docPayloadKitLabel)
	if err != nil {
		t.Fatal(err)
	}
	if again := savCampaign(t, second); again.CarrierRecords != len(chunks) || !reflect.DeepEqual(again.Payload, payload) {
		t.Fatalf("second SAVE holds %d carrier records and payload %v", again.CarrierRecords, again.Payload)
	}

	if dir := os.Getenv("AGAINROM_DOCS_CARRIER_OUT"); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for name, data := range map[string][]byte{"game9503.sav": written, "game9503.expected-payload.bin": payload.Data} {
			if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// docPairCountAfter returns the document count word of raw, found by the
// vanilla pairs that start the run.
func docPairCountAfter(t *testing.T, raw []byte, vanilla []sav.CampaignDocument) uint32 {
	t.Helper()
	var run []byte
	for _, d := range vanilla {
		run = append(run, byte(d.Value), 0, 0, 0, byte(d.Kind), 0, 0, 0)
	}
	at := bytes.Index(raw, run)
	if at < 4 {
		t.Fatal("document run is absent from the SAV")
	}
	return uint32(raw[at-4]) | uint32(raw[at-3])<<8 | uint32(raw[at-2])<<16 | uint32(raw[at-1])<<24
}
