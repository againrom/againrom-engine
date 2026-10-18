package game

import (
	"bytes"
	"strings"
	"testing"
)

func TestMessagePartEditsFollowGameScan(t *testing.T) {
	for _, raw := range []string{
		"\r\n<NPC=21,Part=3>Later\r\n<NPC=51,Part=1,tips=7>\r\nFirst\r\n<Part=2>Second",
		"<part=1,iamfemale>Other audience<part=1,npc=51>\nFirst<part=1,npc=21>Duplicate<part=2>Second",
		"<tagpart=10,npc=51,tips=7>First<part=2>Second",
	} {
		t.Run(raw[:strings.IndexByte(raw, '>')+1], func(t *testing.T) {
			_, e := workspaceFixture(t)
			workspacePage(t, e, "event02.txt", raw)
			s := e.Document().Edits.(*mapInspectionEdits)
			before := bytes.Clone(s.workspace.Pages["event02.txt"])
			fields := s.MessageFields(2)
			body, ok := EventPart(before, 1, EventAudience{})
			if !ok || fields[0].TextValue != strings.TrimPrefix(strings.TrimPrefix(body, "\r\n"), "\n") || fields[1].Value != 51 {
				t.Fatal("popup did not select game's first part", fields)
			}
			if d, err := s.SetMessage(2, fields[0].TextValue, fields[1].Value); err != nil || d != nil || !bytes.Equal(before, s.workspace.Pages["event02.txt"]) {
				t.Fatal("unchanged Apply wrote bytes", err)
			}
			_, tail, _ := eventPartTail(before, 1, EventAudience{})
			end := len(raw) - len(tail) + len(body)
			if _, err := s.SetMessage(2, fields[0].TextValue+"!", 51); err != nil || string(s.workspace.Pages["event02.txt"]) != raw[:end]+"!"+raw[end:] {
				t.Fatal("text Apply changed another part or header", err, string(s.workspace.Pages["event02.txt"]))
			}
			if _, err := s.Undo(); err != nil || !bytes.Equal(before, s.workspace.Pages["event02.txt"]) {
				t.Fatal("text Undo", err)
			}
			if _, err := s.SetMessage(2, fields[0].TextValue, -1); err != nil {
				t.Fatal(err)
			}
			after := s.workspace.Pages["event02.txt"]
			if _, named := EventPartSpeaker(after, 1, EventAudience{}); named {
				t.Fatal("speaker Apply missed game's first part")
			}
			if _, afterTail, _ := eventPartTail(after, 1, EventAudience{}); afterTail != tail || strings.Count(string(after), "<") != strings.Count(raw, "<") {
				t.Fatal("speaker Apply changed body/parts or inserted a tag")
			}
			if _, err := s.Undo(); err != nil || !bytes.Equal(before, s.workspace.Pages["event02.txt"]) {
				t.Fatal("speaker Undo", err)
			}
		})
	}
}

func TestMessageMissingFirstPartKeepsFallback(t *testing.T) {
	_, e := workspaceFixture(t)
	raw := "<Part=2,npc=51>Later only"
	workspacePage(t, e, "event02.txt", raw)
	s := e.Document().Edits.(*mapInspectionEdits)
	fields := s.MessageFields(2)
	if fields[0].TextValue != raw || fields[1].Value != -1 {
		t.Fatal("missing part fallback changed", fields)
	}
	if d, err := s.SetMessage(2, raw, -1); err != nil || d != nil || string(s.workspace.Pages["event02.txt"]) != raw {
		t.Fatal("unchanged fallback wrote bytes", err)
	}
	if _, err := s.SetMessage(2, raw+"!", -1); err != nil || string(s.workspace.Pages["event02.txt"]) != "<part=1>\n"+raw+"!" {
		t.Fatal("changed fallback did not retain its existing creation behaviour", err)
	}
	if _, err := s.Undo(); err != nil || string(s.workspace.Pages["event02.txt"]) != raw {
		t.Fatal("fallback Undo", err)
	}
}

func TestMessageOutOfOrderPopupCancel(t *testing.T) {
	e, s := messagePopupFixture(t, 2)
	raw := "<part=3>Later<part=1,npc=51>First<part=2>Second"
	workspacePage(t, e, "event02.txt", raw)
	e.Select(recordIndex(t, e, "Action", 0), false)
	opened := false
	for i, line := range e.PopupLines() {
		if strings.HasPrefix(line, "Open Message") {
			r, ok := e.PopupRowRect(i)
			messageClick(t, e, r, ok)
			opened = true
			break
		}
	}
	if !opened || len(e.PopupLines()) != 2 || !strings.Contains(e.PopupLines()[0], "First") || strings.Contains(e.PopupLines()[0], "Later") {
		t.Fatal("popup did not open first displayed part", e.PopupLines())
	}
	messageField(t, e, 0)
	e.TextInput("Discard")
	messageButton(t, e, 1)
	if string(s.workspace.Pages["event02.txt"]) != raw {
		t.Fatal("Cancel changed parts")
	}
	messageButton(t, e, 0)
	if string(s.workspace.Pages["event02.txt"]) != raw {
		t.Fatal("unchanged Apply changed parts")
	}
}
