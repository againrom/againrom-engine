//go:build release

package game

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

// TestReleaseSecondMapEditorOpensEveryMissionText opens every installed
// second-game map: a campaign map's pages window holds the sections of its
// mission's text file, read back to the install's bytes, and any other map's
// the sections of quest.txt (R2-ENGINE-049). Each untouched map, saved into
// a mod folder, is its installed bytes, and no text file is written.
func TestReleaseSecondMapEditorOpensEveryMissionText(t *testing.T) {
	x := installedInspector(t)
	sources, raws := installedSecondMaps(t, x)
	modDir := filepath.Join(t.TempDir(), "census")
	writeTestModManifest(t, modDir)
	var campaign, quest, sections, named, unnamed, equal int
	files := map[string]bool{}
	for i, source := range sources {
		d, err := x.Open(source)
		if err != nil {
			t.Fatal(source, err)
		}
		host := d.Edits.(*mapInspectionEdits)
		fields := host.WorkspaceFields("pages")
		if host.eventNote != "" || len(fields) != 2 || len(fields[0].Choices) == 0 {
			t.Errorf("%s: the mission text does not open: %q", source, host.eventNote)
			continue
		}
		if sourceMission(source) > 0 {
			campaign++
		} else {
			quest++
		}
		files[host.workspace.Sections] = true
		sections += len(fields[0].Choices)
		for _, target := range d.Script.Targets[ui.ScriptMessageParameter] {
			if len(target.Uses) > 0 {
				named++
			} else {
				unnamed++
			}
		}
		target, err := host.SaveAs(filepath.Join(modDir, "maps", fmt.Sprintf("m%03d.alm", i)))
		if err != nil {
			t.Errorf("%s: save refused: %v", source, err)
			continue
		}
		if got, err := os.ReadFile(target); err != nil || firstDifference(raws[i], got) >= 0 {
			t.Errorf("%s: the untouched map does not save equal: %v", source, err)
			continue
		}
		equal++
	}
	if _, err := os.Stat(filepath.Join(modDir, mod.FilesDir)); !os.IsNotExist(err) {
		t.Errorf("an untouched map's save wrote install files: %v", err)
	}
	if campaign == 0 || equal != len(sources) {
		t.Fatalf("%d campaign maps opened, %d of %d saved equal", campaign, equal, len(sources))
	}
	t.Logf("PASS (%s): %d maps; %d campaign maps and %d others open %d text files with %d sections; %d event sections named by an action, %d by none; %d of %d untouched maps save equal into a mod, no text written",
		installLanguage(x), len(sources), campaign, quest, len(files), sections, named, unnamed, equal, len(sources))
}

