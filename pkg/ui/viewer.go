// Package ui is the windowed map viewer: an Ebitengine run loop over the pure
// camera model (pkg/render/camera) and the terrain pipeline (pkg/render/terrain).
//
// The engine is confined to this package and the cmd tier; the camera
// arithmetic and the terrain decode it drives stay engine-free and
// unit-testable. All pan/zoom/edge-scroll behaviour here is the project's
// own UX design, not a reproduction of the original engine's camera.
package ui

import (
	"image"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"againrom/pkg/audio"
	"againrom/pkg/render/camera"
	"againrom/pkg/render/frame"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
)

// Viewer UX constants — ours, not decoded from the game.
const (
	// EdgeMargin is how close to a window edge the cursor scrolls the view.
	EdgeMargin = 24

	// PanSpeed is the edge-scroll and key-pan speed in world pixels per tick.
	// It is a world-space rate, so scrolling covers the same ground at any zoom.
	PanSpeed = 12

	// WheelZoomStep is the multiplicative zoom applied per wheel notch.
	WheelZoomStep = 1.2

	// DefaultWindowW and DefaultWindowH size the window at startup. They size a
	// WINDOW and not a frame; the mission frame below is what the screen is
	// composed at, and the two are equal only by coincidence.
	DefaultWindowW = 1024
	DefaultWindowH = 768

	// MissionFrameW and MissionFrameH are the mission screen's MINIMUM virtual
	// frame size in frame pixels. MissionFrameH remains the logical height at
	// every window size. MissionFrameW is retained at 4:3 and in narrower
	// windows; wider windows grow the logical width so the picture reaches both
	// horizontal edges instead of keeping a 4:3 pillarbox.
	//
	// THE SIZE IS THE OWNER'S, NOT A DECODE (owner, DIV-210). The original
	// selects one of three literal arms at run time and its shipped default is
	// 640x480; this build selects none and keeps 1024x768 as its minimum
	// because he asked for it. What IS decoded is what that arm yields, and
	// that is MissionPanelW below and the minimum viewport it leaves.
	MissionFrameW = 1024
	MissionFrameH = 768

	// MissionPanelW is the width of the frame's right strip, in frame pixels.
	//
	// SESS-VIEW-028, High: the map view object is constructed with the rect (0,
	// 0, screenW - 0xa0, screenH), and 0xa0 is 160. The strip is the panel.
	MissionPanelW = 160

	// startColumns is the value AuthoredStartColumns states, and it is
	// unexported so there is one way to read it. See that function for what it
	// is and why it is that.
	startColumns = 27

	// MarqueeThickness is how wide the selection rectangle's outline is drawn,
	// in SCREEN pixels.
	//
	// It is screen pixels and not world pixels because the rectangle itself is:
	// it is built from the press point and the current cursor and never passes
	// through the world-to-screen transform, so a zoom changes what the box
	// covers and never how thick its edge looks.
	MarqueeThickness = 2

	ShotMarkSize = 4
)

// MarqueeColor is the opaque colour the selection rectangle's outline is
// drawn in.
//
// It lives here beside EdgeMargin and PanSpeed rather than in the render tier's
// glyph palette, and so does the thickness above: every builder there takes a
// cell and a scale, and a screen rectangle has neither.
//
// It is deliberately NOT the white the selected units' marks carry. Two passes
// of one colour are indistinguishable to anything reading the pass slice, and
// the pass slice is this package's only automated view of what a frame draws —
// so a shared colour would cost the outline every assertion that can tell it
// from a mark.
var MarqueeColor = color.RGBA{R: 0x80, G: 0xff, B: 0x80, A: 0xff}

var ShotMarkerColor = color.RGBA{R: 0xff, G: 0x60, B: 0x00, A: 0xff}

// Mode is which of the two terrain paths the viewer renders through.
//
// It is never stored: Mode() computes it from the viewer's current state.
type Mode int

const (
	// ModeFlat draws every cell as an axis-aligned CellSize square. It is the
	// mode for a map with no usable altitude grid, and the mode SetFlat(true)
	// deliberately selects over a valid one — a diagnostic overlay no longer
	// forces it on its own.
	ModeFlat Mode = iota

	// ModeDisplaced draws each cell on the shared projection's vertex mesh.
	ModeDisplaced
)

func (m Mode) String() string {
	if m == ModeDisplaced {
		return "displaced"
	}
	return "flat"
}

