package game

import (
	"errors"
	"fmt"
	"image"
	"strings"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/textinput"
	"againrom/pkg/mapload"
	"againrom/pkg/render/menu"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// FrontEnd is the assembled game front-end: everything an install yields, loaded
// and validated, before any window opens.
//
// Construction is where every startup failure surfaces. Once a FrontEnd exists,
// the menu's assets are complete and dimension-consistent, the four required
// archives are open, the definition table is parsed, the tileset is built and the
// map list is known — so nothing
// downstream has to cope with a half-defined install.
type FrontEnd struct {
	InstallResources
	RuntimeServices
	CampaignSession
	Presentation
	PersistenceContext
}

// InstallResources is everything a lawful install yields, read once and never
// written again. A game does not change any of it, and every game this process
// opens is handed the same values.
//
// A field is here when the install is its only source. The art caches that are
// resolved on FIRST USE rather than at construction are NOT here, because what
// says whether they have been read is drawn state and sits in Presentation.
type InstallResources struct {
	endingAssets *ui.EndingView
	Cutscenes    *CutsceneBank // optional selected install; never session/save state
	Archives     *Archives
	Assets       *menu.Assets
	Tiles        *terrain.Tileset
	Maps         []MapEntry

	// Faces is where the picture of a speaker named in a mission's event text
	// comes from, and it is NIL — there is nothing in this tree that can
	// implement it yet.
	//
	// IT IS A FIELD AND NOT A CONSTRUCTION because the hop it stands for is a
	// missing decode rather than a missing decision. Where a speaker's face is
	// kept is not established by any published claim; the window, both its
	// shapes, both the tests that choose between them and the per-part face
	// selection are all built and all reachable, and what is absent is the one
	// answer that would let something be assigned here. A dialogue therefore
	// opens with its pane drawn and empty, which is the shipped behaviour and is
	// disclosed as such.
	Faces FaceSource

	// NPCFaces is what picture each `npc<n>` record of the scenario registry
	// calls for, keyed by that subscript — the table a dialogue's `npc=` tag is
	// resolved through (0141; data/npcface.go carries the reading and
	// speakers.go the measurement that chose it).
	//
	// IT IS BUILT ONCE, HERE, because a speaker's record is a fact about the
	// CAMPAIGN and not about a map: half the shipped speakers are not placed on
	// the map that quotes them, which is what ruled out reading the tag as a
	// placement key. Nil is a registry that would not open, and then every
	// dialogue draws its pane empty exactly as it did before this table existed.
	NPCFaces map[int32]data.NPCFace

	// Statics is the static-object class bundle every map this front-end opens
	// is drawn with. Like the tileset it is built ONCE, in NewFrontEnd, out of
	// graphics.res: the registry and every object sheet live in the one archive
	// the tiles come from, so a per-map load would re-read and re-decode the
	// same art on every trip back through the picker.
	//
	// The game draws this layer UNCONDITIONALLY and gains no flag for it: trees
	// and stones are the map's content, not a diagnostic, and the layer's own
	// instrument is the cross on the -markers switch that already exists. So
	// nothing here is a toggle — a front-end either has the bundle or failed to
	// start.
	//
	// It is nil only in a FrontEnd a test assembled by hand; NewFrontEnd either
	// fills it or returns an error. A nil one loads and draws nothing, which is
	// what keeps such a test's map screen the pre-story one.
	Statics *terrain.StaticSet

	// Units is the unit-art bundle every open map's entities resolve against.
	// Like Statics it is built ONCE, in NewFrontEnd, out of the graphics
	// archive already open, and handed to every map the picker opens; the
	// sprites it yields are the map's content, not a diagnostic, so there is no
	// flag anywhere on the path — a front-end either has the bundle or failed
	// to start.
	//
	// It is nil only in a FrontEnd a test assembled by hand; NewFrontEnd either
	// fills it or returns an error. A nil one draws every entity as the square
	// — byte for byte the screen this story inherits.
	Units *terrain.UnitSet

	// Structures is the structure-art bundle every open map's placed buildings
	// resolve against. Like Statics and Units it is built ONCE, in NewFrontEnd,
	// out of the container filesystem already open, and handed to every map the
	// picker opens.
	//
	// THERE IS NO FLAG ANYWHERE ON THIS PATH, exactly as there is none for the
	// object and unit bundles: a placed structure is the map's own content, not a
	// diagnostic, and a game that drew every other layer and left its buildings
	// out would be a game missing its buildings. The developer viewer keeps its
	// -structures switch, because a developer tool's job is to show one layer at
	// a time.
	//
	// It is nil only in a FrontEnd a test assembled by hand; NewFrontEnd either
	// fills it or returns an error. A nil one places nothing, which is byte for
	// byte the map screen this story inherits.
	Structures *terrain.StructureSet

	// Projectiles is the spell art bundle every cast on every map this
	// front-end opens is drawn from, keyed by picture id. Like the three
	// bundles above it is built ONCE, in NewFrontEnd, out of the container
	// filesystem already open.
	//
	// A NIL BUNDLE DRAWS NO SPELL ART and is not an error state: it is what a
	// FrontEnd a test assembled by hand holds, and what every driver in this
	// tree held before this story. A picture with no sheet in a bundle that
	// does exist is not an error either — 21 of the 28 spells compute a picture
	// the game ships no art for.
	Projectiles *terrain.EffectSet

	// Font is the game's own default font, loaded once here beside the two
	// bundles and handed to every map this front-end opens; the unit
	// information panel is drawn with it.
	//
	// A FAILED LOAD IS NOT FATAL, and that is where it parts company with the
	// two registries above. Those gate a map's CONTENT — an install that
	// cannot read its unit classes cannot draw the game — and refusing before
	// a window opens is worth it. This gates a three-line box to look at, and
	// making it fatal would put a working game on original assets behind a
	// cosmetic asset, on both releases, and make one missing archive entry
	// equivalent to a missing archive.
	//
	// So a front-end that could not load one holds no value here and the
	// reason beside it, and every map still opens and plays: a viewer holding
	// no font draws no panel and fails at nothing. The reason is kept rather
	// than discarded because an install whose font will not read is a fact
	// about that install, and dropping it is how such a thing goes unnoticed.
	Font lazy[*text.Font]

	// Words is the program-chosen word set this install resolves. It is read
	// once in NewFrontEnd and is immutable afterwards; App hands it to the
	// application and the town screen reads its notice-button word.
	//
	// A FrontEnd a test assembled by hand carries the ZERO value, whose fields
	// are empty strings. That is deliberate rather than an oversight: the only
	// reader inside this package is the town screen's button word, which draws
	// nothing on an empty label, and a hand-built front end draws no town. Every
	// path that draws goes through App, which never sees the zero value.
	Words ui.Words

	// ChargenAssets is the immutable, install-backed presentation handed to
	// every fresh character generator. It is loaded once at construction; a
	// generator must never open archive entries while a player is clicking it.
	ChargenAssets *ChargenAssets

	// TownSchoolArt is the install-backed skill-school surface. Unlike the
	// generator it is cosmetic: a missing node leaves a total text fallback and
	// its address-bearing reason beside the value.
	TownSchoolArt lazy[*ui.TownSchoolArt]
	TownTavernArt lazy[*ui.TownTavernArt]

	// TownSquareArt is the install-backed town square picture (1016), on the
	// same cosmetic rule as the two above: a missing node leaves the
	// row-button layout that drew before it existed, and its reason beside
	// the value.
	TownSquareArt lazy[*town.Art]

	// AttackPointer is the game's own attack cursor, resolved once here beside
	// the font and handed to every map this front-end opens; the map screen
	// draws it at the cursor while an attack is armed.
	//
	// IT IS THE FONT'S RULE EXACTLY, including the failure: a load that fails
	// carries no picture here and its reason beside it, and every map still
	// opens and plays. A viewer holding no picture draws its own authored mark
	// instead, so the mode stays visible either way — which is what makes the
	// non-fatal reading defensible here, where for the font it rests on a panel
	// simply being absent.
	//
	// The reason is kept rather than discarded for the font's own reason: an
	// install whose cursor art will not read is a fact about that install.
	AttackPointer  lazy[*image.RGBA]
	CursorRegistry lazy[*ui.CursorRegistry]

	// CommandPanelArt is the mission command panel's own four bitmaps,
	// resolved once here and handed to every map this front-end opens
	// (docs/1028-command-panel contract B1; `MENU-COMBAT-018`). AttackPointer's
	// own rule exactly, including the failure: a load that fails carries no
	// bitmaps here and its reason beside them, and every map still opens and
	// plays — the panel then draws nothing, which is the same "absent leaves
	// the slot empty" reading TownSquareArt states above for its own bundle.
	BottomHUDArt    lazy[*ui.BottomHUDArt]
	CommandPanelArt lazy[*ui.CommandPanelArt]

	// Table is the placeable-definition table every map this front-end opens
	// resolves its placements against. Like the two bundles above it is loaded
	// ONCE, in NewFrontEnd, and handed to every map the picker opens, so the
	// walk runs once per process rather than once per trip through the picker.
	//
	// It is nil only in a FrontEnd a test assembled by hand; NewFrontEnd either
	// fills it or returns an error. A nil one resolves nothing, which leaves
	// every placement in the ground domain at the provisional health pair —
	// byte for byte the world this story inherits.
	Table *mapload.Table

	// StartWeapon is the weapon character generation hands the party's hero,
	// resolved out of the SAME walk of the same file the table came from,
	// carrying beside it why there is none.
	//
	// The two are one value, like the font above, so a caller cannot reach the
	// weapon without the reason there is not one. A nil weapon is not fatal and
	// is not silent: the hero is bare, which the original permits, and CheckLine
	// says so.
	StartWeapon    lazy[*data.Weapon]
	SackFrames     []*terrain.StaticFrame
	SackBoundaries []*terrain.StaticFrame

	// Bodies is the shipped body list MissionParty's derivation reads, carried
	// straight from Definitions.Bodies alongside StartWeapon so a caller cannot
	// reach one without the other: resolving a party's weapon into a body name
	// needs both.
	Bodies data.BodyList

	// SoundBank is the resolved sound archive, or nil when it is absent,
	// unreadable, or its registry will not parse. OpenSounds' own nil already
	// answers every slot with silence — see SoundBank.Sample's comment
	// (sound.go) for why that holds even once this field is boxed into the
	// ui.SoundBank interface SetAudio takes.
	SoundBank  *SoundBank
	SpeechBank *SpeechBank

	// SoundClasses is every unit class's sound-slot array and attack delay,
	// resolved ONCE here out of the graphics archive already open, the shape
	// Units and Statics are resolved in. It is nil wherever LoadUnitSounds' own
	// nil answers (sound.go) — a nil map answers every class lookup with "no
	// sound", which is the swing's own silent state.
	SoundClasses map[int32]UnitSound

	// Music combines its channel volume with Sound's master settings. The bank opens
	// Allods/MUSIC.RES on first demand and the retained device owns one track.
	MusicBank *MusicBank
	Humans    data.Collection

	// Campaign is the mission set the scenario registry declares
	// (campaign.go), read ONCE here out of the container already open, the
	// shape Table and the NPC table are read in.
	//
	// LIKE Bodies THERE IS NO REFUSAL BEHIND IT: an install whose scenario
	// registry is absent or will not parse yields the zero Campaign, which
	// names no next mission and stops the front end from saying anything
	// about what follows — it does not stop a mission from opening. The
	// error is kept beside it rather than dropped, because "this install
	// declares no campaign" and "this install's campaign would not read"
	// are two different facts and only one of them is worth reporting.
	Campaign lazy[Campaign]
}

// RuntimeServices is the process's own devices and injectable seams: the
// opened audio hardware, the clocks and the bounded presentation generators.
//
// IT OWNS THE SEAM AND NOT WHAT THE SEAM PRODUCES. A generator lives here; the
// latch or counter its draws advanced is drawn state and lives in
// Presentation. Nothing here reaches simulation state or a save, and a new
// game keeps every field: a device opened once is reused for every game.
type RuntimeServices struct {
	// TownAnimationNow is an optional process-level presentation clock. Nil
	// uses time.Now. Session resets keep this service; townScreen resets the
	// animation state separately. It never participates in simulation or saves.
	TownAnimationNow func() time.Time

	// TownAnimationRandom supplies optional bounded presentation draws in
	// [0,n). Nil uses a private generator, never simulation RNG.
	TownAnimationRandom func(n int) int

	// TownAmbientRandom is the birds' separate bounded presentation source.
	// Keeping it separate preserves the established sign/fluger draw stream.
	TownAmbientRandom func(n int) int

	// TavernRandom is the tavern interior's separate bounded draw
	// seam. It is not shared with the exterior controller, so entering the
	// room cannot consume or reshape an outdoor presentation sequence.
	TavernRandom func(n int) int

	// ShopRandom is the shop interior's own bounded presentation draw seam.
	// It is not shared with the exterior, tavern or simulation generators.
	ShopRandom func(n int) int

	// SchoolRandom is the training room's separate 15-bit presentation draw
	// seam. It supplies idle extras and terminal holds without consuming the
	// exterior, tavern, shop or simulation generators.
	SchoolRandom func(n int) int

	// Sound is this front-end's master configuration — whether audio is
	// enabled at all and the master volume — SNAPSHOTTED from package state
	// (soundOptions, sound.go) at construction. A later SetSoundOptions call
	// does not reach a FrontEnd already built, exactly as a later SetPartySkill
	// does not reach a hero already generated.
	Sound SoundOptions

	// SoundChannels and SpeechPlayer belong to the process, not a loaded world.
	SoundChannels      audio.ChannelVolumes
	acknowledgmentsOff bool
	SpeechPlayer       audio.Player

	// SoundPlayer is the opened sound device, or nil.
	SoundPlayer audio.Player
	MusicPlayer ui.MusicDevice
	MusicSeed   int64

	// CutsceneAudioPlayer streams the installed decoder's own track for the
	// movie currently playing, under master controls only. Original
	// movie-to-channel routing remains Unknown (DIV-1255). Nil is silence,
	// exactly like a missing SoundPlayer or MusicPlayer.
	CutsceneAudioPlayer ui.CutsceneAudioDevice

	// AmbientPlayer owns the two retained mission loops. AmbientSeed drives
	// presentation-only deadline choices and never enters simulation state.
	AmbientPlayer ui.AmbientDevice
	AmbientSeed   int64

	// runtime keeps launch flags and the scenario frame adapter across games.
	runtime runtimeSwitches
}

// CampaignSession is the game currently being played, and it is EXACTLY what
// resetSessionForNewGame drops. That equality is the component's whole point:
// leaving a running game for the main menu has to end the session, and the
// question "is this field session state" now has one answer per field written
// where the field is declared.
//
// originalCity is the one field here whose subject is persistence rather than
// play. It is kept here because its LIFETIME is the session's exactly — a
// fresh game, a mission load or an old save without provenance clears it -
// and splitting it out would make the reset reach across two components.
type CampaignSession struct {
	fame        SnapshotFame
	quickSpells [4]uint32 // session-owned real spell IDs; 0 is unbound

	// Difficulty belongs to the campaign, not the process or the hero. Zero is
	// the legacy/default Normal value; new and restored sessions store 1..3.
	Difficulty mapload.Difficulty

	// Carried is the party the LAST WON MISSION left behind — its members
	// holding the experience, the packs and the worn sets their entities
	// ended with (mapload.CarryParty). It is EMPTY until a mission is won,
	// and an empty one is what makes every mission before the first win open
	// on MissionParty's own default hero exactly as it always did.
	//
	// IT LIVES FOR THE LIFE OF THE PROCESS AND GOES NOWHERE ELSE. Nothing
	// writes it to a file and nothing reads one: persisting a campaign is a
	// different goal with a different cost, and this field is the whole of
	// what the continuity hotfix claims — that winning a mission and opening
	// the next one no longer throws the character away.
	Carried []mapload.PartyMember

	// Offered is the mission the MAP LIST was last told to offer, and ZERO
	// wherever it was told nothing — before the first win, after the last
	// mission, and after a win that opened its successor directly instead of
	// returning here at all. It is a record rather than a gate: every mission
	// row is still choosable and choosing one still carries the party.
	Offered int

	// Town is the between-missions state (town.go). It is built over Campaign
	// at construction and is never nil on a front end this package builds —
	// every method on it also answers for a nil receiver, so a hand-built
	// FrontEnd in a test is a front end that has not reached a town rather than
	// one that panics.
	//
	// IT SITS BESIDE Carried AND ON ITS TERMS. Carried is the party; this is
	// everything else that survives a mission — the gold, what is done, what
	// the buildings handed out and what they have left. NOTHING WRITES EITHER
	// TO A FILE AND NOTHING READS ONE: persisting a campaign is a different
	// goal with a different cost, and this field is the whole of what 0142
	// claims. Where a save would attach is here and in Carried above, and there
	// is nothing else to find.
	Town *Town

	// originalCity is the detached semantic provenance of the original town
	// save that installed this session. It is deliberately neither raw .sav
	// bytes nor simulation state. SnapshotOriginalCity retains its semantic
	// model and eligibility baselines across native city saves. A fresh game,
	// mission load or old .ags without provenance clears it.
	originalCity *originalCitySaveState

	// live, liveMission and liveParty are which driver a save would reach,
	// which mission it is, and the party that mission was opened with (0143
	// plan D-4). Nothing above the map screen held the driver before this
	// story: MissionOpenerWith builds one inside a closure and returns seven
	// function values, so a save had no way to name the world it is a save
	// of.
	//
	// It is a PLAIN FIELD AND NOT A STACK: exactly one map screen is open at
	// a time, and every door that opens one passes through loadMap or
	// missionOpener.
	live        *mapWorld
	liveMission int
	liveParty   []mapload.PartyMember

	// Shop is the town merchant: his shelves and the table between him and the
	// player. It sits here and not on Town for the reason Town's own doc gives
	// for its six fields — a Town is what a save writes, and SHOP-SAVE-015
	// establishes no part of a shop's stock is ever written. arriveInTown fills
	// it; nothing else does.
	Shop *Shop
}

// Presentation is what is drawn and the state that decides how: the display
// options, the one installed town screen, the animation latches the runtime
// generators advanced, and every UI bundle resolved on first use rather than
// at construction.
//
// A first-use cache is here and not in InstallResources because its lazy TRIED
// flag is a fact about this process's drawing, not about the install: the
// install either holds the picture or does not, and what changes at runtime is
// only whether it has been looked for yet.
type Presentation struct {
	// Markers selects the diagnostic placement-marker overlays every map this
	// front-end opens is shown with. NewFrontEnd sets all three on, which is
	// the default the game ships with TODAY — the flag exists so that
	// reconsidering it is a decision someone makes rather than a change someone
	// notices.
	//
	// The third field is the static-object cross. It joined the other two here
	// rather than gaining a flag of its own because the game asks ONE question
	// about its diagnostics, and because the instrument that makes a misplaced
	// sprite visible has to exist in the binary the layer is played in — not
	// only in the developer tools the owner does not run.
	//
	// The field is what the game's -markers flag writes; it is read on every
	// load, so it can be flipped between two picks, not only at startup.
	Markers Markers

	// townProcess is the town composer's process-scoped state: the paint
	// clock, the wildlife generator, the bird delay latch and the star
	// terminal counter (TOWN-417, TOWN-419). It survives campaign resets and
	// never enters native or simulation state.
	townProcess     town.Process
	graphics        ui.GraphicsOptions
	showPathfinding bool

	// townUI is the one town screen App installed, held so that Restore can
	// put it back at the square (0143). See TownScreen's own doc.
	townUI *townScreen

	// shopArtCache is the shop screen's artwork. Its TRIED flag, not the
	// presence of a picture, is what says the load has run: an install missing
	// every shop picture must be read once and not once a frame.
	shopArtCache lazy[*ui.ShopScreenArt]

	// tipArtCache is the tip panel's own art and dialogArt the shared installed
	// window, portrait and minimap chrome, both cached on shopArtCache's own
	// precedent (1018 spec behaviour 1): tried once, not once a frame.
	tipArtCache lazy[*ui.TipPanelArt]
	dialogArt   lazy[*ui.DialogFrame]

	// docArtCache and docFontCache are the campaign documents panel's own art
	// and font, cached on the same rule the two fields above are (docsart.go).
	docArtCache        lazy[*ui.DocumentPanelArt]
	docFontCache       lazy[*text.Font]
	characterPaneCache lazy[TownCharacterPaneArt]

	// characterCornerCache is the same cache for the pane's nine corner
	// bitmaps (characterpane.go's own LoadCharacterPaneCornerArt). It is
	// separate from the field above because the two are resolved by two calls
	// with two failure modes: an install missing one corner node still gets
	// both mode-switched bodies, and a pane with bodies and no corners falls
	// back to the authored chrome for the corners alone rather than for the
	// whole region.
	characterCornerCache lazy[*ui.CharacterPaneCornerArt]
	characterFillerCache lazy[ui.TownPane]

	// worldMapCache is the install-backed campaign-map manifest. It is loaded
	// on the first gates entry and retained for the process, including failed
	// optional reads, so a missing picture is not retried every frame or visit.
	worldMapCache lazy[*worldMapAssets]
}

// PersistenceContext is where a preference or a record is read from and
// written to, plus the in-memory mirrors this build keeps so a frame does not
// read a file. The zero value has no path and behaves as "no store", which is
// the shape a hand-built FrontEnd in a test holds.
type PersistenceContext struct {
	hallStore fameStore

	// Options is the small persisted-preference store this build reads and
	// writes TipsMode, the normal GameSpeed rung and Ctrl+W's WimpyMode through
	// (1018 spec behaviour 4; stories 1064 and 1069; tipstore.go). The zero
	// value has no Path and behaves as "no store": TipsOff always reads the
	// original's own default (tips shown, TOWN-186) and SetTipsOff's own
	// write is silently a no-op — the same shape a nil Archives already
	// gives every other install-backed field.
	Options OptionsStore

	// tipsOff mirrors Options.TipsMode() and is read once by LoadOptions
	// rather than from disk on every frame the toggle's own drawn state is
	// asked for.
	tipsOff bool

	// smoothingOff mirrors Options.TextSmoothing() (DIV-1385) and
	// Options.FrameSmoothing() the same way, negated: the zero value leaves
	// method C's text overlay and the Catmull-Rom frame scaler ON, matching
	// every profile that predates either control, the same reasoning
	// tipsOff's own comment gives for tips.
	smoothingOff smoothingSwitches

	// wimpyMode is the current process preference label for Ctrl+W's next
	// cycle. It is not simulation state and loading it never applies it to a
	// world; parameter 3 remains the only threshold writer.
	wimpyMode int
}

// arriveInTown is the town arrival: the campaign's own (CampaignSession.
// arriveInTown, which latches the town open and stocks the merchant), then the
// town screen's reset of the party's remembered world-map position.
func (f *FrontEnd) arriveInTown() {
	if f == nil {
		return
	}
	f.CampaignSession.arriveInTown(f.townInstall())
	// The party's own remembered world-map position does not survive a
	// return to town (`DIV-137`, authored): this is the one place every
	// return to town passes through, whether from finishing a mission or
	// loading a save made in town.
	if !f.Town.restoredCampaign() {
		f.townUI.resetWorldPosition()
	}
}

// addChapterCompanions grants the companions a chapter's town hands out.
func (f *FrontEnd) addChapterCompanions(chapter int) {
	if f == nil {
		return
	}
	f.CampaignSession.addChapterCompanions(f.townInstall(), chapter)
}

// SetDeterministicFrames selects the no-wall-clock adapter used by the
// production headless scenario runner. It must be called before a map is opened.
func (f *FrontEnd) SetDeterministicFrames(on bool) {
	if f != nil {
		f.runtime.deterministicFrames = on
	}
}

// NewFrontEnd loads and validates everything the front-end needs from the asset
// root at root.
//
// The order is: archives, the definition table, menu assets, the object bundle,
// tileset, map list. Every failure is reported before a window could open, which
// is what makes the headless check and the windowed run agree about whether an
// install is usable. The table comes BEFORE the map list deliberately: a root
// that cannot yield one must list no map rather than offer rows that fail on
// the pick.
//
// The tileset and the object bundle are built ONCE here and reused for every map
// the user picks, so graphics.res is read exactly once per process rather than
// on every trip back through the picker.
//
// AN UNREADABLE OBJECT REGISTRY FAILS STARTUP (0017 AC-7), AND SO DOES AN
// UNREADABLE UNIT REGISTRY (AC-10). Each is the one part of its layer that
// is an error at all — a class whose art cannot be decoded is skipped and
// counted, and an id naming no class draws its fallback — because a
// registry that will not read is an install missing a piece the game needs,
// exactly like a missing archive, and the check exists to say so before a
// window opens rather than after. Both loads are headless and GPU-free, so
// -check reaches them with no graphics context.
func NewFrontEnd(root string) (*FrontEnd, error) {
	share, err := sharedInstall(root)
	if err != nil {
		return nil, err
	}
	archives := share.archives
	// Before the map list is scanned, so a root with no table lists no map.
	defs, err := LoadDefinitionsFor(archives.Containers, archives.Game())
	if err != nil {
		return nil, err
	}
	table := defs.Table

	loadMenu := menu.Load
	if archives.Game().Edition().SecondMenu {
		loadMenu = menu.LoadSecond
	}
	assets, err := loadMenu(archives.Containers)
	if err != nil {
		return nil, err
	}

	if share.staticsErr != nil {
		return nil, share.staticsErr
	}
	statics := cloneCandidateStatics(share.statics)

	if share.unitsErr != nil {
		return nil, share.unitsErr
	}
	units := cloneCandidateUnits(share.units)

	openingParty := MissionParty(defs.StartWeapon, defs.Bodies, table)
	LoadHeroBody(archives.Containers, units, openingParty[0].BodyDir, data.HeroBody(openingParty[0].Body))

	// AN UNREADABLE STRUCTURE REGISTRY FAILS STARTUP, for the reason an
	// unreadable object or unit registry does: a registry that will not read is
	// an install missing a piece the game needs, and the check exists to say so
	// before a window opens rather than after. Every class it cannot draw is a
	// skip the census counts, so only the registry itself is fatal.
	if share.structuresErr != nil {
		return nil, share.structuresErr
	}
	structures := cloneCandidateStructures(share.structures)

	// THE SOUND SUBSYSTEM, resolved here exactly once and apart from the four
	// required archives above: the archive and its slot table, the per-class
	// sound array and attack delay off the graphics archive already open, and
	// the device itself, opened at this PROCESS'S OWN soundOptions —
	// cmd/againrom's SetSoundOptions call, when there is one, has already run
	// by the time this line does (main.go's own ordering, mirroring
	// SetPartySkill's).
	soundBank := OpenSounds(root)
	soundClasses := share.unitSounds
	audioSettings := audio.Settings{
		Master: soundOptions.Volume,
		Muted:  !soundOptions.Enabled,
	}
	sharedAudio, _ := ui.OpenSharedAudio(soundChannelVolumes.Settings(audio.EffectsChannel, audioSettings),
		soundChannelVolumes.Settings(audio.SpeechChannel, audioSettings))
	processAudio := sharedAudio.NewScope()
	soundPlayer, speechPlayer := processAudio.Player(audio.EffectsChannel), processAudio.Player(audio.SpeechChannel)
	musicBank := OpenMusic(root)
	musicPlayer, _ := ui.OpenMusic(soundChannelVolumes.Settings(audio.MusicChannel, audioSettings))
	ambientPlayer := processAudio.Ambient()
	cutsceneAudioPlayer, _ := ui.OpenCutsceneAudio(audioSettings)

	// The campaign spine, off the same container filesystem and under the
	// carried-error rule the font below states (campaign.go). It is read HERE
	// and once, beside the definition table and the NPC table, rather than at
	// the moment a mission ends — the tier that owns a running mission opens
	// nothing, and a front end that went looking for an archive after a win
	// would answer differently depending on when it was asked.
	campaign, campaignErr := LoadCampaign(archives.Containers)
	if campaignErr == nil {
		campaign = WithDefaultTransitionRewards(campaign)
	}

	// Off the same container filesystem as everything above, and its error is
	// CARRIED rather than returned. The two results are assigned together, so a
	// caller cannot reach the font without the variable that says why there is
	// none.
	font, fontErr := LoadFont(archives.Containers, DefaultFont)

	// Character generation is now on every ordinary mission-entry route, so its
	// fixed controls are part of a usable install rather than an optional map
	// decoration. Resolve the whole immutable payload before a window can open;
	// a missing node keeps its address in the construction error.
	//
	// A base whose profile states it ships no generation art (the demo) skips the
	// load: its new game opens the first mission with the default party, so the
	// payload is never read.
	var chargenAssets *ChargenAssets
	if !archives.Base.Profile.Limits.NoCharacterGeneration {
		if chargenAssets, err = LoadChargenAssets(archives.Containers); err != nil {
			return nil, err
		}
	}
	townSchoolArt, townSchoolArtErr := share.townSchool, share.townSchoolErr
	townTavernArt, townTavernArtErr := share.townTavern, share.townTavernErr
	townSquareArt, townSquareArtErr := share.townSquare, share.townSquareErr

	// The attack pointer and the cursor registry, off the same container
	// filesystem and under the same carried-error rule (docs/1030-cursor-lifecycle
	// B1). The pointer is frame 0 of the registry's own decoded attack sheet.
	cursors := loadCursorArt(archives.Containers)
	pointer, pointerErr := cursors.pointer, cursors.pointerErr
	cursorRegistry, cursorRegistryErr := cursors.registry, cursors.registryErr

	// The command panel's four bitmaps, off the same container filesystem and
	// under the same carried-error rule (docs/1028-command-panel contract B1).
	commandPanelArt, commandPanelArtErr := LoadCommandPanelArt(archives.Containers)
	bottomHUDArt, bottomHUDArtErr := LoadBottomHUDArt(archives.Containers)

	sackFrames := LoadSackFrames(archives.Containers)

	// THE PROGRAM-CHOSEN WORDS ARE READ ONCE, HERE, beside the selector below
	// and by the same rule. They are a property of the install, which does not
	// change while the program runs, and the drawing path must stay free of
	// archive reads.
	//
	// It cannot fail: an install stating neither text file yields a set in which
	// every index is absent, and every word then stays this build's authored
	// English. That is the same shape the font, the attack pointer and the spell
	// art already carry — a cosmetic source does not stop a mission opening.
	words := LoadInstallWords(archives.Containers)

	// THE LANGUAGE SELECTOR IS READ ONCE, HERE, and written onto the one font
	// this front-end keeps. It is read at startup rather than per draw because
	// it is a property of the install, which does not change while the program
	// runs, and because the drawing path must stay free of archive reads.
	//
	// It cannot fail: an install that names no language selects 0, the identity
	// rule, so this line either makes Russian text draw correctly or changes
	// nothing at all. A nil font is left alone — there is nothing to set it on,
	// and a selector without an atlas draws no more than an atlas without one.
	if font != nil {
		font.Selector = LanguageSelector(archives.Containers)
	}

	// The loose scan still walks the host — the filesystem enumerates no
	// directory tier, so nothing else can say which files an install ships
	// beside its containers — but every row it lists is READ through the
	// loose filesystem opened over that same root.
	loose, err := DirMaps(root, archives.Loose)
	if err != nil {
		return nil, err
	}

	return &FrontEnd{
		InstallResources: InstallResources{
			Archives: archives,
			Assets:   assets,
			// The resolved word set. App hands it to the application, and the town
			// screen reads its notice-button word directly, because that screen
			// composes its own dialogue layout.
			Words: reportWordsFor(words, archives.Game()),
			// Off the CONTAINER FILESYSTEM, like the menu and the two loaders above:
			// terrain.TilePathPrefix and DirtPath carry graphics.res's identity
			// segment now, so the tile strips are named by address and the render
			// package says which container they come from. The graphics handle this
			// line used to stand beside is gone with the teardown: nothing in this
			// package holds a `*res.Archive` any more.
			Tiles: terrain.LoadTileset(archives.Containers),
			// The campaign's speaker table, read once (0141). It reaches for the
			// same registry the placement table already opened; a failure here is
			// no faces and no error, because that file's own failure is already
			// reported by the loader above and reporting it twice would say the
			// install is broken in two ways.
			NPCFaces:   LoadNPCFaces(archives.Containers),
			Statics:    statics,
			Units:      units,
			Structures: structures,
			// THE SPELL ART, and its failure is NOT fatal. The three bundles above are
			// the map's own furniture and a run without them is a run missing its
			// ground; a projectile sheet is what a cast looks like, and the font's
			// rule applies — a mission does not fail to open over a cosmetic asset.
			// An install with no projectile registry draws no spell art and starts.
			Projectiles: loadProjectilesOrNil(archives.Containers),
			// Off the CONTAINER FILESYSTEM's enumeration, filtered to the scenario
			// identity, rather than off a scan of the scenario handle: the campaign
			// maps are addressed like every other entry in the set. Every container
			// this front-end reads is now reached the one way, by an address carrying
			// its identity.
			//
			// WithMissions is composed OVER the finished list, not folded into
			// BuildMapList: the mission rows are a second question asked of the same
			// answer, so this is the one place both questions are asked and
			// BuildMapList itself stays a pure function of the two sources.
			Maps:          WithMissions(BuildMapList(loose, ArchiveMaps(archives.Containers))),
			Font:          resolved(font, fontErr),
			ChargenAssets: chargenAssets,
			TownSchoolArt: resolved(townSchoolArt, townSchoolArtErr),
			TownTavernArt: resolved(townTavernArt, townTavernArtErr),
			TownSquareArt: resolved(townSquareArt, townSquareArtErr),
			Table:         table,

			AttackPointer: resolved(pointer, pointerErr),

			CursorRegistry: resolved(cursorRegistry, cursorRegistryErr),

			BottomHUDArt:    resolved(bottomHUDArt, bottomHUDArtErr),
			CommandPanelArt: resolved(commandPanelArt, commandPanelArtErr),

			SackFrames:     sackFrames,
			SackBoundaries: LoadSackBoundaries(archives.Containers),

			StartWeapon: resolved(defs.StartWeapon, defs.StartWeaponErr),
			Bodies:      defs.Bodies,

			SoundBank:    soundBank,
			SpeechBank:   OpenSpeech(root),
			SoundClasses: soundClasses,
			MusicBank:    musicBank,

			// Off the SAME walk as Table itself — defs.Table.Humans is exactly
			// what LoadDefinitions (table.go) already assembled the placement
			// collections from, so this is a second field naming a value that
			// exists rather than a second read of the file.
			Humans: defs.Table.Humans,

			// Read out of the container filesystem already open, beside the NPC
			// table and on its reasoning (campaign.go). NOT FATAL and reported
			// nowhere but the field: an install that will not yield a campaign
			// still opens every mission it holds, and the only thing it cannot
			// do is name what follows one.
			Campaign: resolved(campaign, campaignErr),
		},
		RuntimeServices: RuntimeServices{
			Sound:               soundOptions,
			SoundChannels:       soundChannelVolumes,
			SpeechPlayer:        speechPlayer,
			SoundPlayer:         soundPlayer,
			MusicPlayer:         musicPlayer,
			MusicSeed:           time.Now().UnixNano(),
			AmbientPlayer:       ambientPlayer,
			AmbientSeed:         time.Now().UnixNano(),
			CutsceneAudioPlayer: cutsceneAudioPlayer,
		},
		CampaignSession: CampaignSession{
			fame: SnapshotFame{Known: true},
			// Built over the campaign just read, on the same statement, so there is no
			// window in which a front end holds a campaign and no town for it. A zero
			// campaign yields a town with no chapter, which is what an install
			// declaring none has always meant one level up.
			Town: NewTown(campaign),
		},
		Presentation: Presentation{
			// The default lives here and nowhere else, so the command's flag has one
			// value to override rather than a second copy to agree with.
			Markers: Markers{Objects: true, Units: true, Statics: true},
		},
	}, nil
}

// CheckLine is the one-line summary the headless mode prints: how many rows the
// map list holds, and for how many of the eight brooch buttons the hit mask
// carries a non-empty region.
//
// A button with an empty mask region is REPORTED, not rejected — reporting it is
// precisely what this mode is for. An install where the count is short of eight
// still exits successfully; only a missing or inconsistent asset fails.
func (f *FrontEnd) CheckLine() string {
	regions := f.Assets.MaskRegions()
	withRegion := 0
	for _, n := range regions {
		if n > 0 {
			withRegion++
		}
	}
	line := fmt.Sprintf("againrom: %d map rows, %d of %d buttons have a mask region",
		len(f.Maps), withRegion, menu.ButtonCount)
	// The font is reported only when it FAILED. It is not fatal, so this is
	// the one place a headless run could ever say so, and a line that named it
	// on every successful run would bury the case that matters.
	if f.Font.Err() != nil {
		line += fmt.Sprintf("; no font: %v", f.Font.Err())
	}
	// The attack pointer, reported on the same rule and for the same reason: it
	// is not fatal, so a headless run is the one place it could ever be said,
	// and saying it on a successful run would bury the case that matters.
	if f.AttackPointer.Err() != nil {
		line += fmt.Sprintf("; no attack pointer: %v", f.AttackPointer.Err())
	}
	// The cursor registry, on the same rule. A registry that will not resolve
	// is not fatal either: every screen then falls back to the operating
	// system's arrow, silently and correctly, which is exactly the state no
	// instrument in this build could otherwise report.
	if f.CursorRegistry.Err() != nil {
		line += fmt.Sprintf("; no cursor registry: %v", f.CursorRegistry.Err())
	}
	// THE HERO'S SHEET IS REPORTED ON EVERY RUN, which is the opposite of the
	// rule above and deliberate. The band is what this install makes of the
	// definition table's own numbers, so it is the one figure a headless run
	// can state that says whether the player's own unit can fight; printing it
	// only on failure would bury the case that matters HERE, since a bare hero
	// and an armed one both start cleanly and only the numbers differ.
	line += "; " + f.heroSheet()
	return line
}

// heroSheet is the party hero's character and the numbers it comes to: his four
// statistics, his trained skill, and then either his weapon and its band or why
// he has none.
//
// THE SPREAD IS STATED BESIDE THE NUMBERS AND NOT INSTEAD OF THEM. A line
// carrying the numbers alone cannot say WHICH BUILD produced them, and the
// build is the thing that moves — the numbers are a function of it.
// Stating both is what lets one line witness that a change to the spread
// reached the derive.
//
// Nothing else builds a band, so the composition `[base, base + spread]` — which
// is the character sheet's own — has one spelling.
func (f *FrontEnd) heroSheet() string {
	h := PartyHero()
	s := PartySpread()
	line := fmt.Sprintf("hero Body %d, Reaction %d, Mind %d, Spirit %d, %s %d",
		s.Body, s.Reaction, s.Mind, s.Spirit,
		data.SkillName(PartySkillSlot()), h.Skill[PartySkillSlot()])

	if f.StartWeapon.Value() == nil {
		if f.StartWeapon.Err() != nil {
			return line + fmt.Sprintf(", bare: %v", f.StartWeapon.Err())
		}
		// A front-end assembled by hand, which resolved nothing and therefore
		// has no reason to give. It is still bare and still says so.
		return line + ", bare"
	}
	c := h.Derive(f.StartWeapon.Value())
	return line + fmt.Sprintf(", %s %d-%d, to-hit %d, defence %d",
		f.StartWeapon.Value().Name, c.DamageBase, c.DamageBase+c.DamageSpread, c.ToHit, c.Defence)
}

// App builds the windowed front-end over this install.
//
// The loader closure is where a picker row becomes a running map: it reads the
// chosen row's bytes from wherever that row came from, then goes through the same
// LoadMapViewer the standalone viewer uses. A row that fails here is reported and
// marked unusable rather than crashing the program — a map can list cleanly on
// its metadata and still fail its full decode.
//
// THE NEW-GAME GENERATION GATE IS INSTALLED HERE, and this is the tier that
// decides which rows want it (0140): a MISSION row does, every other row does
// not. pkg/ui asks the gate one row at a time and is told a model and a callback
// or nothing at all — it never learns what a mission is, which is what keeps its
// import list clear of pkg/data.
//
// A ROW WANTS GENERATION BECAUSE THE PLAYER IS STARTING A CAMPAIGN, not
// because a flag was given, and not merely because a mission is being
// entered. A mission the CAMPAIGN itself opens — the successor of one just
// won — is the same campaign being continued, and the character is the one
// he has been playing.
//
// THE SEAM CANNOT REACH THE SECOND CASE, which is why the distinction is
// structural and not a condition anyone has to maintain: pkg/ui reads this gate
// in choose() alone, choose() runs on the picker alone, and a won mission's
// successor is entered by advanceNotice, which goes straight to the same enter()
// every other door uses. See pkg/ui's own TestCampaignAdvanceNeverArmsGeneration,
// which installs a gate claiming EVERY row and wins a mission.
//
// It replaces 0119's own decision, which threaded a generated party through
// exactly one door and opened every picker row on this front end's default hero.
//
// A LOOSE MAP ROW IS UNTOUCHED. Those are not missions, they carry no party at
// all, and generating a character to walk one would be inventing a game mode
// (loadMap's own e.Mission == 0 arm).
func (f *FrontEnd) App(title string) *ui.App {
	// THE ENCODER READS THE SAME FONT'S SELECTOR the SetWords call below
	// names in full (MENU-KEY-013); declared once here so the map-list
	// hover hint's own re-encode, immediately below, and SetWords share it
	// rather than resolving the font a second time.
	selector := 0
	if f.Font.Value() != nil {
		selector = f.Font.Value().Selector
	}
	// THE MAP-LIST HOVER HINT'S ROW-OWNED TEXT (TEXT-HOVERTEXT-052) IS
	// RE-ENCODED HERE, not carried as alm.Info decoded it. Description is
	// UTF-8 (alm.Info's own documented shape); every other hover string in
	// this tree stays in the install's own single-byte alphabet because the
	// font this build shares across every hint (SetTooltipFont, below) walks
	// bytes and not runes. Selector 0 when the font itself would not load
	// leaves the description ASCII-only, matching every other word this
	// front end resolves under a font it could not read.
	rows := make([]ui.PickerRow, 0, len(f.Maps))
	for _, e := range f.Maps {
		rows = append(rows, ui.PickerRow{Text: e.Text(), Choosable: e.Choosable(),
			Width: e.Width, Height: e.Height, Description: EncodeInstallText(e.Description, selector),
			Word70: e.Word70, Word74: e.Word74})
	}
	a := ui.NewApp(title, f.Assets, rows, f.loadMap)
	a.SetCheatCommands(f.chatCommand, f.debugLetter)
	tooltipDelay, _ := f.Options.TooltipDelay()
	a.SetTooltipDelayPreference(tooltipDelay, f.Options.SetTooltipDelay)
	a.SetTooltipFont(f.tipFont())
	a.SetTextSmoothing(!f.smoothingOff.text)
	a.SetFrameSmoothing(!f.smoothingOff.frame)
	edition := f.Base().Profile.Edition()
	if f.Cutscenes != nil {
		a.SetCutscenes(f.Cutscenes)
	} else if f.Archives != nil && edition.CutsceneArchive != "" {
		a.SetCutscenes(OpenCutscenes(f.Archives.Root, edition.CutsceneArchive))
	}
	if f.Archives != nil && !edition.StartupCutscenes {
		a.SetStartupCutscenesEnabled(false)
	}
	// GameSpeed is process-local driver state, not campaign/save state. Read it
	// when this App is built, then let the App retain the chosen normal rung
	// across every map it opens. The sink writes +/- and Game Options changes; the
	// separate unpaced selector never reaches OptionsStore.
	gameSpeed, _ := f.Options.GameSpeed()
	a.SetMapCadencePreference(gameSpeed, func(rung int) {
		_ = f.Options.SetGameSpeed(rung)
	})
	a.SetChargenGate(f.newGameChargen)
	if f.directNewGame() {
		a.SetNewGameDirect(func() ui.MapOpener { return f.DirectNewGame(f.Base().Profile.Mission()) })
	}
	a.SetCampaignEnding(f.campaignEnding)
	a.SetHallOfFame(f.hallOfFame)
	a.SetCampaignEndingExit(f.leaveCampaignEnding)
	a.SetGameMenuArt(f.gameMenuArt())
	f.wireMediaMenu(a)
	f.wireLoadWindow(a)
	// Interface events have no map Viewer, but resolve through the same
	// install-backed bank and concrete device as unit and spell sounds.
	a.SetAudio(f.SoundPlayer, f.SoundBank)
	a.SetSpeechAudio(f.SpeechPlayer)
	a.SetAmbientDevice(f.AmbientPlayer)
	a.SetCutsceneAudio(f.CutsceneAudioPlayer)
	// The cursor registry rides beside the chargen gate, on its own reasoning:
	// resolved once at startup, installed once here (docs/1030-cursor-
	// lifecycle B1, B2). Nil is the ordinary case for an install whose cursor
	// art would not read, and every screen then draws no cursor of its own.
	a.SetCursorRegistry(f.CursorRegistry.Value())
	// THE WORDS BEFORE THE TOWN. SetTown builds the town screen, which reads
	// this front end's own copy; setting the application's set first keeps the
	// order of these three lines the order a reader would guess.
	//
	// THE ENCODER IS textinput.EncodeRune UNDER selector, ABOVE. It is the
	// same call chargen.go's EncodeName already makes for a typed character
	// name; a nil font (FontErr) left selector 0, ASCII-only, which is what
	// an install with no readable font already draws.
	a.SetWords(f.Words, f.Font.Value(), func(r rune) (byte, bool) {
		return textinput.EncodeRune(r, selector)
	})
	// THE TOWN IS INSTALLED AND NOT ENTERED. It stands behind every screen from
	// here on, so a mission that ends into it has somewhere to end; what puts
	// the player there is the campaign reaching its own town boundary, and
	// nothing else.
	a.SetTown(f.TownScreen())
	// THE DOCUMENTS PANEL'S SEAM. The source is a value over this front end and
	// not over the collection, so the panel reads the collection as it stands
	// when it opens rather than as it stood here; the art and the font are
	// resolved once, and nil is the ordinary case for an install whose nodes
	// would not read, which leaves the panel drawing a black sheet it can still
	// be left from.
	a.SetDocuments(documentSource{f}, f.documentArt(), f.documentFont())
	a.SetGameMenuSettings(
		func() bool { return !f.TipsOff() },
		func(on bool) { f.SetTipsOff(!on) },
		f.gameMenuSound,
		f.setGameMenuSound,
	)
	f.wireGameOptions(a)
	f.wireSoundOptions(a)
	a.SetMusic(f.MusicBank, f.MusicPlayer, f.MusicSeed)
	return a
}

// newGameChargen is ui.ChargenGate over this front end's map list: a mission row
// chosen from it opens the generation screen, and every other row opens its map
// (0140).
//
// THE NAME PREDATES THE OWNER'S REVERSAL and is kept rather than chased: this
// gate fires in choose() alone (flow.go's own field comment), and choose()
// only ever runs on a row picked from THIS list, which cmd/againrom's default
// wiring no longer routes NEW GAME through at all — only -picker's own debug
// route does now (SetNewGameChargen's own doc on ui.App carries the direct
// route the default takes instead). "The list a picker row is chosen from" is
// still the whole of what this method gates; which button reaches that list is
// cmd/againrom's decision and not this method's.
//
// THE SETUP AND THE PARTY COME FROM THE SAME TWO EXPRESSIONS cmd/againrom's own
// -mission door and default NEW GAME arm both use — ChargenSetup and
// ChargenParty, chargen.go's own pair — so the screen a picker row opens and
// the screen either of those opens are one screen, and a change to what
// generation offers reaches all three or none.
//
// A FRESH MODEL PER ROW, deliberately. ui.NewChargen is called inside the gate
// rather than once in App: a player who backs out of generating a character and
// picks another mission is starting again, not resuming a spread he abandoned,
// and a shared model would hand him the second mission with the first one's
// half-spent points already on it.
//
// THE GENERATED PARTY REACHES THE CHOSEN MISSION AND NOTHING ELSE. It is handed
// to MissionOpenerWith inside begin and held nowhere this front end can read it
// back from, so a mission the CAMPAIGN opens — a successor after a win — still
// comes through MissionOpener and NextParty, which carry the character the
// player has been playing (continuity's own doc below, and continuity_test.go's
// own coverage of the carry, both unchanged by this story). Generation is what
// STARTING a campaign does; it is not what finishing a mission does, and a
// screen between two campaign missions would throw away the hero the win was
// earned with.
func (f *FrontEnd) newGameChargen(row int) *ui.ChargenEntry {
	if row < 0 || row >= len(f.Maps) || f.directNewGame() {
		return nil
	}
	n := f.Maps[row].Mission
	if n <= 0 {
		return nil
	}
	return &ui.ChargenEntry{
		Model: ui.NewChargen(f.ChargenSetup()),
		Begin: func(res ui.ChargenResult) (ui.MapOpener, error) {
			// The previous campaign survives a failed open. The successful
			// candidate replaces its session fields and town UI together.
			return f.NewGameOpener(n, res), nil
		},
	}
}

// resetSessionForNewGame drops everything the PREVIOUS game session left on
// the front end, so a mission chosen after a return to the main menu opens
// against a fresh campaign rather than the one that was running before the
// player left (1019: the brooch's NEW GAME did not end the running session,
// so a mission picked afterwards inherited the old town's gold, chapter and
// finished/available/taken state, and could even resume drawing the old
// mission's own world through f.live).
//
// IT IS CALLED FROM EVERY SITE THAT COMMITS TO INSTALLING A GENUINELY
// DIFFERENT GAME into this long-lived FrontEnd (1019 round 2's own P
// finding): newGameChargen's Begin, and both arms of RestoreOriginal — the
// between-mission arm calls it and then installs decoded party, purse and
// campaign state, while the mid-mission arm installs the same validated
// campaign projection beside the opener's restored party. It is called beside townUI.resetForNewGame
// at each site, on that method's own precedent: pkg/ui has no notion of a
// game session and cannot call back into this package, so the reset happens
// where the new session is confirmed to begin rather than where the old one
// was left.
//
// resume.go's installCandidate is the fourth such site and does NOT call this
// method: it already assigns Town, Carried and Offered unconditionally from
// the restored candidate before its own townOnly/mission split, and its
// townOnly arm already clears live/liveMission/liveParty itself — verified
// while enumerating this population (round 2), not assumed from the earlier
// round's own report of it.
//
// THIS ENUMERATION'S OWN METHOD: every write site of FrontEnd's eight session
// fields (Carried, Offered, Town, originalCity, live, liveMission, liveParty,
// Shop) in this
// package's non-test files was found with
// `grep -rn "\.Town\s*=\|\.Carried\s*=\|\.Offered\s*=\|\.live\b\|\.Shop\s*="
// pkg/game/*.go` and read. Every hit is either (a) an ONGOING GAMEPLAY
// mutation inside an already-running session — FinishMissionWithRoster's own
// Carried/Offered writes on a win, the tavern's hire/fire, arriveInTown's own
// Shop rebuild, liveDriver's own overwrite on opening ANY mission — which
// must not be touched, or (b) one of the four SESSION-INSTALL sites named
// above. A method, not a count, because the count changes if a fifth
// install site is ever added and the grep does not: re-run it and reclassify
// every new hit the same way.
//
// THE WHOLE OF THE SESSION STATE THIS METHOD RESETS IS THE EIGHT FIELDS
// BELOW, WRITTEN IN SIX STATEMENTS: the live triple is assigned together.
// That is the same eight the enumeration above names. An earlier draft of
// this comment said five in one place and seven in the other, counting
// statements once and fields once, which is why the unit is spelt out here.
// The classification is verified against this package's own field comments
// (FrontEnd's own doc): Carried, Offered, Town and live/liveMission/liveParty
// are documented as living for "the life of the process" or "for a mission"
// respectively, and neither anticipates a second game in the same process.
// Shop is rebuilt whole by arriveInTown on every town arrival
// (SHOP-TOWN-022) and is reset here anyway, so a caller can never observe
// the previous game's merchant stock in the window before the new game's
// first arrival.
//
// TOWN IS REBUILT, NOT ZEROED: NewTown(f.Campaign) is the same construction
// NewFrontEnd uses at process start, so a second game starts exactly as the
// first one did rather than on a *Town holding stale internal maps.
//
// The whole population OF THIS METHOD'S OWN FIELDS, and no more and no less,
// is enforced by TestResetSessionForNewGameDropsExactlyTheSessionPopulation
// (frontend_session_test.go): it walks FrontEnd with reflect on a value with
// every field set non-zero, calls this method, and requires every field back
// at its own expected value except the install-scoped fields named in that
// test's own keep-list, each with its own one-line reason. A field added to
// the struct later fails that test until it is classified. THE POPULATION OF
// CALL SITES is a second question this test does not answer — see
// worldmapsession_test.go for the site-by-site witness.
func (f *FrontEnd) resetSessionForNewGame() {
	if f == nil {
		return
	}
	f.CampaignSession.clear(f.Campaign.Value())
}

// loadMap turns the picker's row index into a viewer, the tick that advances the
// world running under it, and the seam an order reaches that same world through.
//
// The world is built from mv.Map — THE MAP THIS CALL ALREADY DECODED, and
// not from a second decode of the same bytes. One decode is what makes "the
// world and the terrain on screen describe the same map" true by
// construction rather than by a consistency check somebody has to keep: the
// bounds agree because there is only one map, and there is nothing for a
// second decode to disagree with.
//
// Nothing is built inside LoadMapViewer, which serves cmd/mapview too:
// building the world out here is exactly what leaves the standalone viewer
// no world to own or advance. A world it ignored would still be a world it
// owned.
//
// A failure before this point returns no tick and no order seam at all, so a
// row that will not decode leaves the front-end with nothing to advance and
// nothing to order.
func (f *FrontEnd) loadMap(index int) (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
	if index < 0 || index >= len(f.Maps) {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("no map at row %d", index)
	}
	e := f.Maps[index]

	// A mission row (e.Mission > 0) routes to the mission door instead of the
	// plain read below, and does nothing else. MissionOpener is loadMap's own
	// twin — same return tuple, same LoadMapViewer, same font and bundles —
	// so this is a delegation to the one place a mission already opens, not a
	// second construction of a map screen: a change to how a mission starts
	// reaches both the -mission flag and this row through the one closure, and
	// cannot reach one and miss the other. Every other row is Mission == 0 and
	// falls through to the read this function has always performed.
	//
	// THE WINDOWED APPLICATION DOES NOT REACH THIS ARM FOR A MISSION ROW ANY
	// MORE (0140). App above installs a ui.ChargenGate that claims exactly the
	// rows with e.Mission > 0, and pkg/ui asks that gate BEFORE it calls this
	// loader — so a mission chosen from the map list opens the generation
	// screen and the map is opened afterwards, by MissionOpenerWith, out of the
	// character the player just made.
	//
	// THE ARM STAYS ALL THE SAME, and not as a leftover. This is a ui.MapLoader
	// and its contract is to answer for any row it is handed: a caller with no
	// gate installed — a test, a front end assembled by hand, a later front end
	// that wants a mission row to open directly — must still get that mission's
	// map rather than a read of a file the row does not name. What the arm opens
	// it on is MissionOpener, which is NextParty: the default hero, or the one
	// carried out of the last won mission.
	if e.Mission > 0 {
		return f.MissionOpener(e.Mission)()
	}

	data, err := f.mapBytes(e)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("read %s: %w", e.Source, err)
	}
	// The art is ON and takes no flag: the static-object layer is game content
	// here, not a diagnostic, so every map the game opens draws it. The bundle
	// is the one loaded at startup, handed over per map because the load path
	// takes it as a parameter and offers no setter. Every class carrying a
	// cycle is a candidate. The viewer applies the live fog state at draw time,
	// so visibility changes need no map rebuild. The structure layer rides
	// exactly the same way and for the same reason: the bundle loaded at
	// startup, handed over per map, art on and no flag anywhere on the path. A
	// front-end holding none — a hand-assembled one — passes nil and draws
	// no building, which is the map screen this story inherits.
	mv, err := LoadMapViewerFor(f.Archives.Game(), f.Tiles, data, e.Source, f.Markers,
		StaticLayer{Set: f.Statics, Art: true, AnimGate: terrain.AnimGateAll},
		StructureLayer{Set: f.Structures, Art: true})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("load %s: %w", e.Source, err)
	}
	// The unit bundle rides the same way the object one does: the one loaded at
	// startup, handed over per map, no flag anywhere on the path. A front-end
	// holding none — a hand-assembled one — passes nil and draws every
	// entity as the square. The advance handed back is the PACED one: the
	// front-end calls it once per map-screen tick, and it runs however many
	// whole logic ticks the elapsed wall-clock time is worth at the decoded
	// rate.
	//
	// The order seam beside it is that same world's enqueue, and the two are
	// bound to ONE mapWorld here: the queue an order lands in is the queue the
	// very next advance drains, and neither outlives the map screen, because
	// the front-end holds both only through the closures this line hands back.
	//
	// The cadence seam is that same world's setCadenceMode, bound to the SAME
	// mapWorld by this one line: the clock a rate re-rates is the clock the
	// very next advance paces by, and the stop it sets is the one that advance
	// reads. The water half of that rate is written on the other side of the
	// seam, in the front-end, out of the same statement — so nothing here can
	// re-rate a world without re-rating the water it is drawn beside.
	//
	// The blow seam is that same world's affect, bound by that same line: the
	// queue a blow lands in is the queue an order lands in and the one the very
	// next advance drains, so a key press and a left click reach the world by
	// one path and in the order they were made.
	//
	// The picker map is dressed with its own art port: the font, card font,
	// attack pointer, command panel, bottom HUD, character panes and sack
	// sheets, in that order, and none of the words, dialog frame or pane filler
	// a mission viewer receives. The readout is off in the game; F1 brings it
	// back.
	pickerArtSource{in: &f.InstallResources, pr: &f.Presentation}.resolve().apply(mv.Viewer)
	// A nil SoundBank is lawful: the viewer then plays nothing.
	audio := f.runtimeAudio()
	audio.attach(mv.Viewer)
	audioReady := false
	defer func() {
		if !audioReady {
			audio.release(mv.Viewer)
		}
	}()

	mw, err := openMapWorld(mv.Map, f.Table, f.Units, mv.Viewer)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("load %s: %w", e.Source, err)
	}
	audio.wireReplies(mw)
	mw.view.SetPathfinding(f.showPathfinding)
	mw.view.SetGraphicsOptions(f.graphics)
	mw.view.SetTextSmoothing(!f.smoothingOff.text)
	mw.view.SetFrameSmoothing(!f.smoothingOff.frame)
	if err := f.applyFreshGameOptions(mw, nil); err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	mv.Viewer.SetGameMenuContext(func() ui.GameMenuContext {
		return gameMenuContext(mw, false, mv.Map.Description)
	})
	audio.wireEffects(mw, mv.Viewer)
	// The autocast toggle rides a viewer method and not the tuple.
	mv.Viewer.SetAutocastSink(mw.setAutocast)
	mv.Viewer.SetQuickSpells(&f.quickSpells)
	// Ctrl+F is a setting key whose far side appends to this world's queue.
	mv.Viewer.SetFormationSink(mw.cycleFormation)
	mv.Viewer.SetDefendSink(mw.defend)
	mw.restoreStructureUseMetadata(f.Table)
	mv.Viewer.SetStructureUseSink(mw.useStructure)
	// Ctrl+W's sibling setting also rides a Viewer-installed seam. FrontEnd
	// owns the persisted label; mapWorld owns the pending player command.
	mv.Viewer.SetRetreatSink(func() { f.cycleRetreat(mw) })
	mv.Viewer.SetPlayerRetreatSink(mw.playerRetreat)
	f.liveDriver(mw, 0, nil)
	// mw.grab RIDES HERE TOO, exactly as mw.attackOrCast beside it does: a
	// plain map opened from the picker names no party, so mw.grab's own guard
	// on invSubjectSet is what keeps the key inert here — the same "every
	// plain map yields no character" rule partyCharacters and partyArt already
	// state, applied to the one new key. mw.stance and mw.march RIDE HERE TOO,
	// exactly as mw.grab does above: the loader hands back every seam the map
	// screen may reach into this world by, and the whole set lives and dies
	// with the viewer.
	pace := mw.paced
	if f.runtime.deterministicFrames {
		pace = mw.deterministicFrame
	}
	mv.Viewer.SetEntities(mw.entityDraws())
	audioReady = true
	return mv.Viewer, pace, mw.enqueue, mw.setCadenceMode, mw.affect, nil, mw.attackOrCast, mw.grab,
		mw.stance, mw.march, nil
}

