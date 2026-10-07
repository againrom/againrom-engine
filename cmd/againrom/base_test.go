package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/game"
	"againrom/pkg/ui"
)

// registerProfile makes the synthetic install at root a build of p for the
// test's duration, so the base machinery can be driven over a constructed root.
// The profile is matched by the size and digest of the root's own main archive.
func registerProfile(t *testing.T, root string, p base.Profile) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "main.res"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	p.Builds = []base.Build{{Size: int64(len(b)), SHA256: hex.EncodeToString(sum[:])}}
	old := base.Profiles
	base.Profiles = append([]base.Profile{p}, old...)
	t.Cleanup(func() { base.Profiles = old })
}

func noGenerationProfile(mission int, files ...string) base.Profile {
	return base.Profile{
		ID: "rom1-test", Title: "Constructed base", Language: "english", Files: files,
		Limits: base.Limits{NoCharacterGeneration: true, FirstMission: mission, Notes: []string{"a stated limit"}},
	}
}

func TestParseBaseFlag(t *testing.T) {
	for _, id := range base.IDs() {
		o, err := parse([]string{"-base", id})
		if err != nil || o.base != id {
			t.Errorf("-base %s: %v %q", id, err, o.base)
		}
	}
	if o, err := parse(nil); err != nil || o.base != "" {
		t.Errorf("default: %v %q", err, o.base)
	}
	for _, bad := range []string{"rom3-ru", "rom1", "demo"} {
		if _, err := parse([]string{"-base", bad}); err == nil || !strings.Contains(err.Error(), base.ROM1Demo) {
			t.Errorf("-base %s: err = %v, want a refusal listing the known profiles", bad, err)
		}
	}
}

func TestCheckNamesTheBase(t *testing.T) {
	root := missionInstall(t)
	code, out, stderr := checkRun(t, []string{"-assets", root, "-check"}, "")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(out, "againrom: base rom1-en (Rage of Mages, English; unrecognised build, matched by language)\n") {
		t.Fatalf("the check does not name the base:\n%s", out)
	}
	if strings.Contains(out, "base limit") {
		t.Fatalf("a release profile states a limit:\n%s", out)
	}
	if !strings.Contains(out, "mission 10 at") || !strings.Contains(out, "chargen offers") {
		t.Fatalf("a release profile changed the mission or generation report:\n%s", out)
	}
}

func TestBaseFlagRefusesAnotherBase(t *testing.T) {
	root := missionInstall(t)
	code, _, stderr := checkRun(t, []string{"-assets", root, "-check", "-base", base.ROM1Demo}, "")
	if code != 2 || !strings.Contains(stderr, base.ROM1Demo) || !strings.Contains(stderr, base.ROM1EN) {
		t.Fatalf("exit %d stderr %q, want a refusal naming both bases", code, stderr)
	}
	if code, _, stderr := checkRun(t, []string{"-assets", root, "-check", "-base", base.ROM1EN}, ""); code != 0 {
		t.Fatalf("matching -base: exit %d: %s", code, stderr)
	}
}

func TestBaseFlagOnARootThatIsNotAnInstall(t *testing.T) {
	code, _, stderr := checkRun(t, []string{"-assets", t.TempDir(), "-check", "-base", base.ROM1EN}, "")
	if code != 2 || !strings.Contains(stderr, "not a game base: missing main.res") {
		t.Fatalf("exit %d stderr %q, want a refusal naming the missing archives", code, stderr)
	}
}

