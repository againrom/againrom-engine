// Command terraintool renders a ROM1 map's terrain to a PNG, censuses a
// map's units against the unit-art bundle, and audits the unit classes'
// animation descriptors against their own sheets (developer tool).
//
// Usage:
//
//	terraintool render     -assets <dir> -map <file.alm> -out <file.png> [-scale N]
//	                       [-graphics <file.res>] [-objects] [-units] [-statics]
//	                       [-staticmarkers] [-structures] [-ruins] [-objectanim]
//	                       [-tick N] [-flat]
//	terraintool units      -assets <dir> -map <file.alm> [-graphics <file.res>]
//	terraintool unitanim   -assets <dir> [-graphics <file.res>]
//	terraintool structures -assets <dir> -map <file.alm> [-graphics <file.res>]
//	                       [-databin <file.res>]
//	terraintool tiers      -assets <dir> [-graphics <file.res>] [-class N -out <file.png>]
//
// terraintool is the wiring the render tier deliberately does not carry: it opens
// graphics.res as a container filesystem through pkg/game, loads the terrain
// tile strips into the tileset by their addresses, decodes the .alm with
// pkg/formats/alm, composites the map's tile grid with pkg/render/terrain, and
// writes the result as a PNG. It prints one summary line — map size, cell count,
// output size, the number of cells that fell back to a placeholder, and the
// geometry and vertical origin the image was drawn on — and exits non-zero on any
// load failure.
//
// -statics draws the map's static-object layer — the type-3 cell grid's
// trees, bushes, stones and fences — over the finished terrain, and
// -staticmarkers puts a diagnostic cross on every cell that layer resolves.
// The two are independent switches on purpose: the cross is positioned from
// the CELL alone and the art from the object class, so the two are
// independent derivations of one ground point and an object standing away
// from its own cross is a placement bug the owner can see. Object art is
// full resolution and this raster path does not resample it, so -statics
// requires -scale 1; -staticmarkers is usable at any scale.
//
// -structures draws the map's PLACED STRUCTURES — its buildings — over
// the finished terrain, in the one merged order the windowed viewer draws
// them in: a class with Flat set first, then the rest merged with the object
// layer by rectangle row. The two front-ends place from ONE builder and
// merge with ONE helper, which is what makes it impossible for them to
// disagree about where a building stands or which drawable is in front. Like
// -statics it is full resolution and unresampled, so it requires -scale 1.
// -ruins is the destruction diagnostic: every drawable structure drawn from
// its ruin block, respecting Indestructible; ours, not a state this tree
// models.
//
// The structures subcommand is the corpus census (AC-10). It loads the
// structure bundle exactly as the game does, builds the map's placement list
// through the one builder, and prints the two counter levels plus the three
// observations AC-10 asks for and does not reconcile:
//
//	structures: D drawn, N no class, U undrawable, V variable-size
//	cells: S strips, F frames (O overhang), X outside the map
//	observations: variable-size ids [..], extensions E, rectangle mismatches M of C
//
// The third observation needs the definition table, from -databin or
// <assets>/world.res, and is the only part of this tool that opens it: it asks,
// class by class, whether the REGISTRY rectangle this story draws and the
// definition table's own sizeX/sizeY agree. They are independently sourced and
// nothing at this pin asserts that they do; the count is recorded and reconciled
// with nothing. Without a table the observation is reported as unavailable rather
// than as zero.
//
// -tick selects the animation counter the still is taken at, and -objectanim
// is a diagnostic of OURS that opens every object class's cycle whatever the
// map's tile words hold. The decoded gate on an object's cycle is a test on
// four tile words that no shipped cell satisfies, so WITHOUT -objectanim the
// object layer draws every shipped map exactly as it drew it before that
// story, at every -tick. Both are off/zero by default and reach the object
// layer alone; the summary reports how many placements have an open cycle
// beside the placement count.
//
// The terrain is height-displaced by default (0012): every cell is a quad whose
// corners the map's altitudes push up or down, and the image spans exactly the
// rows that mesh reaches, so its height is not W*32*scale and its row 0 stands
// for a native row that -flat's does not. -flat selects the whole-image flat
// raster of 0004/0007 instead, byte-for-byte as it was before 0012 — a
// diagnostic, no longer the default. Geometry and light are independent axes, so
// -flat and -unshaded select two unrelated things and all four combinations
// render.
//
// The units subcommand is the corpus census — 0022 AC-11's harness. It
// loads the unit-art bundle out of graphics.res exactly as the game does,
// builds the map's own world through pkg/game's UnitCensus — the tool
// restates no join — and prints one line:
//
//	units: E entities, S sprites, C no-class, F no-frame
//
// E is the map's placed-unit count, S the entities whose class id resolves to
// a drawn frame, C the ids naming no class, F the classes without drawable
// art; the three counts partition the E entities. -assets, -map and -graphics
// mean exactly what render's do; it writes no file and exits non-zero on any
// load failure — an unreadable archive, registry or map is an error here,
// because a census over a broken install would be a wrong count, not a count
// of a broken install.
//
// The unitanim subcommand is the 0024 corpus instrument — AC-10's audit
// over a lawful install's unit registry. No map is involved: it loads the
// unit-art bundle exactly as the game does, hands it to pkg/game's
// UnitAnimAudit, and prints one line per class, ascending by id, plus a
// summary:
//
//	class <id>: predicted <P>, frames <F>, in-range <I>, guarded <G>
//	unitanim: <C> classes, <M> mismatched, <I> in-range, <G> guarded
//
// P is the descriptor's predicted sheet total, F the sheet's own frame
// count, and I/G split the full selection domain — both states, all 8
// octants, every step of each track — into the selections the bounds guard
// left unchanged and the ones it refused. A mismatched class or a guarded
// selection is DATA about the install, recorded in the output and never an
// error: the tool exits non-zero only on a load failure, exactly as units
// does.
//
// The tiers subcommand is the 0057 corpus instrument. No map is involved:
// which colours a class ships is a fact about the registry and the archive.
// It loads the unit-art bundle exactly as the game does and prints one line
// per class that carries tiers, ascending by id, plus a summary:
//
//	class <id> "<name>": <N> tier(s), <L> loaded, <B> fell back, <F> frames, tier 1 <T1>
//	tiers: <T> of <C> classes carry tiers, <D> declared, <L> loaded, <B> fell back,
//	       <S> take the sheet's own table at tier 1
//
// N is the class's declared tier count and a fall-back is a tier whose colour
// table was absent or refused — read off the bundle's own slices, not off a
// counter beside them. T1 says whether tier 1's table EQUALS the sheet's own,
// which is the one claim about the shipped corpus a census can make and a
// picture cannot: a tier that is not a recolour keeps the sheet's very frames.
// A fall-back is DATA about the install, recorded and never an error: the tool
// exits non-zero only on a load failure, exactly as units and unitanim do.
//
// Given -class and -out together it also writes the STILL: that class's tiers
// side by side, ascending, one panel each, every panel drawing the same frame
// index — the class's own idle selection at octant 0 and tick 0 — through the
// same lit blit the window builds its textures with, so the panels differ in
// colour alone. Like a rendered map it is a converted game asset: point -out
// outside the repository.
//
// The asset root comes from -assets or AGAINROM_ASSETS and is never compiled in.
// It is a developer-run tool for verifying the decoder against a lawful install
// and is never part of the test suite's game-facing path. Point -out at a
// git-ignored folder (the repo ignores terraintool-out/): a rendered map is a
// converted game asset and must never be committed.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"

	"againrom/pkg/formats/alm"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
)

