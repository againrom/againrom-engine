// Package archtest enforces the againrom dependency DAG as an executable
// check rather than a review convention. It is a non-tier build/test helper
// under internal/ and is not itself policed by the DAG rules.
//
// The check is split into a pure evaluator (Check / CheckSimTests), which is
// exercised by a table of synthetic import graphs including the exact
// determinism-wall violation, and a loader (Load), which reads the live module
// tree. Import classification is by path so it cannot go hollow: an import is
// intra-module iff it equals the module path or has the module-path prefix (the
// module "againrom" has no dot, so a naive "no dot means stdlib" rule is wrong);
// otherwise it is stdlib iff its first path segment has no dot, else external.
package archtest

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ModulePath is the Go module path of this repository.
const ModulePath = "againrom"

// ebitenModule is the windowing/rendering engine, permitted only in the UI and
// cmd tiers (see externalAllowed).
const ebitenModule = "github.com/hajimehoshi/ebiten/v2"

// starlarkModule is the mod script interpreter, permitted only in pkg/modrt.
const starlarkModule = "go.starlark.net"

// allow maps a module-relative package path to the set of intra-module packages
// it may import. An entry ending in "/" is a wildcard matching any package under
// that prefix (e.g. "pkg/" matches every pkg/* package). Every package under the
// module — except the internal/ helpers — must appear here; an unlisted package
// is a fail-closed violation, so packages added later cannot silently escape the
// DAG.
var allow = map[string][]string{
	"pkg/formats/res":     {},
	"pkg/formats/reg":     {},
	"pkg/formats/alm":     {},
	"pkg/formats/spr256":  {"pkg/formats/pal"},
	"pkg/formats/spr16":   {"pkg/formats/pal"},
	"pkg/formats/databin": {},
	"pkg/formats/pal":     {},
	// The bitmap and sprite decoders read their colour entries through the
	// palette leaf, the one owner of the blue-green-red entry layout.
	"pkg/formats/bmp": {"pkg/formats/pal"},
	// The WAV leaf owns the RIFF chunk walk; it imports nothing of this tree.
	"pkg/formats/wav":  {},
	"pkg/formats/fame": {},
	// The icon leaf reads the icon resources of the original executable, for
	// the window's icon. It walks a PE resource directory and decodes icon
	// bitmaps, imports nothing of this tree and converts no text.
	"pkg/formats/winicon": {},
	// textinput is a formats leaf because CP1251 conversion is a byte-format
	// concern. It may use the formats tier's narrowly-scoped x/text grant but
	// cannot reach UI or game state.
	"pkg/formats/textinput": {},
	// The item-name leaf pairs itemname.bin's key array with itemname.txt's
	// line array and applies no code-page conversion of its own (0151 T12,
	// ITEM-DISPNAME-036): a line is stored as the bytes the file ships, for
	// pkg/render/text's own converter to read at draw time. It is added to
	// noExternalFormats below on that ground, and it depends on nothing else
	// in this tree — both files are read by a caller and handed in whole.
	"pkg/formats/itemname": {},
	// The save leaf reads the ORIGINAL GAME's save file. Its set was empty
	// until 0150 and the reasoning that emptied it was about the MAP TIER: the
	// block-plane records are a delta over a map's own terrain, and a leaf that
	// could import the map tier would be one refactor from resolving that delta
	// itself instead of reporting it. That reasoning is untouched and
	// pkg/mapload is still denied here.
	"pkg/formats/sav": {"pkg/formats/reg"},
	"pkg/vfs":         {"pkg/formats/res"},
	"pkg/data":        {"pkg/vfs", "pkg/formats/reg", "pkg/rules"},
	"pkg/sim":         {"pkg/rules", "pkg/random"},
	"pkg/rules":       {},
	// The random service is a leaf: its generators and named streams import
	// nothing of this tree, so every tier that draws can name it.
	"pkg/random":  {},
	"pkg/mapload": {"pkg/formats/alm", "pkg/data", "pkg/sim", "pkg/mod", "pkg/base", "pkg/rules", "pkg/random"},
	"pkg/mapedit": {"pkg/formats/alm"},
	"pkg/render":  {"pkg/sim", "pkg/vfs"},
	// The terrain and menu tiers decode their bitmaps through the bitmap
	// leaf and keep no decoder of their own.
	"pkg/render/terrain":    {"pkg/formats/bmp"},
	"pkg/render/refraction": {},
	"pkg/render/backdrop":   {},
	"pkg/render/camera":     {},
	"pkg/render/frame":      {},
	"pkg/render/menu":       {"pkg/formats/bmp"},
	// The text tier is a LEAF: it holds the font model, the placement rule and
	// the blit as plain data plus arithmetic, and a loader outside it fills the
	// data. So it gains the graph a node and no outgoing edge, and this empty
	// set is what makes that mechanical rather than a promise.
	"pkg/render/text":      {},
	"pkg/render/debugtext": {"pkg/render/text"},
	// The smoothed-text overlay leaf (DIV-1385) resamples and recomposites
	// the glyphs pkg/render/text's own capture window records, so it names
	// that one tier for *text.Glyph and text.DrawCall and gains no other
	// outgoing edge: it knows nothing about a font file, a UI widget or a
	// window, only the pixels a Draw call already placed.
	"pkg/render/textsmooth": {"pkg/render/text"},
	// The final-frame scaler leaf holds the Catmull-Rom kernel, its Kage
	// source and a reference resampler as plain data plus arithmetic.
	"pkg/render/catmullrom": {},
	// The audio tier holds resampling, positional gain and mixing as plain
	// data plus arithmetic — nothing here knows a slot, an archive or a
	// listener exists. Its one edge is the WAV format leaf, which owns the
	// chunk walk, so no other tier's type crosses in.
	"pkg/audio": {"pkg/formats/wav"},
	// The town composer builds a town view from a description. It holds no
	// game fact and imports only the standard library; a game supplies its
	// descriptions, art and hooks.
	"pkg/town": {},
	// Video is a presentation/transport leaf; it cannot import game or sim.
	// It reads a movie's sidecar registry through the reg format leaf.
	"pkg/video": {"pkg/video/smacker", "pkg/formats/reg"},
	// The Smacker decoder is a third-party-derived codec leaf: pure
	// bitstream/Huffman/DPCM decode, no knowledge of a player, a stream
	// protocol or a game. It cannot import pkg/video, which is the eventual
	// caller (an import the other way would be the cycle), so like pkg/audio
	// and pkg/render/text it gains the graph a node and no outgoing edge.
	"pkg/video/smacker": {},
	// The UI tier may use the whole render tier: pkg/render and everything under
	// it (terrain, camera). The trailing slash is the prefix form. It also
	// gains the audio leaf (0126 plan T2): sound.go's SetAudio takes an
	// audio.Player and sounddev.go builds the one concrete audio.Player this
	// tree ships, and pkg/audio's own empty allow-set (above) is what keeps
	// this a one-directional grant rather than the audio leaf learning what
	// a viewer, a camera or an entity is.
	"pkg/ui":             {"pkg/render", "pkg/render/", "pkg/audio", "pkg/video", "pkg/random"},
	"cmd/cutscenehelper": {"pkg/video"},
	"cmd/audioprobe":     {"pkg/video"},
	"pkg/game":           {"pkg/"},
	"cmd/againrom":       {"pkg/", "internal/buildinfo"},
	"cmd/restool":        {"pkg/formats/res", "pkg/vfs"},
	"cmd/regtool":        {"pkg/formats/reg", "pkg/formats/res"},
	// classdump's -databin verb reaches the definition table through the vfs
	// tier and resolves a map's placements through the map-loading tier, so its
	// row grows by those two and by the table's own parser.
	"cmd/classdump": {"pkg/data", "pkg/formats/reg", "pkg/formats/res", "pkg/formats/alm",
		"pkg/formats/databin", "pkg/vfs", "pkg/mapload"},
	// almtool's pass verb censuses the passability plane a map describes, and
	// it counts through the derivation's own classifier rather than a copy of
	// it, so the tool reaches the tier that owns that classifier.
	"cmd/almtool": {"pkg/formats/alm", "pkg/mapload"},
	"cmd/sprtool": {"pkg/formats/res", "pkg/formats/spr256", "pkg/formats/spr16"},
	// savtool reads and edits an original save and names the format leaf alone:
	// every verb it has is a question about one file.
	"cmd/savtool": {"pkg/formats/sav"},
	// missionrun starts a mission through the front-end's own loading path and
	// then drives it: pkg/game opens the install and starts the mission,
	// pkg/mapload answers which entity a script identifier or a party slot is
	// and what the map's passability plane says, and pkg/sim is what a command
	// and an outcome are named in. pkg/data JOINS THE ROW at 0132 T1: the
	// attack report reads a worn or carried code's seven-digit name and the
	// slot its class names through data.ItemCode's own accessors, and the
	// weapon name a loaded table resolves it to through the existing
	// code-to-weapon recovery, data.WeaponFromCode — the same one
	// cmd/wearcheck and cmd/paneldump already call, not a second reader of the
	// tables invented here.
	"cmd/missionrun": {"pkg/game", "pkg/mapload", "pkg/sim", "pkg/data"},
	// scriptcoverage starts every campaign mission through pkg/game, retains
	// the authored node identities from formats/alm, reads the builder's joins
	// from pkg/mapload and observes the resulting world through pkg/sim. Those
	// four tiers are the evidence chain; the tool opens no lower archive leaf.
	"cmd/scriptcoverage": {"pkg/formats/alm", "pkg/game", "pkg/mapload", "pkg/sim"},
	// savecheck drives 0143's save/load loop against a real install in two
	// processes. It names the wiring tier for the front end and the store,
	// and pkg/sim for the world it reads a tick and a hash off — the same
	// two missionrun above it already names, and no format or vfs import:
	// the container filesystem reaches it as a return value.
	"cmd/savecheck": {"pkg/game", "pkg/sim"},
	// scenariofixture prepares explicit native UI-regression input through
	// the frontend and changes only a detached world's controlled endpoint.
	"cmd/scenariofixture": {"pkg/game", "pkg/sim"},
	// presenceprobe censuses the five map-presence instant opcodes (0164)
	// against a real install: how many nodes each shipped mission's own script
	// compiles, and how many of their unit and group references resolve
	// against the world the loader builds. It names the wiring tier for the
	// mission, pkg/mapload for the difficulty and pkg/sim for the opcodes and
	// the compiled record — the same tiers missionrun above it already names.
	"cmd/presenceprobe": {"pkg/game", "pkg/mapload", "pkg/sim"},
	// texttool measures and renders a font. It names only the loader's tier and
	// the drawing tier: the container filesystem reaches it as a return value,
	// so the tool never has to name the format or vfs tiers to open an install.
	"cmd/texttool": {"pkg/game", "pkg/render/text"},
	// dlgtool censuses the event-text corpus against an install. It names the
	// loader's tier alone, for texttool's own reason: the container filesystem
	// reaches it as a return value, so listing and reading an archive entry
	// costs it no import of the vfs or format tiers.
	"cmd/dlgtool":     {"pkg/game"},
	"cmd/terraintool": {"pkg/render/terrain", "pkg/formats/res", "pkg/formats/alm", "pkg/game", "pkg/base"},
	// The read-only editor receives authored content from the game loader and
	// drives only UI inspection and camera input, not simulation or file formats.
	"cmd/mapedit": {"pkg/game", "pkg/ui"},
	// mapview's -blocked tint draws the plane a PLACED STRUCTURE reaches, so
	// the tool reaches the tier that derives one and the two tiers it takes a
	// definition table out of an archive with. pkg/data is not on the row: a
	// parsed collection satisfies that tier's interface implicitly, so the
	// table crosses as behaviour rather than as a named type.
	"cmd/mapview": {"pkg/ui", "pkg/render/terrain", "pkg/formats/res", "pkg/formats/alm", "pkg/game",
		"pkg/mapload", "pkg/formats/databin", "pkg/vfs"},
	// paneldump composes the unit panel against a lawful install and measures
	// the box, so it reaches the tier that draws one and the tier that starts a
	// mission, plus the two types those two hand across: the unit registry a
	// name is read out of and the font the box is measured with. pkg/sim is not
	// on the row — the entity fields it copies arrive through pkg/game's own
	// started world, so the simulation crosses as values rather than as a named
	// type. pkg/data JOINS THE ROW at 0128 T4: filling the panel's own worn row
	// turns a worn code into a row name through data.ItemCode's D() accessor
	// and the collections defs.Table already carries, which is a lookup this
	// tool can do and the running window cannot.
	"cmd/paneldump": {"pkg/ui", "pkg/game", "pkg/mapload", "pkg/render/terrain", "pkg/render/text", "pkg/data"},
	// shopdump is 0157 round 3's own instrument, and it reaches the same two
	// tiers cmd/paneldump does for the same reason: pkg/game opens the install
	// and runs the campaign session that opens a town, and pkg/ui composes the
	// screen whose pixels are the evidence. It names no simulation type.
	"cmd/shopdump": {"pkg/ui", "pkg/game"},
	// screenshot composes named screens through the single seam pkg/ui/app.go's
	// Draw itself now also goes through (composeScreen, HeadlessFrame,
	// ComposeTownScreen), reaching them the same two ways cmd/shopdump and
	// cmd/plaquescreens already do: pkg/game opens the install, drives a live
	// ui.App for the menu/chargen/mission screens, and finishes a campaign
	// mission with no App at all for the town screens. It names no simulation
	// type. pkg/data is named for ONE predicate: which carried item code opens
	// the campaign documents panel (data.RaisesDocuments). The drive has to
	// find that item in the pack bar to reach the screen through the production
	// double-click rather than through a seam of its own.
	"cmd/screenshot":   {"pkg/ui", "pkg/game", "pkg/data"},
	"cmd/screencensus": {"pkg/ui", "internal/gatedtests"},
	// tooltipshot is TEXT-HOVERTEXT-052's own instrument: it opens an install
	// through pkg/game and paints the map-list and monster-spell hints through
	// pkg/ui's own hint wrappers and paint seam, the same two tiers
	// cmd/screenshot already names. pkg/mapload is named for one value alone,
	// mapload.DifficultyNormal, the difficulty game.StartMission's own
	// production signature requires; it decodes nothing here itself.
	// pkg/render/text is named for *text.Font, the type game.FrontEnd.Font
	// already returns and ui.ComposeTooltipHint already takes.
	"cmd/tooltipshot": {"pkg/ui", "pkg/game", "pkg/mapload", "pkg/render/text"},
	// worldmapcheck opens an install and drives the production town/world-map
	// adapter without a window. All archive, campaign and UI values cross
	// through pkg/game, so the command names that tier alone.
	"cmd/worldmapcheck": {"pkg/game"},
	// campaigncensus prints the second game's campaign support census. Every
	// install read, map decode and world run crosses through pkg/game.
	"cmd/campaigncensus": {"pkg/game"},
	// mapunitcensus is 1029's own instrument. It opens an install through
	// pkg/game (OpenArchives, MissionMap) and decodes each campaign map with
	// pkg/formats/alm to read the one field the story added to the entity
	// record, alm.Unit.UnitID. It names the format tier directly because the
	// question it answers is about the MAP FILE rather than about a loaded
	// world: pkg/mapload would answer it only after the loader had already
	// carried the value, which is the thing under measurement.
	"cmd/mapunitcensus": {"pkg/game", "pkg/formats/alm"},
	// rotationcensus is 1047 round 2's own instrument. It opens each campaign
	// mission through game.FrontEnd.MissionOpenerWith -- the production
	// mission door -- and reads sim.Entity.RotationSpeed off the live world
	// game.FrontEnd.LiveWorld() returns. It names pkg/game alone: the live
	// world it reads is a *sim.World value returned by pkg/game's own API,
	// not a type this command imports pkg/sim to name.
	"cmd/rotationcensus": {"pkg/game"},
	// menuaccelcheck is 1014's own instrument (MENU-KEY-013): it opens an
	// install through pkg/game, drives the production in-game menu and its
	// HeadlessType dispatch through pkg/ui exactly a keyboard session would,
	// and names pkg/formats/res only for its labels verb's diagnostic
	// CP866-to-string decode (res.DecodeCP866) — no decode this tool performs
	// feeds a decision the game itself makes.
	"cmd/menuaccelcheck": {"pkg/game", "pkg/ui", "pkg/formats/res"},
	// schoolcheck is 1015's own instrument. It opens an install through
	// pkg/game (OpenArchives, LoadTownSchoolArt) and asks pkg/ui for the
	// production school rectangles and the production hit test, so the
	// correlation it measures off the shipped art is compared against the
	// code the game runs rather than against a copy of it. It names
	// pkg/formats/bmp to decode the sixteen rotation frames the front end
	// does not load, and pkg/render/terrain only for the EntrySource type
	// pkg/game already hands it. It names no simulation type.
	"cmd/schoolcheck": {"pkg/game", "pkg/ui", "pkg/formats/bmp", "pkg/render/terrain"},
	// buttonframecheck is 1017's own instrument, on schoolcheck's own
	// pattern: it opens an install through pkg/game (OpenArchives) and asks
	// pkg/ui for the production button wells (TownSurfaceButtonWell,
	// ShopButtonRect), so the correlation it measures off the shipped art is
	// compared against the code the game runs. It names pkg/formats/bmp to
	// decode the area pictures and button bitmaps directly, and
	// pkg/render/terrain only for the EntrySource type pkg/game already
	// hands it. It names no simulation type.
	"cmd/buttonframecheck": {"pkg/game", "pkg/ui", "pkg/formats/bmp", "pkg/render/terrain"},
	// townsquarecheck is 1016's own instrument, on schoolcheck's own pattern:
	// it opens an install through pkg/game (OpenArchives, LoadTownSquareArt,
	// NewFrontEnd) and asks pkg/ui for the production town square constants
	// and the production hit test, so the correlation it measures off the
	// shipped art is compared against the code the game runs. It names
	// pkg/formats/bmp to decode the base picture, the overlay and the three
	// labels, and pkg/render/terrain for the mask decoder and the
	// EntrySource type pkg/game already hands it. It names no simulation
	// type.
	"cmd/townsquarecheck": {"pkg/game", "pkg/ui", "pkg/formats/bmp", "pkg/render/terrain"},
	// tippanelcheck is 1018 round 3's own instrument: it opens an install
	// through pkg/game (NewFrontEnd) and drives the production town square,
	// tavern, school, shop and chargen screens through pkg/ui's own exported
	// interfaces and hit tests (TipPanelFits, TownSquareControlAt,
	// TownSurfaceControlAt, ShopControlAt, PreCreateControlAt) to measure
	// each tip rect's own minimum fitting height and live-control coverage
	// against shipped art and text. It decodes nothing itself and names no
	// simulation type.
	"cmd/tippanelcheck":  {"pkg/game", "pkg/ui"},
	"cmd/divcensus":      {"internal/divledger"},
	"cmd/divreconstruct": {"internal/divledger"},
	"cmd/plaqueseams":    {"pkg/game", "pkg/formats/bmp", "pkg/vfs"},
	// plaquescreens is the same hotfix's before/after evidence tool, on
	// schoolcheck's own pattern: it opens an install through pkg/game
	// (NewFrontEnd, FinishMission, TownScreen) and composes the four
	// reported screens through pkg/ui's own production entry points
	// (ComposeTownSurface, ComposeChargenFrame, ComposeShopScreen), so what
	// it renders is what the game itself draws. It names no simulation
	// type.
	"cmd/plaquescreens": {"pkg/game", "pkg/ui"},
	"cmd/wearcheck":     {"pkg/data", "pkg/game", "pkg/mapload"},
	// spellcheck is 0161's own instrument: it opens an install through
	// game.OpenArchives and builds the projectile art bundle through
	// game.LoadProjectiles — both pkg/game — then reads a cast's two
	// picture ids and its flight length out of pkg/data and walks the frame the
	// selectors in pkg/render/terrain choose. The three tiers are exactly the
	// three the loaded bundle is made of, and it reaches pkg/render/terrain for
	// the selection functions and the sheet TYPE, never for the drawing.
	"cmd/spellcheck": {"pkg/data", "pkg/game", "pkg/render/terrain"},
	// spelleffectcheck is 1001's no-window integration instrument. It opens the
	// install and a real mission through pkg/game, walks the shipped spell rows
	// through pkg/data/pkg/mapload, compiles every campaign ALM to census script
	// cast producers, and asks pkg/sim for the controlled point/area/wall wiring
	// result. It imports no Client or VFS tier and writes no converted asset.
	"cmd/areaoverlaycheck": {"pkg/data", "pkg/game", "pkg/mapload", "pkg/render/terrain", "pkg/sim"},
	"cmd/spelleffectcheck": {"pkg/data", "pkg/formats/alm", "pkg/game", "pkg/mapload", "pkg/sim"},
	// weaponspellcheck is round-3's own weapon-spell instrument (docs/1001-
	// spell-effects/round3-weapon.md): it opens an install and a real mission
	// through pkg/game, resolves the world's own spell table through
	// pkg/mapload, and builds a SECOND, synthetic two-actor pkg/sim.World
	// directly — the lifted actor beside a dummy on open ground — so a
	// weapon-borne release is measured against the shipped row without the
	// mission's own terrain, script or AI group state in the way. It opens no
	// window and writes no converted asset.
	"cmd/weaponspellcheck": {"pkg/game", "pkg/mapload", "pkg/sim"},
	"cmd/appearcheck":      {"pkg/data", "pkg/game", "pkg/mapload", "pkg/render/terrain", "pkg/ui"},
	// effectmarkcheck is 1002's no-window integration instrument. It opens the
	// install and a real campaign mission through pkg/game, reads the shipped
	// projectile sheets and unit classes there, builds the effect-mark records
	// through pkg/render/terrain, and asks pkg/sim for the controlled cast
	// result the client's mark elements are opened from. It opens no window,
	// imports no pkg/ui, and writes no converted asset.
	"cmd/effectmarkcheck": {"pkg/game", "pkg/mapload", "pkg/render/terrain", "pkg/sim"},
	// knowledge/tools/claim is vendored, not authored here: it is the claim
	// reader the pinned knowledge submodule ships (pipeline/KNOWLEDGE-PUBLISH.md),
	// swept into this module's own tree because the submodule carries no go.mod
	// of its own (unlike a vendored dependency that does — see Load's
	// nested-module skip above) and is not this repository's code to design
	// architecture for. It is registered rather than exempted so an import this
	// project would object to still fails closed instead of never being
	// checked; its own source is stdlib-only, so the allow-set is empty.
	"knowledge/tools/claim": {},
	// The launcher edits an ini file and lists mod folders: pkg/ini and pkg/mod
	// are leaves, and pkg/game answers whether a folder is an install. It never
	// reaches the simulation.
	"pkg/ini": {},
	"pkg/mod": {"pkg/rules"},
	// The base profile leaf names the known game installs and detects one from
	// a directory listing and the main archive's digest. It imports nothing of
	// this tree; pkg/game and the commands read it.
	"pkg/base": {},
	// The mod runtime runs a mod's Starlark script and hands the simulation
	// nothing but a finished rules.Rules value. It is the only package that
	// may import the interpreter, and no package below pkg/sim may reach it.
	"pkg/modrt":   {"pkg/mod", "pkg/rules"},
	"cmd/starter": {"pkg/base", "pkg/game", "pkg/ini", "pkg/mod", "internal/buildinfo"},
}

