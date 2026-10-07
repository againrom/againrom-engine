# Verification — 0119-chargen

## What was built

`-chargen` opens a generation screen before a mission and the character it produces is the one the
mission is played with and drawn as. `-skill`, `-mission` and every path without the flag behave as
they did. And the hero's statistics now reach his health: the two pools the derived-stat graph has
been producing since 0113 are wired onto the entity, off the shipped row a generated character
starts from.

The command the owner types:

```
againrom -assets <dir> -mission 10 -chargen
```

## The owner's defect, witnessed

He reported that his hero always has 100 health whatever his statistics are. Against a lawful
install, `-check -mission 10`:

| | before this story | after |
|---|---|---|
| default hero, Body 43 | `health 100/100, mana 0/0` | `health 145/145, mana 0/0` |

That line is a **lookup, not a recompute** — `MissionLine` reads the pair back off the simulation
entity `mapload.StartMission` wrote, so it proves the numbers reached the world rather than that the
report agrees with itself.

Health moves with Body, measured through the whole path from a generation result to an entity:

| Body | 15 | 25 | 43 |
|---|---|---|---|
| entity `MaxHP` | 36 | 60 | 145 |

**Witnessed by reverting, not by reading the assertion.** Putting `HP: SpawnHP, MaxHP: SpawnHP` back
at `pkg/mapload/start.go` and running the health test:

```
pools_test.go:45: Body 25's health pair is 100/100, want the recompute's own 27 on both
pools_test.go:49: Body 45's health pair is 100/100, want the recompute's own 77 on both
```

Restored, green. A second revert-witness is in T4's commit body.

**And a mage now has a pool at all.** A party member used to be minted with no mana pair, which was
a hard blocker on every future spell — the first comparison in a cast would refuse. A generated mage
at Spirit 25 reaches the map with `mana 60/60`.

## The generated character, end to end

Driven through `FrontEnd.ChargenSetup` → `ui.Chargen` → `FrontEnd.ChargenParty` →
`game.StartMission` against a lawful install, reading the resulting entity:

| choice | base row | figure | body drawn | entity health | entity mana |
|---|---|---|---|---|---|
| male fighter, Blade | `PC_Danath` | `mfighter/5` | swordsman (3) | 60/60 | 0/0 |
| female fighter, Blade | `PC_Naira` | `ffighter/1` | swordsman (3) | 60/60 | 0/0 |
| male mage, Fire | `PC_Fergard` | `mmage/3` | swordsman (3) | 29/29 | 60/60 |
| female mage, Astral | `PC_Reniesta` | `fmage/1` | archer (14) | 29/29 | 60/60 |

All four rows resolve, which is `SESS-HERO-014`'s own 4/4 reproduced from this tree's decode.

**What reaches the map, and what does not.** The **skill** changes the map sprite — a different
trained slot is a different starting weapon is a different body name through the name chain, so
Blade draws a swordsman and Shooting an archer, with different damage bands. The **sex** does not:
the drawing reads the equipment, and the two routines that turn a drawable into pixels were read end
to end by research without either testing the sex bit. Sex reaches the **inventory window's figure**
instead, through the four shipped figure directories. That is the opposite of what this lane was
briefed and it is the claim's own finding.

## Every premise this lane was handed that turned out wrong

1. **"Only the sex bit reaches the drawing."** Inverted. `HERO-APPEAR-045` says the sex bit is put on
   the drawable and *neither* pixel routine reads it. The class-derived equipment is what reaches the
   drawing; sex reaches the inventory figure.
2. **"Recompute's output stops at the info panel and never reaches a combatant."** Half wrong, and
   the seat corrected itself mid-story. The whole **combat** block was already wired. Only the two
   pools were not, and `recompute.go` said so in its own words.
3. **"`pkg/game/inventory.go`'s two figure constants are ONE EDIT WIDE."** Not true. Retiring them
   took a new helper, three call sites, an import and the header — five places. The previous author's
   claim is recorded here as measured, not as believed.
4. **My own DD-12, "a positive maximum is the gate."** Wrong, caught by T4 against a landed pin. The
   health arm's column gate covers only its first term, so a *trained* character with the zero
   profile derives a small positive maximum — 2, for this tree's hero — and gating on positivity
   would have shipped him at two health. The **column** is the gate.
5. **My own DD-7, "a shipped row states its archetype in its type id."** Wrong, caught against the
   install. The four rows carry 3, 14, 24 and 24 there — the *drawn class ids* of the appearance name
   chain — while the `0x21 + 2*class + gender` id is built by a constructor at run time and stored
   nowhere. Every archetype was silently resolving to the same row: sex and class changed the figure
   directory and nothing else. The archetype is the **slot**, and the mana column corroborates the
   published name order (0, 0, 70, 70).

## The logarithm margin, which was owed before the pools could be wired

`recompute.go` stated that `logBase11` carried no measured margin against the original's own
logarithm, unlike `pow11`'s 2e-4, and that whoever wired this seam owed the measurement first. It is
measured, over 8.3e7 integer experience values in one pass, and the result is **not uniformly
clean** — which is why the debt existed.

