package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func secondStaffMission(t *testing.T) (*FrontEnd, *ui.App) {
	t.Helper()
	f := secondGameFront(t)
	app := f.App("second staff experience")
	app.Layout(1024, 768)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if err := f.startGeneratedTown(ui.ChargenResult{Name: "Staff mage", Choices: []int{0, 1, 0}, Stats: []int{31, 27, 24, 29}}); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"TAVERN", "TALK 517"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatalf("initial inn dialogue did not open: %v", err)
	}
	for pages := 0; pages < 64; pages++ {
		if err := app.HeadlessActivate("notice"); err != nil {
			break
		}
	}
	for _, target := range []string{"EXIT", "GATES", "mission 10"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenMap {
		t.Fatalf("first campaign mission did not open: %v", app.Screen())
	}
	for range 20 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	dismissSecondMissionDialogue(t, app, f.live)
	f.live.stopped = true
	secondCheatUnlock(t, f, app)
	secondCheatWantReply(t, f, secondCheatLines(t, f, app, "#modify army +god"), 7, sim.SelfSlot)
	f.live.stopped = false
	return f, app
}

func secondStaffTrolls(t *testing.T, f *FrontEnd) []sim.Entity {
	t.Helper()
	c := f.Table.Units
	var out []sim.Entity
	for _, e := range f.live.world.Entities() {
		for i := 1; i < c.Len(); i++ {
			if p := c.EntryParams(i); e.Alive() && len(p) > 0x1d && p[0x1d] == e.TypeID && c.EntryName(i) == "Troll" {
				out = append(out, e)
				break
			}
		}
	}
	for i := 1; i < len(out); i++ {
		if out[i].MaxHP > out[0].MaxHP {
			out[0], out[i] = out[i], out[0]
		}
	}
	if len(out) < 2 {
		t.Fatalf("mission 10 holds %d trolls, want at least 2", len(out))
	}
	return out
}

func secondStaffStrike(t *testing.T, f *FrontEnd, app *ui.App, victim sim.EntityID) (before, after int32, v sim.Entity) {
	t.Helper()
	mw := f.live
	id := mw.mission.ids[0]
	for range 200 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		dismissSecondMissionDialogue(t, app, mw)
	}
	v, _ = mw.world.Entity(victim)
	if err := mw.world.HeadlessPlace(id, v.X-1, v.Y); err != nil {
		t.Fatal(err)
	}
	h, _ := mw.world.Entity(id)
	before, hp := h.SkillXP[1], v.HP
	mw.strike(uint32(id), uint32(victim))
	for tick := 0; tick < 600 && v.HP == hp; tick++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		dismissSecondMissionDialogue(t, app, mw)
		v, _ = mw.world.Entity(victim)
	}
	if v.HP == hp {
		t.Fatalf("the staff did not wound unit %d in 600 ticks", victim)
	}
	h, _ = mw.world.Entity(id)
	return before, h.SkillXP[1], v
}

func TestReleaseSecondStaffStrongestTrollPaysNoExperience(t *testing.T) {
	f, _ := secondStaffMission(t)
	hero, _ := f.live.world.Entity(f.live.mission.ids[0])
	if hero.ActorLoad.Source.Class != 0 || !hero.GainsXP || !sim.InPersistBand(hero.TypeID) || hero.WeaponSpell != 1 || hero.Skill[1] != 20 {
		t.Fatalf("hero: source class %d, gains %v, type %d, staff spell %d, fire %d", hero.ActorLoad.Source.Class, hero.GainsXP, hero.TypeID, hero.WeaponSpell, hero.Skill[1])
	}
	trolls := secondStaffTrolls(t, f)
	for n, troll := range trolls[:2] {
		marked := n == 0
		if troll.SuppressRewards != marked {
			t.Fatalf("troll %d (health %d) reward mark %v, want %v", troll.ID, troll.MaxHP, troll.SuppressRewards, marked)
		}
		f, app := secondStaffMission(t)
		before, hit, v := secondStaffStrike(t, f, app, troll.ID)
		if err := f.live.world.HeadlessDamage(troll.ID, v.HP-1); err != nil {
			t.Fatal(err)
		}
		_, kill, v := secondStaffStrike(t, f, app, troll.ID)
		t.Logf("troll %d owner %d health %d value %d mark %v: fire experience %d, hit %d, kill %d", troll.ID, troll.Owner, troll.MaxHP, troll.ExperienceValue(), marked, before, hit, kill)
		if v.Alive() || !v.HasKillCredit || v.KillCreditSpell != 1 {
			t.Fatalf("troll %d: alive %v, credit %v, spell %d", troll.ID, v.Alive(), v.HasKillCredit, v.KillCreditSpell)
		}
		if paid, unpaid := hit > before && kill > hit, hit == before && kill == hit; marked && !unpaid || !marked && !paid {
			t.Errorf("troll %d marked %v: fire experience %d, hit %d, kill %d", troll.ID, marked, before, hit, kill)
		}
	}
}

func TestReleaseSecondGodLeavesANativeHeroAtTheSourceDerive(t *testing.T) {
	f, _ := secondStaffMission(t)
	id := f.live.mission.ids[0]
	native, _ := f.live.world.Entity(id)
	if native.ActorLoad.Source.Class != 0 || native.Spirit >= 60 {
		t.Fatalf("hero source class %d, Spirit %d", native.ActorLoad.Source.Class, native.Spirit)
	}
	s := data.HumanState{Body: native.NativeBasis.Body, Reaction: uint16(native.Reaction), Mind: uint16(native.Mind), Spirit: uint16(native.Spirit)}
	for j := range s.Modifier.Defence.Protection {
		s.Modifier.Defence.Protection[j], s.Modifier.Defence.Resistance[j] = 100, 100
	}
	source, err := s.Derive()
	if err != nil {
		t.Fatal(err)
	}
	card := f.live.chars[id]
	t.Logf("Spirit %d: native protection %v resistance %v card %v", native.Spirit, native.Protection, native.Resistance, card.Protection)
	for i := range native.Protection {
		want := int32(source.Defence.Protection[i+1])
		if native.Protection[i] != want || want >= 100 || card.Protection[i] != int(want) || native.Resistance[i] != source.Defence.Resistance[i+1] {
			t.Errorf("slot %d at Spirit %d: native protection %d, card %d, resistance %d; source protection %d, resistance %d",
				i, native.Spirit, native.Protection[i], card.Protection[i], native.Resistance[i], want, source.Defence.Resistance[i+1])
		}
	}
}
