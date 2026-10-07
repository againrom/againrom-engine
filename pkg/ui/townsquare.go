package ui

import (
	"image"
	"image/draw"

	"againrom/pkg/render/text"
)

// TownSquareArt is the town square's immutable install-backed presentation:
// the base picture, its overlay strip, the raster hit-test mask and the
// three door labels. Optional exactly as TownSchoolArt is (pkg/game's
// LoadTownSquareArt): a nil Background means no install-backed picture, and
// the caller keeps the row-button layout drawTown has always drawn for the
// square.
type TownSquareArt struct {
	Background image.Image     // townmain.bmp, 640x480
	Add        image.Image     // town_add.bmp, 552x92, composited at TownSquareAddOrigin
	Mask       *image.Paletted // townmask.bmp, 640x480, the raster hit test
	Labels     [3]image.Image  // shop, tavern, school — TownSquareLabel* indices
	Exterior   *TownExteriorArt
	// Optional motion can fail without removing the usable square/mask.
	ExteriorProblems []string
}

// TownExteriorArt contains independent, optional installed animation families.
// A missing family is omitted, never replaced with invented art.
type TownExteriorArt struct {
	Shop, Tavern, Fighter, Mage, Guard []image.Image
	Door, Sign, Fluger                 []image.Image
	// Birds is the nine independent Birds1..Birds9 .16a families. Stars is
	// the one atomic S00..S08 bitmap family. A nil member is a local cosmetic
	// fallback and never removes the other exterior families.
	Birds [9][]image.Image
	Stars []image.Image
	// Horse, Baba and Dervish are the installed wildlife sheets indexed by
	// position (zero-based) and, for horse and baba, by variant. A nil sheet
	// is a local cosmetic fallback.
	Horse   [TownHorsePositions][TownHorseSheets][]image.Image
	Baba    [TownBabaPositions][TownBabaSheets][]image.Image
	Dervish [TownDervishPositions][]image.Image
}

const (
	TownHorsePositions   = 5
	TownHorseSheets      = 3
	TownBabaPositions    = 4
	TownBabaSheets       = 2
	TownDervishPositions = 4
)

// TownViewOrigin is the town view's left and top on a screen of the given
// size (TOWN-503). The square is one 640x480 picture the window centres.
func TownViewOrigin(screenW, screenH int) image.Point {
	return image.Pt((screenW-640)/2, (screenH-480)/2)
}

// townHorseOrigin, townBabaOrigin and townDervishOrigin are the compiled
// (x, y) top-left rectangles of each wildlife position, relative to the town
// view (TOWN-490, TOWN-503).
var (
	townHorseOrigin   = [TownHorsePositions]image.Point{{104, 404}, {104, 404}, {256, 344}, {448, 400}, {140, 400}}
	townBabaOrigin    = [TownBabaPositions]image.Point{{216, 364}, {308, 424}, {384, 424}, {580, 384}}
	townDervishOrigin = [TownDervishPositions]image.Point{{224, 364}, {324, 424}, {392, 420}, {592, 388}}
)

// TownFamilyFrame names one wildlife family's selected position, sheet and
// frame. A zero value is not drawn.
type TownFamilyFrame struct {
	Visible                bool
	Position, Sheet, Frame int
}

// TownExteriorFrame is read-only presentation state, not a simulation value.
type TownExteriorFrame struct {
	Shop, Tavern, Fighter, Mage, Guard int
	Door, Sign, Fluger                 int
	Birds                              [3]TownExteriorSpriteFrame
	BirdOverlayVisible                 bool
	Star                               TownExteriorSpriteFrame
	Horse, Baba, Dervish               TownFamilyFrame
}

// TownExteriorSpriteFrame makes visibility explicit for optional episodes.
// Frame zero is a real picture, so a zero-valued descriptor must not paint.
type TownExteriorSpriteFrame struct {
	Family  int
	Frame   int
	Visible bool
}

const (
	TownSquareLabelShop = iota
	TownSquareLabelTavern
	TownSquareLabelSchool
)

