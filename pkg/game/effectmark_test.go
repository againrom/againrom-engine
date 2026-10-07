package game

import (
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// emSheets is a projectile set answering for every mark record index this file
// exercises, so a dropped record is a dropped record and never a missing sheet.
func emSheets(pictures ...int) *terrain.EffectSet {
	frames := make([]*terrain.EffectFrame, 8)
	for i := range frames {
		frames[i] = &terrain.EffectFrame{Width: 8, Height: 8, Pixels: make([]color.RGBA, 64)}
	}
	set := &terrain.EffectSet{Sheets: map[int]*terrain.EffectSheet{}}
	for _, p := range pictures {
		set.Sheets[p] = &terrain.EffectSheet{Frames: frames, Phases: 8, RotationPhases: 1, CenterX: 4, CenterY: 4}
	}
	return set
}

// emWorld is a caster who knows spell, a victim beside him, and the row itself.
func emWorld(t *testing.T, spell uint16, rule sim.SpellRule) *sim.World {
	t.Helper()
	caster := sim.Entity{ID: 1, X: 10, Y: 10, HP: 100, MaxHP: 100, Owner: 1, TokenSize: 1,
		Mind: 60, Mana: 900, MaxMana: 900, KnownSpells: 1 << spell, ScanRange: 19,
		Speed: 10, AttackCharge: 4, AttackRelax: 2}
	victim := sim.Entity{ID: 2, X: 12, Y: 10, HP: 100, MaxHP: 100, Owner: 1, TokenSize: 1, Speed: 10}
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 64, Height: 64}, sim.ModeCanonical, nil,
		[]sim.Entity{caster, victim}, nil, []sim.SpellRule{rule})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	return w
}

func emCast(t *testing.T, w *sim.World, spell uint16) {
	t.Helper()
	sim.Step(w, []sim.Command{{Kind: sim.KindCast, Entity: 1, X: 2, Y: int32(spell)}})
	for i := 0; i < 64 && len(w.ActiveEffects()) == 0; i++ {
		sim.Step(w, nil)
	}
	if len(w.ActiveEffects()) == 0 {
		t.Fatalf("spell %d attached no effect in 64 ticks", spell)
	}
}

// protectionFireRule is the shipped shape of spell 5 as this package needs it:
// a defensive, unit-targeted, duration-mode Protection.
func protectionFireRule() sim.SpellRule {
	return sim.SpellRule{ID: 5, ManaCost: 10, School: 1, MaxRange: 8, TargetsUnit: true,
		Defensive: true, EffectKind: sim.EffectProtectionFire, EffectMode: sim.EffectDuration,
		EffectMagnitude: 20, SpellDuration: 60}
}

// TestAnAttachedEffectOpensAnElementAndACloseRemovesIt — MAGIC-MARK-060: the
// mark set is not stored. An element is opened at 0xffff for every (actor,
// kind) the simulation carries, the rebuild decrements it, and the element goes
// when the effect does.
func TestAnAttachedEffectOpensAnElementAndACloseRemovesIt(t *testing.T) {
	w := emWorld(t, 5, protectionFireRule())
	mw := &mapWorld{world: w, projectiles: emSheets(terrain.MarkProtectionFire)}

	mw.advanceEffectMarks()
	if len(mw.markElements) != 0 {
		t.Fatalf("an unmarked world opened %d elements, want none", len(mw.markElements))
	}

	emCast(t, w, 5)
	mw.advanceEffectMarks()
	if len(mw.markElements) != 1 {
		t.Fatalf("the cast opened %d elements, want 1", len(mw.markElements))
	}
	el := mw.markElements[0]
	if el.Kind != terrain.MarkProtectionFire {
		t.Errorf("element kind %#x, want %#x — 2*5 + 8", el.Kind, terrain.MarkProtectionFire)
	}
	if el.Count != markElementStart {
		t.Errorf("element opened at %d, want %d", el.Count, markElementStart)
	}

	mw.advanceEffectMarks()
	if got := mw.markElements[0].Count; got != markElementStart-1 {
		t.Errorf("after a second rebuild the countdown is %d, want %d — one decrement per rebuild",
			got, markElementStart-1)
	}

	// Run the effect out and the element goes with it.
	for i := 0; i < 20000 && len(w.ActiveEffects()) > 0; i++ {
		sim.Step(w, nil)
	}
	if len(w.ActiveEffects()) != 0 {
		t.Fatal("the effect had not expired; the close half of the test is untested")
	}
	mw.advanceEffectMarks()
	if len(mw.markElements) != 0 {
		t.Fatalf("the expired effect left %d elements, want none", len(mw.markElements))
	}
}

