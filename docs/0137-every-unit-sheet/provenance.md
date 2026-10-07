# Provenance — 0137 every unit states its own character sheet

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-3 — a `Units` row states four statistics, five `prot Fire..Astral` and five `res.Blade..res.Shooting`, and an `XPvalue` | `UNIT-COMBAT-015` — the slot legend, `prot Fire..Astral` slots 19…23 and `res.Blade..res.Shooting` slots 24…28, `XPvalue` slot 37 | High |
| FR-3 — which of our two arrays is which family | `UNIT-COMBAT-015`'s slot numbers read against `pkg/data/unitdef.go`'s own slot switch: `Protection[0..4]` takes slots 19…23 and `Resistance[0..4]` slots 24…28, so `Protection` is the elemental family and `Resistance` the weapon-kind one. The names are NOT inverted relative to the panel's rows | High |
| FR-4 — the five weapon-kind numbers belong to the value set a unit's information display carries, and that value set holds no skill level at all | `UNIT-PANEL-010` — the 25-store copy map, which lists the five `res.*` and the five `prot *` and no skill. **The row is RETRACTED**, and the retraction is read: its overturn says "the copy map, the key and the arithmetic were all right; the *scope* was wrong" — it was published as a display-only rule and the `.alm` spawner runs the same arithmetic (`UNIT-GATE-013`). Nothing this story takes from it depends on the retracted scope | Medium — the copy map is High in the row and in its overturn, but it is cited through a retracted row and is graded down for that alone |
| FR-5 — a `Humans` row carries no protection and no resistance column, so a person's must be derived | `DAT-HUMANS-009` — the arm streams no `absorbtion`, no `prot Fire..Astral`, no `res.Blade..res.Shooting` | High |
| FR-5 — a person's five elemental protections are Spirit halved, and his five damage-kind resistances are never re-derived | `HERO-RESIST-012` | High |
| FR-3, FR-5 — a `Units` row carries no skill column and a `Humans` row carries six | `UNIT-COMBAT-015` slot legend (no skill among the twenty a blow consumes, and slots 19…28 are the two families); `DAT-HUMANS-009`'s complete 26-slot map, whose `Skill.*` block is what `pkg/data/humandef.go` streams | High |

## Ours by choice

| What the spec fixes | Why it is ours |
|---|---|
| FR-4 — the five weapon-kind numbers are stated **in the sheet's five weapon-skill positions** for a creature | **AUTHORED.** `UNIT-PANEL-011` is active and says the layout is on nothing: "which cached value appears at which position … cannot be settled without the index's own table and the drawing loop … a consumer reproducing the panel therefore has the *value set* and its *arithmetic* on evidence, and its *layout* on nothing." The position is the **owner's ruling**, corroborated only in that the value set contains those five numbers and no skill |
| FR-4 — slot 0, the General position, is stated by nothing for a creature | Follows from the absence of any such column, but the decision to print five numbers rather than six zeroes is ours |
| FR-9 — the sheet's column order in the tool | The **owner's** requested order. It is not the copy map's file order and does not claim to be |
| FR-9 — `WEIGHT` prints as unstated | No unit definition row has a weight column; `weight` exists only on the item collections. Printing `-` rather than `0` is ours |
| FR-1 — a placement that reaches no entry states no character | The existing rule for the tier lookup beside it (`entityTiers`), reused rather than re-decided |
| FR-11 — the campaign verb's table gains the two equipment collections | Engineering: the verb already claims its rows say what a player's world holds, and two collections a later story added never reached this call site |

## Open

| What | Why no meaning is assigned |
|---|---|
| Where each value is drawn on the original's sheet | `UNIT-PANEL-011`, active: unknown by construction, and this story assigns a position only where the owner ruled one |
| A person's `XPvalue` | The `Humans` arm streams no such column (`DAT-HUMANS-009`); the tool prints it unstated rather than as the entity's zero |
| A unit's weight | See above; a weight would be a property of what a unit carries, which this story does not reach |

## Removed

| Statement | Why |
|---|---|
| "A creature's five skill rows are absent in the data" | Withdrawn before the spec was written. The five `res.*` columns exist and are what the display carries; the absence is only of a *level*, and only at slot 0 |