// Viewer draws a map's terrain through a camera. Construct it with NewViewer —
// or, for a map that also carries a static-object layer, NewViewerWithStatics —
// and hand it to Run.
type Viewer struct {
	graphics            GraphicsOptions
	sackBoundaries      map[*terrain.StaticFrame]*terrain.StaticFrame
	sackHighlight       bool
	sackOutlineImages   map[sackOutlineKey]*ebiten.Image
	tooltip             *tooltipController
	tooltipManaged      bool
	applicationRestored bool
	playerPaused        bool
	playerPauseLabel    string
	editorView          bool // read-only authored map canvas; no mission HUD
	missionCutscene     int  // movie family, requested only by an accepted completion
	completionCutscene  string
	entryCampaignStart  bool // character creation precedes the first mission
	title               string
	grid                terrain.Grid
	set                 *terrain.Tileset
	cam                 *camera.Camera

	// place is the mission frame's fit into the current window, and canvas is
	// the frame-sized image the screen is composed on before it is placed.
	//
	// THE CAMERA'S VIEW IS THE VIEWPORT AND NOT THE FRAME. cam.ViewW/ViewH are
	// the current frame minus the right strip and any visible bottom bars, so
	// every rectangle this package derives from the view size stays clear of
	// the HUD. The canvas is the whole expanded frame.
	//
	// EVERY POSITION INSIDE THIS PACKAGE IS A FRAME POSITION once it is past
	// step, command and noticeButtonAt, which are the three doors a window
	// position comes in through (1026 B4).
	place            frame.Placement
	canvas           *ebiten.Image
	statusBarScratch *ebiten.Image

	// textSmoothingEnabled is owner decision method C's own on/off switch
	// (DIV-1385), mirrored from the App by SetTextSmoothing. textOverlay
	// smooths the mission character panel, mission card and mission tooltip
	// — the only three mission HUD boxes this story wires; the rest of the
	// mission HUD (spellbook, inventory, command panel, message line,
	// in-mission notices, damage numerals) still bakes text at native
	// resolution, unaudited debt named in the story record.
	textSmoothingEnabled bool
	// frameSmoothingOff is the FrameSmoothing switch negated, so a Viewer
	// that never receives it uses the Catmull-Rom scaler.
	frameSmoothingOff bool
	textOverlay       textOverlay
	textEraser        textEraser
	// canvasLog records the HUD drawn over canvas after the first captured
	// box (pixelLog).
	canvasLog pixelLog
	// settledBuf/settledTex hold the frame read back with its visible
	// glyphs erased (settleTextFrame), on the frames canvasLog cannot
	// decide; textSettleFallbacks counts those frames.
	settledBuf          *image.RGBA
	settledTex          *ebiten.Image
	textSettleFallbacks int
	textCaptured        int
	textKept            int
	// textCalls is every glyph the last frame captured, before the pixel log
	// settled which of them show.
	textCalls []text.DrawCall

	// frameW, frameH are the current virtual-frame size in logical pixels.
	// frameH remains MissionFrameH. frameW begins at MissionFrameW and grows
	// with a wider window's aspect; the right strip follows its right edge and
	// the newly available width belongs to the map viewport.
	frameW, frameH int

	// startCell is the cell an armed start view opens on, and startArmed is
	// whether one is armed. Both are written by SetStartView alone and cleared
	// by the application; the bool is what makes it a one-shot rather than a
	// follow camera, since the zero Point is a legitimate cell and cannot serve
	// as the absent value.
	startCell  image.Point
	startArmed bool
	startSAV   bool

	// proj is the map's height-displaced vertex mesh, or nil when the grid
	// carried no usable altitudes. Non-nil is exactly "the altitude grid is
	// valid", which is one half of Mode's predicate; the other half is the flat
	// diagnostic below, read live — the overlay flags no longer enter it.
	//
	// It is built once, in NewViewer: it is a pass over every vertex of the map
	// and nothing here can replace the grid afterwards, so a per-frame rebuild
	// would re-walk a 256x256 mesh for a value that cannot change.
	proj *terrain.Projection

	// levels is the map's per-vertex relief-lighting grid: width g.Width,
	// height g.Height, a byte in [0,95] per vertex, or nil when the altitude
	// grid was not usable. Built once, in NewViewer, beside proj and under the
	// SAME validAltitudes(g) guard: that shared guard, not LevelGrid's own
	// totality, is what makes levels != nil and proj != nil decide one map the
	// same way. LevelGrid rejects a slice shorter than w*h but accepts an
	// overlong one, where validAltitudes demands exact length, so hoisting this
	// call out of the branch — or substituting LevelGrid's own nil-return as
	// the guard — would desynchronise the two.
	//
	// Lighting uses v.sun, which is the day/night cycle's cache: no flag and no
	// map field selects another one, and nothing but a relight moves it. Until
	// 0092 this read "the fixed daytime light" and was right — that is
	// exactly the divergence 0092 closes.
	levels []uint8

	// sun is THE ONE CACHE every consumer of the day/night cycle reads, and
	// relightAt is its only writer. What the window draws is therefore the sun
	// as of the LAST RELIGHT and never the sun of the current tick — which is
	// the original's own arrangement, not an optimisation: it keeps the angle,
	// the per-vertex grid and every shading table describing one instant
	// instead of three.
	//
	// It is SEEDED WITH THE CYCLE-OFF SUN, the value this tree drew before the
	// cycle existed, so a viewer that is never told the time draws exactly what
	// it drew before (AC-9). That the switch below reads ON while the seed is
	// the switch-off value is not a contradiction: the switch selects which arm
	// the NEXT relight takes, and a viewer that never relights has never asked
	// what time it is.
	sun terrain.Light

	// lightClock is the last sub-tick clock this viewer was given and
	// lightOffset is the diagnostic shift added to it. Both are plain counters
	// in the engine's own sub-tick unit; neither is canonical state, no byte
	// form carries either, and no world can hold one.
	//
	// The OFFSET MOVES IN WHOLE IN-GAME HOURS ONLY, and that is what keeps it
	// invisible to the relight cadence: an hour is 60 minutes, 60 is three
	// whole relight periods, so RelightDue answers the same for the shifted
	// clock as for the raw one.
	lightClock  uint64
	lightOffset uint64

	// timeFlow is the day/night cycle's switch. It is ON by default and is
	// written in the constructor rather than left at its zero value, because
	// the zero value is the wrong default and a field nobody writes is
	// indistinguishable from one written to false on purpose.
	//
	// It is NOT PERSISTED, and that is a disclosed divergence (spec D-2): the
	// original keeps it in the registry and in the savegame's options section,
	// this tree reads no registry, and its byte form is the world rather than the
	// session's options. Every run therefore starts at the compiled default.
	timeFlow bool

	// unshaded is the standalone developer viewer's diagnostic: on, drawing is
	// full palette brightness; off (the default), corner-interpolated shading.
	// Set only by SetUnshaded; the game front-end never touches it, so it gains
	// no flag and no call.
	unshaded bool

	// flat is the other half of Mode's predicate: a deliberate request for flat
	// terrain over an otherwise-valid altitude grid, set only by SetFlat. The
	// game front-end never touches it, mirroring unshaded.
	flat bool

	// cache holds one GPU image per distinct (slot, sub-cell, tint) actually
	// drawn, so each terrain sub-cell is uploaded at most once per band it is
	// actually seen under.
	cache map[cacheKey]*ebiten.Image

	placeholder *ebiten.Image

	// anim drives water animation.
	//
	// speedIndex is the game speed index this viewer was last GIVEN, kept here
	// because the ticker holds a period alone and cannot answer for one. It is
	// written by SetSpeedIndex and by nothing else — a period written through
	// SetPeriod moves the clock and deliberately leaves this where it was, so
	// the standalone summary keeps answering the question it answers today
	// rather than reporting an index nobody selected.
	anim              *terrain.Ticker
	speedIndex        int
	animate           bool
	last              time.Time
	unpaced           bool
	animationPaused   bool
	animationRestored bool

	// objectCells/unitCells are the placed objects' and units' anchor cells, and
	// showObjects/showUnits are the two overlays' independent toggles. All four
	// are set by the cmd tier through SetObjects/SetUnits; the draw path and the
	// transform live in overlay.go.
	objectCells []image.Point
	showObjects bool
	unitCells   []image.Point
	showUnits   bool

	// blockedCells/showBlocked are the fourth diagnostic overlay's cells and
	// its toggle, set through SetBlocked and read by the draw path alone.
	//
	// They are plain map cells like the four above, and the tier that set them
	// derived them from a block plane this package never sees. That is not the
	// same plane as grid.Block, which the map-loading tier fills without a
	// definition table: the two agree on bit 1 — the margin, BorderCell's only
	// question — and can differ on bit 0, which is the whole of what this
	// overlay is for.
	//
	// The list is a PLANE's worth of cells rather than a record list's, which is
	// why its rects builder does not ride the shared glyph transform.
	blockedCells []image.Point
	showBlocked  bool

	// showGrid is the cell lattice's toggle, off by default and moved only by
	// SetGrid/ToggleGrid.
	//
	// IT CARRIES NO CELL LIST, and that is the difference from the four
	// overlays above rather than an omission. Those mark cells a caller
	// derived — records, a plane — so which cells they cover is the caller's
	// answer. This one covers whatever the terrain pass covered this frame,
	// which is a question only the viewer can answer and one that changes on
	// every pan; a stored list would be a per-frame answer kept in a field.
	showGrid bool

	// entities is the open world's entities — one MapEntity per entity of the
	// world the tier above this one holds, pushed here after each of that
	// world's steps, each a plain map cell beside the render-tier art it
	// resolved to, or nil art for one that draws as the square.
	//
	// THERE IS NO SHOW FLAG BESIDE IT, and that asymmetry with the two overlays
	// above is deliberate: having been given entities IS the switch. The entity
	// layer is the map's content rather than a diagnostic, so there is nobody
	// to toggle it — a viewer that was never given any (the standalone
	// viewer, and every pre-0020 caller) produces no entity pass at all and
	// draws exactly what it drew before either story (AC-10).
	//
	// It names no simulation, format or data type: a cell is a plain integer
	// point, exactly as objectCells and unitCells are, and the art is the
	// render tier's own bundle entry. The conversion from an entity to a cell,
	// and the class-id-to-art resolution, both happen in the tier that may see
	// a world, a bundle and a window at once, which is what leaves this package
	// unable to spell any of those types at all.
	entities []MapEntity

	structureStates       []MapStructure
	structureInfo         map[uint32]*terrain.StructureClass
	structureSet          *terrain.StructureSet
	unitInspectionPicture func(uint32) *image.RGBA

	// sackFrames is the decoded sack sheet's frames, in the sheet's own order,
	// or nil for a viewer never given one.
	//
	// IT IS SET ONCE, by SetSackFrames, mirroring the font's and the attack
	// pointer's own installation on BOTH opener paths: the sheet is decoded
	// once in the front end and handed to each viewer it opens, never rebuilt
	// per map and never rebuilt per frame.
	sackFrames []*terrain.StaticFrame

	// sacks is this refresh's ground-sack list: one MapSack per entry of the
	// world's own list, in the world's own order, or nil for a viewer never
	// given any.
	//
	// IT IS PUSHED EVERY TICK, by SetSacks, beside SetEntities — the two
	// setters share SetEntities' own shape and reason: the slice is ADOPTED
	// rather than copied, and REPLACES whatever was held before, so a sack the
	// world no longer holds draws one entry fewer on the very next frame rather
	// than lingering as a stale copy.
	sacks []MapSack

	// phaseUS/phasePeriodUS is WHERE THIS FRAME STANDS INSIDE THE CURRENT TICK:
	// the microseconds elapsed toward the next tick, and the tick length they
	// are measured in. Both are pushed by the tier that paces the world, from
	// the very accumulator that decides when a tick fires, and they are two
	// plain ints because this seam's values are scalars by rule and because the
	// displacement they scale lands on whole render pixels.
	//
	// THE ZERO PAIR IS "NOT TOLD", and it is the zero value on purpose: a
	// period of zero draws no displacement at all, so a front-end that never
	// pushes one — the standalone viewer, and every caller written before
	// this story — draws every entity on its own cell exactly as it did
	// before.
	//
	// A STOP NEEDS NO STATEMENT HERE. The paced advance returns before the
	// elapsed span reaches its accumulator, so a stopped frame pushes nothing
	// and the pair last written is still the one in force: many frames drawn
	// over a stopped world draw the identical picture.
	phaseUS       int
	phasePeriodUS int

	// staticsFlat and staticsDisplaced are the map's static-object placement
	// lists, one per geometry, and staticCounts is the census of what the build
	// skipped. All three are built once, in NewViewerWithStatics, and nothing
	// here rebuilds them.
	//
	// BOTH GEOMETRIES EXIST FROM CONSTRUCTION, so moving between them SELECTS a
	// list rather than recomputing one: SetFlat moves Mode() and touches
	// neither slice, and the draw path shifts nothing per frame. Rebuilding in
	// SetFlat would be a second build path free to drift from construction's,
	// and shifting the flat list per frame would stop the window placing from
	// the built list unchanged — which is the whole content of the two
	// renderers' agreement about where a cell's art stands.
	//
	// staticsDisplaced is nil exactly when proj is nil: it is built inside the
	// same validAltitudes(g) branch, beside proj and levels, so the three cannot
	// come to disagree about whether this map has a usable altitude layer.
	// Mode() == ModeDisplaced implies proj != nil, so staticPlacements never
	// selects a list that was not built.
	//
	// One census serves both lists because which cells resolve is a question
	// about placement bytes and loaded classes alone — the geometry reaches
	// only the anchors.
	staticsFlat       []terrain.StaticPlacement
	staticSet         *terrain.StaticSet
	staticAnimGate    bool
	baseTileWords     []uint16
	scorchedCells     []uint16
	showPathfinding   bool
	scorchInitialized bool
	scorchTick        uint64
	scorchHistory     []uint16
	burningScenery    map[uint16]uint64
	staticsDisplaced  []terrain.StaticPlacement
	staticCounts      terrain.StaticCounts

	// animFlat and animDisplaced are the CYCLE-CAPABLE subsets the same two
	// builds returned. The game supplies every class with a timeline here;
	// staticPlacements applies current fog visibility at draw time. The index
	// sets stay valid across SetFog because fog changes no placement list.
	//
	// staticScratch is the per-counter pass's buffer, this viewer's own and
	// never handed out. It is assigned ONLY from a result the pass actually
	// wrote into: where nothing can change the pass hands back the built list
	// itself, and storing that would leave the scratch aliasing the builder's
	// slice for a later frame to write through.
	//
	// It stays nil for the whole life of an ordinary map — no cell opens the
	// gate, the switch is on, and the pass returns the built list every frame
	// — so this layer allocates nothing per frame and nothing at all.
	animFlat      []int
	animDisplaced []int
	staticScratch []terrain.StaticPlacement

	// structuresFlat and structuresDisplaced are the map's STRUCTURE placement
	// lists, one per geometry, and structureCounts the two-level census of what
	// the build skipped. All three are built once, in NewViewerWithStatics,
	// beside the object layer's own three and under the same rules: both
	// geometries exist from construction, so moving between them SELECTS a list
	// rather than recomputing one, and structuresDisplaced is nil exactly when
	// proj is nil.
	//
	// A structure entry is ONE IMAGE ROW of one rectangle cell, not one sprite
	// per building: a structure covers its whole TileWidth x TileHeight rectangle
	// and each of those cells is drawn as a vertical strip of tile-sized frames
	// (spec, "The strip"). So these lists are far longer than the object layer's
	// per-cell one, and a placement count is not a building count.
	//
	// One census serves both lists because which placements resolve is a question
	// about keys and loaded classes alone — the geometry reaches only the
	// top-lefts.
	structuresFlat      []terrain.StructurePlacement
	structuresDisplaced []terrain.StructurePlacement
	structureCounts     terrain.StructureCounts

	// structureAnimFlat and structureAnimDisplaced are the ANIMATED SUBSETS the
	// same two builds returned: the index, in the list beside them, of every
	// entry that can change with the counter. They are built once with their
	// lists and never rebuilt, because whether an entry can change is a
	// per-class fact no counter moves.
	//
	// structureScratch is the per-counter pass's buffer, this viewer's own and
	// never handed out. It is assigned ONLY from a result the pass actually wrote
	// into: where nothing can change the pass hands back the built list itself,
	// and storing that would leave the scratch aliasing the builder's slice for a
	// later frame to write through.
	structureAnimFlat      []int
	structureAnimDisplaced []int
	structureScratch       []terrain.StructurePlacement

	// planeOrder is the draw order of the structure and object planes merged by
	// rectangle row, built ONCE at construction by the render tier's one merge.
	//
	// ONE ORDER SERVES BOTH GEOMETRIES, and that is not an economy: the merge
	// compares rectangle ROWS, both lists carry the same cells in the same order
	// in either geometry, and a height reaches only a top-left. So a second order
	// could only ever agree with this one or reveal a bug in a builder.
	//
	// It is built rather than walked per frame because it CAN be: both
	// per-counter passes patch entries in place and preserve their lists' length
	// and order, so an index stays valid for the life of the two builds, and the
	// draw path allocates nothing.
	planeOrder []terrain.PlaneRef

	// depthOrder is planeOrder with this frame's ENTITY sprites merged into it
	// by cell row. It is a reused BUFFER and not state: it is overwritten
	// wholesale on every call to planeSprites and carries nothing between
	// frames but its capacity.
	//
	// It cannot be built once the way planeOrder is, and that is the difference
	// between the two rather than an inconsistency: the two art lists keep their
	// length and order for the life of the viewer, and the entity list is rebuilt
	// from the world's snapshot every frame.
	depthOrder []terrain.DepthRef

	// showStructureArt is whether the structure sprites paint. It is set at
	// CONSTRUCTION alone, mirroring showStaticArt: the bundle arrives with the
	// map or not at all, and there is no setter to hand one over afterwards
	// (inherited).
	//
	// It gates the DRAW and not the build, so a viewer given a bundle knows which
	// placements resolve whether or not it paints them — which is what lets a
	// headless -check report the census with no window and no art.
	showStructureArt bool

	// ruinedStructures is the DESTRUCTION DIAGNOSTIC: every drawable structure
	// drawn from its ruin block, respecting Indestructible.
	//
	// It is a draw-time diagnostic and never simulation state. Live structure
	// health arrives separately through ruinedStructureIDs; the global switch
	// remains only for the standalone developer viewer.
	//
	// It is set only by SetRuins, mirroring unshaded and flat. The game front-end
	// does not set it; it refreshes ruinedStructureIDs from the simulation.
	ruinedStructures   bool
	ruinedStructureIDs map[uint32]bool

	// structurePhase is TERR-STRUCT-102's per-drawable "this+0x70" phase,
	// mirrored one entry per structure this viewer has observed in CURRENT
	// sight at least once: the anim counter value that structure's own phase
	// was last advanced at (hotfix, owner report 3, DIV-1351). freezeStructurePhases
	// is the sole writer and reader. It is presentation-only, carries no
	// history across a fresh viewer, and is never hashed or persisted.
	structurePhase map[uint32]uint32

	// showStaticArt is whether the object sprites paint; showStaticMarkers is
	// whether the layer's own diagnostic cross draws. They are two independent
	// switches — either is usable without the other — and both are set at
	// CONSTRUCTION alone: there is no SetStatics, because the bundle arrives
	// with the map or not at all.
	//
	// Neither gates the BUILD. The lists above exist whenever a bundle was
	// supplied, so a viewer asked for the cross alone still knows which cells
	// resolve, and turning the art off changes what is drawn and never what was
	// placed.
	showStaticArt     bool
	showStaticMarkers bool

	// staticImages caches one GPU texture per DISTINCT (object frame, sprite
	// ramp row, sky tint) actually drawn — the frame's own pointer identity,
	// which the builder put into every placement of that class so a forest of
	// two hundred trees uploads a single texture, beside the row and the tint
	// its pixels were resolved with.
	//
	// THE ROW IS IN THE KEY AND THE MIRROR IS NOT, and the asymmetry is the
	// point: a mirrored sprite's pixels are identical and its reflection is a
	// GeoM, where two rows are genuinely two sets of pixels. With one row per
	// rendered frame this holds one entry per frame in the ordinary case and
	// two across an unshaded toggle — and, from 0100 on, one per distinct
	// band a session actually draws under, bounded by TERR-LIGHT-128's 24.
	//
	// It is NIL UNTIL THE FIRST DRAW BUILDS ONE, and deliberately not made in
	// the constructor: a viewer that has been built, queried and culled but
	// never drawn must hold no GPU state at all, which is what lets -check
	// reach this whole layer with no graphics context. staticImage allocates it
	// on the way to storing the first texture.
	staticImages map[spriteTextureKey]*ebiten.Image

	// stoneImages is the grayscale companion cache, keyed by the same frame,
	// lighting row and tint because grayscale is derived after those pixels.
	// It stays nil until a Stone Cursed unit is actually drawn.
	stoneImages map[spriteTextureKey]*ebiten.Image

	// spellLighting and spellTerrainLighting are rebuilt from live canonical
	// area effects on every game push. The former is the per-cell actor input;
	// the latter is the shared vertex plane terrain tiles sample. Both are
	// presentation-only and therefore neither hashed nor persisted.
	spellLighting        map[image.Point]spellLightScale
	spellTerrainLighting map[image.Point]float32

	// structureLighting and structureTerrainLighting are spellLighting's own
	// sibling planes (same absolute brightness-ladder units, same per-cell
	// keying) for the light-source structures' own pulsing mask (owner report
	// 2, DIV-1313 amended; pkg/ui/towerglow.go's refreshStructureLighting).
	// Rebuilt once per frame from drawFrame, never read live from inside a
	// vertex or sprite loop. Presentation-only: neither hashed nor persisted.
	structureLighting        map[image.Point]float32
	structureTerrainLighting map[image.Point]float32

	shadowMasks map[*terrain.StaticFrame]*ebiten.Image

	// dragging is whether a primary-button drag is currently in progress, and
	// dragX/dragY is the cursor position the LAST ticked drag observed, in
	// screen pixels. Both are set only by dragIntent and cleared only by step
	// when the button reads up, so nothing here can outlive a drag: a release
	// wherever the cursor is ends it, and the next press starts a fresh anchor
	// rather than resuming a stale one.
	dragging     bool
	dragX, dragY int

	// dragMoved is how far the cursor has travelled since the current gesture
	// began, in screen pixels: |dx| + |dy| accumulated over every held tick,
	// which is the very delta that tick panned the view by. So "the view moved"
	// and "this was a drag" are ONE number and cannot disagree, and a press
	// that wandered out and back reads as the drag it was.
	//
	// held is whether a primary PRESS edge has been seen and not yet released.
	// Together with the accumulator it is the whole of the tap/drag question:
	// a release with no press is nothing at all, and a release under TapSlop
	// is a tap at that release's own position (0028 AC-2).
	//
	// Both are zeroed wherever a gesture begins — dragIntent's anchor branch
	// and command's press edge — because neither of those alone catches every
	// start: the anchor never runs for a press and a release inside one frame,
	// and the press edge never fires for a button already held when the map
	// opened.
	dragMoved int
	held      bool

	// pressX/pressY is where the CURRENT gesture's button went down, in screen
	// pixels, and boxing is which gesture that press began: true for one that
	// draws a selection rectangle, false for one that pans.
	//
	// All three of these and dragMoved are written in dragIntent's ANCHOR
	// BRANCH and nowhere else, so one branch decides "a gesture began" and the
	// press point, the latch and the accumulator cannot disagree about when.
	//
	// The press point OUTLIVES THE DRAG ANCHOR deliberately. dragX/dragY are
	// overwritten every held tick, so nothing there remembers where a gesture
	// started, and a rectangle between the press and the cursor needs exactly
	// that.
	pressX, pressY int
	boxing         bool

	// commandMode is whether this viewer is the front-end's map screen, with a
	// world under it, rather than the standalone terrain viewer.
	//
	// It is written where the flow stores the tick and the order seam and
	// cleared where the flow drops them, so it lives and dies with the screen
	// that owns a world — the same lifetime, on the same statements, as the two
	// halves of that seam.
	//
	// It is UNEXPORTED and reachable from no exported method, so cmd/mapview
	// can no more set it than it can call command: "the standalone viewer keeps
	// its plain-drag pan and draws no rectangle" is Go's export rule — the
	// mechanism that already keeps that viewer selection-less — rather than a
	// condition someone has to keep true.
	commandMode bool

	// The unit information panel.
	//
	// font is the game's own font a front-end handed over, and NIL IS THE
	// ORDINARY STATE: every viewer built before this story, every hand-built
	// one and the standalone developer viewer hold none, and a viewer with no
	// font draws no panel and fails at nothing. panelLayout is what the panel
	// looks like; panelSerial changes whenever either is replaced, which is
	// what puts them in the refresh key without the key having to compare a
	// layout — a value carrying a picture pointer and a row slice is not
	// comparable, and a deep comparison would be a second definition of layout
	// identity to keep in step with the type.
	//
	// panelPic is the last composed picture and panelKey what it was composed
	// from; panelFresh says that picture has not reached panelImg yet, and the
	// two are written in ONE STATEMENT so a fresh picture can never be
	// presented under a texture still holding the last one.
	//
	// panelImg is ONE image for the viewer's whole life, refreshed in place
	// with the picture's pixels and reallocated only when the box changes
	// size. That is not tidiness: the refresh key holds the cell and the
	// health pair, so a selected unit that is walking or under fire
	// invalidates on every tick, and nothing in this tree disposes a texture —
	// every other holder is a cache over a bounded domain. A new image per
	// rebuild would make the panel the first unbounded allocator on the draw
	// path. It is still built on the first DRAW and never before, like every
	// other texture here, so a viewer that has been built, queried and culled
	// but never drawn holds no GPU state at all.
	//
	// panelBuilds counts compositions and is read by nothing but a test: that
	// the picture is rebuilt on a change and on no other frame is otherwise
	// unobservable from outside.
	font          *text.Font
	panelLayout   PanelLayout
	panelCardFont *text.Font
	panelSerial   int
	panelKey      panelKey
	panelPic      *image.RGBA
	// panelText is the glyphs panelPic was drawn with, re-captured each
	// frame the cached picture is presented (text.Record).
	panelText   []text.DrawCall
	panelFresh  bool
	panelImg    *ebiten.Image
	panelBuilds int

	// The mission's own statistics card, in the column's fourth slot under
	// the filler's background strip (missionCardPresent, missionpane.go;
	// `DIV-344`, the owner's own both-at-once directive). Every field mirrors
	// one of the panel's above and holds it for the same reason: the picture
	// is a per-frame composition with a key, so it is rebuilt on a key change
	// and uploaded on the fresh flag rather than every frame.
	missionCardKey   panelKey
	missionCardPic   *image.RGBA
	missionCardText  []text.DrawCall
	missionCardFresh bool
	missionCardImg   *ebiten.Image

	// Widget 8's structure readout (structureReadoutPresent): the picture is
	// a function of the three lines and the font, so it is rebuilt on a key
	// change and uploaded on the fresh flag.
	structKey   structureReadoutKey
	structPic   *image.RGBA
	structText  []text.DrawCall
	structFresh bool
	structImg   *ebiten.Image

	// selStructure is the one structure a plain click selected, valid only
	// while no unit is selected (selectedStructure).
	selStructure InspectionSubject

	fillerImg *ebiten.Image

	// The debug readout. Every field below mirrors one of the panel's above and
	// carries its reason; only the two at the end are this box's own.
	//
	// readout is the world half a front-end pushed — the clock's period, its
	// stop, the world's tick and digest — and its ZERO VALUE IS "NOT TOLD".
	// Together with the font gate it is what leaves a viewer that was never
	// pushed one drawing the frame it always drew.
	//
	// readoutHidden IS STORED INVERTED, so that SHOWN is the zero value: the
	// readout is on by default because one that must be switched on cannot
	// report the key that was pressed before it. A default set in a constructor
	// is a default every struct literal gets wrong; a default that is the zero
	// value is one nothing has to know about.
	//
	// stepCost is the ONE FIELD HERE THAT IS A QUESTION rather than an answer.
	// Both of its inputs — which cell the cursor resolved to and which unit
	// is selected — exist only in this package, so nothing on the far side
	// can push the value the way it pushes the clock's period; and this package
	// may not compute the movement law, so it cannot answer it either. What
	// crosses is therefore the question. Its zero value is nil and answers
	// nothing everywhere, which is the readout's own "not told" in another
	// shape: every viewer written before this story draws what it drew, with no
	// call added to it.
	//
	// IT IS DELIBERATELY NOT IN readoutSubject. That struct is compared with ==
	// to decide whether the box is recomposed, and a func makes a struct
	// non-comparable at RUN TIME rather than at build time — the comparison
	// panics on the second composed frame. Only this field's resolved output
	// joins the subject.
	readout       Readout
	readoutHidden bool
	readoutLayout PanelLayout
	readoutSerial int
	readoutKey    readoutKey
	readoutPic    *image.RGBA
	readoutText   []text.DrawCall
	readoutFresh  bool
	readoutImg    *ebiten.Image
	readoutBuilds int
	// The frame-rate readout (fpsreadout.go): off at load, so the zero value is
	// the default.
	fpsShown bool
	fps      fpsMeter
	fpsKey   fpsKey
	fpsPic   *image.RGBA
	fpsText  []text.DrawCall
	fpsFresh bool
	fpsImg   *ebiten.Image
	stepCost StepCostFunc

	// The notice. Every field below mirrors one of the readout's above and
	// carries its reason.
	//
	// notice and noticeKind are the whole of what this tier is told: the words
	// and which surface they go in. WHICH PART of an event text they are,
	// whether a second announcement was dropped and whether the mission is over
	// are the driver's and are not here.
	//
	// noticeOpen IS NOT STORED INVERTED, unlike the readout's flag, and the
	// difference is deliberate: a notice's ordinary state is CLOSED, so closed
	// is the zero value and every viewer built before this story — the
	// standalone developer viewer included — draws no notice with nothing set
	// on it.
	//
	// noticeLayouts is indexed by NoticeKind, so the kind selects the geometry
	// and the palette with no branch of its own; an array rather than separate
	// fields keeps that selection in one place when another surface is added.
	//
	// noticeBackdrop is the dim drawn over the map behind whatever notice is
	// open, and it is ONE FIELD BESIDE the layouts rather than a member of
	// each. The dim belongs to "a notice is open" and the layouts are per kind,
	// so a member would be two values that have to agree — and two values
	// that have to agree is how a dialogue box and an outcome box come to
	// darken the map differently for no stated reason.
	//
	// noticePortrait is which of the DIALOGUE window's two shapes is open, and
	// it sits beside the words rather than being derived from them. It is a
	// property of the FILE the window is paging through, not of the part on
	// screen, so deriving it here would apply a file's test to one part's text
	// and change the shape of the window as the player paged.
	//
	// noticeFace is the picture standing in that shape's pane, and
	// noticeFaceSerial is what makes it comparable. It OUTLIVES A PART and not a
	// window: a part that names no speaker leaves it exactly as it is, which is
	// the original's behaviour, and every path that opens or closes a window
	// settles it. The serial rather than the pointer is what the rebuild key
	// carries, because a caller may hand back the same buffer with different
	// pixels in it.
	// noticeFaceWindow is the rectangle cut from that picture — the speaker's
	// own `PortraitX1`/`PortraitY1` window, or the zero value for a speaker
	// whose record states none. It is written with noticeFace at every one of
	// those paths and is covered by the same serial: the two are one statement
	// about one speaker, so a window can never outlive the picture it was read
	// for or reach the composition without it.
	notice     string
	noticeKind NoticeKind
	noticeOpen bool
	// A load disclosure may temporarily replace the failure's picture, but
	// cannot release its input gate, including when no font can draw the page.
	noticeTerminal       bool
	help                 *helpState
	helpScroll           []*image.RGBA
	noticePortrait       bool
	noticeFace           *image.RGBA
	noticeFaceWindow     image.Rectangle
	noticeFaceSerial     int
	noticeLayouts        [4]NoticeLayout
	dialogFrame          *DialogFrame
	fogCanvas, fogInk    *ebiten.Image
	failureLoadCheck     func() bool
	noticeBackdrop       color.RGBA
	noticeBackdropCustom bool
	dialogueBackdrop     dialogueBackdropState

	// words is the resolved program-chosen word set. A viewer built by hand
	// carries the authored English; a front end over an install replaces it
	// through SetWords, which is the one writer.
	words Words
	// gameMenuContext is the live campaign/session projection installed by the
	// map's wiring tier. gamemenucontext.go owns the seam and its copy rule.
	gameMenuContext GameMenuContextSource
	// menuUp is the in-game menu's half of the popup answer. The flow raises it
	// while the panel stands over this viewer's map and lowers it when the
	// panel closes; popupOpen in popup.go is the one reader, and every
	// consequence — the stop, the camera pin, the ambient clock, the dim, the
	// map arm's input gate — follows from there.
	//
	// It is a field on the VIEWER and not on the flow alone, because three of
	// the four readers are the viewer's own and cannot reach a flow.
	menuUp bool

	docUp bool

	noticeSerial      int
	noticeKey         noticeKey
	noticePic         *image.RGBA
	noticeButtonState DialogueButtonState
	noticeText        []text.DrawCall
	noticeFresh       bool
	noticeImg         *ebiten.Image
	noticeBuilds      int

	// The inventory window lives here and not on App: a mission's open door
	// hands the front end a *Viewer, so every mission-lifetime value arrives
	// as a Viewer method. A mission always gets a brand-new Viewer, so the
	// zero values below are the reset.
	//
	// hudHidden holds the four display switches, stored inverted so that
	// shown is the zero value (hudtoggles.go).
	//
	// invKey/invPic/invFresh/invImg/invBuilds mirror the panel's presentation
	// cache for the picture wornPresent composes. invKey is a revision, not
	// the subject: invRev counts the subjects handed over and
	// SetInventorySubject alone bumps it.
	//
	// packScroll is the pack bar's first drawn element. It is not clamped at
	// rest; packScrollAt pulls it into range on every read. packImg is the
	// bar's texture, uploaded per frame like the minimap's.
	invSubject    InventorySubject
	invHasSubject bool
	hudHidden     [hudPanelCount]bool

	// The command panel (docs/1028-command-panel; commandpanel.go). It lives
	// here for invSubject's own reason at the top of this block: a mission's
	// own open door hands the front end nothing but *ui.Viewer and its seams.
	//
	// commandPanelArt is SetCommandPanelArt's own value, nil for none
	// supplied. commandPanelImg and commandHoverImg are its two upload
	// caches, the pack bar's own reuse rule: recompose every frame, reuse the
	// ebiten.Image unless its size changed.
	//
	// cmdOverlayHidden is `TOWN-092`'s own stored, signed selected-index
	// field, read here as a bool: a margin or disabled press clears the
	// DRAWN selection without touching whatever mode is actually armed
	// (commandPanelSelected's own doc).
	//
	// cmdDragCell is the drag machine's own "last cell resolved" for B2's
	// left-drag re-entry, -1 for none, reset whenever the primary button is
	// not held.
	//
	// cmdPendingGuard and cmdPendingStandGround are the one-shot bridge from
	// a Guard or Stand Ground cell press to app.go's own stance dispatch —
	// the same seam the two keyboard bindings already call, consumed once by
	// consumeCommandStancePending.
	bottomHUDArt          *BottomHUDArt
	commandPanelArt       *CommandPanelArt
	commandPanelImg       *ebiten.Image
	commandHoverImg       *ebiten.Image
	cmdOverlayHidden      bool
	cmdDragCell           int
	cmdPendingGuard       bool
	cmdPendingStandGround bool

	// The character pane's own shipped art and its one-shot menu request
	// (missionpane.go). paneFigureArt and paneStatsArt are the two
	// mode-switched 160x242 bodies `TOWN-356` names HumanBackR and TextBackR;
	// paneCornerArt is the nine corner bitmaps. All four are pushed by the tier
	// that can open an archive, on SetCommandPanelArt's own shape, and a viewer
	// holding none composes the authored fill instead (or, for the filler, no
	// fourth layer at all — columnFillerPresent's own doc).
	//
	// paneMenuRequest is rect F's own one-shot: the corner posts `0x416`, the
	// same message `MENU-ESC-010` reads on the frame window's Esc arm, and the
	// front end drains it through TakeCharacterPaneMenu. It is a request and
	// not a direct call for TakeInventoryEquip's own reason — this package
	// raises no surface of its own.
	//
	// paneCornerGrab is whether the gesture in progress BEGAN on one of the
	// six corners — invGrab's own latch, for the same reason: what a press is
	// has to be fixed as the button goes down and hold for the whole press.
	paneFigureArt   TownPane
	paneStatsArt    TownPane
	paneCornerArt   *CharacterPaneCornerArt
	paneFillerArt   image.Image
	paneMenuRequest bool
	paneCornerGrab  bool

	invRev    int
	invKey    int
	invPic    *image.RGBA
	invFresh  bool
	invImg    *ebiten.Image
	invBuilds int
	// portraitOwner/portrait are SetUnitPortrait's own value: the FLAT
	// picture of one unit and which unit it belongs to, 0 for none (0141;
	// `UNIT-PICT-035`). They sit here rather than on InventorySubject because
	// they describe whatever is SELECTED and that is very often not the
	// subject — the subject is the party's own character and this box follows
	// the selection (inventory.go's dollSubject).
	portraitOwner uint32
	portrait      *image.RGBA

	packScroll int
	packImg    *ebiten.Image
	// packStarPhases belongs to visible grid slots, not item identities
	// (ITEM-STARPHASE-099). packStarScroll/packStarCols are the last geometry
	// reconciled by syncPackStarPhases so scrolling can move phases with the
	// exact asymmetric rule instead of resetting every item.
	packStarPhases []uint32
	packStarScroll int
	packStarCols   int

	// invGrab is whether the gesture currently in progress BEGAN on the open
	// inventory window (hotfix: the window swallows its own clicks).
	//
	// It is a LATCH ON THE GESTURE and not a per-frame point test, for
	// boxing's own reason one field block up: what a press is has to be fixed
	// as the button goes down and hold for the whole press, or dragging off
	// the window would start panning the map halfway through a gesture the
	// player made on a window.
	//
	// It is written by command (command.go) rather than by dragIntent's anchor
	// branch, unlike boxing, and the difference is an ORDERING one: step runs
	// before command on the map arm, and dragIntent's anchor branch pans zero
	// by construction, so a latch taken at the press in command is already
	// standing on every frame that can pan. Written in the anchor branch it
	// would have to be read by a statement that runs before it.
	invGrab bool

	invClickCell    int
	invClickFrames  int
	invEquipTap     bool
	invEquipRequest int

	// docRequest is the one-shot campaign-documents request, raised by the tier
	// above through RaiseDocuments and drained by the App on the map arm. It is
	// a PLAIN BOOL and not an index-plus-one, unlike invEquipRequest above it,
	// because the request carries nothing: the panel shows the whole collection
	// and does not open on an element.
	docRequest bool

	// invWornClickCell/invWornClickFrames/invUnequipRequest are the same
	// double-click tracking, run over the worn box's twelve cells instead of
	// the pack (0151, defect 4). They are a separate window and a separate
	// one-shot request from the three fields above: a press on a worn cell
	// must not extend or spend a count a pack-cell press started, and the
	// reverse, so command (command.go) decrements both counts unconditionally
	// on every swallowed frame and matches a press against only the one whose
	// box it landed in.
	//
	// invUnequipRequest is the worn slot index a matching press named, held
	// as INDEX PLUS ONE — invEquipRequest's own "zero still means nothing
	// pending" tilt, needed here for the identical reason: slot index 0 is
	// itself a real cell. TakeInventoryUnequip (inventory.go) is the one
	// reader and clears this in the same statement it reads it in.
	invWornClickCell   int
	invWornClickFrames int
	invUnequipRequest  int

	// invDollUnequipRequest is the THIRD one-shot request (1005, "the
	// interactive doll"), TakeInventoryDollUnequip's own value, raised by
	// EITHER of the doll's two new gestures — a tap on a slot that never
	// crossed TapSlop, or a drag lifted off the doll and released into the
	// pack — command.go's own two writers for the one field, each holding
	// the same INDEX PLUS ONE encoding invUnequipRequest already does.
	invDollUnequipRequest int

	// dollSuppressOwner/dollSuppressSlot/dollSuppressPic/dollSuppressMask are
	// SetDollSuppressedFigure's own value (inventory.go): which entity, which
	// 1-based slot, which picture and which picture's OWN MASK a drag lifting
	// that slot off the doll is standing in for the subject's own composed
	// figure and SlotMask with, pushed once per frame by pkg/game's
	// refreshDollDrag and read by dollSubject and dollFigureSlotAt. Slot 0 (a
	// fresh Viewer's own zero value) is no suppression at all.
	//
	// dollSuppressMask MUST be the mask composeInventorySubject built beside
	// dollSuppressPic, never invSubject's own mask substituted in its place
	// (1005 round-1 adversarial review): the two pictures paint different
	// pixels for the lifted slot's own layer, and a hit test or hover popup
	// reading the wrong mask would name a slot the drawn picture does not show
	// there, or miss the layer the suppressed picture exposes underneath —
	// the exact defect DIV-085 rules out for the ordinary figure, restated here
	// for the suppressed one.
	dollSuppressOwner uint32
	dollSuppressSlot  int
	dollSuppressPic   *image.RGBA
	dollSuppressMask  *SlotMask

	// dragCandKind/dragCandIdx/dragActive/dragIcon are the drag machine's own
	// state (1005, "the interactive doll"): a press landing on a pack cell or
	// a doll slot is a CANDIDATE origin from the moment it lands, captured by
	// command.go alongside that press's own double-click bookkeeping; it
	// becomes an ACTIVE drag — dragIcon holding the picture carried on the
	// cursor from then on — only once the SAME accumulator TapSlop already
	// judges a map gesture by (v.dragMoved) crosses that one threshold. A
	// drag is view state and issues no command of its own: command.go reads
	// the release point against the doll box and the pack bar and raises the
	// same one-shot requests a double-click or a tap already would.
	//
	// dragCandKind IS THE LIVE FLAG, dragMoved's own idiom: dragNone is a
	// fresh Viewer's own zero value and every write clears it back to that
	// the moment a gesture's release has been resolved, so a stale candidate
	// from a spent press is never read by the next one.
	dragCandKind uint8
	dragCandIdx  int
	dragActive   bool
	dragIcon     *image.RGBA
	dragImg      *ebiten.Image

	// invDropRequest/invDropWorn/invDropX/invDropY are the ground drop's own
	// one-shot request (1005 round 2, `ITEM-DROP-008`, `DIV-088`): a drag
	// released outside every inventory box, during a mission, on
	// invDropRequest's own INDEX PLUS ONE encoding — invEquipRequest's own
	// tilt, needed for the identical reason: a container element index of 0
	// and an equipment slot index of 0 are both real cells. TakeInventoryDrop
	// (inventory.go) is the one reader and clears this in the same statement
	// it reads it in.
	//
	// invDropWorn TELLS THE TWO DRAG ORIGINS APART — a doll slot (true) or a
	// pack element (false) — because a THIRD one-shot field cannot, on its
	// own, say which container invDropRequest-minus-one names, unlike
	// invEquipRequest and invUnequipRequest, which never need to.
	//
	// invDropX/invDropY are the cell the cursor named at release
	// (`v.groundCellAt`, world cell units), carried through unresolved: this
	// package states no window geometry of its own, since `pkg/sim`'s own
	// dropToGround (drop.go) applies ITEM-DROP-008's Chebyshev window against
	// the entity's OWN position at APPLICATION time rather than at release
	// time.
	invDropRequest int
	invDropWorn    bool
	invDropX       int32
	invDropY       int32

	// gold is the purse transaction's own draft, held share and request
	// (goldmodal.go).
	gold    goldState
	goldImg *ebiten.Image

	spellbook      []SpellEntry
	spellbookOwner uint32
	spellbookHeld  bool // a unit is selected; entity id 0 is a real owner
	selectedSpell  uint32
	spellbookFixed bool       // the installed 24-cell catalog, rather than a custom compact book
	spellArmed     bool       // current spell and armed Cast mode are independent
	spellNeedsBook bool       // book-chosen: live only while the book is shown
	castOnce       bool       // armed by C or the Cast cell over a closed book: one cast, then the book closes
	quickSpells    *[4]uint32 // session-owned; never retained by an actor
	itemCast       *itemCastSelection
	itemCastSink   MapItemCast

	// autocastSink is SetAutocastSink's own value (spellbook.go): where the
	// autocast toggle leaves this package, nil for a viewer never given one.
	autocastSink MapAutocast
	// formationSink is SetFormationSink's own value (formationcommand.go):
	// where Ctrl+F leaves this package, nil for a standalone viewer.
	formationSink    MapFormation
	defendSink       MapDefend
	structureUseSink MapStructureUse
	// retreatSink is SetRetreatSink's matching value for Ctrl+W's persisted
	// three-state player command.
	retreatSink          MapRetreat
	playerRetreatSink    MapPlayerRetreat
	playerRetreatBlocked bool
	spellbookImg         *ebiten.Image

	// spellBolts is SetSpellBolts' own value (spellbolt.go): this frame's
	// spells in flight, bursts and pass-tagged retained cell overlays, replaced
	// whole every frame the tier above pushes one.
	spellBolts []SpellBolt

	// healSprites is the presentation-only rising shower of frames from the
	// shipped healing sheet, handed over from semantic positive Heal events. It
	// is replaced every frame and never feeds input, targeting or sim state.
	healSprites []HealSprite

	// effectImages is the texture behind each drawn spell sheet frame, keyed by
	// the frame's own pointer identity and built on that frame's first draw. It
	// is nil until a draw asks for one, so a viewer that has been constructed,
	// queried and culled but never drawn holds no map at all — staticImages'
	// own rule, one sheet format over.
	effectImages map[*terrain.EffectFrame]*ebiten.Image

	messages     messageList
	messagePic   *image.RGBA
	messageBuilt int
	messageFresh bool
	messageImg   *ebiten.Image
	// messageBlit is messagePic without its glyphs while the text overlay
	// draws them, and messageText those glyphs (glyphPicture).
	messageBlit   *image.RGBA
	messageText   []text.DrawCall
	messageSmooth bool

	// itemPopupImg is the item popup's own uploaded texture (0151, defect 5),
	// packBarPresent's own reuse rule (inventory.go): itemPopupPresent
	// recomposes the picture every frame it draws, and this field is only
	// reallocated when that picture's own size changes, never on every
	// frame's own WritePixels.
	itemPopupImg *ebiten.Image

	// cursorX/cursorY is the cursor position the last camera step observed, in
	// screen pixels, and hasCursor whether one has ever been observed.
	//
	// The readout resolves its cell at the DRAW, from these, rather than
	// storing a cell: which cell a pixel names is a function of the camera, and
	// the camera moves between the step and the draw on any frame that pans. A
	// stored cell would be a per-frame answer kept in a field — the very shape
	// the lattice's own comment refuses.
	//
	// hasCursor is not decoration: without it a viewer that has never been
	// stepped resolves (0, 0), which is a real cell on every map, and the
	// readout would state a cell nobody pointed at.
	cursorX, cursorY int
	hasCursor        bool

	// hoverLitID/hoverLitOK cache one frame's hoverInspection() unit subject,
	// for spellSpriteFactor's own max-composed hover gain (hotfix, owner
	// report 1, DIV-1349: hovering any unit should brighten its own sprite,
	// as if it were lit — most visible at night, same as the owner's report
	// on structure light masks).
	//
	// REFRESHED ONCE PER FRAME, FROM drawFrame, NEVER FROM INSIDE
	// planeSprites/spellSpriteFactor. hoverInspection calls inspectionAt,
	// which itself calls planeSprites to pixel-test the topmost sprite under
	// the cursor — reading hoverInspection from inside the very pass that
	// FEEDS it would recurse into planeSprites a second time on every
	// candidate cell. Caching the answer once, before that pass runs, is
	// what breaks the cycle.
	hoverLitID uint32
	hoverLitOK bool

	// winCursorX/winCursorY is the SAME observation one door earlier: the
	// cursor position as the window reported it, before step maps it onto the
	// frame, and hasWinCursor whether one has ever been observed.
	winCursorX, winCursorY int
	hasWinCursor           bool

	// primaryDown is whether the primary button was down on the tick step last
	// ran, stored beside the position above and for the same reason: the
	// edge-scroll term is suppressed while it is held, and the edge-arrow
	// cursor must be suppressed on exactly the same ticks.
	primaryDown bool

	// ctrlLatch, altLatch and shiftLatch are the mission map's own three
	// modifier latches (`AI-KEYMOD-059`: three globals, each set on the key's
	// down, cleared on its up, and cleared wholesale on focus loss).
	//
	// SHIFT CHANGES NO CURSOR and is stored anyway. `AI-CURSOR-226` establishes
	// that the Shift latch is read at exactly one address in the whole cursor
	// routine and that address is inside the cascade an ordinary hover does not
	// run, so the hover cascade takes Ctrl and Alt only. Its mission-map
	// consumer is the SELECTION routine (`AI-SELECT-122`), which reads the
	// latch — so it is one of three fields here rather than a parameter
	// threaded from whichever tick happened to sample it.
	//
	// THEY ARE STORED RATHER THAN PASSED because the cursor cascade is a
	// function of the viewer's state alone: `advanceCursorManager` is called
	// from `step`, which is handed an `Input`, while `mapCursorName` is reached
	// from a draw as well. `primaryDown` one field up is stored for the
	// identical reason and by the identical statement.
	ctrlLatch, altLatch, shiftLatch bool

	// groups is the ten numbered selection groups the digit row assigns and
	// recalls (`AI-KEY-125`'s digit rows, `AI-SELECT-122`'s group clauses).
	// Slot n holds whatever ids Ctrl+n was pressed over, in the selection's own
	// ascending order.
	//
	// IT STORES IDS AND NOT ENTITIES. A group outlives the entity snapshot it
	// was assigned from -- the snapshot is replaced whole every tick -- so a
	// recall filters through presentSelected the same way every other reader of
	// a selection does, and a group whose members have all died recalls the
	// empty set rather than a stale one.
	groups [10]selection

	// secondaryDown is whether the SECONDARY button is down on this tick, and
	// rightPanned whether the press currently in progress has already moved the
	// camera (`AI-INPUT-127`).
	//
	// THE RIGHT BUTTON PANS AND THE LEFT ONE DOES NOT, on the mission map. That
	// is the inversion this story is: `AI-INPUT-121` gives the left button as
	// inactive-down-to-start, active-up-to-act with a marquee in between, and
	// `AI-INPUT-127` gives the right one as capture-to-pan, up-to-cancel.
	// rightPanned is `AI-INPUT-127`'s own "marked a drag": a marked drag
	// performs no cancel on the way up.
	secondaryDown, rightPanned bool

	// paneSecondaryGrab is a secondary-button gesture whose down edge belonged
	// to the mission character pane. The pane owns that gesture until release,
	// so moving off its body cannot start a map pan from a down edge the map
	// never received. command writes the edge latch after step has sampled the
	// same frame; the down frame itself has zero drag delta, so no movement can
	// escape before the latch is raised.
	paneSecondaryGrab bool

	// rightDragX/rightDragY is where the secondary button was last sampled, the
	// right-drag's own counterpart of dragX/dragY. It is a separate pair rather
	// than a shared one because both buttons may be held at once and each
	// gesture must measure from its own last position.
	rightDragX, rightDragY int
	rightDragging          bool

	// pointerDeferred is whether the CALLER composes this viewer's pointer,
	// after whatever it draws over the frame this viewer returns (DeferPointer).
	// False is the standalone developer viewer, which has nothing above it and
	// draws its own at the end of drawFrame.
	pointerDeferred bool

	// sel is the selected units' ids, ascending, or the nil zero value for
	// none.
	//
	// It lives HERE because the pick and the highlight both need the camera,
	// the extent and the snapshot, all of which are already here; and it is
	// written only by command, which is unexported, so the standalone viewer
	// has no route to a selection at all. It is front-end state alone — never
	// world state, never hashed, never serialized.
	//
	// NOTHING PRUNES IT. An id the current snapshot no longer holds stays in
	// the set and is skipped where the set is read, so what is selected is a
	// function of the taps and releases alone and never of which ticks happened
	// to run between two of them.
	sel selection

	// selectionPicked is the units this frame's latest selection form put into
	// sel. replySelection hands them on once per frame and clears it.
	selectionPicked []uint32

	// armed is whether an attack has been ARMED and not yet spent. What is
	// being reconstructed orders an attack in two clicks — a control arms a
	// mode, and only a later click aims it — and this is the mode.
	//
	// IT LIVES HERE, BESIDE THE SELECTION, and not on the flow: the gate it is
	// raised through reads the selection, which is this object's, and two owners
	// would be two lifetimes to keep level. It also settles what happens when
	// the map screen is left for nothing — a viewer is dropped there, and the
	// mode goes with it, so no statement anywhere has to remember to lower it.
	//
	// It is FRONT-END STATE ALONE: nothing pushes it, no seam carries it, and
	// the far side cannot ask whether it is up.
	armed bool

	// command is the ARMED AIMED STANDING ORDER — none, patrol or march. It
	// lives here beside armed, for armed's own reasons: it is raised over the
	// selection this object holds, and a viewer dropped on leaving the map
	// screen takes it away with no statement anywhere having to lower it.
	//
	// IT IS ONE BYTE AND NOT TWO FLAGS. Two flags could both be up, and there
	// is no press defined for that; one byte cannot express it.
	//
	// It is named aimed and not command because this type already has a method
	// of that name — the impure shell around decide.
	//
	// IT IS A SEPARATE FIELD FROM armed AND THE TWO ARE NEVER MERGED, for the
	// reason attackHeld is separate from armed: they are raised by different
	// keys, through different gates, and the attack arm has a second writer
	// that is a LEVEL. What keeps them from both claiming one press is not this
	// field's type — it is armCommand lowering the attack mode, armAttack
	// lowering this, and command's own gesture treating an armed order as the
	// answer whatever the modifier is doing.
	aimed uint8

	// attackHeld is the MODIFIER LATCH — the second writer of the attack
	// mode, and the one the decode describes.
	//
	// IT IS A LEVEL AND NOT A TOGGLE: it is assigned this tick's modifier state
	// whole, so releasing the key lowers the mode with no second press. What is
	// being reconstructed sets its gate on the key going down, clears it on the
	// same key coming up, and clears every gate together on focus loss — a
	// key-held latch by construction, with no second model fitting it.
	//
	// IT IS A SEPARATE FIELD FROM armed AND THE TWO ARE NEVER MERGED. The key's
	// mode is SPENT by the press that consumes it; this one is not, because the
	// player is still holding the key. One field would make the consuming press
	// lower a mode the modifier is still asserting, which is a toggle a press
	// cancels — exactly the shape this is not. Kept apart, the one-line spend in
	// command is correct unedited.
	//
	// IT IS UNGATED, and that asymmetry is the decode's rather than ours: the
	// ownership gate is what the COMMAND PANEL's arming path reads, and the key
	// is this build's stand-in for that path; the modifier path reads no player
	// value at the hover or at the click.
	attackHeld bool

	// The attack pointer. The picture pushed for it, its upload cache, and what
	// the engine was last asked for.
	//
	// attackPointer is NIL for none supplied, which is the developer viewer's
	// state and the state of a front-end whose install would not yield the art;
	// the authored cross is what makes that visible.
	//
	// attackPointerTex is the cursorTexture both cursor draw sites share. It
	// rewrites the engine texture when the SOURCE PICTURE changes and not
	// otherwise, which is what the manager's ten attack frames need. It
	// replaced a fresh flag set once per viewer at map open and cleared by the
	// first upload, under which frames 1..9 never reached the screen
	// (cursortexture.go).
	//
	// pointerHidden is what the engine was last told about the SYSTEM cursor,
	// and its zero value is the engine's own default, so no baseline call is
	// owed. It is a record of a request rather than a second source of truth:
	// what decides is attackPointerPresent, and this only stops the request
	// being repeated every frame. A viewer inside an App session leaves the one
	// engine call for the whole window to App.applyPointerMode, which reads the
	// shared manager's own cache instead (pointerModeChange's own header).
	attackPointer    *image.RGBA
	attackPointerTex cursorTexture
	pointerHidden    bool

	// mapCursorTex is the upload cache for the mission map's own cursor
	// outside attack mode (1031 B1-B3, missioncursor.go): a separate instance
	// from attackPointerTex, on that field's own reason — the two can hold
	// different pictures on the same frame's transition, and one cache
	// comparing against the wrong site's last upload would re-decide nothing
	// wrong but re-upload every frame.
	mapCursorTex cursorTexture

	cursorMgr *CursorManager

	// cursorAttackFromMode is true for exactly as long as the "attack" name
	// currently stored on cursorMgr came from advanceCursorManager's
	// attackShown() branch rather than from mapCursorName's own hover
	// selection (1031 B3): both write the same string, "attack" is a
	// legitimate hover selection in its own right, and only THIS field tells
	// the two apart. It is what makes the one tick advanceCursorManager runs
	// behind the mode (its own header) invisible at the map's own cursor too:
	// on the tick the mode comes down, the manager is still carrying the
	// PREVIOUS tick's attackShown() write, and mapCursorPresent must still
	// refuse to draw it — adversarial pass 2's F1 return, correcting a guard
	// that read the picture's NAME for that instead, and refused every
	// hostile hover's own legitimate "attack" along with it.
	cursorAttackFromMode bool

	// localOwner is which roster slot the LOCAL PARTICIPANT holds, pushed by
	// the tier that owns the world.
	//
	// It is one half of the arming gate's comparison; the other half is the
	// selected entity's own Owner. The comparison is made HERE and not behind
	// the seam because that is where the thing being reconstructed makes it —
	// the routine is a view method reading one view field — and because two
	// uint32 name no simulation type, so making it here costs this package
	// nothing it is not allowed to spend.
	//
	// ZERO IS NO LOCAL PARTICIPANT ESTABLISHED, sharing the far side's own zero
	// for "owned by nobody"; roster slots are 1-based, so no slot is shadowed.
	// A zero compares against nothing and the gate is then open, which is this
	// build's honest state: nothing in this tree performs a session join, and
	// which slot a participant holds is that path's answer and not the map's.
	localOwner uint32

	// fogPlane is a participant's fog-of-war plane, one byte per map cell in
	// row-major order — FogUnseen/FogExplored/FogVisible — and
	// fogCols/fogRows are its OWN dimensions, all three pushed together by
	// SetFog.
	//
	// THEY CROSS AS A PLAIN []byte AND TWO INTS, objectCells' and
	// blockedCells' own shape a few fields up: this package may not import
	// pkg/sim, and the plane is per-participant VIEW rather than content this
	// tier could derive on its own — the tier that owns the world builds it
	// and pushes it here exactly as SetEntities and SetSacks already push
	// their own per-tick state.
	//
	// A NIL OR EMPTY fogPlane MEANS "NO FOG": fogAt (fog.go) answers FogVisible
	// for every cell while it is empty, so a viewer nothing has ever called
	// SetFog on — every viewer built before this story, the standalone
	// developer viewer, the map picker — draws exactly the frame it drew
	// before this story existed.
	//
	// fogCols/fogRows are the PLANE's own dimensions and never v.grid's (0118
	// plan R-4): fogAt bounds-checks against these two, so a plane whose size
	// disagrees with the terrain grid answers unseen for the cells outside it
	// rather than panicking or reading past the slice.
	fogPlane         []byte
	fogCols, fogRows int

	// fogReveal is the debug reveal's own flag. While set, fogAt answers
	// FogVisible for every cell without touching fogPlane at all, so AC-11's
	// "turning it off restores the previous drawing exactly" holds by
	// construction — there is no plane edit to undo.
	fogReveal bool

	// The terrain colors stay cached for this grid and tileset; the uploaded
	// texture is rebuilt only when its dimensions change.
	minimapColours []color.RGBA
	minimapImg     *ebiten.Image

	// minimapGrab latches a primary gesture that BEGAN on the minimap (0140):
	// raised by a press landing on its box, lowered by the release, and read by
	// command.go alone. It is invGrab's own shape one box over, and it exists
	// for the same two reasons — a gesture that started on this box stays this
	// box's until the button comes up, and while it is up the view follows the
	// cursor, which is what makes the green view outline draggable rather than
	// only clickable.
	minimapGrab bool

	minimapActX, minimapActY int

	// The floating damage numerals. numeral.go holds the rules; these are the
	// three pieces of state they need.
	//
	// numeralHP is the health this viewer last saw for each entity, and it is the
	// ONLY source of a damage figure in the tree — the front end subtracts, the
	// way the original's client does, because what crosses the seam is a health
	// level and never a delta. It is rebuilt from each frame's own entities, so a
	// viewer reused across two maps cannot attribute one map's health to the
	// other's entity of the same id.
	//
	// numeralsHidden IS STORED INVERTED, so that SHOWN is the zero value: the
	// display is on from the moment a map opens (it is the original's own
	// default), and a default that is the zero value is one no struct literal can
	// get wrong. It is the readout's own idiom, one field family over.
	// numeralTick is the ambient animation count the drift was last stepped at,
	// so a frame that crossed several ticks drifts a figure several steps and one
	// that crossed none drifts nothing. Its zero value is the counter's own, and
	// nothing drifts before the first record exists.
	//
	// numeralImgs is the upload cache, one texture per PLACEMENT SLOT rather
	// than per record: the placement list is rebuilt every frame and is short,
	// and a texture per record would tie GPU state to a value the ingest
	// appends to. A slot's texture is reallocated only when the picture at that
	// slot changes size, which is the panel's own rule; nothing in this tree
	// disposes a texture, so the cache is bounded by the most figures ever live
	// at once and not by how many blows have landed.
	numerals       []damageNumeral
	numeralHP      map[uint32]int
	numeralTick    uint32
	numeralLast    time.Time
	numeralPaused  bool
	numeralsHidden bool
	numeralImgs    []*ebiten.Image

	// healthBarsHidden is the show-health setting's state (round 3,
	// `keyboard.tsv` row 48's `Ctrl`+`H`). IT IS STORED INVERTED for
	// numeralsHidden's own reason one field family up: the bars are drawn from
	// the moment a map opens, and a default that is the zero value is one no
	// struct literal can get wrong.
	//
	// Drawing reads it in one place, entityStatusShown (overlay.go), which the
	// status bar walk asks, so "hidden" is a property of that one walk rather
	// than of several drawing sites agreeing to stay quiet.
	healthBarsHidden bool

	// A landed blow's SOUND (0126 spec; sound.go holds the rule and the
	// throttle, sounddev.go the one device this tree ships).
	//
	// soundPlayer and soundBank are what SetAudio hands over, and NEITHER
	// CARRIES A Settings VALUE — that is the one place this story's plan
	// changed shape from its own draft. plan.md wrote SetAudio taking a third
	// argument, audio.Settings, but audio.Player.Play(Sample, Placement) takes
	// none, so a viewer holding settings would have nowhere to apply them; the
	// volume and the mute belong to the CONCRETE DEVICE instead, which is the
	// one thing on this seam that can actually reach audio.Stereo.
	//
	// Both are nil for every viewer nothing has called SetAudio on — the
	// standalone developer viewer, every hand-built test viewer, and any
	// front-end that failed to open a sound device — and nil is a LAWFUL
	// STATE for either one alone or both together (AC-11): playSlotAt refuses
	// before either is read, so a nil device is silence and not a crash waiting
	// for a frame that happens to try to play something.
	//
	// voices is each drawable's voice memory (sound.go), keyed by entity id.
	// IT IS REBUILT EVERY stepSound CALL FROM THE FRAME'S OWN ENTITIES,
	// carrying an entry forward only for an id still present — numeralHP's own
	// reason, one field family over: a viewer reused across two maps must not
	// let a stale timestamp from one map's entity gate a different entity that
	// happens to reuse its id in the next one.
	soundPlayer             audio.Player
	speechPlayer            audio.Player
	audioScope              *AudioScope
	soundBank               SoundBank
	voices                  map[uint32]voiceMemory
	entrySounds             []MapEntity
	soundMessages           []soundMessage
	commandAcknowledgment   func(VoiceGesture, []uint32, time.Time)
	retreatSpoken           []uint32
	selectionAcknowledgment func([]uint32, time.Time)

	// Mission ambience is presentation-only. Static candidates are derived
	// once from the immutable map/object bundle; Wall of Fire cells are replaced
	// from live world effects and the controller owns only wall-clock/RNG state.
	ambient         *ambientController
	ambientStatics  []ambientStaticCell
	ambientWallFire []image.Point

	quit bool
}

