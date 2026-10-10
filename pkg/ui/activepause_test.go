package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestActivePauseRetainsAnimationPhaseAcrossIdleAndResume(t *testing.T) {
	for _, unpaced := range []bool{false, true} {
		t.Run(fmt.Sprint(unpaced), func(t *testing.T) {
			h := newHaltFix(t, haltOpts{})
			h.a.flow.unpaced = unpaced
			h.a.flow.syncCadence(false)
			h.frame(haltNeutral())
			count, phase := h.v.anim.Count(), h.v.anim.Remainder()
			h.frame(appInput{Pause: true})
			h.run(100, haltNeutral())
			if h.v.anim.Count() != count || h.v.anim.Remainder() != phase {
				t.Fatalf("paused presentation advanced from count%d phase%d to count%d phase%d", count, phase, h.v.anim.Count(), h.v.anim.Remainder())
			}
			h.frame(appInput{Pause: true})
			if h.v.anim.Count() != count || h.v.anim.Remainder() != phase {
				t.Fatal("first resume frame moved retained animation phase")
			}
			h.frame(haltNeutral())
			if h.v.anim.Count() == count {
				t.Fatal("resumed presentation never advances")
			}
		})
	}
}

func TestActivePauseRetainsNumeralFlightAndRemainingLife(t *testing.T) {
	for _, owner := range []uint32{1, 2} {
		v, now := clockViewer(t, 1, owner)
		push(v, now.Add(250*time.Millisecond), numeralUnit(1, owner, 33))
		wantOff, wantAge := v.numerals[0].off, 250*time.Millisecond
		v.setPlayerPaused(true)
		for _, elapsed := range []time.Duration{time.Second, time.Minute, time.Hour} {
			now = at0.Add(elapsed)
			push(v, now, numeralUnit(1, owner, 33))
			if len(v.numerals) != 1 {
				t.Fatalf("owner%d: numeral expired during pause at %v", owner, elapsed)
			}
			if v.numerals[0].off != wantOff || now.Sub(v.numerals[0].born) != wantAge {
				t.Fatal("paused numeral moved or spent remaining life", v.numerals[0].off, now.Sub(v.numerals[0].born))
			}
		}
		v.setPlayerPaused(false)
		now = now.Add(time.Minute)
		push(v, now, numeralUnit(1, owner, 33))
		if len(v.numerals) != 1 || v.numerals[0].off != wantOff || now.Sub(v.numerals[0].born) != wantAge {
			t.Fatal("resume repaid an unobserved paused interval")
		}
		push(v, now.Add(750*time.Millisecond), numeralUnit(1, owner, 33))
		if len(v.numerals) != 1 {
			t.Fatal("remaining lifetime lost its inclusive boundary")
		}
		push(v, now.Add(750*time.Millisecond+time.Microsecond), numeralUnit(1, owner, 33))
		if len(v.numerals) != 0 {
			t.Fatal("resumed numeral lifetime never expires")
		}
	}
}

func TestActivePauseKeepsSoundDeliveryAndEntityRetirementLive(t *testing.T) {
	v := numeralViewer(t)
	rec := &recordingPlayer{}
	v.SetAudio(rec, soundBankFor())
	v.SetSpeechAudio(rec)
	u := soundUnit(1, 1, 100)
	push(v, at0, u)
	v.setPlayerPaused(true)
	push(v, at0.Add(time.Hour), blow(v, &u, 60))
	if len(rec.plays) != 1 || rec.plays[0].Sample.PCM[0] != soundHighSlot {
		t.Fatal("player pause skipped pending sound delivery", heard(rec))
	}
	push(v, at0.Add(2*time.Hour))
	if len(v.voices) != 0 || len(v.soundMessages) != 0 {
		t.Fatal("player pause retained absent sound entities or undrained messages")
	}
}

