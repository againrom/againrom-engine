# Provenance — the evidence this story is built on

Research facts come from the `research/` submodule at this story's pin, cited by **claim
id**. Nothing here is taken from an experiment folder: a claim carries its own amendment
and retraction state, an experiment cannot tell you it has been superseded — and this
story is built directly on top of one such supersession.

## Claims relied on

| Claim | Confidence | What this story takes from it |
|---|---|---|
| `MOVE-FORM-036` | **High** | A group move **does** distribute: on the formation arm each member records `memberCellX − centroidCellX` and `memberCellY − centroidCellY`, reads them back as bytes and is issued `target + (dx, dy)`; on the plain arm every member gets the unmodified target. The issue routine **clamps** the destination into the playable rectangle. The offsets are scratch — nothing re-forms a group later. |
| `MOVE-GATE-035` | **High** | One local flag decides both arms. The centroid is the members' **1/256-cell** positions summed and divided by the member count, the high byte of each taken. The spread gate is a **Chebyshev distance in whole cells** to that centroid, compared `JBE`, and one member over the threshold clears the flag **for the whole group**. The same flag forks the per-member loop and gates the rate store. |
| `MOVE-GROUP-030` | **High** for the arithmetic | The group term is the **minimum `Speed` over the members**: a running byte initialised to **250**, a signed 16-bit compare per member, the new minimum taken as that speed's **low byte**. Stored beside the Move order code. Its clear is **retracted** — the only routine that clears it is unreachable. |
| `MOVE-GROUP-037` | **High** for the writer set | The complete instruction-level writer set of the term, with the instrument and both of its blind spots discharged by name: the two gated stores, one unreachable clear, the record's constructor, and a raw whole-record (de)serialization — **so the term is in every save**. A **player order always allocates a new group**, so it starts at zero; a member dying is unlinked and its group pointer zeroed, and **the group keeps the departed member's speed**; the order ending does not clear it. |
| `MOVE-RATE-029` | **High** | Where the term enters: the ground arm takes the group byte **zero-extended when it is nonzero**, else the class speed sign-extended; the non-ground arm takes **the same two sources in the same order**. So a nonzero term replaces the class speed in **every** domain. Already implemented for the class-speed half by 0056. |
| `MOVE-STEP-010` | **High** | A position is a cell byte and a fraction byte per axis, and **`0x80` is centred**. This is what makes the centroid's summands carry a half-cell each, and so what makes the mean round to nearest rather than truncate. |
| `AI-SPREAD-038` | **High** for the value | The spread threshold is a compile-time **2**, on the AI manager, carried by **no shipped byte**. Its own worked example — two units four cells apart pass, five apart fail — is the independent check on our centroid reading. |
| `AI-FORM-037` | **High** for the behaviour | The formation mode is a per-`Player` byte: **0** never in formation, **2** in formation iff the spread test passes, **any other nonzero** in formation unconditionally. Constructor default **2**; **one** `Set formation` node ships in the whole 38-map corpus, and its value is that same 2. |
| `MOVE-ORDER-023` | **partially retracted** | Cited only to record that its headline is gone. What survives — the three routines it read, and that two of them re-issue each actor's own stored cell — is not contradicted by anything here. |
| `MOVE-SEARCH-001`, `MOVE-ALT-019`, `MOVE-ALT-021` | High | Not implemented here: they are why a **blocked** distributed destination needs no new rule. The search already settles for a substitute near an unreachable goal, and that machinery landed in 0037/0045. |

## Ours by choice — and what would settle each

- **The term lives on the entity, not on a group object.** The original hangs a `0x50`-byte
  AI record off a group object owned by a player. We hold one byte per entity. The
  argument that they are indistinguishable *here* is in `analysis.md`; what it rests on is
  that nothing in this tree adds a member to a group, moves one between groups, or writes
  the term outside an order. **The seam** is `moverSpeed` — the one function that decides
  which speed a transit uses — plus the three named writers. An ownership or scripting
  story that can change membership after an order re-homes the byte and changes those four
  sites; nothing else reads it.
- **We reproduce the never-cleared term.** It is very likely a defect: a group keeps a rate
  it can no longer justify, including a dead member's. It is reproduced because it is
  observable behaviour and fidelity is the point. **Lifting it is a switch, not a rewrite:**
  `clearGroupSpeed` exists and is called from the three sites the original writes; adding a
  fourth call at arrival — `restAt` — is the whole of the fix, and the story's own test
  asserts the *current* behaviour so the switch fails loudly rather than silently.
- **The offset read-back is signed.** The rows say "as bytes" and not which extension. We
  take a **signed** byte. Under the default formation mode the spread gate admits only
  offsets in `[-2, +2]`, so the choice is unobservable there; it becomes observable only
  under a formation mode we do not model. Pinned by a test that asserts the narrowing
  explicitly rather than by accident. **What would settle it:** the two read instructions
  at the cited addresses — `MOVSX` or `MOVZX`.
- **The playable rectangle is our own bounds.** The clamp is decoded; the rectangle's four
  bytes are not published. **What would settle it:** the world constructor's writes to that
  field.
- **The formation mode is fixed at the shipped default.** Values `0` and *other nonzero*
  are not reachable from any order this tree can issue. This narrows the input domain and
  changes no arm of the law. **What would settle nothing** — the behaviour is already
  High; what is missing is a `Player` to carry the byte.
- **The centroid divides with floor semantics.** The original divides **unsigned**, over
  coordinates that are unsigned bytes. Our positions are signed and may sit off the map, so
  a sum can be negative, where Go's own division truncates toward zero instead. Floor is
  taken because it agrees with the unsigned divide on the entire domain the law can reach
  and is defined outside it.
- **A plain move order clears the term.** In the original there is no non-group player
  move at all — every player order allocates a group, whose byte starts at zero. Our
  `KindMoveTo` predates groups and is kept; clearing the term is what makes it behave as
  that fresh group of one, in rate terms, for every speed the shipped data carries.

## Open

- Whether a *running* game ever leaves a stale term in play is graded **Medium** in
  `MOVE-GROUP-037` itself — the persistence is a reading of which routines write the byte,
  not an observation of a session. We reproduce the reading, not an observation.
- Whether the group term should survive an owner change, a garrison or a return to the
  map. Those routines are read in the ledger and none of them is reachable here.

## Removed

Nothing was removed. `MOVE-ORDER-023`'s headline was never built on: this tree had no
group order at all before this story, and its units spread only through the substitute
picker — which is exactly what that row said, and remains true of the picker.
