package sim

import (
	"maps"
	"slices"
)

// The Player's Diary counts hero-typed kills per Units row. A creature card
// reads one level from it: min(count>>1, 7) of the victim's own row (UNIT-147).
// This file owns the count and the level; the saved Diary itself is the
// SavedDiary row the rest of the package already carries.

const (
	// DiaryCountCap is the largest count the original's writer stores.
	DiaryCountCap = 17
	// KnowledgeFull is the level at which every card group is drawn.
	KnowledgeFull = 7

	diaryFirstCreatureRow = 64
	diaryFirstCreatureTID = 64
	diaryLastCreatureTID  = 80
	diaryMaxFace          = 4
	diaryHumanRowLimit    = 63
	diaryDefaultRemaining = 1024
)

// DiaryUnit is the Units row and face a creature was built from. The row keys
// the Diary; the face selects the nibble the card reads.
type DiaryUnit struct {
	Row, Face uint8
}

// diaryRuntime is install-derived input to the Diary rules, carried across a
// decode like the rules and the ghost template: the Units collection length
// and the row each placed creature came from, keyed by the map unit id its
// placement carries (an entity id stops indexing its placement once a load
// has withdrawn some).
type diaryRuntime struct {
	rows  int
	units map[uint16]DiaryUnit
}

// SetDiaryUnits installs the Units collection length and the placement rows.
// A world with no rows records no kill.
func (w *World) SetDiaryUnits(rows int, units map[uint16]DiaryUnit) {
	w.diary = diaryRuntime{rows: rows, units: maps.Clone(units)}
}

// CopyDiaryUnits installs src's Diary input on w.
func (w *World) CopyDiaryUnits(src *World) {
	if src != nil {
		w.diary = diaryRuntime{rows: src.diary.rows, units: maps.Clone(src.diary.units)}
	}
}

func (w *World) playerDiary() (SavedDiary, bool) {
	for _, d := range w.savedDiaries {
		if d.Owner.Player {
			return d, true
		}
	}
	return SavedDiary{}, false
}

// diaryUnitOf is the row and face an entity's Diary scope names. A source
// binding states them for a restored actor; a placed creature states them
// through the installed placement rows.
func (w *World) diaryUnitOf(e Entity) (DiaryUnit, bool) {
	if e.SourceBinding.Class != 0 {
		return DiaryUnit{Row: e.SourceBinding.TokenRow, Face: e.SourceBinding.Face}, true
	}
	u, ok := w.diary.units[e.MapUnitID]
	return u, ok
}

func diaryCount(d SavedDiary, row int) uint32 {
	i, found := slices.BinarySearchFunc(d.Entries, row, func(e SavedDiaryEntry, r int) int { return e.Index - r })
	if !found {
		return 0
	}
	return d.Entries[i].Count
}

// KnowledgeLevel is the card level the local player holds for e: full for the
// local player's own units, otherwise the Diary nibble of a creature's row and
// face, and zero for every other unit (UNIT-146).

func (w *World) KnowledgeLevel(e Entity) int {
	if e.Owner == SelfSlot {
		return KnowledgeFull
	}
	if e.TypeID < diaryFirstCreatureTID || e.TypeID > diaryLastCreatureTID {
		return 0
	}
	u, ok := w.diaryUnitOf(e)
	if !ok || u.Face < 1 || u.Face > diaryMaxFace || int(u.Row) < diaryFirstCreatureRow {
		return 0
	}
	d, ok := w.playerDiary()
	if !ok {
		return 0
	}
	return min(int(diaryCount(d, int(u.Row))>>1), KnowledgeFull)
}

func (w *World) knowledgeLevelOf(id EntityID) int {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return 0
	}
	return w.KnowledgeLevel(w.entities[i])
}

// recordDiaryKill is the Diary writer, run when body ti has just died. A victim
// whose credited killer is a living hero-typed unit of the local player adds one
// to the local player's count for the victim's row. A Humanoid victim is
// admitted only for rows up to 63 (UNIT-147). It runs at the moment the
// experience award runs and takes no victim-owner gate.
func (w *World) recordDiaryKill(ti int) {
	victim := w.entities[ti]
	if w.diary.rows <= 0 || !victim.HasKillCredit {
		return
	}
	ci := indexOfEntity(w.entities, victim.KillCreditSource)
	if ci < 0 {
		return
	}
	killer := w.entities[ci]
	if !InPersistBand(killer.TypeID) || killer.HP < 0 || killer.Owner != SelfSlot {
		return
	}
	u, ok := w.diaryUnitOf(victim)
	row := int(u.Row)
	if !ok || row >= w.diary.rows || victim.Humanoid && row > diaryHumanRowLimit {
		return
	}
	w.addDiaryKill(row)
}

// addDiaryKill is the original's mutation on the local Player's Diary: the
// word remainder falls while nonzero and the count rises while it is at most
// 16.
func (w *World) addDiaryKill(row int) {
	at := slices.IndexFunc(w.savedDiaries, func(d SavedDiary) bool { return d.Owner.Player })
	if at < 0 {
		w.savedDiaries = append(cloneSavedDiaries(w.savedDiaries),
			SavedDiary{Owner: SavedDiaryOwner{Player: true}, Length: w.diary.rows})
		at = len(w.savedDiaries) - 1
	}
	d := &w.savedDiaries[at]
	if row >= d.Length {
		return
	}
	entries := slices.Clone(d.Entries)
	i, found := slices.BinarySearchFunc(entries, row, func(e SavedDiaryEntry, r int) int { return e.Index - r })
	if !found {
		entries = slices.Insert(entries, i, SavedDiaryEntry{Index: row, Remaining: diaryDefaultRemaining})
	}
	e := &entries[i]
	if e.Remaining != 0 {
		e.Remaining--
	}
	if e.Count <= DiaryCountCap-1 {
		e.Count++
	}
	d.Entries = entries
}
