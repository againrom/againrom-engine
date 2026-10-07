# Provenance — the world's own clock

Pinned at research `a13b3b8`, frozen for the story; `claims/retracted.md` and the
registry's standing corrections were read first, and neither bears on terrain
animation.

## Backing

**Two rows, and both of them are what this story keeps rather than what it adds.**

| Spec anchor | Source | Confidence |
|---|---|---|
| **The game's own nine speeds keep their own periods** — 16 ticks a second at 62 ms, a water cycle in 992 ms, an index outside the table clamped (FR-2) | `TERR-ANIM-008` — the paced loop fires one tick per `dtMs`; `SetGameSpeed` computes `dtMs = 1000/tps` by integer division over a clamped index 0..8 mapping to `{8,10,12,14,16,20,24,28,32}`, and map load pushes index 4 | High (per-index cadence and the map-load default) |
| **The water counter is paced by the world's tick and by no clock of its own** (FR-5, C-2) | `TERR-ANIM-007` — a whole-binary write scan finds exactly two writes to the counter, one init and one increment, the increment being the handler of the logic-tick message; nothing else advances water | High |

Everything else the contract fixes is below, and none of it is derived from a claim.

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| **A rate model spanning 1 to 1024 ticks a second** (FR-1, C-3) | Divergent in range, and the claim is explicit about the range it replaces: `TERR-ANIM-008` (High) has nine speeds, 8 to 32, on an index the in-game keys step by one. Ours is not a reproduction of that and must not be read as one. What *is* the same is the **kind** of control: there a speed change moves the tick period and nothing else, so scaling the tick rate is the original's own notion of game speed, and only its span is ours |
| **A stop — an active pause, the world halted, the screen not** (FR-3, FR-4, C-1) | Nothing decoded. The pinned ledgers hold no pause on any screen; the loop the corpus decodes has a period and no zero, and no claim names a key, a message or a flag that suspends it. Entirely ours, including every clause about what keeps running |
| **Our own rates get exact microsecond periods while the decoded nine keep their truncated milliseconds** (FR-1, FR-2) | The truncation is the game's own arithmetic — `TERR-ANIM-008` records the division that produces 62 rather than 62.5, which is why a cycle takes 992 ms. It is a fact about the original and is kept for the speeds that are the original's. A rate of ours is not a number the game ever held, so it inherits none of that and is carried to the microsecond instead |
| **The catch-up bound is a span of world time, dropped rather than queued** (FR-6) | Nothing decoded. `TERR-ANIM-008` gives the original's loop a wall-clock accumulator and publishes no bound on one call's catch-up. Ours drops the surplus, so a stall costs world time once |
| **Space stops, and two keys double and halve the rate** (FR-7) | Half divergent, and the divergence is named: `TERR-ANIM-008` decodes the in-game `+`/`-` pair stepping the speed index by one. We keep the pair and give it a different ladder, because a step of one over a range of 1024 is not a control. Space is ours entirely — there is nothing for it to be faithful to |
| **A rate scales the tick, never a unit's stride** (C-4) | The original keeps the two apart as well, and more finely: `TERR-MOVE-056` and `MOVE-SPEED-011` (both High) derive a per-step duration per unit and per cell, so a mover's own speed is a second quantity there beside the tick period. Ours has one cell per tick — 0026's choice, not this story's — so there is only the one quantity to scale |

## Open / undecoded

- **Whether the original had a pause at all.** Nothing in the pinned ledgers speaks to
  it and no research item was opened — a decision, not an omission. The stop is ours by
  construction, so a decoded answer could not change the contract; it could only say
  afterwards how far from the game this lands.
- **Which speed a resumed session runs at.** `TERR-ANIM-008` is **Medium** on that half
  and decodes the config path that loads the index. Deliberately not consumed: this
  story persists nothing, so the weaker half of the claim is not load-bearing here.
- **The engine's other paced arms.** The claim names the real-time single-player loop,
  selected by a field test. What the other arms do with the cadence is undecoded, and
  nothing here assumes an answer.

## Appended 2026-08-01 — the owner's ruling on FR-2's last clause, and the divergence it discloses

