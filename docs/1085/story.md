# 1085 — Direct physical structure attacks

## Intent and as built

A selected player unit can arm Attack, click a visible structure sprite, approach
an accessible firing cell, and keep striking through the existing physical action.
Mere hover remains read-only; neither selection nor inventory changes. Unit and
structure handles have an explicit kind, so unit zero never aliases structure zero.
The live inspection card reflects HP changes and the map chooses ruin frames when
signed health becomes non-positive. Move, attacker death, an invalid target, or
ruin ends the retained order.

Research is pinned to accepted `7f3c5270c78925f46914f7f93e52cf8e50bf2e3c`.
`UNIT-STRUCTORDER-062` through `UNIT-STRUCTSTOP-066` and their promoted dependencies
were read through the claim tool; no raw experiment or implementation-derived ROM1
authority was used. Building's combat token is one, not its blocking rectangle.
Physical damage reads only the third base/spread pair: zero spread suppresses the
damage draw; otherwise subtract five from base plus the inclusive random draw and
floor the result at zero. Zero maximum health is immune. Damage subtracts a word,
not the effect consumer's zero-clamped health. Shared charge, distance flight,
relaxation and recovery boundaries remain the action clock.

Native form 67 reuses attack byte +48: absent=0, unit=1, structure=2. Record width
stays 294. Form 66 upgrades losslessly; malformed old presence bytes cannot become
new structure handles. Existing independent unit-only byte fixtures and previous
hashes are retained under a version-byte peel, including the save envelope.

## Proof at the focused checkpoint

- `go test -trimpath -count=1 ./pkg/sim ./pkg/mapload ./pkg/ui ./pkg/game
  ./internal/gatedtests`: all five packages PASS without installs.
- Simulation tables independently pin damage, RNG consumption, range, token size,
  word underflow, charge/flight/recovery timing, repeated orders, blocked anchors,
  unreachable cancellation, unit/structure zero separation, unit death/removal,
  command replacement, corrupt save tags and mid-action hash continuation.
- UI tests drive the production Attack key and pointer press/release dispatch.
  Unarmed, unseen, explored-only and ruined targets cannot issue an attack;
  hover/press alone do not issue one. Selection and inventory remain the unit's.
- `TestReleaseStructurePhysicalAttack` PASS separately with both preserved EN and
  RU roots. The installed-data arena uses a generated party actor and real map10
  terrain/structures, with actor placement and population explicitly controlled:

  | Equipped weapon | Third pair | Real target | Health result |
  |---|---|---|---|
  | Native Short Bow `0801120` | 0 + U[0,0] | structure 4 at (33,49), 3x3 | 1000 → 1000; zero damaging hits |
  | Installed Flame Thrower `0001125` | 2 + U[0,17] | structure 7 at (53,10), 3x3 | 1000 → -6; 129 observed HP changes |

  The positive fixture equips the actual item through the existing loadout path;
  an independent table calculation checks its material/shape scaling. It never
  assigns a damaging component directly. Both cases traverse actual pointer input
  and SaveStore. Mid-approach and mid-charge saves each continue for 30 ticks with
  identical hashes. The installed card's composed pixels change only in the
  damaging case; the positive target changes from no ruin entries to all ruin
  entries and holds no attack afterward. This is an installed-data headless drive,
  not a foreground GUI witness or proof the campaign can acquire that loadout.
- The existing `0155-synthetic-melee.json` headless unit-attack scenario passes
  with both root settings: target dies at tick21, survivor HP37, no unsupported
  nodes. It is synthetic and does not establish an additional installed witness.
- A locally built `missionrun`, EN, `-trace -ticks 1`: mission10 UNSUPPORTED=0,
  mission20 UNSUPPORTED=0. Script populations remain 16/27/12 and 14/15/11, matching
  `pipeline/milestone-baseline.txt`; this story moves the playable structure route,
  not the script-gap census.
- Gofmt, whitespace and `check-no-game-assets.sh` checks pass. The gated-test
  population includes the new
  installed test. Reconciled `check-div-claims.sh --ids` selects 270/270 live rows and reports
  67 retraction-bearing rows, no malformed rows. New DIV-546 and edited DIV-285 use
  the unretracted cursor-dispatch part of `AI-CLICK-050`, not its retracted drag
  semantics. Root's pre/post allocation sweeps report all 32 answers, missing zero.

## Single correction pass

The sole adversarial review returned two P2 defects; its original report is
unchanged. Both failures reproduced with the independent reviewer overlay before
the fixes. Firing-cell search now bounds the complete existing reach predicate by
`Reach + floor(TokenSize/2)`, retaining wide arithmetic and full-footprint tests.
Invalid structure targets now clear their pursuit destination, route and stall
along with the attack handle, without discarding the already-started crossing.

Permanent regressions cover a two-cell attacker reaching the independently valid
(8,10) firing cell, a blocked non-anchor body cell, radius/rounding extremes, and
spell ruin during transit18. Cancellation leaves transit17 and the same committed
cell; a save at that boundary preserves all 200 subsequent hashes without another
step toward the abandoned destination. All four original `TestReview1085` overlay
tests now PASS, including the unchanged positive identity/damage probes. The
reviewer's Attack/Combat/Commanded/Reach/Relaxation/Hash/Structure/Tap/Command
regression selection and the gated-test inventory PASS. The installed EN/RU
structure test passes again with the same zero-damage and 1000-to-minus6 results.
No damage rule, save layout, ledger policy or original-behaviour claim changed.

## Open debt and handoff

DIV-457's missing vertical route is closed. DIV-546 keeps all-class admission,
the nearest reachable firing-cell policy, unreachable cancellation and omitted
structure weapon-spell diversion/riders explicit. DIV-547 keeps lifetime policy
explicit: this build stops at signed HP <= 0, retains negative word HP for the
existing ruin presentation, and retains the structure record and obstruction.
Original global destruction, target lifetime and active-attack save reconstruction
remain Unknown. No eternal wrapping attack, original SAV writer, new structure
selection, town entry, autonomous structure AI or multi-cell area-spell rule is
claimed. DIV-284/285/286 retain their unsolved capability, town and selection debt.
Only DIV-546/547 of the reserved 546..551 range are used; the rest stay retired.

No install was written and no desktop window was opened. Reconciliation includes
story1084's exact landed master `bf4fe0600352dddaa46fdd8b88b9a84e1848e27c`, including
its world-method/writer inventory update. The only conflict was the additive
divergence-table head; both stories' rows are retained. This is a pushed focused
checkpoint, not a landing. The sole review and correction pass are complete;
root owns final Go/paired-release/scenario gates, landing and the exact-master
`builds/current/` rebuild. No second review is requested.
