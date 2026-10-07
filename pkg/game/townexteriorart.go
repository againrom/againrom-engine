package game

import (
	"fmt"
	"image"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// TOWN-005/006/400..404. Each family loads atomically, independently of both
// the static square and the other families. Invalid motion never blocks entry.
func loadTownExteriorArt(src terrain.EntrySource) (*ui.TownExteriorArt, []string) {
	a := &ui.TownExteriorArt{}
	var problems []string
	for _, spec := range []struct {
		name  string
		count int
		out   *[]image.Image
	}{
		{"shopie", 30, &a.Shop}, {"tavern", 10, &a.Tavern},
		{"fighter", 11, &a.Fighter}, {"mage", 11, &a.Mage}, {"guards", 8, &a.Guard},
	} {
		path := graphicsPrefix + "interface/townbirds/" + spec.name + "/sprites.16a"
		frames := effectFrames(src, path)
		valid := len(frames) == spec.count
		for _, f := range frames {
			valid = valid && f.Width > 0 && f.Height > 0 && f.Width <= 640 && f.Height <= 480
		}
		if !valid {
			problems = append(problems, fmt.Sprintf("%s: expected %d readable frames", path, spec.count))
			continue
		}
		for _, f := range frames {
			*spec.out = append(*spec.out, f.RGBA())
		}
	}
	// TOWN-415. Each .16a is one independent 57-frame family. A corrupt
	// Birds4 must not discard Birds1..3 or Birds5..9; selection keeps the
	// original three-family grouping and an absent selected family simply
	// contributes no picture to that episode.
	for family := range a.Birds {
		path := graphicsPrefix + fmt.Sprintf("interface/townbirds/birds%d/sprites.16a", family+1)
		decoded := effectFrames(src, path)
		valid := len(decoded) == 57
		for _, f := range decoded {
			valid = valid && f.Width > 0 && f.Height > 0 && f.Width <= 640 && f.Height <= 480
		}
		if !valid {
			problems = append(problems, fmt.Sprintf("%s: expected 57 readable frames", path))
			continue
		}
		for _, f := range decoded {
			a.Birds[family] = append(a.Birds[family], f.RGBA())
		}
	}
	// TOWN-439/006. Horse, baba and dervish sheets load one by one: every
	// position's horse A1..A3 (15 frames), baba A1/A2 (31/32) and dervish
	// (30). A sheet that fails leaves only that nil.
	loadSheet := func(path string, count int, out *[]image.Image) {
		decoded := effectFrames(src, path)
		valid := len(decoded) == count
		for _, f := range decoded {
			valid = valid && f.Width > 0 && f.Height > 0 && f.Width <= 640 && f.Height <= 480
		}
		if !valid {
			problems = append(problems, fmt.Sprintf("%s: expected %d readable frames", path, count))
			return
		}
		for _, f := range decoded {
			*out = append(*out, f.RGBA())
		}
	}
	for p := range a.Horse {
		for v := range a.Horse[p] {
			loadSheet(graphicsPrefix+fmt.Sprintf("interface/townbirds/horse%d/a%d/sprites.16a", p+1, v+1), 15, &a.Horse[p][v])
		}
	}
	for p := range a.Baba {
		for v := range a.Baba[p] {
			loadSheet(graphicsPrefix+fmt.Sprintf("interface/townbirds/baba%d/a%d/sprites.16a", p+1, v+1), 31+v, &a.Baba[p][v])
		}
	}
	for p := range a.Dervish {
		loadSheet(graphicsPrefix+fmt.Sprintf("interface/townbirds/dervish%d/sprites.16a", p+1), 30, &a.Dervish[p])
	}
	for _, spec := range []struct {
		pattern     string
		count, w, h int
		out         *[]image.Image
	}{
		{"door/t%02d.bmp", 9, 36, 48, &a.Door},
		{"sign/v%02d.bmp", 10, 40, 32, &a.Sign},
		{"fluger/f%02d.bmp", 8, 64, 64, &a.Fluger},
	} {
		var frames []image.Image
		for i := 0; i < spec.count; i++ {
			path := townSquareArtPrefix + fmt.Sprintf(spec.pattern, i)
			pic, err := readChargenBMP(src, path)
			if err == nil {
				err = chargenSize(pic, spec.w, spec.h, path)
			}
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s: %v", path, err))
				break
			}
			frames = append(frames, pic) // original opaque BMP patches
		}
		if len(frames) == spec.count {
			*spec.out = frames
		}
	}
	// TOWN-418. The selected pointer ranges over one nine-picture array, so
	// the bitmap family is atomic even though every bird sheet above has its
	// own independent fallback boundary.
	var stars []image.Image
	for i := 0; i < 9; i++ {
		path := townSquareArtPrefix + fmt.Sprintf("stars/s%02d.bmp", i)
		pic, err := readChargenBMP(src, path)
		if err == nil {
			err = chargenSize(pic, 64, 44, path)
		}
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", path, err))
			stars = nil
			break
		}
		stars = append(stars, pic)
	}
	if len(stars) == 9 {
		a.Stars = stars
	}
	return a, problems
}
