# Verification

## Observed player result

The controlled installed-data App drive changes mission-10 structure 4 from
HP 1000 to 674, 359 and 0 after three Fire Ball casts on both EN and RU. Its
nine live draw entries become nine ruin entries; the production HP card changes
pixels and reads 0/1000. Native SaveStore before/windup/ruin checkpoints preserve
bytes, hashes and twelve next-tick reports. No install bytes are modified or
tracked. The lawful roots are read-only inputs.

## Focused checks

- `go test ./pkg/sim ./pkg/ui` with the structure, blast, area, continuation,
  constructed-pair, legacy-shape and headless-spell test selectors: PASS.
- `go test ./pkg/game -run '^TestReleaseMultiCellStructureAreaSpell$' -count=1 -v`
  with `AGAINROM_ASSETS` set separately to preserved EN and RU: PASS, the same
  HP sequence and 9/9 ruin entries on each root.
- `gofmt` and `git diff --check`: clean.
- Reconciled asset-free `go test ./pkg/sim ./pkg/ui ./pkg/game ./pkg/mapload
  ./internal/gatedtests -count=1`: all five packages PASS.
- Reconciled EN/RU `go test -trimpath -count=1 ./pkg/game` selecting
  `TestReleaseMultiCellStructureAreaSpell`, `TestReleaseStructurePhysicalAttack`
  and `TestReleaseMission101FireBallDestroysAndDrawsTheShippedSwitch`: PASS on
  both roots. The original single-cell migration witness remains green.
- `scripts/check-no-game-assets.sh`: `clean (tree scan)`.
- `check-div-claims.sh`: 271 live rows, 375 cited IDs, nine cells per row;
  66 existing partial-retraction matches, no malformed rows. New rows 552/553
  are not matches; closed 458 cites the promoted registration/damage claims.
- Seat allocation sweep before and after the divergence edits: PASS, 32
  namespaces, zero missing answers, divergence floor 558. Only 552 and 553
  are used from the reserved range; 554 through 557 remain unused.

The ordinary Go cache was unreadable for some dependencies. Successful checks
use `GOCACHE=<seat>/review/go-cache`.

## Mission census

An exact-worktree diagnostic `missionrun` build with VCS stamping disabled ran
`-mission 10 -trace -ticks 1` and `-mission 20 -trace -ticks 1` against EN:
`UNSUPPORTED=0` and `UNSUPPORTED=0`. It reports `16 checks, 27 instants, 12
triggers` and `14 checks, 15 instants, 11 triggers`, respectively. Those match
the master inventories in `pipeline/milestone-baseline.txt`; that file carries
inventories rather than literal unsupported-node counters. No script-population
reduction is claimed. The observable change is the installed structure drive
above. This diagnostic binary is not a release build.

## Landing boundary

Initial base: `bf4fe0600352dddaa46fdd8b88b9a84e1848e27c`.
Research pin: `26c755b2be9513b9ec90edba6527db4fdd613dc4`.
Reconciled accepted/published physical-attack master:
`d96012454eddcd015d2f5e4de662c72b7bb6307c`.

The three additive conflicts retain both closed rows (457/458), both installed
tests and both constructor operations (slot reconstruction and physical-target
validation). All four live policy rows 546/547/552/553 survive. The 1085
spell-ruin regression now supplies a positive spread (150..151), because its
former fixed pair correctly does zero Building damage under the new resolver.
An additional multi-cell lethal-ring/pursuit test compares native reports and
hashes for fifty ticks. No physical-attack fix is duplicated.

The seat requested focused candidate checks above and owns the merge-final
`go test -trimpath -count=1 ./...`, paired release/scenario/milestone gates,
one fresh adversarial review, master push and current build. Those are not
claimed as completed by this lane. The exact branch SHA is reported with its
verified remote SHA rather than self-embedded in a changing tracked file.
