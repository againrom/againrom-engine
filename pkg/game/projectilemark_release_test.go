package game

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

func targetMarkInstalledFront(t *testing.T) *FrontEnd {
	t.Helper()
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: installed target-mark art witness")
	}
	graphics, err := OpenContainers(filepath.Join(root, GraphicsArchive))
	if err != nil {
		t.Fatal(err)
	}
	pictures, err := LoadProjectiles(graphics)
	if err != nil {
		t.Fatal(err)
	}
	units, err := LoadUnits(graphics)
	if err != nil {
		t.Fatal(err)
	}
	var actor *terrain.UnitClass
	first := int32(1<<31 - 1)
	for id, class := range units.Classes {
		if len(class.Frames) != 0 && id < first {
			actor, first = class, id
		}
	}
	if actor == nil {
		t.Fatal("installed unit art absent")
	}
	// The synthetic World uses class zero; the art witness supplies one loaded
	// actor sheet there without making a claim about installed map reachability.
	units.Classes[0] = actor
	f := targetMarkFront(t)
	f.Projectiles, f.Units = pictures, units
	return f
}

func TestReleaseRetainedProjectileTargetMarksInstalledArt(t *testing.T) {
	counterSilentDevice(t)
	for _, picture := range []int32{20, 30} {
		t.Run(fmt.Sprint(picture), func(t *testing.T) {
			f := targetMarkInstalledFront(t)
			app := f.App("installed target-mark art")
			app.Layout(1024, 768)
			t.Cleanup(app.StopAudio)
			if err := app.OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
			caster := poolEntity(t, f.live.world, 91).ID
			target := poolEntity(t, f.live.world, 92)
			cx, cy, _ := f.live.world.ProjectilePoint(caster)
			tx, ty, _ := f.live.world.ProjectilePoint(target.ID)
			if _, ok := f.live.world.ReleaseCast(sim.CastRecord{Caster: caster, Picture: picture,
				X: cx, Y: cy, AimX: tx, AimY: ty, Target: target.ID, HasTarget: true, Segments: 2}); !ok {
				t.Fatal("controlled positive-count release refused")
			}
			cold := targetMarkCold(t, f, targetMarkInstalledFront)
			var report sim.Report
			cold.live.tickStep(nil, true, func(r sim.Report) { report = r })
			if len(report.Casts) != 0 || len(cold.live.healBursts) != 0 || len(cold.live.markElements) != 1 {
				t.Fatal("retained branch did not independently create target mark", report.Casts, cold.live.markElements)
			}
			cold.live.tickStep(nil, true)
			if len(cold.live.world.SavedProjectiles().Items) != 0 {
				t.Fatal("projectile must retire before art witness")
			}
			e, ok := cold.live.entity(target.ID)
			if !ok {
				t.Fatal("target lost")
			}
			marks := cold.live.markDraws(target.ID, cold.live.effectTileSize(target.ID))
			if len(marks) == 0 {
				t.Fatal("installed target-mark sheet/records absent")
			}
			cold.live.view.SetFogReveal(true)
			cold.live.view.Camera().SetZoom(1)
			cold.live.view.Camera().CenterOn(float64(int(e.X)*terrain.CellSize+terrain.CellSize/2),
				float64(int(e.Y)*terrain.CellSize+terrain.CellSize/2))
			draws, err := cold.live.view.HeadlessArtDraws()
			if err != nil {
				t.Fatal(err)
			}
			for _, mark := range marks {
				want := mark.Sheet.Frame(mark.Mark.Phase)
				found := false
				for _, d := range draws {
					if d.Effect == want {
						found = true
						break
					}
				}
				if !found {
					t.Fatal("target mark missing from actual actor art submission", mark.Mark)
				}
			}
			t.Logf("picture%d: ordinary SAV/cold-App LOAD, Casts0 Heals0, retired Prj, %d installed target-mark records submitted", picture, len(marks))
		})
	}
}
