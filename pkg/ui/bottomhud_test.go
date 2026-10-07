package ui

import (
	"image"
	"image/color"
	"testing"
)

func TestOriginalBookRowsAndPointerIdentity(t *testing.T) {
	// MAGIC-ICON-024's literal slot order, independent of the game producer.
	ids := []uint32{1, 2, 3, 4, 5, 23, 24, 16, 15, 14, 13, 12, 6, 7, 8, 9, 10, 25, 26, 22, 21, 20, 19, 18}
	for _, size := range []image.Point{{640, 480}, {800, 600}, {1024, 768}} {
		bar, cols, ok := bookBarRect(size)
		if !ok || cols != 12 || bar != image.Rect(0, size.Y-175, size.X-160, size.Y-90) {
			t.Fatal("book does not meet the pack", size, bar, cols)
		}
		cells := bookCellRects(bar, cols)
		for i, r := range cells {
			x, y := (size.X-640)/2+6+38*(i%12), size.Y-169+38*(i/12)
			if r != image.Rect(x, y, x+36, y+36) {
				t.Fatalf("size=%v slot=%d rectangle=%v", size, i, r)
			}
		}
	}
	a := spellbookTestApp(t)
	a.Layout(1024, 768)
	v := a.flow.viewer
	v.SetEntities(sbEntities())
	v.sel = selection{sbUnitID}
	v.SetInventorySubject(InventorySubject{ID: sbUnitID})
	entries := make([]SpellEntry, 24)
	for i, id := range ids {
		entries[i] = SpellEntry{ID: id, Name: "spell"}
	}
	v.SetSpellbookCatalog(sbUnitID, entries)
	if bar, cols, ok := v.spellbookBar(); !ok || bar != image.Rect(0, 593, 864, 678) || cols != 12 {
		t.Fatalf("fixture book=%v columns=%d shown=%v frame=%dx%d", bar, cols, ok, v.frameW, v.frameH)
	}
	for i, id := range ids {
		x, y := 216+38*(i%12), 617+38*(i/12)
		if err := a.HeadlessPointer("press", x, y); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessPointer("release", x, y); err != nil {
			t.Fatal(err)
		}
		if v.selectedSpell != id {
			t.Fatalf("slot=%d selected=%d want=%d", i, v.selectedSpell, id)
		}
	}
}

func TestOriginalBookMasksUnknownsAndKeepsNativePixels(t *testing.T) {
	gold, green, hidden := color.RGBA{180, 120, 30, 255}, color.RGBA{20, 40, 20, 255}, color.RGBA{5, 10, 5, 255}
	art := &BottomHUDArt{Book: solidPic(480, 85, green), UnknownSpell: solidPic(36, 36, hidden)}
	art.BookLeft[1], art.BookRight[1] = solidPic(192, 85, gold), solidPic(208, 85, gold)
	entries := make([]SpellEntry, 24)
	for i := range entries {
		entries[i] = SpellEntry{ID: uint32(i + 1)}
		// Distinct pixels at every literal atlas origin expose a transposition
		// or a crop/rescale even when the ID lookup would still work.
		art.Book.SetRGBA(6+38*(i%12), 6+38*(i/12), color.RGBA{byte(i + 50), 70, 90, 255})
	}
	entries[4].Unavailable = true
	bar := image.Rect(0, 593, 864, 678)
	pic := composeOriginalSpellBar(art, entries, 2, bar, 0)
	for i := range entries {
		x, y := 198+38*(i%12), 6+38*(i/12)
		want := color.RGBA{byte(i + 50), 70, 90, 255}
		if i == 1 {
			want = color.RGBA{liftChannel(want.R), liftChannel(want.G), liftChannel(want.B), 255}
			if want.R <= byte(i+50) || want == spellbookSelected {
				t.Fatal("selected slot is not lifted")
			}
		}
		if i == 4 {
			want = hidden
		}
		if got := pic.RGBAAt(x, y); got != want {
			t.Fatalf("slot=%d pixel=%v want=%v", i, got, want)
		}
	}
	if c := pic.RGBAAt(198+38, 6+1); c != (color.RGBA{liftChannel(green.R), liftChannel(green.G), liftChannel(green.B), 255}) {
		t.Fatalf("selected slot edge=%v", c)
	}
	if pic.RGBAAt(0, 0) != gold || pic.RGBAAt(863, 84) != gold {
		t.Fatal("missing book wing")
	}
	// An unavailable saved current cell receives neither the known picture
	// nor its selection border.
	pic = composeOriginalSpellBar(art, entries, 5, bar, 0)
	if pic.RGBAAt(350, 6) != hidden || pic.RGBAAt(385, 41) != hidden {
		t.Fatal("unknown spell mask was narrowed or selected")
	}
}