// mapBytes reads one listed map from whichever source it came from —
// through a filesystem either way, with no host read and no archive handle
// beside them.
//
// The row's own kind is what picks the address, and that is all FromArchive
// now says: an archive row is the scenario identity and the entry path the
// row listed under, a loose row is its bare file name — a separator-less
// address, which names no container identity and so resolves under the asset
// root exactly as every address that names none does.
//
// The loose read folds its name, which is DirMaps' note in full: the row a scan
// listed in the case the host stores it under is read in the folded one (0027 C-3,
// R-2).
func (f *FrontEnd) mapBytes(e MapEntry) ([]byte, error) {
	if e.FromArchive {
		return f.Archives.Containers.ReadFile(scenarioPrefix + e.Source)
	}
	return f.Archives.Loose.ReadFile(e.Source)
}

// PartyClass, the class key the one party member a mission starts with carried,
// was RETIRED by 0085 and is not replaced by another constant.
//
// It was ours, it was one constant, and its own comment said the value was
// arbitrary and asserted nothing. It was not arbitrary in effect: the value it
// held names the roster's bare-handed human body, so the one figure the player
// controls arrived drawn as an unarmed man holding nothing — a picture that was
// a consequence of a constant rather than of anything about him.
//
// What replaced it is not a better number. A player's character does not
// carry a drawn class in the original at all: the shipped key is discarded
// on arrival and the class is produced from what the character visibly
// wears. MissionParty performs that production, `pkg/data`'s appearance law
// is the production itself, and the one step of it this tree could not take
// — which name a character wearing a given item takes — is no longer
// authored either: the first equipment slot and the shipped body list supply
// it, and MissionParty derives it there rather than choosing one.

