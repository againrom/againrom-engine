package ui

import (
	"fmt"
	"strings"
	"testing"
)

func TestChargenPreCreateOwnsIdentityAndStoredName(t *testing.T) {
	c := NewChargen(ChargenSetup{
		Name: "Danath", PreCreate: &ChargenPreCreate{},
		EncodeName: func(r rune) (byte, bool) {
			if r == 'Ж' {
				return 0x86, true
			}
			if r == 'x' {
				return 'x', true
			}
			return 0, false
		},
		Choices: []ChargenChoice{{Options: []string{"male", "female"}, Parent: -1}, {Options: []string{"fighter", "mage"}, Parent: -1}, {Options: []string{"one", "two"}, Parent: -1, Start: 1}},
		Stats:   []ChargenStat{{Floor: 0, Ceiling: 50, Start: 25}}, Cost: triangular(50), Budget: 1000,
	})
	if c.Stage() != PreCreateStage {
		t.Fatalf("Stage = %v, want pre-create", c.Stage())
	}
	c.EditName("😀", false)
	if c.NameText() != "Danath" {
		t.Fatalf("rejected rune cleared name: %q", c.NameText())
	}
	c.EditName("Ж", false)
	typed := "Danath" + string([]byte{0x86})
	if c.NameText() != typed {
		t.Fatalf("stored display bytes = %x, want the byte appended to %x", c.NameText(), typed)
	}
	c.SelectPreChoice(3)
	c.Forward()
	if c.Stage() != DetailedStage {
		t.Fatal("Forward did not enter detailed stage")
	}
	r, ok := c.Result()
	if !ok || r.Name != typed || r.Choices[0] != 1 || r.Choices[1] != 1 || r.Choices[2] != 1 {
		t.Fatalf("Forward result = %+v, %v", r, ok)
	}
	// Back enters the pre-create page again (TEXT-073): the typed name stays
	// and the first picture is lit.
	if !c.Back() || c.Stage() != PreCreateStage || c.NameText() != typed || c.PreChoice() != 0 {
		t.Fatalf("detailed Back = name %x, picture %d; want the typed name and picture 0", c.NameText(), c.PreChoice())
	}
}

func TestChargenResetPreservesSkillAndReplacesPreview(t *testing.T) {
	calls := 0
	c := NewChargen(ChargenSetup{
		Name: "Danath", PreCreate: &ChargenPreCreate{},
		Choices: []ChargenChoice{
			{Options: []string{"male", "female"}, Parent: -1},
			{Options: []string{"fighter", "mage"}, Parent: -1},
			{Options: []string{"one", "two", "three"}, Parent: -1},
		},
		Stats: []ChargenStat{{Floor: 0, Ceiling: 50, Start: 25}}, Cost: triangular(50), Budget: 2000,
		Preview: func(r ChargenResult) ChargenPreview {
			calls++
			return ChargenPreview{Subject: PanelSubject{Name: r.Name, Speed: r.Choices[2]}}
		},
	})
	c.Forward()
	if calls != 1 {
		t.Fatalf("Forward preview calls = %d, want 1", calls)
	}
	if !c.SelectSkill(2) || !c.AdjustStat(0, 1) {
		t.Fatal("fixture edits were refused")
	}
	if calls != 3 {
		t.Fatalf("accepted edits preview calls = %d, want 3", calls)
	}
	c.Reset()
	r, ok := c.Result()
	if !ok || r.Choices[2] != 2 || r.Stats[0] != 25 {
		t.Fatalf("Reset result = %+v, %v", r, ok)
	}
	if got := c.Preview().Subject.Speed; got != 2 || calls != 4 {
		t.Fatalf("Reset preview = (%d, %d calls), want (2, 4)", got, calls)
	}
	before := c.Preview()
	if c.AdjustStat(0, 30) {
		t.Fatal("over-ceiling statistic edit was accepted")
	}
	if got := c.Preview(); got != before || calls != 4 {
		t.Fatalf("refused edit rebuilt preview = (%+v, %d calls), want (%+v, 4)", got, calls, before)
	}
	if !c.Back() || c.Stage() != PreCreateStage {
		t.Fatal("Back did not return to pre-create")
	}
	c.Forward()
	r, ok = c.Result()
	if !ok || r.Choices[2] != 0 || r.Stats[0] != 25 || calls != 5 {
		t.Fatalf("Back/Forward result = %+v, %v (%d preview calls), want reset draft and 5 calls", r, ok, calls)
	}
}

