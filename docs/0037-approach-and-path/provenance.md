# Provenance — 0037 approach and path

Submodule pin: research `f35be34`. Every row below was read at that pin.

## Claims this story is built on

| Claim | Confidence | What it settles here |
|---|---|---|
| `MOVE-ALT-018` | High | The substitution happens **inside the search's own tail**, not in the caller, and it is reached when the goal is still unlabelled. The static branch takes the ring picker around the requested cell with a bound of `(D>>2) + 4`. `altTarget` is a target **actor** and selects the other picker; nothing here passes one. |
| `MOVE-ALT-019` | High | The picker itself: rings `r = 1, 2, …` around the requested cell, the four sides walked for `i = -r..r`, the **whole ring scanned before its best is taken**, the accept **strictly** less, the centre never probed, and growth while `r + 1 < limit`. |
| `MOVE-ALT-021` | High | What a candidate is tested against and what the choice is measured from: **the label plane and nothing else**. "Free" means the failed wave labelled it — a passable cell the budget never reached is invisible — and the metric is the label, so the cell taken is the one cheapest to reach **from the mover**, not the one nearest the request. Named the rival it half-confirms, which is why the criterion for it discriminates rather than illustrates. |
| `MOVE-TERM-003` (amended) | High | The flat thousand is an **override with a gate**, and one half of that gate is the goal's own footprint being free on the static plane. A search that must settle has a goal that is not free, so it falls through to the computed form — and the form the far arm carries has slack **5**, where the near arm's is 3 (`MOVE-PARAM-006`). |
| `MOVE-SEARCH-001` | High | The search is a label-correcting wave with no closed set and no distance-estimate term, which is why the picker can read the plane the search leaves and why there is no "expanded node" set for the baseline's rule to have been written over. |
| `MOVE-COST-002` | High | A label is accumulated step cost, which is what makes "minimum label" mean "cheapest to reach". |
| `MOVE-ALT-022` | High / Medium | The substitute is consumed by one route extraction and **never written back**; the order goes on naming the cell that was asked for. The row's own Medium clause is the run-time consequence — that the unit re-substitutes from wherever it then stands — and it says outright that this was derived, not observed. See the divergence below. |
| `MOVE-ORDER-023` | Medium | No multi-unit distribution exists anywhere: a group order writes one cell into every member. The spread a player sees is emergent, each mover settling for itself against a plane that already carries the earlier movers' claims. Nothing here distributes a group order, and that is this row's doing rather than an omission. |

## Ours by choice

- **Which searches settle.** The decoded search substitutes on both its static and its dynamic
  branch. Only the far search settles here. The near search is aimed at a sub-goal four cells off on
  a route the far search chose, and a near search coming back with some other cell would be a second,
  unstated rule about where a mover steps; the stall counter and the give-up that rest on its failure
  are landed contract (0045 FR-7) and are left alone.
- **Which mode settles.** The optimised search is ours and reconstructs nothing. It floods outward
  from the **target**, so a failed one leaves a plane of costs *to the target* — which says nothing
  about what the mover can reach. There is no substitute derivable from it, and none was invented.
- **The ring bound and the budget are honoured, the tolerance is not.** `MOVE-ALT-022`'s Chebyshev
  tolerance test raises a UI message and, by that row's own reading, does not cancel the move. There
  is no message channel here to raise one on, so nothing is implemented and nothing is faked.
- **The line's colour and width.** Render-side choices with no claim behind them.
- **How many lines are drawn is no longer a choice at all.** *(2026-08-01.)* This row read "the
  line's colour, width and **cap**", and the cap was the one thing in it that decided what a player
  could see: over thirty-two drawable units the overlay drew nothing. The owner ruled against it from
  a build. It is **removed** rather than re-set, because a different number would be the same
  unbacked judgement, and what replaces it is measured rather than chosen — a leg that cannot reach
  the window is not issued, which changes no pixel at any camera and removes 95% of a full-map group
  order's work at the default zoom. The figures are in `verification.md`; the residual it does not
  cover is `plan.md`'s R-5. No claim was consulted for any of this and none is cited: it is a fact
  about **our** renderer, measured here, not a fact about the game.

## Divergence, disclosed

**The order takes the cell the search settled for.** `MOVE-ALT-022` is explicit that the engine does
not do this. The reason it can afford not to is `MOVE-REFRESH-012`: the static route is recomputed
only on a target change or every sixteenth dynamic re-search, so re-substituting costs a sweep
occasionally. This tree has no period anywhere — a stored route is replaced exactly when it stops
serving, and a route whose last cell is not the current target never serves — so an order left
pointing at an unreachable cell would run a whole-map sweep on every tick of the walk. Taking the
substitute as the order's cell makes the sweep run once.

What it costs, stated rather than hidden: a mover settles for the **first** substitute rather than
the one its later position would have chosen, and a cell freed after it settles will not be
re-tried. Reverting this needs a refresh period, which is a decision 0029 took in the other
direction on measured evidence and is not this story's to reopen.

## What this story supersedes

0045's **AC-2** — "three orders no route serves … each target is cleared on the first tick, the unit
unmoved" — is a live assertion and is no longer true of the canonical mode: two of its three
fixtures are now walked. The criterion is amended in place in that story's own contract rather than
left to be found by running its tests. Its **FR-7** is untouched and deliberately so: a far search
that finds no route still ends the order in that tick, and what this story changes is when a far
search finds none.

The same supersession reaches 0036's AC-9 through `pkg/mapload`'s routing tests, where a crossing
closed by water was witnessed against 0029's landed rule rather than against AC-9's sentence. It is
re-witnessed the same way and the disagreement is again reported rather than absorbed.

## Not consulted

No third-party reimplementation, port, decompilation or format schema, and no web source, was read
for any fact here. The baselines folded into this story are our own earlier clean-room work and were
treated as hypotheses: where one disagreed with a decoded row, the row was taken and the
disagreement written down above.

## Appended 2026-08-01 — pin `130bb79`: `MOVE-TERM-003`'s substitution clause is now classed REFUTED, and this story is on the right side of it

`retracted.md` gives the clause a Kind for the first time: **REFUTED** — "the substitution is real
but it is not the caller's, and the substitute is not the nearest free cell. A consumer that
believed it put the fallback in the wrong routine and picked by distance where the engine picks by
label." Recorded here because this is the story that would have been built on it.

It was not. The `MOVE-ALT-018` row above already states the substitution happens **inside the
search's own tail, not in the caller**, `MOVE-ALT-021` already states the metric is the label
plane and not proximity, and `MOVE-TERM-003` is cited only for the budget half, which survives.
Nothing in this contract moves; the sweep's finding is that nothing needed to.
