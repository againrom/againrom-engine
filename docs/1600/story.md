# One cast request boundary for every issuer

## Intent

Make the busy rule a cast meets one kind with one builder. Every cast
issuer (manual, scroll, script, creature, engage, the automatic casts and
a retained order asked again) builds a `castRequest` and meets its
issuer's row of `castBusyRows`. Every action that is not a cast and waits
for one asks one busy query, `castHolds`. No player change: in both games
every cast, scroll, strike, escort, guard, retreat, pickup, defend and
attack order runs as before, and every tick's hashed state, random draw
and SAV byte stays the same.

## Authority

Owner direction for audit 20: one builder per kind, the first game as
profile data (`gameProfile`), architecture work counts 0 toward the
release, no player change. The kind, census and guard are
`review/audit-20/PLAN.md` rank R20 and the story brief. The rows keep the
claims their sites cited: AI-356 and AI-354 (a paid wind-up keeps the
caster, a manual cast queues behind it and behind a retained strike),
AI-376 (a draw replaces a retained order), AI-554, MAGIC-239 and MAGIC-221
(a loaded strike holds a computer caster's drawn cast), MAGIC-340 and
MAGIC-348 (an order waits for its mana and walks into reach), DIV-1015 and
DIV-028. No claim, divergence row or knowledge pin changes.

## As-built behaviour

`pkg/sim/castrequest.go` holds the boundary. A `castRequest` is
`{issuer, caster, spell, aim, paid}`; `castAim` is a unit target or a
cell, and the victim an engage draw aims from. Each issuer's row:

| issuer | gate | mana waits | a cast that owns the caster | strike | other |
|---|---|---|---|---|---|
| `castManual` | none | no | queues behind a paid wind-up | queues (retained old strike) | a repeat runs on |
| `castScroll` | none | no | refuses | queues | |
| `castScript` | none | no | passes | passes | no caster |
| `castCreature` | fresh | yes | replaces the retained order (slot aim) | holds | |
| `castEngage` | fresh | yes | replaces the retained order (victim aim) | holds | a recovery refuses |
| `castAuto` | fresh | no | passes | passes | |
| `castRetained` | retained | the record's `Paid` | passes | holds | |

The gate is the admission's action guard: `gateFresh` is
`actorActionBusy`, `gateRetained` is `retainedCastBusy` (the caller's own
record not counted), `gateNone` asks none because the issuer ends the old
action itself. The two admission bodies in `pkg/sim/spell.go`
(`bookSpellAdmissionRange`, `bookSpellCellAdmissionRange`) take the gate
in place of the retained and checkAction flags and are called only
through `castAdmission`. A wind-up's start asks `castBeginRefusal`, the
fresh gate, for every issuer.

The boundary's functions: `castOwned` (the owned column; the replace rows
rewrite the retained order through `replaceRetainedCast`), `castRepeats`
(the manual repeat), `castStrikeWaits` (the strike column),
`castRecoveryRefuses`, `castRefusal` and `castActionBusy`. The refusal
constants and their predicates (`isRangeRefusal`, `isSightRefusal`,
`isManaRefusal`) live there. `actorCastBusy`, `actorActionBusy`,
`retainedCastBusy`, `strikeHoldsCast` and `replaceRetainedCast` moved
there unchanged.

The busy query `castHolds(i, why)` has five purposes:
`waitCastAction` (`actorCastBusy`; escort, guard arm, player defend,
retreat and its executor progress, pickup completion, saved groups, the
script's attack, the step's attack and escort gates, target orders, the
AI's attack order), `waitRetreatProgress`, `waitStrikeAdvance`,
`waitIdleHeal` and `waitStructureUse` (a paid wind-up). Each caller keeps
its own `OffMap`, `stoneCursed`, `Transit` and attack-phase conditions.

No row differs between the games; the games' differences around a cast
stay the profile's `releaseRevalidates` and `creatureAim`.
`creatureCastRefusal`, `creatureReplaceCastOrder`, `World.paidWindUp` and
the `bookSpellRefusal`, `bookSpellAdmission`, `bookSpellCellRefusal` and
`bookSpellCellAdmission` wrappers are removed.

`internal/archtest/castrequest.go` (`CastRequestSites`) refuses, in
production `pkg/sim` files other than the boundary, a call of
`actorCastBusy`, `actorActionBusy`, `retainedCastBusy`,
`orderState.castBusy`, `strikeHoldsCast`, `replaceRetainedCast`,
`retainsOldStrike` or either admission body; and in every production
`pkg` file other than the boundary, an `==`, `!=` or `case` naming one of
the boundary's refusal constants or its string. The allow list is empty.
The order-phase scan's record-reader allowance moved from
`creaturecast.go` to `castrequest.go` with `replaceRetainedCast`. No
import is added, so no `dag` row.

## Proof

- `TestCastRequestRecord` (`pkg/sim/castrequestrecord_test.go`),
  committed alone before the boundary: 35 cases, each in a first-game and
  a second-game World, 70 rows. Each records every tick's World hash,
  random state and the caster's cast, scroll, wait, pending order, attack,
  walk, retreat and mana, digested into
  `pkg/sim/testdata/castrequest.txt`. The cases cover every issuer in its
  busy cases (manual unit, cell, strike, paid and unpaid wind-up, repeat,
  approach; scroll free, in a strike, in a cast; script unit and cell;
  creature hold, replace, recovery, approach; engage near, far, crossing,
  hold, seven seeded recovery and hand-back runs; autocast) and every
  waiter (escort heal, guard, retreat, pickup, defend, attack order).
  After the change the digests are equal.
- A planted change, the engage row's strike `strikePasses`, fails 12 of
  the 70 rows.
- The guard's self-tests plant each shape and name it at its line; a
  planted `actorCastBusy` call in `guardarm.go` and a planted
  `"insufficient mana"` compare in `engage.go` fail the live test.
- Focused tests: `./pkg/sim -run
  'Cast|Scroll|Strike|Retained|Engage|Escort|Guard|Retreat|Pickup|Defend|TargetOrder|Script|Busy'
  -skip '^TestRelease'`, the archtest guard, `Dag` and order-phase tests,
  and storyguard's `TestLiveTreeClean`, assertions unchanged. Two tests
  changed only the call expression of a removed wrapper.

## Open debt

- The second game's creature fixture never loads its strike in 400
  ticks, so the record's second-game creature hold row records no hold;
  the engage hold row covers the second game's hold.
- `mageEngageCast`'s mana branch is not reached: the engage row lets the
  order wait for mana, so admission never returns the mana refusal there.
  It is kept, through `isManaRefusal`.
- `internal/archtest/spellarms.go` still lists `bookSpellRefusal` among
  the book lookups; no production call remains, and its self-test reads
  synthetic source.
- The release tests that cast, use a scroll, or save and load a cast or
  scroll record must run on both games at the next release, first
  `dragonwitness_release_test.go`.
