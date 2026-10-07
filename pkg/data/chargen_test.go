package data_test

import (
	"testing"

	"againrom/pkg/data"
)

// THE COST FUNCTION'S OWN PUBLISHED FIGURES, and they are a discriminating check
// rather than a restatement: they were computed from the formula and compared
// against a second research row that quotes them as consequences, so agreement
// is two independently written rows agreeing to the unit.
func TestTheCostOfAStatisticIsCumulativeAndEscalates(t *testing.T) {
	for _, tc := range []struct{ n, want int32 }{
		{15, 2},   // the floor: two points, which is why flooring three frees 134
		{25, 10},  // the start: four of these plus a pool of 100 is the budget
		{45, 164}, // the click ceiling, and already past the whole budget
	} {
		if got := data.PointCost(tc.n); got != tc.want {
			t.Errorf("PointCost(%d) = %d, want %d", tc.n, got, tc.want)
		}
	}

	// The escalation, at both ends of the range where it is visible.
	if got := data.StepCost(44); got != 22 {
		t.Errorf("the step into 45 costs %d, want 22", got)
	}
	if got := data.StepCost(20); got != 1 {
		t.Errorf("a step in the low twenties costs %d, want 1", got)
	}
}

// A REFUND IS EXACTLY THE STEP THAT REACHED THE VALUE, over the whole range a
// click can move. It is algebra rather than an observation — the cost is
// cumulative, so both differences are of the same function — and asserting it
// over every v is what makes that an identity instead of a spot check.
func TestARefundIsTheStepThatReachedIt(t *testing.T) {
	for v := data.StatFloor + 1; v <= data.StatCeiling; v++ {
		refund := data.PointCost(v) - data.PointCost(v-1)
		if charge := data.StepCost(v - 1); refund != charge {
			t.Fatalf("at %d the refund is %d and the charge that reached it %d", v, refund, charge)
		}
	}
}

// THE POOL AND THE BUDGET ARE ONE NUMBER SEEN TWICE. The screen starts at four
// times the initialiser with 100 in hand; the accepting side loads 140. If these
// two constants ever disagreed, a client could spend a pool the server refuses.
func TestThePoolAndTheBudgetAreOneNumber(t *testing.T) {
	if got := 4*data.PointCost(data.ChargenStat) + data.ChargenPool; got != data.ChargenBudget {
		t.Errorf("4*T(25) + pool = %d, want the budget %d", got, data.ChargenBudget)
	}
}

// THE THREE REACHABLE MAXIMA, each found by search rather than asserted: how
// high one statistic goes with the others left at the start, with the others
// floored, and how high all four go together. They are the shape of the legal
// space, and a cost curve that was subtly wrong would move at least one.
func TestHowHighAStatisticCanActuallyBeBought(t *testing.T) {
	// Highest legal value of Body for a given value of the other three.
	peak := func(others int32) int32 {
		best := int32(0)
		for n := data.StatFloor; n <= data.StatCeiling; n++ {
			if (data.Spread{Body: n, Reaction: others, Mind: others, Spirit: others}).Legal() {
				best = n
			}
		}
		return best
	}
	if got := peak(data.ChargenStat); got != 42 {
		t.Errorf("with the others at the start a statistic reaches %d, want 42", got)
	}
	if got := peak(data.StatFloor); got != 43 {
		t.Errorf("with the others floored a statistic reaches %d, want 43", got)
	}

	all := int32(0)
	for n := data.StatFloor; n <= data.StatCeiling; n++ {
		if (data.Spread{Body: n, Reaction: n, Mind: n, Spirit: n}).Legal() {
			all = n
		}
	}
	if all != 34 {
		t.Errorf("all four together reach %d, want 34", all)
	}
	if got := (data.Spread{Body: 34, Reaction: 34, Mind: 34, Spirit: 34}).Remaining(); got != 0 {
		t.Errorf("the flat 34 leaves %d of the budget, want 0 — it spends it exactly", got)
	}
	// And the click ceiling is unreachable in generation, which is why the two
	// bounds are not the same bound.
	if (data.Spread{Body: data.StatCeiling, Reaction: data.StatFloor,
		Mind: data.StatFloor, Spirit: data.StatFloor}).Legal() {
		t.Error("a statistic at the click ceiling was generated; the budget must refuse it")
	}
}

