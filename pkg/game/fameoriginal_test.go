package game

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// The existing synthetic original body and campaign-tail encoder are separate
// inputs. This helper joins them without using the production campaign writer.
func fameOriginalCampaignTail(t *testing.T, body []byte, campaign sav.CampaignProjection) []byte {
	t.Helper()
	tail := int(binary.LittleEndian.Uint32(body[4:])) + 0x100
	if tail > len(body) {
		t.Fatal("synthetic original container tail is out of bounds")
	}
	out := append([]byte(nil), body[:tail]...)
	out = append(out, synth.Reg(0, nil)...)
	return append(out, encodeOriginalCampaignProjection(campaign)...)
}

func assertFameOriginalCounters(t *testing.T, raw []byte, time, events uint32) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	campaign, present, err := file.Campaign()
	if err != nil || !present {
		t.Fatalf("campaign present=%v err=%v", present, err)
	}
	if campaign.MissionTime != time || campaign.ScoreEvents != events || !campaign.ScoreEventsKnown {
		t.Fatalf("SAV counter dwords=%08x/%08x known=%v, want %08x/%08x", campaign.MissionTime,
			campaign.ScoreEvents, campaign.ScoreEventsKnown, time, events)
	}
}

func TestFameOriginalCityCountersBecomeKnownAndSurviveAGS(t *testing.T) {
	campaign, projection := restoredCampaignFixture()
	// This fixture exercises raw campaign counters with no definition table;
	// reconstruction of a missing hired actor is a separate surface.
	projection.MercenaryHired = make([]bool, 15)
	projection.MissionTime, projection.ScoreEvents = 0xfffffff1, 0x80000007
	raw := originalSaveWithCampaign(t, 0, projection)
	assertFameOriginalCounters(t, raw, 0xfffffff1, 0x80000007)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(campaign, nil)}, CampaignSession: CampaignSession{Town: NewTown(campaign), fame: SnapshotFame{Known: true, Time: 18, Events: 4, Result: &FameResult{Name: "Prior game", Score: 52, Recorded: true}}}}
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || !town || open != nil {
		t.Fatalf("original city restore: opener=%v town=%v err=%v", open != nil, town, err)
	}
	want := SnapshotFame{Known: true, Time: 0xfffffff1, Events: 0x80000007}
	if !reflect.DeepEqual(f.fame, want) {
		t.Fatalf("city fame=%+v, want %+v", f.fame, want)
	}
	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	native, err := EncodeSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(native)
	if err != nil {
		t.Fatal(err)
	}
	cold := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(campaign, nil)}}
	if _, town, err := cold.Restore(decoded); err != nil || !town {
		t.Fatalf("AGS city restore: town=%v err=%v", town, err)
	}
	if !reflect.DeepEqual(cold.fame, want) {
		t.Fatalf("AGS lost raw signed counter bits: %+v", cold.fame)
	}
}

func TestFameOriginalWorldCountersCommitOnlyOnAdoption(t *testing.T) {
	f := missionFrontEnd(t)
	f.Campaign = resolved(Campaign{Main: []int{10, 20}, Chapters: map[int]Chapter{10: {Mission: 10}, 20: {Mission: 20}}}, nil)
	f.Town = NewTown(f.Campaign.Value())
	f.fame = SnapshotFame{Known: true, Time: 70, Events: 9, Result: &FameResult{Name: "Prior", Score: 31}}
	before := cloneFame(f.fame)
	previousTown := f.Town
	projection := campaignProjectionAt(f.Campaign.Value(), 10)
	projection.MissionTime, projection.ScoreEvents = 0x80000010, 0xfffffffe
	raw := fameOriginalCampaignTail(t, savedFileWithSession(t, 10, nil, -1, 0, -1, 0, 0), projection)
	assertFameOriginalCounters(t, raw, 0x80000010, 0xfffffffe)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town || open == nil {
		t.Fatalf("original world prepare: opener=%v town=%v err=%v", open != nil, town, err)
	}
	if !reflect.DeepEqual(f.fame, before) || f.Town != previousTown {
		t.Fatal("preparing original world replaced active campaign history")
	}
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		t.Fatal(err)
	}
	want := SnapshotFame{Known: true, Time: 0x80000010, Events: 0xfffffffe}
	if !reflect.DeepEqual(f.fame, want) || f.live == nil || f.liveMission != 10 {
		t.Fatalf("original world adoption fame=%+v live=%v mission=%d", f.fame, f.live != nil, f.liveMission)
	}
	f.live.observeFame()
	if !reflect.DeepEqual(f.fame, want) {
		t.Fatal("cold world baseline changed imported counters")
	}
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Fame == nil || !reflect.DeepEqual(*snapshot.Fame, want) ||
		snapshot.Campaign.MissionTime != want.Time || snapshot.Campaign.ScoreEvents != want.Events {
		t.Fatalf("world snapshot lost counter source: %+v", snapshot.Fame)
	}
}

