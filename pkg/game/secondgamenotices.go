package game

import (
	"time"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// observeScriptMessages hands the messages the script raised this step to the
// campaign service.
func (mw *mapWorld) observeScriptMessages(messages []int32) {
	mw.mission.campaign().scriptMessages(mw, messages)
}

// observeSecondGameMessages takes the special message numbers the second
// game's script raises and queues every other one as a dialogue. The
// objective events 253 and 254 and event 250 play an interface entry when it
// is idle, and 250 also posts the new-mission line for 30 s; none writes
// state or opens a window (R2-ENGINE-891).
func (mw *mapWorld) observeSecondGameMessages(messages []int32) {
	m := mw.mission
	for _, event := range messages {
		switch event {
		case 255:
			mw.latchOutcome(sim.OutcomeLost, lossCause{})
		case 250:
			mw.view.PlayUISoundIdle(ui.UISoundMissionComplete)
			if line := mw.view.Words().NewMission; line != "" {
				mw.view.PostMessage(line, ui.MessageWhite, 30*time.Second)
			}
		case sim.ObjectiveEventDone:
			mw.view.PlayUISoundIdle(ui.UISoundMissionComplete)
		case sim.ObjectiveEventMode4:
			mw.view.PlayUISoundIdle(ui.UISoundMissionFailed)
		default:
			m.pendingMessages = append(m.pendingMessages, event)
		}
	}
}

// DIV-2379: deferred ordinary messages keep emission order. Native UI433
// enqueues IDs while busy; its complete replay order remains Unknown.
func (mw *mapWorld) settleSecondGameNotices() {
	m := mw.mission
	if m.open {
		return
	}
	for len(m.pendingMessages) > 0 {
		event := m.pendingMessages[0]
		m.pendingMessages[0] = 0
		m.pendingMessages = m.pendingMessages[1:]
		if mw.openDialogue(int(event)) {
			return
		}
	}
	if m.announced && !m.outcomeShown {
		mw.showOutcome()
	}
}
