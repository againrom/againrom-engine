package game

import (
	"fmt"
	"image"
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func projectilePainterAnchor(t *testing.T, v *ui.Viewer, b ui.SpellBolt) image.Point {
	t.Helper()
	v.SetFlat(true)
	v.SetFogReveal(true)
	v.SetSpellBolts([]ui.SpellBolt{b})
	cam := v.Camera()
	cam.X, cam.Y, cam.Zoom = 0, 0, 1
	cam.ViewW, cam.ViewH = 8192, 8192
	draws, err := v.HeadlessArtDraws()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range draws {
		if d.Kind != "effect" || d.Effect != b.Sheet.Frame(b.Frame) {
			continue
		}
		x := b.Sheet.CenterX
		if b.Mirror {
			x = d.Effect.Width - x
		}
		px, py := d.Geometry.Apply(float64(x), float64(b.Sheet.CenterY))
		return image.Pt(int(px), int(py))
	}
	t.Fatal("projectile painter submitted no matching art")
	return image.Point{}
}

func TestOrdinaryBowProjectilePainterReachesTargetAnchor(t *testing.T) {
	for _, delta := range []image.Point{{3, 0}, {3, 3}, {0, 3}, {-3, 3}, {-3, 0}, {-3, -3}, {0, -3}, {3, -3}} {
		t.Run(fmt.Sprint(delta), func(t *testing.T) {
			units, set := shotArchive(t)
			target := shotVictim(2, int32(4+delta.X), int32(4+delta.Y))
			target.TokenSize = 2
			mw := shotWorld(t, units, set, nil, shotShooter(1, shotClassArcher, 4, 4, 8), target)
			mw.strike(1, 2)
			for tick := 0; tick < 64; tick++ {
				mw.tick()
				items := mw.world.SavedProjectiles().Items
				if len(items) == 0 || items[0].ActionSegments != 0 {
					continue
				}
				p := items[0]
				before := mw.world.Hash()
				b := mw.savedProjectileDraws()[0]
				got := projectilePainterAnchor(t, mw.view, b)
				want := image.Pt(int(target.X)*32+16, int(target.Y)*32+16)
				if got != want {
					t.Errorf("ordinary release landed at fine(%d,%d), painter anchor%v, target anchor%v", p.X, p.Y, got, want)
				}
				if mw.world.Hash() != before || !reflect.DeepEqual(items, mw.world.SavedProjectiles().Items) {
					t.Fatal("painting changed simulation/projectile state")
				}
				return
			}
			t.Fatal("ordinary bow release did not reach final flight tick")
		})
	}
}

func TestMovingBowTargetUsesCurrentFinePainterPoint(t *testing.T) {
	units, set := shotArchive(t)
	target := shotVictim(2, 7, 4)
	target.Speed = 1
	shooter := shotShooter(1, shotClassArcher, 1, 4, 16)
	shooter.Reach = 8
	mw := shotWorld(t, units, set, nil, shooter, target)
	mw.strike(1, 2)
	for tick := 0; len(mw.world.SavedProjectiles().Items) == 0 && tick < 32; tick++ {
		mw.tick()
	}
	mw.enqueue(2, 7, 7)
	for tick := 0; tick < 64; tick++ {
		mw.tick()
		for _, p := range mw.world.SavedProjectiles().Items {
			if p.ActionSegments != 0 {
				continue
			}
			e, _ := mw.entity(2)
			if !e.Stride.Present || e.Transit == 0 {
				t.Fatal("fixture did not land during an accepted stride")
			}
			elapsed := int(e.TransitTotal - e.Transit)
			fx := int(e.Stride.FromX)*256 + 128 + elapsed*int(e.Stride.StepX)
			fy := int(e.Stride.FromY)*256 + 128 + elapsed*int(e.Stride.StepY)
			want := image.Pt(fx/8, fy/8)
			set.Sheet(1).CenterX, set.Sheet(1).CenterY = 0, 0
			got := projectilePainterAnchor(t, mw.view, mw.savedProjectileDraws()[0])
			if got != want {
				t.Errorf("moving target fine(%d,%d), painter%v want%v", fx, fy, got, want)
			}
			return
		}
	}
	t.Fatal("moving target received no ordinary arrow")
}

func TestCarriedProjectilePainterProjectsLowFineCoordinates(t *testing.T) {
	units, set := shotArchive(t)
	mw := shotWorld(t, units, set, nil, shotShooter(1, shotClassArcher, 1, 4, 8), shotVictim(2, 4, 4))
	if err := mw.world.ImportOriginalProjectiles(sim.SavedProjectiles{FreeIndex: 2, IDs: []uint16{1}, Items: []sim.SavedProjectile{{ID: 1, X: 43, Y: 77, Picture: 1, Action: 1, ActionSegments: 2, ActionX: 500, ActionY: 500}}}); err != nil {
		t.Fatal(err)
	}
	if err := mw.world.ImportOriginalWorldEffectDrivers(&sim.SavedWorldEffects{Projectiles: []sim.SavedProjectileDriver{{ID: 1, Phases: 1}}}); err != nil {
		t.Fatal(err)
	}
	set.Sheet(1).CenterX, set.Sheet(1).CenterY = 0, 0
	got := projectilePainterAnchor(t, mw.view, mw.savedProjectileDraws()[0])
	if want := image.Pt(5, 9); got != want {
		t.Errorf("absolute fine(43,77) painter%v want%v", got, want)
	}
}
