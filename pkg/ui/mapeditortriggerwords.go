package ui

import "strings"

// The trigger editor's words with a number or role in their id: each id
// is read by a literal, so the word tables' check sees every one.

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

// signatureRoleWords maps a signature role to its word's locale key: the one
// home of the role words.
var signatureRoleWords = map[string]string{
	"unit2":    "editor.role.unit2",
	"player2":  "editor.role.player2",
	"cell":     "editor.role.cell",
	"cell2":    "editor.role.cell2",
	"corner":   "editor.role.corner",
	"corner2":  "editor.role.corner2",
	"radius":   "editor.role.radius",
	"selector": "editor.role.selector",
	"value":    "editor.role.value",
	"variable": "editor.role.variable",
	"message":  "editor.role.message",
	"money":    "editor.role.money",
	"spell":    "editor.role.spell",
	"power":    "editor.role.power",
	"time":     "editor.role.time",
	"range":    "editor.role.range",
	"command":  "editor.role.command",
	"health":   "editor.role.health",
	"undying":  "editor.role.undying",
	"global":   "editor.role.global",
	"delta":    "editor.role.delta",
	"reason":   "editor.role.reason",
}

// roleWord is the word of a signature role.
func (e *MapEditor) roleWord(role string) (string, bool) {
	id, ok := signatureRoleWords[role]
	if !ok {
		return "", false
	}
	return e.word(id), true
}
