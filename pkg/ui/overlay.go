package ui

import (
	"image"
	"image/color"
	"math"
	"slices"

	"againrom/pkg/render/terrain"
)

// screenRect is one marker arm placed in the view, in screen pixels: the
// top-left corner and the on-screen size.
//
// The fields are float64 because that is what the camera works in. The
// narrowing to the float32 the vector drawer takes happens once, at the call
// site in Draw, so the transform itself stays exactly the camera's own
// arithmetic and can be compared against the camera contract without a rounding
// step in between.
type screenRect struct {
	X, Y, W, H float64
}

// overlayPass is one pass's contribution to a frame: the sprites it draws
// and/or the rectangles it fills in one colour, all already placed in the view.
//
// Draw does nothing but walk the passes in order — each pass's Sprites,
// then its Rects — so the slice overlayPasses returns IS the draw order.
// That is the point of the type: the terrain -> objects -> units order the
// composed viewers owe would otherwise be the order of two loops in Draw,
// which no test can observe. It matters more than it looks — the unit
// cross is a strict subset of the object cross at every scale, so passes in
// the wrong order do not render a degraded marker, they render no unit
// marker at all at a coincident cell (0009 R-4).
//
// EVERY PASS IS AN INSTRUMENT AGAIN. The type carried a Sprites field for
// four stories, for the one pass that was content rather than a diagnostic
// — the entity layer's sprite half — and that pass is now part of the
// content band drawArt paints before this slice exists. So the field is gone
// and Color applies to everything a pass can hold.
type overlayPass struct {
	HalfAdd bool
	Color   color.RGBA
	Rects   []screenRect
}

// SetObjects configures the placed-objects diagnostic overlay: whether to draw
// it and which anchor cells to mark.
//
// The cells are plain integer map cells, already shifted from the map's
// fixed-point anchors by the cmd tier — the viewer never sees a map type, which
// is what keeps the format tier out of the UI tier. The overlay is an
// independent toggle, drawn between the terrain and any later overlay.
func (v *Viewer) SetObjects(show bool, cells []image.Point) {
	v.showObjects = show
	v.objectCells = cells
	v.syncWorld()
}

// objectScreenRects places every marker arm in the current view, dropping the
// ones the view does not reach.
//
// Each arm comes from the render tier at the native CellSize pixels per cell and
// is already clipped to the map, then goes through the *same* transform Draw
// applies to a terrain tile: the top-left corner through the camera, the size
// scaled by the camera's zoom. Reading v.cam.Zoom is deliberate — the tile loop
// reads that same field, so the two transforms agree in every camera state,
// including one a caller produced by assigning Zoom directly without re-clamping.
//
// A marker therefore tracks its terrain cell through pan and zoom rather than a
// fixed pixel grid, and no independent pixel snapping is applied: the float
// coordinates go to the drawer unrounded. Rects lying wholly outside the view are
// culled here; the on-screen part of a partly visible rect is realised by the
// framebuffer clip when it is drawn, which draws the intersection without
// inventing an edge.
//
// Splitting this out of the draw path is what makes it testable without an
// engine context. It returns nil when the overlay is off or holds no cells.
func (v *Viewer) objectScreenRects() []screenRect {
	return v.overlayScreenRects(v.showObjects, v.objectCells, terrain.ObjectMarkerRects)
}

// SetUnits configures the placed-units diagnostic overlay: whether to draw it and
// which anchor cells to mark.
//
// It is an independent toggle from SetObjects — either overlay may be
// enabled without the other — and takes the same already-shifted integer
// map cells, so the viewer still never sees a map type.
func (v *Viewer) SetUnits(show bool, cells []image.Point) {
	v.showUnits = show
	v.unitCells = cells
	v.syncWorld()
}

// ObjectOverlay and UnitOverlay report what each diagnostic overlay is
// configured to draw: whether it is on, and how many anchor cells it holds. They
// read state and change none, mirroring Animation() — the cmd and game tiers own
// the decision to enable an overlay, so they need a way to see what they set
// without opening a window, and a front-end that silently draws no markers is
// otherwise indistinguishable from one that draws them (0010 AC-16).
//
// The cells themselves are deliberately not exposed: what they contain is the
// caller's own derivation, and handing back the slice would invite a caller to
// mutate the viewer's state through it.
func (v *Viewer) ObjectOverlay() (on bool, cells int) { return v.showObjects, len(v.objectCells) }

// UnitOverlay reports the unit overlay's toggle and cell count, exactly as
// ObjectOverlay does for the object overlay.
func (v *Viewer) UnitOverlay() (on bool, cells int) { return v.showUnits, len(v.unitCells) }

// StaticOverlay reports the static-object layer's two switches: whether the
// object sprites paint, and whether the layer's own diagnostic cross draws.
//
// It exists for the reason ObjectOverlay and UnitOverlay do, applied to a
// seam that needs it more. Both switches are set at construction and there
// is no setter, so a front-end that passed them the wrong way round — art
// where the cross was asked for, and the reverse — builds exactly the same
// placement lists, reports exactly the same Statics() count, and differs
// only in what a window paints. Statics() answers what was BUILT; this
// answers what will be DRAWN, and the two questions have different answers
// on purpose.
//
// No cell count comes back beside them: the cross's cells are not a caller's
// derivation here but the placement list's own, and Statics() already reports
// how many that is.
func (v *Viewer) StaticOverlay() (art, markers bool) { return v.showStaticArt, v.showStaticMarkers }

// SetBlocked configures the blocked-cell tint: whether to draw it and which
// cells to wash.
//
// It mirrors SetObjects and SetUnits field for field, and takes plain integer map
// cells for the same reason — the tier that owns a map, a table and the block
// derivation resolves which cells are closed before the value gets here, so this
// package still cannot spell a plane, a table or a format type. It is an
// independent toggle and re-syncs the camera's world for the reason the two
// setters above do: syncWorld is cheap and idempotent, not because this toggle
// can move the mode.
//
// WHAT THE CELLS MEAN IS THE CALLER'S. This viewer holds a block plane of its own
// in grid.Block and does NOT derive the tint from it — that plane is built with
// no definition table, so it cannot know a bridge from the water under it. Which
// is exactly why the cells arrive from outside rather than being read here.
func (v *Viewer) SetBlocked(show bool, cells []image.Point) {
	v.showBlocked = show
	v.blockedCells = cells
	v.syncWorld()
}

// BlockedOverlay reports the tint's toggle and cell count, exactly as
// ObjectOverlay does for the object overlay and for its reason: the tier that
// derived the cells needs to see that they arrived without opening a window.
func (v *Viewer) BlockedOverlay() (on bool, cells int) { return v.showBlocked, len(v.blockedCells) }

// GridMinCellPixels is how many screen pixels one map cell must span for the
// lattice to be drawn at all.
//
// A cell smaller than this cannot carry a readable outline: the line is one
// NATIVE pixel wide and the camera scales it, so at sixteen screen pixels a
// side it is already half a pixel and at the camera's minimum zoom a cell is
// four pixels across. What that draws is a wash over the whole map rather
// than a lattice — and it is also where the cost lives, because the band is
// the view measured in cells and grows as the inverse square of the zoom. One
// number bounds both, and it is the legibility one: the instrument stops when
// it stops informing, not when it starts costing.
const GridMinCellPixels = 16

// SetGrid turns the cell lattice on or off.
//
// It takes NO CELLS, unlike the four setters above: the lattice covers whatever
// the terrain pass covers this frame, which is a question the viewer answers and
// a caller cannot.
//
// It calls no syncWorld and needs none. The camera's world extent is a property
// of the terrain's own geometry; drawing an outline over cells that are already
// being drawn cannot move it, and the three overlays above call it only because
// it is cheap and idempotent.
func (v *Viewer) SetGrid(show bool) { v.showGrid = show }

// ToggleGrid flips the lattice and reports where it landed. It is the shape the
// key binding wants — a press edge has no value to pass — and it is here rather
// than in the front-end so that the standalone viewer and the map screen cannot
// offer different behaviour.
func (v *Viewer) ToggleGrid() bool {
	v.showGrid = !v.showGrid
	return v.showGrid
}

// GridOverlay reports whether the lattice is on, for the reason ObjectOverlay
// reports its own switch: a front-end that silently draws no outline is
// otherwise indistinguishable from one that draws them. There is no cell count
// beside it, because there is no cell list — the count is the band's, and the
// band is this frame's.
func (v *Viewer) GridOverlay() bool { return v.showGrid }

// gridSegment is one edge of the diagnostic cell lattice, in screen pixels:
// one line from one mesh corner this cell owns to the next.
type gridSegment struct {
	X0, Y0, X1, Y1 float64
}

// gridLineWidth is the lattice's stroke thickness in NATIVE pixels, scaled by
// the camera the same way the strip it replaces was — terrain.CellGridRects'
// own cellGridThickness, always 1 native pixel at the cellpx == CellSize this
// pass has always built at.
const gridLineWidth = 1

