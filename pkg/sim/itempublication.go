package sim

// publishActorItems clears pending flags on reached Items (SAV-1114).
// The carried arm requires an owned persistent actor (SAV-POSTLOAD-222);
// TakeSack uses it without the equipment arm.
func (w *World) publishActorItems(i int, recipient uint32, equipment bool) (int, []SavedObjectID) {
	if w.savedObjects == nil || w.entities[i].Owner != recipient {
		return 0, nil
	}
	var reached []SavedObjectID
	if InPersistBand(w.entities[i].TypeID) && i < len(w.carried) {
		for _, st := range w.carried[i] {
			reached = append(reached, st.ObjectID)
		}
	}
	carriedEnd := len(reached)
	if equipment && i < len(w.equipment) {
		for _, item := range w.equipment[i] {
			reached = append(reached, item.ObjectID)
		}
	}
	var next *SavedObjects
	cleared := 0
	var announced []SavedObjectID
	for ordinal, object := range reached {
		if row := w.savedObjects.item(object); object == 0 || row == nil || row.Token.T08 == 0 {
			continue
		}
		if next == nil {
			next = w.savedObjects.Clone()
		}
		if row := next.item(object); row.Token.T08 != 0 {
			row.Token.T08 = 0
			row.Coverage.Unknown &^= SavedUnknownFlags
			cleared++
			if ordinal < carriedEnd {
				announced = append(announced, object)
			}
		}
	}
	if next != nil {
		w.savedObjects = next
	}
	return cleared, announced
}

// PublishSessionEntry is session entry's publication of every actor's Items
// to recipient, with both arms. It returns how many flags it cleared.
func (w *World) PublishSessionEntry(recipient uint32) int {
	cleared, _ := w.PublishSessionEntryWithCarried(recipient)
	return cleared
}

// PublishSessionEntryWithCarried also returns each flagged carried Item reached
// by the publication. Equipped Items still clear their flags without a line.
func (w *World) PublishSessionEntryWithCarried(recipient uint32) (int, []SavedObjectID) {
	cleared := 0
	var announced []SavedObjectID
	for i := range w.entities {
		n, carried := w.publishActorItems(i, recipient, true)
		cleared += n
		announced = append(announced, carried...)
	}
	return cleared, announced
}
