package game

import (
	"bytes"
	"image"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// TestReleaseWideInterfaceUsesInstalledTownAndMissionRoutes is run once for
// each preserved language root. It crosses the production front-end builders;
// fixtures cannot stand in for the menu art or mission decode it measures.
func TestReleaseWideInterfaceUsesInstalledTownAndMissionRoutes(t *testing.T) {
	f := releaseFront(t)
	app := f.App("1072-wide-release")
	app.SetSaveSeams(nil,
		func() []ui.SaveEntry {
			return []ui.SaveEntry{{Name: "1072-town", Label: "1072 town witness"}}
		},
		func(name string) (ui.MapOpener, bool, error) { return nil, name == "1072-town", nil },
	)
	app.Layout(640, 480)
	nativeMenu, note, err := app.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatalf("native installed menu: bounds unavailable: note=%q err=%v", note, err)
	}
	app.Layout(1920, 1080)
	menu, note, err := app.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatalf("wide installed menu: bounds unavailable: note=%q err=%v", note, err)
	}
	if got, want := menu.Bounds(), image.Rect(0, 0, 640, 480); got != want {
		t.Fatalf("wide-window installed menu bounds = %v, want %v", got, want)
	}
	if !bytes.Equal(menu.Pix, nativeMenu.Pix) {
		t.Fatal("wide-window Layout stretched or otherwise changed the installed main menu")
	}

	// Cross the same load-to-town and square-door input routes the live app
	// uses, then measure an installed tavern before and after the wide Layout.
	if err := app.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("1072 town witness"); err != nil {
		t.Fatal(err)
	}
	rows := app.HeadlessRows()
	if len(rows) == 0 {
		t.Fatal("installed town square has no tavern row")
	}
	if err := app.HeadlessActivate(rows[0].Text); err != nil {
		t.Fatalf("enter installed tavern: %v", err)
	}
	app.Layout(640, 480)
	nativeTown, _, err := app.HeadlessFrame()
	if err != nil {
		t.Fatalf("native installed tavern: %v", err)
	}
	app.Layout(1920, 1080)
	wideTown, _, err := app.HeadlessFrame()
	if err != nil {
		t.Fatalf("wide installed tavern: %v", err)
	}
	if got, want := wideTown.Bounds(), image.Rect(0, 0, 640, 480); got != want {
		t.Fatalf("wide-window installed tavern bounds = %v, want %v", got, want)
	}
	if !bytes.Equal(wideTown.Pix, nativeTown.Pix) {
		t.Fatal("wide-window Layout stretched or otherwise changed the installed town")
	}

	v, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(10)()
	if err != nil {
		t.Fatalf("open installed mission 10: %v", err)
	}
	beforeHash := f.live.world.Hash()
	v.Layout(ui.MissionFrameW, ui.MissionFrameH)
	baseZoom := v.Camera().Zoom
	baseViewport := v.ViewportSize()
	v.Layout(1920, 1080)
	if got, want := v.FrameSize(), image.Pt(1366, 768); got != want {
		t.Fatalf("wide mission frame = %v, want %v", got, want)
	}
	if got, want := v.ViewportSize(), image.Pt(1206, 593); got != want {
		t.Fatalf("wide mission viewport = %v, want %v", got, want)
	}
	if baseViewport != image.Pt(864, 593) || v.Camera().Zoom != baseZoom {
		t.Fatalf("wide mission changed base presentation: base viewport=%v zoom=%v, wide zoom=%v",
			baseViewport, baseZoom, v.Camera().Zoom)
	}

	// TERR-216.
	cam := v.Camera()
	cam.Pan(-1e9, -1e9)
	if cam.X <= 0 || cam.Y <= 0 {
		t.Fatalf("wide mission near edge reached black world margin: camera=(%v,%v)", cam.X, cam.Y)
	}
	cam.Pan(1e9, 1e9)
	rightSlack := cam.WorldW() - (cam.X + float64(cam.ViewW)/cam.Zoom)
	bottomSlack := cam.WorldH() - (cam.Y + float64(cam.ViewH)/cam.Zoom)
	if rightSlack <= 0 || bottomSlack <= 0 {
		t.Fatalf("wide mission far edge reached black world margin: right=%v bottom=%v",
			rightSlack, bottomSlack)
	}
	m := f.live.mission.state.Map
	wantY := float64((m.Height-8-int(float64(cam.ViewH)/(cam.Zoom*32)))*32 - terrain.Project(m.Altitudes, m.Width, m.Height).MinV)
	if cam.Y != wantY {
		t.Fatalf("wide mission far camera Y %v, want %v", cam.Y, wantY)
	}
	if got := f.live.world.Hash(); got != beforeHash {
		t.Fatalf("Layout changed world hash from %#x to %#x", beforeHash, got)
	}
}
