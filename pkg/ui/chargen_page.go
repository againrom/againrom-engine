package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"againrom/pkg/render/text"
)

// ChargenPresentation is all immutable artwork and wording resolved from one
// install. UI receives pixels and bytes only; it never opens an archive.
type ChargenPresentation struct {
	Background  image.Image
	PreMask     *image.Paletted
	Amulet      image.Image
	Forward     image.Image
	Plate       image.Image
	Choices     [4][3]image.Image // rest, hover, selected
	Levels      [3][3]image.Image // on, l, lon (TOWN-223); idle is baked in Background
	Sparkles    []*image.RGBA
	Columns     [2]image.Image
	ColumnMask  [2]*image.Paletted
	Skills      [2][5][3]image.Image
	StatButtons [2][5]image.Image // minus/plus × native source states
	// NavArt is the command panel's background, Inn\ButtonsArea.bmp at
	// (480,0), and NavButtons are Accept, Reset and Back, off and on
	// (MENU-139).
	NavArt     image.Image
	NavButtons [3][2]image.Image
	// NavSeam is interface/inn/ruover.bmp, the same strip
	// ui.TownSchoolArt.UpperSeam carries, closing the same 16-column gap
	// beside NavArt (DIV-166, DIV-168).
	NavSeam   image.Image
	PlateSeam image.Image
	DollPane  TownPane
	// CardSeam is chrgen/fullstatsr.bmp, 16x242, closing the character
	// card's own x:[160,176) beside it (TOWN-312's `+0x6c` source,
	// `TOWN-234`'s `(160,238,16,242)` blit; DIV-189, 1022 spec B2/B3/B10).
	// The card's own body slot, chrgen/fullstatsl.bmp, is not held here as a
	// separate pane image: it is CardBackground below, composed INTO the
	// card by CompactPanelLayout rather than drawn under it as a plain
	// TownPane body, which is what "drawn on fullstatsl" (1022 contract B3)
	// means.
	CardSeam image.Image
	// CardBackground is chrgen/fullstatsl.bmp, 160x242, the character
	// card's own shipped body (1022 spec B2/B3). It is *image.RGBA, not
	// image.Image, because it is handed to ui.CompactPanelLayout's own
	// Background field directly.
	CardBackground *image.RGBA
	Font           *text.Font
	// NameFont is font4, which draws the pre-create prompt and name
	// (TEXT-077). Nil falls back to Font.
	NameFont *text.Font
}

type chargenControl uint8

const (
	chargenNone chargenControl = iota
	chargenName
	chargenChoice0
	chargenChoice1
	chargenChoice2
	chargenChoice3
	chargenBack
	chargenForward
	chargenSkill0
	chargenSkill1
	chargenSkill2
	chargenSkill3
	chargenSkill4
	chargenStatMinus0
	chargenStatMinus1
	chargenStatMinus2
	chargenStatMinus3
	chargenStatPlus0
	chargenStatPlus1
	chargenStatPlus2
	chargenStatPlus3
	chargenReset
	chargenPlay
	chargenLevel0
	chargenLevel1
	chargenLevel2
)

const (
	chargenPageW = 640
	chargenPageH = 480
)

func preControlRegion(id chargenControl) image.Rectangle {
	switch id {
	case chargenName:
		return image.Rect(224, 310, 362, 337) // the name field's rect (TEXT-075)
	case chargenChoice0, chargenChoice1, chargenChoice2, chargenChoice3:
		return image.Rectangle{}
	case chargenBack:
		return image.Rect(536, 133, 640, 328)
	case chargenForward:
		return image.Rect(463, 354, 580, 451)
	default:
		return image.Rectangle{}
	}
}

var preChoiceOrigin = [...]image.Point{{16, 273}, {416, 190}, {124, 166}, {288, 130}}

// Installed-art fit, not original rectangle-table coordinates (DIV-532).
var preLevelOrigin = [...]image.Point{{60, 196}, {0, 110}, {48, 65}}
var preMaskCode = [...]uint8{80, 140, 100, 120}

const (
	preBackMaskCode    = 160
	preForwardMaskCode = 180
)

var preForwardOrigin = image.Pt(468, 373)
var preAmuletOrigin = image.Pt(528, 140)

// preCycleChoices is the portrait slot order mf, ff, fm, mm (TOWN-520) as
// choice indexes; the cycle and the paint loop walk it.
var preCycleChoices = [...]int{0, 2, 3, 1}

// preCreateCycleSteps are the guided cycle's targets per tip step as mask
// region codes: portraits, levels, then amulet and OK (TOWN-519, TOWN-520).
var preCreateCycleSteps = [][]int{{80, 100, 120, 140}, {20, 40, 60}, {preBackMaskCode, preForwardMaskCode}}

// detailedSkillCycleSteps is the skill cycle's one step over the five skills.
var detailedSkillCycleSteps = [][]int{{0, 1, 2, 3, 4}}

// preStateArt is TOWN-521's state art: 1 on, 2 l, 3 lon, 0 none.
func preStateArt(states [3]image.Image, state int) image.Image {
	if state < 1 || state > 3 {
		return nil
	}
	return states[state-1]
}

// preChoiceState and preLevelState are the state bits: bit 0 chosen, bit 1
// under the pointer or the keyboard focus.
func (c *Chargen) preChoiceState(i int, hover chargenControl) int {
	id := chargenChoice0 + chargenControl(i)
	return preStateBits(c.preChoice == i, hover == id || preFocusControl(c.focus) == id)
}

func (c *Chargen) preLevelState(i int, hover chargenControl) int {
	id := chargenLevel0 + chargenControl(i)
	return preStateBits(c.preLevel == i, hover == id || preFocusControl(c.focus) == id)
}

