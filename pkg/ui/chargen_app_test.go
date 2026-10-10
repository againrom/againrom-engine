package ui

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui/systemclick"
)

// chargenLegalSetup is a small, always-legal-at-Start setup: one choice row
// and two statistic rows, sized so Start costs comfortably less than the
// budget.
func chargenLegalSetup() ChargenSetup {
	return ChargenSetup{
		Title:   "GENERATE A HERO",
		Choices: []ChargenChoice{{Name: "Sex", Options: []string{"Male", "Female"}, Parent: -1}},
		Stats: []ChargenStat{
			{Name: "A", Floor: 0, Ceiling: 10, Start: 3},
			{Name: "B", Floor: 0, Ceiling: 10, Start: 3},
		},
		Cost:    triangular(10),
		Budget:  100,
		Confirm: "ENTER: begin",
	}
}

// chargenNamedSetup is chargenLegalSetup with a name in the field. The
// pre-create page's OK, Enter and hero double-click continue only with a
// name (TEXT-CHARGEN-028), so a test of what leaves the page, or of what
// keeps a press from leaving it, needs one to test that rule and not the name.
func chargenNamedSetup() ChargenSetup {
	s := chargenLegalSetup()
	s.Name = "Hero"
	return s
}

// chargenIllegalSetup starts illegal, mirroring chargen_test.go's own AC-6
// fixture: two statistics whose Start already costs one more than the budget
// allows, so the refusal tests need no Adjust walk to reach an illegal
// spread — Adjust itself refuses every step that would create one, so the
// only way to one is to start there.
func chargenIllegalSetup() ChargenSetup {
	cost := triangular(10)
	return ChargenSetup{
		Title: "t",
		Stats: []ChargenStat{
			{Name: "A", Floor: 0, Ceiling: 10, Start: 5},
			{Name: "B", Floor: 0, Ceiling: 10, Start: 5},
		},
		Cost:    cost,
		Budget:  cost[5] + cost[5] - 1,
		Confirm: "go",
	}
}

// okOpener returns a MapOpener over a synthetic grid, mirroring app_test.go's
// own okLoader for MapLoader: a viewer over nothing running under it.
func okOpener(t *testing.T) MapOpener {
	t.Helper()
	return func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		v, err := NewViewer("m", grid(20, 20), &terrain.Tileset{})
		return v, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
}