// MissionOpenerWith is the door: it turns a campaign mission NUMBER and a
// PARTY into a running map screen, the tick that advances the mission under
// it, and every seam that reaches it — including the advance a notice is
// dismissed through. MissionOpener, below, is this with the front end's own
// default party; 0119-chargen's own door (cmd/againrom) calls this directly
// with a generated one.
//
// THE PARTY IS AN ARGUMENT AND NOT SOMETHING BUILT HERE, for the reason the
// weapon is one to MissionParty: this is now the SEAM a generated character
// reaches the mission through, and a function that resolved its own party
// could only ever resolve the front end's default one. NOTHING ELSE ABOUT
// THIS CLOSURE MOVED to make room for that — the font, the readout, the
// attack pointer, the sack sheet and the start view are exactly where and
// as they were.
//
// IT IS loadMap's TWIN AND NOT A SECOND LOAD PATH. It reads the entry, goes
// through the same LoadMapViewer, hands the viewer the same font and the same
// bundles, and binds the same seams to one driver — the differences are that the
// world comes from a mission START rather than a plain load, and that the driver
// it binds also answers for the mission's words and its ending.
//
// THE MAP IS DECODED ONCE. LoadMapViewer decodes it on the way to the terrain
// and StartMissionFrom takes that same decoded map, so the world and the terrain
// on screen describe one map by construction.
//
// THE THREE STARTUP FAILURES ARE THREE DIFFERENT MESSAGES. A number that
// names no mission is refused before anything is opened and names the
// number, because there is no address for it to name; an entry that is
// absent and an entry that will not decode each name the address they failed
// at, and the three messages differ from each other.
func (f *FrontEnd) MissionOpenerWith(n int, party []mapload.PartyMember) ui.MapOpener {
	return f.missionOpener(n, party, nil, nil, nil)
}

