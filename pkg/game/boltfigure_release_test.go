package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// TestReleaseLightningFigureOverItsLife drives one installed Lightning cast
// through its 13 driver calls on mission 41. Every call hands the drawer one
// display stamp per stored point, on the installed lightnin sheet (5 frames),
// at the published ramp frame (MAGIC-280, MAGIC-281, ANIM-BOLTRAMP-035).
func TestReleaseLightningFigureOverItsLife(t *testing.T) {
	f, hero := launchWitness(t, true)
	mw := f.live
	sheet, chain := mw.projectiles.Sheet(data.PicturePathFirst), mw.projectiles.Sheet(data.PicturePathSecond)
	if sheet == nil || len(sheet.Frames) != 5 || chain == nil || len(chain.Frames) != 35 {
		t.Fatalf("installed bolt sheets hold %v/%v frames, want 5 and 35", sheet, chain)
	}
	b := launchObserve(t, mw, hero, launchFacings[2], launchDirections[2])
	mw.bolts = []spellBolt{b}
	mw.bolts[0].age = 0
	ents := mw.world.EntityView()
	for call := 1; len(mw.bolts) > 0; call++ {
		obj := mw.bolts[0]
		ax, ay := mw.boltDisplayPoint(castOrigin(obj.from, obj.launch), obj.from)
		bx, by := mw.boltDisplayPoint(obj.to.Mul(256), obj.to)
		stored := boltFigure(ax, ay, bx, by, 34, (&boltRNG{state: boltSeed(obj)}).next)
		var stamps int
		for _, d := range mw.boltDraws(ents) {
			if d.Sheet != sheet {
				continue
			}
			if !d.Display || d.Frame != boltRamp[call-1] || d.Pos != image.Pt(int(stored[stamps].X), int(stored[stamps].Y)) {
				t.Fatalf("call %d stamp %d: %+v, want display frame %d at %v", call, stamps, d, boltRamp[call-1], stored[stamps])
			}
			stamps++
		}
		if stamps == 0 || stamps != len(stored) {
			t.Fatalf("call %d drew %d stamps for %d stored points", call, stamps, len(stored))
		}
		t.Logf("call %2d: frame %d, %d stamps from %v to %v", call, boltRamp[call-1], stamps, stored[0], stored[len(stored)-1])
		mw.advanceBolts()
		if call > 13 {
			t.Fatal("the object outlived 13 calls")
		}
	}
}

// TestReleaseBoltFigureRenders writes PNGs of one Lightning cast at calls 1,
// 4, 7, 10 and 13 when AGAINROM_BOLT_RENDERS names a directory outside the
// install. It uses only the mission witness, the cast observation and the
// headless map frame.
func TestReleaseBoltFigureRenders(t *testing.T) {
	out := os.Getenv("AGAINROM_BOLT_RENDERS")
	if out == "" {
		t.Skip("AGAINROM_BOLT_RENDERS is not set")
	}
	f, hero := launchWitness(t, true)
	mw := f.live
	root, _ := filepath.Abs(os.Getenv("AGAINROM_ASSETS"))
	out, _ = filepath.Abs(out)
	if rel, err := filepath.Rel(root, out); err == nil && !strings.HasPrefix(rel, "..") {
		t.Fatal("render directory is inside the install")
	}
	dir := filepath.Join(out, filepath.Base(root))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	e, _ := mw.entity(hero)
	if err := mw.world.ImportOriginalActorFacings([]sim.OriginalActorFacing{{ID: hero, Facing: launchFacings[2]}}); err != nil {
		t.Fatal(err)
	}
	at := image.Pt(int(e.X)+5, int(e.Y)+2)
	mw.bolts = nil
	mw.observeCasts([]sim.CastEvent{{Caster: hero, Spell: spLightning, Owner: e.Owner, FromX: e.X, FromY: e.Y,
		ToX: int32(at.X), ToY: int32(at.Y), Facing: launchFacings[2]}})
	for call := 1; len(mw.bolts) > 0; call++ {
		if call%3 == 1 {
			mw.push()
			pix, _, err := mw.view.HeadlessMapFrame(image.Pt(int(e.X)+2, int(e.Y)+1), 320, 224)
			if err != nil {
				t.Fatal(err)
			}
			launchWritePNG(t, dir, fmt.Sprintf("call-%02d.png", call), pix)
		}
		mw.advanceBolts()
	}
}
