package ui

import "testing"

// The reply row's Ответ field is UTF-8 in the window and reaches the row in the
// font's bytes, converted once as the row is built, as the header's text is.
func TestReplyRowShowsTheTextInTheFontBytes(t *testing.T) {
	e := &MapEditor{}
	e.hostText = func(s string) string { return "font:" + s }
	e.obj.fields = []InspectionField{{Key: dlgReplyText, Text: true}}
	e.obj.buf = []string{"Продолжить борьбу"}
	got := e.replyScope().Values["text"]
	if want := "font:Продолжить борьбу"; got != want {
		t.Fatalf("the reply row shows %q, want %q", got, want)
	}
}
