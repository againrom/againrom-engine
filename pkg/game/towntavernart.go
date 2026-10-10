package game

import (
	"fmt"
	"image"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

const townTavernArtPrefix = graphicsPrefix + "interface/inn/"

// LoadTownTavernArt resolves the three fixed tavern children and the
// type-indexed roster pictures once. Like the school surface it is cosmetic and
// is carried with an error rather than making startup fatal.
func LoadTownTavernArt(src terrain.EntrySource) (*ui.TownTavernArt, error) {
	a := &ui.TownTavernArt{}
	var err error
	if a.LeftPicture, err = readChargenBMP(src, townTavernArtPrefix+"leftpicture.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(a.LeftPicture, 160, 242, townTavernArtPrefix+"leftpicture.bmp"); err != nil {
		return nil, err
	}
	if a.LeftStats, err = readChargenBMP(src, townTavernArtPrefix+"leftstats.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(a.LeftStats, 160, 238, townTavernArtPrefix+"leftstats.bmp"); err != nil {
		return nil, err
	}
	if a.LeftStatsSeam, err = readChargenBMP(src, townTavernArtPrefix+"luover.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(a.LeftStatsSeam, 16, 238, townTavernArtPrefix+"luover.bmp"); err != nil {
		return nil, err
	}
	a.LeftStatsSeam = keyBlack(a.LeftStatsSeam.(*image.RGBA))
	if a.LeftPictureSeam, err = readChargenBMP(src, townTavernArtPrefix+"ldover.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(a.LeftPictureSeam, 16, 242, townTavernArtPrefix+"ldover.bmp"); err != nil {
		return nil, err
	}
	a.LeftPictureSeam = keyBlack(a.LeftPictureSeam.(*image.RGBA))
	if a.Center, err = readChargenBMP(src, townTavernArtPrefix+"centerarea.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(a.Center, 320, 480, townTavernArtPrefix+"centerarea.bmp"); err != nil {
		return nil, err
	}
	if a.ManBack, err = readChargenBMP(src, townTavernArtPrefix+"manback.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(a.ManBack, 48, 64, townTavernArtPrefix+"manback.bmp"); err != nil {
		return nil, err
	}
	if a.ManBackTalk, err = readChargenBMP(src, townTavernArtPrefix+"manbacktalk.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(a.ManBackTalk, 48, 64, townTavernArtPrefix+"manbacktalk.bmp"); err != nil {
		return nil, err
	}
	if a.Upper, err = readChargenBMP(src, townTavernArtPrefix+"buttonsarea.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(a.Upper, 160, 238, townTavernArtPrefix+"buttonsarea.bmp"); err != nil {
		return nil, err
	}
	// UpperSeam closes TownWideUpperRegion's own 16-column gap beside Upper
	// (DIV-166, DIV-168); see LoadTownSchoolArt for the rendering evidence that
	// selects this strip among the three 238-row candidates.
	if a.UpperSeam, err = readChargenBMP(src, townTavernArtPrefix+"ruover.bmp"); err != nil {
		return nil, err
	}
	if err = chargenSize(a.UpperSeam, 16, 238, townTavernArtPrefix+"ruover.bmp"); err != nil {
		return nil, err
	}
	a.UpperSeam = keyBlack(a.UpperSeam.(*image.RGBA))
	// The three shipped button plaques, each an off/on pair (1017; DIV-159),
	// read off the tavern's original well order top to bottom. The owner-directed
	// four-command composition reuses the first plaque for Sleep and Hire/Fire.
	for i, name := range []string{"button1", "button2", "button3"} {
		for state, suffix := range []string{"off", "on"} {
			addr := townTavernArtPrefix + name + suffix + ".bmp"
			if a.Buttons[i][state], err = readChargenBMP(src, addr); err != nil {
				return nil, err
			}
			if err = chargenSize(a.Buttons[i][state], 140, 46, addr); err != nil {
				return nil, err
			}
		}
	}
	// Sleep turns the production tavern panel into the shop's complete
	// four-button composition (DIV-483), not four large tavern plaques forced
	// into a three-button background. Load the matching 176x238 body and all
	// four native plaques atomically; the 120x52 edge plaques are what leave
	// the body's top and bottom ornaments uncovered.
	commandUpperAddr := shopArtPrefix + "shopmenu.bmp"
	commandUpper, err := readChargenBMP(src, commandUpperAddr)
	if err != nil {
		return nil, err
	}
	if err = chargenSize(commandUpper, 176, 238, commandUpperAddr); err != nil {
		return nil, err
	}
	a.CommandUpper = keyBlack(commandUpper)
	for i, name := range shopButtonFiles {
		addr := shopArtPrefix + name
		pic, readErr := readChargenBMP(src, addr)
		if readErr != nil {
			return nil, readErr
		}
		r := ui.ShopButtonRect(i)
		if err = chargenSize(pic, r.Dx(), r.Dy(), addr); err != nil {
			return nil, err
		}
		a.CommandButtons[i] = keyBlack(pic)
	}
	for typ := 1; typ <= 15; typ++ {
		frames, err := tavernSheetFrames(src, fmt.Sprintf("unit%d", typ))
		if err != nil {
			return nil, err
		}
		a.UnitFrames[typ], a.Units[typ] = frames, frames[0]
	}
	// The talk-cell sheets TOWN-470 lists. A sheet that does not read leaves
	// its slot empty: a talk cell then draws only its ground (DIV-1408).
	for _, typ := range tavernTalkSheets {
		if frames, err := tavernSheetFrames(src, fmt.Sprintf("unit%d", typ)); err == nil {
			a.UnitFrames[typ], a.Units[typ] = frames, frames[0]
		}
	}
	for i, name := range [2]string{"herofighter", "heromage"} {
		if frames, err := tavernSheetFrames(src, name); err == nil {
			a.HeroFrames[i] = frames
		}
	}
	scene, problems := loadRoomSceneArt("tavern", src)
	a.Scene = scene
	return a, problems
}

// tavernTalkSheets are the Unit<n> inn sheets above the fifteen mercenary
// types that TOWN-470 finds shipped on both roots.
var tavernTalkSheets = []int{29, 30, 32, 41, 42, 43, 44, 52, 61, 62, 64}

func tavernSheetFrames(src terrain.EntrySource, dir string) ([]image.Image, error) {
	addr := fmt.Sprintf("%s%s/sprites.16a", townTavernArtPrefix, dir)
	frames := effectFrames(src, addr)
	if len(frames) == 0 {
		return nil, fmt.Errorf("%s: no readable frames", addr)
	}
	out := make([]image.Image, len(frames))
	for i, frame := range frames {
		out[i] = frame.RGBA()
	}
	return out, nil
}
