package game

import (
	"image"

	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// heroPane is the character pane p shows in both games: the shown member's
// figure or statistics card in p's rectangle, the card by the generator's
// presentation, the corners TOWN-354 gates (TOWN-353), the book corner when
// p has one. A nil p has no book.
func (t *townScreen) heroPane(p *town.PaneSpec) (ui.TownCharacterView, *ui.SlotMask, [12][]string) {
	var spec town.PaneSpec
	if p != nil {
		spec = *p
	}
	var panes TownCharacterPaneArt
	var corners *ui.CharacterPaneCornerArt
	if t.art != nil {
		panes, corners = t.art.characterPanes(), t.art.characterCorners()
	}
	var presentation *ui.ChargenPresentation
	if a := t.in.ChargenAssets; a != nil {
		presentation = a.Presentation
	}
	party := t.shopParty()
	t.clampTownMember()
	i := t.shopMemberIndex()
	var subject ui.PanelSubject
	if len(party) > 0 {
		subject = partyPanelSubject(party[i], t.in.Table, t.in.Words)
		subject.Unplaced, subject.Selected = true, 1
	}
	v := presentation.CardView(subject, spec.Rect.Rectangle(), image.Point{}, panes.Stats.Body, panes.Stats.Seam)
	v.HasSubject = len(party) > 0
	v.Statistics = t.townStats
	v.FigurePane, v.StatsPane, v.CornerArt, v.Session = panes.Figure, panes.Stats, corners, ui.CharacterPaneShop
	v.NoBook, v.BookOpen = !spec.Book, spec.Book && t.shopBook
	v.Member, v.MemberCount = townPickerPosition(party, i)
	if presentation == nil {
		v.Font, v.CardFont = t.in.Font.Value(), t.in.Font.Value()
	}
	if spec.Font == town.PaneFontInstall {
		v.Font = t.in.Font.Value()
	}
	var info [12][]string
	var mask *ui.SlotMask
	if t.shop.face != nil {
		var figure *image.RGBA
		figure, mask, info = t.shop.face(i)
		if len(party) > 0 {
			v.Figure = figure
		}
	}
	return v, mask, info
}
