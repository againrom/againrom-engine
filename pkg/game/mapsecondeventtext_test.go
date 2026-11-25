package game

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
	"againrom/pkg/words"
)

// eventTextFixture is the text of mission 10 the fixture install holds: a
// briefing and events 5 and 7.
const eventTextFixture = "#briefing\r\nGo north.\r\n#event5\r\nThe gate opens.\r\nRun!\r\n#event7\r\nNobody reads this.\r\n"

// secondEventTextHost is the fixture second-game map opened as installed
// mission 10, over an install whose main.res holds text, with one action
// that sends event 5.
func secondEventTextHost(t *testing.T, text string) (*mapInspectionEdits, *ui.InspectionDocument) {
	t.Helper()
	x := secondTestInspector(t)
	dir := t.TempDir()
	graphics, main := filepath.Join(dir, GraphicsArchive), filepath.Join(dir, MainArchive)
	for path, b := range map[string][]byte{graphics: synth.Archive(nil), main: synth.Archive([]synth.File{{Path: "text/mission10.txt", Data: []byte(text)}})} {
		if err := os.WriteFile(path, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fsys, err := vfs.Open([]string{graphics, main}, nil)
	if err != nil {
		t.Fatal(err)
	}
	x.archives.Containers = fsys
	d, err := x.inspect(secondFixtureMap(), ui.MapSourceKey(ui.SourceScenario, "10.alm"))
	if err != nil {
		t.Fatal(err)
	}
	host := d.Edits.(*mapInspectionEdits)
	if _, err := host.AddRecord(ui.InspectionNewRecord{Kind: ui.KindAction}); err != nil {
		t.Fatal(err)
	}
	d, err = host.SetFields(ui.KindAction, 0, []ui.InspectionFieldValue{{Key: "value0", Value: 5}})
	if err != nil || d == nil {
		t.Fatalf("send event 5: %v", err)
	}
	return host, d
}

// The pages window of an installed second-game mission lists the sections of
// the mission text the game reads for it, each with the action naming it or
// that no node names it, shows a section's text and edits it; a new key adds
// a section at the end. The send-event action's number is the text field
// whose targets are the event sections (R2-ENGINE-049, R2-ENGINE-050).
func TestSecondMapEditorShowsAndEditsTheMissionTextSections(t *testing.T) {
	host, d := secondEventTextHost(t, eventTextFixture)
	if !host.Offers(reflect.TypeFor[ui.InspectionWorkspace]()) || host.Offers(reflect.TypeFor[ui.InspectionMessages]()) {
		t.Fatal("the second-game host must offer the pages window and no first-game message window")
	}
	if len(host.WorkspaceFields("metadata")) != 0 || len(host.WorkspaceFields("package")) != 0 {
		t.Fatal("the second-game workspace offers a scope other than its pages")
	}
	x := host.inspector
	fields := host.WorkspaceFields("pages")
	if len(fields) != 2 || fields[0].TextValue != "briefing" || fields[1].TextValue != "Go north.\n" {
		t.Fatalf("pages fields %+v", fields)
	}
	var labels []string
	for _, c := range fields[0].Choices {
		labels = append(labels, c.Label)
	}
	unnamed := x.word("editor.text.unnamed")
	if want := []string{"briefing: " + unnamed, "event5: Action 1", "event7: " + unnamed}; !slices.Equal(labels, want) {
		t.Fatalf("section choices %q, want %q", labels, want)
	}
	if kind := d.Script.Catalog.ParameterKind(false, 2, [10]uint32{mapload.ScriptInt}, 0); kind != ui.ScriptMessageParameter {
		t.Fatalf("send event's number has kind %d", kind)
	}
	targets := d.Script.Targets[ui.ScriptMessageParameter]
	if len(targets) != 3 || targets[0].Value != 5 || targets[0].Page != "event5" || targets[0].Text != "The gate opens.\nRun!" || targets[1].Page != "event7" || targets[2].NewChoice != ui.NewMessageChoice {
		t.Fatalf("event targets %+v", targets)
	}
	if target := d.Script.OpenTarget(ui.ScriptMessageParameter, 5); target == nil || len(target.Uses) != 1 {
		t.Fatalf("event 5's target names no using action: %+v", target)
	}

	if _, err := host.SetWorkspaceField("pages", "page", "2"); err != nil {
		t.Fatal(err)
	}
	if body := host.WorkspaceFields("pages")[1].TextValue; body != "The gate opens.\nRun!\n" {
		t.Fatalf("event5 body %q", body)
	}
	if d, err := host.SetWorkspaceField("pages", "body", "The gate stays shut."); err != nil || d == nil {
		t.Fatalf("edit event5: %v", err)
	}
	text := string(host.workspace.Pages["main/text/mission10.txt"])
	if want := "#briefing\r\nGo north.\r\n#event5\r\nThe gate stays shut.\r\n#event7\r\nNobody reads this.\r\n"; text != want {
		t.Fatalf("edited text %q", text)
	}
	if got := secondGameTextSection([]byte(text), "event5"); got != "The gate stays shut.\r\n" {
		t.Fatalf("the game's lookup reads %q", got)
	}
	if !host.Dirty() {
		t.Fatal("a text edit leaves the map clean")
	}
	if _, err := host.SetWorkspaceField("pages", "page", "event9"); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetWorkspaceField("pages", "body", "Late."); err != nil {
		t.Fatal(err)
	}
	if text := string(host.workspace.Pages["main/text/mission10.txt"]); !strings.HasSuffix(text, "Nobody reads this.\r\n#event9\r\nLate.\r\n") {
		t.Fatalf("new section %q", text)
	}
	for _, refused := range []struct{ key, value string }{
		{"body", "A # cuts the section."},
		{"page", "Event1"},
	} {
		if _, err := host.SetWorkspaceField("pages", refused.key, refused.value); err == nil {
			t.Fatalf("%s %q is not refused", refused.key, refused.value)
		}
	}
	// "#event" is found first at "#event5": a write under it would land
	// inside event5's header.
	if _, err := host.SetWorkspaceField("pages", "page", "event"); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetWorkspaceField("pages", "body", "x"); err == nil {
		t.Fatal("a key whose lookup finds another key's section is not refused")
	}
	if _, err := host.Undo(); err != nil {
		t.Fatal(err)
	}
	if text := string(host.workspace.Pages["main/text/mission10.txt"]); strings.Contains(text, "event9") || !strings.Contains(text, "stays shut") {
		t.Fatalf("undo left %q", text)
	}
}

// writeTestModManifest makes dir a second-game mod folder with a maps folder.
func writeTestModManifest(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "maps"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "id = \"" + filepath.Base(dir) + "\"\ntitle = \"T\"\nversion = \"1\"\napplies-to = [\"rom2-en\", \"rom2-ru\"]\n"
	if err := os.WriteFile(filepath.Join(dir, "mod.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}

// An edited mission text is written with the map into the mod the map is
// saved in, at the install path its language folder overrides, as UTF-8; the
// map's bytes are the model's. Saved outside a mod, the map is written, the
// note says the text is not, and the edit stays unsaved. A mod that already
// holds another file there keeps it.
func TestSecondMapEditedMissionTextSavesIntoTheMod(t *testing.T) {
	host, _ := secondEventTextHost(t, eventTextFixture)
	if _, err := host.SetWorkspaceField("pages", "page", "event7"); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetWorkspaceField("pages", "body", "Сундук открыт."); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain", "m10.alm")
	if err := os.MkdirAll(filepath.Dir(plain), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SaveAs(plain); err != nil {
		t.Fatal(err)
	}
	if host.SaveNote() != host.inspector.word("editor.save.eventtext") || !host.Dirty() {
		t.Fatalf("a save outside a mod notes %q, dirty %v", host.SaveNote(), host.Dirty())
	}
	held := filepath.Join(dir, "held")
	writeTestModManifest(t, held)
	other := filepath.Join(held, "files", "en", "main", "text", "mission10.txt")
	if err := os.MkdirAll(filepath.Dir(other), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("#event7\r\nTheirs.\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SaveAs(filepath.Join(held, "maps", "m10.alm")); err == nil || !strings.Contains(err.Error(), "files/en/main/text/mission10.txt") {
		t.Fatalf("a save over another file of the mod is not refused: %v", err)
	}
	if b, _ := os.ReadFile(other); string(b) != "#event7\r\nTheirs.\r\n" {
		t.Fatalf("the mod's own file became %q", b)
	}
	modDir := filepath.Join(dir, "mine")
	writeTestModManifest(t, modDir)
	target, err := host.SaveAs(filepath.Join(modDir, "maps", "m10.alm"))
	if err != nil {
		t.Fatal(err)
	}
	if host.SaveNote() != "" || host.Dirty() {
		t.Fatalf("a save into a mod notes %q, dirty %v", host.SaveNote(), host.Dirty())
	}
	saved, err := os.ReadFile(target)
	if err != nil || string(saved) != string(host.model.Bytes()) {
		t.Fatalf("the saved map is not the model's bytes: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(modDir, "files", "en", "main", "text", "mission10.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "#briefing\r\nGo north.\r\n#event5\r\nThe gate opens.\r\nRun!\r\n#event7\r\nСундук открыт.\r\n"; string(b) != want {
		t.Fatalf("the mod's text %q", b)
	}
}

// The ten 64-byte text fields of a second-game node are named with the
// confidence of the search that found no reader, in both languages
// (R2-ENGINE-1681, Medium).
func TestSecondNodeTextFieldsNameTheirConfidence(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		if label := words.For(lang).Text("editor.node.text"); !strings.Contains(label, "Medium:") {
			t.Errorf("%s: %q", lang, label)
		}
	}
	_, d := secondEventTextHost(t, eventTextFixture)
	texts := d.Script.Actions[0].Texts
	if len(texts) == 0 || texts[0].Key != "text0" || !strings.Contains(texts[0].Label, "Medium:") {
		t.Fatalf("the action's text fields %+v name no confidence", texts)
	}
}
