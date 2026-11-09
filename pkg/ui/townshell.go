package ui

import (
	"fmt"
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

	// CardFont is the statistics card's own font, font2, where its sixteen
	// two-column rows fit 160x242; Font, font1, stays the chrome's. A nil
	// CardFont falls back to Font.
	CardFont *text.Font

	// CardLayout is the statistics card's own layout, or nil for
	// CompactPanelLayout. The town screens set none and compose the shipped
	// card.
	//
	// A layout with no Background of its own takes the pane's own body, which
	// is what CompactPanelLayout is handed here.
	CardLayout *PanelLayout
	// CardShadow, when set, is the statistics card's text shadow.
	CardShadow *TextShadow

	FigurePane TownPane
	StatsPane  TownPane

	// Session is `campaign+0x3dc`, the bitmask `TOWN-346` gates all six corner
	// rectangles on. The ZERO VALUE IS NOT A SHIPPED VALUE: the paint routine's
	// own first act refuses a mask with no bit of `0x627` set. paneSession
	// reads a zero as CharacterPaneShop, so a hand-built fixture keeps its
	// party picker.
	Session CharacterPaneSession

	ScreenH int

	// ModeFlag IS TOWN-346's OWN `flag`, PRECOMPUTED, not derived here from
	// PaneRect and ScreenH. The mission composes this pane on a canvas of its
	// own (PaneRect is (0,0)-(160,242), canvas-relative) and hit-tests it in
	// window pixels (the pane's real screen rectangle) — two different
	// `pane.Max.Y` values, and `characterPaneModeFlag` moves its own
	// threshold with the one it is handed. Every town and shop screen composes this pane
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
	Masks      [2]*image.Paletted
	Skills     [2][5][3]image.Image

	// Scene is the school centre's pictures by the town description's entry
	// names: the training column, the diamond and the training movies. The
	// game's room page draws them.
	Scene map[string][]image.Image

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

	// Scene is the tavern centre's pictures by the town description's entry
	// names. The game's room page draws them; a missing entry leaves only that
	// motion absent.
	Scene map[string][]image.Image
	// Picture answers the page art of a name, nil for none.
	Picture func(name string) image.Image
}

// RoomScene paints a page or a named layer group and answers its controls.
type RoomScene interface {
	Size() image.Point
	Paint(dst *image.RGBA, groups ...string)
	ControlAt(p image.Point) (TownSquareControl, bool)
	TipAt(p image.Point) (int, bool)
}

// TownRoomView is one town frame, including its room page and shared overlays.
type TownSurfaceView = TownRoomView
type TownSquareView = TownRoomView
type TownRoomView struct {
	Pane          bool
	Dolls         []TownSquareDoll
	SuppressHover bool // App paints the shared delayed overlay after widening.
	Kind          TownSurfaceKind
	Title         string
	Cells         []TownSurfaceCell
	Buttons       []TownSurfaceButton
	// Press is the control currently pressed AND under the cursor; it draws
	// the button's own on-bitmap instead of its off-bitmap (1017).
	Press TownSurfaceControl
	Hero  TownCharacterView
	// HeroSlotMask and HeroSlotInfo are the shown member's doll slot map and
	// the item information lines of each worn slot, the same lines the shop
	// shows over the same doll. They are read only for hover.
	HeroSlotMask *SlotMask
	HeroSlotInfo [12][]string
	Message      string
	Font         *text.Font
	// CardFont is the compact 48x64 tavern card's text font. The room chrome
	// keeps Font; the card uses the already-loaded smaller production font so
	// the complete whole-squad price remains inside its shipped bounds.
	// A nil CardFont falls back to Font for synthetic and degraded callers.
	CardFont   *text.Font
	HiredLabel string
	BookView   *ShopScreenView
	// ButtonStyle is the game's room button captions; nil keeps the first
	// game's own.
	ButtonStyle *RoomButtonStyle

	// SchoolArt is optional so hand-built front ends and missing cosmetic data
	// keep a total text fallback. SchoolClass is fighter=0, mage=1; -1 means
	// that no selected member can own a school panel.
	SchoolArt   *TownSchoolArt
	SchoolClass int
	// SchoolColumnSet reports that the scene draws the column at
	// SchoolColumnFrame, including the intermediate frames with no skill
	// panel. Zero preserves static callers.
	SchoolColumnFrame int
	SchoolColumnSet   bool
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
	// Scene is the room's composed centre; nil draws none.
	Scene RoomScene
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
	// Page is the tavern's seats and selected person as the game built
	// them from the room's data; the cells fill each seat's frames and
	// captions.
	Page              RoomPage
	CandidateSlotInfo [12][]string
	CandidatePixels   *image.RGBA
	// CandidateText is the glyphs CandidatePixels was drawn with (text.Record),
	// re-captured each frame the cached picture is pasted.
	CandidateText []text.DrawCall
	Hover         image.Point
	HasHover      bool

	// Tip is the room's tip panel; its rectangle comes from the room's town
	// description.
	Tip TipPanelView
}

