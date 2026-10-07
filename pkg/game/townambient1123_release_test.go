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
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/formats/bmp"
	"againrom/pkg/formats/spr16"
	"againrom/pkg/ui"
)

type townSquareOracle struct {
	base, add image.Image
	labels    map[int]image.Image
	motion    map[string][]image.Image
	birds     [9][]image.Image
	stars     []image.Image
	horse     [5][3][]image.Image
	baba      [4][2][]image.Image
	dervish   [4][]image.Image
}

// Table origins of the three wildlife families, read from the claims and not
// from production.
var (
	oracleHorseAt   = [5]image.Point{{104, 404}, {104, 404}, {256, 344}, {448, 400}, {140, 400}}
	oracleBabaAt    = [4]image.Point{{216, 364}, {308, 424}, {384, 424}, {580, 384}}
	oracleDervishAt = [4]image.Point{{224, 364}, {324, 424}, {392, 420}, {592, 388}}
)

func loadTownSquareOracle(t *testing.T, f *FrontEnd) townSquareOracle {
	t.Helper()
	read := func(path string) []byte {
		t.Helper()
		b, err := f.Archives.Containers.ReadFile(path)
		if err != nil {
			t.Fatal(path, err)
		}
		return b
	}
	readBMP := func(path string, keyBlack bool) image.Image {
		t.Helper()
		decoded, err := bmp.Decode(read(path))
		if err != nil {
			t.Fatal(path, err)
		}
		pic := image.NewRGBA(image.Rect(0, 0, decoded.Width, decoded.Height))
		for i, c := range decoded.Pix {
			a := uint8(255)
			if keyBlack && c.R == 0 && c.G == 0 && c.B == 0 {
				a = 0
			}
			pic.SetRGBA(i%decoded.Width, i/decoded.Width, color.RGBA{R: c.R, G: c.G, B: c.B, A: a})
		}
		return pic
	}
	read16A := func(path string) []image.Image {
		t.Helper()
		decoded, err := spr16.DecodeA(read(path), true)
		if err != nil {
			t.Fatal(path, err)
		}
		out := make([]image.Image, 0, len(decoded.Frames))
		for _, frame := range decoded.Frames {
			pic := image.NewRGBA(image.Rect(0, 0, frame.Width, frame.Height))
			for i, p := range frame.Pixels {
				if !p.Painted {
					continue
				}
				c := decoded.Palette[p.Index]
				a := (uint32(p.Level) + 1) * 255 / 16
				pic.SetRGBA(i%frame.Width, i/frame.Width, color.RGBA{
					R: byte(uint32(c.R) * a / 255), G: byte(uint32(c.G) * a / 255),
					B: byte(uint32(c.B) * a / 255), A: byte(a),
				})
			}
			out = append(out, pic)
		}
		return out
	}

	o := townSquareOracle{
		base: readBMP("graphics/interface/town/townmain.bmp", false),
		add:  readBMP("graphics/interface/town/town_add.bmp", true),
		labels: map[int]image.Image{
			1: readBMP("graphics/interface/town/shop_l.bmp", false),
			2: readBMP("graphics/interface/town/tavern_l.bmp", false),
			4: readBMP("graphics/interface/town/trener_l.bmp", false),
		},
		motion: make(map[string][]image.Image),
	}
	for _, name := range []string{"shopie", "tavern", "fighter", "mage", "guards"} {
		o.motion[name] = read16A("graphics/interface/townbirds/" + name + "/sprites.16a")
	}
	for _, family := range []struct {
		name, pattern string
		count         int
	}{{"door", "door/t%02d.bmp", 9}, {"sign", "sign/v%02d.bmp", 10}, {"fluger", "fluger/f%02d.bmp", 8}} {
		for i := 0; i < family.count; i++ {
			o.motion[family.name] = append(o.motion[family.name], readBMP("graphics/interface/town/"+fmt.Sprintf(family.pattern, i), false))
		}
	}
	for family := range o.birds {
		o.birds[family] = read16A(fmt.Sprintf("graphics/interface/townbirds/birds%d/sprites.16a", family+1))
		if len(o.birds[family]) != 57 {
			t.Fatalf("literal Birds%d count %d", family+1, len(o.birds[family]))
		}
	}
	for i := 0; i < 9; i++ {
		o.stars = append(o.stars, readBMP(fmt.Sprintf("graphics/interface/town/stars/s%02d.bmp", i), false))
	}
	for p := range o.horse {
		for v := range o.horse[p] {
			o.horse[p][v] = read16A(fmt.Sprintf("graphics/interface/townbirds/horse%d/a%d/sprites.16a", p+1, v+1))
		}
	}
	for p := range o.baba {
		for v := range o.baba[p] {
			o.baba[p][v] = read16A(fmt.Sprintf("graphics/interface/townbirds/baba%d/a%d/sprites.16a", p+1, v+1))
		}
	}
	for p := range o.dervish {
		o.dervish[p] = read16A(fmt.Sprintf("graphics/interface/townbirds/dervish%d/sprites.16a", p+1))
	}
	return o
}

