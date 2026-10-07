// Command mapview opens a ROM1 map's terrain in a resizable window and lets you
// explore it with the mouse and keyboard (developer tool).
//
// Usage:
//
//	mapview -assets <dir> -map <file.alm> [-graphics <file.res>] [-databin <file.res>]
//	        [-speed 0..8] [-noanimation] [-objects] [-units] [-statics]
//	        [-staticmarkers] [-structures] [-ruins] [-objectanim] [-unshaded] [-flat] [-blocked] [-check]
//
// mapview drives the loading path pkg/game owns: opening graphics.res as a
// container filesystem, loading the terrain tile strips into the tileset by
// their addresses, decoding the .alm and handing the resulting tile grid to
// pkg/ui, which draws it through the pure camera model. That path lives in a
// library package because the game front-end uses it too, and the two must not
// offer different camera behaviour. What this command owns is its command line
// and its summary wording.
//
// Controls are the project's own design, not a reproduction of the original
// engine's camera: arrow keys or WASD pan, moving the cursor to a window edge
// edge-scrolls, the mouse wheel zooms about the cursor, and Esc closes the
// window.
//
// Water animation, by contrast, is the decoded game model: -noanimation
// matches the game's own switch — it draws water phase 0, not a frozen
// current phase, and it collapses every static object to sheet frame 0
// rather than its own Index — and -speed selects from the game's speed
// table, defaulting to the index map load pushes.
//
// -objectanim is a diagnostic of OURS and asserts nothing about the game:
// the decoded gate on an object's cycle is a test on four tile words that no
// shipped cell satisfies, so the object layer draws every shipped map
// exactly as it drew it before that story. The flag opens the cycle of every
// class with a non-zero period regardless of any tile word, which is what
// makes the arm visible and reviewable at all. It is off by default and
// reaches the object layer alone.
//
// -structures draws the map's PLACED STRUCTURES — its buildings. It is the
// map's own content and not a diagnostic, exactly as -statics is, and it is
// the one flag that is worth pairing with -objects: the type-4 marker
// overlay that flag draws puts a cross on each structure's anchor cell,
// derived from the record alone, and a building standing away from its own
// cross is how a wrong placement is caught. The two derivations are kept
// apart deliberately.
//
// -ruins is a diagnostic of OURS and asserts nothing about the game's state:
// it draws every drawable structure from its RUIN block, which is addressed
// from the end of its sheet, respecting each class's Indestructible.
// Destruction is not modelled anywhere in this tree — the arm's live input
// is a health value the engine fills from a create message and a
// definition-table row — so the flag is what makes a decoded block
// reviewable at all without inventing the state it would be reached through.
// It changes the picture and no count; it needs -structures, and the game
// front-end has no equivalent flag.
//
// -unshaded is a developer diagnostic: it renders at full palette brightness
// instead of the window's corner-interpolated relief shading. It moves no
// light of its own and the game front-end has no equivalent flag — the
// game is always displaced and lit, exactly as it is always displaced.
//
// -flat is the same kind of diagnostic, for geometry rather than shading: it
// selects flat terrain over an otherwise-valid, displaceable altitude grid
// instead of the height-displaced projection. The map stays lit exactly as
// it would displaced — flat mode has its own corner-interpolated shading
// (0014) — so this is the flag that makes flat-and-lit reachable at all;
// without it, flat only ever occurred for a map with no usable altitude
// data, which is also unlit. The game front-end has no equivalent flag; it
// is always displaced, mirroring -unshaded.
//
// -blocked is a developer diagnostic of OURS: it washes every cell the
// derived block plane closes to a GROUND mover, so a bridge reads as an open
// line across a closed river and a doorway as a gap in a wall. It is the
// only flag that needs the definition table — a structure's footprint
// lives there, not in the map — which it takes from -databin or from
// <assets>/world.res, and a run that cannot open one fails before any window
// appears. It SELECTS FLAT TERRAIN, exactly as -flat does: the wash is a
// whole cell of the cell lattice and displaced terrain paints that cell as a
// quad its four corner altitudes span, so on a slope — which is where a
// river bank is — the two do not register, and a full-cell fill that is
// offset answers the wrong question about the wrong cell. The game front-end
// has no equivalent flag.
//
// -check loads everything and prints the same summary line without opening a
// window, so the load path can be exercised headlessly. The asset root comes
// from -assets or AGAINROM_ASSETS and is never compiled in.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// graphicsArchive is the archive the terrain tiles live in, resolved relative to
// the configured asset root (research TERR-LOC-001).
const graphicsArchive = "graphics.res"

