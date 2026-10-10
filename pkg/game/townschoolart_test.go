package game

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"
)

// synthBMPBlack is synthBMP with the first n pixels of the raster forced to
// pure black, so the keying rule has a population to act on. Position is not
// asserted anywhere: the tests count the keyed pixels.
func synthBMPBlack(w, h int, c color.RGBA, n int) []byte {
	b := synthBMP(w, h, c)
	const header = 54
	stride := (w*3 + 3) &^ 3
	for i := 0; i < n; i++ {
		x, y := i%w, i/w
		p := header + y*stride + x*3
		b[p], b[p+1], b[p+2] = 0, 0, 0
	}
	return b
}

const townSchoolBlackPixels = 37

func townSchoolSource() chargenSource {
	src := chargenSource{
		townSchoolArtPrefix + "trnhall.bmp":     synthBMP(480, 480, color.RGBA{R: 0x31, A: 0xff}),
		townSchoolArtPrefix + "buttonsarea.bmp": synthBMP(160, 238, color.RGBA{G: 0x31, A: 0xff}),
		townTavernArtPrefix + "ruover.bmp":      synthBMP(16, 238, color.RGBA{B: 0x31, A: 0xff}),
	}
	for _, name := range []string{"b1", "b2"} {
		for _, suffix := range []string{"off", "on"} {
			src[townSchoolArtPrefix+"buttons/"+name+suffix+".bmp"] = synthBMP(140, 46, color.RGBA{B: 0x22, A: 0xff})
		}
	}
	addSchoolDiamondFixture(src)
	for frame := 0; frame < 16; frame++ {
		src[fmt.Sprintf("%scolumn/rt%04d.bmp", townSchoolArtPrefix, frame)] =
			synthBMP(148, 208, color.RGBA{R: uint8(0x40 + frame), G: 0x22, B: 0x11, A: 0xff})
	}
	classes := []struct {
		dir    string
		skills []string
		w      int
	}{
		{"fighter", []string{"sword", "axe", "club", "pike", "bow"}, 92},
		{"mage", []string{"fire", "water", "air", "earth", "astral"}, 100},
	}
	for _, class := range classes {
		base := townSchoolArtPrefix + "column/" + class.dir + "/"
		src[base+"mask.bmp"] = synthBMP8(class.w, 120, 0x37, 0x87, 0x9e, 0xd2, 0xff)
		for _, skill := range class.skills {
			for _, state := range []string{"on", "shine", "shine_on"} {
				src[base+skill+"/"+state+".bmp"] =
					synthBMPBlack(20, 20, color.RGBA{B: 0x31, A: 0xff}, townSchoolBlackPixels)
			}
		}
	}
	return src
}

func townSchoolTrainingSource() chargenSource {
	src := townSchoolSource()
	for class, shape := range schoolTrainingFamilyShape {
		base := townSchoolMoviesPrefix + shape.dir + "/"
		for frame := 0; frame <= shape.transitionLast; frame++ {
			c := color.RGBA{R: uint8(10 + frame), A: 0xff}
			if class == schoolMageClass {
				c = color.RGBA{B: uint8(20 + frame), A: 0xff}
			}
			src[fmt.Sprintf(base+"tr%04d.bmp", frame)] = synthBMP(shape.width, 224, c)
		}
		for frame := 1; frame <= shape.idleLast; frame++ {
			c := color.RGBA{G: uint8(30 + frame), A: 0xff}
			if class == schoolMageClass {
				c = color.RGBA{R: uint8(100 + frame), G: 1, A: 0xff}
			}
			src[fmt.Sprintf(base+"m%04d.bmp", frame)] = synthBMP(shape.width, 224, c)
		}
	}
	return src
}

func TestLoadTownSchoolTrainingAllSixtyTwoFramesAndIndependentFamilies(t *testing.T) {
	art, err := LoadTownSchoolArt(ROM1TownDescription(), townSchoolTrainingSource())
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for class, shape := range schoolTrainingFamilyShape {
		families := []struct {
			name   string
			frames []image.Image
			count  int
			first  int
		}{
			{"tr", art.Scene[schoolMovieNames[class][0]], shape.transitionLast + 1, 0},
			{"m", art.Scene[schoolMovieNames[class][1]], shape.idleLast, 1},
		}
		for _, family := range families {
			if len(family.frames) != family.count {
				t.Fatalf("%s/%s count = %d, want %d", shape.dir, family.name, len(family.frames), family.count)
			}
			for i, frame := range family.frames {
				if frame == nil || frame.Bounds().Dx() != shape.width || frame.Bounds().Dy() != 224 {
					t.Fatalf("%s/%s%04d = %v", shape.dir, family.name, family.first+i, frame)
				}
				total++
			}
		}
	}
	if total != 62 {
		t.Fatalf("training population = %d, want 62", total)
	}

	for _, missing := range []struct {
		class, family int
		path          string
	}{
		{schoolFighterClass, 0, townSchoolMoviesPrefix + "fighter/tr0007.bmp"},
		{schoolFighterClass, 1, townSchoolMoviesPrefix + "fighter/m0004.bmp"},
		{schoolMageClass, 0, townSchoolMoviesPrefix + "mage/tr0011.bmp"},
		{schoolMageClass, 1, townSchoolMoviesPrefix + "mage/m0006.bmp"},
	} {
		src := townSchoolTrainingSource()
		delete(src, missing.path)
		got, err := LoadTownSchoolArt(ROM1TownDescription(), src)
		if err != nil {
			t.Fatalf("optional missing %s failed base room: %v", missing.path, err)
		}
		for class, shape := range schoolTrainingFamilyShape {
			for family, frames := range [][]image.Image{got.Scene[schoolMovieNames[class][0]], got.Scene[schoolMovieNames[class][1]]} {
				want := shape.transitionLast + 1
				if family == 1 {
					want = shape.idleLast
				}
				if class == missing.class && family == missing.family {
					want = 0
				}
				if len(frames) != want {
					t.Fatalf("missing %s changed class%d/family%d to %d frames, want %d", missing.path, class, family, len(frames), want)
				}
			}
		}
	}

	src := townSchoolTrainingSource()
	src[townSchoolMoviesPrefix+"mage/m0009.bmp"] = synthBMP(171, 224, color.RGBA{A: 0xff})
	got, err := LoadTownSchoolArt(ROM1TownDescription(), src)
	if err != nil || got.Scene["mage-m"] != nil || len(got.Scene["mage-tr"]) != 23 || len(got.Scene["fighter-m"]) != 9 {
		t.Fatalf("malformed mage/m degraded outside its family: err=%v art=%d entries", err, len(got.Scene))
	}
}

