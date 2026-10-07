package ui

import (
	"image"
	"image/color"
	"image/draw"
	"reflect"
	"testing"

	"againrom/pkg/render/text"
)

// TestTownShellButtonWellsSitInsideTheSharedOrigin proves 1017's fix: both
// rooms' button wells are placed relative to TownUpperRegion.Min, the same
// origin the shipped area picture is drawn at, and neither room claims the
// sixteen-pixel outline overlap (x in [464,480)) as a button hit target. The
// pre-fix geometry claimed that strip for the school — the bug this story
// was opened to fix — putting a button bitmap outside its own well.
func TestTownShellButtonWellsSitInsideTheSharedOrigin(t *testing.T) {
	school := TownSurfaceView{Kind: TownSurfaceSchool, Buttons: []TownSurfaceButton{{Enabled: true}}}
	if c, ok := TownSurfaceControlAt(school, image.Pt(470, 90)); ok {
		t.Fatalf("school claimed the outline overlap x=470 as %#v", c)
	}
	if c, ok := TownSurfaceControlAt(school, image.Pt(554, 94)); !ok || c.Kind != TownSurfaceControlButton || c.Index != 0 {
		t.Fatalf("school well 0 centre = %#v, %v; want button 0", c, ok)
	}
	tavern := TownSurfaceView{Kind: TownSurfaceTavern, Buttons: []TownSurfaceButton{{Enabled: true}}}
	if c, ok := TownSurfaceControlAt(tavern, image.Pt(470, 20)); ok {
		t.Fatalf("tavern claimed x=470 as %#v", c)
	}
	if c, ok := TownSurfaceControlAt(tavern, image.Pt(554, 41)); !ok || c.Kind != TownSurfaceControlButton || c.Index != 0 {
		t.Fatalf("tavern well 0 centre = %#v, %v; want button 0", c, ok)
	}
}

// TestTavernButtonsUseShopFourButtonComposition pins DIV-483 independently:
// the exact shop panel has narrow edge plaques and wide middle plaques. The
// same rectangles own drawing and input, while the exposed ornament pixels
// remain the panel body's pixels.
func TestTavernButtonsUseShopFourButtonComposition(t *testing.T) {
	wells := []image.Rectangle{
		image.Rect(30, 15, 150, 67),
		image.Rect(19, 67, 159, 113),
		image.Rect(19, 114, 159, 160),
		image.Rect(30, 160, 150, 212),
	}
	colors := []color.RGBA{
		{R: 0xd1, G: 0x2a, B: 0x31, A: 0xff},
		{R: 0x31, G: 0xd1, B: 0x2a, A: 0xff},
		{R: 0x2a, G: 0x31, B: 0xd1, A: 0xff},
		{R: 0xd1, G: 0xb2, B: 0x31, A: 0xff},
	}
	panelColor := color.RGBA{R: 0x16, G: 0x28, B: 0x39, A: 0xff}
	art := &TownTavernArt{CommandUpper: uniform(176, 238, panelColor)}
	v := TownSurfaceView{Kind: TownSurfaceTavern, TavernArt: art}
	for i, well := range wells {
		pic := uniform(well.Dx(), well.Dy(), colors[i])
		if i == 0 || i == 3 {
			pic.SetRGBA(0, 0, color.RGBA{}) // a rounded keyed edge must reveal CommandUpper
		}
		art.CommandButtons[i] = pic
		v.Buttons = append(v.Buttons, TownSurfaceButton{Enabled: true})
		if got := TownSurfaceButtonWell(TownSurfaceTavern, i); got != well {
			t.Fatalf("button %d local well = %v, want shop composition %v", i, got, well)
		}
		absolute := well.Add(TownWideUpperRegion.Min)
		if got := TownSurfaceButtonRect(TownSurfaceTavern, i); got != absolute {
			t.Fatalf("button %d screen rect = %v, want %v", i, got, absolute)
		}
	}

	// At rest the panel body is the only picture in every well. Pressing
	// one button draws exactly its own bitmap at its own rect (TOWN-260).
	for press := -1; press < len(wells); press++ {
		pv := v
		if press >= 0 {
			pv.Press = TownSurfaceControl{Kind: TownSurfaceControlButton, Index: press}
		}
		pressed := ComposeTownSurface(pv)
		for i, well := range wells {
			absolute := well.Add(TownWideUpperRegion.Min)
			for y := absolute.Min.Y; y < absolute.Max.Y; y++ {
				for x := absolute.Min.X; x < absolute.Max.X; x++ {
					want := panelColor
					if i == press && !((i == 0 || i == 3) && x == absolute.Min.X && y == absolute.Min.Y) {
						want = colors[i]
					}
					if got := pressed.RGBAAt(x, y); got != want {
						t.Fatalf("press %d button %d pixel (%d,%d) = %#v, want %#v", press, i, x, y, got, want)
					}
				}
			}
		}
	}
	frame := ComposeTownSurface(v)
	for i, well := range wells {
		absolute := well.Add(TownWideUpperRegion.Min)
		centre := image.Pt((absolute.Min.X+absolute.Max.X)/2, (absolute.Min.Y+absolute.Max.Y)/2)
		if c, ok := TownSurfaceControlAt(v, centre); !ok || c.Kind != TownSurfaceControlButton || c.Index != i {
			t.Errorf("button %d centre %v = %#v, %v", i, centre, c, ok)
		}
	}
	for _, p := range []image.Point{{X: 484, Y: 40}, {X: 624, Y: 180}, {X: 554, Y: 113}} {
		if got := frame.RGBAAt(p.X, p.Y); got != panelColor {
			t.Errorf("exposed panel ornament pixel %v = %#v, want panel %#v", p, got, panelColor)
		}
		if c, ok := TownSurfaceControlAt(v, p); ok {
			t.Errorf("exposed panel point %v claimed %#v", p, c)
		}
	}
}

