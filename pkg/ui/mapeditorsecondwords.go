package ui

// secondOpWord is the name of a second-game operation a claim reads; each id
// is read by a literal, so the word tables' check sees every one.
func (e *MapEditor) secondOpWord(condition bool, k uint32) (string, bool) {
	if condition {
		switch k {
		case 5:
			return e.word("editor.second.check.5"), true
		case 18:
			return e.word("editor.second.check.18"), true
		case 19:
			return e.word("editor.second.check.19"), true
		case 23:
			return e.word("editor.second.check.23"), true
		case 24:
			return e.word("editor.second.check.24"), true
		case 25:
			return e.word("editor.second.check.25"), true
		case 26:
			return e.word("editor.second.check.26"), true
		case 27:
			return e.word("editor.second.check.27"), true
		case 65538:
			return e.word("editor.second.check.65538"), true
		}
		return "", false
	}
	switch k {
	case 2:
		return e.word("editor.second.action.2"), true
	case 3:
		return e.word("editor.second.action.3"), true
	case 4:
		return e.word("editor.second.action.4"), true
	case 5:
		return e.word("editor.second.action.5"), true
	case 8:
		return e.word("editor.second.action.8"), true
	case 10:
		return e.word("editor.second.action.10"), true
	case 35:
		return e.word("editor.second.action.35"), true
	case 36:
		return e.word("editor.second.action.36"), true
	case 38:
		return e.word("editor.second.action.38"), true
	case 39:
		return e.word("editor.second.action.39"), true
	case 65538:
		return e.word("editor.second.action.65538"), true
	}
	return "", false
}
