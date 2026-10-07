package ui

import (
	"image"
	"image/color"
	"image/draw"

	"againrom/pkg/render/text"
)

var characterCardFill = color.RGBA{33, 44, 33, 255}
var characterCardHeading = color.RGBA{148, 89, 0, 255}

// characterCardBodyHeight is the body height characterCardRowY is laid out for.
const characterCardBodyHeight = 242

// Preserve the installed ornament; recolor only its flat writing surface to
// the color in the owner's original-game reference.
func characterCardBackground(src *image.RGBA) *image.RGBA {
	if src == nil || src.Bounds().Empty() {
		return src
	}
	b := src.Bounds()
	fill := src.RGBAAt(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2)
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if src.RGBAAt(x, y) == fill {
				dst.SetRGBA(x, y, characterCardFill)
			}
		}
	}
	return dst
}

func fitCharacterCardLines(l PanelLayout, f *text.Font, lines []panelLine, items []panelItem) {
	bg := l.Background
	hasRole := false
	for _, ln := range lines {
		if ln.field == PanelFieldRole && ln.value != "" {
			hasRole = true
			break
		}
	}
	// Text may use only the flat face over its entire glyph height. The shoulder
	// and lower corners have different widths; a rectangular canvas is too wide.
	for i := range lines {
		ln := &lines[i]
		// Fit the complete values against the actual face, not an already
		// truncated result from the generic rectangular column budget.
		ln.label, ln.value = items[i].label, items[i].value
		ln.rightLabel, ln.rightValue = items[i].rightLabel, items[i].rightValue
		if y, ok := characterCardRowY(ln.field); ok {
			ln.at.Y = l.Pad.Y + 26 + y
			if ln.field == PanelFieldSight || ln.field == PanelFieldMoveSpeed {
				// A card shorter than the reference body gives up the difference at the
				// foot, where the last two rows would otherwise run onto the ornament.
				ln.at.Y -= max(0, characterCardBodyHeight-bg.Bounds().Dy())
			}
		} else if ln.field == PanelFieldName {
			ln.at.Y = l.Pad.Y
			if hasRole {
				ln.at.Y += f.Height()
			}
		}
		left, right := characterCardRowExtent(l, bg, ln.at.Y, f.Height())
		if right <= left {
			continue
		}
		if ln.field == PanelFieldName && ln.value != "" {
			// A name too wide for its row wraps at a space onto the next row
			// rather than losing its last letters. A name that fits, and one no
			// split fits, keep the single line below. Under a role line the two
			// rows close up by one pixel so the second stays above the first
			// attribute row.
			y1, step := ln.at.Y, f.Height()
			if hasRole {
				y1, step = y1-1, step-1
			}
			if first, second, ok := wrapCardName(l, bg, f, ln.value, right-left, y1, step); ok {
				ln.value, ln.wrap = first, second
				ln.at.Y = y1
				ln.valueX, ln.width = panelCellMetrics(f, l.LabelGap, ln.label, ln.value)
				l1, r1 := characterCardRowExtent(l, bg, y1, f.Height())
				ln.at.X = max(l1, min(cardNameCentre-ln.width/2, r1-ln.width))
				w2, _ := f.Measure(second)
				l2, r2 := characterCardRowExtent(l, bg, y1+step, f.Height())
				ln.wrapAt = image.Pt(max(l2, min(cardNameCentre-w2/2, r2-w2)), y1+step)
				continue
			}
		}
		if ln.field == PanelFieldName || ln.field == PanelFieldRole {
			ln.value = panelFitValue(f, l.LabelGap, ln.label, ln.value, right-left)
			ln.valueX, ln.width = panelCellMetrics(f, l.LabelGap, ln.label, ln.value)
			ln.at.X = max(left, min(cardNameCentre-ln.width/2, right-ln.width))
			continue
		}
		if placeCharacterCardRow(l, bg, f, ln, &items[i]) {
			continue
		}
		x := max(left, ln.at.X)
		end := min(right, l.Size.X-max(l.Pad.X, l.RightInset))
		if ln.right {
			rx := ln.at.X + ln.rightX
			if ln.rightLabel == "" {
				// Large health/mana pools may borrow the inter-column gap.
				// Keep the left attribute and both halves of the pool intact.
				rw, _ := f.Measure(ln.rightValue)
				_, lw := panelCellMetrics(f, l.LabelGap, ln.label, ln.value)
				rx = max(x+lw+2, min(rx, end-rw))
			} else if rw, _ := f.Measure(ln.rightValue); ln.rightValue != "" {
				// A wide value (a three-digit resistance) leaves less than one
				// space between the caption and the number. The row's right cell
				// slides left by the shortfall, never past its own left cell.
				need := f.Advance(ln.rightLabel) + max(l.LabelGap, f.Advance(" ")) + rw
				if end-rx < need {
					_, lw := panelCellMetrics(f, l.LabelGap, ln.label, ln.value)
					rx = max(x+lw+l.ColumnGap, min(rx, end-need))
				}
			}
			ln.label = panelFitLabelForValue(f, l.LabelGap, ln.label, ln.value, max(0, rx-l.ColumnGap-x))
			ln.value = panelFitValue(f, l.LabelGap, ln.label, ln.value, max(0, rx-l.ColumnGap-x))
			vw, _ := f.Measure(ln.value)
			ln.valueX = max(f.Advance(ln.label)+l.LabelGap, rx-l.ColumnGap-x-vw)
			ln.rightLabel = panelFitLabelForValue(f, l.LabelGap, ln.rightLabel, ln.rightValue, max(0, end-rx))
			ln.rightValue = panelFitValue(f, l.LabelGap, ln.rightLabel, ln.rightValue, max(0, end-rx))
			rw, _ := f.Measure(ln.rightValue)
			ln.rightValueX = max(f.Advance(ln.rightLabel)+l.LabelGap, end-rx-rw)
			ln.rightX = rx - x
		} else {
			if !items[i].fullWidth {
				// A missing right cell does not turn an attribute into a
				// full-width total. Keep MIND/SPIRIT at the shared left edge.
				columnW := (l.Size.X - l.Pad.X - max(l.Pad.X, l.RightInset) - l.ColumnGap) / 2
				end = min(end, l.Pad.X+columnW)
			}
			ln.label = panelFitLabelForValue(f, l.LabelGap, ln.label, ln.value, end-x)
			ln.value = panelFitValue(f, l.LabelGap, ln.label, ln.value, end-x)
			vw, _ := f.Measure(ln.value)
			ln.valueX = max(f.Advance(ln.label)+l.LabelGap, end-x-vw)
		}
		ln.at.X, ln.width = x, end-x
	}
}

