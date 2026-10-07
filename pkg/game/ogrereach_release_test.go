package game

import (
	"os"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const (
	reachWarrior = sim.EntityID(86)
	reachOgre    = sim.EntityID(42)
)

// reachStep advances the real World one tick through the App, answering the
// mission's own notices and dialogue the way a player does.
func reachStep(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	before := f.live.world.Tick()
	for k := 0; k < 12 && f.live.world.Tick() == before; k++ {
		if _, _, up := f.LiveNotice(); up {
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		}
		if app.Screen() != ui.ScreenMap {
			if err := app.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if f.live.world.Tick() == before {
		t.Fatal("the App did not advance the real World clock", app.Screen())
	}
}

// reachGap is the number of cells between the nearest cells of two square
// bodies, 1 for bodies that touch, counted by listing cells.
func reachGap(a sim.Entity, as int32, b sim.Entity, bs int32) int32 {
	best := int32(1 << 30)
	for i := range as {
		for j := range as {
			for k := range bs {
				for l := range bs {
					dx, dy := a.X+i-(b.X+k), a.Y+j-(b.Y+l)
					best = min(best, max(max(dx, -dx), max(dy, -dy)))
				}
			}
		}
	}
	return best
}

// An Ogre (a body of 2 cells) of mission 60 meets the party's warrior, who walks
// to a cell six from the Ogre on one of its four sides. Every blow the Ogre lands
// on him lands with the two bodies touching, and a warrior who stands when it
// lands answers it. The Ogre used to strike a unit one cell past the edge of its
// body on its west and north sides.
//
// The west side holds rocks the Ogre cannot walk between, so the warrior there
// is never reached and the witness asserts that no blow lands across them.
func TestReleaseOgreStrikesOnlyWhatItsBodyTouches(t *testing.T) {
	path := os.Getenv("AGAINROM_OGRES2_SAV")
	if path == "" {
		t.Skip("AGAINROM_OGRES2_SAV is not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	answeredSides := 0
	for _, side := range []struct {
		name   string
		dx, dy int32
	}{{"west", -6, 0}, {"north", 0, -6}, {"east", 6, 0}, {"south", 0, 6}} {
		t.Run(side.name, func(t *testing.T) {
			f := shopOrderFront(t)
			app, _ := openOriginalSAVApp(t, f, raw, "ogres2.sav")
			t.Cleanup(app.StopAudio)
			w, _ := f.LiveWorld()
			ogre, _ := w.Entity(reachOgre)
			if ogre.TokenSize != 2 || ogre.Reach != 1 {
				t.Fatal("the Ogre is not a two-cell body of reach 1", ogre.TokenSize, ogre.Reach)
			}
			// The mission's other Ogres leave the field so that every blow the
			// warrior takes is this Ogre's.
			for _, other := range w.Entities() {
				if other.TypeID == ogre.TypeID && other.ID != ogre.ID {
					f.live.pending = append(f.live.pending, sim.Kill(other.ID))
				}
			}
			f.live.enqueue(uint32(reachWarrior), int(ogre.X+side.dx), int(ogre.Y+side.dy))
			warriorHP := int32(0)
			struck, standing := 0, false
			for tick := range 3000 {
				reachStep(t, f, app)
				o, _ := w.Entity(reachOgre)
				u, _ := w.Entity(reachWarrior)
				if warriorHP == 0 {
					warriorHP = u.HP
				}
				if u.HP < warriorHP {
					warriorHP = u.HP
					if gap := reachGap(o, 2, u, 1); gap != 1 {
						t.Fatalf("the Ogre at (%d,%d) struck the warrior at (%d,%d), %d cells from its body", o.X, o.Y, u.X, u.Y, gap)
					}
					if struck == 0 {
						struck, standing = tick, !u.HasTarget
					}
				}
				if struck != 0 && standing && u.HasAttackTarget && u.AttackTarget == reachOgre {
					answeredSides++
					return
				}
				if struck != 0 && tick-struck > 90 {
					break
				}
			}
			if struck != 0 && standing {
				t.Fatalf("the warrior standing at the Ogre's blow of tick %d did not answer within 90 ticks", struck)
			}
		})
	}
	if answeredSides == 0 {
		t.Fatal("no side showed a standing warrior answering a blow")
	}
}
