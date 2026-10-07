# Provenance — the mission trigger runtime

## What the contract rests on

| Claim | Carries | Confidence |
|---|---|---|
| `ALM-TRIG-044` | the type-7 payload is three counted arrays, 796/796/184 | Medium (corpus, exclusive over 150 000 models) |
| `ALM-TRIG-045` | the 796-byte node, field by field; parameters stored BY SLOT; the opcode wins over the parameter block | Medium / confirmed at instruction level by `TRIG-REC-011` |
| `ALM-TRIG-046` | the nine parameter type codes and the domain each names | Medium |
| `ALM-TRIG-047` | the 184-byte trigger: three condition-id PAIRS, four action ids, three comparison codes, the `+0xb4` flag; slots hold node **ids**, not indices | Medium for the layout; its two Unknowns closed by `TRIG-CMP-006`/`TRIG-FIRE-007` |
| `ALM-UNIT-048` | the placed-unit record's `+0x40` is the unit id and `+0x42` the group id | **Medium** — see the risk below |
| `TRIG-EVAL-001` | one pass per full tick, on a named phase of sixteen sub-ticks; every check, then every pattern; **before** any actor acts | High (two independent readings) |
| `TRIG-STORE-002` | 100 signed registers and 1000 latch bytes; a check owns one register and writes it once per pass; neither array is bounds-checked | High |
| `TRIG-COND-003` | the 22-arm condition table; arm 18 writes no register and counts a loss; arm 19 reads another register; the `0x10002` build-time constant is the whole notion of a mission variable | High for the table and the three structural oddities |
| `TRIG-ACT-004` | the 34-arm action table; arms 3, 4, 5 and 8; opcodes at or above `0x10002` never dispatched | High for the four named arms |
| `TRIG-CMP-006` | six codes `== != > < >= <=` in that order, pairs ANDed with short-circuit, above 5 permanently false | High |
| `TRIG-FIRE-007` | `+0xb4` is the fire-once flag; the latch is indexed by the trigger's position in the map's array; clear-evaluate-set | High |
| `TRIG-END-009` | win and lose are counters an authored arm increments; the reporter tests **lose first**, both for **exactly one**, and latches | High for the counters, the tests and the latch |
| `TRIG-BIND-010` | three passes; a trigger whose first pair's left id is zero is dropped whole; an id naming no built node resolves to subscript 0; only both-set pairs become triples | High for the pass structure and the drop rule |
| `TRIG-REC-011` | the compiled records; an instant's ten plain ints in **encounter order** beside its resolved references; `Target_Unit` is **three id bands** | High for the read order and the three bands |
| `TRIG-SAVE-008` | the registers and latches serialize; a load re-compiles then overwrites the volatile half; omitting the latches replays every one-shot trigger | High (its "in every save" clause is retracted and is not relied on) |
| `MISSION-WIN-003` | exactly one winning action per campaign map, none on any loose map | High |
| `MISSION-VIP-004` | every check node is compiled and evaluated whether a trigger names it or not; a "protect this unit" objective is a trigger with **no action** | Medium |
| `MISSION-SLOT-008` | a mission variable and a check result share one array, and the shipped corpus already collides | Medium |
| `MISSION-M10-009` | the campaign's first mission end to end: the escort chain, its coordinates, its variable, its win | High |
| `MISSION-TYP-010` | the campaign uses 17 of 22 check arms and 26 of 34 instant arms | Medium |
| `MISSION-DROP-002` | the map's whole contribution to the start is one packed cell, from a build-time action | High |
| `MISSION-START-001` | a campaign map places nobody for the player; the party already exists | High |
| `ALM-REQ-056` | an absent record is the loader's skipped arm, not a malformed map | High |

## The risk this story runs, named

**`ALM-UNIT-048` is Medium and it reaches hashed state.** Which of the placed-unit record's two
identifier words a `Target_Unit` names decides which entity a condition measures, and a condition's
answer is a register the digest covers. The claim rests on corpus agreement — 169 of 169 group
references resolving against `+0x42`, and the distinctness counts that refute the rival — rather than
on an instruction that dereferences either field. Its published rival, the earlier reading, is
refuted by that same corpus.

What limits the exposure: the fact is consumed by the **binder**, one tier above the determinism
wall, and `pkg/sim` never sees it. A compiled check names an entity; who that entity is, is the
binder's answer and carries the binder's confidence. What would raise it is an instruction that
dereferences `+0x40` on the resolver's own path.

## Ours by choice

- **The distance metric.** `TRIG-COND-003`'s arms 3, 6, 7 are published as computing a distance and
  the metric is not named by any claim. Chebyshev is authored here on the engine's own habit —
  `HERO-TARGET-024`, `AI-RADIUS-014`, `AI-ROAM-025`, `AI-SPREAD-038` and `MOVE-ALT-019` all measure
  whole-cell Chebyshev — and it is a one-function seam. **Nothing that ships rests on it:** the
  acceptance witness drives every distance to zero, where every candidate agrees. This is a
  **research request**, not a closed question.
- **Bounds-checking the two arrays**, where the original has none and overruns.
- **One outcome** rather than one per player: there is no player object in this tree.
- **A node that cannot resolve a reference keeps its register**, where the original drops the node.
  `MISSION-SLOT-008` grades the slot law Medium on exactly this question and `TRIG-BIND-010` measures
  0 of 1304 shipped nodes failing, so the readings agree on everything that ships.
- **The compiled program is canonical state** carried by the byte form, where the original re-derives
  it. A world here holds no map.
- **A downed unit is dead to a script.** Our third life state has no counterpart in the original.
- **Only ordinal 1 of the hero band resolves**, this tree having one player.
- **The byte-form version** rises to 9 and the previous form is refused, as every version before it
  has been.

## Open

- The distance metric (above): `L12348`, `L12344` and `R2056` are the arms; which
  helper they call and what it computes was not read.
- Every check arm but eight and every instant arm but four. Each is a table entry, and the compiled
  script names every one a given map needs — so what a map costs is measurable rather than guessed.
- The group-command sub-dispatch (`TRIG-GROUP-005`), which belongs with the AI; ownership transfer
  and inventory transfer, which belong with the world; message-raising (`MISSION-TEXT-005`), which is
  presentation.
- `TRIG-INI-012`'s mission `.ini` overlay: no such file ships and no part of the trigger machinery
  reads it, so nothing here consults one.

## Removed

Nothing was removed. `TRIG-SAVE-008`'s retracted "in every save" clause was never relied on: what is
used is the field order and the consumer consequence, both of which survive the retraction.
