package game

import (
	"fmt"
	"strings"

	"againrom/pkg/ui"
)

// headlessCreateCharacter drives the generation screen to the requested
// character and presses Play.
//
// ORDER. Pre-create: the picture, then the name, then Forward. Detailed: the
// skill, then the statistics in the screen's own order, then Play. A picture
// press writes its own name over a default text (TEXT-074), so the name is
// typed after it. The skill precedes the statistics because Forward reseeds
// the skill row off the class it commits, so setting it first means nothing
// after it moves.
func headlessCreateCharacter(app *ui.App, want HeadlessCharacter) error {
	state, ok := app.HeadlessChargenState()
	if !ok {
		return fmt.Errorf("create_character requires the generation screen, got %s", app.Screen())
	}
	if state.Stage != ui.ChargenStagePreCreate {
		// The two-stage generator opens on the pre-create page. A screen that
		// opens on the detailed page is the legacy one-stage model, which no
		// scenario in this repository drives (0163 SC-3), and driving it would
		// mean guessing which page the choices belong to.
		return fmt.Errorf("create_character requires the two-stage generator; "+
			"this screen opened on the %s page", state.Stage)
	}
	if err := headlessChargenPicture(app, &state, want.Sex, want.Class); err != nil {
		return err
	}
	if err := headlessChargenName(app, &state, want.Name); err != nil {
		return err
	}
	if want.Difficulty != "" {
		if err := headlessChargenPress(app, &state, ui.ChargenControlDifficulty, want.Difficulty); err != nil {
			return err
		}
		if c, ok := state.Control(ui.ChargenControlDifficulty, want.Difficulty); !ok || !c.Chosen {
			return fmt.Errorf("create_character: difficulty %q did not take", want.Difficulty)
		}
	}
	if err := headlessChargenPress(app, &state, ui.ChargenControlForward, ""); err != nil {
		return err
	}
	if state.Stage != ui.ChargenStageDetailed {
		return fmt.Errorf("create_character: forward left the %s page showing: %s",
			state.Stage, headlessChargenWhy(app))
	}
	if want.Skill != "" {
		if err := headlessChargenPress(app, &state, ui.ChargenControlSkill, want.Skill); err != nil {
			return err
		}
		if c, ok := state.Control(ui.ChargenControlSkill, want.Skill); !ok || !c.Chosen {
			return fmt.Errorf("create_character: skill %q did not take: %s",
				want.Skill, headlessChargenWhy(app))
		}
	}
	if err := headlessChargenStats(app, &state, want.Stats); err != nil {
		return err
	}
	if err := headlessChargenPress(app, &state, ui.ChargenControlPlay, ""); err != nil {
		return err
	}
	if app.Screen() == ui.ScreenChargen {
		return fmt.Errorf("create_character: play left the generation screen showing: %s",
			headlessChargenWhy(app))
	}
	return nil
}

// headlessChargenName types the requested name one character at a time. The
// field only appends (TEXT-075), so the text it holds is first removed with
// one backspace per byte.
//
// A NAME THE SCREEN DID NOT TAKE IS AN ERROR AND NOT A WARNING. EditName drops
// a character the install's own encoder cannot store and stops at ten bytes, so
// a scenario asking for a name this install cannot hold would otherwise enter a
// mission under a name nobody wrote.
func headlessChargenName(app *ui.App, state *ui.HeadlessChargen, name string) error {
	if name == "" {
		return nil
	}
	if err := headlessChargenFocus(app, state, ui.ChargenControlName, ""); err != nil {
		return err
	}
	for range len(state.Name) {
		if err := app.HeadlessType("", true); err != nil {
			return err
		}
	}
	if err := app.HeadlessType(name, false); err != nil {
		return err
	}
	if err := headlessChargenRead(app, state); err != nil {
		return err
	}
	if state.Name != name {
		return fmt.Errorf("create_character: name %q was typed and the field holds %q",
			name, state.Name)
	}
	return nil
}

// headlessChargenPicture selects the one combined picture standing for the
// requested sex and class. A field the scenario did not name matches whatever
// the currently chosen picture holds for it, so naming a class alone keeps the
// sex the screen opened on.
func headlessChargenPicture(app *ui.App, state *ui.HeadlessChargen, sex, class string) error {
	if sex == "" && class == "" {
		return nil
	}
	current, _ := headlessChosen(*state, ui.ChargenControlPicture)
	parts := strings.Fields(current)
	if sex == "" && len(parts) > 0 {
		sex = parts[0]
	}
	if class == "" && len(parts) > 1 {
		class = parts[1]
	}
	label := sex + " " + class
	if _, ok := state.Control(ui.ChargenControlPicture, label); !ok {
		return fmt.Errorf("create_character: no picture is %q; the screen offers %s",
			strings.TrimSpace(label), headlessQuoted(state.Labels(ui.ChargenControlPicture)))
	}
	if err := headlessChargenPress(app, state, ui.ChargenControlPicture, label); err != nil {
		return err
	}
	if got, _ := headlessChosen(*state, ui.ChargenControlPicture); !strings.EqualFold(got, label) {
		return fmt.Errorf("create_character: picture %q did not take; the screen shows %q: %s",
			label, got, headlessChargenWhy(app))
	}
	return nil
}

