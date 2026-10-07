package ui

import "image"

// The item-cell star trail is a presentation layer. ITEM-STARPIX-098 gives
// the CRT recurrence and the complete point kernel, but leaves the process
// seed and the draws made before each grid constructor Unknown. This build
// therefore normalises only that unknown starting state to zero (DIV-497).
// The recurrence, coordinate range, phase lookup and colour remain decoded.
const (
	itemStarPointCount = 1024
	itemStarPointMask  = itemStarPointCount - 1
)

type itemStarField struct {
	points [itemStarPointCount]image.Point
}

// Four grids are enough for the complete measured production population:
// mission pack, shop shelf, shop table and shop shown-member pack. Keeping
// one table per grid preserves ITEM-STARPIX-098's constructor ownership.
var itemStarFields = func() [4]itemStarField {
	var fields [4]itemStarField
	var state uint32 // normalised Unknown CRT state; see DIV-497 above.
	for i := range fields {
		fields[i] = makeItemStarField(&state)
	}
	return fields
}()

func makeItemStarField(state *uint32) itemStarField {
	var field itemStarField
	for i := range field.points {
		x := int(itemStarRand(state)/0x1ff) + 8
		y := int(itemStarRand(state)/0x1ff) + 8
		field.points[i] = image.Pt(x, y)
	}
	return field
}

func itemStarRand(state *uint32) uint32 {
	*state = *state*0x343fd + 0x269ec3
	return (*state >> 16) & 0x7fff
}

var itemStarAlpha = [...]uint8{63, 127, 191, 255, 191, 127, 63}

// drawItemStarTrail paints the seven-age RGB (255,0,255) trail at an item
// picture origin. phase introduces one centre every two increments. During
// startup only centres already introduced are present; afterwards seven age
// buckets remain. The only clip is dst.Bounds: the shop passes its full frame,
// matching ITEM-STARCOMP-100's renderer-global half-open clip, and no caller
// adds a per-cell clip. The authored mission bar's narrower intermediate target
// is the disclosed DIV-498 debt.
func drawItemStarTrail(dst *image.RGBA, origin image.Point, phase uint32, field *itemStarField) {
	if dst == nil || field == nil {
		return
	}
	head := int(phase >> 1)
	ages := len(itemStarAlpha)
	if head+1 < ages {
		ages = head + 1
	}
	for age := 0; age < ages; age++ {
		point := field.points[(head-age)&itemStarPointMask].Add(origin)
		drawItemStarKernel(dst, point, itemStarAlpha[age])
	}
}

// drawItemStarKernel is ITEM-STARPIX-098's thirteen pixels: centre, four
// axial neighbours at distance one, four at distance two, and four diagonals.
func drawItemStarKernel(dst *image.RGBA, centre image.Point, alpha uint8) {
	paintItemStarPixel(dst, centre, alpha)
	a1, a2, ad := uint8(uint16(alpha)*3/4), alpha/2, alpha/4
	for _, d := range []image.Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		paintItemStarPixel(dst, centre.Add(d), a1)
		paintItemStarPixel(dst, centre.Add(image.Pt(d.X*2, d.Y*2)), a2)
	}
	for _, d := range []image.Point{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		paintItemStarPixel(dst, centre.Add(d), ad)
	}
}

// paintItemStarPixel source-overs premultiplied magenta onto one destination
// pixel. A fully bright centre is exactly RGBA(255,0,255,255).
func paintItemStarPixel(dst *image.RGBA, p image.Point, alpha uint8) {
	if alpha == 0 || !p.In(dst.Bounds()) {
		return
	}
	off := dst.PixOffset(p.X, p.Y)
	a, inv := uint32(alpha), uint32(0xff-alpha)
	dst.Pix[off+0] = uint8(a + uint32(dst.Pix[off+0])*inv/0xff)
	dst.Pix[off+1] = uint8(uint32(dst.Pix[off+1]) * inv / 0xff)
	dst.Pix[off+2] = uint8(a + uint32(dst.Pix[off+2])*inv/0xff)
	dst.Pix[off+3] = uint8(a + uint32(dst.Pix[off+3])*inv/0xff)
}

