# Provenance — why the player's slot is 1, and what is ours

Claim ids are read from the `research/` submodule at its pin. Confidence is the ledger's own.

## Backing

| Spec anchor | Source | Confidence | What it carries |
|---|---|---|---|
| FR-1 | `ALM-OWN-039` | High | An owner word is a **1-based physical slot in the type-5 array**, not the record's own id word — the rival reading fails on 713 type-4 and 2238 type-6 shipped records. This is what makes "slot 1" a position rather than a name. |
| FR-1 | `TRIG-EMPTY-018` | High (for its figures) | Names the slot outright: *"roster slot 1, the player's"*, while counting check nodes over both roots. |
| FR-1 | `MISSION-START-001` | High (positive half) | Measures what **roster slot 1** owns over the 28 campaign maps of both roots — 19 of 2333 type-6 records — so slot 1 is the key the player's placements are counted by. |
| FR-1 | `PARTY-PERSIST-014` | Medium | `R0423` builds one `Player` per type-5 record **except for roster slot 1**, which reuses the surviving player; the campaign start passes the argument that makes that arm live. Medium because whether the shipped campaign's mission edge always avoids the teardown was not established — which does not touch the slot number. |
| FR-1 | `ALM-GRP-020`, `ALM-GRP-041` | High | The type-5 record is the player/group roster; the first observed name is `Self`, and the 32-byte name field is a read boundary. |
| FR-2 | `AI-DIPLO-005` | High | The matrix index is `type-5 slot + 1`, the store is `matrix[idx][k+1]`, and **column 0 is never written**. An entity outside the matrix is unreachable in both directions. |
| FR-2 | `PARTY-OWN-001` | High | The `Player` carries two identities and they are different numbers: `+0x04` is `slot+1`, the value the wire and the document array use; `+0x08` is the map's own type-5 id word. The matrix indexes by the first. |
| FR-6 | `AI-STAND-076` | High (behaviour) / Medium (name) | Group order 3 is **Stand Ground** — its scorer refuses anything past reach and its members never take a step — and the load walk gives it to the groups of type-5 slot 0, the player's own `Self`. |
| FR-2, FR-10 | this tree's own `pkg/sim/engage.go` | — | `const selfSlot = 1`, carrying the derivation and the corpus note that record 0 is named `Self` on all 38 EN maps. The story's central fact was already written down in one of the two files that disagree; the rows above confirm which of them is right. |
| FR-5, AC-9 | `AI-DIPLO-005` | High | The relation is directional and asymmetric and shipped content uses it: 102 of 866 ordered off-diagonal pairs disagree with their mirror across 19 of 38 maps. A map faction fighting another is authored, not a fault. |

## Ours by choice

| Decision | Why it is ours, and what would change it |
|---|---|
| The slot number is written **once** and shared by the placement and by the stance rule. | Nothing decoded says the two must be one symbol; two literals would agree today. It is engineering against the drift the ledger itself warns about, and a story that lets a client be seated at another slot turns the constant into a value without touching either consumer. |
| A party member's **group word stays 0**. | Nothing read here says what group word the campaign party's units carry. Zero is what the field already held; the stance rule reads the owner slot and not the group word, so the choice is invisible to this build's behaviour and is disclosed rather than decoded. |
| The front end is told the local participant holds slot 1. | Nothing decoded says a front end learns its slot from the party placement rather than from a session join — the original's answer comes from the join, which this tree does not perform. It is taken because the alternative is not neutral: the numeral comparison is satisfied today only by two zeros, so leaving it would move behaviour this story is meant to preserve. A session-join story replaces the source without touching the consumer. |
| The roster readout is a **developer tool mode** that re-derives the effective row rather than reading the loader's. | It witnesses a map's own statement; where it lives is ours, and the import DAG makes the alternative impossible — `cmd/almtool -> pkg/sim` is a forbidden edge and the loader's answer is a `sim` type. |
| The readout prints the **effective** row — low byte, diagonal forced to 2 — beside the raw words. | The engine's narrowing is `AI-DIPLO-005`'s; showing both is so a reader can see the file and the consequence without holding the rule in his head. |

## Open — deliberately assigned no meaning

| Undecoded | Held at |
|---|---|
| Whether the campaign party's units carry a group word of their own, and what it is. | Group 0, disclosed at FR-3. Bounded by measurement rather than by argument: over both roots, **28 `GiveGroup` nodes and none names group 0**, so the one arm that could re-own a group is not aimed at the party by anything that ships. |
| **What the original does with a group of a HUMAN PARTICIPANT under order 3.** `AI-STAND-076` reads the arm as branching on exactly that — an AI owner takes one path and *"a human participant's unit"* takes `R0015`, which nobody has read. This is the story's largest open item and it is the one that decides FR-6a. | Nothing: this build applies the arm it has, unchanged, to every slot. What that costs is measured at FR-6a rather than argued away. |
| Which mission-script arm decides mission 10, and what fells `u21`. | Nothing here authors either; the outcome, its tick and the acquiring entity are recorded as measured, before and after. |

## Removed

| Dropped | Why |
|---|---|
| Any requirement that mission 10 be won. | It is not won before the change and it is not won after; measured on both roots. Making a predicate fit one map is how the other 27 break. |
| A claim that `0086` disclosed the party's slot. | Checked: `docs/0086-ai-engages/spec.md` contains no mention of a party. The boundary was specified; the population standing in it was not. |
