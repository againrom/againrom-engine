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

// continueMission carries an acknowledged win through ordinary campaign
// completion. The simulation outcome or the current client's win admits it;
// a destination alone cannot complete a mission. Party carry precedes opening
// a successor, and the driver's one-shot acknowledgement prevents a replay.
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
				client, ok := t.(interface{ clientWon(int, *Mission) bool })
				if !ok || !client.clientWon(n, ms) {
					return dest, msg, open
				}
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

func (t frontTransitions) clientWon(n int, ms *Mission) bool {
	live := t.f.live
	return live != nil && live.world == ms.World && live.mission != nil &&
		live.mission.number == n && live.mission.outcome == sim.OutcomeWon
}

// won is the one win predicate of a mission's completion: the world's script
// outcome, or the client outcome clientWon accepts.
func (t frontTransitions) won(n int, ms *Mission) bool {
	return ms.World != nil && ms.World.Outcome() == sim.OutcomeWon || t.clientWon(n, ms)
}

func (t frontTransitions) beforeAdvance(n int, ms *Mission, action ui.NoticeAction) string {
	return t.f.campaign().beforeAdvance(t.f, n, ms, action)
}

func (t frontTransitions) leave(n int, ms *Mission, action ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) {
	return t.f.leaveMission(n, ms, action)
}

func (t frontTransitions) finishWon(n int, ms *Mission) (int, string) {
	return t.f.campaign().finishWon(t.f, n, ms)
}

func (t frontTransitions) route(n, successor int) winRoute {
	return t.f.campaign().route(t.f, n, successor)
}

func (t frontTransitions) opener(n int) ui.MapOpener { return t.f.MissionOpener(n) }

func (t frontTransitions) returnToTown(n int) { t.f.campaign().returnToTown(t.f, n) }
