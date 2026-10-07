package ui

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/vfs"
)

// TestReleaseDocumentsPanelComposesTheShippedPictureElement commits the
// picture-element witness adversarial review pass 1 built as a throwaway
// (pipeline/reviews/1035-adversarial-pass1-return.md, finding 4): the one
// shipped picture element ([Mission60]'s AddPictureDocum 1, value 1,
// MISSION-DOC-021) composed through the production composeDocumentsPanel,
// compared pixel for pixel against an independently built frame. Nothing else
// committed drives this arm: the story's own scenario walks mission 10, whose
// three elements are all texts (documents_release_test.go's own header, in
// pkg/game).
//
// This file lives in package ui, not beside the other release tests in
// pkg/game, for the same import-cycle reason commandpanel_release_test.go
// gives: composeDocumentsPanel and newDocumentPanel are unexported, and
// pkg/game already imports pkg/ui. It loads the panel's eleven bitmaps and
// the one shipped picture directly through pkg/vfs and pkg/formats/bmp, the
// same two packages pkg/game's LoadDocumentPanelArt and LoadDocumentPage
// (docsart.go) use, at the same addresses that file names — duplicated here
// as literals because those constants are unexported in a different package.
// bmpToRGBA and draw2 are commandpanel_release_test.go's own helpers, reused
// rather than restated a second time.
func TestReleaseDocumentsPanelComposesTheShippedPictureElement(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the picture element composition witness needs a lawful install")
	}
	hosts := []string{
		filepath.Join(root, "main.res"),
		filepath.Join(root, "graphics.res"),
		filepath.Join(root, "scenario.res"),
		filepath.Join(root, "world.res"),
		filepath.Join(root, "movies.res"),
	}
	containers, err := vfs.Open(hosts, nil)
	if err != nil {
		t.Fatalf("vfs.Open(%v): %v", hosts, err)
	}
	read := func(addr string) *image.RGBA {
		t.Helper()
		data, err := containers.ReadFile(addr)
		if err != nil {
			t.Fatalf("%s: %v", addr, err)
		}
		im, err := bmp.Decode(data)
		if err != nil {
			t.Fatalf("%s: %v", addr, err)
		}
		return bmpToRGBA(im)
	}

	art := &DocumentPanelArt{Sheet: read("graphics/interface/docs/sheet.bmp")}
	arrowFiles := [3]string{"00_", "01_", "11_"}
	for i, stem := range arrowFiles {
		art.Left[i] = read("graphics/interface/docs/arrows/" + stem + "l.bmp")
		art.Right[i] = read("graphics/interface/docs/arrows/" + stem + "r.bmp")
	}
	okFiles := [4]string{"ok_off.bmp", "ok_on.bmp", "ok_l_off.bmp", "ok_l_on.bmp"}
	for i, name := range okFiles {
		art.OK[i] = read("graphics/interface/docs/ok/" + name)
	}

	// The one shipped picture element, [Mission60]'s AddPictureDocum 1
	// (MISSION-DOC-021, docs/1035-documents/spec.md).
	picture := read("graphics/interface/docs/1.bmp")
	if picture.Bounds().Dx() != 464 || picture.Bounds().Dy() != 344 {
		t.Fatalf("graphics/interface/docs/1.bmp is %dx%d, want 464x344 -- the shipped picture this witness pins has changed shape",
			picture.Bounds().Dx(), picture.Bounds().Dy())
	}

	p := newDocumentPanel(art, nil, []DocumentPage{{Picture: picture}})
	got := composeDocumentsPanel(p)

	// EXPECTED FRAME, BUILT INDEPENDENTLY OF THE COMPOSER: the sheet with
	// idle arrows and idle OK, then the picture painted at docContentAt (92,
	// 72) at natural size over it -- composeDocumentsPanel's own draw order
	// is sheet, arrows, OK, then the document last, over the sheet
	// (documents.go's own header on composeDocumentsPanel). No resampling
	// and no colour key: the picture's own pixels are the frame's pixels
	// wherever it lands.
	// blitAt is the offset copy this expected-frame builder needs: draw2
	// (commandpanel_release_test.go) samples src at the SAME coordinates as
	// the destination, which only agrees with an "at" placement when the
	// source's own bounds start at the destination rectangle's own origin.
	// docLeftRect, docRightRect and docOKRect do not (they sit at (0,200),
	// (576,200) and (560,416) in frame pixels; the loaded bitmaps are all
	// zero-based), so this helper places src's own (0,0) at "at" instead,
	// the same placement copyNative (documents.go) uses.
	blitAt := func(dst *image.RGBA, src *image.RGBA, at image.Point) {
		b := src.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				dst.SetRGBA(at.X+x-b.Min.X, at.Y+y-b.Min.Y, src.RGBAAt(x, y))
			}
		}
	}

	want := image.NewRGBA(docSheetRect)
	blitAt(want, art.Sheet, docSheetRect.Min)
	blitAt(want, art.Left[0], docLeftRect.Min)
	blitAt(want, art.Right[0], docRightRect.Min)
	blitAt(want, art.OK[0], docOKRect.Min)
	blitAt(want, picture, docContentAt)

	if !imagesEqual(got, want) {
		t.Error("composeDocumentsPanel does not match an independently built frame for the shipped picture element")
	}

	// THE OVERHANG IS THE FINDING'S OWN NUMBER: the picture is 464px wide
	// against docContentWidth's 456, an 8px overhang the document rectangle
	// does not clip (documents.go's own header on docContentWidth). Assert
	// it composed rather than assume the whole-frame compare above already
	// covers it: a clip that started cutting the picture to the content
	// rectangle would narrow both sides of that compare together and still
	// pass it.
	overhang := image.Rect(docContentAt.X+docContentWidth, docContentAt.Y, docContentAt.X+464, docContentAt.Y+344)
	matched := 0
	for y := overhang.Min.Y; y < overhang.Max.Y; y++ {
		for x := overhang.Min.X; x < overhang.Max.X; x++ {
			if got.RGBAAt(x, y) == picture.RGBAAt(x-docContentAt.X, y-docContentAt.Y) {
				matched++
			}
		}
	}
	total := overhang.Dx() * overhang.Dy()
	if matched != total {
		t.Errorf("the 8px overhang past the content rectangle: %d of %d picture pixels matched, want all of them",
			matched, total)
	}
	picturePixels := picture.Bounds().Dx() * picture.Bounds().Dy()
	t.Logf("picture element: %d of %d pixels matched the composed frame; overhang %d of %d",
		picturePixels, picturePixels, matched, total)
}