// missionOpener is MissionOpenerWith's body, with one extra argument: the
// snapshot a LOAD is resuming, or nil for a mission being started (0143 plan
// D-2).
//
// RESUMING RE-RUNS THE WHOLE OPENER AND THEN SUBSTITUTES THE WORLD. Everything
// below — the decode, the viewer, the font, the pointer, the sack sheet, the
// audio, the start view, openMission and the swing sound — is what a mission
// screen IS, and two of those statements carry their own comment naming them as
// call sites where a forgotten line leaves every automated test green and the
// game silently broken. A second construction path from a saved world would be
// a second chance to forget one of them.
//
// prepare IS THE ORIGINAL GAME'S SAVE REACHING THE SAME DOOR, and it is a second
// argument rather than a second opener for exactly the reason above. It runs
// between the decode and StartMissionFrom, which is the only window in which the
// map's own unit records can still be moved: after the start they have become
// world entities and the world is what a resume of OUR format substitutes
// instead. The two resumes therefore enter at two different points of one
// construction and share every statement after them.
//
// IT RETURNS THE SAVE'S EXPLORED PLANE, which is the one thing that arm
// reads out of the save and CANNOT write from where it runs: the fog plane
// is built inside openMission, several statements below, so a prepare that
// wrote it would be writing a plane that does not exist yet. It returns what
// it decoded — against the map's own dimensions, which only it has — and
// this function applies it beside the residue, after the constructor's
// tick-0 push.
func (f *FrontEnd) missionOpener(n int, party []mapload.PartyMember, snap *Snapshot,
	initialPurse *uint32, prepare func(*alm.Map) (*originalFog, error)) ui.MapOpener {
	return f.missionOpenerMode(n, party, snap, initialPurse, prepare, nil, nil, f.Difficulty, f.Town)
}

