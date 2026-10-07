package ui

import (
	"fmt"
	"time"
)

// headlessNow is the timestamp the first headless dispatch on an App carries.
// It used to be the ONLY timestamp every headless dispatch carried, unvarying
// call to call: the sole reader was the pre-create double-click gate, which
// keyboard activation does not reach, so a frozen instant cost nothing.
// `worldMapTickInterval` (`DIV-136`, app.go) is the first headless-reachable
// gate that needs elapsed time to make progress at all rather than only to
// debounce a double click within one still instant, so headlessAt below now
// advances a clock private to each App instead of handing out this same
// value forever.
var headlessNow = time.Unix(0, 0)

// headlessClockStep is how far headlessAt advances App's own headless clock
// on every call. It stays well under every debounce window this package
// gates on (townSurfaceAt's 350ms, chargenDoubleClickWindow's 500ms) for any
// ordinarily-sized run of consecutive headless dispatches between two
// related presses, while crossing worldMapTickInterval's 100ms floor after a
// handful of calls, so a scenario drives WorldMapTick to completion with
// ordinary wait_ticks steps instead of needing one call per elapsed
// millisecond.
const headlessClockStep = 20 * time.Millisecond

// headlessAt advances the deterministic input clock. Movies use elapsed wall
// time, matching HeadlessCutsceneStep and the decoder's stall watchdog.
func (a *App) headlessAt() time.Time {
	if a.cutscene != nil {
		return time.Now()
	}
	if a.headlessClock.IsZero() {
		a.headlessClock = headlessNow
	}
	t := a.headlessClock
	a.headlessClock = a.headlessClock.Add(headlessClockStep)
	return t
}

// The kinds a HeadlessChargenControl reports. They name what the control
// does, not where it is drawn: a scenario reaches a control by the focus
// index this package hands it and never by a coordinate.
const (
	ChargenControlName       = "name"
	ChargenControlPicture    = "picture"
	ChargenControlDifficulty = "difficulty"
	ChargenControlBack       = "back"
	ChargenControlForward    = "forward"
	ChargenControlSkill      = "skill"
	ChargenControlStatDown   = "stat_down"
	ChargenControlStatUp     = "stat_up"
	ChargenControlReset      = "reset"
	ChargenControlPlay       = "play"
)

// The two stage words a HeadlessChargen reports. They are the scenario
// vocabulary for ChargenStage, which is an unexported-in-effect integer as far
// as a JSON event is concerned.
const (
	ChargenStagePreCreate = "precreate"
	ChargenStageDetailed  = "detailed"
)

// HeadlessChargenControl is one control of the generation screen, with the
// focus index that reaches it.
//
// Label is the screen's OWN wording, taken from the setup the wiring tier
// supplied: a picture's label is its sex and class option labels joined by a
// space, a skill's is the skill option label for the current class, and a
// statistic's is the statistic's own name. This package authors none of
// them, so a scenario that names a label is naming what the screen shows
// rather than a second vocabulary maintained here.
type HeadlessChargenControl struct {
	Focus  int    `json:"focus"`
	Kind   string `json:"kind"`
	Label  string `json:"label,omitempty"`
	Value  int    `json:"value,omitempty"`
	Chosen bool   `json:"chosen,omitempty"`
}

// HeadlessChargen is the generation screen as a scenario sees it: which page is
// showing, where the focus is, what the name field holds, what the budget has
// left, whether the spread is one the screen would confirm, and every control
// the page offers.
//
// Only the CURRENT page's controls are listed. The two pages have separate focus
// orders and a control on the other one cannot be reached without changing page,
// so listing both would hand a caller focus indices that address nothing.
type HeadlessChargen struct {
	Difficulty int                      `json:"difficulty"`
	Stage      string                   `json:"stage"`
	Focus      int                      `json:"focus"`
	Name       string                   `json:"name"`
	Remaining  int                      `json:"remaining"`
	Legal      bool                     `json:"legal"`
	Controls   []HeadlessChargenControl `json:"controls"`
}

// Control reports the one control of the given kind carrying the given label,
// and whether there is exactly one. An empty label matches a kind that has a
// single control (forward, play, back, reset, name).
func (h HeadlessChargen) Control(kind, label string) (HeadlessChargenControl, bool) {
	found, count := HeadlessChargenControl{}, 0
	for _, c := range h.Controls {
		if c.Kind != kind {
			continue
		}
		if label != "" && !equalFoldTrim(c.Label, label) {
			continue
		}
		found, count = c, count+1
	}
	return found, count == 1
}

// Labels reports every label offered under one kind, in focus order. It is what
// a refusal names when a scenario asks for an option no row offers.
func (h HeadlessChargen) Labels(kind string) []string {
	var out []string
	for _, c := range h.Controls {
		if c.Kind == kind {
			out = append(out, c.Label)
		}
	}
	return out
}

// HeadlessChargenState reports the generation screen. The second result is
// false on every other screen, so a caller cannot mistake a zero value for a
// screen that is showing nothing.
func (a *App) HeadlessChargenState() (HeadlessChargen, bool) {
	if a == nil || a.flow == nil || a.flow.screen != ScreenChargen || a.flow.chargen == nil {
		return HeadlessChargen{}, false
	}
	c := a.flow.chargen
	state := HeadlessChargen{
		Difficulty: c.Difficulty(),
		Focus:      c.Focus(),
		Name:       c.NameText(),
		Remaining:  c.Remaining(),
		Legal:      c.Legal(),
	}
	if c.Stage() == PreCreateStage {
		state.Stage = ChargenStagePreCreate
		state.Controls = c.preCreateControls()
		return state, true
	}
	state.Stage = ChargenStageDetailed
	state.Controls = c.detailedControls()
	return state, true
}

