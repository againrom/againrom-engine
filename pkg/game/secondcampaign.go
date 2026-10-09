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

type secondCampaign struct {
	bank      [1024]int32
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

func (c *secondCampaign) talk() bool {
	if c.current != (secondLocation{kind: 2, id: 1}) || c.bank[768] != 10 {
		return false
	}
	mission := secondLocation{kind: 1, id: 10}
	if !c.has(mission) {
		c.available = append(c.available, mission)
	}
	return true
}

func (c *secondCampaign) leaveTown() bool {
	if c.current == (secondLocation{2, 2}) {
		c.current = secondLocation{}
		return true
	}
	if c.current != (secondLocation{kind: 2, id: 1}) || !c.has(secondLocation{kind: 1, id: 10}) {
		return false
	}
	c.available = []secondLocation{{kind: 1, id: 10}}
	c.current = secondLocation{}
	return true
}

func (c *secondCampaign) canEnter(n int) bool {
	return c.current == (secondLocation{}) && c.has(secondLocation{kind: 1, id: n})
}

func (c *secondCampaign) finish(n int, w *sim.World) error {
	if !secondContinues(n) || c.current != (secondLocation{kind: 1, id: n}) || !c.has(c.current) || w == nil || w.Outcome() != sim.OutcomeWon {
		return fmt.Errorf("campaign continuation after mission %d is unavailable", n)
	}
	bank, ok := w.ROM2ScenarioState()
	if !ok || bank[775] != 0 {
		return fmt.Errorf("campaign continuation requires ordinary state without bank775 restoration")
	}
	return nil
}

// secondContinues names the missions whose victory the controller continues.
func secondContinues(n int) bool { return n == 10 || n == 20 || n == 21 }

func (c *secondCampaign) complete(w *sim.World) {
	bank, _ := w.ROM2ScenarioState()
	c.completeBank(bank)
}

// completeBank applies the ordinary departure to the bank a won mission left.
func (c *secondCampaign) completeBank(bank [1024]int32) {
	n := c.current.id
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
	case 20:
		add(secondLocation{2, 2})
		if c.bank[772] != 0 {
			add(secondLocation{1, 21})
		}
		c.bank[532], c.bank[552] = 1, 2
	}
	if n%10 == 0 {
		c.bank[768] += 10
	}
	c.current = secondLocation{}
	c.selected, c.room = secondLocation{}, secondTownSquare
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
		if c.current.id == 2 {
			return []ui.TownRow{{Text: "GATES", Choosable: true}}
		}
		if c.room == secondTownInn {
			return []ui.TownRow{{Text: "TALK", Choosable: true}, {Text: "GATES", Choosable: c.has(secondLocation{kind: 1, id: 10})}}
		}
		return []ui.TownRow{{Text: "TAVERN", Choosable: true}, {Text: "GATES", Choosable: c.has(secondLocation{kind: 1, id: 10})}}
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
	if c := t.state(); c != nil && (c.current == (secondLocation{2, 2}) || c.has(secondLocation{2, 2})) {
		return []string{"Town 2 conversations and services are unavailable."}
	}
	return nil
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
	if i == 1 || c.current.id == 2 {
		c.leaveTown()
		c.room = secondTownSquare
		return ui.TownAction{}
	}
	if c.room != secondTownInn {
		c.room = secondTownInn
	} else {
		payload, err := readSecondTownTalk(t.install)
		if err != nil {
			return ui.TownAction{Msg: err.Error()}
		}
		if c.talk() {
			c.payload, c.part = payload, 1
		}
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

func readSecondTownTalk(install *InstallResources) ([]byte, error) {
	payload, err := install.Archives.Containers.ReadFile(mainPrefix + "text/town.txt")
	if err != nil {
		return nil, fmt.Errorf("initial campaign town: %w", err)
	}
	body := secondGameTextSection(secondGameMissionBytes(payload, LanguageSelector(install.Archives.Containers)), "npc517talk10")
	if body == "" {
		return nil, fmt.Errorf("initial inn conversation is unavailable")
	}
	return []byte(body), nil
}
