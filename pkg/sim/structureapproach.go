package sim

import "sort"

func (w *World) approachCell(scratch *routeScratch, i int, centre cell, radius int64, reaches func(cell) bool) bool {
	var candidates []cell
	for y := max(int64(0), int64(centre.y)-radius); y <= min(int64(w.bounds.Height)-1, int64(centre.y)+radius); y++ {
		for x := max(int64(0), int64(centre.x)-radius); x <= min(int64(w.bounds.Width)-1, int64(centre.x)+radius); x++ {
			at := cell{int32(x), int32(y)}
			if (reaches == nil || reaches(at)) && w.open(scratch, terrainRelation, i, at.x, at.y) && w.restFree(scratch, i, at.x, at.y) {
				candidates = append(candidates, at)
			}
		}
	}
	start := cellOf(&w.entities[i])
	sort.SliceStable(candidates, func(a, b int) bool { return candidates[a].chebyshevTo(start) < candidates[b].chebyshevTo(start) })
	for _, goal := range candidates {
		if route, ok := w.searchRoute(scratch, i, terrainRelation, noWindow, w.farBudgetFor(i), exactGoal, goal.x, goal.y); ok && len(route) > 0 {
			w.walkTo(scratch, i, goal.x, goal.y)
			w.routes[i] = route
			return true
		}
	}
	return false
}
