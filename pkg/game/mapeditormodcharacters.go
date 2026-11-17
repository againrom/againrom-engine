package game

import (
	"errors"
	"fmt"
	"image"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"againrom/pkg/data"
	"againrom/pkg/locale"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

func (s *mapInspectionEdits) characterSources(key, label, meaning string) ui.InspectionField {
	x := s.inspector
	choices, _ := x.identityChoices(s.mission())
	f := ui.InspectionField{Key: key, Label: label, Meaning: meaning,
		Choices: []ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.characters.choose"))}}}
	base := x.contextBase().table
	for _, c := range choices {
		id, ok := x.identityOf(c.Value)
		if !ok || id.Kind == mapload.UnitCreature || id.Row <= 0 || base == nil || base.Humans == nil || id.Row >= base.Humans.Len() {
			continue
		}
		f.Choices = append(f.Choices, c)
	}
	f.Groups, f.Pictures, f.IconTip = x.identityGroups(f.Choices[1:]), s.identityPictures, true
	return f
}

func (s *mapInspectionEdits) characterFace(key string, r mod.CharacterRow) ui.InspectionField {
	x := s.inspector
	f := ui.InspectionField{Key: key, Label: x.word("editor.characters.face"), Meaning: x.word("editor.characters.face.hint"),
		Value: int64(r.Face), Choices: []ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.characters.inherit"))}}}
	t := x.table
	if r.Target != "" {
		t = x.contextBase().table
	}
	if t == nil || t.Humans == nil {
		return f
	}
	name := r.Row
	if r.Target != "" {
		name = r.Target
	}
	rows := characterRows(t.Humans, name)
	if len(rows) == 0 {
		return f
	}
	if r.Kind != "" {
		if row, ok := characterKindRow(t.Humans, r.Kind, t.Humans.EntryName(rows[0])); ok {
			rows = []int{row}
		}
	}
	d, err := data.NewHumanDef(t.Humans.EntryName(rows[0]), t.Humans.EntryParams(rows[0]))
	if err != nil {
		return f
	}
	dir, face := data.FigureFor(d.TypeID, d.Face, d.Gender)
	f.Choices = append(f.Choices, x.figureFaceChoices(dir)...)
	f.Icons = maps.Clone(x.recordPictures[fmt.Sprintf("face:%s", dir)])
	if f.Icons == nil {
		f.Icons = map[int64]*image.RGBA{}
	}
	f.Icons[0] = f.Icons[int64(face)]
	return f
}

func (s *mapInspectionEdits) charactersFields() []ui.InspectionField {
	x := s.inspector
	if !s.modPageHolds() {
		return nil
	}
	read, err := s.heldRows(s.workspace.Project, s.characterNames, mod.CharactersFile)
	if err != nil {
		return []ui.InspectionField{x.pageLine("error", "", err.Error())}
	}
	rows := read.Characters
	if len(rows.Rows) == 0 && !s.characterAdding {
		return []ui.InspectionField{{Key: "begin", Label: x.word("editor.characters.add"),
			Meaning: x.word("editor.characters.add.hint"), Action: true, Column: 1}}
	}
	out := []ui.InspectionField{s.characterSources("new", x.word("editor.characters.add"), x.word("editor.characters.add.hint")),
		s.characterSources("change", x.word("editor.characters.change"), x.word("editor.characters.change.hint"))}
	if s.characterAdding {
		out = append(out, ui.InspectionField{Key: "cancel", Label: x.word("editor.characters.cancel"),
			Meaning: x.word("editor.characters.cancel.hint"), Action: true, Column: 1})
	}
	if read, err = s.heldRows(s.workspace.Project, nil, textFiles()...); err != nil {
		return []ui.InspectionField{x.pageLine("error", "", err.Error())}
	}
	sets := read.Strings
	for i, r := range rows.Rows {
		prefix := "character." + strconv.Itoa(i) + "."
		name := r.Name
		if name == "" {
			name = fmt.Sprintf(x.word("editor.characters.default"), i+1)
			if r.Target != "" {
				name = r.Target
			}
		}
		if r.Target != "" {
			name = fmt.Sprintf(x.word("editor.characters.changed"), name)
		}
		name = fmt.Sprintf("%s (%d)", name, i+1)
		start := len(out)
		if !r.Off {
			for _, l := range textLanguages() {
				f := workspaceText(prefix+"name."+l, fmt.Sprintf(x.word("editor.characters.name"), languageLabel(l)), sets[l][r.NameKey])
				f.Meaning = x.word("editor.characters.name.hint")
				if r.Target != "" && r.NameKey == "" {
					f.Meaning = x.word("editor.characters.name.inherit.hint")
				}
				out = append(out, f)
			}
			out = append(out, s.characterFace(prefix+"face", r))
		}
		out = append(out, ui.InspectionField{Key: prefix + "remove", Label: x.word("editor.characters.remove"),
			Meaning: x.word("editor.characters.remove.hint"), Action: true, Column: 1})
		for j := start; j < len(out); j++ {
			out[j].Group = name
		}
	}
	return out
}

func (s *mapInspectionEdits) setCharactersField(key, value string) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	x := s.inspector
	if err := x.modWritable(); err != nil {
		return nil, err
	}
	if key == "begin" || key == "cancel" {
		s.characterAdding = key == "begin"
		return nil, nil
	}
	gone := errors.New(x.word("editor.characters.err.gone"))
	doc, err := s.editKind(func(k kindFiles) error {
		return k.editRows(s.characterNames, characterFiles(), func(rows *mod.Data) error {
			return s.editCharacters(k, &rows.Characters, rows.Strings, key, value, gone)
		})
	})
	if err == nil && (key == "new" || key == "change") && value != "0" {
		s.characterAdding = false
	}
	return doc, err
}

// characterFiles are the characters file and the strings files of its names.
func characterFiles() []string { return append([]string{mod.CharactersFile}, textFiles()...) }

// characterNames is the characters page's rowText.
func (s *mapInspectionEdits) characterNames(d *editorModDocument) (func(key string) (string, bool), error) {
	return d.textLookup(s.inspector.pageLanguage())
}

// editCharacters makes the page's edit of key to value on d and sets.
func (s *mapInspectionEdits) editCharacters(k kindFiles, d *mod.CharacterData, sets map[string]mod.Strings, key, value string, gone error) error {
	x := s.inspector
	if key == "new" || key == "change" {
		if value == "0" {
			return nil
		}
		n, err := strconv.ParseInt(value, 10, 64)
		choices := s.characterSources(key, "", "")
		if err != nil || !slices.ContainsFunc(choices.Choices[1:], func(c ui.InspectionChoice) bool { return c.Value == n }) {
			return gone
		}
		id, _ := x.identityOf(n)
		base := x.contextBase().table.Humans
		r := mod.CharacterRow{}
		if key == "new" {
			r.Key = mod.RecordKey("character", func(key string) bool {
				return slices.ContainsFunc(d.Rows, func(c mod.CharacterRow) bool { return c.Key == key }) ||
					slices.ContainsFunc(s.modCharacters(), func(c mod.Character) bool { return c.Key == key })
			})
			r.Row = base.EntryName(id.Row)
		} else {
			r.Target = base.EntryName(id.Row)
			if slices.ContainsFunc(d.Rows, func(c mod.CharacterRow) bool { return c.Target == r.Target }) {
				return errors.New(x.word("editor.characters.err.changed"))
			}
			def, err := data.NewHumanDef(r.Target, base.EntryParams(id.Row))
			if err != nil {
				return err
			}
			r.Face = max(1, min(mod.MaxFace, def.Face))
		}
		if key == "new" {
			values := map[string]string{}
			for _, l := range textLanguages() {
				values[l] = fmt.Sprintf(words.For(l).Text("editor.characters.default"), len(d.Rows)+1)
			}
			r.NameKey = newModText(sets, "character.name", values)
		}
		d.Rows = append(d.Rows, r)
	} else {
		parts := strings.SplitN(key, ".", 3)
		if len(parts) != 3 || parts[0] != "character" {
			return gone
		}
		i, err := strconv.Atoi(parts[1])
		if err != nil || i < 0 || i >= len(d.Rows) {
			return gone
		}
		r := &d.Rows[i]
		switch {
		case parts[2] == "remove":
			for _, m := range k.w.Project.Missions {
				if r.Key != "" && slices.ContainsFunc(m.Party, func(g mod.GuestMember) bool { return g.Character == r.Key }) {
					name := m.Title
					if name == "" {
						name = m.Key
					}
					return fmt.Errorf(x.word("editor.characters.err.used"), name, m.Number)
				}
			}
			d.Rows = slices.Delete(d.Rows, i, i+1)
		case parts[2] == "face" && !r.Off:
			n, err := strconv.ParseInt(value, 10, 32)
			f := s.characterFace(key, *r)
			if err != nil || !slices.ContainsFunc(f.Choices, func(c ui.InspectionChoice) bool { return c.Value == n }) {
				return gone
			}
			r.Face = int32(n)
			if r.Target != "" && r.Face == 0 && r.NameKey == "" && r.Kind == "" && len(r.Strip) == 0 {
				d.Rows = slices.Delete(d.Rows, i, i+1)
			}
		case strings.HasPrefix(parts[2], "name.") && !r.Off:
			l := strings.TrimPrefix(parts[2], "name.")
			if !slices.Contains(textLanguages(), l) {
				return gone
			}
			if n := utf8.RuneCountInString(value); n == 0 || n > 40 || strings.TrimSpace(value) == "" {
				return errors.New(x.word("editor.characters.err.name"))
			}
			values := map[string]string{}
			for _, lang := range textLanguages() {
				values[lang] = sets[lang][r.NameKey]
				if values[lang] == "" && lang != l && r.Target != "" && (lang == s.installTextLanguage() || lang == locale.Fallback) {
					values[lang] = s.gameCharacterName(r.Target)
				}
			}
			shared := slices.ContainsFunc(d.Rows[:i], func(c mod.CharacterRow) bool { return c.NameKey == r.NameKey }) ||
				slices.ContainsFunc(d.Rows[i+1:], func(c mod.CharacterRow) bool { return c.NameKey == r.NameKey })
			shared = shared || len(s.textPlaces(k.w.Project, mod.CharactersFile)[r.NameKey]) > 0
			if r.NameKey == "" || shared {
				r.NameKey = newModText(sets, "character.name", values)
				for _, lang := range textLanguages() {
					if values[lang] == "" {
						delete(sets[lang], r.NameKey)
					}
				}
			}
			if sets[l] == nil {
				sets[l] = mod.Strings{}
			}
			sets[l][r.NameKey] = value
		default:
			return gone
		}
	}
	return nil
}

// installTextLanguage is the text language of the install the editor reads.
func (s *mapInspectionEdits) installTextLanguage() string {
	if x := s.inspector; x.archives != nil {
		if l, ok := locale.ByEntry(x.archives.Base.Profile.Language); ok {
			return l.Code
		}
	}
	return locale.Fallback
}

// gameCharacterName is the name the install gives the game character whose
// definition row is target, in UTF-8: its npcnames line, else the row's name.
func (s *mapInspectionEdits) gameCharacterName(target string) string {
	x := s.inspector
	base := x.contextBase().table.Humans
	s.mission()
	for _, id := range x.identities {
		if id.Kind == mapload.UnitCreature || id.Row <= 0 || id.Row >= base.Len() || base.EntryName(id.Row) != target {
			continue
		}
		if e, ok := x.table.NPCCatalog.Entry(id.NPC); ok {
			return x.npcName(e, false)
		}
	}
	return target
}
