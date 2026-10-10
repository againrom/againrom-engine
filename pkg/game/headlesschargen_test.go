package game

// 0163 T1: create_character drives the production generation screen.
//
// EVERY TEST HERE BUILDS ITS OWN ChargenSetup rather than a FrontEnd, because a
// FrontEnd cannot be constructed without a lawful install (NewFrontEnd fails on
// an unreadable chargen node) and golden rule 2 forbids a test that needs one.
// The setup carries a PreCreate block, which is what selects the two-stage
// generator; every option label, statistic name, cost curve and budget in it is
// authored here and none is a decoded number.

import (
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// chargenTestSetup is a two-stage setup shaped like the shipped one: a sex row,
// a class row, and a class-dependent skill row, over four statistics with a
// triangular cost curve. The budget is generous enough that the tests reach the
// refusals they mean to reach and not an accidental one.
// chargenTestSurplus is what the opening spread leaves unspent. It is 25, which
// is exactly enough to buy one statistic from its 25 start to its 45 ceiling
// with 5 to spare, so a test can reach the ceiling refusal and the budget
// refusal separately.
const chargenTestSurplus = 25

func chargenTestSetup() ui.ChargenSetup {
	// A LINEAR CURVE AND A STATED SURPLUS. The shipped curve is data.PointCost
	// and is not restated here (golden rule 2): what these tests need is a
	// budget with known headroom, so a point costs its own value and the
	// opening spread leaves exactly chargenTestSurplus unspent.
	cost := make([]int, 51)
	for v := range cost {
		cost[v] = v
	}
	stat := func(name string) ui.ChargenStat {
		return ui.ChargenStat{Name: name, Floor: 15, Ceiling: 45, Start: 25}
	}
	return ui.ChargenSetup{
		Title: "GENERATE A HERO",
		Name:  "Danath",
		Choices: []ui.ChargenChoice{
			{Name: "Sex", Options: []string{"Male", "Female"}, Parent: -1},
			{Name: "Class", Options: []string{"Fighter", "Mage"}, Parent: -1},
			{Name: "Skill", Parent: 1, OptionsFor: [][]string{
				{"Blade", "Axe", "Bludgeon", "Pike", "Shooting"},
				{"Fire", "Water", "Air", "Earth", "Astral"},
			}},
		},
		Stats:     []ui.ChargenStat{stat("Body"), stat("Reaction"), stat("Mind"), stat("Spirit")},
		Cost:      cost,
		Budget:    4*cost[25] + chargenTestSurplus,
		Confirm:   "ENTER: begin",
		PreCreate: &ui.ChargenPreCreate{Prompt: "choose", Back: "BACK", Art: &ui.ChargenPresentation{Layout: generatorDescriptions["rom1"]}},
		Detailed: &ui.ChargenDetailed{Back: "BACK", Reset: "RESET", Play: "PLAY",
			EmptyName: "a hero needs a name", ReservedName: "that name is reserved"},
	}
}

// armedChargen is an application showing the generation screen, and the result
// its Play control hands over. The opener is nil, which App.stepChargen reads as
// "begin ran and had nothing to open": the screen closes, which is exactly the
// observable this file needs, and no viewer or map is built.
func armedChargen(t *testing.T) (*ui.App, *ui.ChargenResult) {
	t.Helper()
	app := ui.NewApp("test", nil, nil, nil)
	var got ui.ChargenResult
	// THE OPENER IS REAL, over a synthetic grid built here. A nil opener is
	// not a failure to App.stepChargen -- it leaves the screen showing -- and
	// production never hands back one, so a test with a nil opener would
	// exercise a state no player reaches and would make Play look refused.
	begin := func(res ui.ChargenResult) (ui.MapOpener, error) {
		got = res
		return func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect,
			ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
			v, err := ui.NewViewer("m", terrain.Grid{Width: 8, Height: 8, Tiles: make([]uint16, 64)},
				&terrain.Tileset{})
			return v, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}, nil
	}
	if err := app.OpenChargen(ui.NewChargen(chargenTestSetup()), begin); err != nil {
		t.Fatalf("OpenChargen: %v", err)
	}
	return app, &got
}

