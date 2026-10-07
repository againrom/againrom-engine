# 0101 — shadows: verification

Branch `story/0101-shadows` off master `fa415af`, research pin `e6f9ee6`
(`git submodule status` showed no leading character).

## The gate

```
$ go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go')
$ go test -trimpath -count=1 ./...   # exit 0, 31 packages ok, 0 FAIL
$ bash scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)                                  EXIT=0
$ bash scripts/check-doc-budget.sh                                       EXIT=0
$ bash scripts/check-sdd-audit.sh                                        FAIL set empty for 0101
```

`gofmt -l` printed nothing. The audit's note/warning count is not comparable from a lane and is not
quoted; the FAIL set is, and 0101 contributes none.

## AC-1..AC-7 — the model

```
$ go test -trimpath -count=1 -run Shadow -v ./pkg/render/terrain/
--- PASS: TestShadowSlopeAtDefaultTheta      (AC-1)
--- PASS: TestUnitShadowShiftSweep           (AC-2)
--- PASS: TestUnitShadowPlaceAC3             (AC-3, P-3)
--- PASS: TestBlitShadowAC4                  (AC-4)
--- PASS: TestShadowChannelAC5               (AC-5)
--- PASS: TestStructureShadowShiftAC6        (AC-6)
--- PASS: TestObjectShadowPlaceAC7           (AC-7)
--- PASS: TestShadowMask                     (P-4)
--- PASS: TestShadowLevelPairing             (FR-14)
ok  againrom/pkg/render/terrain
```

`ShadowSlope(DefaultTheta)` came out `0.57735025728078293` against `TERR-LIGHT-113`'s published
`0.57735025728078282` — one unit in the last place, which is the x87 `FPTAN`'s answer against Go's
`math.Tan` and is the whole of AC-1's `1e-15`. The day band's own ends came out `-0.577350` and
`+0.575413`; the claim states `-0.5774 .. +0.5754`. Neither number was used to fit `theta * 2 / 3`
— they are what it reproduces, which is what makes D-2 falsifiable rather than fitted.

## AC-8 — the screen pass

```
$ go test -trimpath -count=1 -run Shadow -v ./pkg/ui/
--- PASS: TestShadowDrawsOrderCountAndGating                     (AC-8, FR-19)
--- PASS: TestShadowDrawsBlendAndAlphaPerKind                    (AC-8, FR-14, FR-17)
--- PASS: TestShadowDrawsShearsObjectsOnlyAndMatchesTheRowLaw    (AC-8, FR-4, FR-18)
--- PASS: TestShadowDrawsSkipsVariableSizeAndFramelessStructures (AC-8, FR-9)
--- PASS: TestShadowDrawsCullsAPlacementTheViewDoesNotReach      (AC-8, FR-20)
--- PASS: TestShadowMaskCacheHoldsOneEntryPerFrameAcrossTwoHours (AC-8, FR-16)
ok  againrom/pkg/ui
```

Four pre-existing `drawArt` order assertions in `structures_test.go` were updated: the pass is
observable in the same recorder, so their sequences legitimately grew. None was weakened — each now
names the shadow draws by their own expected shift.

## AC-9 — nothing moved that should not

Both roots, before and after, byte-identical:

```
$ ./againrom.exe -check
againrom: 34 map rows, ... (RU)          againrom: 38 map rows, ... (EN)
$ ./missionrun.exe -mission 10 -census -waypoint u21:56:21:3 -waypoint p0:66:16:3
outcome lost at tick 272
  moved  u21  slot 2 group 2  (36,51) -> (39,41)  12 step(s), -3 hp
  moved  u32  slot 4 group 7  (48,44) -> (40,42)   8 step(s), 10 hp
census: 2 of 36 unit(s) moved, 1 fell, over 272 tick(s)
```

## P-1, P-2 — the shadow is a drawing, not state

`git diff --name-only fa415af..HEAD` touches no file under `pkg/sim`; no byte-form version was taken
and no digest input changed. `math.Tan` appears once in `shadow.go` and nowhere else in the story;
`Light.Theta` is the only angle any caster reads.

## Six mutations, each reverting one load-bearing line

Only the FAIL set is quoted. Each was applied alone and reverted.

| Reverted to | Killed by |
|---|---|
| `ShadowSlope` returns the cycle-off slope at every hour — a **fixed lean** | `TestUnitShadowShiftSweep` (AC-2), 7 assertions |
| `UnitShadowPlace` adds no X shift | `TestUnitShadowPlaceAC3` |
| one channel of `BlitShadow` written from a constant instead of the destination | `TestBlitShadowAC4` |
| `StructureShadowShift` returns 0 | `TestStructureShadowShiftAC6` |
| `ObjectShadowPlace` anchors on the drawn frame, not frame 0 | `TestObjectShadowPlaceAC7` |
| `drawShadows` runs after `drawPlane` | `TestShadowDrawsOrderCountAndGating` + 3 pre-existing |

**The third one initially survived**, and that is the finding. AC-4's part 1 asserted only that the
two results were *unequal*, which a per-channel substitution passes because the other two channels
still carry the difference. It now asserts the three covered pixels per channel against literals —
at level 8 the law is a halving, so 0 stays 0, 255 goes to 127, and the untouched 128 neighbours go
to 64 — and the mutation kills.

## The pixels — a probe over a real map

`builds/_probe-0101/` (untracked, its own module, `replace againrom => ../..`) composites
`FORESTER.ALM` through `CompositeLit` and runs `BlitShadow` over the object and structure
placements, which is the exact-integer law FR-15 states rather than the screen's blend.

```
$ ./probe.exe -map .../FORESTER.ALM -minute N
minute    0  theta -0.78540  slope -0.57735  objectLevel 4  8604 object + 1566 strip shadows
             13529122 of 67108864 pixels darkened (20.16%), mean red drop 35.3
minute  360  theta +0.00000  slope +0.00000  objectLevel 4   13612663 (20.28%), mean 35.1
minute  719  theta +0.78322  slope +0.57541  objectLevel 4   13462122 (20.06%), mean 35.4
minute 1080  theta +0.00000  slope +0.00000  objectLevel 8   13612663 (20.28%), mean 43.0
$ ./probe.exe ... -noshadow
             0 object shadows, 0 of 67108864 pixels darkened
```

A fifth of the map's pixels move, by 35 of 255 on average. The last row is `TERR-LIGHT-126`'s
schedule arriving at the ground: same geometry, level 4 to 8, mean drop 35.1 to 43.0. Rendered
crops at minutes 0 and 719 against the `-noshadow` control show the same trees casting to the left
and then to the right, which is AC-2 as a picture.

## What was NOT verified

**No shadow was seen through the window.** This machine runs the lane headless: the screen pass is
witnessed against a recording target down to its blend, alpha and transform, and the law is
witnessed on real pixels through the probe, but nothing here put an `*ebiten.Image` on a display.

**D-1's sense is unfalsified, not confirmed.** All three casters lean the same way and the picture
is coherent; whether it is the way the engine leans is what the corpus does not say.

**The unit shadow was not seen over real terrain.** The probe reads a map, not a mission, so it
places no units; the 18-pixel unit translation is witnessed by AC-2, AC-3 and the `pkg/ui` pass
alone.
