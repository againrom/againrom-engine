package game

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/hajimehoshi/bitmapfont/v4"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
	"againrom/pkg/ui"
)

type debugFontLine struct {
	text string
	x, y int
}

func debugFontClippedLine(s string) string {
	r := []rune(s)
	if len(r) > 104 {
		return string(r[:101]) + "..."
	}
	return s
}

// The reference uses the public face, not the captured masks. Each native
// atlas cell is clipped before placement; DebugPrintAt places it at X+1.
func debugFontNativeReference(lines []debugFontLine) (*image.RGBA, []text.DrawCall) {
	native := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	var calls []text.DrawCall
	for _, line := range lines {
		x, y := line.x+1, line.y
		for _, r := range line.text {
			if r == '\n' {
				x, y = line.x+1, y+16
				continue
			}
			if r >= 0 && r <= 255 {
				cell := image.NewRGBA(image.Rect(0, 0, 6, 16))
				for _, pass := range []struct {
					ink color.RGBA
					at  image.Point
				}{{color.RGBA{A: 128}, image.Pt(1, 1)}, {color.RGBA{255, 255, 255, 255}, image.Point{}}} {
					d := font.Drawer{Dst: cell, Src: image.NewUniform(pass.ink), Face: bitmapfont.Face,
						Dot: fixed.Point26_6{X: fixed.I(pass.at.X), Y: bitmapfont.Face.Metrics().Ascent + fixed.I(pass.at.Y)}}
					d.DrawString(string(r))
				}
				draw.Draw(native, cell.Bounds().Add(image.Pt(x, y)), cell, image.Point{}, draw.Over)
				for _, ink := range []color.RGBA{{A: 128}, {255, 255, 255, 255}} {
					g := &text.Glyph{Width: 6, Height: 16, Advance: 6, Pixels: make([]text.Pixel, 6*16)}
					for n := range g.Pixels {
						g.Pixels[n] = text.Pixel{Level: text.MaxLevel, Painted: cell.RGBAAt(n%6, n/6) == ink}
					}
					calls = append(calls, text.DrawCall{Glyph: g, X: x, Y: y, Color: ink,
						Flat: true, SourceOver: true, Clip: native.Bounds()})
				}
			}
			x += 6
		}
	}
	return native, calls
}

func debugFontCalls(t *testing.T, a *ui.App, expected int) []text.DrawCall {
	t.Helper()
	captured, kept, readbacks := a.TextSettle()
	fates := a.TextFates()
	if readbacks != 0 || captured != len(fates) {
		t.Fatalf("settlement captured=%d fates=%d kept=%d readbacks=%d", captured, len(fates), kept, readbacks)
	}
	var calls []text.DrawCall
	showing := 0
	for _, ft := range fates {
		if !ft.Call.SourceOver {
			continue
		}
		switch ft.Fate {
		case textsmooth.Kept:
			showing++
		case textsmooth.Blank:
		default:
			t.Errorf("debug glyph (%d,%d) stays %s", ft.Call.X, ft.Call.Y, ft.Fate)
		}
		c := ft.Call
		c.Under, c.Mask = slices.Clone(c.Under), slices.Clone(c.Mask)
		calls = append(calls, c)
	}
	if len(calls) != expected || showing == 0 || kept < showing {
		t.Fatalf("debug capture has %d calls, %d showing and %d total kept; want exactly %d calls and showing overlay", len(calls), showing, kept, expected)
	}
	return calls
}

func debugFontRaster(t *testing.T, calls []text.DrawCall) *image.RGBA {
	t.Helper()
	dst := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	for _, c := range calls {
		if c.Glyph == nil || c.Glyph.Width != 6 || c.Glyph.Height != 16 || c.Glyph.Advance != 6 ||
			(c.Color != (color.RGBA{A: 128}) && c.Color != (color.RGBA{255, 255, 255, 255})) {
			t.Fatalf("debug glyph lost native cell or ink: %+v", c)
		}
		for n, p := range c.Glyph.Pixels {
			at := image.Pt(c.X+n%6, c.Y+n/6)
			if p.Painted && at.In(dst.Bounds()) && (c.Clip.Empty() || at.In(c.Clip)) {
				if p.Level != text.MaxLevel {
					t.Fatalf("debug cell (%d,%d) carries non-native level %d", at.X, at.Y, p.Level)
				}
				draw.Draw(dst, image.Rectangle{Min: at, Max: at.Add(image.Pt(1, 1))}, image.NewUniform(c.Color), image.Point{}, draw.Over)
			}
		}
	}
	return dst
}

func debugFontSaveWitness(t *testing.T, name string, native, smooth *image.RGBA) {
	t.Helper()
	out := os.Getenv("AGAINROM_DEBUGFONT_WITNESS_OUT")
	if out == "" {
		return
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	name = filepath.Base(os.Getenv("AGAINROM_ASSETS")) + "-" + name
	for _, artifact := range []struct {
		label string
		pic   *image.RGBA
	}{{"native-debug-fragment", native}, {"method-c-debug-fragment", smooth}} {
		path := filepath.Join(out, name+"-"+artifact.label+".png")
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, artifact.pic)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			t.Fatalf("write %s: encode=%v close=%v", path, err, closeErr)
		}
	}
}

