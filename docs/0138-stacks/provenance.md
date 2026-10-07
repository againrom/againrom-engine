# 0138-stacks — provenance

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| Terms "Stack", FR-1 — an element is one object carrying a code and a count | `ITEM-STACK-003` (`item+0x42`, u16, default 1) | High |
| FR-2, FR-3 — two identical items become one element, and merging is a real operation on the container class rather than only on a screen | `ITEM-STACK-003` (the merge at `L04602`, inside the container class's own range) | High |
| Terms "Identical", FR-2 — what makes two items the same thing | `SHOP-DUP-028` (`R0928`: equal kind id, then the stackable pair test) | High |
| Terms "Identical" — every item this build models is stackable | `ITEM-STACK-003` (`vt+0x50` = `byte +0x44 == 3` **or** the effect list at `+0x20` is empty; `Armor`, `Shield` and `Weapon` inherit that body verbatim) | High for the predicate; the reduction to *equal code* is ours — see *Ours by choice* |
| FR-6 — a take moves the whole sack and merges it in | `ITEM-PICK-016` (one execution path, no partial outcome), `ITEM-CONT-004` (no capacity, no limit, nothing refuses) | High / Medium for "nothing refuses" |
| FR-7 — a container has no slot count and no capacity, so merging can never be forced by one | `ITEM-CONT-004` | High for the class shape |
| Out of scope, "weight" — the load is `Σ weight × count` and would need a per-unit weight | `ITEM-STACK-003` (`item+0x4a`, s16, default 1) | High |

## Ours by choice

| Statement | Why it is ours |
|---|---|
| **FR-2 — every add merges.** `ITEM-STACK-003` names the container class's merge sequence but not its callers, and `SHOP-DUP-028` shows the original's shop generator appending without dedup, so *when* the original merges a unit's container is not established. Merging on every add is **AUTHORED** — it is what the owner asked to see and it is the only rule that makes "one slot, ×3" hold however the items arrived. |
| **Terms "Identical" — two elements are identical when their codes are equal.** The decoded predicate is *equal kind id, and both stackable*; this build's item is a bare sixteen-bit code with no effect list and no `+0x44` byte, so the "effect list is empty" arm holds for every item and the predicate reduces to equality of the code. When enchantment is modelled the predicate grows a second term; until then a stackable test would have exactly one answer and is not written. |
| **FR-4, FR-10 — the byte form does not change and the version does not move.** A count is a grouping of codes the carry section already carries, not new state: the encoder expands an element into `Count` copies of its code and the decoder folds equal codes back. Under FR-2 that is a bijection on every world this package can build. |
| **Terms "Stack" — the count is 32 bits wide.** `ITEM-STACK-003` reads a `u16` at `item+0x42`. Here the count groups a list this build already writes with a 32-bit length and constructs at any length, so a 16-bit count could silently lose units of a container the previous build both writes and reads. The width is ours; the field it stands for is not. |
| **FR-9 — equipping takes one unit.** The claim has a split by a chosen quantity; no screen in this build asks for a quantity, and equipping the whole of a stack of three is not a reading anything supports. |
| FR-11 — the count is drawn in a cell's own corner, at two or more, in the window's own font. Every dimension of that window is already authored (0112 plan DD-3, DD-4). |

## Open

| Undecoded | What this story does about it |
|---|---|
| `+0x44`'s value space beyond the literal `3` (`ITEM-STACK-003`'s own **Unknown**: two neighbouring string arrays at `L13180` and `L13181` were never traced to a consumer indexed by it) | No meaning is built for it. No item class is treated as stackable-by-class and none as unstackable. |
| The effect list at `item+0x20`, and enchantment generally | Not modelled anywhere in this build. Named in *Ours by choice* as the reason the identity test reduces to equality of the code. |
| The per-unit weight at `item+0x4a` and a container's running load | Nothing in this build carries a weight or a load. `Σ weight × count` is not implemented and not stubbed. |
| The `+0x08` flag word a merge ORs together | This build's item has no flag word to OR. |

## Removed

| Dropped | Why |
|---|---|
| A `Stackable` predicate on the item code | It would answer *yes* for all 65 536 codes (see *Ours by choice*), which is a decision dressed as a lookup. |
| A stack limit | `ITEM-CONT-004` reads the whole container class and finds no routine comparing anything against a limit. |
| Stacking inside a sack | A `Sack` carries a flat code list. A drop expands and a take re-merges, so no visible behaviour rests on it. |
</content>
</invoke>