// TownCandidateHoverLines returns the selected candidate's normal worn-item
// tooltip at p. The answer is read-only presentation data: this helper does
// not expose a control, item id, or mutation target to the input layer.
func TownCandidateHoverLines(v TownSurfaceView, p image.Point) ([]string, bool) {
	return v.candidateDollHover().LinesAt(p)
}

// candidateDollHover describes the figure the tavern shows of the selected
// candidate: the page's figure rectangle, the figure placed in it as the page
// draws it, and the candidate's worn item text. A view that is not the
// tavern, or has no candidate or no slot map, describes no doll.
func (v TownRoomView) candidateDollHover() DollHover {
	if v.Kind != TownSurfaceTavern || !v.Candidate.HasSubject || v.CandidateSlotMask == nil {
		return DollHover{}
	}
	f, m := v.Page.Person.Figure, v.CandidateSlotMask
	return DollHover{At: f.Origin(image.Pt(m.W, m.H)), Clip: f.Rect, Mask: m, Info: v.CandidateSlotInfo,
		Tip: DollTip{Kind: tooltipText, Beside: true, Key: fmt.Sprintf("candidate-item/%d", v.Candidate.Subject.ID)}}
}

// heroDollHover describes the shown member's figure on the shared character
// pane, or no doll while the pane shows statistics, has no member or no slot
// map.
func (v TownRoomView) heroDollHover() DollHover {
	return townHeroDollHover(v.Hero, v.HeroSlotMask, v.HeroSlotInfo)
}

func townHeroDollHover(hero TownCharacterView, mask *SlotMask, info [12][]string) DollHover {
	if !hero.HasSubject || hero.Statistics || mask == nil {
		return DollHover{}
	}
	d := characterPaneDollHover(hero, mask, info)
	d.Tip = DollTip{Kind: tooltipText, Key: fmt.Sprintf("hero-item/%d/%d", hero.Subject.ID, hero.Member)}
	return d
}

// characterPaneDollHover describes the figure the shared character pane
// paints, over mask and with each slot's item text: the 160x240 source at the
// pane's figure origin, without centring, answering inside the painted figure
// except where the Book corner and the bottom controls stand. Their
// rectangles are the ones a click on the pane reaches, and the mode corner
// still shares its figure pixels through the gesture split (DIV-308).
func characterPaneDollHover(pane TownCharacterView, mask *SlotMask, info [12][]string) DollHover {
	r := pane.paneRect()
	d := DollHover{At: characterPaneFigureOrigin(r), Clip: characterPaneFigureRect(r), Mask: mask, Info: info}
	if CharacterPaneCornerLive(characterPaneModeFlag(r, CharacterPaneShop, 0), CharacterPaneShop, CharacterPaneBook) {
		d.Holes = append(d.Holes, CharacterPaneCornerRect(r, CharacterPaneBook))
	}
	for _, c := range [...]CharacterPaneCorner{CharacterPaneBackpack, CharacterPanePrev, CharacterPaneNext, CharacterPaneMenu} {
		d.Holes = append(d.Holes, CharacterPaneCornerRect(r, c))
	}
	return d
}