// drawFamilies paints horse, baba and dervish, the painter's last three
// sprite draws, in that order.
func (o townSquareOracle) drawFamilies(dst *image.RGBA, frame ui.TownExteriorFrame) {
	put := func(pic image.Image, p image.Point) {
		b := pic.Bounds()
		draw.Draw(dst, b.Add(p.Sub(b.Min)), pic, b.Min, draw.Over)
	}
	if h := frame.Horse; h.Visible && h.Position >= 0 && h.Position < 5 && h.Sheet >= 0 && h.Sheet < 3 && h.Frame < len(o.horse[h.Position][h.Sheet]) {
		put(o.horse[h.Position][h.Sheet][h.Frame], oracleHorseAt[h.Position])
	}
	if b := frame.Baba; b.Visible && b.Position >= 0 && b.Position < 4 && b.Sheet >= 0 && b.Sheet < 2 && b.Frame < len(o.baba[b.Position][b.Sheet]) {
		put(o.baba[b.Position][b.Sheet][b.Frame], oracleBabaAt[b.Position])
	}
	if d := frame.Dervish; d.Visible && d.Position >= 0 && d.Position < 4 && d.Frame < len(o.dervish[d.Position]) {
		put(o.dervish[d.Position][d.Frame], oracleDervishAt[d.Position])
	}
}

func (o townSquareOracle) compose(frame ui.TownExteriorFrame, selector int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
	draw.Draw(dst, dst.Bounds(), o.base, image.Point{}, draw.Src)
	put := func(pic image.Image, p image.Point, op draw.Op) {
		if pic == nil {
			return
		}
		b := pic.Bounds()
		draw.Draw(dst, b.Add(p.Sub(b.Min)), pic, b.Min, op)
	}
	for _, selected := range frame.Birds {
		if selected.Visible && selected.Family >= 0 && selected.Family < len(o.birds) && selected.Frame >= 0 && selected.Frame < len(o.birds[selected.Family]) {
			put(o.birds[selected.Family][selected.Frame], image.Point{}, draw.Over)
		}
	}
	if frame.BirdOverlayVisible {
		put(o.add, image.Point{}, draw.Over)
	}
	if label := o.labels[selector]; label != nil {
		put(label, map[int]image.Point{1: {264, 264}, 2: {144, 332}, 4: {436, 300}}[selector], draw.Src)
	}
	for _, layer := range []struct {
		name  string
		frame int
		at    image.Point
	}{
		{"tavern", frame.Tavern, image.Pt(124, 312)}, {"sign", frame.Sign, image.Pt(360, 232)},
		{"door", frame.Door, image.Pt(180, 148)}, {"fighter", frame.Fighter, image.Pt(516, 344)},
		{"mage", frame.Mage, image.Pt(452, 328)}, {"shopie", frame.Shop, image.Pt(276, 296)},
		{"fluger", frame.Fluger, image.Pt(308, 64)}, {"guards", frame.Guard, image.Pt(184, 158)},
	} {
		if layer.frame >= 0 && layer.frame < len(o.motion[layer.name]) {
			put(o.motion[layer.name][layer.frame], layer.at, draw.Over)
		}
	}
	if frame.Star.Visible && frame.Star.Frame >= 0 && frame.Star.Frame < len(o.stars) {
		put(o.stars[frame.Star.Frame], image.Pt(340, 288), draw.Src)
	}
	o.drawFamilies(dst, frame)
	return dst
}