func TestTownSchoolUsesClassSpecificRasterMask(t *testing.T) {
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{uint8(i), uint8(i), uint8(i), 0xff}
	}
	art := &TownSchoolArt{}
	art.Masks[0] = image.NewPaletted(image.Rect(0, 0, 92, 120), palette)
	art.Masks[0].SetColorIndex(3, 4, 0xff) // fighter slot 0
	art.Masks[1] = image.NewPaletted(image.Rect(0, 0, 100, 120), palette)
	art.Masks[1].SetColorIndex(5, 6, 0x87) // mage slot 0
	cells := make([]TownSurfaceCell, 10)
	cells[0].Enabled, cells[5].Enabled = true, true

	fighter := TownSurfaceView{Kind: TownSurfaceSchool, SchoolArt: art, SchoolClass: 0, Cells: cells}
	if c, ok := TownSurfaceControlAt(fighter, image.Pt(195, 196)); !ok || c.Kind != TownSurfaceControlCell || c.Index != 0 {
		t.Fatalf("fighter mask = %#v, %v", c, ok)
	}
	mage := TownSurfaceView{Kind: TownSurfaceSchool, SchoolArt: art, SchoolClass: 1, Cells: cells}
	if c, ok := TownSurfaceControlAt(mage, image.Pt(193, 194)); !ok || c.Kind != TownSurfaceControlCell || c.Index != 5 {
		t.Fatalf("mage mask = %#v, %v", c, ok)
	}
	// An unpainted mask byte and a disabled slot are not rectangular fallbacks.
	if c, ok := TownSurfaceControlAt(fighter, image.Pt(196, 196)); ok {
		t.Fatalf("unpainted mask byte claimed %#v", c)
	}
	cells[0].Enabled = false
	fighter.Cells = cells
	if c, ok := TownSurfaceControlAt(fighter, image.Pt(195, 196)); ok {
		t.Fatalf("disabled mask slot claimed %#v", c)
	}
}

func TestTownShellCharacterControlsShareOneGeometry(t *testing.T) {
	for _, kind := range []TownSurfaceKind{TownSurfaceTavern, TownSurfaceSchool} {
		v := TownSurfaceView{Kind: kind, Hero: TownCharacterView{HasSubject: true, MemberCount: 2}}
		for p, want := range map[image.Point]TownSurfaceControlKind{
			image.Pt(490, 450): TownSurfaceControlPrevious,
			image.Pt(620, 250): TownSurfaceControlMode,
			image.Pt(610, 450): TownSurfaceControlNext,
		} {
			c, ok := TownSurfaceControlAt(v, p)
			if !ok || c.Kind != want {
				t.Errorf("kind %d point %v = %#v, %v; want %d", kind, p, c, ok, want)
			}
		}
		if got := ComposeTownSurface(v); got == nil || got.Bounds() != image.Rect(0, 0, 640, 480) {
			t.Fatalf("kind %d composed %v", kind, got)
		}
	}
}

