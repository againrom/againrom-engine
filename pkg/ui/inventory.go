package ui

import (
	"image"
	"image/color"
	"strconv"

	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
)

// The inventory: what one character's figure, equipment and carried pack
// look like as pictures, and the open/closed state a binding drives.
//
// IT IS THREE BOXES SINCE 0140, NOT ONE WINDOW, and the three were arrived
// at in two rulings on the same day. The first replaced the centred window
// with the arrangement the owner's own install draws: the worn set and the
// doll together in the window's bottom left, symmetric with the unit panel
// opposite them, and the carried pack a BAR along the bottom that scrolls
// with arrows at its right end. The second SPLIT the pair — "now as for
// the doll and the equipment screen, we separate them: the doll goes on the
// RIGHT above the character's data, the worn things on the LEFT above the
// spellbook" — so what was one window is now the doll box (dollPresent),
// the worn box (wornPresent) and the pack bar (packBarPresent), placed by
// hud.go's shared arithmetic.
//
// ALL THREE ARE SWITCHES NOW, AND NONE OF THEM IS A WINDOW. Until the same
// day's third ruling the doll box was a window with a life of its own — binding
// I opened it only over its own single selected character, and it closed itself
// when the selection moved off. The owner replaced that with four display
// settings under the map (hudtoggles.go), which is a different kind of thing
// entirely: the setting says whether the player wants the box, the SELECTION
// says whether there is anything to put in it, and each present function below
// asks both. The eligibility half survives that change unaltered — it is the
// same inventoryEligible, in the same place in the same order — and only the
// toggle half moved.
//
// THE DOLL IS THE ONE BOX THAT FOLLOWS ANY SELECTED UNIT, and it is the only
// place in this file where the subject and the selection can name two different
// entities (owner: "show the doll of the character AND of any enemy, they have
// a doll picture too"). What it can draw for one is dollSubject's own question,
// and the honest answer today is narrower than the ruling — see its doc.
//
// THE BOUNDARY IS THE DRAWING TIER'S OWN (spec "the drawing tier"; plan
// D-6). This file receives pictures, an entity id and a set of occupied
// slots, and composes them into images; it opens nothing, decodes nothing,
// and holds no archive, definition table or equipment slot. It names no
// pkg/data or pkg/formats type anywhere below.
//
// STATE LIVES ON Viewer, beside the panel's and the notice's own (plan D-8:
// "the subject reaches the viewer through a setter at mission open"). A
// mission's own open door hands the front-end nothing but *ui.Viewer and four
// seams — pkg/game/frontend.go's MissionOpener returns *ui.Viewer as its
// first result and has no *ui.App in scope at all; the App itself is built
// once, by FrontEnd.App(title), and never handed back to that path — so a
// setter reachable at mission open can only be a Viewer method, exactly as
// SetFont and SetLocalOwner already are. The fields are declared in
// viewer.go, beside panelPic/panelFresh/panelImg, following that file's own
// convention of collecting every Viewer field in one place; every method
// below takes *Viewer as its receiver.
//
// THE TOGGLE NEVER TOUCHES THE COMMAND PATH. Eligibility is read through
// presentSelected — command.go's own filter for "selected and still in the
// snapshot", called and not reimplemented — and nothing here writes v.sel,
// issues an order or moves the camera (fence: "Do not let the binding reach
// the command path, the selection or the camera").

// InventorySubject is everything the inventory draws about one character:
// which entity it belongs to, its already-composed figure (base sheet with
// every occupied slot's layer already painted on — plan D-7 puts that painting
// in the wiring tier, never here), the icon for each of the twelve equipment
// slots, in slot order, and the carried pack's own pictures.
//
// EVERY PICTURE CROSSES AS A PLAIN *image.RGBA, the same seam type the
// panel's face and the notice's portrait already use — never an archive
// address, so a nil picture here is an UNREAD ADDRESS ARRIVING, not a bare
// pointer this file invented. Every reader below tests a picture for nil
// before it touches its bounds, which is what makes the composition TOTAL
// over the whole of what a subject can carry: no code, no equipment and no
// absent art makes composing either box fail.
//
// SLOTS IS A FIXED-LENGTH ARRAY OF TWELVE, and not a slice sized by a
// constant beside it (plan D-13). This package cannot read the slot count out
// of pkg/data — the allow-map forbids the import — so the array's own length
// IS the count, and nothing below repeats the number: every walk ranges over
// Slots or takes len(Slots), and a caller that built a different length would
// not compile against this type at all.
//
// PACK IS ONE PICTURE PER CARRIED ELEMENT, IN THE CARRIER'S OWN ORDER —
// the wiring tier's own composition of an entity's container, not this
// tier's business to interpret. A nil entry is an empty cell exactly as a
// nil Slots entry is: the same reader below draws both alike.
//
// IT IS A SLICE, AND IT WAS AN ARRAY OF EIGHT UNTIL 0140. The window used to
// hold a fixed cell count, so a carrier holding more than eight elements
// simply had the remainder undrawn — a limit of the WINDOW, disclosed as
// such (0112 spec D-3). The owner's bar scrolls instead ("potentially
// endlessly"), which removes the reason for a cap: what pkg/sim's container
// has always been (UNBOUNDED) is now what this type carries, and how many
// cells the screen has room for is decided by the BAR, per window size, in
// hud.go. The array also existed so InventorySubject would stay comparable
// — it was held by value as a cache key — and that is why the doll box's
// cache is now keyed by a revision counter instead (viewer.go's invRev).
//
// PackCount IS THE PACK'S OWN COUNT, ONE ENTRY PER ELEMENT AND SHAPED
// EXACTLY LIKE Pack. A PackCount shorter than Pack reads as no count for the
// cells past its end rather than as a panic, which is what keeps every
// hand-built fixture in this package composing.
//
// IT IS THE SAME SIZE AS THE FIGURE IT DESCRIBES, W by H, and it is built
// BESIDE the composition in pkg/game (composeUnitFigure, composeInventorySubject)
// rather than by reading the finished picture back: the compositor already
// knows, at the moment it paints a layer, which slot that layer belongs to,
// and that is the one fact a picture's own pixels cannot recover once every
// layer has been flattened into it.
//
// Slot IS ONE BYTE PER PIXEL, row-major, index y*W+x — image.RGBA's own
// Pix indexing with one byte instead of four, because a slot number is
// 0..12 and needs no wider a field.
type SlotMask struct {
	W, H int
	Slot []uint8
}

// At is the equipment slot owning mask pixel (x,y), and whether one does —
// false for a coordinate outside the mask and for a pixel no layer painted,
// which a caller reads alike as "no slot here" (a nil mask answers the same
// way, so a subject built with no mask at all is not a special case for a
// reader of this method).
func (m *SlotMask) At(x, y int) (int, bool) {
	if m == nil || x < 0 || y < 0 || x >= m.W || y >= m.H {
		return 0, false
	}
	n := m.Slot[y*m.W+x]
	if n == 0 {
		return 0, false
	}
	return int(n), true
}

