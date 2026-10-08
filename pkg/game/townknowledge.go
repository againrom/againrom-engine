package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// carryKnowledge keeps the local Player's Diary the mission w ended with, so
// the next mission entered from the town opens with it.
func (t *Town) carryKnowledge(w *sim.World) {
	if t == nil || w == nil {
		return
	}
	t.knowledge = nil
	for _, d := range w.SavedDiaries() {
		if d.Owner.Player {
			t.knowledge = []sim.SavedDiary{d}
		}
	}
}

// knowledgeDiaries is the Diary a fresh mission starts with: none before a
// mission has been won in this campaign.
func (t *Town) knowledgeDiaries() []sim.SavedDiary {
	if t == nil {
		return nil
	}
	return t.knowledge
}

// projectTownKnowledge writes the town's carried Diary into the local
// Player's Diary record of a city document, over the record's own Self. A
// town with no carried Diary leaves the record as the producer built it.
func projectTownKnowledge(doc *sav.DocumentData, knowledge []sim.SavedDiary) error {
	var carried *sim.SavedDiary
	for i := range knowledge {
		if knowledge[i].Owner.Player {
			carried = &knowledge[i]
		}
	}
	if carried == nil {
		return nil
	}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Player" {
			continue
		}
		participant, err := savedStructureValue(r, "Participant")
		if err != nil {
			return err
		}
		if participant != 0 {
			continue
		}
		for j := range r.Inline {
			if r.Inline[j].Record.Class != "Diary" {
				continue
			}
			next, err := sav.ProjectDocumentDiary(r.Inline[j].Record, sav.Diary{Length: carried.Length, Entries: savedDiaryEntriesToSav(carried.Entries)})
			if err != nil {
				return err
			}
			r.Inline[j].Record = next
		}
	}
	return nil
}

// savedPlayerDiary is the local Player's Diary a loaded city file carries, as
// the town's carried Diary. An all-default Diary carries no knowledge and leaves the town without one.
func savedPlayerDiary(player *sav.Record) []sim.SavedDiary {
	d, ok, err := sav.PlayerDiary(player)
	if err != nil || !ok || len(d.Entries) == 0 {
		return nil
	}
	return []sim.SavedDiary{{Owner: sim.SavedDiaryOwner{Player: true}, Length: d.Length, Entries: savDiaryEntriesToSaved(d.Entries)}}
}
