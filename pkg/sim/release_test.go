package sim

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

func TestAReleasedMemberHoldsNoneOfItsSevenFields(t *testing.T) {
	t.Parallel()

	// The victim starts at the edge of sight, not adjacent: the first decision
	// acquires it and the SAME tick's move phase gives the attacker a real
	// pursuit destination (engage.go's own header — "the move loop ... aims
	// every attacker at its victim before advancing it"), so the release below
	// has an actual HasTarget=true to clear rather than one already empty.
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}),
		engFighter(1, 2, 5, 5), engFighter(2, 3, 5+engSight, 5))
	engRun(w, 1)
	if v, held := engVictim(w, 1); !held || v != 2 {
		t.Fatalf("the fixture's own first decision holds %v/%v, want entity 2 acquired", v, held)
	}
	if e := w.entities[indexOfEntity(w.entities, 1)]; !e.HasTarget {
		t.Fatalf("the attacker is not mid-pursuit after acquiring — this test's own setup assumption is wrong")
	}
	i := indexOfEntity(w.entities, 2)
	w.entities[i].X, w.entities[i].Y = 44, 44
	w.decide(w.aiGroups()[0])

	e := w.entities[indexOfEntity(w.entities, 1)]
	if e.HasAttackTarget || e.AttackTarget != 0 {
		t.Errorf("victim: HasAttackTarget=%v AttackTarget=%v, want neither held", e.HasAttackTarget, e.AttackTarget)
	}
	if e.AttackPhase != AttackReady {
		t.Errorf("attack phase is %v, want %v", e.AttackPhase, AttackReady)
	}
	if e.AttackCountdown != 0 {
		t.Errorf("attack countdown is %d, want 0", e.AttackCountdown)
	}
	if !e.HasTarget || e.TargetX != 5 || e.TargetY != 5 {
		t.Errorf("destination: HasTarget=%v (%d,%d), want the post (5,5) held — the walk home",
			e.HasTarget, e.TargetX, e.TargetY)
	}
	if e.Stall != 0 {
		t.Errorf("stall count is %d, want 0", e.Stall)
	}
	if r := w.routes[indexOfEntity(w.entities, 1)]; len(r) != 0 {
		t.Errorf("stored route is %v, want none — the walk home writes a destination directly and searches no route", r)
	}
}

func TestAReleasedMemberWalksHomeThenStandsThereForeverAfter(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}),
		engFighter(1, 2, 5, 5), engFighter(2, 3, 5+engSight, 5))
	engRun(w, 1)
	if e := w.entities[indexOfEntity(w.entities, 1)]; !e.HasTarget {
		t.Fatalf("the attacker is not mid-pursuit after acquiring — this test's own setup assumption is wrong")
	}
	i := indexOfEntity(w.entities, 2)
	w.entities[i].X, w.entities[i].Y = 44, 44
	// entity 2 is itself a guard member (owner 3, not SelfSlot), so this raw
	// teleport puts IT off its own post (10,5) too — left alone, its own
	// group would walk it home and, over enough ticks, straight back into
	// entity 1's notice radius, re-arming a second engagement this test is
	// not about. Anchoring its post at the cell it was just moved to is the
	// same fix engage_test.go's own frozen-base test takes for the same
	// reason.
	w.entities[i].PostX, w.entities[i].PostY = 44, 44
	w.decide(w.aiGroups()[0])
	if _, held := engVictim(w, 1); held {
		t.Fatal("the fixture's own release did not fire — this test would pass for the wrong reason")
	}
	// The release leaves the member off its own post (5,5) — it had pursued
	// its victim before losing it — so the walk home must have given it that
	// post as a destination in the very same decide() call above.
	if e := w.entities[indexOfEntity(w.entities, 1)]; !e.HasTarget || e.TargetX != 5 || e.TargetY != 5 {
		t.Fatalf("the release did not walk the member home to (5,5) — this test would pass for the "+
			"wrong reason: HasTarget=%v (%d,%d)", e.HasTarget, e.TargetX, e.TargetY)
	}

	before := w.entities[indexOfEntity(w.entities, 1)]
	for k := 0; k < 200; k++ {
		Step(w, nil)
	}
	after := w.entities[indexOfEntity(w.entities, 1)]
	if after.X != 5 || after.Y != 5 {
		t.Errorf("the walked-home member is at (%d,%d), want its post (5,5)", after.X, after.Y)
	}
	if after.HasTarget {
		t.Errorf("the arrived member still holds a destination — FR-6 leaves an at-post, "+
			"victimless member in every field, so arriving must be where it stops: got (%d,%d)",
			after.TargetX, after.TargetY)
	}
	if after.HP != before.HP {
		t.Errorf("a released member's health changed from %d to %d — it struck or was struck", before.HP, after.HP)
	}

	// And it stays there (AC-2): several more decision cycles at the post
	// change nothing further — no oscillation, no re-issue.
	settled := after
	for k := 0; k < 64; k++ {
		Step(w, nil)
	}
	if final := w.entities[indexOfEntity(w.entities, 1)]; final != settled {
		t.Errorf("an arrived, released member drifted from %+v to %+v", settled, final)
	}
}

