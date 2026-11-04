package game

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"math/bits"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/data"
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
	return installText(nil, editorPageSource.holding(b), x.pageCode()).text()
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

// heroFaceNotes are the hover lines of the portraits of the speakers whose
// person depends on the player's hero, by speaker.
func (x *MapInspector) heroFaceNotes() map[int64]string {
	notes := make(map[int64]string, len(speakerRoles))
	for _, r := range speakerRoles {
		notes[r.ID] = x.inspectionUTF8(x.word("editor.message.herofacetip"))
	}
	return notes
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
	icons, notes := s.npcPortraits(choices), x.heroFaceNotes()
	if len(source) > 1 {
		s.heroFaces(ui.HeroAudience(source[1]), &icons, notes)
	}
	return []ui.InspectionField{workspaceText("body", x.inspectionUTF8(x.word("editor.pages.body")), body), {Key: "speaker", Label: x.inspectionUTF8(x.word("editor.message.speaker")), Value: speaker, Choices: choices, Icons: icons, IconTip: true, IconNotes: notes}}
}

// heroFaces gives the speakers whose person depends on the player's hero the
// face the dialogue shows the heroes of a branch cell: the face the speaker
// resolution gives a starting hero of each of them, when it gives one face
// for all. Otherwise such a speaker keeps the list's face and says the face
// for these heroes is not known. A cell of every hero changes nothing.
func (s *mapInspectionEdits) heroFaces(heroes ui.HeroAudience, icons *map[int64]*image.RGBA, notes map[int64]string) {
	if heroes == 0 || heroes == ui.EveryHero {
		return
	}
	x := s.inspector
	for _, role := range speakerRoles {
		face, known := s.heroFace(role.ID, heroes)
		if !known {
			notes[role.ID] = x.inspectionUTF8(x.word("editor.message.herofaceunknown"))
			continue
		}
		if *icons == nil {
			*icons = map[int64]*image.RGBA{}
		}
		(*icons)[role.ID], notes[role.ID] = face, x.inspectionUTF8(x.word("editor.message.herofacecell"))
	}
}

// heroFace is the one face speaker id shows every hero of heroes, through
// the speaker resolution over the mission start with a planted starting hero
// of each. There is none without an install or a mission world, for the
// player's own hero (the face the player chooses), or when two of the heroes
// see different faces.
func (s *mapInspectionEdits) heroFace(id int64, heroes ui.HeroAudience) (*image.RGBA, bool) {
	x := s.inspector
	if x == nil || x.table == nil || x.archives == nil {
		return nil, false
	}
	sp := s.missionStart()
	rec, ok := sp.records[int32(id)]
	if sp.world == nil || !ok || rec.Tokens.Has(data.NPCTokenMe) {
		return nil, false
	}
	var face *image.RGBA
	for h := range 4 {
		if heroes&(1<<h) == 0 {
			continue
		}
		key := heroFaceKey{id, h}
		pic, tried := sp.heroFaces[key]
		if !tried {
			r := s.speakerResolver(sp)
			r.cast = newSpeakerCast(plantedHero(h < 2, h%2 == 0), &speakerWorld{actors: sp.cast.actors, alive: sp.cast.alive, worn: sp.cast.worn})
			if f, window, ok := r.SpeakerFace(int(id)); ok {
				pic = ui.SpeakerPortrait(f, window)
			}
			sp.heroFaces[key] = pic
		}
		if pic == nil || face != nil && (face.Rect != pic.Rect || !bytes.Equal(face.Pix, pic.Pix)) {
			return nil, false
		}
		face = pic
	}
	return face, face != nil
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
		edited := false
		for i := range parts {
			if parts[i].Source >= 0 && d.Tags[parts[i].Source].Open == open {
				parts[i].Body, parts[i].Speaker, edited = body, speaker, true
			}
		}
		if edited {
			return s.SetMessageParts(page, parts)
		}
	}
	return s.SetMessageParts(page, []ui.InspectionMessagePart{{Source: -1, Number: 1, Body: body, Speaker: speaker}})
}

