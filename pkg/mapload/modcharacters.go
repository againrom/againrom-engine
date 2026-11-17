package mapload

// ModCharacter is a character a mod edited, in the form the panels read: the
// definition rows the edit applies to and the display name it gives them. A
// character whose edit gives no name is not listed.
type ModCharacter struct {
	Mod  string
	Rows []string
	Name string
}

// CharacterName is the display name a mod gives the person built from the
// definition row of that name, and false when no mod names it.
func (c ModContext) CharacterName(row string) (string, bool) {
	name, found := "", false
	for _, ch := range c.Characters {
		for _, r := range ch.Rows {
			if r == row {
				name, found = ch.Name, true
			}
		}
	}
	return name, found
}

// ShownName is the one answer to the name a panel shows for a person: its own
// name, else the name a mod gives the person's definition row, else the class
// name. A row of "" is a person with no definition row.
func (c ModContext) ShownName(own, row, class string) string {
	if own != "" {
		return own
	}
	if row != "" {
		if name, ok := c.CharacterName(row); ok {
			return name
		}
	}
	return class
}
