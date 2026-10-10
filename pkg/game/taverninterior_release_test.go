package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/ui"
)

func TestReleaseTavernInterior1121InstalledAppFramesSoundsAndNative(t *testing.T) {
	f := releaseFront(t)
	if f.TownTavernArt.Value() == nil || f.TownTavernArt.Err() != nil {
		t.Fatalf("tavern art = %#v / %v", f.TownTavernArt, f.TownTavernArt.Err())
	}

	read := func(path string) []byte {
		t.Helper()
		payload, err := f.Archives.Containers.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		return payload
	}
	decodePatch := func(path string, keyed bool) image.Image {
		t.Helper()
		decoded, err := bmp.Decode(read(path))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		pic := image.NewRGBA(image.Rect(0, 0, decoded.Width, decoded.Height))
		for i, c := range decoded.Pix {
			alpha := uint8(0xff)
			if keyed && c.R == 0 && c.G == 0 && c.B == 0 {
				alpha = 0
			}
			pic.SetRGBA(i%decoded.Width, i/decoded.Width, color.RGBA{R: c.R, G: c.G, B: c.B, A: alpha})
		}
		return pic
	}

	// Literal paths and counts are the oracle. It does not call the production
	// interior loader or reuse its family slices.
	oracle := make(map[string][]image.Image)
	for _, spec := range []struct {
		name          string
		pattern       string
		first, loaded int
	}{
		{"candle", "graphics/interface/inn/candle/t%04d.bmp", 0, 10},
		{"cauldron", "graphics/interface/inn/cauldron/t%04d.bmp", 0, 21},
		{"breath", "graphics/interface/inn/tender/breath/br%04d.bmp", 1, 24},
		{"drink", "graphics/interface/inn/tender/drink/dr%04d.bmp", 1, 40},
	} {
		for i := 0; i < spec.loaded; i++ {
			oracle[spec.name] = append(oracle[spec.name], decodePatch(fmt.Sprintf(spec.pattern, spec.first+i), spec.name == "candle" || spec.name == "cauldron"))
		}
		last := oracle[spec.name][spec.loaded-1]
		for i := 0; i < spec.loaded-1; i++ {
			if imagesEqual(last, oracle[spec.name][i]) {
				t.Fatalf("%s terminal file duplicates entry %d", spec.name, i)
			}
		}
	}
	production := map[string][]image.Image{
		"candle": f.TownTavernArt.Value().Scene["candle"], "cauldron": f.TownTavernArt.Value().Scene["cauldron"],
		"breath": f.TownTavernArt.Value().Scene["breath"], "drink": f.TownTavernArt.Value().Scene["drink"],
	}
	for name, want := range oracle {
		got := production[name]
		if len(got) != len(want) {
			t.Fatalf("%s count = %d, want %d", name, len(got), len(want))
		}
		for i := range want {
			if !imagesEqual(got[i], want[i]) {
				t.Fatalf("%s frame %d differs from literal independent decode", name, i)
			}
		}
	}

	f.SoundBank = OpenSounds(f.Archives.Root)
	if f.SoundBank == nil {
		t.Fatal("installed sound bank did not open")
	}
	for _, path := range []string{
		"town/inn/drink.wav", "town/inn/glotok.wav", "town/inn/steam.wav",
		"town/inn/water.wav", "town/inn/chair.wav", "town/inn/enter.wav", "town/shop/breath.wav",
	} {
		if _, ok := f.SoundBank.namedSample(path); !ok {
			t.Fatalf("installed sound missing: %s", path)
		}
	}

	// Reach an installed chapter with stock already unlocked by earlier authored
	// chapters. The witness needs a selected miniature but does not hire it.
	targetChapter := releaseMercenaryChapter(t, f.Campaign.Value())
	town := NewTown(f.Campaign.Value())
	for mission := range f.Campaign.Value().Chapters {
		if mission < targetChapter {
			town.Won(mission)
		}
	}
	if got := town.Chapter(); got != targetChapter {
		t.Fatalf("prepared chapter = %d, want %d", got, targetChapter)
	}
	f.Town = town
	f.Carried = f.NextParty()
	f.arriveInTown()
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodeSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err := store.Write(time.Unix(100, 0), payload); err != nil {
		t.Fatal(err)
	}

	now, roll := time.Unix(100, 0), 0
	f.TownAnimationNow = func() time.Time { return now }
	f.TavernRandom = func(n int) int {
		if n != 32768 {
			t.Fatalf("random range = %d", n)
		}
		return roll
	}
	recorder := &tavernInteriorRecorder{}
	f.SoundPlayer = recorder
	countInstalledRequest := func(slot string) int {
		t.Helper()
		want, ok := f.SoundBank.namedSample(tavernSoundKey(slot))
		if !ok {
			t.Fatalf("installed sound unavailable for slot %s", slot)
		}
		n := 0
		for _, got := range recorder.samples {
			if got.Rate == want.Rate && slices.Equal(got.PCM, want.PCM) {
				n++
			}
		}
		return n
	}
	app := f.App("1121-installed-tavern")
	app.Layout(640, 480)
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
	if err := app.HeadlessActivate("load game"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	s := f.TownScreen().(*townScreen)
	s.CloseTip()
	before, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	beforeBytes, err := EncodeSave(before, "same")
	if err != nil {
		t.Fatal(err)
	}

	writeFrame := func(name string, frame *image.RGBA) {
		t.Helper()
		dir := os.Getenv("AGAINROM_STORY1121_FRAMES")
		if dir == "" {
			return
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		rootName := filepath.Base(filepath.Clean(f.Archives.Root))
		out, err := os.Create(filepath.Join(dir, rootName+"-"+name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(out, frame); err != nil {
			out.Close()
			t.Fatal(err)
		}
		if err = out.Close(); err != nil {
			t.Fatal(err)
		}
	}

	check := func(name string, actual *image.RGBA, candle, cauldron int, tender string, tenderFrame int) {
		t.Helper()
		view := s.TownSurface()
		without := view
		without.Scene = nil
		expected := ui.ComposeTownSurface(without)
		layers := []struct {
			pic image.Image
			at  image.Point
		}{{oracle["candle"][candle], image.Pt(160, 48)}, {oracle["cauldron"][cauldron], image.Pt(420, 160)}}
		if tender != "" {
			layers = append(layers, struct {
				pic image.Image
				at  image.Point
			}{oracle[tender][tenderFrame], image.Pt(240, 152)})
		}
		var compared int
		for _, layer := range layers {
			b := layer.pic.Bounds()
			placed := b.Add(layer.at.Sub(b.Min))
			clip := placed.Intersect(image.Rect(160, 0, 480, 480))
			if !clip.Empty() {
				src := b.Min.Add(clip.Min.Sub(placed.Min))
				draw.Draw(expected, clip, layer.pic, src, draw.Over)
			}
			// The left/right child panes and bottom roster draw later. This
			// interior crop contains only the central painter's own result.
			probe := clip.Intersect(image.Rect(176, 0, 464, 280))
			for y := probe.Min.Y; y < probe.Max.Y; y++ {
				for x := probe.Min.X; x < probe.Max.X; x++ {
					if actual.RGBAAt(x, y) != expected.RGBAAt(x, y) {
						t.Fatalf("%s pixel %d,%d = %v, want %v", name, x, y, actual.RGBAAt(x, y), expected.RGBAAt(x, y))
					}
					compared++
				}
			}
		}
		if compared == 0 {
			t.Fatalf("%s compared no central pixels", name)
		}
		writeFrame(name, actual)
	}

	frame := drawTavernApp(t, app, s, &now, 0)
	check("entry", frame, 0, 0, "", 0)
	frame = drawTavernApp(t, app, s, &now, 101*time.Millisecond)
	check("cycle-0", frame, 0, 0, "", 0)
	frame = drawTavernApp(t, app, s, &now, 0)
	check("cycle-1", frame, 1, 1, "", 0)
	frame = drawTavernApp(t, app, s, &now, 2900*time.Millisecond)
	check("breath", frame, 1, 1, "breath", 0)

	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	roll = 16 // 3001ms, odd: drink
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	tip := s.TownSurface().Tip
	if !tip.Showing() {
		t.Fatal("tavern re-entry did not construct its popup")
	}
	close := ui.TipPanelCloseRect(tip.Rect)
	p := close.Min.Add(close.Max).Div(2)
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
	if s.TownSurface().Tip.Showing() {
		t.Fatal("owned Close left the recreated tavern popup showing")
	}
	frame = drawTavernApp(t, app, s, &now, 3002*time.Millisecond)
	check("drink", frame, 0, 0, "drink", 0)

	selected := ""
	for _, cell := range s.TownSurface().Cells {
		if strings.HasPrefix(cell.Semantic, "Mercenary ") {
			selected = cell.Semantic
			break
		}
	}
	if selected == "" {
		t.Fatal("installed tavern has no selectable mercenary")
	}
	if err := app.HeadlessActivate(selected); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	frame = drawTavernApp(t, app, s, &now, 0)
	writeFrame("selected", frame)
	if tavernTender(s).State != 1 || s.tavernSelection.kind != tavernCandidateMercenary {
		t.Fatal("selected-card animation displaced keeper state or selection")
	}

	// Hold the odd episode's delay open, reach index30 through actual App
	// paints, and witness the indexed installed request without a one-shot
	// substitute.
	oneShotsBefore := recorder.ones
	tavernTender(s).Delay = time.Hour
	for tavernTender(s).Index[tavernDrink] < 30 {
		drawTavernApp(t, app, s, &now, 84*time.Millisecond)
	}
	drawTavernApp(t, app, s, &now, 0)
	if countInstalledRequest("drink") != 1 || recorder.ones != oneShotsBefore {
		t.Fatalf("drink request = %d retained / %d new one-shot",
			countInstalledRequest("drink"), recorder.ones-oneShotsBefore)
	}
	drawTavernApp(t, app, s, &now, 11*time.Second)
	if countInstalledRequest("steam") == 0 || countInstalledRequest("water") == 0 ||
		countInstalledRequest("chair") == 0 || countInstalledRequest("breath") == 0 ||
		countInstalledRequest("enter") != 2 {
		t.Fatalf("installed requests = steam%d water%d chair%d breath%d enter%d",
			countInstalledRequest("steam"), countInstalledRequest("water"),
			countInstalledRequest("chair"), countInstalledRequest("breath"),
			countInstalledRequest("enter"))
	}

	after, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	afterBytes, err := EncodeSave(after, "same")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeBytes, afterBytes) {
		t.Fatal("installed presentation changed native save bytes")
	}
	t.Logf("installed tavern: frames %d/%d/%d/%d; central pixels checked; retained requests %d; no physical GUI or audible-output claim",
		len(oracle["candle"]), len(oracle["cauldron"]), len(oracle["breath"]), len(oracle["drink"]), len(recorder.samples))
}
