package mapload_test

import (
	"sort"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
)

// npcRegistry writes a scenario NPC registry carrying one section per entry of
// defs and parses it back, so the lookup under test only ever sees a tree it
// could have got from a real file. Nothing here reads an install.
func npcRegistry(t *testing.T, defs map[int32]int32) *reg.Reg {
	t.Helper()
	ids := make([]int, 0, len(defs))
	for id := range defs {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	nodes := make([]synth.RegNode, 0, len(ids))
	for _, id := range ids {
		nodes = append(nodes, synth.RegNode{
			Name: "npc" + strconv.Itoa(id), Kind: 0x01,
			Children: []synth.RegNode{{Name: "DataBinID", Kind: 0x02, Int: defs[int32(id)]}},
		})
	}
	r, err := reg.Parse(synth.Reg(0x11, nodes))
	if err != nil {
		t.Fatalf("reg.Parse: %v", err)
	}
	return r
}

// The slot numbers this suite writes keys into. They are spelled out here rather
// than imported, so a test asserting that a key column moved a placement is not
// reading that column's position out of the code it is testing.
const (
	slotHealthMax = 4
	slotHumanType = 0x10
	slotServerID  = 0x18
	slotUnitType  = 0x1d
	slotUnitFace  = 0x1e
	slotMovement  = 0x20
	rowWidth      = 38
)

// defEntry and defCollection are a definition collection built in test code:
// index 0 is the reserved empty entry, and every other row is written by slot.
type defEntry struct {
	name    string
	params  []int32
	strings []string
}

type defCollection []defEntry

func (c defCollection) Len() int                    { return len(c) }
func (c defCollection) EntryName(i int) string      { return c[i].name }
func (c defCollection) EntryParams(i int) []int32   { return c[i].params }
func (c defCollection) EntryStrings(i int) []string { return c[i].strings }

// defRow is a parameter row of the streamed width whose cells are all empty but
// the named slots — so every field of the definition it yields is the
// constructor's default except the ones a test set on purpose.
func defRow(slots map[int]int32) []int32 {
	p := make([]int32, rowWidth)
	for i := range p {
		p[i] = -1
	}
	for s, v := range slots {
		p[s] = v
	}
	return p
}

func unitDefRow(typeID, face, healthMax int32) []int32 {
	return defRow(map[int]int32{slotUnitType: typeID, slotUnitFace: face, slotHealthMax: healthMax})
}

// armTable is the table the arm test resolves against. Its two collections are
// keyed so that every arm reaches something distinguishable, and it carries the
// two shapes a search has to walk past: an empty-named entry sitting on a key,
// and a second entry repeating one.
//
// Its humans keys are all BELOW the class-key floor, because that is the
// only band from which a humans arm is now reachable at all.
func armTable(t *testing.T) *mapload.Table {
	return &mapload.Table{
		Units: defCollection{
			{}, // 0: reserved
			{name: "u-26", params: unitDefRow(26, 0, 30)},      // 1
			{name: "u-27", params: unitDefRow(27, 0, 30)},      // 2
			{name: "", params: unitDefRow(0x40, 1, 30)},        // 3: nameless, on the key
			{name: "u-40-f1", params: unitDefRow(0x40, 1, 30)}, // 4: what 0x140 must reach
			{name: "u-77-a", params: unitDefRow(0x77, 3, 30)},  // 5: the earlier duplicate
			{name: "u-77-b", params: unitDefRow(0x77, 3, 30)},  // 6
		},
		Humans: defCollection{
			{},
			{name: "h-9", params: defRow(map[int]int32{slotHumanType: 0x09, slotServerID: 900})},
			{name: "h-dup-early", params: defRow(map[int]int32{slotHumanType: 0x0a, slotServerID: 901})},
			{name: "h-dup-late", params: defRow(map[int]int32{slotHumanType: 0x0b, slotServerID: 901})},
		},
		NPC: armNPC(t),
	}
}

// armNPC is a scenario NPC lookup naming server id 900, so the npc arm reaches
// an entry a definition id would not: subscript 51 answers 900 → humans entry 1,
// while every case that carries one carries definition id 901 → humans entry 3.
// That is what makes "the npc arm never reads its own definition id" a
// measurable claim rather than a restatement.
func armNPC(t *testing.T) *data.NPCDefs {
	t.Helper()
	return data.LoadNPCDefs(npcRegistry(t, map[int32]int32{51: 900}))
}

// TestEveryArmIsTakenAndCounted asserts each outcome on the arm returned and on
// the entry it reached.
func TestEveryArmIsTakenAndCounted(t *testing.T) {
	tbl := armTable(t)
	for _, tc := range []struct {
		name  string
		unit  alm.Unit
		arm   mapload.Arm
		index int
	}{
		// The class key is the outermost rung: at or above the floor nothing
		// else is consulted, flag word and definition id included.
		{"the floor beats the npc flag", alm.Unit{Flags: 1, ClassID: 0x40, ClassSubID: 1, DefID: 901},
			mapload.ArmUnits, 4},
		{"the floor beats a definition id", alm.Unit{ClassID: 0x40, ClassSubID: 1, DefID: 901},
			mapload.ArmUnits, 4},
		// Inside the humans band the flag word's bit 0 wins over the id, and it
		// reaches the entry the NPC lookup names rather than the one the id does.
		{"the npc flag", alm.Unit{Flags: 1, ClassID: 0x09, ClassSubID: 51, DefID: 901}, mapload.ArmNPC, 1},
		{"an npc subscript naming nothing", alm.Unit{Flags: 1, ClassID: 0x09, ClassSubID: 7}, mapload.ArmNPC, 0},
		// A written definition id overrides the class key and searches DOWNWARD,
		// so it reaches the later of the two entries on id 901.
		{"a definition id", alm.Unit{ClassID: 0x09, DefID: 901}, mapload.ArmServerID, 3},
		{"a definition id naming nothing", alm.Unit{ClassID: 0x09, DefID: 55}, mapload.ArmServerID, 0},
		// The uninitialised fill is NOT an id: the placement falls through to
		// its class key, which here is a humans one.
		{"the sentinel id", alm.Unit{ClassID: 0x09, DefID: 0xcdcdcdcd}, mapload.ArmHumansByType, 1},
		{"a low key takes the humans arm", alm.Unit{ClassID: 0x09}, mapload.ArmHumansByType, 1},
		// The floor itself, and the key one above it.
		{"key 26 is the floor", alm.Unit{ClassID: 26}, mapload.ArmUnits, 1},
		{"key 27 is above it", alm.Unit{ClassID: 27}, mapload.ArmUnits, 2},
		// One below the floor is the humans band, and reaches nothing here.
		{"key 25 is below it", alm.Unit{ClassID: 25}, mapload.ArmHumansByType, 0},
		// The band compares the SIGNED word, so a negative key is in the humans
		// band however its low byte reads. Nothing shipped exercises this — the
		// domain is 1..80 — so the case pins a decision rather than a
		// measurement.
		{"a negative key is below the floor", alm.Unit{ClassID: -24}, mapload.ArmHumansByType, 0},
		// Truncation, and the nameless entry the search must walk past to get
		// there.
		{"key 0x40 skips the empty name", alm.Unit{ClassID: 0x40, ClassSubID: 1}, mapload.ArmUnits, 4},
		{"key 0x140 reaches what 0x40 reaches", alm.Unit{ClassID: 0x140, ClassSubID: 0x101}, mapload.ArmUnits, 4},
		// Two entries on one key: the earlier wins.
		{"the earlier duplicate wins", alm.Unit{ClassID: 0x77, ClassSubID: 3}, mapload.ArmUnits, 5},
		{"an unmatched key", alm.Unit{ClassID: 0x7e, ClassSubID: 9}, mapload.ArmUnits, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := mapload.Resolve(tc.unit, tbl)
			if got.Arm != tc.arm || got.Index != tc.index {
				t.Errorf("Resolve = {%v %d}, want {%v %d}", got.Arm, got.Index, tc.arm, tc.index)
			}
			if got.Found() != (tc.index != 0) {
				t.Errorf("Found() = %v for index %d", got.Found(), got.Index)
			}
		})
	}
}

