package ui

import (
	"fmt"
	"image"
	"slices"
	"testing"

	"againrom/pkg/scriptprofile"
)

// replyListEdits hosts one dialogue whose one line offers one reply that
// tests conds conditions and runs acts actions.
type replyListEdits struct {
	dialogueListEdits
	conds, acts int
}

func (d *replyListEdits) ReadDialogue(string) (DialogueData, error) {
	r := DialogueReplyData{Text: map[string]string{"en": "go"}}
	for i := range d.conds {
		r.Pairs = append(r.Pairs, WizardPair{Left: WizardNode{ID: uint32(1000 + i)}, Right: WizardNode{ID: uint32(2000 + i)}})
	}
	for i := range d.acts {
		r.Actions = append(r.Actions, WizardNode{ID: uint32(3000 + i)})
	}
	n := DialogueNodeData{ID: 1, Name: "n1", Text: map[string]string{"en": "line", "ru": "line"}, Replies: []DialogueReplyData{r}}
	return DialogueData{Nodes: []DialogueNodeData{n}}, nil
}

// replyListEditor is the Dialog window at full HD on that dialogue with its
// reply open.
func replyListEditor(t *testing.T, conds, acts int) *MapEditor {
	t.Helper()
	e, _ := speakerPortraitEditor(t, "en")
	e.doc.Script = &InspectionScriptModel{}
	e.doc.Edits = &replyListEdits{dialogueListEdits: dialogueListEdits{speakerPortraitEdits: e.doc.Edits.(*speakerPortraitEdits)}, conds: conds, acts: acts}
	if err := e.openDialogue("talk", false, nil); err != nil {
		t.Fatal(err)
	}
	if e.openReply() == nil {
		e.toggleReply(0)
	}
	if e.openReply() == nil {
		t.Fatal("the reply does not open")
	}
	return e
}

// replyRowOf is the spec row that edits field key of the open reply, -1
// for none.
func replyRowOf(e *MapEditor, key string) int {
	f := e.dialogFieldAt(key)
	s, _, _ := e.objectSpec()
	for row := range s.Rows {
		if g, ok := e.obj.rowField[row]; ok && f >= 0 && g == f {
			return row
		}
	}
	return -1
}

// pressReplyRow presses the first button doing act on the spec row of field
// key, or on the first row whose first action is act when key is "".
func pressReplyRow(t *testing.T, e *MapEditor, key, act string) {
	t.Helper()
	s, _, _ := e.objectSpec()
	want := replyRowOf(e, key)
	for row := range s.Rows {
		acts := e.obj.rowActs[row]
		if k := slices.Index(acts, act); k >= 0 && (key == "" || row == want) {
			e.dialogRowButton(row, k)
			return
		}
	}
	t.Fatalf("no row of %q does %s", key, act)
}

// actionIDs are the open reply's action node ids in row order.
func actionIDs(e *MapEditor) []uint32 {
	var ids []uint32
	for _, r := range e.openReply().rows {
		if !r.condition {
			ids = append(ids, r.act)
		}
	}
	return ids
}

// A reply's every condition and action shows as a row of its own, not only
// its first of each.
func TestDialogReplyShowsAllItsRows(t *testing.T) {
	e := replyListEditor(t, 9, 17)
	for _, key := range []string{"c0.item", "c8.item", "a0.item", "a16.item"} {
		if replyRowOf(e, key) < 0 {
			t.Errorf("the reply shows no row for %s", key)
		}
	}
}

// The reply's rows are added, removed and moved by Alt+Up and Alt+Down, and
// a row past the extended trigger's limit is refused in words.
func TestDialogReplyRowsAddRemoveMoveAndRefuse(t *testing.T) {
	e := replyListEditor(t, 2, 3)
	e.focusObjectField(e.dialogFieldAt("a1.item"))
	_ = e.Key("cardup")
	if got := actionIDs(e); !slices.Equal(got, []uint32{3001, 3000, 3002}) {
		t.Fatalf("Alt+Up left the actions %v", got)
	}
	if e.obj.focus != e.dialogFieldAt("a0.item") {
		t.Fatalf("the focus stayed on field %d, not on the moved row", e.obj.focus)
	}
	_ = e.Key("carddown")
	_ = e.Key("carddown")
	if got := actionIDs(e); !slices.Equal(got, []uint32{3000, 3002, 3001}) {
		t.Fatalf("Alt+Down twice left the actions %v", got)
	}
	if !e.obj.messageChanged {
		t.Fatal("a moved row left the dialogue unchanged")
	}
	pressReplyRow(t, e, "c0.item", actRemoveRow)
	pressReplyRow(t, e, "", actAddAct)
	conds, acts := 0, 0
	for _, r := range e.openReply().rows {
		if r.condition {
			conds++
		} else {
			acts++
		}
	}
	if conds != 1 || acts != 4 {
		t.Fatalf("after a remove and an add the reply holds %d conditions and %d actions", conds, acts)
	}

	most := scriptprofile.Extended
	e = replyListEditor(t, most.Pairs, most.Actions)
	for _, c := range []struct {
		act, word string
		limit     int
	}{
		{actAddPair, "editor.wizard.full.conditions", most.Pairs},
		{actAddAct, "editor.wizard.full.actions", most.Actions},
	} {
		pressReplyRow(t, e, "", c.act)
		if want := fmt.Sprintf(e.word(c.word), c.limit); e.obj.note != want {
			t.Errorf("a row past the limit noted %q, want %q", e.obj.note, want)
		}
	}
	if n := len(e.openReply().rows); n != most.Pairs+most.Actions {
		t.Fatalf("the full reply holds %d rows", n)
	}
}

// A full reply's rows pass the window's height at full HD: the window stays
// on the screen and its right column scrolls alone, by the wheel and by the
// focus, to its last row.
func TestDialogReplyRowsScrollInsideFullHD(t *testing.T) {
	most := scriptprofile.Extended
	e := replyListEditor(t, most.Pairs, most.Actions)
	s, _, _ := e.objectSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	if !l.Rect.In(image.Rect(0, 0, 1920, 1080)) {
		t.Fatalf("the window stands at %v", l.Rect)
	}
	last := len(s.Rows) - 1
	shownRow := func(row int) bool {
		s, _, _ := e.objectSpec()
		for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
			if rr.Index == row {
				return true
			}
		}
		return false
	}
	shown := func() bool { return shownRow(last) }
	if shown() {
		t.Fatal("the full reply fits the window; the test needs rows past its height")
	}
	var p image.Point
	for _, rr := range l.Rows {
		if s.Rows[rr.Index].Column == 1 {
			p = rr.Rect.Min.Add(image.Pt(2, 2))
		}
	}
	a, ok := e.scrollAreaAt(popObject, p)
	if !ok {
		t.Fatal("the right column takes no wheel")
	}
	list := e.objectTop
	e.scrollWheelRows(a, 1<<20)
	if !shown() || e.objectTop != list {
		t.Fatalf("the wheel left the last row hidden (top %d) or moved the list (%d to %d)", e.obj.fieldsTop, list, e.objectTop)
	}
	e.obj.fieldsTop = 0
	key := fmt.Sprintf("a%d.item", most.Actions-1)
	e.focusObjectField(e.dialogFieldAt(key))
	if e.obj.fieldsTop == 0 || !shownRow(replyRowOf(e, key)) {
		t.Fatalf("the focus on the last action left the column at %d", e.obj.fieldsTop)
	}
}