// TERR-SPR-039
func (v *Viewer) gridWorldQuad(col, row int) [4][2]float64 {
	if v.Mode() == ModeDisplaced {
		var out [4][2]float64
		for i, d := range [4][2]int{{0, 0}, {1, 0}, {1, 1}, {0, 1}} {
			x, y := v.proj.WorldCorner(col+d[0], row+d[1])
			out[i] = [2]float64{float64(x), float64(y)}
		}
		return out
	}
	cs := float64(terrain.CellSize)
	x0, y0 := float64(col)*cs, float64(row)*cs
	x1, y1 := x0+cs, y0+cs
	return [4][2]float64{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
}

// gridScreenSegments places the cell lattice in the current view: the four
// edges of one QUADRILATERAL per cell the terrain pass drew this frame, each
// corner through gridWorldQuad and the camera (revised by the item-2 hotfix
// above).
//
// THE BAND IS THE TERRAIN PASS'S OWN, walked rather than recomputed
// (forEachDrawnTile), so the lattice covers exactly the cells the terrain
// covered — unchanged by this hotfix.
//
// EACH CELL EMITS ITS OWN FOUR EDGES, so a shared edge between two drawn
// neighbours is issued twice, once by each cell — the same double coverage
// the old rectangle strips already carried at a shared boundary. The two
// issues carry the identical two endpoints by construction (both call
// WorldCorner at the shared (c,r) pair), which is the property a test can
// check that a rectangle-per-cell lattice could not have held.
//
// A SEGMENT WHOSE BOUNDING BOX MISSES THE VIEW IS NOT ISSUED, by the two axis
// tests segmentMisses already gives the path overlay, widened by the stroke's
// own on-screen width so a line whose centre lies just off-window but whose
// stroke still reaches it is kept — the same reasoning pathScreenSegments'
// own doc gives for not culling on endpoints alone.
//
// THERE IS A ZOOM FLOOR, and it is a legibility rule that happens to also be
// the cost bound. The band is the view measured in CELLS, and that grows as
// 1/zoom squared: at ZoomMin a cell is four screen pixels and a large window
// can cover a whole map's worth of quads, which draws a haze and not a
// lattice. Below GridMinCellPixels the instrument has stopped informing
// before it has started costing, so it stops drawing.
//
// The test is stated POSITIVELY so a non-finite zoom fails it and draws
// nothing, rather than passing a negated one and reaching the placement.
//
// It returns nil when the lattice is off, which is what keeps a viewer that
// was never told to draw it producing the frame it produced before this
// story.
func (v *Viewer) gridScreenSegments() []gridSegment {
	if !v.showGrid {
		return nil
	}
	if !(float64(terrain.CellSize)*v.cam.Zoom >= GridMinCellPixels) {
		return nil
	}
	strokePx := float64(gridLineWidth) * v.cam.Zoom
	loX, loY := -strokePx, -strokePx
	hiX := float64(v.cam.ViewW) + strokePx
	hiY := float64(v.cam.ViewH) + strokePx

	var out []gridSegment
	v.forEachDrawnTile(func(col, row int) {
		corners := v.gridWorldQuad(col, row)
		var screen [4][2]float64
		for i, c := range corners {
			screen[i][0], screen[i][1] = v.cam.WorldToScreen(c[0], c[1])
		}
		for i := 0; i < 4; i++ {
			j := (i + 1) % 4
			x0, y0 := screen[i][0], screen[i][1]
			x1, y1 := screen[j][0], screen[j][1]
			if segmentMisses(x0, x1, loX, hiX) || segmentMisses(y0, y1, loY, hiY) {
				continue
			}
			out = append(out, gridSegment{X0: x0, Y0: y0, X1: x1, Y1: y1})
		}
	})
	return out
}

// blockedScreenRects places one filled rectangle per tinted cell in the current
// view, through the very lift, camera and cull every glyph takes.
//
// IT DOES NOT RIDE overlayScreenRects, and that is the one deliberate difference
// from the three diagnostic overlays. That helper calls a glyph builder per cell,
// and every builder in the family returns a slice — one heap allocation per cell
// per frame. A record overlay's list is hundreds of cells; this one's is a
// plane's, thousands on a large map, and it is rebuilt every frame. So it calls
// terrain.CellFootprint, which returns the rectangle by value, and places it
// itself. The arithmetic is placeArm's own either way, so the tint and a marker
// on the same cell cannot come to disagree about where that cell is.
//
// It returns nil when the overlay is off or holds no cells.
func (v *Viewer) blockedScreenRects() []screenRect {
	if !v.showBlocked || len(v.blockedCells) == 0 {
		return nil
	}
	out := make([]screenRect, 0, len(v.blockedCells))
	for _, cell := range v.blockedCells {
		arm, ok := terrain.CellFootprint(cell.X, cell.Y, v.grid.Width, v.grid.Height, terrain.CellSize)
		if !ok {
			continue
		}
		if r, in := v.placeArm(cell, arm); in {
			out = append(out, r)
		}
	}
	return out
}

// unitScreenRects places every unit marker arm in the current view, by exactly
// the transform objectScreenRects uses; only the glyph differs.
func (v *Viewer) unitScreenRects() []screenRect {
	return v.overlayScreenRects(v.showUnits, v.unitCells, terrain.UnitMarkerRects)
}

// staticMarkerCells is the anchor-cell list the static-object glyph is stamped
// on: one cell per placement, in placement order.
//
// WHICH cells are marked is the whole of what the art contributes here, and
// it is the right contribution: a cross is owed to every cell that resolves
// to a drawable frame, and nothing else in the viewer knows which those are.
// WHERE each mark then goes is decided from the cell alone, by the marker
// geometry, reading no class field and no frame size — so the cross and
// the sprite remain two independent derivations of one ground point and a
// disagreement between them shows on screen as art standing away from its
// own cross.
//
// It reads the built list and copies out the one field it needs, so nothing
// downstream can reach a frame pointer through the marker path.
func staticMarkerCells(places []terrain.StaticPlacement) []image.Point {
	cells := make([]image.Point, len(places))
	for i, p := range places {
		cells[i] = p.Cell
	}
	return cells
}

// staticMarkerScreenRects places the static-object layer's own diagnostic cross,
// by exactly the transform objectScreenRects and unitScreenRects use; only the
// glyph and the source of the cells differ.
//
// The cells come from the placement list for the geometry the viewer is in
// right now, so the cross follows the same selection the art does and a cell
// that resolves to no drawable frame is marked in neither. The glyph is
// terrain.StaticMarkerRects — radius 3 by thickness 1, a strict subset of
// both shipped glyphs at native scale and above — which is what lets this
// pass run LAST of the three without covering a coincident object or unit
// cross.
//
// The switch is read HERE as well as inside the shared transform, and that is
// not a duplicated guard: the cell list has to be materialised out of the
// placements to be passed at all, so returning early is what keeps a viewer with
// the cross off from allocating one slice per frame over a map that may hold
// thousands of placements. The shared transform still owns the decision.
func (v *Viewer) staticMarkerScreenRects() []screenRect {
	if !v.showStaticMarkers {
		return nil
	}
	return v.overlayScreenRects(v.showStaticMarkers, staticMarkerCells(v.staticPlacements()), terrain.StaticMarkerRects)
}

type MapEntity struct {
	ID uint32
	// UnitNameIndex is the actor type id used only by the presentation's
	// unitname.txt lookup. It is carried from the resolved definition and is
	// not inferred from selection identity.
	UnitNameIndex int
	// SpellStateKnown distinguishes a production selection snapshot from a
	// standalone/custom viewer. These are real-ID membership and a resolved
	// per-object Cast predicate, not a second copy of original cell bit indices.
	SpellStateKnown bool
	KnownSpells     uint32
	CastCapable     bool
	Cell            image.Point
	Art             *terrain.UnitClass
	// DrawCategory belongs to the original class and corpse stage, independent
	// of a composed or substituted Art (ANIM-CATEGORY-084).
	DrawCategory terrain.UnitCategory
	Frame        *terrain.StaticFrame
	// Boundary is the spritesb frame paired with Frame, nil when the art has none.
	Boundary *terrain.StaticFrame
	Mirror   bool
	// Stone and Translucent are presentation facts already decided by the
	// world-owning tier. Stone selects a grayscale copy of the unit texture;
	// Translucent draws a detected or owner-visible invisible unit at half alpha.
	Stone, Translucent bool

	// Step is the cell delta this entity moved by on the most recent advance,
	// zero for one that did not move and for one that has taken no step yet.
	// The cell it came from is Cell.Sub(Step).
	Step image.Point

	// Speed is this entity's own speed input: the per-class number the decoded
	// rate law takes, carried across whole and interpreted nowhere here.
	//
	// It is a plain int, so this package still names no simulation type, and it
	// is the ONE scalar this story adds to the seam. It is carried and NOT used
	// to derive anything: how long a crossing takes is TransitSpan below, which
	// the simulation recorded when the unit took its step. Recomputing the span
	// here from this number instead was rejected outright — the rate law lives
	// behind an unexported function in the determinism package, and a second
	// copy of a movement law in the drawing tier is exactly the failure the
	// readout's own rule names, one field over.
	//
	// Zero is a mover with no speed, which is every entity built before a speed
	// existed and every one whose class states none; it crosses a cell in a
	// single tick and TransitSpan says so.
	Speed int

	// GroupSpeed is the GROUP RATE TERM this entity is carrying: the speed of
	// the group it was last ordered with, and zero for one carrying none.
	//
	// IT IS CARRIED BESIDE Speed AND NEVER COMPOSED WITH IT. A nonzero term
	// replaces the entity's own speed wherever a rate is computed, and that
	// composition is the simulation's own — it lives behind a single seam in the
	// package this tier may not name — so applying it here would be a second
	// copy of a movement rule in the drawing tier, exactly what the crossing
	// already refuses. Two raw numbers carry the rule nowhere; one composed
	// number would carry it here and hide the disagreement worth reading.
	//
	// And the disagreement IS the reason it crosses. The term is set by an order
	// and cleared only by another order or by being felled — arriving does not
	// clear it, and neither does the slow member of the group dying — so a unit
	// can walk at a pace nothing else on screen accounts for. Stating the two
	// numbers side by side is the only place in this tree that is visible.
	GroupSpeed int

	// Load is what this entity is carrying: the simulation's own Entity.Load,
	// carried across whole and interpreted nowhere here.
	//
	// It is a plain int, so this package still names no simulation type, and it
	// is carried rather than derived for Speed's own reason one field up: the
	// load law and the overload penalty both live behind the determinism wall,
	// and a second copy of either in the drawing tier is the failure that rule
	// names. Zero is an entity carrying nothing, which is every entity built
	// before a load existed.
	Load int

	// Shot is where THIS ENTITY'S PROJECTILE stands, in the same cell axes Cell
	// above is measured in, scaled by ShotScale — nil for an entity with none
	// in flight.
	//
	// IT CARRIES A POSITION AND NOTHING ELSE. Not a target id, not a reach,
	// not an attack phase — those decide WHETHER a shot exists and HOW FAR
	// ALONG it stands, and that rule stays on the simulation's side of this
	// seam. It is the same discipline Speed and GroupSpeed above already
	// state about the movement law: this package receives a value and derives
	// nothing further from it, so a mark can be drawn from this field without
	// this tier ever learning what an attack, a reach or a cadence is.
	//
	// Shot.X/ShotScale is the cell column it stands over and Shot.X%ShotScale
	// is how far across that cell it is, the fixed-point shape ShotScale
	// names below; Shot.Y is the same over the row.
	Shot *image.Point

	// Transit and TransitSpan are how far through the crossing of that step the
	// entity stands: how many ticks of it are still to run, and how many the
	// whole crossing is. A mover crosses a cell in a whole number of ticks at
	// its own rate, so the displacement between the two cells runs over the
	// CROSSING and not over one tick.
	//
	// A span of zero or one is a mover that crosses a cell in a single tick,
	// which is every mover before a rate existed and every one built without a
	// speed; the displacement then reduces to exactly what it was, over the tick
	// alone. A front-end that fills neither field draws what it drew.
	Transit, TransitSpan int
	// FinePosition is canonical simulation position inside Cell. It replaces
	// coarse crossing interpolation, including while paused and on first LOAD.
	// Terrain lift retains the viewer's current-cell anchoring policy.
	FinePosition bool
	FineX, FineY uint8

	// Route is the cells this entity still intends to walk, in the order it
	// will walk them, ending at the cell its order has settled on — nil for
	// one holding no order. It does NOT begin at the entity's own cell: where
	// the entity is standing is Cell, one field up, and a seam that carried it
	// twice would be the one place the two could disagree.
	Route []image.Point

	// Name is this entity's OWN class's name text, as the registry's bytes —
	// never a class substituted for drawing.
	//
	// It is a field of its own rather than a read through Art, and the two are
	// NOT the same class in every state. A unit that is not alive — downed as
	// well as dead — crosses with the corpse class's art substituted whole, so
	// a name taken off Art would be right for a live unit, right for the common
	// case of a class that names itself as its own dying class, and silently
	// wrong exactly where the two differ. Filling it on the far side, from the
	// class the entity's own id resolves to, is what puts that substitution out
	// of the name's reach by construction rather than by a rule.
	//
	// The empty string is an entity whose id names no class, and a class that
	// carries no name text — one answer for both, and the same "resolved to
	// nothing" Art already spells as nil. This package still names no
	// simulation, format or data type: it is a builtin, carried across.
	Name string

	// Owner is the ROSTER SLOT this entity was placed under, carried across
	// whole and interpreted nowhere here.
	//
	// It is a plain uint32, the field's own width, so this package still names
	// no simulation type; and it is READ IN EXACTLY ONE PLACE — the arming
	// gate, which compares it against the local participant the far side
	// pushes. Nothing draws it, nothing selects on it, and no order carries it.
	//
	// ITS ZERO IS NO OWNER, which is the far side's own convention: a roster
	// slot is 1-based, so zero is free to mean absent without a presence flag
	// beside it. An entity the map did not place carries that zero and is owned
	// by nobody, which is what every entity built before this story is.
	Owner uint32

	// Knowledge is the card level the local player holds for this unit, and
	// KnowledgeKnown says the far side stated one. An entity that states none draws
	// every card group.
	Knowledge      int
	KnowledgeKnown bool

	// Hostile is whether the LOCAL PARTICIPANT treats this entity's owner as an
	// enemy: the simulation's own relation, `sim.Relations.Hostile(SelfSlot,
	// Owner)`, read once per tick and carried across whole (1031 B3).
	//
	// THIS IS THE PROJECTION 1031's HOSTILITY TEST READS, and it is the
	// simulation's own directional relation and not a same-owner test: two
	// units under different owners can both be non-hostile, and the relation
	// is not symmetric (`AI-DIPLO-005`; `pkg/sim/relations.go`'s own header).
	// `UNIT-VPLAYER-021`/`UNIT-VISBIT-044` decode the original's own hostility
	// bit as a VIEW-SIDE row on a per-player record, filled once and matching
	// the session relation matrix on this one bit at Medium confidence — this
	// build has no view-side row of its own and reads the session relation
	// directly, which is the more current answer wherever a script changes a
	// relation mid-mission (`DIV-259`).
	//
	// ITS ZERO VALUE IS "NOT HOSTILE", which is every entity built before this
	// story and every entity with no owner (`relationIndex`'s own rule: slot 0
	// carries no relation cell at all).
	Hostile bool

	// PlayerCharacter is whether this entity is one of the local participant's
	// own CHARACTERS rather than a hired unit or a map-placed actor (story
	// 1034; `PARTY-FLAG-003`, Medium).
	//
	// IT IS THE MISSION CURSOR'S OWN GATE and is carried for that alone here.
	// `AI-CURSOR-242` and `AI-CURSOR-209` both read the selected object's
	// `CUnit+0x18c` bit `0x1`: the first as one of the three ANDed terms of the
	// `pickup` gate, the second as half of the `town` composite.
	// `PARTY-FLAG-003` reads that bit as the player-character flag and
	// explicitly refutes the earlier "hero" gloss for it, so the counterpart
	// here is the world's own guarded set — every party member who is a player
	// character and not a mercenary (`pkg/game/world.go`'s guardedEntities) —
	// and not the single starting hero.
	//
	// ITS ZERO VALUE IS "NOT A PLAYER CHARACTER", which is every entity built
	// before this story and every actor a map places.
	PlayerCharacter bool

	// OriginalPanel is the original information panel's own bounded actor
	// projection. It is carried independently of Combat and Char because the
	// two captions it gates do not mean Mage or AlwaysHits.
	OriginalPanel OriginalPanelActor

	// Life is which of the three states the simulation says this entity is in,
	// as one of the three constants below.
	Life uint8
	// CorpseStage carries the actor's independently stored decay stage.
	CorpseStage uint8
	// HP and MaxHP are the entity's health and the maximum it was built with.
	// They are signed and are NOT clamped: a dead entity's health is negative,
	// and how far below zero it has gone is a fact this seam carries whole.
	HP, MaxHP int
	// Untargetable is the simulation's ordinary-combat verdict. Its zero value
	// keeps hand-built live snapshots targetable; Control Spirit explicitly
	// bypasses it because that spell owns the corpse-specific target rule.
	Untargetable bool
	// Selectable widens selection past Life's ordinary live/downed population.
	// A finishable body at -1 through -9 is still LifeDead for its corpse art,
	// depth and bars, but the owner requires it to remain clickable. The zero
	// value changes nothing: alive/downed entries are selectable through Life,
	// while a terminal corpse remains a miss.
	Selectable bool
	// Restorable is the simulation's restorative admission: ordinary Heal may
	// still target this unit, which for a fallen one means raise it. A fallen
	// entry keeps its minimap mark only while it holds (DIV-1455). A LifeAlive
	// entry never reads it, and its zero value gives a hand-built fallen entry
	// no mark.
	Restorable bool
	// DamageJolt is a presentation-only offset for the unit sprite on this
	// frame. It never moves the cell footprint, selection rim, bars, marks or
	// depth anchor, so a struck fallen body twitches without its click target
	// or gameplay position moving with the picture (DIV-488).
	DamageJolt image.Point

	// Mana and MaxMana are the entity's mana pool and the maximum it was built
	// with, beside the health pair in the same shape: a maximum of zero is a
	// unit with no mana system, and no value of Mana is constrained against it.
	Mana, MaxMana int
	// TokenSize is the side of the actor's square footprint. Zero is one, on
	// the simulation's own normalisation rule.
	TokenSize int

	// SpellFX is how many ticks of a spell effect mark this entity is still
	// carrying, and SpellFXSchool the elemental school of the spell that set
	// it. Zero on both is a unit no spell has just touched, which is every
	// entity on every frame this build drew before this story.
	//
	// THEY ARE PLAIN ints, carried across and interpreted nowhere but in the
	// pass this package builds from them: how a mark is set, how long it
	// lives and which row it names are the simulation's, and the school
	// arrives already resolved off the world's own table so this package
	// never looks a spell up. Speed and GroupSpeed's own rule, one field
	// group over.
	SpellFX, SpellFXSchool int

	// Marks is the DECODED effect-mark record set this entity carries on this
	// frame (1002; MAGIC-MARK-059, MAGIC-MARK-060), each record beside the
	// sheet its record index named. It is a per-frame value and not state: the
	// original does not store the set either — it re-derives it every rebuild
	// from the actor's kind-and-countdown list — so an empty slice is an actor
	// carrying no marking effect, which is every entity of every frame this
	// build drew before this story.
	//
	// It stands BESIDE SpellFX rather than replacing it. SpellFX is 0154's
	// authored school ring, drawn for any effect at all; these are the shipped
	// sprites the original itself draws, for the ten kinds that have one.
	Marks []UnitMark

	// Combat is the eight numbers a blow reads about this entity and Char the
	// character they were derived from, both for the unit information panel to
	// state.
	//
	// THEY HAVE DIFFERENT LIFETIMES AND CROSS ON ONE SEAM ANYWAY, which is the
	// decision worth reading here rather than the fields. Combat is simulation
	// state, read off the entity on the tick it is stated, and it moves whenever
	// a rule says so. Char is a per-entity CONSTANT the loader fixed when the
	// map opened — no part of it is on an entity, in the byte form or in a
	// digest — resolved once on the far side and pushed through here as the
	// resolved value.
	//
	// A constant crossing a per-tick seam is what Name, Owner and Speed above
	// already do, each for its own stated reason, and the alternative was worse
	// in three ways at once: a second push keyed by entity id would put a lookup
	// on the far side of a seam whose whole rule is that this package receives
	// values and derives none; it would need its own replace-or-accumulate rule
	// and its own cache-invalidation rule; and it would be correct only while
	// every map open produced a fresh viewer, which nothing here states or
	// enforces — under reuse it would attribute one map's hero to whichever of
	// the next map's entities inherited his id, and compose a well-formed panel
	// about the wrong unit.
	//
	// Each carries its own "was told" flag, because the zero value of either is
	// exactly what a real unit could carry: a caller that filled in neither has
	// said nothing about this entity, which is not the same as saying it fights
	// for nothing. Every entry this tree builds from a running world fills
	// Combat; Char is filled for every unit whose placement reached a
	// definition entry, and for the party — it used to be the party alone,
	// which is the sentence this one replaces.
	Combat UnitCombat
	Char   UnitCharacter

	// Sound is this entity's OWN class's sound-slot array, carried whole and
	// interpreted nowhere here (0126 spec Terms "Sound slots of a class";
	// plan T2), in the shape Speed and GroupSpeed above already state their
	// reason in: a plain []int, so this package still names no other tier's
	// type, and this tier INDEXES it — by the fixed positions spec.md's Terms
	// table names — and never assigns those positions a meaning of its own.
	//
	// Each element is an ARCHIVE SLOT NUMBER, not a sound: it crosses this
	// seam as an integer exactly as Speed does, and it is sound.go's
	// SoundBank, on the far side of that package's own seam into pkg/game,
	// that turns a slot number into a playable sample. Zero is silence and
	// not an error (spec Terms) — three shipped classes carry a zero swing —
	// and so is an index this slice does not reach: sound.go's classSlot
	// folds "too short" and "zero at the index" into the one answer they
	// already share, rather than this field needing a presence flag beside
	// it.
	//
	// A NIL SLICE IS EVERY ENTITY BUILT BEFORE THIS STORY, and it answers
	// exactly like a present-but-short one: silence at every index, so no
	// caller that has never heard of this field changes what it draws or
	// what it plays.
	Sound []int

	// Voice is the human voice bank this entity's wounds and fall play from,
	// a directory of the sound archive such as "mf_hero", or empty for an
	// entity whose drawn class's Sound array voices them (ANIM-094). The game
	// tier chooses it; sound.go names the recording.
	Voice string
}

// SpellLightCell is one live spell-light input as presentation receives it.
// Terrain and Sprite are independent absolute brightness scales on their
// respective decoded ladders, not factors relative to the current time of day.
// Terrain is written to the cell's four shared vertices; Sprite remains the
// cell-local unit-body input. Wall of Fire deliberately supplies Sprite alone.
type SpellLightCell struct {
	Cell            image.Point
	Terrain, Sprite float32
}

type spellLightScale struct{ sprite float32 }

// SetSpellLighting replaces the derived spell-light plane. The input is copied
// so expiry restoration is simply the next push omitting a cell.
func (v *Viewer) SetSpellLighting(cells []SpellLightCell) {
	if len(cells) == 0 {
		v.spellLighting = nil
		v.spellTerrainLighting = nil
		return
	}
	next := make(map[image.Point]spellLightScale, len(cells))
	vertices := make(map[image.Point]float32, len(cells)*4)
	for _, c := range cells {
		if c.Sprite > 0 {
			if previous, ok := next[c.Cell]; !ok || c.Sprite > previous.sprite {
				next[c.Cell] = spellLightScale{sprite: c.Sprite}
			}
		}
		if c.Terrain > 0 {
			vertices[c.Cell] = c.Terrain
			vertices[c.Cell.Add(image.Pt(1, 0))] = c.Terrain
			vertices[c.Cell.Add(image.Pt(0, 1))] = c.Terrain
			vertices[c.Cell.Add(image.Pt(1, 1))] = c.Terrain
		}
	}
	v.spellLighting = next
	v.spellTerrainLighting = vertices
}

// spellTerrainScale reads one vertex's absolute terrain-ladder override from
// the SPELL plane only. It REPLACES the vertex's own relief shading at the
// caller (cornerScales), which is what lets Darkness darken: its own
// darknessBrightness is 0.5, below any shading a vertex can otherwise hold.
//
// The structure-light mask is deliberately NOT merged in here. It composes by
// MAX at the caller instead, against the vertex's real shading — see
// structureTerrainScale below.
func (v *Viewer) spellTerrainScale(vertex image.Point) (float32, bool) {
	if v.graphics.DisableLighting {
		return 0, false
	}
	value, ok := v.spellTerrainLighting[vertex]
	return value, ok
}

// structureTerrainScale reads one vertex's structure-light mask (DIV-1313
// amended; towerglow.go's refreshStructureLighting). Its caller composes it by
// MAX rather than by replacement, for the reason refreshStructureLighting's own
// falloff base states: this plane's values are built over the FLAT-GROUND
// daytime reference (ShadeScale(46)), and relief lights a sloped vertex up to
// ShadeScale(36) = 1.875, above that reference. Replacing would dim exactly
// those vertices at the mask's rim. A max cannot, whatever the reference is
// worth — a light only ever adds.
func (v *Viewer) structureTerrainScale(vertex image.Point) (float32, bool) {
	if v.graphics.DisableLighting {
		return 0, false
	}
	value, ok := v.structureTerrainLighting[vertex]
	return value, ok
}

// spellSpriteFactor reads one cell's sprite gain, over the SAME normalized
// scale, composed by the same MAX rule as spellTerrainScale above (DIV-1313
// amended).
//
// A cell with a stamped light-stamp corner takes the merged stamp level in
// place of the Light/Darkness value, as the original's stamp sweep overwrites
// the cell-bit stage (MAGIC-273). That merge reads no Lighting option, so a
// Lightning bolt lights units with Dynamic lighting off.
func (v *Viewer) spellSpriteFactor(cell image.Point) float32 {
	// SpriteRow's decoded gain equals this terrain-ladder expression.
	normal := terrain.ShadeScale(4*v.spriteRow() + 32)
	row, stamped := v.lightStampSpriteRow(cell)
	if v.graphics.DisableLighting {
		if stamped {
			return terrain.ShadeScale(4*row+32) / normal
		}
		return 1
	}
	level := float32(0)
	if value, ok := v.spellLighting[cell]; ok {
		level = value.sprite
	}
	if stamped {
		level = terrain.ShadeScale(4*row + 32)
	}
	if structure, ok := v.structureLighting[cell]; ok && structure > level {
		level = structure
	}
	if level > 0 {
		return level / normal
	}
	return 1
}

// ShotScale is the fixed-point scale MapEntity.Shot's coordinates are
// carried in: one cell is ShotScale units on each axis.
const ShotScale = 256

// The three life states as they cross the seam.
//
// LifeAlive is the ZERO VALUE, so a MapEntity built by a caller that names no
// life — every one written before this story — is alive, which is what those
// callers meant when there was no other state to be in.
const (
	LifeAlive  uint8 = 0
	LifeDowned uint8 = 1
	LifeDead   uint8 = 2
)

// SetEntities hands the viewer the open world's entities: one MapEntity per
// entity, in the order the world holds them, as of the most recent step. It
// REPLACES 0020's SetEntityCells and keeps that setter's whole contract,
// with art beside each cell.
//
// It is called EVERY TICK the map screen runs, where the two overlay setters are
// called once when a map opens, and the difference is why it deliberately calls
// no syncWorld: the camera's world extent is a property of the terrain's own
// geometry, and which cells happen to hold entities cannot move it. Re-clamping
// the camera from a per-tick setter would be a per-tick answer to a question
// whose answer was fixed at construction.
//
// The slice is ADOPTED, not copied — the caller is the tier that rebuilt it from
// this tick's entity state and does not keep it — and it replaces whatever was
// held before rather than accumulating, so a world that lost an entity draws one
// item fewer on the very next frame.
//
// There is no show parameter: passing entities is what turns the two entity
// passes on, passing none is what turns them off.
func (v *Viewer) SetEntities(entities []MapEntity) {
	v.entities = entities
	for _, entity := range entities {
		v.soundMessages = append(v.soundMessages, soundMessage{entity: entity})
	}
}

// EntityMarkers reports how many entities the viewer holds, mirroring
// ObjectOverlay/UnitOverlay's shape and reason: the tier that owns the world
// needs to see that its entities arrived without opening a window, and a viewer
// that silently draws no entity is otherwise indistinguishable from one drawing
// them.
//
// There is no on flag to report beside the count, because there is no flag: a
// count of zero IS "no entity pass this frame". The entities themselves are not
// handed back, for the reason ObjectOverlay gives — they are the caller's own
// derivation, and returning the slice would invite a write through it.
func (v *Viewer) EntityMarkers() (entities int) { return len(v.entities) }

// SackMarkers reports how many ground sacks the viewer holds and how many
// frames its sack sheet carries, for EntityMarkers' reason and one sharper one
// (docs/0111-sacks: plan R-4).
//
// Neither list is handed back, for EntityMarkers' reason: both are the caller's
// own derivation and returning either would invite a write through it.
func (v *Viewer) SackMarkers() (sacks, frames int) { return len(v.sacks), len(v.sackFrames) }

// entityShift is ONE entity's world-pixel displacement for THIS frame: how
// far back toward the cell it came from it is drawn, given how much of the
// current tick is still to run.
//
//	span  = max(TransitSpan, 1)
//	left  = Transit*period + (period - clamp(elapsed, 0, period))
//	total = span * period
//	x = -Step.X * CellSize * left / total
//	y = -Step.Y * CellSize * left / total, less the relief term below
//
// So at the CROSSING's start the entity is drawn on the cell it LEFT and at its
// end on the cell it entered, proportionally between, in whole render pixels —
// the render's own cell, so the camera's zoom multiplies the result and never
// the inputs. Every glyph that entity draws adds this same value, which is what
// makes a unit and everything on it move as one body: there is one function, and
// a sprite that could drift from its own health bar would have to be two.
//
// THE DENOMINATOR IS THE CROSSING AND NOT THE TICK. A mover crosses a cell
// in a whole number of ticks at its own rate, and over that crossing it
// stands on one cell while the drawing carries it there; a fraction taken
// over one tick would slide it home inside the first tick and freeze it for
// the rest. At a span of zero or one the two are the same arithmetic, which
// is why every mover with no rate — and every front-end that fills neither
// field — draws exactly what it drew.
//
// The arithmetic is done in 64 bits explicitly. The longest crossing the rate
// law can produce is 256 ticks and a period may be a whole second, so the
// numerator passes what a 32-bit int would hold, on a platform whose int is 32
// bits wide.
//
// IN DISPLACED MODE THE RELIEF RIDES THE SAME VECTOR. The shared transform lifts
// a glyph by the ENTERED cell's own height, so the vertical term carries the
// difference between the left cell's lift and the entered cell's, scaled by the
// same fraction: the drawn height then runs between exactly the two values the
// glyphs on those two cells are lifted by, rather than jumping at the boundary.
// The left cell is Cell-Step and the lift lookup clamps its own indices, so it
// is safe at any value; flat mode adds nothing, because the transform lifts by
// nothing there.
//
// THREE CLAMPS, AND EACH IS A CASE RATHER THAN A GUARD. A zero step is an
// entity that did not move — standing, blocked, downed, dead — and
// returns before anything is computed. A non-positive period is a viewer
// that was never told where it stands inside a tick, which is every
// front-end written before this story and the standalone one still, and
// draws today's picture. And an elapsed outside its period is the state a
// re-rate leaves behind when the new period is shorter than the remainder
// already accumulated; unclamped it would throw an entity past the cell it
// entered or behind the one it left.
func (v *Viewer) entityShift(e MapEntity) image.Point {
	if e.FinePosition {
		return image.Pt((int(e.FineX)-128)*terrain.CellSize/256, (int(e.FineY)-128)*terrain.CellSize/256)
	}
	period := v.phasePeriodUS
	if e.Step == (image.Point{}) || period <= 0 {
		return image.Point{}
	}
	elapsed := v.phaseUS
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > period {
		elapsed = period
	}

	// A span of zero or one is a one-tick crossing, and an owed count outside
	// its own span is a value no simulation here writes; both are brought inside
	// rather than refused, for the reason the elapsed clamp above is — this is a
	// draw path, and a frame is drawn from whatever it is handed.
	span := int64(e.TransitSpan)
	if span < 1 {
		span = 1
	}
	owed := int64(e.Transit)
	if owed < 0 {
		owed = 0
	}
	if owed > span-1 {
		owed = span - 1
	}
	left := owed*int64(period) + int64(period-elapsed)
	total := span * int64(period)

	shift := image.Point{
		X: int(int64(-e.Step.X*terrain.CellSize) * left / total),
		Y: int(int64(-e.Step.Y*terrain.CellSize) * left / total),
	}
	if v.Mode() == ModeDisplaced {
		from := e.Cell.Sub(e.Step)
		rise := v.proj.AnchorHeight(from.X, from.Y) - v.proj.AnchorHeight(e.Cell.X, e.Cell.Y)
		shift.Y -= int(int64(rise) * left / total)
	}
	return shift
}

// entityGlyphRects places one glyph family over a list of entities: the
// render tier's own arms for each entity's CELL, moved by that entity's own
// displacement, then through placeArm — the lift, the camera and the cull
// every glyph in the frame shares.
//
// It is a PER-ENTITY walk and not a cell list handed to overlayScreenRects,
// because a displacement is a property of the entity and not of the cell it
// stands on: two units on one cell in successive frames need not have arrived
// there the same way. The three diagnostic overlays keep the cell-list helper,
// which is why that helper is untouched — they have no entity, and converting
// their lists per frame would allocate over thousands of cells for a vector that
// is always zero.
//
// The shift is added BEFORE placeArm, so the cull runs on the rectangle actually
// drawn and the whole unit is culled or kept as one.
func (v *Viewer) entityGlyphRects(ents []MapEntity, rects func(col, row, cols, rows, cellpx int) []image.Rectangle) []screenRect {
	var out []screenRect
	for _, e := range ents {
		shift := v.entityShift(e)
		for _, arm := range rects(e.Cell.X, e.Cell.Y, v.grid.Width, v.grid.Height, terrain.CellSize) {
			if r, ok := v.placeArm(e.Cell, arm.Add(shift)); ok {
				out = append(out, r)
			}
		}
	}
	return out
}

// entityLayer splits the held entities into the two lists the entity passes
// draw: a sprite placement per entity whose art resolves, the ENTITY itself
// for every other in-map one.
//
// ONE WALK, IN SLICE ORDER — ascending entity id, the world's own — so each
// output list holds its survivors in ascending id, and sprite-or-square is
// total by construction: the single bounds test up front is the only way out
// of the walk, decided BEFORE either geometry, so an off-map entity joins
// neither list whether or not its art would resolve (spec: an off-map entity
// draws neither, as before).
//
// The sprite side takes the same two displaced terms the static-object build
// takes — lift = proj.AnchorHeight(cell) and originY = proj.MinV, both
// unnegated, UnitPlace's StaticAnchor performing the one negation — and 0
// and 0 flat. They are read PER CALL, not in the setter, because a placement
// depends on Mode(), live under SetFlat.
//
// THE SQUARE SIDE HANDS BACK ENTITIES AND NO LONGER BARE CELLS: the square
// is displaced with everything else that entity draws, and a cell cannot say
// by how much. The sprite side adds the same vector to its placement's
// TOP-LEFT, so the exact-rect cull downstream runs on the rectangle actually
// drawn and the placement's own ground point moves with its art. marked IS
// INDEX-PARALLEL TO sprites (1002), for the depth tie's own reason one
// screen down: this walk is the one place that knows both the placement and
// the entity it came from, and the sprite list drops entities, so an index
// into it cannot be mapped back to v.entities anywhere downstream. An entity
// whose art did not resolve draws no marks — it is drawn as a diagnostic
// square, and a mark is a statement about a unit's own sprite.
func (v *Viewer) entityLayer() (sprites []terrain.StaticPlacement, squares []MapEntity, marked []MapEntity) {
	displaced := v.Mode() == ModeDisplaced
	for _, e := range v.entities {
		c := e.Cell
		if c.X < 0 || c.Y < 0 || c.X >= v.grid.Width || c.Y >= v.grid.Height {
			continue
		}
		// THE FOG GATE: an owned unit is drawn whatever its cell answers, everyone
		// else's only while their cell is FogVisible right now. Gated HERE, before
		// the walk splits into sprites and squares, is what keeps a unit from
		// vanishing as a sprite and staying as a square (or as a shadow —
		// shadowDraws in shadow.go reads this same sprite list): both halves of an
		// unresolved entity's draw, and every other consumer of this list, see one
		// already-gated set rather than needing the check repeated.
		if !v.fogGateEntity(e.Owner, c.X, c.Y) {
			continue
		}
		lift, originY := 0, 0
		if displaced {
			lift = v.proj.AnchorHeight(c.X, c.Y)
			originY = v.proj.MinV
		}
		if p, ok := terrain.UnitPlace(c.X, c.Y, e.Art, e.Frame, e.Mirror, lift, originY); ok {
			// DamageJolt moves only the sprite. The ordinary crossing shift is
			// shared by every glyph; the jolt deliberately is not, so the body
			// twitches inside a stable click/selection footprint (DIV-488).
			p.TopLeft = p.TopLeft.Add(v.entityShift(e)).Add(e.DamageJolt)
			p.DepthTie = terrain.TieUnit
			if e.Life == LifeDead {
				p.DepthTie = terrain.TieCorpse
			}
			// THE DEPTH ROW is the LATER of the two rows a crossing spans: the
			// row of the cell the crossing began at, and the row of c, the
			// entity's current SIM cell (its destination while the crossing
			// runs; entityShift, just above, carries the interpolation). c
			// alone drew a unit stepping ONTO a sack or corpse's cell BEHIND
			// it for the southward half of a crossing. The origin cell alone
			// (T3's fix, 0151 defect 7) drew a unit stepping OFF a sack or
			// corpse's cell BEHIND it for the northward half. Comparing the
			// two rows and keeping the later one covers both directions, and
			// sideways and diagonal crossings fall out of the same
			// comparison.
			//
			// p.Cell is read by nothing else this placement reaches:
			// UnitShadowPlace (shadow.go) takes TopLeft and Anchor alone, and
			// the fog gate above already ran on c. DepthOrder's row merge
			// (rowAt, rowOrder — structures.go) is its only reader, so moving
			// it here moves the merge's answer and nothing this placement
			// draws.
			//
			// e.Step STAYS NONZERO FOR THE WHOLE CROSSING, not one tick of
			// it: Step is the delta from the cell the crossing began at
			// (MapEntity's own doc, above), and pkg/game's mw.prev — the
			// memory that delta is read from — is left unwritten for every
			// tick an entity still owes a transit, so Step stays the one
			// nonzero vector across a multi-tick crossing and not just its
			// first tick (pkg/game/world.go, recordCells). The row
			// comparison below therefore holds for the whole crossing and
			// reverts to c alone the tick the crossing ends, which is the
			// same tick Step next reads zero.
			//
			// origin.Y > c.Y exactly when Step.Y is negative: the crossing
			// runs toward a smaller row, north. A southward or sideways step
			// leaves p.Cell at its default, c, which already names the later
			// row — south's later row is the destination, sideways' two rows
			// are equal — so only the northward half of the comparison ever
			// changes p.Cell.
			//
			// A corpse never carries a nonzero Step (a dead unit takes no
			// order), so this is inert for one and the corpse-under-a-moving-
			// unit case is already correct through rowOrder's ordinary sort:
			// both entities share one list, and only the mover's row moves.
			if origin := c.Sub(e.Step); origin.Y > c.Y {
				p.Cell.Y = origin.Y
			}
			sprites = append(sprites, p)
			marked = append(marked, e)
		} else {
			squares = append(squares, e)
		}
	}
	return sprites, squares, marked
}

// selectionScreenRects places every selected unit's mark in the current
// view: one rim per selected id the snapshot still holds, in ascending id,
// by exactly the lift, camera and cull every other glyph takes, and moved by
// its own entity's displacement.
//
// Riding the entity walk is the whole design. The rims take the shipped
// glyphs' own placement, the displaced mode's PER-CELL height lift, the view
// cull and the frame's displacement without a second copy of any of them, so
// each mark carries the very relief offset and the very shift the sprite
// under it carries and cannot drift from either — and a group spread
// across a slope is lifted cell by cell rather than by one offset for the
// set (AC-9, AC-6).
//
// IT READS THE SELECTION, IT DOES NOT TAKE ONE. The selection is the field
// command already stores, so the marks and the pick cannot disagree about
// which units are selected: there is one piece of state and both sides read
// it.
//
// ABSENCE IS FILTERED, NEVER PRUNED, and the filter is presentSelected —
// the SAME walk a left click emits its orders from, so "still there to act
// on" is one predicate and not two kept in step. An entity the world dropped
// is gone from the next snapshot while the front-end still remembers having
// selected it, and a mark standing on the cell it was last seen on would be
// a lie about a unit that no longer exists; but nothing is written back, so
// an id that left and returned is marked again and what is selected stays a
// function of the taps and releases alone.
//
// A UNIT THAT DIED WHILE SELECTED IS FILTERED BY THAT SAME PREDICATE, so it
// loses its mark on the very frame the world reports it dead. There is no second
// rule here to keep in step with the orders': one filter answers both, which is
// what makes "marked exactly when it would be ordered" a property of there being
// one function rather than of two that happen to agree.
//
// The walk is by ID, never by slice position. Which entry of the snapshot a
// selected unit occupies is not stable — the world's own order is by id and an
// entity that leaves shifts every later one — so an index remembered from the
// tick the selection was made would follow whichever unit slid into that slot.
//
// A selection nothing in the snapshot answers to yields no entities, and an
// off-map selected one contributes no geometry inside
// terrain.SelectionMarkerRects — so in every one of those cases the rects
// are empty and overlayPasses appends no pass at all. A SELECTED UNIT THAT
// MAY NOT BE SEEN LOSES ITS MARK TOO (hotfix), and this one is reachable in
// ordinary play: nothing scopes the selection to the local participant's own
// units — canArmAttack tests the owner to decide ARMING, never to decide
// selecting — so tapping an enemy to look at him and then watching him
// walk into the dark left a rim standing over a unit nothing else drew.
//
// THE GATE IS HERE AND NOT INSIDE presentSelected. That filter is the ONE
// predicate the marks and the orders share, and its whole point is that they
// cannot come apart; gating it would silently stop a left click emitting
// orders for a unit in the fog, which is an input rule this hotfix does not
// make. What is dropped is the glyph alone.
func (v *Viewer) selectionScreenRects() []screenRect {
	var seen []MapEntity
	for _, e := range presentSelected(v.sel, v.entities) {
		if v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
			seen = append(seen, e)
		}
	}
	return v.entityGlyphRects(seen, terrain.SelectionMarkerRects)
}

