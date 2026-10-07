package ui

import (
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
)

// TestReleaseCommandPanelComposesFromRealArtInFourStates is docs/1028-
// command-panel's own W-1 witness, committed at the return from adversarial
// pass 1. No committed test called composeCommandPanel end to end before
// this one: pkg/game's TestReleaseCommandPanelArtAndLabelsComposeFromTheRealInstall
// checks LoadCommandPanelArt's output and the label text, never the draw. The
// pass 1 reviewer proved production correct on both roots with a temporary
// test it deleted before finishing; this is that proof, durable, per
// AGENTS.md coverage rule 7 (a composed frame compared against an
// independently built expected frame, not a per-control assertion).
//
// This file lives in package ui, not beside the other release tests in
// pkg/game, because composeCommandPanel and Viewer's own arming methods are
// unexported: an internal pkg/ui test importing pkg/game would import a
// package that itself imports pkg/ui, an import cycle. It loads the panel's
// four shipped bitmaps directly through pkg/vfs and pkg/formats/bmp, the same
// two packages pkg/game's LoadCommandPanelArt (commandpanelart.go) uses, at
// the same four addresses that file names — duplicated here as literals
// because those constants are unexported in a different package.
//
// FOUR STATES, ALL FOUR SHIPPED BITMAPS EXERCISED: inactive (Heads alone),
// active with no arm (Active plus the three disabled-mask cells from
// Disabled), Attack-armed (adds cell 0 from Selected), and Move-armed (adds
// cell 1 from Selected instead of cell 0) — the state this story adds and the
// one the missing readout case (this same return visit) affected.
func TestReleaseCommandPanelComposesFromRealArtInFourStates(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the command panel composition witness needs a lawful install")
	}
	art := releaseCommandPanelArt(t, root)
	headsRGBA := releaseImageToRGBA(art.Heads)
	headsSeamRGBA := releaseImageToRGBA(art.HeadsSeam)
	activeRGBA := releaseImageToRGBA(art.Active)
	activeSeamRGBA := releaseImageToRGBA(art.ActiveSeam)
	disabledRGBA := releaseImageToRGBA(art.Disabled)
	selectedRGBA := releaseImageToRGBA(art.Selected)

	newPanelViewer := func(t *testing.T) *Viewer {
		t.Helper()
		v, err := NewViewer("release-command-panel", terrain.Grid{
			Width: 32, Height: 32, Tiles: make([]uint16, 32*32),
		}, &terrain.Tileset{})
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		v.Layout(MissionFrameW, MissionFrameH)
		v.SetCommandPanelArt(art)
		return v
	}

	// bar0 is the panel's own size (MissionPanelW x hudToggleBarH, the same
	// 160x80 the four shipped bitmaps carry) at a zero origin, so
	// commandCellRects gives cell rectangles in the same local coordinate
	// space composeCommandPanel's own returned image uses, independent of
	// where the real bar stands in the 1024x768 frame.
	bar0 := image.Rect(0, 0, MissionPanelW, hudToggleBarH)
	cellRects := commandCellRects(bar0)
	withSeam := func(body, seam *image.RGBA) *image.RGBA {
		out := image.NewRGBA(image.Rect(0, 0, seam.Bounds().Dx()+body.Bounds().Dx(), body.Bounds().Dy()))
		draw.Draw(out, image.Rect(0, 0, seam.Bounds().Dx(), seam.Bounds().Dy()), seam, seam.Bounds().Min, draw.Over)
		draw.Draw(out, image.Rect(seam.Bounds().Dx(), 0, out.Bounds().Dx(), out.Bounds().Dy()), body, body.Bounds().Min, draw.Src)
		return out
	}

	expectActiveBaseline := func() *image.RGBA {
		out := image.NewRGBA(bar0)
		draw2(out, bar0, activeRGBA)
		for i, r := range cellRects {
			if commandPanelCellSkipped(commandPanelCell(i)) {
				draw2(out, r, disabledRGBA)
			}
		}
		return out
	}

	t.Run("inactive shows Heads alone", func(t *testing.T) {
		v := newPanelViewer(t)
		bar, ok := v.commandPanelBar()
		if !ok {
			t.Fatal("no command panel bar at the mission frame size")
		}
		got := composeCommandPanel(v, bar)
		if !imagesEqual(got, withSeam(headsRGBA, headsSeamRGBA)) {
			t.Error("an inactive panel (no selection) does not match the shipped Heads body and closing seam")
		}
	})

	t.Run("active with no arm shows Active plus the three disabled cells", func(t *testing.T) {
		v := newPanelViewer(t)
		v.SetEntities([]MapEntity{panelEntity(1, "Warrior", 63, 100, 4, 4)})
		v.sel = selection{1}
		bar, ok := v.commandPanelBar()
		if !ok {
			t.Fatal("no command panel bar at the mission frame size")
		}
		got := composeCommandPanel(v, bar)
		want := withSeam(expectActiveBaseline(), activeSeamRGBA)
		if !imagesEqual(got, want) {
			t.Error("an active, unarmed panel does not match Active with the three disabled cells masked from Disabled")
		}
	})

	t.Run("Attack-armed shows cell 0 from Selected", func(t *testing.T) {
		v := newPanelViewer(t)
		v.SetEntities([]MapEntity{panelEntity(1, "Warrior", 63, 100, 4, 4)})
		v.sel = selection{1}
		v.armAttack()
		if !v.armed {
			t.Fatal("setup: armAttack did not arm")
		}
		bar, ok := v.commandPanelBar()
		if !ok {
			t.Fatal("no command panel bar at the mission frame size")
		}
		got := composeCommandPanel(v, bar)
		bodyWant := expectActiveBaseline()
		draw2(bodyWant, cellRects[commandCellAttack], selectedRGBA)
		want := withSeam(bodyWant, activeSeamRGBA)
		if !imagesEqual(got, want) {
			t.Error("an Attack-armed panel does not show cell 0 from the shipped Selected bitmap")
		}
	})

	t.Run("Move-armed shows cell 1 from Selected", func(t *testing.T) {
		v := newPanelViewer(t)
		v.SetEntities([]MapEntity{panelEntity(1, "Warrior", 63, 100, 4, 4)})
		v.sel = selection{1}
		v.armCommand(commandMove)
		if v.aimed != commandMove {
			t.Fatal("setup: armCommand(commandMove) did not arm Move")
		}
		bar, ok := v.commandPanelBar()
		if !ok {
			t.Fatal("no command panel bar at the mission frame size")
		}
		got := composeCommandPanel(v, bar)
		bodyWant := expectActiveBaseline()
		draw2(bodyWant, cellRects[commandCellMove], selectedRGBA)
		want := withSeam(bodyWant, activeSeamRGBA)
		if !imagesEqual(got, want) {
			t.Error("a Move-armed panel does not show cell 1 from the shipped Selected bitmap")
		}
	})
}

