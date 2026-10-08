package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

const quickOwnerExtension = ".quick-owner"

type quickSaveOwner struct {
	Kind     string
	Slot     int
	Sequence uint64
	SHA256   string
}

func quickSaveBase(slot int) validatedSaveName {
	return validatedSaveName(fmt.Sprintf("quick-save-%d", slot+1))
}

func validateQuickSavePair(slot int, raw, owner []byte) (uint64, error) {
	var record quickSaveOwner
	if len(owner) > 1024 || json.Unmarshal(owner, &record) != nil || record.Kind != "againrom-quick-sav" || record.Slot != slot ||
		record.Sequence == 0 || record.SHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) {
		return 0, fmt.Errorf("quick slot %d has no matching ownership record", slot+1)
	}
	if _, err := sav.DecodeDocumentData(raw); err != nil {
		return 0, fmt.Errorf("quick slot %d is corrupt: %w", slot+1, err)
	}
	return record.Sequence, nil
}

type quickSaveSlot struct {
	free     bool
	sequence uint64
	raw      []byte
}

func readQuickSaveSlots(dir string) [3]quickSaveSlot {
	var slots [3]quickSaveSlot
	for slot := range slots {
		base := filepath.Join(dir, string(quickSaveBase(slot)))
		info, err := os.Lstat(base + ".sav")
		ownerInfo, ownerErr := os.Lstat(base + quickOwnerExtension)
		if os.IsNotExist(err) && os.IsNotExist(ownerErr) {
			slots[slot].free = true
			continue
		}
		if err != nil || ownerErr != nil || !info.Mode().IsRegular() || !ownerInfo.Mode().IsRegular() {
			continue
		}
		raw, err := ReadSaveFile(base + ".sav")
		owner, ownerErr := ReadSaveFile(base + quickOwnerExtension)
		if err != nil || ownerErr != nil {
			continue
		}
		if sequence, err := validateQuickSavePair(slot, raw, owner); err == nil {
			slots[slot] = quickSaveSlot{sequence: sequence, raw: raw}
		}
	}
	return slots
}

func quickSavePosition(slots [3]quickSaveSlot) (int, uint64, error) {
	chosen := -1
	var newest uint64
	for slot, value := range slots {
		if value.sequence > newest {
			newest = value.sequence
		}
		if value.free && (chosen < 0 || !slots[chosen].free) || value.sequence != 0 && (chosen < 0 || !slots[chosen].free && value.sequence < slots[chosen].sequence) {
			chosen = slot
		}
	}
	if newest == math.MaxUint64 {
		return 0, 0, fmt.Errorf("quicksave sequence exhausted")
	}
	if chosen < 0 {
		return 0, 0, fmt.Errorf("all three quicksave slots are protected")
	}
	return chosen, newest + 1, nil
}

func newestQuickSave(slots [3]quickSaveSlot) ([]byte, error) {
	chosen := -1
	for slot, value := range slots {
		if value.sequence != 0 && (chosen < 0 || value.sequence > slots[chosen].sequence) {
			chosen = slot
		}
	}
	if chosen < 0 {
		return nil, fmt.Errorf("no complete verified quicksave")
	}
	return slots[chosen].raw, nil
}

func quickDeleteCompanion(path string, raw []byte) (namedSaveTarget, bool) {
	base := strings.TrimSuffix(path, filepath.Ext(path))
	for slot := range 3 {
		if !sameSaveName(filepath.Base(base), string(quickSaveBase(slot))) {
			continue
		}
		target := namedSaveTarget{path: base + quickOwnerExtension}
		var err error
		target.before, err = os.Lstat(target.path)
		if err != nil || !target.before.Mode().IsRegular() {
			return namedSaveTarget{}, false
		}
		target.old, err = ReadSaveFile(target.path)
		if err != nil {
			return namedSaveTarget{}, false
		}
		if _, err := validateQuickSavePair(slot, raw, target.old); err != nil {
			return namedSaveTarget{}, false
		}
		target.oldHash = sha256.Sum256(target.old)
		return target, true
	}
	return namedSaveTarget{}, false
}

func writeQuickSave(store SaveStore, fences []string, slot int, sequence uint64, raw []byte, files namedSaveFileOps) error {
	owner, err := json.Marshal(quickSaveOwner{"againrom-quick-sav", slot, sequence, fmt.Sprintf("%x", sha256.Sum256(raw))})
	if err != nil {
		return err
	}
	prepared, err := prepareOwnedNamedSave(store.Dir, quickSaveBase(slot), []namedSavePayload{{".sav", raw}, {quickOwnerExtension, owner}}, fences, files,
		func(targets []namedSaveTarget) error {
			if targets[0].before == nil && targets[1].before == nil {
				return nil
			}
			if targets[0].before == nil || targets[1].before == nil {
				return fmt.Errorf("quick slot %d is occupied by an unowned file", slot+1)
			}
			previous, err := validateQuickSavePair(slot, targets[0].old, targets[1].old)
			if err != nil {
				return err
			}
			if previous >= sequence {
				return fmt.Errorf("quick slot changed after rotation selection")
			}
			return nil
		}, store.profile)
	if err != nil {
		return err
	}
	_, err = prepared.Commit(true)
	return err
}

func (f *FrontEnd) configureQuickSave(app *ui.App, current *SaveStore, fences []string, queue *autosaveQueue, observers ...func(string, Snapshot)) {
	app.SetQuickSaveControls(ui.QuickSaveControls{
		Save: func(onMap bool) error {
			queue.wait()
			if current.Dir == "" {
				return fmt.Errorf("no quicksave directory")
			}
			snapshot, _, err := f.Snapshot(onMap)
			if err != nil {
				return err
			}
			notifySaveCapture(observers, "f4", snapshot)
			slot, sequence, err := quickSavePosition(readQuickSaveSlots(current.Dir))
			if err != nil {
				return err
			}
			raw, _, err := f.playerMissionSave(snapshot, fmt.Sprintf("quicksave %d", slot+1))
			if err != nil {
				return err
			}
			return writeQuickSave(*current, fences, slot, sequence, raw, nil)
		},
		Load: func() (ui.MapOpener, bool, error) {
			queue.wait()
			raw, err := newestQuickSave(readQuickSaveSlots(current.Dir))
			if err != nil {
				return nil, false, err
			}
			return f.RestoreOriginal(raw)
		},
	})
}