type InventorySubject struct {
	ID       uint32
	Figure   *image.RGBA
	SlotMask *SlotMask
	// HoverSlotMask has SlotMask's geometry and final visible ownership, plus
	// one authored-layer pixel retained for every worn layer that later
	// equipment fully covers. Read-only inspection uses it so every item worn
	// on the composed doll can expose its normal tooltip. Interactive dolls
	// continue to use SlotMask's strict topmost-pixel ownership.
	HoverSlotMask *SlotMask
	Slots         [12]*image.RGBA
	SlotInfo      [12][]string
	Pack          []*image.RGBA
	PackCount     []uint32
	// PackStars is the normal compact-record display bit for each carried
	// element, already resolved by the tier that owns ItemInstance. It is
	// parallel to Pack and deliberately has no worn-slot counterpart:
	// ITEM-STARSURF-097 finds the mission grid call only on carried cells;
	// worn equipment and either held cursor draw the base picture alone.
	PackStars []bool
	// PackPurse marks the purse element, parallel to Pack. Its quantity is
	// the one pack number the grouping routine formats (TOWN-469); item-stack
	// counts stay ungrouped.
	PackPurse []bool
	PackInfo  [][]string

	// WeaponFallback is true when slot 1's picture, popup and icon are all
	// drawn from the subject's own starting-weapon fallback rather than a real
	// equipped code (counterexample 4, round-2 adversarial review, third pass,
	// restating pkg/game's currentFigureEquipment and buildInventorySubject's
	// own shared condition here for command.go to read at press time). Nothing
	// in the live entity backs that slot in this state — no equipment slot
	// and no container element — so sim.unequip and dropFromEquipment both
	// refuse it outright (item == 0). The doll still draws it, still names it
	// in the hover popup (informative and non-mutating), but does not arm a
	// drag or a tap-to-unequip for it, on wornBox's own "not every drawn
	// surface is a drag source" precedent restated for one slot rather than a
	// whole box.
	WeaponFallback bool
}

// The doll box's own authored geometry.
const (
	// invCellSize is one equipment slot's own side, in pixels, and invCellGap
	// the gap between adjacent cells on both axes — this tier's own choice
	// (plan D-12), and not derived from an icon's decoded size: whatever
	// picture a cell is handed is centred inside it and clipped, never scaled,
	// so no format fact about icon dimensions is assumed here.
	invCellSize = 48
	invCellGap  = 4

	// invSlotColumns is how many of the twelve slot cells sit in one row before
	// the layout wraps. The contract states only that there are twelve, in slot
	// order; the wrap width is ours.
	invSlotColumns = 3

	// invFigureW, invFigureH are the figure's own authored box. The figure is
	// centred and clipped inside it exactly as a slot's icon is inside its cell
	// — never scaled — so a nil Figure still leaves a well-formed,
	// deterministically sized box.
	invFigureW, invFigureH = 160, 240

	invPad = 10
)

// The furniture's own authored palette: a frame in the drawing tier's
// existing colours, reused rather than a second one invented here —
// panel.go's fillPanelFrame paints each outer box and drawInventoryCell
// below paints every cell of all three, so the doll box, the pack bar and
// the spellbook bar read as one family.
var (
	invFill       = color.RGBA{R: 0x10, G: 0x12, B: 0x18, A: 0xff}
	invBorder     = color.RGBA{R: 0x8a, G: 0x74, B: 0x46, A: 0xff}
	invCellFill   = color.RGBA{R: 0x1c, G: 0x1f, B: 0x28, A: 0xff}
	invCellBorder = color.RGBA{R: 0x8a, G: 0x74, B: 0x46, A: 0xff}

	// The pack bar's scroll buttons (0140). A live arrow is the frame's own
	// gold, a spent one — the bar already at one end of what is carried — is
	// drawn dim rather than hidden: a button that vanishes moves the button
	// beside it, and the player aims at where it was.
	hudArrowColor = color.RGBA{R: 0xf2, G: 0xe6, B: 0xc4, A: 0xff}
	hudArrowSpent = color.RGBA{R: 0x4a, G: 0x44, B: 0x38, A: 0xff}
)

const invCountPad = 2

// wornSlotRects is the twelve slot cells in SLOT ORDER, in the WORN BOX's own
// pixels, wrapped at invSlotColumns and CENTRED across the box.
func wornSlotRects() [12]image.Rectangle {
	var slots [12]image.Rectangle
	gridW := invSlotColumns*(invCellSize+invCellGap) - invCellGap
	x0 := (sidebarWidth - gridW) / 2
	for i := range slots {
		col, row := i%invSlotColumns, i/invSlotColumns
		x := x0 + col*(invCellSize+invCellGap)
		y := invPad + row*(invCellSize+invCellGap)
		slots[i] = image.Rect(x, y, x+invCellSize, y+invCellSize)
	}
	return slots
}

// wornBoxSize is the worn box's own size: sidebarWidth across, like every other
// box in the column it stands in, and as tall as the slot grid plus the pad.
func wornBoxSize() image.Point {
	rows := (12 + invSlotColumns - 1) / invSlotColumns
	h := rows*(invCellSize+invCellGap) - invCellGap
	return image.Pt(sidebarWidth, h+2*invPad)
}

// drawInventoryPicture centres pic inside area and clips it there, never
// scaled — drawNoticePortrait's own rule (notice.go), applied to the
// figure, to a slot's icon and to a pack cell's icon alike. A nil picture,
// or one with no area, leaves area exactly as it was painted before this
// call: an unread address draws as an absence, not a failure.
//
// IT COMPOSITES AND DOES NOT COPY, and that is the difference between a
// picture drawn ON the box and a picture drawn THROUGH it — see the per-pixel
// comment below for why the ground survives.
func drawInventoryPicture(dst *image.RGBA, area image.Rectangle, pic *image.RGBA) {
	drawInventoryPictureShifted(dst, area, pic, image.Point{})
}

// drawInventoryPictureShifted is drawInventoryPicture with a caller-owned
// presentation offset applied after centring. The tavern's borrowed character
// widget is the one shipped surface that positions its figure a few pixels off
// the generic centred origin; all ordinary inventory callers keep the zero
// offset through drawInventoryPicture above.
func drawInventoryPictureShifted(dst *image.RGBA, area image.Rectangle, pic *image.RGBA, shift image.Point) {
	if pic == nil {
		return
	}
	pb := pic.Bounds()
	if pb.Dx() <= 0 || pb.Dy() <= 0 {
		return
	}
	ox := area.Min.X + (area.Dx()-pb.Dx())/2 + shift.X
	oy := area.Min.Y + (area.Dy()-pb.Dy())/2 + shift.Y
	for y := 0; y < pb.Dy(); y++ {
		dy := oy + y
		if dy < area.Min.Y || dy >= area.Max.Y {
			continue
		}
		for x := 0; x < pb.Dx(); x++ {
			dx := ox + x
			if dx < area.Min.X || dx >= area.Max.X {
				continue
			}
			so := pic.PixOffset(pb.Min.X+x, pb.Min.Y+y)
			a := pic.Pix[so+3]
			if a == 0 {
				continue
			}
			do := dst.PixOffset(dx, dy)
			if a == 0xff {
				copy(dst.Pix[do:do+4], pic.Pix[so:so+4])
				continue
			}
			// SOURCE-OVER ON PREMULTIPLIED CHANNELS, which is what both
			// producers deliver: pkg/game resolves a .16a's coverage into the
			// colour at load and blits a .256 at full opacity, and Go's own
			// RGBA is premultiplied besides. So the composite is the source
			// plus the destination scaled by what the source did not cover,
			// with no divide and no unpremultiply anywhere.
			inv := uint32(0xff - a)
			for i := 0; i < 4; i++ {
				dst.Pix[do+i] = uint8(uint32(pic.Pix[so+i]) + uint32(dst.Pix[do+i])*inv/0xff)
			}
		}
	}
}

// drawInventoryCell paints one cell's own ground — a fill and a one-pixel
// border — and, when pic is non-nil, the picture centred inside it.
//
// THE BORDER COLOUR IS AN ARGUMENT SINCE 0140, because the spellbook bar picks
// its selected cell out by drawing that cell's border in the strip's own
// highlight (spellbook.go) — one cell painter for all three boxes, rather than
// a second one differing in a single colour.
func drawInventoryCell(dst *image.RGBA, box image.Rectangle, pic *image.RGBA, border color.RGBA) {
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			c := invCellFill
			if x == box.Min.X || y == box.Min.Y || x == box.Max.X-1 || y == box.Max.Y-1 {
				c = border
			}
			dst.SetRGBA(x, y, c)
		}
	}
	drawInventoryPicture(dst, box.Inset(1), pic)
}