func TestCreateCharacterReachesTheRequestedCharacter(t *testing.T) {
	app, got := armedChargen(t)
	want := HeadlessCharacter{
		Name: "Aeryn", Sex: "Female", Class: "Mage", Skill: "Water",
		Stats: map[string]int{"Mind": 30, "Body": 20},
	}
	if err := headlessCreateCharacter(app, want); err != nil {
		t.Fatalf("create_character: %v", err)
	}
	if got.Name != "Aeryn" {
		t.Errorf("name = %q, want %q", got.Name, "Aeryn")
	}
	// Choice row 0 is Sex and row 1 is Class, in the setup's own order; option
	// index 1 is the second label each row offers.
	if got.Choices[0] != 1 {
		t.Errorf("sex option = %d, want 1 (Female)", got.Choices[0])
	}
	if got.Choices[1] != 1 {
		t.Errorf("class option = %d, want 1 (Mage)", got.Choices[1])
	}
	if got.Choices[2] != 1 {
		t.Errorf("skill option = %d, want 1 (Water)", got.Choices[2])
	}
	if got.Stats[0] != 20 || got.Stats[2] != 30 {
		t.Errorf("stats = %v, want Body 20 and Mind 30", got.Stats)
	}
	// A statistic nobody named keeps the value the generator opened with.
	if got.Stats[1] != 25 || got.Stats[3] != 25 {
		t.Errorf("stats = %v, want Reaction and Spirit left at 25", got.Stats)
	}
	if app.Screen() == ui.ScreenChargen {
		t.Errorf("the generation screen is still showing")
	}
}

func TestCreateCharacterMatchesTheHandDrivenScreen(t *testing.T) {
	want := HeadlessCharacter{
		Name: "Kore", Sex: "Male", Class: "Fighter", Skill: "Pike",
		Stats: map[string]int{"Spirit": 28},
	}

	app, byStep := armedChargen(t)
	if err := headlessCreateCharacter(app, want); err != nil {
		t.Fatalf("create_character: %v", err)
	}

	hand, byHand := armedChargen(t)
	// Pre-create focus order is name, four pictures, back, forward. Male
	// Fighter is the first picture, so the focus is already on the name. The
	// field only appends, so its opening text is erased first.
	for range len(chargenTestSetup().Name) {
		if err := hand.HeadlessType("", true); err != nil {
			t.Fatalf("backspace: %v", err)
		}
	}
	for _, r := range "Kore" {
		if err := hand.HeadlessType(string(r), false); err != nil {
			t.Fatalf("type: %v", err)
		}
	}
	press := func(downs int) {
		t.Helper()
		for i := 0; i < downs; i++ {
			if err := hand.HeadlessKey("down"); err != nil {
				t.Fatalf("down: %v", err)
			}
		}
		if err := hand.HeadlessKey("enter"); err != nil {
			t.Fatalf("enter: %v", err)
		}
	}
	press(1) // focus 1: the Male Fighter picture
	press(5) // focus 6: FORWARD
	// Detailed focus order is five skills (0-4), then a down and an up control
	// per statistic (5-12), then back, reset, restore and play (13-16). Pike is skill
	// position 3; Spirit is statistic 3, so its up control is focus 12.
	press(3) // focus 3: Pike
	press(9) // focus 12: Spirit up, 25 -> 26
	press(0) // 26 -> 27
	press(0) // 27 -> 28
	press(4) // focus 16: PLAY

	if !reflect.DeepEqual(*byStep, *byHand) {
		t.Fatalf("create_character produced %+v; the same keys by hand produced %+v", *byStep, *byHand)
	}
	if hand.Screen() == ui.ScreenChargen {
		t.Errorf("the hand-driven screen is still showing")
	}
}