func preStateBits(chosen, hovered bool) int {
	state := 0
	if chosen {
		state |= 1
	}
	if hovered {
		state |= 2
	}
	return state
}

// preHoverRegion is the mask region code of the control under the pointer,
// -1 when none is a cycle target.
func preHoverRegion(hover chargenControl) int {
	switch {
	case hover >= chargenChoice0 && hover <= chargenChoice3:
		return int(preMaskCode[hover-chargenChoice0])
	case hover >= chargenLevel0 && hover <= chargenLevel2:
		return 20 * int(hover-chargenLevel0+1)
	case hover == chargenBack:
		return preBackMaskCode
	case hover == chargenForward:
		return preForwardMaskCode
	}
	return -1
}

// composePreCreateCycle draws the guided cycle's target (TOWN-519): lon on a
// chosen portrait or level and l otherwise; Amulet or OK at step 2.
func composePreCreateCycle(dst *image.RGBA, c *Chargen, p *ChargenPresentation) {
	k := c.cycleDraw
	if k < 0 || c.preStep < 0 || c.preStep >= len(preCreateCycleSteps) || k >= len(preCreateCycleSteps[c.preStep]) {
		return
	}
	pick := func(states [3]image.Image, chosen bool) image.Image {
		if chosen {
			return states[2]
		}
		return states[1]
	}
	switch c.preStep {
	case 0:
		i := preCycleChoices[k]
		copyNativeKeyed(dst, pick(p.Choices[i], c.preChoiceState(i, chargenNone) == 1), preChoiceOrigin[i], dst.Bounds())
	case 1:
		copyNativeKeyed(dst, pick(p.Levels[k], c.preLevelState(k, chargenNone) == 1), preLevelOrigin[k], dst.Bounds())
	case 2:
		if k == 0 {
			copyNativeKeyed(dst, p.Amulet, preAmuletOrigin, dst.Bounds())
		} else {
			copyNativeKeyed(dst, p.Forward, preForwardOrigin, dst.Bounds())
		}
	}
}

var detailedSkillOrigin = [2][5]image.Point{
	{{88, 93}, {92, 126}, {88, 182}, {84, 225}, {88, 250}},
	{{200, 150}, {72, 165}, {132, 98}, {140, 228}, {136, 158}},
}

var detailedMaskCode = [2][5]uint8{
	{255, 191, 152, 127, 102},
	{127, 102, 255, 152, 191},
}

// chargenColumnDestination is the centre child's own decoded rect,
// (160,0)-(480,480) (TOWN-232), drawn whole with no source crop (1022 spec
// B1): the source column bitmap is 320x480 (chargenassets.go's own
// validation) and the destination is 320 wide, so the two agree with no
// crop or offset displacement. The prior authored crop and displacement
// (chargenColumnSourceCrop = (72,0)-(252,480), destination (300,0)-(480,480))
// are gone; chargenColumnOffset below is now the plain translation the
// source's own (0,0) origin needs to land at the rect's own (160,0).
var chargenColumnDestination = image.Rect(160, 0, 480, 480)

// chargenCardBox is the character card's own decoded 160x242 slot (1022
// spec B3), the left column's lower half beside chargenPlateSeamRegion and
// chargenLowerSeamRegion's own upper-half counterpart on the right. It was
// (0,207)-(300,480): 300 wide because RenderCharacterPanel composed at the
// map screen's own sidebarWidth (pkg/ui/panel.go), which is also why the
// centre column above was cropped and displaced (DIV-189's own "item 2 is
// the root").
var chargenCardBox = image.Rect(0, 238, 160, 480)
var chargenNavBox = TownUpperRegion
var chargenDollBox = TownCharacterRegion

// chargenCardSeamRegion is the character card's own x:[160,176) seam,
// TOWN-312's `+0x6c` source at TOWN-234's `(160,238,16,242)` blit — the
// lower half of the left column's own decoded border pair, chargenPlateSeamRegion
// being the upper half. Undrawn before this story (DIV-189).
var chargenCardSeamRegion = image.Rect(160, 238, 176, 480)

// chargenLowerSeamRegion is townCharacterSeamRegion (townshell.go), aliased
// under this page's own name: TOWN-234's four border blits split the child's
// right edge into a 238-tall segment (the seam beside chargenNavBox) and a
// 242-tall one beside it (DIV-168), and the second is the same x:[464,480),
// y:[238,480) strip townshell.go's own TownCharacterRegion callers close
// through drawTownPane (1021 spec B1).
var chargenLowerSeamRegion = townCharacterSeamRegion

// chargenColumnOffset translates the source column's own (0,0) origin onto
// chargenColumnDestination's own Min (1022 spec B1): the source is drawn
// whole now, with no crop, so this is a plain translation and no longer a
// crop-relative one.
var chargenColumnOffset = chargenColumnDestination.Min

// chargenColumnSkillClip is chargenColumnDestination with its own right edge
// pulled in to TownWideUpperRegion's own x=464 (round-2 adversarial review,
// owner item: the fire icon overlapping the right panel). NavSeam draws at
// x:[464,480), y:[0,238) before the skill loop below, and the skill loop's
// own clip previously reached x=480 (chargenColumnDestination itself), so a
// mage skill patch reaching that far — measured: class 1's own fire patch,
// bbox x[464,472) y[150,202), 349 of 3808 pixels differ from the shipped
// NavSeam alone — painted back over it. The count is reproduced by
// pointing this call's own clip argument back at chargenColumnDestination
// and running the mage/nav_seam subtest of
// TestReleaseChargenDetailedSeamColumnsDrawShippedStrips against either
// install; it reads 349 on both. An earlier revision said 416, which no run
// reproduces (pass-3 adversarial review). Every skill patch's own native
// origin still sits left of x=464 (detailedSkillOrigin plus
// chargenColumnOffset), so this clip only trims a patch that already
// overruns its own column, never the ordinary case.
var chargenColumnSkillClip = image.Rect(
	chargenColumnDestination.Min.X, chargenColumnDestination.Min.Y,
	TownWideUpperRegion.Min.X, chargenColumnDestination.Max.Y)