// drawInventoryPackCount keeps the small gold quantity inside the lower-left
// corner. The subimage clips even a long quantity to its own inventory cell.
func drawInventoryPackCount(dst *image.RGBA, box image.Rectangle, count uint32, purse bool, f *text.Font) {
	if f == nil || count < 2 {
		return
	}
	s := strconv.FormatUint(uint64(count), 10)
	if purse {
		s = GroupDigits(int64(count))
	}
	w, h := f.Measure(s)
	if w <= 0 || h <= 0 {
		return
	}
	x := box.Min.X + invCountPad
	y := box.Max.Y - invCountPad - h
	clipped := dst.SubImage(box.Intersect(dst.Bounds())).(*image.RGBA)
	f.DrawFlat(clipped, s, x+shopPriceShadow, y+shopPriceShadow, messageShadowColor)
	f.Draw(clipped, s, x, y, shopPriceInk)
}

// drawHudArrow is the scroll cue for viewers without installed HUD art.
func drawHudArrow(dst *image.RGBA, box image.Rectangle, forward, live bool) {
	if box.Empty() {
		return
	}
	drawInventoryCell(dst, box, nil, invCellBorder)
	c := hudArrowColor
	if !live {
		c = hudArrowSpent
	}
	in := box.Inset(5)
	if in.Dx() <= 0 || in.Dy() <= 0 {
		return
	}
	// The triangle's own half-height at the tip is 0 and at the base is
	// in.Dy()/2, so the drawn rows narrow one step per column — an integer
	// walk, so the shape is the same on every machine.
	mid := (in.Min.Y + in.Max.Y) / 2
	for x := in.Min.X; x < in.Max.X; x++ {
		d := x - in.Min.X
		if forward {
			d = in.Max.X - 1 - x
		}
		half := (in.Dy() * d) / (2 * in.Dx())
		for y := mid - half; y <= mid+half; y++ {
			if y < in.Min.Y || y >= in.Max.Y {
				continue
			}
			dst.SetRGBA(x, y, c)
		}
	}
}

// packScrollCueRect keeps the pre-existing authored triangle small while the
// whole decoded end strip remains clickable.
func packScrollCueRect(strip image.Rectangle) image.Rectangle {
	w, h := packScrollCueW, packScrollCueH
	if w > strip.Dx() {
		w = strip.Dx()
	}
	if h > strip.Dy() {
		h = strip.Dy()
	}
	if w <= 0 || h <= 0 {
		return image.Rectangle{}
	}
	x := strip.Min.X + (strip.Dx()-w)/2
	y := strip.Min.Y + (strip.Dy()-h)/2
	return image.Rect(x, y, x+w, y+h)
}

// RenderWorn composes one character's worn set — the twelve equipment
// slots in slot order — as one picture and returns it.
//
// It is a function of its argument alone, no font, and TOTAL — the zero
// InventorySubject, naming no entity and no picture anywhere, still composes
// the frame and the twelve empty cells.
func RenderWorn(s InventorySubject) *image.RGBA {
	size := wornBoxSize()
	img := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	fillPanelFrame(img, size, invFill, invBorder)

	for i, box := range wornSlotRects() {
		drawInventoryCell(img, box, s.Slots[i], invCellBorder)
	}
	return img
}

// renderPackBar composes the carried pack grid, its visible cells and the two
// scroll cues.
//
// The returned layer spans the full frame width, not only the inventory
// rectangle. ITEM-STARCOMP-100's 6..74 trail footprint remains inside each
// 80x80 icon vertically; the full-width layer therefore leaves the final
// frame as the only effective clip and removes the old compact-bar clip.
//
// A CELL PAST THE END OF THE PACK IS AN EMPTY CELL, not an absent one: the bar
// is drawn full width whatever is carried (hud.go's packBarRect), so scrolling
// to the end shows empty cells rather than a shrinking bar.
func renderPackBar(s InventorySubject, scroll, cols int, bar image.Rectangle, f *text.Font) *image.RGBA {
	return renderPackBarStars(s, scroll, cols, bar, f, nil, -1)
}

// renderPackBarStars is renderPackBar with the mission grid's visible-slot
// phases. omit is the selected singleton's container index, or -1. The held
// cursor is painted elsewhere from the unmarked base icon, while a remaining
// stack stays in this grid and continues its phase (ITEM-STARPHASE-099).
func renderPackBarStars(s InventorySubject, scroll, cols int, bar image.Rectangle, f *text.Font, phases []uint32, omit int) *image.RGBA {
	return renderPackBarArt(s, scroll, cols, bar, f, phases, omit, nil)
}

func renderPackBarArt(s InventorySubject, scroll, cols int, bar image.Rectangle, f *text.Font, phases []uint32, omit int, art *BottomHUDArt) *image.RGBA {
	size := image.Pt(bar.Dx()+sidebarWidth, bar.Dy())
	img := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	if art == nil {
		fillPanelFrame(img, bar.Size(), invFill, invBorder)
	} else {
		drawPackGround(img, bar, cols, art)
	}
	layerAt := image.Pt(0, bar.Min.Y)

	for i, box := range packCellRects(bar, cols) {
		box = box.Sub(layerAt)
		idx := scroll + i
		var pic *image.RGBA
		if idx >= 0 && idx < len(s.Pack) && idx != omit {
			pic = s.Pack[idx]
		}
		// Paint the ground first, then the base icon in the complete 80x80
		// rectangle. The previous box.Inset(1) path cut an 80x80 installed
		// icon down to 78x78.
		if art == nil {
			drawInventoryCell(img, box, nil, invCellBorder)
		} else if pic != nil {
			blit(img, art.PackItem, box.Min.X, box.Min.Y)
		}
		drawInventoryPicture(img, box, pic)
		if pic != nil && idx >= 0 && idx < len(s.PackCount) {
			drawInventoryPackCount(img, box, s.PackCount[idx], idx < len(s.PackPurse) && s.PackPurse[idx], f)
		}
		// Mission order is base, quantity, trail. The trail receives the
		// decoded icon origin and the full-width layer carries it to the
		// final-frame clip.
		if pic != nil && i < len(phases) && idx >= 0 && idx < len(s.PackStars) && s.PackStars[idx] {
			drawItemStarTrail(img, box.Min, phases[i], &itemStarFields[0])
		}
	}

	back, fwd := packScrollRects(bar, cols)
	if art == nil {
		drawHudArrow(img, packScrollCueRect(back.Sub(layerAt)), false, scroll > 0)
		drawHudArrow(img, packScrollCueRect(fwd.Sub(layerAt)), true, scroll < packMaxScroll(len(s.Pack), cols))
	} else {
		if scroll > 0 {
			blit(img, art.PackArrow[0], back.Max.X-32, packArrowDY)
		}
		if scroll < packMaxScroll(len(s.Pack), cols) {
			blit(img, art.PackArrow[1], fwd.Min.X, packArrowDY)
		}
	}
	return img
}

const packArrowDY = 2

// packMaxScroll is the furthest the bar may be scrolled: enough that the last
// carried element is on screen, and no further. A pack that fits entirely
// answers 0, so the forward button is spent from the first frame rather than
// scrolling into empty cells.
func packMaxScroll(n, cols int) int {
	if n <= cols {
		return 0
	}
	return n - cols
}