func TestActivePauseDimsWorldBeforeReadableHUDAndResumesExactly(t *testing.T) {
	h := newHaltFix(t, haltOpts{})
	h.v.SetTextSmoothing(false)
	h.v.ShowReadout(true)
	h.v.DeferPointer(true)
	dst := ebiten.NewImage(1024, 768)
	t.Cleanup(dst.Dispose)
	wash := color.RGBA{A: 32}
	washed := func() bool {
		for _, op := range h.v.canvasLog.ops {
			if op.kind == pixelOver && op.pic == nil && op.layer == nil && op.solid == wash && op.rect == image.Rect(0, 0, h.v.cam.ViewW, h.v.cam.ViewH) {
				return true
			}
		}
		return false
	}
	h.v.Draw(dst)
	if washed() {
		t.Fatal("running world is dimmed")
	}
	h.frame(appInput{Pause: true})
	h.v.Draw(dst)
	if !washed() {
		t.Fatal("paused world has no subtle dim submission")
	}
	pic, at, ok := h.v.messagePresent()
	if !ok {
		t.Fatal("paused HUD absent")
	}
	for y := 0; y < pic.Rect.Dy(); y++ {
		for x := 0; x < pic.Rect.Dx(); x++ {
			c := pic.RGBAAt(x, y)
			if c.A != 255 {
				continue
			}
			got, certain := h.v.canvasLog.value(len(h.v.canvasLog.ops), at.X+x, at.Y+y)
			if !certain || got != c {
				t.Fatal("pause tint reached opaque HUD glyph", got, c)
			}
		}
	}
	h.v.SetTextSmoothing(true)
	h.v.Draw(dst)
	if h.v.textKept == 0 || h.v.textSettleFallbacks != 0 {
		t.Fatal("paused HUD smoothing lost readable glyphs", h.v.textKept, h.v.textSettleFallbacks)
	}
	want := slices.Clone(pic.Pix)
	h.run(100, haltNeutral())
	h.v.Draw(dst)
	if got, _, _ := h.v.messagePresent(); !slices.Equal(want, got.Pix) {
		t.Fatal("held dim changed the persistent pause HUD")
	}
	h.frame(appInput{Pause: true})
	h.v.Draw(dst)
	if washed() {
		t.Fatal("resume retained the world dim")
	}
}

func TestActivePauseNativeBindingIsBareZeroAndSpaceIsPanels(t *testing.T) {
	bindings := bindingSource(t)
	panels, pause, group := bindings["Panels"], bindings["Pause"], bindings["GroupKey"]
	if !strings.Contains(panels, "inpututil.IsKeyJustPressed(ebiten.KeySpace)") || !strings.Contains(panels, "!altHeld()") || !strings.Contains(panels, "!ebiten.IsKeyPressed(ebiten.KeyShiftLeft)") || !strings.Contains(panels, "!ebiten.IsKeyPressed(ebiten.KeyShiftRight)") {
		t.Fatal("physical Space binding is not the panel-set toggle", panels)
	}
	if pause != "barePauseKey()" || !strings.Contains(group, "!barePauseKey()") {
		t.Fatal("pause is not the one bare-zero reading shared with the group key", pause, group)
	}
	src, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	body = body[strings.Index(body, "func barePauseKey()"):]
	body = body[:strings.Index(body, "\n}\n")]
	for _, want := range []string{"ebiten.KeyDigit0", "ebiten.KeyNumpad0", "!ctrlHeld()", "!altHeld()", "ebiten.KeyShiftLeft", "ebiten.KeyShiftRight"} {
		if !strings.Contains(body, want) {
			t.Fatalf("bare zero pause binding lacks %s", want)
		}
	}
	if strings.Contains(body, "KeySpace") {
		t.Fatal("Space still pauses")
	}
}

func TestActivePauseZeroUsesPlayerStopAndSpaceDoesNot(t *testing.T) {
	h := newHaltFix(t, haltOpts{})
	if err := h.a.HeadlessKey("space"); err != nil {
		t.Fatal(err)
	}
	if h.a.flow.stopped {
		t.Fatal("Space paused the world")
	}
	pack, book := h.v.hudShown(hudPanelPack), h.v.hudShown(hudPanelBook)
	if err := h.a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	if !h.a.flow.stopped || h.v.hudShown(hudPanelPack) != pack || h.v.hudShown(hudPanelBook) != book {
		t.Fatal("zero changed panels instead of setting player pause")
	}
	before := h.w.world
	h.run(100, haltNeutral())
	if h.w.world != before {
		t.Fatal("paused idle advanced the world")
	}
	if err := h.a.HeadlessKey("0"); err != nil || h.a.flow.stopped {
		t.Fatal("zero did not resume", err)
	}
}

