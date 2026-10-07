package ui

import (
	"image"
	"testing"

	"againrom/pkg/render/text"
)

// recordedText turns recorded glyph draws back into the bytes that selected
// them, for a font whose records are all distinct.
func recordedText(font *text.Font, calls []text.DrawCall) string {
	byGlyph := make(map[*text.Glyph]byte, 256)
	for c := 255; c >= 0; c-- {
		if g := font.GlyphFor(byte(c)); g != nil {
			byGlyph[g] = byte(c)
		}
	}
	out := make([]byte, 0, len(calls))
	for _, call := range calls {
		out = append(out, byGlyph[call.Glyph])
	}
	return string(out)
}

// TestGroupingKeepsTheSignInFront is TOWN-469's rule: a comma before every
// three digits from the right, and a leading sign kept with no comma after it.
func TestGroupingKeepsTheSignInFront(t *testing.T) {
	for _, tc := range []struct {
		got, want string
	}{
		{GroupDigits(-1000), "-1,000"}, {GroupDigits(-999), "-999"}, {GroupDigits(1234567), "1,234,567"},
		{GroupSigned(1000), "+1,000"}, {GroupSigned(-1000), "-1,000"}, {GroupSigned(999), "+999"},
		{GroupSigned(0), "+0"}, {GroupSigned(-100000), "-100,000"},
	} {
		if tc.got != tc.want {
			t.Errorf("got %q, want %q", tc.got, tc.want)
		}
	}
}

// TestShopGridGroupsQuantityPriceAndMoney covers the item-grid money cell and
// the shop stock grid's quantity and price (TOWN-469).
func TestShopGridGroupsQuantityPriceAndMoney(t *testing.T) {
	font := townShellRosterTestFont()
	dst := image.NewRGBA(image.Rect(0, 0, 200, 200))
	r := image.Rect(0, 0, 80, 80)
	for _, tc := range []struct {
		cell   ShopCell
		want   string
		shadow string
	}{
		// An occupied cell prints its price after its quantity, a price of 0
		// included (ITEM-PRICETAG-144).
		{ShopCell{Back: ShopBackItem, Count: 2500}, "2,500" + "0", "2,500" + "0"},
		{ShopCell{Back: ShopBackItem, Count: 1, Price: 12000}, "12,000", "12,000"},
		{ShopCell{Money: true, Count: 534000}, "534,000", "534,000"},
	} {
		calls := text.Record(func() { drawShopCell(dst, nil, font, nil, tc.cell, r, false, nil) })
		var faces, shadows []text.DrawCall
		for _, call := range calls {
			if call.Flat {
				shadows = append(shadows, call)
			} else {
				faces = append(faces, call)
			}
		}
		if got := recordedText(font, faces); got != tc.want {
			t.Errorf("cell %+v drew %q, want %q", tc.cell, got, tc.want)
		}
		if got := recordedText(font, shadows); got != tc.shadow {
			t.Errorf("cell %+v shadow drew %q, want %q", tc.cell, got, tc.shadow)
		}
	}
}

// TestHallScoreIsGroupedAndItsRankIsNot: the hall of fame's score goes
// through the grouping routine; its "%d." rank does not (TOWN-469).
func TestHallScoreIsGroupedAndItsRankIsNot(t *testing.T) {
	if got := hallScoreText(-1234); got != "-1,234" {
		t.Fatalf("hall score = %q", got)
	}
}

// TestGeneratorStepTextIsSignedAndGrouped is the generator's `%+d` text: the
// raise cost negated reads negative, the lowering refund reads positive.
func TestGeneratorStepTextIsSignedAndGrouped(t *testing.T) {
	cost := make([]int, 51)
	for v := range cost {
		cost[v] = v * 1000
	}
	c := NewChargen(ChargenSetup{Stats: []ChargenStat{{Floor: 0, Ceiling: 50, Start: 25}}, Cost: cost, Budget: 100000})
	if got, ok := c.StatStepText(0, true); !ok || got != "-1,000" {
		t.Fatalf("raise text = %q, %v; want -1,000", got, ok)
	}
	if got, ok := c.StatStepText(0, false); !ok || got != "+1,000" {
		t.Fatalf("lower text = %q, %v; want +1,000", got, ok)
	}
}

// TestMissionPackGroupsThePurseOnly: the mission grid's money cell is a
// TOWN-469 grouping site; an item stack's count is not.
func TestMissionPackGroupsThePurseOnly(t *testing.T) {
	font := townShellRosterTestFont()
	icon := image.NewRGBA(image.Rect(0, 0, 18, 18))
	s := InventorySubject{Pack: []*image.RGBA{icon, icon}, PackCount: []uint32{2500, 5001500},
		PackPurse: []bool{false, true}}
	calls := text.Record(func() { renderPackBarArt(s, 0, 9, image.Rect(0, 678, 864, 768), font, nil, -1, nil) })
	got := recordedText(font, calls)
	// Each count draws twice: shadow, then face.
	if want := "25002500" + "5,001,5005,001,500"; got != want {
		t.Fatalf("pack counts drew %q, want %q", got, want)
	}
}

// TestSightPrintsOneTruncatedDecimal is HERO-104's first `%d.%d` row: whole
// cells, then one decimal truncated toward zero. The owner's three mage
// panels read 6.5, 6.3 and 6.3 for mind+reaction 63, 59 and 59.
func TestSightPrintsOneTruncatedDecimal(t *testing.T) {
	for _, tc := range []struct {
		raw  uint16
		want string
	}{{1665, "6.5"}, {1628, "6.3"}, {1535, "5.9"}, {1536, "6.0"}} {
		s := PanelSubject{Char: UnitCharacter{Known: true, Sight256: tc.raw}}
		if got, _ := panelText(s, PanelFieldSight); got != tc.want {
			t.Errorf("sight %d = %q, want %q", tc.raw, got, tc.want)
		}
	}
}
