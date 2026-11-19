package game

import (
	"againrom/pkg/locale"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// maxDialogueReplies is the most replies one page offers, and the number of
// reply actions the notice seam carries.
const maxDialogueReplies = ui.MaxReplies

// replyActions are the notice actions that choose reply 1..4.
var replyActions = [maxDialogueReplies]ui.NoticeAction{ui.NoticeReply1, ui.NoticeReply2, ui.NoticeReply3, ui.NoticeReply4}

// dialogueFilePath is the address a mission's dialogue file is served at: the
// mission's text folder, then dialogs/<name>.toml.
func dialogueFilePath(mission int, name string) string {
	return missionPageDirSource.at(mission).path + "dialogs/" + name + ".toml"
}

// dialogueReply is one reply a page shows: the words in the install's
// alphabet, the node it pages to (0 closes the dialogue), the compiled
// instants its outcome runs, the script variable recording a once reply
// (-1 for none) and whether it is shown greyed and not choosable.
type dialogueReply struct {
	text     string
	next     int
	instants []int32
	mark     int32
	disabled bool
	node     int
	index    int
}

// interactiveDialogue is the open dialogue's own state: the parsed file, the
// node on show and the once replies chosen since it opened, whose variable
// the world writes only when it next steps.
type interactiveDialogue struct {
	file mod.Dialogue
	node int
	done map[[2]int]bool
}

// openInteractive opens dialogue idx of the mission's compiled list on its
// first node, and reports whether there was anything to show. A file that is
// not served or does not parse shows nothing, as an event text that does not
// ship does.
func (mw *mapWorld) openInteractive(idx int) bool {
	m := mw.mission
	if m.state == nil || idx < 0 || idx >= len(m.state.Dialogues) || !mod.ValidDialogueName(m.state.Dialogues[idx]) {
		return false
	}
	path := dialogueFilePath(m.number, m.state.Dialogues[idx])
	data, err := m.src.ReadFile(path)
	if err != nil {
		return false
	}
	file, err := mod.ParseDialogue(path, data)
	if err != nil {
		return false
	}
	m.open, m.kind, m.payload, m.part = true, ui.NoticeDialogue, nil, 0
	m.dialogue = &interactiveDialogue{file: file, node: file.Nodes[0].ID, done: map[[2]int]bool{}}
	m.dialogueAudience = EventAudience{}
	m.tips, m.portrait = 0, false
	mw.view.SetDialogue(mw.nodePage())
	return true
}

// nodePage is the push for the dialogue's current node: its words and the
// replies it offers. A once reply already chosen is not offered; a reply whose
// condition fails is left out or, when its file says so, shown greyed. Two or
// more shown replies take the button's place; one enabled reply is the button's
// own; none enabled leaves a button that closes the dialogue.
func (mw *mapWorld) nodePage() ui.Dialogue {
	m := mw.mission
	d := m.dialogue
	node, _ := d.file.Node(d.node)
	selector := LanguageSelector(m.src)
	lang := ""
	if l, ok := locale.BySelector(selector); ok {
		lang = l.Code
	}
	shown := ui.Dialogue{Text: EncodeModText(node.Text.Words(lang), selector), Portrait: m.portrait}
	m.replies = nil
	var rows []dialogueReply
	enabled := 0
	for i, r := range node.Replies {
		if r.Once && (d.done[[2]int{node.ID, i}] || mw.world.ScriptRegister(int32(r.Mark)) != 0) {
			continue
		}
		holds := mw.replyHolds(r)
		if !holds && !r.Disabled {
			continue
		}
		row := dialogueReply{text: EncodeModText(r.Text.Words(lang), selector), next: r.Next, mark: -1, disabled: !holds, node: node.ID, index: i}
		if r.Once {
			row.mark = int32(r.Mark)
		}
		for _, id := range r.Do {
			if k, ok := m.state.ActionOf[uint32(id)]; ok {
				row.instants = append(row.instants, k)
			}
		}
		if holds {
			enabled++
		}
		rows = append(rows, row)
	}
	switch {
	case enabled == 0:
	case len(rows) == 1:
		m.replies = rows
	default:
		m.replies = rows
		for _, r := range rows {
			shown.Replies = append(shown.Replies, r.text)
			shown.RepliesOff = append(shown.RepliesOff, r.disabled)
		}
	}
	return shown
}

// replyHolds reports that every condition pair of r holds, each asked of the
// world's one pair evaluator through the check subscripts the script compiled.
// A pair naming a condition node the script does not have does not hold.
func (mw *mapWorld) replyHolds(r mod.DialogueReply) bool {
	for _, c := range r.When {
		left, lok := mw.mission.state.CheckOf[uint32(c.Left)]
		right, rok := mw.mission.state.CheckOf[uint32(c.Right)]
		if !lok || !rok || !mw.world.ConditionHolds(sim.ScriptPair{Left: left, Right: right, Cmp: int32(c.Cmp), Used: true}) {
			return false
		}
	}
	return true
}

// chooseReply is the player's pick of reply k (from 1) of the open page. A
// disabled reply is not choosable. The outcome is queued as commands, one
// run-instant for each action and one set-variable for a once reply's mark,
// then the dialogue pages to the reply's node or closes.
func (mw *mapWorld) chooseReply(k int) {
	m := mw.mission
	if k < 1 || k > len(m.replies) || m.dialogue == nil || m.replies[k-1].disabled {
		return
	}
	r := m.replies[k-1]
	for _, in := range r.instants {
		mw.pending = append(mw.pending, sim.RunInstant(in))
	}
	if r.mark >= 0 {
		mw.pending = append(mw.pending, sim.SetVariable(r.mark, 1))
		m.dialogue.done[[2]int{r.node, r.index}] = true
	}
	if _, ok := m.dialogue.file.Node(r.next); ok && r.next > 0 {
		m.dialogue.node = r.next
		mw.view.PageDialogue(mw.nodePage())
		return
	}
	mw.raiseMissionTip()
	mw.settleNotices(messageAcknowledge)
}
