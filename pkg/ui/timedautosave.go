package ui

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	"againrom/pkg/render/text"
)

//go:embed timedautosave_ru.txt
var timedAutosaveRU string

type TimedAutosaveSettings struct {
	Enabled bool
	Minutes int
}

const MaxAutosaveMinutes = int((1<<63 - 1) / int64(time.Minute))

type TimedAutosaveControls struct {
	Read  func() TimedAutosaveSettings
	Write func(TimedAutosaveSettings) error
	Reset func()
	Poll  func(viewer *Viewer, onMap, ready bool) error
	// Notices returns the failures of automatic saves that finished since the
	// last call, as full messages. It runs on the frame thread.
	Notices func() []string
}

// SetBackgroundFlush installs the function FlushBackground calls.
func (a *App) SetBackgroundFlush(flush func()) {
	if a != nil {
		a.backgroundFlush = flush
	}
}

// FlushBackground blocks until work the front end started off the frame thread
// has finished. Exit calls it so an automatic save in flight reaches the disk.
func (a *App) FlushBackground() {
	if a != nil && a.backgroundFlush != nil {
		a.backgroundFlush()
	}
}

func (a *App) SetTimedAutosaveControls(c TimedAutosaveControls) {
	if a != nil && a.flow != nil {
		a.flow.gameOptions.Autosave = c
		a.flow.resetTimedAutosave()
	}
}

func (f *flow) resetTimedAutosave() {
	if reset := f.gameOptions.Autosave.Reset; reset != nil {
		reset()
	}
}

func (a *App) pollTimedAutosave(exit, unfocused bool) {
	f := a.flow
	if notices := f.gameOptions.Autosave.Notices; notices != nil {
		for _, message := range notices() {
			a.postAutosaveMessage(message)
		}
	}
	if f.gameOptions.Autosave.Poll == nil {
		return
	}
	ready := !exit && !unfocused && a.cutscene == nil && !a.cutsceneDrain
	_, dialogue := a.currentDialoguePointerOwner()
	ready = ready && !dialogue
	switch f.screen {
	case ScreenMap:
		ready = ready && f.viewer != nil && !f.viewer.NoticeOpen()
	case ScreenTown:
		ready = ready && f.town != nil && townCanSave(f.town)
	default:
		ready = false
	}
	if err := f.gameOptions.Autosave.Poll(f.viewer, f.screen == ScreenMap, ready); err != nil {
		a.postAutosaveMessage("Timed autosave failed: " + err.Error())
	}
}

func (a *App) postAutosaveMessage(message string) {
	f := a.flow
	if f.screen == ScreenMap && f.viewer != nil {
		f.viewer.PostMessage(message, MessageWhite, 15*time.Second)
	} else {
		f.msg = message
	}
}

func (f *flow) timedAutosaveLabels(d *gameOptionsDraft) (string, string) {
	toggle, interval := "Timed autosave", "< Minutes: %d >"
	if f.menuSelector() == text.SelectorConverting {
		words := strings.Split(strings.TrimSpace(timedAutosaveRU), "\n")
		toggle = f.menuDisplayText(words[0])
		interval = f.menuDisplayText(words[1])
	}
	return toggle, fmt.Sprintf(interval, d.autosave.Minutes)
}
