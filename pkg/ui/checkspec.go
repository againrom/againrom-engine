package ui

// CheckStyle is how a check box shows its state; CheckArt has no text mark.
type CheckStyle uint8

const (
	CheckArt CheckStyle = iota
	CheckText
	CheckRadioText
)

// CheckSpec is one check box; Mark is where a text mark is spelled.
type CheckSpec struct {
	On    bool
	Style CheckStyle
}

var checkMarks = map[CheckStyle][2]string{
	CheckText:      {"[ ] ", "[x] "},
	CheckRadioText: {"( ) ", "(*) "},
}

func (c CheckSpec) Mark() string {
	if c.On {
		return checkMarks[c.Style][1]
	}
	return checkMarks[c.Style][0]
}
