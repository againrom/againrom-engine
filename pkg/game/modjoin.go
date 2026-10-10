package game

import (
	"fmt"
	"slices"

	"againrom/pkg/mod"
)

// SetModCompanions records the join conditions the mods declare. Each names a
// companion the campaign's town grants and a tavern conversation of that
// town's own list; a condition the campaign does not bear out is refused with
// the mod, the file and the line. It runs once, after SetMods and before any
// game is opened.
func (f *FrontEnd) SetModCompanions(d mod.CompanionData) error {
	if d.Empty() {
		return nil
	}
	if f == nil || f.Table == nil {
		return fmt.Errorf("the front end has no definition table to carry the mods")
	}
	c := f.Campaign.Value()
	for _, j := range d.Joins {
		bad := func(format string, args ...any) error {
			return itemFileError(j.Mod, j.File, j.Line, format, args...)
		}
		ch, ok := c.Chapters[j.Chapter]
		if !ok {
			return bad("chapter %d is not a chapter of this install's campaign", j.Chapter)
		}
		if !c.townGrants(j.Companion) {
			return bad("companion %d joins on a map, not in a town; no AddHero of this install's campaign grants it", j.Companion)
		}
		if !slices.Contains(ch.AddHero, j.Companion) {
			return bad("the town of chapter %d does not grant companion %d", j.Chapter, j.Companion)
		}
		if !slices.Contains(ch.InnNPC, j.Talk) {
			return bad("the tavern of chapter %d has no conversation with record %d (it has %v)", j.Chapter, j.Talk, ch.InnNPC)
		}
	}
	f.Table.Mods.Companions = d
	return nil
}

// joinOnTalk joins the held companions whose tavern conversation talk has just
// opened in the current chapter's town, and reports whether any joined.
func (s *CampaignSession) joinOnTalk(in townInstall, talk int) bool {
	if s == nil || in.table == nil || s.Town == nil || in.table.Mods.Companions.Empty() {
		return false
	}
	chapter := s.Town.Chapter()
	joined := false
	for _, j := range in.table.Mods.Companions.ForTalk(chapter, mod.BuildingTavern, talk) {
		if !s.Town.takeAddHero(chapter, j.Companion) {
			continue
		}
		if s.carryTownCompanion(in, chapter, j.Companion) {
			joined = true
		}
	}
	s.settleCompanionArrival(in, joined)
	return joined
}
