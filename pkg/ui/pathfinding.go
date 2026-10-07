package ui

// Pathfinding is an optional display overlay. It never changes route execution.
func (v *Viewer) SetPathfinding(show bool) { v.showPathfinding = show }
func (v *Viewer) PathfindingShown() bool   { return v.showPathfinding }

func (v *Viewer) displayedPathSegments() []pathSegment {
	if !v.showPathfinding {
		return nil
	}
	return v.pathScreenSegments()
}
