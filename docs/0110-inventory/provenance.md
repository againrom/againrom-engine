# Provenance — 0110

Every normative section of `spec.md` appears in exactly one register below. Claim IDs are the
`research/` submodule at this story's pin; confidence is the grade the claim carries, and where a
claim is graded twice the clause this story leans on is named.

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 — four fields at 15..12, 11..8, 7..5, 4..0 | `HERO-APPEAR-049` | High — the formatter and the builder each read whole; the inversion of all 928 shipped leaf names over the whole 16-bit domain produced 928, 0 unproduced, 0 ambiguous, which excludes a layout off by one bit anywhere |
| FR-1 — the same layout on an authored item's code, and class 14's index widening to the low byte | `ITEM-CODE-029` | High for the masks and the four allocation sizes; the meaning of the three constructor arguments is published **Unknown** |
| FR-2 — the two name forms, seven digits either way, and which one class 14 takes | `HERO-APPEAR-049` | High |
| FR-2 — field D is the definition row index and field C is `item+0x45` | `ITEM-APPEAR-023` | High for the builder and its five call sites; **Unknown** what `item+0x45` means |
| FR-2 — a wrong C costs a sheet and not a behaviour | `ITEM-APPEAR-025` | High for the second reader consuming field D alone; **Medium**, and scoped to the figure path, for C affecting nothing else |
| FR-3 — the icon lives at `inventory/<name>.16a` | `ITEM-APPEAR-024` | High — the composition, the open and the polarity of the return are named instructions; the 416-node count is an exact measurement on both roots |
| FR-3 — the figure layer at `equipment/<dir>/primary/<name>.256`, the four directories from the mage bit and the sex bit, and the base sheet from the directory and a face byte | `HERO-APPEAR-051` | High for the routine, its loop bound fixed by its own relative displacement |
| FR-3 — those sheets are one frame of 160x240 and are the info window's figure, not the world's | `HERO-APPEAR-053` | High for the world half (an exhaustive frame-count measurement over both trees); **Medium** for the compositor being the info window — four of 41 callers identified |
| FR-4 — twelve slots, numbered, and equipped is a different place from carried | `ITEM-EQUIP-006` | High for the slot map and the humanoid gate; **Medium** for no per-slot type restriction |
| FR-4 — field B is the equipment slot, and ten of the twelve carry shipped content | `HERO-APPEAR-050` | High — a discriminating corpus test against an instruction-level rule, reproduced on both roots, where four of five candidate offsets are excluded |
| FR-5 — an item is one of four classes and the class picks the definition table | `ITEM-CLASS-001`, `ITEM-DEF-002` | High for both; `ITEM-DEF-002` publishes **Unknown** which column of a row means what |
| FR-8 — the original itself tests whether a slot's sheet opens, and reports rather than fails | `ITEM-APPEAR-024` | High |

## Ours by choice

Engineering this build fixes because nothing decoded fixes it. Each is changeable later without
contradicting any source.

| Choice | Spec anchor | Why it is ours |
|---|---|---|
| Field C of a name-resolved weapon's code is the shape index its own name already yields | DD-1, FR-5 | `ITEM-APPEAR-023` publishes `item+0x45` as three bits nothing names, and `ITEM-DEF-002` leaves every row's columns uninterpreted, so no source says where an item built from a row gets C. AC-3 is a criterion the shipped archive can fail |
| The party member's figure directory and face byte | DD-2, FR-7 | Character generation is a screen this tree does not have; the sex bit and the face byte are its inputs and nothing decoded supplies them for a generated character |
| The window's frame and background are the frame primitive already in the drawing tier, not the shipped chrome | DD-3, FR-6 | A cut, not a claim. The shipped `interface` art exists and is deliberately unread |
| How many pack cells are visible | DD-4, FR-7 | `ITEM-CONT-004` establishes the container has **no slot count and no capacity**, so how much of an unbounded list a window shows is the window's own choice |
| Which input opens and closes the window | DD-5, FR-6 | Nothing decoded names the binding |
| The base sheet is painted first and an occupied slot's layer over it | DD-6, FR-7 | With one occupiable slot the order is forced; the decoded order rule is cut, not contradicted |
| An equipment slot's code is a loader value and reaches no simulation type | DD-7, FR-10 | The same divergence the equipment channel already disclosed |

## Open

Undecoded, and deliberately assigned no meaning here.

- **What field C is.** Published `Unknown`. This story spells it into a name and reads nothing out
  of it.
- **Which column of a definition row carries an item's material and shape bytes.** `ITEM-DEF-002`'s
  own open item. Until it is answered, an item built from a row cannot state its own code, and only
  an item that arrives *as* a code, or one composed from a name, can be drawn.
- **Which slot and which table classes 3 to 13 belong to.** `ITEM-CODE-029` puts them all on one
  constructor; `HERO-APPEAR-050` shows ten of twelve slots with shipped art. Not needed while only
  slot 1 can be occupied.
- **What the original binds to opening this window.**
- **Whether the two shipped `backpack` sheets are the sack on the ground.** Nothing here reads them.

## Removed

Statements considered and dropped, so nothing load-bearing vanishes silently.

- **The paint order over twelve slots, the `secondary` sheets and the two-handed layer swap**
  (`HERO-FIGURE-060`, `HERO-APPEAR-051`). Unobservable while eleven slots are empty; naming a rule
  no test can exercise would be a contract nothing witnesses.
- **The container, the stack count, the per-unit weight and the load penalty** (`ITEM-CONT-004`,
  `ITEM-STACK-003`, `ITEM-LOAD-005`). The pack is drawn empty instead; the claims stand unused.
- **Moving an item, and the ground as a source or destination** (`ITEM-CMD-007`, and the
  contradiction over `actor+0x50` that its retraction row resolves in favour of both sides). No
  command reaches an item in this story.
- **The panel's weapon damage line** (`ITEM-PANEL-022`). Already landed elsewhere; the window
  states no number.
