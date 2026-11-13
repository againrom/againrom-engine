package ui

import "image"

// TestPlayHeroes is the number of kinds a hero is, in the order male
// fighter, female fighter, male mage, female mage.
const TestPlayHeroes = 4

// TestPlayCard is one character the party window before a test play offers.
// Hero marks the map maker's own hero, whom the window names itself; any
// other card carries the game's name of its character. Portrait is nil when
// the game holds no picture of the character, and Caster says that the
// character casts, so the card takes the spells box.
type TestPlayCard struct {
	Name     string
	Portrait *image.RGBA
	Hero     bool
	Caster   bool
}

// TestPlayMember is what the window chose for one card: whether the
// character plays, and whether he comes with every skill at the cap and
// with every spell he can learn.
type TestPlayMember struct {
	On, Skills, Spells bool
}

// InspectionTestPlayParty is implemented by edits whose host lets the map
// maker choose who plays the test. TestPlayCards are the characters offered
// beside a hero of kind hero, in the window's order: the companions as the
// game builds them for that hero. None when the open map's test play takes
// no chosen party or does not play for that hero. TestPlayParty is TestPlay
// for that hero with the choice, one entry per card.
type InspectionTestPlayParty interface {
	TestPlayCards(hero int) []TestPlayCard
	TestPlayParty(hero int, choice []TestPlayMember) (string, error)
}

// testPartyPrompt is what the party window holds while it is open.
type testPartyPrompt struct {
	host  InspectionTestPlayParty
	cards []TestPlayCard
}

// The party window's rows and buttons, in order.
const (
	testPartyRowPlays = iota
	testPartyRowSkills
	testPartyRowSpells
	testPartyRows
)

const (
	testPartyStart = iota
	testPartyKind
	testPartyStay
)

// testPartyKinds are the words of the kinds a hero is, in their order.
var testPartyKinds = [TestPlayHeroes]string{"editor.message.hero.manfighter", "editor.message.hero.womanfighter",
	"editor.message.hero.manmage", "editor.message.hero.womanmage"}

// testPartySide is a card's width, wide enough for the longest companion
// name, and testPartyHigh its portrait's height.
const testPartySide, testPartyHigh = 104, 80

// askTestParty opens the party window before a test play when the host
// offers characters, and reports whether it did. The window starts from the
// choice the last one in this editor made, or from the hero alone.
func (e *MapEditor) askTestParty() bool {
	host, ok := e.doc.Edits.(InspectionTestPlayParty)
	if !ok {
		return false
	}
	hero, cards := testPartyOffer(host, e.testPartyHero)
	if len(cards) == 0 {
		return false
	}
	e.leave(leaveDrag | leavePick)
	e.openPrompt(editorPrompt{kind: promptTestParty, party: &testPartyPrompt{host: host}})
	e.testPartyDeal(hero, cards)
	return true
}

// testPartyOffer is the first kind of hero, from kind from on and round,
// for whom host offers cards, and those cards.
func testPartyOffer(host InspectionTestPlayParty, from int) (int, []TestPlayCard) {
	for i := range TestPlayHeroes {
		hero := (from + i) % TestPlayHeroes
		if cards := host.TestPlayCards(hero); len(cards) != 0 {
			return hero, cards
		}
	}
	return from, nil
}

// testPartyDeal lays cards, offered for a hero of kind hero, in the open
// window. The choice stays, less the spells of a card that no longer casts;
// a different number of cards starts from the hero alone.
func (e *MapEditor) testPartyDeal(hero int, cards []TestPlayCard) {
	if len(e.testParty) != len(cards) {
		e.testParty = make([]TestPlayMember, len(cards))
		e.testParty[0].On = true
	}
	for i, c := range cards {
		e.testParty[i].Spells = e.testParty[i].Spells && c.Caster
	}
	e.testPartyHero, e.prompt.party.cards = hero, cards
}

// TestPartyHero is the kind of hero the party window's cards stand beside.
func (e *MapEditor) TestPartyHero() int { return e.testPartyHero }

// TestPartyOpen reports whether the party window before a test play is open.
func (e *MapEditor) TestPartyOpen() bool { return e.prompt.kind == promptTestParty }

// TestPartyChoice is the party window's choice, one entry per card: the
// open window's, or the one the next window starts from.
func (e *MapEditor) TestPartyChoice() []TestPlayMember {
	return append([]TestPlayMember(nil), e.testParty...)
}

// controlPromptGridCellRect is the window rectangle of the open prompt's grid
// box in row and column col; false for a box that takes no press. In the
// party window the rows are plays, skills and spells and a column is a card.
func (e *MapEditor) controlPromptGridCellRect(row, col int) (image.Rectangle, bool) {
	s, ok := e.promptSpec()
	if !ok || s.Grid == nil {
		return image.Rectangle{}, false
	}
	n := row*s.Grid.Cols + col
	l := layoutPopup(s, e.font, e.bodyFont())
	if row < 0 || col < 0 || col >= s.Grid.Cols || n >= len(l.GridCells) || l.GridInert[n] {
		return image.Rectangle{}, false
	}
	return l.GridCells[n], true
}

