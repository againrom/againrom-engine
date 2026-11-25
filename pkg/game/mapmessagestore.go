package game

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"maps"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/render/textfit"
	"againrom/pkg/ui"
)

// messageStore is where a format keeps a map's pages and the messages among
// them in the workspace: a page file of its own per page, or the #key
// sections of one install text file the map's mission reads (R2-ENGINE-049,
// R2-ENGINE-050). The pages window, the message targets, the text check and
// the save read it; what differs between the formats is this data.
type messageStore struct {
	// file is the install text file whose #key sections are the pages, by
	// the map's campaign mission, 0 for none; nil: the pages are files.
	file func(mission int) textAddress
	// message is the page name of message n and label its target's name,
	// formats of n; least is the lowest number a message takes.
	message, label string
	least          uint32
	// reserved are the numbers a new message skips: the game shows no
	// message for them.
	reserved []uint32
	// fresh is a new message's text; markup: a message's lines drop the
	// dialogue markup of the message window.
	fresh  string
	markup bool
	// firstPage is the page the window opens on, "": the first it lists.
	firstPage string
	// name is the page a typed name that the workspace lacks may become.
	name func(string) (string, error)
}

// messagePageForm is the first game's message page name.
const messagePageForm = "event%02d.txt"

func messageFilename(n uint32) string { return fmt.Sprintf(messagePageForm, n) }

// missionPageNamed is a typed name as a mission page file.
func missionPageNamed(value string) (string, error) {
	name := strings.ToLower(value)
	if !missionPageName(name) {
		return "", wordErr("editor.err.pagename", unitNamesPage)
	}
	return name, nil
}

// sectionKeyNamed is a typed name as a new section key: the lower-case
// letters, digits and underscores the game's lower-cased lookup finds.
func sectionKeyNamed(key string) (string, error) {
	for i, r := range key {
		if !(r >= 'a' && r <= 'z' || i > 0 && (r >= '0' && r <= '9' || r == '_')) {
			return "", wordErr("editor.err.sectionkey", key)
		}
	}
	if key == "" {
		return "", wordErr("editor.err.sectionkey", key)
	}
	return key, nil
}

// number is the message a page name holds, and whether it holds one.
func (st messageStore) number(name string) (uint32, bool) {
	verb := strings.IndexByte(st.message, '%')
	end := strings.IndexByte(st.message[verb:], 'd') + verb
	digits, prefixed := strings.CutPrefix(name, st.message[:verb])
	digits, suffixed := strings.CutSuffix(digits, st.message[end+1:])
	n, err := strconv.ParseUint(digits, 10, 32)
	return uint32(n), prefixed && suffixed && err == nil && uint32(n) >= st.least && fmt.Sprintf(st.message, n) == name
}

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