func TestChargenAppDispatch(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)

	t.Run("OpenChargen arms the screen and opens nothing", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		c := NewChargen(chargenLegalSetup())
		called := 0
		begin := func(ChargenResult) (MapOpener, error) {
			called++
			return nil, nil
		}
		if err := a.OpenChargen(c, begin); err != nil {
			t.Fatalf("OpenChargen: %v", err)
		}
		if a.Screen() != ScreenChargen {
			t.Fatalf("Screen() = %v, want ScreenChargen", a.Screen())
		}
		if a.flow.viewer != nil {
			t.Errorf("OpenChargen opened a viewer")
		}
		if called != 0 {
			t.Errorf("OpenChargen called begin %d times, want 0", called)
		}
	})

	t.Run("OpenChargen refuses a nil model and opens nothing", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		if err := a.OpenChargen(nil, func(ChargenResult) (MapOpener, error) { return nil, nil }); err == nil {
			t.Fatalf("OpenChargen(nil, ...) = nil error, want one")
		}
		if a.Screen() != ScreenMenu {
			t.Errorf("a refused OpenChargen changed the screen to %v, want ScreenMenu unchanged", a.Screen())
		}
	})

	t.Run("the four movement keys reach the model", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		c := NewChargen(chargenLegalSetup())
		if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, nil }); err != nil {
			t.Fatalf("OpenChargen: %v", err)
		}
		if got := c.Focus(); got != 0 {
			t.Fatalf("setup: initial Focus() = %d, want 0", got)
		}

		// Down moves the focus off the choice row (0) onto the first statistic (1).
		a.step(appInput{Down: true}, now)
		if got := c.Focus(); got != 1 {
			t.Fatalf("Down did not move the focus: Focus() = %d, want 1", got)
		}

		// Right steps the focused statistic up by one, Left steps it back.
		before := c.RowText(1)
		a.step(appInput{Right: true}, now)
		if got := c.RowText(1); got == before {
			t.Fatalf("Right did not change the focused statistic: still %q", got)
		}
		a.step(appInput{Left: true}, now)
		if got := c.RowText(1); got != before {
			t.Fatalf("Left did not undo the step: got %q, want %q", got, before)
		}

		// Up moves the focus back onto the choice row.
		a.step(appInput{Up: true}, now)
		if got := c.Focus(); got != 0 {
			t.Fatalf("Up did not move the focus back: Focus() = %d, want 0", got)
		}
	})

	t.Run("Enter on an illegal spread calls begin zero times and leaves a message", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		c := NewChargen(chargenIllegalSetup())
		if c.Legal() {
			t.Fatalf("fixture: want an illegal spread at Start")
		}
		called := 0
		if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) {
			called++
			return nil, nil
		}); err != nil {
			t.Fatalf("OpenChargen: %v", err)
		}

		a.step(appInput{Enter: true}, now)

		if called != 0 {
			t.Errorf("begin was called %d times on an illegal spread, want 0", called)
		}
		if a.Screen() != ScreenChargen {
			t.Errorf("Screen() = %v after an illegal confirm, want ScreenChargen unchanged", a.Screen())
		}
		if a.flow.msg == "" {
			t.Errorf("an illegal confirm left no message")
		}
	})

	t.Run("Enter on a legal spread calls begin exactly once and enters the opener", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		c := NewChargen(chargenLegalSetup())
		if !c.Legal() {
			t.Fatalf("fixture: want a legal spread at Start")
		}
		wantResult, ok := c.Result()
		if !ok {
			t.Fatalf("fixture: Result() reported illegal for a legal spread")
		}

		called := 0
		var gotResult ChargenResult
		begin := func(r ChargenResult) (MapOpener, error) {
			called++
			gotResult = r
			return okOpener(t), nil
		}
		if err := a.OpenChargen(c, begin); err != nil {
			t.Fatalf("OpenChargen: %v", err)
		}

		a.step(appInput{Enter: true}, now)

		if called != 1 {
			t.Fatalf("begin was called %d times, want exactly 1", called)
		}
		if fmt.Sprint(gotResult) != fmt.Sprint(wantResult) {
			t.Errorf("begin received %+v, want %+v", gotResult, wantResult)
		}
		if a.Screen() != ScreenMap {
			t.Errorf("Screen() = %v after a legal confirm with a non-nil opener, want ScreenMap", a.Screen())
		}
	})

	t.Run("a begin error leaves the screen showing with the error as the message", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		c := NewChargen(chargenLegalSetup())
		boom := errors.New("cannot build a party: no base row resolved")
		if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, boom }); err != nil {
			t.Fatalf("OpenChargen: %v", err)
		}

		a.step(appInput{Enter: true}, now)

		if a.Screen() != ScreenChargen {
			t.Errorf("Screen() = %v after a begin error, want ScreenChargen unchanged", a.Screen())
		}
		if a.flow.msg != boom.Error() {
			t.Errorf("flow.msg = %q, want %q", a.flow.msg, boom.Error())
		}
	})

	t.Run("an opener error leaves the screen showing with the error as the message", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		c := NewChargen(chargenLegalSetup())
		boom := errors.New("decode grid: payload short")
		begin := func(ChargenResult) (MapOpener, error) {
			return func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
				return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, boom
			}, nil
		}
		if err := a.OpenChargen(c, begin); err != nil {
			t.Fatalf("OpenChargen: %v", err)
		}

		a.step(appInput{Enter: true}, now)

		if a.Screen() != ScreenChargen {
			t.Errorf("Screen() = %v after an opener error, want ScreenChargen unchanged", a.Screen())
		}
		if a.flow.msg != boom.Error() {
			t.Errorf("flow.msg = %q, want %q", a.flow.msg, boom.Error())
		}
	})

	t.Run("Escape returns to the menu", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		c := NewChargen(chargenLegalSetup())
		if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, nil }); err != nil {
			t.Fatalf("OpenChargen: %v", err)
		}

		if exit := a.step(appInput{Escape: true}, now); exit {
			t.Fatalf("Escape on the chargen screen exited the program")
		}
		if a.Screen() != ScreenMenu {
			t.Errorf("Screen() = %v after Escape, want ScreenMenu", a.Screen())
		}
	})
}