func TestAVictimThatStillScoresSurvivesASecondDecision(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}),
		engFighter(1, 2, 5, 5), engFighter(2, 3, 6, 5))
	w.engagementPass()
	before, held := engVictim(w, 1)
	if !held || before != 2 {
		t.Fatalf("the fixture's own first decision holds %v/%v, want entity 2", before, held)
	}
	beforeCountdown := w.entities[indexOfEntity(w.entities, 1)].AttackCountdown

	w.decide(w.aiGroups()[0])

	after, held := engVictim(w, 1)
	if !held || after != before {
		t.Errorf("a second decision moved the victim from %v to %v/%v, want it kept", before, after, held)
	}
	if got := w.entities[indexOfEntity(w.entities, 1)].AttackCountdown; got != beforeCountdown {
		t.Errorf("the attack countdown moved from %d to %d — the cycle restarted", beforeCountdown, got)
	}
}

func TestAWalkerWithNoVictimIsUntouchedByADecision(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 5, 5))
	i := indexOfEntity(w.entities, 1)
	w.entities[i].TargetX, w.entities[i].TargetY, w.entities[i].HasTarget = 20, 20, true
	w.entities[i].Stall = 3
	w.routes[i] = []cell{{x: 6, y: 5}, {x: 7, y: 5}}
	route := append([]cell(nil), w.routes[i]...)

	w.decide(aiGroup{owner: 2, group: 0, members: []int{i}})

	e := w.entities[indexOfEntity(w.entities, 1)]
	if !e.HasTarget || e.TargetX != 20 || e.TargetY != 20 {
		t.Errorf("destination is HasTarget=%v (%d,%d), want (20,20) kept", e.HasTarget, e.TargetX, e.TargetY)
	}
	if e.Stall != 3 {
		t.Errorf("stall count is %d, want 3 kept", e.Stall)
	}
	got := w.routes[indexOfEntity(w.entities, 1)]
	if len(got) != len(route) || got[0] != route[0] || got[1] != route[1] {
		t.Errorf("stored route is %v, want %v kept", got, route)
	}
}

func TestAnEmptyCandidateListReleasesOnlyTheVictimHolders(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 5, 5), engFighter(2, 2, 6, 5))
	ia, ib := indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 2)
	// A holds a stale attack order — this group has no hostile in sight, so no
	// decision in this run ever gave it one; the state is set directly to
	// isolate the empty-list release from the acquisition that would normally
	// precede it. The victim id is a dummy: the release clears it unread.
	w.entities[ia].AttackTarget, w.entities[ia].HasAttackTarget = 999, true
	w.entities[ia].AttackPhase, w.entities[ia].AttackCountdown = AttackReady, 5
	w.entities[ib].TargetX, w.entities[ib].TargetY, w.entities[ib].HasTarget = 20, 20, true
	w.entities[ib].Stall = 2

	w.decide(aiGroup{owner: 2, group: 0, members: []int{ia, ib}})

	a := w.entities[indexOfEntity(w.entities, 1)]
	if a.HasAttackTarget {
		t.Errorf("the victim-holder kept its victim %v under an empty candidate list", a.AttackTarget)
	}
	b := w.entities[indexOfEntity(w.entities, 2)]
	if !b.HasTarget || b.TargetX != 20 || b.TargetY != 20 || b.Stall != 2 {
		t.Errorf("the walker was touched: HasTarget=%v (%d,%d) stall=%d, want its destination kept",
			b.HasTarget, b.TargetX, b.TargetY, b.Stall)
	}
}

