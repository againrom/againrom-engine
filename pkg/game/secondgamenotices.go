package game

import (
	"time"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type missionMessageSource uint8

const (
	messageLatches missionMessageSource = iota
	messageReport
)

type missionMessagePolicy struct {
	source    missionMessageSource
	retain    bool
	immediate bool
	specials  int
}

// The campaign service supplies the delivery row. Latches reduce the ordered
// report pipeline by discarding raises while a notice owns the screen.
var missionMessagePolicies = [...]missionMessagePolicy{
	{source: messageLatches, retain: false, immediate: false, specials: 0},
	{source: messageReport, retain: true, immediate: true, specials: 1},
}

type missionMessageEffect uint8

const (
	messageDialogue missionMessageEffect = iota
	messageLoss
	messageComplete
	messageFailed
	messageNewMission
)

var missionMessageSpecials = [...]struct {
	number  int32
	effects [2]missionMessageEffect
}{
	{250, [2]missionMessageEffect{messageDialogue, messageNewMission}},
	{sim.ObjectiveEventDone, [2]missionMessageEffect{messageDialogue, messageComplete}},
	{sim.ObjectiveEventMode4, [2]missionMessageEffect{messageDialogue, messageFailed}},
	{ReservedMessageNumber, [2]missionMessageEffect{messageLoss, messageLoss}},
}

func (m *missionNotices) messagePolicy() missionMessagePolicy {
	return m.campaign().messagePolicy()
}

func (mw *mapWorld) observeScriptMessages(messages []int32) {
	mw.acceptMissionMessages(messages, messageReport)
}

func (mw *mapWorld) acceptMissionMessages(messages []int32, source missionMessageSource) {
	m := mw.mission
	if m == nil {
		return
	}
	p := m.messagePolicy()
	if p.source != source || !p.retain && m.open {
		return
	}
	for _, event := range messages {
		if p.immediate && mw.applyMissionMessage(event) {
			continue
		}
		m.pendingMessages = append(m.pendingMessages, event)
	}
}

func (mw *mapWorld) applyMissionMessage(event int32) bool {
	effect := messageDialogue
	for _, row := range missionMessageSpecials {
		if event == row.number {
			effect = row.effects[mw.mission.messagePolicy().specials]
			break
		}
	}
	switch effect {
	case messageDialogue:
		return false
	case messageLoss:
		mw.recordMissionOutcome(sim.OutcomeLost, lossCause{})
	case messageNewMission:
		mw.view.PlayUISoundIdle(ui.UISoundMissionComplete)
		if line := mw.view.Words().NewMission; line != "" {
			mw.view.PostMessage(line, ui.MessageWhite, 30*time.Second)
		}
	case messageComplete:
		mw.view.PlayUISoundIdle(ui.UISoundMissionComplete)
	case messageFailed:
		mw.view.PlayUISoundIdle(ui.UISoundMissionFailed)
	}
	return true
}

func (mw *mapWorld) recordMissionOutcome(outcome sim.Outcome, cause lossCause) {
	mw.latchOutcome(outcome, cause)
}

type missionMessageStage uint8

const (
	messageStep missionMessageStage = iota
	messageAcknowledge
	messageDrain
)

// settleNotices samples every step, even while a notice is open. Outcomes
// latch before delivery and show after it. Acknowledgement drains retained
// messages without resampling latches or deciding another outcome.
func (mw *mapWorld) settleNotices(stage missionMessageStage) {
	m := mw.mission
	if m == nil {
		return
	}
	p := m.messagePolicy()
	if stage == messageStep {
		if !m.announced || m.outcome == sim.OutcomeWon {
			// THE CHARACTERS ARE TESTED FIRST, AHEAD OF Outcome() ITSELF. The engine's
			// own reporter tests a loss before a win (see Outcome() below), and this
			// test placed after it would let a winning trigger latch on the very step
			// a character died — so this is not a rule beside the engine's two, it
			// is that same three-test order carried one test further up, ahead of both
			// counters.
			//
			// A FALLEN CHARACTER WHO CAN STILL BE HEALED IS NOT LOST. The guarded
			// set is the party, and a member's fall defers the loss while his body
			// dwells or Heal can still raise it. A blow or the corpse walk to -10
			// ends that window; a body at exactly 0 never decays (HERO-ZERO-070).
			// The original loses on the primary's fall instead: DIV-219.
			//
			// ONE DEAD CHARACTER IS ENOUGH and the walk stops at it. His fall is
			// the loss's cause, and missionLossReason turns it into the reason.
			o := sim.OutcomeUndecided
			var cause lossCause
			if id, lost := mw.guardedCharacterLost(); lost {
				o, cause = sim.OutcomeLost, lossCause{fallen: id, fell: true}
			}
			// THE OUTCOME COMES FROM Outcome() AND NEVER FROM THE COUNTERS, when the
			// character test above left the question open. The counters only rise,
			// nothing clears them inside a mission, and one repeating check drives the
			// lose counter up without bound — so `> 0` is the wrong predicate and
			// nothing above pkg/sim re-derives an outcome.
			if o == sim.OutcomeUndecided {
				o = mw.world.Outcome()
			}
			if o != sim.OutcomeUndecided {
				if m.outcome != o && m.outcomeShown {
					mw.closeNotice()
					m.outcomeShown, m.delayedVictory = false, false
				}
				mw.recordMissionOutcome(o, cause)
			}
		}

		raised := m.ann.Sample(mw.world)
		mw.acceptMissionMessages(raised, messageLatches)
	} else if stage == messageAcknowledge && p.retain {
		mw.closeNotice()
	} else if stage == messageAcknowledge {
		if m.announced && !m.outcomeShown {
			mw.showOutcome()
		} else {
			mw.closeNotice()
		}
		return
	}
	if !p.retain {
		defer func() {
			clear(m.pendingMessages)
			m.pendingMessages = nil
		}()
	}
	if m.open {
		return
	}
	// DIV-2379: retained messages keep emission order; native replay order
	// while busy remains Unknown.
	for len(m.pendingMessages) > 0 {
		event := m.pendingMessages[0]
		m.pendingMessages[0] = 0
		m.pendingMessages = m.pendingMessages[1:]
		if !p.immediate && mw.applyMissionMessage(event) {
			continue
		}
		if mw.openDialogue(int(event)) {
			return
		}
	}
	if m.announced && !m.outcomeShown {
		mw.showOutcome()
	}
}