// Native font2 rows have a ten-pixel pitch inside each group. The extra
// gaps match the owner's original character-card reference. Reserving each
// field's slot also keeps hidden values from moving unrelated groups.
func characterCardRowY(field PanelField) (int, bool) {
	switch field {
	case PanelFieldBody, PanelFieldHealthHeading:
		return 0, true
	case PanelFieldReaction, PanelFieldHealth:
		return 10, true
	case PanelFieldMind, PanelFieldManaCardHeading:
		return 20, true
	case PanelFieldSpirit, PanelFieldManaCard:
		return 30, true
	case PanelFieldDamage, PanelFieldAbsorption:
		return 44, true
	case PanelFieldToHit, PanelFieldDefence:
		return 54, true
	case PanelFieldSkillsHeading, PanelFieldResistHeading:
		return 68, true
	case PanelFieldSkillBlade, PanelFieldProtFire:
		return 78, true
	case PanelFieldSkillAxe, PanelFieldProtWater:
		return 88, true
	case PanelFieldSkillBludgeon, PanelFieldProtAir:
		return 98, true
	case PanelFieldSkillPike, PanelFieldProtEarth:
		return 108, true
	case PanelFieldSkillShooting, PanelFieldProtAstral:
		return 118, true
	case PanelFieldWeight, PanelFieldSpellcaster:
		return 130, true
	case PanelFieldExperience, PanelFieldArmorPiercing:
		return 142, true
	case PanelFieldSight:
		return 156, true
	case PanelFieldMoveSpeed:
		return 166, true
	}
	return 0, false
}

