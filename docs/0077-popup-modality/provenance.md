# Provenance — a popup takes everything onto itself

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-2 (a popup takes every map-screen input) | `DLG-STOP-012` — the original's halted branch runs **no pacer arm at all**, and the command drain `R0192` is among the four things it omits, which is what makes the halt stricter than the engine's own pause | High — the gate is a single image-wide enumeration returning 1 hit with 0 in orphan; the timer rival is closed by an import-table absence |
| FR-3, edge-scroll half | `DLG-DRAW-015` — the cursor globals the panel's early-outs read are the ones the **pacer's own edge-scroll block** reads, and `DLG-STOP-012` establishes that no pacer arm runs on the halted branch | High for the reader enumeration; **Medium** for the row's own reading that those three early-outs are input paths, which this spec does not rely on — the edge-scroll conclusion comes from the arm not running at all, not from the early-outs |
| FR-5 (every animation in the map picture holds still) | `DLG-STOP-012` — the halted branch omits `ANIM-CLOCK-001`'s `0x401` presentation tick, and `ANIM-CLOCK-001` establishes that animation advances on that tick | High for both rows |
| FR-5's "held time is spent, not owed" | `DLG-CLOCK-014` — the close arm zeroes the pacer phase, so the first iteration after the close rebases and issues exactly one sub-tick instead of the catch-up burst an untouched phase would fire | High — one routine read whole, every step a named instruction, the exit fixed by raw bytes |
| FR-6 (the dim covers everything but the popup) | `DLG-DIM-013` — the darkening is passed **the whole screen rect** and runs once, inside the show routine, **before** the panel's own draw; that routine is the only one of the eleven callers that passes the screen rect | High for the call, its five arguments and the once-ness; the whole-screen extent is a property of the argument list |
| FR-6's "the strength is exactly what it was" | `DLG-DIM-013` + `TERR-FOG-084` — `L = 3` through the shroud table is a per-channel gain of 13/16, which is the strength already shipped | numeric result no stronger than `TERR-FOG-084`, which is High |
| FR-1 (the rule belongs to "a popup is open") | `DLG-ENTRY-016` — all six entry points and both mission-outcome panels reach the identical mechanism, so the bit, the darkening and the gate are the same at every entry and the question resolves to "does the state differ" | High for the shared path and the two outcome arms |
| FR-8 (the suspension and the seam do not move) | the product author's earlier ruling, landed as `0073` FR-1 to FR-5 and unchanged here | owner as author — a ruling, not testimony |

## Ours by choice

| Spec anchor | What we fixed that no source asserts |
|---|---|
| the whole contract | **The product author's ruling of 2026-08-03**, verbatim: everything must darken including the HUD, the same darkening and pause arise for the in-game menu because that is a popup too, all animations must be blocked (the river and the structures are named as still moving), the map must not be movable, and the text window and the game menu must intercept everything onto themselves. He is the product's author, so this is a ruling and not testimony. Where the decode agrees it is cited above; the rest is his. |
| FR-3, the wheel zoom | No published row addresses a wheel zoom, and this tree's zoom is its own control. Holding it with the pan is authored: a map that cannot pan but can still zoom is a half-intercepted screen, and the ruling is about the map not being movable. |
| FR-3, the keyboard pan | Follows from `DLG-STOP-012` only as far as "the pacer arm does not run"; no row names the original's pan-key handler. Held with the other three because FR-1 forbids one input having its own condition. |
| FR-4 (a gesture in flight is abandoned) | Nothing published addresses a gesture interrupted by a panel. Authored as the extension of the shipped rule that nothing survives a release: the next press starts a fresh anchor rather than resuming. |
| FR-6, which surfaces stand over the dim | The original has no counterpart to the debug readout at all (`DLG-DRAW-015` records the absence), so where it goes is ours. It darkens because the ruling says everything does, and exempting an instrument the original does not have would be inventing an exemption. |
| FR-7 (a screen that cannot draw a popup) | Ours, inherited unchanged from `0073` AC-8. No source addresses a panel that fails to load; the reason is a product reason — a dim a player cannot un-dim, and a screen whose inputs are taken by a box that is not there. |
| P-4 (the cadence is declared only on a change) | Ours, inherited unchanged from `0073` P-4. |
| the term **popup** | Ours. The decode calls it a panel and `DLG-MODAL-017` shows the image already contains two modality mechanisms; the product's word for the class the author ruled about is this contract's. |

## Open

| What | Why it is left open |
|---|---|
| Whether the original holds the **wheel zoom** and the **keyboard pan** specifically | No published row names either handler. FR-3 covers them by ruling, and the spec discloses that no fidelity is claimed for the zoom in either direction. |
| Whether the original's own darkening covers its equivalent of a unit information panel | `DLG-DIM-013` establishes the whole-screen extent of the darkening and `DLG-DRAW-015` that nothing else is drawn differently, but neither row establishes what is already on the screen when the darkening runs. `0074` recorded the panel's brightness as a genuine disagreement on the strength of that gap; this story changes the behaviour on the author's ruling and not on a new fact. |
| The player's own pause and the water | Narrowed by FR-5 to popups only, and left open for the pause key. No row addresses the engine's pause and its ambient clock together, and `SESS-PAUSE-020` is cited by `DLG-STOP-012` only for the contrast that the halt is stricter. |
| Whether a later popup will be drawn by the map screen's viewer or owned above it | FR-1 requires only that it raise the one answer. Which side owns the second popup is that story's decision. |

## Removed

| What was dropped | Why |
|---|---|
| A requirement that the readout freeze its frame-rate number with everything else | It is not part of the map picture, and freezing an instrument's own liveness is not what the ruling asks for. Moved to the disclosure list instead of a requirement. |
| A claim that the original also halts a wheel zoom | Not established by any row. It would have been an unverified fact asserted as fidelity; the behaviour stayed and the claim went. |
| A requirement that the in-game menu exist | The author named it as the second popup, not as this story's deliverable. Kept as FR-1's inheritance rule plus a disclosure that it is not built here. |
| `analysis.md` | Its two load-bearing findings — that unit animation is already frozen by the shipped suspension, and that the viewer keeps a second wall-clock accumulator nothing about the suspension reaches — are baseline facts the plan records and evidence this ledger carries. Nothing in the contract became unrecoverable. |
