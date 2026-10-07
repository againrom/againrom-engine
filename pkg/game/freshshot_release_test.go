package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// shotLeaves is the record's sixteen leaves except actiontarget, which is the
// target's identity in each World's own namespace and is compared apart.
func shotLeaves(p sim.SavedProjectile) [16]int32 {
	return [16]int32{p.X, p.Y, p.Z, p.Picture, p.Dir, p.Phase, p.LastAction, p.Action, p.ActionDir,
		0, p.ActionX, p.ActionY, p.ActionZ, p.ActionPhase, p.ActionSegments, p.ActionSpell}
}

func savedProjectileByID(w *sim.World, id uint16) (sim.SavedProjectile, bool) {
	for _, p := range w.SavedProjectiles().Items {
		if p.ID == id {
			return p, true
		}
	}
	return sim.SavedProjectile{}, false
}

func savedProjectileFromSav(p sav.Projectile) sim.SavedProjectile {
	return sim.SavedProjectile{ID: p.ID, X: p.X, Y: p.Y, Z: p.Z, Picture: p.Picture, Dir: p.Dir, Phase: p.Phase,
		LastAction: p.LastAction, Action: p.Action, ActionDir: p.ActionDir, ActionX: p.ActionX, ActionY: p.ActionY,
		ActionZ: p.ActionZ, ActionPhase: p.ActionPhase, ActionSegments: p.ActionSegments, ActionSpell: p.ActionSpell}
}

// shotRun is what an uninterrupted World does after the SAVE: the record's
// leaves on each tick it lives, and the target's hit points after every tick.
type shotRun struct {
	id     uint16
	leaves [][16]int32 // index 0 is the SAVE instant, the last is the record's final tick
	hp     []int32     // hp[k] is the target's hit points after tick k+1
	class  int32
	// startHP is the target's hit points at the SAVE instant.
	startHP int32
}

// shotContinuation loads raw through the original LOAD door in a fresh front
// end and runs it beside want. It reports the first difference: the record,
// its driver and target, each tick's leaves, the tick the record retires on
// and the target's hit points on every tick.
func shotContinuation(t *testing.T, raw []byte, want shotRun) error {
	t.Helper()
	g := releaseFront(t)
	app, _ := openOriginalSAVApp(t, g, raw, "continuation.sav")
	t.Cleanup(app.StopAudio)
	if g.live == nil || g.live.world == nil {
		return fmt.Errorf("LOAD opened no world")
	}
	p, ok := savedProjectileByID(g.live.world, want.id)
	if !ok || shotLeaves(p) != want.leaves[0] {
		return fmt.Errorf("loaded record %d is %+v (present %v), want leaves %v", want.id, p, ok, want.leaves[0])
	}
	var driver sim.SavedProjectileDriver
	if d := g.live.world.SavedWorldEffectDrivers(); d != nil {
		for _, row := range d.Projectiles {
			if row.ID == want.id {
				driver = row
			}
		}
	}
	if !driver.HasTarget {
		return fmt.Errorf("LOAD armed no driver for the record")
	}
	target, ok := g.live.entity(driver.Target)
	if !ok || target.Class != want.class || target.HP != want.startHP {
		return fmt.Errorf("the driver's target is %+v for actiontarget %d", target, p.ActionTarget)
	}
	for k := 1; k <= len(want.hp); k++ {
		g.live.tick()
		p, ok := savedProjectileByID(g.live.world, want.id)
		switch {
		case k < len(want.leaves) && !ok:
			return fmt.Errorf("record retired on tick %d, want leaves %v", k, want.leaves[k])
		case k < len(want.leaves) && shotLeaves(p) != want.leaves[k]:
			return fmt.Errorf("tick %d leaves %v, want %v", k, shotLeaves(p), want.leaves[k])
		case k >= len(want.leaves) && ok:
			return fmt.Errorf("record outlived tick %d", len(want.leaves)-1)
		}
		if got, _ := g.live.entity(target.ID); got.HP != want.hp[k-1] {
			return fmt.Errorf("target hit points %d after tick %d, want %d", got.HP, k, want.hp[k-1])
		}
	}
	return nil
}

