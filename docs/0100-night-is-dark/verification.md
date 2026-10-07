# 0100 — night is dark: verification

Branch `story/0100-night-is-dark` off master `419ce54`. Research pin `e6f9ee6`, verified as no
leading character on `git submodule status`. Two trailered commits, `8fd7cd7` (T1) and `a559b8a`
(T2), and one untrailered docs commit.

## The gate

Run from the worktree root on a clean tree.

| Command | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `gofmt -l $(git ls-files '*.go')` | printed nothing |
| `go test -trimpath -count=1 ./...` | every package `ok` |
| `scripts/check-no-game-assets.sh` | exit 0 |
| `scripts/check-doc-budget.sh` | exit 0, no artifact over its ceiling and no declared overrun added |
| `scripts/check-sdd-audit.sh` | exit 0, FAIL set empty |

`git diff --diff-filter=D --name-only 419ce54 HEAD` is **empty**: this story deletes no file.
**SC-1, SC-2.**

## The schedule, checked against a property the code does not state

**AC-4, P-2, SC-3.** `TestSkyLightIsContinuousOverADay` sweeps all 1440 minutes: the largest step in
any of R, G, B, ambient and range between one minute and the next is **1**, every one of the five
stays within **[0, 48]**, and the joins night to dawn 1 and day to dusk 1 are **exact**. Recomputed
independently from the orchestrator seat against the same package, same answers.

It is a real discriminator and not a restatement, established by reverting the thing it is supposed
to catch. Three mutations, each a single edit, each restored afterwards:

| Mutation | Continuity test |
|---|---|
| `ramp` divides by 60 instead of 120 | FAIL (with the arm test and the sprite-row test) |
| dawn 1's blue ramp coefficient 48 to 47, one digit | **FAIL** |
| dusk 1's ambient `14 + q(10)` to `14 - q(10)`, one sign | FAIL |

A one-digit slip in one of thirty-five constants breaks it. That is the whole reason the property is
in the suite.

**AC-1.** `TestSkyLightArmsAtFirstAndLastMinute` pins all seven arms at their first and last minute
against literals. Read back independently: day opens and closes `[0 0 0 14 32]`; dusk 1 closes
`[23 0 7 23 21]`; dusk 2 opens `[24 0 8 24 20]` and closes `[1 11 47 31 9]`; night is `[0 12 48 32 8]`
throughout; dawn 1 closes `[23 12 1 29 19]`; dawn 2 opens `[24 12 0 28 20]` and closes `[1 1 0 15 31]`.
Every one of those was derived by hand from the schedule before the code existed and agrees.

**AC-2, P-1.** `TestSkyLightCycleOffIsConstant` at minutes 0, 1, 359, 360, 719, 720, 1439, 1440 and
1000000. The function is total over `uint64`: the reduction modulo 1440 is taken first and 60 and
120 both divide 1440, so a raw minute and a reduced one select the same band and the same phase.

**AC-3.** `TestSkyLightDayEqualsCycleOffEqualsDefaultDaytime`. The day arm and the cycle-off arm carry
the same five values and the same shroud pair, and both agree with `DefaultDaytime` — which has
carried 14, 32, (0,0,0) since 0007 by an entirely different route. Two readings, one answer, and
`SunAt` no longer copies the constant, so it is an agreement rather than an identity.

## The tint, and what it can and cannot do

**AC-6, FR-8.** `TestSkyLightTintNeverDarkens` compares, over every minute of a day, every 8-bit
channel value and every level, the shaded output with the band's tint against the output with a zero
tint. Zero counterexamples — recomputed from this seat over the full 1440 x 256 x 96 x 3 space, also
zero. Witnessed by reverting: making `ShadeChannel` **subtract** the tint fails this test,
`TestTintChannelIsShadeChannelAtItsIdentityLevel` and `TestShadeTintBeforeMultiply`.

`TestShadeTintBeforeMultiply` carries the other half — that the tint is inside the multiply, not
added after it. Moving the addition after the multiply fails that test alone, which is the correct
resolution: it is a different property and a different test catches it.

## The level a pixel actually gets

**AC-5, DD-7, SC-6.** `TestSkyLightGroundLevelAtNoonAndMidnight` runs a flat synthetic height field
through `LevelGrid` from `SunAt`'s own light. Level **46** at noon (minute 360) and **64** at
midnight (minute 1080); `ShadeScale` of those is **1.5625** and **1.0**. Night ground is 64% of
noon's brightness, asserted on the multiplier a pixel takes and not on the ambient byte.

**AC-7.** `TestSkyLightSpriteRow`: row 3 at noon, 8 at midnight, and over a day exactly the set
{3, 4, 5, 6, 7, 8}.

End to end, through the shipped composition, a mid-grey (120,120,120):

| in-game time | ground level | ground pixel | sprite row | sprite pixel |
|---|---|---|---|---|
| noon | 46 | (187,187,187) | 3 | (195,195,195) |
| 19:00 | 51 | (185,168,174) | 4 | (198,180,186) |
| midnight | 64 | (120,132,168) | 8 | (120,132,168) |
| 04:00 | 60 | (162,148,135) | 7 | (162,148,135) |

