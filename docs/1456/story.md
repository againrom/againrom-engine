# One session residue carrier descriptor

## Result

Every session residue carrier is one row of `residueCarriers`
(`pkg/game/residuecarrier.go`). The row names the carrier's capture,
validation, actor binding check, restore and SAV key; its row codec and bounds
stay in the functions the row names. The sites that handled carriers by hand
now iterate the table. A new carrier is one row. No player-visible change: SAV
bytes, cold LOAD, the next tick and World hashes are unchanged on the measured
saves and census.

## Intent

A carrier used to be threaded by hand through capture, snapshot restore, the
SAV field list, decode validation, actor binding loops, projection and SAV
restore. A missed binding loop or restore admitted or dropped a row without a
refusal.

## Authority

Owner decision on architecture audit `8dff7769`, rank 5 (detail: audit
`94ab14f3` row 5): one session residue carrier descriptor before the next
carrier. Owner rule "one builder per kind". No ROM1 claim is involved; the
change is internal to this engine's SAV supplement.

## As built

A row (`residueCarrier`) holds:

- `fields`: the `SnapshotResidue` fields the carrier owns.
- `capture`: reads the carrier out of the running map.
- `validate`: checks the decoded SAV rows against the carrier's bounds.
- `actors`: the actor IDs its SAV rows name, the decode refusal of an unbound
  actor (`unbound`, nil where the carrier's restore resolves it), and `early`
  for the carrier the projection binds with the action graph.
- `restore`: `snapshot` from an in-memory `SnapshotResidue`, `saved` from the
  decoded rows of a cold LOAD.
- `sav`: the `currentActionData` fields it writes and the projection that
  writes them, or `unsaved` with the reason.

A part a carrier lacks is a named empty part (`residueNotCaptured`,
`residueNoBounds`, `residueNoActors`, `residueNotRestored`,
`residueNotLoaded`), so every row states all five.

Sites that iterate the table:

| site | function |
|---|---|
| capture | `mapWorld.residue` |
| snapshot restore | `mapWorld.applyResidue` |
| SAV write | `writeResidueCarriers` in `projectCurrentActions` |
| validation | `validateResidueCarriers` in `readCurrentActions` and `projectCurrentActions` |
| actor binding check | `checkResidueActorBindings` in `readCurrentActions`; `bindResidueActors` in `projectCurrentActions` |
| SAV restore | `loadResidueCarriers` in `restoreOriginalActionView` |
| mission-state test | `SnapshotResidue.hasMissionState` reads every field by reflection |

### Descriptor table

| carrier | residue fields | SAV key | row codec |
|---|---|---|---|
| spell bolts | `SpellBolts` | `Bolts` | `SnapshotSpellBolt`; read and dropped |
| heal bursts | `HealBursts` | `Heals` | `SnapshotHealBurst` |
| cast runs | `CastRuns` | `Runs` | `SnapshotCastRun` |
| target marks | `TargetMarks` | `TargetMarks` | `SnapshotTargetMark` |
| attack clocks | `Swing`, `Phase` | `Animation` | `currentAnimationClock` |
| death ages | `deathAges` | `DeathAges` | `currentDeathAge` |
| visual identities | `VisualIdentities`, `VisualNext` | `VisualIdentities`, `VisualNext` | `SnapshotVisualIdentity` |
| pending commands | `PendingGameOptions`, `PendingQueue`, `Commanded` | `Options`, `Pending` | `PendingGameOption`, `currentPendingQueue` |
| group tag | `GroupTag` | `GroupTag` | `uint32` |
| fog | `FogCols`, `FogRows`, `FogExplored`, `FogVisible` | `Fog` (second game only) | `currentMissionFog` |
| mission notices | `PendingMessages` | `PendingMessages` | `int32` |
| departed characters | `DepartedCharacters` | `DepartedCharacters` | `uint32` |
| mission loss | `MissionLost` | none | none |

The table order keeps every existing order a byte or a restore depends on:
cast runs before attack clocks (the clocks override a run's swing), the late
actor bindings in their written order (runs, marks, clocks, death ages,
visual identities), the departed characters bound early, and the mission loss
after the departed characters.

`deathAges` moved from `Snapshot` into `SnapshotResidue`, so its capture and
its binding sit in the table. A carrier with no field of its own can use the
keyed store: `SnapshotResidue.keyed` in memory and `currentActionData.Residue`
(JSON key `Residue`, omitted when empty) in SAV, under the key
`Residue.<name>`. No shipped carrier uses it, so no written SAV carries it;
LOAD refuses a keyed entry no row owns.

### Mirror types

None removed. Each kept type is the row codec of its SAV key, written as JSON
with its field names: `SnapshotHealBurst` (beside `healBurst`, whose fields are
unexported), `SnapshotTargetMark` (a packed kind and count beside
`markElement`, which also holds the target identity rebuilt on restore),
`SnapshotCastRun` (joins `castRun` with the actor's swing and phase),
`SnapshotVisualIdentity` and `SnapshotSpellBolt` (the read-only old form).
Removing one changes SAV bytes.

### Behaviour kept

The SAV restore and snapshot restore run in table order. Relative to the old
order, the departed characters, the pending commands and the attack clocks
restore later; each writes state no other carrier's restore reads. Validation
errors are the same messages; when a document breaks two carriers at once,
which message is reported may differ. The projection now also checks the
pending options, notice queue and visibility bounds it already wrote; a state
outside them could not be loaded.

## Proof

- `TestEveryResidueFieldHasOneCompleteCarrier`: every `SnapshotResidue` field
  has exactly one row, every row has its five parts, every SAV key is one
  `currentActionData` field; the same check fails a table missing a row and a
  row missing each part.
- `TestResidueMissionStateCoversEveryField`: every residue field, set alone,
  is mission state.
- `TestASyntheticResidueCarrierIsOneRow`: a test-only row, registered with no
  other edit, is captured, written under `Residue.probe`, bound by the
  projection (an actor outside the World gets a missing binding), validated,
  bind-checked, restored from a cold LOAD and from a snapshot; decode refuses
  its invalid value, its unbound actor and an unowned key.
- `TestResidueCarriersRefuseAnUnboundActor`: target marks, attack clocks,
  death ages and departed characters refuse an unbound actor at decode; cast
  runs and visual identities leave it to their restore.
- `TestResidueCarrierLossControls`: the same table without the target marks'
  binding check admits an unbound mark, and without the cast runs' restore a
  cold LOAD restores no cast run.
- Bytes: 15 engine-written mission saves (heal bursts, cast runs, target
  marks, old spell bolts, departed characters, second-game visibility planes)
  cold loaded, saved, saved again, ticked, and the new SAV cold loaded and
  ticked, on EN, RU, ROM2 EN and ROM2 RU. Base `88d43fab` and this head give
  identical SAV SHA-256 and World hashes on every line. One further save is
  refused on both by its mod set.
- Census: every mission from a fresh start, 1500 ticks, World hash every 50
  ticks, base `88d43fab` against this head: identical on all four roots.

## Open debt

- `campaignsecond.go` `validateAuthored` still lists, by hand, the SAV keys a
  second-game town save leaves empty, residue keys among them.
- The movement half of the `Animation` key comes from `Snapshot.mapMotion`,
  captured by the second game's campaign rules, not by a row.
- The snapshot restore restores neither death ages nor mission notices; the
  rows state it with empty parts, as before.
- `ReserveResidueEntityIDs`, used only by tests, still lists four carriers.