// EACH BOUND REFUSES FOR ITS OWN REASON, and the third case is the one that
// shows the two are not the same rule: every statistic is inside the range and
// the spread is still illegal, because the budget is what binds.
func TestASpreadOutsideEitherBoundIsRefused(t *testing.T) {
	for _, tc := range []struct {
		why string
		s   data.Spread
	}{
		{"under the floor", data.Spread{Body: 14, Reaction: 15, Mind: 15, Spirit: 15}},
		{"over the click ceiling", data.Spread{Body: 46, Reaction: 15, Mind: 15, Spirit: 15}},
		{"inside the range and over the budget", data.Spread{Body: 44, Reaction: 15, Mind: 15, Spirit: 15}},
	} {
		if tc.s.Legal() {
			t.Errorf("%s: %+v was accepted, cost %d of %d",
				tc.why, tc.s, tc.s.Cost(), data.ChargenBudget)
		}
	}
	// The zero value is not a generated character either, and it is the one
	// every hand-built hero in this tree carries.
	if (data.Spread{}).Legal() {
		t.Error("the zero spread was accepted as generated")
	}
}

// THE GENERATION START IS ITSELF LEGAL, and it leaves exactly the pool the
// screen draws. Remaining is not clamped, so a spread over the budget states how
// far over rather than reporting nothing left.
func TestTheGenerationStartSpendsFortyAndLeavesTheWholePool(t *testing.T) {
	s := data.ChargenSpread()
	if !s.Legal() {
		t.Fatalf("the generation start %+v is not legal", s)
	}
	if got := s.Cost(); got != 40 {
		t.Errorf("the start costs %d, want 40", got)
	}
	if got := s.Remaining(); got != data.ChargenPool {
		t.Errorf("the start leaves %d, want the pool %d", got, data.ChargenPool)
	}
	if got := (data.Spread{Body: 45, Reaction: 45, Mind: 45, Spirit: 45}).Remaining(); got >= 0 {
		t.Errorf("a spread far over the budget leaves %d, want a negative remainder", got)
	}
}

// NewHero writes the spread it was given and exactly one skill slot, and a slot
// outside 1..5 trains nothing — the general case of what the start's own test
// pins for the start.
func TestNewHeroWritesTheSpreadAndOneSlot(t *testing.T) {
	s := data.Spread{Body: 43, Reaction: 26, Mind: 15, Spirit: 15}
	h := data.NewHero(s, data.SkillPike)
	if h.Body != s.Body || h.Reaction != s.Reaction || h.Mind != s.Mind || h.Spirit != s.Spirit {
		t.Fatalf("NewHero(%+v) wrote %+v", s, h)
	}
	want := [data.SkillSlots]int32{}
	want[data.SkillPike] = data.ChargenSkill
	if h.Skill != want {
		t.Errorf("skills %v, want %v", h.Skill, want)
	}
	for _, slot := range []int32{data.SkillGeneral, -1, data.SkillSlots, 99} {
		if got := data.NewHero(s, slot); got.Skill != [data.SkillSlots]int32{} {
			t.Errorf("slot %d trained %v, want nothing", slot, got.Skill)
		}
	}
	// The start is that constructor applied to the initialiser, and asserting
	// the identity is what keeps the two from drifting apart.
	if data.NewCampaignHero(data.SkillBlade) != data.NewHero(data.ChargenSpread(), data.SkillBlade) {
		t.Error("the generation start is no longer NewHero over the initialiser")
	}
}

// SPEED IS A FUNCTION OF REACTION ALONE, and the branch is at 12 rather than
// near it. The four sampled values are the two arms and both of their edges; the
// walk beside them is what says the branch is where it is claimed to be rather
// than one either side, which is the failure a table of samples would miss.
func TestSpeedIsDerivedFromReactionAtABranchOfTwelve(t *testing.T) {
	for _, tc := range []struct{ reaction, want int32 }{
		{0, 0},   // the zero hero: the law's own answer, not a fallback
		{11, 11}, // the low arm's top — speed IS reaction below 12
		{12, 14}, // the branch: 12/5 + 12
		{26, 17}, // the party's own Reaction
		{45, 21}, // the click ceiling, which no generated spread reaches
		{50, 22}, // the post-generation stat cap
	} {
		h := data.Hero{Reaction: tc.reaction}
		if got := h.Speed(); got != tc.want {
			t.Errorf("Reaction %d gives speed %d, want %d", tc.reaction, got, tc.want)
		}
	}

	// The branch is AT 12: below it the two arms disagree, at and above it they
	// agree with the divided form, and the walk shows the crossing happens once.
	// Walked to the stat cap, because past it the cap is what the branch sees
	// and this walk is about the branch.
	for r := int32(0); r <= data.StatCap; r++ {
		want := r
		if r >= 12 {
			want = r/5 + 12
		}
		if got := (data.Hero{Reaction: r}).Speed(); got != want {
			t.Fatalf("Reaction %d gives %d, want %d", r, got, want)
		}
	}

	// THE CAP RUNS FIRST, exactly as it does for the eight: the caps are step 0
	// of the one recompute both are terms of, so a Reaction past the cap derives
	// as the cap and not as itself.
	if a, b := (data.Hero{Reaction: 100}).Speed(), (data.Hero{Reaction: data.StatCap}).Speed(); a != b {
		t.Errorf("Reaction 100 gives %d and the cap gives %d; the cap must bind first", a, b)
	}

	// And speed reads REACTION AND NOTHING ELSE — the other three move it
	// nowhere, which is the same census the eight are held to.
	base := (data.Hero{Reaction: 26}).Speed()
	for _, h := range []data.Hero{
		{Reaction: 26, Body: 43}, {Reaction: 26, Mind: 40}, {Reaction: 26, Spirit: 40},
	} {
		if got := h.Speed(); got != base {
			t.Errorf("%+v gives speed %d, want the %d Reaction alone gives", h, got, base)
		}
	}
}

