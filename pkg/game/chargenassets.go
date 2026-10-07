package game

import (
	"fmt"
	"image"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

const (
	chargenPlatePath     = mainPrefix + "graphics/chrgen/leftup.bmp"
	chargenPrecreatePath = graphicsPrefix + "interface/chrgen/precreate/"
	chargenTextPath      = mainPrefix + "text/main.txt"
	chargenPromptSlot    = 125
	chargenSkillSlot     = 171
	chargenEmptyNameSlot = 193
	chargenReservedSlot  = 194
	chargenPlaySlot      = 238
	chargenResetSlot     = 239
	chargenBackSlot      = 260
	chargenNamesPath     = mainPrefix + "text/npcnames.txt"
	// chargenUnnamed is the name field's seed when no name was given; the
	// page's enter turns it into the first picture's name (TEXT-073).
	chargenUnnamed = "Unnamed"
)

// chargenPictureNameEntries are the npcnames.txt entries a hero press writes,
// in the picture order male fighter, male mage, female fighter, female mage
// (TEXT-074).
var chargenPictureNameEntries = [4]int{20, 22, 21, 23}

var chargenPreChoiceOrigins = [...]image.Point{{16, 273}, {416, 190}, {124, 166}, {288, 130}}
var chargenPreMaskCodes = [...]uint8{80, 140, 100, 120, 160, 180}

var chargenDetailedSkillOrigins = [2][5]image.Point{
	{{88, 93}, {92, 126}, {88, 182}, {84, 225}, {88, 250}},
	{{200, 150}, {72, 165}, {132, 98}, {140, 228}, {136, 158}},
}

var chargenDetailedMaskCodes = [2][5]uint8{
	{255, 191, 152, 127, 102},
	{127, 102, 255, 152, 191},
}

// ChargenAssets is the startup-resolved source payload for character
// generation. Strings retain their original bytes; render/text applies the
// install selector when they are drawn.
type ChargenAssets struct {
	Presentation *ui.ChargenPresentation
	Prompt       string
	Back         string
	Play         string
	Reset        string
	EmptyName    string
	ReservedName string
	SkillHover   [2][5]string
	Selector     int
	// HeroNames are the pictures' npcnames.txt entries, in picture order.
	HeroNames [4]string
}

func chargenRGBA(data []byte, addr string) (*image.RGBA, error) {
	b, err := bmp.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	return portraitRGBA(b), nil
}

func readChargenBMP(src terrain.EntrySource, addr string) (*image.RGBA, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no archive", addr)
	}
	b, err := src.ReadFile(addr)
	if err != nil {
		return nil, err
	}
	return chargenRGBA(b, addr)
}

func readChargenMask(src terrain.EntrySource, addr string) (*image.Paletted, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no archive", addr)
	}
	b, err := src.ReadFile(addr)
	if err != nil {
		return nil, err
	}
	mask, err := terrain.DecodeBMP8(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	return mask, nil
}

func chargenSize(pic image.Image, wantW, wantH int, addr string) error {
	if pic == nil || pic.Bounds().Dx() != wantW || pic.Bounds().Dy() != wantH {
		if pic == nil {
			return fmt.Errorf("%s: missing image", addr)
		}
		return fmt.Errorf("%s: size %dx%d, want %dx%d", addr, pic.Bounds().Dx(), pic.Bounds().Dy(), wantW, wantH)
	}
	return nil
}

func chargenPatchFits(pic image.Image, at image.Point, bounds image.Rectangle, addr string) error {
	if pic == nil || !pic.Bounds().Add(at).In(bounds) {
		if pic == nil {
			return fmt.Errorf("%s: missing patch", addr)
		}
		return fmt.Errorf("%s: patch %v at %v lies outside %v", addr, pic.Bounds(), at, bounds)
	}
	return nil
}

// chargenMaskCodes makes the source mask part of the mandatory presentation
// contract. A correctly sized but blank mask would otherwise load and leave
// every pictorial choice inert at runtime.
func chargenMaskCodes(mask *image.Paletted, want []uint8, addr string) error {
	if mask == nil {
		return fmt.Errorf("%s: missing mask", addr)
	}
	seen := [256]bool{}
	for _, index := range mask.Pix {
		seen[index] = true
	}
	for _, index := range want {
		if !seen[index] {
			return fmt.Errorf("%s: missing required mask index %d", addr, index)
		}
	}
	return nil
}