// missionOpenerMode is missionOpener with one transactional-load hook. When
// activate is nil, it is the ordinary mission door and records the live driver
// before returning. When activate is non-nil, it constructs the complete map
// and returns the one state-changing assignment through that pointer. The load
// path can therefore finish every fallible open step while the current game is
// still live, then run the assignment only after the candidate is accepted.
func (f *FrontEnd) missionOpenerMode(n int, party []mapload.PartyMember, snap *Snapshot,
	initialPurse *uint32, prepare func(*alm.Map) (*originalFog, error), activate *func(),
	candidateUnits *terrain.UnitSet, difficulty mapload.Difficulty, town *Town) ui.MapOpener {
	// The opener owns the roster at creation time. A later town/selection/save
	// write therefore cannot change the members this door will construct, and
	// legacy callers that did not yet carry stable IDs gain them here.
	party = mapload.OwnParty(party)
	units := f.Units
	if candidateUnits != nil {
		units = candidateUnits
	}
	req := missionRequest{n: n, party: party, snap: snap, initialPurse: initialPurse, prepare: prepare,
		activate: activate, units: units, difficulty: difficulty, town: town}
	return func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
		p, err := enterMission(req, f.missionPorts())
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		return p.viewer, p.tick, p.order, p.cadence, p.affect, p.advance, p.attack, p.grab, p.stance, p.march, nil
	}
}