// graphicsArchive is the archive the terrain tiles live in, resolved relative to
// the configured asset root (research TERR-LOC-001).
const graphicsArchive = "graphics.res"

const usage = "usage: terraintool render -assets <dir> -map <file.alm> -out <file.png> [-scale N] [-graphics <file.res>] [-objects] [-units] [-statics] [-staticmarkers] [-structures] [-ruins] [-objectanim] [-tick N] [-flat]"

// unitsUsage is the census subcommand's own line. A second const and not an
// edit of the first: render's own errors keep quoting exactly the text they
// always did, and only the top-level dispatch quotes both.
const unitsUsage = "usage: terraintool units -assets <dir> -map <file.alm> [-graphics <file.res>]"

// unitanimUsage is the audit subcommand's line, a third const for the same
// reason. No -map: the audit is over the registry's classes, not a map's
// placements.
const unitanimUsage = "usage: terraintool unitanim -assets <dir> [-graphics <file.res>]"

// structuresUsage is the structure census subcommand's line, a fourth const for
// the reason the two above are: one wording, in one place, reachable by the
// dispatch and by a test alike.
const structuresUsage = "usage: terraintool structures -assets <dir> -map <file.alm> [-graphics <file.res>] [-databin <file.res>]"

// tiersUsage is the per-tier census's line, a fifth const for the reason the
// four above are: one wording, in one place, reachable by the dispatch and by a
// test alike. No -map: which colours a class ships is a fact about the registry
// and the archive, not about any map's placements.
const tiersUsage = "usage: terraintool tiers -assets <dir> [-graphics <file.res>] [-class N -out <file.png>]"

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "terraintool:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	// units and unitanim are dispatched ahead of the render guard, which
	// otherwise stays the tool's shipped front door: no subcommand at all,
	// and anything that is none of the names, gets every usage line and a
	// non-zero exit.
	if len(args) >= 1 && args[0] == "units" {
		return runUnits(args[1:], out)
	}
	if len(args) >= 1 && args[0] == "unitanim" {
		return runUnitAnim(args[1:], out)
	}
	if len(args) >= 1 && args[0] == "structures" {
		return runStructures(args[1:], out)
	}
	if len(args) >= 1 && args[0] == "tiers" {
		return runTiers(args[1:], out)
	}
	if len(args) < 1 || args[0] != "render" {
		return fmt.Errorf("%s\n%s\n%s\n%s\n%s", usage, unitsUsage, unitanimUsage, structuresUsage, tiersUsage)
	}

	fs := flag.NewFlagSet("terraintool render", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // the error is returned and reported once, by main
	assets := fs.String("assets", "", "path to the game asset root (overrides AGAINROM_ASSETS)")
	graphics := fs.String("graphics", "", "path to graphics.res (default <assets>/"+graphicsArchive+")")
	mapPath := fs.String("map", "", "path to the .alm map to render")
	outPath := fs.String("out", "", "path to write the PNG to")
	scale := fs.Int("scale", 1, "integer pixel scale (1 = one screen pixel per tile pixel)")
	objects := fs.Bool("objects", false, "overlay a diagnostic marker on each placed object's anchor cell and report the object count")
	units := fs.Bool("units", false, "overlay a diagnostic marker on each placed unit's anchor cell and report the unit count")
	statics := fs.Bool("statics", false, "draw the map's static-object layer over the terrain and report the placement count (requires -scale 1: object art is not resampled)")
	staticMarkers := fs.Bool("staticmarkers", false, "overlay a diagnostic marker on each type-3 cell that resolves to a drawable frame, positioned from the cell alone")
	structures := fs.Bool("structures", false, "draw the map's placed structures over the terrain and report the two census levels (requires -scale 1: structure art is not resampled)")
	ruins := fs.Bool("ruins", false, "diagnostic: draw every drawable structure from its ruin block, respecting Indestructible; ours, not a state this tree models")
	objectAnim := fs.Bool("objectanim", false, "diagnostic: open every object class's cycle whatever the map's tile words hold; ours, not the game's, and no shipped map opens one without it")
	tick := fs.Uint("tick", 0, "the animation counter to render the still at (default 0, the value a map opens at)")
	unshaded := fs.Bool("unshaded", false, "render at flat full palette brightness (the 0004 path); no relief shading")
	flat := fs.Bool("flat", false, "draw the whole-image flat raster instead of the height-displaced default: every cell an axis-aligned 32x32 square on a W*32 x H*32 canvas, as before 0012 (a diagnostic)")
	mapLight := fs.Bool("maplight", false, "take the sun from the map's stored +0x08/+0x10/+0x14 fields (a viewer choice, not engine-faithful)")
	theta := fs.Float64("theta", math.NaN(), "override the sun angle in radians (default: the engine's cycle-off 0.78539815)")
	ambient := fs.Int("ambient", -1, "override the ambient intensity byte 0..255 (default: 0x0e)")
	rangeByte := fs.Int("range", -1, "override the directional range byte 0..255 (default: 0x20)")
	if err := fs.Parse(args[1:]); err != nil {
		return fmt.Errorf("%v\n%s", err, usage)
	}
	if *mapPath == "" || *outPath == "" {
		return fmt.Errorf("-map and -out are required\n%s", usage)
	}
	// The refusal comes FIRST, immediately after the required-flag check:
	// before the archive is opened, before the map is read, and far before
	// os.Create. Object art is full resolution and this raster path does not
	// resample it, so an unsatisfiable combination must cost the caller nothing
	// — no output file created, none truncated, none modified, and no image
	// composed. Ordering is the whole mechanism: there is no cleanup path here
	// that could put a file back the way it was.
	if *statics && *scale != 1 {
		return fmt.Errorf("-statics requires -scale 1 (object art is not resampled)")
	}
	// The same refusal, in the same place and for the same reason: a structure
	// frame IS a map cell, and this raster path resamples no sprite.
	if *structures && *scale != 1 {
		return fmt.Errorf("-structures requires -scale 1 (structure art is not resampled)")
	}

	archivePath, err := resolveArchive(*assets, *graphics)
	if err != nil {
		return err
	}

	// ONE container filesystem over the archive, and this tool holds no
	// *res.Archive at all any more. Both consumers name addresses now —
	// terrain's tile constants and pkg/game's registry and sheet constants each
	// carry graphics.res's identity segment — so each states which container
	// its own assets live in and this function is told nothing out of band. The
	// transitional double-read of the one file, which the bundle flip cost, is
	// paid back here: one open serves both.
	//
	// The failure is the filesystem's own `open <path>: <err>` — byte for
	// byte the wording this tool has always printed for an archive that will
	// not open, from one place instead of a wrapper stacked on top of it.
	containers, err := game.OpenContainers(archivePath)
	if err != nil {
		return err
	}
	tiles := terrain.LoadTileset(containers)

	// The bundle is still built under the two flags ALONE — -staticmarkers needs it
	// to know which cells resolve, even though it paints no sprite — and with
	// neither flag it is not built, which is what keeps a flagless run's summary and
	// image exactly the pre-story ones.
	var staticSet *terrain.StaticSet
	if *statics || *staticMarkers {
		if staticSet, err = game.LoadStatics(containers); err != nil {
			return err
		}
	}

	// On its own flag alone, so a flagless run reads no structure registry and
	// its image and summary stay the pre-story ones.
	var structureSet *terrain.StructureSet
	if *structures {
		if structureSet, err = game.LoadStructures(containers); err != nil {
			return err
		}
	}

	mapData, err := os.ReadFile(*mapPath)
	if err != nil {
		return err
	}
	m, err := mapOpener(*assets)(mapData)
	if err != nil {
		return fmt.Errorf("decode %s: %w", *mapPath, err)
	}

	// Overlay rides on the grid beside the tiles because it is part of the MAP.
	// It is set unconditionally and costs a flagless run nothing: no compositor
	// reads the field, and StaticPlacements — its one reader — is called
	// only under a flag. The type-4 records ride on the grid beside the object
	// bytes and for the same reason: they are part of the MAP. Set
	// unconditionally, and costing a flagless run one slice of the map's own
	// records, since StructurePlacements is the field's one reader and it is
	// called only under a flag.
	grid := terrain.Grid{
		Width: m.Width, Height: m.Height, Tiles: m.Tiles, Overlay: m.Overlay,
		Structures: game.StructureRecords(m.Objects),
	}

	// Geometry and light are independent axes, so the two flags select two
	// unrelated things and each of the render tier's four entry points is
	// reachable on its own.
	var render *terrain.Render
	var lightDesc string
	geometryDesc := "projected"
	if *flat {
		geometryDesc = "flat"
	}
	// THE SUN IS RESOLVED ONCE, ABOVE THE BRANCH. It used to be a local of the
	// shaded arm alone, so nothing under -unshaded could see it; the sprite
	// layer below needs it in BOTH arms — under the diagnostic to know it is
	// off, and otherwise to take its row — and one resolution is what keeps
	// -ambient, -theta, -range and -maplight from meaning one thing to the
	// ground and another to the art standing on it. Which compositor runs, and
	// what the descriptor says, are unchanged: the unshaded arm still calls the
	// unshaded pair and still prints the single token "unshaded".
	light := resolveLight(m, *mapLight, *theta, *ambient, *rangeByte)
	if *unshaded {
		lightDesc = "unshaded"
		if *flat {
			render, err = terrain.Composite(tiles, grid, *scale)
		} else {
			render, err = terrain.CompositeProjected(tiles, grid, m.Altitudes, *scale)
		}
	} else {
		lightDesc = fmt.Sprintf("shaded (theta=%.4f ambient=%d range=%d)", light.Theta, light.Ambient, light.Range)
		if *flat {
			render, err = terrain.CompositeLit(tiles, grid, m.Altitudes, light, *scale)
		} else {
			render, err = terrain.CompositeProjectedLit(tiles, grid, m.Altitudes, light, *scale)
		}
	}
	if err != nil {
		return err
	}

	// The overlays draw on the un-displaced cell lattice, translated by the
	// same vertical origin the terrain took. Render.OriginY is the NATIVE row
	// the image's row 0 stands for, so the lattice shift is -OriginY*scale
	// OUTPUT pixels — the units DrawObjectMarkersAt/*Heights take, and the
	// sign that puts native row row*32 at row*32*scale - OriginY*scale. It is 0
	// on the flat raster, so one expression serves both geometries rather than
	// a second call site that could drift from this one.
	markerOffsetY := -render.OriginY * *scale

	var proj *terrain.Projection
	var liftY func(col, row int) int
	var heightAt func(col, row int) int
	if !*flat && (*objects || *units || *statics || *staticMarkers || *structures) {
		p := terrain.Project(m.Altitudes, m.Width, m.Height)
		proj = &p
		liftY = func(col, row int) int { return -proj.AnchorHeight(col, row) * *scale }
		heightAt = proj.AnchorHeight
	}

	// The placement list, in NATIVE units: heightAt is nil on the flat geometry
	// (mirroring the nil-safe liftAt the marker path already takes) and
	// render.OriginY is the native row the image's row 0 stands for, before
	// scale. No -scale enters the list, which is what lets -staticmarkers stay
	// usable at any scale while the blit below is confined to the native one.
	var places, drawn []terrain.StaticPlacement
	var staticCounts terrain.StaticCounts
	staticsDesc := ""
	if *statics || *staticMarkers {
		var animated []int
		places, staticCounts, animated = terrain.StaticPlacements(grid, staticSet, heightAt, render.OriginY, *objectAnim)
		if err := checkStaticGround(places, render.OriginY, heightAt); err != nil {
			return err
		}
		// The counter is resolved ONCE, above the blit loop: the tool writes one
		// image, so the still is one counter's picture and a sequence is the
		// caller's own loop over -tick. The pass is handed a fresh buffer because
		// there is exactly one of these per run, and animation is ON — this tool
		// has no disable switch and gains none.
		//
		// A -tick past a uint32 wraps into one, which is what the counter it
		// names does after 2^32 ticks; the marker passes below and the ground
		// check above read the BUILT list, whose cells and ground points no
		// counter moves.
		drawn = terrain.AnimateStatics(nil, places, animated, uint32(*tick), true)
	}

	// The structure list, built by the SAME function the window builds it with
	// and at the same two terms: the projection's CORNER accessor — not the
	// per-cell one the object layer takes — and render.OriginY, the native
	// row the image's row 0 stands for, which is the window's own MinV. Nothing
	// here adjusts what the builder produced.
	var structurePlaces, structureDrawn []terrain.StructurePlacement
	var structureCounts terrain.StructureCounts
	structuresDesc := ""
	if *structures {
		var cornerAt func(c, r int) int
		if proj != nil {
			cornerAt = proj.Altitude
		}
		var animated []int
		structurePlaces, structureCounts, animated = terrain.StructurePlacements(
			grid, structureSet, cornerAt, render.OriginY)
		structureDrawn = terrain.AnimateStructures(nil, structurePlaces, animated, uint32(*tick), true, *ruins)
	}

	// ONE MERGE, and it is the render tier's. The windowed viewer walks the
	// very same order, so the two front-ends cannot come to disagree about
	// which of two overlapping drawables is in front. With no structures it is
	// the object list in its own order, ref for ref, which is what keeps a
	// flagless run's image byte-identical.
	planeOrder := terrain.PlaneOrder(structurePlaces, places)

	// The layer draws after every compositor and before all three marker
	// passes, in list order — so where two sprites overlap the later cell
	// owns the overlap (0017 "Draw order"), and every marker stays readable
	// over the art. At scale 1 by the refusal above, so a placement's world
	// pixel is an image pixel and nothing is resampled.
	//
	// 0044 lights it. The row is the resolved sun's own — terrain.SpriteRow
	// of the very Light the compositor above was handed — so no second flag
	// selects a sprite row and -unshaded turns the ground and the art off
	// together. ONE loop and one pass either way: the arm is chosen per
	// placement out of a value already computed, never per pixel, and neither
	// arm draws anything the other does not.
	if *statics || *structures {
		row := terrain.SpriteRow(light)
		blit := func(f *terrain.StaticFrame, x, y int) {
			if f == nil {
				return
			}
			if *unshaded {
				terrain.BlitStatic(render.Image, f, x, y)
				return
			}
			terrain.BlitStaticLit(render.Image, f, x, y, light.SkyTint, row)
		}
		// The EARLY pass: a class with Flat set draws before everything else on
		// the map, ground decoration under every other drawable.
		for _, p := range structureDrawn {
			if p.Class != nil && p.Class.Flat {
				blit(p.Frame, p.TopLeft.X, p.TopLeft.Y)
			}
		}
		// Then the merged main plane, in the render tier's own order. Each side
		// is gated on its own flag, so either layer is drawable without the
		// other and neither reorders the other.
		for _, ref := range planeOrder {
			if !ref.Structure {
				if *statics && ref.Index < len(drawn) {
					p := drawn[ref.Index]
					blit(p.Frame, p.TopLeft.X, p.TopLeft.Y)
				}
				continue
			}
			if !*structures || ref.Index >= len(structureDrawn) {
				continue
			}
			p := structureDrawn[ref.Index]
			if p.Class != nil && p.Class.Flat {
				continue
			}
			blit(p.Frame, p.TopLeft.X, p.TopLeft.Y)
		}
	}
	if *structures {
		// Reported on the flag alone, never on the count. The number is the
		// ENTRIES placed — one per image row of each rectangle cell — beside how
		// many placements drew, because a structure is not a sprite and the two
		// numbers answer different questions. The full two-level census is the
		// structures subcommand's.
		structuresDesc = fmt.Sprintf(", structures %d (%d placed)",
			len(structurePlaces), structureCounts.Placements.Drawn)
	}
	if *statics {
		// Reported on the flag alone, never on the count, exactly as the unit
		// token is: a map with no drawable statics must still print "statics 0",
		// or an empty layer and an unrequested one would be indistinguishable.
		// The count is the placements DRAWN — a byte naming no class and a class
		// with no art are skips, not placements, and are not reported here.
		//
		// The open-cycle count rides INSIDE the statics token rather than beside
		// it, so the statics and object tokens stay adjacent and the composition
		// 0017 pins is unmoved. It is the census's own number, the one the build
		// produced.
		staticsDesc = fmt.Sprintf(", statics %d (animated %d)", len(places), staticCounts.Animated)
	}

	// Each overlay is opt-in and draws over the finished image, so with neither
	// flag no overlay code runs at all and the PNG is exactly what the
	// compositor produced. Units are drawn after objects, so a unit marker
	// lands on top of a coincident object marker.
	objectsDesc := ""
	if *objects {
		if liftY != nil {
			terrain.DrawObjectMarkersAtHeights(render.Image, anchorCells(m), m.Width, m.Height, terrain.CellSize**scale, markerOffsetY, liftY)
		} else {
			terrain.DrawObjectMarkersAt(render.Image, anchorCells(m), m.Width, m.Height, terrain.CellSize**scale, markerOffsetY)
		}
		objectsDesc = fmt.Sprintf(", objects %d", len(m.Objects))
	}
	unitsDesc := ""
	if *units {
		if liftY != nil {
			terrain.DrawUnitMarkersAtHeights(render.Image, unitAnchorCells(m), m.Width, m.Height, terrain.CellSize**scale, markerOffsetY, liftY)
		} else {
			terrain.DrawUnitMarkersAt(render.Image, unitAnchorCells(m), m.Width, m.Height, terrain.CellSize**scale, markerOffsetY)
		}
		// Reported on the flag alone, never on the count: a map with no units must
		// still print "units 0", or a unit-free map and an unrequested overlay
		// would be indistinguishable in the output.
		unitsDesc = fmt.Sprintf(", units %d", len(m.Units))
	}
	// The third glyph runs LAST of the three, at the same offsetY and the same
	// liftY: it is a strict subset of both shipped crosses at native scale and
	// above, so drawing it last cannot hide a coincident object or unit marker,
	// where drawing it first would hide this one entirely. WHICH cells are
	// marked comes from the built list; WHERE a mark goes comes from the cell,
	// through the marker geometry alone.
	if *staticMarkers {
		cells := staticCells(places)
		if liftY != nil {
			terrain.DrawStaticMarkersAtHeights(render.Image, cells, m.Width, m.Height, terrain.CellSize**scale, markerOffsetY, liftY)
		} else {
			terrain.DrawStaticMarkersAt(render.Image, cells, m.Width, m.Height, terrain.CellSize**scale, markerOffsetY)
		}
	}

	if err := writePNG(*outPath, render.Image); err != nil {
		return err
	}

	// The dimensions are read off the image that was just written, never
	// recomputed from the map size and the scale: under the projection the
	// height is the vertex mesh's own extent and W*32 x H*32 is no longer it,
	// so a summary derived from the cell grid would be a second, wrong answer
	// rather than a report of the file on disk (AC-12). Geometry and origin are
	// two adjacent tokens after the light descriptor and before the overlay
	// counts, so the "summary + , objects N" composition the overlay stories
	// pin still holds.
	//
	// The statics token sits immediately BEFORE the object token, which keeps
	// the object and unit tokens adjacent — the "summary + , objects N, units
	// M" composition three stories pin.
	b := render.Image.Bounds()
	fmt.Fprintf(out, "terrain: %dx%d cells (%d), %dx%d px at scale %d, tile slots %d/%d, placeholder cells %d, %s, geometry %s, y origin %d%s%s%s%s\n",
		m.Width, m.Height, m.Width*m.Height,
		b.Dx(), b.Dy(), *scale,
		tiles.Loaded, terrain.SlotCount, render.Placeholders, lightDesc,
		geometryDesc, render.OriginY, staticsDesc, structuresDesc, objectsDesc, unitsDesc)
	return nil
}