// cellScreenRect is the ONE PLACE A MAP CELL BECOMES A RECTANGLE IN THE
// VIEW: the cell's own footprint, moved by a displacement the caller
// supplies, then through placeArm — the displaced mode's per-cell height
// lift, the camera and the view cull, in that order. ok is false for an
// off-map cell and for one the view does not reach.
//
// THREE THINGS READ IT AND THEY MUST NOT BE THREE DERIVATIONS. The lattice
// outlines it, the hit test asks whether a gesture met it, and the selection rim
// borders it — the rim through the glyph path it already rides, over the very
// footprint terrain.CellFootprint hands back here. So "the unit you box is the
// unit you get, and its mark lands where you boxed" is a property of one
// expression rather than of three that presently agree.
//
// It writes the lift NOWHERE. placeArm owns that arithmetic and this function
// calls it; a second copy here is precisely the defect 0058 exists to close —
// the pick used to resolve the gesture through the camera's FLAT lattice while
// every glyph took the relief, so on a mountain the two were apart by that
// cell's own height and a box caught a unit two cells away.
//
// The shift is added BEFORE placeArm, so the cull runs on the rectangle actually
// drawn and a unit is culled or kept as one — entityGlyphRects' own rule, for
// its own reason.
func (v *Viewer) cellScreenRect(cell, shift image.Point) (screenRect, bool) {
	foot, ok := terrain.CellFootprint(cell.X, cell.Y, v.grid.Width, v.grid.Height, terrain.CellSize)
	if !ok {
		return screenRect{}, false
	}
	return v.placeArm(cell, foot.Add(shift))
}

