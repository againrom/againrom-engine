package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"reflect"
	"testing"
	"time"
)

func TestCutsceneLibraryWaitsForFirstFrame1184(t *testing.T) {
	for _, deliver := range []bool{false, true} {
		t.Run(fmt.Sprintf("deliver=%v", deliver), func(t *testing.T) {
			a := newTestApp(t, appRows(1), okLoader(t))
			catalog := []CutsceneEntry{{"m20", "Victory"}}
			var persisted []string
			a.SetCutsceneLibrary(catalog, nil, func(key string) error {
				persisted = append(persisted, key)
				return nil
			}, CutsceneLibraryWords{})
			r, w := io.Pipe()
			defer w.Close()
			a.SetCutscenes(&testCutsceneSource{stream: r})
			a.PlayCutscene("m20/01.smk")
			defer a.StopCutscene()
			a.step(appInput{}, time.Now())
			if a.CutsceneFrameNumber() != 0 || len(a.seenCutscenes()) != 0 || len(persisted) != 0 {
				t.Fatal("waiting decoder unlocked the movie", persisted)
			}
			if deliver {
				go func() { _, _ = w.Write(syntheticMovie()); _ = w.Close() }()
				deadline := time.Now().Add(time.Second)
				for a.CutsceneFrameNumber() == 0 && time.Now().Before(deadline) {
					a.step(appInput{}, time.Now())
					time.Sleep(time.Millisecond)
				}
				if a.CutsceneFrameNumber() == 0 || !reflect.DeepEqual(persisted, []string{"m20"}) || len(a.seenCutscenes()) != 1 {
					t.Fatal("first delivered frame did not unlock exactly once", persisted)
				}
			}
			a.HeadlessKey("escape")
			cold := newTestApp(t, appRows(1), okLoader(t))
			cold.SetCutsceneLibrary(catalog, persisted, nil, CutsceneLibraryWords{})
			cold.HeadlessActivate("cutscenes")
			want := 0
			if deliver {
				want = 1
			}
			if len(cold.HeadlessRows()) != want {
				t.Fatal("cold library differs from successfully delivered movies")
			}
		})
	}
}

