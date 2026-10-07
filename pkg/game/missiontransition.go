package game

import (
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// missionTransitions is what a mission's advance seam asks of the campaign.
// continueMission names no component; the production one is frontTransitions.
type missionTransitions interface {
	// leave answers an abandon or restart request for mission n.
	leave(n int, ms *Mission, action ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener)
	// finishWon completes won mission n: it carries the party forward and
	// answers the successor and the line that names what follows.
	finishWon(n int, ms *Mission) (successor int, line string)
	// route answers where winning mission n leads, given finishWon's successor.
	route(n, successor int) winRoute
	// opener builds the opener of mission n over the party the campaign holds.
	opener(n int) ui.MapOpener
	// returnToTown rebuilds the reused gates screen after mission n and
	// travels home.
	returnToTown(n int)
}

// continueMission wraps the driver's own advance seam: dismissing a won
// mission's banner carries the party forward and opens the successor or the
// town the campaign declares, and otherwise names what follows.
//
// It wraps rather than edits. The driver decides what a notice does (it pages
// a dialogue, sends a loss to the menu and a win to the map list) and this
// reads the answer it already gave and acts only on a win.
//
// The win is recognised from the world's outcome and not from the destination:
// the outcome is the fact, and a later destination added beside the map list
// must not silently start carrying a party.
//
// It fires once. The banner closes on the call that returns the destination,
// so a second dismissal finds nothing open and the driver answers NoticeStay.
//
// The opener is built one statement after finishWon returns, because finishWon
// writes the carried party first. It is the default-party opener and not one
// handed the carry directly, so a mission with no party opens the successor
// with the default hero every other door mints: one door, one answer.
//
// A declared successor the tree must refuse never reaches the route: that
// refusal and its sentence are composed inside finishWon, and a positive
// return is the only one that opens.
func continueMission(t missionTransitions, n int, ms *Mission, advance ui.MapAdvance) ui.MapAdvance {
	return func(actions ...ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) {
		if len(actions) > 0 && (actions[0] == ui.NoticeAbandon || actions[0] == ui.NoticeRestart) {
			return t.leave(n, ms, actions[0])
		}
		action := ui.NoticeAdvance
		if len(actions) > 0 {
			action = actions[0]
		}
		if gate, ok := t.(interface {
			beforeAdvance(int, *Mission, ui.NoticeAction) string
		}); ok {
			if refusal := gate.beforeAdvance(n, ms, action); refusal != "" {
				return ui.NoticeStay, refusal, nil
			}
		}
		dest, msg, open := advance(actions...)
		if ms.World == nil {
			return dest, msg, open
		}
		switch dest {
		case ui.NoticeToMapList:
			if ms.World.Outcome() != sim.OutcomeWon {
				return dest, msg, open
			}
			successor, line := t.finishWon(n, ms)
			switch t.route(n, successor) {
			case winStay:
				return ui.NoticeStay, line, nil
			case winEnding:
				return ui.NoticeToEnding, "", nil
			case winMission:
				return ui.NoticeToMission, line, t.opener(successor)
			case winTown:
				// Rebuild the reused gates screen after completion, then travel
				// home. This is not another economic arrival (DIV-137).
				t.returnToTown(n)
				return ui.NoticeToTown, line, nil
			}
			return dest, line, open
		}
		// Failure exits to menu or opens Load. It never returns to town with
		// the pre-mission party.
		return dest, msg, open
	}
}

// advanceFrom is the mission-entry advance port over a transition rule.
func advanceFrom(t missionTransitions) func(n int, ms *Mission, advance ui.MapAdvance) ui.MapAdvance {
	return func(n int, ms *Mission, advance ui.MapAdvance) ui.MapAdvance {
		return continueMission(t, n, ms, advance)
	}
}

// frontTransitions is the production missionTransitions over a front end. It
// is the one place the transition rule meets the front end.
type frontTransitions struct{ f *FrontEnd }

func (t frontTransitions) beforeAdvance(n int, ms *Mission, action ui.NoticeAction) string {
	town, live := t.f.Town, t.f.live
	if town == nil || town.second == nil || live == nil || live.mission == nil {
		return ""
	}
	m := live.mission
	completion := m.kind == ui.NoticeSuccess &&
		(action == ui.NoticeAdvance && m.open || action == ui.NoticeVictory && (m.open || m.delayedVictory && !m.victoryTaken))
	if completion {
		if err := town.second.finish(n, ms.World); err != nil {
			return err.Error()
		}
	}
	return ""
}

func (t frontTransitions) leave(n int, ms *Mission, action ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) {
	return t.f.leaveMission(n, ms, action)
}

func (t frontTransitions) finishWon(n int, ms *Mission) (int, string) {
	if town := t.f.Town; town != nil && town.second != nil {
		if err := town.second.finish(n, ms.World); err != nil {
			return -1, err.Error()
		}
		if refused := t.f.CampaignSession.carryMissionHome(t.f.townInstall(), n, ms.Party, ms.World, ms.Start.IDs, ms.Start.Roster); refused != "" {
			return -1, refused
		}
		town.second.complete(ms.World)
		town.won[n] = true
		t.f.Offered = 0
		for _, l := range town.second.available {
			if l.kind == 1 {
				t.f.Offered = l.id
				break
			}
		}
		return 0, ""
	}
	return t.f.FinishMissionWithRoster(n, ms.Party, ms.World, ms.Start.IDs, ms.Start.Roster)
}

func (t frontTransitions) route(n, successor int) winRoute {
	if t.f.Town != nil && t.f.Town.second != nil {
		if successor < 0 {
			return winStay
		}
		return winTown
	}
	return t.f.CampaignSession.routeAfterWin(t.f.Campaign.Value(), n, successor)
}

func (t frontTransitions) opener(n int) ui.MapOpener { return t.f.MissionOpener(n) }

func (t frontTransitions) returnToTown(n int) {
	if t.f.Town != nil && t.f.Town.second != nil {
		releaseWorldAudio(t.f.runtimeAudio(), t.f.endLive())
		return
	}
	t.f.TownScreen().(*townScreen).beginWorldMapReturn(n)
}