// entityPickRect is the rectangle ONE entity is picked by: its own cell's
// footprint in the view, carrying that entity's own displacement.
//
// It reads NOTHING ABOUT THE UNIT'S ART, and that is the contract rather
// than an omission (and the seam provenance.md names). A unit's body is
// drawn standing ABOVE its cell — the sprite anchor puts its feet on the
// footprint's lower edge and its crown a cell or so higher — so a box
// drawn around a head alone catches nothing, and a box around a unit as it
// stands on the ground catches it. Picking by the drawn sprite instead would
// need a second rule for an entity that resolves no frame and draws a
// square, and a depth rule where two bodies overlap; neither exists and
// neither is asked for.
//
// A DEAD ENTRY IS NOT FILTERED HERE. Whether a corpse is a candidate is the
// picker's question and is asked at the hit, so that a tap on a corpse clears
// exactly as a tap on empty ground does; this function answers only where a
// unit is.
func (v *Viewer) entityPickRect(e MapEntity) (screenRect, bool) {
	return v.cellScreenRect(e.Cell, v.entityShift(e))
}

// groundCellAt resolves a screen point to the ground cell ROM1's corner-mesh
// picker names. It is the destination a cell-naming click or ground drop uses,
// and the cell the hover path queries for fog and sacks (DIV-044).
//
// THE COLUMN IS EXACT BEFORE THE ROW SEARCH. Relief has no horizontal term
// (`TERR-GEOM-035`), so ScreenToWorld followed by floor and CellSize selects
// the native map column. Projection.CellColumnBounds then evaluates both of
// that cell's corner edges at floor(worldX)&31 with ROM1's truncation-toward-
// zero integer lerp (`TERR-GEOM-036` part (d), High). The point's world Y is
// compared to both inclusive bounds. An inverted pair matches nothing; a
// collapsed pair can match its one shared coordinate.
//
// ROWS ARE SEARCHED IN ASCENDING TERRAIN-PASS ORDER and the first match wins.
// That is ROM1's shared-horizontal-edge ownership: the lower bound of row r and
// the upper bound of row r+1 are the same edge and both are inclusive, so row r
// wins. forEachDrawnTile supplies exactly the current terrain pass's clipped
// row population. The rasterizer itself still uses its decoded table walk; the
// claim measures that drawn edge and the picker lerp as distinct models.
func (v *Viewer) groundCellAt(sx, sy float64) (col, row int, inside bool) {
	if math.IsNaN(sx) || math.IsNaN(sy) || math.IsInf(sx, 0) || math.IsInf(sy, 0) {
		return 0, 0, false
	}
	// A POINT OFF THE MAP VIEW IS NOT ON ANY DRAWN CELL (1026 B3, B4; DIV-212).
	// It is not now. The frame has a 160-pixel right strip the map is not drawn
	// in, and the placement leaves a letterbox outside the frame where the
	// window shows nothing at all; both resolve through the transform below to
	// a cell that is on the grid but was never drawn, so without this test a
	// press on either walked the selection to a cell the player cannot see.
	//
	// It is the same refusal the doc above already states for a point past the
	// edge of what is drawn, applied to the two places that edge now has that
	// are inside the window.
	if !v.mapSurfaceCaptures(int(math.Floor(sx)), int(math.Floor(sy))) {
		return 0, 0, false
	}

	if v.Mode() != ModeDisplaced {
		col, row, inside := v.cam.ScreenToCell(sx, sy)
		if !inside || v.grid.BorderCell(col, row) {
			return 0, 0, false
		}
		return col, row, true
	}

	wx, wy := v.cam.ScreenToWorld(sx, sy)
	fc := math.Floor(wx / terrain.CellSize)
	if !(fc >= 0 && fc < float64(v.cam.Cols)) {
		return 0, 0, false
	}
	gc := int(fc)
	nativeX := int(math.Floor(wx))

	found := false
	var gr int
	v.forEachDrawnTile(func(tx, ty int) {
		if found || tx != gc {
			return
		}
		top, bottom := v.proj.CellColumnBounds(tx, ty, nativeX)
		if !(float64(top) <= wy && wy <= float64(bottom)) {
			return
		}
		gr, found = ty, true
	})
	if !found {
		return 0, 0, false
	}
	// The lower render lip is real terrain art but not ground a unit or item
	// may enter. The same border bit already blocks simulation; rejecting it at
	// the picker keeps hover, orders and ground drops from presenting it as a
	// valid destination first.
	if v.grid.BorderCell(gc, gr) {
		return 0, 0, false
	}
	return gc, gr, true
}

