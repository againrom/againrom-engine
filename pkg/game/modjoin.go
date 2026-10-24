package game

import (
	"fmt"
	"slices"

	"againrom/pkg/mod"
)

// SetModCompanions records the join conditions the mods declare, merged by
// mod.MergeCompanions. Each names a companion the campaign's town grants and
// a tavern conversation of that town's own list; a condition the campaign
// does not bear out is refused with the mod, the file and the line. It runs
// once, after SetMods and before any game is opened.
func (f *FrontEnd) SetModCompanions(d mod.CompanionData) error {
	_, err := f.setModCompanions(d)
	return err
}

// setModCompanions is SetModCompanions answering the fields a later mod
// set over an earlier one.
func (f *FrontEnd) setModCompanions(d mod.CompanionData) ([]mod.Override, error) {
	if d.Empty() {
		return nil, nil
	}
	if f == nil || f.Table == nil {
		return nil, fmt.Errorf("the front end has no definition table to carry the mods")
	}
	d, overrides, err := mod.MergeCompanions(d)
	if err != nil {
		return nil, err
	}
	c := f.Campaign.Value()
	for _, j := range d.Joins {
		bad := func(format string, args ...any) error {
			return itemFileError(j.Mod, j.File, j.Line, format, args...)
		}
		ch, ok := c.Chapters[j.Chapter]
		if !ok {
			return nil, bad("chapter %d is not a chapter of this install's campaign", j.Chapter)
		}
		if !c.townGrants(j.Companion) {
			return nil, bad("companion %d joins on a map, not in a town; no AddHero of this install's campaign grants it", j.Companion)
		}
		if !slices.Contains(ch.AddHero, j.Companion) {
			return nil, bad("the town of chapter %d does not grant companion %d", j.Chapter, j.Companion)
		}
		if talks := c.chapter(j.Chapter).InnNPC; !slices.Contains(talks, j.Talk) {
			return nil, bad("the tavern of chapter %d has no conversation with record %d (it has %v)", j.Chapter, j.Talk, talks)
		}
	}
	f.Table.Mods.Companions = d
	return overrides, nil
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