// Violation names one rejected edge.
type Violation struct {
	From   string // module-relative importer package, e.g. "pkg/sim"
	Import string // offending import (module-relative for intra-module, else full path)
	Reason string
}

func (v Violation) String() string {
	if v.Import == "" {
		return v.From + ": " + v.Reason
	}
	return v.From + " -> " + v.Import + ": " + v.Reason
}

type kind int

const (
	stdlib kind = iota
	intra
	external
)

func classify(imp string) kind {
	if imp == ModulePath || strings.HasPrefix(imp, ModulePath+"/") {
		return intra
	}
	first := imp
	if i := strings.IndexByte(imp, '/'); i >= 0 {
		first = imp[:i]
	}
	if strings.Contains(first, ".") {
		return external
	}
	return stdlib
}

func intraAllowed(allowed []string, rel string) bool {
	for _, a := range allowed {
		if strings.HasSuffix(a, "/") {
			if strings.HasPrefix(rel, a) {
				return true
			}
			continue
		}
		if a == rel {
			return true
		}
	}
	return false
}

// noExternalFormats names the formats-tier packages held to the standard library
// alone, denied the tier-wide golang.org/x/text grant below. A format that
// converts no text takes no text-encoding dependency, and the check holds it to
// that rather than leaving it a documented intention (0011 spec AC-11).
//
// The deny is deliberately per-package: pkg/formats/spr256 is documented
// stdlib-only for the same reason and stays unenforced here, because widening
// this map is a change to another story's package.
var noExternalFormats = map[string]bool{
	"pkg/formats/reg":     true,
	"pkg/formats/spr16":   true,
	"pkg/formats/databin": true,
	// A colour table carries no strings at all — 256 entries of three read
	// bytes and a reserved one — so this format converts no text and takes no
	// text-encoding dependency.
	"pkg/formats/pal": true,
	"pkg/formats/wav": true,
	// The item-name leaf carries text but converts none of it (0151 T12): a
	// line is stored as the shipped bytes, and the code-page pass is
	// pkg/render/text's own, applied per drawn byte rather than once at
	// parse time. So this format takes no text-encoding dependency either.
	"pkg/formats/itemname": true,
	// An icon carries no text at all.
	"pkg/formats/winicon": true,
}

