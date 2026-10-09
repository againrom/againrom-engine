package game

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/ui"
)

// preCreateNameRect is the name field (TEXT-075); detailedSkillColumn is the
// class column above the detailed popup.
var (
	preCreateNameRect   = image.Rect(224, 310, 362, 337)
	detailedSkillColumn = image.Rect(160, 0, 480, 280)
)

// tipRenderDir is where the tip witnesses write their frames, or "" when
// AGAINROM_TIPS_RENDER_DIR is unset; the directory is refused inside the
// install.
func tipRenderDir(t *testing.T, f *FrontEnd) string {
	t.Helper()
	out := os.Getenv("AGAINROM_TIPS_RENDER_DIR")
	if out == "" {
		return ""
	}
	root, err := editorPhysicalDirectory(f.Archives.Root)
	if err != nil || !filepath.IsAbs(out) {
		t.Fatal("absolute install and render roots required", err)
	}
	if rel, err := filepath.Rel(root, filepath.Clean(out)); err == nil && (rel == "." || len(rel) < 2 || rel[:2] != "..") {
		t.Fatal("render output is inside the install")
	}
	dir := filepath.Join(out, filepath.Base(filepath.Clean(f.Archives.Root)))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeTipRender(t *testing.T, dir, name string, pic *image.RGBA) {
	t.Helper()
	if dir == "" {
		return
	}
	file, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, pic); err != nil {
		t.Fatal(err)
	}
}

// tipCycleFrames steps the App on its 20 ms headless clock and returns the
// first two frames whose pixels inside watch, outside skip, differ from the
// frame before, with the step count at which each appeared. The name caret
// and the popup are skipped: they change on their own clocks.
func tipCycleFrames(t *testing.T, app *ui.App, steps int, watch image.Rectangle, skip ...image.Rectangle) ([]*image.RGBA, []int) {
	t.Helper()
	var frames []*image.RGBA
	var at []int
	var last [32]byte
	for i := 1; i <= steps && len(frames) < 2; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		frame, _, err := app.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.New()
		for y := watch.Min.Y; y < watch.Max.Y; y++ {
			for x := watch.Min.X; x < watch.Max.X; x++ {
				skipped := false
				for _, r := range skip {
					skipped = skipped || image.Pt(x, y).In(r)
				}
				if !skipped {
					c := frame.RGBAAt(x, y)
					h.Write([]byte{c.R, c.G, c.B})
				}
			}
		}
		var sum [32]byte
		copy(sum[:], h.Sum(nil))
		if i > 1 && sum != last {
			frames, at = append(frames, frame), append(at, i)
		}
		last = sum
	}
	return frames, at
}

