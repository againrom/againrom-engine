package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"time"

	"againrom/pkg/render/text"
)

// ChargenPresentation is all immutable artwork and wording resolved from one
// install, with the description it was resolved from. UI receives pixels and
// bytes only; it never opens an archive.
type ChargenPresentation struct {
	// Layout is the generator description every coordinate, mask byte,
	// timing, key and text slot of both pages is read from.
	Layout     *GeneratorDescription
	Background image.Image
	PreMask    *image.Paletted
	Amulet     image.Image
	Forward    image.Image
	Plate      image.Image
	// Choices and Levels hold each hero's and level's state art, in the
	// order the description lists it.
	Choices     [4][3]image.Image
	Levels      [3][3]image.Image
	Sparkles    []*image.RGBA
	Loops       [][]image.Image
	Columns     [2]image.Image
	ColumnMask  [2]*image.Paletted
	Skills      [generatorClasses][generatorSkills][3]image.Image
	StatButtons [2][statArtStates]image.Image // minus/plus × rest, hover, down, unused, disabled
	// NavArt is the command panel's background and NavButtons are Accept,
	// Reset, Back and Restore, off and on.
	NavArt         image.Image
	NavButtons     [generatorCommandsMax][2]image.Image
	NavSeam        image.Image
	PlateSeam      image.Image
	DollPane       TownPane
	CardSeam       image.Image
	CardBackground *image.RGBA
	Font           *text.Font
	// ValueFont draws the four statistic values. Nil falls back to Font.
	ValueFont *text.Font
	// NameFont draws the pre-create prompt and name. Nil falls back to Font.
	NameFont *text.Font
	// TipFont is the install's font, which the tip panels draw in when the
	// description's panel font is "install".
	TipFont *text.Font
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
	chargenRestore
)

// art is the presentation the model draws from, or nil.
func (c *Chargen) art() *ChargenPresentation {
	if c == nil || c.setup.PreCreate == nil {
		return nil
	}
	return c.setup.PreCreate.Art
}

// layout is the generator description, or nil for a setup without one.
func (c *Chargen) layout() *GeneratorDescription {
	if p := c.art(); p != nil {
		return p.Layout
	}
	return nil
}

// pageBounds is the frame both pages compose in.
func (l *GeneratorDescription) pageBounds() image.Rectangle {
	if l == nil {
		return image.Rectangle{}
	}
	return image.Rectangle{Max: l.Page.Size.Pt()}
}

// newPage is a frame filled with the page backdrop.
func newPage(l *GeneratorDescription) *image.RGBA {
	dst := image.NewRGBA(l.pageBounds())
	if l != nil {
		draw.Draw(dst, dst.Bounds(), &image.Uniform{C: l.Page.Backdrop.RGBA()}, image.Point{}, draw.Src)
	}
	return dst
}

// generatorState is one control's state as its state art reads it.
type generatorState struct{ selected, hovered, neighbourHovered, neighbourSelected bool }

func stateMatches(want *bool, got bool) bool { return want == nil || *want == got }

func (w GeneratorWhen) holds(s generatorState) bool {
	return stateMatches(w.Selected, s.selected) && stateMatches(w.Hovered, s.hovered) &&
		stateMatches(w.NeighbourHovered, s.neighbourHovered) && stateMatches(w.NeighbourSelected, s.neighbourSelected)
}

// drawStateArt draws the first state picture whose condition holds.
func drawStateArt(dst *image.RGBA, arts []GeneratorStateArt, pics [3]image.Image, s generatorState) {
	for k, a := range arts {
		if k >= len(pics) {
			return
		}
		if a.When.holds(s) {
			copyNativeKeyed(dst, pics[k], a.At.Pt(), dst.Bounds())
			return
		}
	}
}

// preFocusControl is the pre-create control the keyboard focus names.
func (c *Chargen) preFocusControl() chargenControl {
	l := c.layout()
	if l == nil || c.focus < 0 || c.focus >= len(l.PreCreate.Focus) {
		return chargenNone
	}
	id, _ := generatorControlNamed(l.PreCreate.Focus[c.focus])
	return id
}

// detailedFocusControl is the detail-page control the keyboard focus names.
func (c *Chargen) detailedFocusControl() chargenControl {
	l := c.layout()
	if l == nil || c.focus < 0 || c.focus >= len(l.Detail.Focus) {
		return chargenNone
	}
	id, _ := generatorControlNamed(l.Detail.Focus[c.focus])
	return id
}