// A victim that stepped past reach leaves the participant's stand-ground member
// with a candidate list the scorer refused whole (AI-REACH-072). The member's arm
// stores the idle order for it (AI-349), so the next decision ends the pursuit
// (AI-353).
func TestAStandGroundMemberDropsAVictimThatStepsPastReach(t *testing.T) {
	t.Parallel()

	rel := engRel(t, [3]uint32{SelfSlot, 9, 1})
	w := engWorld(t, rel, engFighter(1, SelfSlot, 5, 5), engFighter(2, 9, 5+groupScorerReach, 5))
	w.engagementPass()
	if v, held := engVictim(w, 1); !held || v != 2 {
		t.Fatalf("the fixture's own first decision holds %v/%v, want entity 2 acquired", v, held)
	}
	i := indexOfEntity(w.entities, 2)
	w.entities[i].X += groupScorerReach + 1

	w.decide(w.aiGroups()[0])

	if v, held := engVictim(w, 1); held {
		t.Errorf("a stand-ground member kept victim %v once it stepped past reach", v)
	}
}

func TestAVetoedCandidateReleasesAHeldVictim(t *testing.T) {
	t.Parallel()

	build := func(candDomain Domain) *World {
		m := engFighter(1, 2, 5, 5)
		c := engFighter(2, 3, 6, 5)
		c.Domain = candDomain
		w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), m, c)
		w.engagementPass()
		if v, held := engVictim(w, 1); !held || v != 2 {
			t.Fatalf("the fixture's own first decision (domain %v) holds %v/%v, want entity 2", candDomain, v, held)
		}
		return w
	}

	t.Run("veto present releases it", func(t *testing.T) {
		t.Parallel()
		w := build(DomainGround)
		i := indexOfEntity(w.entities, 2)
		w.entities[i].Domain = DomainAir
		w.decide(w.aiGroups()[0])
		if v, held := engVictim(w, 1); held {
			t.Errorf("a ground member kept victim %v after it turned into a flier", v)
		}
	})
	t.Run("veto absent keeps it", func(t *testing.T) {
		t.Parallel()
		w := build(DomainGround)
		w.decide(w.aiGroups()[0])
		if v, held := engVictim(w, 1); !held || v != 2 {
			t.Errorf("without the veto a second decision moved the victim to %v/%v, want entity 2 kept", v, held)
		}
	})
}

func engManyCandidates(cx, cy int32, n int) []Entity {
	out := make([]Entity, 0, n)
	id := EntityID(2)
	for dy := int32(-8); dy <= 8 && len(out) < n; dy++ {
		for dx := int32(-8); dx <= 8 && len(out) < n; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			out = append(out, engFighter(id, 3, cx+dx, cy+dy))
			id++
		}
	}
	return out
}

func TestTheCandidateCountNarrowsToAByte(t *testing.T) {
	t.Parallel()

	build := func(n int) *World {
		member := engFighter(1, 2, 24, 24)
		member.ScanRange = 250
		ents := append([]Entity{member}, engManyCandidates(24, 24, n)...)
		return engWorld(t, engRel(t, [3]uint32{2, 3, 1}), ents...)
	}

	for _, tc := range []struct {
		name string
		n    int
		want bool
	}{
		{"255", 255, true},
		{"256", 256, false},
		{"257", 257, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := build(tc.n)
			w.engagementPass()
			if _, held := engVictim(w, 1); held != tc.want {
				t.Errorf("%d candidates: held=%v, want %v", tc.n, held, tc.want)
			}
		})
	}
}

