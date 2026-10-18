package ui

import "strconv"

func (e *MapEditor) stepChoice(delta int) bool {
	if !e.prompt.fields() {
		return false
	}
	fields := e.promptFields()
	if e.prompt.field >= len(fields) {
		return false
	}
	choices := fields[e.prompt.field].Choices
	if len(choices) == 0 {
		return false
	}
	value, err := strconv.ParseInt(e.prompt.path, 10, 64)
	at := -1
	if err == nil {
		for n, c := range choices {
			if c.Value == value {
				at = n
				break
			}
		}
	}
	if at < 0 {
		at = 0
	} else {
		at = min(max(at+delta, 0), len(choices)-1)
	}
	e.prompt.path = strconv.FormatInt(choices[at].Value, 10)
	return true
}

func choiceTitle(f InspectionField, input string) string {
	value, err := strconv.ParseInt(input, 10, 64)
	if err == nil {
		for _, c := range f.Choices {
			if c.Value == value {
				return c.Label + " / Up, Down"
			}
		}
	}
	return f.Label
}