`verification.md` handed the owner a contract question the papers underdetermined,
and this is the answer.

**The question.** Both cadence consumers are born at the map-load speed's period,
62 000 µs, so an opened map needs no call (DD-6) — but the flow's own rate is born
at **16**, the same speed expressed as *our* rate. DD-5 writes one rate to both
consumers in one unconditional statement, so the **first** cadence key press of any
kind, including Space, which selects no rate at all, writes 16 to both. The rate
model computes 1 000 000 / 16 = **62 500 µs** exactly where the game's own integer
division computed 1000 / 16 = **62 ms**. The period moves by 500 µs and the water
cycle from **992 ms to 1 000 ms**, on a map nobody ever re-rated. FR-2's "a map
opened and never given a rate MUST run as it does now", P-3's "there is no third
state" and DD-5's single unconditional write cannot all three hold.

**The ruling, 2026-08-01: 62 500 and 1 000 ms are both acceptable.** So P-3 and DD-5
stand and **FR-2's last clause is what was amended**; no code moved. The alternative
was to have each consumer remember the last rate it was told and re-rate only on a
change, which invents a second rate value on each side of the seam — the exact thing
DD-5 exists to prevent — and needs the "no rate yet" third state P-3 forbids.

| What the spec now fixes | What the evidence says |
|---|---|
| **The first cadence input of any kind moves a never-rated map onto the rate model** (FR-2) | A **disclosed divergence, by ruling**. `TERR-ANIM-008` (High) has the game compute its default full tick as 1000/16 truncated to 62 ms, so a water cycle there is **992 ms**; after any cadence key ours is **1 000 ms**. We are **0.8 % slow** against the original from that press onward, and the figure is stated here rather than designed around. Nothing decoded is contradicted — the divergence is in what OUR rate model does with the game's own number, not in the number |

The 0.8 % is invisible on screen and does not compound: it is a fixed period, not a
drift, so the two clocks stay 0.8 % apart forever rather than separating. It reaches
hashed sim state through the tick count, which is why it is disclosed at all — a
digest taken over the same wall-clock span differs between a map that was paused
once and one that was not.

## Confidence, and the threshold this story is written to

Policy sets a **High** threshold for anything reaching hashed simulation state, and the
check that this rate does not — which is FR-8, and ours — is on the record rather than
assumed: `sim.Step`
takes a command slice and no duration; that package imports no clock and the
determinism scan over it fails on one; no cadence quantity appears in the canonical
byte form; and the period, the accumulator, the wall-clock baseline, the rate and the
stop are all fields of the driver and the front-end, born with the map screen and
dropped with it. A rate decides how often the step is called and a stop whether it is
called at all — neither is an argument to it. So the threshold is **Medium**, and the
two backing rows above are High for a different reason than the threshold: they are
what the story *keeps*, not what it *adds*.

What a High threshold buys is delivered by mechanism instead: the digest-equality
criteria compare one command stream advanced at two rates and through a stop-and-resume
schedule, at every tick index, and fail on a divergence rather than reporting a
confidence about it.

## Removed, and refused

- **The shipped animation switch as the pause's freeze.** `TERR-ANIM-009` (High): the
  flag forces phase 0 rather than holding the current phase — and in the original it
  reaches further, collapsing a placed object's frame to 0 as well. So "freeze the
  water" is not what that switch does in either tree, and the contract keeps water
  running rather than inheriting a freeze that does not exist.
- **Rate 0 as the stop.** Refused on measurement: the shipped clamp reads an index
  below the table as its slowest row, so 0 is 8 ticks a second and a stop written that
  way would be a slow game.
- **A millisecond period for our own rates.** Refused on measurement: 256 ticks a
  second truncates to 3 ms and runs at 333.
- **Any framing of the rate model as a reproduction.** The nine decoded speeds are the
  game's own and stay exactly as they are; 1 to 1024 is ours, and no sentence of the
  contract implies the original had it or that this is faithful to anything.
- **The decoded persistence path** — the config key the speed index is loaded from. It
  is decoded and deliberately not consumed; a later story may make it ours to schedule.