// SetInventorySubject hands the viewer the one character the inventory can
// currently show, replacing whatever it held before.
//
// IT IS THE MISSION-LIFETIME SETTER D-8 ASKS FOR, called when a mission's
// party is assembled and again whenever the wiring tier finds the subject's
// pack or equipment changed (pkg/game's refreshPack, refreshEquipment) —
// never on the per-tick entity seam for its own sake. It rides the same door
// SetFont and SetLocalOwner do: mv.Viewer.SetFont(...) at mission open
// (pkg/game/frontend.go), not a call through App.
//
// IT DOES NOT OPEN THE DOLL BOX. Opening is the binding's own effect, gated
// on the selection at the moment of a press; a setter that opened it would
// let a mission's own loading sequence show a box the player never asked
// for. It BUMPS THE REVISION, which forces the next presentation to
// recompose: a second call with a different picture at the same entity id is
// not mistaken for an unchanged subject.
func (v *Viewer) SetInventorySubject(s InventorySubject) {
	v.invSubject, v.invHasSubject = s, true
	v.invRev++
	v.invKey, v.invPic, v.invFresh = 0, nil, false
	v.syncMapViewport()
}

// THE VIEWER HOLDS NO SUBJECT AT EVERY MISSION OPEN, with NO STATEMENT OF ITS
// OWN TO SAY SO. Unlike App, which persists across missions and would need an
// explicit reset called at every OpenMission and stepPicker's choose(), Viewer
// does not: flow.enter (flow.go) replaces f.viewer WHOLE with the brand new
// *Viewer the mission's own loader just built, never mutates the one already
// there, and false and the zero InventorySubject are exactly what that fresh
// struct starts with (viewer.go's own field comment). A stale subject from the
// mission just left is therefore never reachable at all: it lived on an object
// flow.enter has already discarded, not on one this file reads from.
//
// THE FOUR SWITCHES SURVIVE A MISSION CHANGE FOR THE SAME REASON, AND THAT
// IS A LOSS THIS STORY ACCEPTS. Moving the four flags to App would keep them
// instead, and that is the wrong trade for now: App is what the menu and the
// picker run on, and a map-screen setting kept there is a second lifetime to
// reason about for a preference the player can restore with one key. It is
// written down here rather than left to be discovered.

// inventoryEligible reports whether the viewer's current selection is
// EXACTLY the one character its own subject belongs to: the present
// selection — presentSelected, command.go's own filter for "selected and
// still in the snapshot" — holds exactly one entity, and that entity's id
// is the subject's own.
//
// A viewer holding no subject at all answers false, which is what keeps a
// press made before any mission has assembled a party from opening a box over
// nothing. There is no separate nil-viewer branch: both callers below reach
// this method through a v already known non-nil — app.go's map arm
// dereferences a.flow.viewer before in.Inventory can be read at all — the
// same guarantee flow.noticeOpen() already gives NoticeOpen.
func (v *Viewer) inventoryEligible() bool {
	if !v.invHasSubject {
		return false
	}
	present := presentSelected(v.sel, v.entities)
	return len(present) == 1 && present[0].ID == v.invSubject.ID
}

// dollSource is WHICH unit the doll box is drawing and WHICH of the two
// pictures this build could find for it — the doll's own rebuild key
// (viewer.go's dollKey).
//
// IT HOLDS POINTERS AND COMPARES THEM, never the pixels behind them. Both
// producers hand out a stable pointer for as long as the picture is unchanged —
// a composed figure lives on the InventorySubject until a new subject is pushed,
// and an entity's frame is the sheet's own frame, swapped rather than mutated as
// the unit walks — so pointer equality answers "the same picture" exactly, at
// three words per frame, and never rebuilds a doll that has not changed.
//
// IT IS COMPARABLE, and it must be: it is compared with == in dollPresent.
// Every field is a pointer or a scalar, which is what keeps it so.
// suppressSlot IS PART OF THE KEY (1005, "the interactive doll"): dragging
// an item off the doll shows the figure composed with that one slot cleared
// (SetDollSuppressedFigure, pushed by pkg/game's refreshDollDrag) FOR THE
// SAME ENTITY AND THE SAME UNDERLYING figure POINTER — the subject has not
// changed, only which equipment set the picture was composed from has — so
// the picture pointer alone cannot tell the two compositions apart and a
// drag that started or ended on this exact frame would present a stale
// texture without this field forcing dollPresent's own rebuild.
type dollSource struct {
	id           uint32
	figure       *image.RGBA
	portrait     *image.RGBA
	frame        *terrain.StaticFrame
	suppressSlot int
}

// dollSubject is the unit the doll box draws this frame and the picture it
// draws for it, or false for a frame that draws none.
//
// THERE ARE THREE PICTURES AND THEY ARE ASKED IN THIS ORDER, which is the whole
// of what 0141 changed here. 0140 shipped two of them and disclosed the gap
// between: a COMPOSED FIGURE for the party's own character, and, for anything
// else, that unit's own drawn world sprite — a placeholder that read as one,
// because the four figure directories this project could name were the four
// human ones and nothing published said where a monster's picture lived.
//
// It now does, and the answer is that a monster HAS NO DOLL. Whether an actor
// shows a composed figure or a flat picture is one test on its class
// (`UNIT-PICT-035`), the two are exclusive, and the flat side is a 24-bit
// bitmap under `graphics\infowindow` at the same 160x240 an equipment sheet is
// (`UNIT-PICT-036`, `SPR256-PICT-043`). The wiring tier resolves that test and
// pushes the picture (SetUnitPortrait); this file never learns a class.
//
// THE WORLD SPRITE SURVIVES AS THE THIRD ARM AND IS NO LONGER A SUBSTITUTION.
// It is reached only where the tier above pushed nothing — a class this bundle
// does not know, a picture the install does not carry, thirteen structure
// sections on the RU root naming a node `GRAPHICS.RES` has no entry for
// (`REG-PICT-083`) — so it is now what a MISSING file falls back to rather than
// what a missing decode did.
// A DRAG LIFTING ONE SLOT OFF THE DOLL SUBSTITUTES THE SUPPRESSED FIGURE
// RIGHT HERE (1005, "the interactive doll"), rather than teaching
// dollPicture or RenderDoll a layer to skip (`DLG-FIGURE-021`'s own
// constraint, the compositor takes no layer selector): pkg/game's
// refreshDollDrag composes a distinct picture with that slot's occupied bit
// cleared and pushes it through SetDollSuppressedFigure, and this is the one
// place that picture is chosen over the ordinary one — only for the SAME
// entity the suppression was pushed for, and only while dollSuppressPic is
// non-nil, so a drag that has already lifted stays substituted for exactly
// as long as pkg/game keeps pushing it.
func (v *Viewer) dollSubject() (dollSource, bool) {
	present := presentSelected(v.sel, v.entities)
	if len(present) == 0 {
		return dollSource{}, false
	}
	e := present[0]
	if v.invHasSubject && v.invSubject.ID == e.ID && v.invSubject.Figure != nil {
		fig := v.invSubject.Figure
		slot := 0
		if v.dollSuppressOwner == e.ID && v.dollSuppressSlot != 0 && v.dollSuppressPic != nil {
			fig, slot = v.dollSuppressPic, v.dollSuppressSlot
		}
		return dollSource{id: e.ID, figure: fig, suppressSlot: slot}, true
	}
	if v.portraitOwner == e.ID && v.portrait != nil {
		return dollSource{id: e.ID, portrait: v.portrait}, true
	}
	return dollSource{id: e.ID, frame: e.Frame}, true
}