// runUnits is the units subcommand — the corpus census AC-11 records over
// a lawful install, one map per invocation (SC-9).
//
// The three flags are render's own, resolved by the same helper: the archive
// through resolveArchive — -graphics winning, else <root>/graphics.res
// with the root from -assets or AGAINROM_ASSETS — and the map a host path.
//
// Every failure returns an error and main exits non-zero, because the census
// is a recorded measurement: a line printed over a half-loaded install would
// be a wrong count that looks like a right one. There is no partial output —
// the one line prints only after everything has loaded.
func runUnits(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("terraintool units", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // the error is returned and reported once, by main
	assets := fs.String("assets", "", "path to the game asset root (overrides AGAINROM_ASSETS)")
	graphics := fs.String("graphics", "", "path to graphics.res (default <assets>/"+graphicsArchive+")")
	mapPath := fs.String("map", "", "path to the .alm map to census")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%v\n%s", err, unitsUsage)
	}
	if *mapPath == "" {
		return fmt.Errorf("-map is required\n%s", unitsUsage)
	}

	archivePath, err := resolveArchive(*assets, *graphics)
	if err != nil {
		return err
	}
	// The census needs the bundle and nothing else, so the container filesystem
	// is the whole of what this subcommand opens: the unit registry and every
	// sheet it names are addressed by graphics.res's identity segment now, and
	// the archive handle the tileset needs has no reader here. The failure text
	// is unchanged — `open <path>: <err>`, the filesystem's own.
	containers, err := game.OpenContainers(archivePath)
	if err != nil {
		return err
	}
	// LoadUnits' one error — an unreadable or unparseable unit registry — is the
	// broken install SC-9 names, passed through untouched.
	set, err := game.LoadUnits(containers)
	if err != nil {
		return err
	}

	mapData, err := os.ReadFile(*mapPath)
	if err != nil {
		return err
	}
	m, err := mapOpener(*assets)(mapData)
	if err != nil {
		return fmt.Errorf("decode %s: %w", *mapPath, err)
	}

	// E is the map's own placed-unit count, NOT the sum of the three buckets:
	// FromALM builds exactly one entity per unit record, so the sum must equal
	// it, and printing the independent count means a census that lost an entity
	// prints a line whose arithmetic is visibly wrong instead of one that hides
	// the loss.
	c := game.UnitCensus(m, set)
	fmt.Fprintf(out, "units: %d entities, %d sprites, %d no-class, %d no-frame\n",
		len(m.Units), c.Sprites, c.NoClass, c.NoFrame)
	return nil
}