// cacheKey is what the ground texture cache is keyed by: the resolved (slot,
// sub-cell) and — from 0100 on — the sky tint the cell's texture was
// built with. The tint belongs beside slot and sub for the reason the ramp
// row belongs in spriteTextureKey: the pixels genuinely differ, so a cell
// resolved under one band's tint must not be handed back to a later band
// that resolves to the same slot and sub-cell.
type cacheKey struct {
	slot, sub int
	tint      [3]uint8
	dirt      uint8 // zero means absent; 1..4 name dirt.bmp sub-cells
}

// NewViewer builds a viewer over a decoded map grid and a loaded tileset, with
// no static-object layer at all: no bundle, and both of that layer's switches
// off. It is the pre-0017 constructor, unchanged in signature and in behaviour.
//
// It DELEGATES rather than duplicating: every rule the constructor below
// states — the validation order, the projection, the level grid, the world
// sync — is stated once, so the two entry points cannot come to validate,
// project or clamp a map differently. Keeping this signature is also what
// leaves every existing caller and every existing test in this package
// untouched; widening it would have grown an argument at each of them that
// none of them uses.
func NewViewer(title string, g terrain.Grid, set *terrain.Tileset) (*Viewer, error) {
	return NewViewerWithStatics(title, g, set, nil, false, false, terrain.AnimGateTiles, nil, false)
}

