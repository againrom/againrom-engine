# Provenance — the two-stage search

Claims are cited at the `research/` submodule pin `a13b3b8`, each at the confidence the clause
relied on carries there rather than its row's headline — several rows are High on a mechanism and
Medium or Unknown on the part a consumer wants. `claims/retracted.md` and the registry's standing
corrections were read first; neither touches the movement ledger.

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 — two searches over two relations, the first of them structurally unable to see a unit | `MOVE-PLANE-005` | High for the two plane displacements and the mask bits that make the split work; the row's writer *enumeration* is Medium and is unused here |
| FR-2, FR-6 — the unit-blind route is recomputed on a target change, not per tick | `MOVE-REFRESH-012` (c) | High — the counter, its threshold, its increment and its reset are named instructions |
| FR-4 — the unit-aware route is rebuilt at least once per cell stepped and does **not** aim at the final goal | `MOVE-REFRESH-012` (a), (d) | High — the free-on-transit site and the target-selection arms are one routine's own branches |
| FR-5 — the sub-goal is a waypoint a few cells along the unit-blind route, the look-ahead being 3 | `MOVE-REFRESH-012` (d), `MOVE-PARAM-006` | High — the seven scalars are immediates in one routine and the shipped file agrees 7/7 in both roots |
| FR-4 — the slack term is 3 for the near search where it is 5 for the far one, over the same `max(scalar, D>>2) + D` budget | `MOVE-PARAM-006`, `MOVE-TERM-003` | High — both budget forms and the three stop tests are named instructions |
| FR-7 — a unit blocked by another unit waits rather than pushing, swapping or stepping aside | `MOVE-WAIT-008` | High on the caller's control flow; **which** blocker yields wait versus re-search is Medium there, and nothing here rests on it |
| FR-9 — update order is the whole of contention priority, the original applying none | `MOVE-TICK-009` | High — no ordering instruction exists in the loop body |

## Ours by choice

| Spec anchor | What is fixed here, and why no source asserts it |
|---|---|
| FR-3 — the stored route in the byte form and in the digest, the version raised, the previous one refused | Our replay contract, not the original's, which serializes no route at all. Settled by measurement: over 130 436 recomputations of a stored route from its own later cells, 129 disagreed with the tail and 51 of those refused outright, so an unhashed route is a divergence with no witness |
| FR-2 — **no** bound of any kind on the unit-blind search, and the deletion of the rectangle the optimised mode had | Ours, and it is the precondition the window rests on: a window is safe exactly when something else is unbounded. The original bounds by generations and never in space, so a rectangle here was ours to begin with and is ours to remove |
| FR-4 — the window: a square of half-width **8**, centred on the mover, clipped to bounds | Ours entirely. Nothing decoded bounds a search in space; the near search there is bounded only by a budget derived from a near sub-goal, and ours makes the bound independent of that distance — which is what makes a per-tick cost statable at all |
| FR-6 — the three tests that make a stored route stale | Ours. The original uses counters — a re-search count against a refresh rate — where these are tests on the stored route itself, so nothing counts and no second byte is stored |
| FR-7 — a unit-blind failure ends the order in the tick that finds it | Ours, and argued rather than decoded: the relation cannot change while a world is advanced and the unit does not move when its search fails, so a second search would be the same search. The original instead substitutes a nearby destination — its pickers are unread, so none is invented here |
| FR-7 — holding, the stall count and the threshold 16 | Ours and **provisional**, inherited unchanged; no give-up counter exists in the original |
| FR-5 — one arm where the original has three | Its selection takes the near waypoint, else one further along, else the final goal when few nodes remain. Our route's last cell **is** the target, so the third arm is vacuous and the other two collapse into one index |
| FR-8 — the cost the contract demands, in labelled cells and in milliseconds | Ours: a performance requirement is a property of our build, and no claim speaks to one |

## Open, and deliberately not consumed

- **Which cell the original substitutes when a search fails.** Both pickers are named and unread,
  and the promoted spec leaves them unspecified. Failure stays handled our own way.
- **Reservations.** `MOVE-CLAIM-007` is High that a unit stamps the cell it means to enter and
  **Unknown** on whether its own stamp blocks its own next search. None is implemented.
- **The verdict table behind waiting.** `MOVE-WAIT-008` grades it Medium, two helpers unread. A
  blocked unit waits in every case here, which is that row's High clause and no more.
- **The refresh cadences.** `StaticRefreshRate` 16 and `DynamicRefreshRate` 32 are decoded at
  High and **not** consumed: this contract recomputes on staleness, not on a period. A later
  story that wants periodic revalidation has the numbers waiting for it.
## Removed, and refused

- **The start-goal rectangle grown by a margin.** The bound the imported baseline put on its one
  search, and the reason this story exists: it makes a goal reachable only by a wide detour
  unreachable, which on a map with a lake is not a corner case. Removed from the optimised mode
  rather than kept beside the window, so exactly one spatial bound exists in the system.