// TownSquareAddOrigin is where town_add.bmp sits over the base picture.
//
// IT IS MEASURED, NOT AUTHORED: cmd/townsquarecheck correlates town_add.bmp
// over the whole 640x480 base and reports the winning offset with the
// runner-up fraction anywhere else. The winner is (0,0) at fraction 1.0000
// (every one of its 30140 opaque pixels byte-identical to the base there),
// against a next-best fraction of 0.1903 — the base already carries this
// exact rooftop strip. Compositing at this origin is provably a no-op
// against the current art; docs/1016-town-square/spec.md records the
// figures and the open question of why the install ships the duplicate.
var TownSquareAddOrigin = image.Pt(0, 0)

// townSquareLabelOrigin is each door label's own placement.
//
// MEASURED BY cmd/townsquarecheck: each label is slid over the base picture
// and scored at every offset. The tool's own match fractions are 0.7715 (shop),
// 0.5893 (tavern) and 0.6884 (school) against runners-up of 0.1546, 0.1618 and
// 0.0951 — factors of 5.0, 3.6 and 7.2. docs/1016-town-square/spec.md records
// them.
//
// THE FACTOR IS A PROPERTY OF THE INSTRUMENT, NOT ONLY OF THE ART (round-3
// review, D-2, which found this comment claiming fiftyfold). Every runner-up
// above sits within six pixels of its winner, and the tool scores with a
// tolerance, so a near-miss of the right label still scores well and the ratio
// stays single-digit. Counting only EXACT pixel matches instead collapses a
// one-pixel shift to almost nothing and puts the same three margins at 52x, 32x
// and 237x; that is where the fiftyfold reading came from, and it is the figure
// PIPELINE-STATUS quotes. Both instruments are right about different
// quantities. Neither factor is a threshold anything tests: the tool asserts
// the winning OFFSET against production, which is the fact that matters.
var townSquareLabelOrigin = [3]image.Point{
	TownSquareLabelShop:   {264, 264},
	TownSquareLabelTavern: {144, 332},
	TownSquareLabelSchool: {436, 300},
}

// TownSquareLabelOrigin is exported for cmd/townsquarecheck, the story's
// measuring instrument (SchoolSkillRect's own precedent, townshell.go): that
// tool derives each label's placement independently by correlation and
// compares it against this value, because a Go test may not read an install
// (golden rule 2). An out-of-range index answers the zero point.
func TownSquareLabelOrigin(i int) image.Point {
	if i < 0 || i >= len(townSquareLabelOrigin) {
		return image.Point{}
	}
	return townSquareLabelOrigin[i]
}

// The five townmask.bmp colour codes that carry an interactive region.
//
// IDENTIFIED TWO WAYS. First, which label (shop_l, tavern_l, trener_l)
// correlates onto the base picture with which code dominating the mask
// underneath it (cmd/townsquarecheck): shop_l over code 144, tavern_l over
// code 128, trener_l — the skill school's training column — over code 192.
// Second, the shipped tip text main/text/tips/town.txt, which names the
// shop centre, the inn (tavern) left, the school right, the gate and the
// Gilded Statue: five names for the five codes with non-trivial area. A
// colour overlay of the mask on the base picture (cmd/townsquarecheck -png)
// shows five distinct regions matching that text: the tavern's doorway
// (left), a stone gate arch in the background between the tavern and the
// shop, the shop's doorway (centre), the golden statue (centre foreground)
// and the school's archway with its training column (right).
//
// TOWN-163 subsequently confirmed these exact five bytes and click selectors;
// TOWN-399 binds the same sampling to delivered pointer/label reactions.
const (
	townSquareCodeTavern = 128
	townSquareCodeShop   = 144
	townSquareCodeGate   = 160
	townSquareCodeStatue = 176
	townSquareCodeSchool = 192
)

// TownSquareMaskCodes is the five codes above, for chargenMaskCodes' presence
// check (pkg/game's LoadTownSquareArt): a correctly sized but blank mask
// would otherwise load and leave the whole square inert at runtime.
var TownSquareMaskCodes = [5]uint8{
	townSquareCodeTavern, townSquareCodeShop, townSquareCodeGate,
	townSquareCodeStatue, townSquareCodeSchool,
}