// chargenPlateBox is the stat plate's own decoded upper-left body slot,
// (0,0)-(160,238) — chargenassets.go's own validated Plate size.
var chargenPlateBox = image.Rect(0, 0, 160, 238)

var chargenPlateSeamRegion = image.Rect(160, 0, 176, 238)

var chargenStatValueBox = [...]image.Rectangle{
	image.Rect(82, 54, 102, 74), image.Rect(82, 86, 102, 106),
	image.Rect(82, 118, 102, 138), image.Rect(82, 150, 102, 170),
}

var chargenStatMinusBox = [...]image.Rectangle{
	image.Rect(107, 54, 127, 74), image.Rect(107, 86, 127, 106),
	image.Rect(107, 118, 127, 138), image.Rect(107, 150, 127, 170),
}

var chargenStatPlusBox = [...]image.Rectangle{
	image.Rect(132, 54, 152, 74), image.Rect(132, 86, 152, 106),
	image.Rect(132, 118, 152, 138), image.Rect(132, 150, 152, 170),
}

var chargenRemainingBox = image.Rect(46, 181, 123, 203)

var detailedFocus = [...]chargenControl{
	chargenSkill0, chargenSkill1, chargenSkill2, chargenSkill3, chargenSkill4,
	chargenStatMinus0, chargenStatPlus0, chargenStatMinus1, chargenStatPlus1,
	chargenStatMinus2, chargenStatPlus2, chargenStatMinus3, chargenStatPlus3,
	chargenBack, chargenReset, chargenPlay,
}

const detailedFocusCount = len(detailedFocus)

func detailedFocusControl(focus int) chargenControl {
	if focus < 0 || focus >= len(detailedFocus) {
		return chargenNone
	}
	return detailedFocus[focus]
}

func detailedControlRegion(id chargenControl) image.Rectangle {
	switch {
	case id >= chargenSkill0 && id <= chargenSkill4:
		// Skill bounds come from their native source patch in
		// detailedControlRect. There is no synthetic full-width skill band.
		return image.Rectangle{}
	case id >= chargenStatMinus0 && id <= chargenStatMinus3:
		i := int(id - chargenStatMinus0)
		return chargenStatMinusBox[i]
	case id >= chargenStatPlus0 && id <= chargenStatPlus3:
		i := int(id - chargenStatPlus0)
		return chargenStatPlusBox[i]
	// Accept, Reset and Back (MENU-139).
	case id == chargenPlay:
		return image.Rect(484, 44, 624, 90)
	case id == chargenReset:
		return image.Rect(484, 91, 624, 137)
	case id == chargenBack:
		return image.Rect(484, 138, 624, 184)
	}
	return image.Rectangle{}
}

func detailedControlRect(c *Chargen, id chargenControl) image.Rectangle {
	r := detailedControlRegion(id)
	if id < chargenSkill0 || id > chargenSkill4 || c == nil || c.setup.PreCreate == nil || c.setup.PreCreate.Art == nil {
		return r
	}
	class := 0
	if len(c.choiceIndex) > 1 && c.choiceIndex[1] != 0 {
		class = 1
	}
	pic := c.setup.PreCreate.Art.Skills[class][int(id-chargenSkill0)][0]
	if pic == nil {
		return image.Rectangle{}
	}
	b := pic.Bounds()
	at := detailedSkillOrigin[class][int(id-chargenSkill0)].Add(chargenColumnOffset)
	return b.Add(at)
}

func detailedControlAt(c *Chargen, p image.Point) chargenControl {
	hasMask := false
	if c != nil && c.setup.PreCreate != nil && c.setup.PreCreate.Art != nil {
		art := c.setup.PreCreate.Art
		class := 0
		if len(c.choiceIndex) > 1 && c.choiceIndex[1] != 0 {
			class = 1
		}
		if mask := art.ColumnMask[class]; mask != nil && p.In(chargenColumnDestination) {
			hasMask = true
			code := mask.ColorIndexAt(p.X-chargenColumnOffset.X, p.Y)
			for skill, want := range detailedMaskCode[class] {
				if code == want {
					return chargenSkill0 + chargenControl(skill)
				}
			}
		}
	}
	for _, id := range detailedFocus {
		if hasMask && id >= chargenSkill0 && id <= chargenSkill4 {
			continue
		}
		if p.In(detailedControlRect(c, id)) {
			return id
		}
	}
	return chargenNone
}

func preControlRect(c *Chargen, id chargenControl) image.Rectangle {
	if id >= chargenLevel0 && id <= chargenLevel2 {
		if c == nil || c.setup.PreCreate == nil || c.setup.PreCreate.Art == nil {
			return image.Rectangle{}
		}
		i := int(id - chargenLevel0)
		if pic := c.setup.PreCreate.Art.Levels[i][0]; pic != nil {
			return pic.Bounds().Add(preLevelOrigin[i])
		}
		return image.Rectangle{}
	}
	region := preControlRegion(id)
	if id < chargenChoice0 || id > chargenChoice3 && id != chargenForward {
		return region
	}
	if c == nil || c.setup.PreCreate == nil || c.setup.PreCreate.Art == nil {
		return image.Rectangle{}
	}
	p := c.setup.PreCreate.Art
	if id == chargenForward {
		if p.Forward == nil {
			return image.Rectangle{}
		}
		return p.Forward.Bounds().Add(preForwardOrigin).Intersect(region)
	}
	i := int(id - chargenChoice0)
	if p.Choices[i][0] == nil {
		return image.Rectangle{}
	}
	b := p.Choices[i][0].Bounds()
	return b.Add(preChoiceOrigin[i])
}

