package ui

import (
	"image"
	"testing"
	"time"
)

// The generator's own tip panel (1018 spec behaviours 1, 2, 3, 4): shown
// only on the pre-create page (TOWN-187's own "chrgen2.txt is a separate,
// once-only-latch mechanism, out of this story's scope" — TipPanel must
// never show past Forward), closeable for the visit, and its toggle
// round-tripped through the setup's own SetTipsOn callback rather than
// computed here (AC-11: this package has no store of its own). TipPanel's
// own TEXT tracks the player's live pre-create portrait selection (round 2 /
// DIV-162 correction, TestPreCreateTipTextFollowsTheLiveClassChoice below):
// TipText is the fighter branch, TipTextMage the mage branch, and
// c.preChoice's own class half (preChoiceParts) picks between them.

func chargenTipSetup() ChargenSetup {
	return ChargenSetup{
		Name:      "Danath",
		PreCreate: &ChargenPreCreate{Art: &ChargenPresentation{Font: shopTipTestFont()}},
		Choices:   []ChargenChoice{{Options: []string{"male", "female"}, Parent: -1}},
		Stats:     []ChargenStat{{Floor: 0, Ceiling: 50, Start: 25}},
		Cost:      triangular(50),
		Budget:    1000,
		TipText:   "the mage draws power from the well",
		TipArt:    tipTestArt(),
		TipsOn:    true,
	}
}

// TipPanel shows during PreCreateStage, with the setup's own text, art and
// toggle state carried through unchanged.
func TestChargenTipPanelShowsOnPreCreate(t *testing.T) {
	c := NewChargen(chargenTipSetup())
	v := c.TipPanel()
	if !v.Showing() {
		t.Fatal("TipPanel on a fresh pre-create stage is not Showing()")
	}
	if v.Text != "the mage draws power from the well" {
		t.Fatalf("TipPanel.Text = %q, want the setup's own TipText", v.Text)
	}
	if !v.ToggleOn {
		t.Fatal("TipPanel.ToggleOn = false, want the setup's own TipsOn (true)")
	}
}

// Forward leaves PreCreateStage; TipPanel then answers a zero, non-Showing
// view, never a stale panel drawn over the detailed page (TOWN-187's own
// chrgen2.txt second popup is out of scope, so the pre-create panel must not
// persist past it).
func TestChargenTipPanelHidesPastPreCreate(t *testing.T) {
	c := NewChargen(chargenTipSetup())
	c.Forward()
	if c.Stage() != DetailedStage {
		t.Fatal("fixture did not reach DetailedStage")
	}
	if v := c.TipPanel(); v.Showing() {
		t.Fatalf("TipPanel() on DetailedStage = %+v, want a zero view", v)
	}
}

// A nil TipArt (no setup) answers a zero view rather than a panel with a
// blank picture — the same "no art, not Showing" contract tippanel_test.go
// already proves for TipPanelView.Showing.
func TestChargenTipPanelWithNoArtDoesNotShow(t *testing.T) {
	setup := chargenTipSetup()
	setup.TipArt = nil
	c := NewChargen(setup)
	if v := c.TipPanel(); v.Showing() {
		t.Fatalf("TipPanel() with no TipArt = %+v, want a zero view", v)
	}
}

// CloseTip dismisses the panel for the rest of this generator's own visit
// (spec behaviour 3): a fresh Chargen is unaffected by another instance's
// close, and Forward/Back does not reopen it.
func TestChargenCloseTipDismissesForTheVisit(t *testing.T) {
	c := NewChargen(chargenTipSetup())
	c.CloseTip()
	if v := c.TipPanel(); v.Showing() {
		t.Fatal("TipPanel() after CloseTip is still Showing()")
	}
	c.Forward()
	c.Back()
	if v := c.TipPanel(); v.Showing() {
		t.Fatal("TipPanel() after CloseTip, Forward and Back is still Showing()")
	}
}

// ToggleTips flips the setup's own TipsOn and calls SetTipsOn with the new
// value (spec behaviour 4) — the persistence itself is the caller's
// responsibility, proved here only by the callback's own argument.
func TestChargenToggleTipsCallsSetTipsOn(t *testing.T) {
	setup := chargenTipSetup()
	var calls []bool
	setup.SetTipsOn = func(on bool) { calls = append(calls, on) }
	c := NewChargen(setup)

	c.ToggleTips()
	if len(calls) != 1 || calls[0] != false {
		t.Fatalf("SetTipsOn calls after one ToggleTips = %v, want [false]", calls)
	}
	if v := c.TipPanel(); v.ToggleOn {
		t.Fatal("TipPanel.ToggleOn after ToggleTips is still true")
	}

	c.ToggleTips()
	if len(calls) != 2 || calls[1] != true {
		t.Fatalf("SetTipsOn calls after a second ToggleTips = %v, want [false true]", calls)
	}
}

// ToggleTips with no SetTipsOn callback (a hand-built test setup, on
// Preview/Derive's own "may be nil" precedent) does not panic, and leaves
// TipsOn unchanged since there is nowhere to persist a flip.
func TestChargenToggleTipsWithNoCallbackIsANoOp(t *testing.T) {
	setup := chargenTipSetup()
	setup.SetTipsOn = nil
	c := NewChargen(setup)
	c.ToggleTips()
	if v := c.TipPanel(); !v.ToggleOn {
		t.Fatal("ToggleTips with no SetTipsOn callback changed ToggleOn")
	}
}

