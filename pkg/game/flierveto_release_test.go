package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// A melee hero's attack click on a hostile flier is refused; a bow hero's is held.
func TestReleaseMeleeHeroIsRefusedAFlierAndABowHeroIsNot(t *testing.T) {
	f := releaseFront(t)
	shapes, materials, weapons := missionTableWeapons(f.Table)
	bow, err := resolveWeaponForSlot(false, shapes, materials, weapons, data.SkillShoot)
	if err != nil || bow.Range < 2 {
		t.Fatalf("no bow for the shoot slot: %v %+v", err, bow)
	}
	const mission = 20
	for _, tc := range []struct {
		name   string
		weapon *data.Weapon
		dist   int32
		hold   bool
	}{
		{"melee", f.StartWeapon.Value(), 1, false},
		{"bow", bow, 3, true},
	} {
		f.Options = OptionsStore{}
		f.SetDeterministicFrames(true)
		party := MissionPartyAs(false, tc.weapon, f.Bodies, f.Table)
		app := f.App("flier veto " + tc.name)
		t.Cleanup(app.StopAudio)
		app.Layout(1024, 768)
		if err := app.OpenMission(f.MissionOpenerWith(mission, party)); err != nil {
			t.Fatal(err)
		}
		live := f.live
		for k := 0; k < 16 && live.mission.open; k++ {
			if err := app.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
		}
		heroID := live.mission.ids[0]
		hero, _ := live.entity(heroID)
		if tc.hold != (hero.Reach > 1) {
			t.Fatalf("%s hero reach is %d", tc.name, hero.Reach)
		}
		rel := live.world.Relations()
		var flier sim.Entity
		for _, e := range live.world.Entities() {
			if !e.Alive() || e.OffMap || e.Domain != sim.DomainAir || !rel.Hostile(e.Owner, hero.Owner) || flier.ID != 0 {
				continue
			}
			for k := 0; k < 200; k++ {
				if err := live.world.HeadlessPlace(heroID, e.X+tc.dist, e.Y); err != nil {
					t.Fatal(err)
				}
				if h, _ := live.entity(heroID); h.X == e.X+tc.dist && h.Y == e.Y {
					flier = e
					break
				}
			}
		}
		if flier.ID == 0 {
			t.Fatalf("mission %d: no hostile flier with a free cell %d east", mission, tc.dist)
		}
		live.push()
		hero, _ = live.entity(heroID)
		inspectionCentre(live, int(hero.X), int(hero.Y))
		if err := app.HeadlessSelectEntity(uint32(heroID)); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessKey("attack"); err != nil {
			t.Fatal(err)
		}
		x, y, err := app.HeadlessEntityPoint(uint32(flier.ID))
		if err != nil {
			t.Fatal(err)
		}
		pursuitAppClick(t, app, x, y)
		if len(live.pending) != 1 || live.pending[0].Kind != sim.KindAttack {
			t.Fatalf("%s: the click queued %+v, want one attack order", tc.name, live.pending)
		}
		held, hurt := false, false
		for tick := 0; tick < 240; tick++ {
			live.tick()
			h, _ := live.entity(heroID)
			if h.HasAttackTarget && h.AttackTarget == flier.ID {
				held = true
			}
			if fl, ok := live.entity(flier.ID); !ok || fl.HP < flier.HP {
				hurt = true
			}
		}
		if held != tc.hold {
			t.Errorf("%s hero held the order on flier %d = %v, want %v", tc.name, flier.ID, held, tc.hold)
		}
		if !tc.hold && hurt {
			t.Errorf("melee hero hurt flier %d", flier.ID)
		}
		t.Logf("%s hero %d reach %d: order held %v, flier %d hurt %v", tc.name, heroID, hero.Reach, held, flier.ID, hurt)
	}
}
