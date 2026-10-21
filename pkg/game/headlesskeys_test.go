package game

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func readScenarioText(t *testing.T, source string) error {
	t.Helper()
	path := filepath.Join(t.TempDir(), "scenario.json")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := ReadHeadlessScenario(path)
	return err
}

// A refusal of an unknown key, command, stage or pointer form lists what the
// scenario format accepts in that place.
func TestHeadlessRefusalsListTheAcceptedValues(t *testing.T) {
	for _, c := range []struct {
		name, source string
		want         []string
	}{
		{"a scenario key", `{"version":10,"surprise":true,"steps":[{"command":"capture","name":"a"}]}`,
			[]string{`scenario: unknown key "surprise"; accepted keys: version, stage, assets,`, "original_saves, saves, steps"}},
		{"a step key", `{"version":10,"steps":[{"command":"capture","name":"a","colour":1}]}`,
			[]string{`scenario.steps[1]: unknown key "colour"; accepted keys: command, target, key,`, "path"}},
		{"a pointer form", `{"version":10,"steps":[{"command":"pointer","action":"press","at":{"screen":"town"}}]}`,
			[]string{`scenario.steps[1].at: unknown key "screen"; accepted keys: spell, doll_slot,`, "world_map_town", "offset_y"}},
		{"a nested key", `{"version":10,"steps":[{"command":"assert_state","state":{"mission":1,"bogus":2}}]}`,
			[]string{`scenario.steps[1].state: unknown key "bogus"; accepted keys: `}},
		{"a command", `{"version":10,"steps":[{"command":"zzz"}]}`,
			[]string{`unknown command "zzz"; accepted commands on stage "frontend" at version 10: `, "activate", "capture", "frame", "pointer"}},
		{"a command above the version", `{"version":9,"steps":[{"command":"frame","path":"a.png"}]}`,
			[]string{`command reached stage "frontend" in scenario version 10; this file declares version 9`}},
		{"a pointer with no form", `{"version":10,"steps":[{"command":"pointer","action":"press","at":{}}]}`,
			[]string{"at names 0 forms; exactly one is required of: spell, doll_slot, doll_box, pack_cell, pack_code, ground, shop,",
				"world_map_mission, world_map_town, world_map_miss, document"}},
		{"a stage", `{"version":10,"stage":"zzz","steps":[{"command":"capture","name":"a"}]}`,
			[]string{`unknown stage "zzz"; accepted stages: frontend, mission`}},
		{"a frame file name", `{"version":10,"steps":[{"command":"frame","path":"a.bmp"}]}`,
			[]string{`frame path "a.bmp": the file must end in .png`}},
		{"a frame with no path", `{"version":10,"steps":[{"command":"frame"}]}`,
			[]string{"missing parameter(s): path"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := readScenarioText(t, c.source)
			if err == nil {
				t.Fatal("accepted")
			}
			for _, want := range c.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("%v\nwant %q", err, want)
				}
			}
		})
	}
	// The checker accepts what the decoder accepts: the key names match
	// without regard to case, and a full scenario passes.
	if err := readScenarioText(t, `{"Version":10,"steps":[{"Command":"capture","name":"a"}]}`); err != nil {
		t.Fatal(err)
	}
}

