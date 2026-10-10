package ui

import (
	"image"
	"testing"
)

// TestEveryTipRectSitsAtItsResearchedOrigin pins the origin and width of all
// five tip rects against the absolute screen rectangles research resolved for
// them.
//
// THE DEFECT THIS WAS WRITTEN FOR. TOWN-332 names the two
// non-interchangeable mechanisms that compute a paint origin in this engine
// and confirms the popup uses the ancestor-chain accumulator.
//
// Four of the five rects were shipped by reading a published literal as an
// absolute rect. Two of the four were right by luck, because their parent is
// the room view itself and its chain is empty (TOWN-328, town and school).
// Two were wrong by exactly the parent's own origin, +160 in X: the tavern,
// whose parent is the centre/roster panel at (160,0)-(480,480) (TOWN-329),
// and the character generator, whose parent is the +0x7c child at the same
// origin (TOWN-314). The fifth, the shop, was right because SHOP-TIP-045 did
// the conversion itself and published both forms.
//
// Nothing failed. Both wrong rects carried the correct WIDTH, so both panels
// drew at the right size in the wrong place, and every test that mentions
// them reads the constant symbolically and moved with it. Reverting the
// tavern's origin to x=0 reddened no test in the module before this one
// existed.
//
// WHY ORIGIN AND WIDTH AND NOT HEIGHT. Height is the deviation DIV-162
// discloses: four of the five rects are taller than the researched 200 rows
// so that the shipped tip text draws whole, and production resolves each
// room's own shown height per root at construction (TipPanelShrinkRect).
// Origin and width are not deviations and are not authored.
//
// The expected values are the claims' own, written as literals here, not
// derived from the constants under test.
func TestEveryTipRectSitsAtItsResearchedOrigin(t *testing.T) {
	for _, tc := range []struct {
		name   string
		claim  string
		got    image.Rectangle
		origin image.Point
		width  int
	}{
		{
			// TOWN-328: the town view is the popup's own parent and its
			// ancestor chain is empty, so TOWN-314's literal
			// (0x148,0,0x280,0xc8) is already absolute.
			name: "town square", claim: "TOWN-328",
			got: TownTipRect, origin: image.Pt(328, 0), width: 312,
		},
		{
			// TOWN-329: the parent is [ESI+0x7c], the tavern's own
			// centre/roster panel at (160,0)-(480,480), so the walk adds
			// (160,0) to the literal (0,0,0x138,0xc8).
			name: "tavern", claim: "TOWN-329",
			got: TavernTipRect, origin: image.Pt(160, 0), width: 312,
		},
		{
			// TOWN-328: the school view is the parent, same as town.
			name: "school", claim: "TOWN-328",
			got: SchoolTipRect, origin: image.Pt(0, 0), width: 456,
		},
		{
			// TOWN-314: the parent is the +0x7c child at
			// (160,0)-(480,480), so the literal (0,0x118,0x138,0x1e0)
			// resolves to (160,280)-(472,480) — lower CENTRE, straddling
			// the screen midpoint, not lower left.
			name: "character generator", claim: "TOWN-314",
			got: ChargenTipRect, origin: image.Pt(160, 280), width: 312,
		},
		{
			// SHOP-TIP-045 published the panel-relative (0,162,312,298)
			// and the view-relative (164,162,476,298) together, which is
			// why this one never carried the defect.
			name: "shop", claim: "SHOP-TIP-045",
			got: shopTipRect, origin: image.Pt(164, 162), width: 312,
		},
	} {
		if tc.got.Min != tc.origin {
			t.Errorf("%s tip rect %v: origin = %v, want %v (%s, the popup's own ABSOLUTE rect after the ancestor walk; a published construction literal is parent-relative)",
				tc.name, tc.got, tc.got.Min, tc.origin, tc.claim)
		}
		if tc.got.Dx() != tc.width {
			t.Errorf("%s tip rect %v: width = %d, want %d (%s)", tc.name, tc.got, tc.got.Dx(), tc.width, tc.claim)
		}
	}
}

// TestShopTipPanelRectKeepsTheWidgetOrigin pins the original panel rectangle
// before any text-dependent height adjustment (SHOP-TIP-045).
func TestShopTipPanelRectKeepsTheWidgetOrigin(t *testing.T) {
	panel := shopTipRect
	if panel.Min != image.Pt(164, 162) {
		t.Errorf("shopTipRect = %v: origin = %v, want (164,162) (SHOP-TIP-045)", panel, panel.Min)
	}
	if panel.Dx() != 312 {
		t.Errorf("shopTipRect = %v: width = %d, want 312 (SHOP-TIP-045)", panel, panel.Dx())
	}
	if panel.Dy() != 136 {
		t.Errorf("shopTipRect = %v: height = %d, want 136 (SHOP-TIP-045)", panel, panel.Dy())
	}
}