func TestPreCreateFlow(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	setup := chargenLegalSetup()
	art := &ChargenPresentation{Layout: testGenerator(), Forward: image.NewRGBA(image.Rect(0, 0, 96, 74))}
	for choice := range art.Choices {
		for state := range art.Choices[choice] {
			art.Choices[choice][state] = image.NewRGBA(image.Rect(0, 0, 160, 240))
		}
	}
	setup.PreCreate = &ChargenPreCreate{Art: art}
	a := newTestApp(t, appRows(1), okLoader(t))
	c := NewChargen(setup)
	if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, nil }); err != nil {
		t.Fatal(err)
	}
	// A press on a picture chooses it at once (VIDEO-SFX-058); its release,
	// here outside every control, changes nothing.
	a.step(appInput{CursorX: 240, CursorY: 300, PrimaryPressed: true}, now)
	if c.PreChoice() != 2 {
		t.Fatalf("press chose %d, want 2", c.PreChoice())
	}
	a.step(appInput{CursorX: 10, CursorY: 10, PrimaryReleased: true}, now)
	if c.PreChoice() != 2 {
		t.Fatalf("release elsewhere changed choice to %d", c.PreChoice())
	}
	choice := preControlRect(c, chargenChoice1).Min
	a.step(appInput{CursorX: choice.X, CursorY: choice.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: choice.X, CursorY: choice.Y, PrimaryReleased: true}, now)
	if c.PreChoice() != 1 {
		t.Fatalf("picture press chose %d, want 1", c.PreChoice())
	}
	// Focus starts at Name; the first accepted input replaces the default.
	a.step(appInput{Typed: "A"}, now)
	if c.NameText() != "A" {
		t.Fatalf("NameText = %q, want A", c.NameText())
	}
	// The test pictures cover OK's origin, so the press that continues is a
	// double click on the hero picture at the first press's point.
	a.step(appInput{CursorX: choice.X, CursorY: choice.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: choice.X, CursorY: choice.Y, PrimaryReleased: true}, now)
	if c.Stage() != DetailedStage {
		t.Fatal("Forward did not open detailed shell")
	}
	a.step(appInput{Escape: true}, now)
	if c.Stage() != PreCreateStage || a.Screen() != ScreenChargen {
		t.Fatal("detailed Escape did not return to pre-create")
	}
	a.step(appInput{Escape: true}, now)
	if a.Screen() != ScreenMenu {
		t.Fatalf("pre-create Escape screen = %v, want menu", a.Screen())
	}
}

func TestPreCreateChoiceDoubleClickMatchesForwardOnce(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	setup := chargenNamedSetup()
	calls := 0
	setup.Preview = func(ChargenResult) ChargenPreview {
		calls++
		return ChargenPreview{}
	}
	art := &ChargenPresentation{Layout: testGenerator()}
	for choice := range art.Choices {
		for state := range art.Choices[choice] {
			art.Choices[choice][state] = image.NewRGBA(image.Rect(0, 0, 8, 8))
		}
	}
	setup.PreCreate = &ChargenPreCreate{Art: art}
	a := newTestApp(t, appRows(1), okLoader(t))
	c := NewChargen(setup)
	if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, nil }); err != nil {
		t.Fatal(err)
	}
	at := preControlRect(c, chargenChoice3).Min
	for click := 0; click < 2; click++ {
		when := now.Add(time.Duration(click) * 100 * time.Millisecond)
		a.step(appInput{CursorX: at.X, CursorY: at.Y, PrimaryPressed: true}, when)
		a.step(appInput{CursorX: at.X, CursorY: at.Y, PrimaryReleased: true}, when)
		if click == 0 && c.Stage() != PreCreateStage {
			t.Fatal("first choice click entered detailed stage")
		}
	}
	if c.Stage() != DetailedStage || c.PreChoice() != 3 || calls != 1 {
		t.Fatalf("choice double-click stage=%v choice=%d preview calls=%d, want detailed/3/1", c.Stage(), c.PreChoice(), calls)
	}
}

