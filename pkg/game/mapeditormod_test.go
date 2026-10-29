package game

import (
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeEditorProjectFixture(t *testing.T, e *ui.MapEditor) string {
	t.Helper()
	dir := writeEditorMod(t, t.TempDir(), "[[mission]]\nkey='first'\nnumber=25\nmap='first.alm'\nafter=20\nparty=[{character='npc22'}]\n[[mission]]\nkey='second'\nnumber=26\nmap='second.alm'\nafter=25\nparty=[{character='sellsword'}]\n")
	for _, name := range []string{"first.alm", "second.alm"} {
		writeTestPlayInput(t, filepath.Join(dir, name), e.Document().SourceBytes)
	}
	writeTestPlayInput(t, filepath.Join(dir, "main.star"), []byte("def init(game, settings):\n    game.data.add('data/characters.toml')\n    game.data.add('data/missions.toml')\n"))
	writeTestPlayInput(t, filepath.Join(dir, "data/characters.toml"), []byte("[[character]]\nkey='sellsword'\nrow='NPC06'\nname='sellsword.name'\n"))
	writeTestPlayInput(t, filepath.Join(dir, "text/en/strings.toml"), []byte("\"sellsword.name\"='Sellsword'\n"))
	writeTestPlayInput(t, filepath.Join(dir, ".author"), []byte("retained author note"))
	return dir
}

func TestEditorTestPlayRetainsTheWholeProjectAndUsesItsID(t *testing.T) {
	f, e := workspaceFixture(t)
	dir := writeEditorProjectFixture(t, e)
	before := tree(t, dir)
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	if _, err := s.SetWorkspaceField("package", "gold", "73"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.TestPlay(); err != nil {
		t.Fatal(err)
	}
	private := filepath.Join(f.profile, TestPlayModID, "ambush")
	r, err := modrt.Load([]mod.Entry{mod.Load(private)}, "rom1-en", nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Missions.Missions) != 2 || r.Missions.Missions[0].Gold != 73 || r.Missions.Missions[1].After != 25 || len(r.Characters.Rows) != 1 {
		t.Fatal("test play lost declarations", r.Missions, r.Characters)
	}
	if tree(t, private)[".author"] != before[".author"] || !reflect.DeepEqual(tree(t, dir), before) || !s.Dirty() {
		t.Fatal("test play changed source or checkpoint")
	}
	if len(f.started) != 1 || !strings.Contains(strings.Join(f.started[0], "|"), "-mods|ambush|") {
		t.Fatal("wrong project launched", f.started)
	}
	writeTestPlayInput(t, filepath.Join(private, "foreign.txt"), []byte("not owned"))
	privateBefore := tree(t, private)
	if _, err := s.TestPlay(); err == nil || len(f.started) != 1 || !reflect.DeepEqual(privateBefore, tree(t, private)) {
		t.Fatal("test play replaced an unowned file")
	}
}

func TestEditorPartyRowsPreserveTemplateAndEmptyOnSaveAndReopen(t *testing.T) {
	f, e := workspaceFixture(t)
	dir := writeEditorProjectFixture(t, e)
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	f.x.mod.catalog = append(f.x.mod.catalog, mod.Character{Key: "npc22", Name: "Companion"})
	var named bool
	for _, field := range s.WorkspaceFields("package") {
		if field.Key == "party.0.character" {
			for _, c := range field.Choices {
				named = named || strings.Contains(c.Label, "Sellsword")
			}
		}
	}
	if !named {
		t.Fatal("declared character is absent from named picker")
	}
	for _, edit := range [][2]string{{"party.0.character", "sellsword"}, {"party.0.level", "2"}, {"party.0.weapon", "0"}, {"party.0.shield", "-1"}, {"party.0.armour", "1"}, {"party.0.add", ""}} {
		if _, err := s.SetWorkspaceField("package", edit[0], edit[1]); err != nil {
			t.Fatal(edit, err)
		}
	}
	if _, err := s.SetWorkspaceField("package", "party.1.character", "sellsword"); err == nil {
		t.Fatal("duplicate character accepted")
	}
	if _, err := s.SetWorkspaceField("package", "party.1.up", ""); err != nil {
		t.Fatal(err)
	}
	if s.workspace.Package.Party[1].Character != "sellsword" {
		t.Fatal("member move lost equipment row")
	}
	copyDir := filepath.Join(t.TempDir(), "ambush")
	if _, err := s.saveWorkspacePackage(copyDir); err != nil {
		t.Fatal(err)
	}
	m := loadWorkspaceMod(t, f.root, copyDir, 0)
	g := m.Party[1]
	if len(m.Party) != 2 || g.Character != "sellsword" || g.Level != 2 || g.Weapon == nil || *g.Weapon != "" || g.Shield != nil || g.Armour == nil || len(g.Armour) != 0 {
		t.Fatal("optional equipment changed", m.Party)
	}
	for len(s.workspace.Package.Party) < mod.MaxGuestParty {
		n := len(s.workspace.Package.Party)
		s.inspector.mod.catalog = append(s.inspector.mod.catalog, mod.Character{Key: fmt.Sprintf("extra%d", n)})
		if _, err := s.SetWorkspaceField("package", "party.0.add", ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.SetWorkspaceField("package", "party.0.add", ""); err == nil {
		t.Fatal("ninth party member accepted")
	}
}

func TestEditorPackageAdmissionRefusesAnUnresolvedPartyBeforeWritingOrStarting(t *testing.T) {
	f, e := workspaceFixture(t)
	dir := writeEditorProjectFixture(t, e)
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	if _, err := s.SetWorkspaceField("package", "party.0.character", "missing"); err != nil {
		t.Fatal(err)
	}
	f.x.table = characterFront().Table
	before := f.x.table.Humans
	parent := t.TempDir()
	for name, action := range map[string]func() (string, error){"Save": func() (string, error) { return s.saveWorkspacePackage(filepath.Join(parent, "ambush")) }, "F5": s.TestPlay} {
		if _, err := action(); err == nil || !strings.Contains(err.Error(), "not in the character catalog") {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if len(tree(t, parent)) != 0 || len(tree(t, f.profile)) != 0 || len(f.started) != 0 || !reflect.DeepEqual(before, f.x.table.Humans) || len(f.x.table.Mods.Characters) != 0 {
		t.Fatal("refused admission wrote files, started, or changed the inspector table")
	}
}

func TestEditorImportedPartySurvivesWorkspaceProjection(t *testing.T) {
	f, e := workspaceFixture(t)
	dir := writeEditorMod(t, t.TempDir(), "[[mission]]\nkey = 'first'\nnumber = 25\nmap = 'map.alm'\nafter = 20\nparty = [{character = 'npc22', weapon = '', armour = []}]\n")
	writeTestPlayInput(t, filepath.Join(dir, "map.alm"), e.Document().SourceBytes)
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	m := s.workspace.Package.mission(s.model.Bytes(), s.workspace.Pages)
	if len(m.Party) != 1 || m.Party[0].Weapon == nil || m.Party[0].Armour == nil {
		t.Fatalf("workspace discarded party: members=%d guest=%d", len(m.Party), m.Guest)
	}
	_ = f
}

func TestEditorModMissionOrderAndHistorySurviveMapSwitches(t *testing.T) {
	_, e := workspaceFixture(t)
	folder := filepath.Join(t.TempDir(), "authored-map")
	if err := workspaceField(e, "package", 0, folder); err != nil {
		t.Fatal(err)
	}
	loose := filepath.Join(t.TempDir(), "second.alm")
	writeTestPlayInput(t, loose, e.Document().SourceBytes)
	_ = e.Key("escape")
	if err := e.Open(loose); err != nil {
		t.Fatal(err)
	}
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	if !e.FocusModField("add") {
		t.Fatal("missing Add open map")
	}
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	if len(s.workspace.Project.Missions) != 2 {
		t.Fatal("Add did not snapshot the open map")
	}
	if err := e.Key("saveas"); err != nil {
		t.Fatal(err)
	}
	second := s.workspace.Project.Missions[1]
	if err := e.Key("cardup"); err != nil {
		t.Fatal(err)
	}
	if got := s.workspace.Project.Missions[0]; got.Number != second.Number || got.After != second.After {
		t.Fatal("list move relinked the chain")
	}
	if err := e.Key("undo"); err != nil {
		t.Fatal(err)
	}
	if s.workspace.Project.Missions[1].Number != second.Number {
		t.Fatal("undo did not restore order")
	}
	if err := workspaceField(e, "package", 9, "1"); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(e, "package", 8, "73"); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(e, "package", 9, "2"); err != nil {
		t.Fatal(err)
	}
	if !e.Dirty() {
		t.Fatal("switch hid a pending edit")
	}
	if err := e.Key("undo"); err != nil {
		t.Fatal(err)
	}
	if s.workspace.Package.Gold != 0 || s.workspace.Project.Selected != 0 {
		t.Fatal("undo did not restore the edited mission")
	}
	if err := e.Key("redo"); err != nil {
		t.Fatal(err)
	}
	if s.workspace.Project.Missions[0].Gold != 73 {
		t.Fatal("redo lost the edit")
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if err := e.Key("undo"); err != nil {
		t.Fatal(err)
	}
	if !e.Dirty() {
		t.Fatal("undo after Save did not mark project dirty")
	}
}

func TestEditorAddOpenMapRetainsBothLanguageTreesAndItsPendingPage(t *testing.T) {
	for _, attached := range []bool{false, true} {
		t.Run(fmt.Sprintf("attached=%t", attached), func(t *testing.T) {
			f, e := workspaceFixture(t)
			working := filepath.Join(t.TempDir(), "authored-map")
			if attached {
				if err := workspaceField(e, "package", 0, working); err != nil {
					t.Fatal(err)
				}
			}
			loose := filepath.Join(t.TempDir(), "second.alm")
			writeTestPlayInput(t, loose, e.Document().SourceBytes)
			for name, body := range map[string]string{"en/event01.txt": "English", "ru/event01.txt": "Russian", "title.txt": "Common"} {
				writeTestPlayInput(t, filepath.Join(mapTextFolder(loose), filepath.FromSlash(name)), []byte(body))
			}
			before := tree(t, filepath.Dir(loose))
			if e.ModWindowOpen() {
				_ = e.Key("escape")
			}
			if err := e.Open(loose); err != nil {
				t.Fatal(err)
			}
			workspacePage(t, e, "event01.txt", "Pending English")
			s := e.Document().Edits.(*mapInspectionEdits)
			if _, err := s.SetWorkspaceField("package", "add", ""); err != nil {
				t.Fatal(err)
			}
			if _, err := s.saveWorkspacePackage(working); err != nil {
				t.Fatal(err)
			}
			index := 0
			if attached {
				index = 1
			}
			m := loadWorkspaceMod(t, f.root, working, index)
			if string(m.TextFiles["event01.txt"]) != "Pending English" || string(m.TextFiles["title.txt"]) != "Common" {
				t.Fatal("Add lost open-map pages", m.TextFiles)
			}
			root := strings.TrimSuffix(m.Texts, "/en")
			if tree(t, working)[root+"/ru/event01.txt"] != "Russian" || !reflect.DeepEqual(before, tree(t, filepath.Dir(loose))) {
				t.Fatal("Add lost the other language or changed source")
			}
		})
	}
}

func TestEditorMissionSelectionResolvesPagesForDeclarationsSharingAMap(t *testing.T) {
	_, e := workspaceFixture(t)
	dir := writeEditorMod(t, t.TempDir(), "[[mission]]\nkey='first'\nnumber=25\nmap='shared.alm'\ntexts='one'\nafter=20\nparty=[{character='npc22'}]\n[[mission]]\nkey='second'\nnumber=26\nmap='shared.alm'\ntexts='two'\nafter=25\nparty=[{character='npc22'}]\n")
	writeTestPlayInput(t, filepath.Join(dir, "shared.alm"), e.Document().SourceBytes)
	writeTestPlayInput(t, filepath.Join(dir, "one/title.txt"), []byte("One"))
	writeTestPlayInput(t, filepath.Join(dir, "two/title.txt"), []byte("Two"))
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	if _, err := s.selectWorkspaceMission(2); err != nil {
		t.Fatal(err)
	}
	if string(s.workspace.Pages["title.txt"]) != "Two" {
		t.Fatal("shared map selected the other declaration's pages")
	}
	if _, err := s.selectWorkspaceMission(1); err != nil {
		t.Fatal(err)
	}
	if string(s.workspace.Pages["title.txt"]) != "One" {
		t.Fatal("shared map did not restore its own pages")
	}
}

func TestEditorRemovingAMissionOpensItsSuccessorWithoutOverwritingItsMap(t *testing.T) {
	_, e := workspaceFixture(t)
	first := bytes.Clone(e.Document().SourceBytes)
	second := bytes.Clone(first)
	offset := inspectionSectionOffsets(t, second)[0]
	copy(second[offset+0x30:], []byte("Second"))
	dir := writeEditorMod(t, t.TempDir(), "[[mission]]\nkey='first'\nnumber=25\nmap='first.alm'\nafter=20\nparty=[{character='npc22'}]\n[[mission]]\nkey='second'\nnumber=26\nmap='second.alm'\nafter=25\nparty=[{character='npc22'}]\n")
	writeTestPlayInput(t, filepath.Join(dir, "first.alm"), first)
	writeTestPlayInput(t, filepath.Join(dir, "second.alm"), second)
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	if _, err := s.SetWorkspaceField("package", "remove", ""); err == nil {
		t.Fatal("removed a chain link without confirmation")
	}
	doc, err := s.SetWorkspaceField("package", "remove", "confirmed")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(doc.SourceBytes, second) || s.workspace.Package.After != 20 {
		t.Fatal("removal lost the successor map or reconnect")
	}
	copyDir := filepath.Join(t.TempDir(), "ambush")
	if _, err := s.saveWorkspacePackage(copyDir); err != nil {
		t.Fatal(err)
	}
	if got := loadWorkspaceMod(t, s.inspector.archives.Root, copyDir); !bytes.Equal(got.MapBytes, second) {
		t.Fatal("Save replaced the remaining map with the removed map")
	}
}

func TestEditorRemovingOnlyGuestMissionsReconnectsSuccessors(t *testing.T) {
	for _, kind := range []string{"replace", "off", "guest", "guest-without-successor"} {
		t.Run(kind, func(t *testing.T) {
			f, e := workspaceFixture(t)
			declarations := "[[mission]]\nkind = 'replace'\nkey = 'swap'\nnumber = 20\nmap = 'swap.alm'\n"
			after := 20
			if strings.HasPrefix(kind, "guest") {
				declarations = "[[mission]]\nkey = 'swap'\nnumber = 26\nmap = 'swap.alm'\nafter = 20\nparty = [{character = 'npc22'}]\n"
				if kind == "guest" {
					after = 26
				} else {
					declarations = strings.Replace(declarations, "after = 20", "after = 25", 1)
				}
			}
			declarations += fmt.Sprintf("\n[[mission]]\nkey = 'first'\nnumber = 25\nmap = 'first.alm'\nafter = %d\nparty = [{character = 'npc22'}]\n", after)
			dir := writeEditorMod(t, t.TempDir(), declarations)
			for _, name := range []string{"swap.alm", "first.alm"} {
				writeTestPlayInput(t, filepath.Join(dir, name), e.Document().SourceBytes)
			}
			source := bytes.Clone(e.Document().SourceBytes)
			f.x.archives.Containers.Overlay("scenario/20.alm", func() ([]byte, error) { return bytes.Clone(source), nil })
			if err := workspaceField(e, "package", 0, dir); err != nil {
				t.Fatal(err)
			}
			copyDir := filepath.Join(t.TempDir(), "ambush")
			if err := workspaceField(e, "package", 0, copyDir); err != nil {
				t.Fatal(err)
			}
			s := e.Document().Edits.(*mapInspectionEdits)
			if kind == "off" {
				s.workspace.Project.Missions[0].Off = true
				s.workspace.Project.Missions[0].Kind = mod.MissionGuest
				s.workspace.Project.Selected = 0
				s.workspace.Package = modPackageSpec(s.workspace.Project.Missions[0], "ambush", s.workspace.Package.Title)
			}
			_, err := s.SetWorkspaceField("package", "remove", "")
			if kind == "guest" {
				if _, ok := err.(*ui.WorkspaceConfirmation); !ok || len(s.workspace.Project.Missions) != 2 || s.workspace.Project.Missions[1].After != 26 {
					t.Fatalf("guest reconnect did not ask without changing links: %v", err)
				}
				_, err = s.SetWorkspaceField("package", "remove", "confirmed")
			}
			if err != nil {
				t.Fatalf("remove %s: %v", kind, err)
			}
			if len(s.workspace.Project.Missions) != 1 || s.workspace.Package.After != 20 {
				t.Fatal("removal changed the campaign link", s.workspace.Package)
			}
			if _, err := s.SaveIntoMod(); err != nil {
				t.Fatal(err)
			}
			if m := loadWorkspaceMod(t, f.root, copyDir); m.After != 20 {
				t.Fatal("saved successor lost campaign link", m)
			}
		})
	}
}

func TestEditorCopiesTheWholeHandwrittenModWithoutChangingAnySourceFile(t *testing.T) {
	f, e := workspaceFixture(t)
	declarations := "# keep author layout\r\n[[mission]]\r\nkey = 'first'\r\nnumber = 25\r\nmap = 'mission/{lang}/first.alm'\r\nafter = 20\r\nparty = [{character = 'npc22', weapon = '', armour = []}] # empty\r\n\r\n[[mission]]\r\nkey = 'second'\r\nnumber = 26\r\nmap = 'mission/{lang}/second.alm'\r\nafter = 25\r\nparty = [{character = 'sellsword'}]\r\n"
	dir := writeEditorMod(t, t.TempDir(), declarations)
	for _, name := range []string{"mission/en/first.alm", "mission/ru/first.alm", "mission/en/second.alm", "mission/ru/second.alm"} {
		writeTestPlayInput(t, filepath.Join(dir, filepath.FromSlash(name)), e.Document().SourceBytes)
	}
	for name, value := range map[string]string{".author": "hidden note", "data/characters.toml": "[[character]]\nkey = 'sellsword'\nrow = 'NPC06'\n", "mission/text/ru/event01.txt": "Русский", "mission/text/en/event01.txt": "English", "notes.txt": "retain bytes"} {
		writeTestPlayInput(t, filepath.Join(dir, filepath.FromSlash(name)), []byte(value))
	}
	before := tree(t, dir)
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	copyDir := filepath.Join(t.TempDir(), "ambush")
	if err := workspaceField(e, "package", 0, copyDir); err != nil {
		t.Fatal(err)
	}
	for name, want := range before {
		if got := tree(t, copyDir)[name]; got != want {
			t.Errorf("copy changed %s", name)
		}
	}
	if !reflect.DeepEqual(before, tree(t, dir)) {
		t.Fatal("copy changed source")
	}
	if err := workspaceField(e, "package", 8, "73"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Document().Edits.(*mapInspectionEdits).SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(filepath.Join(copyDir, mod.MissionsFile))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(declarations, "party = [{character = 'sellsword'}]", "party = [{character = 'sellsword'}]", 1)
	if !bytes.Contains(written, []byte("gold = 73")) || !bytes.Contains(written, []byte(want[strings.Index(want, "[[mission]]\r\nkey = 'second'"):])) {
		t.Fatal("save lost unselected declarations")
	}
	entries, err := mod.Resolve(filepath.Dir(copyDir), []string{"ambush"})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := modrt.Load(entries, BaseID(InspectInstall(f.root)), nil, modrt.Options{})
	if err != nil || len(loaded.Missions.Missions) != 2 {
		t.Fatalf("normal loader: %v", err)
	}
	writeTestPlayInput(t, filepath.Join(copyDir, "notes.txt"), []byte("external edit"))
	unchanged := tree(t, copyDir)
	if _, err := e.Document().Edits.(*mapInspectionEdits).SaveIntoMod(); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("stale input: %v", err)
	}
	if !reflect.DeepEqual(unchanged, tree(t, copyDir)) {
		t.Fatal("stale save wrote files")
	}
}
