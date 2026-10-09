package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Argument parsing and argument refusals run without a game. Installed
// mission and save tests require their asset and corpus roots.

func TestAWaypointIsREFXYRadius(t *testing.T) {
	got, err := parseWaypoint("u21:56:21:3")
	if err != nil {
		t.Fatalf("parseWaypoint: %v", err)
	}
	want := waypoint{kind: refScript, index: 21, x: 56, y: 21, radius: 3}
	if got != want {
		t.Errorf("parseWaypoint = %+v, want %+v", got, want)
	}
	got, err = parseWaypoint("p0:-4:16:0")
	if err != nil {
		t.Fatalf("parseWaypoint: %v", err)
	}
	want = waypoint{kind: refParty, index: 0, x: -4, y: 16, radius: 0}
	if got != want {
		t.Errorf("parseWaypoint = %+v, want %+v", got, want)
	}
}

func TestAMalformedWaypointIsRefused(t *testing.T) {
	for _, s := range []string{
		"", "u21", "u21:56:21", "u21:56:21:3:4",
		"x21:56:21:3", // neither naming
		"u:56:21:3",   // no index
		"u-1:56:21:3", // a negative index
		"u21:x:21:3",  // a coordinate that is not a number
		"u21:56:21:-1",
	} {
		if _, err := parseWaypoint(s); err == nil {
			t.Errorf("parseWaypoint(%q) was accepted", s)
		}
	}
}

// TestTheRefusalsHappenBeforeAnArchiveIsOpened is what keeps this file synthetic:
// each case must fail on its arguments, so the asset root is deliberately set to
// a directory that holds no install and none of them may reach it.
func TestTheRefusalsHappenBeforeAnArchiveIsOpened(t *testing.T) {
	t.Setenv("AGAINROM_ASSETS", t.TempDir())
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"no mission", []string{}, "-mission"},
		{"a mission number that is not one", []string{"-mission", "-3"}, "-mission"},
		{"a tick ceiling of nothing", []string{"-mission", "1", "-ticks", "0"}, "-ticks"},
		{"a malformed attack", []string{"-mission", "1", "-attack", "p0"}, "attack"},
		{"a malformed take", []string{"-mission", "1", "-take", "p0:5"}, "take"},
		{"a waypoint that will not parse", []string{"-mission", "1", "-waypoint", "u21:56:21"}, "waypoint"},
		{"an unknown flag", []string{"-mission", "1", "-nosuchflag"}, "flag"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := run(tc.args, &buf)
			if err == nil {
				t.Fatalf("run(%v) returned no error; it printed %q", tc.args, buf.String())
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("run(%v) = %v, want an error naming %q", tc.args, err, tc.want)
			}
		})
	}
}

func TestAnAbsentAssetRootIsNamedRatherThanGuessed(t *testing.T) {
	t.Setenv("AGAINROM_ASSETS", "")
	var buf bytes.Buffer
	err := run([]string{"-mission", "1"}, &buf)
	if err == nil || !strings.Contains(err.Error(), "asset root") {
		t.Fatalf("run with no asset root = %v, want an error naming the asset root", err)
	}
}

func TestMageIsARecognisedFlag(t *testing.T) {
	t.Setenv("AGAINROM_ASSETS", "")
	var buf bytes.Buffer
	err := run([]string{"-mission", "1", "-mage"}, &buf)
	if err == nil || !strings.Contains(err.Error(), "asset root") {
		t.Fatalf("run with -mage and no asset root = %v, want an error naming the asset root "+
			"(not a flag-parsing error, which would mean -mage is not registered)", err)
	}
}

func TestReleaseMissionRunRefusesBetweenMissionSAV(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" || os.Getenv("AGAINROM_SAVE_CORPUS") == "" {
		t.Skip("AGAINROM_ASSETS and AGAINROM_SAVE_CORPUS are required")
	}
	path := filepath.Join(os.Getenv("AGAINROM_SAVE_CORPUS"), "2026-08-15", "game0010.sav")
	var out bytes.Buffer
	err := run([]string{"-sav", path, "-ticks", "1"}, &out)
	if err == nil || err.Error() != "-sav is between missions; missionrun requires a mission save" {
		t.Fatalf("between-mission SAV drive = %v; output %q", err, out.String())
	}
	if out.Len() != 0 {
		t.Fatalf("refused town SAV drove a mission: %s", out.String())
	}
}