func TestPreCreateChoiceDoubleClickRequiresTheDetectorsTimeAndRectangle(t *testing.T) {
	newArmed := func(t *testing.T) (*App, *Chargen, *int) {
		t.Helper()
		setup := chargenNamedSetup()
		calls := 0
		setup.Preview = func(ChargenResult) ChargenPreview { calls++; return ChargenPreview{} }
		art := &ChargenPresentation{Layout: testGenerator()}
		for choice := range art.Choices {
			for state := range art.Choices[choice] {
				art.Choices[choice][state] = image.NewRGBA(image.Rect(0, 0, 8, 8))
			}
		}
		setup.PreCreate = &ChargenPreCreate{Art: art}
		a := newTestApp(t, appRows(1), okLoader(t))
		c := NewChargen(setup)
		if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, nil }); err != nil {
			t.Fatal(err)
		}
		return a, c, &calls
	}
	click := func(a *App, c *Chargen, id chargenControl, at time.Time) {
		p := preControlRect(c, id).Min
		a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true}, at)
		a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true}, at)
	}
	now := time.Unix(1_700_000_000, 0)
	t.Run("expired second click only selects", func(t *testing.T) {
		a, c, calls := newArmed(t)
		click(a, c, chargenChoice1, now)
		click(a, c, chargenChoice1, now.Add(systemclick.Fallback.Time+time.Millisecond))
		if c.Stage() != PreCreateStage || c.PreChoice() != 1 || *calls != 0 {
			t.Fatalf("expired click stage=%v choice=%d preview=%d", c.Stage(), c.PreChoice(), *calls)
		}
	})
	t.Run("different choice never launches", func(t *testing.T) {
		a, c, calls := newArmed(t)
		click(a, c, chargenChoice1, now)
		click(a, c, chargenChoice2, now.Add(time.Millisecond))
		if c.Stage() != PreCreateStage || c.PreChoice() != 2 || *calls != 0 {
			t.Fatalf("different choice stage=%v choice=%d preview=%d", c.Stage(), c.PreChoice(), *calls)
		}
	})
}

// TestPreCreateKeyboardChoiceDoesNotBecomePointerDoubleClick keeps the timed
// Forward shortcut scoped to a completed mouse gesture. Enter activates the
// focused picture normally each time; two key activations must never reuse the
// pointer-only click latch to leave pre-create.
func TestPreCreateKeyboardChoiceDoesNotBecomePointerDoubleClick(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	setup := chargenNamedSetup()
	previewCalls := 0
	setup.Preview = func(ChargenResult) ChargenPreview {
		previewCalls++
		return ChargenPreview{}
	}
	setup.PreCreate = &ChargenPreCreate{Art: &ChargenPresentation{Layout: testGenerator()}}
	c := NewChargen(setup)
	// Focus the fourth class/sex picture. The first Enter selects it; the
	// second is deliberately inside the pointer double-click interval.
	c.Move(4)
	a := newTestApp(t, appRows(1), okLoader(t))
	if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, nil }); err != nil {
		t.Fatal(err)
	}
	a.step(appInput{Enter: true}, now)
	a.step(appInput{Enter: true}, now.Add(time.Millisecond))
	if c.Stage() != PreCreateStage || c.PreChoice() != 3 || previewCalls != 0 {
		t.Fatalf("two rapid keyboard choice activations stage=%v choice=%d previews=%d, want pre-create/3/0",
			c.Stage(), c.PreChoice(), previewCalls)
	}
}

