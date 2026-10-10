package game

import (
	"fmt"
	"os"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// An object in flight is one World record whatever builds it. Each witness
// here drives one producer on the installed data, takes a SAVE while its
// record flies, and runs the written SAV through a cold LOAD beside the
// uninterrupted World: the record's leaves on every tick, the tick it retires
// on, and the hit points its effect reaches must match.

// flightEffect reads the effect a record's flight leads to in one World: a
// target's hit points or a structure's health.
type flightEffect func(*mapWorld) int32

// recordRun is the uninterrupted World after the SAVE.
type recordRun struct {
	id     uint16
	leaves [][16]int32 // index 0 is the SAVE instant
	effect []int32     // effect[k] is the effect after tick k+1
	start  int32
}

const flightWindow = 48

// unitEffect follows the actor that stands at e's cell with e's class at the
// SAVE instant; a cold LOAD may renumber actors, so each World resolves it once.
func unitEffect(e sim.Entity) flightEffect {
	ids := map[*mapWorld]sim.EntityID{}
	return func(mw *mapWorld) int32 {
		id, ok := ids[mw]
		if !ok {
			for _, c := range mw.world.Entities() {
				if c.Class == e.Class && c.X == e.X && c.Y == e.Y {
					id, ok = c.ID, true
				}
			}
			if !ok {
				return -1 << 30
			}
			ids[mw] = id
		}
		c, _ := mw.entity(id)
		return c.HP
	}
}

// structureEffect is one structure's health.
func structureEffect(id sim.StructureID) flightEffect {
	return func(mw *mapWorld) int32 {
		for _, s := range mw.world.Structures() {
			if s.ID == id {
				return int32(int16(s.Field42))
			}
		}
		return -1 << 30
	}
}

// flightUninterrupted runs the live World on from the SAVE.
func flightUninterrupted(mw *mapWorld, born sim.SavedProjectile, effect flightEffect) recordRun {
	run := recordRun{id: born.ID, leaves: [][16]int32{shotLeaves(born)}, start: effect(mw)}
	flying := true
	for range flightWindow {
		mw.tick()
		if p, ok := savedProjectileByID(mw.world, born.ID); ok && flying {
			run.leaves = append(run.leaves, shotLeaves(p))
		} else {
			flying = false
		}
		run.effect = append(run.effect, effect(mw))
	}
	return run
}

// flightContinuation loads raw through the original LOAD door in a fresh
// front end and runs it beside want.
func flightContinuation(t *testing.T, raw []byte, want recordRun, effect flightEffect) error {
	t.Helper()
	g := releaseFront(t)
	app, _ := openOriginalSAVApp(t, g, raw, "flight.sav")
	t.Cleanup(app.StopAudio)
	if g.live == nil || g.live.world == nil {
		return fmt.Errorf("LOAD opened no world")
	}
	if got := effect(g.live); got != want.start {
		return fmt.Errorf("loaded effect %d, want %d", got, want.start)
	}
	for k := 0; k <= flightWindow; k++ {
		p, ok := savedProjectileByID(g.live.world, want.id)
		switch {
		case k < len(want.leaves) && !ok:
			return fmt.Errorf("record retired at tick %d, want leaves %v", k, want.leaves[k])
		case k < len(want.leaves) && shotLeaves(p) != want.leaves[k]:
			return fmt.Errorf("tick %d leaves %v, want %v", k, shotLeaves(p), want.leaves[k])
		case k >= len(want.leaves) && ok:
			return fmt.Errorf("record outlived tick %d", len(want.leaves)-1)
		}
		if k > 0 {
			if got := effect(g.live); got != want.effect[k-1] {
				return fmt.Errorf("effect %d after tick %d, want %d", got, k, want.effect[k-1])
			}
		}
		if k < flightWindow {
			g.live.tick()
		}
	}
	return nil
}

// flightWitness is one producer's proof from the SAVE on: the SAV store holds
// the live records, the record's bytes are its leaves, a cold LOAD continues
// it and its effect, and dropping the record or changing a leaf fails that.
// It answers the written SAV and the uninterrupted run.
func flightWitness(t *testing.T, app *ui.App, dir string, mw *mapWorld, born sim.SavedProjectile,
	effect flightEffect, label string) ([]byte, recordRun) {
	t.Helper()
	records := mw.world.SavedProjectiles()
	raw := cityRosterF2Save(t, app, SaveStore{Dir: dir}, label)
	store := kitProjectileStore(t, raw)
	if store.FreeIndex != records.FreeIndex || !slices.Equal(store.IDs, records.IDs) || len(store.Items) != len(records.Items) {
		t.Fatalf("SAVE wrote the allocator and IDs %+v for the records %+v", store, records)
	}
	var written sav.Projectile
	for _, w := range store.Items {
		if w.ID == born.ID {
			written = w
		}
	}
	if shotLeaves(savedProjectileFromSav(written)) != shotLeaves(born) {
		t.Fatalf("SAVE wrote %+v for the record %+v", written, born)
	}
	run := flightUninterrupted(mw, born, effect)
	if len(run.leaves) < 2 {
		t.Fatalf("the record lived %d ticks from the SAVE", len(run.leaves))
	}
	if err := flightContinuation(t, raw, run, effect); err != nil {
		t.Fatalf("cold LOAD of the written SAV: %v", err)
	}
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
		if flightContinuation(t, file.Marshal(), run, effect) == nil {
			t.Fatalf("a SAV with a %s passed the continuation check", name)
		}
	}
	return raw, run
}

