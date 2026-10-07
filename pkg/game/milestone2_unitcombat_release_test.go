package game

import "testing"

func TestReleaseMilestone2UnitCombat1158(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	unit1158App(t, raw, releaseFront)
}