// TestTheTenthMissionRunsEndToEndOnALawfulInstall drives the tenth mission on a
// real install and asserts that the whole stack runs: the map loads, the script
// compiles, a move order is carried out, and the trigger machinery decides.
//
// IT DOES NOT ASSERT A WIN, AND A `lost` HERE IS NOT A DEFECT IN THE GAME. It
// used to, under the name TestTheTenthMissionIsDrivenToAWin, and that assertion
// was false in a way worth writing down because it stood for eight days and
// twenty-three recorded drives.
//
// The tenth mission is an ESCORT. Script unit 21 is the subject of a
// check-18 (VIP) node, so its death increments the lose counter directly,
// and the map's own hostiles hunt it from cells nearer than the party's
// start: driven or left alone, u21 dies — at tick 214 when this drive
// marches it across the map, at tick 464 when nothing orders it at all.
// Winning the mission means intercepting and fighting, which is play, not a
// waypoint list. So an unattended drive of this mission loses BY THE
// MISSION'S OWN DESIGN, and reading that `lost` as "the game is broken" is
// reading a property of the drive as a property of the game.
//
// What is still worth running here is everything the outcome is not: 36 entities
// instantiate off a lawful install, the script compiles to its checks/instants/
// triggers, a party move order produces steps, and the reporter reaches a decided
// outcome rather than hanging undecided. Each of those goes red on a real
// regression. The distance to a SIMULATED win is measured somewhere it can only
// improve — the unsupported-instant census in pipeline/check-milestone.sh — and
// not by an outcome word.
//
// It reads a LAWFUL INSTALL and is therefore skipped unless an asset root is
// configured; the suite stays green with no game present. The drive itself is
// unchanged from the assertion this replaces — thirty-three verification.md
// files record this exact one, and a smoke test measuring a different drive
// cannot be compared with any of them. The `-census` flag is new HERE and was
// always on the seat gate's argv: it adds the moved/fell report and, with
// waypoints present, changes not one tick of the drive. check-milestone.sh
// claimed in a comment that the two argvs were identical and they were not;
// after this they are.
func TestTheTenthMissionRunsEndToEndOnALawfulInstall(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("no AGAINROM_ASSETS: the campaign drive needs a lawful install")
	}
	var buf bytes.Buffer
	args := []string{"-mission", "10", "-census", "-waypoint", "u21:56:21:3", "-waypoint", "p0:66:16:3"}
	if err := run(args, &buf); err != nil {
		t.Fatalf("run(%v): %v\n%s", args, err, buf.String())
	}
	got := buf.String()
	t.Log("\n" + got)
	for _, want := range []string{
		"mission 10  scenario/10.alm", // the map loaded off the install
		"waypoint 1  ",                // the drive issued its first order
		"moved  ",                     // at least one unit carried one out
		"outcome ",                    // the reporter decided rather than hanging
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the drive printed no %q line, so that stage did not run:\n%s", want, got)
		}
	}
	if strings.Contains(got, "outcome undecided") {
		t.Errorf("the drive ended undecided: the trigger machinery reached no report:\n%s", got)
	}
}

// An attack is ATTACKER:VICTIM in the same two namings the waypoints use, and
// the grammar is shared so the two cannot drift apart.
func TestAnAttackIsAttackerColonVictim(t *testing.T) {
	got, err := parseStrike("p0:u21")
	if err != nil {
		t.Fatalf("parseStrike: %v", err)
	}
	want := strike{
		attacker: waypoint{kind: refParty, index: 0},
		victim:   waypoint{kind: refScript, index: 21},
	}
	if got != want {
		t.Errorf("parseStrike = %+v, want %+v", got, want)
	}
}

func TestAMalformedAttackIsRefused(t *testing.T) {
	for _, s := range []string{
		"", "p0", "p0:u21:3",
		"x0:u21", // neither naming on the attacker
		"p0:x21", // neither naming on the victim
		"p:u21",  // no index
		"p0:u-1", // a negative index
		"p0:uX",  // an index that is not a number
	} {
		if _, err := parseStrike(s); err == nil {
			t.Errorf("parseStrike(%q) was accepted", s)
		}
	}
}

