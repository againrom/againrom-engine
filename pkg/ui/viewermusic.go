package ui

// SetMusicAreas installs the mission's music areas, in the order the area
// select walks them, and its hero. The music controller reads them while
// this viewer is the shown map; a nil hero reads no area.
func (v *Viewer) SetMusicAreas(areas []MusicArea, hero MusicHero) {
	if v == nil {
		return
	}
	v.musicAreas = musicAreaSource{areas: append([]MusicArea(nil), areas...), hero: hero}
}
