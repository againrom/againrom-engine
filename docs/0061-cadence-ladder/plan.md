# Plan — the cadence ladder

## Decisions

- **DD-1 — the ladder is BUILT from the two period functions, not written out.** `pkg/render/terrain`
  already owns both: the shipped speed table's period and our own rate model's. The ladder is an
  array assembled once from them, so the nine shipped periods have exactly one spelling in the
  package and a change to the table moves the ladder with it. *Prevents:* a second copy of the
  speed table drifting from the first — the failure this whole story is a case of.
  *Covers:* FR-1, FR-2.

- **DD-2 — the extension is geometric; the shipped span is the table.** Below the table's slowest
  setting the ladder halves to the floor and above its fastest it doubles to the ceiling, so the
  two ends are the ones the front-end could already reach and no new bound is invented. A step of
  one across a thousand rates is not a control, which is 0041's own objection and is why the shape
  changes at the join rather than everywhere. *Covers:* FR-1.

- **DD-3 — the opening rung is DERIVED from the opening period.** The front-end asks the ladder
  which rung the map-load period stands on rather than holding a rung number of its own. *Prevents:*
  two constants that have to be kept in step — which is the same defect one level down, and the
  one that produced the bug.
  *Covers:* FR-1, FR-2.

- **DD-4 — the cadence seam carries a PERIOD.** `MapCadence`, `Viewer.SetPeriod` and the world's
  own setter all take a tick length in microseconds; the ladder is consulted once, in the
  front-end's key handler, and the resulting number is adopted verbatim on both sides. A rate
  cannot carry a shipped setting — the game's periods are a truncated whole millisecond and no
  microsecond quotient of a rate reaches them — so a rate on the seam forces each side to compute
  a period that is not the one the map opened holding. *Prevents:* one number becoming two
  quotients. *Covers:* FR-2, FR-5.

- **DD-5 — the readout's setting row is a pure function of the period.** It uses the exact inverse
  of the shipped-period function, which reports **absence** rather than a nearest match. The
  readout cannot reach the key ladder and must not learn to; 0060 withdrew a test for exactly this
  and the rule it established is kept. *Prevents:* a box that names a game setting for a cadence
  the game does not offer. *Covers:* FR-4, FR-6.

- **DD-6 — the front-end holds a RUNG, not a period.** What the keys move is a position; the clamp
  belongs to the position and the ladder converts. Holding a period there would put a second
  arithmetic in the key handler to get back to a position. *Covers:* FR-1, FR-3, FR-5.

## Traceability

| FR | DD | SC |
|---|---|---|
| FR-1 | DD-1, DD-2, DD-3, DD-6 | SC-1, SC-2, SC-3 |
| FR-2 | DD-1, DD-3, DD-4 | SC-1, SC-4 |
| FR-3 | DD-6 | SC-2, SC-3 |
| FR-4 | DD-5 | SC-5 |
| FR-5 | DD-4, DD-6 | SC-3 |
| FR-6 | DD-5 | SC-4, SC-5 |

## Success criteria

- **SC-1 — the ladder's shape, hand-written.** A table of every rung, its period, its rate and
  whether it is one of the game's own, compared against the ladder; strict monotonicity; the two
  ends against the rate bounds; the shipped middle against the speed table's own periods; the map-load
  cadence on a rung. Witnesses AC-1, AC-2, AC-6, P-1.

- **SC-2 — the round trips, exhaustive.** The rung ↔ period round trip over every rung, and the key
  round trip over every start rung against every press count up to four past the ladder's length,
  compared against a closed form rather than a re-run of the code under test. Witnesses AC-3, P-1,
  P-2.

- **SC-3 — the keys, driven.** The whole ladder walked in both directions through the production
  input path, one rung a press, with the period that crosses the seam checked at every step and the
  count of cadence calls checked past both ends; both consumers driven over the same second at
  rungs inside and outside the shipped set. Witnesses AC-1, AC-4, AC-5.

- **SC-4 — the readout still equals the clock.** The box states the clock at every rung and at
  periods on no rung, and a clock re-rated behind the front-end still reaches the box — the
  mutation-checked case 0060 established, re-run. Witnesses AC-6.

- **SC-5 — the setting row.** Every rung's text and three off-ladder periods, hand-written.
  Witnesses AC-7.

- **SC-6 — the gate.** The full local gate green, its FAIL set byte-identical to the baseline taken
  on this branch before the first edit, no file deleted, and the digest-invariance suite green over
  every rung. Witnesses P-3.