func TestCutsceneLibraryEncounterReplayAndInput1184(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetWords(AuthoredWords(), chargenTestFont(), nil)
	catalog := []CutsceneEntry{{"intro", "Introduction"}, {"start", "First mission"}, {"m20", "Victory"}}
	var recorded []string
	a.SetCutsceneLibrary(catalog, nil, func(key string) error { recorded = append(recorded, key); return nil }, CutsceneLibraryWords{})
	if err := a.HeadlessActivate("cutscenes"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenCutsceneLibrary || len(a.HeadlessRows()) != 0 {
		t.Fatal("menu did not open an empty library")
	}
	a.HeadlessKey("enter")
	if a.Screen() != ScreenCutsceneLibrary {
		t.Fatal("empty library started a movie")
	}
	a.HeadlessKey("escape")
	movies := &eventMovies{present: map[string]bool{"logos/nival.smk": true, "intro/04.smk": true, "intro/05.smk": true, "m20/01.smk": true}}
	a.SetCutscenes(movies)
	a.PlayCutsceneSequence([]string{"logos/nival.smk", "intro/04.smk", "intro/05.smk", "start/01.smk"})
	completeEventMovies(t, a)
	if !reflect.DeepEqual(recorded, []string{"intro"}) || len(a.seenCutscenes()) != 1 {
		t.Fatal("missing movie or logo unlocked history", recorded)
	}
	a.HeadlessStep() // drain movie input
	if err := a.HeadlessActivate("cutscenes"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("Introduction"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenCutscene || a.CutsceneName() != "intro/04.smk" {
		t.Fatal("replay failed")
	}
	completeEventMovies(t, a)
	if a.Screen() != ScreenCutsceneLibrary || len(recorded) != 1 {
		t.Fatal("replay lost destination or duplicated history")
	}
	// A newly encountered mission movie is appended in catalog order.
	a.PlayCutscene("m20/01.smk")
	completeEventMovies(t, a)
	if !reflect.DeepEqual(recorded, []string{"intro", "m20"}) {
		t.Fatal(recorded)
	}
	if _, _, err := a.HeadlessFrame(); err != nil {
		t.Fatal(err)
	}
}

func TestCutsceneLibraryScrollAndPressCapture1184(t *testing.T) {
	a := newTestApp(t, nil, nil)
	a.SetWords(AuthoredWords(), chargenTestFont(), nil)
	var entries []CutsceneEntry
	var seen []string
	for i := 0; i < 14; i++ {
		key := fmt.Sprintf("m%d", i+1)
		entries = append(entries, CutsceneEntry{key, key})
		seen = append(seen, key)
	}
	a.SetCutsceneLibrary(entries, seen, nil, CutsceneLibraryWords{})
	a.openCutsceneLibrary()
	a.stepMedia(appInput{End: true}, time.Time{})
	if top, _ := a.movieList().Visible(); a.movieList().Selection() != 13 || top != 4 {
		t.Fatal("last movie not visible")
	}
	a.stepMedia(appInput{Home: true}, time.Time{})
	if top, _ := a.movieList().Visible(); a.movieList().Selection() != 0 || top != 0 {
		t.Fatal("first movie not visible")
	}
	p := movieCancel.Min.Add(image.Pt(3, 3))
	a.stepMedia(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true}, time.Time{})
	a.stepMedia(appInput{PrimaryReleased: true, CursorX: 1, CursorY: 1}, time.Time{})
	if a.Screen() != ScreenCutsceneLibrary {
		t.Fatal("release outside activated cancel")
	}
	a.stepMedia(appInput{Escape: true}, time.Time{})
	if a.Screen() != ScreenMenu {
		t.Fatal("cancel lost menu")
	}
}

func TestCreditsScrollLogoPauseClose1184(t *testing.T) {
	a := newTestApp(t, nil, nil)
	a.SetWords(AuthoredWords(), chargenTestFont(), nil)
	logo := image.NewRGBA(image.Rect(0, 0, 6, 4))
	for i := range logo.Pix {
		logo.Pix[i] = 255
	}
	a.SetCredits(func() CreditsView {
		return CreditsView{Lines: []string{"Nival", "", "", "", "", "", "", "", "GAME DESIGN", "Person"}, Logos: map[string]*image.RGBA{"nival": logo}}
	})
	if err := a.HeadlessActivate("credits"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenCredits {
		t.Fatal("credits menu inert")
	}
	now := time.Unix(500, 0)
	for i := 0; i <= 130; i++ {
		a.step(appInput{}, now.Add(time.Duration(i)*100*time.Millisecond))
	}
	before, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for i := 0; i < len(before.Pix); i += 4 {
		if before.Pix[i] > 0 {
			n++
		}
	}
	if n < 24 {
		t.Fatal("credits never painted logo or text", n)
	}
	offset := a.creditsScroll()
	a.step(appInput{Unfocused: true}, now.Add(time.Hour))
	a.step(appInput{}, now.Add(2*time.Hour))
	if a.creditsScroll() != offset || !bytes.Equal(before.Pix, a.composeCredits().Pix) {
		t.Fatal("credits advanced while unfocused")
	}
	a.step(appInput{Enter: true}, now.Add(2*time.Hour+time.Second))
	if a.Screen() != ScreenMenu || !a.cutsceneDrain {
		t.Fatal("credits did not close cleanly")
	}
	a.step(appInput{}, now.Add(2*time.Hour+2*time.Second))
	a.openCredits(ScreenMenu)
	if a.creditsScroll() != 0 {
		t.Fatal("reopen did not restart credits")
	}
	a.media.steps = 1 << 20
	a.step(appInput{}, now.Add(3*time.Hour))
	if a.Screen() != ScreenMenu {
		t.Fatal("natural credits end stayed open")
	}
}

func TestChargenPresetsAndSparkles1184(t *testing.T) {
	setup := ChargenSetup{Name: "Name", PreCreate: &ChargenPreCreate{Art: &ChargenPresentation{Layout: testGenerator()}}, Stats: []ChargenStat{{Floor: 1, Ceiling: 10, Start: 3}}, Cost: triangular(10), Budget: 100}
	setup.Presets[0], setup.Presets[3] = []int{6}, []int{8}
	c := NewChargen(setup)
	c.Forward()
	if c.statValue[0] != 6 {
		t.Fatal("preset ignored")
	}
	c.AdjustStat(0, -1)
	c.Reset()
	if c.statValue[0] != 6 {
		t.Fatal("reset did not restore selected preset")
	}
	c.Back()
	c.SelectPreChoice(3)
	c.Forward()
	if c.statValue[0] != 8 {
		t.Fatal("archetype reused preceding preset")
	}
	c.Back()
	for i := 0; i < 3; i++ {
		p := image.NewRGBA(image.Rect(0, 0, 2, 2))
		p.SetRGBA(0, 0, color.RGBA{255, 255, 255, 255})
		setup.PreCreate.Art.Sparkles = append(setup.PreCreate.Art.Sparkles, p)
	}
	start := time.Unix(42, 0)
	c.advancePresentation(start, true)
	hidden := ComposeChargenFrame(c)
	c.advancePresentation(start.Add(63*time.Millisecond), true)
	shown := ComposeChargenFrame(c)
	if bytes.Equal(hidden.Pix, shown.Pix) {
		t.Fatal("sparkle never composed")
	}
	c.advancePresentation(start.Add(time.Hour), false)
	c.advancePresentation(start.Add(2*time.Hour), true)
	if !bytes.Equal(shown.Pix, ComposeChargenFrame(c).Pix) {
		t.Fatal("unfocused sparkle clock caught up")
	}
}

func TestTooltipAroundOpenTips1184(t *testing.T) {
	a := NewApp("tips", nil, nil, nil)
	a.Layout(640, 480)
	font := messageFont()
	panel := TipPanelView{Rect: image.Rect(200, 300, 400, 450), Text: "Tips", Art: &TipPanelArt{}, Font: font}
	shop := &tooltipShopFixture{view: ShopScreenView{Font: font, Live: [4]bool{true, true, true, true}, TipPanel: panel}}
	shop.view.Shelf[0] = ShopCell{Back: 1, Info: []string{"Sword", "Damage 5"}}
	a.flow.town, a.flow.screen, a.flow.townList = shop, ScreenTown, NewPicker(nil)
	p := ShopShelfCellRect(0).Min.Add(image.Pt(10, 10))
	if panel.Covers(p) {
		t.Fatal("fixture hides item")
	}
	in := appInput{CursorX: p.X, CursorY: p.Y}
	start := time.Unix(100, 0)
	a.step(in, start)
	a.step(in, start.Add(500*time.Millisecond))
	if state, _ := a.HeadlessTooltip(); !state.Visible {
		t.Fatal("tip outside item still suppresses hover")
	}
	shop.view.TipPanel.Rect = ShopShelfCellRect(0)
	if a.tooltipTarget().key() != "" {
		t.Fatal("covered item leaked through panel")
	}
	c := NewChargen(ChargenSetup{PreCreate: &ChargenPreCreate{Art: &ChargenPresentation{Layout: testGenerator(), Font: font}}, TipArt: &TipPanelArt{}, TipSelect: [3]string{"Tips"}, TipsOn: true})
	a.flow.chargen, a.flow.screen = c, ScreenChargen
	a.flow.words.Hover[255], a.flow.words.Hover[256] = "Back", "Character name"
	if got := a.chargenTooltip(image.Pt(600, 200)); got.key() == "" {
		t.Fatal("generator tips suppressed the visible Back tooltip")
	}
	// The amulet's top lies inside the open tip panel's rect, which owns it.
	if got := a.chargenTooltip(image.Pt(600, 134)); got.key() != "" {
		t.Fatalf("the Back tooltip leaked through the open tip panel: %q", got.key())
	}
}