// heroAudiences are the audiences of the four starting heroes in the cells'
// order (ui.HeroAudience's bits), each as a mission on this map reads its
// dialogue for that hero: the hero's sex and class, the speaker records of
// the mission start, and the tag rules of the editor's game.
func (s *mapInspectionEdits) heroAudiences() [4]EventAudience {
	records := s.missionStart().records
	var out [4]EventAudience
	for h := range out {
		out[h] = partyAudience(plantedHero(h < 2, h%2 == 0), records)
		out[h].tags = s.inspector.eventTags()
	}
	return out
}

// messageCells are page d's parts as the dialog editor's cells, by part
// number in the order the numbers first appear. A number has one cell per
// hero audience the reader tells apart (ui.HeroCells): each the tag the
// reader gives those heroes, or Ends when it gives none. The number's tags
// no hero reaches follow, each Never.
func messageCells(d messageMarkupDocument, heroes [4]EventAudience) []ui.InspectionMessagePart {
	var order []int
	groups := map[int][]int{}
	for _, i := range d.pages() {
		n := d.Tags[i].Number
		if _, ok := groups[n]; !ok {
			order = append(order, n)
		}
		groups[n] = append(groups[n], i)
	}
	at := map[int]int{}
	for i, tag := range d.Tags {
		at[tag.Open] = i
	}
	rules := heroes[0].tagRules()
	cell := func(source, n int) ui.InspectionMessagePart {
		tag := d.Tags[source]
		_, body, _ := tag.text()
		speaker := int64(-1)
		if id, named := rules.speaker(tag.Tag); named {
			speaker = int64(id)
		}
		return ui.InspectionMessagePart{Source: source, Number: n, Body: body, Tag: tag.Tag, Speaker: speaker, Other: tagOthers(tag.Tag)}
	}
	var out []ui.InspectionMessagePart
	for _, n := range order {
		var got [4]int
		for h, aud := range heroes {
			got[h] = -1
			if open, _, _, ok := eventPartRange([]byte(d.Raw), n, aud); ok {
				got[h] = at[open]
			}
		}
		reached := map[int]bool{}
		for _, h := range ui.HeroCells(got[0] != got[2] || got[1] != got[3], got[0] != got[1] || got[2] != got[3]) {
			source := got[bits.TrailingZeros8(uint8(h))]
			part := ui.InspectionMessagePart{Source: -1, Number: n, Speaker: -1, Ends: true}
			if source >= 0 {
				part, reached[source] = cell(source, n), true
			}
			part.Heroes = h
			out = append(out, part)
		}
		for _, i := range groups[n] {
			if !reached[i] {
				part := cell(i, n)
				part.Never = true
				out = append(out, part)
			}
		}
	}
	return out
}

// tagOthers are a tag's conditions besides its part number, speaker and hero
// literals, as the dialog editor lists them.
func tagOthers(tag string) string {
	rest := setHeroArms(setMessageAttribute(setMessageAttribute(tag, partMark, -1), speakerTag, -1))
	return strings.Join(strings.FieldsFunc(rest, func(r rune) bool { return r == ',' || r == ' ' }), ", ")
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
	out := messageCells(d, s.heroAudiences())
	if len(out) == 0 {
		out = append(out, ui.InspectionMessagePart{Source: -1, Number: 1, Body: d.Raw, Speaker: -1})
	}
	return out
}

// SetMessageParts writes parts to the page through the dialogue writer. When
// they are the dialog editor's cells, every write is read again for the four
// heroes, and one under which a hero would read other than its cell is
// refused with nothing written: a speaker can open or close the speaker gate
// and so move its tag to another cell's heroes.
func (s *mapInspectionEdits) SetMessageParts(page uint32, parts []ui.InspectionMessagePart) (*ui.InspectionDocument, error) {
	if s.prepareWorkspace() != nil || len(parts) == 0 {
		return nil, errors.New(s.inspector.word("editor.message.unavailable"))
	}
	raw, exists := s.workspace.Pages[messageFilename(page)]
	if !exists {
		return nil, errors.New(s.inspector.word("editor.message.unavailable"))
	}
	d := decodeMessageMarkup(s.inspector.messageUTF8(raw))
	for _, part := range parts {
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
	}
	cells := slices.ContainsFunc(parts, func(p ui.InspectionMessagePart) bool { return p.Heroes != 0 || p.Never || p.Ends })
	edits, renumber := tagEdits(d, parts)
	heroes := s.heroAudiences()
	if cells {
		edits, renumber = cellEdits(d, messageCells(d, heroes), parts)
	}
	value, err := d.write(edits, renumber)
	if err != nil {
		return nil, err
	}
	if value == d.Raw {
		return nil, nil
	}
	if cells && !slices.Equal(cellReadings(parts), cellReadings(messageCells(decodeMessageMarkup(value), heroes))) {
		return nil, errors.New(s.inspector.word("editor.message.unreached"))
	}
	previous := s.page
	s.page = messageFilename(page)
	defer func() { s.page = previous }()
	return s.SetWorkspaceField("pages", "body", value)
}

