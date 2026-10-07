package game

import (
	"bytes"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/fame"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

func TestReleaseCampaign1180EndingCreditsHallSaveAndReset(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	prepareAcceptedCampaignMission(t, f, 150)
	a := f.App("1180 ending")
	if err := a.OpenMission(f.MissionOpenerWith(150, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	campaignWin1176(t, f, a, 150)
	_, _, pending := campaignSave(t, f, true)
	campaignReturn(t, f, a)
	if f.live != nil || f.liveMission != 0 || !f.completedCampaign() {
		t.Fatal("completed world retained")
	}
	before, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	artifact := t.TempDir()
	if explicit := os.Getenv("AGAINROM_ENDING_ARTIFACTS"); explicit != "" {
		artifact = filepath.Join(explicit, filepath.Base(f.Archives.Root))
		if err := os.MkdirAll(artifact, 0755); err != nil {
			t.Fatal(err)
		}
	}
	store := SaveStore{Dir: filepath.Join(artifact, "ui-saves")}
	f.ConfigureSaveSeams(a, store, OriginalStore{}, nil)
	checkF2 := func(surface string) {
		t.Helper()
		if err := a.HeadlessKey("f2"); err != nil || a.Screen() != ui.ScreenEnding {
			t.Fatal(surface, "F2 did not return or remain at ending", err, a.Screen())
		}
	}
	capture := func(name string) {
		pix, note, err := a.HeadlessFrame()
		if err != nil || note != "" {
			t.Fatal("ending composite", err, note)
		}
		file, err := os.Create(filepath.Join(artifact, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(file, pix)
		closeErr := file.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
	}
	if a.Screen() != ui.ScreenCredits {
		t.Fatal("terminal Victory did not open the installed credits roll", a.Screen())
	}
	capture("credits-first")
	raw, err := f.Archives.Containers.ReadFile(mainPrefix + "text/credits.txt")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(f.endingAssets.Credits, "\n") != strings.ReplaceAll(strings.TrimSpace(vfs.DecodeText(raw)), "\r\n", "\n") {
		t.Fatal("credits replaced source text")
	}
	for i := 0; i < 720; i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	capture("credits-scrolling")
	checkF2("credits")
	capture("hall")
	checkF2("hall")
	installed, err := os.ReadFile(filepath.Join(f.Archives.Root, "famehall.dat"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := fame.Parse(installed)
	hallView, complete := f.campaignEnding()
	if err != nil || !complete || len(parsed) != 10 || !hallView.HallAvailable || len(hallView.Hall) != len(parsed) {
		t.Fatal("installed table", err, parsed)
	}
	if f.endingAssets.HallBackground == nil {
		t.Fatal("installed hall background missing")
	}
	if entries, err := os.ReadDir(store.Dir); !os.IsNotExist(err) && (err != nil || len(entries) != 0) {
		t.Fatal("ending UI wrote a save", entries, err)
	}
	t.Run("completed state has no save point", func(t *testing.T) {
		if _, err := f.ExportCurrentSave(before, "Completed campaign"); err == nil {
			t.Fatal("the completed campaign was written as a SAV")
		}
		hold := f.Town
		if candidate, err := f.prepareRestore(before); err == nil || candidate != nil || !errors.Is(err, errCompletedCampaignFile) {
			t.Fatal("a completed campaign state was accepted by LOAD", err)
		}
		if f.Town != hold || !f.completedCampaign() {
			t.Fatal("the refused LOAD changed the live campaign")
		}
	})
	if err := os.WriteFile(filepath.Join(artifact, "pending.sav"), pending, 0644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if f.Town.Gold() != before.Gold {
		t.Fatal("ending paid again")
	}
	if err := a.HeadlessActivate(a.HeadlessRows()[0].Text); err != nil || a.Screen() != ui.ScreenMenu {
		t.Fatal("hall button", err, a.Screen())
	}
	if f.completedCampaign() || f.Town.Done(150) {
		t.Fatal("the hall button did not reset the campaign")
	}
	setupNew := func(front *FrontEnd, app *ui.App) {
		app.SetNewGameChargen(func() *ui.ChargenEntry {
			return &ui.ChargenEntry{Model: ui.NewChargen(front.ChargenSetup()), Begin: func(result ui.ChargenResult) (ui.MapOpener, error) { return front.NewGameOpener(10, result), nil }}
		})
	}
	setupNew(f, a)
	if err := a.HeadlessActivate("new game"); err != nil || a.Screen() != ui.ScreenChargen {
		t.Fatal("new game", err, a.Screen())
	}
	if err := headlessCreateCharacter(a, HeadlessCharacter{Name: "Fresh", Sex: "Male", Class: "Fighter", Skill: "Blade"}); err != nil {
		t.Fatal(err)
	}
	fresh := releaseFront(t)
	fresh.SetDeterministicFrames(true)
	freshApp := fresh.App("1180 fresh baseline")
	setupNew(fresh, freshApp)
	if err := freshApp.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if err := headlessCreateCharacter(freshApp, HeadlessCharacter{Name: "Fresh", Sex: "Male", Class: "Fighter", Skill: "Blade"}); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenMap || f.liveMission != 10 || f.completedCampaign() || f.Town.Done(150) || f.Town.Gold() != fresh.Town.Gold() {
		t.Fatal("new game retained completed campaign", a.Screen(), f.liveMission, f.Town.Gold())
	}
	if view, complete := f.campaignEnding(); complete || view.Hero != "" {
		t.Fatal("stale ending survived reset")
	}
	unchanged, err := os.ReadFile(filepath.Join(f.Archives.Root, "famehall.dat"))
	if err != nil || !bytes.Equal(installed, unchanged) {
		t.Fatal("read-only hall changed", err)
	}
	t.Logf("actual mission150 script Victory; installed credits=%d lines, hall=%d rows; ending, credits and hall expose no SAVE; completed state refused by SAVE and LOAD, adopted new game passes; artifacts=%s", len(f.endingAssets.Credits), len(parsed), artifact)
}
