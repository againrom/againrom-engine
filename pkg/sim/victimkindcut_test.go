package sim

import (
	"slices"
	"testing"
)

// The second game's repeated-victim cut (R2-ENGINE-1156 step 4,
// R2-ENGINE-1157): the kill route counts the victim's kind after its award
// call, paid or not; an award for a counted kind pays 10X / (n + 10). Fixture:
// a fighter at Mind 24 and a creature of value 100, so a kill offers
// trunc(100 x 0.5) = 50 and the Mind scale makes it trunc(50 x 1.05) = 52.

// vkWorld is the hero (index 0) and one creature per row (index 1 on), each
// counted under its row of the units table.
func vkWorld(t *testing.T, d ScriptDialect, hero Entity, rows ...uint16) *World {
	t.Helper()
	ents := []Entity{hero}
	for i := range rows {
		v := awardVictim(3)
		v.ID, v.X = EntityID(2+i), int32(1+i)
		ents = append(ents, v)
	}
	w := cbWorld(t, 1, ents...)
	w.game = newGameProfile(d)
	for i, row := range rows {
		w.entities[1+i].SourceBinding = SourceBinding{Class: GeneratedUnitBinding, Identity: uint32(2 + i), TokenRow: row}
	}
	return w
}

// vkKill runs the kill route for the creature at ti with the hero's credit and
// returns what slot gained.
func vkKill(w *World, ti int, slot int) int32 {
	before := w.entities[0].SkillXP[slot]
	t := &w.entities[ti]
	t.KillCreditSource, t.HasKillCredit = w.entities[0].ID, true
	w.processKillCredit(ti)
	return w.entities[0].SkillXP[slot] - before
}

func vkFighter() Entity { return awardHero(true, 24, 1, chosenLevels(1)) }

func TestSecondGameKillsOfOneKindPayFullTwiceThenTheCut(t *testing.T) {
	// Calls 1 and 2 pay 52; calls 3, 4, 5 pay 520/11, 520/12, 520/13.
	w := vkWorld(t, ScriptROM2, vkFighter(), 10)
	for call, want := range []int32{52, 52, 47, 43, 40} {
		if got := vkKill(w, 1, 1); got != want {
			t.Errorf("call %d paid %d, want %d", call+1, got, want)
		}
	}
	l := w.entities[0].victimKinds
	if l.n != 1 || l.entries[0] != (victimKindEntry{kind: w.victimKind(w.entities[1]), count: 4}) {
		t.Errorf("list %+v, want one entry of count 4", l)
	}
}

func TestSecondGameTwoKindsAreCountedApart(t *testing.T) {
	w := vkWorld(t, ScriptROM2, vkFighter(), 10, 11)
	for call, c := range []struct {
		ti   int
		want int32
	}{{1, 52}, {1, 52}, {2, 52}, {2, 52}, {1, 47}, {2, 47}} {
		if got := vkKill(w, c.ti, 1); got != c.want {
			t.Errorf("call %d on creature %d paid %d, want %d", call+1, c.ti, got, c.want)
		}
	}
}

// A mission's placed actors carry no source binding: each is counted under
// the row its map placement resolved to, a row past 255 included, so two
// placement rows count apart. The Diary keeps reading rows up to 255.
func TestSecondGameUnboundVictimsOfTwoPlacementRowsCountApart(t *testing.T) {
	w := vkWorld(t, ScriptROM2, vkFighter(), 0, 0)
	for i, id := range []uint16{21, 22} {
		w.entities[1+i].SourceBinding = SourceBinding{}
		w.entities[1+i].MapUnitID = id
	}
	w.SetDiaryUnits(180, map[uint16]DiaryUnit{21: {Row: 10, Face: 1}, 22: {Row: 300, Face: 1}})
	for call, c := range []struct {
		ti   int
		want int32
	}{{1, 52}, {1, 52}, {2, 52}, {2, 52}, {1, 47}, {2, 47}} {
		if got := vkKill(w, c.ti, 1); got != c.want {
			t.Errorf("call %d on creature %d paid %d, want %d", call+1, c.ti, got, c.want)
		}
	}
	if got, want := w.entities[0].victimKinds.counts(), []VictimKindCount{{1<<16 | 10, 2}, {1<<16 | 300, 2}}; !slices.Equal(got, want) {
		t.Errorf("list %+v, want %+v", got, want)
	}
	if _, ok := w.diaryUnitOf(w.entities[2]); ok {
		t.Error("the Diary reads a row past 255")
	}
	if u, ok := w.diaryUnitOf(w.entities[1]); !ok || u.Row != 10 {
		t.Errorf("the Diary reads %+v %v for row 10", u, ok)
	}
}

