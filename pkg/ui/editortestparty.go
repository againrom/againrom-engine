package ui

import "image"

// TestPlayHeroes is the number of kinds a hero is, in the order male
// fighter, female fighter, male mage, female mage.
const TestPlayHeroes = 4

// TestPlayCard is one character the party window before a test play offers.
// Hero marks the map maker's own hero, whom the window names itself; any
// other card carries the game's name of its character and Key, the
// character the card stands for whatever hero it is offered beside, which
// the choice is remembered by. Portrait is nil when the game holds no
// picture of the character, and Caster says that the character casts, so
// the card takes the spells box.
type TestPlayCard struct {
	Name     string
	Key      string
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

// InspectionTestPlayDeclared is implemented by hosts whose open mission may
// declare its own party. The party window then carries a switch: Standard
// starts the mission with that party, Custom with the window's choice.
// TestPlayDeclaresParty reports that the open mission declares one.
type InspectionTestPlayDeclared interface{ TestPlayDeclaresParty() bool }

// testPartyPrompt is what the party window holds while it is open: the
// cards and the kind of hero they are offered beside.
type testPartyPrompt struct {
	host  InspectionTestPlayParty
	cards []TestPlayCard
	hero  int
	// declared: the mission declares its own party; the window then holds the
	// switch between that party and the window's choice.
	declared bool
	// heroIn: the choice had nobody in, so the open window shows the hero in.
	// It belongs to the window; the remembered choice takes it only when the
	// map maker changes the hero's card or presses Start.
	heroIn bool
}

// testPartyHeroKey is the key the hero's card is remembered by.
const testPartyHeroKey = "hero"

// testPartyKey is the key card c's choice is remembered by.
func testPartyKey(c TestPlayCard) string {
	if c.Hero {
		return testPartyHeroKey
	}
	return c.Key
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
	testPartyMode
)

// testPartyKinds are the words of the kinds a hero is, in their order.
var testPartyKinds = [TestPlayHeroes]string{"editor.message.hero.manfighter", "editor.message.hero.womanfighter",
	"editor.message.hero.manmage", "editor.message.hero.womanmage"}

// testPartySide is a card's width, wide enough for the longest companion
// name, and testPartyHigh its portrait's height.
const testPartySide, testPartyHigh = 104, 80

// askTestParty opens the party window before a test play when the host
// offers characters, and reports whether it did. The window starts from the
// kind of hero and the boxes the map maker last chose, kept in the editor's
// profile, or from a man fighter alone. A choice that puts none of the cards
// in the party puts the hero in.
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
	d, _ := e.doc.Edits.(InspectionTestPlayDeclared)
	e.openPrompt(editorPrompt{kind: promptTestParty, party: &testPartyPrompt{host: host, hero: hero, cards: cards, declared: d != nil && d.TestPlayDeclaresParty()}})
	e.prompt.party.heroIn = !e.testPartyAny()
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

// testPartyKeep remembers m as the choice of the card keyed key.
func (e *MapEditor) testPartyKeep(key string, m TestPlayMember) {
	if e.testPartyKept == nil {
		e.testPartyKept = map[string]TestPlayMember{}
	}
	e.testPartyKept[key] = m
}

// testPartyMember is the remembered choice of card c: the hero alone plays
// on a card the map maker never chose for. A spells tick stays remembered
// on a card that does not cast; the window shows it and the play takes it
// only where the card casts.
func (e *MapEditor) testPartyMember(c TestPlayCard) TestPlayMember {
	if c.Hero && e.prompt.party != nil && e.prompt.party.heroIn {
		m := e.testPartyKept[testPartyHeroKey]
		m.On = true
		return m
	}
	if m, ok := e.testPartyKept[testPartyKey(c)]; ok {
		return m
	}
	return TestPlayMember{On: c.Hero}
}

// TestPartyHero is the kind of hero the party window's cards stand beside,
// or the kind the next window starts from.
func (e *MapEditor) TestPartyHero() int {
	if p := e.prompt.party; p != nil {
		return p.hero
	}
	return e.testPartyHero
}

// TestPartyOpen reports whether the party window before a test play is open.
func (e *MapEditor) TestPartyOpen() bool { return e.prompt.kind == promptTestParty }

// TestPartyChoice is the open party window's choice, one entry per card,
// with the spells ticks remembered on cards that do not cast; none while no
// window is open.
func (e *MapEditor) TestPartyChoice() []TestPlayMember {
	p := e.prompt.party
	if p == nil {
		return nil
	}
	out := make([]TestPlayMember, len(p.cards))
	for i, c := range p.cards {
		out[i] = e.testPartyMember(c)
	}
	return out
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
	for _, m := range e.TestPartyChoice() {
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
	kind := e.word("editor.testparty.hero") + ": " + e.word(testPartyKinds[e.prompt.party.hero])
	buttons := []string{e.word("editor.testparty.start"), kind, e.word("editor.button.cancel")}
	custom := !e.prompt.party.declared || e.testPartyCustom
	disabled := []bool{custom && !e.testPartyAny(), !custom, false}
	if e.prompt.party.declared {
		mode := "editor.testparty.standard"
		if e.testPartyCustom {
			mode = "editor.testparty.custom"
		}
		buttons, disabled = append(buttons, e.word("editor.testparty.mode")+": "+e.word(mode)), append(disabled, false)
	}
	s := e.questionSpec(e.word("editor.testparty.title"), notes, buttons)
	s.Disabled = disabled
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
	choice := e.TestPartyChoice()
	for row := range testPartyRows {
		for i, c := range cards {
			name := c.Name
			if c.Hero {
				name = e.word("editor.testparty.hero")
			}
			if row == testPartyRowPlays {
				g.ColHeads = append(g.ColHeads, name)
			}
			g.Cells = append(g.Cells, testPartyCell(row, c, name, choice[i], custom))
		}
	}
	s.Grid = g
	s.Width = min(max(s.Width, 2*popupPad+g.HeadW+len(cards)*g.Side), e.window.X-16)
	s.At = e.promptCorner(centreQuestion, layoutPopup(s, e.font, body).Rect.Size())
	return s
}

// testPartyCell is the box of row on card c, named name, under choice m. A
// card that does not cast shows no spells tick.
func testPartyCell(row int, c TestPlayCard, name string, m TestPlayMember, custom bool) popupGridCell {
	m.Spells = m.Spells && c.Caster
	switch row {
	case testPartyRowPlays:
		cell := popupGridCell{Icon: c.Portrait, Tip: name, Pressed: m.On, Ticked: m.On, Inert: !custom}
		if c.Portrait == nil {
			cell.Label = name
		}
		return cell
	case testPartyRowSkills:
		return popupGridCell{Pressed: m.Skills, Ticked: m.Skills, Inert: !custom}
	}
	return popupGridCell{Pressed: m.Spells, Ticked: m.Spells, Inert: !c.Caster || !custom}
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
	case part == popupButton && n == testPartyMode && e.prompt.party.declared:
		e.testPartyCustom = !e.testPartyCustom
		e.savePanels()
	case part == popupButton && n == testPartyKind:
		p := e.prompt.party
		if hero, cards := testPartyOffer(p.host, p.hero+1); len(cards) != 0 {
			p.hero, p.cards, e.testPartyHero = hero, cards, hero
			e.savePanels()
		}
	case part == popupGridPart:
		if cols := len(e.prompt.party.cards); cols > 0 {
			e.testPartySwitch(n/cols, n%cols)
		}
	}
}

// testPartySwitch switches the box in row of card. The choice is the
// editor's and its profile's, so the next window starts from it, in this
// run and the next.
func (e *MapEditor) testPartySwitch(row, card int) {
	p := e.prompt.party
	if p == nil || card < 0 || card >= len(p.cards) || p.declared && !e.testPartyCustom {
		return
	}
	c := p.cards[card]
	m := e.testPartyMember(c)
	switch row {
	case testPartyRowPlays:
		m.On = !m.On
	case testPartyRowSkills:
		m.Skills = !m.Skills
	case testPartyRowSpells:
		if !c.Caster {
			return
		}
		m.Spells = !m.Spells
	default:
		return
	}
	e.testPartyKeep(testPartyKey(c), m)
	if c.Hero {
		p.heroIn = false
	}
	e.savePanels()
}

// testPartyGo leaves the party window and plays the map with the choice,
// the spells ticks of cards that do not cast left out; with nobody chosen
// it stays. The kind shown is the one the next window starts from.
func (e *MapEditor) testPartyGo() error {
	p := e.prompt.party
	if p != nil && p.declared && !e.testPartyCustom {
		e.leave(leavePrompt)
		return e.testPlay()
	}
	if p == nil || !e.testPartyAny() {
		return nil
	}
	hero, choice := p.hero, e.TestPartyChoice()
	for i, c := range p.cards {
		choice[i].Spells = choice[i].Spells && c.Caster
		if c.Hero && p.heroIn {
			e.testPartyKeep(testPartyHeroKey, e.testPartyMember(c))
			e.savePanels()
		}
	}
	if e.testPartyHero != hero {
		e.testPartyHero = hero
		e.savePanels()
	}
	e.leave(leavePrompt)
	return e.testPlayBy(func() (string, error) { return p.host.TestPlayParty(hero, choice) })
}