- **A flow field.** One flood from a shared destination would serve a whole group's routes at
  once. It selects different routes, so it is a different contract rather than a cheaper
  implementation of this one.
- **Any staggering of searches across ticks.** The ledger is explicit that nothing in the
  original is staggered; ours would be an improvement, and it belongs to a later story.

## The threshold this story is written to

Policy sets **High** for any fact reaching hashed simulation state, and this story puts a route
there — so High is the threshold, and a measurement decided that rather than an assumption either
way. Every research-backed row above is cited at High **on the clause relied on**, checked clause
by clause rather than by row headline, and this story opens no research item.

The facts that actually steer behaviour here — the window, the staleness tests, the failure
rules — are **ours**, so no confidence grade applies to them. What applies instead is a
criterion: a world marshalled at any tick and decoded into another advances identically, digest
for digest.

## The revision, and two sources ahead of this tree's pin

`MOVE-TERM-003` states **three** things about termination and the FR-4 row above took two. The
third is the static 1x1 arm **substituting a flat 1000** generations when `Player+0x28` on the
mover's owner is zero **and** the goal's own footprint is free on the static plane. "Both budget
forms" is accurate about the forms and silent about the override, which was neither implemented,
nor disclosed as a divergence, nor carried as an open question — the omission FR-10 repairs.
`UNIT-OWNER-009` names that field: it is the owning `Player`'s and reads **0 exactly when a human
participant owns the unit**, 1 or 2 for every scenario-authored owner. So the flat 1000 is not an
exotic branch — it is the ordinary budget of a **player-ordered** move, and the scaled form is the
scenario's own.

Both are **ahead of the submodule pin `8c92427`** this tree is frozen at, which this revision does
not bump: that is the story boundary's, and nothing here can check either until it happens. The
same amendment retracts `MOVE-TERM-003`'s "the *caller* then substitutes a nearby goal" — the
search does it in its own tail — which reaches `0037`, and corrects rather than writes over the
"Open" entry above on failure handling.

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-10 — the flat **1000** and both conditions gating it | `MOVE-TERM-003` (amended) | High — the three stop tests, both budget forms and the override are named instructions |
| FR-10 — the gated field is the owner's, zero for a human participant | `UNIT-OWNER-009` | High for the identification and the value space; its Medium clause, that nothing changes it at runtime, is not relied on |

| Spec anchor | What is fixed here, and why no source asserts it |
|---|---|
| FR-10 — the flat budget on **every** far search | A **premise**, not a divergence: every order this build can issue is a player's, since `sim.Entity` carries no owner and nothing here scripts a scenario, so the condition holds wherever it is asked. An ownership story inherits the obligation to re-read it |
| FR-10 — the goal-free half, established structurally | Our footprint is one cell and our static plane is `terrainOpen`, and the search returns before the budget is computed unless the goal is open under the relation — so that half holds wherever a budget is read at all |
| FR-11, FR-12 — the ceilings, and that the byte form does not move | Ours: a cost bound and a compatibility bound are properties of our build, and no claim speaks to either |

Open: **ownership itself**, absent from `sim.Entity`, whose 1 and 2 are what would make
**`StaticScanAhead` = 5** reachable again — decoded at High and, from FR-10 on, consumed by
nothing *here*, the branch it feeds being the scenario's rather than the player's.
`DynamicScanAhead` = 3 is still FR-4's.

**2026-07-31 — both sources are at the pin.** It moved `8c92427` → `f35be34` at the 0047 boundary,
so the paragraph above is wrong: neither is ahead of this tree, and both were re-read here rather
than taken on report. No clause relied on differs. One detail the papers did not carry:
`UNIT-OWNER-009` records the `Player` constructor's own default as **1**, so an owner field, when
it arrives, defaults to the scenario's budget and not the player's. The Open row inherits that.

## Appended 2026-08-01 — pin `130bb79`: the retraction this file recorded now has a Kind, and `analysis.md` still carries the old wording

The 2026-07-31 note above already records that `MOVE-TERM-003`'s "the *caller* then substitutes a
nearby goal" was retracted and that the search does it in its own tail. At `130bb79` that row is
classed **REFUTED** — not a label that moved but a reading that was wrong — with the cost stated:
a consumer that believed it "put the fallback in the wrong routine and picked by distance where
the engine picks by label".

One copy in this story was not reached by that note and is left as written, being a dated record of
what the claims gave at the time: `analysis.md` lists among what the claims do **not** give *"what
a failed search should do beyond 'the caller substitutes a nearby goal', whose pickers are
unread"*. Both halves are now false — the pickers have been read (`MOVE-ALT-019`, `MOVE-ALT-021`:
Chebyshev rings scanned whole, smallest **label** wins, so the substitute is cheapest-to-reach and
not nearest) and the substituting routine is the search. FR-4 and FR-10 cite the budget half only
and do not move; `0037` consumes the corrected reading and `0029` has been swept to match.
