package game

import (
	"image"
	"strconv"
	"strings"
)

// townDialogueState is the page of a town talk; shown is set once a part
// reads.
type townDialogueState struct {
	text        string
	tag         string
	displayPart int
	shown       bool
	hasPortrait bool
	portrait    bool
	speaker     int
	tips        int
	face        *image.RGBA
	faceWindow  image.Rectangle
	closeTips   int
}

// dialogueDefaultText is the first game's dialogue panel text before a page
// shows (DLG-WIN-001); the program holds it in English on both roots.
const dialogueDefaultText = "Nothing to say"

func newTownDialogue(payload []byte) townDialogueState {
	return townDialogueState{text: dialogueDefaultText, hasPortrait: EventHasSpeaker(dialoguePayload(payload)), portrait: EventHasSpeaker(dialoguePayload(payload))}
}

// townTalkLowestPart is the town talk's lowest part; the shared scan answers
// part 0 too.
const townTalkLowestPart = 1

// lookup is the town talk's one part lookup. The accepted tag sets the tips
// even when its words fail; the profile's standing face picks the speaker.
func (d *townDialogueState) lookup(c talkCursor, audience EventAudience, p townTalkProfile) bool {
	if c.part < townTalkLowestPart {
		return false
	}
	rules := audience.tagRules()
	var note func(tag string)
	if p.standingFace {
		note = func(tag string) {
			speaker, named := rules.speaker(tag)
			d.portrait = named
			if named {
				d.speaker = speaker
			}
		}
	}
	r := c.read(audience, note)
	if tips, ok := r.tips(); ok {
		d.tips = tips
	}
	if !r.ok {
		return false
	}
	d.text, d.tag, d.displayPart, d.shown = r.text, r.tag, c.part, true
	if !p.standingFace {
		d.speaker, d.portrait = rules.speaker(r.tag)
		if !d.portrait {
			d.face, d.faceWindow = nil, image.Rectangle{}
		}
	}
	return true
}

func dialogueInteger(s string) (int, bool) {
	s = strings.TrimLeft(s, " \t\r\n")
	end := 0
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		end++
	}
	start := end
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if start == end {
		return 0, false
	}
	v, err := strconv.ParseInt(s[:end], 10, 32)
	return int(v), err == nil
}