// runUnitAnim is the unitanim subcommand — the 0024 corpus instrument,
// AC-10's audit over a lawful install's unit registry (SC-9).
//
// The two flags are render's own pair, resolved by the same helper; there is
// no -map because the audit is a fact about the registry's classes and their
// sheets, not about any map's placements. The sweeping itself is pkg/game's
// UnitAnimAudit: this function opens the archive, loads the bundle and
// prints, and the domain, the counting and the row order stay in the package
// that owns the join.
//
// ONLY A LOAD FAILURE IS AN ERROR — an unreadable archive or an unreadable or
// unparseable unit registry — and a failed run prints nothing: the audit is a
// recorded measurement, and a line over a half-loaded install would be a
// wrong figure that looks like a right one. A mismatched class or a guarded
// selection is what the instrument EXISTS to record (a mismatch is data,
// never an error), so no count, at any value, changes the exit.
func runUnitAnim(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("terraintool unitanim", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // the error is returned and reported once, by main
	assets := fs.String("assets", "", "path to the game asset root (overrides AGAINROM_ASSETS)")
	graphics := fs.String("graphics", "", "path to graphics.res (default <assets>/"+graphicsArchive+")")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%v\n%s", err, unitanimUsage)
	}

	archivePath, err := resolveArchive(*assets, *graphics)
	if err != nil {
		return err
	}
	// The container filesystem alone, as the census subcommand opens: the audit
	// is over the unit bundle and no tileset is built here.
	containers, err := game.OpenContainers(archivePath)
	if err != nil {
		return err
	}
	set, err := game.LoadUnits(containers)
	if err != nil {
		return err
	}

	// One line per class, in the audit's own ascending-id order, then the
	// summary. mismatched counts the classes whose sheet holds a frame count
	// other than the predicted total — either direction — and the two
	// selection sums close the ledger: in-range plus guarded is the whole
	// swept domain.
	rows := game.UnitAnimAudit(set)
	var mismatched, inRange, guarded int
	for _, r := range rows {
		fmt.Fprintf(out, "class %d: predicted %d, frames %d, in-range %d, guarded %d\n",
			r.ID, r.Predicted, r.Frames, r.InRange, r.Guarded)
		if r.Predicted != r.Frames {
			mismatched++
		}
		inRange += r.InRange
		guarded += r.Guarded
	}
	fmt.Fprintf(out, "unitanim: %d classes, %d mismatched, %d in-range, %d guarded\n",
		len(rows), mismatched, inRange, guarded)
	return nil
}

