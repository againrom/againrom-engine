# 1108 — Non-party saved holdings

## Result and contract

Both original mission LOAD doors replace holdings of uniquely matched living
non-party Unit, Humanoid and Human actors across the complete Player/group graph.
Empty holdings clear starter stock. Canonical code, kind, signed price, ordered
state-0 effects, quantity and represented held/worn/carried roles persist through
item operations and ordinary AGS SAVE/fresh FrontEnd LOAD.

This is one required step toward full five-path SAV fidelity, not full item or
world preservation. DIV-746..748 name required follow-up.

## As built

- `sav.ActorHoldings` uses a fresh exact document walk. Repeated actor references
  identify one owner; repeated items across separate ownership fields refuse.
  Sparse armor positions survive without widening Party or character.
- The join counts every nonzero source and target MapUnitID before filtering.
  It excludes persistent party, dead/dying, off-map, unbound and unmatched actors
  with separate counters. There is no first-wins ScriptUnits or definition/cell
  fallback.
- Validation rejects late invalid items, ambiguous identities, unsupported
  effects and unrepresentable equipment without mutating stock or weights.
  Compact stacks are bounded across the whole batch before any flat expansion.
- `sim.ImportOriginalActorStock` updates holdings, load and weapon-item cache
  together. Item removal clears only an item-owned cache. Legitimate independent
  innate/legacy sources survive; conflicts refuse. LOAD never replays equip
  current-pool deltas or teaching.
- A detached native world stages stock and Human Rearm without losing the
  previously restored cell-trigger overlay. Saved spellbooks follow Rearm;
  saved actor pools are the final actor write. Rearm retains the observed second
  physical pair from 1104. Future saved current-profile assignment belongs before
  pools. No 1107 code or newer research pin is consumed. This story adds no
  byte-form version change to the reconciled master.
- Stack folding now also distinguishes concrete Kind; the existing signed Price
  guard remains. Current code-derived weight is not original per-instance weight.

## Authority

Promoted pin `ba21c9aa9a949023b3d678b22ca29b3a3b0cd95f`:
SAV-ROSTER-024, SAV-ID-015, SAV-CARRY-050, SAV-TOKEN-034,
ITEM-SAVE-014 (amended), ITEM-HUMEQ-030, ITEM-STACK-003 (amended),
ITEM-EFFOBJ-072, ITEM-EFFSAVE-077, ITEM-EFFSPLIT-074, ITEM-EFFDISP-075,
ITEM-DEATH-012 (amended), ITEM-CORPSE-034 and ITEM-AUTHDROP-087.
Claims were read through the pinned claim tool, including retraction records.

Original armor slots 1/2 are not held weapon/shield aliases. Claims establish
separate fields, not legitimate multi-role item ownership. Repeated whole actor
references are deduplicated; shared items across fields refuse instead of being
copied or reclassified. Effect list order and duplicates remain intact.

## Proof and required follow-up

See [verification.md](verification.md) for exact natural and synthetic witnesses.
The natural M10 Witch loses three incorrect starter potions; natural M40 staff
price changes from template 440 to saved 981, and survives death, loot, equip and
another ordinary App SAVE/fresh LOAD.

Required follow-up includes full source item weight and attack/defence blocks,
Token and container identity, container metadata, other serialized tails,
fourteen-slot equipment and non-state-0 Effect lifecycle. These omissions affect
fidelity and must not be described as irrelevant. This story neither writes
original mission SAV nor asserts that any of the five full-fidelity paths is
complete. Unmatched/dynamic/dead/off-map holdings remain explicitly outside this
living matched-actor handoff.
Saved current actor load/capacity also requires its own assignment after the
stock/rearm stage; computed load is not proof of the original saved word.
