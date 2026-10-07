# Story 1093 verification

## Observable result

The development mapedit diagnostic now shows a Triggers filter, readable
comparisons/actions and typed links instead of only raw arrays. EN M10 Trigger2
at 800x600 exposes C3 <= C12, C7 == C1 and the empty third pair before node
arguments. RU M10 Trigger2 at 640x480 focuses the Group2 placement at cell36,51
while retaining Trigger2, its list context and late action details. The lane
inspected these CPU-composed frames, which use the actual uploaded rail and
shared marker primitives, not GPU readback. Projected shadows are omitted.

Untracked artifacts are under the seat's `review/story1093/`:
`lane-en-m10-trg2-summary.png` and `lane-ru-m10-trg2-group2.png`. The latter uses
`-focus-reference 3 -scroll-details 20`; the earlier
`lane-ru-m10-trg2-focused.png` uses reference2 (Group1), not Group2. These are
development binaries with VCS stamping disabled, not final-release artifacts.
The seat regenerates exact-candidate/merge proof and updates `builds/current/`.
No native window was launched and no desktop input was sent.

## Tests

`go test -trimpath -count=1 ./...` passed once for initial candidate93d9990d with
`GOFLAGS=-buildvcs=false` and the seat's shared Go cache. Focused editing tests
covered `MapEditor1092`, `MapInspection1092`, `MapEditor1093`,
`MapInspection1093`, command arguments and the gated-test manifest.

`TestReleaseMapInspection1093RawTriggerPopulation` passed separately against
both preserved roots. The expected values come from a separate raw section
walker, not Script(), the binder or the display adapter. It compares every
decoded node ID/opcode/value/tag and every comparison/action slot.

| Input | Maps | Actions | Conditions | Triggers |
|---|---:|---:|---:|---:|
| EN installed catalogue | 38 | 797 | 680 | 421 |
| RU installed catalogue | 34 | 792 | 678 | 418 |
| M10, each root | 1 | 28 | 16 | 13 |

Both M10 section-7 payloads have 37428 bytes. Trigger1 with a zero first left
condition stays inspectable. RU's section-7-absent map remains distinguishable
from a valid empty script and malformed section7. The new release test is in
`internal/gatedtests/testdata/population.txt`; the seat runs the final paired
release invocation rather than repeating that broad chain on the branch.

Synthetic raw fixtures cover full-width IDs, sparse/repeated references,
duplicate action/unit/structure IDs, group membership, missing references,
one-based player slots, hero/static-unit bands, nonzero tag-zero data, unknown
opcodes/types/comparators, half-empty pairs, malformed counts/tails and exact
source-byte retention. UI tests drive the actual eighth-filter Pointer route,
fractional detail wheel, independently drawn late-link rail pixels and focus
without selection/scroll loss. A separate long-label test clicks a wrapped
continuation after resizing 800x600 to640x480. Giant regions use six clipped
primitives rather than per-cell expansion. Existing complete-map Fit and
ordinary Viewer zoom regressions remain green.

## Runtime census and gates

The development missionrun binary loaded EN M10 and M20 with `-trace -ticks 1`:
UNSUPPORTED counts are 0 and 0. Its runtime populations are 16 checks /27
instants /12 triggers and 14 checks /15 instants /11 triggers, matching
`pipeline/milestone-baseline.txt`. The baseline carries no UNSUPPORTED rows.
This slice does not change runtime script execution; the observable result is
the inspector, not a falling runtime census. Both runs exited0, undecided at
tick64.

`gofmt` and `git diff --check` passed. Allocation sweeps bracketed DIV-626 and
reported `missing answers: 0`, floor DIV-634. The divergence claim check parsed
277 live rows with nine cells each; its68 existing partial-retraction matches
do not include DIV-626. No research pin, sim, byte-form or source asset changed.

The staged tree asset guard printed `check-no-game-assets: clean (tree scan)`.
After diagnostic generation, the preserved-install guard printed
`ok — 181 file(s), both roots as recorded` (name/size manifest, not hashes).

## Single correction pass

The sole review returned93d9990d with two P2 findings. Its immutable probes
first reproduced both failures, then passed after the bounded correction.
`FocusReference` now bounds the four projected footprint corners of every
target and each anchor cell's four corners. The same corner projection draws
the marker outline. Work remains constant per target; rendering still emits
six lines per target. Screen-pixel padding can lower this editor camera's zoom
floor for a whole-map target; Fit restores its existing scale, and ordinary
Viewer defaults are unchanged.

The review's two previously hidden anchors now project to
`(147.27071823204417,418.6077348066299)` and
`(172.72928176795577,393.1491712707182)` inside320x480 at
zoom0.7955801104972375, instead of Y528 and Y496. Production regressions use
independent raw signed heights for non-diagonal group members, multi-cell
footprints and a large clipped whole-map target. They assert all outline
endpoints, ground anchors, source bytes and selection/filter/scroll retention.

The X300 check-3 fixture keeps `Cell 300,12 (Par1/2)` and all raw arguments,
but prints `Distance region withheld: byte-masked coordinates; authored point
only` with no semantic square. Tests also cover wide Y, a narrow center on a
wide map, oversized radius, a normal square and the byte-coordinate edge.
This is a conservative inspection boundary, not runtime emulation.

Commands used the shared cache and `GOFLAGS=-buildvcs=false`:

```
go test -trimpath -count=1 ./pkg/ui ./pkg/game ./cmd/mapedit -run '^Test(MapEditor109[23]|MapInspection109[23])' -v
go test -trimpath -count=1 -overlay <seat>/review/reviewer-1093/overlay.json ./pkg/ui ./pkg/game -run '^TestReview1093' -v
go test -trimpath -count=1 -overlay <seat>/review/story1093/seat-overlay.json ./pkg/game -run '^TestSeat1093' -v
```

The focused packages and both unchanged reviewer probes printed PASS. The last
command ran once per EN/RU root via `SEAT1093_ASSETS`; both printed PASS and
confirmed Group2 `(36,51)`, Unit133 `(71,112)`, Structure14 `(64,30)`, raw Deploy,
empty-pair comparator, external-reference exclusions and source retention.
Review report/probes were not edited. The initial broad Go/census/image evidence
above was not rerun or relabelled as corrected-candidate proof. The seat owns
final full Go, paired release and exact merged-binary diagnostics. No GUI was
launched and no install, research pin, simulation or byte-form changed.