// preControlAt is the pre-create hit-test. The name field is its own child
// (TEXT-075); every other control is the Mask.bmp region code under the point
// (TOWN-520). A setup without a mask falls back to the art rectangles.
func preControlAt(c *Chargen, p image.Point) chargenControl {
	if p.In(preControlRegion(chargenName)) {
		return chargenName
	}
	if c != nil && c.setup.PreCreate != nil && c.setup.PreCreate.Art != nil {
		if mask := c.setup.PreCreate.Art.PreMask; mask != nil {
			if !p.In(mask.Bounds()) {
				return chargenNone
			}
			return preMaskControl(mask.ColorIndexAt(p.X, p.Y))
		}
	}
	for id := chargenLevel0; id <= chargenLevel2; id++ {
		if p.In(preControlRect(c, id)) {
			return id
		}
	}
	for id := chargenChoice0; id <= chargenForward; id++ {
		if p.In(preControlRect(c, id)) {
			return id
		}
	}
	return chargenNone
}

// preMaskControl maps a Mask.bmp region code to its control: levels 20, 40
// and 60, portraits 80..140, amulet 160 and OK 180 (TOWN-520).
func preMaskControl(code uint8) chargenControl {
	switch code {
	case 20, 40, 60:
		return chargenLevel0 + chargenControl(code/20-1)
	case preBackMaskCode:
		return chargenBack
	case preForwardMaskCode:
		return chargenForward
	}
	for choice, want := range preMaskCode {
		if code == want {
			return chargenChoice0 + chargenControl(choice)
		}
	}
	return chargenNone
}

// PreCreateControlAt names the live pre-create control p lands on, and
// reports whether it landed on one — exported for cmd/tippanelcheck (1018,
// round 3), which needs a per-pixel answer to count how many of
// ChargenTipRect's own pixels sit over a control, on
// SchoolSkillRect/SchoolPanelRect's own "exported for the tool that measures
// it" precedent. It returned a bare bool until the story's landing, when the
// tool started reporting coverage per control rather than as one total
// (round-3 adversarial review, D-2): a pixel count does not tell the reader
// which controls the panel covers.
func PreCreateControlAt(c *Chargen, p image.Point) (string, bool) {
	switch id := preControlAt(c, p); id {
	case chargenNone:
		return "", false
	case chargenName:
		return "name", true
	case chargenBack:
		return "back", true
	case chargenForward:
		return "forward", true
	default:
		if id >= chargenLevel0 && id <= chargenLevel2 {
			return fmt.Sprintf("difficulty %d", int(id-chargenLevel0)+1), true
		}
		if id >= chargenChoice0 && id <= chargenChoice3 {
			return fmt.Sprintf("choice %d", int(id-chargenChoice0)), true
		}
		return fmt.Sprintf("control %d", int(id)), true
	}
}

func preFocusControl(focus int) chargenControl {
	if focus >= 7 && focus < 10 {
		return chargenLevel0 + chargenControl(focus-7)
	}
	if focus < 0 || focus > 6 {
		return chargenNone
	}
	return chargenControl(int(chargenName) + focus)
}

func copyNative(dst *image.RGBA, src image.Image, at image.Point, clip image.Rectangle) {
	if dst == nil || src == nil {
		return
	}
	r := src.Bounds().Add(at).Intersect(clip).Intersect(dst.Bounds())
	if r.Empty() {
		return
	}
	draw.Draw(dst, r, src, r.Min.Sub(at).Add(src.Bounds().Min), draw.Src)
}

func copyNativeOver(dst *image.RGBA, src image.Image, at image.Point, clip image.Rectangle) {
	if dst == nil || src == nil {
		return
	}
	r := src.Bounds().Add(at).Intersect(clip).Intersect(dst.Bounds())
	if r.Empty() {
		return
	}
	draw.Draw(dst, r, src, r.Min.Sub(at).Add(src.Bounds().Min), draw.Over)
}

// copyNativeKeyed follows the source screen's one transparency convention:
// only a pure RGB-black source pixel is a hole. All other source pixels,
// including rectangular backgrounds, remain native artwork.
func copyNativeKeyed(dst *image.RGBA, src image.Image, at image.Point, clip image.Rectangle) {
	if dst == nil || src == nil {
		return
	}
	r := src.Bounds().Add(at).Intersect(clip).Intersect(dst.Bounds())
	if r.Empty() {
		return
	}
	b := src.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			sx, sy := x-at.X+b.Min.X, y-at.Y+b.Min.Y
			r, g, b, _ := src.At(sx, sy).RGBA()
			if r == 0 && g == 0 && b == 0 {
				continue
			}
			dst.Set(x, y, src.At(sx, sy))
		}
	}
}

func drawBorder(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	r = r.Intersect(dst.Bounds())
	if r.Empty() {
		return
	}
	for x := r.Min.X; x < r.Max.X; x++ {
		dst.SetRGBA(x, r.Min.Y, c)
		dst.SetRGBA(x, r.Max.Y-1, c)
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		dst.SetRGBA(r.Min.X, y, c)
		dst.SetRGBA(r.Max.X-1, y, c)
	}
}

// ComposeChargenFrame composes c's current stage (pre-create or detailed)
// with no hover or press state. It exists for the install-gated release
// witness in pkg/game (composeChargenPage itself, and the chargenControl
// type its hover/pressed parameters carry, are unexported): pkg/game can
// open a real install and build a *Chargen from it, but cannot see this
// package's own page-composition function directly.
func ComposeChargenFrame(c *Chargen) *image.RGBA {
	return composeChargenPage(c, chargenNone, chargenNone)
}

