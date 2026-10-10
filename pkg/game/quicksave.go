package game

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"againrom/pkg/ui"
)

const quickOwnerExtension = ".quick-owner"

type quickSaveSlot struct {
	free     bool
	sequence uint64
	raw      []byte
}

func readQuickSaveSlots(dir string) [3]quickSaveSlot {
	var slots [3]quickSaveSlot
	for slot := range slots {
		base := filepath.Join(dir, string(quickSlots.slotBase(slot)))
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
		if sequence, err := quickSlots.validatePair(slot, raw, owner); err == nil {
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
			return quickSlots.write(*current, fences, slot, sequence, raw, nil)
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
