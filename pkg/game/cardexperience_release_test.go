package game

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// cardExperienceWitness hovers one actor's mission card and requires the XP
// line exactly when the actor is in the player-character band, which is also
// the population the award pays (GainsXP; HERO-SKILLGATE-074).
func cardExperienceWitness(t *testing.T, f *FrontEnd, app *ui.App, e sim.Entity, name string, want bool) {
	t.Helper()
	if band := sim.InPersistBand(e.TypeID); band != want || e.GainsXP != want {
		t.Fatalf("%s: type 0x%x band %v gains %v, want %v", name, e.TypeID, band, e.GainsXP, want)
	}
	subject, rows := cardLoadRows(t, f, app, e.ID)
	label := f.Words.PanelCaptions[46]
	if label == "" {
		t.Fatal("the install states no experience caption")
	}
	got := false
	for _, row := range rows {
		got = got || row.Label == label
	}
	if got != want || subject.Char.ExperienceWithheld == want {
		t.Fatalf("%s: XP line %v withheld %v, want line %v", name, got, subject.Char.ExperienceWithheld, want)
	}
	dir := os.Getenv("AGAINROM_NPC_XP_CARD_OUT")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	pic, err := app.HeadlessMissionCard()
	if err != nil {
		t.Fatal(err)
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

// The XP line stands on a hero's card and on Brian's before he joins, and is
// absent for an enemy person, a creature and a hired mercenary (DIV-3613).
func TestReleaseInformationCardStatesExperienceOnlyForHeroes(t *testing.T) {
	t.Run("mission 40", func(t *testing.T) {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		app := f.App("card experience")
		t.Cleanup(app.StopAudio)
		app.Layout(1024, 768)
		if err := app.OpenMission(f.MissionOpener(40)); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"0", "shift-f4"} {
			if err := app.HeadlessKey(key); err != nil {
				t.Fatal(err)
			}
		}
		live := f.live
		var hero, brian, person, creature *sim.Entity
		for _, e := range live.world.Entities() {
			e := e
			switch {
			case e.ID == live.mission.ids[0]:
				hero = &e
			case live.missionPartyMember(e.ID) != nil || !e.Alive():
			case sim.InPersistBand(e.TypeID) && e.Owner != sim.SelfSlot:
				brian = &e
			case e.Humanoid && e.Owner != sim.SelfSlot && e.Owner != 0 && person == nil:
				person = &e
			case !e.Humanoid && e.Owner != sim.SelfSlot && creature == nil:
				creature = &e
			}
		}
		if hero == nil || brian == nil || person == nil || creature == nil {
			t.Fatalf("mission 40 lacks a case: hero %v brian %v person %v creature %v", hero != nil, brian != nil, person != nil, creature != nil)
		}
		if brian.TypeID != 0x21 || brian.Owner != 2 {
			t.Fatalf("npc25 stands as type 0x%x owner %d, want 0x21 owner 2 (HERO-JOIN-121)", brian.TypeID, brian.Owner)
		}
		cardExperienceWitness(t, f, app, *hero, "m40-hero", true)
		cardExperienceWitness(t, f, app, *brian, "m40-brian-before-joining", true)
		cardExperienceWitness(t, f, app, *person, "m40-enemy-person", false)
		cardExperienceWitness(t, f, app, *creature, "m40-creature", false)
	})
	t.Run("hired mercenary", func(t *testing.T) {
		const typ = 3
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
		app := f.App("card experience")
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
			if member.MercenaryType == typ {
				index = i
			}
		}
		if index < 0 || index >= len(f.live.mission.ids) {
			t.Fatalf("hired type %d is absent from the live party", typ)
		}
		byID := map[sim.EntityID]sim.Entity{}
		for _, e := range f.live.world.Entities() {
			byID[e.ID] = e
		}
		cardExperienceWitness(t, f, app, byID[f.live.mission.ids[0]], "hero-with-hire", true)
		cardExperienceWitness(t, f, app, byID[f.live.mission.ids[index]], "hired-mercenary", false)
	})
}
