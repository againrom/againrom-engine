# Per-instance saved item weight

## Contract

FR-1: Import the signed Item F4A per-unit weight from SAV into every admitted
party, ground and living non-party item. Source weight, including zero, takes
precedence over code-derived fallback. Both original LOAD doors share the
canonical holdings seam.

FR-2: Transfer, split, equip, unequip, drop, pickup, death and consumable
reservation preserve that weight. Load calculations in the world and city use
the actual instance. Equal codes with distinct weights must remain distinct.

FR-3: Form 74 and the AGS envelope preserve every item carrier. Genuine form 73
and older readable forms retain absent-weight fallback without inventing values.
Old source-city baselines retain their old comparison policy.

## Design

DD-1: ItemInstance and ItemStack own signed Weight and WeightPresent. An absent
weight must be zero. Native/code-only producers retain the existing code table.
ITEM-STACK-003, ITEM-LOAD-005 and ITEM-SAVE-014 are the promoted authority.

DD-2: A sparse form-74 section follows original-dead provenance. Each record
contains a sorted unique item ordinal and signed weight; its trailing byte span
locates it before the relation. The traversal covers sacks, carried stacks,
equipment slots and reserved scrolls. No sidecar survives decoding.

DD-3: Distinct instance weights do not merge, including explicit versus absent
weight. Kind and Price retention remain independent guards. This state-retention
choice differs from ROM1's narrower item equality. City import policy4 retains
weights independently of SalesVersion1 and completed-Sell boundaries. Legacy
policy1..3 creates absent-weight comparison values, without clearing live items.

## State and debt

This candidate composes landed master, including story 1107's saved
non-party current profile and the 70 further commits under it. No original
process may launch. This is not a full SAV writer result.

Stored container load, presence, order and insertion index remain required
follow-up. Recomputed item-weight sums do not preserve stale saved actor load.
Full Token, derived item fields and source object lifecycle remain required
follow-up. This story does not invent their producers or default values.

## Proof

See verification.md for the measured candidate and every FR/DD witness.