// SetDollSuppressedFigure hands the viewer the doll picture to draw INSTEAD
// of the subject's own composed figure, for as long as a drag lifts one
// slot off it (1005, "the interactive doll") — SetUnitPortrait's own shape
// and its own per-tick push, pushed by pkg/game's refreshDollDrag once per
// frame the suppressed slot changes. owner is which entity this applies to,
// slot is the 1-based equipment slot suppressed and 0 clears the override —
// zero for either argument, or a nil pic, is read as "nothing suppressed"
// by dollSubject above, which is what lets a caller clear the override with
// one statement rather than a second method.
//
// mask MUST be composeInventorySubject's own SlotMask for pic, from the SAME
// call (1005 round-1 adversarial review): dollFigureSlotAt reads it whenever
// the suppression this call sets is the one in force, so a hover or a press
// over the picture actually on screen is answered from that picture's own
// mask and not the subject's ordinary one — the property DIV-085 states
// for the ordinary figure, extended to the suppressed one.
func (v *Viewer) SetDollSuppressedFigure(owner uint32, slot int, pic *image.RGBA, mask *SlotMask) {
	v.dollSuppressOwner, v.dollSuppressSlot, v.dollSuppressPic, v.dollSuppressMask = owner, slot, pic, mask
}

// SetUnitPortrait hands the viewer the FLAT PORTRAIT of one unit, replacing
// whatever it held before — SetSpellbook's own shape and its own frequency:
// pushed once per tick by the tier that can open an archive, for whatever is
// selected at that moment (pkg/game's pushPortrait).
//
// OWNER IS WHICH ENTITY THIS PICTURE BELONGS TO, 0 for none, and it is what
// keeps a stale portrait off a newly selected unit: the box asks for the
// picture of the entity it is drawing, and a push naming a different one — or
// naming none — simply does not answer for it.
//
// A NIL PICTURE IS NOT AN ERROR AND IS THE ORDINARY CASE for every actor that
// composes a figure instead. The tier above pushes nil for those rather than
// pushing both and leaving this file to choose.
func (v *Viewer) SetUnitPortrait(owner uint32, pic *image.RGBA) {
	v.portraitOwner, v.portrait = owner, pic
}

// dollPicture resolves one dollSource to the picture RenderDoll takes: the
// composed figure when there is one, and otherwise the world frame blitted onto
// its own canvas (terrain.StaticFrame.RGBA).
func dollPicture(src dollSource) *image.RGBA {
	if src.figure != nil {
		return src.figure
	}
	if src.portrait != nil {
		return src.portrait
	}
	if src.frame != nil {
		return src.frame.RGBA()
	}
	return nil
}

// dollBox is where the doll box stands in WINDOW PIXELS, and whether one stands
// anywhere at all.
//
// IT ASKS BOTH QUESTIONS AND IN THIS ORDER: the SETTING first (hudPanelDoll,
// hudtoggles.go — cheap, and the answer for most frames a player has switched
// it off on), then whether anything is selected to draw. See this file's header
// for why those are two questions now and were one before.
//
// IT COMPOSES NOTHING.
func (v *Viewer) dollBox() (image.Rectangle, bool) {
	if !v.hudShown(hudPanelDoll) {
		return image.Rectangle{}, false
	}
	if _, ok := v.dollSubject(); !ok {
		return image.Rectangle{}, false
	}
	return v.characterPaneRect()
}

// wornBox is where the worn box stands in WINDOW PIXELS, and whether one stands
// anywhere at all.
//
// IT IS GATED ON ITS OWN SWITCH AND ON inventoryEligible — the setting, then
// "the selection is exactly the one character this build composed a worn set
// for". The second half is narrower than the doll's and deliberately so: this
// box draws ICONS OF WORN PIECES, one per equipment slot, and the only unit
// this build composes those for is the party's own subject (pkg/game's
// composeInventorySubject). A selected enemy has a worn set in the simulation
// and rows for it in the unit panel, but no icons here, so the honest answer is
// no box rather than twelve empty cells claiming he wears nothing.
func (v *Viewer) wornBox() (image.Rectangle, bool) {
	if !v.hudShown(hudPanelWorn) || !v.inventoryEligible() {
		return image.Rectangle{}, false
	}
	return wornBoxRect(image.Pt(v.frameW, v.frameH))
}

// packBar is where the pack bar stands in WINDOW PIXELS and how many cells it
// draws, or false for a viewer that draws none.
//
// A SUBJECT NAMING NO ENTITY IS NO SUBJECT, and inventoryEligible carries that
// by construction: openMission hands this viewer a subject on every map,
// including one opened from the picker with no party at all, where it is the
// zero value (pkg/game/world.go says so in as many words: "a party-less mission
// and a mission whose subject really is entity 0 both leave subject.ID at 0"),
// and no entity carries id 0 for its selection to match.
func (v *Viewer) packBar() (image.Rectangle, int, bool) {
	if !v.inventoryEligible() {
		return image.Rectangle{}, 0, false
	}
	return v.packBarShown()
}

// packBarShown is the pack bar's rectangle for a switched-on pack: for the
// hero's pack, and empty while nothing is selected, so the pack icon and the
// bar agree (DIV-2264). A selection that is not the one hero still draws no
// bar. packBar adds the eligibility its hit tests and scrolling need.
func (v *Viewer) packBarShown() (image.Rectangle, int, bool) {
	if !v.hudShown(hudPanelPack) || (!v.inventoryEligible() && len(presentSelected(v.sel, v.entities)) != 0) {
		return image.Rectangle{}, 0, false
	}
	return packBarRect(image.Pt(v.frameW, v.frameH))
}

// packScrollAt is the scroll position the bar actually draws with: whatever
// the player has scrolled to, pulled back into what the current pack and the
// current window allow.
//
// IT CLAMPS ON READ AND NOT ONLY ON WRITE, because the two things it is
// bounded by can both move without a press: an item spent shortens the pack,
// and a resized window changes the cell count. A stored position that outlived
// either would leave the bar showing empty cells with no way back.
func (v *Viewer) packScrollAt(cols int) int {
	s := v.packScroll
	if max := packMaxScroll(len(v.invSubject.Pack), cols); s > max {
		s = max
	}
	if s < 0 {
		s = 0
	}
	return s
}

// ScrollPack moves the mission grid by one element per end-strip press and
// reports whether the visible window moved.
func (v *Viewer) ScrollPack(delta int) bool {
	_, cols, ok := v.packBar()
	if !ok {
		return false
	}
	before := v.packScrollAt(cols)
	v.syncPackStarPhases(before, cols)
	after := before + delta
	if max := packMaxScroll(len(v.invSubject.Pack), cols); after > max {
		after = max
	}
	if after < 0 {
		after = 0
	}
	v.packScroll = after
	v.syncPackStarPhases(after, cols)
	return after != before
}

// wornBoxArea and packBarArea are wornBox's and packBar's own rectangles
// with the inventoryEligible half of their gate dropped — the switch alone
// (counterexample 6, round-2 adversarial review, third pass): under a
// multi-unit selection led by the subject, dollBox stays visible
// (dollSubject's own wide rule) while wornBox and packBar draw nothing at
// all (inventoryEligible's own narrow rule, deliberate — see wornBox's own
// doc). The PIXELS those two boxes would otherwise occupy are still the
// inventory window's ground, not the map's: a doll-origin drag CAN be armed
// in this selection state (dollFigureSlotAt shares dollSubject's own wide
// gate), and a release inside where the pack bar or the worn box stands on
// every OTHER selection must not read as "outside every inventory box"
// merely because this particular selection left it undrawn — that reading
// resolved a release aimed at the pack bar's own screen position as a ground
// drop, at a cell the player was never looking at, because inventoryCaptures
// asked packBar (gated narrow) rather than the reserved ground itself.
func (v *Viewer) wornBoxArea() (image.Rectangle, bool) {
	if !v.hudShown(hudPanelWorn) {
		return image.Rectangle{}, false
	}
	return wornBoxRect(image.Pt(v.frameW, v.frameH))
}

func (v *Viewer) packBarArea() (image.Rectangle, bool) {
	if !v.hudShown(hudPanelPack) {
		return image.Rectangle{}, false
	}
	bar, _, ok := packBarRect(image.Pt(v.frameW, v.frameH))
	return bar, ok
}

