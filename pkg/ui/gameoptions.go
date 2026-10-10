package ui

import (
	"fmt"
	"image"
)

// GameOption names a working setting shared by the menu and map shortcuts.
type GameOption uint8

const (
	GameOptionDayNight GameOption = iota
	GameOptionHealth
	GameOptionDamage
	GameOptionFormation
	GameOptionRetreat
	GameOptionPathfinding
	GameOptionSmoothing
	GameOptionShadows
	GameOptionLighting
	GameOptionAnimation
	GameOptionAutoHealing
	gameOptionCount
)

type GameOptionValues [gameOptionCount]int

func (o GameOption) Choices() int {
	if o == GameOptionFormation || o == GameOptionRetreat || o == GameOptionAutoHealing {
		return 3
	}
	if o < gameOptionCount {
		return 2
	}
	return 0
}

type GameOptionWords struct {
	Title, Tips, OK, Cancel, Speed string
	Labels                         [gameOptionCount]string
	Formation                      [3]string
	Retreat                        [3]string
	AutoHealing                    [3]string
}

// Captions are already encoded for the installed font. Write persists before
// applying a change; Read includes commands waiting for the next ordinary tick.
type GameOptionControls struct {
	Autosave TimedAutosaveControls
	Read     func(onMap bool) GameOptionValues
	Write    func(onMap bool, option GameOption, value int) error
	Words    GameOptionWords
	Checks   [2]image.Image
	Radios   [2]image.Image
	// draft is the open page's local copy; nil while the page is closed.
	draft *gameOptionsDraft
}

func (a *App) SetGameOptionControls(c GameOptionControls) {
	if a != nil && a.flow != nil {
		f := a.flow
		c.Words.Labels[GameOptionPathfinding] = f.menuWord("options.pathfinding")
		authored := [...]string{f.menuWord("options.shadows"),
			f.menuWord("options.lighting"), f.menuWord("options.animation")}
		// The three graphics captions come from the install when it supplies
		// them; the authored words stand in only for an absent one.
		for i, label := range authored {
			if o := GameOptionShadows + GameOption(i); c.Words.Labels[o] == "" {
				c.Words.Labels[o] = label
			}
		}
		a.flow.gameOptions = c
	}
}

func optionAction(o GameOption) gameMenuAction { return gameMenuDayNight + gameMenuAction(o) }

func (f *flow) gameOptionsOnMap() bool {
	return f.screen == ScreenMap || f.screen == ScreenGameMenu && f.menuBack == ScreenMap
}

func (f *flow) readGameOptions() GameOptionValues { return f.gameOptions.Read(f.gameOptionsOnMap()) }

func (f *flow) setGameOption(o GameOption, value int) {
	if f.gameOptions.Write == nil || value < 0 || value >= o.Choices() {
		return
	}
	err := f.gameOptions.Write(f.gameOptionsOnMap(), o, value)
	if f.screen == ScreenGameMenu {
		selection := f.menuList.Selection()
		f.rebuildGameMenu(gameMenuGameOptionsPage, selection)
	}
	if err != nil {
		f.msg = fmt.Sprintf("Setting not saved: %v", err)
		if f.screen == ScreenMap {
			f.showTextNotice(f.msg)
		}
	}
}

// False retains the standalone viewer's existing shortcut behavior when no
// preference seam is installed.
func (f *flow) cycleGameOption(o GameOption) bool {
	if f.gameOptions.Read == nil || f.gameOptions.Write == nil {
		return false
	}
	f.setGameOption(o, (f.readGameOptions()[o]+1)%o.Choices())
	return true
}