// preCreateControls lists the pre-create page in its own focus order, which is
// preFocusControl's: the name field, the four combined pictures, Back, Forward.
//
// A PICTURE'S LABEL COMES FROM preChoiceParts, the same helper Forward
// itself reads. The identity a scenario matched against and the identity the
// model commits on Forward are therefore one statement read twice rather
// than two rules that can drift apart.
func (c *Chargen) preCreateControls() []HeadlessChargenControl {
	out := []HeadlessChargenControl{{Focus: 0, Kind: ChargenControlName, Label: c.name}}
	for p := 0; p < 4; p++ {
		sex, class := preChoiceParts(p)
		out = append(out, HeadlessChargenControl{
			Focus: 1 + p, Kind: ChargenControlPicture,
			Label:  c.optionLabel(0, sex) + " " + c.optionLabel(1, class),
			Chosen: c.preChoice == p,
		})
	}
	out = append(out,
		HeadlessChargenControl{Focus: 5, Kind: ChargenControlBack},
		HeadlessChargenControl{Focus: 6, Kind: ChargenControlForward})
	for i, label := range []string{"easy", "normal", "hard"} {
		out = append(out, HeadlessChargenControl{Focus: 7 + i, Kind: ChargenControlDifficulty,
			Label: label, Value: i + 1, Chosen: c.preLevel == i})
	}
	return out
}

// optionLabel is one choice row's option label, or a placeholder naming the
// index when the setup does not offer it. A malformed setup must not panic
// here for chargen.go's own stated reason, and a label nobody can read is
// better than a label that lies about which option it is.
func (c *Chargen) optionLabel(row, option int) string {
	if row < 0 || row >= len(c.setup.Choices) {
		return fmt.Sprintf("#%d", option)
	}
	opts := c.choiceOptions(row)
	if option < 0 || option >= len(opts) {
		return fmt.Sprintf("#%d", option)
	}
	return opts[option]
}

// detailedControls lists the detailed page in detailedFocus's own order: the
// five skill positions, then a down and an up control per statistic, then Back,
// Reset and Play.
//
// A CONTROL WHOSE ROW THE SETUP DOES NOT DECLARE IS OMITTED rather than
// reported with an empty label. detailedFocus is a fixed sixteen entries and a
// hand-built setup may declare fewer skills or fewer statistics; a focus index
// that reaches nothing is one a scenario could navigate to and press with no
// effect, which is exactly the state this surface exists to make impossible.
func (c *Chargen) detailedControls() []HeadlessChargenControl {
	var out []HeadlessChargenControl
	var skills []string
	if len(c.setup.Choices) > 2 {
		skills = c.choiceOptions(2)
	}
	for i := 0; i < 5; i++ {
		if i >= len(skills) {
			continue
		}
		chosen := len(c.choiceIndex) > 2 && c.choiceIndex[2] == i
		out = append(out, HeadlessChargenControl{
			Focus: i, Kind: ChargenControlSkill, Label: skills[i], Chosen: chosen,
		})
	}
	for i := 0; i < 4; i++ {
		if i >= len(c.setup.Stats) {
			continue
		}
		name, value := c.setup.Stats[i].Name, c.statValue[i]
		out = append(out,
			HeadlessChargenControl{Focus: 5 + 2*i, Kind: ChargenControlStatDown, Label: name, Value: value},
			HeadlessChargenControl{Focus: 6 + 2*i, Kind: ChargenControlStatUp, Label: name, Value: value})
	}
	out = append(out,
		HeadlessChargenControl{Focus: 13, Kind: ChargenControlBack},
		HeadlessChargenControl{Focus: 14, Kind: ChargenControlReset},
		HeadlessChargenControl{Focus: 15, Kind: ChargenControlPlay})
	return out
}

// HeadlessType dispatches typed characters and backspace edges through App.step,
// which is the same statement a windowed frame runs. Originally the name field's
// own input; since 1014 (MENU-KEY-013), stepGameMenu also reads Typed on
// ScreenGameMenu, to match a typed rune against a folded accelerator byte, so
// this call is screen-agnostic rather than chargen-specific.
//
// EACH CHARACTER IS ONE STEP. A player types one character per frame, and each
// one the name field takes restarts its caret; delivering a whole string in a
// single frame would exercise a path no keyboard produces.
func (a *App) HeadlessType(text string, backspace bool) error {
	if a == nil || a.flow == nil {
		return fmt.Errorf("headless type: nil application")
	}
	if backspace {
		if a.step(appInput{Backspace: true, Unfocused: a.headlessUnfocused}, a.headlessAt()) {
			return fmt.Errorf("headless backspace requested application exit")
		}
	}
	for _, r := range text {
		if a.step(appInput{Typed: string(r), Unfocused: a.headlessUnfocused}, a.headlessAt()) {
			return fmt.Errorf("headless type %q requested application exit", text)
		}
	}
	return nil
}

// HeadlessNoticeOpen reports whether a notice is open over the map. It is the
// signal that a mission has reached a verdict and is waiting to be dismissed.
func (a *App) HeadlessNoticeOpen() bool {
	return a != nil && a.flow != nil && a.flow.noticeOpen()
}

func equalFoldTrim(a, b string) bool {
	a, b = trimASCIISpace(a), trimASCIISpace(b)
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if foldASCII(a[i]) != foldASCII(b[i]) {
			return false
		}
	}
	return true
}

func trimASCIISpace(s string) string {
	for len(s) > 0 && isHeadlessSpace(s[0]) {
		s = s[1:]
	}
	for len(s) > 0 && isHeadlessSpace(s[len(s)-1]) {
		s = s[:len(s)-1]
	}
	return s
}

func foldASCII(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}
