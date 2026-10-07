package ui

import (
	"image"
	"image/color"
	"image/draw"
	"slices"
	"strconv"

	"againrom/pkg/render/text"
)

// TownSurfaceKind selects only the geometry owned by a town room. The common
// shell still owns the frame, modal order and lower character region.
type TownSurfaceKind uint8

const (
	TownSurfaceTavern TownSurfaceKind = iota
	TownSurfaceSchool
)

var (
	TownContentRegion   = image.Rect(0, 0, 480, 480)
	TownUpperRegion     = image.Rect(480, 0, 640, 238)
	TownWideUpperRegion = image.Rect(464, 0, 640, 238)
	TownCharacterRegion = image.Rect(480, 238, 640, 480)

	// townUpperSeamRegion is the 16 columns TownWideUpperRegion carries beyond
	// TownUpperRegion's own left edge, x:[464,480) — the strip a room's
	// buttonsarea.bmp does not cover (DIV-166, DIV-168).
	townUpperSeamRegion = image.Rect(TownWideUpperRegion.Min.X, TownWideUpperRegion.Min.Y, TownUpperRegion.Min.X, TownUpperRegion.Max.Y)

	// townCharacterSeamRegion is townUpperSeamRegion's own 16-column strip,
	// x:[464,480), continued down TownCharacterRegion's own height,
	// y:[238,480) (1021 spec B1/B3; formerly chargen_page.go's own
	// chargenLowerSeamRegion, shared here so every screen composing
	// TownCharacterRegion through a TownPane draws the same seam rectangle
	// rather than each computing it separately).
	townCharacterSeamRegion = image.Rect(TownWideUpperRegion.Min.X, TownUpperRegion.Max.Y, TownUpperRegion.Min.X, TownCharacterRegion.Max.Y)

	// townCharacterFigure IS THE PANE'S OWN GEOMETRY AT 640x480 AND IS NO
	// LONGER A LITERAL. It was written out as four numbers while the town
	// screen was the only screen composing this pane; the mission screen now
	// composes the same pane at characterPanelBoxRect, which is this rectangle
	// at 640x480 and is NOT at 800x600 or 1024x768. characterpane.go carries
	// the panel-relative expression so the two screens cannot drift:
	// (480,240,640,480).
	townCharacterFigure = characterPaneFigureRect(TownCharacterRegion)

	// townCharacterPrev and townCharacterNext ARE DECODED, NOT AUTHORED, and
	// each one's four numbers are a claim's own. SHOP-PICKER-043 gives the
	// party picker's two rects panel-relative as (1,205,33,237) and
	// (119,205,151,237), which at the shipped 640x480 are these. TOWN-346
	// reproduces both by a different tool, as rects D and E of the character
	// figure widget's six.
	//
	// townCharacterMode IS GONE. DIV-307.
	townCharacterPrev = CharacterPaneCornerRect(TownCharacterRegion, CharacterPanePrev)
	townCharacterNext = CharacterPaneCornerRect(TownCharacterRegion, CharacterPaneNext)
	townCharacterMode = CharacterPaneCornerRect(TownCharacterRegion, CharacterPaneMode)
)

// TownSurfaceCell is one screen-owned choice in the shared left region.
type TownSurfaceCell struct {
	Key           string
	Label, Detail string
	Price         string
	Semantic      string
	Enabled       bool
	Selected      bool
	Hired         bool
	Portrait      bool
	Picture       image.Image
	Frames        []image.Image
	TalkOnly      bool
}

// TownSurfaceButton is one screen-owned command in the upper extension.
type TownSurfaceButton struct {
	Label, Value string
	Enabled      bool
}

// TownCharacterView is the one lower-region character projection shared by
// tavern, school and shop. Subject and Figure are already resolved on the game
// side; this package derives neither stats nor equipment art.
type TownCharacterView struct {
	Subject         PanelSubject
	HasSubject      bool
	Figure          *image.RGBA
	Member          int
	MemberCount     int
	Statistics      bool
	SelectionStatus [3]string
	Font            *text.Font

	// CardFont IS THE STATISTICS CARD'S OWN FONT, SEPARATE FROM THE CHROME'S
	// (1027 B1). Font is the town shell's own font1 and stays that for the
	// chevrons and the mode box. The card
	// is sixteen two-column rows inside 160x242, and at font1 the left column
	// alone reaches 120 of the 142 usable pixels, which leaves a 16-pixel
	// right budget that panelFitValue empties every right-hand value into
	// and clips every right-hand label at the card's own edge (measured, both
	// roots). font2 is the atlas the same card already composes at on the
	// character generator, where it fits. A nil CardFont falls back to Font,
	// so a caller predating this story composes exactly the card it did.
	CardFont *text.Font

	// CardLayout is the statistics card's own layout, or nil for
	// CompactPanelLayout. The town screens set none and compose the shipped
	// card.
	//
	// A layout with no Background of its own takes the pane's own body, which
	// is what CompactPanelLayout is handed here.
	CardLayout *PanelLayout

	FigurePane TownPane
	StatsPane  TownPane

	// Session is `campaign+0x3dc`, the bitmask `TOWN-346` gates all six corner
	// rectangles on. The ZERO VALUE IS NOT A SHIPPED VALUE: the paint routine's
	// own first act refuses a mask with no bit of `0x627` set, so a caller
	// predating this story would draw no corner at all. Every caller in this
	// build names one — CharacterPaneShop on the town side,
	// CharacterPaneMission during a mission — and paneSession below reads a
	// zero as CharacterPaneShop so that a hand-built fixture keeps the party
	// picker it had before.
	Session CharacterPaneSession

	ScreenH int

	// ModeFlag IS TOWN-346's OWN `flag`, PRECOMPUTED, not derived here from
	// PaneRect and ScreenH. The mission composes this pane on a canvas of its
	// own (PaneRect is (0,0)-(160,242), canvas-relative) and hit-tests it in
	// window pixels (the pane's real screen rectangle) — two different
	// `pane.Max.Y` values, and `characterPaneModeFlag` moves its own
	// threshold with the one it is handed (that function's own doc, round-2
	// adversarial review). Every town and shop screen composes this pane
	// directly onto the full-screen canvas, so PaneRect is already the
	// screen rectangle there and ModeFlag is correctly its own zero value,
	// since CharacterPaneShop is never CharacterPaneMission and the flag is
	// false regardless. The mission is the one caller that must set this
	// explicitly, from `(*Viewer).characterPaneModeFlag`, which resolves the
	// pane's real screen rectangle before computing it.
	ModeFlag bool

	// PaneRect is where this pane stands. The zero rectangle means
	// TownCharacterRegion, which is where every town-side screen puts it and
	// where the mission puts it at 640x480; the mission passes its own
	// characterPanelBoxRect so the pane follows the right column at 800x600
	// and 1024x768.
	PaneRect image.Rectangle

	// CardOffset moves the card canvas right of PaneRect's origin; zero except
	// in the tavern.
	CardOffset image.Point

	// PackOpen and BookOpen are the two open flags rect A's and rect B's art
	// is selected by: `FindChildById(mapview, 2) != 0` and
	// `FindChildById(host, 3) != 0` (`TOWN-354`). This build reads them from
	// the two display switches the same two corners flip (hudtoggles.go's
	// hudPanelPack and hudPanelBook), which is what makes the drawn bitmap
	// describe the panel actually on screen.
	PackOpen bool
	BookOpen bool

	// NoBook is a room with no spellbook: the book corner is neither drawn nor
	// hit.
	NoBook bool

	// CornerArt is the nine shipped corner bitmaps, nil for a caller that
	// resolved none. A nil pane draws the corners' outlines and no picture,
	// on DrawTownCharacterRegion's own fallback rule for the body.
	CornerArt *CharacterPaneCornerArt
}

// cardRect is where the statistics card's canvas stands: PaneRect moved by
// CardOffset and as wide as the card layout.
func (v TownCharacterView) cardRect() image.Rectangle {
	r := v.paneRect()
	if v.CardOffset == (image.Point{}) {
		return r
	}
	return image.Rectangle{Min: r.Min.Add(v.CardOffset), Max: r.Max.Add(v.CardOffset)}
}

func (v TownCharacterView) paneRect() image.Rectangle {
	if v.PaneRect.Empty() {
		return TownCharacterRegion
	}
	return v.PaneRect
}

// paneSession is this view's own `campaign+0x3dc`, reading a zero as the shop
// bit — this file's own Session doc says why a zero is not a shipped value.
func (v TownCharacterView) paneSession() CharacterPaneSession {
	if v.Session == 0 {
		return CharacterPaneShop
	}
	return v.Session
}