func TestCreateCharacterRefusals(t *testing.T) {
	cases := []struct {
		name string
		in   HeadlessCharacter
		want string
	}{
		// Body reaches its ceiling and spends 20 of the 25 surplus; Mind then
		// buys 5 more points and stops with the budget empty. The statistics
		// are set in the screen's own order, so Mind is where it runs out.
		{"an unaffordable statistic",
			HeadlessCharacter{Stats: map[string]int{"Body": 45, "Mind": 45, "Spirit": 45}},
			`statistic "Mind" stopped at 30`},
		{"a statistic above its ceiling",
			HeadlessCharacter{Stats: map[string]int{"Body": 46}},
			`statistic "Body" stopped at 45`},
		{"a statistic below its floor",
			HeadlessCharacter{Stats: map[string]int{"Body": 14}},
			`statistic "Body" stopped at 15`},
		{"a class no row offers",
			HeadlessCharacter{Class: "Necromancer"},
			`no picture is "Male Necromancer"`},
		{"a sex no row offers",
			HeadlessCharacter{Sex: "Neither", Class: "Mage"},
			`no picture is "Neither Mage"`},
		{"a skill this class does not have",
			HeadlessCharacter{Class: "Fighter", Skill: "Astral"},
			`offers no skill "Astral"`},
		{"a statistic no row offers",
			HeadlessCharacter{Stats: map[string]int{"Luck": 20}},
			`no statistic is "luck"`},
		{"a reserved name",
			HeadlessCharacter{Name: "self", Class: "Mage"},
			"that name is reserved"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, got := armedChargen(t)
			err := headlessCreateCharacter(app, tc.in)
			if err == nil {
				t.Fatalf("create_character(%+v) = nil error, want one", tc.in)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("create_character error = %v, want it to contain %q", err, tc.want)
			}
			// A refused step hands nothing over and leaves the screen showing,
			// so nothing half-generated can reach a mission (0163 AC-3).
			if !reflect.DeepEqual(*got, ui.ChargenResult{}) {
				t.Errorf("a refused step handed over %+v", *got)
			}
			if app.Screen() != ui.ScreenChargen {
				t.Errorf("a refused step left screen %s, want the generation screen", app.Screen())
			}
		})
	}
}

// The offered labels are listed in a refusal so an author can fix the file
// without reading this package (0163 AC-4).
func TestCreateCharacterNamesTheOfferedLabels(t *testing.T) {
	app, _ := armedChargen(t)
	err := headlessCreateCharacter(app, HeadlessCharacter{Class: "Necromancer"})
	if err == nil {
		t.Fatal("want an error")
	}
	for _, label := range []string{"Male Fighter", "Male Mage", "Female Fighter", "Female Mage"} {
		if !strings.Contains(err.Error(), label) {
			t.Errorf("the refusal %q does not list %q", err, label)
		}
	}
}

// 0163 AC-5: the generation screen is reported while it is showing, and the
// controls it reports carry the labels a player reads.
func TestChargenStateIsReportedOnlyOnTheGenerationScreen(t *testing.T) {
	app, _ := armedChargen(t)
	state, ok := app.HeadlessChargenState()
	if !ok {
		t.Fatal("HeadlessChargenState reported nothing on the generation screen")
	}
	if state.Stage != ui.ChargenStagePreCreate {
		t.Errorf("stage = %q, want %q", state.Stage, ui.ChargenStagePreCreate)
	}
	if got := state.Labels(ui.ChargenControlPicture); len(got) != 4 {
		t.Errorf("pictures = %q, want four", got)
	}
	if state.Name != "Danath" {
		t.Errorf("name = %q, want the setup's own default", state.Name)
	}
	if !state.Legal {
		t.Error("the screen opened on a spread it calls illegal")
	}

	if err := headlessCreateCharacter(app, HeadlessCharacter{Class: "Mage"}); err != nil {
		t.Fatalf("create_character: %v", err)
	}
	if _, ok := app.HeadlessChargenState(); ok {
		t.Errorf("HeadlessChargenState reported a screen after the generator closed (screen is %s)", app.Screen())
	}
}

func TestChargenSkillLabelsFollowTheChosenClass(t *testing.T) {
	for _, tc := range []struct {
		class string
		first string
	}{{"Fighter", "Blade"}, {"Mage", "Fire"}} {
		t.Run(tc.class, func(t *testing.T) {
			app, _ := armedChargen(t)
			// Reach the detailed page through the same step, stopping short of
			// Play by naming no statistics and no skill.
			state, _ := app.HeadlessChargenState()
			if err := headlessChargenPicture(app, &state, "", tc.class); err != nil {
				t.Fatalf("picture: %v", err)
			}
			if err := headlessChargenPress(app, &state, ui.ChargenControlForward, ""); err != nil {
				t.Fatalf("forward: %v", err)
			}
			got := state.Labels(ui.ChargenControlSkill)
			if len(got) != 5 || got[0] != tc.first {
				t.Fatalf("skills for %s = %q, want five starting at %q", tc.class, got, tc.first)
			}
		})
	}
}
