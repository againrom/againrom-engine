package game

import (
	"image"
	"strconv"
	"strings"

	"againrom/pkg/ui"
)

type townDialogueState struct {
	shows       uint64
	payload     []byte
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
	return townDialogueState{payload: payload, text: "Nothing to say", hasPortrait: EventHasSpeaker(dialoguePayload(payload)), portrait: EventHasSpeaker(dialoguePayload(payload))}
}

func (d *townDialogueState) lookup(part int, audience EventAudience) bool {
	if part < 1 {
		return false
	}
	want := partMark + strconv.Itoa(part)
	s := string(dialoguePayload(d.payload))
	for cursor := 0; cursor < len(s); {
		open := strings.IndexByte(s[cursor:], '<')
		if open < 0 {
			return false
		}
		open += cursor
		end := strings.IndexByte(s[open:], '>')
		if end < 0 {
			return false
		}
		end += open
		cursor = end + 1
		tag := s[open+1 : end]
		if !containsFold(tag, want) {
			continue
		}
		speaker, named := tagSpeaker(tag)
		d.portrait = named
		if named {
			d.speaker = speaker
		}
		if !audience.accepts(tag) {
			continue
		}
		if at := indexFold(tag, "tips="); at >= 0 {
			if value, ok := dialogueInteger(tag[at+5:]); ok {
				d.tips = value
			}
		}
		body, ok := acceptedDialogueTail(s[cursor:])
		if !ok {
			return false
		}
		d.text, d.tag, d.displayPart = body, tag, part
		return true
	}
	return false
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
	if !t.dialogue.lookup(t.said, HeroAudience(t.sess.Carried)) {
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
