package game

// 0163 T2: wait_until on the front-end stage, and the parser's per-stage
// vocabulary (AC-6).

import (
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func boolPtr(b bool) *bool { return &b }

func TestFrontEndWaitUntil(t *testing.T) {
	// The application opens on the menu with no map under it, so a condition
	// about the menu holds at once and a condition about the town cannot be
	// reached by stepping. That is enough to witness both halves without a
	// mission: what is under test is the wait, not what a mission does.
	t.Run("a condition that already holds costs no tick", func(t *testing.T) {
		app := ui.NewApp("test", nil, nil, nil)
		line, err := headlessWaitFrontEnd(app, HeadlessStep{
			Command: "wait_until", Until: &HeadlessUntil{Screen: "menu"}, Ticks: 5})
		if err != nil {
			t.Fatalf("wait_until: %v", err)
		}
		if !strings.Contains(line, "after 0 tick(s)") {
			t.Errorf("wait_until reported %q, want it to have spent no tick", line)
		}
	})

	t.Run("a condition never reached names the ceiling", func(t *testing.T) {
		app := ui.NewApp("test", nil, nil, nil)
		_, err := headlessWaitFrontEnd(app, HeadlessStep{
			Command: "wait_until", Until: &HeadlessUntil{Screen: "town"}, Ticks: 7})
		if err == nil {
			t.Fatal("wait_until = nil error, want one")
		}
		for _, want := range []string{"screen town", "within 7 tick(s)", "screen is menu"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("wait_until error = %v, want it to contain %q", err, want)
			}
		}
	})

	t.Run("the notice form reads the map's own notice", func(t *testing.T) {
		app := ui.NewApp("test", nil, nil, nil)
		if _, err := headlessWaitFrontEnd(app, HeadlessStep{
			Command: "wait_until", Until: &HeadlessUntil{Notice: boolPtr(false)}, Ticks: 3}); err != nil {
			t.Fatalf("wait_until notice=false on the menu: %v", err)
		}
		if _, err := headlessWaitFrontEnd(app, HeadlessStep{
			Command: "wait_until", Until: &HeadlessUntil{Notice: boolPtr(true)}, Ticks: 3}); err == nil {
			t.Fatal("wait_until notice=true with no map = nil error, want one")
		}
	})
}

// The two stages admit disjoint condition forms, and each refuses the
// other's.
func TestUntilFormsAreStageSpecific(t *testing.T) {
	cases := []struct {
		name  string
		stage string
		in    HeadlessUntil
		want  string
	}{
		{"a mission form on the front-end stage", StageFrontEnd,
			HeadlessUntil{Outcome: "won"}, `belong to stage "mission"`},
		{"a unit form on the front-end stage", StageFrontEnd,
			HeadlessUntil{Unit: "p0", Dead: boolPtr(true)}, `belong to stage "mission"`},
		{"a screen form on the mission stage", StageMission,
			HeadlessUntil{Screen: "town"}, `belong to stage "frontend"`},
		{"a notice form on the mission stage", StageMission,
			HeadlessUntil{Notice: boolPtr(true)}, `belong to stage "frontend"`},
		{"two front-end forms at once", StageFrontEnd,
			HeadlessUntil{Screen: "town", Notice: boolPtr(true)}, "exactly one of screen, notice or control"},
		{"no front-end form at all", StageFrontEnd,
			HeadlessUntil{}, "exactly one of screen, notice or control"},
		{"an unknown screen", StageFrontEnd,
			HeadlessUntil{Screen: "shop"}, `unknown screen "shop"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.in.validate(tc.stage)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validate(%q) = %v, want an error containing %q", tc.stage, err, tc.want)
			}
		})
	}
}

// Every screen a scenario can name resolves, and the names are the ones the
// trace and the events already print.
func TestScreenNamesRoundTrip(t *testing.T) {
	for _, s := range []ui.Screen{ui.ScreenMenu, ui.ScreenPicker, ui.ScreenMap,
		ui.ScreenChargen, ui.ScreenTown, ui.ScreenGameMenu, ui.ScreenLoad} {
		got, err := headlessScreenNamed(s.String())
		if err != nil {
			t.Errorf("headlessScreenNamed(%q): %v", s.String(), err)
			continue
		}
		if got != s {
			t.Errorf("headlessScreenNamed(%q) = %v, want %v", s.String(), got, s)
		}
	}
}

// A create_character step is refused before it runs when it states nothing,
// and a version-2 file cannot carry one at all (AC-12).
func TestCreateCharacterStepValidation(t *testing.T) {
	cases := []struct {
		name string
		in   HeadlessScenario
		want string
	}{
		{"a character that states nothing",
			HeadlessScenario{Version: 3, Steps: []HeadlessStep{
				{Command: "create_character", Character: &HeadlessCharacter{}}}},
			"states nothing"},
		{"a negative statistic",
			HeadlessScenario{Version: 3, Steps: []HeadlessStep{
				{Command: "create_character", Character: &HeadlessCharacter{
					Stats: map[string]int{"Body": -1}}}}},
			`statistic "Body" asks for -1`},
		{"a character on the mission stage",
			HeadlessScenario{Version: 3, Stage: StageMission, Mission: 10,
				Steps: []HeadlessStep{{Command: "create_character",
					Character: &HeadlessCharacter{Class: "Mage"}}}},
			`belongs to stage "frontend"`},
		{"a character block on another command",
			HeadlessScenario{Version: 3, Steps: []HeadlessStep{
				{Command: "save", Character: &HeadlessCharacter{Class: "Mage"}}}},
			"unexpected parameter(s): character"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.in.Validate()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() = %v, want an error containing %q", err, tc.want)
			}
		})
	}
}