// SAV's current values carry the list and give it back; a list of six
// entries or a kind held twice is refused.
func TestSecondGameVictimListRidesTheCurrentValues(t *testing.T) {
	w := vkWorld(t, ScriptROM2, vkFighter(), 10, 11)
	for _, ti := range []int{1, 1, 2} {
		vkKill(w, ti, 1)
	}
	v := w.entities[0].Values()
	if want := []VictimKindCount{{1<<16 | 10, 1}, {1<<16 | 11, 0}}; !slices.Equal(v.VictimKinds, want) {
		t.Fatalf("values carry %+v, want %+v", v.VictimKinds, want)
	}
	loaded := w.entities[0]
	loaded.victimKinds = victimKindList{}
	if err := loaded.restoreValues(v); err != nil || loaded.victimKinds != w.entities[0].victimKinds {
		t.Fatalf("restored %+v (%v), want %+v", loaded.victimKinds, err, w.entities[0].victimKinds)
	}
	if empty := w.entities[1].Values(); empty.VictimKinds != nil {
		t.Errorf("an empty list writes %+v", empty.VictimKinds)
	}
	for name, bad := range map[string][]VictimKindCount{
		"six entries": {{1, 0}, {2, 0}, {3, 0}, {4, 0}, {5, 0}, {6, 0}},
		"kind twice":  {{1, 0}, {1, 3}},
	} {
		v.VictimKinds = bad
		if err := loaded.restoreValues(v); err == nil {
			t.Errorf("%s: restored", name)
		}
	}
}

func TestSecondGameASixthKindDropsTheOldest(t *testing.T) {
	// Kind 1 is counted twice (count 1), kinds 2..5 once, kind 6 drops kind 1:
	// kind 1 then pays in full again, where a held count of 1 would cut it to 47.
	w := vkWorld(t, ScriptROM2, vkFighter(), 10, 11, 12, 13, 14, 15)
	for _, ti := range []int{1, 1, 2, 3, 4, 5, 6} {
		vkKill(w, ti, 1)
	}
	l := w.entities[0].victimKinds
	if l.n != maxVictimKinds || l.entries[0].kind != w.victimKind(w.entities[2]) || l.entries[4].kind != w.victimKind(w.entities[6]) {
		t.Fatalf("list %+v, want kinds 2..6", l)
	}
	if got := vkKill(w, 1, 1); got != 52 {
		t.Errorf("the dropped kind paid %d, want 52", got)
	}
	if l := w.entities[0].victimKinds; l.n != maxVictimKinds || l.entries[0].kind != w.victimKind(w.entities[3]) || l.entries[4] != (victimKindEntry{kind: w.victimKind(w.entities[1])}) {
		t.Errorf("list %+v, want kinds 3..6 and kind 1 at count 0", l)
	}
}

func TestSecondGameAKillCallThatPaysNothingStillCounts(t *testing.T) {
	t.Run("raised victim", func(t *testing.T) {
		w := vkWorld(t, ScriptROM2, vkFighter(), 10)
		w.entities[1].SuppressRewards = true
		if got := vkKill(w, 1, 1); got != 0 {
			t.Fatalf("a raised victim paid %d", got)
		}
		w.entities[1].SuppressRewards = false
		for call, want := range []int32{52, 47} {
			if got := vkKill(w, 1, 1); got != want {
				t.Errorf("call %d paid %d, want %d", call+2, got, want)
			}
		}
	})
	t.Run("mage with spell 0", func(t *testing.T) {
		w := vkWorld(t, ScriptROM2, awardHero(false, 24, 0, chosenLevels(1)), 10)
		w.entities[1].KillCreditSpell = 0
		before := w.entities[0].SkillXP
		vkKill(w, 1, 1)
		if w.entities[0].SkillXP != before {
			t.Fatalf("a mage's kill with spell 0 paid: %v", w.entities[0].SkillXP)
		}
		if l := w.entities[0].victimKinds; l.n != 1 || l.entries[0].count != 0 {
			t.Errorf("list %+v, want the kind at count 0", l)
		}
	})
}

