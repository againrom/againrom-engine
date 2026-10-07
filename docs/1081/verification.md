# Spell render-pass verification

## Candidate and visible result

The story began at `42da87b3d8973e3736ed08b599d01696801e62aa`, adopted accepted
pin `0d829843102d57aa4778aa4407749a2044fe3938` with master 512eb42b, and
reconciled gameplay master `a05a1b8d70fe09c92a647e4021ab844803472f72` before
final gameplay gates. The reviewed candidate also includes trusted maintenance
master `e31ca935d640832bf1de7114899f0922fa8a4205`. Gameplay content remains
`57dd3595fdcb95820a1ea73e24219ec9157af24d`; the maintenance changes only the
gated-test scanner/tests/manifest and screencensus wording.

Before this slice, retained overlays shared the final all-spell pass after
the body/content band. The production drawArt boundary now submits Fire/Earth,
all shadows, projectiles, Freezing/Poison, then all bodies. Flat decoration and
the existing internal object/structure/sack/body depth merge remain intact.

Both preserved installs produce the same witnessed result: all 441 pixels in
the selected overlapping region agree with the independent raw-palette and
coverage oracle, and all 441 differ from the old final spell band. This holds
for both Freezing and Poison variants. The observable artifacts are
`review/1081/{en,ru}/{freezing,poison}-passes.png` at the seat. The EN Freezing
image was visually inspected.

These are CPU rasterizations of actual production DrawImage submissions from
drawArt, using real installed spell art and synthetic translucent unit markers.
The unit markers deliberately expose lower layers. They are not GPU readbacks,
original-game screenshots, or a live-window witness. No desktop input was sent.
The seat owns the exact-master current-build publication after landing.

## Authority and independent oracles

ANIM-047, read from the accepted pin, is Medium for the five-pass relation and
explicitly Unknown for traversal within the projectile list.
MAGIC-OVERLAYART-051 supplies retained pictures 15, 23, 25, 47, the A/B split,
Fire-before-Earth at one cell, and Freezing's early return before Poison.

The producer regression independently names those picture IDs and passes,
reverses effect input order, checks all four overlapping masks, and repeats
Fire. Before the change it fails on missing tags, reversed walls and both
clouds drawing; afterward it passes. An ordinary object using picture 15 still
belongs to the projectile pass. Existing duplicate-wall tests stay green.

The composition regression submits interleaved tags but expects the literal
eight-call sequence with two full shadow/body sweeps. Its independently
calculated same-cell result is RGBA (51,88,152,255). Before the split, drawArt
submits only the four shadow/body calls and the test fails. Separate checks
retain projectile input order, clear replaced overlays, and prove drawFrame
calls the unified boundary exactly once with no final separate spell pass.

The installed UI witness reads raw .16a palette/coverage for its expected
pixels, independently of EffectFrame.RGBA and the observed production order.
The installed game witness checks the real producer using FrontEnd.Projectiles:
15/47 in A and 23 in B, with duplicate Fire and masked Poison omitted.

## Gates and reproducible witnesses

Focused pass, shadow, object/body-depth, actor-mark, duplicate-wall and gated
population tests pass. gofmt and git diff --check pass. The final-code
`go test -trimpath -count=1 ./...` passes once at 57dd3595. No story commit has
a Co-Authored-By trailer. `check-no-game-assets.sh` prints `clean (tree scan)`.

Run on each EN/RU root:

```text
AGAINROM_ASSETS=<root> go test -trimpath -count=1 ./pkg/ui ./pkg/game -run '^TestRelease(SpellPassesComposeInstalledArtAroundShadowsAndBodies|RetainedOverlayPassesUseInstalledArt)$' -v
```

Both tests pass on both roots and are registered in the gated population.
Optional `AGAINROM_SPELL_PASSES_PNG` names an external review output directory.
The seat must run its paired final release invocation on the merge commit.
No input/session route changes, so no new scenario or broad scenario run is
claimed. No full milestone census is needed for this presentation-only change.

The normal maintenance merge preserves both new witnesses alongside the 96-test
trusted base, for exactly 98 manifest entries. After that merge,
`go test -trimpath -count=1 ./internal/gatedtests ./cmd/screencensus` and the
focused spell-pass/shadow/depth producer and UI tests pass. No gameplay file
differs from published 997c9f7b, so the prior full Go and EN/RU gameplay proof
stands; the full suite was not duplicated for this trusted-base-only update.
The post-merge no-assets and diff checks also pass.

Allocation sweeps bracket the ledger edits and report `missing answers: 0`.
DIV-079 closes; only DIV-528 is consumed, and 529 through 531 are unused.
`check-div-claims.sh` exits 0: pin 0d82984, 263 live rows, 340 cited IDs, and
64 existing partial-retraction matches. Neither changed row is a match.
The preserved-install check prints `ok — 181 file(s), both roots as recorded`.

## Executable checks and remaining limits

againrom and missionrun built from 57dd3595; go version -m reports that exact
revision and vcs.modified=false for both. Command-scoped GIT_DIR/GIT_WORK_TREE
and ephemeral safe.directory entries keep worktree stamping exact; VCS
stamping is never disabled. againrom -assets <root> -check exits 0 on both roots
(EN 66 map rows, RU 62; both 8/8 button regions).

The two required missionrun spot checks use EN assets and
`-mission <N> -trace -ticks 1`. Both exit 0. They remain at the master baseline:

| mission | baseline checks / instants / triggers | master unsupported | candidate unsupported |
|---|---|---|---|
| 10 | 16 / 27 / 12 | 0 | 0 |
| 20 | 14 / 15 / 11 | 0 | 0 |

No simulation, save or hashed-state field changes. DIV-528 retains projectile
traversal, first-seen cell order, non-unit content relative to spell passes,
and the separate Heal/Drain shower's broader order. Fog, relief, frame clocks
and art resource policy are unchanged. This closes the decoded pass relation,
not complete original map-render fidelity. Independent review and landing
remain seat work.