// openSectionFile reads the store's text file into a new workspace: the
// file of the mod holding the map when it carries one for this install's
// language, else the install's own. A file the editor cannot hold leaves
// eventNote.
func (s *mapInspectionEdits) openSectionFile() {
	x := s.inspector
	w := &editorWorkspaceState{Pages: map[string][]byte{}}
	s.workspace, s.eventFrom, s.eventNote = w, "", ""
	defer func() {
		s.workspaceImage = workspaceBytes(w)
		s.savedWorkspace = s.workspaceImage
	}()
	a := s.row().messages.file(s.eventMission())
	w.Sections = a.path
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
// the game serves when no mod holds the file. The row is read under the
// empty code, which keeps the stored bytes.
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
	text, ok := code.modTextOf(raw.bytes(), LanguageSelector(src))
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

// heldPageNames are the pages of w in order: the sections of its text file, else
// its page files by name.
func heldPageNames(w *editorWorkspaceState) []string {
	if w.Sections != "" {
		return eventSectionKeys(string(w.Pages[w.Sections]))
	}
	return slices.Sorted(maps.Keys(w.Pages))
}

// pageText is page name of w as the pages window shows it, UTF-8, and
// whether w holds it.
func (x *MapInspector) pageText(w *editorWorkspaceState, name string) (string, bool) {
	if w.Sections != "" {
		return eventSection(string(w.Pages[w.Sections]), name)
	}
	b, ok := w.Pages[name]
	return installText(nil, editorPageSource.holding(b), x.pageCode()).text(), ok
}

// pageFields are the pages scope's fields: the page, chosen from the pages
// the workspace holds or typed, each named with the actions that show it,
// and its text.
func (s *mapInspectionEdits) pageFields() []ui.InspectionField {
	x := s.inspector
	if s.eventNote != "" {
		return []ui.InspectionField{x.pageLine("file", "", s.eventNote)}
	}
	names := heldPageNames(s.workspace)
	if s.page == "" {
		s.page = s.row().messages.firstPage
		if s.page == "" && len(names) > 0 {
			s.page = names[0]
		}
	}
	place := cmp.Or(s.workspace.Sections, "title/briefing/eventNN/tipsNN.txt")
	page := workspaceText("page", fmt.Sprintf(x.word("editor.pages.file"), place), s.page)
	readers := s.pageReaders()
	for i, name := range names {
		page.Choices = append(page.Choices, ui.InspectionChoice{Value: int64(i + 1), Label: x.inspectionUTF8(name + ": " + readers(name))})
	}
	body, _ := x.pageText(s.workspace, s.page)
	return []ui.InspectionField{page, workspaceText("body", x.word("editor.pages.body"), body)}
}

// pageReaders words who reads a page: the actions whose shown text names
// it, else that no node names it.
func (s *mapInspectionEdits) pageReaders() func(name string) string {
	x, st := s.inspector, s.row().messages
	byMessage := map[uint32][]string{}
	if _, sc, err := wizardScript(s.model); err == nil {
		for i, n := range sc.Actions {
			if showsText(false, n.Opcode, n.Type, 0) {
				byMessage[n.Value[0]] = append(byMessage[n.Value[0]], fmt.Sprintf(x.word("editor.node.action"), i+1))
			}
		}
	}
	return func(name string) string {
		if n, ok := st.number(name); ok && len(byMessage[n]) > 0 {
			return strings.Join(byMessage[n], ", ")
		}
		return x.word("editor.text.unnamed")
	}
}

// setPageField chooses the page, by its number in the list, its name or a
// name the store admits, or edits its text.
func (s *mapInspectionEdits) setPageField(key, value string) (*ui.InspectionDocument, error) {
	if s.eventNote != "" {
		return nil, errors.New(s.eventNote)
	}
	switch key {
	case "page":
		names := heldPageNames(s.workspace)
		value = strings.TrimSpace(value)
		if n, err := strconv.Atoi(value); err == nil && n >= 1 && n <= len(names) {
			value = names[n-1]
		} else if !slices.Contains(names, value) {
			name, err := s.row().messages.name(value)
			if err != nil {
				return nil, err
			}
			value = name
		}
		s.page = value
		return nil, nil
	case "body":
		return s.changeWorkspace(func(w *editorWorkspaceState) error {
			return s.withPage(w, s.page, value)
		})
	}
	return nil, fmt.Errorf("unknown page field %q", key)
}

// textSelector is the selector the game shows this install's text under:
// the editor font's, which the install's language sets.
func (x *MapInspector) textSelector() int {
	if x.font != nil {
		return x.font.Selector
	}
	if x.archives != nil && x.archives.Containers != nil {
		return LanguageSelector(x.archives.Containers)
	}
	return 0
}

// checkPageText refuses text the game would not show as written: not UTF-8
// or holding a NUL, a '#' that would end a section there, a letter the font
// cannot show, or a letter the text file's serve and read-back shows as
// another.
func (x *MapInspector) checkPageText(text string, sections bool) error {
	if !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return wordErr("editor.err.pagetext")
	}
	if sections && strings.Contains(text, "#") {
		return wordErr("editor.err.sectionmark")
	}
	selector := x.textSelector()
	if r, dropped := ModTextDropped([]byte(text), selector); dropped {
		return wordErr("editor.err.pagechar", r)
	}
	if x.archives != nil && x.archives.Containers != nil {
		code := InstallTextCode(x.archives.Containers, x.archives.Game().Edition())
		if lost := code.ModTextUnserved([]byte(text), selector); len(lost) > 0 {
			names := make([]string, len(lost))
			for i, r := range lost {
				names[i] = fmt.Sprintf("%c (%U)", r, r)
			}
			return wordErr("editor.err.pageserve", strings.Join(names, ", "))
		}
	}
	return nil
}

// withPage puts text as page name of w after checkPageText. A page file is
// held as typed, within the page and page count bounds. A section is the
// one the game reads for name, else a new one at the end, CR LF ended; a
// name whose lookup finds another key's section is refused, and so is a
// file over the install file bound.
func (s *mapInspectionEdits) withPage(w *editorWorkspaceState, name, text string) error {
	return s.putPage(w, name, text, true)
}

// putPage is withPage, with the text check only when check: a copy of a page
// the workspace holds keeps that page's bytes.
func (s *mapInspectionEdits) putPage(w *editorWorkspaceState, name, text string, check bool) error {
	if check {
		if err := s.inspector.checkPageText(text, w.Sections != ""); err != nil {
			return err
		}
	}
	if w.Sections == "" {
		if len(text) > mod.MaxFileBytes {
			return wordErr("editor.err.pagesize", mod.MaxFileBytes)
		}
		if _, exists := w.Pages[name]; !exists && len(w.Pages) >= mod.MaxMissionTexts {
			return wordErr("editor.err.pagecount", mod.MaxMissionTexts)
		}
		w.Pages[name] = []byte(text)
		return nil
	}
	lines := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r\n")
	if !strings.HasSuffix(lines, "\r\n") {
		lines += "\r\n"
	}
	next, ok := withTextSection(string(w.Pages[w.Sections]), name, lines)
	if !ok {
		return wordErr("editor.err.sectionshadow", name)
	}
	if len(next) > mod.MaxInstallFileBytes {
		return wordErr("editor.err.pagesize", mod.MaxInstallFileBytes)
	}
	w.Pages[w.Sections] = []byte(next)
	return nil
}

