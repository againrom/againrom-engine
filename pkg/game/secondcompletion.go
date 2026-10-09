package game

import (
	"againrom/pkg/base"
	"againrom/pkg/ui"
)

// secondCompletionMovie names the missions whose departure movie the engine plays.
func secondCompletionMovie(mission int) bool { return mission == 10 }

func secondGameCompletionDirectory(src entrySource, mission int) string {
	if !secondCompletionMovie(mission) {
		return ""
	}
	stem, _ := LoadTextTable(src, mainPrefix+"text/cutpaths.txt").At(1)
	return stem
}

func reportWordsFor(words *InstallWords, game base.Game) ui.Words {
	result := words.Words()
	if game == base.GameROM2 && words != nil {
		for _, value := range []*string{&result.MissionWon, &result.MenuVictory, &result.OutcomeContinue} {
			*value = string(secondGameMissionBytes([]byte(*value), words.Selector))
		}
	}
	return result
}
