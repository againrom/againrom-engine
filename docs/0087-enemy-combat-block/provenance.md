# Provenance — why a scenario human's numbers are what they are

Every claim below was read in the `research/` submodule at this story's pin. The one it does NOT
rest on is the hypothesis that opened it: `Man_Club` is a *Humans* row index, not this mission, and
the row it names is placed on no map this story touches.

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 — the four arms, and which band a placement's class key puts it in | `MISSION-ARM-006` | High |
| FR-1 — the twenty-three slots, their order, their widths and their destinations | `DAT-HUMANS-008` | High |
| FR-1 — an empty cell stores nothing and advances anyway, per cell | `DAT-ACT-006` | High |
| FR-2 — the `Humans` collection carries no damage, no `attackKind`, no absorption, no protection and no resistance column | `DAT-HUMANS-009` | High (the map is the whole routine) |
| FR-2 — a human's damage is the recompute's, `ftol(1.1^Body/20)` into both halves | `DAT-HUMANS-009`, `HERO-COMBAT-011` | High |
| FR-2 — the pair is `(base, spread)` and never `(min, max)` | `HERO-DAMAGE-022` | High |
| FR-2 — the bare roll is `[d, 2d]`, and is `0-0` below Body 32 | `HERO-BARE-037` | High |
| FR-2 — to-hit is `ftol((1.1^Body + 1.1^Reaction)/5)`, defence is `Reaction/3` | `HERO-COMBAT-011`, `DAT-HUMANS-009` | High / Medium (below) |
| FR-2 — the active skill adds `3x` to to-hit and a fifth to the base alone | `HERO-COMBAT-011` | High |
| FR-3 — an equipment cell is a `[tier ][material ]shape` name resolved against `Weapons` | `UNIT-EQUIP-005`, `DAT-ACT-006` | High / Medium on the grammar |
| FR-3 — a weapon's numbers are `ftol(column x shape x material)` off the nine-double ladder | `ITEM-LADDER-019`, `ITEM-DMGFACT-020` | High |
| FR-3 — the `Weapons` slot map, and the melee arm's bound at attack type 10 | `ITEM-WEAPCOL-021` | High |
| FR-3 — a weapon ADDS damage, to-hit and defence and ASSIGNS the cadence | `HERO-EQUIP-017` | High |
| FR-4 — each cadence assignment is guarded on the empty cell, so an empty one leaves the template's standing | `ITEM-WEAPCOL-021`, `UNIT-COMBAT-006` | High |
| FR-5 — health is the `HealthMax` column; `M10_Brigands` is 15 | `DAT-HUMANS-009`, `UNIT-M10-018` | High |
| FR-6 — the spawner's humans arm jumps the whole three-way difficulty block | `UNIT-GATE-013` | High |
| FR-7 — the units arm's own twenty combat columns, unchanged by this story | `UNIT-COMBAT-015`, `UNIT-STREAM-001` | High |
| FR-7 — a non-hero derives nothing; its template numbers ARE its stats | `UNIT-DERIVE-003` | High |
| AC — mission 1's hostile set, and `M10_Brigands` x2 through the humans band | `UNIT-M10-018` | High |

## Ours by choice

| What the spec fixes | Why it is ours |
|---|---|
| **The constructor defaults an empty humans cell falls back to** are the ones this tree already publishes for the base actor. `DAT-ACT-006` establishes the per-cell fallback law and `UNIT-COMBAT-006` names the values on the *units* arm; nothing read here says the `0x1e8` object's constructor agrees with the `0x198` one, and nothing says it differs. Reachable only for a cell a shipped row leaves empty. |
| **Which equipment string is the weapon.** The original resolves all ten by name against three collections; this build reads one weapon and no armour, and takes the FIRST cell that resolves as a melee `Weapons` row. Position is not the rule — a cell that names a shield or a mail is simply not a weapon. |
| **The prefix split inside a name** — longest table entry that is a word-boundary prefix, shape then material. Already ours from 0078 DD-3, and reused rather than re-decided. |
| **Speed comes off the `speed` column** rather than off `HERO-SPEED-008`'s Reaction derive. The column is a decoded cell; the derive is a formula that would replace it only if the recompute runs at spawn, which is open below. Where research names a field as the recompute's output this story takes the recompute; where it does not, it takes the cell. |

## Open — deliberately given no meaning here

- **Whether `R0280` runs at spawn on a `.alm`-placed human.** `DAT-HUMANS-009` grades this
  Medium and settles only that every equip path can reach it. This story takes the recompute for the
  three numbers that row names as its outputs — damage, to-hit, defence — and the streamed cell for
  everything else. If the recompute is later shown to run whole at spawn, two numbers move:
  **health**, which `HERO-HP-005` derives from Body and experience whenever the `HealthMax` column is
  non-zero, and **speed**. `HERO-HP-005`'s arm also needs the `fighter` multiplier, which is settled
  for the two hero classes and for no scenario human — so deriving health here would have meant
  inventing that bit. **This is the one question this story would ask research.**
- **The `Defence` column's fate.** Streamed at slot 9 and, on the same Medium, overwritten by the
  recompute. Carried in the definition and not read.
- **Armour and shields.** Nine of the ten equipment cells. Absorption therefore stays zero on this
  arm, which is what a wielder of a weapon alone carries anyway.
- **`Mana`, `RotationSpeed`, `ScanRange`, `TokenSize`, `TypeID`, `Face`, the six skills' base copies,
  `DyingTime`, `serverID` and `knownSpells`.** Streamed or fetched by the original; carried here
  where the streamer writes them, read by nothing in this tree.

## Removed

- **The auto-hit mark's absorption skip** (`HERO-AUTOHIT-031`, High). Decoded, and eleven of mission
  1's twenty-one hostiles carry the mark. It is not this story's: it changes no outcome while the
  party's own absorption is zero, and it belongs to the blow resolver rather than to a loader. Named
  in the spec's out-of-scope list so it is a known gap rather than a forgotten one.
- **The `Units` arm's `EquipItem` column** (`UNIT-EQUIP-005`). Twenty-six of fifty-six classes carry
  one and none of mission 1's three does.
- **The difficulty adjustment on this arm.** Not omitted — `UNIT-GATE-013` says the spawner jumps it.
