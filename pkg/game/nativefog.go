package game

import "fmt"

// Only the additive visibility sample enables the strict current contract.
// Older explored-only envelopes retain their previous dimension/length admission.
func validateNativeFogResidue(r SnapshotResidue) error {
	if r.FogVisible == nil {
		return nil
	}
	if r.FogCols <= 0 || r.FogRows <= 0 || uint64(r.FogCols) > uint64(maxSaveBytes)/uint64(r.FogRows) {
		return fmt.Errorf("saved visibility has invalid dimensions")
	}
	n := r.FogCols * r.FogRows
	if len(r.FogVisible) != n || len(r.FogExplored) != n {
		return fmt.Errorf("saved visibility has incomplete planes")
	}
	for i, v := range r.FogVisible {
		if v > 1 || r.FogExplored[i] > 1 || v > r.FogExplored[i] {
			return fmt.Errorf("saved visibility contradicts exploration at cell %d", i)
		}
	}
	return nil
}

func (mw *mapWorld) restoreNativeFog(r SnapshotResidue) bool {
	if mw.fog == nil || r.FogCols != mw.fog.cols || r.FogRows != mw.fog.rows || validateNativeFogResidue(r) != nil {
		return false
	}
	// A missing legacy plane is still missing. For a present older short
	// plane, restore its recorded cells and leave the unspecified suffix alone.
	if len(r.FogExplored) == 0 {
		return false
	}
	copy(mw.fog.explored, r.FogExplored)
	if r.FogVisible != nil {
		copy(mw.fog.visible, r.FogVisible)
	} else {
		// The old writer never recorded the previous sample. Keep the derived
		// legacy fallback only inside its actual saved exploration; opening a
		// checkpoint must not reveal additional terrain or hidden occupants.
		for i := range mw.fog.visible {
			if mw.fog.explored[i] == 0 {
				mw.fog.visible[i] = 0
			}
		}
	}
	if mw.world != nil && mw.view != nil {
		mw.push()
	}
	return true
}