// The five shared slots carry the warrior half of the shipped column titles, and
// a slot the table has no column for names nothing.
func TestASkillSlotIsNamedByTheTablesOwnColumn(t *testing.T) {
	for _, tc := range []struct {
		slot int32
		want string
	}{
		{data.SkillGeneral, "General"},
		{data.SkillBlade, "Blade"},
		{data.SkillAxe, "Axe"},
		{data.SkillBludgen, "Bludgen"},
		{data.SkillPike, "Pike"},
		{data.SkillShoot, "Shooting"},
		{data.SkillSlots, ""},
		{-1, ""},
	} {
		if got := data.SkillName(tc.slot); got != tc.want {
			t.Errorf("SkillName(%d) = %q, want %q", tc.slot, got, tc.want)
		}
	}
}

// SIGHT IS A FUNCTION OF MIND AND REACTION TOGETHER, over an integer divisor
// of 25 and a base of 4, with both statistics capped first.
//
// The pairs sampled are the ones the divisor's own boundaries make: the zero
// hero, one either side of each multiple of 25 the sum can reach, and the sum of
// two capped statistics — which is the largest sum a hero can present and is why
// the answer tops out at 8.
func TestSightIsDerivedFromMindAndReactionTogether(t *testing.T) {
	for _, tc := range []struct{ mind, reaction, want int32 }{
		{0, 0, 4},   // the zero hero: the law's own answer, not a fallback
		{24, 0, 4},  // one below the first boundary
		{25, 0, 5},  // and on it
		{0, 25, 5},  // the two statistics enter symmetrically
		{12, 13, 5}, // and they enter as a SUM, not one at a time
		{49, 0, 5},  // one below the second boundary
		{50, 0, 6},  // and on it
		{26, 26, 6}, // the party's own pair
		{50, 50, 8}, // two capped statistics: the largest sum a hero can present
	} {
		h := data.Hero{Mind: tc.mind, Reaction: tc.reaction}
		if got := h.Sight(); got != tc.want {
			t.Errorf("Mind %d, Reaction %d gives sight %d, want %d",
				tc.mind, tc.reaction, got, tc.want)
		}
	}

	// The divisor is 25 and the base is 4, walked rather than sampled, so a
	// boundary one either side of where it is claimed fails here.
	for m := int32(0); m <= data.StatCap; m++ {
		for _, r := range []int32{0, 7, 25, data.StatCap} {
			if got, want := (data.Hero{Mind: m, Reaction: r}).Sight(), (m+r)/25+4; got != want {
				t.Fatalf("Mind %d, Reaction %d gives %d, want %d", m, r, got, want)
			}
		}
	}

	// THE CAPS RUN FIRST, on both statistics and for the eight's reason: a Mind
	// past the cap derives as the cap and not as itself.
	if a, b := (data.Hero{Mind: 300, Reaction: 300}).Sight(),
		(data.Hero{Mind: data.StatCap, Reaction: data.StatCap}).Sight(); a != b {
		t.Errorf("the uncapped pair gives %d and the capped pair %d; the caps must bind first", a, b)
	}

	// And sight reads THOSE TWO AND NOTHING ELSE — Body and Spirit move it
	// nowhere, which is the census every other derived number is held to.
	base := (data.Hero{Mind: 26, Reaction: 26}).Sight()
	for _, h := range []data.Hero{
		{Mind: 26, Reaction: 26, Body: 43}, {Mind: 26, Reaction: 26, Spirit: 40},
	} {
		if got := h.Sight(); got != base {
			t.Errorf("%+v gives sight %d, want the %d the two alone give", h, got, base)
		}
	}
}
