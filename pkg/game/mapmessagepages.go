package game

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/render/textfit"
	"againrom/pkg/ui"
)

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
	for _, candidate := range strings.Split(messagePlainText(body), "\n") {
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

// speakerRoles are the speakers whose person depends on the hero the player
// chose first, so the list labels each by role (its word) instead of by name.
var speakerRoles = []struct {
	ID   int64
	Word string
}{
	{ID: 21, Word: "editor.message.role.21"},
	{ID: 22, Word: "editor.message.role.22"},
	{ID: 23, Word: "editor.message.role.23"},
	{ID: 24, Word: "editor.message.role.24"},
}

func speakerRank(id int64) int {
	if id < 0 {
		return -1
	}
	for i, r := range speakerRoles {
		if r.ID == id {
			return i
		}
	}
	return len(speakerRoles)
}

func (x *MapInspector) speakerRole(id int64) (string, bool) {
	for _, r := range speakerRoles {
		if r.ID == id {
			return x.word(r.Word), true
		}
	}
	return "", false
}

func (s *mapInspectionEdits) MessageFields(page uint32, source ...int) []ui.InspectionField {
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
	if len(source) == 0 {
		d := decodeMessageMarkup(s.inspector.messageUTF8(raw))
		if open, _, _, ok := eventPartRange([]byte(d.Raw), 1, EventAudience{}); ok {
			for _, tag := range d.Tags {
				if tag.Open == open {
					_, body, _ = tag.text()
					break
				}
			}
		}
	}
	if len(source) > 0 {
		parts := s.MessageParts(page)
		found := false
		for _, part := range parts {
			if part.Source == source[0] {
				body, speaker, found = part.Body, part.Speaker, true
				break
			}
		}
		if !found && source[0] != -1 {
			return nil
		}
		if !found {
			body, speaker = "", -1
		}
	}
	x := s.inspector
	choices := []ui.InspectionChoice{{Value: -1, Label: x.inspectionUTF8(x.word("editor.message.nospeaker"))}}
	choices = append(choices, x.npcChoices(255, true)...)
	slices.SortFunc(choices, func(a, b ui.InspectionChoice) int {
		if ra, rb := speakerRank(a.Value), speakerRank(b.Value); ra != rb {
			return ra - rb
		}
		return int(a.Value - b.Value)
	})
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
	raw := s.inspector.messageUTF8(s.workspace.Pages[messageFilename(page)])
	d := decodeMessageMarkup(raw)
	parts := s.MessageParts(page)
	if open, _, _, ok := eventPartRange([]byte(raw), 1, EventAudience{}); ok {
		for i := range parts {
			if parts[i].Source >= 0 && d.Tags[parts[i].Source].Open == open {
				parts[i].Body, parts[i].Speaker = body, speaker
				return s.SetMessageParts(page, parts)
			}
		}
	}
	return s.SetMessageParts(page, []ui.InspectionMessagePart{{Source: -1, Number: 1, Body: body, Speaker: speaker}})
}

func (s *mapInspectionEdits) MessageParts(page uint32) []ui.InspectionMessagePart {
	if s.prepareWorkspace() != nil {
		return nil
	}
	raw, exists := s.workspace.Pages[messageFilename(page)]
	if page == ui.NewMessagePage {
		return []ui.InspectionMessagePart{{Source: -1, Number: 1, Speaker: -1}}
	}
	if !exists {
		return nil
	}
	d := decodeMessageMarkup(s.inspector.messageUTF8(raw))
	var out []ui.InspectionMessagePart
	for _, i := range d.pages() {
		tag := d.Tags[i]
		speaker := int64(-1)
		if id, named := tagSpeaker(tag.Tag); named {
			speaker = int64(id)
		}
		_, body, _ := tag.text()
		out = append(out, ui.InspectionMessagePart{Source: i, Number: tag.Number, Body: body, Speaker: speaker, Tag: tag.Tag})
	}
	if len(out) == 0 {
		out = append(out, ui.InspectionMessagePart{Source: -1, Number: 1, Body: d.Raw, Speaker: -1})
	}
	return out
}

func (s *mapInspectionEdits) SetMessageParts(page uint32, parts []ui.InspectionMessagePart) (*ui.InspectionDocument, error) {
	if s.prepareWorkspace() != nil || len(parts) == 0 {
		return nil, errors.New(s.inspector.word("editor.message.unavailable"))
	}
	raw, exists := s.workspace.Pages[messageFilename(page)]
	if !exists {
		return nil, errors.New(s.inspector.word("editor.message.unavailable"))
	}
	d := decodeMessageMarkup(s.inspector.messageUTF8(raw))
	indices := d.pages()
	renumber := len(indices) != len(parts)
	edits := make([]messagePageEdit, len(parts))
	for i, part := range parts {
		if i < len(indices) && indices[i] != part.Source {
			renumber = true
		}
		fields := s.MessageFields(page, part.Source)
		valid := false
		if len(fields) == 2 {
			for _, choice := range fields[1].Choices {
				valid = valid || choice.Value == part.Speaker
			}
		}
		if !valid {
			return nil, errors.New(s.inspector.word("editor.message.badspeaker"))
		}
		if part.Source >= 0 && part.Source < len(d.Tags) {
			part.Number = d.Tags[part.Source].Number
		}
		edits[i] = messagePageEdit{part.Source, part.Number, part.Body, part.Speaker}
	}
	value, err := d.write(edits, renumber)
	if err != nil {
		return nil, err
	}
	if value == d.Raw {
		return nil, nil
	}
	previous := s.page
	s.page = messageFilename(page)
	defer func() { s.page = previous }()
	return s.SetWorkspaceField("pages", "body", value)
}

// newMessagePage is the text of a new message page: part 1, empty.
const newMessagePage = "<part=1>\n"

// CreateMessage creates a one-page message through the dialogue writer.
func (s *mapInspectionEdits) CreateMessage(owner ui.InspectionMessageOwner, body string, speaker int64) (*ui.InspectionDocument, uint32, error) {
	return s.CreateMessageParts(owner, []ui.InspectionMessagePart{{Source: -1, Number: 1, Body: body, Speaker: speaker}})
}

func (s *mapInspectionEdits) CreateMessageParts(owner ui.InspectionMessageOwner, parts []ui.InspectionMessagePart) (*ui.InspectionDocument, uint32, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, 0, err
	}
	n := freeMessage(s.workspace.Pages)
	d, err := s.pointMessageParts(owner, n, true, parts)
	return d, n, err
}

