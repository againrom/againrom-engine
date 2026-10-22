package game

import (
	"image"
	"strconv"
	"strings"

	"againrom/pkg/ui"
)

type townDialogueState struct {
	text        string
	tag         string
	displayPart int
	hasPortrait bool
	portrait    bool
	speaker     int
	tips        int
	face        *image.RGBA
	faceWindow  image.Rectangle
	closeTips   int
}

func newTownDialogue(payload []byte) townDialogueState {
	return townDialogueState{text: "Nothing to say", hasPortrait: EventHasSpeaker(dialoguePayload(payload)), portrait: EventHasSpeaker(dialoguePayload(payload))}
}

// firstTownTalkLowestPart is the first game's lowest part; the shared scan
// answers part 0 too.
const firstTownTalkLowestPart = 1

// lookup reads the cursor's part. Every tag naming the part, audience-rejected
// or not, sets the portrait and speaker; the accepted tag sets the tips even
// when its words fail.
func (d *townDialogueState) lookup(c talkCursor, audience EventAudience) bool {
	if c.part < firstTownTalkLowestPart {
		return false
	}
	r := c.read(audience, d.noteSpeaker)
	if tips, ok := r.tips(); ok {
		d.tips = tips
	}
	if !r.ok {
		return false
	}
	d.text, d.tag, d.displayPart = r.text, r.tag, c.part
	return true
}

func (d *townDialogueState) noteSpeaker(tag string) {
	speaker, named := tagSpeaker(tag)
	d.portrait = named
	if named {
		d.speaker = speaker
	}
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

func (t *townScreen) lookupTownDialogue() bool {
	if !t.dialogue.lookup(t.talkPage, partyAudience(t.sess.Carried, nil)) {
		return false
	}
	if t.dialogue.portrait {
		if t.mercenaryTalk != 0 {
			t.dialogue.face, _ = t.mercenaryTalkPicture(t.mercenaryTalk)
			rec := t.in.NPCFaces[int32(t.dialogue.speaker)]
			t.dialogue.faceWindow = image.Rectangle{}
			if rec.HasWindow {
				t.dialogue.faceWindow = ui.NoticeFaceWindow(rec.WindowX, rec.WindowY)
			}
		} else {
			t.dialogue.face, t.dialogue.faceWindow, _ = t.speakerFace(t.dialogue.speaker)
		}
	}
	return true
}
