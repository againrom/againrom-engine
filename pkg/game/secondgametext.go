package game

import (
	"fmt"
	"strings"

	"againrom/pkg/base"
	"againrom/pkg/vfs"
)

func ReadEventTextFor(src entrySource, game base.Game, mission, event int) ([]byte, bool) {
	if game != base.GameROM2 {
		return ReadEventText(src, mission, event)
	}
	payload, ok := readSecondGameMissionText(src, mission)
	if !ok {
		return nil, false
	}
	body := secondGameTextSection(payload, fmt.Sprintf("event%d", event))
	return []byte(body), body != ""
}

func readSecondGameMissionText(src entrySource, mission int) ([]byte, bool) {
	if src == nil || mission <= 0 {
		return nil, false
	}
	payload, err := src.ReadFile(fmt.Sprintf("%stext/mission%d.txt", mainPrefix, mission))
	if err != nil {
		return nil, false
	}
	return secondGameMissionBytes(payload, LanguageSelector(src)), true
}

func secondGameFailureText(src entrySource, mission int) []string {
	payload, ok := readSecondGameMissionText(src, mission)
	if !ok {
		return nil
	}
	words := LoadInstallWords(src)
	var labels []string
	for reason := 2; reason < 1024; reason++ {
		label := secondGameTextSection(payload, fmt.Sprintf("failure%d", reason))
		if label == "" {
			if reason > 4 {
				break
			}
			label, _ = words.Global(0x118 + reason)
		}
		labels = append(labels, label)
	}
	return labels
}

// R2-ENGINE-049/050: first substring, two separator bytes, then the next '#'.
func secondGameTextSection(payload []byte, key string) string {
	source := string(dialoguePayload(payload))
	at := strings.Index(source, "#"+strings.ToLower(key))
	if at < 0 {
		return ""
	}
	start := at + len(key) + 3
	if start >= len(source) {
		return ""
	}
	body := source[start:]
	if end := strings.IndexByte(body, '#'); end >= 0 {
		body = body[:end]
	}
	return body
}

// DIV-2378: portable conversion for the installed RU mission text.
func secondGameMissionBytes(payload []byte, selector int) []byte {
	if selector != 1 {
		return payload
	}
	return vfs.WindowsCyrillicToDOS(payload)
}