func TestSecondGameBlowsAreCutButNotCounted(t *testing.T) {
	// A blow of D 50 on HPmax 100 offers 26, 27 at Mind 24. After two kill
	// calls (count 1) it pays 270/11 = 24 and leaves the count at 1.
	w := vkWorld(t, ScriptROM2, vkFighter(), 10)
	vkKill(w, 1, 1)
	vkKill(w, 1, 1)
	before := w.entities[0].SkillXP[1]
	w.payExperience(0, 1, 50, true)
	if got := w.entities[0].SkillXP[1] - before; got != 24 {
		t.Errorf("the blow paid %d, want 24", got)
	}
	if l := w.entities[0].victimKinds; l.n != 1 || l.entries[0].count != 1 {
		t.Errorf("list %+v, want count 1", l)
	}
}

func TestSecondGameCutPrecedesTheEighthAndTheCap(t *testing.T) {
	// V 1540 offers 770, 808 at Mind 24, into an off-main skill at level 0
	// (cap T(1) - T(0) = 100) with the kind at count 2, reached by three kill
	// calls on a raised victim, which pay nothing. The claim's order: cut
	// 8080/12 = 673, ceil(673/8) = 85, cap 100: 85. The cut after the eighth
	// would pay 1010/12 = 84; after the cap, 1000/12 = 83, then 11.
	var levels [skillSlots]int32
	levels[1] = 20
	w := vkWorld(t, ScriptROM2, awardHero(true, 24, 3, levels), 10)
	w.entities[1].XPValue = 1540
	w.entities[1].SuppressRewards = true
	for call := 0; call < 3; call++ {
		vkKill(w, 1, 3)
	}
	w.entities[1].SuppressRewards = false
	if l := w.entities[0].victimKinds; l.n != 1 || l.entries[0].count != 2 || w.entities[0].SkillXP[3] != 0 {
		t.Fatalf("list %+v experience %d, want count 2 and nothing paid", l, w.entities[0].SkillXP[3])
	}
	if got := vkKill(w, 1, 3); got != 85 {
		t.Errorf("paid %d, want 85", got)
	}
}

func TestFirstGameKillsOfOneKindAreNotCut(t *testing.T) {
	// The same five calls under the first game pay 52 each and keep no list,
	// so its byte form carries no victim kind section.
	w := vkWorld(t, ScriptROM1, vkFighter(), 10)
	for call := 0; call < 5; call++ {
		if got := vkKill(w, 1, 1); got != 52 {
			t.Errorf("call %d paid %d, want 52", call+1, got)
		}
	}
	if l := w.entities[0].victimKinds; l.n != 0 {
		t.Errorf("the first game kept a list %+v", l)
	}
	if b := w.encodeInto(nil); b[0] == victimKindFormVersion {
		t.Error("the first game's form carries a victim kind section")
	}
}

func TestVictimKindListsRideTheirOwnFormSection(t *testing.T) {
	w := cbWorld(t, 1, awardHero(true, 24, 1, chosenLevels(1)), awardVictim(3))
	plain, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if plain[0] == victimKindFormVersion {
		t.Fatal("a World with no list wrote the victim kind form")
	}
	w.entities[1].victimKinds.count(7, maxVictimKinds)
	w.entities[1].victimKinds.count(1<<16|9, maxVictimKinds)
	w.entities[1].victimKinds.count(7, maxVictimKinds)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if form[0] != victimKindFormVersion {
		t.Fatalf("form version %d, want %d", form[0], victimKindFormVersion)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if back.entities[1].victimKinds != w.entities[1].victimKinds || back.entities[0].victimKinds.n != 0 {
		t.Errorf("decoded lists %+v and %+v", back.entities[0].victimKinds, back.entities[1].victimKinds)
	}
	if back.Hash() != w.Hash() {
		t.Error("the decoded World hashes differently")
	}
	// A list naming one kind twice is no form this build writes.
	w.entities[1].victimKinds.entries[1].kind = 7
	bad, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if err := new(World).UnmarshalBinary(bad); err == nil {
		t.Error("a list naming one kind twice decoded")
	}
}
