package ui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/pathdialog"
)

// speakerAddEdits is the speaker pick host that writes speaker records. Unit
// 1, which no record shows, offers one; after a write the lists hold it.
type speakerAddEdits struct {
	*speakerPickEdits
	folder   string
	refusal  error
	readOnly string
	addErr   error
	reply    pathdialog.Result
	replyErr error
	added    [][3]any
}

func (h *speakerAddEdits) CanAddSpeaker(int, SpeakerMiss) bool  { return h.readOnly == "" }
func (h *speakerAddEdits) SpeakerAddRefusal(SpeakerMiss) string { return h.readOnly }
func (h *speakerAddEdits) SpeakerBindRefusal(int) error         { return h.refusal }
func (h *speakerAddEdits) SpeakerPictureFolder() string         { return h.folder }
func (h *speakerAddEdits) AddSpeaker(unit int, name, picture string) (int64, error) {
	if h.addErr != nil {
		return 0, h.addErr
	}
	h.added = append(h.added, [3]any{unit, name, picture})
	return 20000, nil
}
func (h *speakerAddEdits) MessageFields(page uint32, source ...int) []InspectionField {
	fields := h.speakerPickEdits.MessageFields(page, source...)
	if len(h.added) > 0 {
		fields[1].Choices = append(fields[1].Choices, InspectionChoice{Value: 20000, Label: "20000: Lady"})
	}
	return fields
}

// speakerAddEditor is the speaker pick editor over speakerAddEdits, with a
// dialog that records each request and answers h.reply.
func speakerAddEditor(t *testing.T, lang string) (*MapEditor, *speakerAddEdits, chan pathdialog.Request) {
	t.Helper()
	e, host := speakerPickEditor(t, lang)
	h := &speakerAddEdits{speakerPickEdits: host, folder: t.TempDir(), reply: pathdialog.Result{Path: "face.png"}}
	e.doc.Edits = h
	requests := make(chan pathdialog.Request, 8)
	bindEditorDialog(e, func(r pathdialog.Request) (pathdialog.Result, error) {
		requests <- r
		return h.reply, h.replyErr
	}, t.TempDir())
	return e, h, requests
}

// armSpeakerPrompt clicks the pick button and the unit no record shows, and
// the add-a-speaker prompt opens.
func armSpeakerPrompt(t *testing.T, e *MapEditor) {
	t.Helper()
	pressSpeakerPick(t, e, 0)
	clickCell(e, cellUnshownUnit)
	if e.prompt.kind != promptSpeaker || e.pick != pickParamUnit {
		t.Fatalf("the click left prompt %v, pick %q, status %q", e.prompt.kind, e.pick, e.statusText())
	}
}

func popupHolds(e *MapEditor, text string) bool {
	return strings.Contains(strings.Join(e.PopupLines(), "\n"), text)
}

func noRequest(t *testing.T, requests chan pathdialog.Request) {
	t.Helper()
	select {
	case r := <-requests:
		t.Fatalf("a window was asked for: %+v", r)
	default:
	}
}

// The click, the name, the file window: the record is written, the window
// listed PNG files from the host's folder, and the speaker field shows the
// new record's name at once, as it does for an installed record, in both
// the armed card and the choice list of the other card.
func TestNewSpeakerClickNameAndPictureWriteTheRecord(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, h, requests := speakerAddEditor(t, lang)
			armSpeakerPrompt(t, e)
			e.TextInput("Lady")
			if err := e.Key("enter"); err != nil {
				t.Fatal(err)
			}
			finishEditorDialog(t, e)
			r := <-requests
			want := []pathdialog.Filter{{Name: e.word("editor.speaker.add.pictures"), Extensions: []string{"png"}}}
			if r.Kind != pathdialog.OpenFile || r.StartDir != h.folder || r.Title != e.word("editor.speaker.add.picture") || !reflect.DeepEqual(r.Filters, want) {
				t.Fatalf("request = %+v", r)
			}
			if len(h.added) != 1 || h.added[0] != ([3]any{1, "Lady", "face.png"}) {
				t.Fatalf("added = %v", h.added)
			}
			if e.prompt.open() || e.pick != pickNone || e.obj.buf[1] != "20000" {
				t.Fatalf("after the write: prompt %v, pick %q, speaker %q", e.prompt.kind, e.pick, e.obj.buf[1])
			}
			if !popupHolds(e, "20000: Lady") {
				t.Fatalf("the speaker field does not show the name:\n%s", strings.Join(e.PopupLines(), "\n"))
			}
			other := false
			for _, c := range e.obj.fields[3].Choices {
				other = other || c.Label == "20000: Lady"
			}
			if !other {
				t.Fatal("the other card's list does not hold the new record")
			}
		})
	}
}

// An empty name leaves the prompt open, with the refusal, and asks no window.
func TestNewSpeakerEmptyNameKeepsThePrompt(t *testing.T) {
	e, h, requests := speakerAddEditor(t, "en")
	armSpeakerPrompt(t, e)
	if err := e.Key("enter"); err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("an empty name answered %v", err)
	}
	if e.prompt.kind != promptSpeaker || e.pathDialog.pending != nil || len(h.added) != 0 {
		t.Fatalf("after the empty name: prompt %v, pending %v, added %v", e.prompt.kind, e.pathDialog.pending != nil, h.added)
	}
	noRequest(t, requests)
}