// characterCardRowExtent is the writable span of the card's flat face over the
// glyph rows y..y+h, with the one-pixel safety margin on each side.
func characterCardRowExtent(l PanelLayout, bg *image.RGBA, y, h int) (left, right int) {
	left, right = 0, l.Size.X
	for ; h > 0; y, h = y+1, h-1 {
		lo, hi := l.Size.X, 0
		for x := 0; x < l.Size.X; x++ {
			if bg.RGBAAt(bg.Bounds().Min.X+x, bg.Bounds().Min.Y+y) == characterCardFill {
				lo, hi = min(lo, x), max(hi, x+1)
			}
		}
		left, right = max(left, lo), min(right, hi)
	}
	return left + 1, right - 1
}

// wrapCardName splits a name that does not fit its row at a space into two
// rows, at y and y+step, that each fit, choosing the split whose wider row is
// narrower. It reports false when the name already fits one row or no split
// fits.
func wrapCardName(l PanelLayout, bg *image.RGBA, f *text.Font, name string, avail, y, step int) (first, second string, ok bool) {
	if w, _ := f.Measure(name); w <= avail {
		return "", "", false
	}
	l1, r1 := characterCardRowExtent(l, bg, y, f.Height())
	l2, r2 := characterCardRowExtent(l, bg, y+step, f.Height())
	best := 0
	for i := 1; i < len(name)-1; i++ {
		if name[i] != ' ' {
			continue
		}
		w1, _ := f.Measure(name[:i])
		w2, _ := f.Measure(name[i+1:])
		if w1 > r1-l1 || w2 > r2-l2 {
			continue
		}
		if wide := max(w1, w2); !ok || wide < best {
			first, second, best, ok = name[:i], name[i+1:], wide, true
		}
	}
	return first, second, ok
}

// The card's text columns in card pixels, measured from the original
// statistics card: labels flush left on a pen, values flush right on an end.
const (
	cardLabelPen       = 6
	cardAttrValueEnd   = 80
	cardLowerValueEnd  = 70
	cardRightLabelPen  = 74
	cardRightValueEnd  = 140
	cardPoolCentre     = 115
	cardNameCentre     = 72
	cardPoolGap        = 2
	cardColumnGap      = 4
	cardTotalsPen      = 16
	cardTotalsValueEnd = 127
	cardGaugePen       = 40
	cardGaugeValueEnd  = 106
)

type cardRowGroup int

const (
	cardGroupNone cardRowGroup = iota
	cardGroupAttributes
	cardGroupLower
	cardGroupTotals
	cardGroupGauge
)

func characterCardGroup(field PanelField) cardRowGroup {
	y, ok := characterCardRowY(field)
	switch {
	case !ok:
		return cardGroupNone
	case y <= 30:
		return cardGroupAttributes
	case y <= 118:
		return cardGroupLower
	case y <= 142:
		return cardGroupTotals
	}
	return cardGroupGauge
}

func characterCardPoolField(field PanelField) bool {
	switch field {
	case PanelFieldHealthHeading, PanelFieldHealth, PanelFieldManaHeading, PanelFieldMana,
		PanelFieldManaCardHeading, PanelFieldManaCard:
		return true
	}
	return false
}

// characterCardWritable reports whether a background pixel may carry text: the
// flat writing surface and the soft shadow its edge fades through. The gold
// outline and the ornaments are neither.
func characterCardWritable(c color.RGBA) bool {
	if c == characterCardFill {
		return true
	}
	return c.R <= 46 && c.G <= 46 && c.B <= 46 && c.B >= 18 &&
		int(c.G)-int(c.B) <= 12 && int(c.G)-int(c.R) <= 12
}

// CharacterCardWritable is characterCardWritable for tests outside the package.
func CharacterCardWritable(c color.RGBA) bool { return characterCardWritable(c) }

// characterCardWritableExtent is the span writable over the glyph rows y..y+h
// of the card, with the one-pixel safety margin on each side.
func characterCardWritableExtent(l PanelLayout, bg *image.RGBA, y, h int) (left, right int) {
	left, right = 0, l.Size.X
	for ; h > 0; y, h = y+1, h-1 {
		lo, hi := l.Size.X, 0
		for x := 0; x < l.Size.X; x++ {
			if characterCardWritable(bg.RGBAAt(bg.Bounds().Min.X+x, bg.Bounds().Min.Y+y)) {
				lo, hi = min(lo, x), max(hi, x+1)
			}
		}
		left, right = max(left, lo), min(right, hi)
	}
	return left + 1, right - 1
}