func debugFontWindowProof(t *testing.T, a *ui.App, name string, lines []debugFontLine) {
	t.Helper()
	a.Layout(smoothingW, smoothingH)
	place := frame.Fit(frame.W, frame.H, smoothingW, smoothingH)
	if place.Scale() <= 1 {
		t.Fatalf("window scale %v must exceed 1", place.Scale())
	}
	native, reference := debugFontNativeReference(lines)
	screen := ebiten.NewImage(smoothingW, smoothingH)
	t.Cleanup(screen.Dispose)
	a.SetTextSmoothing(true)
	a.Draw(screen)
	first := debugFontCalls(t, a, len(reference))
	if raster := debugFontRaster(t, first); !bytes.Equal(raster.Pix, native.Pix) {
		t.Fatal("App debug capture differs bytewise from the independent native face and shadow")
	}
	a.Draw(screen)
	if replay := debugFontCalls(t, a, len(reference)); !reflect.DeepEqual(first, replay) {
		t.Fatal("cached redraw duplicated, changed or lost a debug glyph")
	}
	a.SetTextSmoothing(false)
	a.Draw(screen)
	if captured, kept, fallbacks := a.TextSettle(); captured != 0 || kept != 0 || fallbacks != 0 || len(a.TextFates()) != 0 {
		t.Fatalf("native off control captured=%d kept=%d readbacks=%d fates=%d", captured, kept, fallbacks, len(a.TextFates()))
	}
	a.SetTextSmoothing(true)
	a.Draw(screen)
	if again := debugFontCalls(t, a, len(reference)); !reflect.DeepEqual(first, again) {
		t.Fatal("on/off/on changed or duplicated debug capture")
	}
	ox, oy := place.Origin()
	smooth := image.NewRGBA(screen.Bounds())
	textsmooth.Composite(smooth, first, place.Scale(), ox, oy)
	want := image.NewRGBA(screen.Bounds())
	textsmooth.Composite(want, reference, place.Scale(), ox, oy)
	if !bytes.Equal(smooth.Pix, want.Pix) {
		t.Fatal("method C overlay differs from independently constructed native debug cells")
	}
	nearest := image.NewRGBA(screen.Bounds())
	for y := range native.Bounds().Dy() {
		for x := range native.Bounds().Dx() {
			c := native.RGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			r := textsmooth.TargetRect(x, y, 1, 1, place.Scale(), ox, oy)
			draw.Draw(nearest, r, image.NewUniform(c), image.Point{}, draw.Src)
		}
	}
	if bytes.Equal(smooth.Pix, nearest.Pix) {
		t.Fatal("showing debug text retained the native stepped raster at above-1 scale")
	}
	debugFontSaveWitness(t, name, nearest, smooth)
	t.Logf("debug proof: scale=%g calls=%d native cells equal, redraw and on/off/on equal, zero readbacks; CPU debug fragments only, no GPU framebuffer readback", place.Scale(), len(first))
}

func debugFontInstalledTown(t *testing.T) *ui.App {
	t.Helper()
	path := os.Getenv("AGAINROM_CITY_SAV")
	if path == "" {
		t.Skip("AGAINROM_CITY_SAV is not set: debug status needs an existing town SAV")
	}
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	// The status line proof reads a still picture, so the families take
	// positions clear of the line.
	a := f.App("debug font town")
	draws := &familyDraws{script: []int{rawH(4), rawB(3), rawB(2)}}
	f.townUI.squareView().SetRawDraw("wildlife", draws.next)
	t.Cleanup(a.StopAudio)
	a.Layout(smoothingW, smoothingH)
	save, list, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: filepath.Dir(path)}, nil)
	a.SetSaveSeams(save, func() []ui.SaveEntry {
		var rows []ui.SaveEntry
		for _, row := range list() {
			if row.Name == filepath.Base(path) {
				rows = append(rows, row)
			}
		}
		return rows
	}, load)
	if err := a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("@first"); err != nil || a.Screen() != ui.ScreenTown {
		t.Fatalf("existing SAV LOAD screen=%s: %v, message=%q", a.Screen(), err, a.HeadlessMessage())
	}
	t.Logf("town input %s read-only; installed-resource restore/render witness, source-save language acceptance is separate", path)
	return a
}