// flightLanded is the first tick whose effect differs from the SAVE instant's,
// or -1.
func flightLanded(run recordRun) int {
	for k, v := range run.effect {
		if v != run.start {
			return k + 1
		}
	}
	return -1
}

// flightKitBase numbers the owner kit's files: a spell cast in flight on EN
// and RU, then a structure shot in flight on EN and RU.
const flightKitBase = 9613

// flightKit is a kit file's number and its chooser label.
func flightKit(f *FrontEnd, structure bool) (int, string) {
	edition := kitOwnerEdition(f)
	n, change := flightKitBase, "spell in flight"
	if structure {
		n, change = n+2, "shot at structure"
	}
	if edition == "RU" {
		n++
	}
	return n, fmt.Sprintf("%d %s %s", n, change, edition)
}

// flightKitWrite writes raw into AGAINROM_OWNER_KIT_OUT when it is set.
func flightKitWrite(t *testing.T, f *FrontEnd, raw []byte, structure bool) {
	t.Helper()
	if os.Getenv("AGAINROM_OWNER_KIT_OUT") == "" {
		return
	}
	edition := kitOwnerEdition(f)
	n, label := flightKit(f, structure)
	kitOwnerWrite(t, kitOwnerOutput(t, edition), fmt.Sprintf("game%d.sav", n), raw)
	t.Logf("KIT %s label %q bytes %d", edition, label, len(raw))
}

// flightMageArena opens mission 41 with a mage and a party fighter standing
// five cells east of him. The chargen mage carries his Fire Arrow staff; the
// installed mission mage does not.
func flightMageArena(t *testing.T, chargen bool) (*FrontEnd, *ui.App, string, sim.EntityID, sim.Entity) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	if chargen {
		party = f.ChargenParty(ui.ChargenResult{Name: "Flight", Choices: []int{0, 1, 0}, Stats: []int{30, 30, 40, 40}})
	}
	fighter := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)[0]
	fighter.ID, fighter.Name, fighter.StartingHero = "flight-target", "Flight target", false
	f.Carried = append(party[:1], fighter)
	app := f.App("object in flight")
	t.Cleanup(app.StopAudio)
	dir := t.TempDir()
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	if err := app.OpenMission(f.MissionOpener(41)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	if len(mw.mission.ids) != 2 {
		t.Fatalf("mission party ids %v", mw.mission.ids)
	}
	caster, _ := mw.entity(mw.mission.ids[0])
	target, _ := mw.entity(mw.mission.ids[1])
	if err := mw.world.HeadlessPlace(target.ID, caster.X+5, caster.Y); err != nil {
		t.Fatal(err)
	}
	target, _ = mw.entity(target.ID)
	return f, app, dir, caster.ID, target
}