// CharacterPaneCornerAt is which of the six corners a click at p reaches on
// this view, in the original's own test order and with every gate applied.
// Exported because the shop, the town shell and the mission viewer are three
// packages' worth of callers of one hit test.
func CharacterPaneCornerAt(v TownCharacterView, p image.Point) []CharacterPaneCorner {
	corners := CharacterPaneCornersAt(v.paneRect(), v.paneSession(), v.ScreenH, p)
	if v.NoBook {
		corners = slices.DeleteFunc(corners, func(c CharacterPaneCorner) bool { return c == CharacterPaneBook })
	}
	return corners
}

// paneModeFlag is this view's own ModeFlag, computed by its builder rather
// than here — see the field's own doc.
func (v TownCharacterView) paneModeFlag() bool { return v.ModeFlag }

// TownSchoolArt is the immutable school presentation resolved at startup. The
// class axis is fighter, mage; the skill axis is the shared slot order used by
// detailed character generation; the state axis is on, shine, shine-on.
//
// Faces is each class's own rest face of the training column, drawn over
// Background at SchoolFaceOrigin. Background bakes one of the two faces into
// itself; which one is a property of the install and not of the shown member,
// so the room draws the class's frame itself rather than inheriting whatever is
// baked. Each rest face already carries that class's five unselected
// engravings, so no picture is drawn for a skill in its rest state.
type TownSchoolArt struct {
	Background image.Image
	Upper      image.Image
	Faces      [2]image.Image
	// Column is TOWN-146's sixteen opaque frames, rt0000..rt0015.
	// Faces aliases its endpoints for static callers and provenance tools.
	Column [16]image.Image
	Masks  [2]*image.Paletted
	Skills [2][5][3]image.Image
	// Diamond is TOWN-154's nine opaque 80x76 frames, on0000..on0008.
	Diamond [9]image.Image

	// Training is the four independently complete movies.res families.
	// Class index zero is fighter and one is mage, matching Masks, Skills and
	// Faces. A nil family means that family was absent or incomplete; no
	// partial sequence crosses into the controller (TOWN-427).
	Training [2]SchoolTrainingArt

	// Buttons is the school's two shipped buttons (Train, Exit), each an
	// off/on pair: Buttons[i][0] is off, Buttons[i][1] is on (1017).
	Buttons [2][2]image.Image

	// UpperSeam is interface/inn/ruover.bmp, 16x238, the shipped strip that
	// closes TownWideUpperRegion's own left 16 columns beside Upper. It is
	// stored under the inn directory but its right edge continues Upper's own
	// left edge with no visible join, for all three rooms this build draws it
	// in (DIV-166, DIV-168); it is not school-specific art.
	UpperSeam image.Image
}

// SchoolTrainingArt is one side of the school room's immutable movie cache.
// Transition is tr0000... and Idle is m0001.... The game owns every clock,
// flag, index and cached-picture rule; this type owns only decoded pictures.
type SchoolTrainingArt struct {
	Transition []image.Image
	Idle       []image.Image
}

// SchoolTrainingFrame is the controller's read-only choice for one paint.
// Images rather than indices cross the package seam, so the compositor cannot
// become an animation-state authority. Nil leaves that side's background
// untouched.
type SchoolTrainingFrame struct {
	Mage    image.Image
	Fighter image.Image
}

// TavernSheetLimit bounds the Unit<n> inn sheet numbers: TOWN-470's census
// tops out at Unit64.
const TavernSheetLimit = 65

// TownTavernArt is the tavern's three-child static furniture, the two shipped
// 48x64 roster-card grounds and every frame of each type-keyed roster sprite.
type TownTavernArt struct {
	LeftPicture image.Image
	LeftStats   image.Image
	Center      image.Image
	ManBack     image.Image
	ManBackTalk image.Image
	Units       [TavernSheetLimit]image.Image
	UnitFrames  [TavernSheetLimit][]image.Image
	// HeroFrames are the two shared talk-cell sheets a Hero object draws
	// (TAVERN-TALKPIC-016): index 0 HeroFighter, index 1 HeroMage.
	HeroFrames [2][]image.Image

	LeftStatsSeam   image.Image
	LeftPictureSeam image.Image

	// Upper is the tavern's own 160x238 button-area picture, drawn at
	// TownUpperRegion. It and Buttons remain the measured original three-
	// command presentation used by the provenance tooling.
	Upper image.Image
	// Buttons is the tavern's three shipped button plaques, each an off/on
	// pair.
	Buttons [3][2]image.Image

	// CommandUpper and CommandButtons are the shop's complete four-command
	// panel: the 176x238 menu body and its four native plaques. The owner-
	// directed tavern layout uses this set whole, so the narrower 120x52 top
	// and bottom plaques leave the panel ornaments visible (DIV-483).
	CommandUpper   image.Image
	CommandButtons [4]image.Image

	// UpperSeam is interface/inn/ruover.bmp, the same shipped strip
	// TownSchoolArt.UpperSeam carries, read from the tavern's own directory.
	UpperSeam image.Image

	// Interior is the four independently loaded centre-child animation
	// families. A nil family leaves only that motion absent; Center and every
	// control remain usable. The game owns the indices and clocks.
	Interior TavernInteriorArt
}

// TavernInteriorArt is immutable install art. Loaded counts and loop counts
// are deliberately not represented by one field: the game applies the
// accepted per-family bounds when it selects a picture.
type TavernInteriorArt struct {
	Candle   []image.Image
	Cauldron []image.Image
	Breath   []image.Image
	Drink    []image.Image
}

// TavernInteriorFrame is one paint-owned selection from TavernInteriorArt.
// Images, rather than indices, cross the package seam so the compositor does
// not become the authority for the controller's separate index/cache rules.
type TavernInteriorFrame struct {
	Candle   image.Image
	Cauldron image.Image
	Tender   image.Image
}

// TownSurfaceView is a complete tavern or school frame before a modal dialogue
// is layered over it.
type TownSurfaceView struct {
	SuppressHover bool // App paints the shared delayed overlay after widening.
	Kind          TownSurfaceKind
	Title         string
	Cells         []TownSurfaceCell
	Buttons       []TownSurfaceButton
	// Press is the control currently pressed AND under the cursor; it draws
	// the button's own on-bitmap instead of its off-bitmap (1017).
	Press   TownSurfaceControl
	Hero    TownCharacterView
	Message string
	Font    *text.Font
	// CardFont is the compact 48x64 tavern card's text font. The room chrome
	// keeps Font; the card uses the already-loaded smaller production font so
	// the complete whole-squad price remains inside its shipped bounds.
	// A nil CardFont falls back to Font for synthetic and degraded callers.
	CardFont   *text.Font
	HiredLabel string
	BookView   *ShopScreenView

	// SchoolArt is optional so hand-built front ends and missing cosmetic data
	// keep a total text fallback. SchoolClass is fighter=0, mage=1; -1 means
	// that no selected member can own a school panel.
	SchoolArt   *TownSchoolArt
	SchoolClass int
	// SchoolColumnSet selects the explicit displayed frame, including the
	// intermediate frames with no skill panel. Zero preserves static callers.
	SchoolColumnFrame int
	SchoolColumnSet   bool
	// SchoolTraining carries at most one chosen picture per side. The
	// controller resolves m-over-tr priority before constructing this view.
	SchoolTraining SchoolTrainingFrame
	// SchoolDiamondActive is the current-image and nonzero-step paint gate.
	// The game owns the phase; reading a view never advances it (TOWN-381).
	SchoolDiamondFrame  int
	SchoolDiamondActive bool
	// SchoolIdleShine paints the shine picture of slot SchoolIdleSlot of the
	// shown class while no skill is hovered (TOWN-500). The slot is in the
	// shared slot order.
	SchoolIdleShine bool
	SchoolIdleSlot  int
	HoverCell       int
	// RosterUnpainted is the tavern's paint guard (TOWN-468): while the
	// selection is -1 neither paint loop draws a cell. The hit test still
	// covers every position.
	RosterUnpainted bool
	TavernArt       *TownTavernArt
	TavernInterior  TavernInteriorFrame
	// AnimationFrame is a client presentation phase. Only the selected
	// portrait cell consumes it; every other card holds its first frame.
	AnimationFrame int

	// Candidate is the tavern's selected mercenary, projected into the
	// room's own read-only left column. CandidateSlotMask and
	// CandidateSlotInfo are observations of that same composed figure: they
	// can produce the normal item hover, but they are deliberately absent
	// from TownSurfaceControlAt and from every drag source.
	Candidate         TownCharacterView
	CandidateSlotMask *SlotMask
	CandidateSlotInfo [12][]string
	CandidatePixels   *image.RGBA
	// CandidateText is the glyphs CandidatePixels was drawn with (text.Record),
	// re-captured each frame the cached picture is pasted.
	CandidateText []text.DrawCall
	Hover         image.Point
	HasHover      bool

	// Tip is the room's own tip panel (1018 spec behaviours 1, 2), shared by
	// school and tavern on one field: SchoolTipRect/TavernTipRect (tippanel.go)
	// tell them apart, on Kind's own precedent for the rest of this struct.
	Tip TipPanelView
}