func externalAllowed(pkg, imp string) bool {
	// External dependencies are confined to their named tier.
	//
	// golang.org/x/text (CP866 decoding) belongs to the formats tier, minus the
	// packages denied it above.
	if strings.HasPrefix(pkg, "pkg/formats/") {
		if noExternalFormats[pkg] {
			return false
		}
		return imp == "golang.org/x/text" || strings.HasPrefix(imp, "golang.org/x/text/")
	}

	if pkg == "pkg/modrt" {
		return imp == starlarkModule || strings.HasPrefix(imp, starlarkModule+"/")
	}

	if pkg == "pkg/render/debugtext" {
		return imp == "github.com/hajimehoshi/bitmapfont/v4" ||
			imp == "golang.org/x/image/font" || imp == "golang.org/x/image/math/fixed"
	}

	// Ebitengine (windowing/rendering) belongs to the UI tier and the cmd tier
	// that launches it. Keeping it out of everything else is what stops the
	// engine leaking into the formats, sim, and render tiers, which must stay
	// headless and unit-testable without a window.
	// The launcher draws its own window into an image with the bundled bitmap
	// font, so it alone among the commands names the font packages.
	if pkg == "cmd/starter" {
		switch imp {
		case "github.com/hajimehoshi/bitmapfont/v4", "golang.org/x/image/font", "golang.org/x/image/math/fixed":
			return true
		}
	}

	if pkg == "pkg/ui" || strings.HasPrefix(pkg, "pkg/ui/") || strings.HasPrefix(pkg, "cmd/") {
		return imp == ebitenModule || strings.HasPrefix(imp, ebitenModule+"/")
	}
	return false
}

