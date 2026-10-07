# 0134 — provenance

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 (a base row's cells are what the character wears) | `ITEM-HUMEQ-030` | High for the dispatch and the three destinations; Medium for the corpus figures |
| FR-1 (cell position picks the class; slot comes from the piece's own row) | `ITEM-HUMEQ-030`, `ITEM-ARMSLOT-031` | High |
| FR-2 (one weapon, by trained skill, ten literals in two sets) | `HERO-START-039` | High for the dispatch and the ten literals |
| FR-2 (the mage arm's literal is the wood staff) | `HERO-START-039` | High for the literal |
| FR-3 (base name off slot 1's five-bit row field; `_` off slot 2; `unarmed`→`mage`) | `HERO-APPEAR-042` | High for the gate, the slot expressions and all seventeen arms; Medium for reading slots 0 and 1 as weapon and shield |
| FR-3 (the ordered list is a shipped 25-line text payload) | `HERO-APPEAR-052` | High |
| FR-4 (directory three-way: mage / no armour / the material's own) | `HERO-APPEAR-043` | High |
| FR-4 (the directory slot is the eighth) | `HERO-APPEAR-043` with `HERO-APPEAR-047` | High — the drawable's slot 7 is read, and the wire slot is the equipment slot minus one |
| FR-4 (sixteen material blocks, `heroes` for 0–7 and 14) | `HERO-APPEAR-043`, `REG-MAT-042` | High |
| FR-5 (appearance is recomputed while the game runs, not only at creation) | `HERO-APPEAR-044` | High |
| FR-6 (two directories ship sheets under the same sixteen names) | `HERO-APPEAR-043` | High |
| FR-7 (a name matching no arm is drawn as the class left standing) | `HERO-APPEAR-042` | High |
| FR-8 (the equip path applies no class restriction) | `ITEM-HUMEQ-030`, `ITEM-SUIT-035` | High for the dispatch; Medium for reading the two bits as the two player classes |
| P-4 (nothing is composited over the world sprite — the whole body sheet changes) | `HERO-APPEAR-053` | High |
| AC-9 (the drawn body is independent of the character's sex) | `HERO-APPEAR-045` | High for the scoped statement |

## Ours by choice

| Statement | Why it is ours |
|---|---|
| FR-2: a generated mage takes the mage literal whatever slot he trained | `HERO-START-039` names exactly one mage literal and does not state what selects the mage arm. One literal cannot vary with five slots, so the selector is taken to be the class. Corroborated, not decided, by both shipped mage base rows naming that same weapon in their weapon cell and by their own drawn-class column being the staff-bearing mage's. |
| FR-1: the trained-skill weapon occupies the weapon cell, displacing the base row's own | Both routines are established; their relative order is not. The reading taken is the one under which the trained skill is observable at all, and it is the only reading consistent with the owner's report that the sprite must follow the weapon. |
| FR-5: the refresh runs on the paced path rather than inside a tick | Appearance is drawn state and reaches no hashed value, so it does not belong in the tick's fixed statement order. |
| FR-6: the bundle is filled on demand and holds what has been reached | The original caches the last name and the last material index and recomposes only when either moves. Loading every reachable body up front would decode thirty-two sheet pairs at every start-up for a character who may never change a garment. |
| AC-8: the spell attachment a shipped staff literal carries is dropped | This tree has no weapon that carries a spell. The staff itself resolves; only the attachment is lost, and it is disclosed rather than approximated. |
| P-3: an unresolvable body leaves the picture where it was | The loader's own no-error rule, applied to a second load site. |

## Open

| Question | State |
|---|---|
| What the original does when a character's dying appearance is recomposed | Deliberately not built. The two dying names are reachable in the law and no path in this tree recomposes a fallen character's appearance at all; the existing disclosure stands. |
| Which of the two weapon sources wins when both name one | Not established. See *Ours by choice*. |
| What a worn piece contributes to protection or absorption | Undecoded and out of scope. No number moves in this story. |
| Whether the material index of a piece whose name carries no material word is the first block | Not established. No shipped generated character reaches the material arm, so no shipped behaviour depends on it. |

## Removed

| Dropped statement | Why |
|---|---|
| A class-based refusal to equip | `ITEM-HUMEQ-030` establishes that the equip path applies no per-class restriction. A mage may wear a sword; he simply must not be handed one. |
| Reading the base row's own drawn-class column as the character's class | That column is a placed person's own drawn class. A player's character's class is produced from what he wears; the agreement between the two is evidence, not a shortcut. |
| A byte-form version bump | Nothing here adds a field to a simulation type. The worn set travels in a record the byte form already carries. |