func TestDetailedSkillClicksMoveTheOneSelectedSourceState(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	font := chargenTestFont()
	art := &ChargenPresentation{Layout: testGenerator(), Plate: image.NewRGBA(image.Rect(0, 0, 160, 238)), Font: font}
	art.ColumnMask[0] = chargenMask(320, 480)
	for skill := 0; skill < 5; skill++ {
		for state := 0; state < 3; state++ {
			pic := image.NewRGBA(image.Rect(0, 0, 5, 5))
			for y := 0; y < 5; y++ {
				for x := 0; x < 5; x++ {
					pic.SetRGBA(x, y, color.RGBA{R: uint8(40 + 10*skill + state), A: 255})
				}
			}
			art.Skills[0][skill][state] = pic
		}
		at := detailedSkillOrigin[0][skill]
		art.ColumnMask[0].SetColorIndex(at.X+2, at.Y+2, detailedMaskCode[0][skill])
	}
	seen := []int{}
	setup := ChargenSetup{
		PreCreate: &ChargenPreCreate{Art: art},
		Choices: []ChargenChoice{
			{Options: []string{"m"}, Parent: -1},
			{Options: []string{"fighter"}, Parent: -1},
			{Options: []string{"blade", "axe", "bludgeon", "pike", "shooting"}, Parent: -1},
		},
		Stats: []ChargenStat{{Floor: 0, Ceiling: 50, Start: 25}}, Cost: triangular(50), Budget: 2000,
		Preview: func(r ChargenResult) ChargenPreview { seen = append(seen, r.Choices[2]); return ChargenPreview{} },
	}
	c := NewChargen(setup)
	c.Forward()
	a := newTestApp(t, appRows(1), okLoader(t))
	if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, nil }); err != nil {
		t.Fatal(err)
	}
	for selected := 0; selected < 5; selected++ {
		at := detailedSkillOrigin[0][selected].Add(chargenColumnOffset).Add(image.Pt(2, 2))
		a.step(appInput{CursorX: at.X, CursorY: at.Y, PrimaryPressed: true}, now.Add(time.Duration(selected)*time.Millisecond))
		a.step(appInput{CursorX: at.X, CursorY: at.Y, PrimaryReleased: true}, now.Add(time.Duration(selected)*time.Millisecond))
		frame := composeChargenPage(c, chargenNone, chargenNone)
		const background = 18 // composeChargenDetailedPage's own fill color; this fixture sets no ColumnFill, so a rest corner's un-drawn pixel stays at this value
		for skill := 0; skill < 5; skill++ {
			p := detailedSkillOrigin[0][skill].Add(chargenColumnOffset).Add(image.Pt(2, 2))
			if skill == selected { // selected, not hovered: state 0 (on.bmp, pressed dark)
				want := uint8(40 + 10*skill)
				if got := frame.RGBAAt(p.X, p.Y).R; got != want {
					t.Fatalf("after click %d, skill %d (selected) state pixel=%d want %d", selected, skill, got, want)
				}
				continue
			}
			if got := frame.RGBAAt(p.X, p.Y).R; got != background { // rest: not selected, not hovered -> no patch drawn
				t.Fatalf("after click %d, skill %d (rest) state pixel=%d want background %d (no patch)", selected, skill, got, background)
			}
		}
	}
	if len(seen) != 5 { // Forward plus each actual sword→axe→… selection change.
		t.Fatalf("preview calls=%v, want Forward then four changed skills", seen)
	}
	for skill := 1; skill < 5; skill++ {
		if seen[skill] != skill {
			t.Fatalf("preview %d selected skill=%d, want %d", skill, seen[skill], skill)
		}
	}
}

func TestDetailedPlayUsesSourceRefusalsAndLaunchesOnce(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	setup := chargenLegalSetup()
	setup.Name = ""
	setup.PreCreate = &ChargenPreCreate{Art: &ChargenPresentation{Layout: testGenerator()}}
	setup.Detailed = &ChargenDetailed{EmptyName: "EMPTY", ReservedName: "RESERVED", Play: "PLAY"}
	c := NewChargen(setup)
	c.Forward()
	a := newTestApp(t, appRows(1), okLoader(t))
	called := 0
	if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) {
		called++
		return okOpener(t), nil
	}); err != nil {
		t.Fatal(err)
	}
	play := detailedControlRect(c, chargenPlay).Min
	a.step(appInput{CursorX: play.X, CursorY: play.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: play.X, CursorY: play.Y, PrimaryReleased: true}, now)
	if a.flow.msg != "EMPTY" || called != 0 {
		t.Fatalf("empty play = (%q, %d calls)", a.flow.msg, called)
	}
	c.EditName("Self", false)
	a.step(appInput{CursorX: play.X, CursorY: play.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: play.X, CursorY: play.Y, PrimaryReleased: true}, now)
	if a.flow.msg != "RESERVED" || called != 0 {
		t.Fatalf("reserved play = (%q, %d calls)", a.flow.msg, called)
	}
	for range "Self" {
		c.EditName("", true)
	}
	c.EditName("Hero", false)
	a.step(appInput{CursorX: play.X, CursorY: play.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: play.X, CursorY: play.Y, PrimaryReleased: true}, now)
	if called != 1 || a.Screen() != ScreenMap {
		t.Fatalf("valid play = (%d calls, screen %v)", called, a.Screen())
	}
}

