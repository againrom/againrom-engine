# 0136 — armour is worn, and it counts: provenance

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 (the two columns, and which factor slot scales each) | `ITEM-ARMFILL-032` | High |
| FR-2 (defence rounds, absorption truncates) | `ITEM-ARMFILL-032` | High |
| FR-3 (both stores are 16 bits) | `ITEM-ARMFILL-032` | High |
| FR-5 (a Slot of 0 is refused; a Slot above 12 is discarded) | `ITEM-HUMEQ-030` (`"Illegal armor"` on `armor+0x50 == 0`), `ITEM-ARMSLOT-031` (`CMP ECX,0xc` / `JLE`, `"Invalid armor part "`) | High |
| FR-6 (the worn set adds defence and absorption and NOTHING else) | `ITEM-ARMFOLD-033`; block layout `HERO-FOLD-033` | High |
| FR-6a (every protection and every damage-kind byte receives zero from an armour) | `ITEM-ARMFOLD-033` — the fill writes block members `+0x00` and `+0x02` and "nothing writes the other twelve" | High |
| FR-6b (the two words are defence then absorption, in that order) | `HERO-ARMOUR-018` (`R0845` adds `src+0`→`dst+0`, `src+2`→`dst+2`, with `dst = actor+0xbe`) | High |
| FR-7 (the destination is the row's own Slot column, not the cell it was named in) | `ITEM-ARMSLOT-031` — `L04716 PUSH 0x4`, `L04717`, then `MOV [EDX + ECX*0x4 + 0x198],EAX` | High |
| FR-7a (no per-slot difference in what a piece contributes) | `ITEM-ARMFOLD-033` — "the slot byte indexes the array and is read nowhere else, and neither block add sits under a branch on it" | High |
| FR-8 (nothing refuses a piece on account of who is wearing it) | `ITEM-HUMEQ-030` (two byte-for-byte identical calls, no per-class restriction), `ITEM-SUIT-035` (`sutableFor` drives two display flags) | High for the equip path / Medium for reading the mask's two bits as the two player classes |
| FR-9 (equipment is stored, and a recompute is the caller's own act) | `ITEM-ARMFOLD-033` — `vt+0x54` reads neither the slot array nor either block, and its result is discarded | High |
| The corpus this spec assumes exists: `Slot ∈ {4,5,6,7,8,9,10,12}`, no row using 1, 2, 3 or 11 | `ITEM-ARMSLOT-031` | Medium — a census over both roots and nothing stronger |

## Ours by choice

| Statement | Why it is ours |
|---|---|
| Field B of an armour's item code is the row's Slot; B of 1 is the weapon class and B of 2 the shield class, so a code's B decides which of the three resolvers may read it (DD-2) | The item code is this tree's own composition (`0110` Terms "Item code", `0128` `ResolveArmor`). Nothing decoded says a code carries a class at all. It is exact on shipped data only because no `Armors` row states Slot 1, 2 or 3 — a Medium census. |
| A code is resolved to an armour by its three INDICES, never by recomposing a name and re-entering the name resolver (DD-1) | `WeaponFromCode`'s shape cannot be reused: `impliedShapePrefix` puts `Soft ` back on a leather piece's residue, and the shipped row names already carry that word. Measured — see `verification.md`. |
| The u16 store width is modelled by a conversion rather than by a refusal | `damageByte`'s own precedent in `weapon.go`. No shipped combination reaches it — measured. |
| The equip gate's refusal is silent (no command, nothing shown) | `0124` FR-9's existing wording, unchanged by this story. |
| Shields are out (see the spec's Out of Scope) | `ITEM-HUMEQ-030` gives `Shield::Equip` two arms an armour has none of. |
| A placed person's STARTING armour is not folded (see Out of Scope) | Disclosed divergence, not a neutral boundary: `ITEM-HUMEQ-030` establishes the spawn path reaches the same `Equip`. |

## Open

| Left with no meaning | Why |
|---|---|
| The other twelve members of a worn piece's block | `ITEM-ARMFOLD-033` grades **Unknown** whether a later path writes them; the effect and magic-item routines are unread. This story writes the two and leaves twelve at zero, which is what the read routines do. |
| Magic capacity (`+0x48`) and the weight the fill also scales (column 3) | `ITEM-ARMFILL-032` states both; no consumer in this tree reads either, and `MagCap` is only Medium for its name. |
| Whether `sutableFor`'s two bits are the two player classes | `ITEM-SUIT-035`, Medium. Nothing here reads the column, so the grade costs this story nothing. |

## Removed

| Dropped | Why |
|---|---|
| `FR-10a`, which disclosed "a piece worn past the first slot is counted and is not pictured" as a limit this story would leave standing | Reversed mid-story on the owner's own instruction that the doll must show the piece's sprite. A semantic reversal, so the id is retired rather than reworded: FR-10b, FR-10c and FR-10d are the new contract and FR-10a is never reused. What made the old reading defensible was a stale comment in the composition itself, and what made it indefensible is that `0134` landed a generated character already wearing three pieces. |
| A class-based equip refusal | Considered because the column exists and reads like a restriction. `ITEM-SUIT-035` and `ITEM-HUMEQ-030` together rule it out: two display flags, and an equip path with no such arm. |
| `HERO-ARMOUR-018`'s corpus figure "`#.absorbtion` is 0 on 22 of the 30 armour rows" | Not used as backing. Re-measured here as **25 of 30**, identical on both roots. The row's conclusion — why an armoured character can read `БРОНЯ 0` — is unaffected and is what this story leans on. Correcting a claim is research's act, not this repository's. |