// A frame path is refused inside an install and over a file, and the check
// writes nothing.
func TestHeadlessFramePathBoundary(t *testing.T) {
	install := t.TempDir()
	for _, name := range []string{MainArchive, GraphicsArchive, ScenarioArchive, WorldArchive, MoviesArchive} {
		if err := os.WriteFile(filepath.Join(install, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Root: install}}}
	outside := t.TempDir()
	taken := filepath.Join(outside, "taken.png")
	if err := os.WriteFile(taken, []byte("taken"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		path string
		want error
	}{
		{filepath.Join(install, "frame.png"), ErrHeadlessFrameInstall},
		{filepath.Join(install, "deeper", "frame.png"), ErrHeadlessFrameInstall},
		{taken, ErrHeadlessFrameExists},
		{outside, ErrHeadlessFrameExists},
		{filepath.Join(outside, "new.png"), nil},
		{filepath.Join(outside, "new", "deeper", "frame.png"), nil},
	} {
		if err := f.checkHeadlessFramePath(c.path); !errors.Is(err, c.want) && (c.want != nil || err != nil) {
			t.Errorf("%s: %v, want %v", c.path, err, c.want)
		}
	}
	// A scenario naming the same file twice is refused before it runs.
	twice := []HeadlessStep{{Command: "frame", Path: filepath.Join(outside, "a.png")}, {Command: "frame", Path: filepath.Join(outside, "a.png")}}
	if err := f.checkHeadlessFramePaths(twice); err == nil || !strings.Contains(err.Error(), "step 1 writes the same file") {
		t.Errorf("a path named twice: %v", err)
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 1 {
		t.Errorf("the checks wrote into %s: %v", outside, entries)
	}
	if entries, _ := os.ReadDir(install); len(entries) != 5 {
		t.Errorf("the checks wrote into the install: %v", entries)
	}
	if (&FrontEnd{}).checkHeadlessFramePath(taken) == nil {
		t.Error("a front end with no install accepts a frame path")
	}
}

// headlessDoc is the Headless runs section of docs/MODS.md.
func headlessDoc(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("../../docs/MODS.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := strings.ReplaceAll(string(raw), "\r\n", "\n")
	_, rest, ok := strings.Cut(doc, "\n## Headless runs\n")
	if !ok {
		t.Fatal("docs/MODS.md has no Headless runs section")
	}
	section, _, _ := strings.Cut(rest, "\n## ")
	return section
}

// missingFromDoc are the names the section does not carry in code quotes.
func missingFromDoc(section string, names []string) []string {
	var missing []string
	for _, name := range names {
		if !strings.Contains(section, "`"+name+"`") {
			missing = append(missing, name)
		}
	}
	return missing
}

// The section carries the fixture scenario as its example, and names every
// scenario key, command, pointer form and capture field.
func TestHeadlessDocExampleIsTheFixtureAndNamesTheFormat(t *testing.T) {
	section := headlessDoc(t)
	want, err := os.ReadFile("testdata/headless-town.json")
	if err != nil {
		t.Fatal(err)
	}
	norm := func(b []byte) string { return strings.TrimSpace(strings.ReplaceAll(string(b), "\r\n", "\n")) }
	if !strings.Contains(section, "```json\n"+norm(want)+"\n```") {
		t.Fatal("docs/MODS.md does not carry testdata/headless-town.json as its example")
	}
	// Loss control: an example that drifts from the fixture is not found.
	drift := strings.Replace(norm(want), `"east test"`, `"east trial"`, 1)
	if strings.Contains(section, "```json\n"+drift+"\n```") {
		t.Fatal("a drifted example matches")
	}
	var names []string
	for name := range headlessCommands {
		names = append(names, name)
	}
	names = append(names, jsonKeys(reflect.TypeOf(HeadlessScenario{}))...)
	names = append(names, jsonKeys(reflect.TypeOf(HeadlessPoint{}))...)
	names = append(names, jsonKeys(reflect.TypeOf(HeadlessTown{}))...)
	names = append(names, jsonKeys(reflect.TypeOf(HeadlessFrame{}))...)
	names = append(names, "command", "target", "key", "path", "action", "at")
	slices.Sort(names)
	names = slices.Compact(names)
	if missing := missingFromDoc(section, names); len(missing) > 0 {
		t.Fatalf("docs/MODS.md does not name %q", missing)
	}
	// Loss control: a section without a command is found short.
	if missing := missingFromDoc(strings.ReplaceAll(section, "`frame`", ""), []string{"frame"}); len(missing) != 1 {
		t.Fatal("a section without frame is not found short")
	}
}

func TestHeadlessDocPointerActionsAreTheAcceptedList(t *testing.T) {
	listed := func(section string) []string {
		_, rest, ok := strings.Cut(section, "`action` of `pointer`:")
		if !ok {
			t.Fatal("the section has no pointer action list")
		}
		para, _, _ := strings.Cut(rest, "\n\n")
		para, _, _ = strings.Cut(para, "each with")
		var names []string
		for i, f := range strings.Split(para, "`") {
			if i%2 == 1 {
				names = append(names, f)
			}
		}
		slices.Sort(names)
		return names
	}
	want := slices.Clone(ui.HeadlessPointerActions)
	slices.Sort(want)
	section := headlessDoc(t)
	if got := listed(section); !slices.Equal(got, want) {
		t.Fatalf("the page lists %q, the game accepts %q", got, want)
	}
	if got := listed(strings.Replace(section, "`wheel-up`", "", 1)); slices.Equal(got, want) {
		t.Fatal("a page missing wheel-up matches")
	}
	if got := listed(strings.Replace(section, "`wheel-up`", "`wheel-up`, `wheel-left`", 1)); slices.Equal(got, want) {
		t.Fatal("a page with an extra action matches")
	}
	for _, action := range []string{"shift-press", "ctrl-hover", "shift-ctrl-wheel-up"} {
		if _, _, _, err := ui.ParseHeadlessPointerAction(action); err != nil {
			t.Errorf("%s: %v", action, err)
		}
	}
	for _, action := range []string{"shift-bogus", "ctrl-shift-press", "wheel-left", ""} {
		if _, _, _, err := ui.ParseHeadlessPointerAction(action); err == nil || !strings.Contains(err.Error(), "wheel-down") {
			t.Errorf("%q: %v", action, err)
		}
	}
}