// TestTownStatisticsKeepsLeftContentAndPlacesTheCardInTheCharacterPane
// proves 1022 spec B6: the DOLL/STATS toggle switches only the character
// pane (TownCharacterRegion), so the room's own left content and roster
// cells stay live in both modes, and the native character card composes
// over the pane's own rectangle, replacing the doll figure, rather than
// over any left-side rectangle.
func TestTownStatisticsKeepsLeftContentAndPlacesTheCardInTheCharacterPane(t *testing.T) {
	figure := image.NewRGBA(image.Rect(0, 0, 160, 200))
	figureColor := color.RGBA{R: 0x31, G: 0xa7, B: 0x5c, A: 0xff}
	draw.Draw(figure, figure.Bounds(), &image.Uniform{C: figureColor}, image.Point{}, draw.Src)
	v := TownSurfaceView{
		Kind:  TownSurfaceTavern,
		Cells: []TownSurfaceCell{{Enabled: true}, {Enabled: true}, {Enabled: true}, {Enabled: true}},
		Hero: TownCharacterView{HasSubject: true, Statistics: true, MemberCount: 2,
			Subject: panelWideSubjectFixture(), Figure: figure, Font: panelWideFont()},
		Font: panelWideFont(),
	}
	v.Buttons = []TownSurfaceButton{{Enabled: true}}

	// Tavern cell 3 is an ordinary cell hit in both modes now: nothing on
	// the left is replaced any more.
	cell := townSurfaceCellRect(TownSurfaceTavern, 3, len(v.Cells))
	p := image.Pt((cell.Min.X+cell.Max.X)/2, (cell.Min.Y+cell.Max.Y)/2)
	plain := v
	plain.Hero.Statistics = false
	if c, ok := TownSurfaceControlAt(plain, p); !ok || c.Kind != TownSurfaceControlCell || c.Index != 3 {
		t.Fatalf("plain left cell = %+v,%v, want cell 3", c, ok)
	}
	if c, ok := TownSurfaceControlAt(v, p); !ok || c.Kind != TownSurfaceControlCell || c.Index != 3 {
		t.Fatalf("statistics left cell = %+v,%v, want cell 3 (B6: content stays live)", c, ok)
	}

	for at, want := range map[image.Point]TownSurfaceControlKind{
		image.Pt(554, 41):  TownSurfaceControlButton, // Sleep well, DIV-483
		image.Pt(490, 450): TownSurfaceControlPrevious,
		image.Pt(620, 250): TownSurfaceControlMode, //
		image.Pt(610, 450): TownSurfaceControlNext,
	} {
		if c, ok := TownSurfaceControlAt(v, at); !ok || c.Kind != want {
			t.Errorf("persistent control at %v = %+v,%v, want kind %d", at, c, ok, want)
		}
	}

	frame := ComposeTownSurface(v)
	refLayout := CompactPanelLayout(nil)
	card := RenderCharacterPanel(refLayout, v.Hero.Font, v.Hero.Subject)
	if card.Bounds().Dx() > TownCharacterRegion.Dx() || card.Bounds().Dy() > TownCharacterRegion.Dy() {
		t.Fatalf("card size = %v exceeds character pane %v", card.Bounds().Size(), TownCharacterRegion.Size())
	}
	// Excluded: TownCharacterPersistentControls() (the prev/next chevrons
	// and the mode box, which keep an opaque backing plate under Statistics
	// mode). Every other rectangle is native card content and must match.
	controls := TownCharacterPersistentControls()
	for y := 0; y < card.Bounds().Dy(); y++ {
		for x := 0; x < card.Bounds().Dx(); x++ {
			p := TownCharacterRegion.Min.Add(image.Pt(x, y))
			covered := false
			for _, r := range controls {
				if p.In(r) {
					covered = true
					break
				}
			}
			if covered {
				continue
			}
			if got, want := frame.RGBAAt(p.X, p.Y), card.RGBAAt(x, y); got != want {
				t.Fatalf("character pane statistics pixel %v = %#v, want native card %#v", p, got, want)
			}
		}
	}
}

// TestSchoolDrawsTheShownClassOwnColumnFace covers the room's first defect of
// 1015: the background bakes one class's column face, and before this the other
// class's five patches were drawn over it. The face is now the shown member's
// own, drawn at SchoolFaceOrigin over whatever the background carries there.
func TestSchoolDrawsTheShownClassOwnColumnFace(t *testing.T) {
	art := &TownSchoolArt{Background: uniform(480, 480, color.RGBA{R: 0x11, A: 0xff})}
	face := [2]color.RGBA{{R: 0xa0, A: 0xff}, {B: 0xa0, A: 0xff}}
	for class := range art.Faces {
		art.Faces[class] = uniform(148, 208, face[class])
	}
	for class := 0; class < 2; class++ {
		v := TownSurfaceView{Kind: TownSurfaceSchool, SchoolArt: art, SchoolClass: class,
			Cells: make([]TownSurfaceCell, 10), HoverCell: -1}
		got := ComposeTownSurface(v)
		for _, p := range []image.Point{SchoolFaceOrigin, SchoolFaceOrigin.Add(image.Pt(147, 207))} {
			if c := got.RGBAAt(p.X, p.Y); c != face[class] {
				t.Fatalf("class %d face pixel at %v = %#v, want %#v", class, p, c, face[class])
			}
		}
		// One pixel outside the frame stays the room background.
		if c := got.RGBAAt(SchoolFaceOrigin.X-1, SchoolFaceOrigin.Y); c != (color.RGBA{R: 0x11, A: 0xff}) {
			t.Fatalf("class %d overpainted left of the frame: %#v", class, c)
		}
	}
}