// cardCell fits one label and value between a pen and an end and returns the
// value's offset from the pen. The value keeps its budget before the label.
func cardCell(f *text.Font, gap int, label, value string, pen, end int) (string, string, int) {
	avail := max(0, end-pen)
	label = panelFitLabelForValue(f, gap, label, value, avail)
	value = panelFitValue(f, gap, label, value, avail)
	vw, _ := f.Measure(value)
	return label, value, max(f.Advance(label)+gap, avail-vw)
}

// placeCharacterCardRow places one body row on the card's columns. It reports
// false for a row it does not own, which the caller places as before.
func placeCharacterCardRow(l PanelLayout, bg *image.RGBA, f *text.Font, ln *panelLine, it *panelItem) bool {
	group := characterCardGroup(ln.field)
	if group == cardGroupNone || (it.right && group != cardGroupAttributes && group != cardGroupLower) {
		return false
	}
	wl, wr := characterCardWritableExtent(l, bg, ln.at.Y, f.Height())
	if wr <= wl {
		return false
	}
	gap := l.LabelGap
	pen, end := cardLabelPen, cardLowerValueEnd
	switch group {
	case cardGroupAttributes:
		end = cardAttrValueEnd
	case cardGroupTotals:
		pen, end = cardTotalsPen, cardTotalsValueEnd
	case cardGroupGauge:
		pen, end = cardGaugePen, cardGaugeValueEnd
	}

	centred := func(label, value string) (int, string, string) {
		shown := value
		if shown == "" {
			shown = label
		}
		w, _ := f.Measure(shown)
		return min(max(cardPoolCentre-w/2, wl), max(wl, wr-w)), label, value
	}

	rightPen := cardRightLabelPen
	rightEnd := min(cardRightValueEnd, wr)
	poolRight := it.right && characterCardPoolField(it.rightField)
	if poolRight {
		x, rl, rv := centred(it.rightLabel, it.rightValue)
		rightPen = x
		ln.rightLabel, ln.rightValue = rl, rv
		if rl == "" {
			ln.rightValueX = 0
		} else {
			ln.rightValueX = f.Advance(rl) + gap
		}
		// A wide pool pushes the left cell's end in.
		end = min(end, x-cardPoolGap)
	}

	if it.right && !poolRight {
		// A wide value (a three-digit resistance) leaves less than one space
		// between the caption and the number. The right cell slides left by the
		// shortfall, never into its own left cell.
		rvw, _ := f.Measure(it.rightValue)
		need := f.Advance(it.rightLabel) + max(gap, f.Advance(" ")) + rvw
		if rightEnd-rightPen < need {
			_, lw := panelCellMetrics(f, gap, it.label, it.value)
			rightPen = max(max(wl, cardLabelPen)+lw+cardColumnGap, rightEnd-need)
			end = min(end, rightPen-cardColumnGap)
		}
	}

	switch {
	case characterCardPoolField(ln.field) && !it.right:
		x, label, value := centred(it.label, it.value)
		ln.at.X, ln.label, ln.value = x, label, value
		ln.valueX = 0
		if label != "" {
			ln.valueX = f.Advance(label) + gap
		}
		ln.width = x
		return true
	default:
		pen = max(pen, wl)
		end = min(end, wr)
		ln.label, ln.value, ln.valueX = cardCell(f, gap, it.label, it.value, pen, end)
		ln.at.X, ln.width = pen, end-pen
	}

	if it.right {
		if poolRight {
			ln.rightX = rightPen - ln.at.X
			return true
		}
		rightPen = max(rightPen, wl)
		rl, rv, rvx := cardCell(f, gap, it.rightLabel, it.rightValue, rightPen, rightEnd)
		ln.rightLabel, ln.rightValue, ln.rightValueX = rl, rv, rvx
		ln.rightX = max(rightPen, wl) - ln.at.X
	}
	return true
}