- The step-two term is `ftol(statistic*mult + log*mult)` and `statistic*mult` is an **integer**, so
  it cannot move the fractional part. The sweep is therefore one-dimensional, over experience alone.
- Over the whole reachable experience range the smallest nonzero distance to a truncation boundary
  is **1.78e-15**, and the values carrying it are exactly three: `e ∈ {500, 1050, 1655}`, the only
  ones for which `e/5000 + 1` is exactly a power of 1.1, because `(1.1^k - 1)*5000` is an integer
  only for k in 1..3. At k=2 and k=3 our value sits a few units in the last place **below** an
  integer the true logarithm hits exactly, so `ftol` answers 1 and 2 where exact arithmetic answers
  2 and 3. A differently rounded `log` would answer one more there. **This is disclosed, not
  fixed**, and the test names the three inputs rather than reporting one worst case.
- Over the experience a character this tree can construct — generation writes one slot at level 10
  or 20 — the smallest nonzero margin is **8.86e-3**, some thirteen orders of magnitude clear of a
  logarithm's last place. No character this tree can build reaches the three risky values.
- Step three, `ftol(pool * (1.1^stat/100 + 1))`, over the reachable pool range: **2.37e-5**.

`recompute.go`'s two doc blocks were rewritten from the debt into the finding.

## Acceptance

| | Witnessed by |
|---|---|
| AC-1 | `TestAC1TheNoFlagPartyMatchesTodaysHeroLiterally` — today's values written out, not two calls compared |
| AC-2, AC-4 | `pkg/ui/chargen_test.go`: a step costs the cost difference and the refund returns Remaining exactly; a refused step moves nothing, not the value, not Remaining, not the focus |
| AC-3 | an exhaustive depth-first walk of the reachable state space through the exported surface: 2489 legal states over 19912 `Adjust` calls, every one legal |
| AC-5 | changing the class re-labels the skill row and keeps its index |
| AC-6 | `Result()` answers false on an illegal spread and the screen stays |
| AC-7 | synthetic `Humans` collection: each archetype reaches its own slot, an unresolvable slot falls back, none resolving answers the zero profile |
| AC-8, AC-10 | `pkg/mapload/pools_test.go`, plus the install table above |
| AC-9 | a **trained** zero-profile hero derives a positive maximum from the recompute and is still minted at the spawn constant — the test that stops the gate being simplified back to positivity |
| AC-11 | `internal/archtest`'s import graph, unchanged: `pkg/ui/chargen.go` imports `fmt` alone |
| AC-12 | `-chargen` without `-mission` exits 2 with `againrom: -chargen requires -mission`, against an empty directory, so no archive is opened |
| AC-13 | the margin measurement above |
| P-1 | one `ChargenParty`, one `HumanDef.Profile()`; `MissionParty` goes through the same assembly |
| P-2 | the model imports `fmt` and nothing else; no clock, no engine, no randomness |
| P-3 | every label, bound, cost and option is a field of `ChargenSetup` |
| P-4 | the drawing tier receives strings and integers; the archive stays in `pkg/game` |
| P-5 | the authored places: the screen's labels, the archetype order of the four names, the class-flag reading, the default axes |
| P-6 | every test builds its collection in test code; `check-no-game-assets.sh` clean |

## What was not built, and what is authored

- **The screen is plain and authored.** A header, a focus marker, one line per row, a remaining
  counter and a confirm line, in the engine's debug font — the same font and the same call the map
  picker already uses. No original layout, no art, no portrait carousel. English labels, because
  that font renders no Cyrillic byte.
- **The windowed screen is witnessed by unit tests driving `App.step`, and by a 12-second run of the
  real binary against a lawful install that exited only when it was killed.** It was not visually
  inspected. A rendered frame is the one piece of evidence this lane does not hold.
- **A mage is handed a sword.** The mage weapon arm — `Wood Staff {castSpell=Fire_Arrow:10}` — needs
  a weapon carrying a spell, which this tree has no representation for.
- **Nothing casts.** The pool now exists; the spell does not.
- **Which archetype the pool graph's class flag names is AUTHORED.** This tree sets it for a row with
  no mana column, which makes a fighter the tougher of the two. The competing reading is live and
  named in the code: the flag's runtime writer sets it for an actor holding a spellbook and a pool,
  which would swap them. Research resolves neither. One function to invert.
- **Which gender addend is female never had to be decided**, because the archetype became the slot.
- **The face changed.** Without the flag the character is today's on every axis but two: his health
  pair, which is the defect, and his figure **face**, which now comes off the base row — 5 rather
  than the constant 1.
- **A generated character reaches only the `-mission N` given at startup.** A mission later picked
  from the map list runs the default hero, and that is written down at the door rather than left to
  be discovered.
- **The byte form did not move.** No field was added to a simulation entity, `pkg/sim` is untouched,
  and the version allocated to this story is returned unused.

## Gate

On the clean committed tree: `go build ./...`, `go vet ./...`, `gofmt -l` over our own files, and
`go test -trimpath -count=1 ./...` green in every package; `check-no-game-assets.sh`,
`check-doc-budget.sh` and `check-sdd-audit.sh` clean of FAILs. The deletion set against the branch
point is empty.
