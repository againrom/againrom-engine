# 0129 — give all: tasks

## T1 — the second reference and the arm

Touch `pkg/sim/script.go` and add `pkg/sim/giveall_test.go`. Implement FR-1, FR-3, FR-4, FR-5, FR-6,
P-1, P-2, P-3. `internal/archtest` scans this package's non-test source for `os`, `time`,
`math/rand`, float identifiers and float literals — use none.

- `ScriptInstant` gains `Unit2 EntityID` and `HasUnit2 bool` (DD-1), doc'd in the shape of
  `ScriptCheck`'s own pair. Do **not** touch `Args`.
- `ScriptInstantGiveAll int32 = 28` in the opcode const block, with a doc line saying what it moves
  and that it arms no trigger.
- `scriptInstantSupported` gains it; `runInstant` gains its case, in the same commit (DD-6).
- The arm: `indexOfEntity` both references; append the giver's whole slice to the receiver's (DD-2),
  then clear the giver's to nil (DD-4). The container is `w.carried`; the twelve worn places are
  `w.equipment` and are not read here at all (FR-5). Two references resolving to one entity is a
  refusal, not a loop (DD-3).

Do **not** change `formatVersion` or the byte form here — T2 owns both.

Tests assert **exact lists**, never a count and never non-emptiness. One each for AC-1, AC-2, AC-4;
one per FR-6 case for AC-3, each comparing the whole `MarshalBinary` output across the step; AC-6
reads `Script.Unsupported` and `InertTriggers`. P-2: a refusal leaves the generator state equal.

## T2 — byte form version 38

Touch `pkg/sim/scriptbinary.go` and `pkg/sim/corpseloot_test.go`; add to `pkg/sim/scriptform_test.go`.
Implement FR-7. T1 landed the field unserialized on purpose (DD-7); this task closes that.

- `scriptInstantLen` 59 → **64**. The value at `+59` as a little-endian `uint32`, the flag at `+63`.
  Every existing offset is unmoved (DD-5).
- Encode and decode both; the flag goes through `scriptFlag`, which refuses anything but 0 and 1.
- Amend this file's own offset table and the paragraph describing an instant record.
- `formatVersion` 36 → **38** in `pkg/sim/binary.go`. 37 is allocated to a parallel story; take 38.
  Extend that constant's version narrative with a sentence naming what moved and why a version-37
  buffer is not this one.
- `pkg/sim/corpseloot_test.go` carries a tripwire pinned at 36. **Re-pin it to 38** and add a sentence
  to its doc block saying this story moved it and what widened. Never delete or weaken it.
- Put the number 38 in **no test's name**. A name that spells the live version has gone stale here
  three times; the name is what must stop carrying it.

AC-7: a world whose script carries an instant with a second reference round-trips equal; the form's
first byte is `formatVersion`; a buffer declaring 37 is refused; two worlds differing only in that
field hash differently.

## T3 — the binder fills it

Touch `pkg/mapload/script.go` and its test. Implement FR-2.

`bindParams` already resolves a node's first two unit parameters into `bound.unit`/`unit2` and their
flags, for every node it binds. `CompileScriptFrom`'s **action** pass drops the second; carry it onto
the `sim.ScriptInstant` literal exactly as the condition pass already carries it onto
`sim.ScriptCheck` (DD-1). Nothing about resolution, reporting or the unresolved list changes.

Amend `bindParams`'s doc block where it explains the two-reference pair, so it no longer reads as
though only a check has one.

AC-5, synthetic ALM script fixtures in this package's own style: a node naming two unit parameters
compiles to an instant with both references set, in parameter order; a node naming one leaves the
second absent; a node whose second parameter falls outside every resolvable band leaves the second
absent and appears in the report's unresolved list.
