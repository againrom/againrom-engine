package sim

import "testing"

func diaryHero(id EntityID) Entity {
	e := spEnt(id, 0, 0)
	e.Owner, e.TypeID = SelfSlot, HumanTypeID
	return e
}

// diaryBody is a creature credited to killer.
func diaryBody(id EntityID, typeID int32, killer EntityID) Entity {
	e := spEnt(id, 1, 0)
	e.Owner, e.TypeID, e.MapUnitID = 3, typeID, uint16(id)
	e.KillCreditSource, e.HasKillCredit = killer, true
	return e
}

// diaryKill takes the victim from alive to dead and runs the death processor.
func diaryKill(w *World, id EntityID) {
	i := indexOfEntity(w.entities, id)
	before := w.entities[i].HP
	w.entities[i].HP = -1
	w.processNewKillCredits(map[EntityID]int32{id: before})
}

func diaryWorld(t *testing.T, units map[uint16]DiaryUnit, ents ...Entity) *World {
	t.Helper()
	w := spWorld(t, 1, nil, ents...)
	w.SetDiaryUnits(119, units)
	return w
}

func TestDiaryCountsAHeroKillWhenTheVictimDies(t *testing.T) {
	w := diaryWorld(t, map[uint16]DiaryUnit{2: {Row: 70, Face: 2}}, diaryHero(1), diaryBody(2, 66, 1))
	diaryKill(w, 2)
	d, ok := w.playerDiary()
	if !ok || d.Length != 119 || len(d.Entries) != 1 {
		t.Fatalf("Player diary = %+v, %v", d, ok)
	}
	if e := d.Entries[0]; e.Index != 70 || e.Count != 1 || e.Remaining != 1023 {
		t.Fatalf("entry = %+v, want row 70 count 1 remaining 1023", e)
	}
}

func TestDiaryIgnoresKillsTheWriterDoesNotAdmit(t *testing.T) {
	cases := map[string]func(hero, body *Entity){
		"killer owned by another slot": func(h, b *Entity) { h.Owner = 2 },
		"killer outside the hero band": func(h, b *Entity) { h.TypeID = 10 },
		"killer felled":                func(h, b *Entity) { h.HP = -1 },
		"no credit":                    func(h, b *Entity) { b.HasKillCredit, b.KillCreditSource = false, 0 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			hero, body := diaryHero(1), diaryBody(2, 66, 1)
			mutate(&hero, &body)
			w := diaryWorld(t, map[uint16]DiaryUnit{2: {Row: 70, Face: 2}}, hero, body)
			diaryKill(w, 2)
			if _, ok := w.playerDiary(); ok {
				t.Fatal("the diary rose")
			}
		})
	}
}

func TestDiaryHumanoidVictimRowAbove63IsRejected(t *testing.T) {
	body := diaryBody(2, 66, 1)
	body.Humanoid = true
	w := diaryWorld(t, map[uint16]DiaryUnit{2: {Row: 70, Face: 2}}, diaryHero(1), body)
	diaryKill(w, 2)
	if _, ok := w.playerDiary(); ok {
		t.Fatal("a Humanoid row above 63 was counted")
	}
}

func TestDiaryCountCapsAtSeventeenAndRemainderFloorsAtZero(t *testing.T) {
	w := diaryWorld(t, nil, diaryHero(1))
	for range 20 {
		w.addDiaryKill(70)
	}
	d, _ := w.playerDiary()
	if e := d.Entries[0]; e.Count != 17 || e.Remaining != 1004 {
		t.Fatalf("entry = %+v, want count 17 remaining 1004", e)
	}
	w.savedDiaries[0].Entries[0].Remaining = 0
	w.addDiaryKill(70)
	if d, _ := w.playerDiary(); d.Entries[0].Remaining != 0 {
		t.Fatal("remainder went below zero")
	}
}

func TestKnowledgeLevelFollowsTheDiaryNibble(t *testing.T) {
	monster := spEnt(2, 1, 0)
	monster.Owner, monster.TypeID, monster.MapUnitID = 3, 66, 2
	w := diaryWorld(t, map[uint16]DiaryUnit{2: {Row: 70, Face: 2}}, diaryHero(1), monster)
	want := []int{0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 7, 7}
	for count, level := range want {
		for len(w.savedDiaries) == 0 || diaryCount(w.savedDiaries[0], 70) < uint32(count) {
			w.addDiaryKill(70)
		}
		if got := w.knowledgeLevelOf(2); got != level {
			t.Fatalf("count %d: level %d, want %d", count, got, level)
		}
	}
	if got := w.knowledgeLevelOf(1); got != KnowledgeFull {
		t.Fatalf("own unit level %d", got)
	}
}

func TestKnowledgeLevelIsZeroOutsideTheTable(t *testing.T) {
	for name, tc := range map[string]struct {
		typeID int32
		unit   DiaryUnit
	}{
		"human type":       {10, DiaryUnit{Row: 70, Face: 2}},
		"type past table":  {81, DiaryUnit{Row: 70, Face: 2}},
		"face zero":        {66, DiaryUnit{Row: 70, Face: 0}},
		"face past nibble": {66, DiaryUnit{Row: 70, Face: 5}},
		"row below 64":     {66, DiaryUnit{Row: 30, Face: 2}},
	} {
		t.Run(name, func(t *testing.T) {
			m := spEnt(2, 1, 0)
			m.Owner, m.TypeID, m.MapUnitID = 3, tc.typeID, 2
			w := diaryWorld(t, map[uint16]DiaryUnit{2: tc.unit}, m)
			for range 16 {
				w.addDiaryKill(int(tc.unit.Row))
			}
			if got := w.knowledgeLevelOf(2); got != 0 {
				t.Fatalf("level %d, want 0", got)
			}
		})
	}
}

func TestDiaryRoundTripsThroughTheByteForm(t *testing.T) {
	w := diaryWorld(t, nil, diaryHero(1))
	w.addDiaryKill(70)
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	back.SetDiaryUnits(119, nil)
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	if d, ok := back.playerDiary(); !ok || diaryCount(d, 70) != 1 || back.diary.rows != 119 {
		t.Fatalf("decoded diary = %+v, %v, rows %d", d, ok, back.diary.rows)
	}
}
