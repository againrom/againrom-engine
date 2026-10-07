package game

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestReleaseActivePauseRenderedWorld(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("AGAINROM_ASSETS required for installed GPU presentation")
	}
	dir := t.TempDir()
	if out := os.Getenv("AGAINROM_PAUSED_PRESENTATION_OUTPUT"); out != "" {
		var err error
		dir, err = effectRimOutputPath(root, out)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestReleaseActivePauseRenderedWorld$")
	cmd.Env = append(os.Environ(), "AGAINROM_PAUSED_RENDER_CHILD="+dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("installed GPU child: %v\n%s", err, out)
	}
	t.Logf("%s", out)
}

type pausedPresentationProbe struct {
	draw func() error
	done bool
	err  error
}

func (g *pausedPresentationProbe) Layout(_, _ int) (int, int) { return 1024, 768 }
func (g *pausedPresentationProbe) Update() error {
	if g.done {
		return ebiten.Termination
	}
	return nil
}
func (g *pausedPresentationProbe) Draw(_ *ebiten.Image) {
	if !g.done {
		g.err, g.done = g.draw(), true
	}
}

func runPausedPresentationProbe(dir string) error {
	if _, err := effectRimOutputPath(os.Getenv("AGAINROM_ASSETS"), dir); err != nil {
		return err
	}
	f, err := NewFrontEnd(os.Getenv("AGAINROM_ASSETS"))
	if err != nil {
		return err
	}
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{Path: filepath.Join(dir, "options.txt")}
	app := f.App("paused presentation witness")
	defer app.StopAudio()
	app.SetCutscenes(nil)
	party := f.ChargenParty(ui.ChargenResult{Name: "Pause", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 20, 20}})
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		return err
	}
	app.Layout(1024, 768)
	for n := 0; app.HeadlessNoticeOpen() && n < 32; n++ {
		if err := app.HeadlessKey("enter"); err != nil {
			return err
		}
	}
	if app.HeadlessNoticeOpen() {
		return fmt.Errorf("installed mission notice remains open")
	}
	v := f.live.view
	v.SetTextSmoothing(false)
	v.ShowReadout(false)
	v.DeferPointer(true)
	id := f.live.mission.ids[0]
	e, ok := f.live.world.Entity(id)
	if !ok || e.HP <= 7 {
		return fmt.Errorf("installed damage control lacks a live hero")
	}
	for i := range f.live.fog.visible {
		f.live.fog.visible[i], f.live.fog.explored[i] = 1, 1
	}
	f.live.push()
	v.Camera().CenterOn(float64(e.X*32), float64(e.Y*32))
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		return err
	}
	for range 12 {
		var err error
		if app.HeadlessNoticeOpen() {
			err = app.HeadlessKey("enter")
		} else {
			err = app.HeadlessStep()
		}
		if err != nil {
			return err
		}
	}
	for n := 0; app.HeadlessNoticeOpen() && n < 32; n++ {
		if err := app.HeadlessKey("enter"); err != nil {
			return err
		}
	}
	if app.HeadlessNoticeOpen() {
		return fmt.Errorf("installed damage producer remains under a notice")
	}
	f.LiveDamage(uint32(id), 7)
	wounded, _ := f.live.world.Entity(id)
	if err := app.HeadlessStep(); err != nil {
		return err
	}
	if _, n := v.DamageNumerals(); n != 1 {
		after, _ := f.live.world.Entity(id)
		return fmt.Errorf("App/sim damage producer yielded %d numerals, want1; health%d/%d→%d/%d, pause%v notice%v tick%d", n, wounded.HP, wounded.MaxHP, after.HP, after.MaxHP, v.SaveApplication().PlayerPaused, app.HeadlessNoticeOpen(), f.live.world.Tick())
	}
	g := &pausedPresentationProbe{}
	g.draw = func() error {
		tex := ebiten.NewImage(1024, 768)
		defer tex.Dispose()
		capture := func(name string) (*image.RGBA, error) {
			v.Draw(tex)
			pic := image.NewRGBA(image.Rect(0, 0, 1024, 768))
			tex.ReadPixels(pic.Pix)
			file, err := os.Create(filepath.Join(dir, name+".png"))
			if err != nil {
				return nil, err
			}
			err = png.Encode(file, pic)
			closed := file.Close()
			if err != nil {
				return nil, err
			}
			return pic, closed
		}
		bright, err := capture("running")
		if err != nil {
			return err
		}
		count := v.AnimationCounter()
		world, err := f.live.world.MarshalBinary()
		if err != nil {
			return err
		}
		if err := app.HeadlessKey("0"); err != nil {
			return err
		}
		paused, err := capture("paused")
		if err != nil {
			return err
		}
		viewport := v.ViewportSize()
		worldRect := image.Rect(0, 48, viewport.X, viewport.Y)
		dimmed, exactDim, hudSame := 0, 0, 0
		for y := 0; y < 768; y++ {
			for x := 0; x < 1024; x++ {
				a, b := bright.RGBAAt(x, y), paused.RGBAAt(x, y)
				if image.Pt(x, y).In(worldRect) {
					if a != b && b.R <= a.R && b.G <= a.G && b.B <= a.B {
						dimmed++
					}
					near := func(src, dst uint8) bool { delta := int(dst) - int(src)*223/255; return delta >= -1 && delta <= 1 }
					if a != b && near(a.R, b.R) && near(a.G, b.G) && near(a.B, b.B) {
						exactDim++
					}
				} else if x >= viewport.X || y >= viewport.Y {
					if a != b {
						return fmt.Errorf("world pause tinted HUD at%d,%d: %v/%v", x, y, a, b)
					}
					hudSame++
				}
			}
		}
		if exactDim < 50000 || dimmed < 50000 || hudSame < 50000 {
			return fmt.Errorf("GPU negative control lacks visible world/HUD: dim%d exact%d HUD%d", dimmed, exactDim, hudSame)
		}
		for range 150 {
			if err := app.HeadlessStep(); err != nil {
				return err
			}
		}
		held, err := capture("held-three-seconds")
		if err != nil {
			return err
		}
		for y := worldRect.Min.Y; y < worldRect.Max.Y; y++ {
			for x := worldRect.Min.X; x < worldRect.Max.X; x++ {
				if held.RGBAAt(x, y) != paused.RGBAAt(x, y) {
					return fmt.Errorf("paused world/numeral GPU pixel moved at%d,%d", x, y)
				}
			}
		}
		if _, n := v.DamageNumerals(); n != 1 || v.AnimationCounter() != count {
			return fmt.Errorf("paused animation/numeral count changed: %d/%d numerals%d", v.AnimationCounter(), count, n)
		}
		x, y, err := app.HeadlessGroundPoint()
		if err != nil {
			return err
		}
		for _, edge := range []string{"press", "release"} {
			if err := app.HeadlessPointer(edge, x, y); err != nil {
				return err
			}
		}
		if len(f.live.pending) == 0 || !slices.Contains(app.HeadlessSelection(), uint32(id)) {
			return fmt.Errorf("paused UI lost selection or command admission")
		}
		after, err := f.live.world.MarshalBinary()
		if err != nil || !bytes.Equal(world, after) {
			return fmt.Errorf("paused presentation changed World: %v", err)
		}
		snap, _, err := f.Snapshot(true)
		if err != nil {
			return err
		}
		saved, err := f.ExportCurrentSave(snap, "Paused rendered world")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "paused.sav"), saved, 0600); err != nil {
			return err
		}
		queue := slices.Clone(f.live.pending)
		if err := app.HeadlessKey("0"); err != nil {
			return err
		}
		if v.SaveApplication().PlayerPaused || v.AnimationCounter() != count {
			return fmt.Errorf("first resume frame retained dim or advanced animation")
		}
		if _, err := capture("resumed"); err != nil {
			return err
		}
		for range 4 {
			if err := app.HeadlessStep(); err != nil {
				return err
			}
		}
		if v.AnimationCounter() <= count {
			return fmt.Errorf("resumed animation did not continue")
		}
		proof := struct {
			WorldSHA256, SAVSHA256                          string
			AnimationBefore, AnimationAfter                 uint32
			DimmedPixels, ExactDimPixels, UntintedHUDPixels int
			Queue                                           []sim.Command
		}{fmt.Sprintf("%x", sha256.Sum256(world)), fmt.Sprintf("%x", sha256.Sum256(saved)), count, v.AnimationCounter(), dimmed, exactDim, hudSame, queue}
		raw, err := json.MarshalIndent(proof, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "proof.json"), raw, 0600); err != nil {
			return err
		}
		fmt.Printf("installed mission20 App/sim/Viewer GPU: dim%d exact%d HUD%d; 150 paused frames, numeral retained; count%d→%d; queue%d; World%s SAV%s\n", dimmed, exactDim, hudSame, count, v.AnimationCounter(), len(queue), proof.WorldSHA256, proof.SAVSHA256)
		return probeInstalledScenery(f, dir)
	}
	ebiten.SetWindowSize(64, 48)
	ebiten.SetWindowPosition(-32000, -32000)
	ebiten.SetRunnableOnUnfocused(true)
	if err := ebiten.RunGameWithOptions(g, &ebiten.RunGameOptions{InitUnfocused: true, SkipTaskbar: true}); err != nil && err != ebiten.Termination {
		return err
	}
	return g.err
}

