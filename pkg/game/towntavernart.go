package game

import (
	"fmt"

	"againrom/pkg/render/terrain"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// LoadTownTavernArt resolves the tavern page's art once from the tavern
// room of description d: its own panels, plaques and roster sheets, and its
// scene. Like the school surface it is cosmetic and is carried with an error
// rather than making startup fatal.
//
// The page art's required entries fail the whole load: the panels, the
// three plaque pairs, the four-command body and plaques the owner-directed
// composition draws (DIV-483), and the fifteen mercenary sheets. The talk-cell
// and hero sheets are optional: one that does not read leaves its slot empty
// and a talk cell then draws only its ground (DIV-1408). The answer's error is
// the scene's: a required scene entry that fails leaves Scene nil, and an
// optional one is named.
func LoadTownTavernArt(d *town.Description, src terrain.EntrySource) (*ui.TownTavernArt, error) {
	set, err := roomPageArt(d, "tavern", src)
	if err != nil {
		return nil, err
	}
	pic := set.Picture
	a := &ui.TownTavernArt{
		LeftPicture:     pic("left-picture"),
		LeftStats:       pic("left-stats"),
		LeftStatsSeam:   pic("left-stats-seam"),
		LeftPictureSeam: pic("left-picture-seam"),
		Center:          pic("center"),
		ManBack:         pic("man-back"),
		ManBackTalk:     pic("man-back-talk"),
		Upper:           pic("upper"),
		UpperSeam:       pic("upper-seam"),
		CommandUpper:    pic("command-upper"),
	}
	for i := range a.Buttons {
		for state := range a.Buttons[i] {
			a.Buttons[i][state] = pic(fmt.Sprintf("button/%d/%d", i, state))
		}
	}
	for i := range a.CommandButtons {
		a.CommandButtons[i] = pic(fmt.Sprintf("command-button/%d", i))
	}
	for _, name := range []string{"unit", "talk-unit"} {
		for typ, frames := range set.Numbered(name) {
			if typ > 0 && typ < len(a.UnitFrames) {
				a.UnitFrames[typ], a.Units[typ] = frames, frames[0]
			}
		}
	}
	for i, name := range [2]string{"hero-fighter", "hero-mage"} {
		a.HeroFrames[i] = set.Pictures(name)
	}
	scene, problems := loadRoomSceneArt(d, "tavern", src)
	a.Scene = scene
	return a, problems
}
