# 0160-dialogue-dress — provenance

Research pin: `733bdc9`.

## Claims this story builds on

| Claim | Confidence | What it supplies |
|---|---|---|
| `DLG-FIGURE-020` | High | The dialogue figure is the world figure compositor's output on the speaker's own drawable. The dialogue site is not a second composer. |
| `DLG-FIGURE-021` | High | No layer is excluded at the dialogue site. Equipment slot 6 is the head slot and is drawn in both halves of the compositor. |
| `DLG-SPEAKER-022` | High for the two arms and for the empty array; Unknown which arm a given shipped dialogue takes at run time | The speaker is a live actor when one matches the section, and otherwise a synthesised drawable whose twelve equipment slots are cleared and never written. A live speaker's figure carries whatever its slots hold at that moment. |
| `DLG-SPEAKER-023` | High for the token list and the bits; Medium for the two censuses | The live-actor predicate: seventeen `Flags` tokens combined one term per token present, plus the `Face` and `Picture` record comparisons and a state gate. Names the bits `Hero` 0, `Mage` 1, `Female` 2, `Human` 4, `Me` 5. |
| `DLG-DRESS-024` | High for the `40.alm` join; Medium for the per-slot census | Mission 40's `npc25` is `Humans` row 42 `PC_Paladin` with a seven-cell outfit including a plate helm in slot 6. 110 of 166 outfit-bearing rows fill slot 6. |
| `REG-NPC-088` | (carried from 0141) | The record's `Picture` is stored into the synthesised actor's typeID and its `Face` into that actor's face byte. Those are the two fields `DLG-SPEAKER-023`'s comparisons read. |
| `MISSION-ARM-006` | (carried from 0065) | A type-6 placement with the npc flag resolves through the section's `DataBinID` to a `Humans` row. Already implemented in `pkg/mapload`. |
| `ITEM-HUMEQ-030` | (carried from 0128) | The cell-to-slot rule for a `Humans` row's ten equipment cells. Already implemented as `wearRow`. |

## Owner correction

- **A speaker that resolves to no live actor is bare.** `DLG-SPEAKER-022` establishes twelve cleared
  equipment slots. The 2026-08-15 owner override dressed that arm from a `Humans` row. On
  2026-08-24 the owner observed the resulting invented armour on the tavern keeper and the school's
  quest giver and reversed that override. Hotfix `00ddc4fd` removes the derived outfit table and
  restores the published original rule. Live matched actors still use their own current equipment.

## What is ours by choice
- **The `Hero` and `Human` predicate terms test one actor property.** `DLG-SPEAKER-023` names bit 0
  as `Hero` and bit 4 as `Human` but does not decode what writes bit 0. This build answers both
  terms with "this actor was placed down the humans band", which is bit 4's own meaning.
- **The `Platoon` term is not evaluated** and narrows nothing. Nothing in this tree holds a
  platoon.
- **Candidate order is ascending entity id.** The original walks the client actor list at
  `this+0x9b8`; that list's order is not decoded.

## Open

- Which of "fully dressed" and "dressed without the helmet" the original shows for a live matched
  actor is EXP-0165's question. This story draws whatever that actor's resolved worn set holds,
  head slot included, which is `DLG-FIGURE-021`. The synthetic arm is separately settled as bare.
- `DLG-SPEAKER-022` grades as **Unknown** which arm a given shipped dialogue takes at run time. This
  story therefore cannot be verified against a per-mission expected arm; it is verified against the
  predicate and against mission 40's decoded join.

## Removed

Nothing decoded was cut.
