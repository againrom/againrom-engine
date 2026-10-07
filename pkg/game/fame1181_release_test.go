package game

import (
	"bytes"
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/fame"
	"againrom/pkg/ui"
)

func TestReleaseFame1181TerminalResultColdHallAndSAV(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	prepareAcceptedCampaignMission(t, f, 150)
	// Controlled prehistory; the installed final objective and completion route
	// supply the measured last mission. This is not a whole campaign playthrough.
	f.fame = SnapshotFame{Known: true, Time: 100, Events: 4000}
	artifact := t.TempDir()
	if base := os.Getenv("AGAINROM_FAME_ARTIFACTS"); base != "" {
		artifact = filepath.Join(base, filepath.Base(f.Archives.Root))
		if err := os.MkdirAll(artifact, 0755); err != nil {
			t.Fatal(err)
		}
	}
	store := SaveStore{Dir: filepath.Join(artifact, "saves")}
	a := f.App("1181 earned result")
	f.ConfigureSaveSeams(a, store, OriginalStore{}, nil)
	installedPath := filepath.Join(f.Archives.Root, fameFileName)
	installed, err := os.ReadFile(installedPath)
	if err != nil {
		t.Fatal(err)
	}
	// A local empty table admits any signed result and controls the row oracle.
	if err := os.WriteFile(filepath.Join(artifact, fameFileName), []byte{0, 0, 0, 0}, 0600); err != nil {
		t.Fatal(err)
	}
	if err := a.OpenMission(f.MissionOpenerWith(150, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	missionAutosaveRow(t, f, store, 150)
	if f.live.world.Tick() != 0 {
		t.Fatal("accepted final mission did not start at tick zero")
	}
	campaignWin1176(t, f, a, 150)
	pending, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	pendingRaw, err := EncodeSave(pending, "Pending earned result")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "pending.ags"), pendingRaw, 0600); err != nil {
		t.Fatal(err)
	}
	time := uint32(100) + uint32(int32(uint32(f.live.world.Tick()))/16)
	events := f.fame.Events
	var xp int32
	for _, e := range f.live.world.Entities() {
		if e.ID == f.live.mission.ids[0] {
			for _, v := range e.SkillXP {
				xp += v
			}
		}
	}
	wantScore := int32(int64((float64(xp) / (float64(int32(time)) * 10.0)) * float64(int32(events))))
	saveFiles := func() map[string][]byte {
		t.Helper()
		entries, err := os.ReadDir(store.Dir)
		if err != nil {
			t.Fatal("read mission saves", err)
		}
		files := make(map[string][]byte, len(entries))
		for _, entry := range entries {
			raw, err := os.ReadFile(filepath.Join(store.Dir, entry.Name()))
			if err != nil {
				t.Fatal("read mission save", entry.Name(), err)
			}
			files[entry.Name()] = raw
		}
		return files
	}
	beforeEnding := saveFiles()
	unchangedSaves := func() {
		t.Helper()
		after := saveFiles()
		if len(after) != len(beforeEnding) {
			t.Fatalf("terminal route changed save names: before %d, after %d", len(beforeEnding), len(after))
		}
		for name, raw := range beforeEnding {
			if saved, ok := after[name]; !ok || !bytes.Equal(raw, saved) {
				t.Fatal("terminal route changed an existing save", name)
			}
		}
	}
	campaignReturn(t, f, a)
	unchangedSaves()
	if f.fame.Result == nil || f.fame.Time != time || f.fame.Result.Score != wantScore || !f.fame.Result.Recorded {
		t.Fatalf("earned result=%+v, time=%d events=%d xp=%d want=%d", f.fame, time, events, xp, wantScore)
	}
	readHall := func() []fame.Record {
		rows, _, err := readFameFile(filepath.Join(artifact, fameFileName))
		if err != nil {
			t.Fatal(err)
		}
		return rows
	}
	if rows := readHall(); len(rows) != 1 || rows[0].Score != wantScore || rows[0].Name != f.fame.Result.Name {
		t.Fatal("stored row", rows)
	}
	capture := func(name string) {
		pix, note, err := a.HeadlessFrame()
		if err != nil || note != "" {
			t.Fatal(err, note)
		}
		out, err := os.Create(filepath.Join(artifact, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(out, pix)
		closeErr := out.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
	}
	if a.Screen() != ui.ScreenCredits {
		t.Fatal("the ending did not start with the credits roll", a.Screen())
	}
	capture("earned")
	if err := a.HeadlessKey("enter"); err != nil || a.Screen() != ui.ScreenEnding {
		t.Fatal("a key did not end the roll into the hall", err, a.Screen())
	}
	unchangedSaves()
	capture("earned-hall")
	if len(readHall()) != 1 {
		t.Fatal("the ending added a record twice")
	}
	if rows := a.HeadlessRows(); len(rows) != 1 {
		t.Fatal("the hall has one button", rows)
	}
	result := *f.fame.Result
	if err := a.HeadlessKey("f2"); err != nil || a.Screen() != ui.ScreenEnding {
		t.Fatal("ending exposed a SAVE point", err, a.Screen())
	}
	unchangedSaves()
	completed, _, err := f.Snapshot(false)
	if err != nil || len(completed.World) != 0 {
		t.Fatal("completed result snapshot", err, len(completed.World))
	}
	if _, err := f.ExportCurrentSave(completed, "Earned result"); err == nil {
		t.Fatal("the completed campaign was written as a SAV")
	}
	if err := a.HeadlessActivate(a.HeadlessRows()[0].Text); err != nil || a.Screen() != ui.ScreenMenu {
		t.Fatal("the hall button did not return to the main menu", err, a.Screen())
	}
	unchangedSaves()
	if f.fame.Result != nil || f.fame.Known || f.completedCampaign() {
		t.Fatal("the terminal route did not reset the campaign", f.fame)
	}
	if rows, err := f.hallStore.read(); err != nil || len(rows) != 1 || rows[0].Score != wantScore || rows[0].Name != result.Name || !result.Recorded {
		t.Fatal("hall changed the stored row", rows, err)
	}
	if after, err := os.ReadFile(installedPath); err != nil || !bytes.Equal(installed, after) {
		t.Fatal("installed hall changed", err)
	}
	receipt, _ := json.MarshalIndent(struct {
		Time, Events uint32
		XP, Score    int32
		Name         string
	}{time, events, xp, wantScore, result.Name}, "", "  ")
	if err := os.WriteFile(filepath.Join(artifact, "earned-result.json"), receipt, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("installed mission150 Victory, time=%d events=%d xp=%d score=%d; no ending SAVE, durable one-row hall and a completed state no SAVE or LOAD accepts; artifacts=%s", time, events, xp, wantScore, artifact)
}
