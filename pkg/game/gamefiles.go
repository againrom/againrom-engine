package game

import (
	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/databin"
	"againrom/pkg/mapload"
	"againrom/pkg/render/menu"
	"againrom/pkg/screenart"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// gameFiles are the readers of the installed files the two games lay out
// differently, each a value the one body calls. The last three are handed to
// the definition table for the placement decoder.
type gameFiles struct {
	table         databin.Layout
	openMap       func([]byte) (*alm.Map, error)
	compileScript func(*alm.Map, mapload.ScriptRefs) (*sim.Script, mapload.ScriptReport, error)
	loadMenu      func(*screenart.Set) (*menu.Assets, error)
	briefing      func(src entrySource, e base.Edition, mission int) string
	eventText     func(src entrySource, e base.Edition, mission, event int) ([]byte, bool)
	unitKeys      mapload.UnitKeys
	spellArms     func([]sim.SpellRule)
	freshPlayers  mapload.PlayerPolicy
	// itemNameKeys is the key half of the item-name table; the lines are
	// main.res text/itemname.txt in both games (R2-ENGINE-267).
	itemNameKeys string
	// autoCastCaptions are the dialogs text lines of the automatic-casting
	// row: its label and the three levels, or none.
	// terrainCosts is the address of the installed Terrain cost table every
	// World of the game prices cells with; empty keeps the compiled table.
	terrainCosts     string
	autoCastCaptions []int
	// savePages are the game's own load and save pages, or nil for the
	// first game's LOAD window and save dialog. ownSaves makes both pages
	// list only this game's game*.sav files by their label.
	savePages func(*FrontEnd) *ui.SavePages
	ownSaves  bool
}

var firstGameFiles = gameFiles{
	table:         databin.ROM1Layout,
	openMap:       alm.Open,
	compileScript: mapload.CompileScript,
	loadMenu:      menu.Load,
	briefing: func(src entrySource, _ base.Edition, mission int) string {
		return MissionObjectives(src, mission)
	},
	eventText: func(src entrySource, _ base.Edition, mission, event int) ([]byte, bool) {
		return ReadEventText(src, mission, event)
	},
	unitKeys:     mapload.ClassUnitKeys,
	freshPlayers: mapload.SlotPlayers,
	itemNameKeys: ItemNameKeyAddress,
	terrainCosts: worldPrefix + "data/map.reg",
}

var secondGameFiles = gameFiles{
	table:         databin.ROM2Layout,
	openMap:       alm.OpenROM2,
	compileScript: mapload.CompileROM2Script,
	loadMenu:      menu.LoadSecond,
	briefing:      secondGameBriefing,
	eventText: func(src entrySource, e base.Edition, mission, event int) ([]byte, bool) {
		return readSecondGameEventText(src, InstallTextCode(src, e), mission, event)
	},
	unitKeys:     mapload.ServerUnitKeys,
	spellArms:    mapload.SecondGameSpellArms,
	freshPlayers: mapload.NoFreshPlayers,
	itemNameKeys: worldPrefix + "data/itemname.bin",
	// R2-ASSET-085.
	autoCastCaptions: []int{170, 171, 172, 173},
	savePages:        secondSavePages,
	ownSaves:         true,
}

// filesOf are the file readers of game g.
func filesOf(g base.Game) *gameFiles { return campaignOf(g).files() }

// tableEdition is the edition a definition table was read under; a nil table,
// or one read without an edition, is the first game's.
func tableEdition(t *mapload.Table) base.Edition {
	if t == nil || t.Edition == nil {
		return base.Game("").Edition()
	}
	return *t.Edition
}

// tableCampaign and tableFiles are the campaign service and file readers of
// that edition.
func tableCampaign(t *mapload.Table) campaignService { return campaignOf(tableEdition(t).Game) }

func tableFiles(t *mapload.Table) *gameFiles { return tableCampaign(t).files() }