// TestResolveWithNoTableStillNamesTheArm: a nil table is no entries, not a
// missing branch, so a placement is still answered with the arm it took.
func TestResolveWithNoTableStillNamesTheArm(t *testing.T) {
	for _, tc := range []struct {
		unit alm.Unit
		arm  mapload.Arm
	}{
		{alm.Unit{Flags: 1}, mapload.ArmNPC},
		{alm.Unit{DefID: 7}, mapload.ArmServerID},
		{alm.Unit{ClassID: 5}, mapload.ArmHumansByType},
		{alm.Unit{ClassID: 200}, mapload.ArmUnits},
	} {
		got := mapload.Resolve(tc.unit, nil)
		if got.Arm != tc.arm || got.Found() {
			t.Errorf("Resolve(%+v, nil) = {%v %d}, want {%v no match}", tc.unit, got.Arm, got.Index, tc.arm)
		}
	}
	// And a table holding one collection and not the other is the same story.
	half := &mapload.Table{Units: defCollection{{}, {name: "u", params: unitDefRow(200, 0, 30)}}}
	if got := mapload.Resolve(alm.Unit{ClassID: 5}, half); got.Arm != mapload.ArmHumansByType || got.Found() {
		t.Errorf("the humans arm over a table with no humans collection = {%v %d}", got.Arm, got.Index)
	}
	if got := mapload.Resolve(alm.Unit{ClassID: 200}, half); got.Arm != mapload.ArmUnits || got.Index != 1 {
		t.Errorf("the units arm over the same table = {%v %d}, want {units 1}", got.Arm, got.Index)
	}
}

