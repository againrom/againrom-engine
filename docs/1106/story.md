# 1106 — Saved cell-trigger overlay

Original mission LOAD must restore the saved six-byte cell triggers over the
fresh ALM bindings before publishing the mission. Both the frontend and the
diagnostic resume route use the same handoff.

## Contract and authority

Base: `d521686f108552918e115e853542cd42142b49e6`. Research pin:
`ba21c9aa9a949023b3d678b22ca29b3a3b0cd95f`.
Reconciled master: `038f4b0bc3523cdc6991ab0edeac147bfec632de`, including
stories1104 and1105. This slice adds no further native form change.
`SAV-CELLLOAD-109` establishes ordered lookup/insert/overwrite over constructed
cells, with construction-only cells retained. `SAV-CELLLOAD-111` identifies
payload bytes `+2c..+31`: operation, power, source x/y, relocation x/y.
`SAV-LOAD-057` establishes construction before restore.

- FR-1: preserve all six bytes, all packed u16 keys, input-order last write,
  construction-only cells, and operation-zero clearing.
- FR-2: validate the whole batch before writing. No declaration-time block or
  map-bounds filter applies. Malformed LOAD preserves the old live session.
- FR-3: LOAD does not attach, cast, or advance a tick. The next ordinary ground
  footprint entry uses the existing consumer, including its domain, footprint,
  repeated-entry and occupancy-attempt rules.
- FR-4: ordinary App SAVE to AGS and fresh LOAD preserve tails, queued native
  casts and deterministic continuation without changing the native byte form.
- DD-1: decode the six bytes in `sav`; use one explicit simulation restore
  writer and one shared game-level handoff before mission publication.
- DD-2: retain operation 26 and unknown operations without inventing consumers.
  Keep the remaining saved cell payload outside this bounded projection.

`UNIT-M10ENTRY-055` and `UNIT-M10LIFE-057` establish the existing entry consumer.
`TRIG-CAST-033` excludes original in-flight temporary actors from original SAV;
native pending-cast continuity is a separate supported path. The original
runtime outcome after restoring a trigger remains Unknown.

## Exclusions

No full original WORLD writer, dynamic population export, cell identity rebind,
terrain baseline restore, result-register overwrite, orders, original in-flight
casts, area-effect restore, or operation-26 relocation consumer. Ten identity
slots and the remaining baseline/count/residue fields described by
`SAV-CELLLOAD-110` and `SAV-CELLLOAD-111` remain separate work. No unpublished
implementation or research candidate is incorporated. The owner authorized SAV
branch publication. The initial candidate is published as
`828464414b108c27044de847035a0b5e0edc6872`; final reconciliation, the single
adversarial review and serialized landing remain seat-owned.

Proof and measured limitations are recorded in `verification.md`.
