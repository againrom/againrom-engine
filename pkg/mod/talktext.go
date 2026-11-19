package mod

import (
	"fmt"
	"regexp"
	"strings"
)

var talkTag = regexp.MustCompile(`^\s*<part=1[^>]*>\r?\n?`)

// TalkText is the text file of a tavern talk of NPC record npc that says
// body, in the part form of the install's talk texts.
func TalkText(npc int, body string) []byte {
	body = strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(body), "\r\n", "\n"), "\n", "\r\n")
	return []byte(fmt.Sprintf("<part=1 npc=%d>\r\n%s\r\n", npc, body))
}

// TalkBody is the words of the first part of a talk text file.
func TalkBody(b []byte) string {
	s := talkTag.ReplaceAllString(string(b), "")
	if i := strings.Index(s, "<part="); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}
