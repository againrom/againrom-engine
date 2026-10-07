package game

// Original LOAD supplies both counters directly. An older AGS projection that
// discarded +128 cannot establish an original campaign's full score history.
func fameFromTown(t *Town) SnapshotFame {
	if t != nil && t.progress != nil {
		return SnapshotFame{Known: t.progress.scoreEventsKnown, Time: t.progress.missionTime, Events: t.progress.scoreEvents}
	}
	return SnapshotFame{}
}

// Fame history completeness does not gate writing its observed counters.
func fameForOriginal(s Snapshot) (SnapshotFame, error) {
	return fameFromSnapshot(s), nil
}
