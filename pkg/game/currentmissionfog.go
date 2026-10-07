package game

import (
	"fmt"
	"slices"
)

type currentMissionFog struct {
	Cols, Rows        int
	Explored, Visible []byte
}

func captureCurrentMissionFog(r SnapshotResidue) *currentMissionFog {
	if r.FogVisible == nil {
		return nil
	}
	return &currentMissionFog{r.FogCols, r.FogRows, slices.Clone(r.FogExplored), slices.Clone(r.FogVisible)}
}
func (f *currentMissionFog) residue() SnapshotResidue {
	return SnapshotResidue{FogCols: f.Cols, FogRows: f.Rows, FogExplored: f.Explored, FogVisible: f.Visible}
}
func (f *currentMissionFog) validate() error {
	if f.Cols <= 0 || f.Cols > 256 || f.Rows <= 0 || f.Rows > 256 || f.Visible == nil {
		return fmt.Errorf("invalid current visibility extent")
	}
	return validateNativeFogResidue(f.residue())
}