// NewViewerWithStatics builds a viewer over a decoded map grid, a loaded
// tileset and, optionally, the map's static-object bundle. It validates its
// inputs so a malformed map surfaces as an error before any window opens.
//
// A grid carrying a valid altitude layer is projected here and the viewer
// starts in displaced mode. An altitude layer of the wrong length selects
// flat mode and is not an error; validateGrid checks the tile grid alone.
//
// statics is the object-class bundle, or nil; art is whether the sprites
// paint and markers whether the diagnostic cross draws. The bundle arrives
// here or not at all: there is no setter. structures and structureArt follow
// the same rule beside it. With a bundle the constructor builds the
// placement lists once per geometry, purely and without a GPU, the displaced
// list inside the validAltitudes(g) branch that builds proj and levels. A nil
// bundle builds no placement.
//
// A displaced placement's height is proj.AnchorHeight and its vertical origin
// proj.MinV, the raster's Render.OriginY, so a placement lands on the same
// world point as in the raster. Both are passed unnegated; StaticAnchor
// performs the one subtraction. The map's own type-4 records travel in the
// Grid with the tile, altitude, object and block layers.
func NewViewerWithStatics(title string, g terrain.Grid, set *terrain.Tileset, statics *terrain.StaticSet, art, markers, objectAnim bool,
	structures *terrain.StructureSet, structureArt bool) (*Viewer, error) {
	if err := validateGrid(g); err != nil {
		return nil, err
	}
	if set == nil {
		return nil, errNilTileset
	}
	v := &Viewer{
		textSmoothingEnabled: true,
		tooltip:              &tooltipController{delay: DefaultTooltipDelay},
		staticSet:            statics,
		staticAnimGate:       objectAnim,
		baseTileWords:        g.Tiles,
		title:                title,
		grid:                 g,
		set:                  set,
		cam: camera.New(g.Width, g.Height,
			MissionViewportSize().X, MissionViewportSize().Y),
		// A viewer that is never laid out still has a usable placement, so the
		// three input doors map rather than refuse. Fitting the frame into a
		// window of the frame's own size is the identity.
		place:  frame.Fit(MissionFrameW, MissionFrameH, MissionFrameW, MissionFrameH),
		frameW: MissionFrameW,
		frameH: MissionFrameH,
		cache:  make(map[cacheKey]*ebiten.Image),
		// Water animates by default at the speed index map load selects, as the
		// game does (TERR-ANIM-008/009) — the game's own truncated millisecond
		// widened to the clock's unit, never a microsecond quotient of its own.
		anim:              terrain.NewTicker(terrain.SpeedIndexPeriod(terrain.DefaultSpeedIndex)),
		speedIndex:        terrain.DefaultSpeedIndex,
		animate:           true,
		showStaticArt:     art,
		showStaticMarkers: markers,
		showStructureArt:  structureArt,
		// The panel's appearance is the authored value from construction, so a
		// front-end that hands over a font gets the shipped panel without also
		// having to choose a layout. A caller that wants another supplies it
		// through SetPanelLayout; the panel itself is off until a font arrives.
		panelLayout: CompactPanelLayout(nil),
		// The readout's appearance, on the panel's own rule and for its own
		// reason. Its VISIBILITY is not set here: shown is the zero value, so
		// there is nothing to set.
		readoutLayout: AuthoredReadoutLayout(),
		// The notice surfaces' appearance, on the panel's own rule and for its own
		// reason. None is OPEN here — closed is the zero value — so a viewer
		// that is never pushed one draws exactly what it drew before this story,
		// and the standalone developer viewer never gains a box.
		noticeLayouts: [4]NoticeLayout{AuthoredDialogueLayout(), AuthoredOutcomeLayout(), AuthoredSuccessLayout(), AuthoredFailureLayout()},
		// The authored word set, on the layouts' own rule and for their reason.
		// The layouts above already carry their control words, so a viewer that is
		// never told an install's words remains self-contained.
		words: AuthoredWords(),
		// The day/night cycle's cache and its switch. The seed is the cycle-off
		// sun this tree already drew; the switch is on, which is the original's
		// compiled default, and it decides the arm the first relight takes rather
		// than anything drawn before one.
		sun:            terrain.DefaultDaytime,
		timeFlow:       true,
		ambientStatics: ambientStaticCells(g, statics),
		// The dim behind them, on the same rule and for the same reason. It is
		// never DRAWN here — nothing is drawn without a notice open — so the
		// standalone developer viewer carries the value and composes exactly the
		// frame it composed before this story.
		noticeBackdrop: AuthoredNoticeBackdrop(),
		// -1 is "no cell", the drag machine's own sentinel (commandpanel.go):
		// 0 is a real cell index and cannot stand for none.
		cmdDragCell: -1,
	}

	// The flat list is unconditional: flat mode is reachable on every map, both
	// as the fallback for an unusable altitude layer and as the diagnostic
	// SetFlat deliberately selects over a valid one. A nil lift and a zero
	// origin ARE the flat geometry, mirroring the nil-safe lift the raster path
	// passes.
	v.staticsFlat, v.staticCounts, v.animFlat = terrain.StaticPlacements(g, statics, nil, 0, objectAnim)

	// The structure layer's flat list, beside the object layer's and under the
	// same rule: a nil corner accessor and a zero origin ARE the flat geometry.
	v.structuresFlat, v.structureCounts, v.structureAnimFlat = terrain.StructurePlacements(g, structures, nil, 0)
	v.structureSet = structures
	if structures != nil {
		v.structureInfo = make(map[uint32]*terrain.StructureClass)
		for _, rec := range g.Structures {
			v.structureInfo[rec.ID] = structures.Classes[rec.Key&0xff]
		}
	}

	if validAltitudes(g) {
		p := terrain.Project(g.Altitudes, g.Width, g.Height)
		v.proj = &p
		// Built from v.sun and not from the constant it was seeded with, so
		// construction and every later relight take the light by one route. At
		// construction the two are the same value.
		v.levels = terrain.LevelGrid(g.Altitudes, g.Width, g.Height, v.sun)

		// The census is geometry-independent, so this build's copy is the one
		// already recorded and is deliberately discarded rather than stored
		// twice: which cells resolve is a question about bytes and classes, and
		// two records could only ever agree or reveal a bug in the builder.
		v.staticsDisplaced, _, v.animDisplaced = terrain.StaticPlacements(g, statics, v.proj.AnchorHeight, v.proj.MinV, objectAnim)

		// The structure layer's displaced list. Its vertical origin is the same
		// proj.MinV the object side takes — this viewer's world Y is Vertex(c,r)
		// - MinV, which is the raster's Render.OriginY — so one placement lands
		// on the same world point in the window as in the raster.
		//
		// ITS HEIGHT TERM IS proj.Altitude AND NOT proj.AnchorHeight, which is the
		// one place this layer's geometry parts company with the object layer's. A
		// structure stands at ONE height, sampled at its RECTANGLE'S CENTRE —
		// and that centre falls on a half-integer whenever an extent is odd, so
		// the sample is bilinear over four corner heights and a per-cell helper,
		// which can only answer for a whole cell, cannot express it. AnchorHeight
		// is the four-corner mean of one cell, which is the odd-by-odd case of
		// this and not the general one.
		v.structuresDisplaced, _, v.structureAnimDisplaced = terrain.StructurePlacements(g, structures, v.proj.Altitude, v.proj.MinV)
	}

	// The merge is taken over the FLAT pair, and it serves both geometries: the
	// two pairs carry the same cells in the same order, and the merge compares
	// rectangle rows alone.
	v.planeOrder = terrain.PlaneOrder(v.structuresFlat, v.staticsFlat)

	// Syncing HERE, and not only in the overlay setters, is the whole of AC-9:
	// pkg/game's front-end reaches NewViewer and never touches either overlay,
	// so a viewer that only learned its extent from an overlay toggle would run
	// the game displaced over a camera still clamped to Rows*CellSize, clipping
	// up to the full altitude spread off the bottom of every map.
	v.syncWorld()
	return v, nil
}

// staticLists is the built pair the viewer draws from right now: the displaced
// list and its animated subset in displaced mode, the flat pair otherwise.
//
// It SELECTS and builds nothing. Both pairs were built at construction, so
// this is one branch rather than a per-frame walk of the map, and the list
// the window places from is the list the builder produced.
//
// It reads Mode() and not proj, so the flat diagnostic moves the choice at once,
// exactly as it moves the terrain path; and since ModeDisplaced implies
// proj != nil, the pair it returns is one that was actually built.
func (v *Viewer) staticLists() ([]terrain.StaticPlacement, []int) {
	if v.Mode() == ModeDisplaced {
		return v.staticsDisplaced, v.animDisplaced
	}
	return v.staticsFlat, v.animFlat
}

// staticPlacements is what the object layer draws THIS FRAME: the built list
// run through the per-counter pass at the counter the water phase reads and
// the animation switch as it stands now.
//
// THE COUNTER AND THE SWITCH ARE READ HERE, AT THE DRAW, and nothing is gated at
// the build: turning the switch changes what is painted and never what was
// placed, which is 0017's own rule. Because advanceAnimation already returns
// before the accumulator when the switch is off, "off" is both frame 0 and a
// held counter with no second statement to keep true, and switching back on
// resumes from the held value by doing nothing.
//
// Current fog is read here rather than at construction. A visible object may
// cycle; an explored but hidden object stays drawn on its built Index frame;
// an unseen object is dropped by the ground fog gate below. If no candidate is
// visible, the pass hands back the built list itself.
func (v *Viewer) staticPlacements() []terrain.StaticPlacement {
	places, animated := v.staticLists()
	out := terrain.AnimateVisibleStatics(v.staticScratch, places, animated, v.anim.Count(), v.animate && !v.graphics.StaticObjects,
		v.staticAnimationVisible)
	if !samePlacements(out, places) {
		v.staticScratch = out
	}
	// THE FOG GATE, applied here and not at planeSprites' own call site: this
	// is one of the four builders planeSprites reads (statics.go), and
	// shadowDraws (shadow.go) and the diagnostic static-marker pass
	// (overlay.go's staticMarkerScreenRects) read it too — gating the source
	// is what keeps a culled object from being reintroduced by either of those,
	// with no second gate to keep in step. See fogGateStaticPlacements' own doc
	// for the pointer-identity promise this call preserves when no plane is
	// pushed.
	return v.fogGateStaticPlacements(out)
}

// staticAnimationVisible is the live animation gate for map objects. Objects
// stay drawn in explored fog, but their cycle advances only while the owning
// cell is visible now. fogAt also makes a viewer with no fog plane animate all
// cycle-capable objects.
func (v *Viewer) staticAnimationVisible(col, row int) bool {
	return v.fogAt(col, row) == FogVisible
}

// samePlacements reports whether two slices are the same run of memory. It is
// how staticPlacements tells the pass's own buffer from the built list handed
// back unchanged — storing the latter as the scratch would leave a later frame
// writing through the builder's slice.
func samePlacements(a, b []terrain.StaticPlacement) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}

// Statics reports the static-object layer without opening a window: how many
// placements the viewer draws from in the geometry it is in now, and the census
// the build returned — the placements made and the two skip kinds counted apart.
//
// It mirrors ObjectOverlay/UnitOverlay's shape and reason: the cmd and game
// tiers own the decision to load a bundle, so they need a way to see what came
// of it, and a front-end silently drawing no objects is otherwise
// indistinguishable from one drawing them. cmd/mapview's -check line is this
// method's caller.
//
// The count is the placements DRAWABLE, which is the same number in either
// geometry; the skips are not placements and are reported separately, and so
// is the census's open-cycle count — a subset of the placements and never
// a fourth skip kind. The placements themselves are deliberately not handed
// back — they carry pointers into the bundle, and a caller holding the
// slice could write through them.
//
// It reports off the BUILT list and not off the per-counter pass: the pass
// preserves the list's length at every counter, so running it here would cost a
// buffer to answer a question the build already answered.
func (v *Viewer) Statics() (placements int, counts terrain.StaticCounts) {
	places, _ := v.staticLists()
	return len(places), v.staticCounts
}

// structureLists is the built pair the viewer draws from right now: the
// displaced list and its animated subset in displaced mode, the flat pair
// otherwise.
//
// It SELECTS and builds nothing, exactly as staticLists does, and it reads Mode()
// rather than proj — so the flat diagnostic moves the choice at once, and since
// ModeDisplaced implies proj != nil the pair it returns is one that was built.
func (v *Viewer) structureLists() ([]terrain.StructurePlacement, []int) {
	if v.Mode() == ModeDisplaced {
		return v.structuresDisplaced, v.structureAnimDisplaced
	}
	return v.structuresFlat, v.structureAnimFlat
}

// structurePlacements is what the structure layer draws THIS FRAME: the
// built list run through the per-counter pass at the counter the water phase
// reads and the animation switch as it stands now.
//
// THE COUNTER AND THE SWITCH ARE READ HERE, AT THE DRAW, and nothing is
// gated at the build: turning the switch changes what is painted and never
// what was placed. On a map whose structures do not animate the pass hands
// back the built list itself, so this is one comparison per frame and the
// window places from the very slice the builder produced.
func (v *Viewer) structurePlacements() []terrain.StructurePlacement {
	places, animated := v.structureLists()
	out := terrain.AnimateStructureStates(v.structureScratch, places, animated, v.anim.Count(),
		v.animate && !v.graphics.StaticObjects, v.ruinedStructures, v.ruinedStructureIDs)
	if !sameStructures(out, places) {
		v.structureScratch = out
	}
	// THE PER-STRUCTURE SIGHT GATE (TERR-STRUCT-102, DIV-1351): patches the
	// entries AnimateStructureStates just selected with the shared live
	// counter, overriding any whose own structure is not in CURRENT sight
	// back to that structure's own frozen phase. It is a separate pass and
	// not a parameter of the call above because AnimateStructureStates has
	// one counter for the whole map by design (structures.go's own doc on
	// SelectStructureFrame) and this viewer is the one caller with a fog
	// model to gate an individual structure's own phase with.
	v.freezeStructurePhases(out, animated)
	// THE FOG GATE — staticPlacements' own comment applies term for term,
	// over structures rather than objects.
	return v.fogGateStructurePlacements(out)
}

// freezeStructurePhases holds each structure's own animation phase at its
// last-observed value while that structure's cell is out of CURRENT sight —
// TERR-STRUCT-102's "this+0x70 is advanced ... gated this+0x78==0", read as a
// per-drawable field the original re-checks every tick, not a per-cell one
// (owner report 3, hotfix DIV-1351: "объект не должен анимироваться если он
// вне обзора, даже если объект раскрыт").
//
// IT FREEZES; IT DOES NOT RESET. `phase = (phase+1) mod N` simply does not run
// while gated, so the original's own last-selected phase stays selected —
// SelectStructureFrame's `phaseFrame = timeline[phase]` keeps answering with
// whatever phase this map last wrote for that structure. Passing animate=false
// here instead would select the BASE grid frame (SelectStructureFrame's own
// fallback), which is a visible snap-back to the idle sprite every time the
// camera looks away and not what the claim states.
//
// VISIBILITY IS "ANY OF THE STRUCTURE'S OWN CELLS IS FogVisible RIGHT NOW",
// decided once per structure and applied to every one of its entries: the
// original's own single per-drawable field is Unknown in which cell it
// mirrors, so this is an AUTHORED CHOICE rather than a decoded one, made once
// here rather than per entry so a building spanning a sight boundary does not
// animate one strip and freeze its neighbour.
//
// A STRUCTURE THIS VIEWER HAS NEVER OBSERVED IN CURRENT SIGHT holds phase 0 —
// not the live counter AnimateStructureStates already selected with above —
// so a structure that is merely explored (a loaded save's fog memory, never
// yet watched by this running viewer) does not animate on the very first
// frame either. The phase cache is presentation-only and carries no history
// across a fresh viewer, which is the one gap this hotfix leaves open.
//
// RUIN IS UNAFFECTED: SelectStructureFrame's ruin arm is taken before its
// counter argument is ever read, so passing this structure's own frozen or
// live counter through a ruined entry gives the identical answer either way,
// and this pass need not special-case it.
func (v *Viewer) freezeStructurePhases(places []terrain.StructurePlacement, animated []int) {
	if len(animated) == 0 || !v.animate || v.graphics.StaticObjects || v.ruinedStructures {
		return
	}
	visible := make(map[uint32]bool)
	for i := range places {
		id := places[i].StructureID
		if visible[id] {
			continue
		}
		if v.fogAt(places[i].Cell.X, places[i].Cell.Y) == FogVisible {
			visible[id] = true
		}
	}
	live := v.anim.Count()
	if v.structurePhase == nil {
		v.structurePhase = make(map[uint32]uint32)
	}
	for _, i := range animated {
		if i < 0 || i >= len(places) {
			continue
		}
		p := &places[i]
		if p.Class == nil {
			continue
		}
		if visible[p.StructureID] {
			v.structurePhase[p.StructureID] = live
			continue
		}
		frozen := v.structurePhase[p.StructureID] // 0 when never yet observed
		ruined := v.ruinedStructureIDs[p.StructureID]
		// StructureClass.frame is package-private to pkg/render/terrain; this
		// is that same bounds-checked lookup, reproduced rather than exported
		// for one caller, since Frames is already this package's own public
		// field and SelectStructureFrame is already the exported selector.
		if idx := terrain.SelectStructureFrame(p.Class, p.GridIndex, frozen, true, ruined); idx >= 0 && idx < len(p.Class.Frames) {
			p.Frame = p.Class.Frames[idx]
		}
	}
}

// sameStructures reports whether two slices are the same run of memory. It is how
// structurePlacements tells the pass's own buffer from the built list handed back
// unchanged — storing the latter as the scratch would leave a later frame writing
// through the builder's slice.
func sameStructures(a, b []terrain.StructurePlacement) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}

// Structures reports the structure layer without opening a window: how many
// entries the viewer draws from in the geometry it is in now, and the
// two-level census the build returned.
//
// It mirrors Statics' shape and reason — the cmd and game tiers own the decision
// to load a bundle, so they need a way to see what came of it, and a front-end
// silently drawing no buildings is otherwise indistinguishable from one drawing
// them. cmd/mapview's -check line is this method's caller.
//
// THE FIRST NUMBER IS ENTRIES AND NOT BUILDINGS. One structure covers a rectangle
// of cells and each of those is a strip of image rows, so a map with 137 drawn
// placements has thousands of entries; the census's own two levels are what
// answer "how many buildings" and "how many cells", and they are returned beside
// it rather than derived from it.
//
// The entries themselves are deliberately not handed back — they carry pointers
// into the bundle, and a caller holding the slice could write through them.
func (v *Viewer) Structures() (entries int, counts terrain.StructureCounts) {
	places, _ := v.structureLists()
	return len(places), v.structureCounts
}

// MarginCells is how many of this map's cells lie in the engine margin, i.e.
// how many cells the viewer will dim. It is 0 for a grid carrying no block
// plane, which is the same 0 a map with no margin would report — the two
// are not distinguished here, because what this answers is "how many cells
// are dimmed" and the answer is the same either way.
//
// It is a COUNT and not the plane, for the reason Statics hands back a count
// rather than its placements: a caller holding the slice could write through
// it, and the whole value of deriving the margin once is that nothing
// downstream can disagree about it afterwards.
//
// It counts by asking BorderCell, cell by cell, rather than by popcounting the
// plane. That is deliberate and it is what makes this an instrument: a census
// that read the bytes its own way would agree with a broken reader, and this
// one cannot — it reports exactly what the draw path will do.
func (v *Viewer) MarginCells() int {
	n := 0
	for y := 0; y < v.grid.Height; y++ {
		for x := 0; x < v.grid.Width; x++ {
			if v.grid.BorderCell(x, y) {
				n++
			}
		}
	}
	return n
}

// validAltitudes reports whether the grid's altitude layer can be projected:
// exactly Width*Height entries over positive dimensions.
//
// The shared projection is total over such a grid, so this is the whole of
// the validity half of Mode's predicate; there is nothing further for the
// viewer to check there.
func validAltitudes(g terrain.Grid) bool {
	return g.Width > 0 && g.Height > 0 && len(g.Altitudes) == g.Width*g.Height
}

// Mode reports which terrain path the viewer renders through right now. It is
// computed, never stored: the projection half cannot go stale because nothing
// can replace the grid after construction, and the flat half is read live, so
// SetFlat on a running viewer moves the mode at once.
func (v *Viewer) Mode() Mode {
	if v.proj != nil && !v.flat {
		return ModeDisplaced
	}
	return ModeFlat
}

func (v *Viewer) Lit() bool {
	return v.levels != nil && !v.unshaded
}

// lightTint is the sky tint this frame's textures carry: v.sun's own tint
// wherever Lit() holds, and zero wherever it does not. Both texture caches
// — cacheKey here and spriteTextureKey in statics.go — key on this one
// method's answer rather than each re-deciding when a tint applies, so the
// ground and the sprite passes cannot come to disagree about it.
//
// It reads Lit() and not v.unshaded alone, which is a wider gate than
// spriteRow's own (spriteRow reads v.unshaded only, deliberately, because a
// row has no relief term — see its own comment).
func (v *Viewer) lightTint() [3]uint8 {
	if !v.Lit() {
		return [3]uint8{}
	}
	return v.sun.SkyTint
}

// Sun is the light this viewer is drawing by: the day/night cycle's cache,
// as of the last relight. It is a value, so a caller cannot write through it
// into the viewer.
func (v *Viewer) Sun() terrain.Light { return v.sun }

// TimeFlow reports whether the day/night cycle is running.
func (v *Viewer) TimeFlow() bool { return v.timeFlow }

// SetLightClock gives the viewer the world's clock, in the engine's own
// sub-ticks, and relights if this sub-tick is one the cadence fires on.
//
// THE DECISION IS HERE AND NOT AT THE CALLER, which is the whole of why "what is
// drawn is the last relight's sun" holds. A front-end that decided for itself
// would be a second opinion about when the sun moves, and two front-ends could
// then disagree; handing over a clock and nothing else leaves them no way to.
//
// It is called once per world tick and MAY be called with a clock it has
// already seen — a relight is idempotent, so a repeat costs a rebuild and
// changes nothing. A viewer never given a clock never reaches this and keeps
// its seed.
//
// The offset enters the cadence test as well as the sun, which is free rather
// than load-bearing: it moves in whole in-game hours and an hour is three whole
// relight periods, so the shifted clock fires on the same sub-ticks as the raw
// one.
func (v *Viewer) SetLightClock(subTicks uint64) {
	v.lightClock = subTicks
	if terrain.RelightDue(subTicks + v.lightOffset) {
		v.relight()
	}
}

// RestoreScheduledLightClock reconstructs the last standard scheduled light
// for a native save without advancing the world or forcing its current minute.
// The actual clock is retained for later ticks and explicit forced relights.
// Native saves do not carry switch/diagnostic history; this does not infer an
// original SAV load policy or change SetLightClock's ordinary cadence.
func (v *Viewer) RestoreScheduledLightClock(subTicks uint64) {
	v.lightClock = subTicks
	const period = terrain.SubTicksPerMinute * terrain.RelightPeriodMinutes
	v.relightAt(subTicks - subTicks%period)
}

// SetTimeFlow moves the day/night cycle's switch and FORCES a relight,
// whatever the clock reads.
//
// It relights on every call, including one that writes the value already there.
// The original's own key does the same — it flips the flag and posts a forced
// relight unconditionally — and a relight is idempotent, so the only thing a
// change test would buy is a branch that can disagree with the flip.
func (v *Viewer) SetTimeFlow(on bool) {
	v.timeFlow = on
	v.relight()
}

// ToggleTimeFlow flips the switch, which is what the key bound to it does.
func (v *Viewer) ToggleTimeFlow() { v.SetTimeFlow(!v.timeFlow) }

// StepLightClock advances the DIAGNOSTIC lighting-clock offset by one
// in-game hour and forces a relight.
//
// IT IS OURS AND THE ORIGINAL HAS NOTHING LIKE IT (spec D-3). It exists because
// a whole in-game day takes some twenty-four minutes of real time at the shipped
// speed, so the cycle is otherwise inspectable only by waiting; twenty-four
// presses walk a day and return the picture to where it started, which is a
// check as well as a convenience.
//
// It reaches WHAT IS DRAWN AND NOTHING ELSE. The offset lives on the viewer,
// which cannot spell a world, a tick or a digest, so the bound is the type
// system's rather than a rule to keep.
func (v *Viewer) StepLightClock() {
	v.lightOffset += terrain.SubTicksPerMinute * 60
	v.relight()
}

// relight rebuilds the sun at the current clock through the shared cache writer.
func (v *Viewer) relight() { v.relightAt(v.lightClock) }

// relightAt rebuilds the sun from the clock and the switch, and then
// rebuilds the whole per-vertex relief grid from it.
//
// THE WHOLE GRID GOES, EVERY TIME. The original has no cheaper path — a changed
// angle reaches its screen only through the full per-vertex rebuild and the
// shading-table rebuild behind it — and neither does this: the angle enters
// every vertex through 32/cos|theta| and the lateral tan|theta| shear, so no
// sub-region of the grid is unaffected by a move.
//
// The rebuild is gated on there BEING a grid, and that is not a second reading
// of the constructor's validity guard: the question here is whether that guard
// ran, and levels being non-nil is exactly its verdict. A viewer whose altitudes
// were unusable therefore stays unlit through any number of relights, and its
// Lit() answer never moves.
func (v *Viewer) relightAt(subTicks uint64) {
	v.sun = terrain.SunAt(terrain.ClockMinute(subTicks+v.lightOffset), v.timeFlow)
	if v.levels == nil {
		return
	}
	v.levels = terrain.LevelGrid(v.grid.Altitudes, v.grid.Width, v.grid.Height, v.sun)
}

// syncWorld tells the camera how tall the world is for the mode the viewer is
// now in, and is called from every entry point that can change that mode:
// NewViewer, both overlay setters and SetFlat.
//
// Displaced, the world is the projection's own canvas, MaxV-MinV — which is
// Rows*CellSize only when every altitude is equal, and may be either taller or
// SHORTER. Flat, it is the tile grid's Rows*CellSize, the expression New itself
// initialises with, so a flat viewer is byte-for-byte the one that shipped.
//
// SetWorldHeight re-clamps, so a mode flip takes effect at once rather than at
// the next pan. The view POSITION is deliberately not re-anchored across the
// flip: the position stands and the clamp decides what survives.
func (v *Viewer) syncWorld() {
	// Drop a projection-dependent resolver before changing the projection mode
	// or its world height. The final sync below installs the new mode's bound;
	// clearing first prevents the old resolver from clipping that transition.
	v.cam.ClearClampBounds()
	if v.Mode() == ModeDisplaced {
		v.cam.SetWorldHeight(float64(v.proj.CanvasHeight()))
	} else {
		v.cam.SetWorldHeight(float64(v.grid.Height) * camera.CellSize)
	}
	v.syncCameraBounds()
}

// playableCellRect is the bounding rectangle of cells the terrain pass does
// not deliberately paint as the engine's black map margin. BorderCell is the
// authority: the margin depth remains owned by map ingest and is not repeated
// in camera code.
func (v *Viewer) playableCellRect() (image.Rectangle, bool) {
	if len(v.grid.Block) == 0 {
		return image.Rectangle{}, false
	}
	if r, ok := v.playableCellRectFromEdges(); ok {
		return r, true
	}
	return v.playableCellRectScan()
}

func (v *Viewer) playableCellRectFromEdges() (image.Rectangle, bool) {
	w, h := v.grid.Width, v.grid.Height
	minX, minY := w, h
	maxX, maxY := -1, -1
	foundMargin := false
	for y := 0; y < h; y++ {
		lo := 0
		for lo < w && v.grid.BorderCell(lo, y) {
			lo++
		}
		if lo == w {
			foundMargin = foundMargin || w > 0
			continue
		}
		hi := w - 1
		for v.grid.BorderCell(hi, y) {
			hi--
		}
		if lo > 0 || hi < w-1 {
			foundMargin = true
		}
		minX, maxX = min(minX, lo), max(maxX, hi)
		minY, maxY = min(minY, y), max(maxY, y)
	}
	if !foundMargin || maxX < minX || maxY < minY {
		return image.Rectangle{}, false
	}
	return image.Rect(minX, minY, maxX+1, maxY+1), true
}