// TestReleaseSecondMapEditedEventTextShowsInTheGame edits the event section an
// installed campaign map's action sends, saves the map into a mod, loads the
// mod, opens the mission and raises the event: the dialogue the game builds
// shows the edited text.
func TestReleaseSecondMapEditedEventTextShowsInTheGame(t *testing.T) {
	x := installedInspector(t)
	sources, _ := installedSecondMaps(t, x)
	var host *mapInspectionEdits
	var source string
	var event uint32
	for _, s := range sources {
		if sourceMission(s) == 0 {
			continue
		}
		d, err := x.Open(s)
		if err != nil {
			t.Fatal(s, err)
		}
		for _, target := range d.Script.Targets[ui.ScriptMessageParameter] {
			if _, interactive := mapload.InteractiveIndex(int32(target.Value)); len(target.Uses) > 0 && !interactive {
				host, source, event = d.Edits.(*mapInspectionEdits), s, target.Value
				break
			}
		}
		if host != nil {
			break
		}
	}
	if host == nil {
		t.Fatal("no campaign map sends an event its text holds")
	}
	edited := "Againrom edited event " + strconv.Itoa(int(event)) + "."
	if _, err := host.SetWorkspaceField("pages", "page", "event"+strconv.Itoa(int(event))); err != nil {
		t.Fatal(err)
	}
	// The author rewrites the line under the section's part 1 tag and keeps
	// the tag, which the dialogue builder pages by.
	lines := strings.Split(host.WorkspaceFields("pages")[1].TextValue, "\n")
	at := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(l, "<") && containsFold(l, partMark+"1") })
	if at < 0 || at+1 >= len(lines) {
		t.Fatalf("event %d's section has no part 1: %q", event, lines)
	}
	lines[at+1] = edited
	if _, err := host.SetWorkspaceField("pages", "body", strings.Join(lines, "\n")); err != nil {
		t.Fatal(err)
	}
	mods := t.TempDir()
	modDir := filepath.Join(mods, "edited-text")
	writeTestModManifest(t, modDir)
	if _, err := host.SaveAs(filepath.Join(modDir, "maps", "m.alm")); err != nil || host.SaveNote() != "" {
		t.Fatalf("save into the mod: %v %q", err, host.SaveNote())
	}
	root := os.Getenv("AGAINROM_ASSETS")
	entries, err := mod.Resolve(mods, []string{"edited-text"})
	if err != nil {
		t.Fatal(err)
	}
	res, err := modrt.Load(entries, BaseID(InspectInstall(root)), nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	f, err := NewModFrontEnd(root, res.Files)
	if err != nil {
		t.Fatal(err)
	}
	cleanupFrontAudio(t, f)
	witnessSeeds(f)
	f.SetDeterministicFrames(true)
	if err := f.SetMods(res.Rules, res.Set, false); err != nil {
		t.Fatal(err)
	}
	app := f.App("edited event text")
	number := sourceMission(source)
	open, err := f.MissionBegin(number)(ui.ChargenResult{Name: "Test", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if f.live == nil || f.live.mission == nil || f.live.mission.number != number {
		t.Fatalf("mission %d did not open", number)
	}
	if !f.live.openDialogue(int(event)) {
		t.Fatalf("event %d opens no dialogue", event)
	}
	message, _, up := f.LiveNotice()
	if !up || !strings.Contains(message, edited) {
		t.Fatalf("event %d shows %q, want the edited %q", event, message, edited)
	}
	t.Logf("PASS (%s): %s event %d, edited in the pages window and saved into %s, shows %q in mission %d",
		installLanguage(x), source, event, filepath.ToSlash(filepath.Join("files", x.eventTextLanguage(), host.workspace.Sections)), message, number)
}

// TestReleaseSecondMapEditorRefusesTextTheGameShowsAsOtherLetters types the
// numero sign into a campaign map's section: a Russian install refuses it by
// name, because its mod text file serves that font byte back as another
// letter, and accepts text that survives; the English install accepts both.
func TestReleaseSecondMapEditorRefusesTextTheGameShowsAsOtherLetters(t *testing.T) {
	x := installedInspector(t)
	sources, _ := installedSecondMaps(t, x)
	for _, s := range sources {
		if sourceMission(s) == 0 {
			continue
		}
		d, err := x.Open(s)
		if err != nil {
			t.Fatal(s, err)
		}
		host := d.Edits.(*mapInspectionEdits)
		keys := eventSectionKeys(string(host.workspace.Pages[host.workspace.Sections]))
		if len(keys) == 0 {
			continue
		}
		if _, err := host.SetWorkspaceField("pages", "page", keys[0]); err != nil {
			t.Fatal(err)
		}
		_, err = host.SetWorkspaceField("pages", "body", "Sign No 5: №")
		ru := LanguageSelector(x.archives.Containers) == 1
		if ru != (err != nil) || ru && !strings.Contains(err.Error(), "U+2116") {
			t.Fatalf("%s: the numero sign gives %v on a Russian install %v", s, err, ru)
		}
		if _, err := host.SetWorkspaceField("pages", "body", "Sign No 5."); err != nil {
			t.Fatalf("%s: text that survives is refused: %v", s, err)
		}
		return
	}
	t.Fatal("no campaign map holds a section")
}
