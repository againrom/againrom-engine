package game

import (
	"image"
	"image/color"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/databin"
	"againrom/pkg/mapload"
	"againrom/pkg/render/menu"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// gameFiles are the readers of the installed files the two games lay out
// differently, each a value the one body calls. The last three are handed to
// the definition table for the placement decoder.
type gameFiles struct {
	hover ui.HoverProfile
	// dollSounds name the sfx.res members the mission inventory plays when an
	// item goes on the figure and when one comes off; empty names stay silent.
	dollSounds     dollSoundProfile
	cursorHotspots [28]image.Point
	table          databin.Layout
	openMap        func([]byte) (*alm.Map, error)
	compileScript  func(*alm.Map, mapload.ScriptRefs) (*sim.Script, mapload.ScriptReport, error)
	menuLayout     menu.Layout
	briefing       func(src entrySource, e base.Edition, mission int) string
	eventText      func(src entrySource, e base.Edition, mission, event int) ([]byte, bool)
	unitKeys       mapload.UnitKeys
	carryStockOnly bool
	spellArms      func([]sim.SpellRule)
	freshPlayers   mapload.PlayerPolicy
	// storedFormMerge keeps a plain item and a constructed one of the same
	// code apart in a pack merge: no second-game claim states its equality.
	storedFormMerge bool
	// creatureBooks is the creature book source; nil is the first game's.
	creatureBooks *mapload.CreatureBooks
	// deathItems builds a table's death item draw; nil, the first game's,
	// draws no death item.
	deathItems func(*mapload.Table) sim.DeathItemDraw
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
	itemPacket   string
	// itemNamer names an item with neither a saved nor a shipped name.
	itemNamer mapload.ItemNamer
	// autoCastCaptions are the dialogs text lines of the automatic-casting
	// row: its label and the three levels, or none.
	// terrainCosts is the address of the installed Terrain cost table every
	// World of the game prices cells with; empty keeps the compiled table.
	terrainCosts string
	// planeCosts is the address of the same table the cell planes and the
	// map checker read; both games read it.
	planeCosts       string
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
	// heroSavedType writes a party hero's SAV T0E from the hero, the
	// player-character constructor's type (HERO-CLASS-013, HERO-191); no
	// second-game claim states the rule.
	heroSavedType bool
	// missionBookClosed opens a new mission with the spellbook switched off,
	// as the first game's own mission-start saves store it (DIV-4020); the
	// second game follows by owner direction while no second-game claim
	// states its mission-start panels.
	missionBookClosed bool
	// missionPackClosed opens a new mission with the inventory row switched
	// off: the first game's MISSION-ROUTE-093, the second game's R2-ENGINE-588
	// (DIV-4020).
	missionPackClosed bool
	// missionHeroSelected selects the party's hero when a new mission opens,
	// so the portrait, the command grid and the sheet show him under the
	// opening dialogue, as the second game's original draws them
	// (R2-ASSET-148).
	missionHeroSelected bool
	// purseMoneyWord makes the inventory purse's hover the install's one-line
	// money word.
	purseMoneyWord bool
	// missionScreen is the game's mission screen layout data.
	missionScreen ui.MissionScreen
	// gameMenu is the game's game-menu data.
	gameMenu ui.GameMenuProfile
	// dialogueDefault is the text a mission dialogue shows when its message
	// has no page 1 (DIALOGUE-049); empty opens no dialogue for it. The
	// second game follows the first by owner direction while no second-game
	// claim states its panel's default (DIV-4053).
	dialogueDefault string
	// humanSpeakers adds data.HumanSpeakerFaces beside the registry's
	// records, which win; heroSpeaker is the player's key.
	humanSpeakers bool
	heroSpeaker   int32
	// schoolPoolExperience grows the pools by the five school skills alone.
	schoolPoolExperience bool
	// fallbackSpeakers adds data.FallbackSpeakerFace for every key that has
	// no other record.
	fallbackSpeakers bool
	unitSpeakers     bool
	portraitWindows  portraitWindowProfile
	// manaSpellcaster holds the actor value the SPELLCASTER caption tests
	// nonzero only for an actor with a mana pool (DIV-4086).
	manaSpellcaster bool
	// bookSpellcaster makes a person's SPELLCASTER actor value his spellbook
	// mask: zero without a positive mana maximum, else his known spell ids.
	bookSpellcaster bool
	// fogStampOnMove folds a moved party body's sight into the fog between
	// the periodic refreshes; the second game's map keeps the refresh alone.
	fogStampOnMove bool
	// flight is the record producer's profile.
	flight flightProfile
	// targetMarkOnRelease attaches Heal and Drain showers to their actor.
	targetMarkOnRelease bool
	// teleportCast is how a Teleport cast's caster plays and sounds.
	teleportCast teleportCastProfile
	// commonBurstSound silences the cell sound of the picture 51 burst, which
	// sounds from its common driver phase.
	commonBurstSound bool
	// projectileCounter tracks and seeds the World's saved projectile free
	// index across missions.
	projectileCounter bool
	// mageCardWeaponSpell states a unit card's weapon spell from the mage's
	// worn weapon instead of the spell its attack releases.
	mageCardWeaponSpell bool
	// loadedTavernStock orders the tavern's mercenary cells after a town LOAD
	// by the stock ids the loaded allocator gives (TAVERN-ORDER-015).
	loadedTavernStock bool
	// heldMarks is the held mark of a lasting effect by the arm its spell
	// runs, where the game's geometry or picture is not the arm's own kind.
	heldMarks map[uint16]heldMark
	// unitVoices are the game's command and selection voices.
	unitVoices unitVoiceRules
	// objectives is the game's row of the objectives panel and outcome text
	// (questobjectives.go).
	objectives objectivesProfile
	// generator is the game's hero generator row (chargen.go).
	generator generatorProfile
	// hire is the game's hire rows, moment and cost (hire.go).
	hire *hireProfile
}

func (f *gameFiles) spellTakesBody(id uint32) bool {
	if id > 0xffff {
		return false
	}
	rows := []sim.SpellRule{{ID: uint16(id)}}
	if f.spellArms != nil {
		f.spellArms(rows)
	}
	return sim.SpellArmTakesBody(rows[0].ArmID())
}

// spellcasterValue is the actor value the SPELLCASTER caption tests, from
// the value the projection gives, the actor's mana maximum and known-spell
// mask. person is false only for a creature on the map, whose value the
// card's creature rule reads (DIV-1809). The mask keeps spell ids 1..28, the
// slots a constructed book populates (TAVERN-035).
func (g *gameFiles) spellcasterValue(value, maxMana int32, spells uint32, person bool) int32 {
	if g.manaSpellcaster && maxMana <= 0 {
		return 0
	}
	if g.bookSpellcaster && person {
		if maxMana <= 0 {
			return 0
		}
		return int32(spells & 0x1ffffffe)
	}
	return value
}

// terrainCostsAddress is the installed Terrain cost table's address.
const terrainCostsAddress = worldPrefix + "data/map.reg"

// dollSoundProfile is the pair of members a figure's put-on and take-off play.
type dollSoundProfile struct{ on, off string }

var firstGameFiles = gameFiles{
	hire:              &tavernHires,
	hover:             ui.KindRectangleHover,
	cursorHotspots:    firstCursorHotspots,
	portraitWindows:   portraitWindowProfile{fallback: ui.AuthoredDialogueLayout().PortraitWindow},
	missionPackClosed: true,
	purseMoneyWord:    true,
	missionScreen:     ui.MissionScreen{ColumnW: ui.MissionPanelW, NoticeOnScreen: true},
	boundedSaveView:   true,
	orderThresholds:   true,
	heroSavedType:     true,
	missionBookClosed: true,
	table:             databin.ROM1Layout,
	openMap:           alm.Open,
	compileScript:     mapload.CompileScript,
	menuLayout:        menu.FirstLayout,
	briefing: func(src entrySource, _ base.Edition, mission int) string {
		return MissionObjectives(src, mission)
	},
	eventText: func(src entrySource, _ base.Edition, mission, event int) ([]byte, bool) {
		return ReadEventText(src, mission, event)
	},
	unitKeys:        mapload.ClassUnitKeys,
	freshPlayers:    mapload.SlotPlayers,
	creatureBooks:   mapload.FirstGameCreatureBooks,
	itemNameKeys:    ItemNameKeyAddress,
	itemNamer:       mapload.CodeItemNamer,
	terrainCosts:    terrainCostsAddress,
	planeCosts:      terrainCostsAddress,
	dialogueDefault: dialogueDefaultText,

	fogStampOnMove:      true,
	flight:              flightProfile{registryRows: true},
	targetMarkOnRelease: true,
	teleportCast:        teleportCastProfile{run: true, messageSound: true},
	commonBurstSound:    true,
	projectileCounter:   true,
	mageCardWeaponSpell: true,
	loadedTavernStock:   true,
	// TAVERN-035, TEXT-135.
	bookSpellcaster: true,
	unitVoices:      firstGameVoices,
	generator:       generatorProfile{documents: true, start: generatorStartMission},
	objectives:      objectivesProfile{},
}

// secondGameMenu's town popup stands at (100,100), 440x340 (R2-ENGINE-441).
var secondGameMenu = ui.GameMenuProfile{Rows: ui.SecondGameMenuRows, TownRootPanel: image.Rect(100, 100, 540, 440),
	NativeMissionWindows: true, MissionBackdrop: color.RGBA{A: 64}, WindowsCloseMenu: true, EndQuestFocus: 1, AbortGameFocus: 2}

var secondGameFiles = gameFiles{
	hire:           &bankHires,
	dollSounds:     dollSoundProfile{on: "put_on.wav", off: "put_off.wav"},
	hover:          ui.KindRectangleHover,
	cursorHotspots: secondCursorHotspots,
	portraitWindows: portraitWindowProfile{
		fallback: ui.AuthoredDialogueLayout().PortraitWindow,
		byType: map[int32]image.Point{
			64:  {X: 30, Y: 72},
			65:  {X: 26, Y: 23},
			66:  {X: 10, Y: 44},
			67:  {X: -1, Y: -1},
			68:  {X: 15, Y: 21},
			69:  {X: 32, Y: 21},
			70:  {X: -1, Y: -1},
			71:  {X: 7, Y: 49},
			72:  {X: -1, Y: -1},
			73:  {X: -1, Y: -1},
			74:  {X: 8, Y: 72},
			75:  {X: -1, Y: -1},
			76:  {X: -1, Y: -1},
			77:  {X: -1, Y: -1},
			78:  {X: -1, Y: -1},
			79:  {X: 30, Y: 72},
			80:  {X: 23, Y: 31},
			81:  {X: 43, Y: 7},
			82:  {X: 37, Y: 28},
			83:  {X: 37, Y: 28},
			84:  {X: 37, Y: 28},
			85:  {X: 37, Y: 28},
			86:  {X: 37, Y: 28},
			87:  {X: 37, Y: 28},
			88:  {X: 37, Y: 28},
			89:  {X: 37, Y: 24},
			90:  {X: 37, Y: 24},
			91:  {X: 37, Y: 24},
			92:  {X: 37, Y: 24},
			93:  {X: 37, Y: 15},
			94:  {X: 37, Y: 15},
			95:  {X: 37, Y: 15},
			96:  {X: 37, Y: 15},
			97:  {X: 35, Y: 15},
			98:  {X: 35, Y: 15},
			99:  {X: 35, Y: 14},
			100: {X: 35, Y: 21},
			101: {X: 32, Y: 18},
			102: {X: 15, Y: 21},
		},
	},
	planeCosts:          terrainCostsAddress,
	missionPackClosed:   true,
	missionHeroSelected: true,
	missionScreen:       ui.MissionScreen{ColumnW: ui.MissionPanelW, NoticeOnScreen: true, PickUsableInExploredGround: true, MinimapLocal: color.RGBA{R: 82, G: 121, B: 231, A: 0xff}},
	gameMenu:            secondGameMenu,
	missionBookClosed:   true,
	dialogueDefault:     dialogueDefaultText,
	table:               databin.ROM2Layout,
	openMap:             alm.OpenROM2,
	compileScript:       mapload.CompileROM2Script,
	menuLayout:          menu.SecondLayout,
	briefing:            secondGameBriefing,
	eventText: func(src entrySource, e base.Edition, mission, event int) ([]byte, bool) {
		text := installText(src, missionTextSource.at(mission), InstallTextCode(src, e))
		if !text.present() {
			return nil, false
		}
		body := text.numberedSection("event", event)
		return []byte(body), body != ""
	},
	unitKeys:             mapload.ServerUnitKeys,
	carryStockOnly:       true,
	spellArms:            mapload.SecondGameSpellArms,
	freshPlayers:         mapload.NoFreshPlayers,
	storedFormMerge:      true,
	creatureBooks:        mapload.ClassCellBooks,
	deathItems:           secondDeathItems,
	storedOwnWeight:      true,
	storedWeaponSpell:    true,
	releasedSpellReadout: true,
	itemNameKeys:         worldPrefix + "data/itemname.bin",
	itemPacket:           worldPrefix + "data/itemname.pkt",
	itemNamer:            mapload.RowItemNamer,
	// R2-ASSET-085.
	autoCastCaptions: []int{170, 171, 172, 173},
	savePages:        secondSavePages,
	ownSaves:         true,
	// The starting rows' server ids give key 21 (R2-ENGINE-072).
	humanSpeakers: true,
	heroSpeaker:   21,
	// R2-ENGINE-579.
	schoolPoolExperience: true,
	// R2-ASSET-150.
	purseMoneyWord: true,
	// R2-ENGINE-580.
	manaSpellcaster: true,
	// R2-ENGINE-583.
	fallbackSpeakers: true,
	unitSpeakers:     true,
	flight: flightProfile{FlightProfile: sim.FlightProfile{Release: sim.ReleaseAtClient, StartFromMovedCaster: true, ShotsNeedArt: true},
		drawnClass: true},
	targetMarkOnRelease: true,
	// R2-ENGINE-704: state 8 for the type's attack frame list, Sfx 554 at
	// AttackDelay from that run and none from the cast message.
	teleportCast: teleportCastProfile{run: true},
	heldMarks:    secondHeldMarks,
	objectives:   objectivesProfile{subobjectives: true, lossEntries: true, npcKeys: true},
	unitVoices:   secondGameVoices,
	generator:    generatorProfile{templates: true, classProfile: true, campaignOutfit: true, roomCard: true, start: generatorStartTown},
}

// speakerFaces is the speaker table of t's campaign: the catalog's portraits,
// then the campaign's Human, fallback and creature rows for keys it lacks.
func speakerFaces(t *mapload.Table) map[int32]data.NPCFace { return npcFaceList(t, true) }

// CatalogFaces is the catalog's portraits alone, the rows of speakerFaces that
// come from the catalog.
func CatalogFaces(t *mapload.Table) map[int32]data.NPCFace { return npcFaceList(t, false) }

type portraitWindowProfile struct {
	fallback image.Rectangle
	byType   map[int32]image.Point
}

// npcFaceList builds speaker records. R2-ENGINE-819's BMP coordinates and final
// pixels are Medium; native conversion and clip are Unknown (DIV-4467).
func npcFaceList(t *mapload.Table, whole bool) map[int32]data.NPCFace {
	if t == nil {
		return nil
	}
	faces := t.NPCCatalog.Portraits()
	files := tableFiles(t)
	if !whole || !files.humanSpeakers || t.Humans == nil && t.Units == nil {
		return faces
	}
	companions := map[int32]int32{}
	for _, p := range (townInstall{table: t}).joins() {
		if p.ServerID != 0 {
			companions[p.ServerID] = int32(p.Identity)
		}
	}
	out := map[int32]data.NPCFace{}
	if t.Humans != nil {
		out = data.HumanSpeakerFaces(t.Humans, files.heroSpeaker, companions, func(row int) data.Equipment {
			return equipmentFromSlots(mapload.RowOutfit(row, t))
		})
	}
	for k, v := range faces {
		out[k] = v
	}
	if files.fallbackSpeakers {
		for k := data.FallbackSpeakerFirst; k <= data.FallbackSpeakerLast; k++ {
			if _, held := out[k]; !held {
				out[k], _ = data.FallbackSpeakerFace(k)
			}
		}
	}
	if files.unitSpeakers && t.Units != nil {
		for row := 1; row < t.Units.Len(); row++ {
			params := t.Units.EntryParams(row)
			key, ok := data.UnitServerID(params)
			if !ok || key <= 0 || t.Units.EntryName(row) == "" {
				continue
			}
			if _, held := out[key]; held {
				continue
			}
			def, err := data.NewUnitDef(t.Units.EntryName(row), params)
			if err == nil {
				rec := data.NPCFace{Kind: data.NPCPortrait, Class: def.TypeID, HasClass: true, Face: int(def.Face),
					Tokens: data.NPCTokens(data.NPCTokenNotHuman) | data.NPCTokens(data.NPCTokenPicture) | data.NPCTokens(data.NPCTokenFace)}
				window := files.portraitWindows.fallback
				if origin, found := files.portraitWindows.byType[def.TypeID]; found && origin.Y != -1 {
					window = ui.NoticeFaceWindow(origin.X, origin.Y)
				}
				if window != ui.AuthoredDialogueLayout().PortraitWindow {
					rec.WindowX, rec.WindowY, rec.HasWindow = window.Min.X, window.Min.Y, true
				}
				out[key] = rec
			}
		}
	}
	return out
}

// filesOf are the file readers of game g.
func filesOf(g base.Game) *gameFiles { return campaignOf(g).files() }

// tableEdition is mapload.TableEdition(t), as a value.
func tableEdition(t *mapload.Table) base.Edition { return *mapload.TableEdition(t) }

// tableCampaign and tableFiles are the campaign service and file readers of
// that edition.
func tableCampaign(t *mapload.Table) campaignService { return campaignOf(tableEdition(t).Game) }

func tableFiles(t *mapload.Table) *gameFiles { return tableCampaign(t).files() }
