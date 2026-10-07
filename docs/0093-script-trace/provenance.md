# 0093 — provenance

This story builds an **observer**. It asserts no new fact about the original engine: every
statement it makes about the mission script is a statement about the runtime already in
`pkg/sim/script.go`, which carries its own backing from `0063`. The registers below record which
of this story's contract sentences are downstream of somebody else's evidence and which are ours.

## Backing — what rests on a research claim

| Spec anchor | Source | Confidence | What it backs |
|---|---|---|---|
| FR-1, FR-2 | `TRIG-EVAL-001` | — as landed in 0063 | That a pass is *every check, then every trigger*, once per full tick, is the shape the trace is organised around. |
| FR-2, AC-3 | `TRIG-COND-003`, `TRIG-CMP-006` | — as landed in 0063 | A trigger's condition is up to three register pairs and one of six comparison codes; the trace records exactly three pair slots and names a seventh code as permanently false. |
| FR-2 | `TRIG-FIRE-007` | — as landed in 0063 | The latch is the trigger's position in the map's own array, which is why a firing carries both that number and the compiled subscript. |
| FR-3, AC-4 | `MISSION-VIP-004` | **Medium** | That a "protect this unit" objective is a **check** whose side effect is the loss counter, belonging to no trigger. This is the claim the whole third event kind exists for, and the claim the tenth mission's answer lands on. |

The confidences are not re-graded here. Each was weighed when the runtime landed; this story reads
the runtime, not the corpus, and a second grading from a lane that ran no experiment would be a
confidence invented by restatement.

## Ours by choice — what the spec fixes that no source asserts

| Decision | Why it is ours |
|---|---|
| That the trace is a **return value** rather than a stored hook or a callback. Nothing in the original has an observer at all. | Engineering. Changeable later without contradicting anyone. |
| The three **silence reasons** and the choice to record only those three. The two by-design silent arms (constant, VIP) are not "a measurement not taken". | Ours; the runtime distinguishes four no-write paths and we name the three that mean something went unmeasured. |
| Every **name** the readout prints — `groupcount`, `within`, `LOSE`, `>=` and the rest. These are our own labels for opcodes whose numbers are the evidence; an opcode outside our tables prints as its number rather than as a guess. | Ours; presentation only. No name is a claim about an arm this build does not have. |
| That a firing records its pairs **for every firing**, not only for one that wins or loses. | Ours; uniform is cheaper to trust than conditional. |
| The readout printing a standing silence **once** rather than per pass. | Ours; presentation. |

## Open — deliberately assigned no meaning

- **Check op 14 and instant ops 2, 6, 20, 28.** `10.alm` authors all five and this build evaluates
  none. The trace names them by number and says this build does not run them. It assigns them no
  behaviour, and a trigger reading op 14's register stays inert.
- **Whether a compiled check list is the whole node list.** `MISSION-VIP-004` records that this is
  undiscriminated. The trace reports the subscripts of the list it is given and makes no claim
  that they are the map's own node numbers; the *latch* is the number that is.

## Removed

Nothing. No statement was dropped from this contract during the pass.
