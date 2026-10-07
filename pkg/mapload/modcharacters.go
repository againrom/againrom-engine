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
