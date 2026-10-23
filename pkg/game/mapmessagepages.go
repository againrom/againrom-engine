package game

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/render/textfit"
	"againrom/pkg/ui"
)

var messageSpeaker = regexp.MustCompile(`(?i)npc=([0-9]+)`)
var messageMarkup = regexp.MustCompile(`<[^>]*>`)

func messageFilename(n uint32) string { return fmt.Sprintf("event%02d.txt", n) }

func freeMessage(pages map[string][]byte) uint32 {
	for n := uint32(1); ; n++ {
		if n == uint32(ReservedMessageNumber) {
			continue
		}
		if _, used := pages[messageFilename(n)]; !used {
			return n
		}
	}
}

func (x *MapInspector) messageTargets(targets map[uint32][]ui.InspectionScriptTarget, pages map[string][]byte) {
	var ids []uint32
	for name := range pages {
		if !strings.HasPrefix(name, "event") || !strings.HasSuffix(name, ".txt") {
			continue
		}
		n, err := strconv.ParseUint(strings.TrimSuffix(strings.TrimPrefix(name, "event"), ".txt"), 10, 32)
		if err == nil && n > 0 && name == messageFilename(uint32(n)) {
			ids = append(ids, uint32(n))
		}
	}
	slices.Sort(ids)
	rows := make([]ui.InspectionScriptTarget, 0, len(ids)+1)
	for _, n := range ids {
		body, _ := splitMessage(x.messageUTF8(pages[messageFilename(n)]))
		lines := messageLines(body)
		line := ""
		if len(lines) > 0 {
			line = lines[0]
		}
		if len(lines) > 1 {
			line += textfit.Mark
		}
		label := x.inspectionUTF8(fmt.Sprintf("%d: %s", n, line))
		for i := range lines {
			lines[i] = x.inspectionUTF8(lines[i])
		}
		rows = append(rows, ui.InspectionScriptTarget{Value: n, Record: -1, Name: label, Choice: label, Text: strings.Join(lines, "\n")})
	}
	rows = append(rows, ui.InspectionScriptTarget{Record: -1, Name: x.inspectionUTF8(x.word("editor.message.new")), NewMessage: true})
	targets[ui.ScriptMessageParameter] = rows
}

