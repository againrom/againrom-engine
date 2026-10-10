package game

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"againrom/pkg/ui"
)

const timedOwnerExtension = ".timed-owner"

func (s OptionsStore) timedAutosave() (ui.TimedAutosaveSettings, error) {
	v := ui.TimedAutosaveSettings{Enabled: true, Minutes: 5}
	m, err := s.readAll()
	if err != nil {
		return v, err
	}
	if m["TimedAutosave"] == "0" {
		v.Enabled = false
	}
	if n, err := strconv.Atoi(m["TimedAutosaveMinutes"]); err == nil && n > 0 && n <= ui.MaxAutosaveMinutes {
		v.Minutes = n
	}
	return v, nil
}

func (s OptionsStore) setTimedAutosave(v ui.TimedAutosaveSettings) error {
	if v.Minutes < 1 || v.Minutes > ui.MaxAutosaveMinutes {
		return fmt.Errorf("autosave minutes outside valid duration")
	}
	m, err := s.readAll()
	if err != nil {
		return err
	}
	m["TimedAutosave"] = "0"
	if v.Enabled {
		m["TimedAutosave"] = "1"
	}
	m["TimedAutosaveMinutes"] = strconv.Itoa(v.Minutes)
	return s.writeAll(m)
}

func timedSavePosition(dir string) (int, uint64, error) {
	var newest uint64
	next := 0
	var writable [3]bool
	for slot := range 3 {
		base := filepath.Join(dir, string(timedSlots.slotBase(slot)))
		saveInfo, saveErr := os.Lstat(base + ".sav")
		ownerInfo, ownerStatErr := os.Lstat(base + timedOwnerExtension)
		if os.IsNotExist(saveErr) && os.IsNotExist(ownerStatErr) {
			writable[slot] = true
			continue
		}
		if saveErr != nil || ownerStatErr != nil || !saveInfo.Mode().IsRegular() || !ownerInfo.Mode().IsRegular() {
			continue
		}
		raw, err := ReadSaveFile(base + ".sav")
		owner, ownerErr := ReadSaveFile(base + timedOwnerExtension)
		if err != nil || ownerErr != nil {
			continue
		}
		sequence, err := timedSlots.validatePair(slot, raw, owner)
		if err == nil {
			writable[slot] = true
			if sequence > newest {
				newest, next = sequence, (slot+1)%3
			}
		}
	}
	if newest == math.MaxUint64 {
		return 0, 0, fmt.Errorf("timed autosave sequence exhausted")
	}
	for offset := range 3 {
		slot := (next + offset) % 3
		if writable[slot] {
			return slot, newest + 1, nil
		}
	}
	return 0, 0, fmt.Errorf("all three timed autosave slots are protected")
}

func (f *FrontEnd) configureTimedAutosave(app *ui.App, current *SaveStore, fences []string, clock func() time.Time, q *autosaveQueue, observers ...func(string, Snapshot)) {
	if clock == nil {
		clock = time.Now
	}
	settings, _ := f.Options.timedAutosave()
	var deadline time.Time
	reset := func() { deadline = clock().Add(time.Duration(settings.Minutes) * time.Minute) }
	// finished applies the outcome of every timed save that completed since the
	// last frame: a success restarts the interval, a failure is reported and
	// the one-minute retry set when the save began stands.
	finished := func() error {
		var failure error
		for _, r := range q.take(true) {
			if r.err != nil {
				failure = r.err
			} else {
				reset()
			}
		}
		return failure
	}
	app.SetTimedAutosaveControls(ui.TimedAutosaveControls{
		Read: func() ui.TimedAutosaveSettings { return settings },
		Write: func(v ui.TimedAutosaveSettings) error {
			if err := f.Options.setTimedAutosave(v); err != nil {
				return err
			}
			settings = v
			reset()
			return nil
		},
		Reset: reset,
		Notices: func() []string {
			var out []string
			for _, r := range q.take(false) {
				if r.message != "" {
					out = append(out, r.message)
				}
			}
			return out
		},
		Poll: func(viewer *ui.Viewer, onMap, ready bool) error {
			if err := finished(); err != nil {
				return err
			}
			now := clock()
			if !settings.Enabled || now.Before(deadline) || !ready || current.Dir == "" {
				return nil
			}
			if onMap && !f.showsViewer(viewer) {
				return nil
			}
			if q.busy() {
				return nil
			}
			deadline = now.Add(time.Minute)
			snapshot, _, err := f.Snapshot(onMap)
			if err != nil {
				return err
			}
			notifySaveCapture(observers, "timed", snapshot)
			store := *current
			view, snapshot := f.detachedExporter(snapshot)
			q.submit(f.runsAutosaveInline(), true, "", func() autosaveResult {
				slot, sequence, err := timedSavePosition(store.Dir)
				if err != nil {
					return autosaveResult{timed: true, err: err}
				}
				raw, _, err := view.playerMissionSave(snapshot, fmt.Sprintf("timed autosave %d", slot+1))
				if err == nil {
					err = timedSlots.write(store, fences, slot, sequence, raw, nil)
				}
				return autosaveResult{timed: true, err: err}
			})
			return finished()
		},
	})
}