// dollInventoryActive reports that the visible doll is the inventory
// subject's composed figure. Only this state can arm a doll-origin drag
// while inventoryEligible is false. An enemy portrait, no selection, a
// subject that does not lead the selection, or a switched-off doll cannot
// reserve the otherwise invisible worn and pack areas.
func (v *Viewer) dollInventoryActive() bool {
	if !v.hudShown(hudPanelDoll) || !v.invHasSubject || v.invSubject.Figure == nil {
		return false
	}
	present := presentSelected(v.sel, v.entities)
	return len(present) > 0 && present[0].ID == v.invSubject.ID
}

// inventoryCaptures reports that one of the inventory's own boxes stands under
// the window pixel (x, y) — the whole of "a click on the window is the
// window's" (hotfix), now over the three boxes 0140 split it into.
//
// IT IS ALL OR NONE, deliberately: the doll box, the worn box and the pack bar
// are one surface as far as the map underneath them is concerned, and a caller
// asking "did this press belong to the inventory" wants one answer. Which of
// the three it was is inventoryPackCellAt's and packArrowAt's business, and
// both are asked only after this one has said yes.
//
// A SWITCHED-OFF BOX CAPTURES NOTHING, and that is the one thing this function
// must get right that it did not have to before: a box the player has turned
// off is not there, so the map underneath it takes the press exactly as if it
// had never been drawn. Every one of the three asks its own switch inside the
// three calls below, so there is no fourth place for that rule to be forgotten.
//
// Under a subject-led multi-selection the doll remains interactive while the
// narrower worn and pack boxes draw no content. Their switched-on rectangles
// are reserved ONLY WHILE A DOLL-ORIGIN DRAG IS IN FLIGHT — armed as a
// candidate or already active, dragCandKind == dragFromDoll either way
// (round-2 adversarial review, fifth pass, counterexample C): the comment
// above already said "reserved for that drag only", but the code reserved
// them for the SELECTION STATE alone, with no drag required. A plain press
// or release landing on that screen position with no doll drag under way —
// an ordinary map move order that happens to fall where the pack bar would
// stand on any other selection — was swallowed here and never reached the
// map's own dispatch, because dollInventoryActive() answers from v.sel and
// v.entities alone and says nothing about whether a gesture is actually
// under way. No selection, another leading subject, an enemy portrait, a
// switched-off doll, or a subject-led multi-selection with no doll drag in
// flight all leave the invisible rectangles as map ground.
func (v *Viewer) inventoryCaptures(x, y int) bool {
	p := image.Pt(x, y)
	if box, ok := v.dollBox(); ok && p.In(box) {
		return true
	}
	if box, ok := v.wornBox(); ok && p.In(box) {
		return true
	}
	if bar, _, ok := v.packBar(); ok && p.In(bar) {
		return true
	}
	// The two broad areas are needed only while a doll-origin drag is in
	// flight under a subject-led multi-selection: the doll remains
	// interactive there while the narrower boxes draw no content, and this
	// is that drag's own release surface. Everywhere else — including the
	// same multi-selection with no drag under way — the invisible areas
	// stay map ground, preserving inventoryEligible's existing contract.
	if v.dragCandKind != dragFromDoll || !v.dollInventoryActive() {
		return false
	}
	if box, ok := v.wornBoxArea(); ok && p.In(box) {
		return true
	}
	bar, ok := v.packBarArea()
	return ok && p.In(bar)
}

func (v *Viewer) groundSurfaceCaptures(x, y int) bool {
	return v.inventoryCaptures(x, y) || v.commandPanelCaptures(x, y) ||
		v.spellbookCaptures(x, y) || v.minimapCaptures(x, y) || v.panelCaptures(x, y)
}

// InventoryDoubleClickFrames is how many map-screen frames may separate two
// primary presses on the same pack cell and still count as one double-click
// — the ONE place that number is written. command runs exactly once per
// map-screen frame (app.go's map arm calls it once per Update), so a
// countdown decremented once per call there IS a frame count, with no clock
// read anywhere near it and nothing else able to disagree with it.
//
// THIRTY IS THIS PROJECT'S OWN, DISCLOSED RATHER THAN DERIVED: nothing in
// what this window reconstructs says how a double-click on a pack cell was
// ever timed, because the original never drew our window at all (0110's own
// boundary). It is chosen the way TapSlop's four pixels is (command.go)
// — a magnitude in the middle of what it stands in for rather than at
// either edge: small enough that two presses meant as separate do not link
// up, large enough that two presses a player intends as one double-click do
// not miss each other.
const InventoryDoubleClickFrames = 30

// inventoryPackCellAt reports which CARRIED ELEMENT, if any, stands under
// the window pixel (x, y) — the one geometry this story's hit test is
// built on, and the test command.go's double-click tracking runs every
// primary press through before it touches any of that state.
//
// THE ANSWER IS AN INDEX INTO THE CONTAINER AND NOT INTO THE BAR (0140). The
// bar shows a window onto the pack and the tier above equips by container
// index (pkg/game's enqueueEquip reads CarriedStacks[idx]), so the scroll is
// added here, once, rather than left for a caller to remember. That is also
// what makes the unbounded pack reachable: element 30 is nameable exactly when
// the bar is scrolled to it.
//
// IT READS packBar'S OWN RECTANGLE AND packCellRects' OWN CELLS, AND NOTHING
// ELSE — a second placement expression beside the first is exactly the pair
// that drifts.
//
// A CELL PAST THE END OF THE PACK ANSWERS NO ELEMENT. The bar is drawn full
// width whatever is carried, so most of it is usually empty cells, and an
// empty cell names nothing to equip — which is what keeps "a double-click on
// empty bar raises no request" true with no separate exclusion written for it.
func (v *Viewer) inventoryPackCellAt(x, y int) (int, bool) {
	bar, cols, ok := v.packBar()
	if !ok {
		return 0, false
	}
	scroll := v.packScrollAt(cols)
	p := image.Pt(x, y)
	for i, r := range packCellRects(bar, cols) {
		if !p.In(r) {
			continue
		}
		idx := scroll + i
		if idx >= len(v.invSubject.Pack) {
			return 0, false
		}
		return idx, true
	}
	return 0, false
}

// packArrowAt reports which decoded end strip, if any, the window pixel names:
// -1 for back and +1 for forward.
func (v *Viewer) packArrowAt(x, y int) (int, bool) {
	bar, cols, ok := v.packBar()
	if !ok {
		return 0, false
	}
	back, fwd := packScrollRects(bar, cols)
	p := image.Pt(x, y)
	switch {
	case p.In(back):
		return -1, true
	case p.In(fwd):
		return 1, true
	}
	return 0, false
}

// wornPresent is the worn box's picture for this frame and where its top-left
// corner goes, or false for a frame that draws none — the panel's own
// panelPresent shape.
//
// THE REBUILD RULE IS A REVISION COUNTER, and it was a comparison of the whole
// subject by value until 0140 — InventorySubject stopped being comparable when
// the pack became a slice (see the type's own doc). The counter is strictly
// stronger for this box's purposes: SetInventorySubject already forced a
// recompose on every call, so the picture was never held across one, and this
// says the same thing without needing the value. Unlike the doll's key it needs
// nothing about the selection, because this box only ever draws the ONE subject
// its own gate admits.
func (v *Viewer) wornPresent() (*image.RGBA, image.Point, bool) {
	box, ok := v.wornBox()
	if !ok {
		return nil, image.Point{}, false
	}
	if v.invPic == nil || v.invKey != v.invRev {
		v.invPic, v.invFresh = RenderWorn(v.invSubject), true
		v.invKey = v.invRev
		v.invBuilds++
	}
	if v.invPic == nil {
		return nil, image.Point{}, false
	}
	return v.invPic, box.Min, true
}

