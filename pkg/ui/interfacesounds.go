package ui

// The generator description names the chrgen members of sfx.res that the main
// menu, the Hall of Fame and the school room also request (VIDEO-SFX-059,
// VIDEO-SFX-060). Those screens read them from the description the profile's
// edition names, so one place names each sound.

// OKSound is the pre-create page's continue member, ok.wav, which a press on a
// main menu button and on the Hall of Fame OK also requests. A nil
// description names none.
func (d *GeneratorDescription) OKSound() string {
	if d == nil {
		return ""
	}
	return d.PreCreate.Forward.Sound
}

// SkillSound is the member the detailed page's skill slot requests for a
// class, fighter 0 and mage 1; the school room requests the same member for
// its cell in that class's column. Outside the description it is empty.
func (d *GeneratorDescription) SkillSound(class, slot int) string {
	if d == nil || class < 0 || class >= len(d.Detail.Classes) {
		return ""
	}
	skills := d.Detail.Classes[class].Skills
	if slot < 0 || slot >= len(skills) {
		return ""
	}
	return skills[slot].Sound
}

// SetInterfaceGenerator names the generator description whose sounds the
// main menu and the Hall of Fame request.
func (a *App) SetInterfaceGenerator(d *GeneratorDescription) { a.interfaceGenerator = d }
