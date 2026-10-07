package game

import (
	"image"
	"os"
	"testing"

	"againrom/pkg/ui"
)

// TestReleaseSchoolMaskNamesTheSkillItIsDrawnOver is 1015's install-gated
// witness for schoolMaskSlot.
//
// WHY IT EXISTS. The first half is right and the second half is the wrong
// reading of the rule: the rule's operative clause is that `go test ./...`
// runs green with no game present, and this package already carries thirteen
// other tests that skip on an empty AGAINROM_ASSETS and read the install
// when it is named, ten of them named TestRelease*. `go test -v ./...` with
// no root prints the whole population as --- SKIP lines: sixteen, fourteen
// here and two in cmd/missionrun. cmd/schoolcheck stays the story's
// measuring instrument; this is the regression witness that runs inside the
// ordinary chain whenever a root is named.
//
// WHAT IT WITNESSES. Before 1015 the table answered the original's stored-slot
// order while the rectangles were drawn in detailed chargen order, so a click on
// the third icon of either class selected the fourth skill. Restoring that table
// leaves every other School test in pkg/ui and pkg/game green, because the
// defective table is also a bijection and no synthetic fixture can carry the
// shipped mask's own colour geometry. This test reddens on it: it presses the
// centre of each production rectangle over the production mask and requires the
// cell the hit test answers to be the cell that rectangle draws.
func TestReleaseSchoolMaskNamesTheSkillItIsDrawnOver(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("no AGAINROM_ASSETS: the school mask witness needs a lawful install")
	}
	f := releaseFront(t)
	if f.TownSchoolArt.Value() == nil || f.TownSchoolArt.Value().Masks[0] == nil || f.TownSchoolArt.Value().Masks[1] == nil {
		t.Fatalf("production school art did not resolve: %v", f.TownSchoolArt.Err())
	}
	cells := make([]ui.TownSurfaceCell, 10)
	for i := range cells {
		cells[i].Enabled = true
	}
	for class := 0; class < 2; class++ {
		v := ui.TownSurfaceView{
			Kind:        ui.TownSurfaceSchool,
			Cells:       cells,
			SchoolArt:   f.TownSchoolArt.Value(),
			SchoolClass: class,
		}
		for slot := 0; slot < 5; slot++ {
			r := ui.SchoolSkillRect(class, slot)
			if r.Empty() {
				t.Fatalf("class %d slot %d has no production rectangle", class, slot)
			}
			p := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
			c, ok := ui.TownSurfaceControlAt(v, p)
			if !ok || c.Kind != ui.TownSurfaceControlCell {
				t.Fatalf("class %d slot %d: the centre %v of its own drawn rect %v answers no cell",
					class, slot, p, r)
			}
			if want := class*5 + slot; c.Index != want {
				t.Fatalf("class %d slot %d: a click at %v, the centre of the rect that slot is DRAWN at, selects cell %d, want %d",
					class, slot, p, c.Index, want)
			}
		}
	}
}
