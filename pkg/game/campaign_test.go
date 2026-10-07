package game

import (
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
)

// Kind words: a directory, a signed integer, an integer array. Spelled here
// rather than imported because pkg/formats/reg keeps them unexported and
// internal/synth takes the raw word.
const (
	kindDir      = 0x01
	kindInt      = 0x02
	kindIntArray = 0x06
	kindRoot     = 0x11
)

// scenarioFixture is a scenario registry in the SHAPE the shipped one has and
// at a fraction of its size: a [General] count, two main missions no building
// names, two main missions buildings do name, and the two side missions those
// buildings offer beside them.
//
// It exercises the three things a reader of the real file has to survive and
// that a smaller fixture would not: an offer key written as a BARE INTEGER
// (Mission30) and as an ARRAY (Mission40), and the ZERO SENTINEL inside that
// array — InnMission's own "this NPC has nothing to give".
func scenarioFixture() []byte {
	return synth.Reg(kindRoot, []synth.RegNode{
		{Name: "General", Kind: kindDir, Children: []synth.RegNode{
			{Name: "TotalMissions", Kind: kindInt, Int: 4},
		}},
		{Name: "Mission10", Kind: kindDir, Children: []synth.RegNode{
			{Name: "Mercenaries", Kind: kindInt, Int: 1},
			{Name: "AutoGetMission", Kind: kindInt, Int: 20},
		}},
		{Name: "Mission20", Kind: kindDir, Children: []synth.RegNode{
			{Name: "Mercenaries", Kind: kindInt, Int: 1},
		}},
		{Name: "Mission30", Kind: kindDir, Children: []synth.RegNode{
			{Name: "InnMission", Kind: kindInt, Int: 30},
			{Name: "ShopMission", Kind: kindInt, Int: 31},
		}},
		{Name: "Mission31", Kind: kindDir, Children: []synth.RegNode{
			{Name: "MapObject", Kind: kindInt, Int: 17},
			{Name: "Payment", Kind: kindInt, Int: 700},
		}},
		{Name: "Mission40", Kind: kindDir, Children: []synth.RegNode{
			{Name: "InnMission", Kind: kindIntArray, Ints: []int32{0, 40, 41}},
		}},
		{Name: "Mission41", Kind: kindDir, Children: []synth.RegNode{
			{Name: "Payment", Kind: kindInt, Int: 900},
		}},
	})
}

func readFixture(t *testing.T) Campaign {
	t.Helper()
	r, err := reg.Parse(scenarioFixture())
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return ReadCampaign(r)
}

// The four sets a campaign is: what it declares, which sections are main,
// which are side, and which missions a building offers.
//
// EVERY EXPECTATION IS THE WHOLE SET AND NOT ITS LENGTH. A reader that dropped
// the array-shaped key would still produce a plausible Offered — it would
// merely start in the wrong place — and a length assertion would not see it.
func TestTheCampaignIsReadAsFourSets(t *testing.T) {
	c := readFixture(t)

	if c.Declared != 4 {
		t.Errorf("Declared = %d, want 4", c.Declared)
	}
	if want := []int{10, 20, 30, 40}; !reflect.DeepEqual(c.Main, want) {
		t.Errorf("Main = %v, want %v", c.Main, want)
	}
	if want := []int{31, 41}; !reflect.DeepEqual(c.Side, want) {
		t.Errorf("Side = %v, want %v", c.Side, want)
	}
	// 0 is the sentinel and is not a mission; 40 and 41 arrive from the
	// ARRAY-shaped key and 30 and 31 from the two bare integers.
	if want := []int{30, 31, 40, 41}; !reflect.DeepEqual(c.Offered, want) {
		t.Errorf("Offered = %v, want %v", c.Offered, want)
	}
	if got := c.MapObjects[31]; got != 17 {
		t.Errorf("MapObjects[31] = %d, want 17", got)
	}
}

// The town's boundary is READ, not known: it is the lowest mission any
// building names, and the two missions below it are the ones no building names
// at all.
func TestTheTownBeginsAtTheLowestOfferedMission(t *testing.T) {
	c := readFixture(t)

	n, ok := c.TownBegins()
	if !ok || n != 30 {
		t.Fatalf("TownBegins() = %d, %v; want 30, true", n, ok)
	}
	for _, before := range []int{10, 20} {
		if c.offers(before) {
			t.Errorf("mission %d is offered by a building; the prologue is the part that is not", before)
		}
	}
	for _, after := range []int{30, 31, 40, 41} {
		if !c.offers(after) {
			t.Errorf("mission %d is offered by no building", after)
		}
	}
}

// The ladder, and — the point of the flag — WHICH OF ITS STEPS THIS TREE IS
// ENTITLED TO CLAIM. 10 and 20 are the campaign's own successors because no
// building could have handed either out; every step from 30 on is a placeholder
// for a gate choice and says so.
func TestTheNextMissionSaysWhetherItIsTheCampaignsOrOurs(t *testing.T) {
	c := readFixture(t)

	cases := []struct {
		won  int
		want Offer
		ok   bool
	}{
		{won: 10, want: Offer{Mission: 20, Town: false}, ok: true},
		{won: 20, want: Offer{Mission: 30, Town: true}, ok: true},
		{won: 30, want: Offer{Mission: 40, Town: true}, ok: true},
		// A side mission is not a step: the ladder continues from where it
		// was, at the next MAIN mission above it.
		{won: 31, want: Offer{Mission: 40, Town: true}, ok: true},
		{won: 40, want: Offer{}, ok: false},
		{won: 41, want: Offer{}, ok: false},
	}
	for _, tc := range cases {
		got, ok := c.NextMission(tc.won)
		if ok != tc.ok || got != tc.want {
			t.Errorf("NextMission(%d) = %+v, %v; want %+v, %v",
				tc.won, got, ok, tc.want, tc.ok)
		}
	}
}