// chargenTextAt is one required generator slot, or the address-bearing error a
// missing one is.
//
// IT READS THE SHARED TABLE. Before that story this file carried its own
// splitter, which guessed a NUL separator when the payload held more than
// one NUL and split on LF otherwise. Neither rule is decoded.
//
// THE GENERATOR STILL FAILS ON AN ABSENT SLOT while every word 0168 added falls
// back to English. The difference is deliberate and is LoadChargenAssets' own
// stated rule: presenting a partial generator would look like an authored
// screen, where a menu row in the wrong language plainly is not.
func chargenTextAt(t *TextTable, path string, slot int) (string, error) {
	s, ok := t.At(slot)
	if !ok {
		return "", fmt.Errorf("%s slot %d is absent", path, slot)
	}
	return s, nil
}

// heroPictureNames is the four pictures' npcnames.txt entries in picture
// order, or the address-bearing error a missing one is. The name field's hero
// press and the party a hero started without the generator carries both read
// their names here, so the two cannot name one picture differently.
func heroPictureNames(src entrySource) ([4]string, error) {
	var names [4]string
	b, err := src.ReadFile(chargenNamesPath)
	if err != nil {
		return names, err
	}
	rows := SplitTextTable(b)
	for i, entry := range chargenPictureNameEntries {
		if names[i], err = chargenTextAt(rows, chargenNamesPath, entry); err != nil {
			return [4]string{}, err
		}
	}
	return names, nil
}