The ground and the sprite land on the **same triple** at midnight, because level 64's multiplier and
row 8's gain are both exactly 1.0. Those are two separately decoded ladders and nothing joins them
in the code, so their meeting is a check.

## The tint reaching the screen

**AC-8.** `TestGroundCacheKeyDiscriminatesOnTintAlone` and `TestSpriteCacheKeyDiscriminatesOnTintAlone`:
equal lights give equal keys, a tint-only difference gives different keys, on both paths. This is
what 0092 DD-3a deferred — the sprite cache baked the tint in without keying on it, so a varying
tint would have served the first band's textures for the whole session.

**AC-9.** `TestCellPixelsAddsTheTintSaturatingAt255` and
`TestCellPixelsZeroTintIsByteIdenticalToTheSource`. The tinted texture differs from the untinted one
by exactly the tint per channel, saturating at 255, and under a zero tint — day, and the cycle off —
it is byte-identical to what the tree uploaded before this story.

**AC-10, FR-9.** `TestGroundTextureUnshadedIsUntintedAndCornersAreOne`. Under `-unshaded` the texture
is the untinted source and every corner multiplier is 1. `TestGroundPlaceholderKeyIgnoresTint` pins
the placeholder fill as untinted at the key, so the one shared diagnostic image is not split across
bands that draw it identically.

**P-4.** One sun. `v.sun` is written only by `relight()` and both caches read the tint through the
single `lightTint()` method; nothing names a tint or a row of its own. Checked by grep: `SkyTint`
has no reader in `pkg/ui` outside `lightTint`, and `ShroudObject`/`ShroudUnit` have no reader
anywhere outside `skylight.go` and its test — **FR-5's "nothing reads them" is a fact about the tree,
not a discipline.**

**P-3.** A relight is idempotent: `relight()` recomputes the sun, the grid and (through the keys)
the same cache entries from the clock and the switch alone, so a second call at the same clock
reaches the same state. Unchanged from 0092, which contracted it; nothing in this story adds a
write outside `relight()`.

## The regression evidence, on both lawful roots

**SC-4.** `againrom -check`:

- **ru** — `34 map rows, 8 of 8 buttons have a mask region; hero Body 43, Reaction 26, Mind 15,
  Spirit 15, Blade 10, Iron Short Sword 10-16, to-hit 49, defence 8`
- **en** — the same line with `38 map rows`.

Identical to the same command on master before this story, captured first.

**AC-11, SC-5.** `missionrun -mission 10 -census -waypoint u21:56:21:3 -waypoint p0:66:16:3`, both
roots, identical to the pre-story baseline down to the step counts:

    waypoint 1  u21 -> (56,21) r3 : was STOPPED BY THE WORLD DECIDING, short of (39,41),
                Chebyshev 20, after 272 ticks
    outcome lost at tick 272
      moved  u21    slot 2 group 2  (36,51) -> (39,41)  12 step(s), -3 hp
      moved  u32    slot 4 group 7  (48,44) -> (40,42)  8 step(s), 10 hp
    census: 2 of 36 unit(s) moved, 1 fell, over 272 tick(s)

**FR-11** holds structurally: `git show --stat` over both trailered commits touches no path under
`pkg/sim`, no byte form and no digest.

## The build

**SC-7.** `builds/0100-night-is-dark/` holds `againrom.exe`, `mapview.exe` and `missionrun.exe` with
a README. Every command in that README was run from that directory before it was written down,
including the `AGAINROM_ASSETS`-only form with no `-assets` flag. It names `N` for the cycle's
switch and `F3` for the one-hour lighting step, which is how a day is walked in twenty-four presses
rather than in twenty-four minutes.

## What this story did not do

The shadow and silhouette pass. `Light` carries the two shroud indices on the band's schedule (4/2
by day and with the cycle off, 6/3 in both twilights, 8/4 at night) and **nothing reads them** —
that is FR-5's whole content, and it is there so the shadow story inherits the schedule instead of
re-reading it.

**D-2 stands as written.** The ground's tint rides an 8-bit texture and so clamps at 255 before the
vertex multiply, where the engine clamps only after it. The two agree at every level up to 64, which
is all flat ground at every band; above it the divergence is at most 2 of 255 and falls with the
multiplier. Not measured against the original, because the original is not run.

**One gate is wider than FR-9 asks and it is deliberate.** `lightTint()` returns zero whenever
`Lit()` is false, which includes a viewer whose altitude grid was never usable — a case FR-9 does not
name. Such a map draws its sprites at the schedule's own row while adding no colour, matching the
all-1 corner scale its ground already gets. The alternative reading (gate the tint on `unshaded`
alone, as `spriteRow` does) would tint sprites over a ground that is not lit at all. Neither is
decoded; the chosen one is stated here rather than left to be discovered.