// focusIndex is the focus position of a control on the showing page, or -1.
func (c *Chargen) focusIndex(id chargenControl) int {
	l := c.layout()
	if l == nil {
		return -1
	}
	list := l.PreCreate.Focus
	if c.stage == DetailedStage {
		list = l.Detail.Focus
	}
	for i, name := range list {
		if got, _ := generatorControlNamed(name); got == id {
			return i
		}
	}
	return -1
}

// heroState is hero i's state: chosen, under the pointer or the keyboard
// focus, and the same two facts of the hero beside it.
func (c *Chargen) heroState(i int, hover chargenControl) generatorState {
	focus := c.preFocusControl()
	id := chargenChoice0 + chargenControl(i)
	s := generatorState{selected: c.preChoice == i, hovered: hover == id || focus == id}
	if l := c.layout(); l != nil && i < len(l.PreCreate.Heroes) {
		if n := l.PreCreate.Heroes[i].Neighbour; n != nil {
			nid := chargenChoice0 + chargenControl(*n)
			s.neighbourHovered, s.neighbourSelected = hover == nid || focus == nid, c.preChoice == *n
		}
	}
	return s
}

func (c *Chargen) levelState(i int, hover chargenControl) generatorState {
	id := chargenLevel0 + chargenControl(i)
	return generatorState{selected: c.preLevel == i, hovered: hover == id || c.preFocusControl() == id}
}

// preControlRegion is a pre-create control's own rectangle: the name field,
// and the regions that clip Back's and Forward's art.
func preControlRegion(l *GeneratorDescription, id chargenControl) image.Rectangle {
	if l == nil {
		return image.Rectangle{}
	}
	p := &l.PreCreate
	switch id {
	case chargenName:
		return p.Name.Rect.Rectangle()
	case chargenBack:
		if p.Back.Region != nil {
			return p.Back.Region.Rectangle()
		}
	case chargenForward:
		if p.Forward.Region != nil {
			return p.Forward.Region.Rectangle()
		}
	}
	return image.Rectangle{}
}

// preHoverRegion is the mask byte of the control under the pointer, -1 when
// none.
func (c *Chargen) preHoverRegion(hover chargenControl) int {
	l := c.layout()
	if l == nil {
		return -1
	}
	p := &l.PreCreate
	switch {
	case hover >= chargenChoice0 && hover <= chargenChoice3:
		return p.Heroes[hover-chargenChoice0].Mask
	case hover >= chargenLevel0 && hover <= chargenLevel2:
		return p.Levels[hover-chargenLevel0].Mask
	case hover == chargenBack:
		return p.Back.Mask
	case hover == chargenForward:
		return p.Forward.Mask
	}
	return -1
}

// composePreCreateCycle draws the guided cycle's lit target. A lit hero or
// level draws as hovered, with its selection read only while the keyboard
// focus is elsewhere; a lit hero's neighbour draws as beside a hovered hero.
// A lit Back or Forward draws its art.
func composePreCreateCycle(dst *image.RGBA, c *Chargen, p *ChargenPresentation) {
	l, k := c.layout(), c.cycleDraw
	if l == nil || k < 0 || c.preStep < 0 || c.preStep >= len(l.Tips.PreCreateCycle) || k >= len(l.Tips.PreCreateCycle[c.preStep]) {
		return
	}
	id := preMaskControl(l, uint8(l.Tips.PreCreateCycle[c.preStep][k]))
	focus := c.preFocusControl()
	switch {
	case id >= chargenChoice0 && id <= chargenChoice3:
		i := int(id - chargenChoice0)
		s := c.heroState(i, chargenNone)
		s.selected, s.hovered = s.selected && focus != id, true
		drawStateArt(dst, l.PreCreate.Heroes[i].Art, p.Choices[i], s)
		if n := l.PreCreate.Heroes[i].Neighbour; n != nil {
			ns := c.heroState(*n, chargenNone)
			ns.neighbourHovered = true
			drawStateArt(dst, l.PreCreate.Heroes[*n].Art, p.Choices[*n], ns)
		}
	case id >= chargenLevel0 && id <= chargenLevel2:
		i := int(id - chargenLevel0)
		s := c.levelState(i, chargenNone)
		s.selected, s.hovered = s.selected && focus != id, true
		drawStateArt(dst, l.PreCreate.Levels[i].Art, p.Levels[i], s)
	case id == chargenBack:
		copyNativeKeyed(dst, p.Amulet, l.PreCreate.Back.Art.At.Pt(), dst.Bounds())
	case id == chargenForward:
		copyNativeKeyed(dst, p.Forward, l.PreCreate.Forward.Art.At.Pt(), dst.Bounds())
	}
}

