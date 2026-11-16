package game

import (
	"bytes"
	"fmt"
	"image/color"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

func charactersPage(t *testing.T, lang string) (*MapInspector, *mapInspectionEdits, string) {
	t.Helper()
	x, s := contextEditor(t)
	x.setEditorMod(nil)
	x.archives.Base.Profile.Language = map[string]string{"en": "english", "ru": "russian"}[lang]
	entry := map[string]string{"en": "english 0", "ru": "russian 1"}[lang]
	writeTestPlayInput(t, filepath.Join(x.archives.Root, MainArchive), synth.Archive([]synth.File{{Path: "id", Data: []byte(entry)}}))
	for _, name := range []string{GraphicsArchive, WorldArchive, MoviesArchive} {
		writeTestPlayInput(t, filepath.Join(x.archives.Root, name), synth.Archive(nil))
	}
	if got := BaseID(InspectInstall(x.archives.Root)); got != "rom1-"+lang {
		t.Fatalf("fixture base %q", got)
	}
	if lang == "ru" {
		x.font = &text.Font{Selector: text.SelectorConverting}
	}
	table := fixtureItemTable(10, 5, 5)
	rows := dbCollection{{}}
	for i := 1; i <= 4; i++ {
		p := humansParams(int32(30+i), 0, 2)
		p[0], p[1], p[2], p[3] = int32(30+i), 15, 10, 12
		p[data.HumanSlotTypeID], p[data.HumanSlotGender], p[0x18] = 20, 0, int32(100+i)
		rows = append(rows, dbEntry{name: fmt.Sprintf("Guard_%d", i), params: p})
	}
	table.Humans = rows
	x.table = table
	dir, _ := data.FigureFor(20, 2, 0)
	for _, face := range []int{2, 7} {
		pic := invSparseSheet(color.RGBA{R: byte(face * 20), G: 0x40, A: 255}, 0)
		x.archives.Containers.Overlay(graphicsPrefix+data.ItemFigureBasePath(dir, face), func() ([]byte, error) { return pic, nil })
	}
	folder := filepath.Join(t.TempDir(), "authored-map")
	doc, err := s.SetWorkspaceField("package", "folder", folder)
	if err != nil || doc == nil {
		t.Fatalf("new mod: %v", err)
	}
	return x, doc.Edits.(*mapInspectionEdits), folder
}

func characterField(t *testing.T, s *mapInspectionEdits, key string) ui.InspectionField {
	t.Helper()
	for _, f := range s.WorkspaceFields(ui.ModCharactersScope) {
		if f.Key == key {
			return f
		}
	}
	t.Fatalf("no character field %s", key)
	return ui.InspectionField{}
}

func setCharacter(t *testing.T, s *mapInspectionEdits, key, value string) {
	t.Helper()
	steps := len(s.undoWorkspace)
	if _, err := s.SetWorkspaceField(ui.ModCharactersScope, key, value); err != nil {
		t.Fatalf("%s = %q: %v", key, value, err)
	}
	if len(s.undoWorkspace) != steps+1 {
		t.Fatalf("%s made %d undo steps", key, len(s.undoWorkspace)-steps)
	}
}

func addCharacter(t *testing.T, s *mapInspectionEdits, action string) {
	t.Helper()
	if fields := s.WorkspaceFields(ui.ModCharactersScope); len(fields) == 1 && fields[0].Key == "begin" {
		before := workspaceBytes(s.workspace)
		if _, err := s.SetWorkspaceField(ui.ModCharactersScope, "begin", ""); err != nil || !bytes.Equal(before, workspaceBytes(s.workspace)) {
			t.Fatalf("open character choices: %v", err)
		}
	}
	f := characterField(t, s, action)
	if len(f.Choices) < 2 {
		t.Fatal("no game characters offered")
	}
	setCharacter(t, s, action, strconv.FormatInt(f.Choices[1].Value, 10))
}

func TestModCharactersPageWritesNamesFacesAndGameEdits(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			x, s, folder := charactersPage(t, lang)
			before := tree(t, folder)
			addCharacter(t, s, "new")
			setCharacter(t, s, "character.0.name.en", "Visitor")
			setCharacter(t, s, "character.0.name.ru", "Гостья")
			face := characterField(t, s, "character.0.face")
			if len(face.Choices) != 3 || face.Icons[0] == nil || face.Icons[7] == nil {
				t.Fatalf("face choices: %+v, icons %v", face.Choices, face.Icons)
			}
			setCharacter(t, s, "character.0.face", "7")
			rows, err := s.characterRows(s.workspace)
			if err != nil || len(rows.Rows) != 1 || rows.Rows[0].Key == "" || rows.Rows[0].Row != "Guard_1" || rows.Rows[0].Face != 7 {
				t.Fatalf("new character: %+v, %v", rows, err)
			}
			for _, l := range []string{"en", "ru"} {
				look, err := s.workspace.Project.textLookup(l)
				if err != nil {
					t.Fatal(err)
				}
				back, err := mod.ParseCharacters("m", mod.CharactersFile, s.workspace.Project.Files[mod.CharactersFile], look, l)
				want := map[string]string{"en": "Visitor", "ru": "Гостья"}[l]
				if err != nil || back.Rows[0].Name != want {
					t.Fatalf("%s name: %+v, %v", l, back, err)
				}
			}
			if len(s.modCharacters()) != 1 || s.modCharacters()[0].Name == "" {
				t.Fatal("mission party has no new character")
			}
			addCharacter(t, s, "change")
			setCharacter(t, s, "character.1.name.en", "Game guard")
			setCharacter(t, s, "character.1.face", "7")
			rows, err = s.characterRows(s.workspace)
			if err != nil || rows.Rows[1].Target != "Guard_1" || rows.Rows[1].Key != "" || x.table.Humans.EntryParams(1)[data.HumanSlotFace] != 7 {
				t.Fatalf("game edit: %+v, %v", rows, err)
			}
			if !reflect.DeepEqual(before, tree(t, folder)) {
				t.Fatal("editing wrote the folder before Ctrl+S")
			}
			if _, err := s.SaveIntoMod(); err != nil {
				t.Fatal(err)
			}
			if tree(t, folder)[mod.CharactersFile] != string(s.workspace.Project.Files[mod.CharactersFile]) {
				t.Fatal("project save lost characters")
			}
		})
	}
}

