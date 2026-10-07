package sim

// beginSavedRouteContinuation hands a motion's still-unwalked saved route to
// ordinary native movement, the moment its imported crossing stops owning
// the body: either at import, for a motion that starts centered with no
// crossing to finish, or on advanceSavedMotion's own arrival branch, once
// the crossing it was finishing reaches its first center. A pending saved
// move turn hands off here when it finishes. Other named issues still defer
// the route instead of silently skipping their unexecuted step (DIV-952).
//
// SAV-634 reads ROM1's own load arm reconstructing both lists as ordinary
// runtime state through the class's own append path, with no separate
// "first tick after load" consumer distinct from the runtime drivers a live
// unit would also call — nothing on the file side distinguishes a loaded
// route from one a fresh order produced. This is that reading's native-side
// counterpart: the same near-search step machinery that walks any other
// entity's stored w.routes[i] — subGoal, consume, restAt, occupancy — is
// what walks this one too, so a saved route is fulfilled without a fresh far
// search and without a second, hand-rolled stepping law next to the one this
// package already has.
//
// SAV-630 reads route element 0 as the step nearest the unit for both lists,
// so the walk is index 0 upward. A leading run of elements equal to the
// unit's own current cell — the crossing that just finished, or a file saved
// exactly on a cell boundary — is trimmed first, rather than hand a
// zero-distance cell to the near search as its first sub-goal.
//
// A chosen list must also ANCHOR to the unit's own saved cell before it is
// walked at all (DIV-953): SAV-630's own corpus corroboration is silent
// outside Chebyshev 2 of a list's first element, and two release-corpus
// records — one per list — sit 45 and 51 cells from the unit that carries
// them, which the near search would otherwise retarget across the whole map
// rather than refuse. A list failing this is treated exactly as an empty one.
//
// m.StaticRoute and m.DynamicRoute are read here, never written: they stay
// exactly what the file decoded to, in the walked copy handed to w.routes[i]
// instead. The corpus and release audits (originalmoverroute1134_test.go)
// compare a resumed world's carried mover/route state against the source
// file's own bytes before any tick runs — the same "carried, not consumed at
// load" shape stories 1130-1133 already use for their own restored state —
// and exportOriginalMoverRoutes re-exports those same fields. Spending them
// here would desync both from the file the moment this entry point fires
// during resume itself (the centered-at-import case), before either audit
// or a real SAVE ever observes the world.
//
// SAV-631, SAV-632
func (w *World) beginSavedRouteContinuation(i int, m *SavedActorMotion) bool {
	e := &w.entities[i]
	here := uint16(e.Y)<<8 | uint16(e.X)
	trim := func(list []uint16) []uint16 {
		for len(list) > 0 && list[0] == here {
			list = list[1:]
		}
		return list
	}
	raw := m.DynamicRoute
	packed := trim(raw)
	if len(packed) == 0 {
		raw = m.StaticRoute
		packed = trim(raw)
	}
	if len(packed) == 0 {
		return false
	}
	// DIV-953: SAV-630's own corpus corroboration anchors each list's first
	// (file/decode-order) element to the unit's own saved cell -- within 1
	// for DynamicRoute on 173/177 records, within 2 for StaticRoute on
	// 511/518 -- and is silent beyond that on the remaining 4 and 7. This
	// guard applies the looser of the two corroborated radii, Chebyshev 2,
	// to whichever list is chosen: SAV-630 reports two measurements of the
	// same underlying claim (the list starts at the unit), not two different
	// claims, and the corpus's own outliers sit at 45 and 51 cells -- both
	// far outside either radius, so the wider of the two still refuses them
	// without narrowing DynamicRoute past what a real record is known to do
	// (a dynamic list beginning exactly 2 cells from its unit, corroborated
	// on the release corpus itself). A list failing this is treated exactly
	// as an empty one: it is not applied, and the unit keeps its saved
	// position. raw[0] is the untrimmed element SAV-630 measured, not
	// packed[0]: trimming only ever drops a leading run equal to `here`
	// itself, so checking the chosen list before trimming is the same test
	// SAV-630 ran.
	const anchorRadius = 2
	anchor := cell{x: int32(raw[0] & 0xff), y: int32(raw[0] >> 8)}
	if (cell{x: e.X, y: e.Y}).chebyshevTo(anchor) > anchorRadius {
		return false
	}
	route := make([]cell, len(packed))
	for k, p := range packed {
		route[k] = cell{x: int32(p & 0xff), y: int32(p >> 8)}
	}
	last := route[len(route)-1]
	m.Current, m.Active, m.Issue = false, false, "native movement continues the imported route"
	w.cancelTurnForTargetChange(i, last.x, last.y)
	w.routes[i] = route
	e.HasTarget, e.TargetX, e.TargetY = true, last.x, last.y
	return true
}