var (
	tavernCandidateStatsRect = image.Rect(0, 0, 160, 238)
	tavernCandidateDollRect  = image.Rect(0, 238, 160, 480)
	// Owner correction: move the left candidate four pixels to the right.
	// Drawing and equipment hover use this same origin, (8,240) at native size.
	tavernCandidateDollShift = image.Pt(8, 1)
)

// tavernCandidateCardOffset: the original draws the panel into (12,0)-(172,238).
var tavernCandidateCardOffset = image.Pt(12, 0)

func tavernCandidateStats(v TownCharacterView, art *TownTavernArt) TownCharacterView {
	v.Statistics, v.PaneRect = true, tavernCandidateStatsRect
	if art != nil {
		v.StatsPane = TownPane{Body: art.LeftStats, Seam: art.LeftStatsSeam}
		v.CardOffset = tavernCandidateCardOffset
	}
	return v
}

// TownCandidateHoverLines returns the selected candidate's normal worn-item
// tooltip at p. The answer is read-only presentation data: this helper does
// not expose a control, item id, or mutation target to the input layer.
func TownCandidateHoverLines(v TownSurfaceView, p image.Point) ([]string, bool) {
	if v.Kind != TownSurfaceTavern || !v.Candidate.HasSubject || v.CandidateSlotMask == nil || !p.In(tavernCandidateDollRect) {
		return nil, false
	}
	m := v.CandidateSlotMask
	at := image.Pt(
		tavernCandidateDollRect.Min.X+(tavernCandidateDollRect.Dx()-m.W)/2+tavernCandidateDollShift.X,
		tavernCandidateDollRect.Min.Y+(tavernCandidateDollRect.Dy()-m.H)/2+tavernCandidateDollShift.Y,
	)
	slot, ok := m.At(p.X-at.X, p.Y-at.Y)
	if !ok || slot < 1 || slot > len(v.CandidateSlotInfo) || len(v.CandidateSlotInfo[slot-1]) == 0 {
		return nil, false
	}
	return v.CandidateSlotInfo[slot-1], true
}

type TownSurfaceControlKind uint8

const (
	TownSurfaceControlNone TownSurfaceControlKind = iota
	TownSurfaceControlButton
	TownSurfaceControlCell
	TownSurfaceControlPrevious
	TownSurfaceControlNext
	TownSurfaceControlMode
	TownSurfaceControlBook
	TownSurfaceControlBookPage
)

type TownSurfaceControl struct {
	Kind  TownSurfaceControlKind
	Index int
}

func townSurfaceUpperRect(kind TownSurfaceKind) image.Rectangle {
	if kind == TownSurfaceSchool {
		return TownWideUpperRegion
	}
	return TownUpperRegion
}

// townSurfaceButtonWells gives each school button's own well, in coordinates
// local to the button-area picture's own origin. The school's
// picture is drawn at TownUpperRegion, not the wider TownWideUpperRegion the
// school's own widget occupies; a well measured against the widget's edge
// instead of the picture's own origin lands off the shipped button art. The
// school wells are measured by sliding each shipped button bitmap over its
// room's own area picture. The tavern delegates to the shop's four rectangles
// below instead of maintaining a second copy of that composition.
var townSurfaceButtonWells = map[TownSurfaceKind][]image.Rectangle{
	TownSurfaceSchool: {
		image.Rect(4, 71, 144, 117),
		image.Rect(4, 117, 144, 163),
	},
}

// TownSurfaceButtonWell is the well townSurfaceButtonRect draws button i in
// and hit-tests it against, local to that room's button-area picture: the
// school's 160-wide body at TownUpperRegion.Min, or the tavern's shop-derived
// 176-wide body at TownWideUpperRegion.Min. Exported for buttonframecheck.
// Unknown kind/index returns the zero rectangle.
func TownSurfaceButtonWell(kind TownSurfaceKind, i int) image.Rectangle {
	if kind == TownSurfaceTavern {
		r := ShopButtonRect(i)
		if r.Empty() {
			return image.Rectangle{}
		}
		return r.Sub(TownWideUpperRegion.Min)
	}
	if wells, ok := townSurfaceButtonWells[kind]; ok && i >= 0 && i < len(wells) {
		return wells[i]
	}
	return image.Rectangle{}
}

// TownSurfaceButtonRect is the same rectangle used to draw button i and as
// the hit-test argument for it: draw position and hit rectangle are one
// RECT, never two independently maintained ones. Tavern delegates directly
// to ShopButtonRect, so its four-button composition cannot drift from the
// panel the owner selected. Unknown kinds retain the flat-box fallback.
func TownSurfaceButtonRect(kind TownSurfaceKind, i int) image.Rectangle {
	if kind == TownSurfaceTavern {
		return ShopButtonRect(i)
	}
	if wells, ok := townSurfaceButtonWells[kind]; ok && i >= 0 && i < len(wells) {
		return wells[i].Add(TownUpperRegion.Min)
	}
	r := townSurfaceUpperRect(kind)
	left := r.Min.X + 10
	if kind == TownSurfaceSchool {
		left = r.Min.X
	}
	return image.Rect(left, 16+i*68, r.Max.X-10, 68+i*68)
}

func townSurfaceButtonRect(kind TownSurfaceKind, i int) image.Rectangle {
	return TownSurfaceButtonRect(kind, i)
}

// townButtonLabelRect and townButtonValueRect are a surface button's own two
// text bands within its well r: the label on the first line, the button's
// current value on the second. Both are named producers rather than inline
// literals (contract B2) so a shift of either can be pinned directly against
// a hand-transcribed rectangle, the same shape preControlRegion and
// schoolSkillRects already carry.
func townButtonLabelRect(r image.Rectangle) image.Rectangle {
	return image.Rect(r.Min.X, r.Min.Y+2, r.Max.X, r.Min.Y+24)
}

func townButtonValueRect(r image.Rectangle) image.Rectangle {
	return image.Rect(r.Min.X, r.Min.Y+23, r.Max.X, r.Max.Y-2)
}

// townSurfaceMessageRect is the room's own status-message band, drawn under
// the character region in every mode (1022 spec B6: it no longer suppresses
// during Statistics). Named producer rather than an inline literal (story
// 1024, round 2: a real, ungated composition test must be able to fail when
// this rectangle moves; see TestTownSurfaceMessageIsDrawnAtItsOwnRect).
func townSurfaceMessageRect() image.Rectangle {
	return image.Rect(12, 448, 468, 478)
}

// townSurfaceCellRect is the roster cell's own draw and hit rectangle.
// total is the roster's own cell count (len(v.Cells)); the tavern case
// below anchors its whole grid to the room's own bottom edge, which needs
// the row count, and the school case ignores it (a fixed 2x5 skill grid).
func townSurfaceCellRect(kind TownSurfaceKind, i, total int) image.Rectangle {
	switch kind {
	case TownSurfaceSchool:
		// The two five-slot rows deliberately share this one mapping. The
		// first row is fighter, the second mage.
		col, row := i%5, i/5
		return image.Rect(10+col*92, 64+row*146, 94+col*92, 184+row*146)
	default:
		return tavernTextCellRect(i, total, 472)
	}
}

func tavernTextCellRect(i, total, gridBottom int) image.Rectangle {
	const cols, width, stride, cellHeight, rowStride = 2, 144, 152, 84, 92
	// gridTopFloor keeps the grid from climbing over the title text
	// (drawn at y:[12,44)) for a roster deep enough to need more rows
	// than the room's own height admits between the title and the
	// bottom edge. A roster past that depth overlaps its own earlier
	// rows rather than the title or running off either edge; the
	// original's own roster population and this build's are both
	// bounded well under that depth (a chapter's own mercenary list
	// plus its NPC offers), so this is a floor for the unreached case,
	// not a scroll or pagination this story does not build.
	const gridTopFloor = 56
	if total <= 0 {
		total = 1
	}
	rows := (total + cols - 1) / cols
	col, row := i%cols, i/cols
	rowsBelow := rows - 1 - row
	yMax := gridBottom - rowsBelow*rowStride
	if yMax-cellHeight < gridTopFloor {
		yMax = gridTopFloor + cellHeight
	}
	x0 := 176 + col*stride
	return image.Rect(x0, yMax-cellHeight, x0+width, yMax)
}

const (
	tavernPortraitCols       = 6
	tavernPortraitWidth      = 48
	tavernPortraitHeight     = 64
	tavernPortraitStrideX    = 48
	tavernPortraitStrideY    = 64
	tavernPortraitGridLeft   = 176
	tavernPortraitGridBottom = 480
)

func tavernPortraitCellRect(i int) image.Rectangle {
	col, row := i%tavernPortraitCols, i/tavernPortraitCols
	yMax := tavernPortraitGridBottom - row*tavernPortraitStrideY
	x0 := tavernPortraitGridLeft + col*tavernPortraitStrideX
	return image.Rect(x0, yMax-tavernPortraitHeight, x0+tavernPortraitWidth, yMax)
}

