package game

import "fmt"

// SAV-CAMPAIGN-084: town activation consumes the current AddHero array,
// even when a corresponding companion is already present. A later absence
// of that member must not recreate the grant.
func (t *Town) pendingNativeHeroGrants(chapter int) []int {
	if t == nil || t.heroGrants[chapter] {
		return nil
	}
	return append([]int(nil), t.camp.Chapters[chapter].AddHero...)
}

func validateSnapshotHeroGrants(c Campaign, s Snapshot) error {
	if len(s.ConsumedHeroGrants) != 0 && !s.HeroGrantState {
		return fmt.Errorf("saved native companion grants have no native grant-state owner")
	}
	previous := 0
	for _, chapter := range s.ConsumedHeroGrants {
		if chapter <= previous || len(c.Chapters[chapter].AddHero) == 0 {
			return fmt.Errorf("saved native companion grant chapter %d is not an ordered authored grant", chapter)
		}
		previous = chapter
	}
	return nil
}

func restoreNativeHeroGrants(t *Town, s Snapshot) {
	t.heroGrants = make(map[int]bool)
	if s.HeroGrantState {
		for _, chapter := range s.ConsumedHeroGrants {
			t.heroGrants[chapter] = true
		}
		return
	}
	// Old native AGS did not persist consumption. Their normal return path
	// visited the current and earlier town chapters before SAVE. Preserve
	// that reached-town state even if the granted companion has since died.
	// This is an explicit compatibility default (DIV-1181), not recovered
	// grant history. An unopened town still has all its grants pending.
	if !s.Open {
		return
	}
	current := t.Chapter()
	for _, chapter := range t.camp.Main {
		if len(t.camp.Chapters[chapter].AddHero) != 0 {
			t.heroGrants[chapter] = true
		}
		if chapter == current {
			break
		}
	}
}