func TestChargenNamePreservesCaseAndStopsAtTenBytes(t *testing.T) {
	c := NewChargen(ChargenSetup{})
	c.EditName("XyZ123456789", false)
	if got := c.NameText(); got != "XyZ1234567" {
		t.Fatalf("NameText = %q, want the first ten bytes with case preserved", got)
	}
	c.EditName("", true)
	if got := c.NameText(); got != "XyZ123456" {
		t.Fatalf("NameText after backspace = %q, want %q", got, "XyZ123456")
	}
}

func TestDetailedPointBuyRefusalsKeepTheWholeProjection(t *testing.T) {
	cost := triangular(45)
	type refusedEdit struct {
		name   string
		stats  []ChargenStat
		budget int
		row    int
		delta  int
	}
	cases := []refusedEdit{
		{
			name: "floor",
			stats: []ChargenStat{
				{Name: "Body", Floor: 15, Ceiling: 45, Start: 15},
				{Name: "Reaction", Floor: 15, Ceiling: 45, Start: 25},
			},
			budget: cost[15] + cost[25], row: 0, delta: -1,
		},
		{
			name: "ceiling",
			stats: []ChargenStat{
				{Name: "Body", Floor: 15, Ceiling: 45, Start: 45},
				{Name: "Reaction", Floor: 15, Ceiling: 45, Start: 25},
			},
			budget: cost[45] + cost[25] + 100, row: 0, delta: 1,
		},
		{
			name: "budget",
			stats: []ChargenStat{
				{Name: "Body", Floor: 15, Ceiling: 45, Start: 25},
				{Name: "Reaction", Floor: 15, Ceiling: 45, Start: 25},
			},
			budget: cost[25] + cost[25], row: 1, delta: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := NewChargen(ChargenSetup{
				Name:      "Danath",
				PreCreate: &ChargenPreCreate{},
				Choices: []ChargenChoice{
					{Options: []string{"male", "female"}, Parent: -1},
					{Options: []string{"fighter", "mage"}, Parent: -1},
					{Options: []string{"blade", "axe", "bludgeon", "pike", "shooting"}, Parent: -1},
				},
				Stats: tc.stats, Cost: cost, Budget: tc.budget,
				Preview: func(r ChargenResult) ChargenPreview {
					calls++
					return ChargenPreview{Subject: PanelSubject{
						Name:  r.Name,
						Speed: r.Choices[2],
						Char:  UnitCharacter{Body: r.Stats[0], Reaction: r.Stats[1]},
					}}
				},
			})
			c.Forward()
			beforeResult, beforeOK := c.Result()
			beforePreview, beforeRemaining := c.Preview(), c.Remaining()
			if !beforeOK || calls != 1 {
				t.Fatalf("fixture before refusal = (%+v, %v), preview calls %d", beforeResult, beforeOK, calls)
			}

			if c.AdjustStat(tc.row, tc.delta) {
				t.Fatalf("AdjustStat(%d, %d) was accepted", tc.row, tc.delta)
			}
			afterResult, afterOK := c.Result()
			if fmt.Sprint(afterResult) != fmt.Sprint(beforeResult) || afterOK != beforeOK {
				t.Fatalf("refusal changed draft: before (%+v, %v), after (%+v, %v)",
					beforeResult, beforeOK, afterResult, afterOK)
			}
			if got := c.Remaining(); got != beforeRemaining {
				t.Fatalf("refusal changed remaining: got %d, want %d", got, beforeRemaining)
			}
			if got := c.Preview(); got != beforePreview || calls != 1 {
				t.Fatalf("refusal changed/rebuilt preview: got %+v calls %d, want %+v calls 1", got, calls, beforePreview)
			}
		})
	}
}

