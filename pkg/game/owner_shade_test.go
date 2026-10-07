package game

import (
	"fmt"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func ownerShadeFrame() *terrain.StaticFrame {
	var palette [256]color.RGBA
	palette[55] = color.RGBA{R: 1, G: 2, B: 3, A: 0xff}
	return &terrain.StaticFrame{
		Width: 1, Height: 1,
		Pixels:  []terrain.StaticPixel{{Index: 55, Opaque: true}},
		Palette: palette,
	}
}

// TestEntityDrawsAppliesOwnerShadeAtTheProductionFrameSeam is the mutation
// witness for the USE site. Removing entityDraws' ownerFrame call leaves every
// draw on base and fails the first composed-pixel assertion below.
func TestEntityDrawsAppliesOwnerShadeAtTheProductionFrameSeam(t *testing.T) {
	base := ownerShadeFrame()
	shared := &terrain.UnitClass{Frames: []*terrain.StaticFrame{base}, OwnerShaded: true}
	classPalette := &terrain.UnitClass{Frames: []*terrain.StaticFrame{base}}
	set := &terrain.UnitSet{
		Classes:          map[int32]*terrain.UnitClass{1: shared, 2: classPalette},
		HasOwnerPalettes: true,
	}
	set.OwnerPalettes[1][55] = color.RGBA{R: 11, G: 21, B: 31, A: 0xff}
	set.OwnerPalettes[2][55] = color.RGBA{R: 12, G: 22, B: 32, A: 0xff}

	w, err := sim.NewWorld(1, sim.Bounds{Width: 12, Height: 12}, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 0, X: 1, Y: 1, Class: 1, Owner: 1},
		{ID: 1, X: 2, Y: 2, Class: 1, Owner: 2},
		{ID: 2, X: 3, Y: 3, Class: 1, Owner: 17},
		{ID: 3, X: 4, Y: 4, Class: 2, Owner: 1},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	mw := &mapWorld{world: w, units: set,
		swing: make(map[sim.EntityID]int), died: make(map[sim.EntityID]int),
		ownerFrames: make(map[ownerFrameKey]*terrain.StaticFrame)}
	draws := mw.entityDraws()
	if len(draws) != 4 {
		t.Fatalf("entityDraws returned %d entries, want 4", len(draws))
	}

	for i, want := range []color.RGBA{
		{R: 11, G: 21, B: 31, A: 0xff},
		{R: 12, G: 22, B: 32, A: 0xff},
		{R: 11, G: 21, B: 31, A: 0xff},
		{R: 1, G: 2, B: 3, A: 0xff},
	} {
		got := draws[i].Frame.RGBA().RGBAAt(0, 0)
		if got != want {
			t.Errorf("draw %d composed pixel = %+v, want %+v", i, got, want)
		}
	}
	if draws[0].Frame == base || draws[1].Frame == base {
		t.Error("shared-human draws kept the base frame identity")
	}
	if draws[0].Frame != draws[2].Frame {
		t.Error("owners 1 and 17 did not share the same wrapped shade/frame identity")
	}
	if draws[0].Frame == draws[1].Frame {
		t.Error("different owner shades alias one frame identity")
	}
	if draws[3].Frame != base {
		t.Error("a class-palette unit was owner-shaded")
	}
	if len(draws[0].Frame.Pixels) == 0 || &draws[0].Frame.Pixels[0] != &base.Pixels[0] {
		t.Error("the owner-shaded frame copied indexed pixel memory")
	}
	if got := base.Palette[55]; got != (color.RGBA{R: 1, G: 2, B: 3, A: 0xff}) {
		t.Fatalf("base palette was mutated to %+v", got)
	}

	again := mw.entityDraws()
	if again[0].Frame != draws[0].Frame || again[1].Frame != draws[1].Frame {
		t.Error("a repeated production snapshot rebuilt owner frame identities")
	}
}

func TestEntityDrawsFallsBackWhenTheOwnerPaletteIsUnavailable(t *testing.T) {
	base := ownerShadeFrame()
	class := &terrain.UnitClass{Frames: []*terrain.StaticFrame{base}, OwnerShaded: true}
	set := &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{1: class}}
	w, err := sim.NewWorld(1, sim.Bounds{Width: 4, Height: 4}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 0, X: 1, Y: 1, Class: 1, Owner: 1}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	draws := seamDraws(w, set)
	if len(draws) != 1 || draws[0].Frame != base {
		t.Fatalf("fallback draws = %+v, want the exact base frame %p", draws, base)
	}
}

