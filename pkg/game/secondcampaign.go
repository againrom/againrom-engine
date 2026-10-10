package game

import (
	"fmt"
	"image"
	"strings"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type secondLocation struct {
	kind int
	id   int
}

type secondTownRoom uint8

const (
	secondTownSquare secondTownRoom = iota
	secondTownInn
)

// secondAux is the eight-entry auxiliary array of five DWORD fields (+0,
// +4, +8, +0xC, +0x10) the departure stores outside the bank. Its
// consumers are Unknown; it is only stored and persisted (DIV-2437).
type secondAux [8][5]int32

type secondCampaign struct {
	bank      [1024]int32
	aux       secondAux
	available []secondLocation
	current   secondLocation
	room      secondTownRoom
	selected  secondLocation
	payload   []byte
	part      int
}

func newSecondCampaign() *secondCampaign {
	initial := secondLocation{kind: 2, id: 1}
	c := &secondCampaign{available: []secondLocation{initial}, current: initial}
	c.bank[768] = 10
	return c
}

func (c *secondCampaign) has(location secondLocation) bool {
	for _, candidate := range c.available {
		if candidate == location {
			return true
		}
	}
	return false
}

// leaveTown is the type-2 departure: town 1 removes its node, a later town
// keeps it, and both clear current (R2-ENGINE-144). Town 1's gate opens
// only once mission 10 is available.
func (c *secondCampaign) leaveTown() bool {
	if c.current.kind != 2 || !c.gateOpen() {
		return false
	}
	if c.current.id == 1 {
		remaining := c.available[:0]
		for _, l := range c.available {
			if l != c.current {
				remaining = append(remaining, l)
			}
		}
		c.available = remaining
	}
	c.current = secondLocation{}
	return true
}

func (c *secondCampaign) gateOpen() bool {
	return c.current.id != 1 || c.has(secondLocation{kind: 1, id: 10})
}

func (c *secondCampaign) canEnter(n int) bool {
	return c.current == (secondLocation{}) && c.has(secondLocation{kind: 1, id: n})
}

// finish admits the ordinary completion of mission n. won is the front end's
// one win predicate: the script outcome or the announced client outcome, which
// the victory command reaches through the same completion (R2-ENGINE-303).
func (c *secondCampaign) finish(n int, w *sim.World, won bool) error {
	if !secondContinues(n) || c.current != (secondLocation{kind: 1, id: n}) || !c.has(c.current) || w == nil || !won {
		return fmt.Errorf("campaign continuation after mission %d is unavailable", n)
	}
	bank, ok := w.ROM2ScenarioState()
	if !ok || bank[775] != 0 {
		return fmt.Errorf("campaign continuation requires ordinary state without bank775 restoration")
	}
	return nil
}

// secondContinues names the missions whose victory the controller continues:
// every ordinary ID whose slot 896+ID lies inside the bank (R2-SESSION-052).
func secondContinues(n int) bool { return n > 0 && n < 128 }

// complete applies the ordinary departure and answers its output DWORD.
func (c *secondCampaign) complete(w *sim.World) int {
	bank, _ := w.ROM2ScenarioState()
	return c.completeBank(bank)
}

// secondStageAux is field +4 of the auxiliary entries each stage case
// stores, and how many entries from 0 it covers (R2-SESSION-050).
var secondStageAux = map[int32]struct {
	value   int32
	entries int
}{
	30: {10000, 4}, 40: {22000, 4}, 50: {60000, 8}, 60: {150000, 8}, 70: {400000, 8},
	80: {800000, 8}, 90: {1500000, 8}, 100: {5000000, 8}, 110: {10000000, 8},
}

// completeBank applies the ordinary departure to the bank a won mission left
// and answers the output DWORD: -1 unless a case body stores one
// (R2-ENGINE-145, R2-ENGINE-146, R2-ENGINE-148, R2-SESSION-047..050).
func (c *secondCampaign) completeBank(bank [1024]int32) int {
	n := c.current.id
	output := -1
	c.bank = bank
	c.bank[773] = 0
	for i := 0; i < 20; i++ {
		if c.bank[532+i] != 0 {
			c.bank[532+i] = 1
			if c.bank[512+i] != 0 {
				c.bank[532+i] = 2
			}
		}
		c.bank[512+i] = 0
	}
	c.bank[896+n] = 1
	remaining := c.available[:0]
	for _, l := range c.available {
		if l != c.current {
			remaining = append(remaining, l)
		}
	}
	c.available = remaining
	add := func(l secondLocation) {
		if !c.has(l) {
			c.available = append(c.available, l)
		}
	}
	switch n {
	case 10:
		add(secondLocation{1, 20})
		output = 1
	case 20:
		add(secondLocation{2, 2})
		if c.bank[772] != 0 {
			add(secondLocation{1, 21})
		}
		c.bank[532], c.bank[552] = 1, 2
	case 30:
		output = 2
	case 31:
		add(secondLocation{1, 32})
	case 40:
		add(secondLocation{1, 50})
		add(secondLocation{1, 60})
		c.bank[773] = 23
	case 50:
		// The bank780-gated fixed record has no resolved type or ID and is
		// not appended (DIV-2629).
		add(secondLocation{2, 3})
		c.bank[771], c.bank[534], c.bank[554] = 1, 1, 3
		for i := 4; i < 8; i++ {
			c.aux[i][0], c.aux[i][2], c.aux[i][3] = 499, 100, 2
			if i == 5 || i == 6 {
				c.aux[i][2], c.aux[i][3] = 20, 1
			}
		}
		c.aux[6][0] = 0
	case 60:
		add(secondLocation{1, 80})
		c.bank[537], c.bank[557], c.bank[774] = 1, 2, 1
	case 70:
		c.bank[535], c.bank[555] = 1, 3
		if c.bank[777] == 0 && c.bank[778] != 0 {
			output = 3
		}
		c.bank[777], c.bank[770] = 1, 1
	case 80:
		if c.bank[778] == 0 && c.bank[777] != 0 {
			output = 3
		}
		c.bank[778] = 1
	case 100:
		c.bank[538], c.bank[558] = 1, 2
	case 110:
		output = 4
		if c.bank[779] != 0 {
			output = 5
		}
	}
	if n%10 == 0 {
		c.bank[768] += 10
	}
	if stage, ok := secondStageAux[c.bank[768]]; ok {
		for i := 0; i < stage.entries; i++ {
			c.aux[i][1] = stage.value
		}
		if extra := map[int32]int32{90: 12000, 100: 40000}[c.bank[768]]; extra != 0 {
			for _, i := range []int{0, 1, 3} {
				c.aux[i][0] = extra
			}
		}
	}
	c.current = secondLocation{}
	c.selected, c.room = secondLocation{}, secondTownSquare
	return output
}

type secondCampaignScreen struct {
	session *CampaignSession
	install *InstallResources
	open    missionDoor
	frame   func() *ui.DialogFrame
	start   func() error
}

func (f *FrontEnd) startSecondCampaign() error {
	releaseWorldAudio(f.runtimeAudio(), f.endLive())
	f.CampaignSession.clear(Campaign{})
	f.Town.second = newSecondCampaign()
	f.Carried = MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table)
	return nil
}