func (v *Viewer) playableCellRectScan() (image.Rectangle, bool) {
	minX, minY := v.grid.Width, v.grid.Height
	maxX, maxY := -1, -1
	foundMargin := false
	for y := 0; y < v.grid.Height; y++ {
		for x := 0; x < v.grid.Width; x++ {
			if v.grid.BorderCell(x, y) {
				foundMargin = true
				continue
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	// A viewer with no derived margin keeps the complete-world clamp it has
	// always had. This path covers developer and synthetic grids whose block
	// plane is absent as well as a real map format that carries no margin.
	if !foundMargin || maxX < minX || maxY < minY {
		return image.Rectangle{}, false
	}
	return image.Rect(minX, minY, maxX+1, maxY+1), true
}

const lowerRenderLipRows = 4

func (v *Viewer) renderCellRect() (image.Rectangle, bool) {
	cells, ok := v.playableCellRect()
	if ok {
		cells.Max.Y = min(v.grid.Height, cells.Max.Y+lowerRenderLipRows)
	}
	return cells, ok
}

// TERR-216.
func (v *Viewer) syncCameraBounds() {
	cells, ok := v.playableCellRect()
	if !ok {
		v.cam.ClearClampBounds()
		return
	}
	origin := 0
	if v.Mode() == ModeDisplaced {
		origin = v.proj.MinV
	}
	v.cam.SetCellClampBounds(float64(cells.Min.X*camera.CellSize),
		float64(cells.Min.Y*camera.CellSize-origin),
		float64(cells.Max.X*camera.CellSize),
		float64(cells.Max.Y*camera.CellSize-origin))
}

// SetAnimated turns water animation on or off. Off reproduces the static render
// exactly, because the game's disable switch forces phase 0 rather than freezing
// the current phase (TERR-ANIM-009).
func (v *Viewer) SetAnimated(on bool) { v.animate = on }

// SetSpeedIndex selects the logic-tick cadence from the game's speed table; the
// index is clamped into range. It writes BOTH the period the clock runs at and
// the index this viewer reports, in that one place, so the summary and the
// cadence cannot come to name different speeds.
func (v *Viewer) SetSpeedIndex(speedIndex int) {
	v.speedIndex = terrain.ClampSpeedIndex(speedIndex)
	v.anim.SetPeriod(terrain.SpeedIndexPeriod(v.speedIndex))
}

// SetPeriod re-rates the water counter to a tick length in microseconds and
// REPORTS the period it adopted.
//
// Reporting is the point. The caller hands the returned value straight to the
// other consumer of that cadence in the same statement, so one period value has
// two readers and no path re-rates one without the other. A setter that returned
// nothing would leave the two writes as two statements, which is exactly the
// shape that lets a later edit move one of them. What it reports is what the
// ticker actually took, so a period the clock refuses is already in what crosses
// the seam rather than in what was asked for.
//
// IT TAKES A PERIOD AND NOT A RATE. This is the water's reader of a cadence
// the world's clock reads too, and the two must hold the SAME number. Handed
// a rate, each side divided for itself — and the game's own nine speeds
// are a truncated whole millisecond that no microsecond quotient of a rate
// reproduces, so a shipped cadence survived one division and not the other.
// Deciding what to call the period is the ladder's business, not this
// type's; terrain.CadenceRung and terrain.SpeedIndexOf read one back for
// anything that has to name it.
//
// THE WATER FOLLOWS THIS CADENCE BY DECODE, not by our choice: the counter is
// advanced by the handler of the game's own paced logic tick, whose period is
// the speed setting's (TERR-ANIM-007, TERR-ANIM-008, ANIM-CLOCK-001 — all High),
// so an ambient animation running at anything but the world's cadence would be
// a divergence rather than a decision.
//
// advanceAnimation holds the counter and its fractional phase during player
// pause or a popup. The period setter changes neither stop nor phase.
//
// It leaves the speed index this viewer reports where it was. A period is a
// second, independent selection over the same clock, so a viewer given one runs
// at it while still able to say which of the game's nine speeds it was last
// handed.
func (v *Viewer) SetPeriod(periodUS int) int {
	v.anim.SetPeriod(periodUS)
	return v.anim.Period()
}

// SetCadenceMode adopts the period, selects whether map animation advances
// from elapsed time or exactly once per eligible map frame, and applies an
// explicit paced-phase reset. It reports the adopted period so flow can hand
// that same scalar to the world-side reader.
//
// Leaving unpaced mode or receiving reset clears the fractional phase and
// wall-clock baseline but preserves the counter. This is Ctrl+numpad minus's
// phase/epoch reset: the next paced frame establishes a baseline and cannot pay
// back the unpaced span as a catch-up burst.
func (v *Viewer) SetCadenceMode(periodUS int, unpaced, reset bool) int {
	v.anim.SetPeriod(periodUS)
	if reset || v.unpaced && !unpaced {
		v.anim.ResetPhase()
		v.last = time.Time{}
	}
	v.unpaced = unpaced
	return v.anim.Period()
}

// SetUnshaded turns the unshaded diagnostic on (full palette brightness) or
// off (corner-interpolated shading, the default). It is the standalone
// developer viewer's only entry to the flag; the game front-end never calls
// it and gains no flag of its own.
//
// Deliberately calls no syncWorld: lighting never moves geometry — the
// level grid is a pure function of the altitudes alone — so nothing cached
// depends on this state the way the overlay setters' world extent depends on
// theirs.
func (v *Viewer) SetUnshaded(on bool) { v.unshaded = on }

// SetFlat turns the flat diagnostic on (draw the un-displaced lattice even
// over a valid altitude grid) or off (the default: displace whenever the
// grid allows it). It mirrors SetUnshaded's shape — the standalone
// developer viewer's only entry to the flag; the game front-end never calls
// it and gains no flag of its own.
//
// Unlike SetUnshaded, this DOES call syncWorld: flat and displaced differ in
// world size, so a mode flip from this setter must re-clamp the camera at
// once, exactly as the overlay setters already do when they used to move the
// mode.
func (v *Viewer) SetFlat(on bool) {
	v.flat = on
	v.syncWorld()
}

// SetRuins turns the structure destruction diagnostic on (draw every
// drawable structure from its ruin block) or off (the default: the intact
// art). It mirrors SetUnshaded's and SetFlat's shape and reason — the
// standalone developer viewer's only entry to the flag, which the game
// front-end never calls and gains no flag of its own.
//
// It changes WHAT IS PAINTED and never what was placed: no count moves, no entry
// moves, no cell is added or dropped, and a class carrying Indestructible stays
// intact under it. So it needs no syncWorld and no rebuild — the switch is read
// at the draw, in the per-counter pass, exactly where the animation switch is.
func (v *Viewer) SetRuins(on bool) { v.ruinedStructures = on }

// MapStructure is live, read-only health and location for art and inspection.
type MapStructure struct {
	ID        uint32
	Health    uint16
	MaxHealth uint16
	Cell      image.Point
}

// SetStructures adopts the current inspection snapshot and destroyed set.
// Placement and passability do not change when health reaches zero.
func (v *Viewer) SetStructures(structures []MapStructure) {
	v.structureStates = structures
	if len(structures) == 0 {
		clear(v.ruinedStructureIDs)
		return
	}
	ruined := v.ruinedStructureIDs
	if ruined == nil {
		ruined = make(map[uint32]bool)
	} else {
		clear(ruined)
	}
	for _, s := range structures {
		if int16(s.Health) <= 0 {
			ruined[s.ID] = true
		}
	}
	v.ruinedStructureIDs = ruined
}

// StructureRuinFrames reports how many draw entries for id currently select
// the class's ruin block. It is a headless witness over the production list.
func (v *Viewer) StructureRuinFrames(id uint32) (entries, ruined int) {
	for _, p := range v.structurePlacements() {
		if p.StructureID != id {
			continue
		}
		entries++
		if p.Class != nil && !p.Class.Indestructible {
			// Build the expected ruin address independently of the production
			// selector this diagnostic observes. A selector and its witness must
			// not share the same mistaken formula.
			idx := len(p.Class.Frames) - p.Class.GridCells() + p.GridIndex
			if idx != p.GridIndex && idx >= 0 && idx < len(p.Class.Frames) && p.Frame == p.Class.Frames[idx] {
				ruined++
			}
		}
	}
	return entries, ruined
}

type MapSack struct {
	Cell       image.Point
	FrameIndex int
}

// SetSackFrames hands the viewer the decoded sack sheet's frames, in the
// sheet's own order. It is the front end's own entry, called ONCE per opener
// path at construction — on the font's and the attack pointer's own
// precedent — and never per refresh, because the sheet itself is decoded
// once and never changes for the life of a run.
//
// A viewer this is never called on holds a nil slice, which sackLayer reads
// exactly as it reads an empty one: no frame resolves, so every sack this
// viewer is later given draws nothing, silently.
func (v *Viewer) SetSackFrames(frames []*terrain.StaticFrame) {
	v.sackFrames = frames
}

// SetSacks hands the viewer the world's ground-sack list for this refresh:
// one MapSack per entry, in the world's own order. It mirrors SetEntities'
// own contract exactly, and is called EVERY TICK beside it: the slice is
// ADOPTED, not copied — the caller is the tier that rebuilt it from this
// tick's world state and does not keep it — and it REPLACES whatever was
// held before rather than accumulating.
//
// There is no show parameter, mirroring SetEntities' own reason: a sack is
// world content and not a diagnostic switch, so passing sacks is what turns
// the sack stream on and passing none is what turns it off.
func (v *Viewer) SetSacks(sacks []MapSack) {
	v.sacks = sacks
}

// SetPhase tells the viewer where this frame stands inside the current tick:
// the microseconds elapsed toward the next one, and the tick length they are
// measured in.
//
// IT IS THE PACING CLOCK'S OWN TWO VALUES and never a clock of the drawing's.
// The tier above holds one accumulator, the world's advance is a function of it,
// and this is that same quantity read out — so a stopped world's picture cannot
// slide and a re-rate cannot put an entity somewhere its own advance disagrees
// with. Two accumulators would be two clocks to keep in step, disagreeing
// exactly under a pause and a rate change.
//
// It stores and does NOTHING ELSE — no clamp, no rebuild, no syncWorld. What a
// remainder outside its period means is the drawing's question, answered where
// the arithmetic is; the camera's world extent is a property of the terrain and
// cannot move on a per-frame value.
//
// A front-end that never calls it leaves the zero pair, which is "not told" and
// draws no displacement at all.
func (v *Viewer) SetPhase(elapsedUS, periodUS int) {
	v.phaseUS, v.phasePeriodUS = elapsedUS, periodUS
}

// Phase reports the pair last pushed, mirroring EntityMarkers' shape and reason:
// the tier that paces the world needs to see that its phase arrived without
// opening a window, and a viewer drawing every entity on its own cell because it
// was told nothing is otherwise indistinguishable from one that was told and
// ignored it.
func (v *Viewer) Phase() (elapsedUS, periodUS int) { return v.phaseUS, v.phasePeriodUS }

// Animation reports whether water animates and at which speed index, so the cmd
// tier and tests can inspect the configuration without a window.
func (v *Viewer) Animation() (on bool, speedIndex int) { return v.animate, v.speedIndex }

// AnimationCounter reports the current logic-tick counter driving the water
// phase.
func (v *Viewer) AnimationCounter() uint32 { return v.anim.Count() }

// Camera exposes the camera so the cmd tier and tests can inspect or position the
// view without going through the run loop.
func (v *Viewer) Camera() *camera.Camera { return v.cam }

// FrameSize reports the current mission composition size in logical pixels.
// It is a read-only headless seam: callers observe Layout's production result
// rather than reproducing its aspect arithmetic.
func (v *Viewer) FrameSize() image.Point { return image.Pt(v.frameW, v.frameH) }

// ViewportSize reports the current game surface inside the mission frame. The
// width excludes the right HUD strip; the height ends above an open inventory
// or spellbook bar.
func (v *Viewer) ViewportSize() image.Point { return image.Pt(v.cam.ViewW, v.cam.ViewH) }

// AuthoredStartColumns is the base 1024-arm span used to choose opening zoom.
// Wider logical frames keep that zoom and reveal more than this many columns.
//
// THE NAME IS NOW WRONG AND IS KEPT, which is worth saying first because it is
// the only thing about this function a reader can mistake. The value was
// AUTHORED — 20, ours by choice, disclosed as an UPPER BOUND rather than a
// measurement, because the original's screen carried a side panel whose geometry
// was undecoded and the only honest claim was 640/32. The panel is decoded now
// (SESS-VIEW-028) and this is the original's own figure, so the verdict is gone
// and only the identifier remains; renaming it is an edit across the tree that
// this story is not, and a name is cheaper to leave wrong than a number.
//
// WHERE 15 COMES FROM. The map view object is constructed with the rect
// `(0, 0, screenW - 0xa0, screenH)` — the screen minus a 160-PIXEL RIGHT STRIP,
// which is the panel — and the column span it stores is `(right - left) / 32`,
// a signed divide toward zero. The screen is one of three literal arms, and the
// arm a consumer selecting nothing reaches is 640x480: the command line is read
// for -800, -1024 and -640 in that order, then the RESOLUTION registry buffer for
// -800 and -1024, and both remaining paths fall through to the 640 arm — which
// the registry read agrees with, copying the literal "-640" into its own buffer
// when the query fails. So the shipped default screen gives (640 - 160)/32 = 15,
// against 20 at 800x600 and 27 at 1024x768.
//
// It is a COLUMN COUNT and not a zoom. A zoom means nothing without a view size
// and so is not a property of the original at all, while a tile count is. The
// number of ROWS is still deliberately not stated, and that IS still a
// divergence rather than an omission: the decoded viewport is a column span and
// a row span per resolution — 15, 18 and 24 — and an open side panel recomputes
// the row span at run time from the panel's own height. This build uses the
// 1024 arm's 27 columns only to pin zoom. The base frame remains 768 pixels
// high, visible bottom bars shorten its live game surface, and wider frames
// reveal additional columns.
func AuthoredStartColumns() int { return startColumns }

// MissionViewportSize is the BASE map-view size inside the minimum mission
// frame: 1024x768 minus the right strip. A live Viewer reports its expanded
// geometry through ViewportSize.
//
// It is SESS-VIEW-028's rect (0, 0, screenW - 0xa0, screenH) evaluated at the
// minimum frame this build starts the mission at. At native zoom it spans
// MissionViewportSize().X/32 columns by MissionViewportSize().Y/32 rows, which
// is the claim's own 27 by 24 for a 1024x768 screen.
func MissionViewportSize() image.Point {
	return viewportSize(MissionFrameW, MissionFrameH)
}

// viewportSize is SESS-VIEW-028's rect evaluated at any frame size: the frame
// minus the 160-pixel right strip. A frame narrower than the strip has no
// viewport at all and yields a zero width rather than a negative one.
func viewportSize(frameW, frameH int) image.Point {
	w := frameW - MissionPanelW
	if w < 0 {
		w = 0
	}
	return image.Pt(w, frameH)
}

// mapViewportSize is the live game surface: left of the fixed right column and
// above whichever bottom HUD panel is drawn. A switched-on panel with no
// selected hero stands empty and reserves the same rows as with one, so the
// map's scroll limit does not depend on the selection.
func (v *Viewer) mapViewportSize() image.Point {
	if v.editorView {
		return image.Pt(max(1, v.frameW-editorRailWidth), v.frameH)
	}
	view := viewportSize(v.frameW, v.frameH)
	if bar, _, ok := v.packBarShown(); ok && bar.Min.Y < view.Y {
		view.Y = bar.Min.Y
	}
	if bar, _, ok := v.spellbookBar(); ok && bar.Min.Y < view.Y {
		view.Y = bar.Min.Y
	}
	if view.Y < 0 {
		view.Y = 0
	}
	return view
}

// noticeViewportSize: an empty panel with no selection does not shrink a notice.
func (v *Viewer) NoticeViewportSize() image.Point { return v.noticeViewportSize() }

func (v *Viewer) noticeViewportSize() image.Point {
	view := image.Pt(v.cam.ViewW, v.cam.ViewH)
	if v.editorView {
		return view
	}
	full := viewportSize(v.frameW, v.frameH)
	if bar, _, ok := v.packBar(); ok && bar.Min.Y < full.Y {
		full.Y = bar.Min.Y
	}
	if bar, _, ok := v.spellbookBar(); ok && v.spellbookHeld && bar.Min.Y < full.Y {
		full.Y = bar.Min.Y
	}
	return image.Pt(view.X, max(full.Y, view.Y))
}

// syncMapViewport applies mapViewportSize without re-anchoring the camera.
// Growing the surface therefore reveals more terrain below at the same world
// origin. Only the camera clamp may move it, and only when that newly revealed
// bottom would cross the drawable map edge.
func (v *Viewer) syncMapViewport() {
	if v == nil || v.cam == nil {
		return
	}
	view := v.mapViewportSize()
	if v.cam.ViewW == view.X && v.cam.ViewH == view.Y {
		return
	}
	v.cam.ViewW, v.cam.ViewH = view.X, view.Y
	v.cam.Clamp()
}

// mapSurfaceCaptures reports whether the FRAME position (x, y) is on the live
// game surface: the rectangle the camera draws the world into, left of the
// right column and above every visible bottom bar.
//
// It answers false for the right strip and for every position outside the frame
// — the letterbox a placement leaves when the window's aspect is not the frame's.
// Its argument is a frame position, so its caller must be past the applicable
// input door. The attack cursor, mission hover and gesture paths, and the
// ground picker share this viewport gate.
func (v *Viewer) mapSurfaceCaptures(x, y int) bool {
	return x >= 0 && y >= 0 && x < v.cam.ViewW && y < v.cam.ViewH
}

// MissionPanelRect is the minimum 1024x768 frame's right strip. Live right-
// column composers use v.frameW and therefore follow an expanded frame's edge.
func MissionPanelRect() image.Rectangle {
	return image.Rect(MissionFrameW-MissionPanelW, 0, MissionFrameW, MissionFrameH)
}

// Placement is the mission frame's current fit into the window. It is what maps
// a window position onto this screen and back, and a caller outside this package
// needs it for neither: it is exposed for the headless scenario oracles, which
// must hand a scenario a WINDOW pixel for a surface they found in frame pixels.
func (v *Viewer) Placement() frame.Placement { return v.place }

// SetStartView arms the view to open centred on one cell, at the authored
// extent. It is ARMING and not doing: the move happens at the next Layout.
//
// THE DEFERRAL IS FORCED AND NOT A STYLE. The zoom now comes from the fixed
// base viewport, but centring still needs the live viewport width. A viewer
// is armed before the run loop supplies that geometry through Layout.
//
// IT IS A ONE-SHOT. Layout runs every frame, so an arming that survived its own
// application would be a camera that follows rather than one that opens, and
// would fight the player on every frame after the first. It is cleared when
// applied; nothing but this method ever sets it.
//
// A viewer that is never armed is never moved: the standalone developer viewer
// and the map picker's own path call nothing here and open exactly as they did.
func (v *Viewer) SetStartView(cell image.Point) {
	v.startCell, v.startArmed = cell, true
}

// SetSAVStartView uses the original format's unit zoom and integer-cell
// camera origin for a newly constructed SAV-authoritative world.
func (v *Viewer) SetSAVStartView(cell image.Point) {
	v.SetStartView(cell)
	v.startSAV = true
}

// Layout keeps the mission's logical height at MissionFrameH and expands only
// its logical width to cover wider window aspects. A 4:3 window therefore stays
// byte-for-byte at the 1024x768 geometry, while 16:9 and ultrawide windows add
// map viewport to the left of the fixed 160-pixel right HUD strip. Narrow/tall
// windows retain the minimum width and may letterbox vertically.
//
// It is still where an armed start view is applied, because this is where a
// viewer first learns it has been placed at all (SetStartView).
func (v *Viewer) Layout(outsideWidth, outsideHeight int) (int, int) {
	if outsideWidth > 0 && outsideHeight > 0 {
		if v.place.WindowSize() != image.Pt(outsideWidth, outsideHeight) && v.tooltip != nil {
			v.tooltip.reset()
		}
		frameW := frame.ExpandedWidth(MissionFrameW, MissionFrameH, outsideWidth, outsideHeight)
		if frameW != v.frameW || v.frameH != MissionFrameH {
			v.frameW, v.frameH = frameW, MissionFrameH
			if v.canvas != nil {
				disposeFrameCanvas(v.canvas)
				v.canvas = nil
			}
		}
		v.place = frame.Fit(v.frameW, v.frameH, outsideWidth, outsideHeight)
		v.syncMapViewport()
		v.applyStartView()
	}
	return outsideWidth, outsideHeight
}

// windowToFrame maps a window position onto the mission frame, extending the
// frame's own lattice outside it (1026 B4).
//
// IT IS THE ONE MAPPING THIS PACKAGE'S MISSION PATH USES, and it is applied at
// exactly three doors: step, command and noticeButtonAt. Everything downstream
// of those three reads a frame position, which is why no individual hit test in
// this package had to change. A fourth door added later must map here too, and
// the test for one is whether the value it carries came from
// ebiten.CursorPosition or from a scenario's window pixel.
//
// An unusable placement leaves the position alone rather than refusing: a viewer
// that has never been laid out is a viewer nothing has clicked on, and returning
// a position keeps this a total function so no caller grows an error path for a
// case it cannot reach.
func (v *Viewer) windowToFrame(x, y int) (int, int) {
	p, ok := v.place.WindowToFrameExtended(x, y)
	if !ok {
		return x, y
	}
	return p.X, p.Y
}

// applyStartView scales the view to the authored extent and centres it on the
// armed cell, once.
//
// The zoom is the BASE viewport width / (columns * CellSize). It goes through
// SetZoom, so the camera's own limits remain authoritative. The live viewport
// may be wider; that additional width shows additional map at the same scale.
//
// It is ordered zoom THEN centre, and the order matters: the view's extent in
// world pixels is a function of the zoom, so centring first would centre against
// the old scale and leave the cell off-centre by half the difference.
//
// The cell's world point comes from cellWorldCentre, so the view arrives where
// the cell is DRAWN — carrying displaced mode's own lift — and not where a flat
// lattice would put it. On relief those are a whole hillside apart.
func (v *Viewer) applyStartView() {
	if !v.startArmed {
		return
	}
	v.startArmed = false

	if cols := AuthoredStartColumns(); cols > 0 {
		// The opening zoom is pinned to the 1024-arm viewport. Extra logical
		// width reveals more map; it must not zoom up to preserve 27 columns.
		v.cam.SetZoom(float64(MissionViewportSize().X) / float64(cols*camera.CellSize))
	}
	v.cam.CenterOn(v.cellWorldCentre(v.startCell))
	if v.startSAV {
		v.startSAV = false
		v.cam.SetZoom(1)
		v.cam.CenterOn(v.cellWorldCentre(v.startCell))
		v.cam.X = math.Round(v.cam.X/camera.CellSize) * camera.CellSize
		v.cam.Y = math.Round((v.cam.Y+v.projectionMinY())/camera.CellSize)*camera.CellSize - v.projectionMinY()
	}
}

// Input is one tick of viewer input, as read from the engine.
//
// It carries only raw engine reads. In particular there is no "is the cursor
// inside the window" flag: whether the cursor is inside is a question about the
// camera's view size, which the engine does not answer and readInput cannot see,
// so step decides it. Putting it here would mean either a field nothing uses or
// an edge-scroll rule that differs from the one the standalone viewer ships.
//
// PrimaryDown follows the same rule: it is the primary mouse button's LEVEL
// this tick (is it down right now), never an edge. A raw level is what lets
// step resolve every hard case itself — a release outside the window, a
// lost-focus tick, a button already held when the map opened — as
// "PrimaryDown false" or "a first true tick" rather than a second field
// trying to name each case. Its zero value is false, so a struct literal
// that does not name it means "button up", exactly as it did before this
// field existed.
//
// Shift follows PrimaryDown's rule exactly: it is the modifier's LEVEL this
// tick — EITHER Shift key, the two being one modifier and not two — and
// never an edge. A level is what a latch needs: the gesture reads it ONCE,
// on the tick the button goes down, and nothing reads it again, so a level
// that changes mid-drag changes nothing. Its zero value is false, so a
// struct literal that does not name it means "no modifier", exactly as it
// did before this field existed — which is what leaves every shipped drag
// case in this package's suite driving the unmodified gesture, unedited.
type Input struct {
	Unfocused                         bool
	PanLeft, PanRight, PanUp, PanDown bool
	CursorX, CursorY                  int
	WheelY                            float64
	PrimaryDown                       bool
	Shift                             bool

	SecondaryDown bool
	Ctrl          bool
	Alt           bool
}

// readInput samples this tick's viewer input from the engine.
func readInput() Input {
	cx, cy := ebiten.CursorPosition()
	_, wy := ebiten.Wheel()
	// THE FOUR LETTER KEYS ARE GONE (docs/1028-command-panel contract B4;
	// `DIV-233`): the camera keeps only the arrows, freeing A, D, W and S for
	// the command panel's own accelerators (Attack and Move: `A`, `S` and `D`
	// go to the display switches, B5) and the panel's own key vocabulary
	// generally. B4 restores what this removes — screen-edge panning,
	// unconditional in panIntent below — so a player with no arrow keys is
	// not stranded.
	return Input{
		Unfocused:   !ebiten.IsFocused(),
		PanLeft:     ebiten.IsKeyPressed(ebiten.KeyLeft),
		PanRight:    ebiten.IsKeyPressed(ebiten.KeyRight),
		PanUp:       ebiten.IsKeyPressed(ebiten.KeyUp),
		PanDown:     ebiten.IsKeyPressed(ebiten.KeyDown),
		CursorX:     cx,
		CursorY:     cy,
		WheelY:      wy,
		PrimaryDown: ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft),
		Shift:       ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight),
		// THE SECONDARY LEVEL AND THE TWO MISSION MODIFIERS. EITHER key of each
		// pair, the two being one modifier and not two — the same reading Shift
		// above already has.
		SecondaryDown: ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight),
		Ctrl:          ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight),
		Alt:           ebiten.IsKeyPressed(ebiten.KeyAltLeft) || ebiten.IsKeyPressed(ebiten.KeyAltRight),
	}
}

