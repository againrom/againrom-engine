package game

import (
	"fmt"

	"againrom/pkg/base"
)

// ReadEventTextFor is the text of the mission's event as game g's files lay it
// out.
func ReadEventTextFor(src entrySource, game base.Game, mission, event int) ([]byte, bool) {
	return filesOf(game).eventText(src, game.Edition(), mission, event)
}

// readSecondGameEventText is event's section of the second game's mission
// text file.
func readSecondGameEventText(src entrySource, code TextCode, mission, event int) ([]byte, bool) {
	text := installText(src, missionTextSource.at(mission), code)
	if !text.present() {
		return nil, false
	}
	body := text.section(fmt.Sprintf("event%d", event))
	return []byte(body), body != ""
}

func readSecondGameMissionText(src entrySource, code TextCode, mission int) ([]byte, bool) {
	text := installText(src, missionTextSource.at(mission), code)
	return text.bytes(), text.present()
}

func secondGameFailureText(src entrySource, code TextCode, mission int) []string {
	labels, _ := secondGameFailureEntries(src, code, mission)
	return labels
}

// secondGameFailureEntries is the mission's failure cache, entry n for
// reason n + 2 (R2-ENGINE-629), and the reasons whose entry is the main
// text slot because the mission text has no section for them.
func secondGameFailureEntries(src entrySource, code TextCode, mission int) (labels []string, slots []int32) {
	text := installText(src, missionTextSource.at(mission), code)
	if !text.present() {
		return nil, nil
	}
	words := LoadInstallWords(src, code)
	for reason := 2; reason < 1024; reason++ {
		label := text.section(fmt.Sprintf("failure%d", reason))
		if label == "" {
			if reason > 4 {
				break
			}
			label, _ = words.Global(0x118 + reason)
			slots = append(slots, int32(reason))
		}
		labels = append(labels, label)
	}
	return labels, slots
}