// TestReleaseGeneratorTipsCycleAndStep: on the installed generator the
// pre-create popup shows chrsel1..3 at (232,0)-(640,136), steps only on a
// portrait click then a level click, and the highlight cycle changes the frame
// only after the hover wait; the detailed popup shows the class text, its skill
// cycle runs until the first skill click, which retexts chrgen2 and stops it
// (TOWN-518, TOWN-519, TOWN-522, MENU-137).
func TestReleaseGeneratorTipsCycleAndStep(t *testing.T) {
	f := missionTipFront(t)
	dir := tipRenderDir(t, f)
	setup := f.ChargenSetup()
	// The sparkle has its own timer (DIV-1276); without it every frame change
	// on a still pointer is the cycle's.
	precreate, art := *setup.PreCreate, *setup.PreCreate.Art
	art.Sparkles = nil
	precreate.Art, setup.PreCreate = &art, &precreate
	c := ui.NewChargen(setup)
	app := f.App("tip cycles")
	t.Cleanup(app.StopAudio)
	app.Layout(640, 480)
	if err := app.OpenChargen(c, func(res ui.ChargenResult) (ui.MapOpener, error) { return f.NewGameOpener(10, res), nil }); err != nil {
		t.Fatal(err)
	}
	click := func(p image.Point) {
		t.Helper()
		for _, action := range []string{"press", "release"} {
			if err := app.HeadlessPointer(action, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	prePoint := func(owner string) image.Point {
		t.Helper()
		for y := 479; y >= 0; y-- {
			for x := 0; x < 640; x++ {
				p := image.Pt(x, y)
				if got, ok := ui.PreCreateControlAt(c, p); ok && got == owner && !c.TipPanel().Covers(p) {
					return p
				}
			}
		}
		t.Fatalf("no installed pre-create pixel is owned by %q", owner)
		return image.Point{}
	}
	// A resting point that hovers no cycle target.
	rest := image.Pt(-1, -1)
	for y := 479; y >= 0 && rest.X < 0; y-- {
		for x := 0; x < 640; x++ {
			p := image.Pt(x, y)
			if _, ok := ui.PreCreateControlAt(c, p); !ok && !c.TipPanel().Covers(p) {
				rest = p
				break
			}
		}
	}
	if err := app.HeadlessPointer("hover", rest.X, rest.Y); err != nil {
		t.Fatal(err)
	}
	for step, press := range []string{"choice 3", "difficulty 1", ""} {
		v := c.TipPanel()
		if !v.Showing() || v.Rect != ui.PreCreateTipRect || v.Text != setup.TipSelect[step] || c.TipStep() != step {
			t.Fatalf("pre-create step %d popup %q at %v (step %d)", step, v.Text, v.Rect, c.TipStep())
		}
		if err := app.HeadlessPointer("hover", rest.X, rest.Y); err != nil {
			t.Fatal(err)
		}
		frames, at := tipCycleFrames(t, app, 80, image.Rect(0, 0, 640, 480), ui.PreCreateTipRect, preCreateNameRect)
		if len(frames) < 2 || at[1]-at[0] < 15 {
			t.Fatalf("pre-create step %d cycle frames at steps %v, want two after the 500 ms wait", step, at)
		}
		for i, frame := range frames {
			writeTipRender(t, dir, fmt.Sprintf("precreate-step%d-cycle%c", step+1, 'a'+i), frame)
		}
		t.Logf("pre-create step %d: cycle frames at headless steps %v", step, at)
		if press != "" {
			click(prePoint(press))
		}
	}
	click(prePoint("forward"))
	if c.Stage() != ui.DetailedStage {
		t.Fatal("OK did not open the detailed page")
	}
	v := c.TipPanel()
	if !v.Showing() || v.Rect != ui.ChargenTipRect || (v.Text != setup.TipText && v.Text != setup.TipTextMage) {
		t.Fatalf("detailed enter popup %q at %v", v.Text, v.Rect)
	}
	if err := app.HeadlessPointer("hover", 300, 470); err != nil {
		t.Fatal(err)
	}
	frames, at := tipCycleFrames(t, app, 80, detailedSkillColumn)
	if len(frames) < 2 || at[1]-at[0] < 15 {
		t.Fatalf("detailed skill cycle frames at steps %v, want two", at)
	}
	for i, frame := range frames {
		writeTipRender(t, dir, fmt.Sprintf("detailed-class-tip-skillcycle%c", 'a'+i), frame)
	}

	// The first skill click retexts chrgen2 and stops the skill cycle.
	col := "fighter"
	if v.Text == setup.TipTextMage {
		col = "mag"
	}
	raw, err := f.Archives.Containers.ReadFile("graphics/interface/chrgen/" + col + "/mask.bmp")
	if err != nil {
		t.Fatal(err)
	}
	mask, err := bmp.DecodePaletted(raw)
	if err != nil {
		t.Fatal(err)
	}
	// The column mask is 320 wide and starts at x=160; one press per region
	// code until a skill press lands.
	tried := map[uint8]bool{}
	for y := 0; y < mask.Bounds().Dy() && c.TipStep() == 0; y++ {
		for x := 0; x < mask.Bounds().Dx() && c.TipStep() == 0; x++ {
			code := mask.ColorIndexAt(x, y)
			if p := image.Pt(160+x, y); !tried[code] && !c.TipPanel().Covers(p) {
				tried[code] = true
				click(p)
			}
		}
	}
	if got := c.TipPanel(); got.Text != setup.TipTextDetail || c.TipStep() != 1 {
		t.Fatalf("first skill click: popup %q step %d", got.Text, c.TipStep())
	}
	if err := app.HeadlessPointer("hover", 300, 470); err != nil {
		t.Fatal(err)
	}
	if frames, at := tipCycleFrames(t, app, 60, detailedSkillColumn); len(frames) != 0 {
		t.Fatalf("the skill cycle still ran after the first skill click, frames at %v", at)
	}
	frame, _, err := app.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	writeTipRender(t, dir, "detailed-chrgen2-after-skill-click", frame)

	// The statistic buttons' states (MENU-138): + disabled while the class
	// start leaves no points, - disabled at the floor, + hovered and held.
	writeTipRender(t, dir, "detailed-plus-disabled-no-points", frame)
	for i := 0; i < 40; i++ {
		click(image.Pt(117, 64))
	}
	if err := app.HeadlessPointer("hover", 300, 470); err != nil {
		t.Fatal(err)
	}
	floor, _, _ := app.HeadlessFrame()
	writeTipRender(t, dir, "detailed-minus-disabled-at-floor", floor)
	if err := app.HeadlessPointer("hover", 142, 64); err != nil {
		t.Fatal(err)
	}
	hovered, _, _ := app.HeadlessFrame()
	writeTipRender(t, dir, "detailed-plus-hovered", hovered)
	if err := app.HeadlessPointer("press", 142, 64); err != nil {
		t.Fatal(err)
	}
	held, _, _ := app.HeadlessFrame()
	writeTipRender(t, dir, "detailed-plus-held", held)
	if err := app.HeadlessPointer("release", 142, 64); err != nil {
		t.Fatal(err)
	}
	plus := image.Rect(132, 54, 152, 74)
	if sameTipPixels(floor, hovered, plus) || sameTipPixels(hovered, held, plus) {
		t.Fatal("the + button draws the same picture plain, hovered and held")
	}
	if err := app.HeadlessPointer("hover", 552, 160); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessPointer("press", 552, 160); err != nil {
		t.Fatal(err)
	}
	frame, _, _ = app.HeadlessFrame()
	writeTipRender(t, dir, "detailed-back-pressed", frame)
}

// TestReleaseMissionStartTipRender composes the m10 start tip popup and writes
// it for the owner when AGAINROM_TIPS_RENDER_DIR is set.
func TestReleaseMissionStartTipRender(t *testing.T) {
	f := missionTipFront(t)
	dir := tipRenderDir(t, f)
	app, live := openMissionTipMap(t, f, 10)
	pageMissionDialogue(t, app, live, "start dialogue")
	wantMissionTip(t, f, live, 10, 2, "start trigger")
	// The map frame needs the GPU; the popup composes alone at its place.
	pic := ui.ComposeMissionTip(live.view, 1024, 768)
	if pic.RGBAAt(ui.MissionTipRect.Min.X+40, ui.MissionTipRect.Min.Y+40).A == 0 || pic.RGBAAt(ui.MissionTipRect.Max.X+5, ui.MissionTipRect.Min.Y+40).A != 0 {
		t.Fatal("the mission popup does not compose inside its rectangle")
	}
	writeTipRender(t, dir, "mission10-start-tip-popup", pic)
}

// sameTipPixels reports whether a and b agree on every pixel of r.
func sameTipPixels(a, b *image.RGBA, r image.Rectangle) bool {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				return false
			}
		}
	}
	return true
}