// townSurfaceCellRectFor separates the tavern's compact mercenary portraits
// from its ordinary NPC rows. Portraits fill the bottom row left-to-right and
// then climb upward; any NPC rows occupy the remaining rows above them.
func townSurfaceCellRectFor(v TownSurfaceView, i int) image.Rectangle {
	if v.Kind != TownSurfaceTavern || i < 0 || i >= len(v.Cells) {
		return townSurfaceCellRect(v.Kind, i, len(v.Cells))
	}
	portraitOrdinal, textOrdinal := 0, 0
	portraitCount, textCount := 0, 0
	for n, cell := range v.Cells {
		if cell.Portrait {
			if n < i {
				portraitOrdinal++
			}
			portraitCount++
		} else {
			if n < i {
				textOrdinal++
			}
			textCount++
		}
	}
	if v.Cells[i].Portrait {
		return tavernPortraitCellRect(portraitOrdinal)
	}
	portraitRows := (portraitCount + tavernPortraitCols - 1) / tavernPortraitCols
	return tavernTextCellRect(textOrdinal, textCount, tavernPortraitGridBottom-portraitRows*tavernPortraitStrideY)
}

// SchoolFaceOrigin is where the training column's own rotation frame sits in
// the 480x480 room. It is measured, not authored: cmd/schoolcheck correlates
// the shipped frame against the shipped background over the whole background
// and reports the winning offset with the best fraction anywhere else.
var SchoolFaceOrigin = image.Pt(168, 176)

// schoolSkillRects is each class's five skill rectangles, indexed by class then
// by the shared slot order (Blade/Fire, Axe/Water, Bludgen/Air, Pike/Earth,
// Shooting/Astral).
//
// THE TWO CLASSES DO NOT SHARE ONE ARRAY. The two column faces are two faces of
// one cylinder and their engravings are in different places: the fighter's five
// are full-width horizontal bands, the mage's five are small symbols scattered
// over the face. Every rectangle here is the patch's own placement on its own
// face, measured by cmd/schoolcheck and cross-checked against the class mask's
// colour-code bounding boxes. docs/DIVERGENCES.md DIV-145 carries the conflict
// with TOWN-068's one-shared-array reading.
var schoolSkillRects = [2][5]image.Rectangle{
	{
		image.Rect(200, 196, 280, 228),
		image.Rect(200, 216, 280, 252),
		// Research establishes one shared rectangle array but not why its third
		// and fourth entries cross visual order. The owner selected detailed
		// chargen order, so Bludgen/Club uses the visually earlier 28px row and
		// Pike the later 16px row.
		image.Rect(200, 248, 280, 276),
		image.Rect(200, 272, 280, 288),
		image.Rect(200, 288, 280, 308),
	},
	{
		image.Rect(264, 232, 284, 260),
		image.Rect(192, 240, 216, 260),
		image.Rect(224, 200, 252, 224),
		image.Rect(228, 272, 256, 298),
		image.Rect(224, 236, 256, 262),
	},
}

var schoolPanelRects = [2]image.Rectangle{
	image.Rect(192, 192, 284, 312), // fighter
	image.Rect(188, 188, 288, 308), // mage
}

// SchoolSkillRect is the screen rectangle one class's skill icon is drawn at,
// and SchoolPanelRect the rectangle that class's raster mask is placed at.
//
// BOTH ARE EXPORTED FOR cmd/schoolcheck, the story's measuring instrument. That
// tool derives the same rectangles from the shipped art by correlation and from
// the mask's own colour-code bounding boxes, and compares them with these
// values. A Go test may not read an install (golden rule 2), so the comparison
// can only be made from a command. An out-of-range argument answers the empty
// rectangle rather than panicking.
func SchoolSkillRect(class, slot int) image.Rectangle {
	if class < 0 || class > 1 || slot < 0 || slot > 4 {
		return image.Rectangle{}
	}
	return schoolSkillRects[class][slot]
}

func SchoolPanelRect(class int) image.Rectangle {
	if class < 0 || class > 1 {
		return image.Rectangle{}
	}
	return schoolPanelRects[class]
}

// schoolMaskSlot maps one mask colour code to the slot index the rest of this
// package uses: the shared slot order Blade/Fire, Axe/Water, Bludgen/Air,
// Pike/Earth, Shooting/Astral, which is the index schoolSkillRects is drawn by
// and the index the game side turns into a hero skill.
//
// THE ANSWER IS A VISUAL CELL INDEX, NOT A STORED SLOT. schoolControlAt hands
// it to townSurfaceCellState and returns class*5+slot as a cell index;
// townScreen.schoolCell then becomes that index, schoolSurfaceCells labels cell
// i with data.SkillNames[i%5], and selectedSchoolSlot trains hero skill i%5+1.
// Every consumer reads it the same way, so the table must agree with the DRAW
// side and not with the original's own storage order.
//
// UNTIL 1015 IT DID NOT. The table reproduced the original's visual-to-stored
// permutation 1,2,4,3,5 (TOWN-GENERAL-106) while schoolSkillRects had been
// re-ordered to detailed character generation's 1,2,3,4,5 under DIV-121, so a
// click on the third icon selected the fourth skill and spent its price, on
// both classes. DIV-121 records that the owner's chosen order binds the hit
// test as well as the draw.
//
// EACH ENTRY IS MEASURED. cmd/schoolcheck takes each code's bounding box in the
// class mask, places it at the class panel rectangle, and matches it to the
// skill patch it overlaps most; the ten assignments below are that
// correspondence, and the tool exits non-zero when this table stops agreeing
// with it.
func schoolMaskSlot(class int, code uint8) (int, bool) {
	if class == 1 { // mage, 100x120 panel
		switch code {
		case 0x87:
			return 0, true // fire
		case 0x37:
			return 1, true // water
		case 0xff:
			return 2, true // air
		case 0xd2:
			return 3, true // earth
		case 0x9e:
			return 4, true // astral
		}
	} else if class == 0 { // fighter, 92x120 panel
		switch code {
		case 0xff:
			return 0, true // sword
		case 0x9e:
			return 1, true // axe
		case 0xd2:
			return 2, true // club
		case 0x87:
			return 3, true // pike
		case 0x37:
			return 4, true // bow
		}
	}
	return 0, false
}

// townSurfaceCellState is the shared visibility/eligibility answer for pointer
// and semantic activation. Statistics no longer replaces the left surface
// (1022 spec B6: the toggle swaps the character pane's own doll for the card,
// leaving the room's own content, title and roster live in both modes).
func townSurfaceCellState(v TownSurfaceView, i int) (visible, enabled bool) {
	if i < 0 || i >= len(v.Cells) {
		return false, false
	}
	return true, v.Kind != TownSurfaceSchool || v.Cells[i].Enabled
}

func schoolControlAt(v TownSurfaceView, p image.Point) (TownSurfaceControl, bool) {
	if v.SchoolArt == nil || !schoolPanelVisible(v) {
		return TownSurfaceControl{}, false
	}
	mask := v.SchoolArt.Masks[v.SchoolClass]
	r := schoolPanelRects[v.SchoolClass]
	if mask == nil || !p.In(r) {
		return TownSurfaceControl{}, false
	}
	q := p.Sub(r.Min).Add(mask.Bounds().Min)
	if !q.In(mask.Bounds()) {
		return TownSurfaceControl{}, false
	}
	code := mask.ColorIndexAt(q.X, q.Y)
	slot, ok := schoolMaskSlot(v.SchoolClass, code)
	if !ok {
		return TownSurfaceControl{}, false
	}
	i := v.SchoolClass*5 + slot
	if visible, enabled := townSurfaceCellState(v, i); !visible || !enabled {
		return TownSurfaceControl{}, false
	}
	return TownSurfaceControl{Kind: TownSurfaceControlCell, Index: i}, true
}

