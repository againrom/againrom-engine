package game

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func cardLoadRows(t *testing.T, f *FrontEnd, app *ui.App, id sim.EntityID) (ui.PanelSubject, []ui.PanelLineReport) {
	t.Helper()
	live := f.live
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	for _, e := range live.world.Entities() {
		if e.ID == id {
			inspectionCentre(live, int(e.X), int(e.Y))
		}
	}
	live.push()
	releaseHoverInspection(t, app, live, ui.InspectionSubject{Kind: ui.InspectionUnit, ID: uint32(id)})
	subject, ok := live.view.InspectionPanel()
	if !ok || subject.ID != uint32(id) {
		t.Fatalf("panel subject %d/%v, want %d", subject.ID, ok, id)
	}
	return subject, ui.CharacterPanelReport(ui.CompactPanelLayout(nil), f.tipFont(), subject)
}

func cardLoadWitness(t *testing.T, f *FrontEnd, app *ui.App, id sim.EntityID, name string, wantLoad, wantCaption bool) ui.PanelLineReport {
	t.Helper()
	subject, rows := cardLoadRows(t, f, app, id)
	load, caption := f.Words.PanelCaptions[35], f.Words.PanelCaptions[190]
	if load == "" || caption == "" {
		t.Fatalf("the install states no load or spellcaster caption: %q %q", load, caption)
	}
	size := ui.CompactPanelLayout(nil).Size
	var loadRow, captionRow *ui.PanelLineReport
	for i := range rows {
		switch {
		case rows[i].Label == load:
			loadRow = &rows[i]
		case rows[i].Label == caption || rows[i].RightLabel == caption:
			captionRow = &rows[i]
		}
	}
	if (loadRow != nil) != wantLoad {
		t.Fatalf("%s: load row present %v, want %v (subject %+v)", name, loadRow != nil, wantLoad, subject.OriginalPanel)
	}
	if wantLoad {
		if captionRow != nil {
			t.Fatalf("%s: a player character states the spellcaster caption: %+v", name, *captionRow)
		}
		cardLoadShot(t, app, name)
		return *loadRow
	}
	if wantCaption && captionRow == nil {
		t.Fatalf("%s: the spellcaster caption is absent", name)
	}
	if captionRow != nil {
		if captionRow.Right || captionRow.Label != caption || captionRow.At.X < 0 || captionRow.At.X+captionRow.Width > size.X {
			t.Fatalf("%s: caption row %+v is not whole inside the %d-wide card", name, *captionRow, size.X)
		}
	}
	cardLoadShot(t, app, name)
	if captionRow == nil {
		return ui.PanelLineReport{}
	}
	return *captionRow
}

func cardLoadShot(t *testing.T, app *ui.App, name string) {
	t.Helper()
	dir := os.Getenv("AGAINROM_CARD_LOAD_OUT")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var pic *image.RGBA
	pic, err := app.HeadlessMissionCard()
	if err != nil {
		pic, _, err = app.HeadlessCharacterPane()
		if err != nil {
			t.Fatal(err)
		}
	}
	out, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s-%s.png", filepath.Base(os.Getenv("AGAINROM_ASSETS")), name)))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := png.Encode(out, pic); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseInformationCardStatesLoadOnlyForTheHeroes(t *testing.T) {
	t.Run("mission 51 turtle", func(t *testing.T) {
		f, app, turtle := magicWitnessOpen(t)
		hero := cardLoadWitness(t, f, app, f.live.mission.ids[0], "hero", true, false)
		row := cardLoadWitness(t, f, app, turtle.ID, "turtle", false, true)
		if row.At.Y != hero.At.Y {
			t.Fatalf("the caption stands at y=%d, the load row at y=%d", row.At.Y, hero.At.Y)
		}
		plain := magicWitnessPlain(t, f)
		cardLoadWitness(t, f, app, plain, "plain-creature", false, false)
		for _, e := range f.live.world.Entities() {
			if e.Owner != sim.SelfSlot && e.Owner != 0 && e.TypeID < 0x1a && e.Alive() {
				cardLoadWitness(t, f, app, e.ID, fmt.Sprintf("enemy-person-%d", e.ID), false, sim.HumanExperienceValue(e.SkillXP) != 0)
				break
			}
		}
	})
	t.Run("mission 10 enemy person", func(t *testing.T) {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		app := f.App("card load")
		t.Cleanup(app.StopAudio)
		app.Layout(1024, 768)
		if err := app.OpenMission(f.MissionOpener(10)); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessKey("0"); err != nil {
			t.Fatal(err)
		}
		cardLoadWitness(t, f, app, f.live.mission.ids[0], "hero-m10", true, false)
		found := false
		for _, e := range f.live.world.Entities() {
			if e.Owner != sim.SelfSlot && e.Owner != 0 && e.TypeID < 0x1a && e.Alive() && sim.HumanExperienceValue(e.SkillXP) != 0 {
				cardLoadWitness(t, f, app, e.ID, "enemy-person", false, true)
				found = true
				break
			}
		}
		if !found {
			t.Fatal("mission 10 holds no enemy person with an experience value")
		}
	})
	t.Run("mission 61 enemy mage", func(t *testing.T) {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		app := f.App("card load")
		t.Cleanup(app.StopAudio)
		app.Layout(1024, 768)
		if err := app.OpenMission(f.MissionOpener(61)); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessKey("0"); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, e := range f.live.world.Entities() {
			if e.Owner != sim.SelfSlot && e.Owner != 0 && e.TypeID < 0x1a && e.Alive() && e.KnownSpells != 0 && e.MaxMana > 0 {
				cardLoadWitness(t, f, app, e.ID, "enemy-mage", false, true)
				found = true
				break
			}
		}
		if !found {
			t.Fatal("mission 61 holds no enemy person who knows a spell")
		}
	})
	for _, typ := range []int{3, 8} {
		t.Run(fmt.Sprintf("hired mercenary %d", typ), func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			target := releaseMercenaryChapter(t, f.Campaign.Value())
			town := NewTown(f.Campaign.Value())
			for mission := range f.Campaign.Value().Chapters {
				if mission < target {
					town.Won(mission)
				}
			}
			town.gold = 1_000_000
			f.Town = town
			f.Carried = f.NextParty()
			f.arriveInTown()
			f.Town.mercEnabled[typ] = true
			s := f.TownScreen().(*townScreen)
			s.room = roomTavern
			s.composeShopFaces()
			if _, ok := s.toggleMercenary(typ); !ok {
				t.Fatalf("hire type %d refused", typ)
			}
			app := f.App("card load")
			t.Cleanup(app.StopAudio)
			app.Layout(1024, 768)
			if err := app.OpenMission(f.MissionOpenerWith(target, f.Carried)); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessKey("0"); err != nil {
				t.Fatal(err)
			}
			index := -1
			for i, member := range f.live.mission.party {
				if member.MercenaryType == uint8(typ) {
					index = i
				}
			}
			if index < 0 || index >= len(f.live.mission.ids) {
				t.Fatalf("hired type %d is absent from the live party", typ)
			}
			cardLoadWitness(t, f, app, f.live.mission.ids[0], fmt.Sprintf("hero-with-hire-%d", typ), true, false)
			cardLoadWitness(t, f, app, f.live.mission.ids[index], fmt.Sprintf("hired-%d", typ), false, false)
		})
	}
}
