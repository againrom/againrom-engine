# DIV-044 ground picker closure

## Result

Hotfix `caf05579` replaces the mean-of-four placement rectangle in the
displaced ground picker with ROM1's per-column corner-mesh bounds. Corrective
hotfix `97e94be7` routes the cursor readout and its step-cost question through
that picker after fresh adversarial pass 1 found their flat-camera bypass. The
camera, terrain pass and every ground-cell consumer remain on their existing
seams. No canonical state, digest field, serialization field or form version
was added.

The contract was committed before production code at `ca47d204`. The touched
domain is **Client**. The import graph is unchanged: `pkg/ui` continues to read
`pkg/render/terrain`, and no lower tier imports the client.

## As-built surface

`Projection.CellColumnBounds` reads TL/TR and BL/BR through `WorldCorner`,
evaluates both signed integer lerps at `x&31`, and returns the pair unsorted.
`Viewer.groundCellAt` converts screen to camera world coordinates, floors the
native X column, walks the current terrain-pass rows in ascending order, and
returns the first inclusive match. A collapsed span can match its one row; an
inverted span cannot match. Flat mode retains `Camera.ScreenToCell`.

The one result feeds hover fog and sack bits; move, patrol, swarm, pick-up and
empty-ground cast targets; inventory ground drops; the cursor-cell readout; and
the selected unit's step-cost question; and the existing App-to-`mapWorld`
command queue. Entity and minimap picking remain separate surfaces.

## Twelve-aspect closure

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | Signed altitude bytes reach the existing projection; all four corner orientations and the far-edge clamp have literal expected-value tests. No data schema changes. |
| Runtime state | PASS | Pan, zoom, fractional camera offsets, viewport bounds and non-finite coordinates are covered. The picker stores no state. |
| Simulation | N-A | Movement and command execution are unchanged. A paired real-mission drive shows that queuing the UI target does not change the canonical digest before advance. |
| Input | PASS | A tap on the synthetic old/new discriminator queues the exact mesh-selected cell. Shared seams and exclusive viewport edges are covered. |
| AI | N-A | No AI producer or consumer reads ground picking. |
| UI/HUD | PASS | Hover fog/sack lookup, the pick-up cursor, the cursor-cell readout, its step-cost target and the map viewport gate consume the same cell. The readout expectation is literal rather than derived from either picker. |
| Triggers/scripts | N-A | No trigger or script path reads the client picker. |
| Inventory/equipment | PASS | Ground-drop input receives the same mesh-selected cell; drop legality and equipment rules are unchanged. |
| Persistence | N-A | No save field, binary record or version changes. |
| Campaign/session | N-A | No campaign or session rule changes. |
| Shipped content | PASS | EN and RU mission 10 each select frame point `(91,64)`, world point `(219,1818)`, corner cell `(6,57)`; the replaced mean picker selects `(6,56)`. |
| Interactions with existing mechanics | PASS | Hover, click order, ground drop, readout step-cost target and App command-queue integration agree. Pending input remains outside the world digest. |

There is no known in-scope GAP.

## Independent expectations and mutation

The terrain tests state literal bounds for flat, rising, falling, crossed,
collapsed and inverted corner pairs at local columns 0, 1, 16, 17 and 31,
including negative quotients and the next-cell boundary. The UI tests state
literal cell answers for top/bottom bounds, horizontal and vertical seams,
steep slopes, map edges, camera transforms, hover, click, drop and readout.
The readout test also records the selected id and exact cell sent to the
step-cost seam. None calls the production interpolation helper to form its
expected value.

The installed test carries a separate raw-altitude oracle and must first find a
point where the two algorithms disagree. It then drives the production App and
observes the queued `KindGroupMoveTo` target. It runs under the release gate on
both lawful roots.

A temporary mutation restoring `cellScreenRect` and last-match averaging failed
the ground picker population with exit 1. It returned `(0,1)` instead of
`(0,0)` at synthetic point `(1,80)`, lost the sack/pick-up hover, queued and
dropped to row 1, admitted an inverted span, and returned installed mission-10
cell `(6,56)` instead of `(6,57)`. Restoring the exact picker returned the same
selection to green.

A second mutation changed only the corrected readout call back to
`Camera.ScreenToCell`. At `(1,80)` it stated no cursor cell, asked no step-cost
question and produced no step answer; all four independent assertions failed.
Restoring `groundCellAt` returned the test to green. The hover fixture also
makes mesh row 0 visible and mean-selected row 1 unseen, so it now witnesses
the previously blind `fogGateSack` leg.

## Research and divergence reconciliation

The implementation consumes active High claim `TERR-GEOM-036` part (d) at the
pinned research commit `d7ee0c62cfa4a16083d356f24b1870035e1f0209`. It does not
claim that the picker and terrain rasterizer share an interpolation model; the
claim explicitly distinguishes them.

The `DIV-044` revisit condition is met. Its closed row includes the readout
correction, and the original `7036a44` hotfix debt is marked folded in
`docs/hotfix/LEDGER.md`. Fresh pass 1 classified the uniform-fog witness as W:
production was correct, but the test could not distinguish which row its fog
gate read. `DIV-401` records that debt and its same-lane closure after the two
candidate rows were given different visibility. No behavioural mismatch was
introduced.

## Verification

All results below were measured from the isolated hotfix tree after corrective
production commit `97e94be7`; later commits in the candidate are documentation
only.

| Check | Result |
|---|---|
| Implementation build, vet and formatting | PASS; `go build ./...`, `go vet ./...`, and `gofmt -l` with no output |
| Implementation full suite | PASS; `go test -trimpath -count=1 ./...` |
| Claim citations | PASS; 1,290 distinct citations resolve against 1,479 claims and 220 experiments |
| Asset exclusion | PASS; clean tree scan and clean full-history scan |
| Release tests, EN and RU | PASS; 49 of 49 selected on each root, zero skipped |
| Headless scenarios, EN and RU | PASS; 15 of 15 on each root |
| Divergence reader | PASS; the edited tree has 235 live rows, every row has nine cells, and `DIV-044` is absent from the live population |
| Research pin | PASS; branch and master both pin `d7ee0c6` |
| Preserved installs | PASS; 162 files across both roots match the record |
| Research build, vet, formatting and full suite | PASS |
| Research integrity scripts | PASS; 1,479 unique claim IDs, 237 overturned IDs marked, 11 regeneration scripts honour `OUT` |

The milestone census is unchanged on both roots. The worktree-built
`missionrun` reports zero `UNSUPPORTED` lines for mission 10 and zero for
mission 20, the same values master carried before this hotfix. The full 28-map
census and unattended drive match `pipeline/milestone-baseline.txt` on both
roots.

The runnable output is under `builds/div-044-ground-picker/`: `againrom.exe`,
`missionrun.exe`, and a run README. No GUI input was sent on the owner's active
desktop; the production App was driven headlessly through the same input
dispatch instead.