const usage = "usage: mapview -assets <dir> -map <file.alm> [-graphics <file.res>] [-databin <file.res>] [-speed 0..8] [-noanimation] [-objects] [-units] [-statics] [-staticmarkers] [-structures] [-ruins] [-objectanim] [-unshaded] [-flat] [-blocked] [-check]"

// blockedUsage is the -blocked flag's own help text, named for the reason
// the three above are — and its wording carries the two things a caller
// cannot guess: that this flag alone needs the definition table, and that it
// selects flat terrain because a full-cell wash does not register with
// displaced terrain on a slope.
const blockedUsage = "diagnostic: wash every cell the derived block plane closes to a ground mover; needs the definition table (-databin or <assets>/world.res) and selects flat terrain"

// unshadedUsage is the -unshaded flag's own help text: a diagnostic, not a
// second light. Named so the wording registered on the flag and the wording
// a test reads are the same string, not two copies that could drift.
const unshadedUsage = "diagnostic: render at full palette brightness instead of corner-interpolated relief shading; does not move the light itself"

// flatUsage is the -flat flag's own help text: a diagnostic, not a second
// altitude source — it moves no asset root and reads no game install, only
// the mode this process already decoded. Named so the wording registered on
// the flag and the wording a test reads are the same string, mirroring
// unshadedUsage.
const flatUsage = "diagnostic: render flat terrain instead of height-displaced, even over a valid altitude grid; the map stays lit"

// structuresUsage is the -structures flag's own help text, named for the
// reason the three below are. Its wording carries the one thing a caller
// cannot guess: the count reported is ENTRIES, and a structure covers a
// rectangle of cells each drawn as a strip of tile-sized frames, so the
// number is far larger than the building count and is not it.
const structuresUsage = "draw the map's placed structures over the terrain and report how many frame entries were placed"

// ruinsUsage is the -ruins flag's own help text, named for the reason the
// others are. Its wording carries the two things a caller cannot guess: that
// the flag is ours rather than a state the game would reach, and that a
// class carrying Indestructible does not change under it.
const ruinsUsage = "diagnostic: draw every drawable structure from its ruin block, respecting Indestructible; ours, not a state this tree models"

// objectAnimUsage is the -objectanim flag's own help text, named for the
// reason the two above are. Its wording is deliberately careful about what
// the flag is: OURS, not the game's, and an override of a decoded gate
// rather than a second animation model.
const objectAnimUsage = "diagnostic: open every object class's cycle whatever the map's tile words hold; ours, not the game's, and no shipped map opens one without it"

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "mapview:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("mapview", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // the error is returned and reported once, by main
	assets := fs.String("assets", "", "path to the game asset root (overrides AGAINROM_ASSETS)")
	graphics := fs.String("graphics", "", "path to graphics.res (default <assets>/"+graphicsArchive+")")
	mapPath := fs.String("map", "", "path to the .alm map to view")
	check := fs.Bool("check", false, "load the map and tileset, print a summary, and exit without a window")
	noAnimation := fs.Bool("noanimation", false, "draw water statically instead of cycling it")
	speed := fs.Int("speed", terrain.DefaultSpeedIndex, "game speed index 0..8 setting the logic-tick rate (clamped)")
	objects := fs.Bool("objects", false, "overlay a diagnostic marker on each placed object's anchor cell and report the object count")
	units := fs.Bool("units", false, "overlay a diagnostic marker on each placed unit's anchor cell and report the unit count")
	statics := fs.Bool("statics", false, "draw the map's static-object layer over the terrain and report the placement count")
	staticMarkers := fs.Bool("staticmarkers", false, "overlay a diagnostic marker on each type-3 cell that resolves to a drawable frame, positioned from the cell alone")
	structures := fs.Bool("structures", false, structuresUsage)
	ruins := fs.Bool("ruins", false, ruinsUsage)
	objectAnim := fs.Bool("objectanim", false, objectAnimUsage)
	unshaded := fs.Bool("unshaded", false, unshadedUsage)
	flat := fs.Bool("flat", false, flatUsage)
	blocked := fs.Bool("blocked", false, blockedUsage)
	dataBin := fs.String("databin", "", "path to world.res, holding the definition table -blocked needs (default <assets>/"+worldArchive+")")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%v\n%s", err, usage)
	}
	if *mapPath == "" {
		return fmt.Errorf("-map is required\n%s", usage)
	}

	// Resolved BEFORE the load, so a run that cannot name a table fails without
	// having opened the graphics archive or decoded a map. Absent the flag it
	// stays empty and nothing below reaches for a table at all.
	blockedArchive := ""
	if *blocked {
		a, err := resolveDataArchive(*assets, *dataBin)
		if err != nil {
			return err
		}
		blockedArchive = a
	}

	viewer, summary, err := loadWithTable(*assets, *graphics, *mapPath, *objects, *units, *statics, *staticMarkers,
		*objectAnim, blockedArchive, *structures)
	if err != nil {
		return err
	}
	configureViewer(viewer, *noAnimation, *unshaded, *speed)
	// The ruin diagnostic is wired HERE, beside the flat one and for its reason:
	// it is read by the draw path alone, moves no summary token and reorders
	// nothing, so it is composed at the flag level in this tool and the library's
	// own rules are untouched.
	viewer.SetRuins(*ruins)
	// -blocked selects flat as -flat does, so the wash lands on the cells the
	// terrain is actually drawn as. Composed here, at the FLAG level, in this
	// tool: the viewer's own rule that a valid altitude grid stays displaced
	// whichever overlays are on is untouched.
	configureFlat(viewer, *flat || *blocked)
	summary += ", " + cadence(viewer)
	fmt.Fprintln(out, summary)

	// The two-level census, under -check and under -structures alone: the
	// numbers the build produced, in the render tier's own wording, so this
	// tool and the raster harness cannot print different words for one counter.
	// It rides on its own two lines rather than inside the summary because
	// eight counters do not belong in a line three stories pin the composition
	// of.
	if *structures && *check {
		_, counts := viewer.Structures()
		placements, cells := counts.CensusLines()
		fmt.Fprintln(out, placements)
		fmt.Fprintln(out, cells)
	}

	if *check {
		return nil
	}
	return viewer.Run()
}