func TestActivePauseKeepsExactResumeAndOptionsCancel(t *testing.T) {
	for _, rung := range []int{0, 1, 2, 7, 12, 16} {
		for _, unpaced := range []bool{false, true} {
			t.Run(fmt.Sprintf("rung%d-unpaced%v", rung, unpaced), func(t *testing.T) {
				values := GameOptionValues{}
				var log []optionWrite
				a := optionsApp(t, &values, &log, nil)
				a.HeadlessKey("escape")
				a.HeadlessGameMenuAction("return")
				a.flow.rung, a.flow.unpaced = rung, unpaced
				a.flow.syncCadence(false)
				a.HeadlessKey("0")
				if !a.flow.stopped || !a.flow.viewer.SaveApplication().PlayerPaused {
					t.Fatal("pause intent was not mirrored for SAVE")
				}
				a.HeadlessKey("escape")
				a.HeadlessGameMenuAction("game-options")
				a.flow.setDraftSpeed(2)
				a.HeadlessGameMenuAction("options-cancel")
				a.HeadlessGameMenuAction("return")
				if a.flow.rung != rung || a.flow.unpaced != unpaced || !a.flow.stopped {
					t.Fatal("Cancel changed exact paused resume mode")
				}
				a.HeadlessKey("0")
				if a.flow.stopped || a.flow.rung != rung || a.flow.unpaced != unpaced || a.flow.viewer.anim.Period() != terrain.CadencePeriod(rung) {
					t.Fatal("Zero normalized the resume cadence")
				}
				a.HeadlessKey("escape")
				a.HeadlessGameMenuAction("game-options")
				a.HeadlessGameMenuAction("page-return")
				if a.flow.rung != rung || a.flow.unpaced != unpaced || a.flow.stopped {
					t.Fatal("unchanged OK normalized a non-slider cadence")
				}
			})
		}
	}
}

func TestActivePausePositiveChoiceAndFailedCommit(t *testing.T) {
	for _, fail := range []bool{false, true} {
		values := GameOptionValues{}
		var log []optionWrite
		a := optionsApp(t, &values, &log, &fail)
		a.flow.rung, a.flow.unpaced, a.flow.stopped = 16, true, true
		a.flow.syncCadence(false)
		a.flow.setDraftSpeed(2)
		a.HeadlessGameMenuAction("page-return")
		if fail {
			a.HeadlessGameMenuAction("options-cancel")
			if a.flow.rung != 16 || !a.flow.unpaced || !a.flow.stopped {
				t.Fatal("failed OK applied positive cadence")
			}
		} else if a.flow.rung != 4 || a.flow.unpaced || a.flow.stopped {
			t.Fatal("positive choice did not resume its exact rung")
		}
	}
	values := GameOptionValues{}
	var log []optionWrite
	fail := true
	a := optionsApp(t, &values, &log, &fail)
	a.flow.setDraftSpeed(0)
	a.HeadlessGameMenuAction("page-return")
	a.HeadlessGameMenuAction("options-cancel")
	if a.flow.stopped {
		t.Fatal("failed zero OK paused the mission")
	}
}

func TestActivePauseFocusAndDismissalOwnTheFrame(t *testing.T) {
	h := newHaltFix(t, haltOpts{})
	h.frame(appInput{Pause: true, Unfocused: true})
	if h.a.flow.stopped {
		t.Fatal("unfocused Zero changed pause intent")
	}
	h.v.SetNotice("dialogue", NoticeDialogue)
	h.frame(appInput{Enter: true, Pause: true})
	if h.v.NoticeOpen() || h.a.flow.stopped || h.v.playerPaused {
		t.Fatal("dismissal leaked Zero into player pause")
	}
	h.frame(appInput{Pause: true})
	h.v.SetNotice("dialogue", NoticeDialogue)
	h.frame(appInput{Enter: true, Pause: true})
	if !h.a.flow.stopped || !h.v.playerPaused {
		t.Fatal("dismissal lost a preexisting player pause")
	}
	before := h.w.world
	h.run(100, haltNeutral())
	if h.w.world != before {
		t.Fatal("long player pause advanced the world")
	}
	h.frame(appInput{Pause: true})
	if h.w.world-before > 2 {
		t.Fatal("resume repaid wall time spent paused")
	}
}