// columnClass is the class whose column the detail page shows.
func (c *Chargen) columnClass() int {
	if c != nil && len(c.choiceIndex) > 1 && c.choiceIndex[1] != 0 {
		return 1
	}
	return 0
}

// selectableSkills is how many of the showing class's skills are drawn, hit
// and cycled.
func (c *Chargen) selectableSkills() int {
	l := c.layout()
	if l == nil {
		return generatorSkills
	}
	return l.Detail.Classes[c.columnClass()].Selectable
}

// detailedHitOrder is the order the detail page tests rectangles in.
var detailedHitOrder = func() []chargenControl {
	var out []chargenControl
	for i := 0; i < generatorSkills; i++ {
		out = append(out, chargenSkill0+chargenControl(i))
	}
	for i := 0; i < generatorStats; i++ {
		out = append(out, chargenStatMinus0+chargenControl(i), chargenStatPlus0+chargenControl(i))
	}
	return append(out, chargenBack, chargenReset, chargenRestore, chargenPlay)
}()

func detailedControlRegion(l *GeneratorDescription, id chargenControl) image.Rectangle {
	if l == nil {
		return image.Rectangle{}
	}
	s := &l.Detail.Stats
	switch {
	case id >= chargenStatMinus0 && id <= chargenStatMinus3:
		return s.Minus[id-chargenStatMinus0].Rectangle()
	case id >= chargenStatPlus0 && id <= chargenStatPlus3:
		return s.Plus[id-chargenStatPlus0].Rectangle()
	case id == chargenPlay, id == chargenReset, id == chargenBack, id == chargenRestore:
		for _, cmd := range l.Detail.Commands {
			if role, _ := generatorControlNamed(cmd.Role); role == id {
				return cmd.Rect.Rectangle()
			}
		}
	}
	return image.Rectangle{}
}

// detailedControlRect is a control's rectangle; a skill's is its source patch
// at its own origin in the column.
func detailedControlRect(c *Chargen, id chargenControl) image.Rectangle {
	l := c.layout()
	r := detailedControlRegion(l, id)
	if id < chargenSkill0 || id > chargenSkill4 || l == nil || c.art() == nil {
		return r
	}
	class, skill := c.columnClass(), int(id-chargenSkill0)
	if skill >= c.selectableSkills() {
		return image.Rectangle{}
	}
	pic := c.art().Skills[class][skill][0]
	if pic == nil {
		return image.Rectangle{}
	}
	at := l.Detail.Classes[class].Skills[skill].At.Pt().Add(l.Detail.ColumnRect.Rectangle().Min)
	return pic.Bounds().Add(at)
}