func (f *FrontEnd) secondCampaignScreen() ui.TownScreen {
	return &secondCampaignScreen{session: &f.CampaignSession, install: &f.InstallResources,
		open: f.MissionOpener, frame: f.gameMenuArt, start: f.startSecondCampaign}
}

func (t *secondCampaignScreen) StartNewGame() error {
	if _, err := t.install.Archives.Containers.ReadFile(mainPrefix + "text/town.txt"); err != nil {
		return fmt.Errorf("initial campaign town: %w", err)
	}
	if err := t.start(); err != nil {
		return err
	}
	return nil
}

func (t *secondCampaignScreen) state() *secondCampaign {
	if t.session.Town == nil {
		return nil
	}
	return t.session.Town.second
}

func (t *secondCampaignScreen) Header() string {
	if c := t.state(); c != nil && c.current.kind == 2 {
		if c.room == secondTownInn {
			return "ROM2 campaign: tavern"
		}
		return fmt.Sprintf("ROM2 campaign: town %d", c.current.id)
	}
	return "ROM2 campaign: destinations"
}

func (t *secondCampaignScreen) Rows() []ui.TownRow {
	c := t.state()
	if c == nil {
		return nil
	}
	if c.current.kind == 2 {
		gates := ui.TownRow{Text: "GATES", Choosable: c.gateOpen()}
		if c.room != secondTownInn {
			return []ui.TownRow{{Text: "TAVERN", Choosable: true}, gates}
		}
		var rows []ui.TownRow
		for _, o := range c.speakers() {
			rows = append(rows, ui.TownRow{Text: fmt.Sprintf("TALK %d", o.npc), Choosable: true})
		}
		return append(rows, gates)
	}
	var rows []ui.TownRow
	for _, location := range c.available {
		name := "mission"
		if location.kind == 2 {
			name = "town"
		}
		rows = append(rows, ui.TownRow{Text: fmt.Sprintf("%s %d", name, location.id), Choosable: true})
	}
	if c.has(c.selected) {
		rows = append(rows, ui.TownRow{Text: "ENTER", Choosable: c.current == (secondLocation{})}, ui.TownRow{Text: "CANCEL", Choosable: true})
	}
	return rows
}

