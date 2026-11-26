package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
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

// writeDifficulty writes d to the head and, where the campaign keeps it, d
// minus one to the campaign record word (SAV-1270).
func writeDifficulty(doc *sav.DocumentData, rules campaignService, d mapload.Difficulty) {
	doc.Head.Difficulty = uint32(d)
	if word, ok := rules.difficultyRecord(d); ok {
		doc.Campaign.Scalars[1] = word
	}
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
	return f.prepareFreshGame(missionRequest{n: n, party: party}, level)
}

// prepareFreshGame opens r as the first map of a new game: a fresh town, a
// fresh random session and a draft difficulty, committed only on adoption. A
// campaign request enters over the fresh town; a session's request enters no
// town (missionRequest.session).
func (f *FrontEnd) prepareFreshGame(r missionRequest, level int64) (ui.MapOpener, error) {
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
	if r.session == nil {
		if m := newMissionIdentity(r.n, nil, tableMissions(f.Table)).declared(); m != nil {
			if _, refused := f.startGuestMission(r.n, nil, m, r.party, c); refused != "" {
				return nil, fmt.Errorf("%s", refused)
			}
		}
		r.town = c.town
	}
	r.activate, r.units, r.difficulty = &c.activate, c.units, c.difficulty
	open := f.requestOpener(r)
	c.prepared.viewer, c.prepared.tick, c.prepared.order, c.prepared.cadence,
		c.prepared.affect, c.prepared.advance, c.prepared.attack, c.prepared.grab,
		c.prepared.stance, c.prepared.march, err = open()
	if err != nil {
		return nil, err
	}
	// The mission start's reseeds counted in the prepared session.
	fresh = f.randomService().Prepared()
	return openPrepared(c.prepared, func() { f.installCandidate(c) }), nil
}