// ChargenDetailedNavLabelRects returns the three rectangles
// composeChargenDetailedPage draws a text label into over the shipped nav
// art, in Back/Reset/Play order. Meaningful only once c has reached
// DetailedStage; before that each rectangle is empty.
func ChargenDetailedNavLabelRects(c *Chargen) [3]image.Rectangle {
	return [3]image.Rectangle{
		detailedControlRect(c, chargenBack),
		detailedControlRect(c, chargenReset),
		detailedControlRect(c, chargenPlay),
	}
}

// preCreatePromptOrigin and preCreateNameTextOrigin are the top-left corners
// of the pre-create prompt's and name's first cells. Both are left-aligned
// font4 draws (TEXT-077); the page is composed at the 640x480 screen origin.
func preCreatePromptOrigin() image.Point   { return image.Pt(224, 305) }
func preCreateNameTextOrigin() image.Point { return image.Pt(224, 321) }

// The prompt's and the name's full-level inks on the normal palette ramp
// (TEXT-077). The caret takes the name's.
var (
	preCreatePromptInk = color.RGBA{65, 47, 20, 255}
	preCreateNameInk   = color.RGBA{101, 39, 61, 255}
)

// composeChargenPage produces the 640-by-480 pre-create frame. Source images
// are copied at native pixels and clipped to their declared control regions.
func composeChargenPage(c *Chargen, hover, pressed chargenControl, tipState ...TipPanelView) *image.RGBA {
	if c != nil && c.Stage() == DetailedStage {
		return composeChargenDetailedPage(c, hover, pressed, tipState...)
	}
	dst := image.NewRGBA(image.Rect(0, 0, chargenPageW, chargenPageH))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.RGBA{18, 18, 18, 255}}, image.Point{}, draw.Src)
	if c == nil || c.setup.PreCreate == nil {
		return dst
	}
	p := c.setup.PreCreate.Art
	if p == nil {
		return dst
	}
	copyNative(dst, p.Background, image.Point{}, dst.Bounds())
	for i, states := range p.Levels {
		if pic := preStateArt(states, c.preLevelState(i, hover)); pic != nil {
			copyNativeKeyed(dst, pic, preLevelOrigin[i], dst.Bounds())
		}
	}
	for _, i := range preCycleChoices {
		if pic := preStateArt(p.Choices[i], c.preChoiceState(i, hover)); pic != nil {
			copyNativeKeyed(dst, pic, preChoiceOrigin[i], dst.Bounds())
		}
	}
	active := func(id chargenControl) bool {
		return hover == id || pressed == id || preFocusControl(c.focus) == id
	}
	if active(chargenBack) {
		copyNativeKeyed(dst, p.Amulet, preAmuletOrigin, dst.Bounds())
	}
	if active(chargenForward) {
		copyNativeKeyed(dst, p.Forward, preForwardOrigin, dst.Bounds())
	}
	composePreCreateCycle(dst, c, p)
	font := p.NameFont
	if font == nil {
		font = p.Font
	}
	if font != nil {
		promptAt, nameAt := preCreatePromptOrigin(), preCreateNameTextOrigin()
		font.Draw(dst, c.setup.PreCreate.Prompt, promptAt.X, promptAt.Y, preCreatePromptInk)
		name := c.NameText()
		if c.CaretVisible() {
			name += "|"
		}
		font.Draw(dst, name, nameAt.X, nameAt.Y, preCreateNameInk)
	}
	c.paintSparkle(dst)
	composeChargenTip(dst, c, tipState)
	return dst
}

// detailedSkillSelectedHover reports the two switches the owner named for a
// skill cell's picture: whether it reads as the draft's current pick, and
// whether the pointer is over it. They are independent — a skill can be
// selected while unhovered, hovered while unselected, or both at once — and
// are read as a pair rather than folded into one ordinal.
//
// A captured press temporarily owns the sole selected reading: while one
// skill button is captured, it alone reads as selected regardless of the
// committed draft, and every other skill — including the draft's old choice —
// reads as not selected. The draft still names its old skill until release,
// but displaying both would falsely state two selections in one frame.
func detailedSkillSelectedHover(c *Chargen, id, hover, pressed chargenControl) (selected, hovered bool) {
	if pressed >= chargenSkill0 && pressed <= chargenSkill4 {
		return pressed == id, hover == id
	}
	selected = c != nil && id >= chargenSkill0 && id <= chargenSkill4 && len(c.choiceIndex) > 2 &&
		c.choiceIndex[2] == int(id-chargenSkill0)
	return selected, hover == id
}

// detailedSkillPictures is the picture one skill cell draws for each of the
// four (selected, hovered) combinations, named directly rather than ordered
// by an index — the read site picks by the two booleans, never by an
// ordinal. rest is nil by design: the not-selected, not-hovered corner has
// no dedicated picture in the install because the column canvas already
// carries it as its own baked art (raised dark), drawn under the skill loop
// before any patch is composited. Leaving rest nil and skipping the draw for
// that corner is what lets the column's own art show through.
type detailedSkillPictures struct {
	rest           image.Image // not selected, not hovered: nil — the column's own art already shows this corner
	hover          image.Image // not selected, hovered: raised light
	selected       image.Image // selected, hovered: pressed light
	selectedAtRest image.Image // selected, not hovered: pressed dark
}