// messageLines are a message body's lines as the player reads them: markup
// removed, each line trimmed, the empty lines before the first and after the
// last dropped.
func messageLines(body string) []string {
	var lines []string
	for _, candidate := range strings.Split(messageMarkup.ReplaceAllString(body, ""), "\n") {
		line := strings.TrimSpace(candidate)
		if line == "" && len(lines) == 0 {
			continue
		}
		lines = append(lines, line)
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func (x *MapInspector) messageUTF8(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	selector := 0
	if x.font != nil {
		selector = x.font.Selector
	}
	if decoded, ok := DecodeInstallTextFile(b, selector); ok {
		return string(decoded)
	}
	return string(b)
}

func splitMessage(raw string) (string, int64) {
	speaker := int64(-1)
	if header, body, ok := messageHeader(raw); ok {
		if id, named := (EventAudience{}).tagRules().speaker(header); named {
			speaker = int64(id)
		}
		return body, speaker
	}
	return raw, speaker
}

func messageHeader(raw string) (string, string, bool) {
	if open, close, end, ok := eventPartRange([]byte(raw), 1, EventAudience{}); ok {
		body := strings.TrimPrefix(raw[close+1:end], "\r\n")
		return raw[open+1 : close], strings.TrimPrefix(body, "\n"), true
	}
	return "", "", false
}

func (s *mapInspectionEdits) MessageFields(page uint32) []ui.InspectionField {
	if s.prepareWorkspace() != nil {
		return nil
	}
	raw := []byte(newMessagePage)
	if page != ui.NewMessagePage {
		var exists bool
		if raw, exists = s.workspace.Pages[messageFilename(page)]; !exists {
			return nil
		}
	}
	body, speaker := splitMessage(s.inspector.messageUTF8(raw))
	x := s.inspector
	choices := []ui.InspectionChoice{{Value: -1, Label: x.inspectionUTF8(x.word("editor.message.nospeaker"))}}
	if x.table != nil {
		for _, e := range x.table.NPCCatalog.Records() {
			id := int64(e.ID)
			if id > 255 {
				continue
			}
			name := e.Name
			if name == "" && int(id) < len(x.table.NPCNames) {
				name = x.table.NPCNames[id]
			}
			if name == "" {
				name = e.Section
			}
			choices = append(choices, ui.InspectionChoice{Value: id, Label: fmt.Sprintf("%d: %s", id, x.inspectionUTF8(x.messageUTF8([]byte(name))))})
		}
	}
	slices.SortFunc(choices, func(a, b ui.InspectionChoice) int { return int(a.Value - b.Value) })
	found := false
	for _, c := range choices {
		found = found || c.Value == speaker
	}
	if !found {
		choices = append(choices, ui.InspectionChoice{Value: speaker, Label: strconv.FormatInt(speaker, 10)})
	}
	return []ui.InspectionField{workspaceText("body", x.inspectionUTF8(x.word("editor.pages.body")), body), {Key: "speaker", Label: x.inspectionUTF8(x.word("editor.message.speaker")), Value: speaker, Choices: choices}}
}

func (s *mapInspectionEdits) SetMessage(page uint32, body string, speaker int64) (*ui.InspectionDocument, error) {
	fields := s.MessageFields(page)
	if len(fields) == 0 {
		return nil, errors.New(s.inspector.word("editor.message.unavailable"))
	}
	if fields[0].TextValue == body && fields[1].Value == speaker {
		return nil, nil
	}
	valid := false
	for _, c := range fields[1].Choices {
		valid = valid || c.Value == speaker
	}
	if !valid {
		return nil, errors.New(s.inspector.word("editor.message.badspeaker"))
	}
	header := "part=1"
	raw := s.inspector.messageUTF8(s.workspace.Pages[messageFilename(page)])
	prefix, separator, suffix := "", "\n", ""
	if open, close, end, ok := eventPartRange([]byte(raw), 1, EventAudience{}); ok {
		header = raw[open+1 : close]
		prefix, suffix = raw[:open], raw[end:]
		originalBody := raw[close+1 : end]
		separator = originalBody[:len(originalBody)-len(fields[0].TextValue)]
	}
	if speaker != fields[1].Value {
		if match := messageSpeaker.FindStringIndex(header); match != nil {
			replacement := ""
			if speaker >= 0 {
				replacement = fmt.Sprintf("%s%d", speakerTag, speaker)
			}
			header = header[:match[0]] + replacement + header[match[1]:]
		} else if speaker >= 0 {
			header += fmt.Sprintf(",%s%d", speakerTag, speaker)
		}
	}
	previous := s.page
	s.page = messageFilename(page)
	defer func() { s.page = previous }()
	return s.SetWorkspaceField("pages", "body", prefix+"<"+header+">"+separator+body+suffix)
}

// newMessagePage is the text of a new message page: part 1, empty.
const newMessagePage = "<part=1>\n"

// CreateMessage makes a new message page holding body and speaker and names
// it in field key of the record, or in a new action for kind "Message": the
// page NewMessage makes with the text SetMessage writes, as one edit. A
// refused text or speaker changes nothing.
func (s *mapInspectionEdits) CreateMessage(kind string, index int, key, body string, speaker int64) (*ui.InspectionDocument, uint32, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, 0, err
	}
	n := freeMessage(s.workspace.Pages)
	saved := *s
	saved.undoWorkspace = slices.Clone(s.undoWorkspace)
	d, err := s.NewMessage(kind, index, key)
	if err != nil {
		return nil, 0, err
	}
	steps := len(s.undoWorkspace)
	text, err := s.SetMessage(n, body, speaker)
	if err != nil {
		*s = saved
		return nil, 0, err
	}
	if text != nil {
		d = text
	}
	s.undoWorkspace = s.undoWorkspace[:steps]
	return d, n, nil
}

func (s *mapInspectionEdits) NewMessage(kind string, index int, key string) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	if len(s.workspace.Pages) >= mod.MaxMissionTexts {
		return nil, fmt.Errorf(s.inspector.word("editor.message.full"), mod.MaxMissionTexts)
	}
	n := freeMessage(s.workspace.Pages)
	next := s.model.Clone()
	mark := next.Mark()
	if kind == "Message" {
		if _, err := addScriptRecord(next, mapedit.KindAction); err != nil {
			return nil, err
		}
		kind, index, key = "Action", next.Count(mapedit.KindAction)-1, "value0"
	}
	k, err := editKind(kind)
	if err != nil {
		return nil, err
	}
	view, err := next.Map()
	if err != nil {
		return nil, err
	}
	sc, err := view.Script()
	if err != nil || k != mapedit.KindAction || index < 0 || index >= len(sc.Actions) || key != "value0" || scriptParameterKind(false, sc.Actions[index].Opcode, sc.Actions[index].Type, 0) != ui.ScriptMessageParameter {
		return nil, errors.New(s.inspector.word("editor.message.unavailable"))
	}
	if _, err := s.setFieldOn(next, kind, k, index, key, int64(n)); err != nil {
		return nil, err
	}
	next.Squash(mark)
	w := cloneWorkspace(s.workspace)
	w.Pages[messageFilename(n)] = []byte(newMessagePage)
	d, err := s.project(next)
	if err != nil {
		return nil, err
	}
	s.rememberWorkspace(!bytes.Equal(s.model.Bytes(), next.Bytes()))
	s.model, s.workspace, s.workspaceImage = next, w, workspaceBytes(w)
	s.inspector.messageTargets(d.Script.Targets, w.Pages)
	return d, nil
}