func TestLoadTownSchoolArtReadsBothClassMappings(t *testing.T) {
	src := townSchoolSource()
	got, err := LoadTownSchoolArt(ROM1TownDescription(), src)
	if err != nil {
		t.Fatal(err)
	}
	if got.Background == nil || got.Upper == nil || got.Masks[0] == nil || got.Masks[1] == nil ||
		got.Skills[0][4][2] == nil || got.Skills[1][4][2] == nil {
		t.Fatal("complete school art was not retained")
	}
	delete(src, townSchoolArtPrefix+"column/mage/astral/shine_on.bmp")
	if _, err := LoadTownSchoolArt(ROM1TownDescription(), src); err == nil || !strings.Contains(err.Error(), "astral/shine_on.bmp") {
		t.Fatalf("missing art error = %v", err)
	}
}

// TestLoadTownSchoolArtReadsOneRestFacePerClass covers the two column frames
// the room draws over its own background: the fighter rests at rt0000 and the
// mage at rt0015, each 148x208. A missing or mis-sized frame is an
// address-bearing error beside a nil result, which is what NewFrontEnd needs to
// keep a broken install playable.
func TestLoadTownSchoolArtReadsOneRestFacePerClass(t *testing.T) {
	src := townSchoolSource()
	got, err := LoadTownSchoolArt(ROM1TownDescription(), src)
	if err != nil {
		t.Fatal(err)
	}
	for class, face := range got.Faces {
		if face == nil {
			t.Fatalf("class %d has no rest face", class)
		}
		if w, h := face.Bounds().Dx(), face.Bounds().Dy(); w != 148 || h != 208 {
			t.Fatalf("class %d face is %dx%d, want 148x208", class, w, h)
		}
	}
	if got.Faces[0].At(3, 3) == got.Faces[1].At(3, 3) {
		t.Fatal("both classes resolved the same rotation frame")
	}

	src[townSchoolArtPrefix+"column/rt0015.bmp"] = synthBMP(148, 200, color.RGBA{A: 0xff})
	if _, err := LoadTownSchoolArt(ROM1TownDescription(), src); err == nil || !strings.Contains(err.Error(), "rt0015.bmp") ||
		!strings.Contains(err.Error(), "148x200") {
		t.Fatalf("mis-sized face error = %v", err)
	}
	delete(src, townSchoolArtPrefix+"column/rt0000.bmp")
	if _, err := LoadTownSchoolArt(ROM1TownDescription(), src); err == nil || !strings.Contains(err.Error(), "rt0000.bmp") {
		t.Fatalf("missing face error = %v", err)
	}
}

// TestLoadTownSchoolArtKeysPureBlackOnThePatchesOnly covers the transparency
// rule. Pure black is the transparent colour of these 24-bit DIBs, so every
// black pixel of a skill patch loses its alpha; the rest faces stand in for the
// background where they are drawn and keep every pixel opaque.
func TestLoadTownSchoolArtKeysPureBlackOnThePatchesOnly(t *testing.T) {
	got, err := LoadTownSchoolArt(ROM1TownDescription(), townSchoolSource())
	if err != nil {
		t.Fatal(err)
	}
	for class := 0; class < 2; class++ {
		for slot := 0; slot < 5; slot++ {
			for state := 0; state < 3; state++ {
				pic := got.Skills[class][slot][state]
				clear, opaque := 0, 0
				b := pic.Bounds()
				for y := b.Min.Y; y < b.Max.Y; y++ {
					for x := b.Min.X; x < b.Max.X; x++ {
						if _, _, _, a := pic.At(x, y).RGBA(); a == 0 {
							clear++
						} else {
							opaque++
						}
					}
				}
				if clear != townSchoolBlackPixels {
					t.Fatalf("class %d slot %d state %d: %d transparent pixels, want %d",
						class, slot, state, clear, townSchoolBlackPixels)
				}
				if opaque != 20*20-townSchoolBlackPixels {
					t.Fatalf("class %d slot %d state %d: %d opaque pixels", class, slot, state, opaque)
				}
			}
		}
	}
	for class, face := range got.Faces {
		b := face.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if _, _, _, a := face.At(x, y).RGBA(); a == 0 {
					t.Fatalf("class %d rest face is transparent at %d,%d", class, x, y)
				}
			}
		}
	}
}
