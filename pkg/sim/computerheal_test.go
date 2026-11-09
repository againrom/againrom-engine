package sim

import "testing"

const (
	healCasterOwner  uint32 = 7
	healNeutralOwner uint32 = 8
	healAlliedOwner  uint32 = 9
	healHostileOwner uint32 = 10
	healBothOwner    uint32 = 11
)

// computerHealWorld is a mage of owner 7, id 1, among others, under the chosen
// game profile. Its control word decides whether the owner is a computer.
func computerHealWorld(t *testing.T, second bool, control uint32, healHostile bool, others ...Entity) (*World, uint16) {
	t.Helper()
	heal := hlHeal()
	heal.ManaCost, heal.HealHostile = 3, healHostile
	if second {
		heal.ID = 24
		heal = secondRow(heal)
	}
	mage := spMage(1, 2, 2, 30, 60, 60, 1<<heal.ID)
	mage.Owner, mage.Book.State = healCasterOwner, BookPresent
	mage.Book.Slots[heal.ID-1] = BookSpell{Range: 6, Defensive: 1, ManaCost: 3}
	var script *Script
	if second {
		var err error
		if script, err = newScript(ScriptROM2, nil, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	const hostileAndAllied = relationHostile | relationLocked
	// A healthy unit of the human Player keeps the second game's activity mask
	// over the mage's cell.
	companion := spEnt(99, 6, 3)
	companion.Owner = SelfSlot
	w, err := NewStockedSpelledWorld(31, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, append(append([]Entity{mage}, others...), companion), script, engRel(t,
		[3]uint32{healCasterOwner, healAlliedOwner, relationLocked},
		[3]uint32{healAlliedOwner, healCasterOwner, relationLocked},
		[3]uint32{healCasterOwner, healHostileOwner, relationHostile},
		[3]uint32{healHostileOwner, healCasterOwner, relationHostile},
		[3]uint32{healCasterOwner, healBothOwner, hostileAndAllied},
		[3]uint32{healBothOwner, healCasterOwner, hostileAndAllied}), nil, nil, []SpellRule{heal})
	if err != nil {
		t.Fatal(err)
	}
	players := []SavedGroupPlayer{{ID: 91, Slot: healCasterOwner}, {ID: 92, Slot: SelfSlot}}
	controls := []PlayerParticipant{{PlayerID: 91, Value: control}, {PlayerID: 92, Value: 0}}
	for i, e := range others {
		if e.Owner == healCasterOwner {
			continue
		}
		players = append(players, SavedGroupPlayer{ID: 100 + uint32(i), Slot: e.Owner})
		controls = append(controls, PlayerParticipant{PlayerID: 100 + uint32(i), Value: 1})
	}
	if err := w.RestoreCurrentPlayers(players, controls, true); err != nil {
		t.Fatal(err)
	}
	holdGroups(w)
	return w, heal.ID
}

func woundedOf(id EntityID, owner uint32, x int32) Entity {
	e := spEnt(id, x, 2)
	e.Owner, e.HP = owner, 40
	return e
}

// healTargets steps the world and counts the Heal casts of the mage by target.
func healTargets(w *World, heal uint16) map[EntityID]int {
	out := map[EntityID]int{}
	for range 200 {
		for _, event := range StepObserved(w, nil) {
			if event.Caster == 1 && event.Spell == heal {
				out[event.Target]++
			}
		}
	}
	return out
}

func TestComputerMageHealsItsOwnSideAndAllies(t *testing.T) {
	const computer, human = 1, 0
	neutral, ally, own, foe, both := woundedOf(2, healNeutralOwner, 4), woundedOf(3, healAlliedOwner, 5),
		woundedOf(4, healCasterOwner, 5), woundedOf(5, healHostileOwner, 4), woundedOf(6, healBothOwner, 4)
	for _, tc := range []struct {
		name        string
		second      bool
		control     uint32
		healHostile bool
		others      []Entity
		wantHealed  []EntityID
		wantSpared  []EntityID
	}{
		{name: "second game computer leaves a neutral", second: true, control: computer, others: []Entity{neutral}, wantSpared: []EntityID{2}},
		{name: "second game computer heals an ally", second: true, control: computer, others: []Entity{ally}, wantHealed: []EntityID{3}},
		{name: "second game computer heals its own owner", second: true, control: computer, others: []Entity{own}, wantHealed: []EntityID{4}},
		{name: "second game computer takes the ally beside the neutral", second: true, control: computer, others: []Entity{neutral, ally}, wantHealed: []EntityID{3}, wantSpared: []EntityID{2}},
		{name: "second game computer leaves a hostile on a row that heals hostiles", second: true, control: computer, healHostile: true, others: []Entity{foe}, wantSpared: []EntityID{5}},
		{name: "second game computer leaves a hostile that also carries the allied bit", second: true, control: computer, healHostile: true, others: []Entity{both}, wantSpared: []EntityID{6}},
		{name: "second game human keeps the neutral tier", second: true, control: human, others: []Entity{neutral}, wantHealed: []EntityID{2}},
		{name: "second game human keeps a hostile on a row that heals hostiles", second: true, control: human, healHostile: true, others: []Entity{foe}, wantHealed: []EntityID{5}},
		{name: "first game computer keeps the neutral tier", control: computer, others: []Entity{neutral}, wantHealed: []EntityID{2}},
		{name: "first game computer keeps a hostile on a row that heals hostiles", control: computer, healHostile: true, others: []Entity{foe}, wantHealed: []EntityID{5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, heal := computerHealWorld(t, tc.second, tc.control, tc.healHostile, tc.others...)
			got := healTargets(w, heal)
			for _, id := range tc.wantHealed {
				if got[id] == 0 {
					t.Errorf("target %d took no Heal: %v", id, got)
				}
			}
			for _, id := range tc.wantSpared {
				if got[id] != 0 {
					t.Errorf("target %d took %d Heal casts: %v", id, got[id], got)
				}
			}
		})
	}
}

func TestComputerMageHealsItself(t *testing.T) {
	w, heal := computerHealWorld(t, true, 1, false)
	w.entities[0].HP = 40
	if got := healTargets(w, heal); got[1] == 0 {
		t.Fatalf("the mage took no Heal: %v", got)
	}
}

// An escort subject is the one target the escort heal asks about, and a script
// can name a unit of any owner as the subject.
func TestEscortHealFollowsTheComputerHealRule(t *testing.T) {
	for _, tc := range []struct {
		name    string
		second  bool
		control uint32
		owner   uint32
		want    bool
	}{
		{"second game computer leaves a neutral subject", true, 1, healNeutralOwner, false},
		{"second game computer heals an allied subject", true, 1, healAlliedOwner, true},
		{"second game computer heals a subject of its owner", true, 1, healCasterOwner, true},
		{"second game human keeps a neutral subject", true, 0, healNeutralOwner, true},
		{"first game computer keeps a neutral subject", false, 1, healNeutralOwner, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, _ := computerHealWorld(t, tc.second, tc.control, false, woundedOf(2, tc.owner, 4))
			if got := w.escortHeal(0, 1); got != tc.want {
				t.Fatalf("escort heal = %v, want %v", got, tc.want)
			}
		})
	}
}