// TownHeroHoverLines returns the item information of the worn slot whose
// doll pixel lies under p on the shared character pane, or false while the
// pane shows statistics, over a corner control or over a blank pixel.
func TownHeroHoverLines(v TownSurfaceView, p image.Point) ([]string, bool) {
	return v.heroDollHover().LinesAt(p)
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

// townSurfaceMessageRect is the room's status-message band, drawn in every
// mode.
func townSurfaceMessageRect() image.Rectangle {
	return image.Rect(12, 448, 468, 478)
}

// SchoolFaceOrigin is where the training column's own rotation frame sits in
// the 480x480 room. It is measured, not authored: cmd/schoolcheck correlates
// the shipped frame against the shipped background over the whole background
// and reports the winning offset with the best fraction anywhere else.
var SchoolFaceOrigin = image.Pt(168, 176)

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

// characterPaneCornerControl is the town pane corner a click at p reaches
// first among the picker arrows, the mode corner and the book, and whether it
// acts: an arrow over two members or more, the mode corner over a subject.
func characterPaneCornerControl(hero TownCharacterView, p image.Point) (CharacterPaneCorner, bool, bool) {
	for _, c := range CharacterPaneCornerAt(hero, p) {
		switch c {
		case CharacterPanePrev, CharacterPaneNext:
			return c, true, hero.MemberCount > 1
		case CharacterPaneMode:
			return c, true, hero.HasSubject
		case CharacterPaneBook:
			return c, true, true
		}
	}
	return 0, false, false
}

// TownSurfaceControlAt resolves the extension and character regions before
// left content, including the school's sixteen-pixel overlap.
func TownSurfaceControlAt(v TownSurfaceView, p image.Point) (TownSurfaceControl, bool) {
	return v.SurfaceControlAt(p)
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
// pane-body pixel comparison without copying these rects.
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
		// The union of the hit and art rectangles: four of the six blit
		// somewhere other than where they answer (characterPaneArtRect).
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

// DrawTownCaption draws s centred in r in the town shell's caption colour,
// cut to fit r.
func DrawTownCaption(dst *image.RGBA, font *text.Font, s string, r image.Rectangle) {
	drawTownShellText(dst, font, s, r, townShellText)
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
	drawFrame(dst, panelFrame(r, c, townShellBorder))
}

// drawTownShellOutline is drawTownShellBox without its opaque fill, so a
// control box over shipped pane art leaves the art visible; the outline's
// colour still shows the selection.
func drawTownShellOutline(dst *image.RGBA, r image.Rectangle, selected bool) {
	c := townShellBorder
	if selected {
		c = townShellSelected
	}
	drawFrame(dst, frameSpec{Kind: frameOutline, Rect: r, Border: c})
}

// DrawTownCharacterRegion paints the shared member projection and its
// persistent controls. The pane's shipped body and seam draw first; a pane
// with no Body falls back to the authored fill and outline. In Statistics
// mode the card replaces the doll here, over the pane's own body (DIV-175),
// for every caller, the shop included: no drag surface answers under it.
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
		drawFrame(dst, panelFrame(r, townShellPanel, townShellBorder))
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
// A CORNER WITH NO BITMAP FALLS BACK TO THE AUTHORED CHROME, the two picker
// chevrons and an outline, so a control still shows where one answers.
// drawShopChevron's `fill` argument keeps the plate the card needs.
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

// ComposeTownSurface paints one complete shared-shell room. Optional art is
// absent by construction here; the fallback furniture and every live control
// remain visible and hit-testable.
func ComposeTownSurface(v TownSurfaceView) *image.RGBA {
	return ComposeTownRoom(v)
}

// wholePair is an off/on pair when both pictures are present, else none.
func wholePair(pair [2]image.Image) [2]image.Image {
	if pair[0] == nil || pair[1] == nil {
		return [2]image.Image{}
	}
	return pair
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

// CardReport is the statistics card's laid-out rows in card canvas pixels.
func (v TownCharacterView) CardReport() []PanelLineReport {
	return CharacterPanelReport(v.statsLayout(), v.cardFont(), v.Subject)
}

// CardRect is where the statistics card's canvas stands on the screen.
func (v TownCharacterView) CardRect() image.Rectangle { return v.cardRect() }

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
	if v.CardShadow != nil {
		layout.Shadow = v.CardShadow
	}
	return layout
}
