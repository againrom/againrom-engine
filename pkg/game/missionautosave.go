package game

import (
	"fmt"
	"time"

	"againrom/pkg/ui"
)

// autosaveMissionStart captures the mission at tick zero on the frame thread
// and queues the export and the write.
func (f *FrontEnd) autosaveMissionStart(viewer *ui.Viewer, store SaveStore, original OriginalStore, q *autosaveQueue) {
	if store.Dir == "" || f.live == nil || f.live.view != viewer || f.liveMission <= 0 ||
		f.live.mission == nil || f.live.mission.resumed || f.live.world.Tick() != 0 {
		return
	}
	captured, _, err := f.Snapshot(true)
	if err != nil {
		viewer.PostMessage("cannot autosave mission start: "+err.Error(), ui.MessageWhite, 15*time.Second)
		return
	}
	view, captured := f.detachedExporter(captured)
	label := fmt.Sprintf("autosave start mission %d", f.liveMission)
	inline := f.runsAutosaveInline()
	q.submit(inline, false, "cannot autosave mission start: ", func() autosaveResult {
		raw, _, err := view.playerMissionSave(captured, label)
		if err == nil {
			_, err = store.WriteOriginal(original.Dir, raw)
		}
		if err != nil {
			return autosaveResult{err: err, message: "cannot autosave mission start: " + err.Error()}
		}
		return autosaveResult{}
	})
	if inline {
		for _, r := range q.take(false) {
			if r.message != "" {
				viewer.PostMessage(r.message, ui.MessageWhite, 15*time.Second)
			}
		}
	}
}