// TestSchoolSkillRectsAreOwnedPerClass covers the second defect: one shared
// five-rectangle array put every mage icon on the fighter's vertical stack. The
// two arrays now share no rectangle, each class's five lie inside that class's
// own panel, and the mage's five are pairwise disjoint, which the fighter's
// overlapping bands are not — that is why the room needs a raster mask at all.
func TestSchoolSkillRectsAreOwnedPerClass(t *testing.T) {
	for class := 0; class < 2; class++ {
		panel := SchoolPanelRect(class)
		for slot := 0; slot < 5; slot++ {
			r := SchoolSkillRect(class, slot)
			if r.Empty() || !r.In(panel) {
				t.Fatalf("class %d slot %d rect %v is not inside panel %v", class, slot, r, panel)
			}
		}
	}
	for a := 0; a < 5; a++ {
		for b := 0; b < 5; b++ {
			if SchoolSkillRect(0, a) == SchoolSkillRect(1, b) {
				t.Fatalf("fighter slot %d and mage slot %d share rectangle %v", a, b, SchoolSkillRect(0, a))
			}
		}
		for b := a + 1; b < 5; b++ {
			if !SchoolSkillRect(1, a).Intersect(SchoolSkillRect(1, b)).Empty() {
				t.Fatalf("mage slots %d and %d overlap: %v and %v", a, b,
					SchoolSkillRect(1, a), SchoolSkillRect(1, b))
			}
		}
	}
	if got := SchoolSkillRect(2, 0); !got.Empty() {
		t.Fatalf("out-of-range class answered %v", got)
	}
	if got := SchoolPanelRect(-1); !got.Empty() {
		t.Fatalf("out-of-range panel answered %v", got)
	}
}

// TestTownButtonTextRectsSplitTheWellIntoLabelThenValue pins
// townButtonLabelRect and townButtonValueRect against hand-transcribed
// literals, for the well ComposeTownSurface's own button loop draws through
// (townshell.go).
func TestTownButtonTextRectsSplitTheWellIntoLabelThenValue(t *testing.T) {
	well := image.Rect(536, 133, 640, 175) // an arbitrary well, not read from production
	if got, want := townButtonLabelRect(well), image.Rect(536, 135, 640, 157); got != want {
		t.Errorf("townButtonLabelRect(%v) = %v, want %v", well, got, want)
	}
	if got, want := townButtonValueRect(well), image.Rect(536, 156, 640, 173); got != want {
		t.Errorf("townButtonValueRect(%v) = %v, want %v", well, got, want)
	}
}