func TestReleaseDebugFontEscStatusesAreSmoothedAtWindowScale(t *testing.T) {
	for _, route := range []string{"town", "mission"} {
		t.Run(route, func(t *testing.T) {
			var a *ui.App
			if route == "town" {
				a = debugFontInstalledTown(t)
			} else {
				_, a = smoothingMission(t)
			}
			message := fmt.Sprintf("%s SAVE diagnostic: refused A B", route)
			calls := 0
			a.SetSaveDialogSeams(ui.SaveDialogSeams{})
			a.SetSaveSeams(func(onMap bool) (string, error) {
				calls++
				if onMap != (route == "mission") {
					t.Errorf("SAVE context onMap=%v, route=%s", onMap, route)
				}
				return "", errors.New(message)
			}, nil, nil)
			if err := a.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal(err)
			}
			if a.Screen() != ui.ScreenGameMenu || a.HeadlessMessage() != message || calls != 1 {
				t.Fatalf("Esc SAVE status screen=%s message=%q callbacks=%d", a.Screen(), a.HeadlessMessage(), calls)
			}
			debugFontWindowProof(t, a, route+"-esc-status", []debugFontLine{{message, 8, 452}})
			message = route + " SAVE error: B"
			if err := a.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal(err)
			}
			if a.HeadlessMessage() != message || calls != 2 {
				t.Fatalf("changed SAVE status message=%q callbacks=%d", a.HeadlessMessage(), calls)
			}
			screen := ebiten.NewImage(smoothingW, smoothingH)
			t.Cleanup(screen.Dispose)
			a.Draw(screen)
			native, reference := debugFontNativeReference([]debugFontLine{{message, 8, 452}})
			if !bytes.Equal(debugFontRaster(t, debugFontCalls(t, a, len(reference))).Pix, native.Pix) {
				t.Fatal("the changed status retained stale debug text")
			}
			t.Log("reachability: installed App, Esc and validated SAVE row; fault injected through legacy SetSaveSeams, while the executable configures the SAV dialog seam")
		})
	}
}

func TestReleaseDebugFontMapPickerFailureIsSmoothedAtWindowScale(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	if len(f.Maps) == 0 {
		t.Fatal("the installed map list is empty")
	}
	row := f.Maps[0]
	row.Source, row.Name, row.Err, row.Mission = "missing-debugfont.alm", "missing map diagnostic", nil, 0
	row.FromArchive = false
	row.Width, row.Height, row.Word70, row.Word74 = 64, 48, 3, 12
	f.Maps = []MapEntry{row}
	a := f.App("debug font picker")
	t.Cleanup(a.StopAudio)
	a.Layout(smoothingW, smoothingH)
	a.SetNewGameChargen(nil)
	if err := a.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenPicker {
		t.Fatalf("explicit picker route opened %s", a.Screen())
	}
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	rows := a.HeadlessRows()
	message := a.HeadlessMessage()
	if a.Screen() != ui.ScreenPicker || message == "" || len(rows) != 1 || rows[0].Choosable {
		t.Fatalf("failed load screen=%s message=%q rows=%v", a.Screen(), message, rows)
	}
	p := ui.NewPicker([]ui.PickerRow{{Text: row.Text()}})
	debugFontWindowProof(t, a, "picker-failed-load", []debugFontLine{
		{p.HeaderText(), 8, 8}, {p.RowText(0), 8, 40}, {"48x32", 308, 40}, {"3", 398, 40}, {"12", 428, 40},
		{debugFontClippedLine(message), 8, 452},
	})
	t.Log("reachability: explicit -picker wiring, NEW GAME and Enter use the installed App's real loose-file loader; only map metadata names a missing file, no installed file is changed")
}

func TestReleaseMapEditorInstalledTextIsCapturedAtScaleOne(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: editor integration needs a lawful install")
	}
	x, err := NewMapInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(x.Maps) == 0 {
		t.Fatal("the installed editor catalogue is empty")
	}
	e := x.Editor()
	row := x.Maps[0]
	key := "loose/" + row.Source
	if row.FromArchive {
		key = "scenario/" + row.Source
	}
	if err := e.Open(key); err != nil {
		t.Fatal(err)
	}
	e.Layout(smoothingW, smoothingH)
	v := e.Document().Viewer
	if got := v.Placement().Scale(); got != 1 || v.FrameSize() != image.Pt(smoothingW, smoothingH) {
		t.Fatalf("editor native layout frame=%v scale=%g", v.FrameSize(), got)
	}
	screen := ebiten.NewImage(smoothingW, smoothingH)
	t.Cleanup(screen.Dispose)
	e.Draw(screen)
	captured, kept, readbacks := e.TextSettle()
	fates := e.TextFates()
	if captured == 0 || kept == 0 || captured != len(fates) || readbacks != 0 {
		t.Fatalf("editor installed text captured=%d kept=%d fates=%d readbacks=%d", captured, kept, len(fates), readbacks)
	}
	for _, ft := range fates {
		if ft.Call.SourceOver {
			t.Error("installed editor rail unexpectedly used a debug glyph")
		}
		if ft.Fate != textsmooth.Kept && ft.Fate != textsmooth.Blank {
			t.Errorf("editor rail glyph (%d,%d) is %s", ft.Call.X, ft.Call.Y, ft.Fate)
		}
	}
	if e.Dirty() {
		t.Fatal("opening and drawing the installed editor changed map state")
	}
	t.Logf("editor %s: frame=%v scale=1 captured=%d kept=%d; installed PanelImage route, read-only map open, no above-1 claim", key, v.FrameSize(), captured, kept)
}