// configureViewer wires the parsed flags onto a viewer load() already built:
// water animation state and rate, and the unshaded diagnostic. Split out of
// run() so a test can drive this SAME wiring against a real load()-built
// viewer and inspect the result (v.Lit()) directly, without going through
// flag parsing or opening a window — run() itself returns only an error,
// and Run() opens a real window a headless test cannot call.
//
// SetUnshaded moves no summary token and reorders nothing (AC-9): it is read
// by the draw path alone, never by cadence or by load's summary building.
func configureViewer(v *ui.Viewer, noAnimation, unshaded bool, speed int) {
	v.SetAnimated(!noAnimation)
	v.SetSpeedIndex(speed)
	v.SetUnshaded(unshaded)
}

// configureFlat wires -flat onto a viewer load() already built: SetFlat
// alone. It is split out of configureViewer rather than added to its
// parameter list: an existing test (main_test.go,
// TestUnshadedFlagWiresToViewerLit and others) already calls configureViewer
// at its shipped four-argument shape, and this story's budget for editing an
// existing test file is spent elsewhere (SC-4) — cmd/mapview/main_test.go
// stays untouched. SetFlat itself re-syncs the camera's world extent on a
// mode flip, so there is nothing else for this wrapper to do.
func configureFlat(v *ui.Viewer, flat bool) {
	v.SetFlat(flat)
}

// load resolves the archive, decodes the map, builds the tileset and
// constructs the viewer. Every failure surfaces here — before any window
// opens.
//
// showObjects and showUnits enable the two diagnostic overlays independently.
// They are passed to the shared load path, which owns the placement-record ->
// anchor-cell conversion for both front-ends (0010 DD20 as revised, DD33); what
// stays here is the summary, which reports each count on its own flag. The
// conversion used to live in this function, and that is precisely how the game
// ended up without the overlays: the load path both entry points share knew
// nothing about them.
//
// showStatics and showStaticMarkers are the static-object layer's two
// switches: the map's own object art, and the diagnostic cross on every cell
// that art resolves. EITHER ONE LOADS THE BUNDLE — the cross needs it to
// know which cells resolve, even though it reads no class field and paints
// no sprite — and with neither flag none is loaded at all, which is what
// keeps a flagless run's reads, image and summary exactly the pre-story
// ones.
//
// Only the art switch REPORTS. The count is the layer's own content, so it
// belongs to -statics; -staticmarkers alone builds the same lists, marks every
// resolving cell and leaves the summary character-for-character unchanged.
// objectAnim is 0031's diagnostic gate, and it reaches the OBJECT LAYER ALONE:
// it is handed to the load path beside the bundle, decides which cells enter the
// animated subset, and moves no other count, token or pixel. It is passed
// whether or not the bundle is loaded, exactly as the art switch is — with no
// bundle there is nothing for a gate to open.
func load(assetsFlag, graphicsFlag, mapPath string, showObjects, showUnits, showStatics, showStaticMarkers, objectAnim bool) (*ui.Viewer, string, error) {
	return loadWithTable(assetsFlag, graphicsFlag, mapPath, showObjects, showUnits, showStatics, showStaticMarkers,
		objectAnim, "", false)
}