// Every fixture below is built in test code: no game table, no decoded cost
// curve, nothing that could read an install. The numbers are chosen to make
// the tests exercise interesting boundaries, not to resemble the shipped
// budget or bounds -- 0119's spec.md/plan.md own that arithmetic and this
// package is not allowed to read it (AC-11).

// triangular is a synthetic cumulative-cost table: cost(v) = v*(v+1)/2 for v
// in [0, n]. It is monotonically increasing with a growing step, which is
// enough shape to make a budget bite at a specific value rather than
// uniformly.
func triangular(n int) []int {
	cost := make([]int, n+1)
	for v := 0; v <= n; v++ {
		cost[v] = v * (v + 1) / 2
	}
	return cost
}

// TestChargenAC2StepCostIsTheCostDifference is AC-2: raising a statistic by
// one costs exactly the difference of the two cumulative costs, and lowering
// it back refunds exactly that much.
func TestChargenAC2StepCostIsTheCostDifference(t *testing.T) {
	cost := triangular(10) // 0 1 3 6 10 15 21 28 36 45 55
	setup := ChargenSetup{
		Title:   "t",
		Stats:   []ChargenStat{{Name: "A", Floor: 0, Ceiling: 10, Start: 5}},
		Cost:    cost,
		Budget:  50, // Remaining at Start=5 (cost 15): 35
		Confirm: "go",
	}
	c := NewChargen(setup)
	c.Move(1) // the only row: focus lands on the statistic

	before := c.Remaining()
	if want := setup.Budget - cost[5]; before != want {
		t.Fatalf("Remaining() at Start = %d, want %d", before, want)
	}

	// Raise 5 -> 6.
	c.Adjust(1)
	res, ok := c.Result()
	if !ok || res.Stats[0] != 6 {
		t.Fatalf("Result() after raising = (%v, %v), want (6, true)", res, ok)
	}
	delta := cost[6] - cost[5]
	if got, want := c.Remaining(), before-delta; got != want {
		t.Fatalf("Remaining() after raising by one = %d, want %d (before %d - delta %d)",
			got, want, before, delta)
	}

	// Lower it back 6 -> 5: the refund must be exactly the charge.
	c.Adjust(-1)
	res, ok = c.Result()
	if !ok || res.Stats[0] != 5 {
		t.Fatalf("Result() after lowering back = (%v, %v), want (5, true)", res, ok)
	}
	if got := c.Remaining(); got != before {
		t.Fatalf("Remaining() after raising then lowering = %d, want %d (back to where it started)",
			got, before)
	}
}

