# Analysis — the player's own units can fight

## The measurement, re-taken in this seat

`0075` landed an order that resolves a blow, and the party could not use it. Measured here, from a
throwaway probe over `StartMission` with a one-member party on a synthetic map:

```
PARTY: class=100 hp=100 max=100 dmgBase=0 dmgSpread=0 toHit=0 defence=0
       absorption=0 alwaysHits=false charge=8 relax=4 speed=10
```

Six of the eight combat numbers are zero, against 100 hp. The handed observation is confirmed.

## Was the zero a bug or a consequence? BOTH — and the trap is that they look identical

Two true things had to be held at once.

**`start.go`'s `PartyMember` was right to refuse.** It carried `Class` and nothing else, and its
comment said why: a party fighting at numbers this tree invented would be a fabrication. `0067` wired
a *placed* unit's eight numbers from its class row; the party was never in that path, so it took
`data.UnitDefaults()` — the **Units constructor's** defaults, whose six combat cells are zero because
no row wrote them.

**`HERO-BARE-037` says a bare hero really can do zero.** `base = spread = ftol(1.1^Body / 20)`, and
`1.1^Body >= 20` first at Body **32** — so a bare-handed hero at any chargen-legal Body from 15 to 31
rolls `0-0`. Our party stands at the Units constructor's Body 30. **A correct model of a bare hero at
Body 30 produces exactly the numbers we were producing by accident.**

So the zero is a *consequence* arithmetically and a *defect* causally: the two agree on the value and
on nothing else. What makes it a defect is `HERO-START-039` — a new campaign hero is not bare. He is
handed exactly one weapon, chosen from ten string literals by a jump table on his one trained skill,
and character generation gives him four stats at 25 and one skill at 10 (`HERO-BUY-003`,
`HERO-SKILL-009`). With those he does **7-10**, not 0-0.

Stated as the story's shape: **the zero is a consequence of a party that carries nothing, and the fix
is to give it something to carry.** Nothing in the resolver, the roll or the cadence was wrong.

## The falsifier, run before a line of spec was written

`ITEM-SCALE-017` and `ITEM-DMGCOL-018` were published High off a ladder that stood one slot high, so
every weapon read at five times its damage. The rescued value is a *shape* factor of 0.2, not 1.0
(`ITEM-LADDER-019`, `ITEM-DMGFACT-020`), and the wrong one survived review because the High grade
rested on an ordering that is invariant under a uniform shift. A plausible number is therefore not
evidence here; a number that would be **different** under the rival is.

The owner's measured band edges are that number. Against Strength 15 / 32 / 39 / 43 he read 7-10,
8-12, 9-14, 10-16 with an `Iron Short Sword` listed as 5-8 — band edges, not samples. A prototype of
the fold, run in this seat before any code was written, over `base = ftol(1.1^Body/20) + skill/5 +
weaponBase`, `spread = ftol(1.1^Body/20) + weaponSpread`, sheet `[base, base+spread]`
(`HERO-DERIVE-034`, `HERO-SHEET-038`), with the sword at `(5, 3)` and one skill at 10:

```
Body  bare  base  spread  sheet          band edges over Body 15..50
  15     0     7       3  7-10           B15->0 B32->1 B39->2 B43->3 B46->4 B49->5
  32     1     8       4  8-12
  39     2     9       5  9-14
  43     3    10       6  10-16
```

Four of four, exactly, and the step points agree with "the band is constant between them". Under the
retracted ladder the sword is `(23, 17)` and the same four rows read 25-42, 26-44, 27-46, 28-48 — so
the check discriminates rather than merely fits. **The 0.2 is not typed into this tree**: the
resolver re-derives `(5, 3)` from the install's own `Data.bin` through the corrected ladder, which is
what makes the reproduction independent evidence and not a restatement of the claim.

The prototype also measured the truncation margins, because these values reach hashed state through a
float: over the whole legal stat range the damage term comes no closer than **8.98e-3** to an integer
boundary (at Body 46) and the to-hit term no closer than **2.05e-4** (Body 47, Reaction 59). Both are
~1e12 ULPs clear, so the truncated `int32` is not implementation-sensitive.

## What we did not know, and where it was answered

- *Does a hero get a weapon at all, and which?* `HERO-START-039`. Always one; `CMP ECX,0xa` picks
  which of two five-literal sets. Slot 1 (Blade) is `Iron Short Sword`.
- *Which arm does a shipped campaign take?* That row grades it **Medium** — it rests on the absence
  of an immediate-2 writer for `[L03937]`, not on reading the menu chain. The owner's own sheet
  lists the sword at 5-8, which is the **ordinary** arm's blade weapon; that is corroboration
  arriving from the game, and this story takes the ordinary arm.
- *Which stats reach damage?* `HERO-STATDMG-036`: Body alone reaches damage, Body and Reaction reach
  to-hit, by an exhaustive read census inside the sole writer. No stat enters multiplicatively.
- *Can equipment multiply anything?* `HERO-FOLD-035`: the fold is eight `ADD`s and one assignment,
  no `IMUL`, no `FMUL`, no shift.
- *Does our `AlwaysHits` need the second effect?* `HERO-AUTOHIT-031` says the bit also skips
  absorption. Checked: `pkg/sim/combat.go` uses it for the hit test only and applies absorption
  unconditionally two lines later. A hero never carries the bit — its writer is the Units streamer's
  `attackKind == 3` arm — so no party number is wrong, but eight monster classes are over-protected
  against. Named as open below; it is `pkg/sim`'s resolver and not this story's party.

## What is still open, and who is closing it

- The second effect of the auto-hit bit (`HERO-AUTOHIT-031`), above. A `pkg/sim` story.
- A hero's **health** (`HERO-HP-005`), **speed** (`HERO-SPEED-008`) and **reach**: all derived from
  the same stats, none among the eight numbers this story fills. The party keeps `SpawnHP` and the
  constructor's speed. Disclosed in `spec.md`, not modelled.
- The stat point-buy (`HERO-COST-002`, `HERO-BUY-003`, `HERO-BUDGET-004`): there is no screen to
  spend points on, so no cost table is built. The chargen *start* is.
- The mage arm's `Wood Staff {castSpell=Fire_Arrow:10}`: needs a class axis and a spell-carrying
  weapon, and this tree has neither.

## What was looked at and not used

`HERO-CADENCE-023` (the three-phase cycle) and `HERO-CLAMP-030`/`HERO-HEALTH-032` (the resolver's
clamps) — all already shipped by `0069`, `0064` and `0075`. `HERO-MOD-016`'s three parallel modifier
blocks: the fold's *inputs*, not a structure this tree needs, since our derive is a pure function of
a hero and a weapon and holds no live actor.