func (v *Viewer) overlayScreenRects(
	show bool,
	cells []image.Point,
	rects func(col, row, cols, rows, cellpx int) []image.Rectangle,
) []screenRect {
	if !show || len(cells) == 0 {
		return nil
	}

	var out []screenRect
	for _, cell := range cells {
		arms := rects(cell.X, cell.Y, v.grid.Width, v.grid.Height, terrain.CellSize)
		for _, arm := range arms {
			if r, ok := v.placeArm(cell, arm); ok {
				out = append(out, r)
			}
		}
	}
	return out
}

// placeArm is the transform itself, over ONE already-built arm on one cell: the
// displaced mode's per-cell height lift, the camera, and the view cull, in that
// order. ok is false for an arm the view does not reach.
//
// It is split out of the loop above because the health bar cannot ride that
// loop: every glyph there is a function of its CELL alone, so one builder
// serves a whole cell list, while a bar is a function of the cell AND of the
// unit standing on it.
//
// Everything it reads is read PER ARM rather than hoisted: the zoom, the view
// extent and the mode are three field reads and a nil test, and hoisting them
// would be a second set of values for a caller to forget to refresh.
func (v *Viewer) placeArm(cell image.Point, arm image.Rectangle) (screenRect, bool) {
	return v.placeLifted(v.cellLift(cell), arm)
}