// TestChargenAC3ReachableStatesAreLegal is AC-3: no sequence of steps can
// leave the screen with an illegal spread.
//
// It is a real search, not a handful of samples: a depth-first walk over
// every state reachable from Start by unit steps on any of four statistic
// rows, driven ENTIRELY through the exported Move/Adjust/Result surface (the
// same surface a real input handler uses), asserting Legal() after every
// single Adjust call and backtracking with the step's own inverse rather
// than replaying from scratch.
func TestChargenAC3ReachableStatesAreLegal(t *testing.T) {
	cost := triangular(8) // 0 1 3 6 10 15 21 28 36
	const budget = 44     // Start (4,4,4,4) costs 4*10 = 40; Remaining = 4
	setup := ChargenSetup{
		Title: "t",
		Stats: []ChargenStat{
			{Name: "A", Floor: 0, Ceiling: 8, Start: 4},
			{Name: "B", Floor: 0, Ceiling: 8, Start: 4},
			{Name: "C", Floor: 0, Ceiling: 8, Start: 4},
			{Name: "D", Floor: 0, Ceiling: 8, Start: 4},
		},
		Cost:    cost,
		Budget:  budget,
		Confirm: "go",
	}
	c := NewChargen(setup)
	if !c.Legal() {
		t.Fatalf("initial state is not legal; fixture is broken")
	}

	type state [4]int
	read := func() state {
		res, ok := c.Result()
		if !ok {
			t.Fatalf("Result() reported an illegal state mid-search: %v", c)
		}
		return state{res.Stats[0], res.Stats[1], res.Stats[2], res.Stats[3]}
	}
	focusRow := func(row int) {
		for c.Focus() != row {
			c.Move(1)
		}
	}

	visited := map[state]bool{}
	checked := 0
	var explore func()
	explore = func() {
		cur := read()
		if visited[cur] {
			return
		}
		visited[cur] = true
		if !c.Legal() {
			t.Fatalf("state %v is not legal", cur)
		}
		for row := 0; row < 4; row++ {
			for _, d := range []int{-1, 1} {
				focusRow(row)
				before := read()
				c.Adjust(d)
				checked++
				if !c.Legal() {
					t.Fatalf("Adjust(row=%d, d=%d) from %v reached an illegal state", row, d, before)
				}
				after := read()
				if after == before {
					continue // refused: nothing new to explore
				}
				explore()
				// Undo: the exact inverse step. StepCost's own symmetry
				// (pkg/data's own doc: charge at v equals refund at v+1)
				// guarantees this always succeeds, since undoing can only
				// lower cost or return a statistic toward where it already
				// legally stood.
				focusRow(row)
				c.Adjust(-d)
				if got := read(); got != before {
					t.Fatalf("undo of Adjust(row=%d, d=%d) left %v, want %v", row, d, got, before)
				}
			}
		}
	}
	explore()

	t.Logf("AC-3: visited %d legal states via %d Adjust calls", len(visited), checked)
	if len(visited) < 10 {
		t.Fatalf("search explored only %d states; fixture is not exercising the budget", len(visited))
	}
}

// TestChargenAC4RefusalChangesNothing is AC-4: a step that would break a
// bound changes nothing at all -- not the statistic, not Remaining, not the
// focus.
func TestChargenAC4RefusalChangesNothing(t *testing.T) {
	cost := triangular(10)
	setup := ChargenSetup{
		Title: "t",
		Stats: []ChargenStat{
			{Name: "A", Floor: 2, Ceiling: 8, Start: 2}, // pinned at its own floor
			{Name: "B", Floor: 0, Ceiling: 8, Start: 8}, // pinned at its own ceiling
			{Name: "C", Floor: 0, Ceiling: 10, Start: 0},
		},
		Cost:    cost,
		Budget:  cost[2] + cost[8] + cost[0], // exactly what Start already costs
		Confirm: "go",
	}

	check := func(t *testing.T, name string, row int, d int) {
		t.Helper()
		c := NewChargen(setup)
		for c.Focus() != row {
			c.Move(1)
		}
		beforeRes, beforeOK := c.Result()
		beforeRemaining := c.Remaining()
		beforeFocus := c.Focus()

		c.Adjust(d)

		afterRes, afterOK := c.Result()
		if afterOK != beforeOK || fmt.Sprint(afterRes) != fmt.Sprint(beforeRes) {
			t.Fatalf("%s: Result() changed: before (%v,%v) after (%v,%v)", name, beforeRes, beforeOK, afterRes, afterOK)
		}
		if got := c.Remaining(); got != beforeRemaining {
			t.Fatalf("%s: Remaining() changed from %d to %d", name, beforeRemaining, got)
		}
		if got := c.Focus(); got != beforeFocus {
			t.Fatalf("%s: Focus() changed from %d to %d", name, beforeFocus, got)
		}
	}

	// Row 0 (choice-row count is 0, so row 0 is statistic "A"): below its floor.
	check(t, "below floor", 0, -1)
	// Row 1: "B" above its ceiling.
	check(t, "above ceiling", 1, 1)
	// Row 2: "C" is inside [0,10] but the budget is already fully spent, so
	// even a legal-looking +1 must be refused on the budget test alone.
	check(t, "budget exhausted", 2, 1)
}

