package game

import (
	"bytes"
	"fmt"
	"image"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

func targetMarkFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	f.Projectiles = sbSheets()
	return f
}

func targetMarkCold(t *testing.T, f *FrontEnd, factories ...func(*testing.T) *FrontEnd) *FrontEnd {
	t.Helper()
	before, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, func() time.Time { return time.Unix(1, 0) })
	name, err := save(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil || len(a.Heals) != 0 {
		t.Fatal("expected empty legacy Heals", err)
	}
	visuals := f.live.residue()
	if !reflect.DeepEqual(a.TargetMarks, visuals.TargetMarks) {
		t.Fatal("SAV did not write current packed target marks")
	}
	factory := targetMarkFront
	if len(factories) != 0 {
		factory = factories[0]
	}
	cold := factory(t)
	app := cold.App("target mark cold LOAD")
	app.Layout(1024, 768)
	t.Cleanup(app.StopAudio)
	_, list, load := cold.SaveSeams(store, OriginalStore{}, nil)
	app.SetSaveSeams(nil, list, load)
	groundAppLoad(t, app, list, name)
	after, err := cold.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || f.live.world.Hash() != cold.live.world.Hash() ||
		!reflect.DeepEqual(f.live.world.SavedProjectiles(), cold.live.world.SavedProjectiles()) ||
		!reflect.DeepEqual(f.live.world.SavedWorldEffectDrivers(), cold.live.world.SavedWorldEffectDrivers()) {
		t.Fatal("ordinary SAVE/cold LOAD changed complete World, Prj leaves/store/allocator or drivers")
	}
	return cold
}

func TestRetainedProjectileTargetMarkColdLOAD(t *testing.T) {
	counterSilentDevice(t)
	for _, tc := range []struct {
		picture, segments int32
		target            bool
	}{
		{20, 2, true}, {30, 2, true}, {20, 1, true}, {20, 2, false}, {10, 2, true},
	} {
		t.Run(fmt.Sprintf("picture%d-count%d-target%v", tc.picture, tc.segments, tc.target), func(t *testing.T) {
			f := targetMarkFront(t)
			app := f.App("retained target mark")
			app.Layout(1024, 768)
			t.Cleanup(app.StopAudio)
			if err := app.OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
			caster, target := poolEntity(t, f.live.world, 91), poolEntity(t, f.live.world, 92)
			cx, cy, _ := f.live.world.ProjectilePoint(caster.ID)
			tx, ty, _ := f.live.world.ProjectilePoint(target.ID)
			if _, ok := f.live.world.ReleaseCast(sim.CastRecord{Caster: caster.ID, Picture: tc.picture,
				X: cx, Y: cy, Target: target.ID, HasTarget: tc.target, AimX: tx, AimY: ty, Segments: tc.segments}); !ok {
				t.Fatal("release rejected")
			}
			if err := f.live.world.HeadlessPlace(target.ID, target.X+4, target.Y); err != nil {
				t.Fatal(err)
			}
			cold := targetMarkCold(t, f)
			var report sim.Report
			if n := cold.live.tickStep(nil, true, func(r sim.Report) { report = r }); n != 0 || len(report.Casts) != 0 {
				t.Fatal("unexpected command/cast")
			}
			want := tc.segments == 2 && tc.target && (tc.picture == 20 || tc.picture == 30)
			if got := len(cold.live.markElements) != 0; got != want {
				t.Fatalf("retained branch mark=%v, want %v", got, want)
			}
			if len(cold.live.healBursts) != 0 {
				t.Fatal("retained branch created legacy Heals")
			}
			if !want {
				return
			}
			if el := cold.live.markElements[0]; el.Entity != target.ID || el.Kind != int(tc.picture) || el.Count != 32 {
				t.Fatal("target branch did not admit packed kind/count32", el)
			}
			marks := cold.live.markDraws(target.ID, 1)
			if len(marks) < 3 || len(marks) > 5 {
				t.Fatal("missing first shower cohort", len(marks))
			}
			for _, m := range marks {
				if m.Mark.Record != int(tc.picture) || m.Mark.Phase != 0 || m.Sheet != cold.Projectiles.Sheet(int(tc.picture)) {
					t.Fatal("target mark selected wrong builder/sheet/phase", m)
				}
			}
			placed := poolEntity(t, cold.live.world, 92)
			if err := cold.live.world.HeadlessPlace(placed.ID, placed.X+2, placed.Y+1); err != nil {
				t.Fatal(err)
			}
			cold.live.push()
			found := false
			for _, e := range cold.live.entityDraws() {
				if e.ID == uint32(target.ID) {
					found = true
					current := poolEntity(t, cold.live.world, 92)
					if e.Cell != image.Pt(int(current.X), int(current.Y)) || !reflect.DeepEqual(e.Marks, marks) {
						t.Fatal("mark did not follow current actor pose", e.Cell, current.X, current.Y)
					}
				}
			}
			if !found {
				t.Fatal("marked target absent from production projection")
			}
			// The zero-count record retires next tick; its mark retains 31 rebuilds.
			cold.live.tickStep(nil, true)
			if len(cold.live.world.SavedProjectiles().Items) != 0 || cold.live.markElements[0].Count != 31 {
				t.Fatal("projectile retirement closed or reset the independent mark")
			}
			restored := targetMarkCold(t, cold)
			if !reflect.DeepEqual(cold.live.markElements, restored.live.markElements) ||
				!reflect.DeepEqual(cold.live.markDraws(target.ID, 1), restored.live.markDraws(target.ID, 1)) {
				t.Fatal("current-state SAV lost target mark countdown, seed, binding or records")
			}
			for tick := 0; tick < 32; tick++ {
				if tick == 31 {
					e, _ := cold.live.entity(target.ID)
					cold.live.enqueue(uint32(target.ID), int(e.X+1), int(e.Y))
					restored.live.enqueue(uint32(target.ID), int(e.X+1), int(e.Y))
				}
				cold.live.tickStep(nil, true)
				restored.live.tickStep(nil, true)
				if cold.live.world.Hash() != restored.live.world.Hash() || !reflect.DeepEqual(cold.live.markElements, restored.live.markElements) {
					t.Fatalf("mark continuation differs at next action %d", tick)
				}
				if tick == 29 && (len(restored.live.markElements) != 1 || restored.live.markElements[0].Count != 1) {
					t.Fatal("last counter phase missing")
				}
				if tick >= 30 && len(restored.live.markElements) != 0 {
					t.Fatal("mark did not expire at zero")
				}
			}
		})
	}
}

