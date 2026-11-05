package sim

import (
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	updateCastRequest = flag.Bool("update-cast-request", false,
		"rewrite testdata/castrequest.txt from the current code")
	dumpCastRequest = flag.String("cast-request-dump", "",
		"write each cast request record's whole transcript into this directory")
)

// castRequestCase is one cast issue or one action that waits for a cast: the
// world it starts from, the caster whose cast and order state is recorded,
// the setup run after the game's profile is installed (its direct calls and
// steps are recorded too), the commands given on tick 0 and the commands a
// later tick gives.
type castRequestCase struct {
	name   string
	caster EntityID
	ticks  int
	world  func(*testing.T) *World
	start  func(*testing.T, *World, *castRequestRecorder) []Command
	then   func(*testing.T, *World, int) []Command
}

// castRequestRecorder writes one line per recorded moment: the World hash,
// the random state and the caster's cast and order state.
type castRequestRecorder struct {
	b      bytes.Buffer
	caster EntityID
}

func (r *castRequestRecorder) note(w *World, label string) {
	fmt.Fprintf(&r.b, "%s hash %016x rng %016x", label, w.Hash(), w.rng.state)
	i := indexOfEntity(w.entities, r.caster)
	if i < 0 {
		r.b.WriteString(" caster absent\n")
		return
	}
	e := w.entities[i]
	fmt.Fprintf(&r.b, " phase %d", w.orderStateAt(i).phase())
	if k, ok := w.bookCastIndex(e.ID); ok {
		fmt.Fprintf(&r.b, " book %+v", w.bookCasts[k])
	}
	if k, ok := w.scrollIndex(e.ID); ok {
		c := w.scrollCasts[k]
		fmt.Fprintf(&r.b, " scroll %d/%d/%d/%d/%v/%v/%d", c.Target, c.X, c.Y, c.Index, c.AtCell, c.Started, c.Remaining)
	}
	fmt.Fprintf(&r.b, " wait %d pending %+v attack %d/%d/%v/%d/%d walk %v/%d/%d at %d/%d/%d state %d retreat %+v spell %d mana %d hp %d\n",
		e.CastWait, e.PendingOrder, e.AttackPhase, e.AttackCountdown, e.HasAttackTarget, e.AttackTargetKind, e.AttackTarget,
		e.HasTarget, e.TargetX, e.TargetY, e.X, e.Y, e.Transit, e.ActorState, e.Retreat, e.AdmittedBookSpell, e.Mana, e.HP)
}

// step runs one setup tick and records it.
func (r *castRequestRecorder) step(w *World, cmds []Command) {
	Step(w, cmds)
	r.note(w, fmt.Sprintf("setup %d", w.tick))
}

// call records a direct issue and its answer.
func (r *castRequestRecorder) call(w *World, label string, ok bool) {
	r.note(w, fmt.Sprintf("%s %v", label, ok))
}

// crLoadStrike steps until caster 1's attack on victim has loaded its strike.
func crLoadStrike(t *testing.T, w *World, r *castRequestRecorder, victim EntityID) {
	t.Helper()
	ci := indexOfEntity(w.entities, 1)
	r.call(w, "plain engage", w.orderAttack(ci, victim))
	for n := 0; w.entities[ci].AttackPhase == AttackReady && w.entities[ci].AttackCountdown == 0; n++ {
		if n > 400 {
			r.note(w, "strike not loaded")
			return
		}
		r.step(w, nil)
	}
}

// crUntilRetainedRecovery steps until caster 1's retained book record
// reaches its recovery.
func crUntilRetainedRecovery(t *testing.T, w *World, r *castRequestRecorder) {
	t.Helper()
	for range 400 {
		r.step(w, nil)
		if k, ok := w.bookCastIndex(1); ok && w.bookCasts[k].Phase == bookRelaxing && w.bookCasts[k].Retained {
			return
		}
	}
	t.Fatal("the cast never reached a retained recovery")
}

func crManualWorld(t *testing.T) *World { return manualCastWorld(t) }

// crManualScrollWorld is the manual caster carrying one scroll of spell 1.
func crManualScrollWorld(t *testing.T) *World {
	w := manualCastWorld(t)
	w.carried[0] = []ItemStack{StackItem(ItemInstance{Code: 0xe10, Kind: 4, Price: 50,
		Effects: []ItemEffect{{Kind: 41, Operand: 1 | 60<<16}}}, 2)}
	return w
}

func crAt(n int, cmds ...Command) func(*testing.T, *World, int) []Command {
	return func(_ *testing.T, _ *World, tick int) []Command {
		if tick == n {
			return cmds
		}
		return nil
	}
}

func crCast(_ *testing.T, _ *World, _ *castRequestRecorder) []Command {
	return []Command{Cast(1, 2, 1)}
}

