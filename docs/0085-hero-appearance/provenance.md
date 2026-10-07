# Provenance — the hero's appearance

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 (a player's character's drawn class is derived, not carried) | `HERO-APPEAR-041` — for type ids in `[0x20,0x40)` the client discards the shipped id, banks two bits in `+0x18c` and stores the literal `1` into the frame selector's subscript; `HERO-APPEAR-042` — `R0551` then derives the real answer | High: every branch, immediate and store is a named instruction, and the range tests are fixed by their own branch displacements |
| FR-1 (that a *placed* actor is the other case, so one route cannot serve both) | `UNIT-APPEAR-030` — the class array is subscripted by the streamed type id and the two non-hero arms leave it alone; the shipped roster occupies `1..27` and `64..80`, so the hero range is occupied by no class | High for the enumeration and each branch; **Medium** that no other module writes the field |
| FR-2 (the seventeen-name table) | `HERO-APPEAR-042` — seventeen `MOV dword ptr [EAX+0x20], imm32` arms giving fifteen distinct class keys from seventeen names, reproduced on both roots by a second instrument reading raw PE bytes with no disassembler in the path | High |
| FR-2 (the fallback a name matching no arm leaves) | `HERO-APPEAR-041` — the literal `1` is stored **before** the chain runs, and `HERO-APPEAR-042`'s arms are compare-and-store; the roster's own name for `1` is `Unarmed Fighter` | High: it is the composition of two named stores, not a separate observation |
| FR-2 (the shield suffix and the two substitutions) | `HERO-APPEAR-042` — a non-null second slot appends `"_"`; a mage whose name came out `unarmed` becomes `mage`, or `mage_st` under the dying action code; `HERO-APPEAR-046` restates the dying arm from the art's side | High for the arithmetic and the three special cases; **Medium** for reading the slots as *weapon / shield / armour* — the words are an inference and this story does not use them as evidence |
| FR-3 (the directory three-way, and its sixteen-value table) | `HERO-APPEAR-043` — a mage always takes `heroes`; a fighter with no object in the armour slot takes `heroes_l` (branch target fixed by its own `rel8`); otherwise the material's own `Path`, which is `heroes` for blocks 0–7 and 14 and `heroes_l` for 8–13 and 15, byte-identical on both roots | High |
| FR-4 (the composed sheet path, and that the class record's `File` is not it) | `HERO-APPEAR-043` — `graphics\units\` + directory + separator + body name + `\sprites.256`, into two bitmaps at `+0x194`/`+0x198`; the cross product against what ships leaves exactly the two explained gaps | High |
| FR-4 (that the class record still supplies the geometry) | `HERO-APPEAR-043`'s "only the geometry does", and `HERO-APPEAR-046` — every shipped hero sheet holds exactly the frame count its assigned class record predicts on 14 of 16, both residuals equal to an independently derived constant | High; the causal gloss that the art was authored to the rule is the row's own **Medium** and is not used |
| FR-8 (that the appearance is recomputed while the game runs) | `HERO-APPEAR-044` — ten call sites, six inside the client dispatcher, four of them the field-masked state sync | High for the enumeration and the opcode attribution |
| The roster's own id and name columns | `EXP-0039`'s `evidence/unit-classes.csv`, read through `REG-UNITS-018`'s framing | measurement over the shipped registry on both roots |

**One correction, and it is a summary and not a mapping.** `HERO-APPEAR-042` says the seventeen arms
give *fifteen* distinct class keys. The seventeen arms it enumerates give **sixteen**: the seventeen
keys are `1 2 3 4 5 7 8 9 10 11 12 13 14 14 15 24 23`, and `archer`/`bowman` is the only pair that
shares one. The raw-byte reproduction beside the claim carries the same seventeen rows, so the
enumeration and the summary disagree with each other and the enumeration is the evidence. Every
individual name-to-key pair this story uses is unaffected; what is wrong is one adjective. Reported
back rather than worked around.

## Ours by choice

Nothing below is asserted by any source.

| Choice | Note |
|---|---|
| **The body name the party's hero wears.** | The one authored value. The ordered list the equipment slot indexes is not decoded (see *Open*), and this tree has no equipment slot to index it with, so the name is chosen. It is one element of a set the corpus pins, behind one function, and everything after it is derived |
| The rule that produced it — the authored starting weapon is the blade arm's, and the sword family's unshielded one-handed body is `swordsman` | written down so a reader can disagree with the rule rather than with a string |
| That the derivation runs **once**, where the party is built, rather than on every state change | this tree has no channel through which a party member's equipment can change during a mission |
| That an absent or undecodable hero sheet leaves the member drawn from the derived class's own art | the font's precedent: a cosmetic asset does not gate a mission opening |
| That the composite carries the derived class's corpse link | see *Open*, third row |
| The per-entity shape of the art override — a lookup resolved when the map opens | `tiers` and `chars` are the same shape for the same reason |

## Open

| What | Why it is left undecided |
|---|---|
| The **order** of the name list the equipment slot's five-bit field indexes | EXP-0113's own bounded Unknown: the table is BSS-backed and built at run time, so the corpus pins the sixteen names it must yield but not their order. Nothing here reconstructs it, and no mapping from a weapon to a body name is invented — the name is authored instead, and said to be authored |
| What the five-bit field on an equipped object actually is | not decoded, and this tree has no object carrying one |
| Which body a **dying** hero is drawn in, in this tree | the original's rule is decoded and is held in `pkg/data`; this tree's death path substitutes the class record's own corpse class instead, which is the placed unit's rule. Disclosed in FR-8 rather than modelled |
| The `spritesb.256` sibling's role | EXP-0113 states it is built and stored beside the base sheet and that its consumer was not read. Composed by this tree's path function and read by nothing |
| Whether the sex axis reaches anything outside the two routines read | `HERO-APPEAR-045` is deliberately scoped to those two. This story reads neither axis for a fighter and asserts nothing about it |

## Removed

| Dropped | Why |
|---|---|
| A twelve-slot visible-equipment array on the party member | it would be twelve fields of which this tree can fill at most one, and the one it could fill needs the undecoded index to mean anything. An empty structure asserting a shape is worse than a named absence |
| A map from a weapon's shape name to a body name | the correspondence is legible and is not evidence. Inventing it would put an undisclosed decode in the one place this story exists to keep honest |
| Loading every shipped hero body rather than the one the party wears | the set is pinned and the loader could walk it, but nothing would read the other fifteen, and the load already decodes every frame of every unit sheet |