// adjusted is the health maximum the setting must produce, WRITTEN OUT BY HAND
// over the five maxima AC-6 names. Computing these from the constants would
// assert the arithmetic against itself and would hold for any rounding the code
// happened to use — which is the whole question, since the engine's own form is
// two doubles and a truncation.
var adjusted = map[int32][3]int32{
	//         easy   normal   hard
	0:     {0, 0, 0},
	1:     {0, 1, 1},
	99:    {65, 99, 148},
	100:   {66, 100, 150},
	65535: {43253, 65535, 98302},
}

// TestTheAdjustmentTable is AC-6 and SC-6.
func TestTheAdjustmentTable(t *testing.T) {
	values := [3]mapload.Difficulty{mapload.DifficultyEasy, mapload.DifficultyNormal, mapload.DifficultyHard}
	for h, want := range adjusted {
		for i, diff := range values {
			in := data.UnitDefaults()
			in.HealthMax, in.Health = h, h
			in.ToHit, in.Defence = 10, 20

			got, err := mapload.Adjust(in, diff)
			if err != nil {
				t.Fatalf("Adjust(healthMax %d, %d): %v", h, int32(diff), err)
			}
			if got.HealthMax != want[i] {
				t.Errorf("healthMax %d at difficulty %d = %d, want %d", h, int32(diff), got.HealthMax, want[i])
			}
			if got.Health != got.HealthMax {
				t.Errorf("healthMax %d at difficulty %d left health at %d and the maximum at %d",
					h, int32(diff), got.Health, got.HealthMax)
			}

			wantToHit, wantDefence := int32(10), int32(20)
			if diff == mapload.DifficultyHard {
				wantToHit, wantDefence = 60, 70
			}
			if got.ToHit != wantToHit || got.Defence != wantDefence {
				t.Errorf("at difficulty %d to-hit/defence = %d/%d, want %d/%d",
					int32(diff), got.ToHit, got.Defence, wantToHit, wantDefence)
			}
			if diff == mapload.DifficultyNormal && got != in {
				t.Errorf("difficulty 2 changed the definition:\n got %+v\nwant %+v", got, in)
			}
		}
	}
}

