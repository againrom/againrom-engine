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

// DiaryUnit is the definition row and face a placed actor was built from. A
// row up to 255 keys the Diary and the face selects the nibble the card reads;
// every row keys the victim kind of an actor without a source binding.
type DiaryUnit struct {
	Row  uint16
	Face uint8
	// First is the first Units row sharing the placement row's type and face
	// when that is an earlier row; zero when the row is its own first. One
	// look is one count, so the Diary reads and writes First, not Row.
	First uint16
}

// countRow is the Units row the Diary counts the unit's look under.
func (u DiaryUnit) countRow() int {
	if u.First != 0 {
		return int(u.First)
	}
	return int(u.Row)
}

// diaryRuntime is install-derived input to the Diary rules, carried across a
// decode like the rules and the ghost template: the Units collection length
// and the row each placed creature came from, keyed by the map unit id its
// placement carries (an entity id stops indexing its placement once a load
// has withdrawn some).
type diaryRuntime struct {
	rows  int
	units map[uint16]DiaryUnit
	// first maps each Units row that repeats an earlier row's type and face
	// to that look's first row.
	first map[uint16]uint16
}

// SetDiaryUnits installs the Units collection length and the placement rows.
// A world with no rows records no kill.
func (w *World) SetDiaryUnits(rows int, units map[uint16]DiaryUnit) {
	w.diary = diaryRuntime{rows: rows, units: maps.Clone(units), first: w.diary.first}
}

// SetDiaryFold installs the first row of each repeated look.
func (w *World) SetDiaryFold(first map[uint16]uint16) {
	w.diary.first = maps.Clone(first)
}

// FoldDiaryLooks moves every count held on a repeated row of a look onto that
// look's first row, so a Diary written before one look was one count reads and
// records as one. Counts add to the cap and the spent remainders add. A table
// with no repeated row, the first game's, answers diaries unchanged.
func (w *World) FoldDiaryLooks(diaries []SavedDiary) []SavedDiary {
	if len(w.diary.first) == 0 {
		return diaries
	}
	out := cloneSavedDiaries(diaries)
	for i := range out {
		out[i].Entries = w.foldDiaryEntries(out[i].Entries)
	}
	return out
}

func (w *World) foldDiaryEntries(entries []SavedDiaryEntry) []SavedDiaryEntry {
	type tally struct{ count, spent int }
	targets := map[int]bool{}
	for _, e := range entries {
		if first, ok := w.diary.first[uint16(e.Index)]; ok && e.Index >= 0 && e.Index <= 0xffff {
			targets[int(first)] = true
		}
	}
	if len(targets) == 0 {
		return entries
	}
	var kept []SavedDiaryEntry
	sums := map[int]tally{}
	for _, e := range entries {
		to := e.Index
		if first, ok := w.diary.first[uint16(e.Index)]; ok && e.Index >= 0 && e.Index <= 0xffff {
			to = int(first)
		}
		if !targets[to] {
			kept = append(kept, e)
			continue
		}
		t := sums[to]
		t.count += int(e.Count)
		t.spent += max(0, diaryDefaultRemaining-int(e.Remaining))
		sums[to] = t
	}
	for to, t := range sums {
		kept = append(kept, SavedDiaryEntry{
			Index:     to,
			Count:     uint32(min(t.count, DiaryCountCap)),
			Remaining: uint16(max(0, diaryDefaultRemaining-t.spent)),
		})
	}
	slices.SortFunc(kept, func(a, b SavedDiaryEntry) int { return a.Index - b.Index })
	return kept
}

// DiaryRows is the installed Units collection length: a constructed Diary's
// element count.
func (w *World) DiaryRows() int { return w.diary.rows }

// CopyDiaryUnits installs src's Diary input on w.
func (w *World) CopyDiaryUnits(src *World) {
	if src != nil {
		w.diary = diaryRuntime{rows: src.diary.rows, units: maps.Clone(src.diary.units), first: maps.Clone(src.diary.first)}
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
		// A Diary row is one byte; the second game's records hold no Diary.
		return DiaryUnit{Row: uint16(uint8(e.SourceBinding.TokenRow)), Face: e.SourceBinding.Face}, true
	}
	u, ok := w.diary.units[e.MapUnitID]
	return u, ok && u.Row <= 255
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
	if e.TypeID < diaryFirstCreatureTID || e.TypeID > w.game.cardCreatures.types() {
		return 0
	}
	u, ok := w.diaryUnitOf(e)
	if !ok || u.Face < 1 || int32(u.Face) > w.game.cardCreatures.faces() || u.countRow() < diaryFirstCreatureRow {
		return 0
	}
	d, ok := w.playerDiary()
	if !ok {
		return 0
	}
	return min(int(diaryCount(d, u.countRow())>>1), KnowledgeFull)
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
	if !InPersistBand(killer.TypeID) || killer.Dead() || killer.Owner != SelfSlot {
		return
	}
	u, ok := w.diaryUnitOf(victim)
	row := u.countRow()
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