func TestAReleasedMemberRoundTripsThroughTheByteForm(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}),
		engFighter(1, 2, 5, 5), engFighter(2, 3, 5+engSight, 5))
	engRun(w, 1)
	if _, held := engVictim(w, 1); !held {
		t.Fatal("the fixture's own first decision did not acquire a victim")
	}
	i := indexOfEntity(w.entities, 2)
	w.entities[i].X, w.entities[i].Y = 44, 44
	w.decide(w.aiGroups()[0])
	if _, held := engVictim(w, 1); held {
		t.Fatal("the fixture's own release did not fire — this test would pass for the wrong reason")
	}

	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary (round two): %v", err)
	}
	if string(again) != string(b) {
		t.Error("a released member's world does not round-trip through its byte form")
	}
}

func TestTheAlwaysScoringPopulationMatchesTheGoldenDigest(t *testing.T) {
	t.Parallel()

	victim := engFighter(2, 3, 5, 6)
	victim.HP, victim.MaxHP = 1<<20, 1<<20 // outlives the run
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 2}),
		engFighter(1, 2, 5, 5), victim)
	for i := 0; i < 300; i++ {
		Step(w, nil)
	}
	if !w.entities[indexOfEntity(w.entities, 1)].HasAttackTarget {
		t.Fatal("the attacker released its immortal, always-hostile victim — this test would pass for the wrong reason")
	}
	requireStructureLegacyDigest(t, w, 16252837679889806636)
	requireOriginalDeadLegacyDigest(t, w, 5983912820404504995)
	requireSpellbookLegacyDigest(t, w, 10556649293082217782)
	if got, want := fnv1a(strippedWorldOfSecondPhysical(mustMarshal(t, w))), uint64(15213618229883190643); got != want {
		t.Errorf("the digest is %d, want the golden %d captured for this story's own byte form", got, want)
	}
	requireHumanMovementLegacyDigest(t, w, 9935559246186814238)
}

func TestTheByteFormsVersionMatchesTheConstantAndAFixedWorldsDigestIsPinned(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}),
		engFighter(1, 2, 20, 20), engFighter(2, 3, 24, 20))
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if b[0] != formatVersion {
		t.Errorf("the form's version byte is %d, want the constant %d", b[0], formatVersion)
	}
	requireStructureLegacyDigest(t, w, 15932531532561939831)
	requireOriginalDeadLegacyDigest(t, w, 12339812177527668710)
	requireSpellbookLegacyDigest(t, w, 14095003147500693003)
	if got, want := fnv1a(strippedWorldOfSecondPhysical(b)), uint64(5520326590905100892); got != want {
		t.Errorf("the fixed world's digest is %d, want %d", got, want)
	}
	requireHumanMovementLegacyDigest(t, w, 13161292787049859749)
}

func TestACommandedAttackIsNotPrivilegedUnderGuard(t *testing.T) {
	t.Parallel()

	build := func(owner uint32) *World {
		victim := engFighter(2, 3, 5, 6)
		victim.HP, victim.MaxHP = 1<<20, 1<<20
		rel := engRel(t,
			[3]uint32{2, 3, 2}, [3]uint32{3, 2, 2},
			[3]uint32{SelfSlot, 3, 2}, [3]uint32{3, SelfSlot, 2})
		w := engWorld(t, rel, engFighter(1, owner, 5, 5), victim)
		Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 2}})
		return w
	}

	t.Run("guard releases it", func(t *testing.T) {
		t.Parallel()
		w := build(2)
		for i := 0; i < 100; i++ {
			Step(w, nil)
		}
		if got := w.Entities()[0]; got.HasAttackTarget {
			t.Error("a guard member kept a command-issued victim under a peaceful relation")
		}
	})
	t.Run("the exempt stance keeps it", func(t *testing.T) {
		t.Parallel()
		w := build(SelfSlot)
		for i := 0; i < 100; i++ {
			Step(w, nil)
		}
		if got := w.Entities()[0]; !got.HasAttackTarget {
			t.Error("the exempt stance released a command-issued victim")
		}
	})
}