// TownSquareControlKind is what a click on the square's picture resolved to:
// one of the four doors Choose(i) already accepts, or the statue, which
// opens the save/load mini-menu rather than a door (0143's own seam).
type TownSquareControlKind uint8

const (
	TownSquareControlNone TownSquareControlKind = iota
	TownSquareControlDoor
	TownSquareControlMenu
)

// TownSquareControl is one resolved click. Door is the row index Choose(i)
// expects, in pkg/game's townDoors order (0 tavern, 1 shop, 2 school,
// 3 gates); it is valid only when Kind is TownSquareControlDoor.
type TownSquareControl struct {
	Kind TownSquareControlKind
	Door int
}

// TownSquareControlAt reads one pixel of the install's own townmask.bmp and
// answers which door — or the statue — that pixel belongs to.
//
// A CODE THIS TABLE DOES NOT CARRY ANSWERS NO HIT AT ALL, on the school's own
// exact-code rule (schoolMaskSlot, townshell.go): the background code and the
// roughly 140 single- and double-digit codes at region edges are read as
// nothing rather than tolerantly assigned to their nearest neighbour.
func TownSquareControlAt(mask *image.Paletted, p image.Point) (TownSquareControl, bool) {
	if mask == nil || !p.In(mask.Bounds()) {
		return TownSquareControl{}, false
	}
	switch mask.ColorIndexAt(p.X, p.Y) {
	case townSquareCodeTavern:
		return TownSquareControl{Kind: TownSquareControlDoor, Door: 0}, true
	case townSquareCodeShop:
		return TownSquareControl{Kind: TownSquareControlDoor, Door: 1}, true
	case townSquareCodeSchool:
		return TownSquareControl{Kind: TownSquareControlDoor, Door: 2}, true
	case townSquareCodeGate:
		return TownSquareControl{Kind: TownSquareControlDoor, Door: 3}, true
	case townSquareCodeStatue:
		return TownSquareControl{Kind: TownSquareControlMenu}, true
	}
	return TownSquareControl{}, false
}

// TownSquareView is what the town square draws: the resolved art, the font
// its message line uses, and the message line every composed room shares.
type TownSquareView struct {
	Art     *TownSquareArt
	Font    *text.Font
	Message string
	// Selector is TOWN-399's exact mask selector; zero/-1 draws no label.
	Selector int
	Exterior *TownExteriorFrame
	// Tip is the square's own tip panel (1018 spec behaviours 1, 2), drawn
	// last, over the base picture, the labels and the message line.
	Tip TipPanelView
}