// Check validates a set of production imports against the allow-map. pkgImports
// maps each module-relative package path to the import paths written in its
// source. Results are sorted for deterministic output.
func Check(pkgImports map[string][]string) []Violation {
	var vs []Violation
	for pkg, imports := range pkgImports {
		allowed, registered := allow[pkg]
		if !registered {
			vs = append(vs, Violation{From: pkg, Reason: "package not registered in the architecture allow-map (fail-closed)"})
			continue
		}
		for _, imp := range imports {
			switch classify(imp) {
			case stdlib:
				// always permitted
			case external:
				if !externalAllowed(pkg, imp) {
					vs = append(vs, Violation{From: pkg, Import: imp, Reason: "external import not permitted for this tier"})
				}
			case intra:
				rel := strings.TrimPrefix(imp, ModulePath+"/")
				if rel == pkg {
					continue
				}
				if !intraAllowed(allowed, rel) {
					vs = append(vs, Violation{From: pkg, Import: rel, Reason: "import violates the dependency DAG"})
				}
			}
		}
	}
	sort.Slice(vs, func(i, j int) bool {
		if vs[i].From != vs[j].From {
			return vs[i].From < vs[j].From
		}
		return vs[i].Import < vs[j].Import
	})
	return vs
}

// CheckSimTests asserts that pkg/sim's test files import only the standard
// library, keeping the determinism package pure even in its tests.
func CheckSimTests(imports []string) []Violation {
	var vs []Violation
	for _, imp := range imports {
		if classify(imp) != stdlib {
			vs = append(vs, Violation{From: "pkg/sim (test)", Import: imp, Reason: "sim tests must import only the standard library"})
		}
	}
	sort.Slice(vs, func(i, j int) bool { return vs[i].Import < vs[j].Import })
	return vs
}