// packBarPresent is the pack bar's picture for this frame and where its
// top-left corner goes, or false for a frame that draws none (0140).
//
// IT RECOMPOSES EVERY FRAME IT DRAWS, the minimap's and the spellbook's own
// choice rather than the doll box's cached one: the bar is a row of cells and
// a scroll position, both cheap, and it is a function of state that moves
// without passing through SetInventorySubject at all — the scroll. A key that
// had to carry the scroll, the cell count and the window size would cost more
// to keep honest than the composition costs to redo.
func (v *Viewer) packBarPresent() (*image.RGBA, image.Point, bool) {
	bar, cols, ok := v.packBarShown()
	if !ok {
		return nil, image.Point{}, false
	}
	subject, scroll := InventorySubject{}, 0
	if v.inventoryEligible() {
		subject, scroll = v.previewSubject(), v.packScrollAt(cols)
	}
	v.syncPackStarPhases(scroll, cols)
	pic := renderPackBarArt(subject, scroll, cols, bar, v.cardFont(), v.packStarPhases, v.packStarOmit(), v.bottomHUDArt)
	if v.bottomHUDArt != nil {
		back, fwd := packScrollRects(bar, cols)
		p := image.Pt(v.cursorX, v.cursorY)
		if p.In(back) && scroll > 0 {
			blit(pic, v.bottomHUDArt.PackArrow[2], back.Max.X-32, packArrowDY)
		} else if p.In(fwd) && scroll < packMaxScroll(len(subject.Pack), cols) {
			blit(pic, v.bottomHUDArt.PackArrow[3], fwd.Min.X, packArrowDY)
		}
	}
	return pic, image.Pt(0, bar.Min.Y), true
}

// AdvanceInventoryStars is the mission grid's ITEM-STARPHASE-099 message
// clock. pkg/game calls it from the function that is one actual world tick;
// paused frames never enter that function, so paint alone holds every phase.
func (v *Viewer) AdvanceInventoryStars() {
	bar, cols, ok := v.packBar()
	if !ok || bar.Empty() {
		return
	}
	scroll := v.packScrollAt(cols)
	v.syncPackStarPhases(scroll, cols)
	omit := v.packStarOmit()
	for i := range v.packStarPhases {
		idx := scroll + i
		if idx == omit || idx < 0 || idx >= len(v.invSubject.Pack) || idx >= len(v.invSubject.PackStars) {
			continue
		}
		if v.invSubject.Pack[idx] != nil && v.invSubject.PackStars[idx] {
			v.packStarPhases[i]++
		}
	}
}

func (v *Viewer) syncPackStarPhases(scroll, cols int) {
	if cols <= 0 {
		v.packStarPhases = nil
		v.packStarCols, v.packStarScroll = 0, scroll
		return
	}
	if v.packStarCols == 0 {
		v.packStarPhases = make([]uint32, cols)
		v.packStarCols, v.packStarScroll = cols, scroll
		return
	}
	if v.packStarCols != cols {
		resized := make([]uint32, cols)
		copy(resized, v.packStarPhases)
		v.packStarPhases, v.packStarCols = resized, cols
	}
	shiftItemStarPhases(v.packStarPhases, scroll-v.packStarScroll)
	v.packStarScroll = scroll
}

func (v *Viewer) packStarOmit() int {
	if !v.dragActive || v.dragCandKind != dragFromPack || v.dragCandIdx < 0 || v.dragCandIdx >= len(v.invSubject.PackCount) {
		return -1
	}
	if v.invSubject.PackCount[v.dragCandIdx] <= 1 {
		return v.dragCandIdx
	}
	return -1
}

// TakeInventoryEquip reads the pending equip request a double-click on a
// pack cell raised, if any, and clears it in the SAME statement — so a
// request drained once cannot be drained a second time by a caller that asks
// again before the next one is raised. The bool is false and the int
// meaningless whenever command has raised no request since the last drain,
// mirroring every other seam in this package that crosses a one-shot value
// out rather than a standing one.
func (v *Viewer) TakeInventoryEquip() (int, bool) {
	if v.invEquipRequest == 0 {
		return 0, false
	}
	idx := v.invEquipRequest - 1
	v.invEquipRequest = 0
	return idx, true
}

// RaiseDocuments asks the front end to open the campaign documents panel. It
// is the way IN to the one-shot below, and it is raised from the tier above
// rather than from this file: the item that opens the panel is recognised by
// its code, and this package does not read item codes.
//
// A STANDING REQUEST IS OVERWRITTEN, NEVER QUEUED, which is invEquipRequest's
// own rule: the panel opens once whatever raised it twice.
func (v *Viewer) RaiseDocuments() {
	if v == nil {
		return
	}
	v.docRequest = true
}

// TakeDocumentsRequest reads the pending documents request, if any, and clears
// it in the SAME statement — TakeInventoryEquip's own shape, and for its own
// reason: a request drained once cannot be drained again before the next one
// is raised.
//
// THE TIER ABOVE DRAINS IT. Whether the panel actually opens is the front
// end's question, not this file's: a collection with nothing in it refuses
// (flow.openDocuments), and the request is spent either way.
func (v *Viewer) TakeDocumentsRequest() bool {
	if v == nil || !v.docRequest {
		return false
	}
	v.docRequest = false
	return true
}

// inventoryWornSlotAt reports which of the twelve worn-box cells, if any,
// stands under the window pixel (x, y) — wornSlotRects' own geometry, read
// once here and by hoveredItemInfoAt (itempopup.go) for the same box, so a
// press and a hover agree on where each of the twelve cells is.
//
// THE ANSWER IS A SLOT INDEX, ZERO-BASED (0..11), matching Slots, SlotInfo
// and wornSlotRects' own indexing — one below pkg/sim's own 1..12 slot
// numbering (equip.go). A caller turning this into a sim.Command adds the
// one back (pkg/game's enqueueUnequip).
func (v *Viewer) inventoryWornSlotAt(x, y int) (int, bool) {
	box, ok := v.wornBox()
	if !ok {
		return 0, false
	}
	p := image.Pt(x, y).Sub(box.Min)
	for i, r := range wornSlotRects() {
		if p.In(r) {
			return i, true
		}
	}
	return 0, false
}

// dollFigureSlotAt reports which equipment slot, if any, owns the doll
// figure's pixel under the window pixel (x, y) — SlotMask's own lookup, at
// the placement drawInventoryPicture already centres the figure at inside
// dollFigureRect (1005, "the interactive doll").
//
// THE WORN BOX AND THE PACK BAR KEEP inventoryEligible'S NARROWER GATE,
// DELIBERATELY (wornBox's own doc, inventory.go): both compose PER-SLOT
// ICONS this build only ever builds for the party's own subject alone, so a
// multi-unit selection leaves them with nothing correct to draw at all — the
// doll's own composed FIGURE, by contrast, is exactly the one present[0]'s
// own subject picture, whether or not anyone else rides in the same
// selection, which is why only the doll's hit test widens here.
//
// IT READS THE SUPPRESSED MASK WHENEVER dollSubject IS DRAWING THE
// SUPPRESSED PICTURE (1005 round-1 adversarial review, against the ordinary
// invSubject.SlotMask staying in force through a drag): the same three-field
// check dollSubject makes — same owner, a non-zero suppressed slot, a
// non-nil suppressed picture — selects dollSuppressMask in its place, so
// the pixels this answers for are always the pixels of the picture ACTUALLY
// ON SCREEN. A pixel the lifted slot's own layer used to own now answers
// whatever the suppressed mask says paints there instead — no slot over
// the bare body, or the exposed layer's own slot where the compositor's own
// draw order put one there — never a special-cased "nothing": the mask of
// the drawn picture already carries the right answer.
//
// THE ANSWER IS ZERO-BASED (0..11), wornSlotRects' and inventoryWornSlotAt's
// own indexing (Slots, SlotInfo) — one below SlotMask's own 1..12, which is
// pkg/sim's own numbering carried unchanged from the compositor. A caller
// turning this into a sim.Command adds the one back exactly as
// enqueueUnequip already does for the worn box's own drain.
func (v *Viewer) dollFigureSlotAt(x, y int) (int, bool) {
	mask := v.invSubject.SlotMask
	if v.dollSuppressOwner == v.invSubject.ID && v.dollSuppressSlot != 0 && v.dollSuppressPic != nil {
		mask = v.dollSuppressMask
	}
	return v.dollFigureSlotAtMask(x, y, mask)
}