// cellLift is the vertical world-pixel offset every glyph on one cell carries:
// the displaced mode's own -AnchorHeight - MinV, and 0 in flat mode.
//
// It is split out because a glyph that spans TWO cells cannot take either
// cell's lift whole. A spell figure runs from the caster's cell to the target's
// and the two stand at different altitudes, so its stamps interpolate between
// the two lifts this returns (spellBoltLift, spellbolt.go). Every glyph that
// belongs to one cell still reaches it through placeArm and is unchanged.
func (v *Viewer) cellLift(cell image.Point) int {
	if v.Mode() != ModeDisplaced {
		return 0
	}
	return -v.proj.AnchorHeight(cell.X, cell.Y) - v.proj.MinV
}

// placeLifted is placeArm over a lift the caller has already computed: the
// vertical shift, the camera, and the view cull, in that order.
func (v *Viewer) placeLifted(lift int, arm image.Rectangle) (screenRect, bool) {
	arm = arm.Add(image.Pt(0, lift))
	zoom := v.cam.Zoom
	sx, sy := v.cam.WorldToScreen(float64(arm.Min.X), float64(arm.Min.Y))
	w := float64(arm.Dx()) * zoom
	h := float64(arm.Dy()) * zoom
	if sx+w <= 0 || sy+h <= 0 || sx >= float64(v.cam.ViewW) || sy >= float64(v.cam.ViewH) {
		return screenRect{}, false
	}
	return screenRect{X: sx, Y: sy, W: w, H: h}, true
}

