package ui

import (
	"testing"
	"time"
)

// The generator's two tip popups (TOWN-518, TOWN-522, MENU-136, MENU-137):
// the pre-create popup at PreCreateTipRect steps chrsel1..3 on portrait and
// level clicks; the detailed popup at ChargenTipRect shows the class text,
// then chrgen2 after the first skill click.

func chargenTipSetup() ChargenSetup {
	return ChargenSetup{
		Name:      "Danath",
		PreCreate: &ChargenPreCreate{Art: &ChargenPresentation{Font: shopTipTestFont()}},
		Choices: []ChargenChoice{{Options: []string{"male", "female"}, Parent: -1},
			{Options: []string{"fighter", "mage"}, Parent: -1},
			{Options: []string{"a", "b", "c", "d", "e"}, Parent: -1}},
		Stats:         []ChargenStat{{Floor: 0, Ceiling: 50, Start: 25}},
		Cost:          triangular(50),
		Budget:        1000,
		TipSelect:     [3]string{"choose a hero", "choose a level", "press OK"},
		TipText:       "fighter text",
		TipTextMage:   "mage text",
		TipTextDetail: "after the skill",
		TipArt:        tipTestArt(),
		TipsOn:        true,
	}
}

func TestChargenTipPanelShowsOnPreCreate(t *testing.T) {
	c := NewChargen(chargenTipSetup())
	v := c.TipPanel()
	if !v.Showing() || v.Text != "choose a hero" || v.Rect != PreCreateTipRect || !v.ToggleOn {
		t.Fatalf("pre-create enter popup = %+v, want chrsel1 at %v", v, PreCreateTipRect)
	}
}

// TestPreCreateTipStepsOnlyOnTheNamedClicks: a portrait click at step 0 and a
// level click at step 1 advance the step; a level click at step 0, a second
// portrait click and the closed popup do not (TOWN-518).
func TestPreCreateTipStepsOnlyOnTheNamedClicks(t *testing.T) {
	c := NewChargen(chargenTipSetup())
	c.tipLevelClicked()
	if c.TipStep() != 0 {
		t.Fatal("a level click at step 0 advanced the step")
	}
	c.tipPortraitClicked()
	if c.TipStep() != 1 || c.TipPanel().Text != "choose a level" {
		t.Fatalf("portrait click: step %d text %q", c.TipStep(), c.TipPanel().Text)
	}
	c.tipPortraitClicked()
	if c.TipStep() != 1 {
		t.Fatal("a portrait click at step 1 advanced the step")
	}
	c.tipLevelClicked()
	if c.TipStep() != 2 || c.TipPanel().Text != "press OK" {
		t.Fatalf("level click: step %d text %q", c.TipStep(), c.TipPanel().Text)
	}
	c.Forward()
	c.Back()
	if c.TipStep() != 0 || c.TipPanel().Text != "choose a hero" {
		t.Fatal("the page enter did not restart the step at chrsel1")
	}
	c.CloseTip()
	c.tipPortraitClicked()
	if c.TipStep() != 0 || c.TipPanel().Showing() {
		t.Fatal("a click after Close advanced the step or showed the popup")
	}
}

func TestChargenDetailedTipFollowsClassThenFirstSkillClick(t *testing.T) {
	for class, want := range []string{"fighter text", "mage text"} {
		c := NewChargen(chargenTipSetup())
		c.SelectPreChoice([]int{0, 1}[class])
		c.Forward()
		v := c.TipPanel()
		if v.Text != want || v.Rect != ChargenTipRect {
			t.Fatalf("class %d detailed enter popup = %q at %v", class, v.Text, v.Rect)
		}
		c.tipSkillClicked()
		if c.TipPanel().Text != "after the skill" || c.TipStep() != 1 {
			t.Fatalf("first skill click: %q step %d", c.TipPanel().Text, c.TipStep())
		}
	}
}

