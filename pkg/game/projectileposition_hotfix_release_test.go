package game

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseBowProjectilePainterMatchesActorAnchor(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	bow, err := resolveWeaponForSlot(false, f.Table.Shapes, f.Table.Materials, f.Table.Weapons, data.SkillShoot)
	if err != nil {
		t.Fatal(err)
	}
	f.Carried = MissionPartyAs(false, bow, f.Bodies, f.Table)
	if err := f.App("bow target anchor").OpenMission(f.MissionOpener(41)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	hero := mw.mission.ids[0]
	shotAttackNearest(15)(t, mw, hero)
	counts := map[string]int{}
	frames := map[string]bool{}
	moving := 0
	for tick := 0; tick < 400; tick++ {
		shotAnswer(mw, hero)
		mw.tick()
		drivers := mw.world.SavedWorldEffectDrivers()
		if drivers == nil {
			continue
		}
		bolts := mw.savedProjectileDraws()
		for _, p := range mw.world.SavedProjectiles().Items {
			if p.ActionSegments != 0 || p.Picture < 1 || p.Picture > 6 {
				continue
			}
			var target sim.EntityID
			for _, d := range drivers.Projectiles {
				if d.ID == p.ID && d.HasTarget && !d.TargetDetached {
					target = d.Target
				}
			}
			e, ok := mw.entity(target)
			if !ok || !e.Alive() {
				continue
			}
			kind := "hero"
			if target == hero {
				kind = "enemy"
			}
			var actor ui.MapEntity
			for _, d := range mw.entityDraws() {
				if d.ID == uint32(target) {
					actor = d
					break
				}
			}
			if actor.Art == nil || actor.Frame == nil {
				t.Fatal("installed target has no actual sprite", target)
			}
			if actor.FinePosition && (actor.FineX != 128 || actor.FineY != 128) {
				moving++
			}
			// Independent projection from the actor's canonical point, not a projectile placement helper.
			fx, fy := 128, 128
			if x, y, present := mw.world.ActorFinePosition(target); present {
				fx, fy = int(x), int(y)
			}
			want := image.Pt((int(e.X)*256+fx)/8, (int(e.Y)*256+fy)/8)
			for _, b := range bolts {
				if b.Pos != image.Pt(int(p.X), int(p.Y)) || b.Sheet != f.Projectiles.Sheet(int(p.Picture)) {
					continue
				}
				before := mw.world.Hash()
				mw.view.SetEntities([]ui.MapEntity{actor})
				got := projectilePainterAnchor(t, mw.view, b)
				draws, err := mw.view.HeadlessArtDraws()
				if err != nil {
					t.Fatal(err)
				}
				unitFound := false
				for _, d := range draws {
					if d.Kind != "sprite" || d.Frame != actor.Frame {
						continue
					}
					ax := actor.Art.CenterX - actor.Art.Width/2 + actor.Frame.Width/2
					ay := actor.Art.CenterY - actor.Art.Height/2 + actor.Frame.Height/2
					if actor.Mirror {
						ax = actor.Frame.Width - ax
					}
					x, y := d.Geometry.Apply(float64(ax), float64(ay))
					actual := image.Pt(int(x), int(y)).Sub(actor.DamageJolt)
					if dx, dy := actual.X-want.X, actual.Y-want.Y; dx < -1 || dx > 1 || dy < -1 || dy > 1 {
						t.Fatalf("actor%d actual sprite anchor%v canonical projected point%v", target, actual, want)
					}
					unitFound = true
				}
				if !unitFound {
					t.Fatal("production painter omitted target sprite", target)
				}
				if got != want {
					t.Errorf("%s tick%d picture%d shot%d target%d fine(%d,%d): actual painter anchor%v target%v", kind, tick+1, p.Picture, p.ID, target, fx, fy, got, want)
				}
				if before != mw.world.Hash() {
					t.Fatal("production geometry read changed world hash")
				}
				counts[kind]++
				if !frames[kind] {
					writeProjectilePainterFrame(t, f, kind, draws, want, actor, b)
					frames[kind] = true
					t.Logf("%s ordinary final flight: tick%d picture%d shot%d target%d fine(%d,%d) painter%v sprite anchor%v hash%016x", kind, tick+1, p.Picture, p.ID, target, fx, fy, got, want, before)
				}
			}
		}
	}
	if counts["hero"] == 0 || counts["enemy"] == 0 {
		t.Fatal("ordinary installed release route coverage missing", counts)
	}
	t.Logf("installed final-flight painter checks: hero%d enemy%d moving-target%d", counts["hero"], counts["enemy"], moving)
}

func writeProjectilePainterFrame(t *testing.T, f *FrontEnd, kind string, draws []ui.HeadlessArtDraw, anchor image.Point, actor ui.MapEntity, bolt ui.SpellBolt) {
	t.Helper()
	root := os.Getenv("AGAINROM_PROJECTILE_POSITION_ARTIFACTS")
	if root == "" {
		return
	}
	dir := filepath.Join(root, filepath.Base(f.Archives.Root))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	origin := anchor.Sub(image.Pt(96, 128))
	canvas := image.NewRGBA(image.Rect(0, 0, 192, 192))
	for _, d := range draws {
		if !(d.Kind == "sprite" && d.Frame == actor.Frame) && !(d.Kind == "effect" && d.Effect == bolt.Sheet.Frame(bolt.Frame)) {
			continue
		}
		for y := 0; y < d.Pixels.Bounds().Dy(); y++ {
			for x := 0; x < d.Pixels.Bounds().Dx(); x++ {
				c := d.Pixels.RGBAAt(x, y)
				if c.A == 0 {
					continue
				}
				px, py := d.Geometry.Apply(float64(x), float64(y))
				at := image.Pt(int(px), int(py)).Sub(origin)
				if at.In(canvas.Rect) {
					canvas.SetRGBA(at.X, at.Y, c)
				}
			}
		}
	}
	path := filepath.Join(dir, fmt.Sprintf("%s-final-flight.png", kind))
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(out, canvas); err != nil {
		out.Close()
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}
