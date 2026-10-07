package game

import (
	"fmt"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// releaseUnitShadow centres the camera on the entity and returns the shadow
// the viewer draws for it.
func releaseUnitShadow(t *testing.T, live *mapWorld, e sim.Entity) ui.UnitShadow {
	t.Helper()
	inspectionCentre(live, int(e.X), int(e.Y))
	for _, s := range live.view.UnitShadows() {
		if sim.EntityID(s.ID) == e.ID {
			return s
		}
	}
	t.Fatalf("entity %d at (%d,%d) casts no shadow in view", e.ID, e.X, e.Y)
	return ui.UnitShadow{}
}

// releaseHireWithHuman hires the siege pair of releaseHireSiege and one Human
// mercenary of the same chapter's tavern, through the tavern's own controls.
func releaseHireWithHuman(t *testing.T, f *FrontEnd) {
	t.Helper()
	target := releaseSiegeChapter(t, f.Campaign.Value())
	c := f.Campaign.Value()
	town := NewTown(c)
	for mission := range c.Chapters {
		if mission < target {
			town.Won(mission)
		}
	}
	town.gold = 1_000_000
	f.Town = town
	f.Carried = f.NextParty()
	f.arriveInTown()
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodeSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err := store.Write(time.Unix(100, 0), payload); err != nil {
		t.Fatal(err)
	}
	app := f.App("unit shadows tavern")
	t.Cleanup(app.StopAudio)
	app.Layout(640, 480)
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
	for _, target := range []string{"load game", "@first", "TAVERN"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	unlocked := make(map[int]bool)
	for mission, ch := range c.Chapters {
		if mission < target {
			for _, typ := range ch.EnableMercenary {
				unlocked[typ] = true
			}
		}
	}
	humanHired := false
	for _, typ := range c.Chapters[target].Mercenaries {
		siege := typ <= 2
		human := !siege && !humanHired && unlocked[typ] && typ <= len(c.MercenaryCount) && c.MercenaryCount[typ-1] > 0
		if !siege && !human {
			continue
		}
		if err := app.HeadlessActivate(fmt.Sprintf("Mercenary %d", typ)); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessActivate(f.Words.TavernHire); err != nil {
			t.Fatal(err)
		}
		if !f.Town.MercenaryHired(typ) {
			t.Fatalf("mercenary %d is not hired", typ)
		}
		humanHired = humanHired || human
	}
	if !humanHired {
		t.Fatal("the chapter's tavern offers no Human mercenary")
	}
}

// TestReleaseUnitShadowsUseTheObjectPathLevel opens a mission on the installed
// art with the hero, a hired Catapult and a hired Ballista in the party. Each
// of the hero (a Human whose frame carries armour sheets), another Human and a
// siege Unit casts two silhouettes: the shear is the sun's, the first level is
// the sun's object-path shroud index, the same one a structure's main sheet
// reads, and the second is the unit-path index.
func TestReleaseUnitShadowsUseTheObjectPathLevel(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	releaseHireWithHuman(t, f)
	app := f.App("unit shadows")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(releaseHiredMission, f.Carried)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	for k := 0; k < 16 && live.mission.open; k++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	sun := live.view.Sun()
	if sun.ShroudObject == sun.ShroudUnit {
		t.Fatalf("the sun does not discriminate the two shroud indices: both %d", sun.ShroudObject)
	}
	slope := terrain.ShadowSlope(sun.Theta)
	live.view.SetGraphicsOptions(ui.GraphicsOptions{Smoothing: true})

	check := func(name string, e sim.Entity) {
		t.Helper()
		s := releaseUnitShadow(t, live, e)
		if s.Level != int(sun.ShroudObject) {
			t.Errorf("%s %d: shadow level %d, want the object-path index %d (unit-path index %d)", name, e.ID, s.Level, sun.ShroudObject, sun.ShroudUnit)
		}
		if s.Slope != slope {
			t.Errorf("%s %d: shadow slope %v, want the sun's %v", name, e.ID, s.Slope, slope)
		}
		if s.Frame == nil || s.Frame.Width <= 0 || s.Frame.Height <= 0 {
			t.Errorf("%s %d: shadow has no silhouette frame", name, e.ID)
		}
		t.Logf("%s %d: level %d slope %.4f frame %dx%d", name, e.ID, s.Level, s.Slope, s.Frame.Width, s.Frame.Height)
		// The second silhouette is the paired spritesb frame at the unit-path
		// level, drawn when Smoothing is on.
		n := 0
		for _, c := range live.view.UnitShadows() {
			if sim.EntityID(c.ID) != e.ID {
				continue
			}
			n++
			if n == 2 && (!c.Second || c.Level != int(sun.ShroudUnit) || c.Slope != slope) {
				t.Errorf("%s %d: second silhouette second=%v level %d slope %v, want the unit-path level %d", name, e.ID, c.Second, c.Level, c.Slope, sun.ShroudUnit)
			}
		}
		if n != 2 {
			t.Errorf("%s %d casts %d silhouettes, want 2", name, e.ID, n)
		}
	}

	hero, ok := live.entity(live.mission.ids[0])
	if !ok {
		t.Fatal("no hero")
	}
	check("hero", hero)
	for _, name := range []string{"Catapult", "Ballista"} {
		typeID := releaseSiegeTypeID(t, f, name)
		found := false
		for _, e := range live.world.Entities() {
			if e.Owner == sim.SelfSlot && e.TypeID == typeID && e.TokenSize > 1 {
				check(name, e)
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no hired %s in the mission", name)
		}
	}
	human := false
	for _, e := range live.world.Entities() {
		if e.ID == hero.ID || e.TokenSize != 1 || e.OffMap || !e.Alive() {
			continue
		}
		inspectionCentre(live, int(e.X), int(e.Y))
		for _, s := range live.view.UnitShadows() {
			if sim.EntityID(s.ID) == e.ID {
				check("human", e)
				human = true
				break
			}
		}
		if human {
			break
		}
	}
	if !human {
		t.Error("no second one-cell actor in view to witness a Human other than the hero")
	}
}
