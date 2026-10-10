package game

import (
	"fmt"
	"hash/fnv"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// The owner corpus holds one original SAV with a projectile in flight, the
// game0018.sav this test opens: its Prj266 is picture 10 at saved dir 5, three
// segments from the corpse it flies onto. LOADed cold through the App's
// original door it draws ANIM-PROJ-026's frame on the installed sheet: the
// fold (5 - 8) & 0xf = 13 is the flight's own east-south-east, and the
// sheet's halving bit mirrors it onto facing 3. An ordinary menu SAVE one
// tick into the flight and a cold LOAD of the written SAV draw the same
// frame, tick for tick, until it lands. The per-tick world hashes are logged
// as one digest, so a run on the base shows the simulation did not move. The
// unit-shot sheets 1 to 6, which a SAV restores through the same draw, hold a
// frame at every facing and phase.
func TestReleaseRestoredProjectileDrawsItsSavedFacing(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b")
	f := releaseFront(t)
	bolt := f.Projectiles.Sheet(10)
	if bolt == nil || bolt.Phases != 4 || bolt.RotationPhases == 1 || !bolt.Flip {
		t.Fatal("picture 10's installed sheet is not four phases with the halving bit")
	}
	for picture := 1; picture <= 6; picture++ {
		s := f.Projectiles.Sheet(picture)
		if s == nil {
			t.Fatalf("unit-shot picture %d has no installed sheet", picture)
		}
		for facing := range 16 {
			for phase := range s.Phases {
				if _, _, ok := terrain.SelectEffectFrame(s, facing, phase); !ok {
					t.Errorf("unit-shot picture %d has no frame at facing %d phase %d", picture, facing, phase)
				}
			}
		}
	}
	app, path := openOriginalSAVApp(t, f, raw, "game0018.sav")
	items := f.live.world.SavedProjectiles().Items
	if len(items) != 1 || items[0].ID != 266 || items[0].Picture != 10 || items[0].Dir != 5 || items[0].ActionSegments != 3 {
		t.Fatalf("the release subject changed: %+v", items)
	}
	if got := EffectFacing(int(items[0].ActionX-items[0].X), int(items[0].ActionY-items[0].Y)); got != 13 {
		t.Fatalf("Prj266 flies at sheet facing %d, not 13", got)
	}
	digest := fnv.New64a()
	var frames []string
	check := func(front *FrontEnd, when string) string {
		t.Helper()
		fmt.Fprintf(digest, "%s %d %x\n", when, front.live.world.Tick(), front.live.world.Hash())
		items := front.live.world.SavedProjectiles().Items
		// The smoke trail (ANIM-140) is drawn beside the record; it is not
		// saved, so it is compared apart (SAV-1193).
		var draws []ui.SpellBolt
		for _, d := range front.live.savedProjectileDraws() {
			if d.Sheet == front.Projectiles.Sheet(10) {
				draws = append(draws, d)
			}
		}
		if len(items) == 0 && len(draws) == 0 {
			return "landed"
		}
		if len(items) != 1 || len(draws) != 1 {
			t.Fatalf("%s: %d projectiles draw %d sheets", when, len(items), len(draws))
		}
		p, d := items[0], draws[0]
		got := fmt.Sprintf("%v/%d/%t", d.Pos, d.Frame, d.Mirror)
		if p.Dir != 5 || d.Sheet != front.Projectiles.Sheet(10) || d.Pos != image.Pt(int(p.X), int(p.Y)) || d.Frame != 4*3+int(p.Phase) || !d.Mirror {
			t.Errorf("%s: Prj266 at dir %d phase %d draws %s, want frame %d mirrored at (%d,%d)",
				when, p.Dir, p.Phase, got, 4*3+p.Phase, p.X, p.Y)
		}
		return got
	}
	frames = append(frames, check(f, "cold LOAD"))
	f.live.tick()
	before := check(f, "first tick")
	store, name, written := menuSAVE(t, f, app, OriginalStore{Dir: filepath.Dir(path)})
	if got := projectileSAVFields(t, written); got[4] != 5 {
		t.Fatalf("menu SAVE wrote Prj266 at dir %d", got[4])
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	fresh := loadLocalLegacySave(t, store, name)
	if after := check(fresh, "LOAD of the engine SAVE"); after != before {
		t.Fatalf("Prj266 drew %s before the SAVE and %s after its LOAD", before, after)
	}
	if a, b := len(f.live.shots.trail[266]), len(fresh.live.shots.trail[266]); a != 1 || b != 0 {
		t.Fatalf("Prj266 trail holds %d points before the SAVE and %d after its LOAD, want 1 and 0", a, b)
	}
	frames = append(frames, before)
	for tick := 2; tick <= 4; tick++ {
		f.live.tick()
		fresh.live.tick()
		a, b := check(f, "continued"), check(fresh, "continued after LOAD")
		if a != b || f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatalf("tick %d: %s continued, %s after the LOAD", tick, a, b)
		}
		frames = append(frames, a)
	}
	if frames[len(frames)-1] != "landed" {
		t.Fatal("Prj266 did not land", frames)
	}
	t.Logf("Prj266 draws %v; world hash digest %016x", frames, digest.Sum64())
}