// HealthBarsShown reports whether the show-health setting is on. It mirrors
// DamageNumerals' shape and reason: the tier that owns the world needs to see
// that the switch moved without composing a frame and measuring pixels.
func (v *Viewer) HealthBarsShown() bool { return !v.healthBarsHidden }

// ToggleShowHealth flips the show-health setting and reports the new state
// (round 3).
//
// `keyboard.tsv` row 48 and TOWN-186 name the ShowAllHitPoints option. The
// owner's selection reference keeps overhead status for selected actors;
// this setting therefore controls unselected health and mana bars. The exact
// original option predicate remains unclaimed (DIV-333).
//
// It is a flip and not a show, on ToggleDamageNumerals' own precedent: the bars
// are drawn from the moment a map opens, so the first press is the caption's
// "Off".
//
// IT GATES THE PLACEMENT WALK AND NOT A DRAW, which is the difference from the
// numerals: nothing accrues while the bars are hidden, so switching them back on
// shows the current health of every visible unit immediately.
func (v *Viewer) ToggleShowHealth() bool {
	v.healthBarsHidden = !v.healthBarsHidden
	return !v.healthBarsHidden
}

// Selected units retain their overhead status when Show Health is off.
// The ordinary selection mark is this status, not a diagnostic cell rim.
func (v *Viewer) entityStatusShown(id uint32) bool {
	return !v.healthBarsHidden || slices.Contains(v.sel, id)
}