// Load walks the module tree rooted at moduleRoot and returns the production
// (non-test) imports of every policed package (every package except those under
// internal/), plus the imports of pkg/sim's test files.
func Load(moduleRoot string) (prod map[string][]string, simTestImports []string, err error) {
	prod = make(map[string][]string)
	err = filepath.Walk(moduleRoot, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}
		if info.IsDir() {
			name := info.Name()
			if path != moduleRoot && (name == ".git" || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			// A subdirectory with its own go.mod is a nested module (for example
			// a vendored dependency that ships one); its packages belong to a
			// different module and are not policed by this module's DAG.
			if path != moduleRoot {
				if _, statErr := os.Stat(filepath.Join(path, "go.mod")); statErr == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		relDir, rerr := filepath.Rel(moduleRoot, filepath.Dir(path))
		if rerr != nil {
			return rerr
		}
		pkg := filepath.ToSlash(relDir)
		if pkg == "." || strings.HasPrefix(pkg, "internal/") {
			return nil // module root holds no package; internal/ helpers are unpoliced
		}
		imports, ierr := fileImports(path)
		if ierr != nil {
			return ierr
		}
		isTest := strings.HasSuffix(path, "_test.go")
		if pkg == "pkg/sim" && isTest {
			simTestImports = append(simTestImports, imports...)
			return nil
		}
		if isTest {
			return nil // other tiers' test imports are intentionally not policed
		}
		prod[pkg] = append(prod[pkg], imports...)
		return nil
	})
	return prod, simTestImports, err
}

func fileImports(path string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, spec := range f.Imports {
		p, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// FindModuleRoot walks up from start until it finds the directory containing the
// go.mod for module ModulePath.
func FindModuleRoot(start string) (string, error) {
	dir := start
	for {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && strings.Contains(string(data), "module "+ModulePath) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