// checkStaticGround compares, for every placement, the ground point the
// SPRITE was drawn at against the one the MARKER geometry computes for the
// same cell, and refuses the run on the first disagreement.
//
// This is the measurement the whole layer was built to make possible, and it
// lives HERE, at the call site, for a reason the design states outright: an
// invariant enforced by the code it constrains cannot fail. The builder derives
// its anchors from the class and the frame through terrain.StaticAnchor; the
// marker derives its centre from the cell through terrain.MarkerAnchor; neither
// calls the other and neither reads the other's inputs. Moving this comparison
// into the builder would make the two derivations one and the disagreement
// unrepresentable — which is precisely the failure the separation exists to keep
// visible.
//
// p.Ground() is WHAT WAS DRAWN — TopLeft + Anchor, the two values the placement
// already carries — and not a re-derivation of it, or this would be the sprite
// side agreeing with itself.
//
// The comparison is NATIVE, at cellpx == terrain.CellSize, so no -scale enters
// it and -staticmarkers stays usable at any scale. The two negations are written
// out at this call site, where the builder was handed the un-negated height and
// origin: the two remain separate expressions over the same two sources, so a
// wrong sign, a wrong lookup or a wrong cell-to-world mapping on either side
// shows here instead of cancelling out.
//
// WHAT IT DISCRIMINATES IS BOUNDED and the bound is part of the contract, not a
// shortcoming to fix here: the anchor terms cancel algebraically, so this
// catches a wrong cell-to-world mapping or a wrong lift/origin lookup or sign,
// and it cannot catch a wrong CenterX/CenterY convention or a canvas-for-frame
// mix-up. Those are AC-9's — a human seeing art stand away from its own cross.
//
// height is the cell's own altitude lookup, nil on the flat geometry, and is the
// same source the builder was given. The failure is an internal-consistency one,
// deliberately outside the spec's error list: a run whose two derivations
// disagree would compose a wrong picture, and refusing to write it is honest.
func checkStaticGround(places []terrain.StaticPlacement, originY int, height func(col, row int) int) error {
	for _, p := range places {
		lift := 0
		if height != nil {
			lift = height(p.Cell.X, p.Cell.Y)
		}
		x, y := terrain.MarkerAnchor(p.Cell.X, p.Cell.Y, terrain.CellSize, -originY, -lift)
		if g := p.Ground(); g.X != x || g.Y != y {
			return fmt.Errorf("cell (%d,%d): the sprite stands on (%d,%d) but the marker geometry puts that cell's ground point at (%d,%d)",
				p.Cell.X, p.Cell.Y, g.X, g.Y, x, y)
		}
	}
	return nil
}