// releaseCommandPanelArt reads the panel's four shipped bitmaps off root, the
// same addresses pkg/game's LoadCommandPanelArt (commandpanelart.go) reads
// out of the merged archive filesystem OpenArchives builds.
func releaseCommandPanelArt(t *testing.T, root string) *CommandPanelArt {
	t.Helper()
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
	read := func(addr string) image.Image {
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
	return &CommandPanelArt{
		Heads:      read("graphics/interface/headsr.bmp"),
		HeadsSeam:  releaseKeyBlack(read("graphics/interface/headsl.bmp")),
		Active:     read("graphics/interface/commandbarr.bmp"),
		ActiveSeam: releaseKeyBlack(read("graphics/interface/commandbarl.bmp")),
		Disabled:   read("graphics/interface/commandempr.bmp"),
		Selected:   read("graphics/interface/commanddnr.bmp"),
	}
}

func releaseKeyBlack(src image.Image) image.Image {
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := color.RGBAModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.RGBA)
			if c.R == 0 && c.G == 0 && c.B == 0 {
				c.A = 0
			}
			out.SetRGBA(x, y, c)
		}
	}
	return out
}

// bmpToRGBA is pkg/game's portraitRGBA (portrait.go), restated here for the
// same import-cycle reason releaseCommandPanelArt duplicates the address
// constants: an internal pkg/ui test cannot import pkg/game.
func bmpToRGBA(im *bmp.Image) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, im.Width, im.Height))
	for i, c := range im.Pix {
		o := i * 4
		pic.Pix[o], pic.Pix[o+1], pic.Pix[o+2], pic.Pix[o+3] = c.R, c.G, c.B, 0xff
	}
	return pic
}

// releaseImageToRGBA is a no-op when the source is already *image.RGBA
// (bmpToRGBA's own return type) and converts otherwise.
func releaseImageToRGBA(im image.Image) *image.RGBA {
	if rgba, ok := im.(*image.RGBA); ok {
		return rgba
	}
	b := im.Bounds()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.Set(x, y, im.At(x, y))
		}
	}
	return out
}

// draw2 copies src's own pixels at r's own coordinates into dst at r —
// composeCommandPanel's own draw.Draw(img, local, art.X, local.Min, draw.Src)
// restated for the expected-frame builder, sampling src at the SAME
// coordinates as the destination rectangle rather than at its origin, which
// is what makes one shipped bitmap's disabled-cell or selected-cell region
// line up with the panel's own cell grid.
func draw2(dst *image.RGBA, r image.Rectangle, src *image.RGBA) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			dst.SetRGBA(x, y, src.RGBAAt(x, y))
		}
	}
}
