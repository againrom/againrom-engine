# Projectiles and the Fire_Ball burst from original saves

## Intent and authority

A mission-150 original save with a catapult rock, crossbow bolt or Fire_Ball in flight continues on the original's own ticks, and a Fire_Ball area blast, native or restored, builds the picture 13 burst record that SAVE writes and LOAD continues. Authority: `SAV-1141` to `SAV-1147`, `SAV-1149` to `SAV-1152` and `ANIM-101` to `ANIM-103`, `ANIM-109` to `ANIM-112` in the pinned snapshot. Ledger rows: `DIV-944` and `DIV-1721` narrowed, `DIV-1783` corrected, `DIV-1876` to `DIV-1879` new.

## As built

Burst record (`pkg/sim/burst.go`). A Fire_Ball blast, from the native effect walk or from a retained area effect, builds one `Prj` record: picture `2*spell+9` (13), ActionTarget 0, x, y, ActionX and ActionY at the cell centre, actionphase -1, 22 segments, frame `(actionphase/2) mod 11`. It takes the next id from the shared projectile counter, at the head of its hash bucket, so FreeIndex advances as for a unit shot. A blast released while the effect walk runs is queued and built after the walk; the driver then makes its first call in the same step. The phase count of the burst sheet reaches the World through `SetBurstPhases`, armed by the front end at mission open.

Transport delay (`pkg/sim/spelldelivery.go`). Fire_Ball waits distance/speed with the Euclidean distance (`ANIM-111`); other area spells keep the Chebyshev distance. The extra tick of `SAV-1149` is the queue order of the walk (`DIV-1877`).

Unit shot (`pkg/sim/unitshot.go`). The record insertion is one helper shared with the burst.

Restored wind-up (`pkg/sim/burst.go`, `pkg/game/spellbolt.go`). A wind-up restored from an original save has its phase memory and swing count seeded from the action clock (`World.WindUpElapsed`). Before this, the first swing walk read every restored run as freshly begun and released its shot a second time one tick after LOAD (`game0023`: a second rock, id 29).

Bound identities (`pkg/game/savworldeffects_project.go`). SAVE projects the Document's Projectiles from the World, then sets the bound id list to the live driver rows (`followProjectileDrivers`), so a record bound by a loaded Document whose driver is gone leaves the list with its record. SAVE never refuses for a shot or burst released in a loaded World.

Presentation (`pkg/game/spellbolt.go`). The cast no longer draws a presentation burst; the burst is the World record, drawn from its saved position and frame by the existing record draw. The driver row carries the caster's owner for the draw only, so the caster's own burst is not held back by fog; a burst with no known caster (a retained area, a cold LOAD) draws with owner 0 and follows its cell's fog. A projectile driver with no target (the burst) no longer makes `onlyNativeProjectiles` refuse a native document.

## Proof

- Unit (`pkg/sim/burst_test.go`): the blast builds the picture 13 record with the leaves above; the burst shares the projectile counter with a unit shot; the byte form round-trips it; Fire_Ball waits the Euclidean distance; a retained area blast builds the burst.
- Release, EN, owner saves `game0022` to `game0024` (`TestReleaseOriginalProjectileFlightContinuesToLaterSaves`): each save is loaded through the original LOAD door and ticked to each later save's tick. The Projectiles set (allocator, id list, every record's fifteen leaves) equals that save's own store for the records the load point held and for the burst. The unit shots the loaded World releases are built one tick after the original's (`DIV-1876`) and are compared one tick later. Loss controls: the loaded and burst records one tick early fail the comparison, a changed leaf fails it, and the shot is checked to stay one tick behind.
- Release, EN and RU (`TestReleaseNativeFireBallBurstSurvivesSaveAndLoad`): a mage casts Fire_Ball in mission 41; the burst record is built with 21 segments left after its first driver call; SAVE writes the allocator, id list and record; a cold LOAD continues the record leaf for leaf for its 22 ticks and retires it on the same tick. Loss controls: a SAV with the record dropped and one with a changed leaf fail. No RU save holds a projectile, so the RU witness is native only.
- Release, EN and RU (`TestReleaseLoadedOriginalSaveDuringFireBallBurst`): `game0022` and `game0023` are loaded, ticked 6, 8, 10, 14, 18, 22, 26 or 30 ticks, and saved through the save dialog. The written Projectiles equals the live World's records, and a cold LOAD continues the burst leaf for leaf. Without the bound-id fix the save is refused at the first sampled tick.
- The burst draw is asserted in `TestReleaseNativeFireBallBurstSurvivesSaveAndLoad`: a sheet frame owned by the caster on every tick, with at least three distinct frames over the life.
- Milestone-2 acceptance: `engine/scripts/check-milestone2-acceptance.sh`; the result is in the lane return.

## Open debt

- `DIV-1876`: shots released by a loaded World are one tick late against the original; cause not read.
- `DIV-1877`: the burst's extra tick is placed at the build; walk order and the burst's tile mark and view hash are not read.
- `DIV-1878`: pictures 14 and above build no World record.
- `DIV-1879`: no native siege Fire_Ball SAVE and LOAD witness.
- Native Fire_Ball impact lands up to a few ticks later on diagonals: the target point is the aimed cell centre, not the target actor's fine position (`DIV-1877`).
- Wind-up seeding runs only for an original-load document and was checked by reading and the release tests; no native wind-up save was constructed.
- A Document-less native state with projectile records skips area and spell-graph projection (`projectSavedWorldEffects`); only the fresh-mission witness covers it.
- `DIV-1781` to `DIV-1786` (shot leaves, admission, smoke trail) are unchanged.
