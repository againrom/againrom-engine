package ui

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"sort"
	"testing"
)

// frameInstanceHashes draws each frame instance that is a frameSpec at fixed
// inputs and returns a hash of its pixels by name.
func frameInstanceHashes(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	sum := func(name string, pix []byte) {
		out[name] = fmt.Sprintf("%x", sha256.Sum256(pix))[:16]
	}
	canvas := func(w, h int) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for i := 0; i < len(img.Pix); i += 4 {
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = byte(i/4%251), 60, byte(i/4%13*9), 255
		}
		return img
	}
	pic := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for i := 0; i < len(pic.Pix); i += 4 {
		pic.Pix[i], pic.Pix[i+1], pic.Pix[i+2], pic.Pix[i+3] = 220, 10, 10, byte(255*(i/4%2))
	}

	for _, border := range []color.RGBA{invCellBorder, spellbookSelected} {
		for _, p := range []*image.RGBA{nil, pic} {
			img := canvas(40, 40)
			drawInventoryCell(img, image.Rect(4, 5, 36, 33), p, border)
			sum(fmt.Sprintf("cell/%v/%v", border, p != nil), img.Pix)
		}
	}
	for _, selected := range []bool{false, true} {
		img := canvas(60, 50)
		drawTownShellBox(img, image.Rect(5, 6, 50, 40), selected)
		sum(fmt.Sprintf("shellbox/%v", selected), img.Pix)
		img = canvas(60, 50)
		drawTownShellOutline(img, image.Rect(5, 6, 50, 40), selected)
		sum(fmt.Sprintf("shelloutline/%v", selected), img.Pix)
	}
	for _, dir := range []int{-1, 1} {
		for _, live := range []bool{false, true} {
			for _, fill := range []bool{false, true} {
				img := canvas(40, 40)
				drawShopChevron(img, image.Rect(3, 4, 27, 30), dir, live, fill)
				sum(fmt.Sprintf("chevron/%d/%v/%v", dir, live, fill), img.Pix)
			}
		}
	}
	for _, slot := range []int{0, 1} {
		img := canvas(640, 480)
		drawWorldMapCard(img, WorldMapView{}, 0, slot, "", "", 0)
		sum(fmt.Sprintf("worldmapcard/%d", slot), img.Pix)
	}
	for _, selected := range []bool{false, true} {
		img := canvas(60, 40)
		drawPushButton(img, nil, pushButton{Rect: image.Rect(4, 5, 50, 30), Style: &pushButtonRowList, Selected: selected})
		sum(fmt.Sprintf("rowlistbutton/%v", selected), img.Pix)
	}

	e, _ := selectedTextEditor(t, panelFont(), image.Pt(1280, 800))
	sum("chrome/text", e.ChromeImage().Pix)
	x := newTintFixture(t)
	x.e.doc.Edits = &tintProbe{}
	_ = x.e.Key("playertint")
	sum("chrome/legend", x.e.ChromeImage().Pix)
	x.e.selected = 1
	sum("chrome/legend-focus", x.e.ChromeImage().Pix)
	return out
}

// pinnedFrameHashes were recorded before the frames became specs.
var pinnedFrameHashes = map[string]string{
	"cell/{138 116 70 255}/false": "a77a3c9932c458a9",
	"cell/{138 116 70 255}/true":  "87db8cf236114237",
	"cell/{255 215 60 255}/false": "22d46a146c81a98c",
	"cell/{255 215 60 255}/true":  "4f0fc206b786f15c",
	"chevron/-1/false/false":      "75fd591120ea3313",
	"chevron/-1/false/true":       "e19003b9ddd8435a",
	"chevron/-1/true/false":       "8497f17b6ed4a938",
	"chevron/-1/true/true":        "3dcf9af0d455b748",
	"chevron/1/false/false":       "96f8d4a56344ec31",
	"chevron/1/false/true":        "4c3eb88ab348e6fe",
	"chevron/1/true/false":        "4d2ab2e31038dc58",
	"chevron/1/true/true":         "1608ba0f5c3487e7",
	"chrome/legend":               "0720454198a9a732",
	"chrome/legend-focus":         "07ec1f120372a313",
	"chrome/text":                 "4abaf9eeb0acd514",
	"rowlistbutton/false":         "6be89f554820f310",
	"rowlistbutton/true":          "0dee5b049ef0ac70",
	"shellbox/false":              "46430c439204827a",
	"shellbox/true":               "808401d28c8ef9c7",
	"shelloutline/false":          "e297e7ca8af32da0",
	"shelloutline/true":           "fcc1473bee45cdb4",
	"worldmapcard/0":              "a3589423eb7351d2",
	"worldmapcard/1":              "48047d72ad639eb1",
}

// TestFrameInstancesKeepTheirPixels pins the pixels of every frame that moved
// from a painter of its own to a frameSpec.
func TestFrameInstancesKeepTheirPixels(t *testing.T) {
	got := frameInstanceHashes(t)
	names := make([]string, 0, len(got))
	for n := range got {
		names = append(names, n)
	}
	sort.Strings(names)
	if len(pinnedFrameHashes) != len(got) {
		t.Fatalf("%d pinned, %d drawn", len(pinnedFrameHashes), len(got))
	}
	for _, n := range names {
		if pinnedFrameHashes[n] != got[n] {
			t.Errorf("%s: %s, pinned %s", n, got[n], pinnedFrameHashes[n])
		}
	}
}
