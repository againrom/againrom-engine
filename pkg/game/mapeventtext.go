package game

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/render/textfit"
	"againrom/pkg/ui"
)

// The event text is the one file of #key sections the second game shows a
// map's events from (R2-ENGINE-049, R2-ENGINE-050). The workspace holds it as
// a mod's UTF-8 text file under its install path; the pages scope lists its
// sections and edits one, and a save writes it into the mod holding the map.

// eventMission is the campaign mission the open map is: an installed
// scenario's number, or the open mod's mission that names the map.
func (s *mapInspectionEdits) eventMission() int {
	n := sourceMission(s.source)
	if n == 0 && s.inspector.mod != nil {
		if e, ok := s.inspector.mod.entryFor(s.source); ok {
			n = e.Number
		}
	}
	return n
}

// openEventText reads the event text into a new workspace: the file of the
// mod holding the map when it carries one for this install's language, else
// the install's own. A file the editor cannot hold leaves eventNote.
func (s *mapInspectionEdits) openEventText() {
	x := s.inspector
	w := &editorWorkspaceState{Pages: map[string][]byte{}}
	s.workspace, s.eventFrom, s.eventNote = w, "", ""
	defer func() {
		s.workspaceImage = workspaceBytes(w)
		s.savedWorkspace = s.workspaceImage
	}()
	a := s.row().eventText(s.eventMission())
	w.EventText = a.path
	s.eventBase, _ = x.installEventText(a)
	text, from, err := x.eventTextFile(a, s.source)
	if err != nil {
		s.eventNote = fmt.Sprintf(x.word("editor.text.unread"), a.path, err)
		return
	}
	w.Pages[a.path], s.eventFrom = text, from
}

// eventTextFile is the event text at a as a UTF-8 file, and the mod folder it
// came from ("" for the install). The install must hold the file: a mod's
// file only replaces one it holds.
func (x *MapInspector) eventTextFile(a textAddress, source string) ([]byte, string, error) {
	installed, err := x.installEventText(a)
	if err != nil {
		return nil, "", err
	}
	if ui.ParseMapSource(source).Kind == ui.SourceHost {
		if folder, ok := modFolderOf(source); ok {
			files, err := mod.ReadInstallFiles(filepath.Base(folder), folder, x.eventTextLanguage())
			if err != nil {
				return nil, "", err
			}
			for _, f := range files {
				if mod.FoldAddress(f.Address) == mod.FoldAddress(a.path) {
					return f.Bytes, folder, nil
				}
			}
		}
	}
	return installed, "", nil
}

// installEventText is the install's own event text at a as a UTF-8 file: what
// the game serves when no mod holds the file.
func (x *MapInspector) installEventText(a textAddress) ([]byte, error) {
	if x.archives == nil || x.archives.Containers == nil {
		return nil, errNoTextArchive
	}
	src := x.archives.Containers
	raw := installText(src, a, TextCode{})
	if !raw.present() {
		return nil, raw.err()
	}
	code := InstallTextCode(src, x.archives.Game().Edition())
	text, ok := code.modTextOf(raw.body, LanguageSelector(src))
	if !ok {
		return nil, errors.New(x.word("editor.text.codec"))
	}
	return text, nil
}

// eventTextLanguage is the language folder of a mod's install files that
// this install reads.
func (x *MapInspector) eventTextLanguage() string {
	root := ""
	if x.archives != nil {
		root = x.archives.Root
	}
	return modrt.LanguageFor(BaseID(InspectInstall(root)))
}

// eventSectionKeys are the text's section keys in order.
func eventSectionKeys(text string) []string {
	return installText(nil, heldSectionSource.holding([]byte(text)), TextCode{}).sectionKeys()
}

// eventSection is the section the game reads for key, its CR LF line ends as
// LF, and whether the text has one.
func eventSection(text, key string) (string, bool) {
	s := installText(nil, heldSectionSource.holding([]byte(text)), TextCode{}).editedSection(key)
	return s, s != ""
}

// eventNumber is the event a section key names: "event" and a number.
func eventNumber(key string) (int, bool) {
	rest, ok := strings.CutPrefix(key, "event")
	n, err := strconv.Atoi(rest)
	return n, ok && err == nil && n >= 0 && "event"+strconv.Itoa(n) == key
}

// sectionKeyName is a key a new section may take: the lower-case letters,
// digits and underscores the game's lower-cased lookup finds.
func sectionKeyName(key string) bool {
	for i, r := range key {
		if !(r >= 'a' && r <= 'z' || i > 0 && (r >= '0' && r <= '9' || r == '_')) {
			return false
		}
	}
	return key != ""
}

func (s *mapInspectionEdits) eventTextFields() []ui.InspectionField {
	x := s.inspector
	if s.eventNote != "" {
		return []ui.InspectionField{x.pageLine("file", "", s.eventNote)}
	}
	text := string(s.workspace.Pages[s.workspace.EventText])
	keys := eventSectionKeys(text)
	if s.page == "" && len(keys) > 0 {
		s.page = keys[0]
	}
	page := workspaceText("page", fmt.Sprintf(x.word("editor.text.section"), s.workspace.EventText), s.page)
	readers := s.sectionReaders()
	for i, key := range keys {
		page.Choices = append(page.Choices, ui.InspectionChoice{Value: int64(i + 1), Label: x.inspectionUTF8(key + ": " + readers(key))})
	}
	body, _ := eventSection(text, s.page)
	return []ui.InspectionField{page, workspaceText("body", x.word("editor.pages.body"), body)}
}

