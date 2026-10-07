# Story 1063 — raw area-cloud counter

## Player result

Fresh area clouds and mission-script instant 29 use one countdown. A script
duration of one keeps the cloud for one effect tick. A retime to 17 pulses on
the next value, 16, instead of shifting the pulse and expiry one tick early.

## Authority and scope

`MAGIC-AREAPULSE-037` establishes the cloud's raw `effect+0x4c` load,
decrement, positive-multiple-of-16 pulse test and `V0 + 1` observed ticks.
`TRIG-CELLEFFECT-045` establishes instant 29's direct WORD write. The shipped
instant-29 population names spell 3 or 19 and durations 1, 30000 or 60000.

This story changes the cloud countdown and its canonical save meaning. It does
not change spell footprints, damage, movement cost, staged-effect cadence,
audio or drawing.

## As built

- A fresh cloud stores raw `V0`, not `V0 + 1`.
- A positive raw word is decremented before the pulse test. A zero word is
  removed before subtraction, so it cannot wrap to 65535.
- Instant 29 writes the same raw word. Duration one reaches zero after one tick
  and is removed on the following tick.
- Canonical form version 66 records the raw meaning. Lossless migration from
  versions 53 through 65 subtracts one from positive cloud records only. Zero,
  staged records and every other casting-state byte survive unchanged. Version
  65's complete 22-byte structure records also copy unchanged.
- `Remaining` continues to enter `World.Hash` through the canonical form.

`DIV-032` is closed. No new divergence id was needed.

## Proof

`pkg/sim/area_lifetime1063_test.go` covers fresh landing, the actual instant-29
dispatch, pulse phase, the duration-one zero boundary, zero-state round trip,
hash movement, deterministic version-65 migration, byte-exact current round
trip and preservation of staged and structure records. Existing area cadence,
script-cast, save-upgrade and full `pkg/sim` tests also pass.

The final reconciled candidate must run the repository's full Go and no-asset
gates plus the EN/RU release suite. The milestone census is not applicable:
this story changes no supported check, instant or trigger population measured
by that gate.

## Open debt

Version 50 did not carry an area mode. Its existing migration still discloses
the lost area-effect shape and does not guess which legacy record was a cloud.