// freeMessage is the first number a new message in w may take: not
// reserved, no page of that name held, and in a text file no header the
// game's substring lookup would find first.
func (s *mapInspectionEdits) freeMessage(w *editorWorkspaceState) uint32 {
	st := s.row().messages
	var keys []string
	if w.Sections != "" {
		keys = eventSectionKeys(string(w.Pages[w.Sections]))
	}
	for n := max(st.least, 1); ; n++ {
		if slices.Contains(st.reserved, n) {
			continue
		}
		name := fmt.Sprintf(st.message, n)
		shadowed := slices.ContainsFunc(keys, func(key string) bool { return strings.HasPrefix(key, name) })
		if _, used := w.Pages[name]; !used && !shadowed {
			return n
		}
	}
}

// addMessage puts a new message holding text into workspace w and returns
// its number. A refused text leaves w as it is; the page count bound is
// worded as the dialogs a mission holds.
func (s *mapInspectionEdits) addMessage(w *editorWorkspaceState, text string) (uint32, error) {
	return s.addMessageChecked(w, text, true)
}

// addMessageChecked is addMessage with the text check only when check. A
// mission text the editor could not read takes no message: a file made from
// nothing would replace the installed one.
func (s *mapInspectionEdits) addMessageChecked(w *editorWorkspaceState, text string, check bool) (uint32, error) {
	if s.eventNote != "" {
		return 0, errors.New(s.eventNote)
	}
	n := s.freeMessage(w)
	err := s.putPage(w, fmt.Sprintf(s.row().messages.message, n), text, check)
	if bound := (*liftedLimit)(nil); errors.As(err, &bound) && bound.key == "editor.err.pagecount" {
		err = fmt.Errorf(s.inspector.word("editor.message.full"), mod.MaxMissionTexts)
	}
	return n, err
}

// messageTargets are the message targets of workspace w: one per message
// page, by number, with its first line, its lines and the page that holds
// it, then the choice that makes a new message.
func (x *MapInspector) messageTargets(targets map[uint32][]ui.InspectionScriptTarget, w *editorWorkspaceState, st messageStore) {
	type message struct {
		n          uint32
		name, text string
	}
	var held []message
	if w != nil {
		seen := map[uint32]bool{}
		for _, name := range heldPageNames(w) {
			n, ok := st.number(name)
			text, found := x.pageText(w, name)
			if ok && found && !seen[n] {
				seen[n] = true
				held = append(held, message{n, name, text})
			}
		}
	}
	slices.SortStableFunc(held, func(a, b message) int { return cmp.Compare(a.n, b.n) })
	rows := make([]ui.InspectionScriptTarget, 0, len(held)+1)
	for _, m := range held {
		var lines []string
		if st.markup {
			body, _ := splitMessage(m.text)
			lines = messageLines(body)
		} else {
			for _, line := range strings.Split(strings.TrimSpace(m.text), "\n") {
				lines = append(lines, strings.TrimSpace(line))
			}
		}
		line := ""
		if len(lines) > 0 {
			line = lines[0]
		}
		if len(lines) > 1 {
			line += textfit.Mark
		}
		label := x.inspectionUTF8(fmt.Sprintf(st.label+": %s", m.n, line))
		for i := range lines {
			lines[i] = x.inspectionUTF8(lines[i])
		}
		rows = append(rows, ui.InspectionScriptTarget{Value: m.n, Record: -1, Name: label, Choice: label, Text: strings.Join(lines, "\n"), Page: m.name})
	}
	rows = append(rows, ui.InspectionScriptTarget{Record: -1, Name: x.inspectionUTF8(x.word("editor.message.new")), NewChoice: ui.NewMessageChoice})
	targets[ui.ScriptMessageParameter] = rows
}

// sectionsForSave is the edited text file a save writes into the mod
// holding the map, in this install's language folder; none when the store
// keeps page files, or the text is what the install gives and was not
// edited since the last save. A text that differs from the install's is
// offered to every target: the target mod writes it unless the mod plus
// install already serve it (writeMapInMod).
func (s *mapInspectionEdits) sectionsForSave() modTexts {
	w := s.workspace
	if w == nil || w.Sections == "" {
		return modTexts{}
	}
	text := w.Pages[w.Sections]
	if sameWorkspace(s.savedWorkspace, s.workspaceImage) && bytes.Equal(text, s.eventBase) {
		return modTexts{}
	}
	rel := path.Join(mod.FilesDir, s.inspector.eventTextLanguage(), w.Sections)
	return modTexts{files: map[string][]byte{rel: bytes.Clone(text)}, base: map[string][]byte{rel: s.eventBase}, from: s.eventFrom}
}
