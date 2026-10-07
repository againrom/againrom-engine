package game

import "againrom/pkg/sim"

// publishSessionEntry clears reached Item flags (SAV-1114, SAV-POSTLOAD-222),
// queues carried Items for display, and updates the loaded document.
func publishSessionEntry(ms *Mission) {
	cleared, carried := ms.World.PublishSessionEntryWithCarried(sim.SelfSlot)
	if cleared == 0 {
		return
	}
	registry := ms.World.SavedObjects()
	for _, id := range carried {
		if row, ok := registry.Item(id); ok {
			ms.pendingPickups = append(ms.pendingPickups, row.Value)
		}
	}
	if ms.savedDocument == nil || ms.savedDocument.Objects == nil || ms.savedDocument.Document == nil {
		return
	}
	for _, binding := range ms.savedDocument.Objects.Items {
		row, ok := registry.Item(binding.ID)
		if !ok || binding.ObjectIndex == 0 || int(binding.ObjectIndex) > len(ms.savedDocument.Document.Objects) {
			continue
		}
		record := &ms.savedDocument.Document.Objects[binding.ObjectIndex-1]
		if v, err := savedStructureValue(record, "T08"); err == nil && v != row.Token.T08 {
			savedObjectSetValue(record, "T08", row.Token.T08)
		}
	}
}