func detailedControlAt(c *Chargen, p image.Point) chargenControl {
	hasMask := false
	if l, art := c.layout(), c.art(); l != nil && art != nil {
		class := c.columnClass()
		column := l.Detail.ColumnRect.Rectangle()
		if mask := art.ColumnMask[class]; mask != nil && p.In(column) {
			hasMask = true
			code := mask.ColorIndexAt(p.X-column.Min.X, p.Y-column.Min.Y)
			for skill, s := range l.Detail.Classes[class].Skills[:c.selectableSkills()] {
				if int(code) == s.Mask {
					return chargenSkill0 + chargenControl(skill)
				}
			}
		}
	}
	for _, id := range detailedHitOrder {
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
	l, p := c.layout(), c.art()
	if l == nil || p == nil {
		return image.Rectangle{}
	}
	switch {
	case id >= chargenLevel0 && id <= chargenLevel2:
		i := int(id - chargenLevel0)
		if pic := p.Levels[i][0]; pic != nil {
			return pic.Bounds().Add(l.PreCreate.Levels[i].Art[0].At.Pt())
		}
		return image.Rectangle{}
	case id >= chargenChoice0 && id <= chargenChoice3:
		i := int(id - chargenChoice0)
		if pic := p.Choices[i][0]; pic != nil {
			return pic.Bounds().Add(l.PreCreate.Heroes[i].Art[0].At.Pt())
		}
		return image.Rectangle{}
	case id == chargenForward:
		if p.Forward == nil {
			return image.Rectangle{}
		}
		return p.Forward.Bounds().Add(l.PreCreate.Forward.Art.At.Pt()).Intersect(preControlRegion(l, id))
	}
	return preControlRegion(l, id)
}

// preControlAt is the pre-create hit-test. The name field is its own child;
// every other control is the mask byte under the point. A setup without a
// mask falls back to the art rectangles.
func preControlAt(c *Chargen, p image.Point) chargenControl {
	l := c.layout()
	if p.In(preControlRegion(l, chargenName)) {
		return chargenName
	}
	if art := c.art(); art != nil && art.PreMask != nil {
		if !p.In(art.PreMask.Bounds()) {
			return chargenNone
		}
		return preMaskControl(l, art.PreMask.ColorIndexAt(p.X, p.Y))
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

// preMaskControl maps a mask byte to its control. A byte no control names is
// inert.
func preMaskControl(l *GeneratorDescription, code uint8) chargenControl {
	if l == nil {
		return chargenNone
	}
	p := &l.PreCreate
	for i, lv := range p.Levels {
		if lv.Mask == int(code) {
			return chargenLevel0 + chargenControl(i)
		}
	}
	switch int(code) {
	case p.Back.Mask:
		return chargenBack
	case p.Forward.Mask:
		return chargenForward
	}
	for i, h := range p.Heroes {
		if h.Mask == int(code) {
			return chargenChoice0 + chargenControl(i)
		}
	}
	return chargenNone
}

// PreCreateControlAt names the live pre-create control p lands on, and
// reports whether it landed on one. Tools that measure coverage per control
// read it.
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

// ComposeChargenFrame composes c's current stage with no hover or press
// state, for the install-gated release witnesses in pkg/game.
func ComposeChargenFrame(c *Chargen) *image.RGBA {
	return composeChargenPage(c, chargenNone, chargenNone)
}

// ChargenDetailedNavLabelRects returns the rectangles the detailed page draws
// a command label into, in Back, Reset, Play order and then Restore when the
// description lists it. Meaningful only once c has reached DetailedStage.
func ChargenDetailedNavLabelRects(c *Chargen) []image.Rectangle {
	out := []image.Rectangle{
		detailedControlRect(c, chargenBack),
		detailedControlRect(c, chargenReset),
		detailedControlRect(c, chargenPlay),
	}
	if l := c.layout(); l != nil && len(l.Detail.Commands) == generatorCommandsMax {
		out = append(out, detailedControlRect(c, chargenRestore))
	}
	return out
}

// promptOrigin is where the pre-create prompt's first cell draws: its own
// origin left-aligned, or its width before the origin right-aligned.
func promptOrigin(l *GeneratorDescription, font *text.Font, prompt string) image.Point {
	at := l.PreCreate.Name.Prompt.At.Pt()
	if l.PreCreate.Name.Prompt.Align == "right" && font != nil {
		w, _ := font.Measure(prompt)
		at.X -= w
	}
	return at
}

// composeChargenPage produces the pre-create frame. Source images are copied
// at native pixels and clipped to their declared control regions.
func composeChargenPage(c *Chargen, hover, pressed chargenControl, tipState ...TipPanelView) *image.RGBA {
	if c != nil && c.Stage() == DetailedStage {
		return composeChargenDetailedPage(c, hover, pressed, tipState...)
	}
	l, p := c.layout(), c.art()
	dst := newPage(l)
	if l == nil || p == nil {
		return dst
	}
	pre := &l.PreCreate
	copyNative(dst, p.Background, pre.Background.At.Pt(), dst.Bounds())
	c.paintLoops(dst)
	for i := range pre.Levels {
		drawStateArt(dst, pre.Levels[i].Art, p.Levels[i], c.levelState(i, hover))
	}
	for _, i := range pre.HeroOrder {
		drawStateArt(dst, pre.Heroes[i].Art, p.Choices[i], c.heroState(i, hover))
	}
	focus := c.preFocusControl()
	active := func(id chargenControl) bool { return hover == id || pressed == id || focus == id }
	if active(chargenBack) {
		copyNativeKeyed(dst, p.Amulet, pre.Back.Art.At.Pt(), dst.Bounds())
	}
	if active(chargenForward) {
		copyNativeKeyed(dst, p.Forward, pre.Forward.Art.At.Pt(), dst.Bounds())
	}
	composePreCreateCycle(dst, c, p)
	font := p.NameFont
	if font == nil {
		font = p.Font
	}
	if font != nil {
		promptAt, nameAt := promptOrigin(l, font, c.setup.PreCreate.Prompt), pre.Name.TextAt.Pt()
		font.Draw(dst, c.setup.PreCreate.Prompt, promptAt.X, promptAt.Y, pre.Name.Prompt.Ink.RGBA())
		name := c.NameText()
		if c.CaretVisible() {
			name += "|"
		}
		font.Draw(dst, name, nameAt.X, nameAt.Y, pre.Name.Ink.RGBA())
	}
	c.paintSparkle(dst)
	composeChargenTip(dst, c, tipState)
	return dst
}

// detailedSkillSelectedHover reports whether a skill cell reads as the
// draft's pick and whether the pointer is over it. A captured skill press
// owns the sole selected reading until its release.
func detailedSkillSelectedHover(c *Chargen, id, hover, pressed chargenControl) (selected, hovered bool) {
	if pressed >= chargenSkill0 && pressed <= chargenSkill4 {
		return pressed == id, hover == id
	}
	selected = c != nil && id >= chargenSkill0 && id <= chargenSkill4 && len(c.choiceIndex) > 2 &&
		c.choiceIndex[2] == int(id-chargenSkill0)
	return selected, hover == id
}

// detailedSkillPictures is the picture one skill cell draws for each
// (selected, hovered) pair. rest is nil: the column's own art is that corner.
type detailedSkillPictures struct {
	rest           image.Image
	hover          image.Image
	selected       image.Image
	selectedAtRest image.Image
}

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

// detailedSkillArt names a skill's three pictures in the presentation's load
// order: selected at rest, hover, selected and hovered.
func detailedSkillArt(states [3]image.Image) detailedSkillPictures {
	return detailedSkillPictures{hover: states[1], selected: states[2], selectedAtRest: states[0]}
}

// chargenCommandButtons are the description's command buttons: a button shows
// its on picture, with the label 1 px lower, only while pressed and hovered.
func chargenCommandButtons(c *Chargen, detail *ChargenDetailed, hover, pressed chargenControl) []pushButton {
	out := make([]pushButton, len(c.layout().Detail.Commands))
	l, p := c.layout(), c.art()
	ink := plaqueCommandInk
	ink.Rest = l.Detail.CommandInk.RGBA()
	labels := map[chargenControl]string{chargenPlay: detail.Play, chargenReset: detail.Reset, chargenBack: detail.Back, chargenRestore: detail.Restore}
	for i, cmd := range l.Detail.Commands {
		id, _ := generatorControlNamed(cmd.Role)
		r := cmd.Rect.Rectangle()
		out[i] = pushButton{Rect: r, Hover: hover == id, Pressed: pressed == id, Inside: hover == id,
			Face: &plaqueFace{Pictures: plaquePair(p.NavButtons[i]), Ink: ink, Sink: plaqueSink,
				Captions: []plaqueCaption{{Text: labels[id], Rect: r}}}}
	}
	return out
}

// drawChargenMessage writes msg centred in the message strip with a
// one-pixel shadow, trimmed from the end to fit.
func drawChargenMessage(dst *image.RGBA, f *text.Font, box image.Rectangle, msg string) {
	if f == nil || msg == "" {
		return
	}
	s := msg
	for {
		w, _ := f.Measure(s)
		if w <= box.Dx() || len(s) <= 1 {
			break
		}
		s = s[:len(s)-1]
	}
	w, h := f.Measure(s)
	x := box.Min.X + (box.Dx()-w)/2
	y := box.Min.Y + (box.Dy()-h)/2
	f.Draw(dst, s, x+1, y+1, shopShadowColor)
	f.Draw(dst, s, x, y, shopTextColor)
}

// drawChargenValue draws one statistic value at the description's offset
// inside its box, or centred when it names none (TOWN-534).
func drawChargenValue(dst *image.RGBA, p *ChargenPresentation, st *GeneratorStats, stat int, value string) {
	f := chargenValueFont(p)
	box := st.Value[stat].Rectangle()
	if st.ValueAt == nil || f == nil {
		drawChargenCentered(dst, f, value, box, st.ValueInk.RGBA())
		return
	}
	at := box.Min.Add(st.ValueAt.Pt())
	drawChargenShadowed(dst, f, value, at, st.ValueInk.RGBA(), st.ValueShadow)
}

// drawChargenPool draws the remaining-points counter centred on the
// description's point, or centred in the pool box when it names none
// (TOWN-534).
func drawChargenPool(dst *image.RGBA, p *ChargenPresentation, st *GeneratorStats, value string) {
	f := chargenValueFont(p)
	if st.PoolAt == nil || f == nil {
		drawChargenCentered(dst, f, value, st.Pool.Rectangle(), st.PoolInk.RGBA())
		return
	}
	w, _ := f.Measure(value)
	at := st.PoolAt.Pt().Sub(image.Pt(w/2, 0))
	drawChargenShadowed(dst, f, value, at, st.PoolInk.RGBA(), st.ValueShadow)
}

func chargenValueFont(p *ChargenPresentation) *text.Font {
	if p.ValueFont != nil {
		return p.ValueFont
	}
	return p.Font
}

// drawChargenShadowed draws s at at, over its shadow one pixel right and down
// when shadow is named.
func drawChargenShadowed(dst *image.RGBA, f *text.Font, s string, at image.Point, ink color.RGBA, shadow *GeneratorInk) {
	if shadow != nil {
		f.Draw(dst, s, at.X+1, at.Y+1, shadow.RGBA())
	}
	f.Draw(dst, s, at.X, at.Y, ink)
}

// CardView is the detailed page's statistics card as the town's card builder
// draws it: the card pane, its seam, the card font and the preview subject.
func (c *Chargen) CardView() TownCharacterView {
	if c == nil {
		return TownCharacterView{}
	}
	l, p := c.layout(), c.art()
	v := TownCharacterView{Subject: c.preview.Subject, HasSubject: true, Statistics: true}
	if l == nil || p == nil {
		return v
	}
	v.PaneRect, v.CardOffset = l.Detail.Card.Rect.Rectangle(), l.Detail.CardOffset.Pt()
	v.Font, v.CardFont = p.Font, p.Font
	// A nil picture must stay a nil image, not a nil picture inside one.
	if p.CardBackground != nil {
		v.StatsPane.Body = p.CardBackground
	}
	v.StatsPane.Seam = p.CardSeam
	return v
}

func drawChargenCentered(dst *image.RGBA, f *text.Font, value string, box image.Rectangle, c color.RGBA) {
	if f == nil || box.Empty() {
		return
	}
	w, h := f.Measure(value)
	f.Draw(dst, value, box.Min.X+(box.Dx()-w)/2, box.Min.Y+(box.Dy()-h)/2, c)
}

// detailedStatButton is a stat button: disabled, then pressed light while
// hovered with the left button held, light while hovered, else rest.
func detailedStatButton(c *Chargen, plus bool, stat int, hover, pressed chargenControl) pushButton {
	direction, id := 0, chargenStatMinus0+chargenControl(stat)
	if plus {
		direction, id = 1, chargenStatPlus0+chargenControl(stat)
	}
	pictures := c.art().StatButtons[direction]
	return pushButton{Rect: detailedControlRegion(c.layout(), id), Hover: hover == id, Inside: hover == id,
		Pressed: pressed == id || c.statHeld, Disabled: !detailedStatAvailable(c, plus, stat),
		Face: &plaqueFace{Pictures: [plaqueStates]image.Image{
			plaqueRest: pictures[statArtRest], plaqueHover: pictures[statArtHover], plaqueDown: pictures[statArtDown],
			plaqueDisabled: pictures[statArtDisabled]}}}
}

// The stat button pictures in load order.
const (
	statArtRest = iota
	statArtHover
	statArtDown
	statArtUnused
	statArtDisabled
	statArtStates
)

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

// paneRect is a description pane's rectangle, empty when it names none.
func paneRect(p *GeneratorPane) image.Rectangle {
	if p == nil {
		return image.Rectangle{}
	}
	return p.Rect.Rectangle()
}

// composeChargenDetailedPage composes the detailed stage into the same frame
// as pre-create. Bodies draw first, then the column, then every seam over
// the column, then the skill patches, commands, card, figure, message and
// tip panel.
func composeChargenDetailedPage(c *Chargen, hover, pressed chargenControl, tipState ...TipPanelView) *image.RGBA {
	l, p := c.layout(), c.art()
	dst := newPage(l)
	if l == nil || p == nil {
		return dst
	}
	d := &l.Detail
	if p.NavArt != nil {
		drawTownPane(dst, TownPane{Body: p.NavArt}, d.Nav.Rect.Rectangle(), image.Rectangle{})
	} else {
		drawFrame(dst, panelFrame(d.Nav.Rect.Rectangle(), d.NavFrame.Fill.RGBA(), d.NavFrame.Edge.RGBA()))
	}
	drawTownPane(dst, TownPane{Body: p.Plate}, d.Plate.Rect.Rectangle(), image.Rectangle{})
	for stat := 0; stat < len(c.statValue) && stat < generatorStats; stat++ {
		drawPushButton(dst, nil, detailedStatButton(c, false, stat, hover, pressed))
		drawChargenValue(dst, p, &d.Stats, stat, fmt.Sprintf("%d", c.statValue[stat]))
		drawPushButton(dst, nil, detailedStatButton(c, true, stat, hover, pressed))
	}
	drawChargenPool(dst, p, &d.Stats, GroupDigits(int64(c.Remaining())))
	class := c.columnClass()
	column := d.ColumnRect.Rectangle()
	copyNative(dst, p.Columns[class], column.Min, column)
	if p.PlateSeam != nil {
		drawTownPane(dst, TownPane{Seam: p.PlateSeam}, image.Rectangle{}, paneRect(d.PlateSeam))
	}
	if p.CardSeam != nil {
		drawTownPane(dst, TownPane{Seam: p.CardSeam}, image.Rectangle{}, paneRect(d.CardSeam))
	}
	if p.NavSeam != nil {
		drawTownPane(dst, TownPane{Seam: p.NavSeam}, image.Rectangle{}, paneRect(d.NavSeam))
	}
	clip, skills := d.SkillClip.Rectangle(), d.Classes[class].Skills
	for skill := 0; skill < c.selectableSkills(); skill++ {
		id := chargenSkill0 + chargenControl(skill)
		selected, hovered := detailedSkillSelectedHover(c, id, hover, pressed)
		if pic := detailedSkillArt(p.Skills[class][skill]).pick(selected, hovered); pic != nil {
			copyNativeKeyed(dst, pic, skills[skill].At.Pt().Add(column.Min), clip)
		}
	}
	if k := c.cycleDraw; k >= 0 && k < c.selectableSkills() {
		// The skill cycle draws the selected-and-hovered picture on the chosen
		// skill and the hover picture on the others.
		pic := p.Skills[class][k][1]
		if len(c.choiceIndex) > 2 && c.choiceIndex[2] == k {
			pic = p.Skills[class][k][2]
		}
		copyNativeKeyed(dst, pic, skills[k].At.Pt().Add(column.Min), clip)
	}
	if detail := c.setup.Detailed; detail != nil {
		font := p.NameFont
		if font == nil {
			font = p.Font
		}
		for _, b := range chargenCommandButtons(c, detail, hover, pressed) {
			drawPushButton(dst, font, b)
		}
	}
	preview := c.Preview()
	if p.Font != nil {
		drawCharacterPaneBody(dst, c.CardView())
	}
	doll := d.Doll.Rect.Rectangle()
	if p.DollPane.Body != nil {
		drawTownPane(dst, p.DollPane, doll, paneRect(d.DollSeam))
	} else {
		draw.Draw(dst, doll.Inset(2), &image.Uniform{C: l.Page.Backdrop.RGBA()}, image.Point{}, draw.Src)
	}
	if preview.Doll == nil {
		if p.Font != nil {
			at := d.Preview.MissingAt.Pt()
			p.Font.Draw(dst, d.Preview.Missing, at.X, at.Y, d.Preview.MissingInk.RGBA())
		}
	} else {
		copyNativeOver(dst, preview.Doll, doll.Min.Add(d.Preview.FigureOffset.Pt()), doll)
	}
	if p.Font != nil && d.Message != nil {
		drawChargenMessage(dst, p.Font, d.Message.Rectangle(), c.message)
	}
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

// DetailedAttributeBoxes are attribute row stat's label, value, lower and
// raise rectangles.
func DetailedAttributeBoxes(l *GeneratorDescription, stat int) (plate, value, lower, raise image.Rectangle, ok bool) {
	if l == nil || stat < 0 || stat >= len(l.Detail.Stats.Value) {
		return image.Rectangle{}, image.Rectangle{}, image.Rectangle{}, image.Rectangle{}, false
	}
	s := &l.Detail.Stats
	v := s.Value[stat].Rectangle()
	return image.Rect(s.LabelLeft, v.Min.Y, v.Min.X, v.Max.Y), v, s.Minus[stat].Rectangle(), s.Plus[stat].Rectangle(), true
}

// paintLoops draws each loop layer's current member.
func (c *Chargen) paintLoops(dst *image.RGBA) {
	l, p := c.layout(), c.art()
	for i, loop := range l.PreCreate.Loops {
		if i >= len(p.Loops) || len(p.Loops[i]) == 0 {
			continue
		}
		at := 0
		if i < len(c.loopCount) {
			at = c.loopCount[i]
		}
		frame := p.Loops[i][(at+loop.Phase)%len(p.Loops[i])]
		copyNativeKeyed(dst, frame, loop.At.Pt(), dst.Bounds())
	}
}

// advanceLoops steps each loop layer at a paint more than its period after
// its last step. The first paint only stamps the clock.
func (c *Chargen) advanceLoops(now time.Time, active bool) {
	l := c.layout()
	if l == nil || !active {
		return
	}
	loops := l.PreCreate.Loops
	for len(c.loopCount) < len(loops) {
		c.loopCount, c.loopAt = append(c.loopCount, 0), append(c.loopAt, time.Time{})
	}
	for i, loop := range loops {
		switch {
		case c.loopAt[i].IsZero():
			c.loopAt[i] = now
		case now.Sub(c.loopAt[i]) > ms(loop.PeriodMS):
			c.loopCount[i]++
			c.loopAt[i] = now
		}
	}
}

// noKeys is the key map of a page without a description: Enter activates the
// focused control, Escape leaves, typing reaches a focused name field.
var noKeys = GeneratorKeys{Enter: "focused", Escape: "leave", Typing: "focused", FocusKeys: true}

// keys is the showing page's key map.
func (c *Chargen) keys() *GeneratorKeys {
	l := c.layout()
	switch {
	case l == nil:
		k := noKeys
		return &k
	case c.stage == DetailedStage:
		return &l.Detail.Keys
	}
	return &l.PreCreate.Keys
}

// pageCursor is the showing page's cursor name.
func (c *Chargen) pageCursor() string {
	l := c.layout()
	switch {
	case l == nil:
		return "default"
	case c.stage == DetailedStage:
		return l.Detail.Cursor
	}
	return l.PreCreate.Cursor
}

// The page sounds the description names; an empty name requests nothing.
func (c *Chargen) heroSound() string {
	if l := c.layout(); l != nil {
		return l.PreCreate.HeroSound
	}
	return ""
}

func (c *Chargen) levelSound(i int) string {
	if l := c.layout(); l != nil && i >= 0 && i < len(l.PreCreate.Levels) {
		return l.PreCreate.Levels[i].Sound
	}
	return ""
}

func (c *Chargen) buttonSound(id chargenControl) string {
	l := c.layout()
	switch {
	case l == nil:
		return ""
	case id == chargenBack:
		return l.PreCreate.Back.Sound
	}
	return l.PreCreate.Forward.Sound
}

func (c *Chargen) skillSound(i int) string {
	l := c.layout()
	if l == nil {
		return ""
	}
	skills := l.Detail.Classes[c.columnClass()].Skills
	if i < 0 || i >= len(skills) {
		return ""
	}
	return skills[i].Sound
}

func (c *Chargen) statSound() string {
	if l := c.layout(); l != nil {
		return l.Detail.Stats.Sound
	}
	return ""
}

// reservedNames are the names Accept refuses.
func (c *Chargen) reservedNames() []string {
	if l := c.layout(); l != nil {
		return l.Detail.Refusals.Reserved
	}
	return nil
}
