# Provenance — 0092

Submodule pin `9fd0559`, moved for this story from master's `fe1c626` because the round that read
the cycle end to end, and corrected two older rows about its span, landed after the fork.

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 — the switch is the shipped `ShowTimeFlow` game option, not a debug flag | `TERR-LIGHT-108` | High |
| FR-1 — its compiled default is 1, stored before `main` by the static-initializer table | `TERR-LIGHT-108` | High for the store and its reachability; **Medium** that a given install runs with it on |
| FR-1 — the runtime toggle is key `N`, and it forces a relight | `TERR-LIGHT-108` | High |
| FR-2 — the clock is the sub-tick counter; sixteen sub-ticks are one full tick; one full tick is one in-game minute | `SESS-TICK-026` | High |
| FR-2 — the model's argument is `fullTicks + 360`, so a mission opens at 06:00 | `SESS-TICK-026` | High |
| FR-2 — the model adds `0x168` a second time internally, so the day band's phase is `fullTicks % 720` | `SESS-TICK-026`, `TERR-LIGHT-110` | High |
| FR-3 — five arms write the angle, selected by `hour = (t/60) % 24`, and their band boundaries | `TERR-LIGHT-110` | High |
| FR-3 — the cycle-off arm stores the literal `+0.78539815`, and it is a truncated-pi quarter and not `pi/4` | `TERR-LIGHT-030`, `TERR-LIGHT-109` | High |
| FR-3 — the day arm computes `-0.78539815 + m*0.0021816615277777777` with `m = (t+360) % 720` | `TERR-LIGHT-109`, `TERR-LIGHT-110` | High |
| FR-3 — the night arm computes `+0.78539815 - m*0.0065449845833333332` with `m = (t+120) % 240` | `TERR-LIGHT-110` | High |
| FR-3 — the dawn arm stores `-0.78539815` and the dusk arm `+0.78539815` | `TERR-LIGHT-110` | High |
| FR-3 — the total sweep is pi/2 end to end, `-pi/4 .. +pi/4`, and the two rows that said pi were wrong by a factor of two | `TERR-LIGHT-109`, `retracted.md` | High (both doubles dumped as raw bytes from the PE on both roots) |
| FR-4 — the angle reaches every consumer through ONE cache, written by the global's only reader | `TERR-LIGHT-111` | High |
| FR-4 — what the screen shows is the angle as of the last relight, never the current tick's | `TERR-LIGHT-111` | High |
| FR-5 — one unforced relight per 20 in-game minutes, on the sub-tick the full tick rolls; 72 per in-game day | `TERR-LIGHT-114` | High |
| FR-5 — a forced relight bypasses the modulo | `TERR-LIGHT-114`, `TERR-LIGHT-108` | High |
| FR-5 — a changed angle reaches the screen only through the whole per-vertex rebuild; there is no cheaper path | `TERR-LIGHT-114`, `TERR-LIGHT-022` | High |
| FR-6 — the angle enters the relief grid as `32/cos\|theta\|` and the lateral `tan\|theta\|`, and neither can exceed 45.25 and 1.0 | `TERR-LIGHT-013`, `TERR-LIGHT-028`, `TERR-LIGHT-109` | High for the consequence; the gradient form is 0007's own backing |
| FR-7 — the cycle-off and daytime arms set ambient `0x0e`, range `0x20` and tint `(0,0,0)` | `TERR-LIGHT-030`, `TERR-LIGHT-014` | High for the cycle-off arm; **Medium** for the daytime band's tint |
| FR-8 — the sprite ramp row is `ambient>>2` and carries no angle term, so no sprite moves with the sun | `TERR-LIGHT-061` | High |
| D-4, Out of scope — every shadow pass shears by `tan(shear(cached theta))`, and the unit BODY computes the same value and discards it one instruction later (`FSTP ST0`) | `TERR-LIGHT-113`, `TERR-LIGHT-112` | High |
| D-4 — the shear is a dead band of `+-0.05` around zero, then a two-thirds scale | `TERR-LIGHT-112` | High for the arithmetic; **Unknown** for its purpose |
| An in-game hour is 59.5 s of real time at the shipped default, a whole day 23.8 minutes, a relight period 19.8 s | `SESS-TICK-027` | High for the arithmetic; **Medium** as observed durations |

## Ours by choice

| Decision | Why it is ours |
|---|---|
| The option lives on the view and is not persisted | The engine keeps it in the registry and in the savegame's `GameOptions` section. This tree reads no registry and its byte form is the world, not the session's options, so the option is held where it is consumed and is lost at exit. Nothing decoded is contradicted; a future options store is where it would land. |
| The cache is seeded with the cycle-off light and replaced on the first relight | The engine's cache is written before anything draws. Here a caller that never pushes a clock must keep drawing what it drew, so the seed is the value that build already used and the first push overwrites it. |
| A front-end that pushes no clock never relights | A pure viewer choice with no counterpart in the engine, which has exactly one front-end. It is what keeps the standalone developer viewer and the raster tool at their fixed light. |
| The lighting-clock scrub on `F3` | A diagnostic with no counterpart in the engine at all. It moves what is drawn and never the world, on this front-end's own authored convention that a diagnostic takes a function key. |
| Holding ambient, range and tint at the daytime configuration in every arm | See "Open". The alternative was to invent a gradient, which would put an unmeasured number where the contract claims a decoded one. |
| The lighting clock is a `uint64` of sub-ticks | The world's tick is one, and converting at the seam rather than at the source keeps one number crossing it. |

## Open

| Question | Status |
|---|---|
| The per-band **tint and intensity schedule** — what the dawn, dusk and night arms write to the sky-light RGB and to the two intensity bytes | Undecoded. The arms compute through magic-division sequences the round explicitly did not read, and the row that owns the schedule carries no numbers for it and is graded Medium. This story therefore moves the angle alone and holds the intensity at the daytime configuration. **This is a request to research**, and it is the whole of what stands between this build and a night that looks like night. |
| Whether a mission start issues a **forced** relight, or waits for the unforced condition | Not established. It does not matter here: tick 0 satisfies the unforced condition, so both readings relight on the opening frame. |
| Whether the 20-minute relight period is what a live session observes | The cadence is the image's; the wall-clock figure it maps to depends on the pacer's catch-up branch, which the row grading it Medium says so. |
| Why the shear carries a dead band of `+-0.05` around zero | Unknown, and the row that measures it says the purpose is inferred from nothing in the image. |

## Removed

| Statement | Why |
|---|---|
| A reproduction of the shadow shear function | This tree draws no shadow of any kind, so it would ship with no caller. The arithmetic was re-executed as a cross-check on the angle and the result is recorded in `verification.md`; the asymmetry a shadow pass must reproduce is named in the contract's out-of-scope section. |
| A per-band sky tint interpolated between the known daytime value and an authored night value | It would be an invented number wearing a decoded number's clothes. Held at the daytime value and disclosed instead. |
| An in-game clock line in the debug readout | It needs a number out of a shared sequential namespace this lane was not given, and the story's own acceptance does not require one: a whole-day scrub returns the picture to where it started, which is checkable without a display. |