func TestModCharactersPageNameKeepsOtherTextUses(t *testing.T) {
	_, s, _ := charactersPage(t, "en")
	addCharacter(t, s, "new")
	rows, err := s.characterRows(s.workspace)
	if err != nil {
		t.Fatal(err)
	}
	key := rows.Rows[0].NameKey
	setCharacter(t, s, "character.0.name.en", "Visitor")
	rows, err = s.characterRows(s.workspace)
	if err != nil || rows.Rows[0].NameKey != key {
		t.Fatalf("unshared name changed its text key: %+v, %v", rows, err)
	}
	s.workspace.Project.Missions = append(s.workspace.Project.Missions, mod.Mission{Key: "escort", Number: 77, Title: "Visitor", TitleKey: key})
	setCharacter(t, s, "character.0.name.en", "Guard")
	rows, err = s.characterRows(s.workspace)
	if err != nil || rows.Rows[0].NameKey == key {
		t.Fatalf("shared name kept its text key: %+v, %v", rows, err)
	}
	sets, err := s.workspace.Project.strings()
	if err != nil || sets["en"][key] != "Visitor" || sets["en"][rows.Rows[0].NameKey] != "Guard" {
		t.Fatalf("character edit changed another use of its text: %+v, %v", sets, err)
	}
	if r := s.workspace.Project.Missions[len(s.workspace.Project.Missions)-1]; r.TitleKey != key || r.Title != "Visitor" {
		t.Fatalf("character edit changed the mission's title: %+v", r)
	}
}

