package game

import (
	"bytes"
	"image"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/herokind"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// variantFixture opens the dialog editor, in a 1920x1080 window, on page 7
// holding raw, with speakers 21, 23, 51 and 52 in the speaker list.
func variantFixture(t *testing.T, raw string) (*ui.MapEditor, *mapInspectionEdits) {
	t.Helper()
	e, s := triggerMessageFixture(t, false)
	e.Layout(1920, 1080)
	s.inspector.table = &mapload.Table{NPCCatalog: &data.NPCCatalog{Entries: []data.NPCEntry{
		{ID: 21, Name: "Hero"}, {ID: 23, Name: "Monk"}, {ID: 51, Name: "Elder"}, {ID: 52, Name: "Guard"}}}}
	workspacePage(t, e, "event07.txt", raw)
	e.Select(recordIndex(t, e, "Trigger", 0), false)
	configure(t, e, "a0.item")
	return e, s
}

// variantWord is the editor's English word id.
func variantWord(id string) string { return words.For("en").Text(id) }

// heroLabels are the cell labels of the four heroes, in the cells' order.
var heroLabels = []string{"editor.message.hero.womanmage", "editor.message.hero.womanfighter", "editor.message.hero.manmage", "editor.message.hero.manfighter"}

// heroAudience is a planted hero of the cells' order: 0 woman mage, 1
// woman fighter, 2 man mage, 3 man fighter.
func heroAudience(h int) EventAudience {
	k := herokind.AudienceOrder[h]
	return EventAudience{HeroFemale: k.Female(), HeroMage: k.Mage()}
}

// readerText is part n of page as the event text reader gives it to hero h,
// as the editor's text box holds it, and its speaker; ok is false when no tag
// of part n reaches h.
func readerText(page []byte, n, h int) (text string, speaker int64, ok bool) {
	body, ok := EventPart(page, n, heroAudience(h))
	if !ok {
		return "", -1, false
	}
	plain := strings.TrimPrefix(strings.TrimPrefix(body, "\r\n"), "\n")
	speaker = -1
	if id, named := EventPartSpeaker(page, n, heroAudience(h)); named {
		speaker = int64(id)
	}
	return plain[:messageTextEnd(plain, len(plain))], speaker, true
}

// cardRows are the popup row indices of the card numbered label: its head
// first, then every row up to the next card or the "+" row.
func cardRows(t *testing.T, e *ui.MapEditor, label string) []int {
	t.Helper()
	lines := e.PopupLines()
	var rows []int
	for i, line := range lines {
		head := line == label || strings.HasPrefix(line, label+" ")
		plus := reflect.DeepEqual(e.PopupRowButtons(i), []string{"+"})
		switch {
		case len(rows) == 0 && head:
			rows = append(rows, i)
		case len(rows) > 0 && (isCardHead(line) || plus):
			return rows
		case len(rows) > 0:
			rows = append(rows, i)
		}
	}
	if len(rows) == 0 {
		t.Fatalf("no card %s:\n%s", label, strings.Join(lines, "\n"))
	}
	return rows
}

// isCardHead reports a popup line that heads a card: its number, then its
// speaker or nothing.
func isCardHead(line string) bool {
	number, _, _ := strings.Cut(line, " ")
	if number == "" {
		return false
	}
	for _, c := range number {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// cardCells are the cells card label shows: for each, its audience label
// (the line's word) and the text after it, read off the popup lines.
func cardCells(t *testing.T, e *ui.MapEditor, label string) (labels, texts []string) {
	t.Helper()
	lines := e.PopupLines()
	rows := cardRows(t, e, label)
	for k, r := range rows[1:] {
		label := ""
		for _, id := range append(slices.Clone(heroLabels), "editor.message.hero.woman", "editor.message.hero.man", "editor.message.hero.mage", "editor.message.hero.fighter") {
			word := variantWord(id)
			if (lines[r] == word || strings.HasPrefix(lines[r], word+" ")) && len(word) > len(variantWord(label)) {
				label = id
			}
		}
		if label == "" {
			continue
		}
		labels = append(labels, label)
		text := ""
		for _, next := range rows[k+2:] {
			if strings.HasPrefix(lines[next], variantWord("editor.message.other")) || lines[next] == variantWord("editor.message.ends") {
				continue
			}
			text = lines[next]
			break
		}
		texts = append(texts, text)
	}
	return labels, texts
}

// splitButton presses card c's (part index c's) button captioned by word id.
func splitButton(t *testing.T, e *ui.MapEditor, label string, c int, id string) {
	t.Helper()
	head := cardRows(t, e, label)[0]
	k := slices.Index(e.PopupRowButtons(head), variantWord(id))
	if k < 0 {
		t.Fatalf("card %s offers %q, no %q", label, e.PopupRowButtons(head), variantWord(id))
	}
	r, ok := e.RowButtonRect(2*c+1, k, false)
	trigClick(t, e, r, ok, id)
}

// A pair of iamfemale and iammale tags is one card of two cells, the woman's
// then the man's, each holding what the reader gives that hero; a part of one
// variant shows no hero label.
func TestDialogVariantsBySexAreOneCardOfTwoCells(t *testing.T) {
	raw := "<part=1,iamfemale>\r\nShe goes\r\n<part=1,iammale>\r\nHe goes\r\n<part=2>\r\nAll"
	e, s := variantFixture(t, raw)
	labels, texts := cardCells(t, e, "1")
	if !reflect.DeepEqual(labels, []string{"editor.message.hero.woman", "editor.message.hero.man"}) {
		t.Fatalf("card 1 shows cells %q:\n%s", labels, trigLines(e))
	}
	page := s.workspace.Pages["event07.txt"]
	for k, h := range []int{0, 2} {
		if want, _, _ := readerText(page, 1, h); texts[k] != want {
			t.Errorf("cell %s shows %q, the reader gives %q", labels[k], texts[k], want)
		}
	}
	if heads := strings.Count("\n"+trigLines(e)+"\n", "\n1\n"); heads != 1 {
		t.Errorf("part 1 shows %d cards", heads)
	}
	if labels, _ := cardCells(t, e, "2"); len(labels) != 0 {
		t.Errorf("one-variant card 2 shows hero labels %q", labels)
	}
}

// An iammage tag followed by an untagged one splits by class: the mage gets
// the first, the fighter the second.
func TestDialogVariantsByClassMageThenUntagged(t *testing.T) {
	raw := "<part=1,iammage>\r\nArcane\r\n<part=1>\r\nPlain"
	e, s := variantFixture(t, raw)
	labels, texts := cardCells(t, e, "1")
	if !reflect.DeepEqual(labels, []string{"editor.message.hero.mage", "editor.message.hero.fighter"}) {
		t.Fatalf("card 1 shows cells %q:\n%s", labels, trigLines(e))
	}
	page := s.workspace.Pages["event07.txt"]
	for k, h := range []int{0, 1} {
		if want, _, _ := readerText(page, 1, h); texts[k] != want {
			t.Errorf("cell %s shows %q, the reader gives %q", labels[k], texts[k], want)
		}
	}
}

// Four tags naming sex and class are four cells: woman mage, woman fighter,
// man mage, man fighter.
func TestDialogVariantsAllFourAreFourCells(t *testing.T) {
	raw := "<part=1,iamfemale,iammage>\r\nA\r\n<part=1,iamfemale,iamfighter>\r\nB\r\n<part=1,iammale,iammage>\r\nC\r\n<part=1,iammale,iamfighter>\r\nD"
	e, s := variantFixture(t, raw)
	labels, texts := cardCells(t, e, "1")
	if !reflect.DeepEqual(labels, heroLabels) {
		t.Fatalf("card 1 shows cells %q:\n%s", labels, trigLines(e))
	}
	page := s.workspace.Pages["event07.txt"]
	for h := range 4 {
		if want, _, _ := readerText(page, 1, h); texts[h] != want {
			t.Errorf("cell %s shows %q, the reader gives %q", labels[h], texts[h], want)
		}
	}
}

// A later tag for heroes an earlier tag already takes shows as never shown,
// and its cross drops it on Apply; heroes no tag reaches show a cell that
// says the dialogue ends there.
func TestDialogVariantsNeverShownAndEnding(t *testing.T) {
	raw := "<part=1,iamfemale>\r\nShe\r\n<part=1,iamfemale>\r\nAgain\r\n<part=2>\r\nAll"
	e, s := variantFixture(t, raw)
	labels, texts := cardCells(t, e, "1")
	if !reflect.DeepEqual(labels, []string{"editor.message.hero.woman", "editor.message.hero.man"}) || texts[0] != "She" {
		t.Fatalf("card 1 shows cells %q %q:\n%s", labels, texts, trigLines(e))
	}
	lines := e.PopupLines()
	rows := cardRows(t, e, "1")
	never, ends := -1, false
	for _, r := range rows {
		if strings.HasPrefix(lines[r], strings.Replace(variantWord("editor.message.never"), "%s", "Again", 1)) {
			never = r
		}
		ends = ends || lines[r] == variantWord("editor.message.ends")
	}
	if never < 0 || !ends {
		t.Fatalf("card 1 lacks the never-shown tag (%d) or the man's ending (%v):\n%s", never, ends, trigLines(e))
	}
	if _, _, ok := readerText(s.workspace.Pages["event07.txt"], 1, 3); ok {
		t.Fatal("the planted page reaches the man")
	}
	k := slices.Index(e.PopupRowButtons(never), "x")
	if k < 0 {
		t.Fatalf("the never-shown row offers %q", e.PopupRowButtons(never))
	}
	// The never-shown tag is the third cell of the page: the woman's, the
	// man's ending, then it; its row answers for its speaker field 5.
	cross, ok := e.RowButtonRect(5, k, false)
	trigClick(t, e, cross, ok, "never-shown cross")
	messageButton(t, e, 0)
	if got := string(s.workspace.Pages["event07.txt"]); got != "<part=1,iamfemale>\r\nShe\r\n<part=2>\r\nAll" {
		t.Fatalf("Apply after the cross wrote %q", got)
	}
}

// Literals other than the hero's four stay byte for byte through an edit and
// show on their cell as its other conditions.
func TestDialogVariantsKeepOtherLiterals(t *testing.T) {
	raw := "<part=1,npc=51,iamfemale,female>\r\nShe\r\n<part=1,npc=52,iammale,iamfigter>\r\nHe"
	e, s := variantFixture(t, raw)
	lines := strings.Join(e.PopupLines(), "\n")
	for _, want := range []string{variantWord("editor.message.other") + " female", variantWord("editor.message.other") + " iamfigter"} {
		if !strings.Contains(lines, want) {
			t.Errorf("the popup lacks %q:\n%s", want, lines)
		}
	}
	messageField(t, e, 0)
	_ = e.Key("selectall")
	e.TextInput("She!")
	messageButton(t, e, 0)
	want := strings.Replace(raw, "She", "She!", 1)
	if got := string(s.workspace.Pages["event07.txt"]); got != want {
		t.Fatalf("Apply wrote %q, want %q", got, want)
	}
}

// Split by sex, edit and Apply: the reader gives each hero its cell's text
// and speaker, through tags naming the hero; merging back and Apply gives
// every hero the woman's cell. A split merged back before Apply, and an
// untouched page, write nothing.
func TestDialogVariantsSplitEditApplyMerge(t *testing.T) {
	raw := "<part=1,npc=21>\r\nHello\r\n<part=2>\r\nBye"
	t.Run("untouched", func(t *testing.T) {
		e, s := variantFixture(t, raw)
		steps := len(s.undoWorkspace)
		messageButton(t, e, 0)
		if got := string(s.workspace.Pages["event07.txt"]); got != raw || len(s.undoWorkspace) != steps {
			t.Fatalf("an untouched Apply wrote %q", got)
		}
	})
	t.Run("split and merge back", func(t *testing.T) {
		e, s := variantFixture(t, raw)
		steps := len(s.undoWorkspace)
		splitButton(t, e, "1", 0, "editor.message.bysex")
		if labels, _ := cardCells(t, e, "1"); len(labels) != 2 {
			t.Fatalf("By sex shows cells %q:\n%s", labels, trigLines(e))
		}
		splitButton(t, e, "1", 0, "editor.message.bysex")
		messageButton(t, e, 0)
		if got := string(s.workspace.Pages["event07.txt"]); got != raw || len(s.undoWorkspace) != steps {
			t.Fatalf("a split merged back wrote %q", got)
		}
	})
	t.Run("by sex", func(t *testing.T) {
		e, s := variantFixture(t, raw)
		splitButton(t, e, "1", 0, "editor.message.bysex")
		labels, texts := cardCells(t, e, "1")
		if !reflect.DeepEqual(labels, []string{"editor.message.hero.woman", "editor.message.hero.man"}) || texts[0] != "Hello" || texts[1] != "Hello" {
			t.Fatalf("By sex shows %q %q:\n%s", labels, texts, trigLines(e))
		}
		messageField(t, e, 2) // the man's text
		_ = e.Key("selectall")
		e.TextInput("Hi there")
		messageField(t, e, 3) // the man's speaker
		messageChoiceClick(t, e, "No speaker")
		messageButton(t, e, 0)
		page := s.workspace.Pages["event07.txt"]
		for h, want := range []struct {
			text    string
			speaker int64
		}{{"Hello", 21}, {"Hello", 21}, {"Hi there", -1}, {"Hi there", -1}} {
			if text, speaker, ok := readerText(page, 1, h); !ok || text != want.text || speaker != want.speaker {
				t.Errorf("hero %d reads %q by %d (%v), want %q by %d in %q", h, text, speaker, ok, want.text, want.speaker, page)
			}
		}
		if !bytes.Contains(page, []byte("iamfemale")) || !bytes.Contains(page, []byte("iammale")) {
			t.Errorf("the split names no hero: %q", page)
		}
		if text, _, _ := readerText(page, 2, 0); text != "Bye" {
			t.Errorf("part 2 reads %q", text)
		}
		splitButton(t, e, "1", 0, "editor.message.bysex")
		messageButton(t, e, 0)
		page = s.workspace.Pages["event07.txt"]
		for h := range 4 {
			if text, speaker, ok := readerText(page, 1, h); !ok || text != "Hello" || speaker != 21 {
				t.Errorf("after the merge hero %d reads %q by %d in %q", h, text, speaker, page)
			}
		}
	})
	t.Run("by sex and class", func(t *testing.T) {
		e, s := variantFixture(t, raw)
		splitButton(t, e, "1", 0, "editor.message.bysex")
		splitButton(t, e, "1", 0, "editor.message.byclass")
		labels, _ := cardCells(t, e, "1")
		if !reflect.DeepEqual(labels, heroLabels) {
			t.Fatalf("By sex and By class show %q:\n%s", labels, trigLines(e))
		}
		for c, text := range []string{"A", "B", "C", "D"} {
			messageField(t, e, 2*c)
			_ = e.Key("selectall")
			e.TextInput(text)
		}
		messageButton(t, e, 0)
		page := s.workspace.Pages["event07.txt"]
		for h, want := range []string{"A", "B", "C", "D"} {
			if text, speaker, ok := readerText(page, 1, h); !ok || text != want || speaker != 21 {
				t.Errorf("hero %d reads %q by %d in %q", h, text, speaker, page)
			}
		}
	})
}

// Each branch cell's hero-dependent speaker carries the face the speaker
// resolution gives a planted starting hero of that cell, or says that face
// is not known.
func TestDialogVariantHeroSpeakerFaceByCell(t *testing.T) {
	raw := "<part=1,npc=21,iamfemale>\r\nShe\r\n<part=1,npc=21,iammale>\r\nHe"
	_, s := variantFixture(t, raw)
	parts := s.MessageParts(7)
	if len(parts) != 2 {
		t.Fatalf("page 7 reads as %d cells", len(parts))
	}
	unknown := s.inspector.inspectionUTF8(variantWord("editor.message.herofaceunknown"))
	// The cells' heroes as the dialog editor asks for them: the woman's
	// cell is the woman mage and the woman fighter (bits 0 and 1), the man's
	// the man mage and the man fighter (bits 2 and 3).
	for c, female := range []bool{true, false} {
		fields := s.MessageFields(7, parts[c].Source, map[bool]int{true: 3, false: 12}[female])
		if len(fields) != 2 {
			t.Fatalf("cell %d has no fields", c)
		}
		face, known := plantedFace(s, 21, female)
		got := fields[1].Icons[21]
		switch {
		case known && (got == nil || got.Rect != face.Rect || !bytes.Equal(got.Pix, face.Pix)):
			t.Errorf("cell %d shows a face other than the planted hero's", c)
		case !known && fields[1].IconNotes[21] != unknown:
			t.Errorf("cell %d's speaker 21 says %q, want %q", c, fields[1].IconNotes[21], unknown)
		}
	}
}

// plantedFace is the face the speaker resolution gives speaker id for a
// planted starting hero of that sex, false when it gives none: no install
// to draw from, no mission world, or the speaker is the player's own hero.
func plantedFace(s *mapInspectionEdits, id int, female bool) (*image.RGBA, bool) {
	sp := s.missionStart()
	if s.inspector.table == nil || s.inspector.archives == nil || sp.world == nil || sp.records[int32(id)].Tokens.Has(data.NPCTokenMe) {
		return nil, false
	}
	r := s.speakerResolver(sp)
	hero := []mapload.PartyMember{{PlayerCharacter: true, StartingHero: true, FigureDir: string(data.FigureDirFor(false, female)), FigureFace: 1}}
	r.cast = newSpeakerCast(hero, &speakerWorld{actors: sp.cast.actors, alive: sp.cast.alive, worn: sp.cast.worn})
	face, window, ok := r.SpeakerFace(id)
	if !ok {
		return nil, false
	}
	return ui.SpeakerPortrait(face, window), true
}

// A part of four variants, one of two and one of one, in the editor's
// smallest window: each card is one frame, its cells stack inside it one
// under the other, every row across the card's width.
func TestDialogVariantGroupsStackInOneFrame(t *testing.T) {
	raw := "<part=1,iamfemale,iammage>\r\nA\r\n<part=1,iamfemale,iamfighter>\r\nB\r\n<part=1,iammale,iammage>\r\nC\r\n<part=1,iammale,iamfighter>\r\nD" +
		"\r\n<part=2,iamfemale>\r\nShe\r\n<part=2,iammale>\r\nHe\r\n<part=3>\r\nAll"
	e, _ := variantFixture(t, raw)
	popup, _ := e.PopupRect()
	for _, c := range []struct {
		label       string
		part, cells int
	}{{"1", 0, 4}, {"2", 4, 2}, {"3", 6, 0}} {
		card, _, ok := e.CardRects(c.part)
		if !ok {
			t.Fatalf("card %s is not shown:\n%s", c.label, trigLines(e))
		}
		if labels, _ := cardCells(t, e, c.label); len(labels) != c.cells {
			t.Fatalf("card %s shows cells %q:\n%s", c.label, labels, trigLines(e))
		}
		rows := cardRows(t, e, c.label)
		head, _ := e.PopupRowRect(rows[0])
		below := head.Max.Y
		for _, r := range rows[1:] {
			rr, ok := e.PopupRowRect(r)
			switch {
			case !ok:
				t.Fatalf("card %s's row %d is not shown", c.label, r)
			case !rr.In(card):
				t.Errorf("card %s's row %d at %v stands outside its frame %v", c.label, r, rr, card)
			case rr.Min.Y < below:
				t.Errorf("card %s's row %d at %v stands beside the row above, not under it", c.label, r, rr)
			case rr.Min.X != head.Min.X || rr.Max.X != head.Max.X:
				t.Errorf("card %s's row %d at %v is not as wide as its head %v", c.label, r, rr, head)
			}
			below = rr.Max.Y
		}
		t.Logf("card %s: popup %d wide, card %d wide", c.label, popup.Dx(), card.Dx())
	}
}

// Speaker 22 takes the opposite of the hero's sex and opens the speaker
// gate, so its bare female and male tags part the heroes by sex: the woman
// reads "he", the man "she". Speaker 51 opens no gate, so giving the man's
// tag speaker 51 would give it to the women too. Apply refuses that edit and
// keeps the page; an edit that keeps every hero's tag is written.
func TestDialogVariantSpeakerEditReachingAnotherCellIsRefused(t *testing.T) {
	raw := "<part=1,npc=22,female>\r\nshe\r\n<part=1,npc=22,male>\r\nhe"
	_, s := variantFixture(t, raw)
	s.inspector.table = &mapload.Table{NPCCatalog: &data.NPCCatalog{
		Entries: []data.NPCEntry{{ID: 21, Name: "Hero"}, {ID: 22, Name: "Rival"}, {ID: 51, Name: "Elder"}},
		Faces:   map[int32]data.NPCFace{22: {Start: true, Tokens: tokens(data.NPCTokenNotMySex, data.NPCTokenMage)}},
	}}
	s.speakers = nil
	cells := s.MessageParts(7)
	if len(cells) != 2 || cells[0].Source != 1 || cells[1].Source != 0 {
		t.Fatalf("page 7 reads as %+v", cells)
	}
	edit := slices.Clone(cells)
	edit[1].Speaker = 51
	if _, err := s.SetMessageParts(7, edit); err == nil || string(s.workspace.Pages["event07.txt"]) != raw {
		t.Fatalf("a speaker edit reaching the women was written (err %v): %q", err, s.workspace.Pages["event07.txt"])
	}
	edit = slices.Clone(cells)
	edit[1].Body = "she!"
	if _, err := s.SetMessageParts(7, edit); err != nil || string(s.workspace.Pages["event07.txt"]) != "<part=1,npc=22,female>\r\nshe!\r\n<part=1,npc=22,male>\r\nhe" {
		t.Fatalf("a text edit was refused or misplaced (err %v): %q", err, s.workspace.Pages["event07.txt"])
	}
}

// No speaker for the mage's cell writes one tag whether the part is written
// in place (only that edit) or cell by cell (the same edit beside deleting
// the part's never-shown tag): the speaker and one separator beside it go.
func TestDialogVariantNoSpeakerWritesOneTagOnBothPaths(t *testing.T) {
	raw := "<part=1,npc=21,iammage>\r\nA\r\n<part=1,iamfighter>\r\nB\r\n<part=1>\r\nN"
	want := "<part=1,iammage>\r\nA\r\n<part=1,iamfighter>\r\nB"
	for _, drop := range []bool{false, true} {
		_, s := variantFixture(t, raw)
		cells := s.MessageParts(7)
		if len(cells) != 3 || cells[0].Source != 0 || cells[0].Speaker != 21 || !cells[2].Never {
			t.Fatalf("page 7 reads as %+v", cells)
		}
		edit := slices.Clone(cells)
		edit[0].Speaker = -1
		if drop {
			edit = edit[:2]
		}
		if _, err := s.SetMessageParts(7, edit); err != nil {
			t.Fatal(err)
		}
		if got := string(s.workspace.Pages["event07.txt"]); !strings.HasPrefix(got, want) {
			t.Errorf("never-shown tag dropped %v: wrote %q, want it to open %q", drop, got, want)
		}
	}
}