// tagEdits are the writer's edits for parts that are each one tag as
// written, in order; the page is renumbered unless they are its tags in
// order.
func tagEdits(d messageMarkupDocument, parts []ui.InspectionMessagePart) ([]messagePageEdit, bool) {
	indices := d.pages()
	renumber := len(indices) != len(parts)
	edits := make([]messagePageEdit, len(parts))
	for i, part := range parts {
		if i < len(indices) && indices[i] != part.Source {
			renumber = true
		}
		if part.Source >= 0 && part.Source < len(d.Tags) {
			part.Number = d.Tags[part.Source].Number
		}
		edits[i] = messagePageEdit{Source: part.Source, Number: part.Number, Body: part.Body, Speaker: part.Speaker}
	}
	return edits, renumber
}

// cellEdits are the writer's edits for the dialog editor's cells, read the
// page's cells as messageCells reads them. Cells standing as read write
// each tag in place, so an untouched page writes its own bytes. Otherwise a
// part number whose cells stand as read keeps its tags, and any other is
// written cell by cell: each cell one tag naming its heroes by the hero
// literals (heroLiterals), its never-shown tags after; the page is then
// renumbered, as any change of its parts renumbers it.
func cellEdits(d messageMarkupDocument, read, cells []ui.InspectionMessagePart) ([]messagePageEdit, bool) {
	if sameCells(read, cells) {
		texts := map[int]ui.InspectionMessagePart{}
		for _, c := range cells {
			if _, ok := texts[c.Source]; !ok && c.Source >= 0 {
				texts[c.Source] = c
			}
		}
		var edits []messagePageEdit
		for _, i := range d.pages() {
			edits = append(edits, messagePageEdit{Source: i, Number: d.Tags[i].Number, Body: texts[i].Body, Speaker: texts[i].Speaker})
		}
		return edits, false
	}
	byNumber := map[int][]ui.InspectionMessagePart{}
	for _, c := range read {
		byNumber[c.Number] = append(byNumber[c.Number], c)
	}
	var edits []messagePageEdit
	used := map[int]bool{}
	// A tag a second cell takes is copied: a new tag after the same line
	// break, its text the cell's.
	emit := func(c ui.InspectionMessagePart, tag string) {
		e := messagePageEdit{Source: c.Source, Number: c.Number, Body: c.Body, Speaker: c.Speaker, Tag: tag}
		if c.Source >= 0 && used[c.Source] {
			e.Source = -1
			e.Separator, _, _ = d.Tags[c.Source].text()
			if e.Tag == "" {
				e.Tag = d.Tags[c.Source].Tag
			}
		}
		if c.Source >= 0 {
			used[c.Source] = true
		}
		edits = append(edits, e)
	}
	for _, group := range cellGroups(cells) {
		if sameCells(byNumber[group[0].Number], group) {
			sources := map[int]bool{}
			var kept []ui.InspectionMessagePart
			for _, c := range group {
				if !sources[c.Source] {
					sources[c.Source] = true
					kept = append(kept, c)
				}
			}
			slices.SortStableFunc(kept, func(a, b ui.InspectionMessagePart) int { return a.Source - b.Source })
			for _, c := range kept {
				if c.Source >= 0 {
					emit(c, "")
				}
			}
			continue
		}
		var heroes []ui.HeroAudience
		for _, c := range group {
			if !c.Never {
				heroes = append(heroes, c.Heroes)
			}
		}
		sex, class := ui.HeroSplit(heroes)
		for _, c := range group {
			if c.Never || c.Ends && c.Body == "" {
				continue
			}
			tag := partMark + "1"
			if c.Source >= 0 {
				tag = d.Tags[c.Source].Tag
			}
			emit(c, setHeroArms(tag, heroLiterals(c.Heroes, sex, class)...))
		}
		for _, c := range group {
			if c.Never {
				emit(c, "")
			}
		}
	}
	return edits, true
}