// TestChargenAC5ParentChangeKeepsOrClampsIndex is AC-5: changing a parent
// choice re-labels the dependent row and leaves its index where it was, or
// -- when the new option list is too short to hold it -- clamps it to the
// last option rather than resetting to 0.
func TestChargenAC5ParentChangeKeepsOrClampsIndex(t *testing.T) {
	setup := ChargenSetup{
		Title: "t",
		Choices: []ChargenChoice{
			{Name: "Parent", Options: []string{"P0", "P1", "P2"}, Parent: -1},
			{
				Name: "Child",
				OptionsFor: [][]string{
					{"A0", "A1", "A2", "A3"}, // 4 options under P0
					{"B0", "B1"},             // 2 options under P1
					{"C0", "C1", "C2"},       // 3 options under P2
				},
				Parent: 0,
			},
		},
		Confirm: "go",
	}
	c := NewChargen(setup)

	// Move the child to index 3 ("A3") while the parent is on P0 (4 options),
	// driving forward one step at a time: 0 -> 1 -> 2 -> 3.
	c.Move(1) // focus: Child
	c.Adjust(1)
	c.Adjust(1)
	c.Adjust(1)
	if got := c.RowText(1); !strings.Contains(got, "A3") {
		t.Fatalf("Child at index 3 under P0 = %q, want it to name A3", got)
	}

	// Change the parent P0 -> P1 (2 options: index 3 does not fit).
	c.Move(-1) // focus: Parent
	c.Adjust(1)
	if got := c.RowText(0); !strings.Contains(got, "P1") {
		t.Fatalf("Parent after one Adjust = %q, want it to name P1", got)
	}
	// AC-5's clamp half: index 3 does not fit in 2 options, so it must clamp
	// to the LAST option (index 1, "B1") rather than reset to the first
	// ("B0"). A reset-to-zero implementation would fail this assertion.
	if got := c.RowText(1); !strings.Contains(got, "B1") {
		t.Fatalf("Child re-labelled under P1 = %q, want it to clamp to B1 (not reset to B0)", got)
	}

	// Change the parent P1 -> P2 (3 options: the held index 1 fits). Focus is
	// already on Parent from the block above -- moving again would wrap onto
	// Child, since there are only two rows.
	c.Adjust(1)
	if got := c.RowText(0); !strings.Contains(got, "P2") {
		t.Fatalf("Parent after two Adjusts = %q, want it to name P2", got)
	}
	// AC-5's preservation half: index 1 fits in 3 options (C0,C1,C2), so it
	// must STAY at index 1 ("C1"), not reset to 0 ("C0").
	if got := c.RowText(1); !strings.Contains(got, "C1") {
		t.Fatalf("Child re-labelled under P2 = %q, want it to keep index 1 (C1)", got)
	}
}