func probeInstalledScenery(f *FrontEnd, dir string) error {
	addr, _ := MissionMap(20)
	raw, err := f.Archives.Containers.ReadFile(addr)
	if err != nil {
		return err
	}
	mv, err := LoadMapViewer(f.Tiles, raw, addr, Markers{}, StaticLayer{Set: f.Statics, Art: true, AnimGate: terrain.AnimGateAll}, StructureLayer{Set: f.Structures, Art: true})
	if err != nil {
		return err
	}
	v := mv.Viewer
	app := f.App("installed scenery clock")
	if err := app.OpenMission(func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
		return v, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil
	}); err != nil {
		return err
	}
	app.Layout(1024, 768)
	v.SetFlat(true)
	v.DeferPointer(true)
	m := mv.Map
	grid := terrain.Grid{Width: m.Width, Height: m.Height, Tiles: terrain.RenderTileWords(m.Tiles), Altitudes: m.Altitudes, Overlay: m.Overlay, Structures: StructureRecords(m.Objects)}
	statics, _, animated := terrain.StaticPlacements(grid, f.Statics, nil, 0, terrain.AnimGateAll)
	centers := map[string]image.Point{}
	for _, i := range animated {
		if len(statics[i].Class.Timeline) > 1 {
			centers["foliage"] = statics[i].Cell
			break
		}
	}
	structures, _, cycleStructures := terrain.StructurePlacements(grid, f.Structures, nil, 0)
	for _, i := range cycleStructures {
		p := structures[i]
		name := "building"
		if strings.Contains(strings.ToLower(p.Class.Name), "well") || p.Class.ID == 14 || p.Class.ID == 15 {
			name = "well"
		}
		if _, exists := centers[name]; !exists {
			centers[name] = p.Cell
		}
	}
	for i, word := range grid.Tiles {
		if terrain.Resolve(word).Water {
			centers["water"] = image.Pt(i%grid.Width, i/grid.Width)
			break
		}
	}
	tex := ebiten.NewImage(1024, 768)
	defer tex.Dispose()
	capture := func() []byte {
		v.Draw(tex)
		pixels := make([]byte, 1024*768*4)
		tex.ReadPixels(pixels)
		return pixels
	}
	for _, name := range []string{"foliage", "well", "building", "water"} {
		cell, ok := centers[name]
		if !ok {
			return fmt.Errorf("installed mission20 lacks animation witness %s", name)
		}
		v.Camera().CenterOn(float64(cell.X*32), float64(cell.Y*32))
		initial := capture()
		var running []byte
		for sample := 0; sample < 32; sample++ {
			for range 4 {
				if err := app.HeadlessStep(); err != nil {
					return err
				}
			}
			running = capture()
			if !bytes.Equal(initial, running) {
				break
			}
		}
		if bytes.Equal(initial, running) {
			return fmt.Errorf("installed %s animation has no moving-pixel control at%v", name, cell)
		}
		count := v.AnimationCounter()
		if err := app.HeadlessKey("0"); err != nil {
			return err
		}
		paused := capture()
		for range 150 {
			if err := app.HeadlessStep(); err != nil {
				return err
			}
		}
		if !bytes.Equal(paused, capture()) || v.AnimationCounter() != count {
			return fmt.Errorf("installed %s advanced during pause", name)
		}
		if err := app.HeadlessKey("0"); err != nil {
			return err
		}
		resumed := capture()
		if !bytes.Equal(running, resumed) || v.AnimationCounter() != count {
			return fmt.Errorf("installed %s did not remove dim and restore exact retained frame", name)
		}
		pic := &image.RGBA{Pix: paused, Stride: 1024 * 4, Rect: image.Rect(0, 0, 1024, 768)}
		file, err := os.Create(filepath.Join(dir, name+"-paused.png"))
		if err != nil {
			return err
		}
		encoded, closed := png.Encode(file, pic), file.Close()
		if encoded != nil {
			return encoded
		}
		if closed != nil {
			return closed
		}
		fmt.Printf("installed %s cell%v: moving GPU control, 150 paused frames exact, resume pixels exact at retained count%d\n", name, cell, count)
	}
	return nil
}