func (v *Viewer) dollFigureSlotAtUnsuppressed(x, y int) (int, bool) {
	return v.dollFigureSlotAtMask(x, y, v.invSubject.SlotMask)
}

// dollFigureSlotAtMask is dollFigureSlotAt's and
// dollFigureSlotAtUnsuppressed's shared geometry: dollSubject's own
// drawn-figure gate, then dollBox's own placement, over whichever mask the
// caller names. shopDollSlotAt maps the 160x240 composed figure onto the
// pane WITHOUT CENTRING, from characterPaneFigureOrigin, and restricts the
// answer to the full 160x240 figure the pane paints; this does the same,
// over the same two functions, so `TOWN-349`'s per-pixel slot map is one map
// on both screens rather than two arrangements that happen to agree. The
// pane's own corner controls subtract nothing from it: where rect B or rect
// C shares a pixel with a marked slot the two are separated by gesture, in
// command.go's corner arm (`DIV-308`).
func (v *Viewer) dollFigureSlotAtMask(x, y int, mask *SlotMask) (int, bool) {
	src, ok := v.dollSubject()
	if !ok || src.figure == nil {
		return 0, false
	}
	box, ok := v.dollBox()
	if !ok {
		return 0, false
	}
	if mask == nil || mask.W <= 0 || mask.H <= 0 {
		return 0, false
	}
	p := image.Pt(x, y)
	if !v.characterPaneFigureAt(p) {
		return 0, false
	}
	o := characterPaneFigureOrigin(box)
	n, ok := mask.At(x-o.X, y-o.Y)
	if !ok {
		return 0, false
	}
	return n - 1, true
}

// TakeInventoryUnequip reads the pending unequip request a double-click on
// a worn-box cell raised, if any, and clears it in the same statement —
// TakeInventoryEquip's own shape, over the worn box's own one-shot request
// rather than the pack's. This file does not ask whether the cell named is
// occupied: inventoryWornSlotAt is pure geometry, and a double-click on an
// empty cell raises a request the same as one on a worn item.
//
// THE TIER ABOVE DRAINS IT (pkg/game's unequipFromWorn); this file only
// holds it. What a drained index turns into — a command, or nothing at all
// when the slot names no worn item — is sim's own unequip totality
// (pkg/sim/equip.go), unasked here.
func (v *Viewer) TakeInventoryUnequip() (int, bool) {
	if v.invUnequipRequest == 0 {
		return 0, false
	}
	idx := v.invUnequipRequest - 1
	v.invUnequipRequest = 0
	return idx, true
}

// TakeInventoryDollUnequip reads the pending unequip request a press on the
// doll figure raised, if any, and clears it in the same statement —
// TakeInventoryUnequip's own shape, over a THIRD one-shot request rather
// than a second write to that one (1005, "the interactive doll"): a tap on
// a doll slot that never crossed TapSlop, and a drag lifted off the doll and
// released into the pack, both raise this one — command.go's own two
// producers for one drain.
//
// THE TIER ABOVE DRAINS IT (pkg/game's unequipFromDoll, beside
// unequipFromWorn); this file only holds it, and the drained index turns
// into a command through mapWorld.enqueueUnequip exactly as
// TakeInventoryUnequip's own index does — the same rule, not a second copy
// of it.
func (v *Viewer) TakeInventoryDollUnequip() (int, bool) {
	if v.invDollUnequipRequest == 0 {
		return 0, false
	}
	idx := v.invDollUnequipRequest - 1
	v.invDollUnequipRequest = 0
	return idx, true
}

// raiseGroundDrop is command.go's one writer for the ground-drop request
// (1005 round 2, `ITEM-DROP-008`, `DIV-088`): a drag released outside every
// inventory box, during a mission. worn and idx name the drag's own
// origin — a doll slot (worn true, idx zero-based, dollFigureSlotAt's own
// numbering) or a pack element (worn false, idx the drag's own dragCandIdx,
// CarriedStacks' own element ordering) — and x, y are the cell
// v.groundCellAt named at the release point, in world cell units.
//
// A STANDING REQUEST IS OVERWRITTEN, NEVER QUEUED, invEquipRequest's own
// rule restated: command.go raises at most one request per release, so
// there is never a second one to lose.
func (v *Viewer) raiseGroundDrop(worn bool, idx int, x, y int32) {
	v.invDropRequest = idx + 1
	v.invDropWorn = worn
	v.invDropX, v.invDropY = x, y
}

// TakeInventoryDrop reads the pending ground-drop request a release outside
// every inventory box raised, if any, and clears it in the same statement —
// TakeInventoryDollUnequip's own shape, over a fourth one-shot request that
// additionally carries which container the origin names and where the sack
// should land, since a single index cannot hold either (viewer.go's own
// field comment).
//
// THE TIER ABOVE DRAINS IT (pkg/game's dropFromInventory); this file only
// holds it, and the drained values turn into a command through
// mapWorld.enqueueDrop exactly as every other drained request here does —
// through the one function that states the rule, never a second copy of it.
func (v *Viewer) TakeInventoryDrop() (worn bool, idx int, x, y int32, ok bool) {
	if v.invDropRequest == 0 {
		return false, 0, 0, 0, false
	}
	idx = v.invDropRequest - 1
	worn = v.invDropWorn
	x, y = v.invDropX, v.invDropY
	v.invDropRequest = 0
	return worn, idx, x, y, true
}

// DraggedDollSlot is the zero-based worn slot a drag is currently carrying
// off the doll, or false while no drag is active or the active one is
// sourced from the pack instead (1005, "the interactive doll") —
// dollFigureSlotAt's own indexing, read by pkg/game's refreshDollDrag every
// frame to decide which equipment set, if any, to recompose the suppressed
// figure from. It is a LEVEL and not a one-shot: unlike
// TakeInventoryDollUnequip it answers the same slot on every call for as
// long as the drag stands, so a caller that asks twice in one frame is not
// the reason it changes.
func (v *Viewer) DraggedDollSlot() (int, bool) {
	if !v.dragActive || v.dragCandKind != dragFromDoll {
		return 0, false
	}
	return v.dragCandIdx, true
}

// dragItemPresent is the carried item's own picture for this frame and
// where its top-left corner goes, centred on the cursor, or false for a
// frame carrying nothing (1005, "the interactive doll" — "carried on the
// cursor"). It answers false for a viewer no drag has armed and one that
// has armed but resolved to no icon — an index the pack or the slots array
// no longer holds by the time TapSlop was crossed, which is not expected in
// practice inside one held gesture but is refused rather than assumed.
func (v *Viewer) dragItemPresent() (*image.RGBA, image.Point, bool) {
	if !v.dragActive || v.dragIcon == nil || !v.hasCursor {
		return nil, image.Point{}, false
	}
	b := v.dragIcon.Bounds()
	at := image.Pt(v.cursorX-b.Dx()/2, v.cursorY-b.Dy()/2)
	return v.dragIcon, at, true
}