// Update advances one tick of input: keyboard pan, edge-scroll, wheel zoom, quit.
// It also advances the water counter from measured elapsed time, so the cycle
// runs at the decoded cadence regardless of the engine's frame rate.
//
// Esc closing the window is this entry point's own behaviour and stays here. The
// camera half is step, which the front-end drives too — so the two entry points
// cannot offer different camera behaviour without someone editing one method.
func (v *Viewer) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		v.quit = true
		return ebiten.Termination
	}
	v.step(readInput(), time.Now())
	return nil
}

// step advances the camera one tick from an input snapshot and a timestamp:
// animation, pan (keyboard, edge-scroll and drag, combined into one Pan call),
// then wheel zoom, in that order.
//
// This is the whole of the viewer's camera behaviour, and it is the ONE place it
// lives. Both the standalone viewer and the front-end's map screen call it, so
// "the two must not diverge" is a property of there being a single method rather
// than a convention someone has to remember. Taking the input and the clock as
// parameters is what makes that behaviour reachable in a test at all: before this
// split, no test in the repo could drive a pan, a drag or a zoom.
func (v *Viewer) step(in Input, now time.Time) {
	defer v.updateTooltip(in, now)
	v.syncMapViewport()
	v.advanceAnimation(now)

	// Numerals read this frame's entities and timestamp before the popup return.
	// Player pause holds their age; a popup alone holds drift but permits expiry.
	// Sound delivery and entity retirement use the same wrapper and stay live.
	v.stepNumerals(now)

	// The message line keeps its UI clock through player pause and popups.
	v.stepMessages(now)

	// A selected unit replaces a selected structure for good: the selection
	// holds one object (MISSION-065).
	if len(v.sel) > 0 {
		v.selStructure = InspectionSubject{}
	}

	// Where the cursor is, remembered for the readout to resolve a cell from at
	// the draw. It is stored HERE, in the one method both entry points call, so
	// the standalone viewer and the map screen cannot come to disagree about
	// it; it is a raw screen position and nothing is derived from it on this
	// line, so no camera state can be read at the wrong moment.
	//
	// It stays live under a popup, alone among the statements below: it moves
	// no camera, and the readout is still on screen behind the dim. THE FIRST
	// OF THE THREE DOORS (1026 B4). in.CursorX/CursorY arrive in WINDOW pixels,
	// from ebiten.CursorPosition or from a scenario, and everything below this
	// line — v.cursorX, v.pressX/pressY, panIntent's edge-scroll test and
	// dragIntent's travel — reads them as FRAME pixels. Mapping once here is
	// what leaves every one of those unchanged. The window position is kept
	// FIRST, because the line below overwrites the only copy of it.
	v.winCursorX, v.winCursorY, v.hasWinCursor = in.CursorX, in.CursorY, true

	in.CursorX, in.CursorY = v.windowToFrame(in.CursorX, in.CursorY)

	v.cursorX, v.cursorY, v.hasCursor = in.CursorX, in.CursorY, true

	v.primaryDown = in.PrimaryDown

	// The cursor manager's own clock (B3; AI-CURSOR-172), advanced from the
	// same timestamp every other per-tick animation in this method uses.
	//
	// IT STANDS BELOW THE THREE STATEMENTS ABOVE AND ABOVE THE POPUP RETURN
	// BELOW, and both halves are load-bearing. Below the position, because
	// mapCursorName's edge, minimap and hover branches are all functions of the
	// cursor's own position: run above it, each chose a name for the PREVIOUS
	// tick's position while mapCursorPresent placed the picture at this one
	// (adversarial pass 1). Above the popup return, because a mission-end
	// notice is a popup and skipping the call under one would carry the
	// animated attack sword onto the map list (advanceCursorManager's own doc).
	v.advanceCursorManager(now)

	// A POPUP PINS THE MAP AND DROPS THE GESTURE. All four of the player's ways
	// to the camera are below this line — the pan keys and the edge-scroll
	// term inside panIntent, the drag, and the wheel — so ONE return holds
	// every one of them, where a guard on the two camera calls would have to be
	// two guards and would still leave the drag anchored behind the box.
	//
	// The clock call above is deliberately NOT below it: it takes its baseline
	// before deciding whether to advance, so the held span is skipped rather
	// than banked, and skipping the call would bank it.
	//
	// BOTH LATCHES GO, and the second is not this method's own. dragging is the
	// pan/rectangle anchor, and clearing it is the shipped rule that nothing
	// survives a release, applied to a popup opening: a drag interrupted by a
	// box must not resume from where it was pressed, and the rectangle's outline
	// is drawn from this flag, so it also stops being drawn. held is the gesture
	// resolver's press latch, and the resolver is what the map arm's own gate
	// skips — so a press taken before the popup opened would otherwise still be
	// outstanding after it went, and a release with no press would then be
	// judged a tap at a stale press point.
	//
	// A button still HELD across the dismissal anchors afresh here on the next
	// frame, and may draw a rectangle that selects nothing, because its release
	// finds no press latch. That is the shape a button already held when a map
	// opens has always had, and closing it would mean a third latch carried
	// only to suppress an outline. THE ATTACK MODE GOES WITH THEM. It is state
	// a gesture was holding in exactly the sense the two latches above are: the
	// player was mid-way through aiming an order, and a box has taken the map
	// from under him. Both writers are lowered — the held modifier and the
	// key's own toggle — because a popup takes every map-screen input and not
	// a selected subset of them.
	//
	// LOWERING IT HERE IS WHAT MAKES IT STAY DOWN. The front-end runs this method
	// BEFORE its own popup gate, so the modifier is never read on a frame a popup
	// stands on, and there is no tick on which the raise can outrun this clear.
	// Dismissing the box therefore leaves the mode down until the player asks for
	// it again, which is the same answer focus loss gives.
	if v.popupOpen() {
		v.stepAmbient(now)
		v.dragging, v.held = false, false
		v.rightDragging, v.rightPanned = false, false
		v.paneSecondaryGrab = false
		v.ctrlLatch, v.altLatch, v.shiftLatch = false, false, false
		v.secondaryDown = false
		v.clearAttackMode()
		return
	}

	// THE THREE MODIFIER LATCHES, and the secondary button's own level, stored
	// beside primaryDown above (`AI-KEYMOD-059`). They are assigned rather than
	// tested, so a tick with the key up is what lowers the latch, which is the
	// one-way-raise failure `setAttackHeld`'s own doc names.
	//
	// THEY ARE BELOW THE POPUP RETURN DELIBERATELY. A popup takes every
	// map-screen input, so a modifier held while a box stands must not reach
	// the cursor cascade; the return above leaves them where they were and the
	// clear beside `clearAttackMode` lowers them.
	v.ctrlLatch, v.altLatch, v.shiftLatch = in.Ctrl, in.Alt, in.Shift
	// THE DRAG MARK OUTLIVES THE RELEASE BY EXACTLY ONE TICK, and it has to
	// (`AI-INPUT-127`: "a marked drag performs no cancel"). `command` reads
	// `rightPanned` on the tick carrying `SecondaryReleased`, and `step` runs
	// before `command` every frame, so clearing the mark here on the release
	// tick itself would hand `command` a click every time. It is cleared on the
	// first tick the button has been up for a WHOLE tick, which is the tick
	// after the release edge and before any new press can arrive.
	secondaryWasDown := v.secondaryDown
	if !in.SecondaryDown && !secondaryWasDown {
		v.rightPanned = false
	}
	v.secondaryDown = in.SecondaryDown

	paneHere := v.commandMode && v.panelCaptures(in.CursorX, in.CursorY)
	dx, dy := v.panIntent(in)
	if (in.SecondaryDown || secondaryWasDown) && (v.paneSecondaryGrab || paneHere) {
		// The pane suppresses the pointer's own edge-scroll contribution as
		// well as its drag delta. Keyboard pan is an independent command and
		// remains live, so retain only that term while the secondary gesture
		// belongs to or is delivered over the pane.
		dx, dy = keyboardPanIntent(in)
	}
	// THE SECONDARY BUTTON PANS THE MISSION CAMERA (`AI-INPUT-127`: right down
	// captures and stores the origin, each non-zero cell delta sends the camera
	// message and marks a drag, right up releases). It runs above the primary
	// branch so that a tick holding both buttons pans from the right one and
	// marquees with the left, which are two independent gestures there and
	// here.
	//
	// IT IS GATED ON commandMode. The standalone developer viewer keeps its
	// left-drag pan (below) and gains no right-drag one, because it has no
	// selection, no marquee and nothing for the left button to do instead.
	if v.commandMode {
		// THE CHARACTER PANE OWNS ITS SECONDARY GESTURE (`TOWN-344`). A
		// gesture begun there stays owned after the cursor leaves, and a map
		// gesture that merely enters the pane stops producing camera deltas
		// while it is over the widget. rightDragIntent still advances its
		// anchor, so leaving an unowned hover resumes from the last delivered
		// position rather than banking the distance hidden behind the pane.
		rdx, rdy := v.rightDragIntent(in, !v.paneSecondaryGrab && !paneHere)
		if !v.minimapCaptures(in.CursorX, in.CursorY) {
			dx += rdx
			dy += rdy
		}
	}
	if in.PrimaryDown {
		ddx, ddy := v.dragIntent(in)
		// A GESTURE THAT BEGAN ON THE OPEN INVENTORY WINDOW PANS NOTHING
		// (hotfix: the window swallows its own clicks). dragIntent still runs,
		// so the anchor keeps tracking the cursor and the accumulator keeps
		// rising — what is dropped is the delta reaching the camera, exactly
		// how boxing already drops it from inside dragIntent. It is dropped
		// here instead because invGrab is command's to write and command runs
		// after this (viewer.go's own field comment on invGrab).
		if !v.invGrab {
			dx += ddx
			dy += ddy
		}
	} else {
		// Nothing survives a release: the next press starts a fresh anchor rather
		// than resuming this one.
		v.dragging = false
	}
	if dx != 0 || dy != 0 {
		v.cam.Pan(dx, dy)
	}

	// Over a switched-on pack the wheel scrolls the pack, one element per
	// notch as its end strips do, and never zooms the map (owner).
	if bar, onPack := v.packBarArea(); in.WheelY != 0 && onPack && image.Pt(in.CursorX, in.CursorY).In(bar) {
		if in.WheelY > 0 {
			v.ScrollPack(-1)
		} else {
			v.ScrollPack(+1)
		}
	} else if in.WheelY != 0 {
		factor := WheelZoomStep
		if in.WheelY < 0 {
			factor = 1 / WheelZoomStep
		}
		v.cam.ZoomAbout(float64(in.CursorX), float64(in.CursorY), factor)
	}
	v.stepAmbient(now)
}

// advanceAnimation updates the wall baseline even while its accumulator is held.
// Player pause retains count and remainder through the first resumed sample,
// so an unsampled pause tail cannot become a burst of ticks. A popup or disabled
// animation also consumes its wall span. The first paced call sets the baseline;
// later running calls feed whole microseconds, dropping the sub-microsecond tail.
func (v *Viewer) advanceAnimation(now time.Time) {
	held := v.playerPaused || v.animationPaused
	v.animationPaused = v.playerPaused
	if v.unpaced {
		// The unpaced owner loop has no deadline test. Updating last still
		// consumes the real-time span so a later paced restore owes none of it.
		v.last = now
		if v.animate && !v.popupOpen() && !held {
			v.anim.AdvanceOne()
		}
		return
	}
	if v.last.IsZero() {
		v.last = now
		return
	}
	elapsed := now.Sub(v.last)
	v.last = now
	if !v.animate || v.popupOpen() || held {
		return
	}
	v.anim.AdvanceMicros(int(elapsed / time.Microsecond))
}

// panIntent resolves this tick's pan from the keyboard and the cursor's proximity
// to the window edges, in world pixels.
func (v *Viewer) panIntent(in Input) (dx, dy float64) {
	dx, dy = keyboardPanIntent(in)

	left, right, top, bottom := v.edgeScrollBands(in.PrimaryDown)
	if left {
		dx -= PanSpeed
	}
	if right {
		dx += PanSpeed
	}
	if top {
		dy -= PanSpeed
	}
	if bottom {
		dy += PanSpeed
	}
	return dx, dy
}

// keyboardPanIntent is panIntent's explicit-key term without pointer edge
// scrolling. The character pane uses it while a secondary gesture belongs to
// that widget: the pointer cannot leak a pan through the pane, while arrow-key
// input remains a separate live command.
func keyboardPanIntent(in Input) (dx, dy float64) {
	if in.PanLeft {
		dx -= PanSpeed
	}
	if in.PanRight {
		dx += PanSpeed
	}
	if in.PanUp {
		dy -= PanSpeed
	}
	if in.PanDown {
		dy += PanSpeed
	}
	return dx, dy
}

// dragIntent resolves this tick's pan from a primary-button drag, in world
// pixels. It is called only while in.PrimaryDown; step itself clears
// v.dragging once the button reads up, since dragIntent has nothing to decide
// about a button that is not held.
//
// The world point under the cursor must stay under the cursor, so the camera
// moves OPPOSITE the cursor's screen delta, and that delta is divided by the
// zoom — the same cursor-anchoring principle ZoomAbout already applies to
// a zoom, so the two gestures cannot disagree with each other.
//
// It also RAISES THE SLOP ACCUMULATOR, by |sdx| + |sdy| — the same delta
// it hands the camera, before the zoom divides it — and zeroes it on the
// anchor tick where a gesture begins. Keeping that here, rather than in a
// second delta computed beside it, is the whole of why the accumulator
// cannot disagree with the pan: there is one subtraction.
//
// The anchor branch is also where the gesture is LATCHED: the press point is
// stored and boxing is decided from the command mode alone, as it stands on
// that tick (`v.boxing = v.commandMode`, F4/round 2). A press in command
// mode is a selection rectangle; a press on a viewer with no world under it
// is the pan this method has always performed. Nothing reads in.Shift on
// THIS tick, so a modifier pressed or released mid-drag cannot change which
// gesture this was latched as.
func (v *Viewer) dragIntent(in Input) (dx, dy float64) {
	if !v.dragging {
		v.dragging = true
		v.dragX, v.dragY = in.CursorX, in.CursorY
		v.pressX, v.pressY = in.CursorX, in.CursorY
		// THE LEFT BUTTON ALWAYS MARQUEES ON THE MISSION MAP (`AI-INPUT-121`).
		// Until this story it marqueed only without Shift and panned with it,
		// which is this project's own arrangement: the decoded left contract has
		// no pan at all, and Shift is the selection routine's own toggle modifier
		// (`AI-SELECT-122`), not a second gesture. The camera is the RIGHT
		// button's (`AI-INPUT-127`).
		v.boxing = v.commandMode
		v.dragMoved = 0
		return 0, 0
	}

	sdx := in.CursorX - v.dragX
	sdy := in.CursorY - v.dragY
	v.dragX, v.dragY = in.CursorX, in.CursorY
	v.dragMoved += absInt(sdx) + absInt(sdy)

	if v.boxing {
		return 0, 0
	}

	z := v.cam.Zoom
	return -float64(sdx) / z, -float64(sdy) / z
}

// rightDragIntent is dragIntent's counterpart for the SECONDARY button, and
// it is the mission map's only pan by drag (`AI-INPUT-127`: right down
// captures and stores the origin, each delivered move pans, right up
// releases).
//
// IT MARKS THE DRAG rather than measuring a slop threshold, which is the
// decoded shape: the right contract has no threshold anywhere. `rightPanned` is
// raised on the first tick that actually moves the camera, and `command` reads
// it at the release to tell a drag from a click — a marked drag performs no
// cancel and a click cancels an armed mode or deselects everything.
//
// A PRESS AND A RELEASE IN ONE FRAME PANS NOTHING AND IS A CLICK, because the
// anchor branch returns zero by construction exactly as dragIntent's does.
// pan is whether the surface under this sample belongs to the map. A false
// value still advances the anchor but neither moves nor marks the gesture;
// this is how the mission character pane absorbs movement without banking a
// delta that fires when an unowned gesture leaves it.
func (v *Viewer) rightDragIntent(in Input, pan bool) (dx, dy float64) {
	if !in.SecondaryDown {
		// Nothing survives a release: the next press starts a fresh anchor.
		v.rightDragging = false
		return 0, 0
	}
	if !v.rightDragging {
		v.rightDragging = true
		v.rightDragX, v.rightDragY = in.CursorX, in.CursorY
		return 0, 0
	}
	sdx := in.CursorX - v.rightDragX
	sdy := in.CursorY - v.rightDragY
	v.rightDragX, v.rightDragY = in.CursorX, in.CursorY
	if sdx == 0 && sdy == 0 {
		return 0, 0
	}
	if !pan {
		return 0, 0
	}
	v.rightPanned = true
	z := v.cam.Zoom
	return -float64(sdx) / z, -float64(sdy) / z
}

func (v *Viewer) shotScreenRects() []screenRect {
	var out []screenRect
	for _, e := range v.entities {
		if e.Shot == nil {
			continue
		}
		// AND ONLY WHILE THE SHOOTER MAY BE SEEN (hotfix). The mark is keyed off
		// the entity's own cell above and below alike, so the gate is asked about
		// that cell too — a shot leaving an enemy in the fog is as much an enemy
		// indicator as his health bar, and the owner's instruction is the set and
		// not the symptom.
		if !v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
			continue
		}
		// THE HALF CELL IS THE CELL'S OWN CENTRE (1003), EffectGroundPoint's
		// term in spellbolt.go and StaticAnchor's decoded `col*CellSize +
		// CellSize/2`. Without it the mark sat on the cell's top-left corner,
		// 16 px from the art it measures, which is a mark that reports a
		// disagreement it created itself.
		px := e.Shot.X*terrain.CellSize/ShotScale + terrain.CellSize/2
		py := e.Shot.Y*terrain.CellSize/ShotScale + terrain.CellSize/2
		const half = ShotMarkSize / 2
		arm := image.Rect(px-half, py-half, px-half+ShotMarkSize, py-half+ShotMarkSize)
		if r, ok := v.placeArm(e.Cell, arm); ok {
			out = append(out, r)
		}
	}
	return out
}

// Draw paints only the tiles intersecting the current view, each placed
// through the camera transform for the mode the viewer is in, then the map's
// static-object art over them, then the pass slice over that — the entity
// layer's two content passes and the opt-in diagnostic overlays, in the
// slice's own order.
//
// The two terrain paths differ in geometry alone. Both read the same tile
// word, resolve it through the same water-substituting mapping and take the
// same cached GPU image, so a cell's source pixels are a property of the
// cell and the tick, never of the mode. Draw composes the mission screen on
// its current height-768 canvas and then places that canvas in the window
// with one uniform scale. Layout expands the logical width first, so wide
// windows have no vertical pillarbox.
//
// A window with no usable placement draws nothing rather than drawing at window
// resolution: the only such window has a non-positive side, and there is no
// frame in it to see.
func (v *Viewer) Draw(screen *ebiten.Image) {
	if !v.place.Valid() {
		return
	}
	// Selection and panel contents can change during Update, after Layout has
	// already run for this frame. Refresh the actual map surface before any
	// world pixel is composed.
	v.syncMapViewport()
	if v.canvas == nil {
		v.canvas = ebiten.NewImage(v.frameW, v.frameH)
	}
	// Clear the whole frame so neither the map nor HUD leaves pixels from the
	// previous composition after state or logical width changes.
	v.canvas.Clear()
	v.canvasLog.reset(v.canvas.Bounds())
	if v.textSmoothingEnabled {
		// Method C captures the character panel, mission card, hover tooltip
		// and native-size notice. ResetCapture clears the previous frame.
		text.ResetCapture()
	}
	v.drawFrame(v.canvas)
	v.canvasLog.end()
	var capturedText []text.DrawCall
	if v.textSmoothingEnabled {
		capturedText = append(capturedText, text.Captured()...)
	}
	v.textCalls = capturedText

	ox, oy := v.place.Origin()
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(v.place.Scale(), v.place.Scale())
	op.GeoM.Translate(ox, oy)
	op.Filter = ebiten.FilterNearest
	v.textCaptured = len(capturedText)
	frameCanvas, capturedText := settleText(v.canvas, &v.canvasLog, capturedText, &v.textEraser,
		&v.settledBuf, &v.settledTex, &v.textSettleFallbacks)
	v.textKept = len(capturedText)
	frameSmoothingOff = v.frameSmoothingOff
	blitFrameCanvas(screen, frameCanvas, &op)
	if len(capturedText) > 0 {
		v.textOverlay.draw(screen, capturedText, v.place.Scale(), ox, oy)
	}
}

// blitFrameCanvas places the composed frame canvas on the window screen.
//
// It is a package variable so the mission screen's geometry test can observe
// the transform Draw builds. Ebitengine refuses a pixel readback with no
// graphics context, so recording the call Draw makes is the only way to see
// what Draw drew; asserting the transform from a helper that Draw also calls
// would witness the helper rather than the call. Production never replaces it.
//
// THE DEFAULT DRAWS THROUGH drawFinalFrame: op carries the same scale and
// origin Draw always computed — the geometry test above still observes
// exactly that request — and the owner's FrameSmoothing scaler realizes it.
var blitFrameCanvas = func(screen, canvas *ebiten.Image, op *ebiten.DrawImageOptions) {
	drawFinalFrame(screen, canvas, op, &missionFrameSharpBuf, frameSmoothingOff)
}

// disposeFrameCanvas releases the old GPU backing when Layout changes the
// logical bounds. Kept as a package variable so the geometry test observes the
// production disposal call without relying on Ebitengine internals.
var disposeFrameCanvas = func(canvas *ebiten.Image) { canvas.Dispose() }

var blitColumnLayer = func(screen, img *ebiten.Image, op *ebiten.DrawImageOptions, layer string) {
	screen.DrawImage(img, op)
}