func TestFameOriginalFailedRestorePreservesActiveHistory(t *testing.T) {
	t.Run("missing mission map", func(t *testing.T) {
		f := missionFrontEnd(t)
		f.Campaign = resolved(Campaign{Main: []int{10, 20}, Chapters: map[int]Chapter{10: {Mission: 10}, 20: {Mission: 20}}}, nil)
		f.Town = NewTown(f.Campaign.Value())
		f.fame = SnapshotFame{Known: true, Time: 123, Events: 789, Result: &FameResult{Name: "Active", Score: 45}}
		before, oldTown := cloneFame(f.fame), f.Town
		projection := campaignProjectionAt(f.Campaign.Value(), 20)
		projection.MissionTime, projection.ScoreEvents = 900, 800
		raw := fameOriginalCampaignTail(t, savedFile(20, nil), projection)
		open, town, err := f.RestoreOriginal(raw)
		if err == nil || !strings.Contains(err.Error(), "scenario/20.alm") || open != nil || town {
			t.Fatalf("missing map refusal: opener=%v town=%v err=%v", open != nil, town, err)
		}
		if !reflect.DeepEqual(f.fame, before) || f.Town != oldTown {
			t.Fatal("late map refusal replaced active fame or town")
		}
	})
	t.Run("city hired population cap", func(t *testing.T) {
		campaign, projection := restoredCampaignFixture()
		projection.MissionTime, projection.ScoreEvents = 900, 800
		projection.MercenaryWorking[2], projection.MercenaryHired[2] = 13, true
		f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(campaign, nil)}, CampaignSession: CampaignSession{Town: NewTown(campaign), fame: SnapshotFame{Known: true, Time: 123, Events: 789, Result: &FameResult{Name: "Active", Score: 45}}}}
		before, oldTown := cloneFame(f.fame), f.Town
		open, town, err := f.RestoreOriginal(originalSaveWithCampaign(t, 0, projection))
		if err == nil || !strings.Contains(err.Error(), "roster cap") || open != nil || town {
			t.Fatalf("hired cap refusal: opener=%v town=%v err=%v", open != nil, town, err)
		}
		if !reflect.DeepEqual(f.fame, before) || f.Town != oldTown {
			t.Fatal("late city refusal replaced active fame or town")
		}
	})
}

func TestFameOriginalExportWritesUnknownHistoryInstead(t *testing.T) {
	for _, state := range []*SnapshotFame{nil, {Time: 17, Events: 2}} {
		f, _ := originalCityRouteFixture(t)
		snapshot, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		snapshot.Fame = state
		snapshot.Campaign.ScoreEventsKnown = false
		snapshot.Campaign.MissionTime, snapshot.Campaign.ScoreEvents = 999, 888
		raw, err := f.ExportOriginalSave(snapshot, label)
		if err != nil {
			t.Fatalf("imported export of unknown fame=%+v refused: %v", state, err)
		}
		wantTime, wantEvents := uint32(999), uint32(888)
		if state != nil {
			wantTime, wantEvents = state.Time, state.Events
		}
		assertFameOriginalCounters(t, raw, wantTime, wantEvents)
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err := readCurrentActions(&doc)
		if err != nil || a == nil || a.Session == nil || a.Session.Fame == nil || a.Session.Fame.Known {
			t.Fatal("current counter projection invented complete history", err)
		}
		f.originalCity = nil
		f.Town.progress = nil
		if _, err := f.ExportNativeCitySave(snapshot, label); err != nil && strings.Contains(err.Error(), "score history") {
			t.Fatalf("source-free export of unknown fame=%+v still refuses on score history: %v", state, err)
		}
	}
}

func TestFameOriginalUnknownHistoryNoLongerFallsBackToAGS(t *testing.T) {
	f, _ := originalCityRouteFixture(t)
	f.fame = SnapshotFame{Time: 17, Events: 2}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatal(err)
	}
	if !IsOriginal(name) {
		t.Fatalf("unknown history SAVE=%q, want SAV", name)
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	assertFameOriginalCounters(t, raw, 17, 2)
}

func TestReleaseFame1181OriginalCityCounterExportRoundTrip(t *testing.T) {
	f := releaseFront(t)
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Counter Hero", Choices: []int{1, 1, 3}, Stats: []int{31, 27, 24, 29}})
	f.arriveInTown()
	f.Town.mercEnabled[14] = true
	f.addChapterCompanions(f.Town.Chapter())
	f.fame = SnapshotFame{Known: true, Time: 0xfffffff1, Events: 0x80000007}
	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportNativeCitySave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	assertFameOriginalCounters(t, raw, 0xfffffff1, 0x80000007)
	cold := releaseFront(t)
	if open, town, err := cold.RestoreOriginal(raw); err != nil || !town || open != nil {
		t.Fatalf("generated original city load: opener=%v town=%v err=%v", open != nil, town, err)
	}
	if !reflect.DeepEqual(cold.fame, f.fame) {
		t.Fatalf("generated SAV counter baseline=%+v, want %+v", cold.fame, f.fame)
	}
	// Imported provenance exports current counters, including wrapping bits,
	// rather than retaining the prior source-free document's tail values.
	cold.fame.Time, cold.fame.Events = 23, 0xffffffff
	nextSnapshot, nextLabel, err := cold.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	next, err := cold.ExportOriginalSave(nextSnapshot, nextLabel)
	if err != nil {
		t.Fatal(err)
	}
	assertFameOriginalCounters(t, next, 23, 0xffffffff)
	if open, town, err := cold.RestoreOriginal(next); err != nil || !town || open != nil {
		t.Fatalf("updated original city load: opener=%v town=%v err=%v", open != nil, town, err)
	}
	if !reflect.DeepEqual(cold.fame, SnapshotFame{Known: true, Time: 23, Events: 0xffffffff}) {
		t.Fatalf("updated SAV lost current counters: %+v", cold.fame)
	}
	artifact := t.TempDir()
	if base := os.Getenv("AGAINROM_FAME_ARTIFACTS"); base != "" {
		artifact = filepath.Join(base, filepath.Base(f.Archives.Root))
		if err := os.MkdirAll(artifact, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for name, data := range map[string][]byte{"original-city-counters.sav": raw, "original-city-updated-counters.sav": next} {
		if err := os.WriteFile(filepath.Join(artifact, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	receipt, err := json.MarshalIndent(struct{ Initial, Updated SnapshotFame }{f.fame, cold.fame}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "original-city-counters.json"), receipt, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("source-free SAV and imported SAV keep current score counter dwords; artifacts=%s", artifact)
}