// TownSurfaceControlAt resolves the extension and character regions before
// left content, including the school's sixteen-pixel overlap.
func TownSurfaceControlAt(v TownSurfaceView, p image.Point) (TownSurfaceControl, bool) {
	if v.BookView != nil && p.In(shopTableRegion) {
		return TownSurfaceControl{Kind: TownSurfaceControlBookPage}, true
	}
	for i := range v.Buttons {
		if p.In(townSurfaceButtonRect(v.Kind, i)) {
			if !v.Buttons[i].Enabled {
				return TownSurfaceControl{}, false
			}
			return TownSurfaceControl{Kind: TownSurfaceControlButton, Index: i}, true
		}
	}
	// Shared character controls precede the room content.
	for _, c := range CharacterPaneCornerAt(v.Hero, p) {
		switch c {
		case CharacterPanePrev:
			if v.Hero.MemberCount <= 1 {
				return TownSurfaceControl{}, false
			}
			return TownSurfaceControl{Kind: TownSurfaceControlPrevious}, true
		case CharacterPaneNext:
			if v.Hero.MemberCount <= 1 {
				return TownSurfaceControl{}, false
			}
			return TownSurfaceControl{Kind: TownSurfaceControlNext}, true
		case CharacterPaneMode:
			if !v.Hero.HasSubject {
				return TownSurfaceControl{}, false
			}
			return TownSurfaceControl{Kind: TownSurfaceControlMode}, true
		case CharacterPaneBook:
			return TownSurfaceControl{Kind: TownSurfaceControlBook}, true
		}
	}
	if v.Kind == TownSurfaceSchool && v.SchoolArt != nil {
		return schoolControlAt(v, p)
	}
	for i := range v.Cells {
		if p.In(townSurfaceCellRectFor(v, i)) {
			if visible, enabled := townSurfaceCellState(v, i); !visible || !enabled {
				return TownSurfaceControl{}, false
			}
			return TownSurfaceControl{Kind: TownSurfaceControlCell, Index: i}, true
		}
	}
	return TownSurfaceControl{}, false
}

var (
	townShellFill     = color.RGBA{R: 0x0d, G: 0x0e, B: 0x13, A: 0xff}
	townShellPanel    = color.RGBA{R: 0x18, G: 0x16, B: 0x15, A: 0xff}
	townShellBorder   = color.RGBA{R: 0x8a, G: 0x74, B: 0x46, A: 0xff}
	townShellText     = color.RGBA{R: 0xf2, G: 0xe6, B: 0xc4, A: 0xff}
	townShellDisabled = color.RGBA{R: 0x68, G: 0x62, B: 0x59, A: 0xff}
	townShellSelected = color.RGBA{R: 0x54, G: 0x3e, B: 0x1c, A: 0xff}
	tavernCardText    = color.RGBA{R: 0xbd, G: 0x9e, B: 0x4a, A: 0xff}
)

func drawTavernCardText(dst *image.RGBA, font *text.Font, s string, r image.Rectangle, right, bottom bool) {
	if font == nil || s == "" {
		return
	}
	w, _ := font.Measure(s)
	x, y := r.Min.X, r.Min.Y
	if right {
		x = r.Max.X - w
	}
	if bottom {
		y = r.Max.Y - font.Height()
	}
	font.Draw(dst, s, x, y, tavernCardText)
}

// TownCharacterPersistentControls is every corner rectangle
// DrawTownCharacterRegion paints over the shared character pane in EVERY
// mode, DOLL or STATS. Exported so a release test can exclude them from a
// pane-body pixel comparison without hand-duplicating these rects (round-2
// adversarial review, item 2: townpanes_release_test.go's own
// characterPanelControls was a copy of these literals, which could drift from
// them silently).
func TownCharacterPersistentControls() []image.Rectangle {
	r := TownCharacterRegion
	out := make([]image.Rectangle, 0, characterPaneCornerCount)
	for c := CharacterPaneCorner(0); c < characterPaneCornerCount; c++ {
		// CharacterPaneShop's own flag is always false: characterPaneModeFlag
		// returns false for every session but CharacterPaneMission, so this
		// call site never had a real flag to compute in the first place.
		if !characterPaneCornerDrawn(false, CharacterPaneShop, c, false) {
			continue
		}
		// The union of the hit rectangle and the art rectangle, since story
		// 1036: four of the six blit somewhere other than where they answer
		// (characterPaneArtRect), and a comparison excluding only the hit
		// rectangle would still meet painted pixels.
		box := CharacterPaneCornerRect(r, c)
		box = box.Union(characterPaneArtRect(r, c, false))
		box = box.Union(characterPaneArtRect(r, c, true))
		out = append(out, box.Intersect(r))
	}
	return out
}

func townShellTextLayout(font *text.Font, s string, r image.Rectangle) (string, image.Point, bool) {
	if font == nil || s == "" {
		return "", image.Point{}, false
	}
	for {
		w, _ := font.Measure(s)
		if w <= r.Dx()-6 || len(s) <= 1 {
			break
		}
		s = s[:len(s)-1]
	}
	w, h := font.Measure(s)
	return s, image.Pt(r.Min.X+(r.Dx()-w)/2, r.Min.Y+(r.Dy()-h)/2), true
}

func drawTownShellText(dst *image.RGBA, font *text.Font, s string, r image.Rectangle, c color.RGBA) {
	s, at, ok := townShellTextLayout(font, s, r)
	if !ok {
		return
	}
	font.Draw(dst, s, at.X, at.Y, c)
}

func drawTownShellBox(dst *image.RGBA, r image.Rectangle, selected bool) {
	c := townShellPanel
	if selected {
		c = townShellSelected
	}
	draw.Draw(dst, r, &image.Uniform{C: c}, image.Point{}, draw.Src)
	outline(dst, r, townShellBorder)
}

// drawTownShellOutline is drawTownShellBox with its own opaque interior fill
// removed (round-2 adversarial review, owner item: the near-black control
// boxes now sitting on shipped art). Before this story, TownCharacterRegion
// was always the authored townShellPanel fill underneath everything drawn
// over it, so an opaque control box there was invisible against its own
// background. This story draws the room's own shipped body/seam art in that
// region first (DrawTownCharacterRegion), and townCharacterMode's own
// opaque fill covered 3,584px of it on the tavern and the school; the
// selection state still shows through the outline's own colour, which
// switches to townShellSelected exactly as the fill used to, and the
// control's own rectangle and hit test are unchanged.
func drawTownShellOutline(dst *image.RGBA, r image.Rectangle, selected bool) {
	c := townShellBorder
	if selected {
		c = townShellSelected
	}
	outline(dst, r, c)
}

// DrawTownCharacterRegion paints the shared member projection and its
// persistent controls.
//
// THE PANE DRAWS FIRST (1021 spec B3): the mode-switched pane's shipped body
// and seam replace the authored near-black fill this region drew
// unconditionally on every screen before this story — the near-black box
// the owner reported on the shop, the school, the tavern and the generator.
// A selected pane with no Body (a caller predating this story, or a load
// failure) falls back to the same authored fill and outline as before, so
// the region is never left blank.
//
// THE CARD REPLACES THE DOLL IN STATISTICS MODE (1022 spec B6):
// townStatisticsCardRegion used to tie the town's own stats card to the
// generator's chargenCardBox, on the left content surface, which moved to
// (0,238)-(160,480) this story (B3) and would otherwise have carried the
// town's card along with it — into a rectangle B3 already fills with the
// generator's own card. The contract's own reading of the DOLL/STATS toggle
// is that it selects what THIS region shows, not the room's left content, so
// the card now composes here, at TownCharacterRegion, the same 160x242 size
// as chargenCardBox, using the pane's own body (StatsPane, textbackr.bmp per
// DIV-175) as CompactPanelLayout's background — the same shipped bitmap the
// pane draw above already painted underneath it. This applies uniformly to
// every DrawTownCharacterRegion caller, including the shop: shopGridControlAt
// and app.go's own cross-family drag release are both gated off the doll
// while Statistics is true, so no drag surface answers under the card.
func DrawTownCharacterRegion(dst *image.RGBA, v TownCharacterView) {
	drawCharacterPaneBody(dst, v)
	drawCharacterPaneCorners(dst, v)
}

func DrawCharacterPaneBody(dst *image.RGBA, v TownCharacterView) {
	drawCharacterPaneBody(dst, v)
}

// drawCharacterPaneBody paints the pane's background art and content (the
// statistics card or the unlabelled figure crop) without the six corner
// controls.
//
// TOWN-086
func drawCharacterPaneBody(dst *image.RGBA, v TownCharacterView) {
	r := v.paneRect()
	seam := image.Rect(r.Min.X-16, r.Min.Y, r.Min.X, r.Max.Y)
	pane := v.FigurePane
	if v.Statistics {
		pane = v.StatsPane
	}
	if pane.Body != nil {
		drawTownPane(dst, pane, r, seam)
	} else {
		draw.Draw(dst, r, &image.Uniform{C: townShellPanel}, image.Point{}, draw.Src)
		outline(dst, r, townShellBorder)
	}
	if v.HasSubject {
		if v.Statistics {
			layout := v.statsLayout()
			// RenderCharacterPanel composes on its own small canvas, pasted here
			// at a computed offset — capture.go's own ShiftCaptured doc names
			// this exact site. markCapture/shiftCapture are unconditional
			// no-ops with no capture window open, so this costs nothing when
			// text smoothing is off or this call has no outer window (DIV-1385).
			start := markCapture()
			if card := RenderCharacterPanel(layout, v.cardFont(), v.Subject); card != nil {
				b := card.Bounds()
				cr := v.cardRect()
				at := cr.Min.Sub(b.Min)
				if cr == r {
					draw.Draw(dst, b.Add(at).Intersect(r), card, b.Min, draw.Src)
				} else {
					// Recolour the strip left of the canvas; keep the seam under it.
					if bg := v.cardBoard(); bg != nil {
						lead := image.Rect(r.Min.X, r.Min.Y, cr.Min.X, r.Max.Y)
						draw.Draw(dst, lead.Intersect(r), bg, bg.Bounds().Min.Add(lead.Min.Sub(r.Min)), draw.Src)
					}
					draw.Draw(dst, b.Add(at).Intersect(cr), card, b.Min, draw.Over)
				}
				shiftCapture(start, at)
			}
		} else if v.Figure != nil {
			b := v.Figure.Bounds()
			draw.Draw(dst, characterPaneFigureRect(r), v.Figure, b.Min, draw.Over)
		}
	}
	if !v.HasSubject && !v.Statistics && (v.SelectionStatus[0] != "" || v.SelectionStatus[1] != "") {
		drawSelectionStatus(dst, v.cardFont(), r, v.SelectionStatus)
	}
	// The subject's name belongs to CompactPanelLayout's centred first row.
	// DOLL mode paints only the figure: no duplicate label covers equipment or
	// steals its hover/drag pixels.
}

