# 0154 — two spells complete: provenance

Evidence ledger for this story. The contract is `spec.md` and is self-contained;
nothing here is needed to read it.

## Claims this story builds on

| Claim | Confidence | What is taken |
|---|---|---|
| `MAGIC-SPELL-001` | High | The id space is 1..28 and id 0 names no row. Already carried by 0127; re-read here because the autocast field stores an id and must refuse 0. |
| `MAGIC-CAST-003` | High | The cast's whole economy: mage gate, one flat `Mana Cost` subtraction, refusal when the cost exceeds the pool, nothing rolled on a refusal. Unchanged from 0127; the restorative arm pays it identically. |
| `MAGIC-DMG-005` | High | Heal (id 6) and Drain Life (id 11) carry damage columns and are excluded from the damage arm by id. This story reads the second half of that sentence: heal's columns exist and are not damage. |
| `MAGIC-TARGET-017` | High | Heal is the only spell in the whole 6 KB apply that consults diplomacy, at `L05219`, where a hostile relation **refuses** the heal; and heal refuses a target at `health <= -10` or with `target+0x98 == 0`. Both refusals are FR-2's. |
| `MAGIC-PIC-026` | High / Medium | The picture a cast puts on the map is computed, `2*spellId + 8` for the direct arithmetic and `+9` for the burst variant. FR-5 takes the `+8` form. |
| `MAGIC-PIC-027` | High | The direct unit-addressed sender `R0617` writes `msg+0xc = 2*spellId + 8` at `L05310`, and the client's own arm `L03098` opens `AND EDX,0x1` / `JZ L03013` — **an even picture creates no map object; control goes to the branch that looks the caster up and gives it the cast action**. FR-5 is that fork. |
| `MAGIC-ICON-024` | High | A spell's icon is a position in one 480x85 strip; ids 11, 17, 27 and 28 are in no slot. Already carried by 0141; read here because the popup and the autocast border both hang off a cell that may hold no icon. |
| `MAGIC-ICON-025` | Medium | `main.res::text/spell.txt` is 28 lines keyed by spell id. **Not read by this build** — see "open" below. |
| `MAGIC-TRAIN-018` | (via 0135) | A cast trains its own school. Unchanged; the restorative arm pays the same award. |
| `MAGIC-AUTOCAST-020`, `-021` | High | The original's autocast is a property of a **weapon** carrying a spell: the caster's attack is replaced by the cast, and the routine that rolls to hit is never entered. Story 0139 built that. **It is not what this story builds** — see "authored" below. |

## What is ours by choice

**The autocast toggle is authored end to end.** `MAGIC-AUTOCAST-020` and `-021`
describe a weapon-borne autocast: the trigger is `weapon+0x80 != 0` and the
actor's ordinary attack becomes a cast. Nothing published describes a
player-facing toggle that puts a **book** spell on repeat. The owner ruled on
2026-08-14 that he wants one, by a key combination, with a rotating dashed border
on the icon while it is on. He is the author and the ruling is the fact (B3).

Everything below it is ours, chosen for being the least that satisfies the
ruling: which key (`Ctrl+A`), that the toggle acts on the spell already selected
in the book, what an autocast aims at, the cadence between two unbidden casts,
and the border's own geometry and speed.

**Heal's magnitude is ours.** No published claim states what the heal arm
restores. `MAGIC-DMG-005` establishes only that heal's damage columns exist and
are not damage. FR-2 uses the row's own columns at the caster's own power —
the same arithmetic a damage cast already runs — because those columns are the
only per-row magnitude the row carries. It is authored and `spec.md` DD-2 says
so.

**The mark's appearance and life.** `MAGIC-PIC-027` fixes that a cast gives the
caster a cast action and no map object. How long that action stands (ticks) and
what it looks like on screen are ours.

**The popup's content and layout.** `MAGIC-ICON-025` records that the shipped
`text/spell.txt` carries 28 description lines keyed by spell id, and this build
does not read that file: the popup states the numbers this tree already holds in
the world's own spell table. Reading the shipped description text is a later
story, not a silent gap.

## Open

- **Which spells take the deferred `+9` path** is `MAGIC-PIC-027`'s own Unknown
  ("which routine calls each sender"). This build implements only the direct
  sender's `2*id + 8`, so no spell here produces a map object. A later story that
  decodes the deferred path adds the odd-picture branch beside FR-5 rather than
  changing it.
- **`spell+0x10`, the recomputed duration**, is read by nothing per
  `MAGIC-EFFECT-015`. Not used here.
- **The remaining 26 spells.** This story completes two. The other 26 keep the
  0127 behaviour: damaging ones cast, the rest are refused by the table's own
  flags.

## Removed from scope during the story

Nothing. Every behaviour named in the brief landed; `spec.md`'s SC block states
what was cut before work began.

## The second round

| Claim | Confidence | What it supplies |
|---|---|---|
| `MAGIC-PIC-026` | High / Medium | the picture id is `2*id + 8` for a direct cast and `+9` for the burst variant, and `projectiles.reg` holds a defined row at `2*id + 8` for fifteen spells including Fire Arrow. Read for the fact that the art exists and the original draws something; the arithmetic itself is not implemented here |
| `MAGIC-PIC-027` | High / Unknown | the parity fork on the client's `0x86` arm, and that `0x8b` and `0x8c` allocate a projectile with no parity guard. Its Unknown is which routine calls each sender, which is why the fork cannot be resolved from the pin and why spec DD-7 authors instead |
| `SAV-SPELL-044` | — | what a saved `Spell` record's `+0x08` names is Unknown, which is why a restored character's book comes from his `Humans` row and not from his save (spec DD-8) |
| `ANIM-CLOCK-024` | — | the original schedules the swing frame, the swing sound and the damage from three separate numbers. Unchanged by this round: FR-20 aligns this tree's own two clocks with each other and asserts nothing about the original's three |

**What is ours by choice in this round**: that every applied cast flies (DD-7);
the bolt and burst geometry, the flight span of three ticks and the burst of four
(FR-19); the reading of "in battle" (DD-9); the mana reserve of one quarter
(DD-10); and the priority order itself, which is the owner's ruling written down
rather than a decoded fact.

**What is open**: which sender a book-cast Fire Arrow reaches in the original,
and therefore whether it flies there. `EXP-0163` is running on it. Taking its
answer would change DD-7's authored branch into a decoded one and would not
change the seam.