func (in *InstallResources) canonicalizeJoinedRoster(ms *Mission) {
	if in == nil || ms == nil {
		return
	}
	for id, p := range ms.Start.Roster {
		nameIndex := p.CompanionNPC - 1
		if p.CompanionNPC >= 21 && p.CompanionNPC <= 24 {
			selector := -1
			for i, base := range data.ChargenBaseNames() {
				if p.Name == base || strings.HasPrefix(p.Name, base+"_") {
					selector = i
					break
				}
			}
			if selector < 0 {
				selector = 0
				if data.FigureDir(p.FigureDir).Female() {
					selector++
				}
				if p.Mage {
					selector += 2
				}
			}
			nameIndex = 20 + selector
		}
		if p.CompanionNPC != 0 {
			p.Name = in.localizedNPCName(nameIndex, p.Name)
		} else if name, ok := in.modCharacterName(p.Name); ok {
			// A mod named this definition row.
			p.Name = name
		} else if index := int(p.Class); index >= 0 && index < len(in.Words.UnitNames) && in.Words.UnitNames[index] != "" {
			// Ordinary Humans carry a table key such as M_Brigand3 here.
			// Resolve their type before equipment replaces the body class.
			p.Name = in.Words.UnitNames[index]
		}
		// Only a Hero placement derives its class from equipment (ANIM-106).
		hero := true
		if ms.World != nil {
			if e, held := ms.World.Entity(id); held {
				hero = data.FigureIsHero(e.TypeID)
			}
		}
		if hero {
			body, dir, class, matched := data.HeroAppearance(in.Bodies, mapload.EquipmentFromParty(p), p.Mage, false)
			if matched {
				p.Body, p.BodyDir, p.Class = string(body), dir, class
			}
		}
		ms.Start.Roster[id] = p
	}
}