func stoneGrey(p color.RGBA) color.RGBA {
	grey := uint8((77*uint32(p.R) + 150*uint32(p.G) + 29*uint32(p.B) + 128) >> 8)
	return color.RGBA{R: grey, G: grey, B: grey, A: p.A}
}

func stoneOwnerClass() *terrain.UnitClass {
	frames := make([]*terrain.StaticFrame, 6)
	for i := range frames {
		frames[i] = ownerShadeFrame()
		frames[i].Palette[55] = color.RGBA{R: 200, G: 100, B: 20, A: 0xff}
	}
	c := &terrain.UnitClass{
		Frames: frames,
		Anim: terrain.UnitAnim{
			S: 1, D: 1,
			AttackBase: 1, AttackSlot: 1, AttackTrack: []int{0}, AttackOK: true,
			DyingBase: 2, DyingSlot: 1, TailBase: 3, BoneSlot: 3,
		},
		OwnerShaded: true,
	}
	c.Corpse = c
	return c
}

func stoneOwnerSet(class *terrain.UnitClass) *terrain.UnitSet {
	set := &terrain.UnitSet{
		Classes:          map[int32]*terrain.UnitClass{1: class},
		HasOwnerPalettes: true,
	}
	set.OwnerPalettes[1][55] = color.RGBA{R: 250, A: 0xff}
	set.OwnerPalettes[2][55] = color.RGBA{G: 250, A: 0xff}
	return set
}

// stoneOwnerWorld casts spell 20 through the real command and projectile
// cadence. The target may already hold an attack order, which Stone freezes
// rather than clearing; that makes the attack-frame arm observable after the
// effect lands. A 4096-tick duration keeps the effect attached through every
// canonical damage threshold this file asks immediately afterwards.
func stoneOwnerWorld(t *testing.T, set *terrain.UnitSet, owner uint32, attacking bool) (*sim.World, *mapWorld) {
	t.Helper()
	caster := sim.Entity{ID: 1, X: 10, Y: 10, HP: 100, MaxHP: 100, Owner: 1,
		Mind: 60, Mana: 900, MaxMana: 900, KnownSpells: 1 << 20, ScanRange: 19,
		Speed: 10, AttackCharge: 4, AttackRelax: 2}
	victim := sim.Entity{ID: 2, X: 12, Y: 10, HP: 100, MaxHP: 100, Class: 1,
		Owner: owner, Facing: 128, Speed: 10, AttackCharge: 1000,
		AttackTarget: 3, HasAttackTarget: attacking}
	otherOwner := uint32(1)
	if owner == 1 {
		otherOwner = 2
	}
	other := sim.Entity{ID: 3, X: 12, Y: 11, HP: 100, MaxHP: 100, Owner: otherOwner, Speed: 10}
	rule := sim.SpellRule{ID: 20, ManaCost: 1, School: 4, MaxRange: 8, TargetsUnit: true,
		SpellDuration: 4096, EffectKind: sim.EffectAbsorption, EffectMode: sim.EffectDuration,
		EffectMagnitude: 5}
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 64, Height: 64}, sim.ModeCanonical,
		nil, []sim.Entity{caster, victim, other}, nil, []sim.SpellRule{rule})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	emCast(t, w, 20)
	return w, &mapWorld{world: w, units: set, projectiles: emSheets(),
		swing: make(map[sim.EntityID]int), died: make(map[sim.EntityID]int),
		ownerFrames: make(map[ownerFrameKey]*terrain.StaticFrame)}
}

func stoneOwnerTarget(t *testing.T, mw *mapWorld) ui.MapEntity {
	t.Helper()
	for _, draw := range mw.entityDraws() {
		if draw.ID == 2 {
			return draw
		}
	}
	t.Fatal("entityDraws omitted the Stone target")
	return ui.MapEntity{}
}

func stoneOwnerEntity(t *testing.T, w *sim.World) sim.Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID == 2 {
			return e
		}
	}
	t.Fatal("world omitted the Stone target")
	return sim.Entity{}
}