// staticCells is the anchor-cell list the third marker pass takes: one cell per
// placement, in placement order.
//
// It carries the list's CENSUS and none of its geometry — no top-left, no
// frame size, no class field — because the marker's position must come
// from the cell alone. Which cells are marked is exactly which cells
// resolved to a drawable frame, and that is the only thing the art tells the
// cross.
func staticCells(places []terrain.StaticPlacement) []image.Point {
	cells := make([]image.Point, 0, len(places))
	for _, p := range places {
		cells = append(cells, p.Cell)
	}
	return cells
}

// anchorCells converts the map's placed objects to the plain integer cells the
// render tier's overlay takes. Doing the conversion here is what keeps the map
// format out of the render tier: terrain.AnchorCell sees two integers, never an
// alm type. An anchor outside the map is passed through unchanged and the
// overlay drops it.
func anchorCells(m *alm.Map) []image.Point {
	cells := make([]image.Point, 0, len(m.Objects))
	for _, o := range m.Objects {
		col, row := terrain.AnchorCell(o.X, o.Y)
		cells = append(cells, image.Point{X: col, Y: row})
	}
	return cells
}

// unitAnchorCells is anchorCells for the map's placed units. It is a twin rather
// than a generalisation because alm.Object and alm.Unit are distinct types and
// two short loops read better here than a type parameter plus a field
// constraint; the arithmetic the two share is terrain.AnchorCell, which is where
// having a single home actually matters.
func unitAnchorCells(m *alm.Map) []image.Point {
	cells := make([]image.Point, 0, len(m.Units))
	for _, u := range m.Units {
		col, row := terrain.AnchorCell(u.X, u.Y)
		cells = append(cells, image.Point{X: col, Y: row})
	}
	return cells
}

// resolveLight builds the relief light for a shaded render. It defaults to the
// documented daytime sun; -maplight seeds it from the map's stored fields (a
// viewer choice, not engine fidelity, per TERR-LIGHT-023); and -theta/-ambient/
// -range override individual fields on top of whichever base was chosen.
func resolveLight(m *alm.Map, mapLight bool, theta float64, ambient, rangeByte int) terrain.Light {
	light := terrain.DefaultDaytime
	if mapLight {
		light = terrain.LightFromFields(m.Angle, m.Meta.Word10, m.Meta.Word14)
	}
	if !math.IsNaN(theta) {
		light.Theta = theta
	}
	if ambient >= 0 {
		light.Ambient = uint8(ambient)
	}
	if rangeByte >= 0 {
		light.Range = uint8(rangeByte)
	}
	return light
}

// resolveArchive locates graphics.res. An explicit -graphics path wins; failing
// that the archive is looked up under the configured asset root, which comes
// from -assets or AGAINROM_ASSETS and is never hardcoded.
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

func mapOpener(assetsFlag string) func([]byte) (*alm.Map, error) {
	if root := game.ResolveAssetRoot(assetsFlag, os.Getenv("AGAINROM_ASSETS")); root != "" {
		if match, err := game.DetectBase(root); err == nil {
			return game.MapOpener(match.Profile.GameOf())
		}
	}
	return alm.Open
}