func TestModCharactersPageDeleteRefusalUndoAndInvalidEdits(t *testing.T) {
	_, s, _ := charactersPage(t, "en")
	initial := workspaceBytes(s.workspace)
	addCharacter(t, s, "new")
	if _, err := s.Undo(); err != nil || !bytes.Equal(initial, workspaceBytes(s.workspace)) {
		t.Fatalf("add undo: %v", err)
	}
	if _, err := s.Redo(); err != nil {
		t.Fatal(err)
	}
	setCharacter(t, s, "character.0.name.en", "Visitor")
	if _, err := s.Undo(); err != nil || characterField(t, s, "character.0.name.en").TextValue != "Character 1" {
		t.Fatalf("name undo: %v", err)
	}
	if _, err := s.Redo(); err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{"character.0.face": "99", "character.0.name.en": "", "character.0.name.ru": strings.Repeat("я", 41), "character.99.remove": ""} {
		before := workspaceBytes(s.workspace)
		if _, err := s.SetWorkspaceField(ui.ModCharactersScope, key, value); err == nil || !bytes.Equal(before, workspaceBytes(s.workspace)) {
			t.Fatalf("invalid %s accepted or changed draft: %v", key, err)
		}
	}
	rows, _ := s.characterRows(s.workspace)
	key := rows.Rows[0].Key
	s.workspace.Project.Missions = append(s.workspace.Project.Missions, mod.Mission{Key: "escort", Number: 77, Title: "Escort", Party: []mod.GuestMember{{Character: key}}})
	before := workspaceBytes(s.workspace)
	if _, err := s.SetWorkspaceField(ui.ModCharactersScope, "character.0.remove", ""); err == nil || !strings.Contains(err.Error(), "Escort (77)") || !bytes.Equal(before, workspaceBytes(s.workspace)) {
		t.Fatalf("guest delete: %v", err)
	}
	s.workspace.Project.Missions = s.workspace.Project.Missions[:len(s.workspace.Project.Missions)-1]
	setCharacter(t, s, "character.0.remove", "")
	if rows, err := s.characterRows(s.workspace); err != nil || !rows.Empty() || len(s.modCharacters()) != 0 {
		t.Fatalf("delete left character: %+v, %v", rows, err)
	}
	if _, err := s.Undo(); err != nil || len(s.modCharacters()) != 1 {
		t.Fatalf("delete undo: %v", err)
	}
}

func TestModCharactersPageFieldsHavePlainLabelsAndHints(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		_, s, _ := charactersPage(t, lang)
		fields := s.WorkspaceFields(ui.ModCharactersScope)
		if len(fields) != 1 || fields[0].Key != "begin" || !fields[0].Action || fields[0].Meaning == "" {
			t.Fatalf("%s empty page: %+v", lang, fields)
		}
		if _, err := s.SetWorkspaceField(ui.ModCharactersScope, "begin", ""); err != nil {
			t.Fatal(err)
		}
		fields = append(fields, s.WorkspaceFields(ui.ModCharactersScope)...)
		addCharacter(t, s, "new")
		addCharacter(t, s, "change")
		fields = append(fields, s.WorkspaceFields(ui.ModCharactersScope)...)
		for _, key := range []string{"begin", "new", "change", "cancel", "character.0.name.en", "character.0.name.ru", "character.0.face", "character.0.remove", "character.1.name.en", "character.1.name.ru", "character.1.face", "character.1.remove"} {
			if !slices.ContainsFunc(fields, func(f ui.InspectionField) bool { return f.Key == key }) {
				t.Errorf("%s: no field %s checked", lang, key)
			}
		}
		for _, f := range fields {
			if strings.TrimSpace(f.Meaning) == "" || strings.HasPrefix(f.Meaning, "editor.") || len(f.Lines) != 0 || f.ReadOnly {
				t.Errorf("%s field %s has no hint or stands as description", lang, f.Key)
			}
			if strings.ContainsRune(strings.ToLower(f.Meaning+f.Label), 'ё') {
				t.Errorf("%s field %s uses a forbidden letter", lang, f.Key)
			}
			for _, term := range []string{"row", "строк", "key", "ключ", "catalog", "каталог", "target", "npc<", "strip", "лент"} {
				if strings.Contains(strings.ToLower(f.Label), term) {
					t.Errorf("%s label %q contains %s", lang, f.Label, term)
				}
			}
		}
	}
}

func TestModCharactersPageGameFaceEditKeepsTheGameName(t *testing.T) {
	x, s, _ := charactersPage(t, "en")
	addCharacter(t, s, "change")
	setCharacter(t, s, "character.0.face", "7")
	rows, err := s.characterRows(s.workspace)
	if err != nil || len(rows.Rows) != 1 || rows.Rows[0].Target != "Guard_1" || rows.Rows[0].NameKey != "" || rows.Rows[0].Face != 7 {
		t.Fatalf("face-only game edit: %+v, %v", rows, err)
	}
	if _, ok := x.table.Mods.CharacterName("Guard_1"); ok {
		t.Fatal("changing a game face changed its name")
	}
	if _, err := s.Undo(); err != nil || x.table.Humans.EntryParams(1)[data.HumanSlotFace] != 2 {
		t.Fatalf("face undo: %v", err)
	}
	setCharacter(t, s, "character.0.remove", "")
	if fields := s.WorkspaceFields(ui.ModCharactersScope); len(fields) != 1 || fields[0].Key != "begin" {
		t.Fatalf("game edit deletion: %+v", fields)
	}
}