// A take is TAKER:X:Y — the taker named by the same reference grammar the
// waypoints and the attacks use, the cell to take from as two plain numbers.
func TestATakeIsTakerColonXColonY(t *testing.T) {
	got, err := parseTake("p0:5:6")
	if err != nil {
		t.Fatalf("parseTake: %v", err)
	}
	want := take{taker: waypoint{kind: refParty, index: 0}, x: 5, y: 6}
	if got != want {
		t.Errorf("parseTake = %+v, want %+v", got, want)
	}
	got, err = parseTake("u21:-4:16")
	if err != nil {
		t.Fatalf("parseTake: %v", err)
	}
	want = take{taker: waypoint{kind: refScript, index: 21}, x: -4, y: 16}
	if got != want {
		t.Errorf("parseTake = %+v, want %+v", got, want)
	}
}

// TestAMalformedTakeIsRefused refuses the same shapes parseWaypoint and
// parseStrike refuse, on parseRef's own shared ground (0138 T3 "Done when":
// "refuses the same malformed shapes they do").
func TestAMalformedTakeIsRefused(t *testing.T) {
	for _, s := range []string{
		"", "u21", "u21:5", // too few fields
		"u21:5:6:7", // too many fields
		"x21:5:6",   // neither naming
		"u:5:6",     // no index
		"u-1:5:6",   // a negative index
		"u21:x:6",   // an x that is not a number
		"u21:5:y",   // a y that is not a number
	} {
		if _, err := parseTake(s); err == nil {
			t.Errorf("parseTake(%q) was accepted", s)
		}
	}
}

// TestCarriedLineNamesElementsWithACountBesideAnyAboveOne is T3's own report
// witness: the line victimHoldings' "before, carried:" and the take arm's
// own report share, built here against a SYNTHETIC world — no install
// anywhere — through the same constructor pkg/sim's own carry tests use
// (NewStockedWorld). A Stock repeating one code three times folds to one
// element at count 3, and carriedLine must spell that as the item once,
// followed by " [x3]" — the count as a bracketed token of quantity — while
// the second, unrepeated code carries no suffix at all.
func TestCarriedLineNamesElementsWithACountBesideAnyAboveOne(t *testing.T) {
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 1, X: 1, Y: 1}, {ID: 2, X: 2, Y: 2}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 1, Items: []uint16{0x0101, 0x0101, 0x0101, 0x0202}}})
	if err != nil {
		t.Fatalf("sim.NewStockedWorld: %v", err)
	}

	stacks, ok := w.CarriedStacks(1)
	if !ok || len(stacks) != 2 || stacks[0].Count != 3 || stacks[1].Count != 1 {
		t.Fatalf("CarriedStacks(1) = %+v, %v, want two elements at counts 3 and 1", stacks, ok)
	}
	want := "  label: " + formatItem(stacks[0].Code, nil) + " [x3] " + formatItem(stacks[1].Code, nil) + "\n"
	if got := carriedLine("  label", w, 1, nil); got != want {
		t.Errorf("carriedLine = %q, want %q", got, want)
	}

	// An actor holding nothing prints "none" and no count of any kind.
	if got := carriedLine("  label", w, 2, nil); got != "  label: none\n" {
		t.Errorf("carriedLine(empty) = %q, want the none line", got)
	}
}

type fakeScaleTable struct{ names []string }

func (f fakeScaleTable) Len() int { return len(f.names) }
func (f fakeScaleTable) EntryName(i int) string {
	if i < 0 || i >= len(f.names) {
		return ""
	}
	return f.names[i]
}
func (f fakeScaleTable) EntryDoubles(int) []float64 { return nil } // scale()'s own "absent" arm reads this as identity

type fakeCollection struct {
	names  []string
	params [][]int32
}

func (f fakeCollection) Len() int { return len(f.names) }
func (f fakeCollection) EntryName(i int) string {
	if i < 0 || i >= len(f.names) {
		return ""
	}
	return f.names[i]
}
func (f fakeCollection) EntryParams(i int) []int32 {
	if i < 0 || i >= len(f.params) {
		return nil
	}
	return f.params[i]
}
func (f fakeCollection) EntryStrings(int) []string { return nil }

