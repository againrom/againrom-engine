package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// TestReleaseLightningFigureOverItsLife orders an installed Lightning cast on
// mission 41 and follows normal world ticks. The bolt is drawn on exactly 13
// ticks at the ramp frames, one display stamp per stored point on the
// installed lightnin sheet (MAGIC-280, MAGIC-281, ANIM-BOLTRAMP-035).
func TestReleaseLightningFigureOverItsLife(t *testing.T) {
	f, hero := launchWitness(t, true)
	mw := f.live
	sheet, chain := mw.projectiles.Sheet(data.PicturePathFirst), mw.projectiles.Sheet(data.PicturePathSecond)
	if sheet == nil || len(sheet.Frames) != 5 || chain == nil || len(chain.Frames) != 35 {
		t.Fatalf("installed bolt sheets hold %v/%v frames, want 5 and 35", sheet, chain)
	}
	victim := launchIssue(t, mw, hero)
	var id uint16
	var frames []int
	for tick := 0; tick < 240; tick++ {
		mw.tick()
		var obj *spellBolt
		for _, p := range flightRecords(mw) {
			if p.Picture == 34 && (len(frames) == 0 || p.ID == id) {
				if paths := mw.recordPaths(p, 0); len(paths) == 1 {
					obj, id = &paths[0], p.ID
				}
			}
		}
		if obj == nil {
			if len(frames) > 0 {
				break
			}
			continue
		}
		ax, ay := mw.boltDisplayPoint(castOrigin(obj.from, obj.launch), obj.from)
		bx, by := mw.boltDisplayPoint(obj.to.Mul(256), obj.to)
		stored := boltFigure(ax, ay, bx, by, 34, (&boltRNG{state: boltSeed(*obj)}).next)
		frame, stamps := -1, 0
		for _, d := range mw.boltDraws(mw.world.EntityView()) {
			if d.Sheet != sheet {
				continue
			}
			if !d.Display || stamps >= len(stored) || (frame >= 0 && d.Frame != frame) || d.Pos != image.Pt(int(stored[stamps].X), int(stored[stamps].Y)) {
				t.Fatalf("drawn tick %d stamp %d: %+v against %d stored points", len(frames)+1, stamps, d, len(stored))
			}
			frame = d.Frame
			stamps++
		}
		if stamps == 0 || stamps != len(stored) {
			t.Fatalf("drawn tick %d: %d stamps for %d stored points", len(frames)+1, stamps, len(stored))
		}
		frames = append(frames, frame)
		t.Logf("drawn tick %2d: frame %d, %d stamps from %v to %v", len(frames), frame, stamps, stored[0], stored[len(stored)-1])
	}
	if !slices.Equal(frames, boltRamp[:]) {
		t.Fatalf("Lightning at victim %d drew frames %v over %d ticks, want %v", victim, frames, len(frames), boltRamp)
	}
	boltRenders(t, f, hero)
}

// boltRenders writes PNGs of one Lightning cast at calls 1, 4, 7, 10 and 13
// when AGAINROM_BOLT_RENDERS names a directory outside the install.
func boltRenders(t *testing.T, f *FrontEnd, hero sim.EntityID) {
	t.Helper()
	out := os.Getenv("AGAINROM_BOLT_RENDERS")
	if out == "" {
		return
	}
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
	mw.observeCasts([]sim.CastEvent{{Caster: hero, Spell: spLightning, Owner: e.Owner, FromX: e.X, FromY: e.Y,
		ToX: int32(at.X), ToY: int32(at.Y), Facing: launchFacings[2]}})
	for call := 1; len(flightRecords(mw)) > 0 && call <= 40; call++ {
		if call%3 == 1 {
			mw.push()
			pix, _, err := mw.view.HeadlessMapFrame(image.Pt(int(e.X)+2, int(e.Y)+1), 320, 224)
			if err != nil {
				t.Fatal(err)
			}
			launchWritePNG(t, dir, fmt.Sprintf("call-%02d.png", call), pix)
		}
		flightStep(mw)
	}
}