// TestChargenAC6ConfirmingIllegalProducesNoResult is AC-6: confirming an
// illegal spread produces no result.
//
// A legal spread can never become illegal through Adjust alone -- that is
// exactly AC-3 -- so the only way to witness this at the model's own level is
// a setup that starts illegal: two statistic rows whose Start values already
// cost more than the budget allows, even though each individually sits
// inside its own Floor/Ceiling.
func TestChargenAC6ConfirmingIllegalProducesNoResult(t *testing.T) {
	cost := triangular(10)
	setup := ChargenSetup{
		Title: "t",
		Stats: []ChargenStat{
			{Name: "A", Floor: 0, Ceiling: 10, Start: 5},
			{Name: "B", Floor: 0, Ceiling: 10, Start: 5},
		},
		Cost:    cost,
		Budget:  cost[5] + cost[5] - 1, // one short of what Start costs
		Confirm: "go",
	}
	c := NewChargen(setup)
	if c.Legal() {
		t.Fatalf("fixture is legal at Start, want illegal (budget is one short)")
	}
	if res, ok := c.Result(); ok {
		t.Fatalf("Result() on an illegal spread = (%v, true), want ok=false", res)
	}
	// Asking again must not have side effects.
	if res, ok := c.Result(); ok {
		t.Fatalf("second Result() call = (%v, true), want ok=false", res)
	}

	// Lowering a statistic can bring it back into budget; Result() must track
	// that live, not report a state cached from before. Focus already starts
	// on row 0 ("A"); no Move is needed to reach it.
	c.Adjust(-1)
	if !c.Legal() {
		t.Fatalf("after lowering A by one, want legal (budget was one short)")
	}
	res, ok := c.Result()
	if !ok {
		t.Fatalf("Result() after lowering into budget = (_, false), want ok=true")
	}
	if res.Stats[0] != 4 || res.Stats[1] != 5 {
		t.Fatalf("Result().Stats = %v, want [4 5]", res.Stats)
	}
}

func TestChargenMove(t *testing.T) {
	setup := ChargenSetup{
		Title: "t",
		Choices: []ChargenChoice{
			{Name: "Sex", Options: []string{"M", "F"}, Parent: -1},
			{Name: "Class", Options: []string{"W", "M"}, Parent: -1},
		},
		Stats: []ChargenStat{
			{Name: "A", Floor: 0, Ceiling: 10, Start: 5},
			{Name: "B", Floor: 0, Ceiling: 10, Start: 5},
			{Name: "C", Floor: 0, Ceiling: 10, Start: 5},
		},
		Confirm: "go",
	}
	c := NewChargen(setup)
	if got := c.Rows(); got != 5 {
		t.Fatalf("Rows() = %d, want 5", got)
	}
	if got := c.Focus(); got != 0 {
		t.Fatalf("initial Focus() = %d, want 0", got)
	}
	for want := 1; want < 12; want++ {
		c.Move(1)
		if got := c.Focus(); got != want%5 {
			t.Fatalf("after %d Move(1): Focus() = %d, want %d", want, got, want%5)
		}
	}
	// Wrap backward from row 0.
	c.Move(-c.Focus()) // back to row 0
	c.Move(-1)
	if got := c.Focus(); got != 4 {
		t.Fatalf("Move(-1) from row 0: Focus() = %d, want 4 (wrap to the last row)", got)
	}
	// A large jump wraps correctly rather than only handling +-1.
	c.Move(1000003) // 1000003 mod 5 == 3
	if got, want := c.Focus(), (4+1000003)%5; got != want {
		t.Fatalf("Move(1000003) from row 4: Focus() = %d, want %d", got, want)
	}
}