// SelectionStatusLines resolves the zero/plural phrases from the word set; the
// count is a line of its own. One actor has no status.
func SelectionStatusLines(words Words, count int) ([3]string, bool) {
	authored := AuthoredWords().SelectionStatus
	for i := range words.SelectionStatus {
		if words.SelectionStatus[i] == "" {
			words.SelectionStatus[i] = authored[i]
		}
	}
	switch {
	case count == 0:
		return [3]string{words.SelectionStatus[0], words.SelectionStatus[1]}, true
	case count >= 2:
		return [3]string{words.SelectionStatus[2], words.SelectionStatus[3], strconv.Itoa(count)}, true
	default:
		return [3]string{}, false
	}
}

// The selection lines' rows in the upper pane, as the owner's capture has them.
const (
	selectionLinePitch = 12
	selectionLineTop   = 54
)

// selectionShadow is the flat shadow's offset under each line.
const selectionShadow = 1

// drawSelectionStatus states the selection lines in gold over a flat shadow,
// each centred on the 176-pixel column's middle (seam included); an empty line
// is skipped and a wider one starts at the seam's left edge.
//
// DIV-1795
func drawSelectionStatus(dst *image.RGBA, font *text.Font, pane image.Rectangle, lines [3]string) {
	if font == nil {
		return
	}
	column := image.Rect(pane.Min.X-characterPaneSeamW, pane.Min.Y, pane.Max.X, pane.Max.Y)
	for i, line := range lines {
		if line == "" {
			continue
		}
		w, _ := font.Measure(line)
		x, y := column.Min.X+centredOffset(column.Dx(), w), column.Min.Y+selectionLineTop+i*selectionLinePitch
		font.DrawFlat(dst, line, x+selectionShadow, y+selectionShadow, messageShadowColor)
		font.Draw(dst, line, x, y, tavernCardText)
	}
}

// centredOffset is how far in from a span's near edge a run of the given size
// starts when it is centred in the span, an odd remainder falling on the near
// side. A run larger than the span starts at the near edge.
func centredOffset(span, size int) int {
	return max(0, (span-size+1)/2)
}

// drawCharacterPaneCorners blits whichever of the six corners this pane
// paints, under the original's own art gates rather than under its hit
// gates.
//
// THE TWO GATE SETS DIFFER FOR THREE OF THE SIX AND THAT IS REPRODUCED, NOT
// REPAIRED. `TOWN-355` (High for the gate expressions, all instruction-level)
// finds rect A's and rect F's hit tests gated on `sess & 1` while their art is
// gated on `!(sess & 0x226)` and `!(sess & 0x400)`, so with `sess == 3` — a
// shop opened from a mission — both answer clicks and neither is drawn. Rects
// D and E keep their hit test under `sess & 0x226` and lose their art under a
// second condition. Drawing a live control is not optional in general; here
// the original's own mismatch is the behaviour, and characterPaneCornerDrawn
// is the function that carries it. DIV-309.
//
// A CORNER WITH NO BITMAP FALLS BACK TO THE AUTHORED CHROME the pane drew
// before this story — the two picker chevrons and an outline — so a caller
// that resolved no art, or an install missing one node, still shows a control
// where a control answers. drawShopChevron's own `fill` argument keeps the
// legible plate the statistics card needs behind it (1022 round 3).
func drawCharacterPaneCorners(dst *image.RGBA, v TownCharacterView) {
	r := v.paneRect()
	s := v.paneSession()
	figureMode := !v.Statistics
	live := v.MemberCount > 1
	for c := CharacterPaneCorner(0); c < characterPaneCornerCount; c++ {
		if !characterPaneCornerDrawn(v.paneModeFlag(), s, c, false) || v.NoBook && c == CharacterPaneBook {
			continue
		}
		pic, alt := characterPaneCornerPicture(v.CornerArt, c, v.PackOpen, v.BookOpen, figureMode)
		if pic != nil {
			at := characterPaneArtRect(r, c, alt)
			draw.Draw(dst, at.Intersect(r), pic, pic.Bounds().Min, draw.Over)
			continue
		}
		box := CharacterPaneCornerRect(r, c)
		// Preserve the bare top corners when their art is missing. The bottom
		// controls retain their fallback even though the complete figure now
		// extends behind them.
		if c == CharacterPaneBook || c == CharacterPaneMode {
			continue
		}
		switch c {
		case CharacterPanePrev:
			drawShopChevron(dst, box, -1, live, v.Statistics)
		case CharacterPaneNext:
			drawShopChevron(dst, box, +1, live, v.Statistics)
		case CharacterPaneMode:
			if v.Statistics {
				draw.Draw(dst, box, &image.Uniform{C: townShellPanel}, image.Point{}, draw.Src)
			}
			drawTownShellOutline(dst, box, v.Statistics)
			mode := "DOLL"
			if v.Statistics {
				mode = "STATS"
			}
			col := townShellText
			if !v.HasSubject {
				col = townShellDisabled
			}
			drawTownShellText(dst, v.Font, mode, box, col)
		default:
			drawTownShellOutline(dst, box, false)
		}
	}
}

// cardFont is the font DrawTownCharacterRegion composes the statistics card
// with: CardFont where the caller set one, Font otherwise (1027 B1).
func (v TownCharacterView) cardFont() *text.Font {
	if v.CardFont != nil {
		return v.CardFont
	}
	return v.Font
}

// RenderTownCandidateInspection rasterizes the selected tavern candidate's
// immutable statistics card and equipment figure once. Hover text and the
// animated roster card remain outside this image.
func RenderTownCandidateInspection(v TownCharacterView, art *TownTavernArt) *image.RGBA {
	if art == nil || !v.HasSubject && v.Figure == nil {
		return nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, 176, 480))
	if v.HasSubject {
		DrawCharacterPaneBody(dst, tavernCandidateStats(v, art))
	}
	drawInventoryPictureShifted(dst, tavernCandidateDollRect, v.Figure, tavernCandidateDollShift)
	return dst
}

