package game

import (
	"fmt"
	"math"

	"againrom/pkg/ui"
)

func quickSavePosition(slots [3]ownedSlot) (int, uint64, error) {
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

func newestQuickSave(slots [3]ownedSlot) ([]byte, error) {
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
			var snapshot Snapshot
			err := f.captureSave(captureCommitted, onMap, func(s Snapshot, _ string) error {
				snapshot = s
				return nil
			})
			if err != nil {
				return err
			}
			notifySaveCapture(observers, "f4", snapshot)
			kind := quickSlots.forGame(f.savedGame())
			slot, sequence, err := quickSavePosition(kind.scan(current.Dir))
			if err != nil {
				return err
			}
			raw, _, err := f.playerMissionSave(snapshot, fmt.Sprintf("quicksave %d", slot+1))
			if err != nil {
				return err
			}
			return kind.write(*current, fences, slot, sequence, raw, nil)
		},
		Load: func() (ui.MapOpener, bool, error) {
			queue.wait()
			raw, err := newestQuickSave(quickSlots.forGame(f.savedGame()).scan(current.Dir))
			if err != nil {
				return nil, false, err
			}
			return f.RestoreOriginal(raw)
		},
	})
}
