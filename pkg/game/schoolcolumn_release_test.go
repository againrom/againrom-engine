package game

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/formats/bmp"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func TestReleaseSchoolColumnClassTransitionInstalledPixelsAndSound(t *testing.T) {
	f := releaseFront(t)
	if f.TownSchoolArt.Value() == nil {
		t.Fatalf("school art: %v", f.TownSchoolArt.Err())
	}
	// Generate two real installed character projections. A copied fighter
	// with only Mage flipped would test the column but not the picker figure.
	fighter := f.ChargenParty(ui.ChargenResult{Name: "Column fighter", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 20, 20}})
	mage := f.ChargenParty(ui.ChargenResult{Name: "Column mage", Choices: []int{1, 1, 0}, Stats: []int{20, 20, 30, 30}})
	if len(fighter) == 0 || len(mage) == 0 || fighter[0].Mage || !mage[0].Mage {
		t.Fatal("installed chargen did not produce both school classes")
	}
	f.Carried = mapload.OwnParty([]mapload.PartyMember{fighter[0], mage[0]})
	f.Town.gold = 100000
	now := time.Unix(100, 0)
	f.TownAnimationNow = func() time.Time { return now }
	recorder := &schoolRotateRecorder{}
	f.SoundPlayer, f.SoundBank = recorder, OpenSounds(f.Archives.Root)
	if f.SoundBank == nil {
		t.Fatal("installed sound archive did not open")
	}
	// Independently read the literal claim path, outside the production named
	// sound resolver and its cache. The exact decoded waveform must be played.
	sounds, err := OpenContainers(filepath.Join(f.Archives.Root, "sfx.res"))
	if err != nil {
		t.Fatal(err)
	}
	rawSound, err := sounds.ReadFile("sfx/Town/School/Rotate.wav")
	if err != nil {
		t.Fatal(err)
	}
	wantSound, err := audio.DecodeWAV(rawSound, audio.DeviceRate)
	if err != nil || len(wantSound.PCM) == 0 {
		t.Fatalf("installed Rotate: %v", err)
	}
	s := f.TownScreen().(*townScreen)
	s.Choose(2)
	for i := 0; s.room == roomTalk && i < 64; i++ {
		s.AdvanceTownDialogue()
	}
	if s.room != roomSchool {
		t.Fatal("school entry did not finish its dialogue")
	}
	s.CloseTip()
	matched, changed := 0, 0
	var first *image.RGBA
	var magePane *image.RGBA
	checkFrame := func(frame int) {
		t.Helper()
		pixels, err := ui.ComposeTownScreen(s, "")
		if err != nil {
			t.Fatal(err)
		}
		view := s.TownSurface()
		if view.SchoolColumnFrame != frame {
			t.Fatalf("displayed %d, want %d", view.SchoolColumnFrame, frame)
		}
		// Literal path/geometry from TOWN-146, independently decoded. Neither
		// the loader cache nor the production compositor supplies the oracle.
		name := fmt.Sprintf("graphics/interface/training/column/rt%04d.bmp", frame)
		raw, err := f.Archives.Containers.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		want, err := bmp.Decode(raw)
		if err != nil || want.Width != 148 || want.Height != 208 {
			t.Fatalf("%s has wrong dimensions or decode: %v", name, err)
		}
		// The idle shine of a panel's cycling slot paints over the column picture
		// (TOWN-500); its own witness compares those pixels.
		var shine image.Rectangle
		if view.SchoolIdleShine && (frame == 0 || frame == 15) {
			shine = ui.SchoolSkillRect(frame/15, view.SchoolIdleSlot)
		}
		for y := 0; y < 208; y++ {
			for x := 0; x < 148; x++ {
				c := want.At(x, y)
				got := pixels.RGBAAt(168+x, 176+y)
				if got != (color.RGBA{R: c.R, G: c.G, B: c.B, A: 255}) && !image.Pt(168+x, 176+y).In(shine) {
					t.Fatalf("%s source(%d,%d) = %v, composed %v", name, x, y, c, got)
				}
				matched++
				if frame == 7 && first != nil && got != first.RGBAAt(168+x, 176+y) {
					changed++
				}
			}
		}
		if first == nil {
			first = pixels
		}
		if frame == 15 {
			magePane = pixels
		}
		if view.Hero.Member != s.shopMemberIndex() || view.Hero.Subject.Name != f.Carried[view.Hero.Member].Name || view.Hero.Figure == nil {
			t.Fatalf("column frame %d lost the selected character identity/figure", frame)
		}
		if frame == 0 || frame == 4 || frame == 8 || frame == 12 || frame == 15 {
			writeSchoolColumnWitness(t, f, frame, pixels)
		}
		if frame > 0 && frame < 15 {
			for y := 188; y < 312; y++ {
				for x := 188; x < 288; x++ {
					if c, ok := ui.TownSurfaceControlAt(view, image.Pt(x, y)); ok && c.Kind == ui.TownSurfaceControlCell {
						t.Fatalf("intermediate frame %d has an active skill mask at %d,%d", frame, x, y)
					}
				}
			}
		}
	}
	checkFrame(0)
	selectCorner := func(p image.Point, kind ui.TownSurfaceControlKind, member int) {
		t.Helper()
		c, ok := ui.TownSurfaceControlAt(s.TownSurface(), p)
		if !ok || c.Kind != kind {
			t.Fatalf("picker corner %v: %+v / %v", p, c, ok)
		}
		s.TownSurfaceClick(c, false)
		if s.TownSurface().Hero.Member != member || s.TownSurface().Hero.Subject.Name != f.Carried[member].Name {
			t.Fatal("picker corner did not immediately change the shown character")
		}
	}
	selectCorner(image.Pt(614, 459), ui.TownSurfaceControlNext, 1)
	// TOWN-429/430: the column does not track the transition tick 1:1. It
	// stays at the old endpoint until the mage transition reaches its own
	// measured threshold (schoolColumnThreshold), then advances one frame per
	// remaining tick. The transition media's own cycle (23 states) outlasts
	// the column's 15-step walk, so waitClass -- and school input -- stays
	// blocked for a few ticks after the column visually arrives.
	mageTicks := schoolTrainingFamilyShape[schoolMageClass].transitionLast + 1
	mageThreshold := schoolColumnThreshold(schoolMageClass)
	for tick := 1; tick <= mageTicks; tick++ {
		now = now.Add(84 * time.Millisecond)
		frame := 0
		if tick >= mageThreshold {
			if frame = tick - mageThreshold + 1; frame > 15 {
				frame = 15
			}
		}
		checkFrame(frame)
	}
	if s.schoolTrainingBusy() {
		t.Fatal("mage class transition did not settle")
	}
	// Centers are literal evidence geometry, not ui.SchoolSkillRect.
	for slot, p := range []image.Point{{274, 246}, {204, 250}, {238, 212}, {242, 285}, {240, 249}} {
		c, ok := ui.TownSurfaceControlAt(s.TownSurface(), p)
		if !ok || c.Kind != ui.TownSurfaceControlCell || c.Index != 5+slot {
			t.Fatalf("mage endpoint slot %d at %v: %+v / %v", slot, p, c, ok)
		}
	}
	if magePane == nil {
		t.Fatal("missing mage endpoint composition")
	}
	paneChanges := 0
	for y := 260; y < 430; y++ {
		for x := 490; x < 630; x++ {
			if first.RGBAAt(x, y) != magePane.RGBAAt(x, y) {
				paneChanges++
			}
		}
	}
	if paneChanges == 0 {
		t.Fatal("different-class selection retained the fighter's composed figure")
	}
	selectCorner(image.Pt(497, 459), ui.TownSurfaceControlPrevious, 0)
	// The reverse walk is not the mirror tick count: fighter's own threshold
	// and 19-state transition media place the column's final step one tick
	// after the media's own cycle wraps (measured below), unlike the mage
	// walk above where the column finishes first and waits out the media.
	fighterThreshold := schoolColumnThreshold(schoolFighterClass)
	fighterColumnTicks := fighterThreshold - 1 + 15
	for tick := 1; tick <= fighterColumnTicks; tick++ {
		now = now.Add(84 * time.Millisecond)
		frame := 15
		if tick >= fighterThreshold {
			if frame = 15 - (tick - fighterThreshold + 1); frame < 0 {
				frame = 0
			}
		}
		checkFrame(frame)
	}
	if s.schoolTrainingBusy() {
		t.Fatal("fighter class transition did not settle")
	}
	for slot, p := range []image.Point{{240, 212}, {240, 234}, {240, 262}, {240, 280}, {240, 298}} {
		c, ok := ui.TownSurfaceControlAt(s.TownSurface(), p)
		if !ok || c.Kind != ui.TownSurfaceControlCell || c.Index != slot {
			t.Fatalf("fighter endpoint slot %d at %v: %+v / %v", slot, p, c, ok)
		}
	}
	// One checkFrame(0) at entry, plus one call per settled tick in each
	// threshold-gated walk (mageTicks including the media's own tail wait,
	// fighterColumnTicks stopping at the column's own later final step).
	wantCalls := 1 + mageTicks + fighterColumnTicks
	if wantMatched := wantCalls * 148 * 208; matched != wantMatched || changed == 0 || len(recorder.samples) != 2 {
		t.Fatalf("matched=%d want=%d changed=%d sounds=%d", matched, wantMatched, changed, len(recorder.samples))
	}
	for i, sample := range recorder.samples {
		if !reflect.DeepEqual(sample, wantSound) || recorder.places[i] != (audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}) {
			t.Fatalf("rotation %d did not play the installed centered waveform", i)
		}
	}
	t.Logf("school column: 16 installed frames, %d composed endpoints/transitions, %d opaque pixels matched, middle frame differs at %d pixels over both directions; picker changes %d figure pixels; 2 exact Rotate waveforms", wantCalls, matched, changed, paneChanges)
}

func writeSchoolColumnWitness(t *testing.T, f *FrontEnd, frame int, pix *image.RGBA) {
	t.Helper()
	dir := os.Getenv("AGAINROM_SCHOOL_COLUMN_PNG")
	if dir == "" {
		return
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(f.Archives.Root, dir)
	if err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("school capture directory must be outside the install: %q", dir)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(filepath.Join(dir, fmt.Sprintf("school-column-%02d.png", frame)))
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(out, pix)
	closeErr := out.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	t.Logf("school column witness: %s", out.Name())
}
