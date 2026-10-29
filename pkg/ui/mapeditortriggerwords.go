package ui

import "strings"

// The trigger editor's words with a number or role in their id: each id
// is read by a literal, so the word tables' check sees every one.

// checkWord is the name of a condition operation a claim names.
func (e *MapEditor) checkWord(k uint32) (string, bool) {
	switch k {
	case 1:
		return e.word("editor.check.op1"), true
	case 2:
		return e.word("editor.check.op2"), true
	case 3:
		return e.word("editor.check.op3"), true
	case 4:
		return e.word("editor.check.op4"), true
	case 5:
		return e.word("editor.check.op5"), true
	case 6:
		return e.word("editor.check.op6"), true
	case 7:
		return e.word("editor.check.op7"), true
	case 8:
		return e.word("editor.check.op8"), true
	case 9:
		return e.word("editor.check.op9"), true
	case 10:
		return e.word("editor.check.op10"), true
	case 11:
		return e.word("editor.check.op11"), true
	case 12:
		return e.word("editor.check.op12"), true
	case 13:
		return e.word("editor.check.op13"), true
	case 14:
		return e.word("editor.check.op14"), true
	case 15:
		return e.word("editor.check.op15"), true
	case 16:
		return e.word("editor.check.op16"), true
	case 17:
		return e.word("editor.check.op17"), true
	case 18:
		return e.word("editor.check.op18"), true
	case 19:
		return e.word("editor.check.op19"), true
	case 21:
		return e.word("editor.check.op21"), true
	case 22:
		return e.word("editor.check.op22"), true
	case 65538:
		return e.word("editor.check.op65538"), true
	case 0x4101:
		return e.word("editor.check.op16641"), true
	}
	return "", false
}

// actionWord is the name of an action operation a claim names.
func (e *MapEditor) actionWord(k uint32) (string, bool) {
	switch k {
	case 2:
		return e.word("editor.action.op2"), true
	case 3:
		return e.word("editor.action.op3"), true
	case 4:
		return e.word("editor.action.op4"), true
	case 5:
		return e.word("editor.action.op5"), true
	case 6:
		return e.word("editor.action.op6"), true
	case 8:
		return e.word("editor.action.op8"), true
	case 9:
		return e.word("editor.action.op9"), true
	case 10:
		return e.word("editor.action.op10"), true
	case 11:
		return e.word("editor.action.op11"), true
	case 12:
		return e.word("editor.action.op12"), true
	case 13:
		return e.word("editor.action.op13"), true
	case 16:
		return e.word("editor.action.op16"), true
	case 17:
		return e.word("editor.action.op17"), true
	case 18:
		return e.word("editor.action.op18"), true
	case 19:
		return e.word("editor.action.op19"), true
	case 20:
		return e.word("editor.action.op20"), true
	case 21:
		return e.word("editor.action.op21"), true
	case 22:
		return e.word("editor.action.op22"), true
	case 23:
		return e.word("editor.action.op23"), true
	case 24:
		return e.word("editor.action.op24"), true
	case 25:
		return e.word("editor.action.op25"), true
	case 26:
		return e.word("editor.action.op26"), true
	case 27:
		return e.word("editor.action.op27"), true
	case 28:
		return e.word("editor.action.op28"), true
	case 29:
		return e.word("editor.action.op29"), true
	case 30:
		return e.word("editor.action.op30"), true
	case 32:
		return e.word("editor.action.op32"), true
	case 33:
		return e.word("editor.action.op33"), true
	case 34:
		return e.word("editor.action.op34"), true
	case 65538:
		return e.word("editor.action.op65538"), true
	case 0x4100:
		return e.word("editor.action.op16640"), true
	}
	return "", false
}

// cmpWordOf is a comparison code in words.
func (e *MapEditor) cmpWordOf(k int64) (string, bool) {
	switch k {
	case 0:
		return e.word("editor.cmp.0"), true
	case 1:
		return e.word("editor.cmp.1"), true
	case 2:
		return e.word("editor.cmp.2"), true
	case 3:
		return e.word("editor.cmp.3"), true
	case 4:
		return e.word("editor.cmp.4"), true
	case 5:
		return e.word("editor.cmp.5"), true
	}
	return "", false
}

var commandPanelSlots = map[uint32]int{10: 0, 4: 1, 1: 2, 11: 3, 5: 5, 3: 6}

func commandPanelLabel(label string) string {
	label = strings.TrimSpace(label)
	if i := strings.LastIndex(label, "<"); i >= 0 && len(label)-i == 3 && strings.HasSuffix(label, ">") {
		label = strings.TrimSpace(label[:i])
	}
	return label
}

// commandWord is the name of a group command.
func (e *MapEditor) commandWord(k uint32) (string, bool) {
	if slot, ok := commandPanelSlots[k]; ok {
		if w := commandPanelLabel(e.commandWords[slot]); w != "" {
			return w, true
		}
	}
	switch k {
	case 1:
		return e.word("editor.command.1"), true
	case 2:
		return e.word("editor.command.2"), true
	case 3:
		return e.word("editor.command.3"), true
	case 4:
		return e.word("editor.command.4"), true
	case 5:
		return e.word("editor.command.5"), true
	case 10:
		return e.word("editor.command.10"), true
	case 11:
		return e.word("editor.command.11"), true
	case 14:
		return e.word("editor.command.14"), true
	case 15:
		return e.word("editor.command.15"), true
	case 17:
		return e.word("editor.command.17"), true
	case 18:
		return e.word("editor.command.18"), true
	}
	return "", false
}

// roleWord is the word of a signature role.
func (e *MapEditor) roleWord(role string) (string, bool) {
	switch role {
	case "unit2":
		return e.word("editor.role.unit2"), true
	case "player2":
		return e.word("editor.role.player2"), true
	case "cell":
		return e.word("editor.role.cell"), true
	case "cell2":
		return e.word("editor.role.cell2"), true
	case "corner":
		return e.word("editor.role.corner"), true
	case "corner2":
		return e.word("editor.role.corner2"), true
	case "radius":
		return e.word("editor.role.radius"), true
	case "selector":
		return e.word("editor.role.selector"), true
	case "value":
		return e.word("editor.role.value"), true
	case "variable":
		return e.word("editor.role.variable"), true
	case "message":
		return e.word("editor.role.message"), true
	case "money":
		return e.word("editor.role.money"), true
	case "spell":
		return e.word("editor.role.spell"), true
	case "power":
		return e.word("editor.role.power"), true
	case "time":
		return e.word("editor.role.time"), true
	case "range":
		return e.word("editor.role.range"), true
	case "command":
		return e.word("editor.role.command"), true
	case "health":
		return e.word("editor.role.health"), true
	case "undying":
		return e.word("editor.role.undying"), true
	}
	return "", false
}