func TestDetailedHoverIsTransientAndAcceptedEditsClearRefusals(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	art := &ChargenPresentation{Layout: testGenerator()}
	for skill := range art.Skills[0] {
		art.Skills[0][skill][0] = image.NewRGBA(image.Rect(0, 0, 300, 80))
	}
	setup := chargenLegalSetup()
	setup.PreCreate = &ChargenPreCreate{Art: art}
	setup.Detailed = &ChargenDetailed{SkillHover: [2][5]string{{"BLADE HOVER"}}}
	setup.Choices = []ChargenChoice{
		{Options: []string{"male", "female"}, Parent: -1},
		{Options: []string{"fighter", "mage"}, Parent: -1},
		{Options: []string{"blade", "axe", "bludgeon", "pike", "shooting"}, Parent: -1},
	}
	c := NewChargen(setup)
	c.Forward()
	a := newTestApp(t, appRows(1), okLoader(t))
	if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, nil }); err != nil {
		t.Fatal(err)
	}

	a.flow.msg = "DURABLE REFUSAL"
	skill := detailedControlRect(c, chargenSkill0).Min
	a.step(appInput{CursorX: skill.X, CursorY: skill.Y}, now)
	if a.flow.msg != "DURABLE REFUSAL" || a.chargenDetailMessage() != "DURABLE REFUSAL" {
		t.Fatalf("hover overwrote/refused message = durable %q shown %q", a.flow.msg, a.chargenDetailMessage())
	}
	a.step(appInput{CursorX: 0, CursorY: 0}, now)
	if a.chargenHoverText != "" || a.chargenDetailMessage() != "DURABLE REFUSAL" {
		t.Fatalf("leaving skill = hover %q shown %q, want durable refusal restored", a.chargenHoverText, a.chargenDetailMessage())
	}

	plus := detailedControlRect(c, chargenStatPlus0).Min
	a.step(appInput{CursorX: plus.X, CursorY: plus.Y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: plus.X, CursorY: plus.Y, PrimaryReleased: true}, now)
	if a.flow.msg != "" {
		t.Fatalf("accepted stat edit retained durable refusal %q", a.flow.msg)
	}
}