// TestAFourthDifficultyIsRefused: the value set is exactly three.
func TestAFourthDifficultyIsRefused(t *testing.T) {
	for _, v := range []mapload.Difficulty{-1, 0, 4, 100} {
		got, err := mapload.Adjust(data.UnitDefaults(), v)
		if err == nil {
			t.Errorf("difficulty %d was accepted", int32(v))
		}
		if got != (data.UnitDef{}) {
			t.Errorf("a refused difficulty yielded %+v, want the zero value", got)
		}
	}
}

func TestTheAdjustmentIsPure(t *testing.T) {
	base := data.UnitDefaults()
	base.HealthMax, base.Health, base.ToHit, base.Defence = 99, 99, 7, 8

	for _, diff := range []mapload.Difficulty{mapload.DifficultyEasy, mapload.DifficultyNormal, mapload.DifficultyHard} {
		a, err := mapload.Adjust(base, diff)
		if err != nil {
			t.Fatalf("Adjust: %v", err)
		}
		b, err := mapload.Adjust(base, diff)
		if err != nil {
			t.Fatalf("Adjust: %v", err)
		}
		if a != b {
			t.Errorf("difficulty %d gave two answers for one definition:\n %+v\n %+v", int32(diff), a, b)
		}
		if base.HealthMax != 99 || base.Health != 99 || base.ToHit != 7 || base.Defence != 8 {
			t.Errorf("difficulty %d moved its argument: %+v", int32(diff), base)
		}
	}
}

func TestTheAdjustmentLeavesReachAlone(t *testing.T) {
	in := data.UnitDefaults()
	in.Reach = 9 // not 1, so a difficulty that floored it back to the default would show.

	for _, diff := range []mapload.Difficulty{mapload.DifficultyEasy, mapload.DifficultyNormal, mapload.DifficultyHard} {
		got, err := mapload.Adjust(in, diff)
		if err != nil {
			t.Fatalf("Adjust at difficulty %d: %v", int32(diff), err)
		}
		if got.Reach != 9 {
			t.Errorf("difficulty %d left reach at %d, want the input's own 9", int32(diff), got.Reach)
		}
	}
}

// TestARefusedEntryFailsTheWorldAndNamesItself: a resolved row whose damage
// selector takes an unmodelled arm stops the build, naming the entry, rather
// than being given no damage in silence.
func TestARefusedEntryFailsTheWorldAndNamesItself(t *testing.T) {
	bad := unitDefRow(200, 0, 30)
	bad[11], bad[12], bad[13] = 4, 9, 1 // the second-pair arm

	tbl := &mapload.Table{Units: defCollection{{}, {name: "Offender", params: bad}}}
	m := &alm.Map{Width: 32, Height: 32, Units: []alm.Unit{{X: 0x1000, Y: 0x1000, ClassID: 200}}}

	w, err := mapload.FromALMWith(m, tbl, mapload.DifficultyNormal)
	if err == nil {
		t.Fatal("a world was built over an entry whose damage selector is refused")
	}
	if w != nil {
		t.Error("a refused build returned a world beside its error")
	}
	if !strings.Contains(err.Error(), "Offender") {
		t.Errorf("error %q does not name the offending entry", err)
	}
}

// TestAnUndefinedDifficultyRefusesEvenAMapThatResolvesNothing: the value is
// checked before the walk, so a map with no placements cannot accept one by
// never reaching the arithmetic.
func TestAnUndefinedDifficultyRefusesEvenAMapThatResolvesNothing(t *testing.T) {
	for _, m := range []*alm.Map{nil, {Width: 32, Height: 32}} {
		w, err := mapload.FromALMWith(m, nil, mapload.Difficulty(9))
		if err == nil {
			t.Error("difficulty 9 was accepted by a map that resolves nothing")
		}
		if w != nil {
			t.Error("a refused build returned a world beside its error")
		}
	}
}