func crCreatureCertainWorld(t *testing.T) *World {
	book, known := creatureBookOf(map[uint32]uint8{1: 7})
	return creatureWorld(t, []SpellRule{hlArrow()}, creatureCertain, known, book)
}

func crMageRecoveryWorld(t *testing.T) *World {
	other := hlArrow()
	other.ID = 3
	return engageMageWorld(t, 1<<1|1<<3, 100, hlArrow(), other)
}

// crMageRecovery draws for a mage whose retained spell-1 order is in its
// recovery, on seed's stream.
func crMageRecovery(seed uint64) func(*testing.T, *World, *castRequestRecorder) []Command {
	return func(t *testing.T, w *World, r *castRequestRecorder) []Command {
		w.entities[0].KnownSpells = 1 << 1
		r.call(w, "engage", w.orderAttack(0, 2))
		crUntilRetainedRecovery(t, w, r)
		w.entities[0].KnownSpells = 1 << 3
		w.entities[0].Mind = 60
		w.rng = rng{state: seed}
		r.call(w, "draw in recovery", w.orderAttack(0, 3))
		return nil
	}
}

// crMageHandBack draws for a mage whose strike holds a cast order, then hands
// the pass back on seed's stream.
func crMageHandBack(seed uint64) func(*testing.T, *World, *castRequestRecorder) []Command {
	return func(t *testing.T, w *World, r *castRequestRecorder) []Command {
		crLoadStrike(t, w, r, 2)
		w.entities[0].KnownSpells = 1 << 1
		w.entities[0].Mind = 0
		r.call(w, "draw during strike", w.orderAttack(0, 2))
		w.entities[0].Mind = 99
		w.rng = rng{state: seed}
		r.call(w, "hand-back", w.orderAttack(0, 2))
		return nil
	}
}

