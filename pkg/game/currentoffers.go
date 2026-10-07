package game

import "slices"

// Labels retain stable UI indices absent from the compacted ordinary arrays.
// Mission and NPC are anchors only; changed ordinary rows invalidate labels.
type currentOfferLabels struct {
	Chapter   int
	Buildings [3][]TownOffer
}

func snapshotOfferLabels(t *Town) *currentOfferLabels {
	if t == nil {
		return nil
	}
	r := &currentOfferLabels{Chapter: t.Chapter()}
	for b := TownTavern; b <= TownSchool; b++ {
		r.Buildings[b] = t.Offers(b)
	}
	return r
}

func cloneOfferLabels(r *currentOfferLabels) *currentOfferLabels {
	if r == nil {
		return nil
	}
	next := *r
	for b := range next.Buildings {
		next.Buildings[b] = slices.Clone(next.Buildings[b])
	}
	return &next
}

func (t *Town) currentOfferLabels(b TownBuilding, ch Chapter) []TownOffer {
	if t.offerLabels == nil || t.offerLabels.Chapter != ch.Mission || b < TownTavern || b > TownSchool {
		return nil
	}
	labels := t.offerLabels.Buildings[b]
	rows := ch.Shop
	if b == TownTavern {
		rows = ch.Inn[:min(len(ch.Inn), len(ch.InnNPC))]
	} else if b == TownSchool {
		rows = ch.School
	}
	if len(labels) != len(rows) {
		return nil
	}
	for i, mission := range rows {
		if labels[i].Mission != mission || b == TownTavern && labels[i].NPC != ch.InnNPC[i] {
			return nil
		}
	}
	return labels
}

func currentOfferIndex(labels []TownOffer, ordinal int) int {
	if ordinal < len(labels) {
		return labels[ordinal].Index
	}
	return ordinal
}

func (t *Town) takeCurrentOffer(b TownBuilding, index int) (int, bool) {
	ch := t.ChapterData()
	labels := t.currentOfferLabels(b, ch)
	nextLabels := cloneOfferLabels(t.offerLabels)
	if labels == nil && b >= TownTavern && b <= TownSchool {
		nextLabels = snapshotOfferLabels(t)
		labels = nextLabels.Buildings[b]
	}
	ordinal := index
	if labels != nil {
		ordinal = -1
		for i, row := range labels {
			if row.Index == index {
				ordinal = i
				break
			}
		}
	}
	m, ok := t.progress.take(t.camp, b, ordinal)
	if !ok {
		return m, false
	}
	if t.currentMain() != ch.Mission {
		t.offerLabels = nil
	} else if labels != nil {
		next := slices.Clone(labels)
		nextLabels.Buildings[b] = append(next[:ordinal], next[ordinal+1:]...)
		t.offerLabels = nextLabels
	}
	t.taken[offerRef{ch.Mission, b, index}] = true
	t.refreshAvailable()
	return m, true
}