// drawFrame composes the mission screen onto one frame-sized destination.
func (v *Viewer) drawFrame(screen *ebiten.Image) {
	mapSurface := screen.SubImage(image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH).Intersect(screen.Bounds())).(*ebiten.Image)
	// THE HOVER-LIGHTING CACHE, before either terrain or the plane sprites —
	// both read it, and hoverLitID's own doc states why it cannot be read
	// lazily from inside planeSprites itself (hotfix, owner report 1,
	// DIV-1349).
	v.refreshHoverLighting()

	// THE STRUCTURE-LIGHT MASK, for the same reason and at the same point:
	// spellTerrainScale/spellSpriteFactor (overlay.go) read it below, and
	// v.anim's own pulse must read one consistent swing for every vertex and
	// sprite this frame draws (hotfix, owner report 2, DIV-1313 amended;
	// towerglow.go).
	v.refreshStructureLighting()

	if v.Mode() == ModeDisplaced {
		v.drawDisplaced(mapSurface)
	} else {
		v.drawFlat(mapSurface)
	}

	// drawArt owns the complete retained-overlay/shadow/projectile/body order
	// (ANIM-047). Its existing static-object depth merge stays inside the body
	// pass; all of this content remains below diagnostic glyphs and shroud.
	v.drawArt(mapSurface)
	if v.editorView {
		return
	}

	// The separate Heal/Drain shower retains its existing pass.
	v.drawHealArt(mapSurface)
	v.drawShroud(mapSurface)
	if v.playerPaused {
		r := image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH)
		wash := color.RGBA{A: 32}
		vector.DrawFilledRect(screen, 0, 0, float32(r.Dx()), float32(r.Dy()), wash, false)
		v.canvasLog.overSolid(r, wash)
	}

	for _, s := range v.gridScreenSegments() {
		vector.StrokeLine(screen, float32(s.X0), float32(s.Y0), float32(s.X1), float32(s.Y1),
			float32(gridLineWidth)*float32(v.cam.Zoom), terrain.CellGridColor, false)
	}

	// The pass slice: the entity layer's square half, then the diagnostics —
	// placed objects (0008), placed units (0009), the static-object cells
	// (0017). overlayPasses returns them already in draw order and omits any
	// pass with nothing to draw, so this costs nothing and draws nothing by
	// default. Iterating that slice rather than writing one loop per pass is
	// deliberate: it makes the squares -> objects -> units -> statics order a
	// value a test can read.
	//
	// EVERY PASS IS RECTANGLES NOW. The entity layer's SPRITE half was the one
	// pass here that carried textures, and it has moved into the content band
	// drawArt paints above — where a unit takes its place in the same
	// back-to-front order the structure and object planes already share,
	// instead of standing in front of every building on the map. THE LATTICE
	// MOVED OUT OF THIS SLICE (item-2 hotfix) for the same reason: it is
	// strokes, not filled rectangles, and is drawn immediately above instead.
	//
	// The coordinates are passed through as computed — the only rounding is the
	// float32 the drawer takes, never a pixel snap of our own.
	for _, pass := range v.overlayPasses() {
		for _, rect := range pass.Rects {
			if pass.HalfAdd {
				v.drawStatusBarHalfRect(screen, rect, pass.Color)
				continue
			}
			vector.DrawFilledRect(screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H),
				pass.Color, false)
		}
	}

	for _, r := range v.shotScreenRects() {
		vector.DrawFilledRect(screen, float32(r.X), float32(r.Y), float32(r.W), float32(r.H),
			ShotMarkerColor, false)
	}

	// The path overlay, last of all. It is over every glyph rather than in the
	// pass slice because it is the one instrument drawn as STROKES and not as
	// filled quads, and an overlayPass carries rectangles: folding a polyline
	// into that type would widen every pass with a field only one of them could
	// ever hold. It draws nothing at all when nothing is selected and when no
	// selected unit is under orders, which is what keeps a viewer that was
	// never handed a route byte-identical here; when units ARE selected and
	// under orders it draws every one of their lines, however many that is
	// (FR-8a).
	for _, s := range v.displayedPathSegments() {
		vector.StrokeLine(screen, float32(s.X0), float32(s.Y0), float32(s.X1), float32(s.Y1),
			PathWidth, PathColor, false)
	}

	// THE DAMAGE NUMERALS, over every world glyph and under every box. They are
	// the map's own content — the original draws them from its map view's
	// paint — so they go over the units, the bars and the route strokes they
	// belong to, and under the panel, the readout, the dim and the notice, none
	// of which the map may cover.
	//
	// They are outside the pass slice for the reason the two boxes below are: a
	// pass carries filled rectangles, and these are images.
	//
	// numeralPlacements decides everything and this decides nothing — the shape
	// every other picture here has. It answers with nothing for a viewer holding
	// no font, no entities or no figures, and then not a single engine call is
	// made, so the standalone developer viewer and every map screen with no blow
	// on it compose exactly the frame they composed before this story.
	//
	// The picture is blitted at NATIVE SIZE while its anchor came through the
	// camera: the offset scales with the zoom so a figure stays the same distance
	// out of its unit, and the glyph does not, because a numeral is something to
	// read.
	numeralCapture := beginTextCapture(v.textSmoothingEnabled)
	for i, p := range v.numeralPlacements() {
		for len(v.numeralImgs) <= i {
			v.numeralImgs = append(v.numeralImgs, nil)
		}
		pic := p.Pic
		if p.Blit != nil && v.textSmoothingEnabled {
			pic = p.Blit
			text.Append(p.Calls, p.At.X, p.At.Y)
		}
		b := pic.Bounds()
		if v.numeralImgs[i] == nil ||
			v.numeralImgs[i].Bounds().Dx() != b.Dx() || v.numeralImgs[i].Bounds().Dy() != b.Dy() {
			v.numeralImgs[i] = ebiten.NewImage(b.Dx(), b.Dy())
		}
		v.numeralImgs[i].WritePixels(pic.Pix)
		var op ebiten.DrawImageOptions
		op.Filter = ebiten.FilterNearest
		op.GeoM.Translate(float64(p.At.X), float64(p.At.Y))
		screen.DrawImage(v.numeralImgs[i], &op)
	}
	endTextCapture(numeralCapture, image.Point{})

	// The mission surface ends at the top of its open bottom panels. World
	// layers are deliberately composed first, then this opaque HUD backing
	// removes every overhang below the live viewport before any panel or popup
	// is painted. The map is therefore neither visible nor interactive under a
	// book or inventory bar.
	if v.cam.ViewH < v.frameH {
		vector.DrawFilledRect(screen, 0, float32(v.cam.ViewH), float32(v.cam.ViewW),
			float32(v.frameH-v.cam.ViewH), invFill, false)
		v.canvasLog.overSolid(image.Rect(0, v.cam.ViewH, v.cam.ViewW, v.frameH), invFill)
	}
	if v.cam.ViewW < v.frameW {
		vector.DrawFilledRect(screen, float32(v.cam.ViewW), 0, float32(v.frameW-v.cam.ViewW),
			float32(v.frameH), invFill, false)
		v.canvasLog.overSolid(image.Rect(v.cam.ViewW, 0, v.frameW, v.frameH), invFill)
	}

	// The unit information panel, after everything. It is outside the pass
	// slice for the reason the path overlay is: a pass carries filled
	// rectangles and sprite placements in WORLD coordinates through the camera,
	// and this is one image in WINDOW coordinates, so folding it in would widen
	// every pass with a field only one of them could hold.
	//
	// panelPresent decides everything and this decides nothing — which is what
	// leaves the panel's whole frame logic reachable without a window. It
	// answers false for a viewer with no font and for one with no subject, and
	// then not a single engine call is made here, so a viewer that was never
	// given either draws exactly the frame it drew before this story.
	//
	// The upload happens only for a picture that has not had one, and reuses
	// the image already there whenever the box has not changed size.
	//
	// Each text-bearing HUD picture captures in its own local coordinates,
	// then shifts its glyphs by the same origin as its blit.
	{
		panelCapture := beginTextCapture(v.textSmoothingEnabled)
		pic, at, ok := v.panelPresent()
		endTextCapture(panelCapture, at)
		if ok {
			if v.panelFresh {
				b := pic.Bounds()
				if v.panelImg == nil || v.panelImg.Bounds().Dx() != b.Dx() || v.panelImg.Bounds().Dy() != b.Dy() {
					v.panelImg = ebiten.NewImage(b.Dx(), b.Dy())
				}
				v.panelImg.WritePixels(pic.Pix)
				v.panelFresh = false
			}
			var op ebiten.DrawImageOptions
			op.Filter = ebiten.FilterNearest
			op.GeoM.Translate(float64(at.X), float64(at.Y))
			blitColumnLayer(screen, v.panelImg, &op, "panel")
			v.canvasLog.over(pic, at)
		}
	}

	// THE SPELLBOOK BAR, above an open pack bar or at the bottom when alone
	// (plan T6, placed where it now is by the owner's 0140 ruling — hud.go).
	// It is outside the pass slice for the panel's own reason immediately
	// above: one image in window coordinates, not a rectangle in world ones.
	//
	// spellbookPresent decides everything and this decides nothing — the
	// panel's own shape. It answers false for a viewer with no font and for
	// one holding no book, and then not a single engine call is made here,
	// so a viewer that never had a spell pushed to it, or whose selected
	// unit knows none, draws exactly the frame it drew before this story.
	//
	// IT RE-UPLOADS ITS TEXTURE EVERY FRAME IT DRAWS, the minimap's own
	// choice and not the panel's fresh-flagged cache (spellbookImg's own
	// field comment, viewer.go): a grid of small cells costs less to
	// recompose than a rebuild key would save.
	{
		bookCapture := beginTextCapture(v.textSmoothingEnabled)
		pic, at, ok := v.spellbookPresent()
		endTextCapture(bookCapture, at)
		if ok {
			b := pic.Bounds()
			if v.spellbookImg == nil || v.spellbookImg.Bounds().Dx() != b.Dx() || v.spellbookImg.Bounds().Dy() != b.Dy() {
				v.spellbookImg = ebiten.NewImage(b.Dx(), b.Dy())
			}
			v.spellbookImg.WritePixels(pic.Pix)
			var op ebiten.DrawImageOptions
			op.Filter = ebiten.FilterNearest
			op.GeoM.Translate(float64(at.X), float64(at.Y))
			screen.DrawImage(v.spellbookImg, &op)
			v.canvasLog.over(pic, at)
		}
	}

	// The debug readout, last of all and in the other corner. It is the panel's
	// own shape at every step — decided entirely by readoutPresent, which
	// needs no window, uploaded once per composition, placed by the same corner
	// arithmetic — so the two boxes cannot come to disagree about how a box
	// reaches the screen.
	//
	// THE ENGINE'S FRAME RATE IS READ HERE AND NOWHERE ELSE. Handing it in is
	// what leaves the whole of what the readout states reachable in a test with
	// no engine: a measurement taken inside readoutPresent would be a value no
	// assertion could pin.
	{
		readoutCapture := beginTextCapture(v.textSmoothingEnabled)
		pic, at, ok := v.readoutPresent(int(ebiten.ActualFPS() + 0.5))
		endTextCapture(readoutCapture, at)
		if ok {
			if v.readoutFresh {
				b := pic.Bounds()
				if v.readoutImg == nil || v.readoutImg.Bounds().Dx() != b.Dx() || v.readoutImg.Bounds().Dy() != b.Dy() {
					v.readoutImg = ebiten.NewImage(b.Dx(), b.Dy())
				}
				v.readoutImg.WritePixels(pic.Pix)
				v.readoutFresh = false
			}
			var op ebiten.DrawImageOptions
			op.Filter = ebiten.FilterNearest
			op.GeoM.Translate(float64(at.X), float64(at.Y))
			screen.DrawImage(v.readoutImg, &op)
			v.canvasLog.over(pic, at)
		}
	}

	// THE MINIMAP, beside the readout and in the OTHER free corner
	// (minimap.go). Top left is the readout's own and bottom left is the unit
	// panel's (readout.go's own "THE CORNER IS THE OTHER ONE"); top right is
	// simply free, so this project's third authored box docks there.
	//
	// minimapPresent decides everything and this decides nothing — the same
	// shape the panel and the readout share immediately above. It answers
	// false only when the map has no drawable geometry.
	//
	// IT RE-UPLOADS ITS TEXTURE EVERY FRAME IT DRAWS, mirroring the
	// damage-numeral images a few hundred lines up rather than the panel's
	// and the readout's own fresh-flag: minimapPresent's own doc explains
	// why a rebuild key would cost what it saves.
	//
	// IT STANDS BEFORE THE POPUP DIM below, beside the panel and the readout
	// and under their own reasoning (below): it is a HUD instrument a running
	// popup already darkens along with everything else, not an exception carved
	// out for it.
	{
		_, pic, at, ok := v.FPSReadout(time.Now().UnixMilli())
		if ok {
			if v.fpsFresh {
				b := pic.Bounds()
				if v.fpsImg == nil || v.fpsImg.Bounds().Dx() != b.Dx() || v.fpsImg.Bounds().Dy() != b.Dy() {
					v.fpsImg = ebiten.NewImage(b.Dx(), b.Dy())
				}
				v.fpsImg.WritePixels(pic.Pix)
				v.fpsFresh = false
			}
			var op ebiten.DrawImageOptions
			op.Filter = ebiten.FilterNearest
			op.GeoM.Translate(float64(at.X), float64(at.Y))
			screen.DrawImage(v.fpsImg, &op)
			v.canvasLog.over(pic, at)
		}
	}

	if pic, at, ok := v.minimapPresent(); ok {
		b := pic.Bounds()
		if v.minimapImg == nil || v.minimapImg.Bounds().Dx() != b.Dx() || v.minimapImg.Bounds().Dy() != b.Dy() {
			v.minimapImg = ebiten.NewImage(b.Dx(), b.Dy())
		}
		v.minimapImg.WritePixels(pic.Pix)
		var op ebiten.DrawImageOptions
		op.Filter = ebiten.FilterNearest
		op.GeoM.Translate(float64(at.X), float64(at.Y))
		blitColumnLayer(screen, v.minimapImg, &op, "minimap")
		v.canvasLog.over(pic, at)
	}

	if _, shows, _ := v.DialogueBackdropPlan(); shows == 0 {
		if r, c, ok := v.noticeBackdropOf(); ok {
			vector.DrawFilledRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
			v.canvasLog.overSolid(r, c)
		}
		v.paintNotice(screen)
	}

	// The marker is a STROKED rectangle rather than a member of the pass slice: a
	// pass carries FILLED rectangles, and a filled one would cover the very unit
	// it is pointing at. It is drawn under the pointer so the pointer is never
	// hidden by the thing it is naming.
	//
	// Both are decided entirely above, by two pure methods that need no window,
	// and these statements decide nothing — the three boxes' own shape. They draw
	// nothing at all for a viewer whose mode is down, which is every viewer built
	// before this story and the standalone developer viewer under every input.
	if r, ok := v.attackTargetRect(); ok {
		vector.StrokeRect(screen, float32(r.X), float32(r.Y), float32(r.W), float32(r.H),
			AttackMarkerWidth, AttackMarkerColor, false)
		v.canvasLog.unknown(image.Rect(int(math.Floor(r.X)), int(math.Floor(r.Y)),
			int(math.Ceil(r.X+r.W)), int(math.Ceil(r.Y+r.H))).Inset(-AttackMarkerWidth - 1))
	}

	// THE POINTER ITSELF IS NOT DRAWN HERE ANY MORE (owner): all three cursor
	// pictures are drawPointer's, below every box this method composes. The
	// marker above stays where it is in the DRAW ORDER — it is an outline
	// around a unit standing on the map, not a pointer, and moving it into
	// drawPointer's slice would put a world annotation over the doll and the
	// control panel, which contradicts its own nature.
	//
	// ITS DRAW ORDER PUTS IT OVER TWO OF THE FOUR RIGHT-COLUMN BOXES AND UNDER
	// THE OTHER TWO (adversarial pass 2, F2): this statement used to say it
	// stayed "under the panels", which was true of neither the panel block
	// above nor the minimap above it — the marker is composed after both. What
	// keeps it off the column is not the draw order, which was never the right
	// instrument for that, but attackTargetRect's own clamp to the world
	// viewport (clipScreenRectToViewport): the marker annotates a unit
	// standing on the map, so it is confined to the same rect that clips the
	// world it annotates.
	//
	// THE CLAMP ITSELF INSETS BY THE STROKE'S OWN HALF-WIDTH (adversarial pass
	// 3): a rect clipped flush to the viewport edge still painted one pixel
	// column into the panel, because vector.StrokeRect centres its line on the
	// rect's edge rather than drawing inside it. The comment here used to say
	// the marker "never strokes into the column regardless of where in the draw
	// order it falls", which was true of the RECT clipScreenRectTo Viewport
	// returned and false of the PAINT vector.StrokeRect produced from it —
	// the two are not the same claim, and only the first was checked.
	//
	// A VIEWER INSIDE AN APP SESSION MAKES NO ENGINE CALL HERE: that App told
	// the engine once for the whole frame, from the same three answers, through
	// flow.pointerWanted.
	if _, _, shown := v.attackPointerPresent(); v.cursorMgr == nil && v.pointerModeChange(shown) {
		mode := ebiten.CursorModeVisible
		if shown {
			mode = ebiten.CursorModeHidden
		}
		ebiten.SetCursorMode(mode)
	}

	// MENU-COMBAT-017
	if pic, at, ok := v.columnFillerPresent(); ok {
		if v.fillerImg == nil {
			v.fillerImg = ebiten.NewImageFromImage(pic)
		}
		var op ebiten.DrawImageOptions
		op.Filter = ebiten.FilterNearest
		op.GeoM.Translate(float64(at.X), float64(at.Y))
		blitColumnLayer(screen, v.fillerImg, &op, "filler")
		if rgba, isRGBA := pic.(*image.RGBA); isRGBA {
			v.canvasLog.over(rgba, at)
		} else {
			v.canvasLog.unknown(pic.Bounds().Sub(pic.Bounds().Min).Add(at))
		}
	}

	// Widget 8 replaces the lower statistics card's rows while its readout is
	// drawn; the card's body art stays under the readout.
	readoutCapture := beginTextCapture(v.textSmoothingEnabled)
	readoutPic, readoutAt, readoutShown := v.structureReadoutPresent()
	endTextCapture(readoutCapture, readoutAt)
	{
		missionCardCapture := beginTextCapture(v.textSmoothingEnabled)
		pic, at, ok := v.missionCardPresentWith(readoutShown)
		endTextCapture(missionCardCapture, at)
		if ok {
			if v.missionCardFresh {
				b := pic.Bounds()
				if v.missionCardImg == nil || v.missionCardImg.Bounds().Dx() != b.Dx() || v.missionCardImg.Bounds().Dy() != b.Dy() {
					v.missionCardImg = ebiten.NewImage(b.Dx(), b.Dy())
				}
				v.missionCardImg.WritePixels(pic.Pix)
				v.missionCardFresh = false
			}
			var op ebiten.DrawImageOptions
			op.Filter = ebiten.FilterNearest
			op.GeoM.Translate(float64(at.X), float64(at.Y))
			blitColumnLayer(screen, v.missionCardImg, &op, "missionCard")
			v.canvasLog.over(pic, at)
		}
	}

	if readoutShown {
		pic, at := readoutPic, readoutAt
		if v.structFresh {
			b := pic.Bounds()
			if v.structImg == nil || v.structImg.Bounds().Dx() != b.Dx() || v.structImg.Bounds().Dy() != b.Dy() {
				v.structImg = ebiten.NewImage(b.Dx(), b.Dy())
			}
			v.structImg.WritePixels(pic.Pix)
			v.structFresh = false
		}
		var op ebiten.DrawImageOptions
		op.Filter = ebiten.FilterNearest
		op.GeoM.Translate(float64(at.X), float64(at.Y))
		blitColumnLayer(screen, v.structImg, &op, "structureReadout")
		v.canvasLog.over(pic, at)
	}

	// THE WORN BLIT GOES THROUGH blitColumnLayer TOO, since round 2 (W-2): it
	// used to call screen.DrawImage directly, which is not the hooked seam
	// missioncolumn_test.go's recordColumnLayers intercepts, so a test
	// asserting the worn layer does not draw passed even after it started
	// drawing at 1024x768 — the recorder was blind to the very layer the
	// assertion was about, not agreeing with it.
	if pic, at, ok := v.wornPresent(); ok {
		if v.invFresh {
			b := pic.Bounds()
			if v.invImg == nil || v.invImg.Bounds().Dx() != b.Dx() || v.invImg.Bounds().Dy() != b.Dy() {
				v.invImg = ebiten.NewImage(b.Dx(), b.Dy())
			}
			v.invImg.WritePixels(pic.Pix)
			v.invFresh = false
		}
		var op ebiten.DrawImageOptions
		op.Filter = ebiten.FilterNearest
		op.GeoM.Translate(float64(at.X), float64(at.Y))
		blitColumnLayer(screen, v.invImg, &op, "worn")
		v.canvasLog.over(pic, at)
	}

	// The pack bar composes its picture every frame, so the counts on its
	// cells are captured for the overlay the frame they are drawn.
	{
		packCapture := beginTextCapture(v.textSmoothingEnabled)
		pic, at, ok := v.packBarPresent()
		endTextCapture(packCapture, at)
		if ok {
			b := pic.Bounds()
			if v.packImg == nil || v.packImg.Bounds().Dx() != b.Dx() || v.packImg.Bounds().Dy() != b.Dy() {
				v.packImg = ebiten.NewImage(b.Dx(), b.Dy())
			}
			v.packImg.WritePixels(pic.Pix)
			var op ebiten.DrawImageOptions
			op.Filter = ebiten.FilterNearest
			op.GeoM.Translate(float64(at.X), float64(at.Y))
			screen.DrawImage(v.packImg, &op)
			v.canvasLog.over(pic, at)
		}
	}

	// THE COMMAND PANEL LAST OF THE FOUR, because it is the one that must never
	// be covered: every other box down here can be switched off from it, and a
	// switch under something else is a switch the player cannot reach. Nothing
	// currently overlaps it — hudStackTops reserves its own step — so the
	// order costs nothing today and is what keeps the guarantee true if a later
	// box lands on the same pixels.
	if pic, at, ok := v.commandPanelPresent(); ok {
		b := pic.Bounds()
		if v.commandPanelImg == nil || v.commandPanelImg.Bounds().Dx() != b.Dx() || v.commandPanelImg.Bounds().Dy() != b.Dy() {
			v.commandPanelImg = ebiten.NewImage(b.Dx(), b.Dy())
		}
		v.commandPanelImg.WritePixels(pic.Pix)
		var op ebiten.DrawImageOptions
		op.Filter = ebiten.FilterNearest
		op.GeoM.Translate(float64(at.X), float64(at.Y))
		blitColumnLayer(screen, v.commandPanelImg, &op, "controlPanel")
		v.canvasLog.over(pic, at)
	}

	// THE DROP GOLD EDITOR is a modal over every HUD box and under the message
	// line. Its text is captured for the overlay like the pack bar's counts.
	{
		goldCapture := beginTextCapture(v.textSmoothingEnabled)
		pic, at, ok := v.goldModalPresent()
		endTextCapture(goldCapture, at)
		if ok {
			b := pic.Bounds()
			if v.goldImg == nil || v.goldImg.Bounds().Dx() != b.Dx() || v.goldImg.Bounds().Dy() != b.Dy() {
				v.goldImg = ebiten.NewImage(b.Dx(), b.Dy())
			}
			v.goldImg.WritePixels(pic.Pix)
			var op ebiten.DrawImageOptions
			op.Filter = ebiten.FilterNearest
			op.GeoM.Translate(float64(at.X), float64(at.Y))
			screen.DrawImage(v.goldImg, &op)
			v.canvasLog.over(pic, at)
		}
	}

	// THE MESSAGE LINE, last of all — over the inventory window too: it takes
	// no input and gates nothing, so its place in the paint order is a
	// visibility question alone, never a modality one.
	//
	// messagePresent decides everything and this decides nothing: it answers
	// false for a viewer with no font and one with no lines, and then no
	// engine call is made here. The upload happens only for a picture that has
	// not had one, and reuses the image already there while the box keeps its
	// size.
	{
		messageCapture := beginTextCapture(v.textSmoothingEnabled)
		pic, at, ok := v.messagePresent()
		endTextCapture(messageCapture, at)
		if ok {
			blit := v.messageBlitOf(pic)
			if v.messageFresh {
				b := blit.Bounds()
				if v.messageImg == nil || v.messageImg.Bounds().Dx() != b.Dx() || v.messageImg.Bounds().Dy() != b.Dy() {
					v.messageImg = ebiten.NewImage(b.Dx(), b.Dy())
				}
				v.messageImg.WritePixels(blit.Pix)
				v.messageFresh = false
			}
			var op ebiten.DrawImageOptions
			op.Filter = ebiten.FilterNearest
			op.GeoM.Translate(float64(at.X), float64(at.Y))
			screen.DrawImage(v.messageImg, &op)
			v.canvasLog.over(blit, at)
		}
	}

	// Every hover family uses the same delayed controller and final overlay.
	// The third of the three mission HUD boxes DIV-1385 smooths — see the
	// panel above.
	{
		tooltipCapture := beginTextCapture(v.textSmoothingEnabled)
		pic, at, ok := v.tooltipPresent()
		endTextCapture(tooltipCapture, at)
		if ok {
			b := pic.Bounds()
			if v.itemPopupImg == nil || v.itemPopupImg.Bounds().Size() != b.Size() {
				if v.itemPopupImg != nil {
					v.itemPopupImg.Dispose()
				}
				v.itemPopupImg = ebiten.NewImage(b.Dx(), b.Dy())
			}
			v.itemPopupImg.WritePixels(pic.Pix)
			var op ebiten.DrawImageOptions
			op.GeoM.Translate(float64(at.X), float64(at.Y))
			screen.DrawImage(v.itemPopupImg, &op)
			v.canvasLog.over(pic, at)
		}
	}

	if _, shows, _ := v.DialogueBackdropPlan(); shows > 0 {
		v.dialogueBackdrop.draw(screen, &v.canvasLog, shows)
		v.paintNotice(screen)
	}

	// AND THE POINTER OVER ALL OF IT, for a viewer composing its own frame. A
	// viewer inside an App session leaves this to the App, which composes the
	// same three pictures after the in-game menu as well (App.Draw,
	// DeferPointer).
	if !v.pointerDeferred {
		v.drawPointer(screen, 1, 0, 0)
		v.canvasLog.unknown(screen.Bounds())
	}
}

