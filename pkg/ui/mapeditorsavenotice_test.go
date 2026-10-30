package ui

import (
	"errors"
	"strings"
	"testing"
)

// noticeEdits records the status line as the write finds it.
type noticeEdits struct {
	editorTextEdits
	e      *MapEditor
	fail   error
	during []string
}

func (p *noticeEdits) SaveAs(path string) (string, error) {
	p.during = append(p.during, p.e.statusText())
	if p.fail != nil {
		return "", p.fail
	}
	return path, nil
}

func noticeEditor(t *testing.T) (*MapEditor, *noticeEdits) {
	t.Helper()
	p := &noticeEdits{}
	e := testPlayEditor(t, p)
	p.e = e
	return e, p
}

// A save shows the writing notice on the status line before it writes, the
// written notice after it, and the error when the write is refused.
func TestMapEditorSaveShowsWritingWrittenAndFailedNotices(t *testing.T) {
	writing, written, failed := "Saving the map", "Map saved: made.alm", "Save failed: disk full"
	e, p := noticeEditor(t)
	if err := e.saveAs("dir/made.alm"); err != nil {
		t.Fatal(err)
	}
	if len(p.during) != 1 || !strings.Contains(p.during[0], writing) {
		t.Errorf("status while writing: %q, want it to carry %q", p.during, writing)
	}
	if got := e.statusText(); !strings.Contains(got, written) || strings.Contains(got, writing) {
		t.Errorf("status after the write: %q, want %q only", got, written)
	}

	e, p = noticeEditor(t)
	p.fail = errors.New("disk full")
	if err := e.saveAs("dir/made.alm"); err == nil {
		t.Fatal("the refused write returned no error")
	}
	if got := e.statusText(); !strings.Contains(got, failed) || strings.Contains(got, writing) {
		t.Errorf("status after a refused write: %q, want %q", got, failed)
	}
}

// In the window a save waits one frame, so the frame that shows the writing
// notice is drawn before the write blocks the next.
func TestMapEditorWindowSaveWaitsOneFrameForItsNotice(t *testing.T) {
	e, p := noticeEditor(t)
	e.deferSaves = true
	if err := e.saveAs("dir/made.alm"); err != nil {
		t.Fatal(err)
	}
	if len(p.during) != 0 {
		t.Fatal("the save wrote in the frame that asked for it")
	}
	if got := e.statusText(); !strings.Contains(got, "Saving the map") {
		t.Fatalf("status while the save waits: %q", got)
	}
	updateEditorKeys(t, e)
	if len(p.during) != 1 || !strings.Contains(e.statusText(), "Map saved: made.alm") {
		t.Fatalf("the next frame wrote %d times, status %q", len(p.during), e.statusText())
	}
}