// LoadChargenAssets reads the fixed generator presentation once. Required
// controls and words fail as address-bearing construction errors: presenting a
// partial source UI would look like an authored fallback.
func LoadChargenAssets(src terrain.EntrySource) (*ChargenAssets, error) {
	p := &ui.ChargenPresentation{}
	var err error
	// TOWN-223 names this independent animated decoration. It is optional
	// for incomplete diagnostic archives; a full install supplies 15 frames.
	if src != nil {
		if raw, readErr := src.ReadFile(chargenPrecreatePath + "blind/sprites.16a"); readErr == nil {
			p.Sparkles, _ = decodeCursor16AFrames(chargenPrecreatePath+"blind/sprites.16a", raw)
		}
	}
	for level, size := range [3]image.Point{{60, 74}, {76, 112}, {100, 152}} {
		for state, suffix := range []string{"on", "l", "lon"} {
			addr := fmt.Sprintf("%slevels/level%d%s.bmp", chargenPrecreatePath, level, suffix)
			if p.Levels[level][state], err = readChargenBMP(src, addr); err != nil {
				return nil, err
			}
			if err = chargenSize(p.Levels[level][state], size.X, size.Y, addr); err != nil {
				return nil, err
			}
		}
	}
	if p.Background, err = readChargenBMP(src, chargenPrecreatePath+"mainarea.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(p.Background, 640, 480, chargenPrecreatePath+"mainarea.bmp"); err != nil {
		return nil, err
	}
	if p.PreMask, err = readChargenMask(src, chargenPrecreatePath+"mask.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(p.PreMask, 640, 480, chargenPrecreatePath+"mask.bmp"); err != nil {
		return nil, err
	}
	if err = chargenMaskCodes(p.PreMask, chargenPreMaskCodes[:], chargenPrecreatePath+"mask.bmp"); err != nil {
		return nil, err
	}
	if p.Amulet, err = readChargenBMP(src, chargenPrecreatePath+"amulet.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(p.Amulet, 112, 204, chargenPrecreatePath+"amulet.bmp"); err != nil {
		return nil, err
	}
	if err = chargenPatchFits(p.Amulet, image.Pt(528, 140), image.Rect(0, 0, 640, 480), chargenPrecreatePath+"amulet.bmp"); err != nil {
		return nil, err
	}
	if p.Forward, err = readChargenBMP(src, chargenPrecreatePath+"buttonok.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(p.Forward, 100, 56, chargenPrecreatePath+"buttonok.bmp"); err != nil {
		return nil, err
	}
	if err = chargenPatchFits(p.Forward, image.Pt(468, 373), image.Rect(0, 0, 640, 480), chargenPrecreatePath+"buttonok.bmp"); err != nil {
		return nil, err
	}
	if p.Plate, err = readChargenBMP(src, chargenPlatePath); err != nil {
		return nil, err
	}
	if err = chargenSize(p.Plate, 160, 238, chargenPlatePath); err != nil {
		return nil, err
	}
	plateSeamPath := graphicsPrefix + "interface/chrgen/rollstatsr.bmp"
	plateSeam, err := readChargenBMP(src, plateSeamPath)
	if err != nil {
		return nil, err
	}
	if err = chargenSize(plateSeam, 16, 238, plateSeamPath); err != nil {
		return nil, err
	}
	p.PlateSeam = keyBlack(plateSeam)
	navArtPath := graphicsPrefix + "interface/chrgen/buttonsarea.bmp"
	if p.NavArt, err = readChargenBMP(src, navArtPath); err != nil {
		return nil, err
	}
	if err = chargenSize(p.NavArt, 160, 238, navArtPath); err != nil {
		return nil, err
	}
	// NavSeam closes NavArt's own 16-column gap against TownWideUpperRegion,
	// the same strip and the same rendering evidence as the school and
	// tavern rooms (DIV-166, DIV-168; pkg/game/townschoolart.go).
	navSeamPath := graphicsPrefix + "interface/inn/ruover.bmp"
	navSeam, err := readChargenBMP(src, navSeamPath)
	if err != nil {
		return nil, err
	}
	if err = chargenSize(navSeam, 16, 238, navSeamPath); err != nil {
		return nil, err
	}
	p.NavSeam = keyBlack(navSeam)
	dollBodyPath := graphicsPrefix + "interface/humanbackr.bmp"
	if p.DollPane.Body, err = readChargenBMP(src, dollBodyPath); err != nil {
		return nil, err
	}
	if err = chargenSize(p.DollPane.Body, 160, 242, dollBodyPath); err != nil {
		return nil, err
	}
	// CardBackground/CardSeam are the two halves of the card's own left-column
	// slot, and only one of them is decoded (D-8, round-2 adversarial
	// review: the prior comment here cited TOWN-312 for both, and TOWN-312
	// names no body field at all).
	//
	// chrgen/fullstatsl.bmp, the card's own BACKGROUND, is chosen on a size
	// match alone: its 160x242 dimensions equal TOWN-234's card-body rect
	// exactly, but no claim names the archive entry any load or paint call
	// site reads for that rect. This is `DIV-190`, the same open row the
	// plate body and both class columns carry.
	//
	// chrgen/fullstatsr.bmp, the card's own SEAM, IS decoded: TOWN-312 names
	// it as the `+0x6c` source and TOWN-234 gives that field's own blit as
	// `(160,238,16,242)`. Loading and drawing it closes the left column's
	// lower-half border pair (`DIV-189`) the way PlateSeam closes its upper
	// half.
	cardBackgroundPath := graphicsPrefix + "interface/chrgen/fullstatsl.bmp"
	if p.CardBackground, err = readChargenBMP(src, cardBackgroundPath); err != nil {
		return nil, err
	}
	if err = chargenSize(p.CardBackground, 160, 242, cardBackgroundPath); err != nil {
		return nil, err
	}
	cardSeamPath := graphicsPrefix + "interface/chrgen/fullstatsr.bmp"
	cardSeam, err := readChargenBMP(src, cardSeamPath)
	if err != nil {
		return nil, err
	}
	if err = chargenSize(cardSeam, 16, 242, cardSeamPath); err != nil {
		return nil, err
	}
	p.CardSeam = keyBlack(cardSeam)
	dollSeamPath := graphicsPrefix + "interface/humanbackl.bmp"
	dollSeam, err := readChargenBMP(src, dollSeamPath)
	if err != nil {
		return nil, err
	}
	if err = chargenSize(dollSeam, 16, 242, dollSeamPath); err != nil {
		return nil, err
	}
	p.DollPane.Seam = keyBlack(dollSeam)
	for i, hero := range []string{"mf", "mm", "ff", "fm"} {
		for j, state := range []string{"on", "l", "lon"} {
			addr := chargenPrecreatePath + "heroes/" + hero + state + ".bmp"
			if p.Choices[i][j], err = readChargenBMP(src, addr); err != nil {
				return nil, err
			}
			if err = chargenPatchFits(p.Choices[i][j], chargenPreChoiceOrigins[i], image.Rect(0, 0, 640, 480), addr); err != nil {
				return nil, err
			}
		}
	}
	for class, dir := range []string{"fighter", "mag"} {
		column := graphicsPrefix + "interface/chrgen/" + dir + "/column.bmp"
		if p.Columns[class], err = readChargenBMP(src, column); err != nil {
			return nil, err
		}
		if err = chargenSize(p.Columns[class], 320, 480, column); err != nil {
			return nil, err
		}
		mask := graphicsPrefix + "interface/chrgen/" + dir + "/mask.bmp"
		if p.ColumnMask[class], err = readChargenMask(src, mask); err != nil {
			return nil, err
		}
		if err = chargenSize(p.ColumnMask[class], 320, 480, mask); err != nil {
			return nil, err
		}
		if err = chargenMaskCodes(p.ColumnMask[class], chargenDetailedMaskCodes[class][:], mask); err != nil {
			return nil, err
		}
	}
	// The detailed stage's ten families are required at startup as well, even
	// though T1 only opens its shell. Reading them here makes a missing source
	// node a construction failure instead of a later, half-drawn screen.
	//
	// p.Skills[class][skill] holds three of a skill cell's four pictures, in
	// this load order: index 0 is `on.bmp` (selected, not hovered — pressed
	// dark), index 1 is `shine_off.bmp` (not selected, hovered — raised
	// light), index 2 is `shine_on.bmp` (selected, hovered — pressed light).
	// This is a load order, not "rest/hover/selected": the fourth corner
	// (not selected, not hovered — raised dark) has no picture here at all
	// because it is the column canvas's own baked art, read separately
	// above as p.Columns[class]. The read site (pkg/ui/chargen_page.go,
	// detailedSkillArt) is where these three indices are named by role.
	for class, skills := range [][]string{{"sword", "axe", "mace", "pike", "bow"}, {"fire", "water", "air", "earth", "astral"}} {
		classDir := []string{"fighter", "mag"}[class]
		for skill, name := range skills {
			for _, state := range []string{"on", "shine_off", "shine_on"} {
				addr := graphicsPrefix + "interface/chrgen/" + classDir + "/" + name + "/" + state + ".bmp"
				stateIndex := map[string]int{"on": 0, "shine_off": 1, "shine_on": 2}[state] // load order only, see above
				if p.Skills[class][skill][stateIndex], err = readChargenBMP(src, addr); err != nil {
					return nil, err
				}
				if err = chargenPatchFits(p.Skills[class][skill][stateIndex], chargenDetailedSkillOrigins[class][skill], image.Rect(0, 0, 320, 480), addr); err != nil {
					return nil, err
				}
			}
		}
	}
	for direction, names := range [2][5]string{
		{"mnloff", "mloff", "mlon", "mnlon", "mdisable"},
		{"pnloff", "ploff", "plon", "pnlon", "pdisable"},
	} {
		for state, name := range names {
			addr := graphicsPrefix + "interface/chrgen/buttons/" + name + ".bmp"
			if p.StatButtons[direction][state], err = readChargenBMP(src, addr); err != nil {
				return nil, err
			}
			if err = chargenSize(p.StatButtons[direction][state], 20, 20, addr); err != nil {
				return nil, err
			}
		}
	}
	font, err := LoadFont(src, "font2")
	if err != nil {
		return nil, err
	}
	p.Font = font
	if p.NameFont, err = LoadFontA(src, DocumentFont); err != nil {
		return nil, err
	}
	b, err := src.ReadFile(chargenTextPath)
	if err != nil {
		return nil, err
	}
	rows := SplitTextTable(b)
	get := func(slot int) (string, error) { return chargenTextAt(rows, chargenTextPath, slot) }
	a := &ChargenAssets{Presentation: p, Selector: LanguageSelector(src)}
	if a.HeroNames, err = heroPictureNames(src); err != nil {
		return nil, err
	}
	if a.Prompt, err = get(chargenPromptSlot); err != nil {
		return nil, err
	}
	if a.EmptyName, err = get(chargenEmptyNameSlot); err != nil {
		return nil, err
	}
	if a.ReservedName, err = get(chargenReservedSlot); err != nil {
		return nil, err
	}
	for class := range a.SkillHover {
		for skill := range a.SkillHover[class] {
			if a.SkillHover[class][skill], err = get(chargenSkillSlot + class*5 + skill); err != nil {
				return nil, err
			}
		}
	}
	if a.Back, err = get(chargenBackSlot); err != nil {
		return nil, err
	}
	if a.Play, err = get(chargenPlaySlot); err != nil {
		return nil, err
	}
	if a.Reset, err = get(chargenResetSlot); err != nil {
		return nil, err
	}
	p.Font.Selector = a.Selector
	p.NameFont.Selector = a.Selector
	return a, nil
}
