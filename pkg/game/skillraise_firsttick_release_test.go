package game

import (
	"strconv"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// fighterRun opens mission 20 with one generated Blade fighter and gives the
// attack orders raiseSkillByOrders gives, stopping once the world reaches
// stopAt or a notice row is posted. It returns the front, its application,
// the hero and its target ids, and the rows.
func fighterRun(t *testing.T, stopAt uint64) (*FrontEnd, *ui.App, sim.EntityID, sim.EntityID, []ui.MessageLine) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "Raise", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	app := f.App("skill raise first tick")
	f.ConfigureSaveSeams(app, SaveStore{}, OriginalStore{}, nil)
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	app.Layout(messageWitnessW, messageWitnessH)
	id := f.live.mission.ids[0]
	hero := generatedCampaignEntity(t, f, id)
	var target sim.Entity
	distance := int32(1 << 30)
	for _, e := range f.live.world.Entities() {
		dx, dy := e.X-hero.X, e.Y-hero.Y
		if e.Domain == hero.Domain && f.live.world.Relations().Hostile(hero.Owner, e.Owner) && dx*dx+dy*dy < distance {
			distance, target = dx*dx+dy*dy, e
		}
	}
	if target.ID == 0 {
		t.Fatal("mission 20 lacks a hostile target")
	}
	f.live.tick()
	f.live.strike(uint32(id), uint32(target.ID))
	var rows []ui.MessageLine
	for n := 0; n < 2400 && len(rows) == 0 && uint64(f.live.world.Tick()) < stopAt; n++ {
		now, victim := generatedCampaignEntity(t, f, id), generatedCampaignEntity(t, f, target.ID)
		if dx, dy := victim.X-now.X, victim.Y-now.Y; dx*dx+dy*dy < 8 {
			f.live.strike(uint32(id), uint32(target.ID))
		}
		f.live.tick()
		rows = f.live.view.MessageLines()
	}
	return f, app, id, target.ID, rows
}

// saveAndLoad writes the front's mission through F2 SAVE and
// restores it into a fresh front through the application's LOAD.
func saveAndLoad(t *testing.T, f *FrontEnd, app *ui.App) (*FrontEnd, *ui.App) {
	t.Helper()
	dir, _ := autoGetF2Save(t, f, app, "game0001")
	return autoGetSession(t, dir)
}

// A SAV written shortly before a raise is loaded; the raise in the first tick
// after LOAD posts its notice, and a first tick after LOAD that raises nothing
// posts no row for the levels the save already held.
func TestReleaseSkillRaiseInTheFirstTickAfterLoadPostsItsNotice(t *testing.T) {
	probe, _, _, _, rows := fighterRun(t, 1<<62)
	if len(rows) == 0 {
		t.Fatal("the control run posted no raise")
	}
	raiseTick := uint64(probe.live.world.Tick())
	held := 0
	for back := uint64(1); back <= 40; back++ {
		f, app, id, target, rows := fighterRun(t, raiseTick-back)
		if len(rows) != 0 {
			t.Fatalf("a raise posted before tick %d", raiseTick-back)
		}
		fresh, _ := saveAndLoad(t, f, app)
		// The levels the save held are read from the restored mission: the
		// F2 flow steps frames before it writes, so the front's own reading
		// before it may be older than the file.
		saved := generatedCampaignEntity(t, fresh, id)
		if got := fresh.live.view.MessageLines(); len(got) != 0 {
			t.Fatalf("LOAD posted %+v for levels the save already held", got)
		}
		fresh.live.strike(uint32(id), uint32(target))
		fresh.live.tick()
		after := generatedCampaignEntity(t, fresh, id)
		got := fresh.live.view.MessageLines()
		if after.Skill[data.SkillBlade] == saved.Skill[data.SkillBlade] {
			if len(got) != 0 {
				t.Fatalf("a first tick with no raise posted %+v", got)
			}
			held++
			continue
		}
		want := announced(fresh.Words.SkillRaised[data.SkillBlade-1] + ": " + strconv.Itoa(int(after.Skill[data.SkillBlade])))
		if len(got) != 1 || got[0].Text != want[0].Text {
			t.Fatalf("the first-tick raise after LOAD posted %+v, want %+v", got, want)
		}
		t.Logf("raise at tick %d, SAV written %d ticks before, %d earlier LOADs held their levels and posted no row; first tick after LOAD: Blade %d -> %d, notice %q",
			raiseTick, back, held, saved.Skill[data.SkillBlade], after.Skill[data.SkillBlade], got[0].Text)
		return
	}
	t.Fatal("no SAV within 40 ticks of the raise had its raise land in the first tick after LOAD")
}