// TestChargenGate is 0140's own seam: a picker row the gate claims opens the
// generation screen instead of its map, and the map behind it is opened later,
// out of the confirmed character.
//
// Every case is driven through a.step with synthetic input, exactly as
// TestChargenAppDispatch above is — no engine, no window, no install.
func TestChargenGate(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)

	// gateOn returns a gate that claims exactly the rows named and records, in
	// the slice it returns, every row index it was asked about. The record is
	// what witnesses that the gate is consulted per row rather than once.
	gateOn := func(begin func(ChargenResult) (MapOpener, error), rows ...int) (ChargenGate, *[]int) {
		asked := new([]int)
		claim := make(map[int]bool, len(rows))
		for _, r := range rows {
			claim[r] = true
		}
		return func(row int) *ChargenEntry {
			*asked = append(*asked, row)
			if !claim[row] {
				return nil
			}
			return &ChargenEntry{Model: NewChargen(chargenLegalSetup()), Begin: begin}
		}, asked
	}

	t.Run("a claimed row arms generation and never reaches the loader", func(t *testing.T) {
		loaded := 0
		load := func(i int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
			loaded++
			return okLoader(t)(i)
		}
		a := newTestApp(t, appRows(3), load)
		gate, asked := gateOn(func(ChargenResult) (MapOpener, error) { return nil, nil }, 1)
		a.SetChargenGate(gate)

		a.flow.screen = ScreenPicker
		a.step(appInput{Down: true}, now) // select row 1
		a.step(appInput{Enter: true}, now)

		if a.Screen() != ScreenChargen {
			t.Fatalf("Screen() = %v after choosing a claimed row, want ScreenChargen", a.Screen())
		}
		if loaded != 0 {
			t.Errorf("the loader ran %d times for a claimed row, want 0 — the map behind it "+
				"depends on a character that does not exist yet", loaded)
		}
		if a.flow.viewer != nil {
			t.Error("choosing a claimed row opened a viewer")
		}
		if len(*asked) != 1 || (*asked)[0] != 1 {
			t.Errorf("the gate was asked about %v, want exactly [1] — the row that was chosen", *asked)
		}
	})

	t.Run("an unclaimed row loads and shows its map exactly as it always has", func(t *testing.T) {
		loaded := 0
		load := func(i int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
			loaded++
			return okLoader(t)(i)
		}
		a := newTestApp(t, appRows(3), load)
		gate, _ := gateOn(func(ChargenResult) (MapOpener, error) { return nil, nil }, 2)
		a.SetChargenGate(gate)

		a.flow.screen = ScreenPicker
		a.step(appInput{Enter: true}, now) // row 0, which the gate does not claim

		if a.Screen() != ScreenMap {
			t.Fatalf("Screen() = %v after choosing an unclaimed row, want ScreenMap", a.Screen())
		}
		if loaded != 1 {
			t.Errorf("the loader ran %d times for an unclaimed row, want 1", loaded)
		}
	})

	t.Run("no gate is the picker this package has always had", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		a.flow.screen = ScreenPicker
		a.step(appInput{Enter: true}, now)
		if a.Screen() != ScreenMap {
			t.Errorf("Screen() = %v with no gate set, want ScreenMap", a.Screen())
		}
	})

	t.Run("Escape from a gated row returns to the picker, with the row still selected", func(t *testing.T) {
		a := newTestApp(t, appRows(4), okLoader(t))
		gate, _ := gateOn(func(ChargenResult) (MapOpener, error) { return nil, nil }, 2)
		a.SetChargenGate(gate)

		a.flow.screen = ScreenPicker
		a.step(appInput{Down: true}, now)
		a.step(appInput{Down: true}, now) // row 2
		a.step(appInput{Enter: true}, now)
		if a.Screen() != ScreenChargen {
			t.Fatalf("setup: Screen() = %v, want ScreenChargen", a.Screen())
		}

		if exit := a.step(appInput{Escape: true}, now); exit {
			t.Fatalf("Escape on a gated generation screen exited the program")
		}
		if a.Screen() != ScreenPicker {
			t.Fatalf("Screen() = %v after Escape, want ScreenPicker — backing out of generation "+
				"must not unwind the map list too", a.Screen())
		}
		if got := a.flow.picker.Selection(); got != 2 {
			t.Errorf("Selection() = %d after Escape, want 2 — the player's place in the list is kept", got)
		}
	})

	t.Run("a legal confirm on a gated row enters the map the begin callback opens", func(t *testing.T) {
		called := 0
		begin := func(ChargenResult) (MapOpener, error) {
			called++
			return okOpener(t), nil
		}
		a := newTestApp(t, appRows(3), okLoader(t))
		gate, _ := gateOn(begin, 0)
		a.SetChargenGate(gate)

		a.flow.screen = ScreenPicker
		a.step(appInput{Enter: true}, now) // choose row 0: arms generation
		if a.Screen() != ScreenChargen {
			t.Fatalf("setup: Screen() = %v, want ScreenChargen", a.Screen())
		}
		a.step(appInput{Enter: true}, now) // confirm the always-legal spread

		if called != 1 {
			t.Fatalf("begin ran %d times, want 1", called)
		}
		if a.Screen() != ScreenMap {
			t.Fatalf("Screen() = %v after a legal confirm, want ScreenMap", a.Screen())
		}
		if a.flow.viewer == nil {
			t.Error("a legal confirm entered no viewer")
		}
	})

	t.Run("an entry with no model is refused on the picker and leaves the row choosable", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		a.SetChargenGate(func(int) *ChargenEntry { return &ChargenEntry{} })

		a.flow.screen = ScreenPicker
		a.step(appInput{Enter: true}, now)

		if a.Screen() != ScreenPicker {
			t.Fatalf("Screen() = %v after an entry with no model, want ScreenPicker", a.Screen())
		}
		if a.flow.msg == "" {
			t.Error("a refused entry left no message; the player is owed a reason")
		}
		rows := a.flow.picker.Rows()
		if !rows[0].Choosable {
			t.Error("a refused entry marked the row unusable — an entry the wiring tier could not " +
				"build says nothing about whether the map behind it would decode")
		}
	})
}

