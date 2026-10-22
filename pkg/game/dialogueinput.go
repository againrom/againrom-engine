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

// dialogueRead is one part of a dialogue text: the tag the scan accepted, and
// the words of its pointer interval when it has one.
type dialogueRead struct {
	tag      string
	accepted bool
	text     string
	ok       bool
}

// readDialoguePart is the one part reader of the town talk of both games and
// the mission notice; candidate is scanEventPart's.
func readDialoguePart(payload []byte, part int, audience EventAudience, candidate func(tag string)) dialogueRead {
	payload = dialoguePayload(payload)
	open, close, _, ok := scanEventPart(payload, part, audience, candidate)
	if !ok {
		return dialogueRead{}
	}
	s := string(payload)
	text, ok := acceptedDialogueTail(s[close+1:])
	return dialogueRead{tag: s[open+1 : close], accepted: true, text: text, ok: ok}
}

func dialoguePart(payload []byte, part int, audience EventAudience) (string, bool) {
	r := readDialoguePart(payload, part, audience, nil)
	return r.text, r.ok
}

// tips is the `tips=` value the accepted tag carries, and whether it carries
// one (TRIG-TIPS-087). Tags are case-folded, so `Tips=3` counts.
func (r dialogueRead) tips() (int, bool) {
	if !r.accepted {
		return 0, false
	}
	at := indexFold(r.tag, "tips=")
	if at < 0 {
		return 0, false
	}
	return dialogueInteger(r.tag[at+5:])
}

func dialoguePartTips(payload []byte, part int, audience EventAudience) (int, bool) {
	return readDialoguePart(payload, part, audience, nil).tips()
}
