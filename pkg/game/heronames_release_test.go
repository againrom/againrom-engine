package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// The default hero on each installed root: the party a mission opened without
// the generator's name field starts with. He carries the installed name of his
// picture, the same bytes the pre-create field opens with (TEXT-073,
// TEXT-074): the male fighter's, or the male mage's for the mage arm. On the RU
// install those are code page 866 bytes, not the Latin name the party carried
// before. No menu route starts him: NEW GAME, the debug map picker's mission
// rows and the world map all open the generator or a carried party, so the
// witnesses are the mission door the mission tools use, named as such.
func TestReleaseDefaultHeroCarriesHisPicturesInstalledName(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	names, _ := preCreateNames(t, f)
	if f.Table.HeroNames != names {
		t.Errorf("the table holds the hero names % x, the pre-create page opens with % x", f.Table.HeroNames, names)
	}

	// The two arms of the default hero and the party a front end starts with.
	weapon := f.StartWeapon.Value()
	if got := MissionParty(weapon, f.Bodies, f.Table)[0].Name; got != names[0] {
		t.Errorf("MissionParty names the hero % x, want the male fighter's % x", got, names[0])
	}
	if got := MissionPartyAs(true, weapon, f.Bodies, f.Table)[0].Name; got != names[1] {
		t.Errorf("MissionPartyAs(true) names the hero % x, want the male mage's % x", got, names[1])
	}
	if got := f.NextParty()[0].Name; got != names[0] {
		t.Errorf("NextParty names the hero % x, want the male fighter's % x", got, names[0])
	}

	// The hero is the same person in the information window, in an F2 SAV and
	// after a cold LOAD of that SAV.
	window := func(front *FrontEnd, a *ui.App, want, what string) {
		t.Helper()
		id := uint32(front.live.mission.ids[0])
		if s, ok := front.live.view.InspectionPanel(); !ok || s.ID != id {
			if err := a.HeadlessSelectEntity(id); err != nil {
				t.Fatalf("%s: %v", what, err)
			}
		}
		s, ok := front.live.view.InspectionPanel()
		if !ok || s.Kind != ui.InspectionUnit || s.ID != id || s.Name != want {
			t.Fatalf("%s: the information window shows unit %d named % x, want the hero %d named % x", what, s.ID, s.Name, id, want)
		}
	}
	saved := func(front *FrontEnd, a *ui.App, store SaveStore, label, want string) {
		t.Helper()
		missionAutosaveRow(t, front, store, front.liveMission)
		if _, _, up := front.LiveNotice(); up {
			if err := a.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		}
		raw := cityRosterF2Save(t, a, store, label)
		if got := fallenHeroWinSavedHero(t, raw).Name; got != want {
			t.Fatalf("the F2 SAV names the hero % x, want % x", got, want)
		}
		cold, coldApp := fallenHeroWinColdLoad(t, store, label+".sav")
		if cold.liveMission != front.liveMission || len(cold.liveParty) == 0 || cold.liveParty[0].Name != want {
			t.Fatalf("cold LOAD: mission %d (want %d), party %d, hero named % x, want % x",
				cold.liveMission, front.liveMission, len(cold.liveParty), cold.liveParty[0].Name, want)
		}
		window(cold, coldApp, want, "cold LOAD")
	}

	// The debug map picker's mission row opens the generator, so the default
	// hero is not what NEW GAME under -picker starts either.
	t.Run("picker row opens the generator", func(t *testing.T) {
		front := releaseFront(t)
		front.Options = OptionsStore{}
		app := front.App("default hero picker")
		app.Layout(640, 480)
		if err := app.HeadlessActivate("new game"); err != nil || app.Screen() != ui.ScreenPicker {
			t.Fatalf("NEW GAME with no generator armed: screen %s, error %v; want the map picker", app.Screen(), err)
		}
		if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenChargen || front.live != nil {
			t.Fatalf("the picker's first row: screen %s, error %v, mission running %v; want the generator", app.Screen(), err, front.live != nil)
		}
	})

	// The mission door with no generator in front of it, which is what the
	// mission tools and a front end assembled without the gate open: the party
	// is NextParty's default hero, or MissionPartyAs's own mage.
	for _, arm := range []struct {
		name  string
		party func(*FrontEnd) []mapload.PartyMember
		want  string
	}{
		{"tool route, fighter", func(front *FrontEnd) []mapload.PartyMember { return front.NextParty() }, names[0]},
		{"tool route, mage arm", func(front *FrontEnd) []mapload.PartyMember {
			return MissionPartyAs(true, front.StartWeapon.Value(), front.Bodies, front.Table)
		}, names[1]},
	} {
		t.Run(arm.name, func(t *testing.T) {
			front := releaseFront(t)
			front.Options = OptionsStore{}
			front.SetDeterministicFrames(true)
			store := SaveStore{Dir: t.TempDir()}
			app := front.App("default hero " + arm.name)
			app.Layout(640, 480)
			front.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
			if err := app.OpenMission(front.MissionOpenerWith(10, arm.party(front))); err != nil || front.live == nil || len(front.liveParty) == 0 {
				t.Fatalf("mission 10 with the default party: %v", err)
			}
			if got := front.liveParty[0].Name; got != arm.want {
				t.Fatalf("mission 10 opened with the hero named % x, want % x", got, arm.want)
			}
			window(front, app, arm.want, "start")
			saved(front, app, store, "default", arm.want)
		})
	}
}