// sameCells reports that cells stand as read: the same tags for the same
// heroes in the same order, no text where a dialogue ends, and one text and
// speaker for each tag however many cells hold it.
func sameCells(read, cells []ui.InspectionMessagePart) bool {
	if len(read) != len(cells) {
		return false
	}
	texts := map[int]ui.InspectionMessagePart{}
	for i, c := range cells {
		r := read[i]
		if c.Source != r.Source || c.Number != r.Number || c.Heroes != r.Heroes || c.Never != r.Never || c.Ends != r.Ends || c.Ends && c.Body != "" {
			return false
		}
		if t, ok := texts[c.Source]; ok && c.Source >= 0 && (t.Body != c.Body || t.Speaker != c.Speaker) {
			return false
		}
		texts[c.Source] = c
	}
	return true
}

// cellGroups are cells cut into runs of one part number.
func cellGroups(cells []ui.InspectionMessagePart) [][]ui.InspectionMessagePart {
	var out [][]ui.InspectionMessagePart
	for i, c := range cells {
		if i == 0 || cells[i-1].Number != c.Number {
			out = append(out, nil)
		}
		out[len(out)-1] = append(out[len(out)-1], c)
	}
	return out
}

// heroLiterals are the hero literals naming the heroes of a cell of a part
// told apart by sex, by class, or both: its sex's literal when by sex, its
// class's when by class. A cell of every hero names none.
func heroLiterals(h ui.HeroAudience, sex, class bool) []string {
	if h == 0 {
		h = ui.EveryHero
	}
	women, men := h&(ui.HeroWomanMage|ui.HeroWomanFighter) != 0, h&(ui.HeroManMage|ui.HeroManFighter) != 0
	mages, fighters := h&(ui.HeroWomanMage|ui.HeroManMage) != 0, h&(ui.HeroWomanFighter|ui.HeroManFighter) != 0
	// eventArms opens with iamfemale, iammale, iammage and iamfighter.
	var lits []string
	if sex && women != men {
		lit := eventArms[1].lit
		if women {
			lit = eventArms[0].lit
		}
		lits = append(lits, lit)
	}
	if class && mages != fighters {
		lit := eventArms[3].lit
		if mages {
			lit = eventArms[2].lit
		}
		lits = append(lits, lit)
	}
	return lits
}

// heroReading is what one part gives one hero: its text and speaker, or
// nothing.
type heroReading struct {
	shown   bool
	body    string
	speaker int64
}

// cellReadings are what each part of cells gives each of the four heroes,
// part by part in order; a part that gives no hero anything is left out, as
// it writes no tag.
func cellReadings(cells []ui.InspectionMessagePart) [][4]heroReading {
	var out [][4]heroReading
	for _, group := range cellGroups(cells) {
		var r [4]heroReading
		shown := false
		for h := range r {
			for _, c := range group {
				held := c.Heroes
				if held == 0 {
					held = ui.EveryHero
				}
				if c.Never || held&(1<<h) == 0 {
					continue
				}
				if !c.Ends || c.Body != "" {
					r[h], shown = heroReading{true, strings.Trim(c.Body, "\r\n"), c.Speaker}, true
				}
				break
			}
		}
		if shown {
			out = append(out, r)
		}
	}
	return out
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
	if kind == ui.KindMessage {
		if _, err := addScriptRecord(next, mapedit.KindAction); err != nil {
			return nil, err
		}
		kind, index, key = ui.KindAction, next.Count(mapedit.KindAction)-1, "value0"
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
		if kind != ui.KindTrigger || plan.Index != index || owner.Action < 0 || owner.Action >= len(plan.Actions) {
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