// TestChargenRowTextMarksTheFocusedRow checks RowText's marker and content,
// for both a choice row and a statistic row, mirroring picker.go's own
// RowText contract.
func TestChargenRowTextMarksTheFocusedRow(t *testing.T) {
	setup := ChargenSetup{
		Title:   "t",
		Choices: []ChargenChoice{{Name: "Sex", Options: []string{"Male", "Female"}, Parent: -1}},
		Stats:   []ChargenStat{{Name: "Body", Floor: 0, Ceiling: 10, Start: 5}},
		Cost:    triangular(10),
		Budget:  100,
		Confirm: "go",
	}
	c := NewChargen(setup)
	if got, want := c.RowText(0), "> Sex: Male"; got != want {
		t.Fatalf("RowText(0) with focus on row 0 = %q, want %q", got, want)
	}
	if got, want := c.RowText(1), "  Body: 5"; got != want {
		t.Fatalf("RowText(1) with focus on row 0 = %q, want %q", got, want)
	}
	c.Move(1)
	if got, want := c.RowText(0), "  Sex: Male"; got != want {
		t.Fatalf("RowText(0) with focus on row 1 = %q, want %q", got, want)
	}
	if got, want := c.RowText(1), "> Body: 5"; got != want {
		t.Fatalf("RowText(1) with focus on row 1 = %q, want %q", got, want)
	}
	// Out of range is total and silent.
	for _, bad := range []int{-1, -1000, 2, 1 << 20} {
		if got := c.RowText(bad); got != "" {
			t.Fatalf("RowText(%d) = %q, want \"\" (out of range)", bad, got)
		}
	}
}

func TestChargenHeaderAndFooterText(t *testing.T) {
	setup := ChargenSetup{
		Title:   "GENERATE A HERO",
		Stats:   []ChargenStat{{Name: "Body", Floor: 0, Ceiling: 10, Start: 5}},
		Cost:    triangular(10),
		Budget:  100,
		Confirm: "ENTER: begin the mission",
	}
	c := NewChargen(setup)
	if got := c.HeaderText(); !strings.Contains(got, setup.Title) {
		t.Fatalf("HeaderText() = %q, does not carry the title %q", got, setup.Title)
	}
	if got := c.FooterText(); !strings.Contains(got, setup.Confirm) {
		t.Fatalf("FooterText() = %q, does not carry the confirm label %q", got, setup.Confirm)
	}
	want := fmt.Sprintf("%d", 100-triangular(10)[5])
	if got := c.FooterText(); !strings.Contains(got, want) {
		t.Fatalf("FooterText() = %q, does not carry Remaining() = %s", got, want)
	}
}

func TestChargenChoiceStartOpensThereAndStillMoves(t *testing.T) {
	setup := ChargenSetup{
		Title: "t",
		Choices: []ChargenChoice{
			{Name: "Skill", Options: []string{"Blade", "Axe", "Bludgen", "Pike", "Shooting"}, Parent: -1, Start: 3},
		},
		Confirm: "go",
	}
	c := NewChargen(setup)
	if got, want := c.RowText(0), "> Skill: Pike"; got != want {
		t.Fatalf("RowText(0) at construction = %q, want %q -- the row must open on its own Start", got, want)
	}
	res, ok := c.Result()
	if !ok || res.Choices[0] != 3 {
		t.Fatalf("Result() at construction = (%v, %v), want (Choices[0]=3, true)", res, ok)
	}

	// The player's own move still wins from there -- Start seeds the row, it
	// does not lock it.
	c.Adjust(1)
	if got, want := c.RowText(0), "> Skill: Shooting"; got != want {
		t.Fatalf("RowText(0) after Adjust(1) from Start = %q, want %q", got, want)
	}
	c.Adjust(-2)
	if got, want := c.RowText(0), "> Skill: Bludgen"; got != want {
		t.Fatalf("RowText(0) after Adjust(-2) = %q, want %q", got, want)
	}
}

func TestChargenChoiceStartClamps(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start int
		opts  []string
		want  string
	}{
		{"negative clamps to the first option", -7, []string{"A0", "A1", "A2"}, "A0"},
		{"past the end clamps to the last option", 99, []string{"A0", "A1", "A2"}, "A2"},
		{"empty option list answers the empty row regardless", 1, nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setup := ChargenSetup{
				Title:   "t",
				Choices: []ChargenChoice{{Name: "X", Options: tc.opts, Parent: -1, Start: tc.start}},
				Confirm: "go",
			}
			c := NewChargen(setup)
			want := "> X: " + tc.want
			if got := c.RowText(0); got != want {
				t.Fatalf("RowText(0) = %q, want %q", got, want)
			}
		})
	}
}