// testPartyAny reports that the choice has a character who plays.
func (e *MapEditor) testPartyAny() bool {
	for _, m := range e.testParty {
		if m.On {
			return true
		}
	}
	return false
}

// testPartySpec is the party window as a popup description: one column per
// card under the character's name, his portrait as the box that says he
// plays, and under it the skills and spells boxes; a card whose character
// does not cast has no spells box. The lines the notice before a test play
// lists stand in the window, and Start takes no press with nobody chosen.
// The middle button names the kind of hero the cards stand beside, since
// the game composes three of the companions by the hero's class and sex; a
// press takes the next kind. A name wider than its card takes two lines.
func (e *MapEditor) testPartySpec() popupSpec {
	cards := e.prompt.party.cards
	var notes []string
	if e.noticeNeeded(noticeTestPlay) {
		notes = e.noticeNotes()
	}
	kind := e.word("editor.testparty.hero") + ": " + e.word(testPartyKinds[e.testPartyHero])
	s := e.questionSpec(e.word("editor.testparty.title"), notes, []string{e.word("editor.testparty.start"), kind, e.word("editor.button.cancel")})
	s.Disabled = []bool{!e.testPartyAny(), false, false}
	body := e.bodyFont()
	heads := []string{e.word("editor.testparty.plays"), e.word("editor.testparty.skills"), e.word("editor.testparty.spells")}
	line := fontHeight(body, 15) + popupRowExtra
	g := &popupGrid{Rows: testPartyRows, Cols: len(cards), RowHeads: heads, HeadW: 8, HeadH: line, HeadFlat: true}
	if body != nil {
		for _, h := range heads {
			g.HeadW = max(g.HeadW, body.Advance(h)+8)
		}
	}
	g.Side = max(24, min(testPartySide, (e.window.X-16-2*popupPad-g.HeadW)/len(cards)))
	g.RowHigh = []int{min(g.Side, testPartyHigh), line + 4, line + 4}
	for _, c := range cards {
		if len(headLines(body, c.Name, g.Side-4)) > 1 {
			g.HeadH = 2 * line
		}
	}
	for row := range testPartyRows {
		for i, c := range cards {
			name := c.Name
			if c.Hero {
				name = e.word("editor.testparty.hero")
			}
			if row == testPartyRowPlays {
				g.ColHeads = append(g.ColHeads, name)
			}
			g.Cells = append(g.Cells, testPartyCell(row, c, name, e.testParty[i]))
		}
	}
	s.Grid = g
	s.Width = min(max(s.Width, 2*popupPad+g.HeadW+len(cards)*g.Side), e.window.X-16)
	s.At = e.promptCorner(centreQuestion, layoutPopup(s, e.font, body).Rect.Size())
	return s
}

// testPartyCell is the box of row on card c, named name, under choice m.
func testPartyCell(row int, c TestPlayCard, name string, m TestPlayMember) popupGridCell {
	switch row {
	case testPartyRowPlays:
		cell := popupGridCell{Icon: c.Portrait, Tip: name, Pressed: m.On, Ticked: m.On}
		if c.Portrait == nil {
			cell.Label = name
		}
		return cell
	case testPartyRowSkills:
		return popupGridCell{Pressed: m.Skills, Ticked: m.Skills}
	}
	return popupGridCell{Pressed: m.Spells, Ticked: m.Spells, Inert: !c.Caster}
}

// testPartyPress is a press on the party window: a box switches, Start
// plays and Cancel or the close button leaves.
func (e *MapEditor) testPartyPress(p image.Point) {
	s, _ := e.promptSpec()
	part, n := layoutPopup(s, e.font, e.bodyFont()).hit(p)
	switch {
	case part == popupClose || part == popupButton && n == testPartyStay:
		e.cancelPopup()
	case part == popupButton && n == testPartyStart:
		_ = e.testPartyGo()
	case part == popupButton && n == testPartyKind:
		if hero, cards := testPartyOffer(e.prompt.party.host, e.testPartyHero+1); len(cards) != 0 {
			e.testPartyDeal(hero, cards)
		}
	case part == popupGridPart:
		e.testPartySwitch(n/len(e.testParty), n%len(e.testParty))
	}
}

// testPartySwitch switches the box in row of card. The choice is the
// editor's, so the next window in this session starts from it.
func (e *MapEditor) testPartySwitch(row, card int) {
	if e.prompt.party == nil || card < 0 || card >= len(e.testParty) {
		return
	}
	m := &e.testParty[card]
	switch row {
	case testPartyRowPlays:
		m.On = !m.On
	case testPartyRowSkills:
		m.Skills = !m.Skills
	case testPartyRowSpells:
		if e.prompt.party.cards[card].Caster {
			m.Spells = !m.Spells
		}
	}
}

// testPartyGo leaves the party window and plays the map with the choice;
// with nobody chosen it stays.
func (e *MapEditor) testPartyGo() error {
	p := e.prompt.party
	if p == nil || !e.testPartyAny() {
		return nil
	}
	hero, choice := e.testPartyHero, e.TestPartyChoice()
	e.leave(leavePrompt)
	return e.testPlayBy(func() (string, error) { return p.host.TestPlayParty(hero, choice) })
}