// PointMessageParts makes owner name the existing page and writes its
// parts, as one history entry.
func (s *mapInspectionEdits) PointMessageParts(owner ui.InspectionMessageOwner, page uint32, parts []ui.InspectionMessagePart) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	return s.pointMessageParts(owner, page, false, parts)
}

// pointMessageParts makes owner name page n, a new page when fresh, and
// writes its parts; both are one history entry.
func (s *mapInspectionEdits) pointMessageParts(owner ui.InspectionMessageOwner, n uint32, fresh bool, parts []ui.InspectionMessagePart) (*ui.InspectionDocument, error) {
	saved := *s
	saved.undoWorkspace = slices.Clone(s.undoWorkspace)
	d, err := s.pointMessage(owner, n, fresh)
	if err != nil {
		return nil, err
	}
	steps := len(s.undoWorkspace)
	text, err := s.SetMessageParts(n, parts)
	if err != nil {
		*s = saved
		return nil, err
	}
	if text != nil {
		d = text
	}
	s.undoWorkspace = s.undoWorkspace[:steps]
	return d, nil
}

func (s *mapInspectionEdits) NewMessage(owner ui.InspectionMessageOwner) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	return s.pointMessage(owner, freeMessage(s.workspace.Pages), true)
}

// pointMessage makes owner name page n: a new empty page when fresh, else
// a page the map has.
func (s *mapInspectionEdits) pointMessage(owner ui.InspectionMessageOwner, n uint32, fresh bool) (*ui.InspectionDocument, error) {
	if _, exists := s.workspace.Pages[messageFilename(n)]; !fresh && !exists {
		return nil, errors.New(s.inspector.word("editor.message.unavailable"))
	}
	if fresh && len(s.workspace.Pages) >= mod.MaxMissionTexts {
		return nil, fmt.Errorf(s.inspector.word("editor.message.full"), mod.MaxMissionTexts)
	}
	next := s.model.Clone()
	mark := next.Mark()
	kind, index, key := owner.Kind, owner.Index, owner.Key
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
	if err != nil {
		return nil, err
	}
	if owner.Trigger != nil {
		plan := *owner.Trigger
		if kind != "Trigger" || plan.Index != index || owner.Action < 0 || owner.Action >= len(plan.Actions) {
			return nil, errors.New(s.inspector.word("editor.message.unavailable"))
		}
		plan.Actions = slices.Clone(plan.Actions)
		node := plan.Actions[owner.Action]
		if node.Keep {
			i := nodeIndex(sc.Actions, node.ID)
			if i < 0 {
				return nil, errors.New(s.inspector.word("editor.message.unavailable"))
			}
			node.Opcode, node.Types, node.Values = sc.Actions[i].Opcode, sc.Actions[i].Type, sc.Actions[i].Value
		}
		if scriptParameterKind(false, node.Opcode, node.Types, 0) != ui.ScriptMessageParameter {
			return nil, errors.New(s.inspector.word("editor.message.unavailable"))
		}
		node.Keep, node.Values[0] = false, n
		plan.Actions[owner.Action] = node
		if err := writeWizardTrigger(next, plan); err != nil {
			return nil, err
		}
	} else if k != mapedit.KindAction || index < 0 || index >= len(sc.Actions) || key != "value0" || scriptParameterKind(false, sc.Actions[index].Opcode, sc.Actions[index].Type, 0) != ui.ScriptMessageParameter {
		return nil, errors.New(s.inspector.word("editor.message.unavailable"))
	} else if _, err := s.setFieldOn(next, kind, k, index, key, int64(n)); err != nil {
		return nil, err
	}
	next.Squash(mark)
	w := cloneWorkspace(s.workspace)
	if fresh {
		w.Pages[messageFilename(n)] = []byte(newMessagePage)
	}
	d, err := s.project(next)
	if err != nil {
		return nil, err
	}
	s.rememberWorkspace(!bytes.Equal(s.model.Bytes(), next.Bytes()))
	s.model, s.workspace, s.workspaceImage = next, w, workspaceBytes(w)
	s.inspector.messageTargets(d.Script.Targets, w.Pages)
	return d, nil
}
