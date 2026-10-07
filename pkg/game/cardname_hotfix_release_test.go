package game

import (
	"image"
	"testing"

	"againrom/pkg/ui"
)

// Tester items 69 and 93: an installed class name wider than the card's top
// row lost its last letters ("Воин с мечо"). Every installed class name now
// reaches the card whole, on one row or wrapped at a space onto two, with and
// without the role line above it (the tavern candidate card has one).
func TestReleaseEveryInstalledClassNameReachesTheCardWhole(t *testing.T) {
	f := releaseFront(t)
	font := f.tipFont()
	bg, _ := f.characterPanes().Stats.Body.(*image.RGBA)
	if font == nil || bg == nil {
		t.Fatal("the install gave the card no font or no body")
	}
	layout := ui.CompactPanelLayout(bg)
	wrapped, checked := 0, 0
	for index, name := range f.Words.UnitNames {
		if name == "" {
			continue
		}
		for _, role := range []string{"", f.Words.PanelCaptions[mainSlotMercenary]} {
			s := ui.PanelSubject{Name: name, UnitNameIndex: index, Role: role, HP: 10, MaxHP: 10, Words: f.Words,
				DetailLevel: 7, DetailSet: true, Char: ui.UnitCharacter{Known: true, Name: name, UnitNameIndex: index}}
			var got *ui.PanelLineReport
			rows := ui.CharacterPanelReport(layout, font, s)
			for i := range rows {
				if rows[i].FullValue == name {
					got = &rows[i]
					break
				}
			}
			if got == nil {
				t.Fatalf("class %d %q role %q: no name row", index, name, role)
			}
			shown := got.Value
			if got.Wrap != "" {
				shown += " " + got.Wrap
				wrapped++
			}
			if shown != name {
				t.Errorf("class %d %q role %q reaches the card as %q", index, name, role, shown)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no installed class name was checked")
	}
	t.Logf("%d card names checked, %d wrapped", checked, wrapped)
}