func TestWidescreenBookKeepsClosingSeamAtOuterEdge(t *testing.T) {
	gold := color.RGBA{180, 120, 30, 255}
	art := &BottomHUDArt{Book: solidPic(464, 85, gold)}
	art.BookLeft[1] = solidPic(192, 85, gold)
	art.BookRight[1] = image.NewRGBA(image.Rect(0, 0, 208, 85))
	blit(art.BookRight[1], solidPic(192, 85, gold), 0, 0)
	// A 1920x1080 window uses a 1366-pixel native frame. Repeating the
	// full 208-pixel right bitmap put a black gap at native x=1019..1035,
	// exactly one closing seam before the second ornamental section.
	for _, width := range []int{864, 1206, 1888} {
		pic := image.NewRGBA(image.Rect(0, 0, width, 85))
		drawBookGround(pic, art)
		for x := 0; x < width-16; x++ {
			if pic.RGBAAt(x, 40) != gold {
				t.Fatal("closing seam repeated inside the book panel", width, x)
			}
		}
		for x := width - 16; x < width; x++ {
			if pic.RGBAAt(x, 40).A != 0 {
				t.Fatal("ornament extends under the sidebar's 16-pixel seam", width, x)
			}
		}
	}
}

func TestInstalledPackFrameEmptyCellsAndCountPlacement(t *testing.T) {
	gold, green, red := color.RGBA{180, 120, 30, 255}, color.RGBA{20, 40, 20, 255}, color.RGBA{200, 0, 0, 255}
	art := &BottomHUDArt{Pack: solidPic(480, 90, gold), PackItem: solidPic(80, 80, green),
		PackLeft: solidPic(32, 90, gold), PackRight: solidPic(48, 90, gold)}
	s := InventorySubject{Pack: []*image.RGBA{solidPic(18, 18, red)}, PackCount: []uint32{100}}
	bar := image.Rect(0, 678, 864, 768)
	pic := renderPackBarArt(s, 0, 9, bar, panelFont(), nil, -1, art)
	if pic.RGBAAt(64, 6) != green || pic.RGBAAt(96, 38) != red || pic.RGBAAt(145, 7) != gold || pic.RGBAAt(8, 20) != gold || pic.RGBAAt(830, 20) != gold {
		t.Fatal("pack lost the installed background, icon centre or end strip")
	}
	countPixels := 0
	for y := 0; y < 90; y++ {
		for x := 0; x < 864; x++ {
			if pic.RGBAAt(x, y) == shopPriceInk {
				countPixels++
				if x < 66 || x > 96 || y < 60 || y >= 84 {
					t.Fatal("quantity left its lower-left corner", x, y)
				}
			}
		}
	}
	if countPixels == 0 {
		t.Fatal("quantity missing")
	}
	pic = renderPackBarArt(s, 0, 9, bar, panelFont(), nil, 0, art)
	if pic.RGBAAt(64, 6) != gold || pic.RGBAAt(96, 38) != gold {
		t.Fatal("held singleton left an item background or picture")
	}
	// A widened original frame repeats only the centre bay. Its two left
	// and two right bays carry distinct shading and retain their positions.
	for bay := 0; bay < 5; bay++ {
		art.Pack.SetRGBA(40+80*bay, 20, color.RGBA{byte(60 + bay), 0, 0, 255})
	}
	pic = renderPackBarArt(InventorySubject{}, 0, 9, bar, panelFont(), nil, -1, art)
	for i, bay := range []int{0, 1, 2, 2, 2, 2, 2, 3, 4} {
		if pic.RGBAAt(72+80*i, 20).R != byte(60+bay) {
			t.Fatal("inventory frame repeated an edge bay", i, bay)
		}
	}
}