var castRequestCases = []castRequestCase{
	{name: "manual-unit", caster: 1, ticks: 60, world: crManualWorld, start: crCast},
	{name: "manual-cell", caster: 1, ticks: 40, world: crManualWorld,
		start: func(*testing.T, *World, *castRequestRecorder) []Command {
			return []Command{CastAt(1, 26, CellPoint{X: 6, Y: 3})}
		}},
	{name: "manual-during-strike", caster: 1, ticks: 80, world: crManualWorld,
		start: func(_ *testing.T, w *World, r *castRequestRecorder) []Command {
			r.call(w, "attach", w.attachAttack(0, 2, AttackTargetUnit, false))
			w.entities[0].AttackPhase, w.entities[0].AttackCountdown = AttackCharging, 1
			r.note(w, "loaded")
			return []Command{Cast(1, 2, 1)}
		}},
	{name: "manual-replaces-unpaid-wind-up", caster: 1, ticks: 60, world: crManualWorld, start: crCast,
		then: crAt(2, CastAt(1, 26, CellPoint{X: 6, Y: 3}))},
	{name: "manual-behind-paid-wind-up", caster: 1, ticks: 80, world: replacementManaWorld, start: crCast,
		then: crAt(1, Cast(1, 2, 3))},
	{name: "manual-repeat", caster: 1, ticks: 60, world: replacementManaWorld, start: crCast,
		then: crAt(1, Cast(1, 2, 1))},
	{name: "manual-attack-during-paid-wind-up", caster: 1, ticks: 60, world: replacementManaWorld, start: crCast,
		then: crAt(1, Attack(1, 2))},
	{name: "manual-approach", caster: 1, ticks: 160, world: func(t *testing.T) *World { return approachWorld(t) },
		start: crCast},
	{name: "scroll", caster: 1, ticks: 80, world: func(t *testing.T) *World { return scrollFixtureWorld(t, 12, 2) },
		start: func(*testing.T, *World, *castRequestRecorder) []Command {
			return []Command{UseScroll(1, 0, 2)}
		}},
	{name: "scroll-during-strike", caster: 1, ticks: 80, world: func(t *testing.T) *World { return scrollFixtureWorld(t, 2, 2) },
		start: func(_ *testing.T, w *World, r *castRequestRecorder) []Command {
			r.call(w, "attach", w.attachAttack(0, 1, AttackTargetUnit, false))
			w.entities[0].AttackPhase, w.entities[0].AttackCountdown = AttackCharging, 1
			r.note(w, "loaded")
			return []Command{UseScroll(1, 0, 2)}
		}},
	{name: "scroll-during-cast", caster: 1, ticks: 60, world: crManualScrollWorld, start: crCast,
		then: crAt(1, UseScroll(1, 0, 2))},
	{name: "script-unit", caster: scTarget, ticks: 20, world: func(t *testing.T) *World { return scWorld(t, nil) },
		start: func(_ *testing.T, w *World, _ *castRequestRecorder) []Command {
			w.casts = []scriptCast{{FromX: 7, FromY: 8, Spell: uint8(scDamageSpell), Power: 20, Target: scTarget, AtUnit: true}}
			return nil
		}},
	{name: "script-cell", caster: scTarget, ticks: 20, world: func(t *testing.T) *World { return scWorld(t, nil) },
		start: func(_ *testing.T, w *World, _ *castRequestRecorder) []Command {
			w.casts = []scriptCast{{FromX: 7, FromY: 8, ToX: 40, ToY: 41, Spell: uint8(scAreaSpell), Power: 20}}
			return nil
		}},
	{name: "creature-held-by-strike", caster: 1, ticks: 120,
		world: func(t *testing.T) *World {
			w := creatureWorld(t, []SpellRule{hlArrow()}, [CreatureSpellSlots]CreatureSpell{}, 1<<1, creatureArrowBook())
			w.entities[indexOfEntity(w.entities, 2)].X = 6
			return w
		},
		start: func(t *testing.T, w *World, r *castRequestRecorder) []Command {
			crLoadStrike(t, w, r, 2)
			ci := indexOfEntity(w.entities, 1)
			w.entities[ci].CreatureSpells = creatureCertain
			r.call(w, "draw during strike", w.orderAttack(ci, 2))
			return nil
		}},
	{name: "creature-replaces-retained", caster: 1, ticks: 120, world: crCreatureCertainWorld,
		start: func(_ *testing.T, w *World, r *castRequestRecorder) []Command {
			for range scriptPassPhase + 1 {
				r.step(w, nil)
			}
			r.call(w, "draw while pending", w.orderAttack(w.indexOf(1), 2))
			return nil
		}},
	{name: "creature-miss-in-recovery", caster: 1, ticks: 80, world: crCreatureCertainWorld,
		start: func(t *testing.T, w *World, r *castRequestRecorder) []Command {
			for range scriptPassPhase + 1 {
				r.step(w, nil)
			}
			for range 400 {
				r.step(w, nil)
				if k, ok := w.bookCastIndex(1); ok && w.bookCasts[k].Phase == bookRelaxing {
					break
				}
			}
			w.entities[w.indexOf(1)].CreatureSpells = [CreatureSpellSlots]CreatureSpell{{ID: 1}}
			r.call(w, "miss in recovery", w.orderAttack(w.indexOf(1), 2))
			return nil
		}},
	{name: "creature-approach", caster: 1, ticks: 120,
		world: func(t *testing.T) *World {
			book, known := creatureShortBook()
			return creatureWorld(t, []SpellRule{hlArrow()}, creatureCertain, known, book)
		}},
	{name: "engage-near", caster: 1, ticks: 100,
		world: func(t *testing.T) *World { return engageMageWorld(t, 1<<1, 100, hlArrow()) },
		start: func(_ *testing.T, w *World, r *castRequestRecorder) []Command {
			r.call(w, "engage", w.orderAttack(0, 2))
			return nil
		}},
	{name: "engage-far", caster: 1, ticks: 160,
		world: func(t *testing.T) *World { return engageMageWorld(t, 1<<1, 100, hlArrow()) },
		start: func(_ *testing.T, w *World, r *castRequestRecorder) []Command {
			r.call(w, "engage", w.orderAttack(0, 3))
			return nil
		}},
	{name: "engage-crossing", caster: 1, ticks: 80,
		world: func(t *testing.T) *World { return engageMageWorld(t, 1<<1, 100, hlArrow()) },
		start: func(_ *testing.T, w *World, r *castRequestRecorder) []Command {
			w.entities[0].Transit, w.entities[0].TransitTotal = 4, 8
			r.call(w, "engage", w.orderAttack(0, 2))
			return nil
		}},
	{name: "engage-held-by-strike", caster: 1, ticks: 120,
		world: func(t *testing.T) *World { return engageMageWorld(t, 0, 100, hlArrow()) },
		start: func(t *testing.T, w *World, r *castRequestRecorder) []Command {
			w.entities[0].Mind = 0
			crLoadStrike(t, w, r, 2)
			w.entities[0].KnownSpells = 1 << 1
			r.call(w, "draw during strike", w.orderAttack(0, 2))
			return nil
		}},
	{name: "engage-recovery-seed-1", caster: 1, ticks: 60, world: crMageRecoveryWorld, start: crMageRecovery(1)},
	{name: "engage-recovery-seed-2", caster: 1, ticks: 60, world: crMageRecoveryWorld, start: crMageRecovery(2)},
	{name: "engage-recovery-seed-3", caster: 1, ticks: 60, world: crMageRecoveryWorld, start: crMageRecovery(3)},
	{name: "engage-recovery-seed-5", caster: 1, ticks: 60, world: crMageRecoveryWorld, start: crMageRecovery(5)},
	{name: "engage-hand-back-seed-1", caster: 1, ticks: 60,
		world: func(t *testing.T) *World { return engageMageWorld(t, 0, 100, hlArrow()) }, start: crMageHandBack(1)},
	{name: "engage-hand-back-seed-2", caster: 1, ticks: 60,
		world: func(t *testing.T) *World { return engageMageWorld(t, 0, 100, hlArrow()) }, start: crMageHandBack(2)},
	{name: "engage-hand-back-seed-4", caster: 1, ticks: 60,
		world: func(t *testing.T) *World { return engageMageWorld(t, 0, 100, hlArrow()) }, start: crMageHandBack(4)},
	{name: "autocast", caster: 1, ticks: 60,
		world: func(t *testing.T) *World {
			caster := acCaster(1, 2, 2, 50, 1<<1, 1)
			caster.AttackCharge, caster.AttackRelax, caster.Owner = 12, 4, 1
			enemy := spEnt(2, 4, 4)
			enemy.Owner = 2
			return hlWorld(t, 23, acEnemies(t), []SpellRule{hlArrow()}, caster, enemy)
		}},
	{name: "escort-heal-waits", caster: 2, ticks: 120,
		world: func(t *testing.T) *World { return residueDefender(t, 49, 20, 0) }},
	{name: "guard-waits", caster: 1, ticks: 120, world: crCreatureCertainWorld,
		start: func(_ *testing.T, w *World, r *castRequestRecorder) []Command {
			for range scriptPassPhase + 1 {
				r.step(w, nil)
			}
			w.commandGroup([]int{w.indexOf(1)}, orderGuard, cell{})
			w.engageCreatureFromNextToIt()
			r.note(w, "guard")
			return nil
		}},
	{name: "retreat-waits", caster: 1, ticks: 80, world: crManualWorld, start: crCast,
		then: crAt(2, GroupRetreat(1, 2, 91))},
	{name: "pickup-waits", caster: 1, ticks: 60, world: crManualWorld, start: crCast,
		then: func(t *testing.T, w *World, tick int) []Command {
			if tick == 2 && !w.CompleteSackPickup(1) {
				t.Fatal("pickup completion refused")
			}
			return nil
		}},
	{name: "defend-waits", caster: 1, ticks: 60, world: crManualWorld, start: crCast,
		then: crAt(2, GroupDefend(1, 2, 1))},
	{name: "attack-order-waits", caster: 1, ticks: 60, world: crManualWorld, start: crCast,
		then: crAt(2, Attack(1, 2))},
}

