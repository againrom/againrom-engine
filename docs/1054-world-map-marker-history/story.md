# Story 1054: persist world-map marker history

## Player result

A picture-bearing mission selected on the world map keeps its marker after an
againrom save is loaded. Starting or importing a different game clears the
previous game's marker history.

## Behaviour

- A native snapshot carries the exact set of positive mission numbers selected
  at least once. The writer sorts the set so one state has one byte form.
- Town and in-mission snapshots capture the same set before their save paths
  diverge. Native restore installs it after clearing the previous game's town
  presentation state and before the world map can paint.
- A save written before this additive gob field existed restores an empty set.
  The envelope remains version 1 and the simulation form remains version 65.
- New game and original-save import retain their existing full-session reset.
  They never inherit this build's native marker history.
- A non-positive or duplicate mission number in a directly supplied snapshot is
  rejected before the active game changes.

## Authority and scope

`TOWN-123` is High: ROM1 serializes selected picture-bearing missions in the
marker cache that world-map paint reads. This story persists a broader,
presentation-equivalent set containing every positive mission selected in this
build. It narrows `DIV-138` to that membership mismatch rather than closing it.
It does not import the cache from an original SAV, persist the separate
world-map position debt `DIV-137`, or change marker art and selection controls.

Touched domains are campaign/session, persistence, and world-map UI. Simulation,
combat, AI, inventory, mission scripts, and installed content are unchanged.

## Proof

- Synthetic tests cover sorted capture, envelope compatibility, town restore,
  mission-candidate commit, malformed-state refusal, and new-game reset.
- One install-gated test on both lawful roots selects a shipped picture-bearing
  mission through the production world map, saves, encodes, decodes, restores,
  and observes the same shipped marker before proving a fresh game has none.
- Candidate gates are full Go, the no-asset scan, and one EN/RU release run.
  One fresh-context adversarial review is the stopping pass; only a
  player-visible or hashed-state finding returns one bounded correction.
