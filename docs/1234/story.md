# Terminal loot on passable ground

## Intent and authority

The owner requires dragon death sacks to stay off water and impassable bridge
walls. Loot from an impassable death cell must land on nearby free ground.
This rule applies to terminal death, including original-bound item objects.
Original-runtime nearest-cell behaviour is unmeasured; DIV-1441 records that
limit. No ROM1 claim is inferred from this implementation.

## Contract

- Keep the corpse position. Keep ordinary death placement and sack merging
  when the corpse cell permits ground movement and sack registration.
- Otherwise search increasing Chebyshev distance, then ascending Y and X.
  A destination must admit ground movement and sack registration, contain no
  sack, and have no other ground occupant. Search beyond adjacent cells when
  necessary, up to the map bounds.
- Share the destination across native, source-backed and saved-object terminal
  transfers. Preserve item order, object identity, gold and once-only teardown.
- If no destination exists, retain loot and random state and retry before
  completing terminal teardown. Reload must preserve this pending death.
- Leave authored and loaded sacks at their recorded positions. Manual drops
  and script drop-all retain their existing placement rules.

## Proof

[verification.md](verification.md) records the four terminal producers,
pending-death SAV recovery, ordinary Sack wire fields and the EN owner-save
frontend witness. Final gates and review remain landing obligations.

## Open debt

Existing blocked sacks in owner saves require a separately scoped repair of
their current sack position and saved-object metadata. This change does not
rewrite them on LOAD. Original-runtime placement and live display remain
unwitnessed.