// headlessChargenStats buys each named statistic up or down to its requested
// value by pressing that statistic's own up or down control.
//
// THE PRESS COUNT IS BOUNDED BY THE DISTANCE and the loop stops the moment a
// press changes nothing. That is what turns a refusal by Chargen.AdjustStat --
// an unaffordable step, a floor, a ceiling -- into an error naming the
// statistic and the value it stopped at, rather than a loop that never ends.
//
// Refund decreases before buying increases, retaining screen order within
// each pass. Installed presets spend the full budget, so an otherwise legal
// requested spread must first release points from its other statistics.
func headlessChargenStats(app *ui.App, state *ui.HeadlessChargen, stats map[string]int) error {
	if len(stats) == 0 {
		return nil
	}
	named := make(map[string]int, len(stats))
	for name, value := range stats {
		named[strings.ToLower(strings.TrimSpace(name))] = value
	}
	seen := map[string]bool{}
	for _, control := range append([]ui.HeadlessChargenControl(nil), state.Controls...) {
		key := strings.ToLower(strings.TrimSpace(control.Label))
		if want, asked := named[key]; asked && control.Kind == ui.ChargenControlStatUp && want < control.Value {
			if err := headlessChargenStat(app, state, control.Label, want); err != nil {
				return err
			}
		}
	}
	for _, control := range append([]ui.HeadlessChargenControl(nil), state.Controls...) {
		if control.Kind != ui.ChargenControlStatUp {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(control.Label))
		want, asked := named[key]
		if !asked {
			continue
		}
		seen[key] = true
		if err := headlessChargenStat(app, state, control.Label, want); err != nil {
			return err
		}
	}
	for key := range named {
		if !seen[key] {
			return fmt.Errorf("create_character: no statistic is %q; the screen offers %s",
				key, headlessQuoted(state.Labels(ui.ChargenControlStatUp)))
		}
	}
	return nil
}

func headlessChargenStat(app *ui.App, state *ui.HeadlessChargen, name string, want int) error {
	kind := ui.ChargenControlStatUp
	for {
		control, ok := state.Control(ui.ChargenControlStatUp, name)
		if !ok {
			return fmt.Errorf("create_character: statistic %q left the screen", name)
		}
		if control.Value == want {
			return nil
		}
		kind = ui.ChargenControlStatUp
		if control.Value > want {
			kind = ui.ChargenControlStatDown
		}
		before := control.Value
		if err := headlessChargenPress(app, state, kind, name); err != nil {
			return err
		}
		after, ok := state.Control(ui.ChargenControlStatUp, name)
		if !ok {
			return fmt.Errorf("create_character: statistic %q left the screen", name)
		}
		if after.Value == before {
			return fmt.Errorf("create_character: statistic %q stopped at %d on the way to %d "+
				"(%d point(s) left): %s", name, before, want, state.Remaining,
				headlessChargenWhy(app))
		}
	}
}

// headlessChargenPress moves the focus to one control and presses Enter, then
// re-reads the screen. Every state change in this file goes through it.
func headlessChargenPress(app *ui.App, state *ui.HeadlessChargen, kind, label string) error {
	if err := headlessChargenFocus(app, state, kind, label); err != nil {
		return err
	}
	if err := app.HeadlessKey("enter"); err != nil {
		return err
	}
	if app.Screen() != ui.ScreenChargen {
		// Play can leave the screen, which is the one press that is meant to.
		return nil
	}
	return headlessChargenRead(app, state)
}

// headlessChargenFocus walks the focus to one control with the screen's own
// up and down keys. It never writes the focus.
func headlessChargenFocus(app *ui.App, state *ui.HeadlessChargen, kind, label string) error {
	control, ok := state.Control(kind, label)
	if !ok {
		return fmt.Errorf("create_character: the %s page offers no %s %q",
			state.Stage, kind, label)
	}
	// The walk is bounded by the control count: a focus that stops moving is a
	// screen this drive cannot navigate, and a bound is what makes that an
	// error rather than a hang.
	for n := 0; state.Focus != control.Focus; n++ {
		if n > len(state.Controls)+1 {
			return fmt.Errorf("create_character: the focus stopped at %d on the way to %s %q at %d",
				state.Focus, kind, label, control.Focus)
		}
		key := "down"
		if state.Focus > control.Focus {
			key = "up"
		}
		if err := app.HeadlessKey(key); err != nil {
			return err
		}
		if err := headlessChargenRead(app, state); err != nil {
			return err
		}
	}
	return nil
}

func headlessChargenRead(app *ui.App, state *ui.HeadlessChargen) error {
	got, ok := app.HeadlessChargenState()
	if !ok {
		return fmt.Errorf("create_character: the generation screen closed; screen is %s: %s",
			app.Screen(), headlessChargenWhy(app))
	}
	*state = got
	return nil
}

// headlessChargenWhy is the screen's own refusal line, or a stand-in when it
// gave none. Every refusal this file reports ends with it, so the sentence a
// player would read is the sentence the scenario reports.
func headlessChargenWhy(app *ui.App) string {
	if msg := strings.TrimSpace(app.HeadlessMessage()); msg != "" {
		return msg
	}
	return "the screen gave no refusal"
}

func headlessChosen(state ui.HeadlessChargen, kind string) (string, bool) {
	for _, c := range state.Controls {
		if c.Kind == kind && c.Chosen {
			return c.Label, true
		}
	}
	return "", false
}

func headlessQuoted(labels []string) string {
	if len(labels) == 0 {
		return "none"
	}
	out := make([]string, len(labels))
	for i, l := range labels {
		out[i] = fmt.Sprintf("%q", l)
	}
	return strings.Join(out, ", ")
}