// shiftItemStarPhases applies ITEM-STARPHASE-099's asymmetric visible-slot
// scroll. Forward/down moves old slot i+1 into i and clears the new final
// slot. Back/up moves old slot i into i+1 while slot zero remains unchanged.
func shiftItemStarPhases(phases []uint32, delta int) {
	for ; delta > 0; delta-- {
		if len(phases) == 0 {
			return
		}
		copy(phases, phases[1:])
		phases[len(phases)-1] = 0
	}
	for ; delta < 0; delta++ {
		for i := len(phases) - 1; i > 0; i-- {
			phases[i] = phases[i-1]
		}
	}
}

// shopStarState is the paint-owned phase array for the three measured shop
// grids. It never enters TownScreen or a save. Offsets and producer changes
// reconcile before a paint; eligible cells advance once after that paint.
type shopStarState struct {
	ready                   bool
	chosen, member          int
	shelfOffset, packOffset int
	shelf                   [shopShelfN]uint32
	table, pack             [shopStripN]uint32
}

func (s *shopStarState) reset() { *s = shopStarState{} }

func (s *shopStarState) prepare(v *ShopScreenView, origin ShopControl, dragging bool) {
	if s == nil || v == nil {
		return
	}
	if !s.ready {
		s.ready = true
		s.chosen, s.member = v.Chosen, v.Member
		s.shelfOffset, s.packOffset = v.ShelfOffset, v.PackOffset
	}
	if s.chosen != v.Chosen {
		s.shelf = [shopShelfN]uint32{}
		s.chosen, s.shelfOffset = v.Chosen, v.ShelfOffset
	} else {
		shiftItemStarPhases(s.shelf[:], v.ShelfOffset-s.shelfOffset)
		s.shelfOffset = v.ShelfOffset
	}
	if s.member != v.Member {
		s.pack = [shopStripN]uint32{}
		s.member, s.packOffset = v.Member, v.PackOffset
	} else {
		shiftItemStarPhases(s.pack[:], v.PackOffset-s.packOffset)
		s.packOffset = v.PackOffset
	}
	s.copyPhases(v)
	if dragging {
		omitShopStarSingleton(v, origin)
	}
}

func (s *shopStarState) copyPhases(v *ShopScreenView) {
	for i := range v.Shelf {
		v.Shelf[i].StarPhase = s.shelf[i]
	}
	for i := range v.Table {
		v.Table[i].StarPhase = s.table[i]
		v.Pack[i].StarPhase = s.pack[i]
	}
}

func omitShopStarSingleton(v *ShopScreenView, origin ShopControl) {
	var cell *ShopCell
	switch origin.Kind {
	case ShopControlShelfCell:
		if origin.Index >= 0 && origin.Index < len(v.Shelf) {
			cell = &v.Shelf[origin.Index]
		}
	case ShopControlTableCell:
		if origin.Index >= 0 && origin.Index < len(v.Table) {
			cell = &v.Table[origin.Index]
		}
	case ShopControlPackCell:
		if origin.Index >= 0 && origin.Index < len(v.Pack) {
			cell = &v.Pack[origin.Index]
		}
	}
	if cell != nil && cell.Count <= 1 {
		cell.Icon, cell.Star = nil, false
	}
}

// advance credits this paint before its cell painters run, then copies the
// advanced phases into the view. This is the shop-repaint-clock order in
// ITEM-STARPHASE-099; mission uses its separate tick-owned clock.
func (s *shopStarState) advance(v *ShopScreenView) {
	if s == nil || !s.ready || v == nil {
		return
	}
	for i := range v.Shelf {
		if v.Shelf[i].Star && v.Shelf[i].Icon != nil {
			s.shelf[i]++
		}
	}
	if !v.Book {
		for i := range v.Table {
			if v.Table[i].Star && v.Table[i].Icon != nil {
				s.table[i]++
			}
		}
	}
	for i := range v.Pack {
		if v.Pack[i].Star && v.Pack[i].Icon != nil {
			s.pack[i]++
		}
	}
	s.copyPhases(v)
}