// blitPointerLayer places one of the mission pointer's three pictures.
//
// It is a package variable for blitColumnLayer's own reason one layer further
// out: ebitengine refuses a pixel readback with no graphics context, so the
// only way a test without a window can see WHEN a pointer was composed against
// the boxes and the menu around it is to record the call. That is exactly what
// this story's return had to witness — the owner reported the cursor drawn
// under the panels and under the pause menu, and no assertion in this package
// could see either. Production never replaces it.
//
// THE DEFAULT DRAWS THROUGH drawFinalFrame: the deferred pointer (App.Draw's
// map branch) is scaled by the same v.place.Scale() as the mission frame it
// stands over, so it shares that frame's scaler. The frame's own internal
// pointer pass (drawFrame, scale 1) takes the single FilterNearest draw.
var blitPointerLayer = func(dst, img *ebiten.Image, op *ebiten.DrawImageOptions, layer string) {
	drawFinalFrame(dst, img, op, &pointerSharpBuf, frameSmoothingOff)
}

func (v *Viewer) drawPointer(dst *ebiten.Image, scale, originX, originY float64) {
	frameSmoothingOff = v.frameSmoothingOff
	place := func(at image.Point) (float64, float64) {
		return originX + float64(at.X)*scale, originY + float64(at.Y)*scale
	}
	blit := func(img *ebiten.Image, at image.Point, layer string) {
		var op ebiten.DrawImageOptions
		op.Filter = ebiten.FilterNearest
		op.GeoM.Scale(scale, scale)
		x, y := place(at)
		op.GeoM.Translate(x, y)
		blitPointerLayer(dst, img, &op, layer)
	}

	if pic, at, ok := v.attackPointerPresent(); ok {
		if pic == nil {
			// The authored cross, drawn where no art was supplied. It is
			// centred on the pressed pixel, where a picture hangs from it.
			x64, y64 := place(at)
			x, y := float32(x64), float32(y64)
			arm := float32(AttackAuthoredArm) * float32(scale)
			w := float32(AttackAuthoredWidth) * float32(scale)
			vector.StrokeLine(dst, x-arm, y-arm, x+arm, y+arm, w, AttackMarkerColor, false)
			vector.StrokeLine(dst, x-arm, y+arm, x+arm, y-arm, w, AttackMarkerColor, false)
		} else {
			// EVERY FRAME THE MANAGER ADVANCES TO REACHES THE SCREEN
			// (cursortexture.go). The upload used to be gated on a flag set
			// once per viewer at map open, so the ten-frame attack sheet drew
			// frame 0 for the whole session.
			blit(v.attackPointerTex.upload(pic), at, "attackPointer")
		}
	}

	if pic, at, ok := v.mapCursorPresent(); ok {
		blit(v.mapCursorTex.upload(pic), at, "mapCursor")
	}

	// THE CARRIED ITEM, LAST OF THE THREE (1005, "the interactive doll"): a
	// drag in progress is the one thing on this screen the player is actively
	// moving. It is the one of the three that already stood over the boxes.
	if pic, at, ok := v.dragItemPresent(); ok {
		b := pic.Bounds()
		if v.dragImg == nil || v.dragImg.Bounds().Dx() != b.Dx() || v.dragImg.Bounds().Dy() != b.Dy() {
			v.dragImg = ebiten.NewImage(b.Dx(), b.Dy())
		}
		v.dragImg.WritePixels(pic.Pix)
		blit(v.dragImg, at, "heldItem")
	}
}

// DeferPointer tells this viewer that its caller composes the mission pointer
// itself, after everything the caller draws over the frame this viewer returns.
//
// AN App SETS IT AND NOTHING ELSE DOES (flow.go, beside SetCursorManager). A
// standalone viewer (cmd/mapview) has nothing composed over it and draws its
// own.
func (v *Viewer) DeferPointer(deferred bool) { v.pointerDeferred = deferred }

func spriteGeoM(s staticScreenRect, zoom float64) ebiten.GeoM {
	var g ebiten.GeoM
	if s.Mirror {
		g.Scale(-zoom, zoom)
		g.Translate(s.X+s.W, s.Y)
		return g
	}
	g.Scale(zoom, zoom)
	g.Translate(s.X, s.Y)
	return g
}

// triangleTarget is the one method drawFlat and drawDisplaced need out of a
// draw target: *ebiten.Image already satisfies it, so Draw passes screen
// through unchanged and every existing call to Draw(screen *ebiten.Image)
// keeps compiling. Taking this interface, and not *ebiten.Image itself, is
// what makes the two loops' draw calls observable: a test can hand them a
// plain Go struct that records its arguments instead, where reading pixels
// back out of a real image is unavailable before the game starts (SC-8).
//
// Measured, not supposed: with drawFlat and drawDisplaced fixed at
// *ebiten.Image, T3's whole test suite passed even with withScales deleted
// from both loops — every assertion recomposed tileVertices/flatTileVertices
// and cornerScales inside the test body rather than asking the loop what it
// actually submitted, so the story's entire visible effect was reachable but
// unverified (0014 SC-13).
type triangleTarget interface {
	DrawTriangles(vertices []ebiten.Vertex, indices []uint16, img *ebiten.Image, options *ebiten.DrawTrianglesOptions)
}

// drawFlat paints every visible cell as the quad its four un-displaced
// lattice corners span, shaded by that cell's own corner scales.
func (v *Viewer) drawFlat(target triangleTarget) {
	var op ebiten.DrawTrianglesOptions
	op.Filter = ebiten.FilterNearest

	v.forEachDrawnTile(func(col, row int) {
		img := v.cellImage(v.tileWord(col, row), col, row)
		verts := terrainTextureVertices(withScales(flatTileVertices(v.cam, col, row), v.cornerScales(col, row)))
		target.DrawTriangles(verts[:], quadIndices[:], img, &op)
	})
}

// drawDisplaced paints the projected quads with the same cell textures as
// flat mode. Each source UV is offset into the cached image's replicated edge;
// destination positions and corner lighting remain unchanged.
func (v *Viewer) drawDisplaced(target triangleTarget) {
	var op ebiten.DrawTrianglesOptions
	op.Filter = ebiten.FilterNearest

	v.forEachDrawnTile(func(tx, ty int) {
		img := v.cellImage(v.tileWord(tx, ty), tx, ty)
		verts := terrainTextureVertices(withScales(tileVertices(v.cam, v.proj, tx, ty), v.cornerScales(tx, ty)))
		target.DrawTriangles(verts[:], quadIndices[:], img, &op)
	})
}

const terrainTextureBorder = 1

func terrainTextureVertices(verts [4]ebiten.Vertex) [4]ebiten.Vertex {
	for i := range verts {
		verts[i].SrcX += terrainTextureBorder
		verts[i].SrcY += terrainTextureBorder
	}
	return verts
}

// TERR-217.
func (v *Viewer) forEachDrawnTile(fn func(tx, ty int)) {
	if _, bounded := v.playableCellRect(); bounded {
		origin := 0
		if v.Mode() == ModeDisplaced {
			origin = v.proj.MinV
		}
		col := int(math.Floor(v.cam.X / camera.CellSize))
		row := int(math.Floor((v.cam.Y + float64(origin)) / camera.CellSize))
		cols := int(float64(v.cam.ViewW) / (v.cam.Zoom * camera.CellSize))
		rows := int(float64(v.cam.ViewH) / (v.cam.Zoom * camera.CellSize))
		for y := max(0, row); y < min(v.grid.Height, row+rows+lowerRenderLipRows); y++ {
			for x := min(v.grid.Width, col+cols) - 1; x >= max(0, col); x-- {
				fn(x, y)
			}
		}
		return
	}
	if v.Mode() == ModeDisplaced {
		v.forEachDisplacedTile(fn)
		return
	}
	r := v.cam.VisibleTiles()
	for ty := r.Row0; ty < r.Row1; ty++ {
		for tx := r.Col0; tx < r.Col1; tx++ {
			fn(tx, ty)
		}
	}
}

// forEachDisplacedTile calls fn once per tile the displaced pass draws, in the
// order it draws them: ty ascending, then tx. That order is what resolves a
// cliff, where quads genuinely overlap — a later tile owns the shared pixel —
// so it is a property of this loop and not an accident of iteration (SC-9).
//
// The ROWS come from the projection's RowRange over the view window in world
// Y, and NOT from a pad on the camera's own row band: the camera's band is a
// world coordinate while the altitudes that would pad it are native, and the
// two are offset by MinV — up to eight cell rows. The COLUMNS come from
// VisibleTiles unmodified, which is exact rather than padded because no
// altitude term reaches a destination column (TERR-GEOM-035).
//
// Both are already clipped to the map, so every index fn forms is inside the
// tile slice.
func (v *Viewer) forEachDisplacedTile(fn func(tx, ty int)) {
	cols := v.cam.VisibleTiles()

	_, top := v.cam.ScreenToWorld(0, 0)
	_, bottom := v.cam.ScreenToWorld(float64(v.cam.ViewW), float64(v.cam.ViewH))
	r0, r1 := v.proj.RowRange(top, bottom)

	for ty := r0; ty < r1; ty++ {
		for tx := cols.Col0; tx < cols.Col1; tx++ {
			fn(tx, ty)
		}
	}
}

// quadIndices splits a tile's four corners into two triangles along the TL-BR
// diagonal, which the spec fixes rather than leaving to the drawing call: which
// diagonal is used changes the interior of every non-parallelogram tile.
//
// {0,1,3} is TL,TR,BR and {0,3,2} is TL,BR,BL, so the edge the two share is
// 0-3 — TL to BR. The plausible alternative, {0,1,2, 1,3,2}, puts the identical
// four corners on screen and shares 1-2, the OTHER diagonal; it is wrong, and
// wrong only in the interior.
var quadIndices = [6]uint16{0, 1, 3, 0, 3, 2}

// withScales returns verts with each corner's colour multiplied by its own
// scale in sc — TL, TR, BL, BR, the order cornerScales and both vertex
// builders share. It is pure: a Go value in, a Go value out, no window.
func withScales(verts [4]ebiten.Vertex, sc [4]float32) [4]ebiten.Vertex {
	out := verts
	for i := range out {
		out[i].ColorR *= sc[i]
		out[i].ColorG *= sc[i]
		out[i].ColorB *= sc[i]
	}
	return out
}

// tileVertices is tile (tx,ty)'s four screen-space corners, in the order
// quadIndices assumes: TL, TR, BL, BR — mesh vertices (tx,ty), (tx+1,ty),
// (tx,ty+1) and (tx+1,ty+1).
//
// It is pure and needs no graphics context: an ebiten.Vertex is a struct
// literal, so the whole of the displaced geometry is reachable in a test and
// only the DrawTriangles call itself needs a window.
//
//   - Dst is WorldToScreen of WorldCorner, submitted UNROUNDED; the only rounding
//     is the float32 the vertex takes, which the overlay rects already accept.
//     Every vertex comes from the shared projection, so there is no second height
//     convention here to disagree with it.
//   - Src spans 0..CellSize. Only the terrain draw offsets it by one pixel
//     into a padded texture; shroud and editor paths use the raw geometry.
//   - The colour is 1,1,1,1 and deliberately NOT the zero value: ebiten
//     documents ColorA == 0 as fully transparent, which would draw the entire
//     terrain invisible with every geometry test still passing.
func tileVertices(cam *camera.Camera, proj *terrain.Projection, tx, ty int) [4]ebiten.Vertex {
	var out [4]ebiten.Vertex
	i := 0
	for _, d := range [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		wx, wy := proj.WorldCorner(tx+d[0], ty+d[1])
		sx, sy := cam.WorldToScreen(float64(wx), float64(wy))
		out[i] = ebiten.Vertex{
			DstX:   float32(sx),
			DstY:   float32(sy),
			SrcX:   float32(d[0] * terrain.CellSize),
			SrcY:   float32(d[1] * terrain.CellSize),
			ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
		}
		i++
	}
	return out
}

// flatTileVertices is flat mode's sibling to tileVertices: tile (tx,ty)'s
// four screen-space corners over the UN-displaced cell lattice, in the same
// TL, TR, BL, BR order quadIndices assumes — mesh vertices (tx,ty),
// (tx+1,ty), (tx,ty+1) and (tx+1,ty+1), each corner simply *CellSize with no
// altitude term, each through cam.WorldToScreen. Its TL corner is exactly
// what flatTileScreen(tx,ty) already computes; the other three carry the
// same transform to the far edges of the same cell, which flat mode never
// needed before it drew a quad instead of a translated square.
//
// It is pure and needs no graphics context, mirroring tileVertices: the
// source span is the sub-cell's own bounds and the colour is 1,1,1,1,
// deliberately not the zero value, for the same reason tileVertices' own doc
// gives.
func flatTileVertices(cam *camera.Camera, tx, ty int) [4]ebiten.Vertex {
	var out [4]ebiten.Vertex
	i := 0
	for _, d := range [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		wx := float64((tx + d[0]) * terrain.CellSize)
		wy := float64((ty + d[1]) * terrain.CellSize)
		sx, sy := cam.WorldToScreen(wx, wy)
		out[i] = ebiten.Vertex{
			DstX:   float32(sx),
			DstY:   float32(sy),
			SrcX:   float32(d[0] * terrain.CellSize),
			SrcY:   float32(d[1] * terrain.CellSize),
			ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
		}
		i++
	}
	return out
}

// cornerScales combines terrain and spell lighting. Fog is applied after all
// world art by drawShroud, so terrain and protruding structures share one mask.
func (v *Viewer) cornerScales(tx, ty int) [4]float32 {
	sc := v.cornerShading(tx, ty)
	vertices := [4]image.Point{
		image.Pt(tx, ty), image.Pt(tx+1, ty),
		image.Pt(tx, ty+1), image.Pt(tx+1, ty+1),
	}
	for i := range sc {
		if brightness, ok := v.spellTerrainScale(vertices[i]); ok {
			sc[i] = brightness
		}
		// REPLACE FOR THE SPELL PLANE, MAX FOR THE STRUCTURE MASK. Darkness
		// writes 0.5 and must be able to pull a vertex below its own shading;
		// a structure light must never pull one below anything, and its plane
		// is built over a flat-ground reference that relief can exceed
		// (structureTerrainScale's own doc, overlay.go).
		if glow, ok := v.structureTerrainScale(vertices[i]); ok && glow > sc[i] {
			sc[i] = glow
		}
	}
	return sc
}

// cornerShading is cornerScales' own body before the fog factor is composed
// onto it — see cornerScales' own doc above this one for the three branches
// and the reasoning behind each.
func (v *Viewer) cornerShading(tx, ty int) [4]float32 {
	unlit := [4]float32{1, 1, 1, 1}
	if !v.Lit() {
		return v.dimBorder(unlit, tx, ty)
	}
	ref := v.resolveCell(v.tileWord(tx, ty), tx, ty)
	if v.set.Slot(ref.Slot).SubCell(ref.Sub) == nil {
		return unlit
	}
	lv := terrain.CornerLevels(v.levels, v.grid.Width, v.grid.Height, tx, ty)
	return v.dimBorder([4]float32{
		terrain.ShadeScale(int(lv[0])),
		terrain.ShadeScale(int(lv[1])),
		terrain.ShadeScale(int(lv[2])),
		terrain.ShadeScale(int(lv[3])),
	}, tx, ty)
}

// mapBorderDim is the fraction of its brightness a cell in the engine margin
// keeps: ZERO, so the margin is drawn BLACK.
//
// The value is the owner's, twice. 0038 authored 0.5 with a disclosed rationale
// — nothing about the margin's pixels was decoded, so the arbiter was a
// developer looking at a window — and the owner then looked at the original and
// ruled the edge black («в оригинале в принципе край карты черный»). That ruling
// is the owner AS AUTHOR, so it is a fact and not testimony to re-derive.
//
// What 0038 got wrong is not the number, it is that it wrote the number's bound
// into a test. The dim MULTIPLIES the corner lighting so the margin keeps its
// relief, and a black edge has no relief to keep; so 0 does not break the
// composition, it collapses it on purpose. The multiply is kept rather than
// replaced by a branch because it is still the one line that decides this, and
// every reader of a corner scale is unchanged: 0 is a scale like any other.
const mapBorderDim float32 = 0

// dimBorder leaves the admitted lower terrain rows textured without changing movement.
// TERR-217.
func (v *Viewer) dimBorder(sc [4]float32, tx, ty int) [4]float32 {
	if !v.grid.BorderCell(tx, ty) {
		return sc
	}
	if cells, ok := v.renderCellRect(); ok && image.Pt(tx, ty).In(cells) {
		return sc
	}
	for i := range sc {
		sc[i] *= mapBorderDim
	}
	return sc
}

// tileWord is the tile word at (col,row), read row-major.
func (v *Viewer) tileWord(col, row int) uint16 {
	return v.grid.Tiles[row*v.grid.Width+col]
}

// flatTileScreen is where FLAT mode places tile (col,row): the corner of the
// un-displaced cell lattice, (col*CellSize, row*CellSize), through the camera.
// It carries no altitude term at all. Displaced mode places its corners through
// tileVertices instead.
func (v *Viewer) flatTileScreen(col, row int) (sx, sy float64) {
	return v.cam.WorldToScreen(float64(col*terrain.CellSize), float64(row*terrain.CellSize))
}

// cellImage resolves a tile word at its world position to a GPU image,
// uploading and caching on first use. A cell whose slot is absent or too
// short yields the placeholder fill, so drawing never panics on an
// off-corpus word.
//
// The animation phase is folded into the slot, so the existing (slot,
// sub-cell) half of the cache key already distinguishes phases and needs no
// extra field; the tint half is new and is decided BEFORE the lookup,
// exactly as cornerScales decides its own placeholder case before it ever
// reads a level: a placeholder cell's key carries a zero tint whatever
// v.lightTint() answers, because the fill it maps to is never tinted and
// must not be split across bands it draws identically under.
func (v *Viewer) cellImage(word uint16, col, row int) *ebiten.Image {
	src, dirt, key := v.cellTexture(word, col, row)
	if img, ok := v.cache[key]; ok {
		return img
	}

	if src == nil {
		img := v.placeholderImage()
		v.cache[key] = img
		return img
	}

	img := ebiten.NewImageFromImage(paddedCellPixels(scorchedCellPixels(src, dirt, key.tint)))
	v.cache[key] = img
	return img
}

func paddedCellPixels(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx()+2*terrainTextureBorder, b.Dy()+2*terrainTextureBorder))
	for y := 0; y < dst.Rect.Dy(); y++ {
		sy := b.Min.Y + max(0, min(y-terrainTextureBorder, b.Dy()-1))
		for x := 0; x < dst.Rect.Dx(); x++ {
			sx := b.Min.X + max(0, min(x-terrainTextureBorder, b.Dx()-1))
			dst.SetRGBA(x, y, src.RGBAAt(sx, sy))
		}
	}
	return dst
}

// cellPixels is the CPU image a cell's texture is built from: src's own
// palette-resolved pixels, each channel carrying the tint added through
// terrain.TintChannel — the shading transform's own clamp(chan+tint, 0,
// 255), expressed at its identity level rather than as arithmetic of our
// own. Alpha is copied through unchanged; the tint darkens no channel and
// touches none.
//
// IT IS PURE AND UPLOADS NOTHING, for the same reason spritePixels is split
// from staticImage: an *ebiten.Image cannot be read back before the game
// starts, so with the tint applied inside the upload a test could assert only
// that drawing did not panic. Building the tinted RGBA here and uploading THAT
// is what keeps cellImage's own change to one call, and what makes this
// function reachable with no graphics context at all.
//
// Under a zero tint every channel is TintChannel(ch, 0) == ch — ShadeChannel
// at its identity level is the identity — so the result is pixel-identical to
// a straight per-pixel read of src, which is what cellImage uploaded before
// this story (AC-9).
func cellPixels(src *image.Paletted, tint [3]uint8) *image.RGBA {
	return scorchedCellPixels(src, nil, tint)
}

func scorchedCellPixels(src, dirt *image.Paletted, tint [3]uint8) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := paletteRGBA(src, b.Min.X+x, b.Min.Y+y)
			if dirt != nil {
				dx, dy := dirt.Rect.Min.X+x, dirt.Rect.Min.Y+y
				if dirt.ColorIndexAt(dx, dy) != terrain.DirtTransparentIndex {
					c = paletteRGBA(dirt, dx, dy)
				}
			}
			dst.SetRGBA(x, y, color.RGBA{
				R: terrain.TintChannel(c.R, tint[0]),
				G: terrain.TintChannel(c.G, tint[1]),
				B: terrain.TintChannel(c.B, tint[2]),
				A: c.A,
			})
		}
	}
	return dst
}

// paletteRGBA resolves one paletted pixel to color.RGBA. image.Paletted.At
// already returns the palette entry unconverted, so this only widens its type:
// the common case is a direct assertion, and the fallback goes through the
// standard colour-model conversion for a palette built from another concrete
// colour type. It is the one place cellPixels reads a palette, mirroring the
// render tier's own paletteColor (pkg/render/terrain/composite.go) rather than
// calling it — that function is unexported, and a one-line index lookup has no
// second way to disagree with itself the way a blit's transparency handling
// could (statics.go's own file comment).
func paletteRGBA(img *image.Paletted, x, y int) color.RGBA {
	if c, ok := img.At(x, y).(color.RGBA); ok {
		return c
	}
	r, g, b, a := img.At(x, y).RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

// resolveCell picks the mapping for a cell: the animated one at the current tick
// while animating, the static one otherwise. Split out of cellImage so the
// choice is testable without an engine context.
func (v *Viewer) resolveCell(word uint16, col, row int) terrain.TileRef {
	if v.animate && v.fogAt(col, row) == FogVisible {
		return terrain.ResolveAnimated(word, col, row, v.anim.Count())
	}
	return terrain.Resolve(word)
}

// placeholderImage lazily builds the solid fill used for absent slots.
func (v *Viewer) placeholderImage() *ebiten.Image {
	if v.placeholder == nil {
		v.placeholder = ebiten.NewImage(terrain.CellSize+2*terrainTextureBorder, terrain.CellSize+2*terrainTextureBorder)
		v.placeholder.Fill(terrain.PlaceholderColor)
	}
	return v.placeholder
}

// Run opens the window and blocks until the viewer exits. Esc closes it, which
// Ebitengine reports as termination rather than an error.
func (v *Viewer) Run() error {
	ebiten.SetWindowTitle(v.title)
	ebiten.SetWindowSize(DefaultWindowW, DefaultWindowH)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(v); err != nil && err != ebiten.Termination {
		return err
	}
	return nil
}

// SubCellBounds reports the pixel bounds a terrain sub-cell must have. It exists
// so tests can assert the draw path's expectations without an engine context.
func SubCellBounds() image.Rectangle {
	return image.Rect(0, 0, terrain.CellSize, terrain.CellSize)
}

func (v *Viewer) paintNotice(screen *ebiten.Image) {
	noticeCapture := beginTextCapture(v.textSmoothingEnabled)
	pic, at, scale, ok := v.noticePresent()
	endTextCapture(noticeCapture, at)
	if !ok {
		return
	}
	l := v.noticeLayout()
	if l.Style == NoticeStyleDialogue {
		body := image.Rectangle{Max: l.Box.Size().Sub(image.Pt(noticeShadow, noticeShadow))}
		v.dialogueBackdrop.drawFrameShadows(screen, &v.canvasLog, l.Frame, body, at, scale, text.Captured()[:max(0, noticeCapture)])
	}
	if v.noticeFresh {
		b := pic.Bounds()
		if v.noticeImg == nil || v.noticeImg.Bounds().Dx() != b.Dx() || v.noticeImg.Bounds().Dy() != b.Dy() {
			v.noticeImg = ebiten.NewImage(b.Dx(), b.Dy())
		}
		v.noticeImg.WritePixels(pic.Pix)
		v.noticeFresh = false
	}
	var op ebiten.DrawImageOptions
	op.Filter = ebiten.FilterNearest
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(at.X), float64(at.Y))
	clip := screen.Bounds()
	if l.Style == NoticeStyleDialogue {
		clip = dialoguePolicyClip(v.dialogueBackdrop.policy, clip)
	}
	if clip.Empty() {
		return
	}
	screen.SubImage(clip).(*ebiten.Image).DrawImage(v.noticeImg, &op)
	if scale == 1 {
		v.canvasLog.overClipped(pic, at, clip)
	} else {
		size := pic.Rect.Size()
		v.canvasLog.unknown(image.Rect(at.X, at.Y, at.X+int(math.Ceil(float64(size.X)*scale)), at.Y+int(math.Ceil(float64(size.Y)*scale))).Inset(-1))
	}
}