// flightLearn writes spell into the caster's book with mana to spare.
func flightLearn(t *testing.T, mw *mapWorld, caster sim.EntityID, spell uint32) {
	t.Helper()
	e, _ := mw.entity(caster)
	rule, ok := mw.world.Spell(spell)
	if !ok {
		t.Fatalf("no installed row for spell %d", spell)
	}
	book := e.Book
	if !book.HasInstances() {
		book.State = sim.BookPresent
	}
	book.Slots[spell-1] = sim.BookSpell{Range: rule.MaxRange, ManaCost: uint16(rule.ManaCost)}
	if err := mw.world.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{
		ID: caster, KnownSpells: e.KnownSpells | 1<<spell, Book: book}}); err != nil {
		t.Fatal(err)
	}
	if err := mw.world.ImportOriginalActorPools([]sim.OriginalActorPools{{ID: caster, HP: e.MaxHP, MaxHP: e.MaxHP, Mana: 1000, MaxMana: 1000}}); err != nil {
		t.Fatal(err)
	}
}

// flightAwait ticks until a record pick admits is built and answers it. A
// mission notice is closed before each tick, as a SAVE cannot open over one.
func flightAwait(t *testing.T, f *FrontEnd, app *ui.App, pick func(sim.SavedProjectile, sim.SavedProjectileDriver) bool) sim.SavedProjectile {
	t.Helper()
	mw := f.live
	for tick := 0; tick < 600; tick++ {
		for k := 0; k < 8 && app.HeadlessNoticeOpen(); k++ {
			if err := app.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
		}
		f.LiveAdvanceCasts(1)
		d := mw.world.SavedWorldEffectDrivers()
		if d == nil || app.HeadlessNoticeOpen() {
			continue
		}
		for _, row := range d.Projectiles {
			if p, ok := savedProjectileByID(mw.world, row.ID); ok && !row.Retired && pick(p, row) {
				return p
			}
		}
	}
	a, _ := mw.entity(mw.mission.ids[0])
	picture, delay, drawn := mw.classShot(a.Class)
	t.Fatalf("no record was built: %+v; first party actor %d at %d,%d class %d attack %v kind %d target %d phase %d reach %d mana %d known %#x shot %d/%d/%v tick %d",
		mw.world.SavedProjectiles(), a.ID, a.X, a.Y, a.Class, a.HasAttackTarget, a.AttackTargetKind, a.AttackTarget, a.AttackPhase, a.Reach, a.Mana, a.KnownSpells, picture, delay, drawn, mw.world.Tick())
	return sim.SavedProjectile{}
}

// A book Fire Arrow is a cast record (ANIM-147): picture 10 aimed at the
// target cell's centre. SAVE writes it and a cold LOAD continues it to the
// same landing and damage.
func TestReleaseBookCastFlightSurvivesSaveAndLoad(t *testing.T) {
	f, app, dir, caster, target := flightMageArena(t, false)
	mw := f.live
	flightLearn(t, mw, caster, 1)
	mw.castAt(uint32(caster), uint32(target.ID), 1)
	born := flightAwait(t, f, app, func(p sim.SavedProjectile, _ sim.SavedProjectileDriver) bool {
		return int(p.Picture) == data.CastPicture(1) && p.ActionSegments > 1
	})
	if born.ActionX != target.X*256+128 || born.ActionY != target.Y*256+128 {
		t.Fatalf("the cast record aims at %d,%d, want the target cell's centre", born.ActionX, born.ActionY)
	}
	_, label := flightKit(f, false)
	raw, run := flightWitness(t, app, dir, mw, born, unitEffect(target), label)
	if flightLanded(run) < 0 {
		t.Fatal("the cast's damage never landed in the window")
	}
	flightKitWrite(t, f, raw, false)
	t.Logf("record %d picture %d lived %d ticks from the SAVE; effect %d landed on tick %d", born.ID, born.Picture, len(run.leaves), run.start, flightLanded(run))
}

// A staff's Fire Arrow release is cast-diverted (SAV-1129): it builds the same
// cast record a book cast builds, at the release.
func TestReleaseStaffReleaseFlightSurvivesSaveAndLoad(t *testing.T) {
	f, app, dir, caster, target := flightMageArena(t, true)
	mw := f.live
	if e, _ := mw.entity(caster); e.WeaponSpell != 1 {
		t.Fatalf("the chargen mage's weapon spell is %d, want Fire Arrow", e.WeaponSpell)
	}
	mw.strike(uint32(caster), uint32(target.ID))
	born := flightAwait(t, f, app, func(p sim.SavedProjectile, _ sim.SavedProjectileDriver) bool {
		return int(p.Picture) == data.CastPicture(1) && p.ActionSegments > 1
	})
	_, run := flightWitness(t, app, dir, mw, born, unitEffect(target), "staff release")
	if flightLanded(run) < 0 {
		t.Fatal("the staff's blow never landed in the window")
	}
	t.Logf("record %d lived %d ticks from the SAVE; effect %d landed on tick %d", born.ID, len(run.leaves), run.start, flightLanded(run))
}

