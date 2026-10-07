# Mission 121's bridge troll: an investigation, not a fix

## Report

The owner (Russian, verbatim): "в миссии 121 тролль застревает на мосту, видимо,
у нас что-то с выбором цели. сам тролль при этом пройти мост может" — a troll
gets stuck on the bridge on mission 121; the owner suspected target selection,
but noted the troll can cross the same bridge when nothing else is happening.

## Result

Mission 121 has two Fat trolls, and the report is about the one this story's
first pass did not look at. Unit #30 (script id presumably `u63`, matching this
map's own u61→entity29/u64→entity31 pattern, not independently re-verified),
posted near the Horisontal Bridge (structure anchor (35,14), deck row y=15), is
the troll the owner watched: standing on the bridge deck, its attack order held
the whole time, making almost no forward progress. The owner's own save files —
an earlier save (tick 1,191,016) and a later save (tick
2,917,740, roughly 1.73M ticks later, both read-only, never modified) — hold it
there in both snapshots:

| | earlier save | later save |
|---|---|---|
| position | (35,15), on the deck | (37,15), on the deck |
| `HasAttackTarget` / `AttackTarget` | true / entity 44 | true / entity 43 |
| `HasTarget` / destination | true / (36,15) | true / (39,15) |
| `Stall` | 3 | 13 (of a 16-tick cap) |

Two cells of net progress over ~1.73M ticks, the order held continuously in
both snapshots, and the attacked party member changed (44 to 43) without the
order ever releasing: this is a sustained jam, not a single-frame artefact.

The cause is an occupancy conflict, not target selection and not the leash
mechanism this story's first pass proved for the *other* troll (kept below as
a secondary finding). Unit #30 has `TokenSize=2` (a 2x2 footprint). Both saves
show party members packed onto the same stretch of deck it is trying to cross
— five in the second save (ids 43-47). Querying the loaded world's own
`terrainOpenSize`/`occupancyOpenFootprint` around the troll's position in
save 2 (footprint anchor is the cell's top-left corner, per `route.go`,
spanning to `(x+size-1, y+size-1)`) shows every forward cell on the deck
refused: one candidate is terrain-open but occupancy-refused because its
footprint span overlaps a party member's cell; the next candidates are
occupied outright; the rows flanking the deck are terrain-closed for a 2-wide
footprint, so there is no way around. The only open cell in the probed area is
directly behind the troll.

The victim never leaves the troll's sight (`ScanRange`), so `candidates()`
never comes back empty and `decide()` never releases the order
(`AI-GUARD-012`'s own release condition — proven real for the *other* troll,
below — simply never fires here). Each tick `searchRoute`'s near pass finds no
admissible step and increments `Stall` (`pkg/sim/step.go`); at the 16-tick cap
(`stallLimit`), `restAt`/`clearOrder` cancels only the pending walk
(`TargetX`/`TargetY`/`HasTarget`/the stored route), not `HasAttackTarget`/
`AttackTarget` — those are cleared only by `releaseAttack`/`clearAttack`.
`approach()` runs every tick regardless (`combat.go`, "it runs every tick for
an actor holding an attack order"): finding the victim still not in reach, it
reissues the same destination via `walkTo`, and the whole cycle — sixteen
ticks of failed search, a one-tick cancel, an immediate reissue — repeats.
That is what the owner's saves show 1.73M ticks apart: an attacker that is
never released and never advances, which reads exactly as "stuck on the
bridge."

## Proof

`pkg/sim/story1193_escort_stall_test.go`,
`TestAnAttackerBlockedByItsTargetsEscortHoldsTheOrderAndNeverAdvances`,
reproduces the mechanism with a synthetic fixture built from the owner's own
save numbers (a 2x2 attacker, a two-row corridor whose 2x2-admitting anchor is
one row wide, five 1x1 bodies packed across it) rather than the map or the
saves themselves. It documents CURRENT behaviour, not correctness: over 400
ticks after acquisition, the attack order stays held throughout, the attacker
never advances past the escort's leading edge, `Stall` climbs to one below the
16-tick cap at least once, and the walk order is seen cancelled and reissued —
the same give-up-and-retry loop the saves show. `go test -run
TestAnAttackerBlockedByItsTargetsEscortHoldsTheOrderAndNeverAdvances
./pkg/sim/` passes today, against unmodified `pkg/sim` source.

## Open debt

`DIV-1316` (`docs/DIVERGENCES.md`, table "Authored where research is silent",
type `UNKNOWN`): no claim states what an original attacker whose route to a
held target is refused by the target's own escort does — retarget, find
another approach, disengage, or hold and stall exactly as this build does.
This build's own behaviour is authored, not researched; the row does not
claim the original behaves either way.

No fix is in scope for this story. The owner is deciding separately whether
this build should behave differently (retarget past a jammed route, or route
around one) — that is a deliberate choice about this build's own behaviour,
not something today's finding settles by itself.

## Secondary finding: entity 29's leash (does not explain the report)

This story's first pass investigated the *other* Fat troll on this mission —
unit #29, script id `u61`, guard post (12,21), near the Vertical Bridge at
(10,24) — before the owner's own saves identified which troll he had actually
watched. That investigation is real, decoded ROM1 behaviour, and both of the
owner's saves show entity 29 idle at its post (`HasAttackTarget=false`,
`hasTarget=false`), uninvolved in his report. It is kept here as a secondary,
correctly-scoped finding, not as an explanation of the bridge report.

A guarding group's target is rewritten from nothing on every 16-tick decision
cycle (`AI-SCORE-069`, High confidence, `ord+0x20` read end to end): nothing
about an in-progress chase is sticky. A candidate list is the union of every
member's raw sight for that one cycle only (`AI-GROUPSEE-068`, High
confidence, `R0110` read end to end); nothing persists it between
cycles except the retaliation-only "remembered attacker" byte
(`ord+0x58`/`ord+0x5a`, 20-cycle decay), which requires a landed blow. When a
guarding group's candidate list comes back empty, the whole order releases and
an off-post member is walked directly back to the exact cell guard first ran
on (`AI-GUARD-012`, High confidence, `R0137` read end to end: "home is
emergent... beyond it the actor gets order+8=1... walk back").

Entity 29's own walking speed (about 1 cell per 12-17 ticks) is slower than an
ordinary party's. Over one 16-tick decision window a retreating party opens
the gap past 5 cells before the troll closes it: it gives chase, gets partway
toward its own bridge, loses raw sight, and the entire order releases in the
same tick that walks it fully back to (12,21) — short of ever setting foot on
that deck. A direct `KindMoveTo` waypoint order (no AI decision in the loop)
crosses that same bridge deck in full, which is the "the troll itself can
cross" half of the report — just not the half that explains where the
sighted troll actually is.

`scenarios/1193-mission121-bridge-guard-leash.json` drives mission 121 with
this troll (`u61`) and its own single-member party, retreats it at an ordinary
walking cadence, and asserts the troll leaves its post, gets partway toward
the deck, and returns to (12,21) without ever reaching it. `bash
pipeline/check-scenarios.sh <EN root> 1193` passes today, no code change.
`pkg/sim/release_test.go`'s
`TestAReleasedMemberHoldsNoneOfItsSevenFields` and
`TestAReleasedMemberWalksHomeThenStandsThereForeverAfter` (story 0106,
predating this story) already pin the release-and-walk-home mechanism at unit
level.