// overlayPasses returns the frame's passes in draw order. Opt-in diagnostic
// cell markers follow content and overhead status; the selection rim follows
// those markers only while the diagnostic unit overlay is enabled.
// A pass with nothing to draw is omitted entirely.
//
// Draw consumes this and nothing else, so the returned order is the drawn
// order and a test can assert it without opening a window — which is the
// only automated guard the stories have against the passes being swapped
// (0009 R-4). That is also why the sprites live INSIDE this slice rather
// than as a call between drawStatics and the loop in Draw's body:
// sprites-under-squares would then be a call order no test observes
// (rejected).
//
// THE TWO ENTITY PASSES ARE PREPENDED, sprites first, and both halves of
// that order are semantic. Squares over sprites: sprite-or-square is
// exclusive PER ENTITY, so the square pass holds only entities that have no
// sprite anywhere, and a sprite drawn over one would hide the only thing
// that draws that entity. Both under the crosses, for the mirror image of
// the reason the static pass is appended: an entity square is a FILLED
// SQUARE, a superset of every glyph standing on the same cell, and a sprite
// is content outright — so drawn last either would not dim a coincident
// diagnostic cross, it would erase it, and the cell where the two most need
// comparing is exactly the coincident one, at tick 0, where an entity stands
// on the very unit record it was built from. Content may not hide the
// instrument measuring it, so the content goes underneath: a coincident pair
// reads as a cross over the entity's art or block.
//
// Nothing before this line moves. Draw's own body still paints terrain, then
// the static-object art, then this slice, so "the entity layer sits after
// the static art" is inherited from a call order neither 0020 nor 0022
// touches; the readable order is the slice's.
//
// THE STATIC PASS IS APPENDED LAST, and it is safe there only because of
// what its glyph is: radius 3 by thickness 1 is a strict subset of the unit
// cross, and so of the object cross, at native scale and above. A glyph
// reaching further, drawn last, would hide a coincident unit marker entirely
// — not degrade it, erase it — which is the failure the
// objects-then-units order already exists to prevent, reintroduced one pass
// later. The window never builds a glyph below native scale: the arms are
// generated at terrain.CellSize and only then scaled by the camera, so the
// sub-native collapse that would make radii 3 and 4 coincide cannot arise
// here.
//
// THE SELECTION PASS IS APPENDED AFTER IT, later still, and for once the two
// halves of "last" agree instead of trading off. Over the sprite it must be
// to be seen at all — a highlight under the art it highlights is no
// highlight — and it covers nothing, because its glyph is a HOLLOW RIM on
// the footprint's edge while every other glyph in the frame stands within
// scaleDim(6) of the cell's centre, sixteen native pixels away. A filled
// cell was rejected for exactly the reason the entity square is drawn first:
// filled, it could only go underneath, and underneath it would be erased by
// the art.
//
// It is the one pass with no toggle and no cell list of its own: its cells
// are wherever the selected units stand in this frame's snapshot, ONE MARK
// EACH, and with nothing selected — or a selection the snapshot no longer
// holds any of — selectionScreenRects returns nothing and NO PASS IS
// APPENDED AT ALL. So a viewer that has never been given a selection, which
// is every viewer cmd/mapview can build, produces exactly the pass slice it
// produced before either story. THE BLOCKED TINT IS PREPENDED, ahead of the
// entity layer and every glyph. It is the one FULL-CELL fill in the frame,
// so it is also the one pass that could hide another outright; drawn first
// it cannot, and drawn translucent it does not hide the terrain and static
// art already painted under the whole slice. Opaque it would have to run
// underneath everything, and underneath is unreachable from here — Draw
// paints the terrain and then the static-object art before the first pass of
// this slice exists, so "first" means over the art whatever is done with the
// order. The colour is what makes that safe rather than the position.
//
// Absent the toggle it contributes no pass at all, so a viewer that was never
// given cells — every viewer built before this story, and every unflagged run —
// produces exactly the pass slice it produced before.
func (v *Viewer) overlayPasses() []overlayPass {
	var passes []overlayPass
	// THE LATTICE IS NO LONGER ONE OF THIS SLICE'S PASSES (item-2 hotfix). An
	// overlayPass carries filled rectangles and the lattice now draws its own
	// cells' quadrilaterals as STROKES — the shape gridScreenSegments returns
	// — so it is drawn separately in Draw, at the same point in the frame
	// this pass used to occupy: ahead of even the blocked tint, so it can still
	// hide nothing.
	if rects := v.blockedScreenRects(); len(rects) > 0 {
		passes = append(passes, overlayPass{Color: terrain.BlockedCellColor, Rects: rects})
	}
	// THE SPRITE HALF OF THE ENTITY LAYER IS NO LONGER A PASS HERE. It is
	// content, and it moved into the content band drawArt paints before this
	// slice exists, where it takes its place in the one back-to-front order the
	// structure and object planes already share. The SQUARE half stays, because
	// it is a diagnostic glyph for an entity whose art did not resolve — and
	// leaving it here keeps "a square is drawn over the sprite band" true by
	// position rather than by argument.
	_, squares, _ := v.entityLayer()
	if rects := v.entityGlyphRects(squares, terrain.EntityMarkerRects); len(rects) > 0 {
		passes = append(passes, overlayPass{Color: terrain.EntityMarkerColor, Rects: rects})
	}
	if rects := v.objectScreenRects(); len(rects) > 0 {
		passes = append(passes, overlayPass{Color: terrain.MarkerColor, Rects: rects})
	}
	if rects := v.unitScreenRects(); len(rects) > 0 {
		passes = append(passes, overlayPass{Color: terrain.UnitMarkerColor, Rects: rects})
	}
	if rects := v.staticMarkerScreenRects(); len(rects) > 0 {
		passes = append(passes, overlayPass{Color: terrain.StaticMarkerColor, Rects: rects})
	}
	// THE STATUS BAR PASSES, appended after every glyph above and before the
	// selection rim. Under the rim because the rim is the one pass that must be
	// over everything and covers nothing; over the crosses because a bar is
	// content about the unit rather than an instrument measuring where it
	// stands. statusBarPasses appends nothing when no unit shows a bar, so a
	// viewer whose entries name no health produces the pass slice it did
	// before.
	passes = append(passes, v.statusBarPasses()...)
	// THE SPELL EFFECT MARKS COME BETWEEN THE BARS AND THE SELECTION RIM: over
	// the bars, because a mark is a momentary statement about the unit and a
	// bar is a standing one, so the momentary mark must not be hidden by it;
	// and under the rim, which keeps the rim the one pass that is over
	// everything. A marked unit that is also selected then shows both, the rim
	// outermost.
	passes = append(passes, v.spellEffectPasses()...)
	// THE SPELL'S OWN ART IS NO LONGER A PASS. 0154 drew a coloured square for
	// a bolt and an expanding ring for the burst, both of them filled
	// rectangles and both of them here; they are gone, and what replaced them
	// is the game's own sheet, which is a texture and is drawn in the content
	// band with the rest of the map's art (drawSpellArt, spellbolt.go).
	if v.showUnits {
		if rects := v.selectionScreenRects(); len(rects) > 0 {
			passes = append(passes, overlayPass{Color: terrain.SelectionMarkerColor, Rects: rects})
		}
	}
	if rects := v.marqueeScreenRects(); len(rects) > 0 {
		passes = append(passes, overlayPass{Color: MarqueeColor, Rects: rects})
	}
	return passes
}

// marqueeScreenRects is the selection rectangle's outline, in SCREEN PIXELS:
// the four strips of a hollow frame between the press point and the current
// cursor.
//
// IT IS THE ONE PASS BUILT IN SCREEN SPACE. Every other pass starts from cells
// and goes through the world-to-screen transform; this one is already where it
// belongs, so it bypasses that transform entirely, takes no cell, and needs no
// glyph builder in the render tier — where every builder takes a cell and a
// scale, which a screen rectangle has neither of. That is also why it does not
// snap to the cell lattice: the cursor moves by pixels and so does this.
//
// THREE CONDITIONS, ALL READ OFF STATE THAT ALREADY EXISTS. A press is in
// progress (dragging, which step clears the moment the button reads up, so the
// outline disappears on release); the press was latched as a rectangle rather
// than a pan (boxing, which is false for every viewer cmd/mapview can build and
// for every press begun with the modifier); and the gesture has passed the slop,
// so a tap flashes nothing.
//
// IT IS HOLLOW, for the mark's own reason: filled, it would hide the very
// units the drag is choosing. The two side strips span the FULL height
// rather than the band between the other two, so the frame is total at any
// size — a rectangle thinner than twice MarqueeThickness on an axis simply
// has no interior left to leave visible, and the strips meet there instead
// of taking a negative extent. A drag with no horizontal travel at all still
// draws its vertical line, because the sides are placed at the edges rather
// than inset from a width it does not have.
//
// IT IS NOT THE OUTLINE'S TEST AND THE ORIGINAL'S TWO ARE NOT ONE. Through
// this story's round 2 the cursor read marqueeScreenRects, so it answered
// `default` only past the RELEASE threshold — `screenW*10/640`, 16 pixels
// at this build's fixed mission frame (`AI-INPUT-121`). `IsRectEmpty` is a
// near-zero-area test: the original replaces the cursor as soon as the
// rectangle has both a width and a height, which is one pixel of travel on
// each axis, and a drag of 1 through 16 pixels therefore showed the ordinary
// hover cursor here where the original shows `default`.
//
// `IsRectEmpty` on a NORMALISED rectangle is true when either side is zero, so
// its negation is "both axes moved". A drag straight down the screen therefore
// does NOT replace the cursor, which is the Win32 predicate and not a
// simplification of it.
//
// The three conditions marqueeScreenRects carries besides its threshold are
// carried here too: no gesture, no boxing gesture, or one the open inventory
// window took is no live marquee at all.
func (v *Viewer) marqueeCursorLive() bool {
	if !v.dragging || !v.boxing || v.invGrab {
		return false
	}
	return v.dragX != v.pressX && v.dragY != v.pressY
}

func (v *Viewer) marqueeScreenRects() []screenRect {
	// A FOURTH CONDITION, off state that already exists: a gesture the open
	// inventory window took draws no rectangle either (hotfix: the window
	// swallows its own clicks). It selects nothing, so an outline promising
	// that it will is the same defect one layer up.
	// THE THRESHOLD IS marqueeSlop(), NOT TapSlop (F1, round 2). Both the outline
	// and the release (command.go) must agree on the one number that decides
	// whether a gesture is a marquee, or the outline shows for a travel the
	// release will still dispatch as a click. AI-INPUT-121's own comparison is
	// STRICT -- "beyond" the threshold is a rectangle, "at or under" it is a
	// click -- so this reads `<=` to nil out exactly the travel the release
	// taps on, the same boundary command.go:1506 uses.
	if !v.dragging || !v.boxing || v.dragMoved <= v.marqueeSlop() || v.invGrab {
		return nil
	}

	x0, x1 := float64(v.pressX), float64(v.dragX)
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	y0, y1 := float64(v.pressY), float64(v.dragY)
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	w, h := x1-x0, y1-y0
	t := float64(MarqueeThickness)

	return []screenRect{
		{X: x0, Y: y0, W: w, H: t},     // top
		{X: x0, Y: y1 - t, W: w, H: t}, // bottom
		{X: x0, Y: y0, W: t, H: h},     // left
		{X: x1 - t, Y: y0, W: t, H: h}, // right
	}
}