// setGroupOrder overwrites the stored order on the (owner, group) record
// directly — the way a group command will once T2 lands — bypassing
// freezeGroups' own owner-derived default so a record can be put into a
// state no construction on this task reaches: an order that disagrees with
// what its owner would normally be given.
func setGroupOrder(t *testing.T, w *World, owner, group uint32, order uint8) {
	t.Helper()
	for i := range w.groups {
		if w.groups[i].owner == owner && w.groups[i].group == group {
			w.groups[i].order = order
			return
		}
	}
	t.Fatalf("no group record for (owner %d, group %d) — this test's fixture is wrong", owner, group)
}

func TestAReleaseKeysOnTheOwnerNotTheOrder(t *testing.T) {
	t.Parallel()

	t.Run("stand ground, owner not SelfSlot: released", func(t *testing.T) {
		t.Parallel()
		// No relation at all, so candidates() is empty and the release comes
		// from the empty-list site (~line 305, decide's first release).
		w := engWorld(t, engRel(t), engFighter(1, 2, 5, 5))
		ia := indexOfEntity(w.entities, 1)
		w.entities[ia].AttackTarget, w.entities[ia].HasAttackTarget = 999, true
		w.entities[ia].AttackPhase, w.entities[ia].AttackCountdown = AttackReady, 5
		setGroupOrder(t, w, 2, 0, orderStandGround)

		w.decide(aiGroup{owner: 2, group: 0, members: []int{ia}})

		if a := w.entities[indexOfEntity(w.entities, 1)]; a.HasAttackTarget {
			t.Errorf("owner 2 (not SelfSlot) kept a stale victim under Stand Ground — "+
				"the release must key on the owner, and this owner is not exempt: got %v",
				a.AttackTarget)
		}
	})

	t.Run("guard, owner is SelfSlot: kept", func(t *testing.T) {
		t.Parallel()
		// A live, hostile candidate the preference table vetoes outright (a
		// ground member, an air candidate) stays IN the candidate list — the
		// clip only drops by distance — so this member reaches the per-member
		// release (~line 332, decide's second release) rather than the
		// empty-list one.
		m := engFighter(1, SelfSlot, 5, 5)
		c := engFighter(2, 9, 7, 5)
		c.Domain = DomainAir
		w := engWorld(t, engRel(t, [3]uint32{SelfSlot, 9, 1}), m, c)
		ia := indexOfEntity(w.entities, 1)
		w.entities[ia].AttackTarget, w.entities[ia].HasAttackTarget = 999, true
		w.entities[ia].AttackPhase, w.entities[ia].AttackCountdown = AttackReady, 5
		setGroupOrder(t, w, SelfSlot, 0, orderGuard)

		w.decide(aiGroup{owner: SelfSlot, group: 0, members: []int{ia}})

		a := w.entities[indexOfEntity(w.entities, 1)]
		if !a.HasAttackTarget || a.AttackTarget != 999 {
			t.Errorf("SelfSlot's own group lost a stale victim under Guard — "+
				"the release must key on the owner, and SelfSlot is exempt: HasAttackTarget=%v AttackTarget=%v",
				a.HasAttackTarget, a.AttackTarget)
		}
	})
}

