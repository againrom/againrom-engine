# Story 1077 — check-opcode 12 item-test alias

## Player result

A custom mission using check opcode 12 now evaluates the same carried-item
condition as opcode 17 instead of poisoning the result register and making its
reader inert.

## Authority and behaviour

`TRIG-ITEMTEST-040` is High for the relevant facts: the original opcode-12 and
opcode-17 bodies are byte-identical apart from a call displacement resolving to
the same finder; both read the named unit's container and item code, write the
same boolean register result and modify no other simulation state.

Againrom therefore exposes opcode 12 as `ScriptCheckItemTestAlias`, admits both
numbers through one support table, and dispatches both through one item-test
case. Both trace as `itemtest`. No save-form field or version changes.

## Touched surfaces

- `pkg/sim`: opcode name, support and shared dispatch; table-driven runtime and
  unsupported-census tests.
- `cmd/missionrun`: the common trace name.
- `cmd/scriptcoverage`: a campaign-absent synthetic record through the existing
  production controlled-check seam.
- divergence ledger: `DIV-243` moves to the closed ledger.

## Proof

- Focused simulation tests run both numeric opcodes through `NewScript` and the
  ordinary phase-6 `StepTraced` dispatch for held, different, other-container
  and missing-field cases.
- The gap test requires only the unrelated unsupported sentinel to remain in
  `Script.Unsupported`; hiding every check cannot satisfy it.
- The script-coverage witness copies a real installed check-17 record, changes
  only its opcode to 12, and requires dispatch and the claim-backed item-test
  result through `NewControlledScriptWorld`.
- Final candidate gates: `gofmt`, the complete Go suite and
  `scripts/check-no-game-assets.sh`.

## Open debt

None for the opcode-12/17 alias. The synthetic witness deliberately fails when
an installed campaign begins authoring opcode 12, so that population must then
move into the ordinary shipped denominator rather than remain hidden here.