// ComposeTownSquare paints the shipped picture: the base, a selected bird
// episode, its keyed overlay strip, the selected door label and entrance
// motion, then the statue star. The overlay (town_add.bmp) is 40.7% pure
// black — the sky its own footprint does not cover — and
// LoadTownSquareArt keys it (keyBlack) before this function ever sees it, so
// draw.Over here leaves that black transparent; the remaining pixels are
// byte-identical to the base at this offset (measured at DIV-149's closure:
// 30140 opaque pixels, none differing, on both preserved roots), so the
// composite is a true no-op there (round-3 review). None of the three labels
// carries a pure-black pixel (cmd/townsquarecheck counts zero across all
// three), so they are drawn the way the school's own rest faces are
// (LoadTownSchoolArt): standing in for whatever the base already shows
// there, opaque and unkeyed.
//
// IT PAINTS NO HINT TEXT (spec SC-1..SC-3). The per-door availability count,
// the room header and the gold/party footer line are cut from this surface —
// none of it is part of the shipped picture. Header, Footer and Rows still
// answer every one of them for a caller not drawing this art.
//
// A nil or incomplete Art answers a blank canvas; app.go never calls this
// without first confirming Art.Background is non-nil (see townSquareView).
func ComposeTownSquare(v TownSquareView) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
	if v.Art == nil || v.Art.Background == nil {
		return dst
	}
	draw.Draw(dst, dst.Bounds(), v.Art.Background, v.Art.Background.Bounds().Min, draw.Src)
	a, f := v.Art.Exterior, v.Exterior
	if a != nil && f != nil {
		for _, selected := range f.Birds {
			if !selected.Visible || selected.Family < 0 || selected.Family >= len(a.Birds) {
				continue
			}
			frames := a.Birds[selected.Family]
			if selected.Frame < 0 || selected.Frame >= len(frames) || frames[selected.Frame] == nil {
				continue
			}
			pic := frames[selected.Frame]
			draw.Draw(dst, pic.Bounds().Add(image.Pt(0, 0).Sub(pic.Bounds().Min)), pic, pic.Bounds().Min, draw.Over)
		}
	}
	if f != nil && f.BirdOverlayVisible && v.Art.Add != nil {
		b := v.Art.Add.Bounds()
		draw.Draw(dst, b.Add(TownSquareAddOrigin.Sub(b.Min)), v.Art.Add, b.Min, draw.Over)
	}
	for i, lbl := range v.Art.Labels {
		if lbl == nil || v.Selector != [3]int{1, 2, 4}[i] {
			continue
		}
		b := lbl.Bounds()
		draw.Draw(dst, b.Add(townSquareLabelOrigin[i].Sub(b.Min)), lbl, b.Min, draw.Src)
	}
	if a != nil && f != nil {
		// TOWN-399..404 / pinned town paint: literal top-left destinations,
		// ordered like the original paint. Sprite alpha is decoded at load.
		patch := func(frames []image.Image, index int, p image.Point) {
			if index < 0 || index >= len(frames) || frames[index] == nil {
				return
			}
			pic := frames[index]
			b := pic.Bounds()
			draw.Draw(dst, b.Add(p.Sub(b.Min)), pic, b.Min, draw.Over)
		}
		patch(a.Tavern, f.Tavern, image.Pt(124, 312))
		patch(a.Sign, f.Sign, image.Pt(360, 232))
		patch(a.Door, f.Door, image.Pt(180, 148))
		patch(a.Fighter, f.Fighter, image.Pt(516, 344))
		patch(a.Mage, f.Mage, image.Pt(452, 328))
		patch(a.Shop, f.Shop, image.Pt(276, 296))
		patch(a.Fluger, f.Fluger, image.Pt(308, 64))
		patch(a.Guard, f.Guard, image.Pt(184, 158))
		if f.Star.Visible && f.Star.Frame >= 0 && f.Star.Frame < len(a.Stars) && a.Stars[f.Star.Frame] != nil {
			pic := a.Stars[f.Star.Frame]
			b := pic.Bounds()
			draw.Draw(dst, b.Add(image.Pt(340, 288).Sub(b.Min)), pic, b.Min, draw.Src)
		}
	}
	if a != nil && f != nil {
		// TOWN-489: horse, baba and dervish are the painter's last three
		// sprite draws, in that order, above everything else on the picture.
		view := TownViewOrigin(dst.Bounds().Dx(), dst.Bounds().Dy())
		put := func(sel TownFamilyFrame, frames []image.Image, p image.Point) {
			p = p.Add(view)
			if !sel.Visible || sel.Frame < 0 || sel.Frame >= len(frames) || frames[sel.Frame] == nil {
				return
			}
			pic := frames[sel.Frame]
			b := pic.Bounds()
			draw.Draw(dst, b.Add(p.Sub(b.Min)), pic, b.Min, draw.Over)
		}
		if h := f.Horse; h.Position >= 0 && h.Position < TownHorsePositions && h.Sheet >= 0 && h.Sheet < TownHorseSheets {
			put(h, a.Horse[h.Position][h.Sheet], townHorseOrigin[h.Position])
		}
		if b := f.Baba; b.Position >= 0 && b.Position < TownBabaPositions && b.Sheet >= 0 && b.Sheet < TownBabaSheets {
			put(b, a.Baba[b.Position][b.Sheet], townBabaOrigin[b.Position])
		}
		if d := f.Dervish; d.Position >= 0 && d.Position < TownDervishPositions {
			put(d, a.Dervish[d.Position], townDervishOrigin[d.Position])
		}
	}
	if v.Message != "" {
		drawTownShellText(dst, v.Font, v.Message, image.Rect(12, 448, 468, 478), townShellText)
	}
	ComposeTipPanel(dst, v.Tip)
	return dst
}