// Esc at the prompt, and Cancel in the file window, go back to the armed
// speaker pick with the field unchanged and nothing written.
func TestNewSpeakerCancelReturnsToTheSpeakerPick(t *testing.T) {
	back := func(t *testing.T, e *MapEditor, h *speakerAddEdits) {
		t.Helper()
		if e.prompt.open() || e.pick != pickParamUnit || e.obj.buf[1] != "-1" || len(h.added) != 0 {
			t.Fatalf("after cancel: prompt %v, pick %q, speaker %q, added %v", e.prompt.kind, e.pick, e.obj.buf[1], h.added)
		}
	}
	t.Run("the prompt", func(t *testing.T) {
		e, h, _ := speakerAddEditor(t, "en")
		armSpeakerPrompt(t, e)
		_ = e.Key("escape")
		back(t, e, h)
	})
	t.Run("the file window", func(t *testing.T) {
		e, h, requests := speakerAddEditor(t, "en")
		h.reply = pathdialog.Result{Cancelled: true}
		armSpeakerPrompt(t, e)
		e.TextInput("Lady")
		if err := e.Key("enter"); err != nil {
			t.Fatal(err)
		}
		finishEditorDialog(t, e)
		<-requests
		back(t, e, h)
	})
}

// A picture the host refuses brings the name prompt back with the name kept
// and the refusal on the line; the next Enter asks the window again, and a
// picture it takes writes the record. Nothing is written by a refusal.
func TestNewSpeakerRefusedPictureBringsTheNameBack(t *testing.T) {
	e, h, requests := speakerAddEditor(t, "en")
	h.addErr = errors.New("the picture is 64x64; a speaker's picture is 72x96, the message window's face")
	armSpeakerPrompt(t, e)
	e.TextInput("Lady")
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	finishEditorDialog(t, e)
	<-requests
	if e.prompt.kind != promptSpeaker || e.prompt.path != "Lady" || !strings.Contains(e.message, "72x96") || e.pick != pickParamUnit {
		t.Fatalf("after the refusal: prompt %v %q, message %q, pick %q", e.prompt.kind, e.prompt.path, e.message, e.pick)
	}
	if len(h.added) != 0 || e.obj.buf[1] != "-1" {
		t.Fatalf("a refused picture wrote %v, speaker %q", h.added, e.obj.buf[1])
	}
	h.addErr = nil
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	finishEditorDialog(t, e)
	<-requests
	if len(h.added) != 1 || e.prompt.open() || e.obj.buf[1] != "20000" {
		t.Fatalf("the second picture: added %v, prompt %v, speaker %q", h.added, e.prompt.kind, e.obj.buf[1])
	}
}

// With no window to show, the choice is a no-op: back to the pick, the
// reason on the line, nothing written and no typed path.
func TestNewSpeakerNoDialogWritesNothing(t *testing.T) {
	check := func(t *testing.T, e *MapEditor, h *speakerAddEdits) {
		t.Helper()
		if e.prompt.open() || e.pick != pickParamUnit || len(h.added) != 0 || e.obj.buf[1] != "-1" {
			t.Fatalf("no window: prompt %v, pick %q, added %v, speaker %q", e.prompt.kind, e.pick, h.added, e.obj.buf[1])
		}
		if !strings.Contains(e.message, "No file window") {
			t.Fatalf("message %q", e.message)
		}
	}
	t.Run("no dialog bound", func(t *testing.T) {
		e, h, _ := speakerAddEditor(t, "en")
		bindEditorDialog(e, nil, "")
		armSpeakerPrompt(t, e)
		e.TextInput("Lady")
		if err := e.Key("enter"); err != nil {
			t.Fatal(err)
		}
		if e.pathDialog.pending != nil {
			t.Fatal("no dialog bound, yet a request went out")
		}
		check(t, e, h)
	})
	t.Run("dialog unavailable", func(t *testing.T) {
		e, h, requests := speakerAddEditor(t, "en")
		h.replyErr = pathdialog.ErrUnavailable
		armSpeakerPrompt(t, e)
		e.TextInput("Lady")
		if err := e.Key("enter"); err != nil {
			t.Fatal(err)
		}
		finishEditorDialog(t, e)
		<-requests
		check(t, e, h)
	})
}

// While the mod's records did not load the click shows the host's read-only
// line and asks for nothing.
func TestNewSpeakerReadOnlyStaysOnThePick(t *testing.T) {
	e, h, requests := speakerAddEditor(t, "en")
	h.readOnly = "Read-only context: the script failed"
	pressSpeakerPick(t, e, 0)
	clickCell(e, cellUnshownUnit)
	if e.prompt.open() || e.pick != pickParamUnit || e.pickWhy != h.readOnly {
		t.Fatalf("after the click: prompt %v, pick %q, why %q", e.prompt.kind, e.pick, e.pickWhy)
	}
	noRequest(t, requests)
	if len(h.added) != 0 {
		t.Fatalf("added %v", h.added)
	}
}

// A unit the host will not bind a record to is refused at the click, before
// any name or picture is asked, with the host's message on the line.
func TestNewSpeakerBindRefusalStaysOnThePick(t *testing.T) {
	e, h, requests := speakerAddEditor(t, "en")
	h.refusal = errors.New("this unit has no script number")
	pressSpeakerPick(t, e, 0)
	clickCell(e, cellUnshownUnit)
	if e.prompt.open() || e.pick != pickParamUnit || e.pickWhy != h.refusal.Error() {
		t.Fatalf("after the click: prompt %v, pick %q, why %q", e.prompt.kind, e.pick, e.pickWhy)
	}
	noRequest(t, requests)
	if len(h.added) != 0 {
		t.Fatalf("added %v", h.added)
	}
}
