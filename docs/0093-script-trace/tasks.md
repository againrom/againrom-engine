# 0093 — tasks

Kinds: **impl** (one commit, trailered), **test** (folded into the impl commit it witnesses).

## T1 — the record and the observed step

**Kind:** impl. **Covers:** FR-1, FR-2, FR-3, FR-4, FR-5, FR-6; DD-1, DD-2, DD-3, DD-4, DD-5.

**Files:** `pkg/sim/scripttrace.go` (new), `pkg/sim/script.go`, `pkg/sim/step.go`,
`pkg/sim/scripttrace_test.go` (new).

**Fences:** no field on `World` or `Entity`; no change to the byte form or its version; no change
to `Step`'s signature; `triggerHolds`, `runInstant` and `scriptReport` are not edited.

**Done when:** a scripted world can be advanced observed; the tests witness a firing with its
compared values and its instant support marks, a self-counted loss with no firing, all three
silence reasons and both by-design non-silences, an empty record off a non-script tick, register
ownership in both directions, the two-trigger case where an instant moves a register between them,
and byte-and-digest equality between an observed and an unobserved run at every tick of a run
crossing several passes and a report.

## T2 — the readout

**Kind:** impl. **Covers:** FR-7; DD-6, DD-7, DD-8.

**Files:** `cmd/missionrun/trace.go` (new), `cmd/missionrun/main.go`,
`cmd/missionrun/main_test.go`.

**Fences:** off by default; with it off the tool advances through the ordinary entry point; no
opcode-support table in this package; nothing here reads an asset.

**Done when:** the drive takes the flag, every advance in the tool passes through one seam, and
the readout renders a firing, a self-counted loss and a silence from a synthetic world with no
install present — each naming the register's owning check, that check's arm and parameters, and
the unit by the number the map's script uses.

## T3 — the walk stops honestly

**Kind:** impl. **Covers:** FR-8; DD-9.

**Files:** `cmd/missionrun/main.go`.

**Fences:** the radius test itself does not move; only what is returned and printed when the
mission is decided mid-walk.

**Done when:** a walk ended by the decision reports the radius it actually met, and the drive's
line for it says the world decided rather than that the point was reached.

## Traceability

| Task | FR | DD |
|---|---|---|
| T1 | FR-1, FR-2, FR-3, FR-4, FR-5, FR-6 | DD-1, DD-2, DD-3, DD-4, DD-5 |
| T2 | FR-7 | DD-6, DD-7, DD-8 |
| T3 | FR-8 | DD-9 |

**FR-9** landed as hotfix `65dfdf7`, not as a task of this story; folded into the contract by 0130.