// TestTheRebuildTurnsAnElementIntoItsBuilderRecords — MAGIC-MARK-059: the
// record's art resolves through the projectile set by the record index, which is
// the kind itself. A kind whose picture names no sheet drops its records rather
// than drawing a placeholder.
func TestTheRebuildTurnsAnElementIntoItsBuilderRecords(t *testing.T) {
	w := emWorld(t, 5, protectionFireRule())
	emCast(t, w, 5)

	mw := &mapWorld{world: w, projectiles: emSheets(terrain.MarkProtectionFire)}
	mw.advanceEffectMarks()
	got := mw.markDraws(2, 1)
	want := terrain.EffectMarkRecords(terrain.MarkProtectionFire, mw.markElements[0].Count, 1)
	if len(got) != len(want) {
		t.Fatalf("the rebuild produced %d marks, want the builder's %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Mark != want[i] {
			t.Errorf("mark %d = %+v, want %+v", i, got[i].Mark, want[i])
		}
		if got[i].Sheet == nil {
			t.Errorf("mark %d resolved no sheet at record index %#x", i, got[i].Mark.Record)
		}
	}
	if other := mw.markDraws(1, 1); len(other) != 0 {
		t.Errorf("the caster carries %d marks, want none — the effect is on the victim", len(other))
	}

	bare := &mapWorld{world: w, projectiles: emSheets()}
	bare.advanceEffectMarks()
	if got := bare.markDraws(2, 1); len(got) != 0 {
		t.Errorf("a picture with no sheet produced %d marks, want none", len(got))
	}
}

// TestAnInvisibleActorIsNotDrawnForAParticipantThatDoesNotDetectIt —
// MAGIC-ACTOR-066: the unit draw gates the whole actor sprite on the
// `invisibility` kind and skips it when the participant's bit is clear. Here
// the entry leaves the pushed list entirely, so the actor is out of the picture
// and out of selection together.
func TestAnInvisibleActorIsNotDrawnForAParticipantThatDoesNotDetectIt(t *testing.T) {
	rule := sim.SpellRule{ID: 15, ManaCost: 10, School: 4, MaxRange: 8, TargetsUnit: true,
		Defensive: true, EffectMode: sim.EffectDuration, SpellDuration: 60}
	caster := sim.Entity{ID: 1, X: 10, Y: 10, HP: 100, MaxHP: 100, Owner: 2, TokenSize: 1,
		Mind: 60, Mana: 900, MaxMana: 900, KnownSpells: 1 << 15, ScanRange: 19,
		Speed: 10, AttackCharge: 4, AttackRelax: 2}
	watcher := sim.Entity{ID: 2, X: 12, Y: 10, HP: 100, MaxHP: 100, Owner: sim.SelfSlot,
		TokenSize: 1, Speed: 10, ScanRange: 19}
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 64, Height: 64}, sim.ModeCanonical, nil,
		[]sim.Entity{caster, watcher}, nil, []sim.SpellRule{rule})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	// The caster turns itself invisible.
	sim.Step(w, []sim.Command{{Kind: sim.KindCast, Entity: 1, X: 1, Y: 15}})
	for i := 0; i < 64 && !w.HasEffectSpell(1, 15); i++ {
		sim.Step(w, nil)
	}
	if !w.HasEffectSpell(1, 15) {
		t.Fatal("invisibility did not attach in 64 ticks")
	}
	if !w.InvisibleTo(1, sim.SelfSlot) {
		t.Fatal("the caster is visible to SelfSlot, whose only unit carries no detector radius")
	}
	if w.InvisibleTo(1, 2) {
		t.Error("the caster is invisible to its OWN participant; a player always sees his own units")
	}
	if w.InvisibleTo(2, sim.SelfSlot) {
		t.Error("an actor carrying no invisibility effect reported invisible")
	}
}

func TestOwnerVisibleInvisibilityCrossesTheSeamAsTranslucent(t *testing.T) {
	rule := sim.SpellRule{ID: 15, ManaCost: 1, School: 4, MaxRange: 8, TargetsUnit: true,
		Defensive: true, EffectMode: sim.EffectDuration, SpellDuration: 60}
	w := emWorld(t, 15, rule)
	emCast(t, w, 15)
	mw := &mapWorld{world: w, projectiles: emSheets()}
	for _, d := range mw.entityDraws() {
		if d.ID == 2 {
			if !d.Translucent {
				t.Fatal("owner-visible invisible target crossed the seam opaque")
			}
			return
		}
	}
	t.Fatal("owner-visible invisible target was removed from its own participant's draw")
}

func TestStoneCurseCrossesTheSeamAsGrayscale(t *testing.T) {
	rule := sim.SpellRule{ID: 20, ManaCost: 1, School: 4, MaxRange: 8, TargetsUnit: true,
		SpellDuration: 60, EffectKind: sim.EffectAbsorption, EffectMode: sim.EffectDuration,
		EffectMagnitude: 5}
	w := emWorld(t, 20, rule)
	emCast(t, w, 20)
	mw := &mapWorld{world: w, projectiles: emSheets()}
	for _, d := range mw.entityDraws() {
		if d.ID == 2 {
			if !d.Stone {
				t.Fatal("Stone Curse target crossed the seam without its grayscale flag")
			}
			return
		}
	}
	t.Fatal("Stone Curse target did not cross the entity draw seam")
}