// A staged stage of Fire Sacrifice builds one burst record per accepted cell
// (ANIM-148): picture 17 at the cell centre, started at actionphase -1.
func TestReleaseStagedBurstSurvivesSaveAndLoad(t *testing.T) {
	const fireSacrifice = 4
	f, app, dir, caster, target := flightMageArena(t, false)
	mw := f.live
	if !mw.projectiles.HasPicture(data.BurstPicture(fireSacrifice)) {
		t.Skip("this install's registry holds no staged burst picture")
	}
	// Fire Sacrifice has no range: it is cast on the caster's own cell, and
	// the target stands inside its radius.
	if err := mw.world.HeadlessPlace(target.ID, target.X-3, target.Y); err != nil {
		t.Fatal(err)
	}
	target, _ = mw.entity(target.ID)
	flightLearn(t, mw, caster, fireSacrifice)
	if why := mw.world.BookSpellRefusal(caster, caster, fireSacrifice); why != "" {
		t.Fatalf("the book refuses Fire Sacrifice on the caster: %s", why)
	}
	mw.castAt(uint32(caster), uint32(caster), fireSacrifice)
	born := flightAwait(t, f, app, func(p sim.SavedProjectile, _ sim.SavedProjectileDriver) bool {
		return int(p.Picture) == data.BurstPicture(fireSacrifice) && p.ActionSegments > 1
	})
	if born.X != born.ActionX || born.Y != born.ActionY || born.ActionTarget != 0 {
		t.Fatalf("a staged burst %+v, want it standing on its cell with no target", born)
	}
	_, run := flightWitness(t, app, dir, mw, born, unitEffect(target), "staged burst")
	t.Logf("burst %d picture %d lived %d ticks from the SAVE; effect %d changed on tick %d", born.ID, born.Picture, len(run.leaves), run.start, flightLanded(run))
}

// A ranged unit's shot at a structure is an ordinary unit-shot record homing
// on the structure's anchor (SAV-1197). SAVE writes it with the structure's
// key, and a cold LOAD continues it to the same blow.
func TestReleaseStructureShotSurvivesSaveAndLoad(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	bow, err := resolveWeaponForSlot(false, f.Table.Shapes, f.Table.Materials, f.Table.Weapons, data.SkillShoot)
	if err != nil {
		t.Fatal(err)
	}
	party := MissionPartyAs(false, bow, f.Bodies, f.Table)
	f.Carried = party[:1]
	app := f.App("structure shot in flight")
	t.Cleanup(app.StopAudio)
	dir := t.TempDir()
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	for k := 0; k < 16 && mw.mission.open; k++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	shooter := mw.mission.ids[0]
	const sid = sim.StructureID(4)
	var target sim.Structure
	for _, s := range mw.world.Structures() {
		if s.ID == sid {
			target = s
		}
	}
	if target.Width == 0 {
		t.Fatal("mission 10 has no structure 4")
	}
	if err := mw.world.HeadlessPlace(shooter, target.Col-6, target.Row); err != nil {
		t.Fatal(err)
	}
	mw.pending = append(mw.pending, sim.AttackStructure(shooter, sid))
	mw.commanded[shooter] = true
	born := flightAwait(t, f, app, func(p sim.SavedProjectile, d sim.SavedProjectileDriver) bool {
		return d.TargetStructure && p.ActionSegments > 1
	})
	if born.ActionTarget != int32(sid) || born.ActionX != target.Col*256+128 || born.ActionY != target.Row*256+128 {
		t.Fatalf("the structure shot %+v, want key %d aimed at the anchor's centre", born, sid)
	}
	_, label := flightKit(f, true)
	raw, run := flightWitness(t, app, dir, mw, born, structureEffect(sid), label)
	flightKitWrite(t, f, raw, true)
	for _, w := range kitProjectileStore(t, raw).Items {
		if w.ID == born.ID && w.ActionTarget <= 0 {
			t.Fatalf("SAVE wrote actiontarget %d for the structure shot", w.ActionTarget)
		}
	}
	t.Logf("record %d picture %d lived %d ticks from the SAVE; structure health %d changed on tick %d", born.ID, born.Picture, len(run.leaves), run.start, flightLanded(run))
}