// TestFormatItemNamesTheCodeTheSlotAndTheWeapon is formatItem's own witness:
// it is a pure function over a code and a table, so it is exercised here
// with no install anywhere — a synthetic Weapons row for the "resolves as
// a weapon" arm, and a code the tables refuse before either collection is
// read for the "resolves as nothing" arm.
func TestFormatItemNamesTheCodeTheSlotAndTheWeapon(t *testing.T) {
	shapes := fakeScaleTable{names: []string{""}}    // index 0: the reserved, unnamed entry
	materials := fakeScaleTable{names: []string{""}} // same
	weapons := fakeCollection{
		names: []string{"", "Sword"},
		// Row 1's 14 cells: only PhysicalMin/Max (5, 8), ToHit (1), Range (3),
		// Charge (4) and Relax (6) matter for this test, at ResolveWeapon's
		// own column numbers (weapon.go's weaponXxxSlot constants).
		params: [][]int32{nil, {0, 0, 0, 0, 0, 0, 5, 8, 1, 0, 0, 3, 4, 6}},
	}
	table := &mapload.Table{Shapes: shapes, Materials: materials, Weapons: weapons}

	// Field A=0 (material index 0, unnamed), B=1 (data's weapon item class),
	// C=0 (shape index 0, unnamed), D=1 (the Sword row) — the exact
	// composition data.Weapon.Code documents.
	const weaponCode = uint16(1<<8 | 1)
	got := formatItem(weaponCode, table)
	if !strings.HasPrefix(got, "0001001(slot 1)") {
		t.Errorf("formatItem(%04x, table) = %q, want it to start with the seven-digit name and slot 1", weaponCode, got)
	}
	if !strings.Contains(got, `"Sword"`) {
		t.Errorf("formatItem(%04x, table) = %q, want the resolved weapon's name", weaponCode, got)
	}

	// B=3 names no weapon class at all, so data.WeaponFromCode refuses it
	// before either table is read.
	const unresolvedCode = uint16(3 << 8)
	got = formatItem(unresolvedCode, table)
	if !strings.HasPrefix(got, "0003000(slot 3)") {
		t.Errorf("formatItem(%04x, table) = %q, want the seven-digit name and slot even unresolved", unresolvedCode, got)
	}
	if strings.Contains(got, `"`) {
		t.Errorf("formatItem(%04x, table) = %q, want no weapon name for a code the tables refuse", unresolvedCode, got)
	}

	// A nil table — no install loaded at all — takes the same "nothing
	// resolved" arm rather than dereferencing anything.
	if got := formatItem(weaponCode, nil); strings.Contains(got, `"`) {
		t.Errorf("formatItem(%04x, nil) = %q, want no weapon name with no table", weaponCode, got)
	}
}

// The instrument's own honesty: absent is not dead.
//
// hpOf answered a bare 0 for an entity the world does not hold, and every caller
// tests health against zero to decide whether a unit has fallen — so the one
// input that means the drive was wrong produced the strongest positive result
// this tool can print. These run on a hand-built world with no install present.

// absentWorld is one live entity at full health, one downed at exactly zero and
// one dead below it — the three states a reader must tell apart from absence.
func absentWorld(t *testing.T) *sim.World {
	t.Helper()
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 1, X: 1, Y: 1, HP: 40, MaxHP: 40},
		{ID: 2, X: 2, Y: 1, HP: 0, MaxHP: 40},
		{ID: 3, X: 3, Y: 1, HP: -7, MaxHP: 40},
	})
	if err != nil {
		t.Fatalf("sim.NewWorld: %v", err)
	}
	return w
}

func TestAbsentIsDistinguishableFromDowned(t *testing.T) {
	w := absentWorld(t)
	for _, tc := range []struct {
		id   sim.EntityID
		hp   int32
		held bool
	}{
		{1, 40, true},
		{2, 0, true},  // downed: zero is a health a unit really carries
		{3, -7, true}, // dead: still held, and how far below zero survives
		{9, 0, false}, // absent: the same first value the downed unit gives
	} {
		hp, held := hpOf(w, tc.id)
		if hp != tc.hp || held != tc.held {
			t.Errorf("hpOf(%d) = (%d, %v), want (%d, %v)", tc.id, hp, held, tc.hp, tc.held)
		}
	}
	if got := hpText(w, 2); got != "0 hp" {
		t.Errorf("hpText(downed) = %q, want the number", got)
	}
	if got := hpText(w, 9); got != "ABSENT" {
		t.Errorf("hpText(absent) = %q, want the word", got)
	}
}

