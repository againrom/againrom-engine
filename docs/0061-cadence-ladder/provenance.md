# Provenance — the cadence ladder

Pinned at research `cf68f4d`, frozen for the story; `claims/retracted.md` was read first and
holds nothing bearing on terrain animation or the game-speed setting.

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| **The game offers nine speed settings and its own two keys step that setting by one** (FR-1) | `TERR-ANIM-008` — `SetGameSpeed` maps a clamped index `0..8` to `tps ∈ {8,10,12,14,16,20,24,28,32}`, and **the in-game `+`/`-` keys step `idx±1`** (`R0231`) | High |
| **Each setting's period is `1000/tps` by integer division, and map load pushes index 4** (FR-1, FR-2) | `TERR-ANIM-008` — the division is a named instruction, index 4 gives 16 tps, 62 ms, ~992 ms a water cycle | High |
| **The ambient animation counter follows that same setting** (FR-5) | `TERR-ANIM-007` — a whole-binary write scan finds exactly two writes to the counter, the increment being the handler of the paced logic tick; `ANIM-CLOCK-001` — the same function advances every drawable's animation and increments that counter in one pass, and states the consequence for a consumer: the frame "must be stepped by the same tick as the simulation — it slows when the game speed drops" | High / High |

The third row is the one this story consumed that 0041 did not have: `ANIM-CLOCK-001` was published
after 0041 landed. It settles the water counter's cadence **positively** rather than leaving it to
be ruled — the counter is not a second clock that happens to agree, it is the same tick.

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| **Eight rungs of extension past the ends of the shipped set** — three below its slowest speed and five above its fastest, doubling and halving, reaching 1 and 1024 ticks a second (FR-1) | Divergent in range, and the claim is explicit about the range it extends: `TERR-ANIM-008` (High) has nine speeds, 8 to 32. The **span** is 0041's unchanged; what this story changes is that it is now reached only past the ends of the nine, and is reported as being past them. Nothing decoded says the original could run at 1 or at 1024 |
| **A step of one rung across the extension, doubling rather than a rate step** (FR-1) | Nothing decoded. Inside the shipped nine our step is the original's own `idx±1`; outside it there is no original to be faithful to, and 0041's objection stands — a step of one over a range of 1024 is not a control |
| **Saturation at both ends: a press past an end changes nothing and is not remembered** (FR-3, P-2) | Nothing decoded. `TERR-ANIM-008` records a clamp on the index and publishes no behaviour for a key pressed at an end. **AUTHORED** — and the alternative was refused on its consequence, not on evidence: a ladder that remembered presses past an end would leave a key visibly doing nothing for a while afterwards |
| **The readout's setting row — `5/9`, `EXT +1`, `-`** (FR-4) | Nothing decoded describes a parameter readout in the original and none was looked for; the whole box is ours (0060). **AUTHORED.** What is *not* ours is the number in it: which of nine, and the nine, are `TERR-ANIM-008`'s |
| **The stop, and water running while the world is stopped** (unchanged from 0041 FR-3, FR-5) | See the disclosure below |

## Disclosed divergences

**Water animates while the world is stopped, and a claim now bears on that.** 0041 recorded the
stop as entirely ours with "nothing decoded" beside it, which was true at its pin.
`ANIM-CLOCK-001` (High) has since published the other half of the sentence this story quotes
above: the drawable's animation "slows when the game speed drops **and stops when the tick
stops**". Our stop stops the tick and does not stop the water, so what 0041 called an absence of
evidence is now a **stated divergence**. It is left standing rather than changed here: 0041 FR-5
is a shipped contract and the original has no pause for its own tick to be stopped by, so the
claim describes a state the game cannot enter. Recorded so that a later story reopening the pause
finds the claim already cited rather than rediscovering it.

**What this story RETIRES.** 0041's appended ruling of 2026-08-01 disclosed that the first cadence
key press of any kind moved a never-rated map from the game's 62 000 µs to our rate model's
62 500, taking the water cycle from 992 ms to 1 000 ms and leaving us 0.8 % slow from that press
onward. That divergence is **gone**: every value the keys can write is a rung, the map opens on a
rung, and a press that moves only the stop writes back the period the clock already held. The
0.8 % figure should not be carried forward from 0041's provenance; this row is where it ends.

## Open / undecoded

- **What the original's `+`/`-` do at the ends of its own table.** `TERR-ANIM-008` names the clamp
  on the index and not the key's behaviour at a clamped end. Ours saturates, which is ruled above.
- **Whether the original persists the setting.** `TERR-ANIM-008` decodes the config path and is
  **Medium** on which index a resumed session takes. Deliberately not consumed: this story persists
  nothing, so the weaker half is not load-bearing.
- **What the other paced arms do.** Unchanged from 0041, and nothing here assumes an answer.

## Confidence, and the threshold this story is written to

**Medium.** The clock is above the determinism wall and nothing here reaches hashed simulation
state: `pkg/sim` counts ticks and not seconds, takes no duration, imports no clock, and no file
under it was touched. The cadence decides how often `Step` is called and the stop whether it is
called at all — neither is an argument to it.

The two High rows above are what the story **keeps**; the row that is new to this story is the
water counter's, and it is High for the same reason — it is a decode, not a decision.

## Removed, and refused

- **Rounding the map-load period to 62 500.** Refused on consequence: the 62 ms is the game's own
  integer division, and the 992 ms water cycle that follows from it is the figure a published
  movement-rate prediction was checked against. Rounding would break the one independently
  confirmed quantity in this area to make a control tidier.
- **Deleting the wider range.** Refused: it is wanted. What was refused is its *shape* — a parallel
  model that silently replaced the shipped one.
- **Snapping an off-ladder period to the nearest shipped setting** in the readout. Refused: it
  would state that the game offers a setting the game does not offer, which is the substitution
  this story exists to remove.
- **A rate on the cadence seam.** Refused on measurement: the shipped periods are a truncated whole
  millisecond and no microsecond quotient of a rate reaches them, so a rate cannot carry a shipped
  setting across.