// A native shot at a unit is a World record: SAVE writes it, a cold LOAD
// continues it leaf by leaf to its end, and the blow that goes with it lands
// on the same tick as in the uninterrupted run.
func TestReleaseFreshShotWritesProjectileRecord(t *testing.T) {
	dir := t.TempDir()
	f := releaseFront(t)
	bow, err := resolveWeaponForSlot(false, f.Table.Shapes, f.Table.Materials, f.Table.Weapons, data.SkillShoot)
	if err != nil {
		t.Fatal(err)
	}
	f.Carried = MissionPartyAs(false, bow, f.Bodies, f.Table)
	f.SetDeterministicFrames(true)
	app := f.App("fresh arrow")
	t.Cleanup(app.StopAudio)
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	if err := app.OpenMission(f.MissionOpener(41)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	hero := mw.mission.ids[0]
	shotAttackNearest(15)(t, mw, hero)
	var born sim.SavedProjectile
	for tick := 0; born.Picture == 0; tick++ {
		if tick > 600 {
			t.Fatal("no arrow was released")
		}
		shotAnswer(mw, hero)
		mw.tick()
		for _, p := range mw.world.SavedProjectiles().Items {
			if p.Picture != unitShotDeformationPicture {
				born = p
			}
		}
	}
	records := mw.world.SavedProjectiles()
	drivers := mw.world.SavedWorldEffectDrivers()
	if drivers == nil || len(drivers.Projectiles) != len(records.Items) || len(records.IDs) != len(records.Items) || records.FreeIndex <= born.ID {
		t.Fatalf("records and their allocator: %+v %+v", records, drivers)
	}
	var victim sim.Entity
	for _, d := range drivers.Projectiles {
		if d.ID == born.ID {
			victim, _ = mw.entity(d.Target)
		}
	}
	if born.Action != 1 || born.ActionSegments < 1 || born.ActionPhase != 1 || victim.ID == 0 {
		t.Fatalf("a record after its first driver call: %+v", born)
	}

	raw := cityRosterF2Save(t, app, SaveStore{Dir: dir}, "fresh shot")
	store := kitProjectileStore(t, raw)
	if store.FreeIndex != records.FreeIndex || !slices.Equal(store.IDs, records.IDs) || len(store.Items) != len(records.Items) {
		t.Fatalf("SAVE wrote the allocator and IDs %+v for the records %+v", store, records)
	}
	for _, w := range store.Items {
		p, ok := savedProjectileByID(mw.world, w.ID)
		if !ok || shotLeaves(savedProjectileFromSav(w)) != shotLeaves(p) || w.ActionTarget <= 0 {
			t.Fatalf("SAVE wrote %+v for the record %+v", w, p)
		}
	}

	run := shotRun{id: born.ID, class: victim.Class, startHP: victim.HP, leaves: [][16]int32{shotLeaves(born)}}
	flying := true
	for k := 0; k < 24; k++ {
		mw.tick()
		if p, ok := savedProjectileByID(mw.world, born.ID); ok && flying {
			run.leaves = append(run.leaves, shotLeaves(p))
		} else {
			flying = false
		}
		e, _ := mw.entity(victim.ID)
		run.hp = append(run.hp, e.HP)
	}
	if len(run.leaves) < 3 {
		t.Fatalf("the record lived %d ticks", len(run.leaves))
	}
	hit := int32(victim.HP)
	for _, v := range run.hp {
		hit = min(hit, v)
	}
	if hit >= victim.HP {
		t.Fatal("the shot's blow never landed in the window")
	}
	if err := shotContinuation(t, raw, run); err != nil {
		t.Fatalf("cold LOAD of the written SAV: %v", err)
	}
	t.Logf("record %d picture %d segments %d lived %d ticks; target hit points %d, lowest %d", born.ID, born.Picture, born.ActionSegments, len(run.leaves), victim.HP, hit)

	// Loss controls: the same check fails when the record is dropped from the
	// SAV and when one of its leaves is changed.
	for name, edit := range map[string]func(*sav.ProjectileStore){
		"dropped record": func(s *sav.ProjectileStore) {
			s.IDs = slices.DeleteFunc(s.IDs, func(id uint16) bool { return id == born.ID })
			s.Items = slices.DeleteFunc(s.Items, func(p sav.Projectile) bool { return p.ID == born.ID })
		},
		"changed leaf": func(s *sav.ProjectileStore) {
			for i := range s.Items {
				if s.Items[i].ID == born.ID {
					s.Items[i].ActionSegments++
				}
			}
		},
	} {
		file, err := sav.Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		changed := store
		changed.Items = append([]sav.Projectile(nil), store.Items...)
		changed.IDs = append([]uint16(nil), store.IDs...)
		edit(&changed)
		if err := file.SetProjectiles(changed); err != nil {
			t.Fatal(err)
		}
		if shotContinuation(t, file.Marshal(), run) == nil {
			t.Fatalf("a SAV with a %s passed the continuation check", name)
		}
	}
}
