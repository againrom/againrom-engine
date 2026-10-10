package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// trackedShot is one record shot at the walking hero: the tick it was built
// on and the run after it.
type trackedShot struct {
	tick int
	run  shotRun
	dirs []int32
	raw  []byte
}

// walkUnderFire opens mission 41 with a bowman hero who fights until an
// enemy shoots at him, then walks back and forth four cells either side of
// where he stood. It returns the first later record shot at him whose dir
// changes in flight, SAVEd on the tick it is built when that tick is saveAt.
func walkUnderFire(t *testing.T, saveAt int) (trackedShot, bool) {
	dir := t.TempDir()
	f := releaseFront(t)
	bow, err := resolveWeaponForSlot(false, f.Table.Shapes, f.Table.Materials, f.Table.Weapons, data.SkillShoot)
	if err != nil {
		t.Fatal(err)
	}
	f.Carried = MissionPartyAs(false, bow, f.Bodies, f.Table)
	f.SetDeterministicFrames(true)
	app := f.App("tracked shot")
	t.Cleanup(app.StopAudio)
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	if err := app.OpenMission(f.MissionOpener(41)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	hero := mw.mission.ids[0]
	shotAttackNearest(15)(t, mw, hero)
	var legs []sim.CellPoint
	leg := 0
	seen := map[uint16]bool{}
	var open []trackedShot
	for tick := 0; tick < 1500; tick++ {
		h, _ := mw.entity(hero)
		switch {
		case !h.Alive():
			t.Fatalf("the hero fell at tick %d before a tracked shot was found", tick)
		case legs == nil:
			shotAnswer(mw, hero)
		case !h.HasTarget:
			mw.pending = append(mw.pending, sim.MoveTo(hero, legs[leg]))
			leg = 1 - leg
		}
		mw.tick()
		h, _ = mw.entity(hero)
		for i := range open {
			s := &open[i]
			if len(s.run.hp) >= 24 {
				continue
			}
			if p, ok := savedProjectileByID(mw.world, s.run.id); ok && len(s.run.leaves) == len(s.run.hp)+1 {
				s.run.leaves = append(s.run.leaves, shotLeaves(p))
				s.dirs = append(s.dirs, p.Dir)
				if p.Dir != p.ActionDir {
					t.Fatalf("record %d: dir %d differs from actiondir %d", p.ID, p.Dir, p.ActionDir)
				}
			}
			s.run.hp = append(s.run.hp, h.HP)
			if len(s.run.hp) == 24 && s.changed() {
				return *s, true
			}
		}
		born, ok := newShotAt(mw, hero, seen)
		if !ok {
			continue
		}
		if legs == nil {
			legs = []sim.CellPoint{{X: h.X - 4, Y: h.Y}, {X: h.X + 4, Y: h.Y}}
			continue
		}
		s := trackedShot{tick: tick, dirs: []int32{born.Dir},
			run: shotRun{id: born.ID, class: h.Class, startHP: h.HP, leaves: [][16]int32{shotLeaves(born)}}}
		if tick == saveAt {
			s.raw = cityRosterF2Save(t, app, SaveStore{Dir: dir}, "tracked shot")
		}
		open = append(open, s)
	}
	return trackedShot{}, false
}

func (s trackedShot) changed() bool {
	for _, d := range s.dirs[1:] {
		if d != s.dirs[0] {
			return true
		}
	}
	return false
}

// A native shot aims at its moving target on every driver call (ANIM-139):
// a record shot at the walking hero changes its dir in flight. Written by
// SAVE on the tick it is built, a cold LOAD continues the same leaves, its
// dir sequence included, tick for tick, as the uninterrupted run.
func TestReleaseUnitShotTracksAMovingTarget(t *testing.T) {
	first, ok := walkUnderFire(t, -1)
	if !ok {
		t.Fatal("no shot at the walking hero changed its dir in flight")
	}
	again, ok := walkUnderFire(t, first.tick)
	if !ok || again.tick != first.tick || again.raw == nil {
		t.Fatalf("the second run found tick %d (saved %v), want %d", again.tick, again.raw != nil, first.tick)
	}
	if err := shotContinuation(t, again.raw, again.run); err != nil {
		t.Fatalf("record %d, dir sequence %v: cold LOAD of the written SAV: %v", again.run.id, again.dirs, err)
	}
	t.Logf("tick %d: record %d dir sequence %v continues after a cold LOAD", again.tick, again.run.id, again.dirs)
	// Loss control: a SAV whose record holds another actiondir fails.
	store := kitProjectileStore(t, again.raw)
	for i := range store.Items {
		if store.Items[i].ID == again.run.id {
			store.Items[i].ActionDir = (store.Items[i].ActionDir + 8) & 15
		}
	}
	file, err := sav.Open(again.raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.SetProjectiles(store); err != nil {
		t.Fatal(err)
	}
	if shotContinuation(t, file.Marshal(), again.run) == nil {
		t.Fatal("a SAV with a changed actiondir passed the continuation check")
	}
}

// newShotAt is a record built this tick whose driver targets id.
func newShotAt(mw *mapWorld, id sim.EntityID, seen map[uint16]bool) (sim.SavedProjectile, bool) {
	d := mw.world.SavedWorldEffectDrivers()
	if d == nil {
		return sim.SavedProjectile{}, false
	}
	for _, row := range d.Projectiles {
		if row.Retired || seen[row.ID] {
			continue
		}
		seen[row.ID] = true
		if p, ok := savedProjectileByID(mw.world, row.ID); ok && row.HasTarget && row.Target == id && p.ActionPhase == 1 {
			return p, true
		}
	}
	return sim.SavedProjectile{}, false
}