// A nil *Chargen's TipPanel, CloseTip and ToggleTips do not panic, on the
// rest of this file's own defensive-nil-receiver convention.
func TestNilChargenTipMethodsDoNotPanic(t *testing.T) {
	var c *Chargen
	if v := c.TipPanel(); v.Showing() {
		t.Fatal("nil Chargen TipPanel() is Showing()")
	}
	c.CloseTip()
	c.ToggleTips()
}

func TestChargenTipSuppressionSurvivesForwardAndBack(t *testing.T) {
	setup := chargenTipSetup()
	setup.SetTipsOn = func(on bool) {}
	c := NewChargen(setup)
	if v := c.TipPanel(); !v.Showing() {
		t.Fatal("TipPanel() on a fresh pre-create stage is not Showing()")
	}

	c.ToggleTips()
	if v := c.TipPanel(); !v.Showing() {
		t.Fatal("TipPanel() right after ToggleTips is not Showing() — toggling must not retroactively hide an open panel")
	}
	if v := c.TipPanel(); v.ToggleOn {
		t.Fatal("TipPanel().ToggleOn after ToggleTips is still true")
	}

	c.Forward()
	if v := c.TipPanel(); v.Showing() {
		t.Fatal("TipPanel() on DetailedStage is Showing()")
	}
	if !c.Back() {
		t.Fatal("Back() returned false with PreCreate set")
	}
	if v := c.TipPanel(); v.Showing() {
		t.Fatal("TipPanel() after Forward and Back, tips off, is Showing() — suppression did not survive re-entry")
	}

	// Toggling back on mid-visit does not retroactively reopen it either
	// (the same rule, the other direction): only the NEXT entry re-reads it.
	c.ToggleTips()
	if v := c.TipPanel(); v.Showing() {
		t.Fatal("TipPanel() right after re-toggling on, still on pre-create from the suppressed entry, is Showing()")
	}
	c.Forward()
	if !c.Back() {
		t.Fatal("second Back() returned false with PreCreate set")
	}
	if v := c.TipPanel(); !v.Showing() {
		t.Fatal("TipPanel() after a further Forward/Back with tips back on is not Showing()")
	}
}

// TestPreCreateTipTextFollowsTheLiveClassChoice is round 2's Finding B
// witness (DIV-162): a fresh Chargen shows the fighter text (preChoice's own
// zero value, class 0); clicking the male-mage portrait (choice 1,
// preChoiceParts(1) = sex 0, class 1) through the real dispatch
// (stepPreCreate, on TestPreCreateFlow's own pattern, chargen_app_test.go)
// switches TipPanel().Text to the mage text; clicking the female-fighter
// portrait (choice 2, class 0) switches it back. Before this fix, TipText
// was resolved once at ChargenSetup() construction and never read again
// past that; SelectPreChoice never touched it, so this test would see the
// fighter text at every step, including after the mage portrait was chosen.
func TestPreCreateTipTextFollowsTheLiveClassChoice(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	setup := chargenLegalSetup()
	art := &ChargenPresentation{Forward: image.NewRGBA(image.Rect(0, 0, 96, 74)), Font: shopTipTestFont()}
	for choice := range art.Choices {
		for state := range art.Choices[choice] {
			art.Choices[choice][state] = image.NewRGBA(image.Rect(0, 0, 40, 40))
		}
	}
	setup.PreCreate = &ChargenPreCreate{Art: art}
	setup.TipsOn = true
	setup.TipArt = tipTestArt()
	setup.TipText = "fighter: five weapon skills"
	setup.TipTextMage = "mage: five magic spheres"
	a := newTestApp(t, appRows(1), okLoader(t))
	c := NewChargen(setup)
	if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, nil }); err != nil {
		t.Fatal(err)
	}

	if got := c.TipPanel().Text; got != setup.TipText {
		t.Fatalf("TipPanel().Text on a fresh generator = %q, want the fighter text %q", got, setup.TipText)
	}

	click := func(choice chargenControl) {
		t.Helper()
		at := preControlRect(c, choice).Min
		a.step(appInput{CursorX: at.X, CursorY: at.Y, PrimaryPressed: true}, now)
		a.step(appInput{CursorX: at.X, CursorY: at.Y, PrimaryReleased: true}, now)
	}

	click(chargenChoice1) // male mage
	if c.PreChoice() != 1 {
		t.Fatalf("PreChoice() after the mage portrait click = %d, want 1", c.PreChoice())
	}
	if got := c.TipPanel().Text; got != setup.TipTextMage {
		t.Fatalf("TipPanel().Text after picking the mage portrait = %q, want the mage text %q", got, setup.TipTextMage)
	}

	click(chargenChoice2) // female fighter
	if c.PreChoice() != 2 {
		t.Fatalf("PreChoice() after the fighter portrait click = %d, want 2", c.PreChoice())
	}
	if got := c.TipPanel().Text; got != setup.TipText {
		t.Fatalf("TipPanel().Text after picking the fighter portrait = %q, want the fighter text %q", got, setup.TipText)
	}
}
