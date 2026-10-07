# Native projectile registry

A physical ranged swing at a unit now builds a World projectile record on its
release tick. The World advances it, SAVE writes its `Prj` section and a cold
LOAD continues it to its end.

## Intent

The shot of a native ranged swing was a presentation object. SAVE in flight
wrote the empty `Projectiles` subtree and LOAD drew no arrow (DIV-1721,
DIV-1453). The owner needs a save taken with an arrow in flight to keep it.

## Authority

Pinned snapshot k111 (EXP-0428 claims).

| Rule | Claim |
|---|---|
| Built at the class ShootDelay tick of a physical swing, from the class's Projectile picture; none for melee, the cast spawner or a cast-diverted attacker | `SAV-1129` |
| Leaves: start point from the shooter's point and the class ShootOffset; action 1; actiontarget the shooter's target; actionsegments the truncated distance over 200; picture the class's | `SAV-1130` |
| Id from a 16-bit counter (`FreeIndex`), inserted at the head of bucket `(id >> 4) mod 17`, saved in bucket order | `SAV-1131` |
| Pictures 0..12 apply no damage; the blow stays with the swing countdown, independent of the record | `SAV-1132` |
| A record is collected only when its own countdown ends; a missing target leaves it flying to its last aim point; no shooter is stored | `SAV-1133` |
| Section shape, loader behaviour, authentic sample | `SAV-PROJSTORE-428`, `SAV-PROJLOAD-429`, `SAV-PROJCORP-430`, `SAV-1134` |
| Driver arithmetic, draw facing | `ANIM-PROJ-025`, `ANIM-PROJ-026`, `ANIM-PHASECLOCK-028` |

## As built

- `sim.World.ReleaseUnitShot` builds the record: id from `FreeIndex`
  (then `FreeIndex = id + 1`, 16-bit), id inserted at its bucket head in `IDs`,
  leaves per `SAV-1130`, a `SavedProjectileDriver` bound to the target, and one
  driver call in the same tick. A start-to-target distance under 200 units
  gives zero segments, so the record is collected on that first call. The call
  refuses a shooter without a unit target and any picture outside 1..12.
- `pkg/game/unitshot.go` calls it on the tick the swing clock reaches the class
  ShootDelay while the target is in reach, as before. The class picture, the
  sheet's phase count, the release offset and the direction leaf come from the
  installed registries (`ShootOffset` is now carried on the render class). A
  run restored mid-wind-up resumes as the wind-up its saved phase names, so a
  cold LOAD releases the shot the uninterrupted run releases.
- The existing driver (`stepSavedProjectile`) flies the record: re-reads the
  target point each call, moves one segment share, advances phase, collects on
  the call after the last segment. A removed target detaches the record, which
  finishes toward its last point.
- SAVE: a record without a Document section is bound on first projection
  (`bindProjectileDrivers`), written with all sixteen leaves, and its
  actiontarget is the target actor's runtime identity in the Document
  namespace: a native target is given its entity id when that id is free, so
  SAVE then LOAD returns the same record. An allocator that has moved past a
  collected record is written even when no record remains. A fresh-world first
  SAVE constructs the sections too.
- LOAD: both original doors rebuild the record from its section and bind the
  driver to the actor with that runtime identity. Native actors carry no
  source runtime identity, so the continuation check accepts a native target
  by entity binding.
- Draw: a record is drawn from its persisted position, phase and direction, as
  a restored original record is. Pictures 10 and 12 draw smoke at the last six
  positions of the record, a presentation history that is not saved. A swing at
  a structure, or a class picture above 12, keeps the unsaved presentation
  shot.

## Damage timing

The blow resolves on the simulation's own swing countdown, which already adds
the flight allowance `(d * 256 + 128) / 200` ticks (`SAV-1132`). The record
applies no damage and the countdown does not read it. Damage timing is
therefore unchanged: before and after this story it lands at the swing
countdown, not at the arrow's arrival. Witness: the continuing world and the
cold-loaded world have identical target hit points on every tick after the
SAVE, and the blow lands after the record is collected.

## Proof

| Input / instrument | Result |
|---|---|
| `pkg/sim` unit tests: leaves, counter, bucket order, wrap, refusals, flight and collection, binary round trip | pass |
| `TestReleaseFreshShotWritesProjectileRecord`, mission 41 on each edition, hero bow, first non-picture-7 release, ordinary F2 SAVE | EN and RU: store equals World records, allocator and ID order equal; cold LOAD through the original door matches the uninterrupted run leaf by leaf on every tick, retires on the same tick, and the target's hit points match on 24 ticks |
| Loss controls in the same test | A SAV with the record dropped, and a SAV with `actionsegments` changed by one, each fail the continuation check |
| `TestKitProjectileBuild` (authentic record) | unchanged, passes |

## Open debt

- Leaves the claims leave open are authored: DIV-1781 (direction), DIV-1782
  (start offset layout), DIV-1784 (counter reset, target identity), DIV-1786
  (smoke trail). DIV-1785 and DIV-1783 hold the target-removal, shooter-death,
  structure, picture 13 and above, client arm and siege rider Unknowns.
- No original SAV holds an arrow record, so the constructed leaves are checked
  against the claims only. DIV-1721 stays open for an owner resave of an
  original SAV taken with an arrow in flight toward a living target.
- The in-reach release gate is unchanged and still authored (DIV-1453).
