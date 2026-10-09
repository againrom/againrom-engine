package game

import "strings"

func dialoguePayload(payload []byte) []byte {
	if end := strings.IndexByte(string(payload), 0); end >= 0 {
		return payload[:end]
	}
	return payload
}

// acceptedDialogueTail preserves the resource pointer interval. The native
// backward scan is unbounded; ranges before body start are rejected here.
func acceptedDialogueTail(tail string) (string, bool) {
	if end := strings.IndexByte(tail, 0); end >= 0 {
		tail = tail[:end]
	}
	lf := strings.IndexByte(tail, '\n')
	if lf < 0 {
		return "", false
	}
	start, end := lf+1, len(tail)
	if next := strings.IndexByte(tail[start:], '<'); next >= 0 {
		end = strings.LastIndexByte(tail[:start+next], '\r')
		if end < start {
			return "", false
		}
	}
	return tail[start:end], true
}

func dialoguePart(payload []byte, part int, audience EventAudience) (string, bool) {
	_, tail, ok := eventPartTail(dialoguePayload(payload), part, audience)
	if !ok {
		return "", false
	}
	return acceptedDialogueTail(tail)
}

// dialoguePartTips is the `tips=` value the accepted tag of part carries, and
// whether it carries one (TRIG-TIPS-087). Tags are case-folded, so `Tips=3`
// counts.
func dialoguePartTips(payload []byte, part int, audience EventAudience) (int, bool) {
	tag, _, ok := eventPartTail(dialoguePayload(payload), part, audience)
	if !ok {
		return 0, false
	}
	at := indexFold(tag, "tips=")
	if at < 0 {
		return 0, false
	}
	return dialogueInteger(tag[at+5:])
}