// simSourceFiles returns the package directory's own non-test .go file
// names, sorted — the scope every scan below runs over. Tests run with the
// package directory as the working directory, so "." is the package.
func simSourceFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var out []string
	for _, ent := range entries {
		name := ent.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// simFuncs calls visit once for every top-level function declared in the
// package's own non-test source files. skip, if non-empty, names one file to
// leave out of the scan.
func simFuncs(t *testing.T, skip string, visit func(name string, body *ast.BlockStmt)) {
	t.Helper()
	fset := token.NewFileSet()
	for _, name := range simSourceFiles(t) {
		if name == skip {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("ParseFile %s: %v", name, err)
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			visit(fd.Name.Name, fd.Body)
		}
	}
}

// funcsAssigning returns the sorted, deduplicated names of top-level
// functions whose body assigns the literal identifier lit to a selector
// named sel, in lockstep position — the same shape internal/archtest's own
// destination scan uses for HasTarget (single-valued or one arm of a
// multi-value assignment), applied here to HasAttackTarget without touching
// that package. skip, if non-empty, names a file to leave out.
func funcsAssigning(t *testing.T, sel, lit, skip string) []string {
	t.Helper()
	found := map[string]bool{}
	simFuncs(t, skip, func(name string, body *ast.BlockStmt) {
		ast.Inspect(body, func(n ast.Node) bool {
			as, ok := n.(*ast.AssignStmt)
			if !ok || as.Tok != token.ASSIGN || len(as.Lhs) != len(as.Rhs) {
				return true
			}
			for i, lhs := range as.Lhs {
				se, ok := lhs.(*ast.SelectorExpr)
				if !ok || se.Sel.Name != sel {
					continue
				}
				if id, ok := as.Rhs[i].(*ast.Ident); ok && id.Name == lit {
					found[name] = true
				}
			}
			return true
		})
	})
	out := make([]string, 0, len(found))
	for k := range found {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// funcCall reports the name a call expression invokes, by identifier or by
// method/field selector, and "" for anything else (e.g. a call through a
// value, which none of the calls this file scans for are written as).
func funcCall(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		return fn.Sel.Name
	}
	return ""
}

// funcsCalling returns the sorted, deduplicated names of top-level functions
// (other than callee itself) whose body calls a function or method named
// callee.
func funcsCalling(t *testing.T, callee string) []string {
	t.Helper()
	found := map[string]bool{}
	simFuncs(t, "", func(name string, body *ast.BlockStmt) {
		if name == callee {
			return
		}
		ast.Inspect(body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok && funcCall(call) == callee {
				found[name] = true
			}
			return true
		})
	})
	out := make([]string, 0, len(found))
	for k := range found {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// bodyCalls reports whether funcName's own body — not its callees' —
// contains a call to callee.
func bodyCalls(t *testing.T, funcName, callee string) bool {
	t.Helper()
	seen, got := false, false
	simFuncs(t, "", func(name string, body *ast.BlockStmt) {
		if name != funcName {
			return
		}
		seen = true
		ast.Inspect(body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok && funcCall(call) == callee {
				got = true
			}
			return true
		})
	})
	if !seen {
		t.Fatalf("function %s not found in package sources", funcName)
	}
	return got
}

func TestOrderAttackIsTheOnlySetterAndDecidesOnlyClearerIsTheRelease(t *testing.T) {
	t.Parallel()

	if setters := funcsAssigning(t, "HasAttackTarget", "true", "binary.go"); len(setters) != 2 || setters[0] != "ImportOriginalActorActions" || setters[1] != "attachAttack" {
		t.Errorf("HasAttackTarget = true is written (outside binary.go) by %v, want exactly [ImportOriginalActorActions attachAttack]", setters)
	}
	if callers := funcsCalling(t, "attachAttack"); len(callers) != 4 || callers[0] != "armCreatureApproach" || callers[1] != "cmdGroupAttack" || callers[2] != "orderAttack" || callers[3] != "stepWorld" {
		t.Errorf("attachAttack is called by %v, want exactly [armCreatureApproach cmdGroupAttack orderAttack stepWorld]", callers)
	}
	if callers := funcsCalling(t, "releaseAttack"); len(callers) != 2 || callers[0] != "decide" || callers[1] != "savedDecision" {
		t.Errorf("releaseAttack is called by %v, want exactly [decide savedDecision]", callers)
	}
	if bodyCalls(t, "decide", "clearAttack") {
		t.Error("decide's own body reaches clearAttack directly, not only through releaseAttack")
	}
}