// TestStoneOwnerClassesFollowTheResearchedDecayGate closes the repeated
// Stone+owner P class over its whole production population. MAGIC-STONEDRAW-084
// gives both Stone draw overrides the <=2 gate; REG-UNITS-050 identifies that
// field as the corpse stage. Live, attack, fallen and first-bone frames must
// therefore reach one neutral base path. The two later bone frames must leave
// Stone presentation and resume the ordinary owner path even though the effect
// itself is still attached.
//
// Every state starts with a real spell-20 cast and reaches death through the
// canonical damage command. Owners 1 and 2 have deliberately different
// luminance, so a broad Stone guard cannot accidentally pass on equal colours.
// The late-bone assertions cover both canonical pixel walks and the stable
// frame cache; reverting to the old effect-only guard fails before those reads.
func TestStoneOwnerClassesFollowTheResearchedDecayGate(t *testing.T) {
	class := stoneOwnerClass()
	set := stoneOwnerSet(class)
	wantNeutral := color.RGBA{R: 121, G: 121, B: 121, A: 0xff}
	cases := []struct {
		name      string
		attacking bool
		damage    int32
		stage     sim.DecayStage
		frame     int
		stone     bool
	}{
		{name: "live", stage: sim.DecayNone, frame: 0, stone: true},
		{name: "attack", attacking: true, stage: sim.DecayNone, frame: 1, stone: true},
		{name: "fallen", damage: 101, stage: sim.DecayFallen, frame: 2, stone: true},
		{name: "first bone", damage: 110, stage: sim.DecayBones, frame: 3, stone: true},
		{name: "second bone", damage: 120, stage: 3, frame: 4},
		{name: "third bone", damage: 140, stage: 4, frame: 5},
	}

	for _, owner := range []uint32{1, 2} {
		for _, tc := range cases {
			// The owner-2 target is hostile to the caster and its pre-existing
			// attack order is cleared by that cast path before presentation sees
			// it. Owner 1 exercises the attack-frame arm; both owners exercise
			// every corpse-stage boundary and the two different shade tables.
			if owner == 2 && tc.attacking {
				continue
			}
			t.Run(fmt.Sprintf("owner_%d/%s", owner, tc.name), func(t *testing.T) {
				w, mw := stoneOwnerWorld(t, set, owner, tc.attacking)
				if tc.damage > 0 {
					sim.Step(w, []sim.Command{{Kind: sim.KindDamage, Entity: 2, X: tc.damage}})
				}
				entity := stoneOwnerEntity(t, w)
				if entity.Decay != tc.stage {
					t.Fatalf("canonical damage put target at stage %d, want %d", entity.Decay, tc.stage)
				}
				if !w.HasEffectSpell(2, 20) {
					t.Fatal("spell-20 effect expired or was removed before the draw")
				}
				if tc.attacking && !entity.HasAttackTarget {
					t.Fatal("Stone attachment cleared the fixture's admitted attack order")
				}

				draw := stoneOwnerTarget(t, mw)
				base := class.Frames[tc.frame]
				if draw.Art != class {
					t.Fatalf("stage %d selected art %p, want fixture class %p", tc.stage, draw.Art, class)
				}
				if draw.Stone != tc.stone {
					t.Fatalf("stage %d crossed with Stone=%t, want %t", tc.stage, draw.Stone, tc.stone)
				}

				if tc.stone {
					if draw.Frame != base {
						t.Fatalf("stage %d Stone owner %d crossed with frame %p, want base %p",
							tc.stage, owner, draw.Frame, base)
					}
					if len(mw.ownerFrames) != 0 {
						t.Fatalf("stage %d Stone owner %d populated %d owner-frame cache entries",
							tc.stage, owner, len(mw.ownerFrames))
					}
					if got := stoneGrey(draw.Frame.RGBA().RGBAAt(0, 0)); got != wantNeutral {
						t.Fatalf("stage %d Stone owner %d final grey = %+v, want %+v",
							tc.stage, owner, got, wantNeutral)
					}
					return
				}

				if draw.Frame == base {
					t.Fatalf("late-bone stage %d kept the base frame under owner %d", tc.stage, owner)
				}
				want := set.OwnerPalettes[owner&0x0f][55]
				if draw.Frame.Palette != set.OwnerPalettes[owner&0x0f] {
					t.Fatalf("late-bone stage %d did not receive owner %d's complete table", tc.stage, owner)
				}
				if got := draw.Frame.RGBA().RGBAAt(0, 0); got != want {
					t.Fatalf("late-bone stage %d unlit pixel = %+v, want %+v", tc.stage, got, want)
				}
				if got := draw.Frame.RGBALit([3]uint8{}, 8).RGBAAt(0, 0); got != want {
					t.Fatalf("late-bone stage %d lit pixel = %+v, want %+v", tc.stage, got, want)
				}
				if len(mw.ownerFrames) != 1 {
					t.Fatalf("late-bone stage %d populated %d owner-frame cache entries, want 1",
						tc.stage, len(mw.ownerFrames))
				}
				again := stoneOwnerTarget(t, mw)
				if again.Frame != draw.Frame {
					t.Fatalf("late-bone stage %d rebuilt owner frame %p as %p",
						tc.stage, draw.Frame, again.Frame)
				}
			})
		}
	}
}

