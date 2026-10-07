package game

import (
	"strings"
	"time"

	"againrom/pkg/base"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func (m *missionNotices) secondGame() bool {
	return m != nil && m.table != nil && m.table.Game == base.GameROM2
}

func (mw *mapWorld) observeScriptMessages(messages []int32) {
	m := mw.mission
	if !m.secondGame() {
		return
	}
	for _, event := range messages {
		switch event {
		case 255:
			m.announced, m.outcome = true, sim.OutcomeLost
		case 250:
			if words := LoadInstallWords(m.src); words != nil {
				if caption, ok := words.Global(0); ok {
					mw.view.PostMessage(caption, ui.MessageWhite, 5*time.Second)
				}
			}
		case 253, 254:
			// The native special branch is separate from ordinary dialogue.
			// DIV-2379 keeps presentation in the existing map message surface.
			caption := strings.ReplaceAll(mw.view.Words().MenuQuestObjectives, "~", "")
			if caption == "" {
				caption = "Quest objectives"
			}
			mw.view.PostMessage(caption, ui.MessageWhite, 5*time.Second)
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