func TestActivePauseTextAndOutsideMissionZero(t *testing.T) {
	s := &saveDialogSpy{}
	a := newSaveDialogApp(t, s, ScreenMap)
	a.HeadlessSaveEdit("saves", "two", SaveSAV)
	a.flow.saveDialog.setFocus(saveNameControl)
	a.step(appInput{Pause: true, Typed: " words"}, cadenceAt(0))
	a.HeadlessKey("0")
	if a.flow.stopped {
		t.Fatal("save name Zero paused the mission")
	}
	state, _ := a.HeadlessSaveState()
	if state.Request.Name != "two words" {
		t.Fatal("Zero changed typed name", state.Request.Name)
	}
	a = newTestApp(t, appRows(0), okLoader(t))
	c := NewChargen(ChargenSetup{Name: "Hero", PreCreate: &ChargenPreCreate{Art: &ChargenPresentation{Layout: testGenerator()}}})
	if err := a.OpenChargen(c, nil); err != nil {
		t.Fatal(err)
	}
	a.step(appInput{Pause: true, Typed: " name"}, cadenceAt(0))
	if c.NameText() != "Hero name" || a.flow.stopped {
		t.Fatal("native-shaped Zero changed hero name or pause intent", c.NameText())
	}
	values := GameOptionValues{}
	var log []optionWrite
	a = optionsApp(t, &values, &log, nil)
	a.flow.menuBack = ScreenMenu
	a.flow.setDraftSpeed(0)
	if a.flow.gameOptions.draft.speed != 1 {
		t.Fatal("outside-mission zero became a persistent pause preference")
	}
}

func TestActivePauseRestoredIntentAndFreshEntry(t *testing.T) {
	v := newHaltFix(t, haltOpts{}).v
	s := v.SaveApplication()
	s.PlayerPaused, s.Unpaced, s.PeriodUS = true, true, 35000
	if err := v.RestoreSaveApplication(s); err != nil {
		t.Fatal(err)
	}
	writes := 0
	f := flow{preferredRung: terrain.DefaultCadenceRung, preferredRungSet: true, persistRung: func(int) { writes++ }}
	f.enter(v, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if !f.stopped || !f.unpaced || !v.SaveApplication().PlayerPaused || v.anim.Period() != 35000 || writes != 0 {
		t.Fatal("prepared paused LOAD did not adopt saved intent without a profile write")
	}
	f.enter(v, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if f.stopped || f.unpaced || v.SaveApplication().PlayerPaused {
		t.Fatal("fresh entry inherited a consumed LOAD pause")
	}
}

func TestActivePauseHUDUsesThePersistentMessagePicture(t *testing.T) {
	h := newHaltFix(t, haltOpts{})
	h.a.HeadlessKey("0")
	paused, at, ok := h.v.messagePresent()
	if !ok || at.X != 8 || at.Y != 8 || paused.Bounds().Dx() > 80 {
		t.Fatal("paused HUD picture absent or outside its message origin")
	}
	want := bytes.Clone(paused.Pix)
	h.v.ClearMessages()
	h.run(100, haltNeutral())
	paused, _, ok = h.v.messagePresent()
	if !ok || !bytes.Equal(paused.Pix, want) {
		t.Fatal("message expiry/clear removed the player pause cue")
	}
	h.a.HeadlessKey("0")
	if _, _, ok := h.v.messagePresent(); ok {
		t.Fatal("resume retained the pause cue")
	}
}

func TestActivePauseZeroChoiceIsStaged(t *testing.T) {
	values := GameOptionValues{}
	var log []optionWrite
	a := optionsApp(t, &values, &log, nil)
	a.flow.setDraftSpeed(0)
	if a.flow.stopped {
		t.Fatal("draft stopped the map before OK")
	}
	if err := a.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	if !a.flow.stopped {
		t.Fatal("zero choice retained a positive running speed")
	}
	a.HeadlessGameMenuAction("return")
	v := a.flow.viewer
	count, phase := v.anim.Count(), v.anim.Remainder()
	for range 150 {
		a.HeadlessStep()
	}
	if !v.playerPaused || v.anim.Count() != count || v.anim.Remainder() != phase {
		t.Fatal("zero speed pause advanced presentation")
	}
}