// TestStoneDecayGateUsesTheSelectedBodyClass covers the class-choice delta the
// central gate must not disturb. A hero body's selected corpse remains owner
// eligible; a non-owner corpse and a tier-palette corpse remain excluded. The
// stage-3 cases all keep an attached Stone effect, so an effect-only bypass
// would make every expected ordinary path fail.
func TestStoneDecayGateUsesTheSelectedBodyClass(t *testing.T) {
	t.Run("hero live and owner-shaded corpse", func(t *testing.T) {
		record := stoneOwnerClass()
		hero := stoneOwnerClass()
		heroCorpse := stoneOwnerClass()
		hero.Corpse = heroCorpse
		set := stoneOwnerSet(record)
		w, mw := stoneOwnerWorld(t, set, 2, false)
		mw.art = map[sim.EntityID]*terrain.UnitClass{2: hero}

		live := stoneOwnerTarget(t, mw)
		if live.Art != hero || !live.Stone || live.Frame != hero.Frames[0] {
			t.Fatalf("Stone hero live draw = art %p Stone=%t frame %p, want hero %p neutral frame %p",
				live.Art, live.Stone, live.Frame, hero, hero.Frames[0])
		}

		sim.Step(w, []sim.Command{{Kind: sim.KindDamage, Entity: 2, X: 120}})
		if !w.HasEffectSpell(2, 20) {
			t.Fatal("hero corpse lost the attached Stone effect before stage 3")
		}
		late := stoneOwnerTarget(t, mw)
		if late.Art != heroCorpse || late.Stone {
			t.Fatalf("late hero corpse = art %p Stone=%t, want corpse %p ordinary", late.Art, late.Stone, heroCorpse)
		}
		if late.Frame == heroCorpse.Frames[4] || late.Frame.Palette != set.OwnerPalettes[2] {
			t.Fatal("late hero corpse did not take its selected class's owner shade")
		}
	})

	t.Run("non-owner corpse", func(t *testing.T) {
		live := stoneOwnerClass()
		corpse := stoneOwnerClass()
		corpse.OwnerShaded = false
		live.Corpse = corpse
		set := stoneOwnerSet(live)
		w, mw := stoneOwnerWorld(t, set, 2, false)
		sim.Step(w, []sim.Command{{Kind: sim.KindDamage, Entity: 2, X: 120}})
		if !w.HasEffectSpell(2, 20) {
			t.Fatal("non-owner corpse lost the attached Stone effect before stage 3")
		}

		draw := stoneOwnerTarget(t, mw)
		if draw.Art != corpse || draw.Stone || draw.Frame != corpse.Frames[4] {
			t.Fatalf("non-owner corpse draw = art %p Stone=%t frame %p, want art %p base %p",
				draw.Art, draw.Stone, draw.Frame, corpse, corpse.Frames[4])
		}
		if len(mw.ownerFrames) != 0 {
			t.Fatalf("non-owner corpse populated %d owner-frame cache entries", len(mw.ownerFrames))
		}
	})

	t.Run("tier-palette corpse", func(t *testing.T) {
		class := stoneOwnerClass()
		class.OwnerShaded = false
		tier := stoneOwnerClass().Frames
		for _, frame := range tier {
			frame.Palette[55] = color.RGBA{B: 210, A: 0xff}
		}
		class.Tiers = [][]*terrain.StaticFrame{tier}
		class.Corpse = class
		set := stoneOwnerSet(class)
		w, mw := stoneOwnerWorld(t, set, 2, false)
		mw.tiers = map[sim.EntityID]int{2: 1}
		sim.Step(w, []sim.Command{{Kind: sim.KindDamage, Entity: 2, X: 120}})
		if !w.HasEffectSpell(2, 20) {
			t.Fatal("tier-palette corpse lost the attached Stone effect before stage 3")
		}

		draw := stoneOwnerTarget(t, mw)
		if draw.Art != class || draw.Stone || draw.Frame != tier[4] {
			t.Fatalf("tier corpse draw = art %p Stone=%t frame %p, want class %p tier %p",
				draw.Art, draw.Stone, draw.Frame, class, tier[4])
		}
		if len(mw.ownerFrames) != 0 {
			t.Fatalf("tier-palette corpse populated %d owner-frame cache entries", len(mw.ownerFrames))
		}
	})
}