func TestModCharactersPageGuestsBuildAtMissionLevelsWithEquipment(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			x, s, _ := charactersPage(t, lang)
			addCharacter(t, s, "new")
			setCharacter(t, s, "character.0.name.en", "Visitor")
			setCharacter(t, s, "character.0.name.ru", "Гостья")
			setCharacter(t, s, "character.0.face", "7")
			c := s.modCharacters()[0]
			if _, err := s.SetWorkspaceField("package", "landing", "1"); err != nil {
				t.Fatal(err)
			}
			offered := false
			for _, field := range s.WorkspaceFields("package") {
				if field.Key == "party.0.character" {
					offered = slices.ContainsFunc(field.Choices, func(v ui.InspectionChoice) bool { return strings.Contains(v.Label, c.Key) })
				}
			}
			if !offered {
				t.Fatal("the Missions page does not offer the new character")
			}
			weapon, shield := "Dagger", "Buckler"
			for _, level := range []int{1, 4} {
				g := mod.GuestMember{Character: c.Key, Level: level, Weapon: &weapon, Shield: &shield, Armour: []string{"Cap"}}
				f := &FrontEnd{InstallResources: InstallResources{Table: x.table, Humans: x.table.Humans, Archives: x.archives, Font: resolved(x.font, nil)}}
				party, err := f.guestParty(mod.Mission{Party: []mod.GuestMember{g}}, 30, nil, nil)
				if err != nil || len(party) != 1 {
					t.Fatalf("level %d: %+v, %v", level, party, err)
				}
				p := party[0]
				want := map[string]string{"en": "Visitor", "ru": "Гостья"}[lang]
				name, err := encodeSaveLabel(want, f.textSelector())
				if err != nil || p.Name != name {
					t.Fatalf("name %q, want %q: %v", p.Name, name, err)
				}
				eq := mapload.EquipmentFromParty(p)
				w, _ := eq.Code(1)
				shield, _ := eq.Code(2)
				helm, _ := eq.Code(6)
				if p.FigureFace != 7 || int(p.Hero.Body) != 30+level || p.ID != "char:"+c.Key || p.Weapon == nil || w == 0 || shield == 0 || helm == 0 {
					t.Fatalf("level %d guest: %+v", level, p)
				}
				if !slices.ContainsFunc(s.modCharacters(), func(v mod.Character) bool { return v.Key == c.Key }) {
					t.Fatal("character missing from mission offer")
				}
			}
		})
	}
}

func TestModCharactersPageGameRenameChangesOnlyTheTypedLanguage(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, typed := range []string{"en", "ru"} {
			t.Run(lang+"/"+typed, func(t *testing.T) {
				_, s, _ := charactersPage(t, lang)
				addCharacter(t, s, "change")
				setCharacter(t, s, "character.0.name."+typed, "Ivan")
				rows, err := s.characterRows(s.workspace)
				if err != nil || len(rows.Rows) != 1 || rows.Rows[0].NameKey == "" {
					t.Fatalf("rename: %+v, %v", rows, err)
				}
				sets, err := s.workspace.Project.strings()
				if err != nil {
					t.Fatal(err)
				}
				for _, l := range []string{"en", "ru"} {
					got, held := sets[l][rows.Rows[0].NameKey]
					other := l != typed && (l == "en" || l == lang)
					if l == typed && got != "Ivan" || l != typed && other && got != "Guard_1" || l != typed && !other && held {
						t.Fatalf("%s text after typing %s: %q held %v", l, typed, got, held)
					}
				}
			})
		}
	}
}

func TestModCharactersPageNewCharactersTakeTheRecordKeyRule(t *testing.T) {
	_, s, _ := charactersPage(t, "en")
	addCharacter(t, s, "new")
	addCharacter(t, s, "new")
	rows, err := s.characterRows(s.workspace)
	if err != nil || len(rows.Rows) != 2 || rows.Rows[0].Key == rows.Rows[1].Key {
		t.Fatalf("two new characters: %+v, %v", rows, err)
	}
	if want := mod.RecordKey("character", func(k string) bool { return k == rows.Rows[0].Key }); rows.Rows[1].Key != want {
		t.Fatalf("second key %q, the rule makes %q", rows.Rows[1].Key, want)
	}
}
