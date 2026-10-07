package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"againrom/pkg/formats/sav"
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

type timedSaveOwner struct {
	Kind     string
	Slot     int
	Sequence uint64
	SHA256   string
}

func timedSaveBase(slot int) validatedSaveName {
	return validatedSaveName(fmt.Sprintf("timed-autosave-%d", slot+1))
}

func validateTimedSavePair(slot int, raw, owner []byte) (uint64, error) {
	var record timedSaveOwner
	if len(owner) > 1024 || json.Unmarshal(owner, &record) != nil || record.Kind != "againrom-timed-sav" || record.Slot != slot ||
		record.Sequence == 0 || record.SHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) {
		return 0, fmt.Errorf("timed slot %d has no matching ownership record", slot+1)
	}
	if _, err := sav.DecodeDocumentData(raw); err != nil {
		return 0, fmt.Errorf("timed slot %d is corrupt: %w", slot+1, err)
	}
	return record.Sequence, nil
}

func timedSavePosition(dir string) (int, uint64, error) {
	var newest uint64
	next := 0
	var writable [3]bool
	for slot := range 3 {
		base := filepath.Join(dir, string(timedSaveBase(slot)))
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
		sequence, err := validateTimedSavePair(slot, raw, owner)
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

func timedDeleteCompanion(path string, raw []byte) (namedSaveTarget, bool) {
	base := strings.TrimSuffix(path, filepath.Ext(path))
	for slot := range 3 {
		if !sameSaveName(filepath.Base(base), string(timedSaveBase(slot))) {
			continue
		}
		target := namedSaveTarget{path: base + timedOwnerExtension}
		var err error
		target.before, err = os.Lstat(target.path)
		if err != nil || !target.before.Mode().IsRegular() {
			return namedSaveTarget{}, false
		}
		target.old, err = ReadSaveFile(target.path)
		if err != nil {
			return namedSaveTarget{}, false
		}
		if _, err := validateTimedSavePair(slot, raw, target.old); err != nil {
			return namedSaveTarget{}, false
		}
		target.oldHash = sha256.Sum256(target.old)
		return target, true
	}
	return namedSaveTarget{}, false
}

func writeTimedSave(store SaveStore, fences []string, slot int, sequence uint64, raw []byte, files namedSaveFileOps) error {
	owner, err := json.Marshal(timedSaveOwner{"againrom-timed-sav", slot, sequence, fmt.Sprintf("%x", sha256.Sum256(raw))})
	if err != nil {
		return err
	}
	prepared, err := prepareOwnedNamedSave(store.Dir, timedSaveBase(slot), []namedSavePayload{{".sav", raw}, {timedOwnerExtension, owner}}, fences, files,
		func(targets []namedSaveTarget) error {
			if targets[0].before == nil && targets[1].before == nil {
				return nil
			}
			if targets[0].before == nil || targets[1].before == nil {
				return fmt.Errorf("timed slot %d is occupied by an unowned file", slot+1)
			}
			previous, err := validateTimedSavePair(slot, targets[0].old, targets[1].old)
			if err != nil {
				return err
			}
			if previous >= sequence {
				return fmt.Errorf("timed slot changed after rotation selection")
			}
			return nil
		}, store.profile)
	if err != nil {
		return err
	}
	_, err = prepared.Commit(true)
	return err
}

func (f *FrontEnd) configureTimedAutosave(app *ui.App, current *SaveStore, fences []string, clock func() time.Time, q *autosaveQueue) {
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
			store := *current
			view, snapshot := f.detachedExporter(snapshot)
			q.submit(f.runsAutosaveInline(), true, "", func() autosaveResult {
				slot, sequence, err := timedSavePosition(store.Dir)
				if err != nil {
					return autosaveResult{timed: true, err: err}
				}
				raw, _, err := view.playerMissionSave(snapshot, fmt.Sprintf("timed autosave %d", slot+1))
				if err == nil {
					err = writeTimedSave(store, fences, slot, sequence, raw, nil)
				}
				return autosaveResult{timed: true, err: err}
			})
			return finished()
		},
	})
}