// sectionReaders words who reads a section: the actions whose shown text
// names it, else that no node names it.
func (s *mapInspectionEdits) sectionReaders() func(key string) string {
	x := s.inspector
	byEvent := map[int][]string{}
	if _, sc, err := wizardScript(s.model); err == nil {
		for i, n := range sc.Actions {
			if showsText(false, n.Opcode, n.Type, 0) {
				byEvent[int(n.Value[0])] = append(byEvent[int(n.Value[0])], fmt.Sprintf(x.word("editor.node.action"), i+1))
			}
		}
	}
	return func(key string) string {
		if n, ok := eventNumber(key); ok && len(byEvent[n]) > 0 {
			return strings.Join(byEvent[n], ", ")
		}
		return x.word("editor.text.unnamed")
	}
}

func (s *mapInspectionEdits) setEventTextField(key, value string) (*ui.InspectionDocument, error) {
	if s.eventNote != "" {
		return nil, errors.New(s.eventNote)
	}
	switch key {
	case "page":
		keys := eventSectionKeys(string(s.workspace.Pages[s.workspace.EventText]))
		value = strings.TrimSpace(value)
		if n, err := strconv.Atoi(value); err == nil && n >= 1 && n <= len(keys) {
			value = keys[n-1]
		} else if !slices.Contains(keys, value) && !sectionKeyName(value) {
			return nil, wordErr("editor.err.sectionkey", value)
		}
		s.page = value
		return nil, nil
	case "body":
		return s.changeWorkspace(func(w *editorWorkspaceState) error {
			next, err := s.inspector.withSection(string(w.Pages[w.EventText]), s.page, value)
			if err == nil {
				w.Pages[w.EventText] = []byte(next)
			}
			return err
		})
	}
	return nil, fmt.Errorf("unknown page field %q", key)
}

// withSection is text with key's section holding body: the section the game
// reads for key, else a new one at the end. A body with '#' would end the
// section there, and a key whose lookup finds another key's section would
// write into that one; both are refused.
func (x *MapInspector) withSection(text, key, body string) (string, error) {
	if strings.ContainsAny(body, "#\x00") {
		return "", wordErr("editor.err.sectionmark")
	}
	selector := 0
	if x.archives != nil && x.archives.Containers != nil {
		selector = LanguageSelector(x.archives.Containers)
	}
	if r, dropped := ModTextDropped([]byte(body), selector); dropped {
		return "", wordErr("editor.err.pagechar", r)
	}
	if x.archives != nil && x.archives.Containers != nil {
		code := InstallTextCode(x.archives.Containers, x.archives.Game().Edition())
		if lost := code.ModTextUnserved([]byte(body), selector); len(lost) > 0 {
			names := make([]string, len(lost))
			for i, r := range lost {
				names[i] = fmt.Sprintf("%c (%U)", r, r)
			}
			return "", wordErr("editor.err.pageserve", strings.Join(names, ", "))
		}
	}
	lines := strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n")
	if !strings.HasSuffix(lines, "\r\n") {
		lines += "\r\n"
	}
	next, ok := withTextSection(text, key, lines)
	if !ok {
		return "", wordErr("editor.err.sectionshadow", key)
	}
	if len(next) > mod.MaxInstallFileBytes {
		return "", wordErr("editor.err.pagesize", mod.MaxInstallFileBytes)
	}
	return next, nil
}

// eventTargets are the message targets of an event text: one per event
// section, its text and the page that opens it.
func (x *MapInspector) eventTargets(w *editorWorkspaceState) []ui.InspectionScriptTarget {
	text := string(w.Pages[w.EventText])
	var rows []ui.InspectionScriptTarget
	seen := map[int]bool{}
	for _, key := range eventSectionKeys(text) {
		n, ok := eventNumber(key)
		body, found := eventSection(text, key)
		if !ok || !found || seen[n] {
			continue
		}
		seen[n] = true
		lines := strings.Split(strings.TrimSpace(body), "\n")
		line := strings.TrimSpace(lines[0])
		if len(lines) > 1 {
			line += textfit.Mark
		}
		for i := range lines {
			lines[i] = x.inspectionUTF8(strings.TrimSpace(lines[i]))
		}
		label := x.inspectionUTF8(key + ": " + line)
		rows = append(rows, ui.InspectionScriptTarget{Value: uint32(n), Record: -1, Name: label, Choice: label, Text: strings.Join(lines, "\n"), Page: key})
	}
	return rows
}

// eventTextForSave is the edited event text a save writes into the mod
// holding the map, in this install's language folder; none when the text is
// what the install gives and was not edited since the last save. A text that
// differs from the install's is offered to every target: the target mod writes
// it unless the mod plus install already serve it (writeMapInMod).
func (s *mapInspectionEdits) eventTextForSave() modTexts {
	w := s.workspace
	if w == nil || w.EventText == "" {
		return modTexts{}
	}
	text := w.Pages[w.EventText]
	if sameWorkspace(s.savedWorkspace, s.workspaceImage) && bytes.Equal(text, s.eventBase) {
		return modTexts{}
	}
	rel := path.Join(mod.FilesDir, s.inspector.eventTextLanguage(), w.EventText)
	return modTexts{files: map[string][]byte{rel: bytes.Clone(text)}, base: map[string][]byte{rel: s.eventBase}, from: s.eventFrom}
}