func writePNG(name string, img image.Image) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// runStructures is the STRUCTURE CORPUS CENSUS — AC-10's harness.
//
// It loads the structure bundle exactly as the game does, builds the map's
// placement list through the ONE builder both front-ends place from, and prints
// what that build counted. IT WALKS NO LIST TO RECOUNT ONE: every number below
// came out of the build beside the entries it counted, which is what makes a
// printed figure the figure that build produced rather than a second walk's
// opinion of it.
//
// The three observations are REPORTED AND NOT RECONCILED (SC-6). Which class ids
// carry VariableSize, how many placements carry an eight-byte extension, and on
// how many classes the registry rectangle differs from the definition table's are
// facts about one install; none of them is folded into a contract clause or into
// a default, and the third in particular is a question this contract does not
// answer — it reads the registry alone, so it cannot be wrong about its own
// rectangle and has nothing to say about the other.
//
// It writes no file and exits non-zero on any load failure, exactly as units
// does: a census over a broken install would be a wrong count, not a count of a
// broken install. The definition table is the one part that is OPTIONAL — without
// it the third observation is reported unavailable rather than as a zero, because
// "they agree everywhere" and "we did not look" are different findings.
func runStructures(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("terraintool structures", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	assets := fs.String("assets", "", "path to the game asset root (overrides AGAINROM_ASSETS)")
	graphics := fs.String("graphics", "", "path to graphics.res (default <assets>/"+graphicsArchive+")")
	mapPath := fs.String("map", "", "path to the .alm map to census")
	dataBin := fs.String("databin", "", "path to world.res, holding the definition table the third observation compares against (default <assets>/"+worldArchive+")")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%v\n%s", err, structuresUsage)
	}
	if *mapPath == "" {
		return fmt.Errorf("-map is required\n%s", structuresUsage)
	}

	archivePath, err := resolveArchive(*assets, *graphics)
	if err != nil {
		return err
	}
	containers, err := game.OpenContainers(archivePath)
	if err != nil {
		return err
	}
	set, err := game.LoadStructures(containers)
	if err != nil {
		return err
	}

	mapData, err := os.ReadFile(*mapPath)
	if err != nil {
		return err
	}
	m, err := mapOpener(*assets)(mapData)
	if err != nil {
		return fmt.Errorf("decode %s: %w", *mapPath, err)
	}

	// The census is geometry-independent — which placements resolve is a question
	// about keys and loaded classes alone — so the flat build answers it and no
	// projection is made.
	_, counts, _ := terrain.StructurePlacements(terrain.Grid{
		Width: m.Width, Height: m.Height,
		Structures: game.StructureRecords(m.Objects),
	}, set, nil, 0)

	placements, cells := counts.CensusLines()
	fmt.Fprintln(out, placements)
	fmt.Fprintln(out, cells)

	// Observation 2 is the map's own: a record carries an extension only under
	// the key 0x21, and the reader attaches one only then, so its presence is the
	// whole test.
	extensions := 0
	for _, o := range m.Objects {
		if len(o.Ext) > 0 {
			extensions++
		}
	}

	mismatch := structureRectangleReport(*assets, *dataBin, set)
	fmt.Fprintf(out, "observations: variable-size ids %v, extensions %d, %s\n",
		set.VariableSizeIDs(), extensions, mismatch)
	return nil
}

// structureRectangleReport is AC-10's third observation: on how many classes the
// REGISTRY rectangle this story draws and the definition table's own sizeX/sizeY
// disagree.
//
// The two are INDEPENDENTLY SOURCED and nothing at this pin asserts that they
// agree: the art rectangle is the registry's TileWidth x TileHeight and the block
// footprint is the table's, and a story that reads only the registry cannot be
// wrong about its own. So this counts and reports, and reconciles nothing.
//
// It asks the table one class at a time through a PROBE MAP — one placement per
// key, each anchored at a column that IS its key — because the footprint resolver
// answers per placement and drops the keys it cannot resolve, so the anchor is
// what carries the identity through. Nothing about the probe reaches the census
// above it.
//
// A table that will not open is reported as unavailable rather than as zero:
// "they agree everywhere" and "we did not look" are different findings, and the
// second must not read as the first.
func structureRectangleReport(assetsFlag, dataBinFlag string, set *terrain.StructureSet) string {
	path, err := resolveWorldArchive(assetsFlag, dataBinFlag)
	if err != nil {
		return fmt.Sprintf("rectangle mismatches unavailable (%v)", err)
	}
	containers, err := game.OpenContainers(path)
	if err != nil {
		return fmt.Sprintf("rectangle mismatches unavailable (%v)", err)
	}
	rects, err := game.StructureTableRectangles(containers)
	if err != nil {
		return fmt.Sprintf("rectangle mismatches unavailable (%v)", err)
	}

	compared, differ := 0, 0
	var ids []int
	for k := 1; k <= 0xff; k++ {
		c := set.Classes[byte(k)]
		r, ok := rects[k]
		if c == nil || !ok || c.TileWidth <= 0 || c.TileHeight <= 0 {
			continue
		}
		compared++
		if c.TileWidth != r[0] || c.TileHeight != r[1] {
			differ++
			ids = append(ids, k)
		}
	}
	return fmt.Sprintf("rectangle mismatches %d of %d %v", differ, compared, ids)
}

// worldArchive is where the definition table lives under an asset root. It is
// spelt HERE, in the tool that opens it, exactly as the standalone viewer spells
// its own: an address prefix belongs to the tier that opens the archive, and a
// constant shared between two commands would have to live below both.
const worldArchive = "world.res"

// resolveWorldArchive is resolveArchive's sibling for the definition table: the
// -databin path when one was given, and <assets>/world.res otherwise.
func resolveWorldArchive(assetsFlag, dataBinFlag string) (string, error) {
	if dataBinFlag != "" {
		return dataBinFlag, nil
	}
	root := game.ResolveAssetRoot(assetsFlag, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		return "", fmt.Errorf("no definition table; pass -assets, set AGAINROM_ASSETS, or give -databin")
	}
	return filepath.Join(root, worldArchive), nil
}

