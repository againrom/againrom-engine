package game

import "againrom/pkg/base"

// The install addresses the fixtures of this package's art tests write.
// Production reads them only through the screen art descriptions.
const (
	townTavernArtPrefix = graphicsPrefix + "interface/inn/"
	townSchoolArtPrefix = graphicsPrefix + "interface/training/"
	shopArtPrefix       = graphicsPrefix + "interface/"

	commandPanelHeadsPath      = shopArtPrefix + "headsr.bmp"
	commandPanelHeadsSeamPath  = shopArtPrefix + "headsl.bmp"
	commandPanelActivePath     = shopArtPrefix + "commandbarr.bmp"
	commandPanelActiveSeamPath = shopArtPrefix + "commandbarl.bmp"
	commandPanelDisabledPath   = shopArtPrefix + "commandempr.bmp"
	commandPanelSelectedPath   = shopArtPrefix + "commanddnr.bmp"
)

var shopButtonFiles = [4]string{"shopbutton1.bmp", "shopbutton2.bmp", "shopbutton3.bmp", "shopbutton4.bmp"}

// rom1Screens is the first game's screen art description.
var rom1Screens = ScreensDescription(base.Profile{})

// tavernTalkSheets are the talk-cell sheet numbers the tavern room's page art
// names.
func tavernTalkSheets() []int {
	for _, r := range ROM1TownDescription().Rooms {
		for _, e := range r.Art {
			if r.Name == "tavern" && e.Name == "talk-unit" {
				return e.Index[0].Values
			}
		}
	}
	return nil
}