// TestSchoolSkillRectsMatchTheMeasuredLiterals pins every one of the ten
// rectangles against a hand-transcribed copy of schoolSkillRects' own
// literals (townshell.go), the values cmd/schoolcheck measured (this file's
// own header comment).
func TestSchoolSkillRectsMatchTheMeasuredLiterals(t *testing.T) {
	want := [2][5]image.Rectangle{
		{
			image.Rect(200, 196, 280, 228),
			image.Rect(200, 216, 280, 252),
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
	for class := 0; class < 2; class++ {
		for slot := 0; slot < 5; slot++ {
			if got := SchoolSkillRect(class, slot); got != want[class][slot] {
				t.Errorf("SchoolSkillRect(%d,%d) = %v, want %v", class, slot, got, want[class][slot])
			}
		}
	}
}

// TestTownSurfaceMessageIsDrawnAtItsOwnRect pins townSurfaceMessageRect
// against a hand-transcribed literal, the same shape
// TestSchoolSkillRectsMatchTheMeasuredLiterals and
// TestTownButtonTextRectsSplitTheWellIntoLabelThenValue already carry: the
// expected rectangle is typed here independently of the function under
// test, so a shift of the production literal fails this test directly
// (round 2 adversarial review, finding 1: townshell.go:887's inline literal
// had no test that could fail when it moved — this is that test). It also
// confirms ComposeTownSurface's own real render places ink somewhere inside
// that exact rect for a non-empty Message, so the extraction did not
// disconnect the producer from the composer that calls it.
func TestTownSurfaceMessageIsDrawnAtItsOwnRect(t *testing.T) {
	want := image.Rect(12, 448, 468, 478)
	if got := townSurfaceMessageRect(); got != want {
		t.Fatalf("townSurfaceMessageRect() = %v, want %v", got, want)
	}

	v := TownSurfaceView{
		Kind:    TownSurfaceTavern,
		Font:    panelFont(),
		Message: "M",
	}
	frame := ComposeTownSurface(v)

	found := false
	for y := want.Min.Y; y < want.Max.Y && !found; y++ {
		for x := want.Min.X; x < want.Max.X; x++ {
			if frame.RGBAAt(x, y) == townShellText {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("ComposeTownSurface drew no townShellText-coloured pixel inside townSurfaceMessageRect() = %v for a non-empty Message", want)
	}
}

// TestSchoolPaintsEachClassIconAtItsOwnRectangle draws one selected icon per
// class and reads the frame back. The picture lands on that class's rectangle
// and nowhere on the other class's, and a transparent pixel of the patch leaves
// the column face showing through rather than painting black over it.
func TestSchoolPaintsEachClassIconAtItsOwnRectangle(t *testing.T) {
	faceColor := color.RGBA{G: 0x70, A: 0xff}
	ink := color.RGBA{R: 0xd4, G: 0x22, B: 0x88, A: 0xff}
	art := &TownSchoolArt{Background: uniform(480, 480, color.RGBA{R: 0x11, A: 0xff})}
	for class := range art.Faces {
		art.Faces[class] = uniform(148, 208, faceColor)
	}
	for class := 0; class < 2; class++ {
		for slot := 0; slot < 5; slot++ {
			r := SchoolSkillRect(class, slot)
			pic := uniform(r.Dx(), r.Dy(), ink)
			// The patch's own top-left pixel is transparent, standing for the
			// pure black keyBlack clears on the shipped art.
			pic.SetRGBA(0, 0, color.RGBA{})
			for state := 0; state < 3; state++ {
				art.Skills[class][slot][state] = pic
			}
		}
	}
	for class := 0; class < 2; class++ {
		cells := make([]TownSurfaceCell, 10)
		for i := range cells {
			cells[i].Enabled = true
		}
		cells[class*5+2].Selected = true
		v := TownSurfaceView{Kind: TownSurfaceSchool, SchoolArt: art, SchoolClass: class,
			Cells: cells, HoverCell: -1}
		got := ComposeTownSurface(v)
		mine := SchoolSkillRect(class, 2)
		if c := got.RGBAAt(mine.Min.X+1, mine.Min.Y+1); c != ink {
			t.Fatalf("class %d icon at %v = %#v, want %#v", class, mine.Min, c, ink)
		}
		if c := got.RGBAAt(mine.Min.X, mine.Min.Y); c != faceColor {
			t.Fatalf("class %d transparent patch pixel = %#v, want the face %#v", class, c, faceColor)
		}
		// The other class's third rectangle carries no icon of this class.
		other := SchoolSkillRect(1-class, 2)
		if !other.Intersect(mine).Empty() {
			t.Fatalf("the two class rectangles overlap: %v and %v", mine, other)
		}
		if c := got.RGBAAt(other.Max.X-1, other.Max.Y-1); c == ink {
			t.Fatalf("class %d painted into the other class's rectangle %v", class, other)
		}
	}
}

// TestSchoolMaskAnswersAreOneVisualCellPerCode covers the fourth defect's
// shape. Each class's five mask codes answer five DISTINCT slots in 0..4, and
// the answer is a visual cell index: it indexes Cells, so a disabled cell at
// that index suppresses the hit. Which code means which skill is a property of
// the shipped mask and is measured by cmd/schoolcheck, which a test may not
// read (golden rule 2).
func TestSchoolMaskAnswersAreOneVisualCellPerCode(t *testing.T) {
	codes := []uint8{0x37, 0x87, 0x9e, 0xd2, 0xff}
	for class := 0; class < 2; class++ {
		seen := map[int]uint8{}
		for _, code := range codes {
			art := &TownSchoolArt{}
			panel := SchoolPanelRect(class)
			art.Masks[class] = solidMask(panel.Dx(), panel.Dy(), code)
			cells := make([]TownSurfaceCell, 10)
			for i := range cells {
				cells[i].Enabled = true
			}
			v := TownSurfaceView{Kind: TownSurfaceSchool, SchoolArt: art, SchoolClass: class, Cells: cells}
			p := image.Pt(panel.Min.X+3, panel.Min.Y+3)
			c, ok := TownSurfaceControlAt(v, p)
			if !ok || c.Kind != TownSurfaceControlCell {
				t.Fatalf("class %d code 0x%02x = %#v, %v", class, code, c, ok)
			}
			slot := c.Index - class*5
			if slot < 0 || slot > 4 {
				t.Fatalf("class %d code 0x%02x answered cell %d, outside its own row", class, code, c.Index)
			}
			if prev, dup := seen[slot]; dup {
				t.Fatalf("class %d codes 0x%02x and 0x%02x both answer slot %d", class, prev, code, slot)
			}
			seen[slot] = code

			// The same answer indexes Cells: disabling that cell suppresses it.
			cells[c.Index].Enabled = false
			if c, ok := TownSurfaceControlAt(v, p); ok {
				t.Fatalf("class %d code 0x%02x survived disabling cell %d: %#v", class, code, slot, c)
			}
		}
		if len(seen) != 5 {
			t.Fatalf("class %d mask answers %d distinct slots, want 5", class, len(seen))
		}
	}
}

func uniform(w, h int, c color.RGBA) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(pic, pic.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	return pic
}

func solidMask(w, h int, code uint8) *image.Paletted {
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{uint8(i), uint8(i), uint8(i), 0xff}
	}
	m := image.NewPaletted(image.Rect(0, 0, w, h), palette)
	for i := range m.Pix {
		m.Pix[i] = code
	}
	return m
}

// TestSchoolColumnFaceStaysVisibleDuringStatistics witnesses 1022 spec B6's
// removal of the school face's own former !statistics gate. Before this
// story the card composed over the left content (townStatisticsCardRegion,
// this package's own former alias for chargenCardBox) and the school face's
// own 148x208 column at SchoolFaceOrigin overlapped it at
// (168,176)-(316,384), so the gate existed to keep the card's own pixels
// from being overpainted. The card now composes in the character pane
// (TownCharacterRegion, x:480-640) instead, which SchoolFaceOrigin (well
// inside TownContentRegion, x:0-480) never reaches, so the two surfaces no
// longer compete for the same pixels and the face draws in both modes.
func TestSchoolColumnFaceStaysVisibleDuringStatistics(t *testing.T) {
	faceColor := color.RGBA{R: 0xff, G: 0x00, B: 0xd7, A: 0xff}
	face := image.NewRGBA(image.Rect(0, 0, 148, 208))
	draw.Draw(face, face.Bounds(), &image.Uniform{C: faceColor}, image.Point{}, draw.Src)
	art := &TownSchoolArt{}
	art.Faces[0], art.Faces[1] = face, face

	figure := image.NewRGBA(image.Rect(0, 0, 160, 200))
	draw.Draw(figure, figure.Bounds(), &image.Uniform{C: color.RGBA{R: 0x31, G: 0xa7, B: 0x5c, A: 0xff}}, image.Point{}, draw.Src)

	if !SchoolFaceOrigin.Add(face.Bounds().Size()).In(image.Rect(0, 0, TownCharacterRegion.Min.X+1, 480)) {
		t.Fatalf("the face rect at %v now reaches the character pane at x>=%d; this test's premise no longer holds",
			SchoolFaceOrigin, TownCharacterRegion.Min.X)
	}

	for class := 0; class < 2; class++ {
		v := TownSurfaceView{
			Kind: TownSurfaceSchool, SchoolArt: art, SchoolClass: class,
			Cells: make([]TownSurfaceCell, 10),
			Hero: TownCharacterView{HasSubject: true, Statistics: true, MemberCount: 2,
				Subject: panelSubjectFixture(), Figure: figure, Font: panelFont()},
			Font: panelFont(),
		}
		frame := ComposeTownSurface(v)
		for _, p := range []image.Point{SchoolFaceOrigin, SchoolFaceOrigin.Add(image.Pt(147, 207))} {
			if got := frame.RGBAAt(p.X, p.Y); got != faceColor {
				t.Fatalf("class %d: face pixel at %v = %#v, want %#v while statistics is visible", class, p, got, faceColor)
			}
		}
	}
}

// townShellRosterTestFont is a synthetic font with a UNIFORM 9px advance on
// every glyph and no spacing. This is not font1 itself — a synthetic test
// may not read an install (golden rule 2) — it is a deliberately
// worse-than-observed stand-in, so a budget this font accepts is a budget
// the shipped font accepts with margin.
func townShellRosterTestFont() *text.Font {
	glyphs := make([]text.Glyph, 224)
	for i := range glyphs {
		pixels := make([]text.Pixel, 9*10)
		for n := range pixels {
			pixels[n] = text.Pixel{Level: text.MaxLevel, Painted: true}
		}
		glyphs[i] = text.Glyph{Width: 9, Height: 10, Advance: 9, Pixels: pixels}
	}
	return &text.Font{Glyphs: glyphs}
}

func townShellCardTestFont() *text.Font {
	glyphs := make([]text.Glyph, 224)
	for i := range glyphs {
		pixels := make([]text.Pixel, 6*10)
		for n := range pixels {
			pixels[n] = text.Pixel{Level: text.MaxLevel, Painted: true}
		}
		glyphs[i] = text.Glyph{Width: 6, Height: 10, Advance: 6, Pixels: pixels}
	}
	return &text.Font{Glyphs: glyphs}
}

// TestCompactMercenaryCardsFillFromTheBottomAndKeepOnlyCountAndPrice pins the
// owner's screenshot and the shipped 48x64 manback bitmaps: six contiguous
// cards per row, bottom-left first, then upward. The visible copy is only
// current/capacity and price; Semantic remains a headless target and must never
// become painted title ink.
func TestCompactMercenaryCardsFillFromTheBottomAndKeepOnlyCountAndPrice(t *testing.T) {
	font := townShellCardTestFont()
	for i := 0; i < 6; i++ {
		r := tavernPortraitCellRect(i)
		want := image.Rect(176+i*48, 416, 224+i*48, 480)
		if r != want {
			t.Fatalf("bottom portrait %d = %v, want measured %v", i, r, want)
		}
	}
	if r, want := tavernPortraitCellRect(6), image.Rect(176, 352, 224, 416); r != want {
		t.Fatalf("seventh portrait = %v, want first column of the row above %v", r, want)
	}
	cellWidth := tavernPortraitCellRect(0).Dx()
	cases := []string{"3/3", "10/10", "99999", "1000000"}
	for _, s := range cases {
		if w, _ := font.Measure(s); w > cellWidth {
			t.Errorf("%q measures %dpx, wider than the measured %dpx card", s, w, cellWidth)
		}
	}
	v := TownSurfaceView{Kind: TownSurfaceTavern, Font: panelFont(), CardFont: font, Cells: []TownSurfaceCell{{
		Semantic: "Squad 14", Detail: "3/3", Price: "9120", Portrait: true, Enabled: true,
	}}}
	a := ComposeTownSurface(v)
	v.Cells[0].Semantic = "this must never be visible"
	b := ComposeTownSurface(v)
	if !imagesEqual(a, b) {
		t.Fatal("changing the headless squad name changed visible portrait pixels")
	}
	v.Cells[0].Price = "91200"
	if imagesEqual(a, ComposeTownSurface(v)) {
		t.Fatal("changing the visible price changed no portrait pixels")
	}
}

// TestTavernMiniCardDrawsTheProductionSpriteAtTheMeasuredOrigin is the
// production-use geometry witness. Its two source pixels sit at opposite
// corners of the 48x64 frame, so shifting ComposeTownSurface's sprite blit by
// one pixel loses one corner and moves the other.
func TestTavernMiniCardDrawsTheProductionSpriteAtTheMeasuredOrigin(t *testing.T) {
	ground := color.RGBA{R: 0x12, G: 0x24, B: 0x36, A: 0xff}
	mark := color.RGBA{R: 0xe1, G: 0x22, B: 0x33, A: 0xff}
	sprite := image.NewRGBA(image.Rect(0, 0, 48, 64))
	sprite.SetRGBA(0, 0, mark)
	sprite.SetRGBA(47, 63, mark)
	v := TownSurfaceView{Kind: TownSurfaceTavern, TavernArt: &TownTavernArt{
		ManBack: uniform(48, 64, ground),
	}, Cells: []TownSurfaceCell{{Portrait: true, Picture: sprite}}}
	frame := ComposeTownSurface(v)
	r := image.Rect(176, 416, 224, 480)
	for _, p := range []image.Point{r.Min, r.Max.Sub(image.Pt(1, 1))} {
		if got := frame.RGBAAt(p.X, p.Y); got != mark {
			t.Fatalf("sprite corner %v = %#v, want %#v", p, got, mark)
		}
	}
	if got := frame.RGBAAt(r.Min.X+1, r.Min.Y); got != ground {
		t.Fatalf("transparent sprite pixel did not preserve shipped ground: got %#v, want %#v", got, ground)
	}
}

func TestOnlyTheSelectedTavernMiniatureConsumesTheAnimationFrame(t *testing.T) {
	first := uniform(48, 64, color.RGBA{R: 0xa1, A: 0xff})
	second := uniform(48, 64, color.RGBA{G: 0xb2, A: 0xff})
	frames := []image.Image{first, second}
	v := TownSurfaceView{Kind: TownSurfaceTavern, AnimationFrame: 1,
		TavernArt: &TownTavernArt{ManBack: uniform(48, 64, color.RGBA{B: 0x33, A: 0xff})},
		Cells: []TownSurfaceCell{
			{Portrait: true, Selected: true, Frames: frames},
			{Portrait: true, Frames: frames},
		}}
	frame := ComposeTownSurface(v)
	if got := frame.RGBAAt(176, 416); got.G != 0xb2 {
		t.Fatalf("selected card phase 1 pixel = %#v, want second frame", got)
	}
	if got := frame.RGBAAt(224, 416); got.R != 0xa1 {
		t.Fatalf("unselected card phase 1 pixel = %#v, want frozen first frame", got)
	}
}

func TestTavernMiniCardBorderIsOnlyTheShippedBackground(t *testing.T) {
	ground := uniform(48, 64, color.RGBA{R: 0x19, G: 0x2a, B: 0x3b, A: 0xff})
	v := TownSurfaceView{Kind: TownSurfaceTavern, TavernArt: &TownTavernArt{ManBack: ground},
		Cells: []TownSurfaceCell{{Portrait: true, Selected: true, Hired: true}}}
	withState := ComposeTownSurface(v)
	v.Cells[0].Selected, v.Cells[0].Hired = false, false
	withoutState := ComposeTownSurface(v)
	if !imagesEqual(withState, withoutState) {
		t.Fatal("selected/hired state invented a card border outside the shipped background")
	}
}

func TestTalkOnlyCardUsesItsShippedBackgroundAndArtTavernAddsNoTitle(t *testing.T) {
	mercBack := uniform(48, 64, color.RGBA{R: 0x21, A: 0xff})
	talkBack := uniform(48, 64, color.RGBA{G: 0x43, A: 0xff})
	v := TownSurfaceView{Kind: TownSurfaceTavern, Title: "TAVERN", Font: panelFont(),
		TavernArt: &TownTavernArt{
			Center:      uniform(320, 480, color.RGBA{B: 0x17, A: 0xff}),
			ManBack:     mercBack,
			ManBackTalk: talkBack,
		}, Cells: []TownSurfaceCell{{Portrait: true, TalkOnly: true}, {Portrait: true}}}
	withTitle := ComposeTownSurface(v)
	v.Title = ""
	withoutTitle := ComposeTownSurface(v)
	if !imagesEqual(withTitle, withoutTitle) {
		t.Fatal("art-backed tavern painted an authored title absent from the owner frame")
	}
	if got := withTitle.RGBAAt(176, 416); got != talkBack.RGBAAt(0, 0) {
		t.Fatalf("talk-only card background = %#v, want shipped talk background", got)
	}
	if got := withTitle.RGBAAt(224, 416); got != mercBack.RGBAAt(0, 0) {
		t.Fatalf("mercenary card background = %#v, want shipped mercenary background", got)
	}
}

func TestTavernMiniCardTextUsesTheMeasuredOppositeCorners(t *testing.T) {
	ground := color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}
	v := TownSurfaceView{Kind: TownSurfaceTavern, Font: townShellRosterTestFont(),
		TavernArt: &TownTavernArt{ManBack: uniform(48, 64, ground)},
		Cells:     []TownSurfaceCell{{Portrait: true, Price: "1111", Detail: "2/2"}}}
	frame := ComposeTownSurface(v)
	r := image.Rect(176, 416, 224, 480)
	// Four 9px price glyphs end at the card's right edge. Three count glyphs
	// begin at its left edge and end at its bottom edge.
	for _, p := range []image.Point{
		image.Pt(r.Max.X-36, r.Min.Y), image.Pt(r.Max.X-1, r.Min.Y+9),
		image.Pt(r.Min.X, r.Max.Y-10), image.Pt(r.Min.X+26, r.Max.Y-1),
	} {
		if got := frame.RGBAAt(p.X, p.Y); got != tavernCardText {
			t.Fatalf("card text corner %v = %#v, want %#v", p, got, tavernCardText)
		}
	}
	for _, p := range []image.Point{image.Pt(r.Max.X-37, r.Min.Y), image.Pt(r.Min.X+27, r.Max.Y-1)} {
		if got := frame.RGBAAt(p.X, p.Y); got != ground {
			t.Fatalf("outside measured text at %v = %#v, want ground %#v", p, got, ground)
		}
	}
}

func TestTavernCandidateDollRoutesOnlyMarkedPixelsToFullItemLines(t *testing.T) {
	mask := &SlotMask{W: 4, H: 2, Slot: make([]uint8, 8)}
	mask.Slot[1] = 3
	v := TownSurfaceView{Kind: TownSurfaceTavern,
		Candidate:         TownCharacterView{HasSubject: true},
		CandidateSlotMask: mask,
		CandidateSlotInfo: [12][]string{2: {"Steel Helm", "#Armour 4", "#Value 180"}},
	}
	at := image.Pt(86, 359) // centred origin plus the corrected +8,+1 offset
	if got, ok := TownCandidateHoverLines(v, at.Add(image.Pt(1, 0))); !ok || !reflect.DeepEqual(got, v.CandidateSlotInfo[2]) {
		t.Fatalf("occupied candidate pixel = %q, %v; want full slot lines", got, ok)
	}
	for _, p := range []image.Point{at, image.Pt(0, 237), image.Pt(160, 358), image.Pt(79, 480)} {
		if got, ok := TownCandidateHoverLines(v, p); ok || got != nil {
			t.Errorf("blank/outside candidate pixel %v = %q, %v", p, got, ok)
		}
	}
}

func TestTavernCandidateInspectionDrawsStatsDollAndNormalHover(t *testing.T) {
	font := townShellRosterTestFont()
	figure := image.NewRGBA(image.Rect(0, 0, 4, 2))
	mark := color.RGBA{R: 0x55, G: 0xcc, B: 0x77, A: 0xff}
	draw.Draw(figure, figure.Bounds(), &image.Uniform{C: mark}, image.Point{}, draw.Src)
	mask := &SlotMask{W: 4, H: 2, Slot: []uint8{1, 1, 1, 1, 1, 1, 1, 1}}
	art := &TownTavernArt{
		Center:      uniform(320, 480, color.RGBA{R: 4, G: 5, B: 6, A: 0xff}),
		LeftStats:   uniform(160, 238, color.RGBA{R: 7, G: 8, B: 9, A: 0xff}),
		LeftPicture: uniform(160, 242, color.RGBA{R: 10, G: 11, B: 12, A: 0xff}),
	}
	v := TownSurfaceView{Kind: TownSurfaceTavern, Font: font, TavernArt: art,
		Candidate: TownCharacterView{HasSubject: true, Figure: figure, Font: font, CardFont: font,
			Subject: PanelSubject{Name: "Candidate", HP: 4, MaxHP: 5}},
		CandidateSlotMask: mask,
		CandidateSlotInfo: [12][]string{0: {"Sword", "#Damage 3", "#Value 25"}},
	}
	bare := ComposeTownSurface(v)
	if got := bare.RGBAAt(86, 359); got != mark {
		t.Fatalf("centred composed doll pixel = %#v, want %#v", got, mark)
	}
	v.HasHover, v.Hover = true, image.Pt(86, 359)
	hover := ComposeTownSurface(v)
	if imagesEqual(bare, hover) {
		t.Fatal("occupied candidate hover drew no normal item popup")
	}
}

func TestTavernCandidateInspectionIsNotAnInputControl(t *testing.T) {
	v := TownSurfaceView{Kind: TownSurfaceTavern, Candidate: TownCharacterView{HasSubject: true},
		CandidateSlotMask: &SlotMask{W: 1, H: 1, Slot: []uint8{1}},
		CandidateSlotInfo: [12][]string{0: {"item"}},
	}
	for y := 0; y < 480; y++ {
		for x := 0; x < 160; x++ {
			if got, ok := TownSurfaceControlAt(v, image.Pt(x, y)); ok {
				t.Fatalf("read-only candidate pane point (%d,%d) exposed control %+v", x, y, got)
			}
		}
	}
}