// TestChargenTipsModeOffStopsOnlyWhatTheClaimsSay: clearing TipsMode leaves
// the open popup and stops its steps; the next enter builds no popup
// (MENU-136, TOWN-518, TOWN-522).
func TestChargenTipsModeOffStopsOnlyWhatTheClaimsSay(t *testing.T) {
	setup := chargenTipSetup()
	var calls []bool
	setup.SetTipsOn = func(on bool) { calls = append(calls, on) }
	c := NewChargen(setup)
	c.ToggleTips()
	if len(calls) != 1 || calls[0] {
		t.Fatalf("SetTipsOn calls %v, want [false]", calls)
	}
	if v := c.TipPanel(); !v.Showing() || v.ToggleOn {
		t.Fatal("clearing TipsMode deleted the open popup or kept the checkbox on")
	}
	c.tipPortraitClicked()
	if c.TipStep() != 0 {
		t.Fatal("a portrait click with TipsMode clear advanced the step")
	}
	c.Forward()
	if c.TipPanel().Showing() {
		t.Fatal("the detailed enter with TipsMode clear built a popup")
	}
	c.Back()
	if c.TipPanel().Showing() {
		t.Fatal("the pre-create enter with TipsMode clear built a popup")
	}
	c.ToggleTips()
	if c.TipPanel().Showing() {
		t.Fatal("setting TipsMode showed a popup without an enter")
	}
	c.Forward()
	if !c.TipPanel().Showing() {
		t.Fatal("the next enter with TipsMode set built no popup")
	}
}

func TestChargenTipPanelWithNoArtDoesNotShow(t *testing.T) {
	setup := chargenTipSetup()
	setup.TipArt = nil
	if v := NewChargen(setup).TipPanel(); v.Showing() {
		t.Fatalf("TipPanel() with no TipArt = %+v, want a zero view", v)
	}
}

func TestNilChargenTipMethodsDoNotPanic(t *testing.T) {
	var c *Chargen
	if v := c.TipPanel(); v.Showing() {
		t.Fatal("nil Chargen TipPanel() is Showing()")
	}
	c.CloseTip()
	c.ToggleTips()
	c.tipPortraitClicked()
	c.tipSkillClicked()
}

// TestGuidedCycleTiming drives the cycle on a synthetic clock: nothing for
// 500 ms, then one step at the first paint more than 300 ms after the last;
// a hover freezes it and it resumes after the hovered target (TOWN-519).
func TestGuidedCycleTiming(t *testing.T) {
	steps := [][]int{{80, 100, 120, 140}, {20, 40, 60}, {160, 180}}
	var g guidedCycle
	t0 := time.Unix(1000, 0)
	at := func(ms int) time.Time { return t0.Add(time.Duration(ms) * time.Millisecond) }
	type frame struct {
		ms, step, hover, want int
	}
	for _, f := range []frame{
		{0, 0, -1, -1}, {499, 0, -1, -1},
		{500, 0, -1, 0}, {800, 0, -1, 0}, {801, 0, -1, 1},
		{1101, 0, -1, 1}, {1102, 0, -1, 2}, {1403, 0, -1, 2}, {1404, 0, -1, 3}, {1704, 0, -1, 3}, {1705, 0, -1, 0},
		{1710, 0, 120, -1}, {2000, 0, -1, -1}, {2209, 0, -1, -1},
		{2210, 0, -1, 3}, {2511, 0, -1, 3}, {2512, 0, -1, 0},
		{2600, 1, -1, 0}, {2812, 1, -1, 0}, {2813, 1, -1, 1}, {3200, 2, -1, 1}, {3201, 2, -1, 0}, {3300, 7, -1, -1},
	} {
		if got := g.paint(at(f.ms), steps, f.step, f.hover); got != f.want {
			t.Fatalf("paint at %d ms step %d hover %d = %d, want %d", f.ms, f.step, f.hover, got, f.want)
		}
	}
}