// ComposeTownSurface paints one complete shared-shell room. Optional art is
// absent by construction here; the fallback furniture and every live control
// remain visible and hit-testable.
func ComposeTownSurface(v TownSurfaceView) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: townShellFill}, image.Point{}, draw.Src)
	artSchool := v.Kind == TownSurfaceSchool && v.SchoolArt != nil
	artTavern := v.Kind == TownSurfaceTavern && v.TavernArt != nil
	commandTavern := artTavern && len(v.Buttons) == 4 && v.TavernArt.CommandUpper != nil
	// Statistics no longer replaces this room's own left content (1022 spec
	// B6): the DOLL/STATS toggle swaps only the character pane, so the
	// room's own art, title and roster draw exactly as in DOLL mode.
	if artSchool && v.SchoolArt.Background != nil {
		draw.Draw(dst, TownContentRegion, v.SchoolArt.Background, v.SchoolArt.Background.Bounds().Min, draw.Src)
	} else if artTavern && v.TavernArt.Center != nil {
		// LeftStats (238 rows) draws upper, LeftPicture (242 rows) lower
		// (1021 spec B1/B3, DIV-176, swapped from this build's own former order):
		// the right column's own two slots are 238-upper/242-lower
		// (TownUpperRegion, TownCharacterRegion), the inn/ seam names read
		// stats-upper ("luover"/"ruover", 238 tall) and picture-lower
		// ("ldover", 242 tall), and the owner's own screenshot of the
		// original tavern shows stats at the top and the portrait at the
		// bottom. TOWN-282 reads both blits as unconditional and opaque but
		// does not resolve their destination, so the order is authored, not
		// decoded.
		draw.Draw(dst, image.Rect(160, 0, 480, 480), v.TavernArt.Center, v.TavernArt.Center.Bounds().Min, draw.Src)
		// TOWN-407: these are the central child's own pictures, placed after
		// CenterArea.bmp. The two tender families share one destination but
		// the controller exposes at most the currently selected mode.
		for _, layer := range []struct {
			pic image.Image
			at  image.Point
		}{
			{v.TavernInterior.Candle, image.Pt(160, 48)},
			{v.TavernInterior.Cauldron, image.Pt(420, 160)},
			{v.TavernInterior.Tender, image.Pt(240, 152)},
		} {
			if layer.pic == nil {
				continue
			}
			b := layer.pic.Bounds()
			placed := b.Add(layer.at.Sub(b.Min))
			clip := placed.Intersect(image.Rect(160, 0, 480, 480))
			if !clip.Empty() {
				src := b.Min.Add(clip.Min.Sub(placed.Min))
				draw.Draw(dst, clip, layer.pic, src, draw.Over)
			}
		}
		drawTownPane(dst, TownPane{Body: v.TavernArt.LeftStats, Seam: v.TavernArt.LeftStatsSeam}, image.Rect(0, 0, 160, 238), image.Rect(160, 0, 176, 238))
		drawTownPane(dst, TownPane{Body: v.TavernArt.LeftPicture, Seam: v.TavernArt.LeftPictureSeam}, image.Rect(0, 238, 160, 480), image.Rect(160, 238, 176, 480))
	} else {
		drawTownShellBox(dst, TownContentRegion, false)
	}
	if artSchool {
		// TOWN-428's own-painter order is background, one selected movie per
		// side, then the column, skills and diamond below. Both training
		// families are opaque; clip only at the room's content boundary.
		for _, layer := range []struct {
			pic image.Image
			at  image.Point
		}{
			{v.SchoolTraining.Mage, image.Pt(0, 200)},
			{v.SchoolTraining.Fighter, image.Pt(320, 200)},
		} {
			if layer.pic == nil {
				continue
			}
			b := layer.pic.Bounds()
			placed := b.Add(layer.at.Sub(b.Min))
			clip := placed.Intersect(TownContentRegion)
			if clip.Empty() {
				continue
			}
			src := b.Min.Add(clip.Min.Sub(placed.Min))
			draw.Draw(dst, clip, layer.pic, src, draw.Src)
		}
	}
	if artTavern && v.CandidatePixels != nil {
		draw.Draw(dst, v.CandidatePixels.Bounds(), v.CandidatePixels, v.CandidatePixels.Bounds().Min, draw.Over)
		text.Append(v.CandidateText, 0, 0)
	} else if artTavern && (v.Candidate.HasSubject || v.Candidate.Figure != nil) {
		if v.Candidate.HasSubject {
			DrawCharacterPaneBody(dst, tavernCandidateStats(v.Candidate, v.TavernArt))
		}

		// The lower picture is the same already-composed equipment figure used
		// by inventory and character panes. It is centred without another
		// crop, and no character-pane corner controls are painted over it.
		drawInventoryPictureShifted(dst, tavernCandidateDollRect, v.Candidate.Figure, tavernCandidateDollShift)
	}
	if artSchool && v.SchoolArt.Upper != nil {
		// No outline here (round 3): the school's shipped upper picture is 160
		// pixels wide, covering TownUpperRegion (480..640) exactly.
		// TownWideUpperRegion's own 464..640 widget rect is 16 pixels wider,
		// x:[464,480). TOWN-183 reads this widget's own paint routine whole and it
		// issues exactly three blits per call, the 160x238 background once and one
		// of ON/OFF per button, with no border primitive among them, so no
		// authored line is drawn — the gap is closed with UpperSeam instead, the
		// shipped strip DIV-166/DIV-168 identify by rendering, not with an
		// authored border colour. Composed through drawTownPane (1021 spec B1):
		// this room no longer computes the seam rectangle itself, only supplies
		// the two shipped bitmaps.
		drawTownPane(dst, TownPane{Body: v.SchoolArt.Upper, Seam: v.SchoolArt.UpperSeam}, TownUpperRegion, townUpperSeamRegion)
	} else if commandTavern {
		// Sleep made this a four-command panel. Use the shop composition
		// whole: its 176-wide body already owns both edge ornaments, and its
		// narrower first and fourth plaques are positioned around them.
		draw.Draw(dst, TownWideUpperRegion, v.TavernArt.CommandUpper, v.TavernArt.CommandUpper.Bounds().Min, draw.Over)
	} else if artTavern && v.TavernArt.Upper != nil {
		// Same seam, same strip (DIV-166, DIV-168): the tavern's own
		// buttonsarea.bmp is 160 wide and TownWideUpperRegion is 176, so
		// UpperSeam closes the same x:[464,480) gap here.
		drawTownPane(dst, TownPane{Body: v.TavernArt.Upper, Seam: v.TavernArt.UpperSeam}, TownUpperRegion, townUpperSeamRegion)
	} else {
		drawTownShellBox(dst, townSurfaceUpperRect(v.Kind), false)
	}
	if !artTavern && !artSchool {
		drawTownShellText(dst, v.Font, v.Title, image.Rect(176, 12, 470, 44), townShellText)
	}
	for i, c := range v.Cells {
		if artSchool || v.Kind == TownSurfaceTavern && v.RosterUnpainted {
			continue
		}
		r := townSurfaceCellRectFor(v, i)
		ink := townShellText
		if !c.Enabled {
			ink = townShellDisabled
		}
		if c.Portrait {
			cardFont := v.CardFont
			if cardFont == nil {
				cardFont = v.Font
			}
			back := image.Image(nil)
			if artTavern {
				back = v.TavernArt.ManBack
				if c.TalkOnly {
					back = v.TavernArt.ManBackTalk
				}
			}
			if back != nil {
				draw.Draw(dst, r, back, back.Bounds().Min, draw.Src)
			} else {
				drawTownShellBox(dst, r, false)
			}
			pic := c.Picture
			if c.Selected && len(c.Frames) > 0 {
				frame := v.AnimationFrame % len(c.Frames)
				if frame < 0 {
					frame += len(c.Frames)
				}
				pic = c.Frames[frame]
			} else if len(c.Frames) > 0 {
				pic = c.Frames[0]
			}
			if pic != nil {
				b := pic.Bounds()
				placed := b.Add(r.Min.Sub(b.Min))
				clip := r.Intersect(placed)
				if !clip.Empty() {
					src := b.Min.Add(clip.Min.Sub(placed.Min))
					draw.Draw(dst, clip, pic, src, draw.Over)
				}
			}
			drawTavernCardText(dst, cardFont, c.Price, r, true, false)
			drawTavernCardText(dst, cardFont, c.Detail, r, false, true)
			if c.Hired && cardFont != nil {
				label := v.HiredLabel
				if label == "" {
					label = AuthoredWords().TavernHired
				}
				w, _ := cardFont.Measure(label)
				cardFont.Draw(dst.SubImage(r).(*image.RGBA), label, r.Min.X+(r.Dx()-w)/2, r.Min.Y+(r.Dy()-cardFont.Height())/2, color.RGBA{R: 255, A: 255})
			}
		} else {
			drawTownShellBox(dst, r, c.Selected)
			if c.Picture != nil {
				b := c.Picture.Bounds()
				area := r.Inset(2)
				at := image.Pt(area.Min.X+(area.Dx()-b.Dx())/2, area.Min.Y+(area.Dy()-b.Dy())/2)
				placed := b.Add(at.Sub(b.Min))
				clip := placed.Intersect(area)
				if !clip.Empty() {
					src := b.Min.Add(clip.Min.Sub(placed.Min))
					draw.Draw(dst, clip, c.Picture, src, draw.Over)
				}
			}
			drawTownShellText(dst, v.Font, c.Label, image.Rect(r.Min.X+2, r.Min.Y+4, r.Max.X-2, r.Min.Y+34), ink)
			drawTownShellText(dst, v.Font, c.Detail, image.Rect(r.Min.X+2, r.Min.Y+34, r.Max.X-2, r.Max.Y-4), ink)
		}
		if c.Hired && !c.Portrait {
			outline(dst, r.Inset(3), townShellText)
		}
	}
	if artSchool && v.SchoolColumnSet {
		if frame := v.SchoolColumnFrame; frame >= 0 && frame < len(v.SchoolArt.Column) {
			if pic := v.SchoolArt.Column[frame]; pic != nil {
				b := pic.Bounds()
				draw.Draw(dst, b.Add(SchoolFaceOrigin.Sub(b.Min)), pic, b.Min, draw.Src)
			}
		}
	}
	if artSchool && schoolPanelVisible(v) {
		class := v.SchoolClass
		// The shown member's own column face, over whichever face the room
		// background bakes. It is opaque: the frame covers the same pixels the
		// background carries there, and a keyed blit would let the baked face
		// show through the other class's engravings.
		if face := v.SchoolArt.Faces[class]; face != nil && !v.SchoolColumnSet {
			b := face.Bounds()
			draw.Draw(dst, b.Add(SchoolFaceOrigin.Sub(b.Min)), face, b.Min, draw.Src)
		}
		for slot := 0; slot < 5; slot++ {
			i := class*5 + slot
			if i >= len(v.Cells) || !v.Cells[i].Enabled {
				continue
			}
			state := -1 // the rest state is engraved on the class's own face
			if v.Cells[i].Selected {
				state = 0
			}
			if v.HoverCell == i {
				state = 1
				if v.Cells[i].Selected {
					state = 2
				}
			}
			if state < 0 || v.SchoolArt.Skills[class][slot][state] == nil {
				continue
			}
			pic := v.SchoolArt.Skills[class][slot][state]
			r := schoolSkillRects[class][slot]
			b := pic.Bounds()
			at := image.Pt(r.Min.X+(r.Dx()-b.Dx())/2, r.Min.Y+(r.Dy()-b.Dy())/2)
			draw.Draw(dst, b.Add(at.Sub(b.Min)), pic, b.Min, draw.Over)
		}
		// The idle shine paints after every slot's own state: the shine picture
		// for a slot that is not selected, the shine-on picture for the
		// selected one (TOWN-500).
		if v.SchoolIdleShine && v.HoverCell < 0 && v.SchoolIdleSlot >= 0 && v.SchoolIdleSlot < 5 {
			slot := v.SchoolIdleSlot
			i := class*5 + slot
			state := 1
			if i < len(v.Cells) && v.Cells[i].Selected {
				state = 2
			}
			if i < len(v.Cells) && v.Cells[i].Enabled && v.SchoolArt.Skills[class][slot][state] != nil {
				pic := v.SchoolArt.Skills[class][slot][state]
				r := schoolSkillRects[class][slot]
				b := pic.Bounds()
				at := image.Pt(r.Min.X+(r.Dx()-b.Dx())/2, r.Min.Y+(r.Dy()-b.Dy())/2)
				draw.Draw(dst, b.Add(at.Sub(b.Min)), pic, b.Min, draw.Over)
			}
		}
	}
	if artSchool && v.SchoolDiamondActive && v.SchoolDiamondFrame >= 0 && v.SchoolDiamondFrame < len(v.SchoolArt.Diamond) {
		if pic := v.SchoolArt.Diamond[v.SchoolDiamondFrame]; pic != nil {
			b := pic.Bounds()
			// TOWN-154 paint step 6: opaque, independent of the class column.
			draw.Draw(dst, b.Add(image.Pt(200, 60).Sub(b.Min)), pic, b.Min, draw.Src)
		}
	}
	for i, b := range v.Buttons {
		r := townSurfaceButtonRect(v.Kind, i)
		// The button's shipped art, when its room supplies it. School and the
		// original tavern panel have off/on pairs; the shop-derived four-
		// command tavern has one keyed native bitmap per position. Draw
		// position and hit rectangle are the same RECT, which
		// TOWN-183 reads at instruction level: the paint's own walking
		// pointer lands on the same +0x88/+0x98 RECT that TOWN-182's PtInRect
		// hit test reads. State is off unless this control is the one
		// currently pressed AND under the cursor where an ON pair exists.
		var pic [2]image.Image
		var commandPic image.Image
		switch {
		case artSchool && i < len(v.SchoolArt.Buttons):
			pic = v.SchoolArt.Buttons[i]
		case commandTavern && i < len(v.TavernArt.CommandButtons):
			commandPic = v.TavernArt.CommandButtons[i]
		case artTavern:
			artIndex := i
			if len(v.Buttons) == 4 {
				if i == 0 {
					artIndex = 0
				} else {
					artIndex = i - 1
				}
			}
			if artIndex >= 0 && artIndex < len(v.TavernArt.Buttons) {
				pic = v.TavernArt.Buttons[artIndex]
			}
		}
		if commandPic != nil {
			// CommandUpper already carries the plaque. As in the shop
			// (TOWN-260), the button's own bitmap is drawn only while it is
			// pressed and hovered.
			if b.Enabled && v.Press.Kind == TownSurfaceControlButton && v.Press.Index == i {
				draw.Draw(dst, r, commandPic, commandPic.Bounds().Min, draw.Over)
			}
		} else if pic[0] != nil && pic[1] != nil {
			state := 0
			if v.Press.Kind == TownSurfaceControlButton && v.Press.Index == i {
				state = 1
			}
			// Exactly one of ON/OFF is blitted per button, never both
			// (TOWN-183): drawing OFF then painting ON over it would be its
			// own divergence even where the resulting pixels agree.
			draw.Draw(dst, r, pic[state], pic[state].Bounds().Min, draw.Src)
		} else {
			drawTownShellBox(dst, r, false)
		}
		// Captions are separate from the unlabelled plaque art. The model
		// resolves the school's and tavern's install words (TOWN-383/391);
		// this renderer keeps their raw bytes and draws caption and value
		// separately. Source fidelity does not certify the shell's own font
		// metrics or layout. DIV-159 retains the button-action boundary.
		ink := buttonInk(b.Enabled, v.HasHover && v.Hover.In(r))
		down := b.Enabled && v.Press.Kind == TownSurfaceControlButton && v.Press.Index == i
		offset := buttonTextOffset(down)
		drawTownShellText(dst, v.Font, b.Label, townButtonLabelRect(r).Add(offset), ink)
		drawTownShellText(dst, v.Font, b.Value, townButtonValueRect(r).Add(offset), ink)
	}
	DrawTownCharacterRegion(dst, v.Hero)
	// The bottom message no longer suppresses during Statistics (1022 spec
	// B6, merge resolution round 2: master's own !statistics guard here
	// predates B6 and is removed by it, on the same footing as the cells,
	// title and buttons above — ComposeTownSurface's own doc). The rect
	// itself is 1024's own refactor, townSurfaceMessageRect(), unchanged.
	drawTownShellText(dst, v.Font, v.Message, townSurfaceMessageRect(), townShellText)
	if v.BookView != nil {
		drawTownBook(dst, *v.BookView)
	}
	ComposeTipPanel(dst, v.Tip)
	if v.HasHover && !v.SuppressHover {
		if lines, ok := TownCandidateHoverLines(v, v.Hover); ok {
			drawShopHover(dst, v.Font, lines, v.Hover)
		}
	}
	return dst
}