// pick resolves the (selected, hovered) pair detailedSkillSelectedHover
// computes against the four named pictures. Selection is read first because a
// captured or committed pick outranks hover, matching
// detailedSkillSelectedHover's own precedence. A nil result (the rest corner)
// is deliberate, not a missing asset; the caller must not substitute another
// picture for it.
func (t detailedSkillPictures) pick(selected, hovered bool) image.Image {
	switch {
	case selected && hovered:
		return t.selected
	case selected:
		return t.selectedAtRest
	case hovered:
		return t.hover
	default:
		return t.rest
	}
}

// detailedSkillArt builds one skill's four-picture table from the three
// pictures the install ships per skill, `states[0..2]` in ChargenAssets' own
// load order (pkg/game/chargenassets.go: on, shine_off, shine_on) — plus the
// column's own baked art, which stands in for the fourth corner and is never
// held in this table at all.
//
// Selection chooses raised against pressed; hover chooses dark against
// light. `on.bmp` (states[0]) is pressed dark, so it is the selected corner
// when the pointer has left, not the rest corner. `shine_off.bmp` (states[1])
// is raised light: the hover corner. `shine_on.bmp` (states[2]) is pressed
// light: the selected-and-hovered corner. Verified against the shipped art
// on both lawful installs: each patch differs from the column region under
// it on the large majority of its own drawn pixels (not a flat plate), and
// the sword blade's own highlight sits at the patch's upper edge in the
// column's baked art and in shine_off, and at the lower edge in on and in
// shine_on — raised versus pressed, painted in the pixels themselves.
func detailedSkillArt(states [3]image.Image) detailedSkillPictures {
	return detailedSkillPictures{
		rest:           nil,
		hover:          states[1],
		selected:       states[2],
		selectedAtRest: states[0],
	}
}

// drawChargenCommands draws Accept, Reset and Back (MENU-139): a button
// shows its on picture, with the label 1 px lower, only while pressed and
// hovered, and its off picture otherwise. The labels are font4.
func drawChargenCommands(dst *image.RGBA, p *ChargenPresentation, detail *ChargenDetailed, hover, pressed chargenControl) {
	font := p.NameFont
	if font == nil {
		font = p.Font
	}
	for i, control := range []struct {
		id    chargenControl
		label string
	}{{chargenPlay, detail.Play}, {chargenReset, detail.Reset}, {chargenBack, detail.Back}} {
		r := detailedControlRegion(control.id)
		down := pressed == control.id && hover == control.id
		pic := p.NavButtons[i][0]
		if down {
			pic = p.NavButtons[i][1]
		}
		copyNative(dst, pic, r.Min, r)
		if font == nil {
			continue
		}
		ink := color.RGBA{255, 230, 150, 255}
		if hover == control.id {
			ink = buttonInk(true, true)
		}
		w, h := font.Measure(control.label)
		y := r.Min.Y + (r.Dy()-h)/2
		if down {
			y++
		}
		font.Draw(dst, control.label, r.Min.X+(r.Dx()-w)/2, y, ink)
	}
}

func drawChargenFrame(dst *image.RGBA, r image.Rectangle) {
	draw.Draw(dst, r, &image.Uniform{C: color.RGBA{16, 18, 24, 255}}, image.Point{}, draw.Src)
	drawBorder(dst, r, color.RGBA{138, 116, 70, 255})
}

// chargenMessageRect is where the detailed page's own transient
// hover/refusal line is written (1022 round-2, restoring the channel
// `DIV-192` recorded as dropped): the bottom strip of the doll box, entirely
// right of ChargenTipRect's own x:[160,472) so it is unaffected by the tip
// panel's own open/closed state, and below where the doll figure itself
// draws (chargenDollBox.Min.Add(0,2)). It overlays DollPane.Body's own
// backdrop, the same "shadowed text over existing art, not a new box"
// precedent drawShopMessage already sets for the shop's own message strip
// (shopscreen.go, shopMessageRect).
var chargenMessageRect = image.Rect(
	chargenDollBox.Min.X+4, chargenDollBox.Max.Y-18,
	chargenDollBox.Max.X-4, chargenDollBox.Max.Y-4)

// drawChargenMessage writes msg centred in chargenMessageRect with a
// one-pixel shadow, trimming from the end until it fits — drawShopMessage's
// own rule, restated here rather than shared because the two message rects
// are unrelated sizes and neither page holds a reference to the other's.
func drawChargenMessage(dst *image.RGBA, f *text.Font, msg string) {
	if f == nil || msg == "" {
		return
	}
	s := msg
	for {
		w, _ := f.Measure(s)
		if w <= chargenMessageRect.Dx() || len(s) <= 1 {
			break
		}
		s = s[:len(s)-1]
	}
	w, h := f.Measure(s)
	x := chargenMessageRect.Min.X + (chargenMessageRect.Dx()-w)/2
	y := chargenMessageRect.Min.Y + (chargenMessageRect.Dy()-h)/2
	f.Draw(dst, s, x+1, y+1, shopShadowColor)
	f.Draw(dst, s, x, y, shopTextColor)
}

func drawChargenCentered(dst *image.RGBA, f *text.Font, value string, box image.Rectangle, c color.RGBA) {
	if f == nil || box.Empty() {
		return
	}
	w, h := f.Measure(value)
	f.Draw(dst, value, box.Min.X+(box.Dx()-w)/2, box.Min.Y+(box.Dy()-h)/2, c)
}

// detailedStatButton picks a stat button's art (MENU-138): disabled, then
// pressed light while hovered with the left button held, light while
// hovered, else rest. The nl-on pictures are never drawn.
func detailedStatButton(p *ChargenPresentation, c *Chargen, plus bool, stat int, hover, pressed chargenControl) image.Image {
	if p == nil || stat < 0 || stat >= 4 {
		return nil
	}
	direction := 0
	id := chargenStatMinus0 + chargenControl(stat)
	if plus {
		direction, id = 1, chargenStatPlus0+chargenControl(stat)
	}
	state := 0 // nl-off
	switch {
	case !detailedStatAvailable(c, plus, stat):
		state = 4 // disable
	case hover == id && (pressed == id || c != nil && c.statHeld):
		state = 2 // l-on
	case hover == id:
		state = 1 // l-off
	}
	if pic := p.StatButtons[direction][state]; pic != nil {
		return pic
	}
	return p.StatButtons[direction][0]
}

