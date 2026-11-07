package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/witness"
)

func TestReleaseSecondGameMissionMagePaletteInputs(t *testing.T) {
	f := stoneStillFront(t)
	if f.Base().Profile.HasFirstData() {
		t.Skip("mission 42 mage measurement reads the second game")
	}
	classes, err := loadUnitClasses(f.Archives.Containers)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpenerWith(42, nil)(); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	tables := releaseOwnerTables(t, f)
	root := filepath.Base(os.Getenv("AGAINROM_ASSETS"))
	output := os.Getenv("AGAINROM_UNIT_PALETTE_OUTPUT")
	measurements := "root\tentity\tmap-unit\ttype\tclass\tPalette\towner\ttable\tframe\tx\ty\tindex\tR\tG\tB\n"
	mages := 0
	for _, entity := range mw.world.Entities() {
		row, ok := classes.ByID(entity.Class)
		if !ok || !strings.Contains(strings.ToLower(row.SpritePath()), "mage") {
			continue
		}
		mages++
		draw := swingDraw(t, mw, entity.ID)
		if row.Palette != 0 || draw.Frame == nil || draw.Stone {
			t.Fatalf("mage %d: Palette=%d frame=%p Stone=%t, want ordinary shared table", entity.ID, row.Palette, draw.Frame, draw.Stone)
		}
		table := mw.ownerShade(entity.Owner)
		if draw.Frame.Palette != tables[table] {
			t.Fatalf("mage %d: drawn palette differs from independently decoded source table %d", entity.ID, table)
		}
		frame := releaseBaseFrame(draw.Art, draw.Frame)
		frameIndex := -1
		for i, source := range draw.Art.Frames {
			if source == frame {
				frameIndex = i
			}
		}
		if frameIndex < 0 {
			t.Fatalf("mage %d: production frame does not share a source frame", entity.ID)
		}
		bounds := image.Rectangle{}
		for i, px := range frame.Pixels {
			if px.Opaque {
				x, y := i%frame.Width, i/frame.Width
				bounds = bounds.Union(image.Rect(x, y, x+1, y+1))
			}
		}
		pixel, distance := -1, frame.Width+frame.Height
		for i, px := range frame.Pixels {
			x, y := i%frame.Width, i/frame.Width
			if !px.Opaque || y >= bounds.Min.Y+max(bounds.Dy()/3, 1) {
				continue
			}
			band := false
			for k := 1; k < len(tables); k++ {
				band = band || tables[k][px.Index] != tables[0][px.Index]
			}
			dx := 2*x - bounds.Min.X - bounds.Max.X
			if d := max(dx, -dx) + 2*(y-bounds.Min.Y); band && d < distance {
				pixel, distance = i, d
			}
		}
		if pixel < 0 {
			t.Fatalf("mage %d: no owner-colour sample in upper third of opaque bounds %v", entity.ID, bounds)
		}
		x, y := pixel%frame.Width, pixel/frame.Width
		index := frame.Pixels[pixel].Index
		rgb := draw.Frame.RGBA().RGBAAt(x, y)
		if rgb != tables[table][index] {
			t.Fatalf("mage %d: raw production blit disagrees with source table entry", entity.ID)
		}
		line := fmt.Sprintf("%s\t%d\t%d\t%d\t%d\t%d\t%d\thuman.pal[%d]\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
			root, entity.ID, entity.MapUnitID, entity.TypeID, entity.Class, row.Palette, entity.Owner, table,
			frameIndex, x, y, index, rgb.R, rgb.G, rgb.B)
		measurements += line
		t.Log(strings.TrimSpace(line))
		if output != "" {
			witness.PNG(t, filepath.Join(output, fmt.Sprintf("%s-mission42-mage-%d.png", root, entity.ID)), draw.Frame.RGBA())
		}
	}
	if mages == 0 {
		t.Fatal("mission 42 has no measured mage")
	}
	if output != "" {
		if err := os.WriteFile(filepath.Join(output, root+"-mission42-mages.tsv"), []byte(measurements), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("%s: measured all %d mage-sheet entities at mission entry; upper-third band samples, RGB before lighting", root, mages)
}
