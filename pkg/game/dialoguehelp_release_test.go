package game

import (
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func TestReleaseDialogueWarningAndHelp(t *testing.T) {
	x := installedInspector(t)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	message := uint32(0)
	for _, action := range e.Document().Script.Actions {
		if action.Opcode == 2 {
			message = action.ID
			break
		}
	}
	if message == 0 {
		t.Fatal("no message action")
	}
	if _, err := s.SetFields("Trigger", 0, []ui.InspectionFieldValue{{Key: "action0", Value: int64(message)}, {Key: "action1", Value: int64(message)}}); err != nil {
		t.Fatal(err)
	}
	if len(ofRule(s.Report(), "message-overlap")) != 1 || !strings.Contains(strings.Join(e.PanelLines(), " "), x.inspectionUTF8(x.word("editor.message.overlap"))) {
		t.Fatal("Map panel lacks the plain message warning", e.PanelLines())
	}
	messageFrame(t, e, x.pageLanguage(), "message-overlap-warning")
	if err := e.Key("help"); err != nil {
		t.Fatal(err)
	}
	rows, open := e.ReportOpen()
	found := false
	for i, row := range rows {
		if row.Text == x.inspectionUTF8(x.word("editor.keys.dialogue")) {
			found = true
			for range i + 4 {
				_ = e.Key("down")
			}
			break
		}
	}
	if !open || !found {
		t.Fatal("F1 lacks dialogue page help", rows)
	}
	messageFrame(t, e, x.pageLanguage(), "dialogue-f1-help")
}