// continuity is continueMission over this front end.
func (f *FrontEnd) continuity(n int, ms *Mission, advance ui.MapAdvance) ui.MapAdvance {
	return advanceFrom(frontTransitions{f})(n, ms, advance)
}

// MissionOpener is MissionOpenerWith over this front end's own default party
// — MissionParty's own answer, resolved off f.StartWeapon, f.Bodies and
// f.Humans exactly as CheckLine's sheet and MissionLine's report both read
// it. Every caller that opens a mission WITHOUT running the generation
// screen — cmd/againrom absent -chargen, cmd/missionrun, every test that
// has no result to hand over — still goes through this name, unchanged.
func (f *FrontEnd) MissionOpener(n int) ui.MapOpener {
	return f.MissionOpenerWith(n, f.NextParty())
}

// NextParty is the party the next mission this front end opens by number
// starts with: THE ONE CARRIED OUT OF THE LAST WON MISSION where there is one,
// and MissionParty's own default hero where there is not (the continuity
// hotfix).
//
// IT IS THE WHOLE OF THE RPG HALF AT THE FRONT-END TIER. Every mission THIS
// FRONT END opens by number — the picker's mission rows through loadMap, and
// cmd/againrom's -mission — comes through MissionOpener above and therefore
// through this, so winning one mission and opening another keeps the character
// instead of minting a new one. There is no second door to teach and no path
// where half the rule applies.
//
// cmd/missionrun IS NOT ONE OF THEM and deliberately so: it builds no front
// end, calls StartMission with MissionParty's own default directly, and drives
// exactly one mission per process. There is nothing for it to carry from and
// nowhere for it to carry to, which is also why the milestone drive cannot
// move for this change.
//
// IT IS A METHOD AND NOT A BRANCH INSIDE THE CLOSURE, because the party has to
// be resolved WHEN THE OPENER IS BUILT and read again when it runs would be
// worse: an opener built before a win and run after it would silently change
// which party it opens. The picker builds an opener and runs it in one
// statement, so the two are the same instant today; a method is what keeps the
// answer from depending on that staying true.
//
// A GENERATED CHARACTER STILL REACHES A MISSION THROUGH MissionOpenerWith and
// never through here, exactly as loadMap's own comment describes — but once
// his mission is WON, what he earned lands in Carried like anyone's, and the
// next mission he opens by number is his own character carried forward rather
// than the default hero. That is the one behaviour of loadMap's disclosure
// that this hotfix changes, and it changes it in the direction the disclosure
// was apologising for.
func (f *FrontEnd) NextParty() []mapload.PartyMember {
	return f.nextParty(func() []mapload.PartyMember {
		return MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table)
	})
}

// FinishMission is what winning mission n does to this front end: the party
// that fought it becomes the party the next one starts with, and the return
// says what follows — a positive mission this front end is about to open,
// or zero beside the sentence the map list will state.
//
// IT IS A METHOD AND NOT A CLOSURE BODY so that it can be decided against a
// hand-built world with no install anywhere near it.
//
// THE PARTY IS CAPTURED BEFORE ANYTHING ELSE IS DECIDED, and unconditionally
// — a campaign this tree could not read still carries the character
// forward, and a declared successor this tree must refuse still carries it
// too. Every branch below shares this one statement; none needs the registry
// to carry the party, and only some of them need it to say more.
//
// Offered IS WHAT THE MAP LIST WAS TOLD, so a path that reaches no map list
// clears it. A positive return advances directly and shows no map list at
// all: the successor is not offered, it is opened, and the field goes back
// to the nothing it holds before the first win rather than keeping the
// previous win's answer, which would be a stale row nobody was ever shown.
func (in *InstallResources) localizedNPCName(index int, fallback string) string {
	if in == nil || in.Archives == nil || in.Archives.Containers == nil || index < 0 {
		return fallback
	}
	payload, err := in.Archives.Containers.ReadFile(mainPrefix + "text/npcnames.txt")
	if err != nil {
		return fallback
	}
	lines := strings.Split(strings.ReplaceAll(string(payload), "\r\n", "\n"), "\n")
	if index >= len(lines) || lines[index] == "" {
		return fallback
	}
	return lines[index]
}

func (f *FrontEnd) FinishMission(n int, party []mapload.PartyMember,
	w *sim.World, ids []sim.EntityID) (int, string) {

	return f.FinishMissionWithRoster(n, party, w, ids, nil)
}

// FinishMissionWithRoster is FinishMission with the mission's own person
// roster: Start.Roster, the templates the map supplied at load. It is what
// lets an actor a script handed to the player enter the party the mission
// ends with. FinishMission is this with no roster, which carries exactly the
// members that walked in and is what every caller holding no start gets.
// liveMercenaries counts, per mercenary type, the hired men of that type
// still standing when the mission ended. It is MERC-DEATH-006's tally.
//
// The party and ids are parallel: ids[i] is the entity the mission minted for
// party[i]. A member with no entity, or one whose entity is not alive, is not
// counted -- which is the whole point of the tally, because for a hired type
// the pool becomes this number and the dead are what it loses.
func liveMercenaries(party []mapload.PartyMember, w *sim.World, ids []sim.EntityID) [16]int {
	var live [16]int
	if w == nil {
		return live
	}
	alive := make(map[sim.EntityID]bool, len(ids))
	for _, e := range w.Entities() {
		if e.Alive() {
			alive[e.ID] = true
		}
	}
	for i, p := range party {
		if i >= len(ids) || !p.Hired() {
			continue
		}
		typ := int(p.MercenaryType)
		if typ <= 0 || typ >= len(live) || !alive[ids[i]] {
			continue
		}
		live[typ]++
	}
	return live
}

// withoutDeparted drops every walked-in member a script gave to another
// player (the mission's departed set), dead or alive. PARTY-ENDCULL-026 keeps
// only the human Player's own actors at the boundary, and PARTY-ADDHERO-017
// makes a companion one of them only while it is on that list.
func withoutDeparted(party []mapload.PartyMember, ids []sim.EntityID, departed map[sim.EntityID]bool) ([]mapload.PartyMember, []sim.EntityID) {
	if len(departed) == 0 {
		return party, ids
	}
	var keptParty []mapload.PartyMember
	var keptIDs []sim.EntityID
	for i, p := range party {
		if i < len(ids) && departed[ids[i]] {
			continue
		}
		keptParty = append(keptParty, p)
		if i < len(ids) {
			keptIDs = append(keptIDs, ids[i])
		}
	}
	return keptParty, keptIDs
}

