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

func TestReleaseTownExterior1120AppFramesSoundsAndNativeLoad(t *testing.T) {
	f := releaseFront(t)
	if f.TownSquareArt.Value() == nil || len(f.TownSquareArt.Value().Problems) != 0 {
		t.Fatalf("exterior load: %v / %v", f.TownSquareArt.Err(), f.TownSquareArt)
	}
	// This older oracle owns the entrance/sign/fluger families only. The
	// square art is the process's shared install data, so the families are
	// cleared on this front end's own copy.
	f.TownSquareArt = resolved(withoutAmbience(f.TownSquareArt.Value()), nil)
	f.Carried = f.NextParty()
	f.arriveInTown()
	// Accepted mission availability is existing model state, not animation state.
	f.Town.announceMission(f.Town.currentMain())
	now, roll := time.Unix(100, 0), 0
	f.TownAnimationNow = func() time.Time { return now }
	f.TownAnimationRandom = func(int) int { return roll }
	r := &exteriorRecorder{}
	f.SoundPlayer = r
	f.SoundBank = OpenSounds(f.Archives.Root)
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodeSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err = store.Write(now, payload); err != nil {
		t.Fatal(err)
	}
	a := f.App("1120-installed-town")
	a.Layout(640, 480)
	a.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
	if err = a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err = a.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenTown {
		t.Fatal("native App LOAD did not enter town")
	}
	s := f.TownScreen().(*townScreen)
	s.CloseTip()
	initial, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	initialBytes, err := EncodeSave(initial, "same")
	if err != nil {
		t.Fatal(err)
	}
	read := func(path string) []byte {
		t.Helper()
		b, e := f.Archives.Containers.ReadFile(path)
		if e != nil {
			t.Fatal(path, e)
		}
		return b
	}
	bmpImage := func(path string) image.Image {
		t.Helper()
		b, e := bmp.Decode(read(path))
		if e != nil {
			t.Fatal(path, e)
		}
		p := image.NewRGBA(image.Rect(0, 0, b.Width, b.Height))
		for i, c := range b.Pix {
			p.SetRGBA(i%b.Width, i/b.Width, color.RGBA{c.R, c.G, c.B, 255})
		}
		return p
	}
	// Oracle decodes literal claim paths independently. It never calls the
	// production exterior loader, cursorPixel or TownSquareLabelOrigin.
	sheets := map[string][]image.Image{}
	for _, name := range []string{"shopie", "tavern", "fighter", "mage", "guards"} {
		sp, e := spr16.DecodeA(read("graphics/interface/townbirds/"+name+"/sprites.16a"), true)
		if e != nil {
			t.Fatal(e)
		}
		for _, fr := range sp.Frames {
			p := image.NewRGBA(image.Rect(0, 0, fr.Width, fr.Height))
			for i, c := range fr.Pixels {
				if c.Painted {
					col := sp.Palette[c.Index]
					alpha := (uint32(c.Level) + 1) * 255 / 16
					p.SetRGBA(i%fr.Width, i/fr.Width, color.RGBA{byte(uint32(col.R) * alpha / 255), byte(uint32(col.G) * alpha / 255), byte(uint32(col.B) * alpha / 255), byte(alpha)})
				}
			}
			sheets[name] = append(sheets[name], p)
		}
	}
	for _, group := range []struct {
		name, pattern string
		count         int
	}{{"door", "door/t%02d.bmp", 9}, {"sign", "sign/v%02d.bmp", 10}, {"fluger", "fluger/f%02d.bmp", 8}} {
		for i := 0; i < group.count; i++ {
			sheets[group.name] = append(sheets[group.name], bmpImage("graphics/interface/town/"+fmt.Sprintf(group.pattern, i)))
		}
	}
	base := bmpImage("graphics/interface/town/townmain.bmp")
	labels := map[int]image.Image{1: bmpImage("graphics/interface/town/shop_l.bmp"), 2: bmpImage("graphics/interface/town/tavern_l.bmp"), 4: bmpImage("graphics/interface/town/trener_l.bmp")}
	labelPoints := map[int]image.Point{1: {264, 264}, 2: {144, 332}, 4: {436, 300}}
	selector := 0
	pointer := func(code byte, sel int) {
		t.Helper()
		selector = sel
		// Exact installed mask byte, not the production code->control resolver.
		for y := 0; y < 480; y++ {
			for x := 0; x < 640; x++ {
				if f.TownSquareArt.Value().Mask.ColorIndexAt(x, y) == code {
					if e := a.HeadlessPointer("hover", x, y); e != nil {
						t.Fatal(e)
					}
					return
				}
			}
		}
		t.Fatalf("mask lacks byte%x", code)
	}
	families := loadTownSquareOracle(t, f)
	checks, changed := 0, 0
	var first *image.RGBA
	check := func(name string, want exteriorFrame, dt time.Duration) {
		t.Helper()
		pix := exteriorPaint(t, a, &now, dt)
		got := s.sqExteriorFrame()
		// The three wildlife families have their own installed witness.
		want.Horse, want.Baba, want.Dervish = got.Horse, got.Baba, got.Dervish
		if got != want {
			t.Fatalf("%s frame %+v want %+v", name, got, want)
		}
		expected := image.NewRGBA(image.Rect(0, 0, 640, 480))
		draw.Draw(expected, expected.Bounds(), base, image.Point{}, draw.Src)
		put := func(pic image.Image, p image.Point) {
			draw.Draw(expected, pic.Bounds().Add(p), pic, image.Point{}, draw.Over)
		}
		if pic := labels[selector]; pic != nil {
			put(pic, labelPoints[selector])
		}
		for _, layer := range []struct {
			name string
			i    int
			p    image.Point
		}{
			{"tavern", want.Tavern, image.Pt(124, 312)}, {"sign", want.Sign, image.Pt(360, 232)}, {"door", want.Door, image.Pt(180, 148)},
			{"fighter", want.Fighter, image.Pt(516, 344)}, {"mage", want.Mage, image.Pt(452, 328)}, {"shopie", want.Shop, image.Pt(276, 296)},
			{"fluger", want.Fluger, image.Pt(308, 64)}, {"guards", want.Guard, image.Pt(184, 158)},
		} {
			put(sheets[layer.name][layer.i], layer.p)
		}
		families.drawFamilies(expected, got)
		if !bytes.Equal(pix.Pix, expected.Pix) {
			for y := 0; y < 480; y++ {
				for x := 0; x < 640; x++ {
					if pix.RGBAAt(x, y) != expected.RGBAAt(x, y) {
						t.Fatalf("%s pixel%d,%d=%v want%v", name, x, y, pix.RGBAAt(x, y), expected.RGBAAt(x, y))
					}
				}
			}
		}
		checks++
		if first == nil {
			first = pix
		} else {
			for i := 0; i < len(pix.Pix); i += 4 {
				if !bytes.Equal(pix.Pix[i:i+4], first.Pix[i:i+4]) {
					changed++
				}
			}
		}
		if name != "" {
			writeTownExteriorWitness(t, f, name, pix)
		}
	}
	w := exteriorFrame{Door: 8, Guard: 7}
	check("entry", w, 0)
	pointer(0x80, 2)
	for i := 1; i <= 10; i++ {
		w.Tavern = i % 10
		name := ""
		if i == 4 {
			name = "tavern-04"
		}
		check(name, w, 68*time.Millisecond)
	}
	roll = 99
	pointer(0x90, 1)
	roll = 0
	for i := 1; i <= 30; i++ {
		w.Shop = i % 30
		name := ""
		if i == 15 {
			name = "shop-15"
		}
		check(name, w, 68*time.Millisecond)
	}
	roll = 99
	pointer(0xc0, 4)
	w.Fighter, w.Mage, w.Sign, w.Fluger = 1, 1, 1, 1
	check("school-01-sign-fluger", w, 68*time.Millisecond)
	roll = 0
	for i := 2; i <= 10; i++ {
		w.Fighter, w.Mage = i, i
		w.Sign = i % 10
		w.Fluger = 0
		if i < 8 {
			w.Fluger = i
		}
		check("", w, 68*time.Millisecond)
	}
	check("school-10", w, 68*time.Millisecond)
	pointer(0, 0)
	// No pointer-dependent action arms the two ambient sequences.
	roll = 99
	w.Fighter, w.Mage, w.Sign, w.Fluger = 9, 9, 1, 1
	check("school-return-09", w, 68*time.Millisecond)
	roll = 0
	for i := 8; i >= 0; i-- {
		j := 10 - i
		w.Fighter, w.Mage = i, i
		w.Sign = j % 10
		w.Fluger = 0
		if j < 8 {
			w.Fluger = j
		}
		check("", w, 68*time.Millisecond)
	}
	check("school-rest", w, 68*time.Millisecond)
	pointer(0xa0, 8)
	for i := 7; i >= 0; i-- {
		w.Door = i
		name := ""
		if i == 4 {
			name = "gate-04"
		}
		check(name, w, 68*time.Millisecond)
	}
	pointer(0, 0)
	w.Door = 1
	check("gate-reverse", w, 68*time.Millisecond)
	pointer(0xa0, 8)
	w.Door = 0
	check("gate-reenter", w, 68*time.Millisecond)
	// All nine selected sounds must be the actual installed decoded waveforms.
	// Guard1 needs the separately tested unavailable-gate boundary below.
	clearTownTestGateLatches(f.Town)
	pointer(0xa0, 8)
	w.Door, w.Guard = 8, 6
	check("guard-blocked", w, 68*time.Millisecond)
	for i := 5; i >= 0; i-- {
		w.Guard = i
		check("", w, 68*time.Millisecond)
	}
	check("guard-rest", w, 68*time.Millisecond)
	f.Town.announceMission(f.Town.currentMain())
	after, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	afterBytes, err := EncodeSave(after, "same")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(initialBytes, afterBytes) {
		t.Fatal("installed animation trace changed native save")
	}
	sfx, err := OpenContainers(filepath.Join(f.Archives.Root, "sfx.res"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range exteriorSoundPaths {
		raw, e := sfx.ReadFile("sfx/" + path)
		if e != nil {
			t.Fatal(e)
		}
		want, e := audio.DecodeWAV(raw, audio.DeviceRate)
		if e != nil {
			t.Fatal(e)
		}
		found := false
		for _, sample := range r.samples {
			if reflect.DeepEqual(sample, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing exact conditional waveform %s", path)
		}
	}
	for _, p := range r.places {
		if p != (audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}) {
			t.Fatal(p)
		}
	}
	if checks < 70 || changed == 0 {
		t.Fatal("insufficient moving composition witness")
	}
	t.Logf("1120: %d full App.Draw/CPU composed frames, %d pixel comparisons; %d changed pixel observations; 9 installed conditional sound waveforms; native bytes unchanged", checks, checks*640*480, changed)
}

func writeTownExteriorWitness(t *testing.T, f *FrontEnd, name string, pix *image.RGBA) {
	t.Helper()
	dir := os.Getenv("AGAINROM_TOWN_EXTERIOR_PNG")
	if dir == "" {
		return
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(f.Archives.Root, dir)
	if err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatal("capture must stay outside install")
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(file, pix)
	closeErr := file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}
