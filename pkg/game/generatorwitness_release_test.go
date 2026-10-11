package game

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/ui"
)

// generatorWitnessFrame is one composed generator frame and the hover box
// over it, if one shows.
type generatorWitnessFrame struct {
	name    string
	frame   *image.RGBA
	tooltip ui.HeadlessTooltipState
}

// generatorWitnessHover rests the pointer on p long enough for the hover box
// to show and composes the frame.
func generatorWitnessHover(t *testing.T, app *ui.App, p image.Point) (*image.RGBA, ui.HeadlessTooltipState) {
	t.Helper()
	if err := app.HeadlessPointer("move", p.X, p.Y); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	frame, _, err := app.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	state, _ := app.HeadlessTooltip()
	return frame, state
}

// generatorWitnessFrames drives the install's generator through one live
// App: a difficulty level hovered, the name cleared and typed, the detail
// page at rest, and the detail page with a weapon class hovered. It answers
// the frames and the generator they show.
func generatorWitnessFrames(t *testing.T, f *FrontEnd) ([]generatorWitnessFrame, *ui.Chargen) {
	t.Helper()
	f.Options = OptionsStore{}
	f.MusicPlayer = &secondTestMusic{}
	f.SetDeterministicFrames(true)
	app := f.App("generator witness")
	app.Layout(640, 480)
	var c *ui.Chargen
	app.SetNewGameChargen(func() *ui.ChargenEntry {
		c = ui.NewChargen(f.ChargenSetup())
		return &ui.ChargenEntry{Model: c, Begin: f.NewGameBegin(f.Base().Profile.Mission())}
	})
	if err := app.HeadlessActivate("new game"); err != nil || app.Screen() != ui.ScreenChargen {
		t.Fatalf("NEW GAME: screen %s, error %v", app.Screen(), err)
	}
	l := f.generator()
	pre := l.PreCreate
	masks := []byte{byte(pre.Levels[0].Mask), byte(pre.Forward.Mask)}
	at := chargenTraceMaskPoints(t, f, pre.Mask.Key, image.Point{}, masks)
	var out []generatorWitnessFrame
	frame, state := generatorWitnessHover(t, app, at[masks[0]])
	out = append(out, generatorWitnessFrame{"precreate-level-hover", frame, state})
	for range pre.Name.Cap {
		if err := app.HeadlessType("", true); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessType("Aldor", false); err != nil {
		t.Fatal(err)
	}
	frame, state = generatorWitnessHover(t, app, image.Pt(320, 300))
	out = append(out, generatorWitnessFrame{"precreate-name", frame, state})
	secondGeneratorClick(t, app, at[masks[1]])
	if secondGeneratorStage(app) != ui.ChargenStageDetailed {
		t.Fatal("OK did not open the detail page")
	}
	frame, state = generatorWitnessHover(t, app, image.Pt(320, 470))
	out = append(out, generatorWitnessFrame{"detail", frame, state})
	class := l.Detail.Classes[0]
	axe := byte(class.Skills[1].Mask)
	skill := chargenTraceMaskPoints(t, f, class.Mask.Key, l.Detail.ColumnRect.Rectangle().Min, []byte{axe})
	frame, state = generatorWitnessHover(t, app, skill[axe])
	out = append(out, generatorWitnessFrame{"detail-skill-hover", frame, state})
	return out, c
}

// The generator's owner witness: four frames of the install's own
// generator through one live App, each hashed, and written as PNGs under
// AGAINROM_GENERATOR_WITNESS_DIR when it is set. The hashes of the first
// game's frames are its byte-identity record.
func TestReleaseGeneratorWitnessFrames(t *testing.T) {
	f := releaseFront(t)
	if f.ChargenAssets == nil {
		t.Fatal("the install loaded no generator")
	}
	out := os.Getenv("AGAINROM_GENERATOR_WITNESS_DIR")
	tag := f.generator().Generator + "-" + f.Base().Profile.Language
	frames, _ := generatorWitnessFrames(t, f)
	for _, w := range frames {
		t.Logf("%s %s sha256 %x tooltip %v %v", tag, w.name, sha256.Sum256(w.frame.Pix), w.tooltip.Visible, w.tooltip.Bounds)
		if out == "" {
			continue
		}
		file, err := os.Create(filepath.Join(out, fmt.Sprintf("%s-%s.png", tag, w.name)))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, w.frame); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
