# Provenance — the owner a script hands over

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 — the type-6 record carries an owner at file `+0x14`, and it is a **1-based physical slot in the type-5 array** rather than that record's own id word | `ALM-OWN-039` | High. The field's identity and its one consumer are named instructions at two call sites; the slot-vs-id question is decided against the id reading on 713 + 2238 records |
| FR-1 — the field's place and width inside the 70-byte record, between the definition id at `+0x10` and the bounded index at `+0x18` | `ALM-UNIT-040` | High. The loader's 27 sequential reads sum to exactly 70 at format version 990 |
| FR-2 — the value space starts at 1, so zero names no roster slot | `ALM-OWN-039` | High, and the same reading that fixes the field |
| FR-3 — parameter type 2 is `Target_Group` and type 3 is `Target_Player`; a `Target_Group` resolves against the placed record's group word | `ALM-TRIG-046` | Medium — corpus resolution (169/169) rather than an instruction that dereferences either field |
| FR-3 — a `Target_Unit` names one of three id bands, only the lowest of which a map carries | `ALM-TRIG-046` | Medium, and already the binder's shipped behaviour for check nodes |
| FR-4 — instant opcode 22 hands a **group** to a player and opcode 19 hands a **unit** to a player | `MISSION-M10-009` | High. One map read end to end, every node and trigger rendered against the install's own instant catalogue, every parameter resolving |
| FR-4 — ownership at runtime is one field on the actor naming a `Player`, and every "is this mine" consumer compares it | `PARTY-OWN-001` | High. One store inside the serializer, three reads in unrelated modules |
| FR-4 — a player's roster is `Player -> group collection -> group -> actor list`, and membership is containment | `PARTY-ROSTER-002` | High for the three-level structure |
| FR-4 — a runtime group is formed at spawn by equality of the placed record's group word, under an owner | `AI-GROUP-009` | High for the allocation, the constructor defaults and the match instruction; Medium for the corpus figures |
| FR-6 — the instant vocabulary is a 34-arm table, and an opcode the table does not implement reaches no arm | `TRIG-ACT-004` | High for the table, its extent and its default arm |
| FR-4 — that a script's player parameter and a placement's owner field are one id space | Ours, measured: 170/170 (en) and 168/168 (ru) `Target_Player` parameters lie in `[1, type-5 count]`, and the rival roster-id reading fails 26 times on each root | — |

## Ours by choice

| What the spec fixes | Why it is ours |
|---|---|
| An owner is a **number**, not a player object. | This tree has no `Player`, no roster object and no group object. The number is the map's own word and the arms assign it; a later story that builds a roster resolves it. |
| **Zero is no owner**, and it is the value every entity not built from a placement carries — the hero included. | The evidence makes the space 1-based, which leaves zero free; using it costs no presence flag on the entity. It is the opposite trade from the group identifier, whose zero is a real group, and the two are opposite because the two id spaces are. |
| Membership for arm 22 is the **group identifier alone**. | The original keys a runtime group on the owning player as well. Nothing here carries a group object to move, so the identifier is the only expressible reading. |
| The arms include entities that are **dead or downed**. | Membership and identity outlive death everywhere else in this runtime, and an arm that skipped the dead would make a hand-over depend on when it fired. |
| An arm whose reference is **absent or unresolved writes nothing**, rather than writing zero. | Writing zero would be indistinguishable from handing a unit to nobody, and this runtime already separates "measured nothing" from "measured a zero" at every other reference site. |
| The owner is carried at the record's own **32-bit width**, though no shipped map authors a value above 9. | Widening later would move a byte-form offset; narrowing now would invent a limit the file does not state. The customisation limit is the roster itself — sixteen editor player slots (`ALM-GRP-041`). |
| Implementing an instant arm **arms no trigger**, and the spec says so rather than leaving it to be inferred. | It follows from this runtime's own inertness rule, which derives from unimplemented checks; it is written down because the story was opened on the belief that it would arm one. |

## Open

- **The instruction-level body of arms 19 and 22.** Their effect is published as a whole-map catalogue
  rendering, not as a read of either arm. Unknown: whether the original re-keys or moves group
  membership, what it does when two owners share a group identifier, and what it writes besides the
  actor's owner pointer. The published instant-table row names four arms exactly and grades five
  Medium; neither of these two is in either set.
- **Which roster slot the human participant is.** A participant's `Player` is created by the
  session-join path rather than by the map, and the discriminator every ownership consumer reads is a
  field of that object (`UNIT-OWNER-009`), not a slot number. Nothing in this story reads an owner,
  and that is why.
- **The route-budget arm's owner term.** The decoded search budget takes its flat form exactly when a
  human participant owns the mover (`MOVE-TERM-003`, `UNIT-OWNER-009`); this tree takes that form
  unconditionally on the premise that no entity carries an owner. This story falsifies the premise's
  antecedent without settling its consequent, since the term is human-ownership and not a slot.
- **The type-5 roster's per-slot relation words** and the type-4 structure owner field. Neither is
  decoded here.

## Removed

- **"Instant arm 22 gates trigger position 3 on mission 10."** The story's opening premise. Dropped
  because the map refutes it: position 3 reads no register position 2 writes, its one condition is an
  arm this build evaluates, and its trigger is live and evaluated today. Recorded rather than deleted
  because the contract below deliberately claims **no** progress on the win chain, and a reader who
  expected otherwise is owed the reason.
- **A count of triggers this story arms.** It is zero, necessarily, and a success criterion asserting
  a non-zero one would have been unsatisfiable.
