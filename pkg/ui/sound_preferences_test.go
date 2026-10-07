package ui

import (
	"errors"
	"strings"
	"testing"
)

func TestSoundPreferenceFailureKeepsMenuUsableAndReportsIt(t *testing.T) {
	f := openMissionMenu(t)
	enabled, volume := true, 50
	f.menuSound = func() (bool, int, bool) { return enabled, volume, true }
	f.setMenuSound = func(on bool, level int) error {
		return errors.New("read-only preferences")
	}
	f.rebuildGameMenu(gameMenuSoundOptionsPage, 0)
	if !f.chooseGameMenuAccelerator('d') || volume != 50 || f.screen != ScreenGameMenu {
		t.Fatal("failed save changed live settings or lost menu", volume, f.screen)
	}
	if !strings.Contains(f.msg, "Sound settings not saved") || !strings.Contains(f.msg, "read-only preferences") {
		t.Fatal("write failure was hidden", f.msg)
	}
	f.setMenuSound = func(on bool, level int) error { enabled, volume = on, level; return nil }
	if !f.chooseGameMenuAccelerator('u') || volume != 75 || f.msg != "" {
		t.Fatal("successful retry failed or retained error", volume, f.msg)
	}
}