func castRequestRecord(t *testing.T, w *World, c castRequestCase, second bool) []byte {
	t.Helper()
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	r := &castRequestRecorder{caster: c.caster}
	r.note(w, "start")
	var cmds []Command
	if c.start != nil {
		cmds = c.start(t, w, r)
	}
	for tick := range c.ticks {
		switch {
		case tick == 0:
			Step(w, cmds)
		case c.then != nil:
			Step(w, c.then(t, w, tick))
		default:
			Step(w, nil)
		}
		r.note(w, fmt.Sprintf("tick %d", tick))
	}
	return r.b.Bytes()
}

// Every cast issuer and every action that waits for a cast, for each game's
// profile: each tick's World hash, random state and the caster's cast and
// order state. castRequestCases names each recorded issue: the manual,
// scroll, script, creature, engage and automatic issuers in each of their
// busy cases, and the escort, guard, retreat, pickup, defend and attack
// orders that wait for a cast.
func TestCastRequestRecord(t *testing.T) {
	var got strings.Builder
	for _, c := range castRequestCases {
		for _, second := range []bool{false, true} {
			game := "first"
			if second {
				game = "second"
			}
			t.Run(c.name+"/"+game, func(t *testing.T) {
				w := c.world(t)
				record := castRequestRecord(t, w, c, second)
				if *dumpCastRequest != "" {
					if err := os.WriteFile(filepath.Join(*dumpCastRequest, c.name+"-"+game+".txt"), record, 0o644); err != nil {
						t.Fatal(err)
					}
				}
				fmt.Fprintf(&got, "%s %s ticks %d record %x final %016x\n", c.name, game, c.ticks, sha256.Sum256(record), w.Hash())
			})
		}
	}
	path := filepath.Join("testdata", "castrequest.txt")
	if *updateCastRequest {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.String() == string(want) {
		return
	}
	gl, wl := strings.Split(got.String(), "\n"), strings.Split(string(want), "\n")
	for i := range max(len(gl), len(wl)) {
		var g, w string
		if i < len(gl) {
			g = gl[i]
		}
		if i < len(wl) {
			w = wl[i]
		}
		if g != w {
			t.Errorf("record line %d is\n  %s\nwant\n  %s", i+1, g, w)
		}
	}
}
