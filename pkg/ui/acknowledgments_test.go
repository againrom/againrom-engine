package ui

import (
	"errors"
	"testing"
	"time"

	"againrom/pkg/audio"
)

func TestAcknowledgments1188ProductionGestureDeliversBeforeOneReply(t *testing.T) {
	a, v, _ := atOnMap(t)
	v.sel = selection{atLoID, atHiID}
	var delivered []uint32
	a.flow.order = func(id uint32, _, _ int) { delivered = append(delivered, id) }
	replies := 0
	v.SetCommandAcknowledgment(func(_ VoiceGesture, ids []uint32, _ time.Time) {
		replies++
		if len(delivered) != 2 || len(ids) != 2 || ids[0] != atLoID || ids[1] != atHiID {
			t.Fatal("response preceded command delivery or split the gesture", delivered, ids)
		}
	})
	atPress(a, v, atEmptyCol, atEmptyRow)
	if len(delivered) != 2 || replies != 1 {
		t.Fatal("missing command or response", delivered, replies)
	}
	a.flow.order = nil
	atPress(a, v, atEmptyCol, atEmptyRow)
	if replies != 1 {
		t.Fatal("voice without a command sink")
	}
	a.flow.order = func(id uint32, _, _ int) { delivered = append(delivered, id) }
	v.SetCommandAcknowledgment(nil)
	atPress(a, v, atEmptyCol, atEmptyRow)
	if len(delivered) != 4 {
		t.Fatal("missing voice callback suppressed commands")
	}
}

func TestAcknowledgments1188CheckboxAcceptanceCancelAndFailedWrite(t *testing.T) {
	a := NewApp("acknowledgments", appAssets(t), appRows(1), nil)
	a.Layout(640, 480)
	a.flow.viewer, a.flow.screen = fiViewer(t), ScreenMap
	a.flow.menuFont = gameMenuTestFont()
	on, fail, writes := true, false, 0
	a.SetGameMenuSettings(nil, nil, func() (bool, int, bool) { return true, 100, true }, nil)
	a.SetSoundOptionControls(SoundOptionControls{Read: audio.FullChannelVolumes, Words: DefaultSoundOptionWords(),
		ReadAcknowledgments: func() bool { return on }, WriteAcknowledgments: func(value bool) error {
			if fail {
				return errors.New("read-only profile")
			}
			on = value
			writes++
			return nil
		}})
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	open := func() {
		t.Helper()
		if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
			t.Fatal(err)
		}
	}
	toggle := func() {
		t.Helper()
		r := soundOptionRect(gameMenuAcknowledgments)
		p := r.Min.Add(r.Size().Div(2))
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	open()
	toggle()
	if !on || writes != 0 {
		t.Fatal("checkbox applied before OK")
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	open()
	if !a.flow.soundOptions.acknowledgments {
		t.Fatal("Escape retained staged change")
	}
	toggle()
	fail = true
	if err := a.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	if !on || a.flow.menuPage != gameMenuSoundOptionsPage || a.HeadlessMessage() == "" {
		t.Fatal("failed persistence left dialog or changed option")
	}
	fail = false
	if err := a.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	if on || writes != 1 || a.flow.menuPage != gameMenuRoot {
		t.Fatal("OK did not commit exactly once")
	}
}
