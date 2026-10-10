package game

import (
	"fmt"
	"image"

	"againrom/pkg/render/terrain"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

const townSchoolArtPrefix = graphicsPrefix + "interface/training/"

const (
	schoolFighterClass = iota
	schoolMageClass
	schoolClassCount
)

// LoadTownSchoolArt resolves the school's immutable surface. It is cosmetic:
// NewFrontEnd carries an error beside a nil result so a missing picture never
// makes the game unusable, while a complete install performs no archive reads
// in the draw or hit-test path.
//
// The school scene's art is room description d's: the required column
// and diamond series fail the load, and each optional training family is
// dropped alone. The column's endpoints are the two rest faces.
//
// Skill patches use a source-16-bit-zero keyed blit; the rest faces are opaque
// (TOWN-150). keyBlack (shopart.go) instead tests source RGB for pure black.
// cmd/schoolcheck -transparency-only compares those populations over all 30
// shipped patches under standard RGB565/RGB555 channel truncation
// (TERR-LIGHT-019). This is not arbitrary runtime framebuffer-mask support:
// modded near-black RGB can quantize to zero without being pure black.
func LoadTownSchoolArt(d *town.Description, src terrain.EntrySource) (*ui.TownSchoolArt, error) {
	a := &ui.TownSchoolArt{}
	var err error
	if a.Background, err = readChargenBMP(src, townSchoolArtPrefix+"trnhall.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(a.Background, 480, 480, townSchoolArtPrefix+"trnhall.bmp"); err != nil {
		return nil, err
	}
	if a.Scene, err = loadRoomSceneArt(d, "school", src); a.Scene == nil {
		return nil, err
	}
	column := a.Scene["column"]
	a.Faces = [2]image.Image{column[0], column[len(column)-1]}
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
	return a, nil
}