// TestChargenMalformedSetupDoesNotPanic is the required negative case: a
// setup malformed in several independent ways at once must never panic and
// must never index out of range, across a long scripted sequence of every
// exported operation.
func TestChargenMalformedSetupDoesNotPanic(t *testing.T) {
	setup := ChargenSetup{
		Title: "malformed",
		Choices: []ChargenChoice{
			{Name: "SelfParent", Options: []string{"x"}, Parent: 0},  // parents itself
			{Name: "OutOfRange", Options: []string{"y"}, Parent: 99}, // no such row
			{Name: "NoOptions", Parent: -1},                          // flat, but both Options and OptionsFor empty
			{
				Name:   "ShortOptionsFor",
				Parent: 2, // depends on the always-empty row above
				OptionsFor: [][]string{
					{"only one entry, parent offers none"},
				},
			},
			{Name: "Cyclic0", Parent: 5}, // 4 <-> 5 cycle
			{Name: "Cyclic1", Parent: 4},
		},
		Stats: []ChargenStat{
			{Name: "ShortCost", Floor: 0, Ceiling: 200, Start: 50}, // Cost has 5 entries only
			{Name: "InvertedBounds", Floor: 10, Ceiling: 2, Start: 10},
			{Name: "NegativeStart", Floor: -50, Ceiling: 50, Start: -20},
		},
		Cost:    []int{0, 1, 2, 3, 4},
		Budget:  10,
		Confirm: "go",
	}

	run := func(name string, f func()) {
		t.Helper()
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("%s panicked: %v", name, r)
			}
		}()
		f()
	}

	run("construct and read", func() {
		c := NewChargen(setup)
		_ = c.Rows()
		_ = c.Focus()
		_ = c.Legal()
		_ = c.Remaining()
		_ = c.HeaderText()
		_ = c.FooterText()
		for i := -3; i < c.Rows()+3; i++ {
			_ = c.RowText(i)
		}
		if _, ok := c.Result(); ok {
			// Not necessarily wrong, but exercise both branches without
			// asserting which one: InvertedBounds can never be legal, so
			// this path should not be reached in this fixture, and if it
			// somehow were the assertion below still must not panic.
			_ = ok
		}
	})

	run("a long scripted walk over every row", func() {
		c := NewChargen(setup)
		moves := []int{1, 1, -1, 3, -5, 1000, -1000, 0, 2, -2}
		adjusts := []int{1, -1, 5, -5, 1 << 20, -(1 << 20), 0}
		for step := 0; step < 500; step++ {
			c.Move(moves[step%len(moves)])
			c.Adjust(adjusts[step%len(adjusts)])
			_ = c.Legal()
			_ = c.Remaining()
			_ = c.RowText(c.Focus())
			_, _ = c.Result()
		}
	})

	run("wholly empty setup", func() {
		c := NewChargen(ChargenSetup{})
		c.Move(1)
		c.Move(-1)
		c.Adjust(1)
		c.Adjust(-1)
		if got := c.Rows(); got != 0 {
			t.Fatalf("Rows() on an empty setup = %d, want 0", got)
		}
		if got := c.RowText(0); got != "" {
			t.Fatalf("RowText(0) on an empty setup = %q, want \"\"", got)
		}
		res, ok := c.Result()
		if !ok {
			t.Fatalf("Result() on an empty setup = (_, false), want ok=true (vacuously legal)")
		}
		if len(res.Choices) != 0 || len(res.Stats) != 0 {
			t.Fatalf("Result() on an empty setup = %v, want both fields empty", res)
		}
	})

	run("nil Cost table", func() {
		s := setup
		s.Cost = nil
		c := NewChargen(s)
		_ = c.Legal()
		_ = c.Remaining()
		c.Move(len(s.Choices)) // land on the first statistic row
		c.Adjust(1)
		c.Adjust(-1)
	})
}