// AC-3: the campaign's declared successor and this tree's authored order are
// two separate facts, and on the shipped campaign's first step they agree.
//
// This test used to reach AutoGetMission by hand and its comment said
// interpreting the name was not a fact this tree was entitled to —
// REG-SCN-063 supersedes that, and AutoAdvance is the entitled read the
// comment said did not exist.
func TestAutoAdvanceAgreesWithTheLaddersFirstStep(t *testing.T) {
	c := readFixture(t)

	auto, ok := c.AutoAdvance(10)
	if !ok {
		t.Fatal("the fixture declares no successor for mission 10; the cross-check has nothing to compare")
	}
	got, nok := c.NextMission(10)
	if !nok || got.Mission != auto {
		t.Errorf("NextMission(10) = %d; AutoAdvance(10) = %d", got.Mission, auto)
	}
}

func TestOnlyTheSectionThatDeclaresASuccessorReportsOne(t *testing.T) {
	c := readFixture(t)

	if got, ok := c.AutoAdvance(10); !ok || got != 20 {
		t.Errorf("AutoAdvance(10) = %d, %v; want 20, true", got, ok)
	}
	for _, n := range []int{20, 30, 31, 40, 41} {
		if got, ok := c.AutoAdvance(n); ok {
			t.Errorf("AutoAdvance(%d) = %d, true; want no successor", n, got)
		}
	}
}

func TestAutoAdvanceReadsFR1sFourRows(t *testing.T) {
	r, err := reg.Parse(synth.Reg(kindRoot, []synth.RegNode{
		{Name: "Mission11", Kind: kindDir, Children: []synth.RegNode{
			{Name: "Payment", Kind: kindInt, Int: 100},
		}},
		{Name: "Mission20", Kind: kindDir, Children: []synth.RegNode{
			{Name: "AutoGetMission", Kind: kindInt, Int: -1},
		}},
		{Name: "Mission31", Kind: kindDir, Children: []synth.RegNode{
			{Name: "AutoGetMission", Kind: kindInt, Int: 999},
		}},
		{Name: "Mission40", Kind: kindDir, Children: []synth.RegNode{
			{Name: "AutoGetMission", Kind: kindIntArray, Ints: []int32{1, 2}},
		}},
	}))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	c := ReadCampaign(r)

	cases := []struct {
		mission int
		want    int
		ok      bool
		row     string
	}{
		{11, 0, false, "no AutoGetMission key"},
		{20, 0, false, "AutoGetMission = -1"},
		{31, 999, true, "AutoGetMission = v, any other v"},
		{40, 0, false, "AutoGetMission held as an array, not a single integer"},
	}
	for _, tc := range cases {
		got, ok := c.AutoAdvance(tc.mission)
		if ok != tc.ok || got != tc.want {
			t.Errorf("%s: AutoAdvance(%d) = %d, %v; want %d, %v",
				tc.row, tc.mission, got, ok, tc.want, tc.ok)
		}
	}
}

// A campaign this tree could not read is a campaign that names no successor —
// never a refusal, and never a guess.
func TestAnUnreadableCampaignNamesNoSuccessor(t *testing.T) {
	for _, c := range []Campaign{ReadCampaign(nil), {}} {
		if got, ok := c.NextMission(10); ok {
			t.Errorf("NextMission(10) = %+v on an empty campaign; want none", got)
		}
		if n, ok := c.TownBegins(); ok {
			t.Errorf("TownBegins() = %d on an empty campaign; want none", n)
		}
	}
}

// A section this tree cannot name a mission from is skipped and nothing else
// moves — the registry holds [General] beside the mission records, and a
// modded one may hold anything at all.
func TestOnlyMissionSectionsAreCounted(t *testing.T) {
	r, err := reg.Parse(synth.Reg(kindRoot, []synth.RegNode{
		{Name: "General", Kind: kindDir, Children: []synth.RegNode{
			{Name: "TotalMissions", Kind: kindInt, Int: 1},
		}},
		{Name: "Mission10", Kind: kindDir, Children: []synth.RegNode{
			{Name: "Mercenaries", Kind: kindInt, Int: 1},
		}},
		// Neither of these yields a mission number: one has no digits at
		// all, the other is a value node rather than a section.
		{Name: "Missionary", Kind: kindDir, Children: []synth.RegNode{
			{Name: "InnMission", Kind: kindInt, Int: 99},
		}},
		{Name: "Mission50", Kind: kindInt, Int: 7},
	}))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	c := ReadCampaign(r)
	if want := []int{10}; !reflect.DeepEqual(c.Main, want) {
		t.Errorf("Main = %v, want %v", c.Main, want)
	}
	if len(c.Side) != 0 || len(c.Offered) != 0 {
		t.Errorf("Side = %v, Offered = %v; want both empty", c.Side, c.Offered)
	}
}