func schoolPanelVisible(v TownSurfaceView) bool {
	return v.SchoolClass >= 0 && v.SchoolClass < 2 &&
		(!v.SchoolColumnSet || v.SchoolColumnFrame == v.SchoolClass*15)
}

// cardBoard is the recoloured board bitmap, or nil when the pane has none.
func (v TownCharacterView) cardBoard() *image.RGBA {
	bg, _ := v.StatsPane.Body.(*image.RGBA)
	if bg == nil || bg.Bounds().Empty() {
		return nil
	}
	return characterCardBackground(bg)
}

// shiftedCardBackground is the board in card canvas coordinates: body moved
// left by dx, the seam's first dx columns completing the right edge.
func shiftedCardBackground(body *image.RGBA, seam image.Image, dx int) *image.RGBA {
	b := body.Bounds()
	if dx <= 0 || dx >= b.Dx() {
		return body
	}
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, image.Rect(0, 0, b.Dx()-dx, b.Dy()), body, image.Pt(b.Min.X+dx, b.Min.Y), draw.Src)
	if seam != nil {
		sb := seam.Bounds()
		draw.Draw(out, image.Rect(b.Dx()-dx, 0, b.Dx(), min(b.Dy(), sb.Dy())), seam, sb.Min, draw.Over)
	}
	return out
}

// statsLayout is shared by card paint and hover geometry.
func (v TownCharacterView) statsLayout() PanelLayout {
	bg, _ := v.StatsPane.Body.(*image.RGBA)
	if bg != nil && v.CardOffset.X > 0 {
		bg = shiftedCardBackground(bg, v.StatsPane.Seam, v.CardOffset.X)
	}
	layout := CompactPanelLayout(bg)
	if v.CardLayout != nil {
		layout = *v.CardLayout
		if layout.Background == nil {
			// PAD IS RE-MEASURED WITH THE BACKGROUND, never carried over.
			// CompactPanelLayout's Pad is compactPanelInset of its own background
			// (panel.go), so a layout built with none carries the (4,3) fallback --
			// which is what a Viewer's default layout is, CompactPanelLayout(nil) at
			// viewer.go's own construction. Substituting the shipped body here
			// without re-measuring put the card 15 pixels higher than the frame's
			// settled interior: the name row landed on textbackr.bmp's top ornament
			// and the first two left-column labels went under rect B's own spellbook
			// bitmap.
			layout.Background = bg
			layout.Pad = compactPanelInset(bg)
			if layout.CompactCard && bg != nil {
				layout.Background = characterCardBackground(bg)
				layout.RightInset = 19
			}
		}
	}
	return layout
}
