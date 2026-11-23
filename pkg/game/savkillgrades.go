package game

import (
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The second game's local Player record holds its kill grades in the raw
// member sav.KillGradeBlock: 2,560 bytes at Player +0x44, one byte per
// creature serverID, byte 0 never a count and serverIDs above 0x9ff without
// a byte (R2-ENGINE-1272, R2-SESSION-417). The engine counts a kill under
// the creature's Units row, so a SAVE maps each Diary row to the row's
// serverID and a LOAD maps each serverID back through the Units table.

const (
	killGradeLen = 2560
	// killGradeLastID is the largest serverID the block holds a byte for.
	killGradeLastID = killGradeLen - 1
)

// localPlayerDiaryRecord is the Diary record of the document's local Player,
// the Player whose Participant word is 0; nil when the document has none.
func localPlayerDiaryRecord(doc *sav.DocumentData) (*sav.DocumentRecordData, error) {
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Player" {
			continue
		}
		participant, err := savedStructureValue(r, "Participant")
		if err != nil {
			return nil, err
		}
		if participant != 0 {
			continue
		}
		for j := range r.Inline {
			if r.Inline[j].Record.Class == "Diary" {
				return &r.Inline[j].Record, nil
			}
		}
	}
	return nil, nil
}

// diaryUntouchedRemaining is the word a Diary row holds before any kill; the
// remainder falls once per kill, so the kills a row holds are the fall.
const diaryUntouchedRemaining = 1024

// secondKillGrades is the kill-grade block of the local Player's Diary in
// doc, the document a SAVE has projected the current Diary into. A row
// without a serverID in 1..killGradeLastID has no byte. A nil block keeps
// the zero block a Player is built with.
func secondKillGrades(doc *sav.DocumentData, table *mapload.Table) ([]byte, error) {
	block, _, err := secondKillRecord(doc, table)
	return block, err
}

// secondKillRecord is the one reading of the local Player's Diary a SAVE
// writes from: the grade block, each byte capped by the Diary's count, and
// the owner's +0xa48 count, one per counted kill with no cap
// (R2-ENGINE-772). A victim whose class word, the Units row's type id, lies
// in 0x21..0x3f goes to +0xa4c, not +0xa48, so its row adds a grade byte and
// no kill to the count. A row's kills are the larger of its capped count and
// the fall of its remainder, which an original's file holds in the same
// pair.
func secondKillRecord(doc *sav.DocumentData, table *mapload.Table) ([]byte, uint32, error) {
	rec, err := localPlayerDiaryRecord(doc)
	if err != nil || rec == nil || table == nil || table.Units == nil {
		return nil, 0, err
	}
	d, err := sav.ReadDocumentDiary(*rec)
	if err != nil {
		return nil, 0, err
	}
	block := make([]byte, killGradeLen)
	var kills uint32
	for _, e := range d.Entries {
		if e.Index <= 0 || e.Index >= table.Units.Len() || e.Count == 0 {
			continue
		}
		params := table.Units.EntryParams(e.Index)
		if typeID, _, ok := data.UnitKeyPair(params); !ok || !sim.InPersistBand(typeID) {
			kills += max(e.Count, uint32(max(0, diaryUntouchedRemaining-int(e.Remaining))))
		}
		id, ok := data.UnitServerID(params)
		if !ok || id < 1 || id > killGradeLastID {
			continue
		}
		block[id] = max(block[id], byte(min(e.Count, 0xff)))
	}
	return block, kills, nil
}

// killGradeDiary is the local Player's Diary an original second-game file
// holds in its Player record's kill-grade block, by Units row; false for a
// record without the block or with an all-zero block.
func killGradeDiary(player *sav.Record, table *mapload.Table) (sim.SavedDiary, bool) {
	if player == nil || table == nil || table.Units == nil {
		return sim.SavedDiary{}, false
	}
	block := player.Raw[sav.KillGradeBlock]
	if len(block) != killGradeLen {
		return sim.SavedDiary{}, false
	}
	var entries []sim.SavedDiaryEntry
	for id := 1; id <= killGradeLastID; id++ {
		if block[id] == 0 {
			continue
		}
		row := data.FindUnitByServerID(table.Units, int32(id))
		if row <= 0 {
			continue
		}
		count := uint32(block[id])
		entries = append(entries, sim.SavedDiaryEntry{Index: row, Count: count, Remaining: uint16(max(0, 1024-int(count)))})
	}
	if len(entries) == 0 {
		return sim.SavedDiary{}, false
	}
	slices.SortFunc(entries, func(a, b sim.SavedDiaryEntry) int { return a.Index - b.Index })
	return sim.SavedDiary{Owner: sim.SavedDiaryOwner{Player: true}, Length: table.Units.Len(), Entries: entries}, true
}