// TestChargenDerivedFitIsMeasuredAgainstTheLayout is 0140's own arithmetic,
// asserted rather than written in a sentence. The last line the budget permits
// must end at or above the message line, and one more must not — which is
// exactly the claim that was got wrong before this story ("six lines is what
// fits under the footer"; it is sixteen) and kept four values off the screen.
//
// It is a statement about WHERE lines land, not about what they say. The
// painted pixels are outside any test in this package.
func TestChargenDerivedFitIsMeasuredAgainstTheLayout(t *testing.T) {
	// The shipped screen's row count: a sex, a class, a skill and the four
	// statistics. It is written here rather than read from the wiring tier,
	// which this package may not import — the property below holds for every
	// row count, and this is the one the game actually paints.
	const shippedRows = 7

	if got := chargenDerivedFit(shippedRows); got != 16 {
		t.Errorf("chargenDerivedFit(%d) = %d, want 16 — the block the shipped screen can hold",
			shippedRows, got)
	}

	// The property, over every row count a screen could plausibly have: the
	// last permitted line ends at or above the message row, and one more line
	// would not.
	for rows := 0; rows <= 20; rows++ {
		fit := chargenDerivedFit(rows)
		if fit < 0 {
			t.Fatalf("chargenDerivedFit(%d) = %d, want no negative budget", rows, fit)
		}
		lineY := func(k int) int { return chargenTop + (rows+chargenDerivedGap+k)*chargenLine }
		if fit > 0 {
			if bottom := lineY(fit-1) + chargenLine; bottom > chargenMessageY {
				t.Errorf("rows=%d: the last permitted block line ends at y=%d, past the message "+
					"line at %d", rows, bottom, chargenMessageY)
			}
		}
		if bottom := lineY(fit) + chargenLine; bottom <= chargenMessageY {
			t.Errorf("rows=%d: one more line would still end at y=%d, clear of the message line "+
				"at %d — the budget is short by at least one", rows, bottom, chargenMessageY)
		}
	}
}

// TestChargenBlockIsClippedToWhatFits drives the whole draw path with a block
// longer than the screen can hold. It witnesses that drawing does not panic and
// that the model still reports every line — the clip is the draw path's, not the
// model's, so a wiring tier that hands over too many is told nothing and the
// block simply ends where the frame does.
//
// WHICH LINES REACHED THE CANVAS IS NOT ASSERTED. That is pixels, and this
// package has no way to read them back.
func TestChargenBlockIsClippedToWhatFits(t *testing.T) {
	setup := chargenLegalSetup()
	const over = 40
	setup.Derive = func(ChargenResult) []ChargenDerived {
		out := make([]ChargenDerived, over)
		for i := range out {
			out[i] = ChargenDerived{Name: fmt.Sprintf("row%d", i), Value: "1"}
		}
		return out
	}

	c := NewChargen(setup)
	if got := len(c.DerivedText()); got != over+1 {
		t.Fatalf("DerivedText() = %d lines, want %d — the model reports what it is handed", got, over+1)
	}

	a := newTestApp(t, appRows(3), okLoader(t))
	if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, nil }); err != nil {
		t.Fatalf("OpenChargen: %v", err)
	}
	a.Draw(ebiten.NewImage(frame.W, frame.H))
}

func TestCampaignAdvanceNeverArmsGeneration(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, seam, ended := advanceApp(t, 1600, 900)

	asked := 0
	a.SetChargenGate(func(int) *ChargenEntry {
		asked++
		return &ChargenEntry{
			Model: NewChargen(chargenLegalSetup()),
			Begin: func(ChargenResult) (MapOpener, error) { return nil, nil },
		}
	})

	o := &successorOpener{startCell: image.Pt(100, 90)}
	seam.dest, seam.msg, seam.open = NoticeToMission, "mission won", o.open

	a.step(appInput{Enter: true}, now)

	if a.Screen() == ScreenChargen {
		t.Fatal("winning a mission opened the generation screen; the successor must take the " +
			"carried character, not a newly generated one (owner, 2026-08-10)")
	}
	if a.Screen() != ScreenMap {
		t.Fatalf("screen = %v after a won mission's advance, want ScreenMap", a.Screen())
	}
	if a.flow.viewer != o.v {
		t.Error("the screen showing is not the successor opener's own viewer")
	}
	if a.flow.viewer == ended {
		t.Error("the ended mission's viewer is still showing")
	}
	if asked != 0 {
		t.Errorf("the chargen gate was consulted %d times on a campaign transition, want 0 — "+
			"the gate belongs to choosing a row, and a successor is not chosen", asked)
	}
	if a.flow.chargen != nil {
		t.Error("a campaign transition armed a generation model")
	}
}
