//go:build release

package game

import (
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/ui"
)

// scratchInstallMod writes a scratch mod of the install's game family; with
// missions it holds a replace mission of mission 20 and its map.
func scratchInstallMod(t *testing.T, x *MapInspector, missions bool) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "ambush")
	manifest := strings.Replace(editorModManifest, `["rom1"]`, `["`+base.ModFamilyOf(x.installGame())+`"]`, 1)
	files := map[string]string{"mod.toml": manifest, "main.star": "def init(game, settings):\n    pass\n"}
	if missions {
		files["main.star"] = "def init(game, settings):\n    game.data.add(\"data/missions.toml\")\n"
		files["data/missions.toml"] = treeModMissions
	}
	for name, body := range files {
		writeFileAll(t, filepath.Join(dir, name), []byte(body))
	}
	if missions {
		for _, lang := range []string{"en", "ru"} {
			writeFileAll(t, filepath.Join(dir, "maps", lang, "map.alm"), synth.ALM(synth.ALMOptions{Width: 8, Height: 8, Name: "mission"}))
		}
	}
	return dir
}

// openInstallMod opens the mod at dir in an editor of the install.
func openInstallMod(t *testing.T, x *MapInspector, dir string) *ui.MapEditor {
	t.Helper()
	e := x.Editor()
	e.Layout(1920, 1080)
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	if !e.ModWindowOpen() {
		t.Fatal("no mod window")
	}
	return e
}

// A mod with no mission opens its window on an install with no map open; a
// text and an item record edit and save, and the saved mod reloads equal.
func TestReleaseModWindowEditsAndSavesWithNoMap(t *testing.T) {
	x := installedInspector(t)
	e := openInstallMod(t, x, scratchInstallMod(t, x, false))
	// An opened mod is read-only; the copy under another parent is the
	// one edited.
	dir := filepath.Join(t.TempDir(), "ambush")
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	if e.Document() != nil || !e.ModWindowOpen() {
		t.Fatal("no mod window with no map")
	}
	s := x.modHost().(*mapInspectionEdits)
	if msg := x.modContextRefusal(); msg != "" {
		t.Fatalf("the install's context refuses edits: %s", msg)
	}
	before := heldKeys(t, s)
	setTexts(t, s, textsNewKey, "1")
	k := madeKey(t, before, heldKeys(t, s))
	setTexts(t, s, "en:"+k, "The Pass")
	setItems(t, s, itemsNewArmour, "1")
	price := ""
	for _, f := range s.WorkspaceFields(ui.ModItemsScope) {
		if strings.HasPrefix(f.Key, "item.") && strings.HasSuffix(f.Key, ".price") {
			price = f.Key
			break
		}
	}
	if price == "" {
		t.Fatal("the items page offers no price of the new item")
	}
	setItems(t, s, price, "77")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	saved := tree(t, dir)

	y := installedInspector(t)
	openInstallMod(t, y, dir)
	s2 := y.modHost().(*mapInspectionEdits)
	if f, ok := textsField(s2, "en:"+k); !ok || f.TextValue != "The Pass" {
		t.Fatalf("the reloaded text %+v", f)
	}
	if f, ok := itemsField(s2, price); !ok || f.TextValue != "77" {
		t.Fatalf("the reloaded item price %+v", f)
	}
	if _, err := s2.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, tree(t, dir)) {
		t.Fatal("the reloaded mod saves differently")
	}
}

// The second game's missions are the install's archive maps; a map the mod
// holds unused offers to replace the first of them, named by its map name
// where the map has one.
func TestReleaseSecondGameModListsTheInstallMissions(t *testing.T) {
	x := installedInspector(t)
	if x.installGame() != base.GameROM2 {
		t.Skip("requires a second-game install")
	}
	c, err := x.installCampaign()
	if err != nil || len(c.Main) == 0 {
		t.Fatalf("missions %v: %v", c.Main, err)
	}
	for _, m := range x.Maps {
		if n, ok := missionNumber(m); ok && !slices.Contains(c.Main, n) {
			t.Errorf("mission %d of the install is not listed", n)
		}
	}
	dir := scratchInstallMod(t, x, true)
	writeFileAll(t, filepath.Join(dir, "maps", "en", "spare.alm"), synth.ALM(synth.ALMOptions{Width: 8, Height: 8, Name: "spare"}))
	e := openInstallMod(t, x, dir)
	if e.Document() == nil {
		t.Fatal("the mission mod opened no map")
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	actions, ok := s.ModMapActions()
	if !ok {
		t.Fatal("no map list")
	}
	first := c.Main[0]
	if first == 20 {
		first = c.Main[1]
	}
	for _, a := range actions {
		for _, b := range a.Buttons {
			if strings.Contains(b.Value, "|") {
				if !strings.HasPrefix(b.Value, strconv.Itoa(first)+"|") || !strings.Contains(b.Caption, strconv.Itoa(first)) {
					t.Errorf("replace button %+v is not for mission %d", b, first)
				}
				return
			}
		}
	}
	t.Fatalf("no replace button among %+v", actions)
}

func TestReleaseEditorCampaignReadFollowsTheInstallGame(t *testing.T) {
	x := installedInspector(t)
	want, wantErr := campaignOf(x.archives.Game()).loadCampaign(x.archives.Containers)
	got, gotErr := x.loadCampaign()
	if (wantErr == nil) != (gotErr == nil) || len(want.Main) != len(got.Main) || len(want.Chapters) != len(got.Chapters) {
		t.Fatalf("editor read %d missions %d chapters, err %v; game reader %d, %d, err %v",
			len(got.Main), len(got.Chapters), gotErr, len(want.Main), len(want.Chapters), wantErr)
	}
	if !x.installGame().Edition().HasRegistry() {
		if gotErr != nil {
			t.Fatalf("a game with no registry read an error: %v", gotErr)
		}
		if c, err := x.installCampaign(); err != nil || len(c.Main) == 0 {
			t.Fatalf("install missions %v: %v", c.Main, err)
		}
	}
}
