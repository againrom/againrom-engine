package game

import (
	"bytes"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// insidePages is a page tree in both languages: a briefing and the page of
// message 1.
func insidePages() map[string][]byte {
	return map[string][]byte{
		"en/briefing.txt": []byte("Hold the bridge.\r\n"),
		"ru/briefing.txt": []byte("Удержите мост.\r\n"),
		"en/event01.txt":  []byte("<part=1>\r\nThey come.\r\n"),
		"ru/event01.txt":  []byte("<part=1>\r\nОни идут.\r\n"),
	}
}

// insideMap is a small map whose one trigger sends message 1, its script
// moved into the Againrom record; extended false leaves it original.
func insideMap(t *testing.T, extended bool) []byte {
	t.Helper()
	b := synth.ALM(withScript(synth.ALMOptions{Width: 4, Height: 4, Name: "inside"}, messageScript(1)))
	if !extended {
		return b
	}
	m, err := mapedit.New(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	out, err := m.SaveBytes()
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// withInside is b with pages in its texts record.
func withInside(t *testing.T, b []byte, pages map[string][]byte) []byte {
	t.Helper()
	out, err := alm.AppendTextsRecord(b, pages)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func insideInspector(t *testing.T) *MapInspector {
	t.Helper()
	x := newTestPlayFixture(t).x
	x.tiles, x.font = &terrain.Tileset{}, &text.Font{Selector: 1}
	return x
}

func insideHost(t *testing.T, x *MapInspector, path string) *mapInspectionEdits {
	t.Helper()
	d, err := x.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return d.Edits.(*mapInspectionEdits)
}

func pagesOf(t *testing.T, path string) map[string][]byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pages, held, err := alm.TextsOf(b)
	if err != nil || !held {
		t.Fatalf("%s holds no texts record: %v", path, err)
	}
	return pages
}

func treeBytes(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	for name, body := range tree(t, dir) {
		out[name] = []byte(body)
	}
	return out
}

// Saved with the switch on, the pages go in the map and no folder is
// written; the map reopens with the switch on, and the game's reader gives
// the same EN and RU pages as the folder the switch-off save writes, whose
// map is the same map without the record.
func TestTextsInsideSaveWritesThePagesIntoTheMap(t *testing.T) {
	x := insideInspector(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "bridge.almx")
	writeTestPlayInput(t, source, insideMap(t, true))
	for name, b := range insidePages() {
		writeTestPlayInput(t, filepath.Join(dir, "bridge.text", filepath.FromSlash(name)), b)
	}
	s := insideHost(t, x, source)
	if s.TextsInside() {
		t.Fatal("the switch is on for a map without the record")
	}
	if err := s.SetTextsInside(true); err != nil || !s.TextsInside() {
		t.Fatalf("the switch on an extended map: %v", err)
	}
	if _, err := s.SetWorkspaceField("pages", "page", "event02.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetWorkspaceField("pages", "body", "Hold on."); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	inside := filepath.Join(out, "inside.almx")
	if _, err := s.SaveAs(inside); err != nil {
		t.Fatal(err)
	}
	if got := tree(t, out); len(got) != 1 {
		t.Fatalf("the save wrote more than the map: %v", got)
	}
	want := insidePages()
	want["en/event02.txt"] = []byte("Hold on.")
	if got := pagesOf(t, inside); !maps.EqualFunc(got, want, bytes.Equal) {
		t.Fatalf("the record holds %q", got)
	}
	again := insideHost(t, x, inside)
	if !again.TextsInside() || again.Dirty() {
		t.Fatalf("the reopened map: switch %v dirty %v", again.TextsInside(), again.Dirty())
	}
	if err := again.SetTextsInside(false); err != nil {
		t.Fatal(err)
	}
	folderDir := t.TempDir()
	folder := filepath.Join(folderDir, "folder.almx")
	if _, err := again.SaveAs(folder); err != nil {
		t.Fatal(err)
	}
	if got := treeBytes(t, filepath.Join(folderDir, "folder.text")); !maps.EqualFunc(got, want, bytes.Equal) {
		t.Fatalf("the switch-off save wrote the folder %q", got)
	}
	a, _ := os.ReadFile(inside)
	b, _ := os.ReadFile(folder)
	if plain, held := alm.WithoutTextsRecord(a); !held || !bytes.Equal(plain, b) {
		t.Fatal("the switch-off map is not the same map without the record")
	}
	for _, lang := range []string{"en", "ru"} {
		fromRecord, otherRecord, err := mod.Mission{Map: "inside.almx"}.ReadTexts(out, lang)
		if err != nil {
			t.Fatal(err)
		}
		fromFolder, otherFolder, err := mod.Mission{Map: "folder.almx"}.ReadTexts(folderDir, lang)
		if err != nil || !reflect.DeepEqual(fromRecord, fromFolder) || !reflect.DeepEqual(otherRecord, otherFolder) {
			t.Fatalf("%s: the record reads %q, the folder %q: %v", lang, fromRecord, fromFolder, err)
		}
		if !bytes.Equal(fromRecord["briefing.txt"], want[lang+"/briefing.txt"]) || !bytes.Equal(fromRecord["event01.txt"], want[lang+"/event01.txt"]) {
			t.Fatalf("%s pages %q", lang, fromRecord)
		}
	}
}

// A map that holds a record and a folder reads the record first and a page
// only the folder has. A switch-off save writes both into the new folder;
// a switch-on save over the map's own file leaves its folder as it was.
func TestTextsInsideRecordAndFolderLoseNothing(t *testing.T) {
	x := insideInspector(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "bridge.almx")
	writeTestPlayInput(t, source, withInside(t, insideMap(t, true), insidePages()))
	writeTestPlayInput(t, filepath.Join(dir, "bridge.text", "en", "event01.txt"), []byte("Stale"))
	writeTestPlayInput(t, filepath.Join(dir, "bridge.text", "en", "event09.txt"), []byte("Folder only"))
	if p := x.editorPages(source); p.Err != nil || string(p.Files()["event01.txt"]) != "<part=1>\r\nThey come.\r\n" || string(p.Files()["event09.txt"]) != "Folder only" {
		t.Fatalf("the editor's pages %q: %v", p.Files(), p.Err)
	}
	s := insideHost(t, x, source)
	if !s.TextsInside() {
		t.Fatal("a map with the record opens with the switch off")
	}
	folderBefore := tree(t, filepath.Join(dir, "bridge.text"))
	if _, err := s.SaveInPlace(); err != nil {
		t.Fatal(err)
	}
	want := insidePages()
	want["en/event09.txt"] = []byte("Folder only")
	if got := pagesOf(t, source); !maps.EqualFunc(got, want, bytes.Equal) {
		t.Fatalf("the record after a save in place holds %q", got)
	}
	if !reflect.DeepEqual(folderBefore, tree(t, filepath.Join(dir, "bridge.text"))) {
		t.Fatal("the save with the switch on changed the folder beside the map")
	}
	if err := s.SetTextsInside(false); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if _, err := s.SaveAs(filepath.Join(out, "plain.almx")); err != nil {
		t.Fatal(err)
	}
	if got := treeBytes(t, filepath.Join(out, "plain.text")); !maps.EqualFunc(got, want, bytes.Equal) {
		t.Fatalf("the switch-off save wrote %q", got)
	}
	if b, _ := os.ReadFile(filepath.Join(out, "plain.almx")); bytes.Contains(b, []byte("They come.")) {
		t.Fatal("the switch-off map carries the pages")
	}
}

// An .alm never holds the record: the switch asks to extend an original
// map, a save under an .alm name is refused, a save in place over an own
// .alm writes the .almx beside it, and the map check names the record under
// an .alm name.
func TestTextsInsideNeverRidesAnAlm(t *testing.T) {
	x := insideInspector(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "bridge.alm")
	original := insideMap(t, false)
	writeTestPlayInput(t, source, original)
	s := insideHost(t, x, source)
	var need *ui.ScriptFormNeeded
	if err := s.SetTextsInside(true); !errors.As(err, &need) || need.Fold || s.TextsInside() {
		t.Fatalf("the switch on an original map: %v", err)
	}
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTextsInside(true); err != nil || !s.TextsInside() {
		t.Fatalf("the switch after the extension: %v", err)
	}
	refused := filepath.Join(dir, "other.alm")
	if _, err := s.SaveAs(refused); err == nil || errors.As(err, &need) {
		t.Fatalf("a save under an .alm name: %v", err)
	}
	if _, err := os.Stat(refused); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("the refused save wrote a file")
	}
	target, err := s.SaveInPlace()
	if err != nil || filepath.Ext(target) != ".almx" {
		t.Fatalf("the save in place wrote %s: %v", target, err)
	}
	if b, _ := os.ReadFile(source); !bytes.Equal(b, original) {
		t.Fatal("the save in place changed the .alm")
	}
	held := pagesOf(t, target)
	if len(held) != 0 {
		t.Fatalf("a map with no pages holds %q", held)
	}
	b := withInside(t, insideMap(t, true), insidePages())
	if extendedName(b, "m.alm") == nil || extendedName(b, "m.almx") != nil {
		t.Fatal("the map writer's name rule")
	}
	if plain, _ := alm.WithoutTextsRecord(b); extendedName(append([]byte(nil), plain...), "m.alm") == nil {
		t.Fatal("the script record under an .alm name passed")
	}
	rows := x.CheckModMap(mod.Mission{Map: "maps/m.alm", MapBytes: b}, "")
	if !strings.Contains(findingText(rows), x.word("editor.message.almxname")) {
		t.Fatalf("the check of the record under an .alm name: %s", findingText(rows))
	}
}

func findingText(rows []ui.InspectionFinding) string {
	var out []string
	for _, r := range rows {
		out = append(out, r.Rule+": "+r.Text)
	}
	return strings.Join(out, "\n")
}

// insideMod writes a mod named m whose mission 25 plays maps/bridge.almx:
// with the pages in the record when inside, else in its folder.
func insideMod(t *testing.T, inside bool) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "m")
	b := insideMap(t, true)
	if inside {
		b = withInside(t, b, insidePages())
	} else {
		for name, page := range insidePages() {
			writeTestPlayInput(t, filepath.Join(dir, "maps", "bridge.text", filepath.FromSlash(name)), page)
		}
	}
	writeTestPlayInput(t, filepath.Join(dir, "maps", "bridge.almx"), b)
	writeTestPlayInput(t, filepath.Join(dir, mod.ManifestName), []byte("id = \"m\"\ntitle = \"Inside\"\nversion = \"1\"\n"))
	manifest := strings.NewReplacer("MAP", "maps/bridge.almx", "title    = \"t\"\n", "", "briefing = \"b\"\n", "").Replace(guestAlmxManifest)
	writeTestPlayInput(t, filepath.Join(dir, filepath.FromSlash(mod.MissionsFile)), []byte(manifest))
	return dir
}

// insidePlay is what mission 25 shows when the mod at dir is loaded in
// lang: the briefing, the page served for message 1 and the parts the
// message trigger shows when it fires.
type insidePlay struct {
	Pages    map[string][]byte
	Briefing string
	Served   []byte
	Parts    []string
	Findings string
}

func playInside(t *testing.T, dir, lang string) insidePlay {
	t.Helper()
	src, _, err := mod.ReadInside(dir, mod.MissionsFile)
	if err != nil {
		t.Fatal(err)
	}
	d, err := mod.ParseMissions("m", mod.MissionsFile, src, nil, lang)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Missions[0].ReadFiles(dir); err != nil {
		t.Fatal(err)
	}
	m := d.Missions[0]
	var play insidePlay
	play.Pages = m.TextFiles
	play.Findings = findingText(newTestPlayFixture(t).x.CheckModMap(m, ""))
	f := guestFixtureFront(t)
	if _, err := f.SetModMissions(d); err != nil {
		t.Fatal(err)
	}
	play.Briefing = MissionObjectivesFor(f.Archives.Containers, base.GameROM1, 25)
	play.Served, _ = f.Archives.Containers.ReadFile(missionEvent(t, 25, 1))
	w, raises := announceWorld(t, 1)
	p := &PlayWorld{World: w, Party: []sim.EntityID{0}}
	p.watchAnnouncements(&Mission{Number: 25, World: w, Raises: raises}, f.Archives.Containers, nil)
	for range 32 {
		p.step(nil)
	}
	if len(p.announced) != 1 {
		t.Fatalf("%d announcements", len(p.announced))
	}
	play.Parts = p.announced[0].Parts
	return play
}

// A mod whose map holds its pages inside passes the mod check, and its
// mission shows the same briefing and message text as the mod that holds
// them in the folder, in EN and RU, read from the folder and packed alike.
// The same map with neither the record nor the folder still has its
// message without a page reported.
func TestTextsInsideModMissionShowsItsPages(t *testing.T) {
	x := newTestPlayFixture(t).x
	inside, folder := insideMod(t, true), insideMod(t, false)
	m, b, err := mod.Pack(inside)
	if err != nil {
		t.Fatal(err)
	}
	packed := filepath.Join(t.TempDir(), mod.PackedName(m.ID))
	if err := os.WriteFile(packed, b, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, lang := range []string{"en", "ru"} {
		got, want, res := playInside(t, inside, lang), playInside(t, folder, lang), playInside(t, packed, lang)
		if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(got, res) {
			t.Fatalf("%s: inside %+v\nfolder %+v\npacked %+v", lang, got, want, res)
		}
		if strings.Contains(got.Findings, ruleTextsFolder) || strings.Contains(got.Findings, "event01.txt") ||
			strings.Contains(got.Findings, x.word("editor.message.almxname")) || !bytes.Equal(got.Pages["briefing.txt"], insidePages()[lang+"/briefing.txt"]) ||
			!strings.Contains(got.Briefing, map[string]string{"en": "Hold the bridge.", "ru": ""}[lang]) || got.Briefing == "" || len(got.Served) == 0 || len(got.Parts) != 1 {
			t.Fatalf("%s: %+v", lang, got)
		}
		if lang == "en" && strings.TrimSpace(got.Parts[0]) != "They come." {
			t.Fatalf("the message shows %q", got.Parts)
		}
	}
	bare := filepath.Join(t.TempDir(), "m")
	writeTestPlayInput(t, filepath.Join(bare, "maps", "bridge.almx"), insideMap(t, true))
	missing := mod.Mission{Map: "maps/bridge.almx", Kind: mod.MissionReplace, Number: 25}
	if err := missing.ReadFiles(bare); err != nil {
		t.Fatal(err)
	}
	if rows := findingText(x.CheckModMap(missing, "")); !strings.Contains(rows, "event01.txt") {
		t.Fatalf("a message with no page is not reported: %s", rows)
	}
}
