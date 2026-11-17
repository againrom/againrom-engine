package game

import (
	"fmt"
	"image"
	"os"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// frameTimingFonts installs the install's fonts for the editor fixture, so the
// timings spend what the shipped text spends.
func frameTimingFonts(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		return
	}
	a, err := OpenArchives(root)
	if err != nil {
		t.Fatal(err)
	}
	font, err := LoadFont(a.Containers, DefaultFont, FontShades)
	if err != nil {
		t.Fatal(err)
	}
	font.Selector = LanguageSelector(a.Containers)
	body, err := LoadFont(a.Containers, DocumentFont, FontCoverage)
	if err != nil {
		body = font
	}
	body.Selector = font.Selector
	workspaceFonts = []*text.Font{font, body}
	t.Cleanup(func() { workspaceFonts = nil })
}

// frameTimer times n frames of the editor: Update, then Draw onto a window-size
// picture, after a warm-up. step runs before each frame.
type frameTimer struct {
	e      *ui.MapEditor
	screen *ebiten.Image
	out    *os.File
	size   image.Point
}

func (f *frameTimer) run(name string, step func(i int)) {
	n, _ := strconv.Atoi(os.Getenv("AGAINROM_EDITOR_FRAMES_N"))
	if n <= 0 {
		n = 60
	}
	var total, update, draw []float64
	for i := -5; i < n; i++ {
		if step != nil {
			step(i)
		}
		t0 := time.Now()
		func() {
			defer func() { _ = recover() }()
			_ = f.e.Update()
		}()
		t1 := time.Now()
		f.e.Draw(f.screen)
		t2 := time.Now()
		if i >= 0 {
			total = append(total, float64(t2.Sub(t0).Microseconds())/1000)
			update = append(update, float64(t1.Sub(t0).Microseconds())/1000)
			draw = append(draw, float64(t2.Sub(t1).Microseconds())/1000)
		}
	}
	q := func(v []float64, p float64) float64 {
		s := slices.Clone(v)
		slices.Sort(s)
		return s[min(len(s)-1, int(p*float64(len(s))))]
	}
	fmt.Fprintf(f.out, "%dx%d\t%s\tframe median %.2f p95 %.2f\tupdate median %.2f\tdraw median %.2f p95 %.2f\n",
		f.size.X, f.size.Y, name, q(total, .5), q(total, .95), q(update, .5), q(draw, .5), q(draw, .95))
}

// TestEditorFrameTiming writes the editor's frame costs to the file the
// environment names; it runs nothing otherwise.
func TestEditorFrameTiming(t *testing.T) {
	path := os.Getenv("AGAINROM_EDITOR_FRAMES_OUT")
	if path == "" {
		t.Skip("AGAINROM_EDITOR_FRAMES_OUT names no output file")
	}
	out, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	frameTimingFonts(t)
	for _, size := range []image.Point{{X: 1920, Y: 1080}, {X: 2560, Y: 1440}} {
		_, e, _, _ := makeMapEditor(t, "")
		e.Layout(size.X, size.Y)
		f := &frameTimer{e: e, screen: ebiten.NewImage(size.X, size.Y), out: out, size: size}
		f.run("map-idle", nil)
		f.run("map-pointer", func(i int) {
			e.Pointer(ui.Input{CursorX: 300 + (i%50)*17, CursorY: 300 + (i%30)*11}, time.Now())
		})
		for page := 0; page < 8; page++ {
			if err := e.Key("package"); err != nil {
				t.Fatal(err)
			}
			if !e.ModWindowOpen() {
				continue
			}
			f.run(fmt.Sprintf("mod-open-step%d-idle", page), nil)
			f.run(fmt.Sprintf("mod-open-step%d-pointer", page), func(i int) {
				e.Pointer(ui.Input{CursorX: 300 + (i%50)*17, CursorY: 300 + (i%30)*11}, time.Now())
			})
		}
	}
}
