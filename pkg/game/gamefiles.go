package game

import (
	"againrom/pkg/base"
	"againrom/pkg/data"
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
	// storedFormMerge keeps a plain item and a constructed one of the same
	// code apart in a pack merge: no second-game claim states its equality.
	storedFormMerge bool
	// classCreatureBooks leaves Dragon and Daemon without the first game's
	// fixed books: no second-game claim states them.
	classCreatureBooks bool
	// storedOwnWeight keeps a native Human's stored own weight in a SAVE
	// instead of its worn weight: no second-game claim states the rule.
	storedOwnWeight bool
	// storedWeaponSpell writes a worn Weapon's Spell as the World holds it,
	// without building a missing one: no second-game claim states the rule.
	storedWeaponSpell bool
	// releasedSpellReadout describes a spell weapon and states a caster's card
	// damage from the spell its attack releases, the readout this build kept
	// before ITEM-211 and HERO-172: no second-game claim states either.
	releasedSpellReadout bool
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
	savePages       func(*FrontEnd) *ui.SavePages
	ownSaves        bool
	boundedSaveView bool
	// orderThresholds keeps each actor's Withdraw and Wimpy in its SAV Order
	// block, written on SAVE and read on LOAD (AI-WITHDRAW-027); no
	// second-game claim places them there.
	orderThresholds bool
	// missionBookClosed opens a new mission with the spellbook switched off,
	// as the first game's own mission-start saves store it (DIV-4020); the
	// second game follows by owner direction while no second-game claim
	// states its mission-start panels.
	missionBookClosed bool
	// dialogueDefault is the text a mission dialogue shows when its message
	// has no page 1 (DIALOGUE-049); empty opens no dialogue for it. The
	// second game follows the first by owner direction while no second-game
	// claim states its panel's default (DIV-4053).
	dialogueDefault string
	// humanSpeakers adds data.HumanSpeakerFaces beside the registry's
	// records, which win; heroSpeaker is the player's key.
	humanSpeakers bool
	heroSpeaker   int32
}

var firstGameFiles = gameFiles{
	boundedSaveView:   true,
	orderThresholds:   true,
	missionBookClosed: true,
	table:             databin.ROM1Layout,
	openMap:           alm.Open,
	compileScript:     mapload.CompileScript,
	loadMenu:          menu.Load,
	briefing: func(src entrySource, _ base.Edition, mission int) string {
		return MissionObjectives(src, mission)
	},
	eventText: func(src entrySource, _ base.Edition, mission, event int) ([]byte, bool) {
		return ReadEventText(src, mission, event)
	},
	unitKeys:        mapload.ClassUnitKeys,
	freshPlayers:    mapload.SlotPlayers,
	itemNameKeys:    ItemNameKeyAddress,
	terrainCosts:    worldPrefix + "data/map.reg",
	dialogueDefault: dialogueDefaultText,
}

var secondGameFiles = gameFiles{
	missionBookClosed: true,
	dialogueDefault:   dialogueDefaultText,
	table:             databin.ROM2Layout,
	openMap:           alm.OpenROM2,
	compileScript:     mapload.CompileROM2Script,
	loadMenu:          menu.LoadSecond,
	briefing:          secondGameBriefing,
	eventText: func(src entrySource, e base.Edition, mission, event int) ([]byte, bool) {
		return readSecondGameEventText(src, InstallTextCode(src, e), mission, event)
	},
	unitKeys:             mapload.ServerUnitKeys,
	spellArms:            mapload.SecondGameSpellArms,
	freshPlayers:         mapload.NoFreshPlayers,
	storedFormMerge:      true,
	classCreatureBooks:   true,
	storedOwnWeight:      true,
	storedWeaponSpell:    true,
	releasedSpellReadout: true,
	itemNameKeys:         worldPrefix + "data/itemname.bin",
	// R2-ASSET-085.
	autoCastCaptions: []int{170, 171, 172, 173},
	savePages:        secondSavePages,
	ownSaves:         true,
	// The starting rows' server ids give key 21 (R2-ENGINE-072).
	humanSpeakers: true,
	heroSpeaker:   21,
}

// speakerFaces is the speaker table of t's campaign.
func speakerFaces(t *mapload.Table) map[int32]data.NPCFace {
	if t == nil {
		return nil
	}
	faces := t.NPCCatalog.Portraits()
	files := tableFiles(t)
	if !files.humanSpeakers || t.Humans == nil {
		return faces
	}
	out := data.HumanSpeakerFaces(t.Humans, files.heroSpeaker, func(row int) data.Equipment {
		return equipmentFromSlots(mapload.RowOutfit(row, t))
	})
	for k, v := range faces {
		out[k] = v
	}
	return out
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
