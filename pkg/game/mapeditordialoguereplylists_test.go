package game

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// millFile is the committed mill mod's file rel.
func millFile(t *testing.T, rel string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(millModFolder, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The mill's thanks node offers one reply that tests nine conditions and
// runs seventeen actions. Picked, its actions are queued in the file's
// order and run: the first sets variable 10 to 1, fifteen pay gold and the
// last sets variable 10 back to 0. With one of the nine conditions failing
// the reply is not offered.
func TestMillLongReplyRunsItsActionsInOrder(t *testing.T) {
	for _, c := range []struct {
		name string
		fail bool
	}{{"all hold", false}, {"one fails", true}} {
		t.Run(c.name, func(t *testing.T) {
			data := millFile(t, millDialogueFile)
			if c.fail {
				data = bytes.Replace(data, []byte("{ left = 3, right = 3, cmp = 0 }]"), []byte("{ left = 3, right = 3, cmp = 1 }]"), 1)
			}
			f := extendedScriptFront(t, millFile(t, "maps/mill.almx"))
			f.Archives.Containers.Overlay(dialogueFilePath(10, "mill"), func() ([]byte, error) { return data, nil })
			w := openExtendedMission(t, f, 40)
			mw := f.live
			if mw.mission.dialogue == nil {
				t.Fatal("the mill dialogue did not open")
			}
			mw.advanceNotice(ui.NoticeReply1)
			missionSteps(mw, 1)
			if c.fail {
				if len(mw.mission.replies) != 0 {
					t.Fatalf("a reply with a failing condition is offered: %+v", mw.mission.replies)
				}
				return
			}
			node, _ := mw.mission.dialogue.file.Node(2)
			if len(mw.mission.replies) != 1 || len(node.Replies) != 1 || len(node.Replies[0].When) != 9 || len(node.Replies[0].Do) != 17 {
				t.Fatalf("the thanks node offers %+v", mw.mission.replies)
			}
			var want []int32
			for _, id := range node.Replies[0].Do {
				want = append(want, mw.mission.state.ActionOf[uint32(id)])
			}
			gold := w.Purse(1)
			mw.advanceNotice(ui.NoticeReply1)
			var got []int32
			for _, cmd := range mw.pending {
				if cmd.Kind == sim.KindRunInstant {
					got = append(got, cmd.X)
				}
			}
			if !slices.Equal(got, want) {
				t.Fatalf("the reply queued instants %v, want %v", got, want)
			}
			missionSteps(mw, 1)
			if v := w.ScriptRegister(10); v != 0 {
				t.Fatalf("the last action did not run last: variable 10 is %d", v)
			}
			if w.Purse(1) <= gold {
				t.Fatalf("the gold actions paid nothing: %d then %d", gold, w.Purse(1))
			}
		})
	}
}

// The committed mill, its long reply included, opens in the Dialog window,
// and Apply and Save mod with no edit write the very same file.
func TestMillLongReplyRoundTripsThroughSaveMod(t *testing.T) {
	_, e, own := millMod(t, "")
	s := formHost(e)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	want := millFile(t, millDialogueFile)
	if err := s.prepareWorkspace(); err != nil || s.workspace == nil {
		t.Fatal("the mod has no workspace", err)
	}
	if s.workspace.Dialogues == nil {
		s.workspace.Dialogues = map[string][]byte{}
	}
	s.workspace.Dialogues["mill"] = want
	if err := e.OpenDialogue("mill"); err != nil {
		t.Fatal(err)
	}
	dlgApply(t, e)
	dlgCancel(t, e)
	_ = e.Key("escape")
	saveMod(t, e)
	if got := tree(t, own)[millDialogueFile]; got != string(want) {
		t.Fatalf("Save mod wrote the mill:\n%s\nwant:\n%s", got, want)
	}
}
