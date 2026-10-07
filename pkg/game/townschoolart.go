package game

import (
	"fmt"
	"image"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

const townSchoolArtPrefix = graphicsPrefix + "interface/training/"
const townSchoolMoviesPrefix = moviesPrefix + "training/"

const (
	schoolFighterClass = iota
	schoolMageClass
	schoolClassCount
)

var schoolTrainingFamilyShape = [schoolClassCount]struct {
	dir            string
	width          int
	transitionLast int
	idleLast       int
}{
	{dir: "fighter", width: 160, transitionLast: 18, idleLast: 9},
	{dir: "mage", width: 172, transitionLast: 22, idleLast: 11},
}

// LoadTownSchoolArt resolves the school's immutable surface. It is cosmetic:
// NewFrontEnd carries an error beside a nil result so a missing picture never
// makes the game unusable, while a complete install performs no archive reads
// in the draw or hit-test path.
//
// TOWN-146/147/149: all sixteen 148x208 column frames are retained. Frame 0
// is the fighter endpoint, frame 15 the mage endpoint. The fourteen interior
// frames carry no skill panel. No archive read occurs during rotation.
//
// Skill patches use a source-16-bit-zero keyed blit; the rest faces are opaque
// (TOWN-150). keyBlack (shopart.go) instead tests source RGB for pure black.
// cmd/schoolcheck -transparency-only compares those populations over all 30
// shipped patches under standard RGB565/RGB555 channel truncation
// (TERR-LIGHT-019). This is not arbitrary runtime framebuffer-mask support:
// modded near-black RGB can quantize to zero without being pure black.
func LoadTownSchoolArt(src terrain.EntrySource) (*ui.TownSchoolArt, error) {
	a := &ui.TownSchoolArt{}
	var err error
	if a.Background, err = readChargenBMP(src, townSchoolArtPrefix+"trnhall.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(a.Background, 480, 480, townSchoolArtPrefix+"trnhall.bmp"); err != nil {
		return nil, err
	}
	for frame := range a.Column {
		addr := fmt.Sprintf("%scolumn/rt%04d.bmp", townSchoolArtPrefix, frame)
		if a.Column[frame], err = readChargenBMP(src, addr); err != nil {
			return nil, err
		}
		if err = chargenSize(a.Column[frame], 148, 208, addr); err != nil {
			return nil, err
		}
	}
	a.Faces = [2]image.Image{a.Column[0], a.Column[15]}
	// TOWN-154: nine opaque 80x76 frames, in archive-number order.
	for frame := range a.Diamond {
		addr := fmt.Sprintf("%sdiamond/on%04d.bmp", townSchoolArtPrefix, frame)
		if a.Diamond[frame], err = readChargenBMP(src, addr); err != nil {
			return nil, err
		}
		if err = chargenSize(a.Diamond[frame], 80, 76, addr); err != nil {
			return nil, err
		}
	}
	if a.Upper, err = readChargenBMP(src, townSchoolArtPrefix+"buttonsarea.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(a.Upper, 160, 238, townSchoolArtPrefix+"buttonsarea.bmp"); err != nil {
		return nil, err
	}
	// UpperSeam is read from the inn directory, not the training one: the
	// school ships no 16-wide strip of its own (DIV-166's own census over
	// both roots finds none under training/), and rendering ruover.bmp
	// beside the school's own buttonsarea.bmp shows no visible join, unlike
	// the two other 238-row candidates, luover.bmp and chrgen/rollstatsr.bmp
	// (DIV-168, cmd/plaqueseams).
	if a.UpperSeam, err = readChargenBMP(src, townTavernArtPrefix+"ruover.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(a.UpperSeam, 16, 238, townTavernArtPrefix+"ruover.bmp"); err != nil {
		return nil, err
	}
	a.UpperSeam = keyBlack(a.UpperSeam.(*image.RGBA))
	// The two shipped buttons, Train and Exit, each an off/on pair (1017;
	// DIV-159). TOWN-154 names the sizes and the filenames' own archive
	// destinations; which one is Train and which is Exit is not decoded and
	// is read off the school's own well order top to bottom, matching the
	// button dispatcher in pkg/game/townshell.go.
	for i, name := range []string{"b1", "b2"} {
		for state, suffix := range []string{"off", "on"} {
			addr := townSchoolArtPrefix + "buttons/" + name + suffix + ".bmp"
			if a.Buttons[i][state], err = readChargenBMP(src, addr); err != nil {
				return nil, err
			}
			if err = chargenSize(a.Buttons[i][state], 140, 46, addr); err != nil {
				return nil, err
			}
		}
	}

	classes := []struct {
		dir    string
		skills [5]string
		w      int
	}{
		{dir: "fighter", skills: [5]string{"sword", "axe", "club", "pike", "bow"}, w: 92},
		{dir: "mage", skills: [5]string{"fire", "water", "air", "earth", "astral"}, w: 100},
	}
	for class, row := range classes {
		maskAddr := townSchoolArtPrefix + "column/" + row.dir + "/mask.bmp"
		if a.Masks[class], err = readChargenMask(src, maskAddr); err != nil {
			return nil, err
		}
		if err = chargenSize(a.Masks[class], row.w, 120, maskAddr); err != nil {
			return nil, err
		}
		if err = chargenMaskCodes(a.Masks[class], []uint8{0x37, 0x87, 0x9e, 0xd2, 0xff}, maskAddr); err != nil {
			return nil, err
		}
		for slot, skill := range row.skills {
			for state, name := range []string{"on", "shine", "shine_on"} {
				addr := townSchoolArtPrefix + "column/" + row.dir + "/" + skill + "/" + name + ".bmp"
				pic, err := readChargenBMP(src, addr)
				if err != nil {
					return nil, err
				}
				a.Skills[class][slot][state] = keyBlack(pic)
				if a.Skills[class][slot][state].Bounds().Dx() > 80 || a.Skills[class][slot][state].Bounds().Dy() > 36 {
					return nil, fmt.Errorf("%s: skill image is %v, outside the shared 80x36 envelope", addr, a.Skills[class][slot][state].Bounds().Size())
				}
			}
		}
	}
	// TOWN-427/428: the four movies.res families are optional presentation
	// children of this otherwise-required room art. Each family is atomic and
	// independent: one missing or malformed member disables only that family,
	// while complete siblings and every existing school control remain usable.
	for class, shape := range schoolTrainingFamilyShape {
		base := townSchoolMoviesPrefix + shape.dir + "/"
		a.Training[class].Transition = loadSchoolTrainingFamily(src, base+"tr%04d.bmp", 0, shape.transitionLast+1, shape.width)
		a.Training[class].Idle = loadSchoolTrainingFamily(src, base+"m%04d.bmp", 1, shape.idleLast, shape.width)
	}
	return a, nil
}

// loadSchoolTrainingFamily admits no partial sequence. All four shipped
// groups are opaque 224-row BMPs; fighter is 160 wide and mage is 172 wide.
// A failure is cosmetic absence rather than a LoadTownSchoolArt error because
// the base school was already usable before these optional movie families.
func loadSchoolTrainingFamily(src terrain.EntrySource, pattern string, first, count, width int) []image.Image {
	frames := make([]image.Image, count)
	for i := range frames {
		addr := fmt.Sprintf(pattern, first+i)
		pic, err := readChargenBMP(src, addr)
		if err != nil || pic == nil || pic.Bounds().Dx() != width || pic.Bounds().Dy() != 224 {
			return nil
		}
		frames[i] = pic
	}
	return frames
}
