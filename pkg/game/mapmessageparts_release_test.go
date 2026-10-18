package game

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func messageFirstLine(body string) string {
	for _, line := range strings.Split(messageMarkup.ReplaceAllString(body, ""), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}

func TestReleaseMessagePickerParts(t *testing.T) {
	x := installedInspector(t)
	maps, pages, absent, textEdits, speakerEdits, outOfOrder := 0, 0, 0, 0, 0, 0
	for _, entry := range x.Maps {
		key := "loose/" + entry.Source
		if entry.FromArchive {
			key = "scenario/" + entry.Source
		}
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(key, err)
		}
		maps++
		s := e.Document().Edits.(*mapInspectionEdits)
		almBefore := bytes.Clone(s.model.Bytes())
		for _, row := range e.Document().Script.Targets[ui.ScriptMessageParameter] {
			if row.NewMessage {
				continue
			}
			pages++
			t.Run(fmt.Sprintf("%s/%d", key, row.Value), func(t *testing.T) {
				name := messageFilename(row.Value)
				before := bytes.Clone(s.workspace.Pages[name])
				decoded := x.messageUTF8(before)
				body, found := EventPart([]byte(decoded), 1, EventAudience{})
				if !found {
					absent++
					body = decoded
				}
				want := x.inspectionUTF8(fmt.Sprintf("%d: %s", row.Value, messageFirstLine(body)))
				if row.Name != want || row.Choice != want {
					t.Errorf("row differs from game part 1: got %q want %q", row.Choice, want)
				}
				fields := s.MessageFields(row.Value)
				if len(fields) != 2 {
					t.Fatal("missing popup fields", fields)
				}
				if d, err := s.SetMessage(row.Value, fields[0].TextValue, fields[1].Value); err != nil || d != nil || !bytes.Equal(before, s.workspace.Pages[name]) {
					t.Fatal("unchanged Apply changed bytes", err)
				}
				if !found {
					if fields[0].TextValue != decoded || fields[1].Value != -1 {
						t.Fatal("absent-part fallback changed", fields)
					}
					return
				}
				tag, tail, _ := eventPartTail([]byte(decoded), 1, EventAudience{})
				start := len(decoded) - len(tail)
				end := start + len(body)
				headerStart := start - len(tag) - 2
				if strings.IndexByte(decoded, '<') != headerStart {
					outOfOrder++
				}
				speaker := int64(-1)
				if id, named := EventPartSpeaker([]byte(decoded), 1, EventAudience{}); named {
					speaker = int64(id)
				}
				if fields[0].TextValue != strings.TrimPrefix(strings.TrimPrefix(body, "\r\n"), "\n") || fields[1].Value != speaker {
					t.Fatalf("popup differs from game part 1: speaker=%d want=%d body=%q", fields[1].Value, speaker, fields[0].TextValue)
				}
				checkUndo := func() {
					t.Helper()
					if _, err := s.Undo(); err != nil || !bytes.Equal(before, s.workspace.Pages[name]) || !bytes.Equal(almBefore, s.model.Bytes()) {
						t.Fatal("Undo did not restore exact page/ALM bytes", err)
					}
				}
				if _, err := s.SetMessage(row.Value, fields[0].TextValue+"!", fields[1].Value); err != nil {
					t.Fatal(err)
				}
				after := x.messageUTF8(s.workspace.Pages[name])
				gotBody, ok := EventPart([]byte(after), 1, EventAudience{})
				if !ok || gotBody != body+"!" || after != decoded[:end]+"!"+decoded[end:] {
					t.Error("text Apply changed the wrong part, header or surrounding bytes")
				}
				textEdits++
				checkUndo()
				other := int64(-1)
				for _, choice := range fields[1].Choices {
					if choice.Value >= 0 && choice.Value != fields[1].Value {
						other = choice.Value
						break
					}
				}
				if other == fields[1].Value {
					t.Fatal("speaker loss control has no alternative")
				}
				if _, err := s.SetMessage(row.Value, fields[0].TextValue, other); err != nil {
					t.Fatal(err)
				}
				after = x.messageUTF8(s.workspace.Pages[name])
				afterTag, afterTail, ok := eventPartTail([]byte(after), 1, EventAudience{})
				id, named := EventPartSpeaker([]byte(after), 1, EventAudience{})
				if !ok || named != (other >= 0) || named && int64(id) != other || afterTail != tail || !strings.HasPrefix(after, decoded[:headerStart]+"<") || len(after)-len(afterTail)-len(afterTag)-2 != headerStart || strings.Trim(messageSpeaker.ReplaceAllString(afterTag, ""), ",") != strings.Trim(messageSpeaker.ReplaceAllString(tag, ""), ",") {
					t.Error("speaker Apply changed the wrong header, attributes or surrounding parts")
				}
				speakerEdits++
				checkUndo()
				if x.pageLanguage() == "ru" && (key == "scenario/10.alm" && row.Value == 2 || key == "scenario/30.alm" && row.Value == 15) {
					t.Logf("%s page=%d first_speaker=%d line=%q; text/speaker Apply preserve other parts and Undo restores exact bytes", key, row.Value, speaker, messageFirstLine(body))
				}
			})
		}
	}
	if pages == 0 || textEdits == 0 || speakerEdits == 0 || x.pageLanguage() == "ru" && outOfOrder < 2 {
		t.Fatal("empty page/edit/out-of-order population", pages, textEdits, speakerEdits, outOfOrder)
	}
	t.Logf("lang=%s maps=%d pages=%d absent_part1=%d text_apply=%d speaker_apply=%d nonfirst_part1=%d", x.pageLanguage(), maps, pages, absent, textEdits, speakerEdits, outOfOrder)
}