func TestAnAbsentVictimNeverFalls(t *testing.T) {
	w := absentWorld(t)
	before := w.Tick()
	spent, fell := strikeDown(newTracer(false, io.Discard, nil, nil), w, 1, 9, 500)
	if fell {
		t.Errorf("strikeDown onto an absent victim reported a kill after %d ticks", spent)
	}
	if spent > 1 {
		t.Errorf("strikeDown onto an absent victim spent %d ticks; it cannot progress", spent)
	}
	if w.Tick() == before {
		t.Errorf("the order was never issued, so this measured nothing")
	}
	// The downed one, by contrast, HAS fallen and must still say so.
	if _, fell := strikeDown(newTracer(false, io.Discard, nil, nil), w, 1, 2, 500); !fell {
		t.Errorf("strikeDown onto a downed victim reported no kill")
	}
}

func TestADriveNamingAnAbsentUnitFailsBeforeAnyOrder(t *testing.T) {
	w := absentWorld(t)
	// Two placed units, so party slot 0 is entity 2 and slot 7 is entity 9 —
	// a well-formed id naming nothing, which is exactly the shape that lied.
	ms := &game.Mission{Map: &alm.Map{Units: make([]alm.Unit, 2)}, World: w}
	script := map[uint16]sim.EntityID{21: 1, 22: 9}

	for _, tc := range []struct {
		name string
		p    waypoint
		want string
	}{
		{"a party slot past the party's end", waypoint{kind: refParty, index: 7}, "p7"},
		{"a script unit the world never held", waypoint{kind: refScript, index: 22}, "u22"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := w.Tick()
			id, err := resolve(ms, script, tc.p)
			if err == nil {
				t.Fatalf("resolve = %d with no error", id)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("resolve = %v, want an error naming %q", err, tc.want)
			}
			if w.Tick() != before {
				t.Errorf("the refusal spent a tick")
			}
		})
	}

	// And a reference the world does hold still resolves, by both namings.
	if id, err := resolve(ms, script, waypoint{kind: refParty, index: 0}); err != nil || id != 2 {
		t.Errorf("resolve(p0) = (%d, %v), want (2, nil)", id, err)
	}
	if id, err := resolve(ms, script, waypoint{kind: refScript, index: 21}); err != nil || id != 1 {
		t.Errorf("resolve(u21) = (%d, %v), want (1, nil)", id, err)
	}
}

// A WALK CUT SHORT BY THE WORLD DECIDING IS NOT A WALK THAT ARRIVED.
//
// reach answered true on that input — the one value its caller reads to decide
// whether the drive got there — so a mission decided mid-walk printed "reached
// (43,46), Chebyshev 25" for a waypoint 25 cells away. That line is where the
// tenth mission's loss was read as an interception short of its waypoint, a
// cause no measurement ever supported.
func TestAWalkEndedByTheDecisionDoesNotClaimTheRadius(t *testing.T) {
	s, err := sim.NewScript(
		[]sim.ScriptCheck{{Op: sim.ScriptCheckConstant, Register: 0}},
		[]sim.ScriptInstant{{Op: sim.ScriptInstantLose}},
		[]sim.ScriptTrigger{{
			Pairs:    [3]sim.ScriptPair{{Left: 0, Right: 0, Cmp: sim.ScriptCmpEQ, Used: true}},
			Instants: [4]int32{0, -1, -1, -1},
			Once:     true,
		}},
	)
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: 64, Height: 64}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 1, X: 0, Y: 0, HP: 10, MaxHP: 10}}, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	open := func(x, y int32) bool { return x < 0 || y < 0 || x >= 64 || y >= 64 }
	p := waypoint{kind: refScript, index: 1, x: 63, y: 63, radius: 1}
	spent, ok := reach(newTracer(false, io.Discard, nil, nil), w, 1, p, open, 64, 64, 500)
	if w.Outcome() == sim.OutcomeUndecided {
		t.Fatalf("the world never decided in %d ticks; this test measures nothing", spent)
	}
	if ok {
		t.Errorf("reach reported the radius met after %d ticks; the unit is at %v and the point "+
			"is (63,63)", spent, []int32{mustX(w), mustY(w)})
	}
}