func detailedStatAvailable(c *Chargen, plus bool, stat int) bool {
	if c == nil || stat < 0 || stat >= len(c.statValue) || stat >= len(c.setup.Stats) {
		return false
	}
	value, row := c.statValue[stat], c.setup.Stats[stat]
	if !plus {
		return value > row.Floor
	}
	if value >= row.Ceiling || value < 0 || value+1 >= len(c.setup.Cost) {
		return false
	}
	return c.setup.Cost[value+1]-c.setup.Cost[value] <= c.Remaining()
}

// composeChargenDetailedPage composes the detailed stage into the same fixed
// virtual frame as pre-create. Every pointer rectangle is computed from the
// same region or clipped source image used below.
func composeChargenDetailedPage(c *Chargen, hover, pressed chargenControl, tipState ...TipPanelView) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, chargenPageW, chargenPageH))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.RGBA{18, 18, 18, 255}}, image.Point{}, draw.Src)
	if c == nil || c.setup.PreCreate == nil || c.setup.PreCreate.Art == nil {
		return dst
	}
	p := c.setup.PreCreate.Art
	// The five regions are deliberately independent surfaces: the source
	// column owns only the middle, while Card, navigation and Doll never share
	// either its pixels or their input geometry.
	//
	// EVERY BODY/SEAM PAIR BELOW DRAWS THROUGH drawTownPane (1022 spec B2):
	// the generator calls the same pane helper `1021`'s town rooms call
	// rather than a private copy of the body-opaque/seam-keyed rule, which
	// is the reuse the owner asked for ("нужно ввести LeftPane, RightPane и
	// везде их использовать"). A body and its own seam still draw at
	// different POINTS in this function where the centre column's own
	// opaque copy would otherwise erase a seam drawn before it (see the
	// column's own comment below); drawTownPane is called once per half in
	// that case, with the other half of its own TownPane value left zero.
	navPane := TownPane{Body: p.NavArt, Seam: p.NavSeam}
	if navPane.Body != nil {
		// The picture's own fourth well (DIV-156) is drawn as shipped and
		// left inert: no control rectangle covers it, so a click there does
		// nothing. Its function is undecoded; overpainting it would cover
		// plaque art with a flat fill that follows no edge in the picture.
		drawTownPane(dst, TownPane{Body: navPane.Body}, chargenNavBox, image.Rectangle{})
	} else {
		drawChargenFrame(dst, chargenNavBox)
	}
	// chargenDollBox's own frame drew here in round 1; it is fully overdrawn
	// by drawTownPane's own DollPane body below (round-2 adversarial review
	// item 4), and the call was dead. Removed rather than left inert.
	platePane := TownPane{Body: p.Plate, Seam: p.PlateSeam}
	drawTownPane(dst, TownPane{Body: platePane.Body}, chargenPlateBox, image.Rectangle{})
	cardPane := TownPane{Seam: p.CardSeam}
	// PlateSeam and CardSeam both draw AFTER the column below, not here with
	// their own bodies: chargenPlateBox and chargenCardBox (x:[0,160)) sit
	// entirely left of chargenColumnDestination (x:[160,480)), so drawing
	// their bodies here is safe, but chargenPlateSeamRegion and
	// chargenCardSeamRegion (x:[160,176)) fall inside it, and the column's own
	// opaque draw.Src copy would erase a seam drawn here first — the same
	// reasoning navPane.Seam's own comment below already gives for its own
	// x:[464,480) strip, TOWN-234's four-border-column order applied to the
	// left column's own two seams as well. B1 widened
	// chargenColumnDestination from x:[300,480) to x:[160,480) (1022 spec
	// B1), which brought x:[160,176) inside it for the first time; drawing
	// both seams here, before that widening landed, erased neither.
	// TestReleaseChargenDetailedSeamColumnsDrawShippedStrips (an install-
	// gated test, not run by go test ./... per golden rule 2) caught the
	// erasure against a real install.
	for stat := 0; stat < len(c.statValue) && stat < 4; stat++ {
		copyNative(dst, detailedStatButton(p, c, false, stat, hover, pressed), chargenStatMinusBox[stat].Min, chargenStatMinusBox[stat])
		drawChargenCentered(dst, p.Font, fmt.Sprintf("%d", c.statValue[stat]), chargenStatValueBox[stat], color.RGBA{255, 255, 255, 255})
		copyNative(dst, detailedStatButton(p, c, true, stat, hover, pressed), chargenStatPlusBox[stat].Min, chargenStatPlusBox[stat])
	}
	drawChargenCentered(dst, p.Font, GroupDigits(int64(c.Remaining())), chargenRemainingBox, color.RGBA{255, 230, 150, 255})
	class := 0
	if len(c.choiceIndex) > 1 && c.choiceIndex[1] != 0 {
		class = 1
	}
	copyNative(dst, p.Columns[class], chargenColumnOffset, chargenColumnDestination)
	// The seam strips draw after the column, not with their own bodies above:
	// the column's own destination (chargenColumnDestination) reaches both
	// x:[160,176) and x:[464,480), and an opaque draw.Src copy at the
	// column's own step would otherwise erase whatever was drawn at either
	// strip first. TOWN-234 raw-disassembles the original's own order the
	// same way — one background blit over the child's full rect, then the
	// four border-column blits on top of it — so every strip belongs after
	// the column here (DIV-168).
	if platePane.Seam != nil {
		drawTownPane(dst, TownPane{Seam: platePane.Seam}, image.Rectangle{}, chargenPlateSeamRegion)
	}
	if cardPane.Seam != nil {
		// The character card's own x:[160,176) seam (TOWN-312's `+0x6c`
		// source; DIV-189). The card's own body slot, fullstatsl.bmp, does
		// not draw here as a bare TownPane body: it composes INTO the card
		// below through CompactPanelLayout's own Background field, which is
		// what "drawn on fullstatsl" (1022 contract B3) means, so a
		// redundant bare draw here would only be immediately overpainted.
		drawTownPane(dst, TownPane{Seam: cardPane.Seam}, image.Rectangle{}, chargenCardSeamRegion)
	}
	if navPane.Seam != nil {
		// Closes TownWideUpperRegion's own 16-column gap beside chargenNavBox,
		// the same strip the town shell's school and tavern rooms close with
		// (DIV-166, DIV-168). Seam is keyed at load, see the plate seam above.
		drawTownPane(dst, TownPane{Seam: navPane.Seam}, image.Rectangle{}, townUpperSeamRegion)
	}
	for skill := 0; skill < 5; skill++ {
		id := chargenSkill0 + chargenControl(skill)
		selected, hovered := detailedSkillSelectedHover(c, id, hover, pressed)
		// A nil pick is the rest corner's deliberate absence, not a missing
		// asset (see detailedSkillPictures): the column already drew this
		// corner's own art above, so nothing more is composited here.
		if pic := detailedSkillArt(p.Skills[class][skill]).pick(selected, hovered); pic != nil {
			copyNativeKeyed(dst, pic, detailedSkillOrigin[class][skill].Add(chargenColumnOffset), chargenColumnSkillClip)
		}
	}
	if k := c.cycleDraw; k >= 0 && k < 5 {
		// The skill cycle draws shine_on on the chosen skill and shine_off
		// on the others (TOWN-522).
		pic := p.Skills[class][k][1]
		if len(c.choiceIndex) > 2 && c.choiceIndex[2] == k {
			pic = p.Skills[class][k][2]
		}
		copyNativeKeyed(dst, pic, detailedSkillOrigin[class][k].Add(chargenColumnOffset), chargenColumnSkillClip)
	}
	if detail := c.setup.Detailed; detail != nil {
		drawChargenCommands(dst, p, detail, hover, pressed)
	}
	preview := c.Preview()
	if p.Font != nil {
		// CompactPanelLayout, not AuthoredPanelLayout (1022 spec B3): the
		// 160x242 card composed for chargenCardBox, with the shipped
		// fullstatsl.bmp (CardBackground) as its own background rather than
		// an authored fill — "drawn on fullstatsl" (contract). A nil
		// CardBackground (a load failure) falls back to CompactPanelLayout's
		// own authored fill, the same degrade every other optional source on
		// this page already has.
		first := text.CapturedLen()
		card := RenderCharacterPanel(CompactPanelLayout(p.CardBackground), p.Font, preview.Subject)
		if card != nil {
			copyNative(dst, card, chargenCardBox.Min, chargenCardBox)
			// The card's glyphs were placed on its own canvas.
			text.ShiftCaptured(first, chargenCardBox.Min.X, chargenCardBox.Min.Y)
		}
	}
	if p.DollPane.Body != nil {
		drawTownPane(dst, p.DollPane, chargenDollBox, chargenLowerSeamRegion)
	} else {
		draw.Draw(dst, chargenDollBox.Inset(2), &image.Uniform{C: color.RGBA{18, 18, 18, 255}}, image.Point{}, draw.Src)
	}
	if preview.Doll == nil {
		if p.Font != nil {
			p.Font.Draw(dst, "PREVIEW UNAVAILABLE", 482, 250, color.RGBA{255, 230, 150, 255})
		}
	} else {
		copyNativeOver(dst, preview.Doll, chargenDollBox.Min.Add(image.Pt(0, 2)), chargenDollBox)
	}
	// The message strip draws over the doll box's own backdrop, after the doll
	// and before the tip panel (1022 round-2): restoring DIV-192's dropped
	// channel does not move the tip panel's own draw order, and the strip sits
	// entirely right of ChargenTipRect so neither can cover the other
	// regardless of which draws last.
	if p.Font != nil {
		drawChargenMessage(dst, p.Font, c.message)
	}
	// The tip panel draws last (1022 spec B5), on composeChargenPage's own
	// pre-create precedent (ComposeTipPanel is that function's own last
	// call too): TipPanel() now answers on DetailedStage as well as
	// PreCreateStage, and the panel's own rect, ChargenTipRect
	// (160,280)-(472,480), reaches into the centre column and the skill
	// icons this function draws above, so it must compose over them rather
	// than under.
	composeChargenTip(dst, c, tipState)
	return dst
}

func composeChargenTip(dst *image.RGBA, c *Chargen, state []TipPanelView) {
	v := c.TipPanel()
	if len(state) != 0 {
		v = state[0]
	}
	ComposeTipPanel(dst, v)
}

// DetailedAttributeBoxes are attribute row stat's plate, value, lower and
// raise rectangles.
func DetailedAttributeBoxes(stat int) (plate, value, lower, raise image.Rectangle, ok bool) {
	if stat < 0 || stat >= len(chargenStatValueBox) {
		return image.Rectangle{}, image.Rectangle{}, image.Rectangle{}, image.Rectangle{}, false
	}
	v := chargenStatValueBox[stat]
	return image.Rect(8, v.Min.Y, v.Min.X, v.Max.Y), v, chargenStatMinusBox[stat], chargenStatPlusBox[stat], true
}
