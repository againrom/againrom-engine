package ui

import "sort"

func targetAt(ents []MapEntity, at func(MapEntity) (screenRect, bool), x, y float64, allowCorpse bool) (uint32, bool) {
	var entries []hoverRegistration
	for _, e := range ents {
		if e.Untargetable && !allowCorpse {
			continue
		}
		if r, ok := at(e); ok {
			entries = append(entries, hoverRegistration{hit: hoverHit{subject: InspectionSubject{InspectionUnit, e.ID}, rect: r}})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].hit.subject.ID > entries[j].hit.subject.ID })
	h := registeredHover(entries, int(x), int(y))
	return h.subject.ID, h.subject.Kind == InspectionUnit
}

func decideFixture(cur selection, ents []MapEntity, groundAt func(float64, float64) (int, int, bool), at func(MapEntity) (screenRect, bool), local uint32, g gesture) (selection, []uint32, []order, bool) {
	if g.hit.Kind == InspectionNone {
		if id, ok := targetAt(ents, at, float64(g.x), float64(g.y), g.bodySpell); ok {
			g.hit = InspectionSubject{InspectionUnit, id}
		}
	}
	return decide(cur, ents, groundAt, at, local, g)
}
