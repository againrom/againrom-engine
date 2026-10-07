# Plan — the mission trigger runtime

## Decisions

- **DD-1 — three tiers, and the split is forced.** The leaf grammar goes in `pkg/formats/alm`, the
  compiled program and the runtime in `pkg/sim`, the binder between them in `pkg/mapload`. `pkg/sim`
  may import nothing but the standard library, so it cannot read a map; `pkg/formats/alm` is a leaf
  and cannot name a world. `pkg/mapload` is the one tier that holds both. (FR-1, FR-2)

- **DD-2 — the compiled program is canonical state carried by the byte form.** It is not a cache:
  two worlds differing only in their script diverge, and a world here holds no map to re-derive one
  from. Left outside, it would be a fact about how a world behaves that no digest, no round trip and
  no replay could see — which is what the field-set pin exists to refuse. Byte-form version 9; the
  section goes at the tail so every earlier offset is unmoved. (FR-4, P-1, P-3)

- **DD-3 — the phase is a modulo on the world's own tick.** A tick is a sub-tick; the pass runs on
  phase 6 of sixteen and the reporter on phase 15. Both numbers are the scheduler's own, and both are
  spelled once beside a single cycle length so the two cannot come to disagree. The pass stands after
  the command phase and before the movement loop, which is the one placement its contract fixes.
  (FR-3)

- **DD-4 — loudness is compile-time, not a runtime flag.** Two tables — one per vocabulary — decide
  what this build evaluates, and they are the only place either answer is given, so the report and
  the dispatch cannot disagree. A condition whose arm is unimplemented poisons its register; every
  trigger reading a poisoned register is marked inert when the script is built. A register owned by
  **no** condition is not poisoned: that is an authored mission variable, legitimately zero.
  (FR-8, AC-8, AC-9)

  The rival — evaluate the trigger anyway — is what must not ship: the unwritten register reads zero,
  the authored operand of the shipped shape *is* zero, and the trigger fires on the first pass of the
  mission. Inertness is therefore the safer arm in the direction that matters, and it is also the
  loud one.

- **DD-5 — the references are injected.** A `Target_Unit` names one of three id spaces and one of
  them is a hero ordinal resolved against the live party. A campaign map places no hero, so the band
  cannot be resolved from a map by anybody. The binder takes a resolver from its caller; `pkg/sim`
  sees an entity id and a presence flag and nothing else, which is also what keeps the Medium
  identifier-word reading out of the determinism wall. (FR-2, AC-3)

- **DD-6 — the two arrays are bounds-checked and the overrun is refused.** An authored subscript
  outside the register file reads zero and writes nothing. One reader and one writer, each named, so
  the rule is stated once. (FR-3, P-2)

- **DD-7 — `NewScript` refuses what a binder can only reach by being wrong** — a register subscript
  outside the file, two conditions owning one register, a latch outside the array, a pair reading
  outside the file, an instant slot naming no instant — and does **not** refuse an unimplemented
  opcode. Refusing one would take the arms this build does implement down with it. (P-2)

- **DD-8 — a second constructor rather than a sixth parameter.** `NewWorld` becomes
  `NewScriptedWorld` with no script. The script is the one input most worlds do not have, and every
  existing call site keeps the call it was written with. Both take mode and grid positionally, so
  the objection that put them there does not reappear. (FR-4)

- **DD-9 — the distance metric is one function with one call site per arm.** It is the story's only
  unevidenced number and it is written where a single edit closes it, with the reason and the rival
  named in place. The acceptance witness is driven to distance zero so nothing that ships depends on
  it. (AC-3, AC-7)

- **DD-10 — a developer verb carries the report to a real map.** `almtool script` prints the decode,
  the compile, the binder's findings and the unimplemented tally per opcode. It is how the loudness
  requirement is checked against a shipped map rather than against a fixture. (FR-8)

## Success criteria

- **SC-1** — the alm decode and the identifier words, over synthetic byte streams: the three arrays
  field for field, five refusals, the absent and the present-empty record, and the raw body unmoved.
- **SC-2** — the runtime, in `pkg/sim`: the phase and the ordering, the alphabet and both AND arms,
  the latch under both flags, the register file's two id spaces, the constant preset, the bounded
  subscript, the outcome table, and every implemented check arm.
- **SC-3** — the loudness rule, twice: on a hand-built script and on a compiled one, behaviourally
  (does not fire, does not latch, leaves the outcome undecided) and by report.
- **SC-4** — the byte form: a mid-mission cut and resume that does not re-fire a spent one-shot
  trigger, the program crossing record for record, both worlds stepping to identical digests, and
  the section's refusals.
- **SC-5** — **the campaign's first mission's win chain driven to a win**, with the message arm left
  unimplemented inside it, over a two-pass chain whose second pass is what the ordering contract
  predicts.
- **SC-6** — the binder: the three passes, encounter-order packing, the three unit bands, the miss
  value, the dropped trigger, the latch by map position, and the whole road from map bytes.
- **SC-7** — the gate: build, vet, gofmt, the whole suite, no game assets, the doc budget, the SDD
  audit's FAIL set empty.

## Traceability

| FR | DD | SC |
|---|---|---|
| FR-1 | DD-1 | SC-1, SC-6 |
| FR-2 | DD-1, DD-5, DD-7 | SC-6 |
| FR-3 | DD-3, DD-6 | SC-2 |
| FR-4 | DD-2, DD-8 | SC-4 |
| FR-5 | DD-2 | SC-2, SC-4 |
| FR-6 | DD-3 | SC-2 |
| FR-7 | DD-3 | SC-2, SC-5 |
| FR-8 | DD-4, DD-10 | SC-3 |

| AC | witnessed by | | P | witnessed by |
|---|---|---|---|---|
| AC-1 | SC-1 | | P-1 | SC-4, SC-7 |
| AC-2 | SC-1, SC-6 | | P-2 | SC-2, SC-3 |
| AC-3 | SC-2, SC-6 | | P-3 | SC-4 |
| AC-4 | SC-2 | | | |
| AC-5 | SC-2, SC-4 | | | |
| AC-6 | SC-2 | | | |
| AC-7 | SC-2, SC-5 | | | |
| AC-8 | SC-3 | | | |
| AC-9 | SC-3 | | | |

## Risks

- **The version bump touches every pinned form and digest in the tree.** Three packages pin bytes or
  digests. Each is re-derived rather than re-recorded: the version-9 form is the version-8 form with
  byte 0 raised and a zero run appended, so stripping the run and lowering the byte must reproduce
  the externally-derived number that was already pinned. That check is what makes the new numbers a
  derivation. **The next number is not assumed to be ours** — a sibling lane may need one, and the
  orchestrator resolves it at merge.
- **DD-9 is the story's one unevidenced choice.** It is confined, disclosed and unexercised by any
  witness.
