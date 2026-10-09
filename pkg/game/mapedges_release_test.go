package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

func TestReleaseMission90CastleMeetsViewTop(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("map edges")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(90)); err != nil {
		t.Fatal(err)
	}
	if f.live == nil || f.live.view == nil || f.live.mission == nil {
		t.Fatal("mission 90 opened without its production viewer")
	}
	m := f.live.mission.state.Map
	castleAt := -1
	for i, o := range m.Objects {
		if col, row := terrain.AnchorCell(o.X, o.Y); col == 104 && row == 9 && o.Kind&0xff == 57 {
			castleAt = i
			break
		}
	}
	if castleAt < 0 {
		t.Fatal("mission 90 has no Castle at (104,9)")
	}
	class := f.Structures.Classes[57]
	if class == nil || class.ID != 57 || class.TileWidth != 11 || class.TileHeight != 5 || class.FullHeight != 5 || len(class.Frames) < 55 {
		t.Fatalf("installed Castle class = %+v, want 11x5 strips", class)
	}
	corners := [4]uint8{m.Altitudes[11*m.Width+109], m.Altitudes[11*m.Width+110], m.Altitudes[12*m.Width+109], m.Altitudes[12*m.Width+110]}
	if corners != ([4]uint8{28, 28, 28, 44}) {
		t.Fatalf("footprint-centre corners = %v, want (28,28,28,44)", corners)
	}
	g := terrain.Grid{Width: m.Width, Height: m.Height, Tiles: terrain.RenderTileWords(m.Tiles),
		Altitudes: m.Altitudes, Structures: StructureRecords(m.Objects[castleAt : castleAt+1])}
	projection := terrain.Project(g.Altitudes, g.Width, g.Height)
	places, _, _ := terrain.StructurePlacements(g, f.Structures, projection.Altitude, projection.MinV)
	if len(places) != 55 {
		t.Fatalf("Castle contributed %d strips, want 55", len(places))
	}
	top := places[0].TopLeft.Y
	for _, p := range places {
		top = min(top, p.TopLeft.Y)
		if lift := p.Cell.Y*32 - p.TopLeft.Y - projection.MinV; lift != 32 {
			t.Fatalf("Castle strip %v shared lift = %d, want 32", p.Cell, lift)
		}
	}
	if top+projection.MinV != 256 {
		t.Fatalf("Castle first strip native Y = %d, want 256", top+projection.MinV)
	}
	v := f.live.view
	cam := v.Camera()
	cam.X, cam.Y = float64((104+115)*32-cam.ViewW)/2, -1e9
	cam.Clamp()
	if cam.Y+float64(projection.MinV) != 256 || float64(top)-cam.Y != 0 {
		t.Fatalf("highest native camera Y = %.3f, Castle first strip Y = %.3f; want 256 and screen 0 with no ground row above", cam.Y+float64(projection.MinV), float64(top)-cam.Y)
	}
	if cam.Zoom != 1 || v.Mode() != ui.ModeDisplaced {
		t.Fatalf("default mission camera zoom %.3f mode %v, want native displaced view", cam.Zoom, v.Mode())
	}
	plane := make([]byte, m.Width*m.Height)
	for i := range plane {
		plane[i] = ui.FogVisible
	}
	v.SetFog(plane, m.Width, m.Height)
	v.SetFogReveal(false)
	draws, err := v.HeadlessArtDraws()
	if err != nil {
		t.Fatal(err)
	}
	mask, err := v.HeadlessShroudMask()
	if err != nil {
		t.Fatal(err)
	}
	art, receipts := effectRimArt(t, v, ui.MapEntity{})
	final := image.NewRGBA(art.Bounds())
	copy(final.Pix, art.Pix)
	draw.Draw(final, final.Bounds(), mask, mask.Bounds().Min, draw.Over)
	topFrames := make(map[*terrain.StaticFrame]bool)
	for _, frame := range class.Frames[:11] {
		topFrames[frame] = true
	}
	checked, submitted := 0, 0
	for _, d := range draws {
		if d.Kind != "sprite" || !topFrames[d.Frame] {
			continue
		}
		_, sy := d.Geometry.Apply(0, 0)
		if sy != 0 {
			continue
		}
		submitted++
		for sx := d.Pixels.Rect.Min.X; sx < d.Pixels.Rect.Max.X; sx++ {
			if d.Pixels.RGBAAt(sx, d.Pixels.Rect.Min.Y).A != 255 {
				continue
			}
			x, y := d.Geometry.Apply(float64(sx)+0.5, float64(d.Pixels.Rect.Min.Y)+0.5)
			p := image.Pt(int(math.Floor(x)), int(math.Floor(y)))
			if !p.In(final.Bounds()) || p.Y != 0 {
				t.Fatalf("Castle's first opaque art row lands at %v", p)
			}
			if got := mask.RGBAAt(p.X, p.Y); got.A != 0 {
				t.Fatalf("shroud clips the Castle at screen %v: mask %v", p, got)
			}
			if got, want := final.RGBAAt(p.X, p.Y), art.RGBAAt(p.X, p.Y); got != want || want.A != 255 {
				t.Fatalf("Castle top pixel %v final = %v, production art = %v", p, got, want)
			}
			checked++
		}
	}
	if submitted != 11 || checked == 0 {
		t.Fatalf("production Castle first row: %d submitted strips and %d opaque pixels, want 11 strips and a nonempty row at y0", submitted, checked)
	}
	t.Logf("mission 90 Castle: centre corners %v lift 32; camera (%.0f,%.0f), first art row 0, ground rows above 0; %d top strips, %d opaque top pixels survive non-revealed all-visible shroud", corners, cam.X, cam.Y, submitted, checked)
	if output := os.Getenv("AGAINROM_MAP_EDGES_WITNESS_DIR"); output != "" {
		root, err := filepath.Abs(f.Archives.Root)
		if err != nil {
			t.Fatal(err)
		}
		dir, err := effectRimOutputPath(root, output)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		writeStatusBarFrame(t, filepath.Join(dir, "castle-art.png"), art)
		writeStatusBarFrame(t, filepath.Join(dir, "castle-shroud.png"), mask)
		writeStatusBarFrame(t, filepath.Join(dir, "castle-final.png"), final)
		proof := map[string]any{"mission": 90, "corners": corners, "shared_lift": 32, "camera": [2]float64{cam.X, cam.Y},
			"projection_min_v": projection.MinV, "native_camera_y": cam.Y + float64(projection.MinV),
			"first_art_row": 0, "ground_rows_above": 0, "top_strips": submitted, "opaque_top_pixels": checked,
			"fog_revealed": v.FogRevealed(), "world_hash": fmt.Sprintf("%x", f.live.world.Hash()),
			"draws": receipts, "art_sha256": fmt.Sprintf("%x", sha256.Sum256(art.Pix)), "mask_sha256": fmt.Sprintf("%x", sha256.Sum256(mask.Pix)),
			"final_sha256": fmt.Sprintf("%x", sha256.Sum256(final.Pix)),
			"raster_limit": "CPU raster of production art submissions and shroud fill/triangles; excludes terrain, HUD and GPU readback"}
		raw, err := json.MarshalIndent(proof, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "castle-receipts.json"), raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
}