// loadWithTable is load with the one argument -blocked needs: the archive its
// definition table is read out of, empty for every other run.
//
// It is a SECOND ENTRY POINT rather than a widened load, and that is
// deliberate twice over. main_test.go pins load's signature as a function
// value — a compile-time assertion that this tool's load path has gained
// neither a tick nor an ordering seam — and widening it would break that
// pin for a reason it is not about. And 0015 made the same call for the same
// shape when -flat needed wiring: an existing test file stays untouched and
// the new parameter arrives through a new door. load is now the delegating
// half and holds no logic of its own, so the two cannot come apart.
//
// blockedArchive is a PATH and not a parsed table because the open is a failure
// this function owes its caller before a window exists, beside the graphics open
// and the map decode — and because a caller passing a parsed table would have
// had to open one to find out it could not.
func loadWithTable(assetsFlag, graphicsFlag, mapPath string, showObjects, showUnits, showStatics, showStaticMarkers, objectAnim bool,
	blockedArchive string, showStructures bool) (*ui.Viewer, string, error) {
	archivePath, err := resolveArchive(assetsFlag, graphicsFlag)
	if err != nil {
		return nil, "", err
	}

	// The archive open, the tileset build and the map-bytes-to-viewer step all
	// live in pkg/game now, so this tool and the game front-end share one loading
	// path and cannot drift apart. What stays here is presentation: the summary
	// wording below and the `decode <path>` label are this tool's own contract,
	// and freezing them in a library API would impose a developer tool's phrasing
	// on the front-end.
	//
	// THE CONTAINER FILESYSTEM AND THE TILESET COME BACK TOGETHER, so this tool
	// opens the graphics archive exactly once however many of its layers are
	// asked for. The archive that used to arrive here instead of the filesystem
	// existed so the object bundle could be loaded off the handle ALREADY OPEN;
	// the bundle is addressed by container now, and the filesystem is what an
	// address resolves against, so the same value serves both and the
	// transitional second open is gone. The frozen `open <path>: <err>` wording
	// every failing-archive test reads is the filesystem's own and is unchanged
	// by any of it.
	//
	// The tileset behind it is built off that filesystem: terrain's tile
	// constants are addresses now, so nothing on this path names a bare entry.
	containers, tiles, err := game.OpenGraphics(archivePath)
	if err != nil {
		return nil, "", err
	}

	// The bundle comes off the filesystem opened above: the registry and every
	// object sheet carry graphics.res's identity segment now, so the loader is
	// told which container to read rather than handed one.
	//
	// EITHER FLAG loads it and neither leaves it loaded, exactly as before:
	// with no flag the image and summary stay the pre-story ones. What no
	// longer depends on the flags is the OPEN — one archive read serves a run
	// with the layer and a run without it.
	var staticSet *terrain.StaticSet
	if showStatics || showStaticMarkers {
		if staticSet, err = game.LoadStatics(containers); err != nil {
			return nil, "", err
		}
	}

	// The structure bundle comes off the same filesystem, on its own flag
	// alone. Absent it nothing is read, no class is decoded, and the picture
	// and summary are character-for-character the pre-story ones. An unreadable
	// or unparseable registry is the one failure that is an error; every class
	// it cannot draw is a skip the census counts.
	var structureSet *terrain.StructureSet
	if showStructures {
		if structureSet, err = game.LoadStructures(containers); err != nil {
			return nil, "", err
		}
	}

	mapData, err := os.ReadFile(mapPath)
	if err != nil {
		return nil, "", err
	}
	loaded, err := game.LoadMapViewer(tiles, mapData, filepath.Base(mapPath),
		game.Markers{Objects: showObjects, Units: showUnits, Statics: showStaticMarkers},
		// Absent both flags this is the zero layer — no bundle and no art — which
		// is this tool's shipped behaviour exactly.
		game.StaticLayer{Set: staticSet, Art: showStatics, AnimGate: objectAnim},
		// Absent the flag this is the zero layer — no bundle and no art — which is
		// this tool's shipped behaviour exactly.
		game.StructureLayer{Set: structureSet, Art: showStructures})
	if err != nil {
		return nil, "", fmt.Errorf("decode %s: %w", mapPath, err)
	}
	m, viewer, title := loaded.Map, loaded.Viewer, loaded.Title

	summary := fmt.Sprintf("mapview: %s %dx%d cells (%d), tile slots %d/%d",
		title, m.Width, m.Height, m.Width*m.Height, tiles.Loaded, terrain.SlotCount)

	// Each overlay is opt-in, so absent its flag nothing is drawn and the
	// summary is character-for-character its earlier shape. The unit token is
	// appended after the object token and both here, inside load(), so the
	// composed summary reads "... objects N, units M, water speed ..." —
	// object count before unit count, whichever order the flags were given on
	// the command line.
	//
	// The statics token sits immediately BEFORE the object token, which is what
	// keeps the object and unit tokens adjacent to each other and to the
	// cadence — the composition three stories pin — and is the position the
	// raster tool puts it in too.
	if showStatics {
		// Reported on the flag alone, never on the count, exactly as the unit
		// token is: a map whose layer resolves nothing must still print
		// "statics 0", or an empty layer and an unrequested one would be
		// indistinguishable. The count is the placements DRAWABLE — a byte naming
		// no loaded class and a class with no art are skips, not placements — and
		// it is the same number in either geometry, so -flat cannot move it.
		//
		// The open-cycle count rides INSIDE the statics token rather than beside
		// it, so the object and unit tokens stay adjacent and the composition
		// three stories pin is unmoved. It is the census's own number — the one
		// the build produced — and it is 0 on every shipped map, which is the
		// whole reason it is worth printing.
		placements, counts := viewer.Statics()
		summary += fmt.Sprintf(", statics %d (animated %d)", placements, counts.Animated)
	}
	// Reported on the flag alone, never on the count, exactly as the two tokens
	// either side of it are. It sits between the statics and object tokens so the
	// object and unit tokens stay adjacent to each other and to the cadence — the
	// composition three stories pin — and so a flagless run's summary is
	// character-for-character its pre-story shape.
	//
	// THE NUMBER IS ENTRIES AND NOT BUILDINGS: one structure covers a rectangle of
	// cells and each of those is a vertical strip of tile-sized frames, so this
	// counts frames placed. How many structures drew is the census's own number.
	if showStructures {
		entries, counts := viewer.Structures()
		summary += fmt.Sprintf(", structures %d (%d placed)", entries, counts.Placements.Drawn)
	}
	if showObjects {
		summary += fmt.Sprintf(", objects %d", len(m.Objects))
	}
	if showUnits {
		// On the flag alone, never on the count: a unit-free map still reports
		// "units 0".
		summary += fmt.Sprintf(", units %d", len(m.Units))
	}

	// THE TINT IS LAST of what this function builds, after the object and unit
	// tokens and before the cadence run() appends. The position is not a
	// preference: the shipped tests reconstruct the -objects summary by
	// splicing its token at exactly this point, so a token inserted anywhere
	// earlier would move a composition three stories pin.
	if blockedArchive != "" {
		cells, err := blockedOverlay(m, blockedArchive)
		if err != nil {
			return nil, "", err
		}
		viewer.SetBlocked(true, cells)
		summary += fmt.Sprintf(", blocked %d", len(cells))
	}

	return viewer, summary, nil
}

// cadence describes the configured water animation, so a headless -check run
// records the rate it would have animated at.
func cadence(v *ui.Viewer) string {
	on, idx := v.Animation()
	if !on {
		return "water static"
	}
	dt := terrain.TickMillis(idx)
	return fmt.Sprintf("water speed %d (%d tps, %d ms/tick, %d ms/cycle)",
		idx, terrain.TicksPerSecond(idx), dt, dt*terrain.WaterCycleTicks)
}

func resolveArchive(assetsFlag, graphicsFlag string) (string, error) {
	if graphicsFlag != "" {
		return graphicsFlag, nil
	}
	root := game.ResolveAssetRoot(assetsFlag, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		return "", fmt.Errorf("no asset root configured; pass -assets, set AGAINROM_ASSETS, or give -graphics")
	}
	return filepath.Join(root, graphicsArchive), nil
}