func TestTargetMarkSavedInputDefaultsAndBounds(t *testing.T) {
	legacy := []SnapshotHealBurst{{At: image.Pt(4, 4), Picture: 20, Tile: 1, Age: 7}}
	mw := sbWorld(t)
	mw.restoreHealBursts(legacy)
	mw.restoreTargetMarks(nil)
	if len(mw.healBursts) != 1 || mw.healBursts[0].age != 7 || len(mw.markElements) != 0 || len(mw.healSpriteDraws()) == 0 {
		t.Fatal("old saved Heals or absent target-mark default changed")
	}
	for _, packed := range []uint32{0, 20 << 16, 20<<16 | 33, 21<<16 | 32} {
		if validateTargetMarks([]SnapshotTargetMark{{Entity: 0, Element: packed}}) == nil {
			t.Fatalf("accepted invalid packed mark %#x", packed)
		}
	}
	valid := SnapshotTargetMark{Entity: 0, Element: 20<<16 | 32}
	if err := validateTargetMarks([]SnapshotTargetMark{valid}); err != nil {
		t.Fatal("actor zero refused", err)
	}
	if validateTargetMarks([]SnapshotTargetMark{valid, valid}) == nil {
		t.Fatal("duplicate target/kind accepted")
	}
}

func TestProjectileTargetMarkActorZeroUsesTheSameBuilder(t *testing.T) {
	target := sim.Entity{ID: 0, X: 3, Y: 3, HP: 10, MaxHP: 10,
		SourceBinding: sim.SourceBinding{Class: 1, ArchiveIndex: 1, Identity: 101, RuntimeID: 91},
		ActorLoad:     sim.ActorLoad{Present: true, Source: sim.SourceActor{Class: 1}}}
	w, err := sim.NewWorld(1, sim.Bounds{Width: 12, Height: 12}, sim.ModeCanonical, nil,
		[]sim.Entity{target, {ID: 1, X: 5, Y: 3, HP: 10, MaxHP: 10}})
	if err != nil {
		t.Fatal(err)
	}
	mw := sbWorld(t)
	mw.world, mw.projectiles.Homing = w, map[int]bool{20: true}
	marks := mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 0, Spell: 6, ToX: 3, ToY: 3}})
	mw.advanceEffectMarks(marks...)
	if len(mw.markElements) != 1 || mw.markElements[0].Entity != 0 || len(mw.markDraws(0, 1)) == 0 || len(mw.healBursts) != 0 {
		t.Fatal("actor zero was treated as a point or missing target", mw.markElements)
	}
	r := mw.residue()
	mw.restoreTargetMarks(r.TargetMarks)
	if len(r.TargetMarks) != 1 || r.TargetMarks[0].Entity != 0 || len(mw.markDraws(0, 1)) == 0 {
		t.Fatal("actor-zero mark lost through current visual residue")
	}
}

