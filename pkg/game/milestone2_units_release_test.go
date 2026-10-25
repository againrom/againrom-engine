package game

import (
	"testing"

	"againrom/internal/firstgame"
)

func TestReleaseMilestone2UnitScalars1156(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	_, raw := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	unit1156App(t, raw, releaseFront)
}
