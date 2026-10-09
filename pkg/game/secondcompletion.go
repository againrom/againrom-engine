package game

import (
	"againrom/pkg/base"
	"againrom/pkg/ui"
)

// secondGameCompletionDirectory is the movie stem a departure output selects:
// the cutpaths row the output indexes (R2-ENGINE-075). Only outputs 1..5
// are stored by a case body (R2-ENGINE-148); -1 and every other value
// select no movie (DIV-2628).
func secondGameCompletionDirectory(src entrySource, output int) string {
	if output < 1 || output > 5 {
		return ""
	}
	stem, _ := LoadTextTable(src, mainPrefix+"text/cutpaths.txt").At(output)
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