func TestReleaseTownAmbient1123InstalledBirdStarCrowdAndNative(t *testing.T) {
	f := releaseFront(t)
	if f.TownSquareArt.Value() == nil || f.TownSquareArt.Value().Exterior == nil || len(f.TownSquareArt.Value().ExteriorProblems) != 0 {
		t.Fatalf("town ambient art: %v / %v", f.TownSquareArt.Err(), f.TownSquareArt.Value().ExteriorProblems)
	}
	oracle := loadTownSquareOracle(t, f)
	for family := range oracle.birds {
		if got := f.TownSquareArt.Value().Exterior.Birds[family]; len(got) != len(oracle.birds[family]) {
			t.Fatalf("production Birds%d count %d", family+1, len(got))
		} else {
			for i := range got {
				if !bytes.Equal(got[i].(*image.RGBA).Pix, oracle.birds[family][i].(*image.RGBA).Pix) {
					t.Fatalf("production Birds%d frame%d differs from literal decoder", family+1, i)
				}
			}
		}
	}
	if len(f.TownSquareArt.Value().Exterior.Stars) != 9 {
		t.Fatalf("production stars %d", len(f.TownSquareArt.Value().Exterior.Stars))
	}

	f.Carried = f.NextParty()
	f.arriveInTown()
	f.Town.announceMission(f.Town.currentMain())
	now := time.Unix(400, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.TownAnimationRandom = func(int) int { return 0 }
	rolls := &townAmbientRolls{values: []int{0, 0, 999, 1, 2, 0}}
	f.TownAmbientRandom = rolls.draw
	f.townLatches.birdDelayReady, f.townLatches.birdDelay = true, time.Second
	voices := &exteriorRecorder{}
	crowd := &townCrowdRecorder{}
	f.SoundPlayer, f.AmbientPlayer = voices, crowd
	f.SoundBank = OpenSounds(f.Archives.Root)
	before, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	nativeBefore, err := EncodeSave(before, label)
	if err != nil {
		t.Fatal(err)
	}
	app, screen := exteriorApp(t, f)
	if len(crowd.starts) != 1 || crowd.starts[0].kind != ui.AmbientTownCrowd {
		t.Fatalf("installed crowd start %+v", crowd.starts)
	}

	selector := -1
	check := func(name string, dt time.Duration, inspect func(ui.TownExteriorFrame)) {
		t.Helper()
		pix := exteriorPaint(t, app, &now, dt)
		frame := *screen.townExteriorFrame()
		if inspect != nil {
			inspect(frame)
		}
		want := oracle.compose(frame, selector)
		if !bytes.Equal(pix.Pix, want.Pix) {
			for y := 0; y < 480; y++ {
				for x := 0; x < 640; x++ {
					if pix.RGBAAt(x, y) != want.RGBAAt(x, y) {
						t.Fatalf("%s pixel %d,%d=%v want %v", name, x, y, pix.RGBAAt(x, y), want.RGBAAt(x, y))
					}
				}
			}
		}
		writeTownAmbient1123Witness(t, f, name, pix)
	}
	check("s00", 0, func(frame ui.TownExteriorFrame) {
		if !frame.Star.Visible || frame.Star.Frame != 0 || frame.BirdOverlayVisible {
			t.Fatalf("entry frame %+v", frame)
		}
	})
	check("delay-equality", time.Second, func(frame ui.TownExteriorFrame) {
		if frame.BirdOverlayVisible {
			t.Fatal("strict bird delay admitted equality")
		}
	})
	check("bird1", time.Millisecond, func(frame ui.TownExteriorFrame) {
		if !frame.BirdOverlayVisible || !frame.Birds[0].Visible || frame.Birds[0].Family != 0 || frame.Birds[1].Visible {
			t.Fatalf("bird1 frame %+v", frame)
		}
	})
	for screen.exterior.birdProgress[0] < 56 {
		check("", 68*time.Millisecond, nil)
	}
	check("bird-terminal", 68*time.Millisecond, func(frame ui.TownExteriorFrame) {
		if !frame.BirdOverlayVisible || frame.Birds[0].Visible || !screen.exterior.birdTerminalPaint {
			t.Fatalf("terminal frame %+v", frame)
		}
	})
	check("bird-post", time.Millisecond, func(frame ui.TownExteriorFrame) {
		if frame.BirdOverlayVisible || screen.exterior.birdActive {
			t.Fatalf("post-terminal frame %+v", frame)
		}
	})
	if voice, ok := screen.exterior.voices[exteriorBird].(*exteriorVoice); ok {
		voice.playing = false
	}
	check("bird3", f.townLatches.birdDelay+time.Millisecond, func(frame ui.TownExteriorFrame) {
		if !frame.BirdOverlayVisible || screen.exterior.birdCount != 3 {
			t.Fatalf("bird3 frame %+v", frame)
		}
		for i := 0; i < 3; i++ {
			if !frame.Birds[i].Visible || frame.Birds[i].Family != 3+i {
				t.Fatalf("bird3 slot%d %+v", i, frame.Birds[i])
			}
		}
	})

	// A fresh effective view restores S00 while retaining process counters.
	f.townLatches.birdDelay = 24 * time.Hour
	screen.resetTownExterior()
	screen.TownSquareActive(true)
	selector = -1
	check("s00-reentry", 0, func(frame ui.TownExteriorFrame) {
		if !frame.Star.Visible || frame.Star.Frame != 0 {
			t.Fatalf("reentry star %+v", frame.Star)
		}
	})
	starPoint := image.Pt(-1, -1)
	for y := 0; y < 480 && starPoint.X < 0; y++ {
		for x := 0; x < 640; x++ {
			if f.TownSquareArt.Value().Mask.ColorIndexAt(x, y) == 0xb0 {
				starPoint = image.Pt(x, y)
				break
			}
		}
	}
	if starPoint.X < 0 {
		t.Fatal("installed mask has no selector16 pixel")
	}
	if err := app.HeadlessPointer("hover", starPoint.X, starPoint.Y); err != nil {
		t.Fatal(err)
	}
	selector = 16
	for i := 1; i <= 7; i++ {
		check("", 68*time.Millisecond, nil)
	}
	check("s08", 68*time.Millisecond, func(frame ui.TownExteriorFrame) {
		if !frame.Star.Visible || frame.Star.Frame != 8 {
			t.Fatalf("S08 frame %+v", frame.Star)
		}
	})
	check("star-hidden", 68*time.Millisecond, func(frame ui.TownExteriorFrame) {
		if frame.Star.Visible {
			t.Fatalf("terminal star %+v", frame.Star)
		}
	})

	after, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	nativeAfter, err := EncodeSave(after, label)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(nativeBefore, nativeAfter) {
		t.Fatal("installed town ambience changed native save bytes")
	}
	sfx, err := OpenContainers(filepath.Join(f.Archives.Root, "sfx.res"))
	if err != nil {
		t.Fatal(err)
	}
	wantSound := func(path string) audio.Sample {
		t.Helper()
		raw, e := sfx.ReadFile("sfx/" + path)
		if e != nil {
			t.Fatal(path, e)
		}
		sample, e := audio.DecodeWAV(raw, audio.DeviceRate)
		if e != nil {
			t.Fatal(path, e)
		}
		return sample
	}
	for _, path := range []string{"town/birds1.wav", "town/birds2.wav", "town/stars.wav"} {
		want := wantSound(path)
		found := false
		for _, got := range voices.samples {
			found = found || reflect.DeepEqual(got, want)
		}
		if !found {
			t.Fatalf("literal installed sound %s was not requested", path)
		}
	}
	if !reflect.DeepEqual(crowd.starts[0].sample, wantSound("town/crowd.wav")) {
		t.Fatal("crowd loop did not use literal installed waveform")
	}
	if len(crowd.starts) != 2 || len(crowd.stops) != 1 {
		t.Fatalf("crowd reentry lifecycle starts%d stops%d", len(crowd.starts), len(crowd.stops))
	}
	t.Log("1123 installed oracle: nine 57-frame bird families, S00..S08, bird1/bird3/terminal/post and repeating crowd; literal sounds and native bytes match")
}

func writeTownAmbient1123Witness(t *testing.T, f *FrontEnd, name string, pix *image.RGBA) {
	t.Helper()
	if name == "" {
		return
	}
	base := os.Getenv("AGAINROM_STORY1123_FRAMES")
	if base == "" {
		return
	}
	dir, err := filepath.Abs(filepath.Join(base, strings.ToLower(filepath.Base(f.Archives.Root))))
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(f.Archives.Root, dir)
	if err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatal("story1123 capture must stay outside install")
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	encodeErr := png.Encode(file, pix)
	closeErr := file.Close()
	if encodeErr != nil {
		t.Fatal(encodeErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}
