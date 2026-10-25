package game

import (
	"testing"

	"againrom/internal/firstgame"
)

func TestReleaseMilestone2MoverRoutes1160(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	_, raw := groundCorpusFile(t, "2026-08-02/game0009.sav", "60267c82072c77446ab9b34913318e89eab8f70e49f3510ae64aaaf423819bd6")
	mover1160App(t, raw, releaseFront)
}