// withoutFallenBodies drops every player character whose body the mission-end
// cull left lying at health -10 or below: the client cull keeps a player
// character only above -10 (PARTY-CULL-004), and the edge carries only what
// both culls keep (PARTY-PERSIST-028). Hired men keep their own drop, and the
// starting hero stays (DIV-1488).
func withoutFallenBodies(party []mapload.PartyMember, ids []sim.EntityID, w *sim.World) ([]mapload.PartyMember, []sim.EntityID) {
	if w == nil {
		return party, ids
	}
	lying := make(map[sim.EntityID]bool)
	for _, e := range w.Entities() {
		if !e.Alive() && e.HP <= -10 {
			lying[e.ID] = true
		}
	}
	var keptParty []mapload.PartyMember
	var keptIDs []sim.EntityID
	for i, p := range party {
		if i < len(ids) && lying[ids[i]] && p.PlayerCharacter && p.MercenaryType == 0 && !p.StartingHero {
			continue
		}
		keptParty = append(keptParty, p)
		if i < len(ids) {
			keptIDs = append(keptIDs, ids[i])
		}
	}
	return keptParty, keptIDs
}

func (f *FrontEnd) FinishMissionWithRoster(n int, party []mapload.PartyMember,
	w *sim.World, ids []sim.EntityID, roster map[sim.EntityID]mapload.PartyMember) (next int, message string) {
	return f.finishMission(f.townInstall(), n, party, w, ids, roster,
		townArrival{arrive: f.arriveInTown, welcome: f.addChapterCompanions})
}

// AdvanceLine is the headless report of what winning mission n does, printed
// with no window: the successor n's own campaign section declares, or that
// it declares none — and where one is declared and can be opened, this
// OPENS it and reports its own address together with the started hero's
// health, mana and skill experience there.
//
// IT RUNS THE REAL DECISION AND NOT A GUESS AT IT. Where FinishMission
// answers a positive successor, this starts THAT mission too, with
// f.NextParty() — the party FinishMission's own write to f.Carried just
// fed, which is what MissionOpener would open the successor with on the
// windowed path (continuity's own doc, "the opener is built one statement
// after FinishMission returns").
//
// WHAT THIS DOES NOT EXERCISE IS THE RECOGNITION OF A WIN. continuity calls
// FinishMission only after asking ms.World.Outcome() == sim.OutcomeWon; this
// method calls it on a mission that was merely STARTED, never played,
// because nothing above pkg/sim can decide an outcome, and forcing one onto
// a world here would make the report evidence about the forcing rather than
// about the game (its own rejection). The unit tests are what witness the
// recognition itself, over a world that really was won; this method
// witnesses everything the recognition leads to.
//
// THE LINE SAYS WHAT WINNING WOULD OPEN AND NEVER THAT A MISSION WAS WON: its
// verb is "opens", a fact about the campaign's own declaration, and the hero
// it reports is the one the successor STARTS with, not one that fought
// anything there.
func (f *FrontEnd) AdvanceLine(n int) (string, error) {
	diff, err := campaignDifficulty(int64(f.Difficulty))
	if err != nil {
		return "", err
	}
	ms, err := StartMission(f.Archives.Containers, n, f.Table, diff, MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table))
	if err != nil {
		return "", err
	}
	// AdvanceLine bypasses MissionOpener, so install the same participant purse
	// that door installs before asking the real completion path what follows.
	// Without this handoff the headless owner-facing report would say 500 after
	// mission 20 while the windowed route says 600: it would have discarded the
	// fresh participant's 100 before applying the transition reward.
	ms.World.SetPurse(sim.SelfSlot, uint32(f.Town.Gold()))
	successor, completion := f.FinishMission(n, ms.Party, ms.World, ms.Start.IDs)
	if successor < 0 {
		return "", errors.New(completion)
	}
	if successor <= 0 {
		// WINNING INTO THE TOWN IS NOT WINNING INTO NOTHING. Until this story the
		// two were one answer, because there was no town; reporting them the same
		// way now would say the campaign stops exactly where it starts being a
		// town, which is the opposite of what happens. The line states the chapter
		// the town stands in and what each of its three buildings holds there,
		// which is the whole of what a headless run can witness about a screen.
		if f.Town.Open() {
			return f.townLine(n), nil
		}
		return fmt.Sprintf("againrom: winning mission %d opens nothing - mission %d declares no successor", n, n), nil
	}
	sms, err := StartMission(f.Archives.Containers, successor, f.Table, diff, f.NextParty())
	if err != nil {
		return "", err
	}
	line := fmt.Sprintf("againrom: winning mission %d opens mission %d at %s, %dx%d, %d entities",
		n, successor, sms.Address, sms.Map.Width, sms.Map.Height, len(sms.World.Entities()))
	if e, ok := heroEntity(sms); ok {
		line += fmt.Sprintf("; hero health %d/%d, mana %d/%d, skill xp %v", e.HP, e.MaxHP, e.Mana, e.MaxMana, e.SkillXP)
	}
	return line, nil
}

// MissionLine is the headless report for a mission the check mode was asked
// to start: what was built, and from where.
//
// IT REACHES ALL THREE STARTUP FAILURES AND OPENS NO WINDOW, which is the whole
// of "the check mode accepts -mission too and reports the same outcome": the
// number, the read and the decode are all settled before anything a window
// touches, so a headless run and a windowed one agree about whether a mission
// can be started.
//
// A script this build cannot read is REPORTED and is not a failure, exactly as a
// font that will not load is: the mission still starts, and a run that said
// nothing about it would bury the one case worth seeing.
//
// IT IS ALSO WHERE THE INVENTORY FIGURE'S UNREAD ADDRESSES ARE REPORTED
// (docs/0110-inventory T4: plan D-11). buildInventorySubject (inventory.go)
// writes to no stream and pkg/game prints nowhere on its own — this is the
// one surface a mission's own front end already carries to cmd/againrom's
// stdout, at the same "-check -mission N" run that already states the
// mission it belongs to, so this is where "printed once at mission open" is
// answered. openMission (world.go) builds the SAME subject off the SAME
// builder when a mission is actually opened and discards the list there,
// because that path writes to no stream of its own.
//
// THE WINDOWED PATH (MissionOpener) STAYS SILENT, and that is disclosed
// rather than hidden: its closure carries no output stream to report
// through, and widening it to add one reaches ui.MapOpener and
// cmd/againrom, both outside this task's files. A developer running
// `-check -mission N` sees an unread address; a player running `-mission N`
// alone does not.
func (f *FrontEnd) MissionLine(n int) (string, error) {
	diff, err := campaignDifficulty(int64(f.Difficulty))
	if err != nil {
		return "", err
	}
	ms, err := StartMission(f.Archives.Containers, n, f.Table, diff, MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table))
	if err != nil {
		return "", err
	}
	line := fmt.Sprintf("againrom: mission %d at %s, %dx%d, %d entities, party at (%d, %d), %d raise(s)",
		ms.Number, ms.Address, ms.Map.Width, ms.Map.Height, len(ms.World.Entities()),
		ms.Start.Drop.X, ms.Start.Drop.Y, len(ms.Raises))
	if ms.Start.Fallback {
		line += "; the map authorised no start cell, so the party stands on the drawn fallback"
	}
	if ms.RaiseErr != nil {
		line += fmt.Sprintf("; no announcements: %v", ms.RaiseErr)
	}
	if e, ok := heroEntity(ms); ok {
		line += fmt.Sprintf("; health %d/%d, mana %d/%d", e.HP, e.MaxHP, e.Mana, e.MaxMana)
	}
	// buildInventorySubject's own report (see this method's comment above):
	// every address it could not read for the mission's one party member, in
	// the order it tried them. An empty list adds nothing — a run that read
	// everything states so by saying nothing, the font's and the attack
	// pointer's own rule (CheckLine).
	if _, unread := buildInventorySubject(f.Archives.Containers, ms); len(unread) > 0 {
		line += fmt.Sprintf("; inventory art unread: %s", strings.Join(unread, ", "))
	}
	return line, nil
}

// heroEntity is the started hero's OWN entity — ms.Start.IDs[0]'s, matched by
// id against ms.World.Entities() the way heropicture_test.go's own fixture
// already does — or the zero Entity and false for a mission with no party at
// all, which is total rather than an index into a slice that might not carry
// one.
//
// IT IS A LOOKUP AND NOT A RECOMPUTE. The health and mana pairs MissionLine
// reports are what mapload.StartMission already wrote onto the entity at
// mission start; reading them back off the entity is what proves they
// reached it, where computing them again here would only prove this function
// agrees with itself.
func heroEntity(ms *Mission) (sim.Entity, bool) {
	if ms == nil || len(ms.Start.IDs) == 0 {
		return sim.Entity{}, false
	}
	id := ms.Start.IDs[0]
	for _, e := range ms.World.Entities() {
		if e.ID == id {
			return e, true
		}
	}
	return sim.Entity{}, false
}

// startViewCell is the cell a mission's view opens centred on: the first party
// member's, or — where the start placed nobody — the cell the start decided on.
//
// IT READS THE START'S REPORT AND RE-DERIVES NOTHING, and that is the whole
// reason it is this and not a look at the map. Two published facts make the
// difference load-bearing rather than stylistic. The engine picks its drop
// cell with a UNIFORM RANDOM INDEX over the map's array rather than taking
// the first (MISSION-DROP-002), so re-deriving means reproducing a draw. And
// a campaign map may itself place units for the player — 5 of the 28 do
// (MISSION-START-001) — which the placement walk then moves to the drop,
// so a position taken off the drop table is right on 23 maps and wrong on 5.
// What the start reports is right on all of them.
//
// THE ANCHOR IS THE FIRST MEMBER AND NOT THE PARTY'S EXTENT. The engine places
// the hero at the drop cell exactly, with radius 0, and crowds everyone else
// around him (MISSION-START-001), so the first member's cell is the spawn in the
// sense the requirement means — and nothing published says a view should be
// centred on a party's bounding box instead, so the simpler reading is taken and
// written down rather than left for the code to imply.
//
// A PARTY OF NONE STILL OPENS SOMEWHERE, and Drop is the cell it opens on: the
// start decides where the party would have stood whether or not anyone stands
// there, so this is a value the load produced and not a second guess at one. The
// three candidates were this, the map's centre and the world origin; the origin
// is the defect this story exists to remove, and the map's centre would answer a
// question about the map where every other arm answers one about the start.
func startViewCell(st mapload.Start) image.Point {
	c := st.Drop
	if len(st.Cells) > 0 {
		c = st.Cells[0]
	}
	return image.Pt(int(c.X), int(c.Y))
}