func (t *secondCampaignScreen) Footer() []string {
	c := t.state()
	if c == nil {
		return nil
	}
	var out []string
	for id := 2; id <= 3; id++ {
		if town := (secondLocation{2, id}); c.current == town || c.has(town) {
			out = append(out, fmt.Sprintf("Town %d services are unavailable.", id))
		}
	}
	return out
}

func (t *secondCampaignScreen) Choose(i int) ui.TownAction {
	rows := t.Rows()
	if i < 0 || i >= len(rows) || !rows[i].Choosable {
		return ui.TownAction{}
	}
	c := t.state()
	if c.current.kind != 2 {
		if i < len(c.available) {
			c.selected = c.available[i]
		} else if i == len(c.available) {
			if c.selected.kind == 2 {
				c.current, c.selected, c.room = c.selected, secondLocation{}, secondTownSquare
				return ui.TownAction{}
			}
			return ui.TownAction{Open: t.open(c.selected.id)}
		} else {
			c.selected = secondLocation{}
		}
		return ui.TownAction{}
	}
	if i == len(rows)-1 {
		c.leaveTown()
		c.room = secondTownSquare
		return ui.TownAction{}
	}
	if c.room != secondTownInn {
		c.room = secondTownInn
		return ui.TownAction{}
	}
	o := c.speakers()[i]
	payload, err := readSecondTownTalk(t.install, secondTalkKey(o))
	if err != nil {
		return ui.TownAction{Msg: err.Error()}
	}
	c.talkTo(o)
	if payload != nil {
		c.payload, c.part = payload, 1
	}
	return ui.TownAction{}
}

func (t *secondCampaignScreen) Back() bool {
	c := t.state()
	if c == nil {
		return false
	}
	if c.payload != nil {
		t.AdvanceTownDialogue()
		return true
	}
	if c.selected != (secondLocation{}) {
		c.selected = secondLocation{}
		return true
	}
	if c.room == secondTownInn {
		c.room = secondTownSquare
		return true
	}
	return false
}

func (t *secondCampaignScreen) dialogueBody() (string, bool) {
	c := t.state()
	if c == nil {
		return "", false
	}
	if c.payload == nil {
		return "", false
	}
	audience := HeroAudience(t.session.Carried)
	audience.SecondGame = true
	return dialoguePart(c.payload, c.part, audience)
}

func (t *secondCampaignScreen) dialogueLayout() ui.NoticeLayout {
	layout := t.install.Words.OnLayout(ui.AuthoredDialogueLayout()).WithPortrait(false)
	layout.Frame = t.frame()
	return layout
}

func (t *secondCampaignScreen) TownDialogue() (*image.RGBA, bool) {
	body, ok := t.dialogueBody()
	if !ok {
		return nil, false
	}
	return ui.RenderNotice(t.dialogueLayout(), t.install.Font.Value(), strings.TrimSpace(body), nil), true
}

func (t *secondCampaignScreen) TownDialogueButton() (image.Rectangle, bool) {
	_, ok := t.dialogueBody()
	return t.dialogueLayout().Button, ok
}

func (t *secondCampaignScreen) AdvanceTownDialogue() ui.TownAction {
	c := t.state()
	if c == nil {
		return ui.TownAction{}
	}
	c.part++
	if _, ok := t.dialogueBody(); !ok {
		c.payload, c.part = nil, 0
	}
	return ui.TownAction{}
}

func (t *secondCampaignScreen) CanSave() bool { return t.state().savePoint() }

func (c *secondCampaign) savePoint() bool {
	return c != nil && c.selected == (secondLocation{}) && c.payload == nil && c.part == 0 && captureSecondCampaign(c).validateTown() == nil
}

// noticePending reports whether the campaign holds an open dialogue or a
// selected destination; no campaign holds neither.
func (c *secondCampaign) noticePending() bool {
	return c != nil && (c.payload != nil || c.part != 0 || c.selected != (secondLocation{}))
}

// readSecondTownTalk reads the npc%dtalk%d section a TALK dispatches; a
// missing section shows no conversation (DIV-2635).
func readSecondTownTalk(install *InstallResources, key string) ([]byte, error) {
	payload, err := install.Archives.Containers.ReadFile(mainPrefix + "text/town.txt")
	if err != nil {
		return nil, fmt.Errorf("initial campaign town: %w", err)
	}
	body := secondGameTextSection(secondGameMissionBytes(payload, LanguageSelector(install.Archives.Containers)), key)
	if body == "" {
		return nil, nil
	}
	return []byte(body), nil
}