// A root whose main archive is a known build and which lacks a file of its
// profile is refused, naming the file, with or without -base.
func TestPartialBaseIsRefused(t *testing.T) {
	root := missionInstall(t)
	registerProfile(t, root, noGenerationProfile(10, "video4.res"))
	for _, args := range [][]string{{"-assets", root, "-check"}, {"-assets", root, "-check", "-base", "rom1-test"}} {
		code, _, stderr := checkRun(t, args, "")
		if code != 2 || !strings.Contains(stderr, "missing video4.res") || !strings.Contains(stderr, "rom1-test") {
			t.Fatalf("%v: exit %d stderr %q, want the partial base refused naming the file", args, code, stderr)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "Video4.res"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := checkRun(t, []string{"-assets", root, "-check", "-base", "rom1-test"}, ""); code != 0 {
		t.Fatalf("complete base: exit %d: %s", code, stderr)
	}
}

// A profile with no character generation moves the default mission to its own
// first mission, states its limits in the check, and drops the generation line.
func TestCheckStatesAProfilesLimits(t *testing.T) {
	root := missionInstall(t)
	registerProfile(t, root, noGenerationProfile(20))
	code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-base", "rom1-test"}, "")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	for _, want := range []string{
		"againrom: base rom1-test (Constructed base; exact build)",
		"againrom: base limit: no character generation; new game opens mission 20 with the default party",
		"againrom: base limit: a stated limit",
		"againrom: mission 20 at scenario/20.alm",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "chargen offers") {
		t.Errorf("a base without generation reports generation:\n%s", out)
	}
	// An explicit -mission still names the mission asked for.
	_, out, _ = checkRun(t, []string{"-assets", root, "-check", "-mission", "10"}, "")
	if !strings.Contains(out, "mission 10 at scenario/10.alm") {
		t.Errorf("an explicit -mission was overridden by the profile:\n%s", out)
	}
}

func baseFront(t *testing.T, root string) (*game.FrontEnd, *ui.App, options) {
	t.Helper()
	o, err := parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	front, err := frontEnd(root, o, game.OptionsStore{})
	if err != nil {
		t.Fatal(err)
	}
	app := front.App("test")
	app.Layout(640, 480)
	return front, app, o
}

// NEW GAME on a base without generation opens the profile's first mission with
// no generation screen, and loads no generation art.
func TestNewGameOpensTheFirstMissionWithoutGeneration(t *testing.T) {
	root := missionInstall(t)
	registerProfile(t, root, noGenerationProfile(20))
	front, app, o := baseFront(t, root)
	if front.ChargenAssets != nil {
		t.Fatal("generation art was loaded for a base that ships none")
	}
	if err := armNewGameDoor(app, front, o); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMenu {
		t.Fatalf("screen = %v, want the menu", app.Screen())
	}
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMap {
		t.Fatalf("screen after NEW GAME = %v, want the map", app.Screen())
	}
	mission := 20
	scenario := game.HeadlessScenario{Version: game.HeadlessScenarioVersion, Steps: []game.HeadlessStep{
		{Command: "wait_ticks", Ticks: 8},
		{Command: "assert_state", State: &game.HeadlessStateAssertion{Screen: "map", Mission: &mission}},
	}}
	if err := game.RunHeadlessScenario(front, app, scenario, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
}

// A first mission the install cannot open falls back to the picker and says why
// on its message line, instead of exiting.
func TestNewGameFallsBackWhenTheFirstMissionWillNotOpen(t *testing.T) {
	root := missionInstall(t)
	registerProfile(t, root, noGenerationProfile(99))
	front, app, o := baseFront(t, root)
	if err := armNewGameDoor(app, front, o); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenPicker {
		t.Fatalf("screen = %v, want the picker after a failed first mission", app.Screen())
	}
}

// -mission and -picker keep their meaning on every base except that -mission
// cannot open generation a base does not ship.
func TestMissionFlagOnABaseWithoutGeneration(t *testing.T) {
	root := missionInstall(t)
	registerProfile(t, root, noGenerationProfile(20))

	front, app, _ := baseFront(t, root)
	o, err := parse([]string{"-mission", "10"})
	if err != nil {
		t.Fatal(err)
	}
	err = armNewGameDoor(app, front, o)
	if err == nil || !strings.Contains(err.Error(), "rom1-test") || !strings.Contains(err.Error(), "mission 20") {
		t.Fatalf("err = %v, want a refusal naming the base and its first mission", err)
	}
}

func TestPickerKeepsTheMapListOnABaseWithoutGeneration(t *testing.T) {
	root := missionInstall(t)
	registerProfile(t, root, noGenerationProfile(20))
	code, _, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker"}, "")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	front, app, _ := baseFront(t, root)
	app.SetNewGameDirect(nil)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenPicker {
		t.Fatalf("screen = %v, want the picker under -picker", app.Screen())
	}
	// A mission row opens its mission with no generation step.
	if err := app.HeadlessActivate("Mission 10: 10.alm"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMap {
		t.Fatalf("screen = %v, want the map: the base ships no generation to put in front of a row", app.Screen())
	}
	_ = front
}

// Both release roots and the demo are named by -check. The demo's own witness is
// in pkg/game; this one runs the command over the lawful install named by
// AGAINROM_ASSETS and requires the profile its language names.
func TestReleaseCheckNamesTheBase(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the check over a lawful install needs one")
	}
	info := game.InspectInstall(root)
	want := map[string]string{"english": base.ROM1EN, "russian": base.ROM1RU}[info.Language]
	if want == "" {
		t.Fatalf("install language %q is neither english nor russian", info.Language)
	}
	var out, errOut bytes.Buffer
	code := runWithProfilePaths([]string{"-assets", root, "-check", "-base", want}, noEnv, &out, &errOut, "", t.TempDir())
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "againrom: base "+want+" (") || !strings.Contains(out.String(), "; exact build)") {
		t.Fatalf("the check does not name %s as an exact build:\n%s", want, out.String())
	}
	if strings.Contains(out.String(), "base limit") || !strings.Contains(out.String(), "mission 10 at") {
		t.Fatalf("a release base changed the report:\n%s", out.String())
	}
	other := base.ROM1RU
	if want == base.ROM1RU {
		other = base.ROM1EN
	}
	if code := runWithProfilePaths([]string{"-assets", root, "-check", "-base", other}, noEnv, &out, &errOut, "", t.TempDir()); code != 2 {
		t.Fatalf("-base %s over a %s install: exit %d, want 2", other, want, code)
	}
}