func TestTargetMarkSeedKeepsVisualIdentityAfterRemint(t *testing.T) {
	native := &mapWorld{world: visualIdentityWorld(t, []sim.EntityID{81, 84}, 84)}
	native.advanceEffectMarks(sim.ProjectileMark{Target: 84, Picture: 20})
	cold := &mapWorld{world: visualIdentityWorld(t, []sim.EntityID{0, 5}, 5)}
	cold.restoreVisualIdentities([]SnapshotVisualIdentity{{Entity: 0, Label: 81}, {Entity: 5, Label: 84}}, 85)
	cold.advanceEffectMarks(sim.ProjectileMark{Target: 5, Picture: 20})
	if len(native.markElements) != 1 || len(cold.markElements) != 1 || native.markElements[0].Seed != cold.markElements[0].Seed {
		t.Fatal("remint treated actor zero as the shower's source or changed its target seed")
	}
}

func TestProjectileTargetMarkFreshRiderAndRetainedShareOneElement(t *testing.T) {
	mw := sbWorld(t)
	mw.world = visualIdentityWorld(t, []sim.EntityID{1, 2}, 2)
	marks := mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: 6, Rider: true, Weapon: true}})
	if len(mw.world.SavedProjectiles().Items) != 0 || len(marks) != 1 || len(mw.healBursts) != 0 {
		t.Fatal("fresh weapon rider lost its semantic mark or built a parallel shower", marks)
	}
	marks = append(marks, sim.ProjectileMark{Target: 2, Picture: 20})
	mw.advanceEffectMarks(marks...)
	if len(mw.markElements) != 1 || mw.markElements[0].Count != 32 || len(mw.markDraws(2, 1)) == 0 {
		t.Fatal("fresh and retained observations did not share one element", mw.markElements)
	}
	seed := mw.markElements[0].Seed
	mw.advanceEffectMarks(marks[1], marks[0])
	if len(mw.markElements) != 1 || mw.markElements[0].Count != 32 || mw.markElements[0].Seed != seed {
		t.Fatal("observation order changed the birth seed or duplicated its lifetime")
	}
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: 6, Rider: true, AtCell: true}})
	if len(mw.healBursts) != 1 {
		t.Fatal("targetless fixed-anchor path was not preserved")
	}
}

func TestProjectileTargetMarksRefreshKindsAndClearStaleBindings(t *testing.T) {
	mw := sbWorld(t)
	mw.world = visualIdentityWorld(t, []sim.EntityID{1, 2}, 2)
	mw.projectiles.Homing = map[int]bool{20: true, 30: true}
	const target = sim.EntityID(2)
	for _, spell := range []uint16{6, 6, 11} {
		marks := mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: target, Spell: spell, ToX: 2, ToY: 1}})
		mw.advanceEffectMarks(marks...)
	}
	if len(mw.healBursts) != 0 || len(mw.markElements) != 2 || mw.markElements[0].Count != 31 || mw.markElements[1].Count != 32 {
		t.Fatal("fresh casts did not share by-kind countdown owner", mw.markElements, mw.healBursts)
	}
	before := mw.world.Hash()
	mw.advanceEffectMarks(sim.ProjectileMark{Target: target, Picture: 20}, sim.ProjectileMark{Target: target, Picture: 20})
	if len(mw.markElements) != 2 || mw.markElements[0].Count != 32 || mw.markElements[1].Count != 31 || mw.world.Hash() != before {
		t.Fatal("same-kind refresh duplicated or coupled distinct kinds", mw.markElements)
	}
	wide := mw.markDraws(target, 3)
	if len(wide) == 0 || wide[0].Mark.Record != terrain.MarkKind(6) {
		t.Fatal("shared target shower builder absent")
	}
	// A replacement source record cannot inherit a previous actor's mark.
	entities := mw.world.Entities()
	for i := range entities {
		if entities[i].ID == target {
			entities[i].SourceBinding = sim.SourceBinding{Class: 1, ArchiveIndex: 1, Identity: 101, RuntimeID: 91}
			entities[i].ActorLoad = sim.ActorLoad{Present: true, Source: sim.SourceActor{Class: 1}}
		}
	}
	replacement, err := sim.NewWorld(1, mw.world.Bounds(), sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	mw.world = replacement
	if len(mw.markDraws(target, 1)) != 0 {
		t.Fatal("replacement record inherited marks before rebuild")
	}
	mw.advanceEffectMarks()
	if len(mw.markElements) != 0 {
		t.Fatal("replacement source actor inherited target marks")
	}
	mw.advanceEffectMarks(sim.ProjectileMark{Target: target, Picture: 20, Binding: entities[1].SourceBinding})
	if len(mw.markElements) != 1 {
		t.Fatal("replacement cannot open its own mark")
	}
	without, err := sim.NewWorld(1, sim.Bounds{Width: 64, Height: 64}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10}})
	if err != nil {
		t.Fatal(err)
	}
	mw.world = without
	mw.advanceEffectMarks()
	if len(mw.markElements) != 0 || len(mw.markDraws(target, 1)) != 0 {
		t.Fatal("removed target left marks")
	}
}
