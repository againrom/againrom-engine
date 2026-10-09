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
		// The tavern buttons and the square's five hints are the installed
		// words the first town draws (DIV-2378).
		values := []*string{&result.MissionWon, &result.MenuVictory, &result.OutcomeContinue,
			&result.TavernTalk, &result.TavernExit, &result.TavernHired}
		for slot := 233; slot <= 237; slot++ {
			values = append(values, &result.Hover[slot])
		}
		for _, value := range values {
			*value = string(secondGameMissionBytes([]byte(*value), words.Selector))
		}
	}
	return result
}