// runTiers is the tiers subcommand — 0057's corpus instrument and its
// still.
//
// The two flags are render's own pair, resolved by the same helper; there is no
// -map because which colours a class ships is a fact about the registry and the
// archive, not about any map's placements. The census reads the bundle
// pkg/game's own loader produced and derives nothing of its own: the declared
// tier count IS the length of a class's per-tier slices and a fallback IS an
// empty one, so a number printed here cannot drift from the slices it describes.
//
// ONLY A LOAD FAILURE IS AN ERROR — an unreadable archive or an unreadable or
// unparseable unit registry — and a failed run prints nothing: the census is a
// recorded measurement, and lines over a half-loaded install would be wrong
// figures that look like right ones. A fallback is what the instrument EXISTS to
// record, so no count, at any value, changes the exit.
//
// The still is written only when BOTH -class and -out are given. Neither alone
// is enough and neither alone is an error: the census is the subcommand's own
// output, and the picture is a second thing it can be asked for.
func runTiers(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("terraintool tiers", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // the error is returned and reported once, by main
	assets := fs.String("assets", "", "path to the game asset root (overrides AGAINROM_ASSETS)")
	graphics := fs.String("graphics", "", "path to graphics.res (default <assets>/"+graphicsArchive+")")
	class := fs.Int("class", -1, "the unit class ID to render the tier still of; with -out")
	outPath := fs.String("out", "", "path to write the tier still to; with -class")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%v\n%s", err, tiersUsage)
	}

	archivePath, err := resolveArchive(*assets, *graphics)
	if err != nil {
		return err
	}
	containers, err := game.OpenContainers(archivePath)
	if err != nil {
		return err
	}
	set, err := game.LoadUnits(containers)
	if err != nil {
		return err
	}

	// Ascending ID, the unitanim audit's own order, so two runs over one install
	// print the same lines and a Go map's range order reaches no output.
	ids := make([]int, 0, len(set.Classes))
	for id := range set.Classes {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)

	var withTiers, declared, loaded, fell, sheetOwn int
	for _, id := range ids {
		c := set.Classes[int32(id)]
		n := len(c.Tiers)
		if n == 0 {
			continue
		}
		withTiers++
		declared += n
		own := 0
		for _, s := range c.Tiers {
			if len(s) > 0 {
				own++
			}
		}
		loaded += own
		fell += n - own
		// Whether tier 1's table EQUALS the sheet's own, read as slice identity:
		// the loader hands the base slice back for an equal table and builds a
		// new one otherwise, so identity here is a byte comparison of the two
		// tables performed at load. Reported per class because it is the one
		// claim about the shipped corpus a census can make and a picture cannot.
		first := "recoloured"
		if sameFrameSlice(c.Tiers[0], c.Frames) {
			first = "the sheet's own"
			sheetOwn++
		}
		fmt.Fprintf(out, "class %d %q: %d tier(s), %d loaded, %d fell back, %d frames, tier 1 %s\n",
			id, c.Name, n, own, n-own, len(c.Frames), first)
	}
	fmt.Fprintf(out, "tiers: %d of %d classes carry tiers, %d declared, %d loaded, %d fell back, %d take the sheet's own table at tier 1\n",
		withTiers, len(ids), declared, loaded, fell, sheetOwn)

	if *class < 0 || *outPath == "" {
		return nil
	}
	return writeTierStill(set, int32(*class), *outPath)
}

// sameFrameSlice reports whether two frame slices are the SAME slice — same
// backing array, same length — which is how the census reads "this tier's
// colour table equals the sheet's own": the loader hands the base slice back
// unchanged for an equal table and builds a fresh one for any other, so the
// comparison it performed at load is legible here without a second walk of 256
// entries that could disagree with it.
func sameFrameSlice(a, b []*terrain.StaticFrame) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}
	return &a[0] == &b[0]
}

// tierStillGutter is the transparent margin between two panels of the still, in
// native pixels: wide enough that two panels never touch and narrow enough that
// the eye compares them side by side.
const tierStillGutter = 8

// writeTierStill composes one class's tiers into a PNG, left to right in
// ascending tier order.
//
// EVERY PANEL DRAWS THE SAME FRAME INDEX — the class's own idle selection at
// octant 0 and tick 0, chosen once off the base slice and used for every tier —
// so the panels differ in COLOUR ALONE and the picture answers exactly the
// question it is asked. A per-tier re-selection would be the same index by
// construction (a tier changes no descriptor and no count) and would leave the
// reader unable to know that.
//
// The pixels come through the render tier's own lit blit, at the same daytime
// row the window builds its textures at, so the still is the game's pixels
// rather than a second palette walk of this tool's. Nothing here reads a palette
// entry.
//
// The field is opaque mid-grey rather than transparent: the picture is looked at
// in whatever viewer the owner has, and a sprite over a transparent field reads
// differently in a light one and a dark one.
func writeTierStill(set *terrain.UnitSet, id int32, path string) error {
	c := set.Classes[id]
	if c == nil {
		return fmt.Errorf("no unit class %d in this install's registry", id)
	}
	if len(c.Tiers) == 0 {
		return fmt.Errorf("class %d %q declares no tiers", id, c.Name)
	}
	if len(c.Frames) == 0 {
		return fmt.Errorf("class %d %q loaded no frames", id, c.Name)
	}

	index, _ := terrain.SelectUnitFrame(c.Anim, len(c.Frames), false, 0, 0, 0)
	row := terrain.SpriteRow(terrain.DefaultDaytime)
	tint := terrain.DefaultDaytime.SkyTint

	panels := make([]*image.RGBA, 0, len(c.Tiers))
	width, height := 0, 0
	for tier := 1; tier <= len(c.Tiers); tier++ {
		frames := c.TierFrames(tier)
		if index >= len(frames) {
			return fmt.Errorf("class %d %q: tier %d holds %d frame(s), short of index %d",
				id, c.Name, tier, len(frames), index)
		}
		p := frames[index].RGBALit(tint, row)
		panels = append(panels, p)
		width += p.Bounds().Dx() + tierStillGutter
		if h := p.Bounds().Dy(); h > height {
			height = h
		}
	}
	width += tierStillGutter

	canvas := image.NewRGBA(image.Rect(0, 0, width, height+2*tierStillGutter))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.RGBA{R: 0x3a, G: 0x3a, B: 0x3a, A: 0xff}},
		image.Point{}, draw.Src)
	x := tierStillGutter
	for _, p := range panels {
		// Bottom-aligned inside the band, so panels of unequal height stand on
		// one line and the eye compares colour rather than position.
		at := image.Rect(x, tierStillGutter+height-p.Bounds().Dy(),
			x+p.Bounds().Dx(), tierStillGutter+height)
		draw.Draw(canvas, at, p, p.Bounds().Min, draw.Over)
		x += p.Bounds().Dx() + tierStillGutter
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, canvas); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
