package game

import (
	"fmt"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// campaignDifficulty accepts the old native-save zero as Normal. Every other
// value must name a level (UNIT-GATE-012); corrupt nonzero input never wraps.
func campaignDifficulty(value int64) (mapload.Difficulty, error) {
	if value == 0 {
		return mapload.DifficultyNormal, nil
	}
	if value < 1 || value > 3 {
		return 0, fmt.Errorf("invalid campaign difficulty %d (want 1..3)", value)
	}
	return mapload.Difficulty(value), nil
}

// NewGameOpener is shared by process-start chargen and the menu's NEW GAME.
// Merely obtaining the opener never commits or resets the current campaign.
func (f *FrontEnd) NewGameOpener(n int, res ui.ChargenResult) ui.MapOpener {
	return func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
		open, err := f.prepareNewGame(n, res)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		return open()
	}
}

// prepareNewGame builds against a fresh town and a draft difficulty. The old
// session survives every fallible step; openPrepared commits only on adoption.
func (f *FrontEnd) prepareNewGame(n int, res ui.ChargenResult) (ui.MapOpener, error) {
	return f.prepareNewGameWith(n, int64(res.Difficulty), f.ChargenParty(res))
}

// prepareNewGameWith is prepareNewGame over a party already built and a
// difficulty value as campaignDifficulty reads it.
func (f *FrontEnd) prepareNewGameWith(n int, level int64, party []mapload.PartyMember) (ui.MapOpener, error) {
	c, err := newCampaignCandidate(f.Campaign.Value(), f.Units, level)
	if err != nil {
		return nil, err
	}
	// A new game begins a fresh random session; its first mission is built
	// over it while the running game keeps its own until the commit.
	fresh := f.randomService().Fresh(clockSessionSeed())
	c.randomSession = &fresh
	f.randomService().Prepare(fresh)
	defer f.randomService().Cancel()
	open := f.missionOpenerMode(n, party, nil, nil, nil,
		&c.activate, c.units, c.difficulty, c.town)
	c.prepared.viewer, c.prepared.tick, c.prepared.order, c.prepared.cadence,
		c.prepared.affect, c.prepared.advance, c.prepared.attack, c.prepared.grab,
		c.prepared.stance, c.prepared.march, err = open()
	if err != nil {
		return nil, err
	}
	return openPrepared(c.prepared, func() { f.installCandidate(c) }), nil
}