func mustX(w *sim.World) int32 { x, _ := at(w, 1); return x }
func mustY(w *sim.World) int32 { _, y := at(w, 1); return y }

// TestTheCensusCountsAUnitThatCameBack — the census counts tick-to-tick cell
// changes, not the difference between two snapshots.
//
// This is the whole reason it is a sampler and not a before/after diff. A unit
// walked out and walked home has moved, and a patrolling unit does exactly that
// on a two-node ring; a snapshot compare would report the map as still. The
// owner's count of the original — two units move, both peasants in the village —
// is a count of that kind, so ours has to be as well.
func TestTheCensusCountsAUnitThatCameBack(t *testing.T) {
	w := absentWorld(t)
	c := newCensus(true, io.Discard, nil, w)
	tr := newTracer(false, io.Discard, nil, nil)
	tr.watch(c)

	walk := func(x, y int32) {
		for i := 0; i < 200; i++ {
			if e := entityAt(w, 1); e.X == x && e.Y == y {
				return
			}
			tr.step(w, []sim.Command{{Kind: sim.KindMoveTo, Entity: 1, X: x, Y: y}})
		}
		t.Fatalf("unit 1 never reached (%d,%d)", x, y)
	}
	walk(4, 4)
	walk(1, 1)

	if e := entityAt(w, 1); e.X != 1 || e.Y != 1 {
		t.Fatalf("unit 1 ended at (%d,%d), not where it began: the test is not the one described", e.X, e.Y)
	}
	if c.steps[1] < 2 {
		t.Errorf("census counted %d step(s) for a unit that walked out and home; a snapshot "+
			"compare would have counted 0, which is what this test exists to refuse", c.steps[1])
	}
	for _, id := range []sim.EntityID{2, 3} {
		if c.steps[id] != 0 {
			t.Errorf("census counted %d step(s) for unit %d, which was never ordered anywhere", c.steps[id], id)
		}
	}
}

// entityAt is the entity the world holds, or the zero value.
func entityAt(w *sim.World, id sim.EntityID) sim.Entity {
	for _, e := range w.Entities() {
		if e.ID == id {
			return e
		}
	}
	return sim.Entity{}
}

// TestAnOrderIsVerbColonRefWithACellOnlyWhereTheVerbTakesOne is 0146's own
// grammar, and it is a pure test over the string: every way it can be wrong is
// witnessed with no install present.
func TestAnOrderIsVerbColonRefWithACellOnlyWhereTheVerbTakesOne(t *testing.T) {
	for _, tc := range []struct {
		in       string
		wantKind uint8
		wantX    int32
		wantY    int32
	}{
		{"guard:u21", sim.KindGroupStance, sim.OrderGuard, 0},
		{"stand:p0", sim.KindGroupStance, sim.OrderStandGround, 0},
		{"patrol:u21:56:21", sim.KindGroupPatrolTo, 56, 21},
		{"march:p2:8:9", sim.KindGroupSwarmTo, 8, 9},
	} {
		got, err := parseStanding(tc.in)
		if err != nil {
			t.Errorf("parseStanding(%q): %v", tc.in, err)
			continue
		}
		if got.kind != tc.wantKind || got.ref.x != tc.wantX || got.ref.y != tc.wantY {
			t.Errorf("parseStanding(%q) = kind %d (%d,%d), want kind %d (%d,%d)",
				tc.in, got.kind, got.ref.x, got.ref.y, tc.wantKind, tc.wantX, tc.wantY)
		}
	}
}

// TestAMalformedOrderIsRefused is the same grammar's refusals. A cell handed to
// a verb that names none is refused rather than ignored: it would otherwise
// read as an order aimed somewhere, which is exactly what it is not.
func TestAMalformedOrderIsRefused(t *testing.T) {
	for _, in := range []string{
		"", "guard", "hold:u1", "guard:u1:2:3", "patrol:u1", "patrol:u1:2",
		"march:x1:2:3", "patrol:u1:two:3",
	} {
		if _, err := parseStanding(in); err == nil {
			t.Errorf("parseStanding(%q) was accepted", in)
		}
	}
}
