# Projectile in flight: M10 acceptance kit

Two SAV files carry a projectile record in flight, for the owner to load in the
original game and resave. The record is the corpus's one authentic projectile,
written back by the ordinary SAVE path with all sixteen leaves unchanged.

## Intent

The earlier combat kit held 0 projectile records. DIV-1185 names original-runtime
acceptance of projectile records as open. The owner needs a file whose SAVE
instant has a projectile in flight, on both editions.

## Authority

`SAV-PROJSTORE-428` and `SAV-PROJLOAD-429` define the sixteen leaves, the
allocator and the ordered IDs. `SAV-PROJCORP-430` identifies the one authentic
nonempty store. No claim gives the leaves a fresh arrow would carry.

## As built

- A native ranged swing releases a presentation shot (DIV-1453). The World holds
  no projectile record for it. `TestReleaseFreshShotWritesNoProjectileRecord`:
  a party archer shoots an enemy archer in mission 30; with one shot in flight
  the World has 0 records and the written SAV holds the canonical empty store.
  DIV-1721 records this gap. No fresh record is constructed, so no constructor
  value is invented.
- `TestKitProjectileBuild` loads the corpus file `2026-08-15/game0018.sav`
  (SHA-256 `1e2eb21f...25eb6b`, mission 40, one arrow, picture 10, runtime ID
  266, target runtime ID 157, a late-dead actor) through the original LOAD door
  on each edition. It saves through the F2 dialog and asserts the written
  `Projectiles` store equals the source store. It writes only into
  `AGAINROM_OWNER_KIT_OUT` and skips without it.
- The same EN-origin state is loaded by the RU engine for the RU file; the two
  files differ in label encoding and in nothing the receiver prints.
- The receiver `TestKitOwnerReceiveM10` now prints each projectile's leaves and a
  per-tick trace, and the tick on which the record retires.

## Proof

| Input / instrument | Observed result on EN and RU |
|---|---|
| Cold LOAD of each kit file, one process per file | 1 projectile record, 1 driver; Prj266 at 19278,27985, actionphase 2 of 3 segments, aim 19840,28288; 6 Sacks, 3 bodies |
| Same process, per-tick | tick 1 at 19465,28086; tick 2 at 19652,28187; tick 3 at 19840,28288; record retired on tick 4; 0 records after 96 ticks |
| Target at landing | native corpse, HP -57, not alive, at cell 77,110, unchanged |
| `savtool verify` | both files re-emit byte for byte |

## Open debt

- Original behaviour of an arrow at arrival on a living target, and whether the
  original defers damage to arrival, are Unknown. The authentic sample's target
  is already dead, so the kit shows no hit or miss and no HP change.
- A native projectile registry at release would hash every ranged fight and
  needs evidence for the fresh leaves (DIV-1721 revisit condition).
