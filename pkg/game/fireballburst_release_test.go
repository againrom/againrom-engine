package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// burstRun is the burst record's leaves on each tick it lives, index 0 being
// the SAVE instant.
type burstRun struct {
	id     uint16
	leaves [][16]int32
}

// burstContinuation loads raw through the LOAD door and runs it beside want:
// the record's leaves on every tick and the tick it retires on.
func burstContinuation(t *testing.T, raw []byte, want burstRun) error {
	t.Helper()
	g := releaseFront(t)
	app, _ := openOriginalSAVApp(t, g, raw, "burst.sav")
	t.Cleanup(app.StopAudio)
	if g.live == nil || g.live.world == nil {
		return fmt.Errorf("LOAD opened no world")
	}
	for k := 0; k <= len(want.leaves); k++ {
		p, ok := savedProjectileByID(g.live.world, want.id)
		switch {
		case k < len(want.leaves) && !ok:
			return fmt.Errorf("record retired at tick %d, want leaves %v", k, want.leaves[k])
		case k < len(want.leaves) && shotLeaves(p) != want.leaves[k]:
			return fmt.Errorf("tick %d leaves %v, want %v", k, shotLeaves(p), want.leaves[k])
		case k >= len(want.leaves) && ok:
			return fmt.Errorf("record outlived tick %d", len(want.leaves)-1)
		}
		g.live.tick()
	}
	return nil
}

// A native Fire_Ball cast builds the picture 13 burst when its area blasts;
// SAVE writes the record, and a cold LOAD continues it leaf by leaf to its
// retirement. Run on the EN and the RU install.
func TestReleaseNativeFireBallBurstSurvivesSaveAndLoad(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	if len(party) != 1 || !party[0].Mage {
		t.Fatalf("mission party is not one mage: %+v", party)
	}
	fighter := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)[0]
	fighter.ID, fighter.Name, fighter.StartingHero = "fire-ball-target", "Fire Ball target", false
	f.Carried = append(party, fighter)
	app := f.App("native fire ball burst")
	t.Cleanup(app.StopAudio)
	dir := t.TempDir()
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	if err := app.OpenMission(f.MissionOpener(41)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	casterID := mw.mission.ids[0]
	caster, ok := mw.entity(casterID)
	rule, ruleOK := mw.world.Spell(fireBallSpell)
	if !ok || !ruleOK || caster.MaxMana == 0 || len(mw.mission.ids) != 2 {
		t.Fatalf("fixture: caster %+v rule %v ids %v", caster, ruleOK, mw.mission.ids)
	}
	target, _ := mw.entity(mw.mission.ids[1])
	if err := mw.world.HeadlessPlace(target.ID, caster.X+5, caster.Y); err != nil {
		t.Fatal(err)
	}
	book := caster.Book
	if !book.HasInstances() {
		book.State = sim.BookPresent
	}
	book.Slots[fireBallSpell-1] = sim.BookSpell{Range: rule.MaxRange, ManaCost: uint16(rule.ManaCost)}
	if err := mw.world.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{
		ID: casterID, KnownSpells: caster.KnownSpells | 1<<fireBallSpell, Book: book,
	}}); err != nil {
		t.Fatal(err)
	}
	mw.castAt(uint32(casterID), uint32(target.ID), fireBallSpell)
	var born sim.SavedProjectile
	var evs []sim.CastEvent
	for tick := 0; born.Picture == 0; tick++ {
		if tick > 400 {
			c, _ := mw.entity(casterID)
			tg, _ := mw.entity(target.ID)
			t.Fatalf("no burst record was built: mana %d cost %d known %b caster %+v target hp %d records %+v events %+v range %d", c.Mana, rule.ManaCost, c.KnownSpells, c.AttackPhase, tg.HP, mw.world.SavedProjectiles(), evs, rule.MaxRange)
		}
		evs = append(evs, f.LiveAdvanceCasts(1)...)
		for _, p := range mw.world.SavedProjectiles().Items {
			if p.Picture == fireBallBurstPicture {
				born = p
			}
		}
	}
	if born.ActionTarget != 0 || born.ActionSegments != 21 || born.ActionPhase != 0 {
		t.Fatalf("a burst after its first driver call: %+v", born)
	}
	records := mw.world.SavedProjectiles()
	raw := cityRosterF2Save(t, app, SaveStore{Dir: dir}, "burst")
	store := kitProjectileStore(t, raw)
	if store.FreeIndex != records.FreeIndex || !slices.Equal(store.IDs, records.IDs) {
		t.Fatalf("SAVE wrote the allocator and IDs %+v for the records %+v", store, records)
	}
	run := burstRun{id: born.ID, leaves: [][16]int32{shotLeaves(born)}}
	// The World record draws the explosion for the caster's owner, so the
	// caster's own burst is not held back by fog, and its frame advances.
	frames := map[int]bool{}
	for k := 0; k < 40; k++ {
		for _, d := range mw.savedProjectileDraws() {
			if d.Sheet == nil || d.Owner != caster.Owner {
				t.Fatalf("burst draw %+v, want a sheet frame owned by %d", d, caster.Owner)
			}
			frames[d.Frame] = true
		}
		mw.tick()
		p, ok := savedProjectileByID(mw.world, born.ID)
		if !ok {
			break
		}
		run.leaves = append(run.leaves, shotLeaves(p))
	}
	if len(run.leaves) < 10 || len(run.leaves) >= 40 {
		t.Fatalf("the burst lived %d ticks", len(run.leaves))
	}
	if len(frames) < 3 {
		t.Fatalf("the burst drew %d distinct frames over %d ticks", len(frames), len(run.leaves))
	}
	if err := burstContinuation(t, raw, run); err != nil {
		t.Fatalf("cold LOAD of the written SAV: %v", err)
	}
	t.Logf("burst %d picture %d lived %d ticks from the SAVE", born.ID, born.Picture, len(run.leaves))

	// Loss controls: the same check fails when the record is dropped and when
	// one of its leaves is changed.
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
		changed.Items = slices.Clone(store.Items)
		changed.IDs = slices.Clone(store.IDs)
		edit(&changed)
		if err := file.SetProjectiles(changed); err != nil {
			t.Fatal(err)
		}
		if burstContinuation(t, file.Marshal(), run) == nil {
			t.Fatalf("a SAV with a %s passed the continuation check", name)
		}
	}
}
