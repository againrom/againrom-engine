package game

import (
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// canLeaveMission reports whether mission n, running in world w, was entered
// from the town and is still undecided: the one state in which a mod's menu
// entry may abandon or restart it. A mission that has been won or lost is
// finished through its own panel, and a map opened from the list has no town to
// return to.
func (s *CampaignSession) canLeaveMission(n int, w *sim.World) bool {
	return s != nil && s.Town.Open() && n > 0 && w != nil && w.Outcome() == sim.OutcomeUndecided
}

// leaveMission answers a mod menu entry's request to leave mission ms without
// completing it.
//
// A mission the front end opened from the town leaves the town as it was
// entered: the party, the purse, the items, the town's hired pools and the
// town's own records are the ones the front end held at entry, because a mission
// works on copies of them and only a completion writes them back. The mission
// stays offered, since it was never completed.
//
// A mission restored from a SAV has no such town: the SAV carries the mission's
// party and the front end holds none. That party, as it stands now with what the
// mission gave it, goes home through the carry a won mission uses (survivors,
// city objects, hired pools, purse), with no win, payment or fame recorded.
//
// Abandon travels the party back to the town through the same home route a won
// mission uses, without the economic arrival a completion pays. Restart answers
// with an opener for the same mission, built from the party the town holds. The
// quick-spell bindings go back to those the mission was opened with.
func (f *FrontEnd) leaveMission(n int, ms *Mission, action ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) {
	if !f.canLeaveMission(n, ms.World) {
		return ui.NoticeStay, "", nil
	}
	screen, ok := f.TownScreen().(*townScreen)
	if !ok {
		return ui.NoticeStay, "", nil
	}
	if refused := f.CampaignSession.leaveLive(f.townInstall(), n, ms); refused != "" {
		return ui.NoticeStay, refused, nil
	}
	if action == ui.NoticeRestart {
		return ui.NoticeToMission, "", f.MissionOpener(n)
	}
	screen.beginWorldMapReturn(n)
	return ui.NoticeToTown, "", nil
}
