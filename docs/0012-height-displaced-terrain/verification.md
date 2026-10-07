# Verification — height-displaced terrain geometry (ROM1)

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11. Automated evidence was produced with **no
game install visible to the test suite**. The developer-run pass used a lawful GOG install supplied
through `-assets`; **no game byte, map file or rendered image is committed** — every image was
written outside the repository, hashed there and deleted as the run went. The suite is run as
`go test -trimpath -count=1 ./...`, because on this machine Windows Defender quarantines the
un-trimmed `terrain.test.exe` as a false positive: a machine fact, recorded rather than worked
around — no security setting was changed and no package restructured to dodge it.

## Gate results

From the repo root at `bbbb4b1`:

```
$ go build ./...                                    (clean)
$ go vet ./...                                      (clean)
$ gofmt -l $(git ls-files '*.go')                   (empty)
$ go test -trimpath -count=1 ./...
  all 17 packages with tests ok, 0 failures - incl. internal/archtest (the fail-closed import DAG),
  pkg/render/terrain, cmd/terraintool
$ bash scripts/check-no-game-assets.sh              check-no-game-assets: clean (tree scan)
$ bash scripts/check-no-game-assets.sh --history    check-no-game-assets: clean (history scan)
$ bash scripts/check-doc-budget.sh                  every artifact under ceiling; plan <= spec, tasks <= plan
```

The trailer bijection is over the whole story, so it is stated at its end rather than at `bbbb4b1`:

```
$ git log --format='%(trailers:key=SDD-Task,valueonly)' 813bcef..HEAD
  9 IDs, T1..T9 one each - T8 being the commit that adds this file - 0 duplicates,
  0 Co-Authored-By trailers anywhere in the range
```

## Automated criteria

SC-1…SC-14 are each carried by a named test; SC-15 is the developer-run pass below. The `PASS` lines
are the tool's, the trailing columns this record's:

```
$ go test -trimpath -count=1 -v ./pkg/render/terrain/ ./cmd/terraintool/
--- PASS: TestProjectVertices               SC-1   FR-2, FR-9, AC-3
--- PASS: TestStepTable                     SC-2   FR-6, AC-5, R-3, R-5
--- PASS: TestProjectedSeam                 SC-3   FR-6, FR-8, R-5
--- PASS: TestProjectedSelector             SC-4   FR-3, FR-4, AC-4
--- PASS: TestProjectedSpanSampling         SC-5   FR-5, AC-6, P-1
--- PASS: TestProjectedShadingDomain        SC-6   FR-7, AC-6
--- PASS: TestProjectedCollapsedColumn      SC-7   FR-5, AC-7, P-5
--- PASS: TestProjectedOwnership            SC-8   FR-8, AC-8
--- PASS: TestProjectedPixelsInBounds       SC-9   FR-9, P-2
--- PASS: TestProjectedEqualsFlat           SC-10  FR-1, AC-1, P-3
--- PASS: TestProjectedExtremeDeltas        SC-11  FR-6, AC-9, P-6
--- PASS: TestProjectedRejectsBadArguments  SC-12  FR-10, AC-10, P-4, P-6
--- PASS: TestProjectedRejectsBeforeProjecting      T9, DD-9
--- PASS: TestMarkersAtOrigin               SC-13  FR-11, AC-11
--- PASS: TestRenderGeometrySummary         SC-14  FR-12, AC-12
--- PASS: TestRenderFlatSelection           SC-14  FR-1, AC-2
--- PASS: TestRenderMarkersAtProjectedOrigin        FR-11, DD-8
--- PASS: TestProjectedCompositeMatchesRecipe       FR-5, FR-6, end to end
--- PASS: TestProjectedUnshadedMatchesRecipe        DD-4
--- PASS: TestInterpSpan                            DD-7
ok  againrom/pkg/render/terrain
ok  againrom/cmd/terraintool
```

## Developer-run evidence — AC-13, SC-15

Binary: `builds/0012-height-displaced-terrain/terraintool.exe`, built from `bbbb4b1`. Corpus: the 38
shipped maps — 10 loose `.alm` at the install root, 28 inside `scenario.res`, extracted with
`restool extract` outside the repository. Their cells total **880 704**, the figure research records
for the corpus, so the 38 swept here are the 38.

```
$ terraintool render -assets "$INSTALL" -map "$INSTALL/Kids.alm" -out "$OUT/Kids.png"
terrain: 80x80 cells (6400), 2560x2571 px at scale 1, tile slots 52/128, placeholder cells 0, shaded (theta=0.7854 ambient=14 range=32), geometry projected, y origin -69
$ terraintool render -assets "$INSTALL" -map "$INSTALL/Kids.alm" -out "$OUT/Kids-flat.png" -flat
terrain: 80x80 cells (6400), 2560x2560 px at scale 1, tile slots 52/128, placeholder cells 0, shaded (theta=0.7854 ambient=14 range=32), geometry flat, y origin 0

# the sweep, per map, per geometry; every image hashed and deleted before the next
for m in "$INSTALL"/*.alm "$INSTALL"/*.ALM "$OUT"/corpus/*.alm; do
  for g in "" -flat; do
    terraintool render -assets "$INSTALL" -map "$m" -out "$OUT/sweep.png" -scale $S $g
    pngmeta "$OUT/sweep.png"   # IHDR dimensions + MD5, read back by a separate decoder
    rm "$OUT/sweep.png"
  done
done
```

**The whole corpus renders, and the summary describes the file.** 76 renders at scale 1 (38 maps ×
{default, `-flat`}) and 64 at scale 2 (the 32 the cap still admits on both geometries; Cross is
counted under R-1): **140 renders, exit 0 on every one, no panic.** On each, the summary's `WxH`
equals the dimensions read back from the written PNG's own IHDR by a separate decoder — **0
mismatches**; the `geometry` token matches the flag; `y origin` is 0 on every `-flat` render; and
**every map's projected PNG differs from its flat one — 38 of 38 at scale 1, 33 of 33 at scale 2.**
Scale multiplies the height exactly: `Kids` 2560x2571 → 5120x5142.

```
map            cells   projected  origin  grow        sloped  dmax  md5 default (projected)          md5 -flat
Beast.ALM    256x256   8192x8192     -46     0   57904/65536   109  924dfe3462d09850252451576f5aa4b0 f24f04cebb6b17699831f357fdb49646
Cross.ALM    256x256   8192x8192     -32     0   57946/65536   110  86c2156dec0b0016b1ab4049ffdea56d 326ff89427dcb28af76e4309e203c929
Forester.alm 256x256   8192x8204     -48    12   63732/65536    89  c06fc91de1df1b65234f42ac9a29731e 4090d9a948d450354200ebe45a9304ad
Horror.alm   256x256   8192x8203     -53    11   65124/65536    99  b672ec3fa4460a135c34cb1736fae164 1eac8aa0d907e2f2709134f2d106a3df
Islands.alm  256x256   8192x8203     -34    11   65181/65536   102  276e9b5c2bd9cce9b8568e0cddb335a7 d945b6a0d1f2d85ff96282a1858ba231
Kids.alm       80x80   2560x2571     -69    11    5870/6400     35  dbd5535a1d762af8fcf36d861dbc87f3 cec6de282e37644d72bd093e40921435
Kids2.ALM      80x80   2560x2571     -69    11    5926/6400     62  2a087c5d2d8cd1751e3413905f588c10 4a1700eeba0bcd0b78f74df698e60b8b
LuMoir.alm   144x144   4608x4608     -30     0   15671/20736   101  6be38ed9252dff271c1c6557d82788ff e19c6b1b5bfb5fcb2b926eaee9f30b9f
Tomb.ALM     256x256   8192x8203     -37    11   65164/65536   102  b78358b57bcc4e0f1d201425856e0e08 6f931442514290d2690507ed96a9f921
Waters.alm   144x144   4608x4619     -39    11   20543/20736    91  9639f2f7acccabcd8777cc9d0c4d1584 12036c3f36d5d9e93660f9bc3609eeea
10.alm         80x80   2560x2571     -64    11    5015/6400     48  f5eb22cc440d5c7a9b3f9785fd76e8ca 5377ab97d0abaf42e4c9f5b27308fe46
20.alm       144x144   4608x4619     -50    11   18862/20736    72  6cfe2c820a838f00415b4b9e122e242a 2b2ea9bc19b46f02249ac723eeb66ddb
30.alm         80x80   2560x2571     -59    11    5565/6400     45  d16c241e8ba26bb1fe2ff1eb804f5a50 a50224b30d63b3ad2f89bc89de724f55
31.alm         80x80   2560x2560       0     0    2221/6400    125  dfea674fee4ef2f5f9bb855a6f73f4a8 76efd343cfb7bbf647ea350f0847120f
40.alm       144x144   4608x4619     -13    11   19766/20736    94  17667ce26593224afae372bf33867286 fc1a741c5b28a31eb9c958bc390704b3
41.alm         80x80   2560x2571     -26    11    6246/6400     79  d428ba10c3c3cce7455d7c08f5c64ba6 9c0801b8643c097e716949048d57fbf3
50.alm       144x144   4608x4619     -13    11   20332/20736   122  6642eb6058d4a7142e8acc0d7df40cf0 81f1c5578fd13e6f8afcb57b87d3ccb7
51.alm         80x80   2560x2571     -69    11    6303/6400     68  635f0a16ffad3fab30b10f29ee29b441 151dbac5f33a6d68e1b2f3f730ef69ed
60.alm       144x144   4608x4619     -41    11   20275/20736   113  ab35dd29041beb998b78a709c4386b44 e1f035750fc056c47730fbb548375182
61.alm         80x80   2560x2648    -126    88    4557/6400     87  2901bea2ec0af620dec58307be44be6f 1af2b40fcd7a2ffd7b9069baeb158905
70.alm       144x144   4608x4619     -76    11   20312/20736    88  5e1af92f16cf6201bf357316ce1dc304 0857c5759eb763ef402336de26cd250d
71.alm         80x80   2560x2560     -12     0    2107/6400     89  897f711416c676bfea691c6c4bfc91a3 3b1833187fff155a351036aeaeb19451
80.alm       144x144   4608x4619     -38    11   20406/20736    88  4e445fcdfcbb5c1cc4bf8a78b7bedc59 1ac45c14d690dc98356303f5d14b3c52
81.alm         72x72    2304x2304      0     0    1676/5184    120  b352b0931daa59417db5fb72e7b9edc9 05b4bd38492d6baf98af7c4632aa059e
90.alm       144x144   4608x4619     -36    11   20421/20736    54  71a4de7d73661ea28cfbea6194c7584b 4d1d656037480f49f15d0c35a3f868ed
91.alm         80x80   2560x2560     -23     0    2681/6400    104  a55120365bed46c1d6f2926c1e1b0c30 52ce5483cced0588bacf2af1c4c3185c
100.alm      144x144   4608x4609     -64     1   16319/20736    94  2127f3f63b94e4130496c496a44cb826 37d150cc4b3558efa4d5e9d476437cf9
101.alm        80x80   2560x2571     -69    11    6387/6400     75  a10994f7e182142dbb74d6c4c3769779 c95597ef0c0a975d61e37e28436922bc
110.alm        80x80   2560x2571     -12    11    5824/6400    116  9a4bb0e9e6655a736b4731e19c265cbb 2d9be73f17b263d79a7712386c95886a
111.alm      112x144   3584x4608     -19     0   11661/16128    84  efc492b926e59c7642955a9be562bc98 4b9c308018d33fc299cbd161f94397c1
120.alm      144x144   4608x4619     -69    11   17489/20736    70  1fa8345590e7b51b84cfc7112707dda2 4b6594f61bd9f1c170517fbb98327d6b
121.alm        80x80   2560x2560       0     0     795/6400     90  6b816049d0b54e0b3fafb9c711c132f9 106b13399462efdf58b6d6c325aea7bc
130.alm      144x144   4608x4619     -35    11   19858/20736    82  b92817024f98948aa326b5bf34250e06 4cedc24ecc709495a39050b717b21f20
131.alm      144x144   4608x4609     -34     1   12790/20736    80  19bcdc82140b7319681e84aed2ad9996 e37f271d5fbf988789b5f744e88f1c05
140.alm      256x256   8192x8201      -9     9   64799/65536    95  68f66c941e7c60a4ef997f5c2dadaa0d 20f6950669373abb2db2b8af5bb7e775
141.alm        80x80   2560x2560     -14     0    3550/6400     97  3a81cd2642e8137024136a64abff7b31 780d270f6cbc707e36d5790a40a3f2a3
150.alm      144x144   4608x4619     -35    11   20529/20736   102  baef4e195a8857a33273a6369d788286 56ebfb11b545d8d1ff240d59641a97e9
151.alm      144x144   4608x4608       0     0    7365/20736   127  6a1836ce6034982f00b8aef3d7526876 6d4966e067465c0459737478be4097ae
```

`grow` is `outputHeight − H*32`, the native rows the mesh adds; `origin` is the reported `y origin`
(`minV`); `sloped` counts cells whose four corner altitudes are not all equal — FR-3's own selector;
`dmax` is the largest step count `d` any cell edge of that map walks. `sloped` and `dmax` come from a
probe run outside the repository over the decoded altitude grids, not from the images.

### Relief, as numbers

The projection changes the image height on 27 of 38 maps (`grow` 1…88) and the origin on 34 (−9…−126);
it changes the pixels on **all 38**. `61.alm` is R-6 made concrete: one vertex at altitude 126 drags
the origin to −126 and adds 88 near-empty native rows for the sake of one map corner.

**Where the canvas cannot show it, the pixels still do.** `121.alm` renders 2560x2560 at `y origin 0`
on **both** geometries — the two summary lines differ in the single word `projected`/`flat` — yet 795
of its 6 400 cells are sloped and the PNGs are different files (`6b816049…` vs `106b1339…`). That is
the case the contract admits and no dimension can report: altitudes that vary without moving either
extreme. `31.alm`, `81.alm` and `151.alm` are in the same class; seven more (`Beast`, `Cross`,
`LuMoir`, `71`, `91`, `111`, `141`) add no rows while the origin still moves.

**Uncovered pixels, FR-8's other half.** A projected image is transparent wherever no drawn column
covers it; the flat raster is opaque everywhere:

```
image                     dimensions   transparent      opaque  transparent share
121.alm  default           2560x2560             0     6553600   0.000 %   (grow 0)
121.alm  -flat             2560x2560             0     6553600   0.000 %
Kids.alm default           2560x2571         27984     6553776   0.425 %
Kids.alm -flat             2560x2560             0     6553600   0.000 %
61.alm   default           2560x2648        166800     6612080   2.460 %
61.alm   -flat             2560x2560             0     6553600   0.000 %
Islands.alm default        8192x8203         89808    67109168   0.134 %
Islands.alm -flat          8192x8192             0    67108864   0.000 %
```

`121.alm` is the discriminating row: zero transparent pixels, identical dimensions, different image.
Transparency measures the canvas the relief opens up, not the relief.

### R-1 — the budget regression is real, and it is exactly five maps

The cap is `1<<28 = 268 435 456` pixels, equality allowed. A 256x256 map at scale 2 is `16384 x
16384` flat: the cap **exactly**. Add any canvas growth and it is over.

```
$ terraintool render ... -map Islands.alm -out isl.png -scale 2
terraintool: terrain: composite would be 16384x16406 px, over the 268435456-pixel cap; try a smaller scale
              (exit 1, no file written)
$ terraintool render ... -map Islands.alm -out isl.png -scale 2 -flat
terrain: 256x256 cells (65536), 16384x16384 px at scale 2, ..., geometry flat, y origin 0

$ terraintool render ... -map Cross.ALM -out cross.png -scale 2          # same size, zero growth
terrain: 256x256 cells (65536), 16384x16384 px at scale 2, ..., geometry projected, y origin -32

refused at scale 2, no file written   Forester.alm 16384x16408 | Horror.alm, Islands.alm, Tomb.ALM
                                      16384x16406 | 140.alm 16384x16402
accepted at scale 2, both geometries  the other 33, incl. Beast.ALM and Cross.ALM at 16384x16384
Kids.alm scale 6  projected           15360x15426, accepted
Kids.alm scale 7  projected / -flat   17920x17997 / 17920x17920, both refused - the cap, not the
                                      projection
```

Arithmetic over the measured canvases, scale 1…64, finds no other `(map, scale)` where the flat
product fits and the projected one does not: these five at scale 2 are the whole regression.
`Beast.ALM` and `Cross.ALM` pass projected at exactly `16384x16384` — same size, same scale, same cap
as the refused five, and no rows added.

### R-2 — the superseded hashes

Each recorded digest was re-taken at the new default, and the same invocation under `-flat` checked
against the record. The recorded column was measured by binaries built at `c63cb25`/`09bef64` —
genuinely older code, not a rebuild of today's tool, so the right-hand equality is a cross-binary
check and not a tautology.

```
recorded in                            invocation (scale 1)   recorded                          new default                       -flat today
0007 verification.md:160               Islands.alm -unshaded  8ccafd00b293b315339aaba59c2e3b44  a23a8b4be0cdbe63eb6912a93f1d4d7a  8ccafd00b293b315339aaba59c2e3b44  = recorded
0008 verification.md:235               Islands.alm            d945b6a0d1f2d85ff96282a1858ba231  276e9b5c2bd9cce9b8568e0cddb335a7  d945b6a0d1f2d85ff96282a1858ba231  = recorded
0008 verification.md:234               Islands.alm -objects   fbeac976cc5e3d9a07472d5e8ccb88b8  f7a73a60b5050f711f25b582b1c8f592  fbeac976cc5e3d9a07472d5e8ccb88b8  = recorded
0009 verification.md:138               Kids.alm               cec6de282e37644d72bd093e40921435  dbd5535a1d762af8fcf36d861dbc87f3  cec6de282e37644d72bd093e40921435  = recorded
0009 verification.md:139               Kids.alm -objects      26fc5caf493ed7d95e97f1ae66e866c5  a5f3cbd8ee0e4d811448a90dd31c33c0  26fc5caf493ed7d95e97f1ae66e866c5  = recorded
0009 verification.md:142               Kids.alm -units        9a6282ed8de4d00f481fb21ce65be016  ebef88d95cc7197ad07cc8afb3637762  9a6282ed8de4d00f481fb21ce65be016  = recorded
0009 verification.md:140               LuMoir.alm             e19c6b1b5bfb5fcb2b926eaee9f30b9f  6be38ed9252dff271c1c6557d82788ff  e19c6b1b5bfb5fcb2b926eaee9f30b9f  = recorded
0009 verification.md:141               LuMoir.alm -objects    5c0b14fee5ebb81ee2dd1c4501b2c4c0  022a23ddf2c652dce3e88f96edf4a055  5c0b14fee5ebb81ee2dd1c4501b2c4c0  = recorded
0009 verification.md:143               LuMoir.alm -units      ac07de7e4ccf768b0814f28af5401c77  5d0954682c58aa29e632258066003c9f  ac07de7e4ccf768b0814f28af5401c77  = recorded
0009 verification.md:144               Horror.alm -units      da5fb519e73a31357c1dd9679e123b7a  870f2440572ec9997ee57ba14aadda17  da5fb519e73a31357c1dd9679e123b7a  = recorded
0004 verification.md:231 (no digest)   Cross.ALM -unshaded    summary line only                 47beb2d3f401d64aa4ab882c74f11ddc  ee283b416267b32d4af96b661e825131

superseded without a digest moving
  0004:231       the summary line for -unshaded on Cross.ALM. That invocation now selects the
                 projected geometry; the line gains "geometry projected, y origin -32". Its
                 "8192x8192 px ... placeholder cells 0" still reads true - Cross's canvas happens
                 not to grow - but the pixels behind it changed, and the flat image now needs
                 -unshaded -flat.
  0007:244-245   two Islands.alm summary lines at 8192x8192; both now read 8192x8203, y origin -34.
  0006           supersedes nothing: its developer-run evidence is mapview -check cadence lines,
                 and 0012 changes no viewer path.
```

**Ten digests superseded, ten reproduced exactly under `-flat`** — FR-1's "identical to the
pre-change render", on real map data, at ten flag combinations across four maps, against records
written by earlier binaries.

**The markers moved with the terrain lattice, not relative to it**, so 0009's marker figures stand.
Marker-coloured pixels were collected from the projected and the flat `-objects -units` render of two
maps, by a decoder outside the repository:

```
map           projected image  markers (object / unit)   flat image  markers    shift    unmatched  coordinate digest
Kids.alm      2560x2571        1971 (1104 / 867)         2560x2560   1971       +69 rows  0 / 0 / 0  d03b9e99c93da7c71c25f4189bed346e (both)
Islands.alm   8192x8203       27181 (19803 / 7378)       8192x8192  27181       +34 rows  0 / 0 / 0  023fbb4d3b78faa990bd62e94554ce8d (both)
```

The shift is `−OriginY * scale`, DD-8's own expression: **0 flat pixels unmatched, 0 of the wrong
colour, 0 projected pixels unaccounted**, on both maps, with identical coordinate digests once the
flat set is shifted. The totals are 0008's `19 803` and 0009's `1 104` / `867`, unchanged.

### What shipped data does and does not exercise

```
maps                       38
cells / altitude bytes     880704
bytes >= 0x80 (negative)   0
cell edges walked          1761408
edges with d = 63          574   on 30 maps
edges with d = 127         6     on 1 map (151.alm)
edges with d > 127         0
```

The corpus reaches **both** step counts at which the walks disagree in the overdraw direction — 574
edges at `d = 63` on 30 maps, 6 at `d = 127` on `151.alm` — so SC-3's seam classification is
witnessed on shipped maps, not only on fixtures. Nothing above 127 occurs, so the extension's other
sign (`d = 191`, `d = 240`, the seam row drawn by neither cell) stays synthetic-only, as R-3 states.

## Limitations and criteria not claimed

- **AC-13's "relief is visible" — the eye's half is NOT RUN.** Nobody has looked at a rendered image
  and judged it. Recorded above is the measurable half: growth, origin, sloped share, transparent
  pixels, 38 of 38 images changed. Whether the relief *reads* as relief — slopes as slopes, the seam
  rows not as tearing — is the owner's call, on the eight kept images. This mirrors 0007's AC-8
  fidelity half and 0008/0009's live-window halves, all still pending.
- **The signed altitude read is not witnessed by shipped data.** 0 of 880 704 altitude bytes reach
  `0x80`; every corpus altitude is in `[0,127]`. FR-2's signed read and AC-9's `-128 beside 127` rest
  entirely on SC-1 and SC-11's synthetic grids. This run confirms the corpus figure, not the sign.
- **The far-edge ring is our clamp, not a measurement.** Every map's last vertex row and column take
  it, so nothing here distinguishes it from what the engine draws there.
- **Markers do not register with the terrain under them** — measured above; on sloped ground a cross
  sits near, not on, its unit's ground. Story 0015's, per *Out of scope*.
- **`mapview` was not run.** 0012 changes no viewer path; `pkg/ui` draws cells itself, still on the
  flat lattice. No evidence about the viewer is offered here.
- **R-1's flat side is one rendered witness plus arithmetic.** `Islands.alm -flat -scale 2` was
  rendered in full and accepted at `16384x16384`; the other four refused maps are 256x256 too and the
  budget test reads `W`, `H`, `scale` only, so their flat side follows without four more
  quarter-gigapixel renders.

## Images: kept and deleted

**169** images were written by this pass, all outside the repository; each sweep image was hashed and
deleted before the next, so one large PNG existed at a time and the peak footprint stayed under
250 MB where keeping the corpus would have cost tens of gigabytes. Eight are kept for the owner:
`Kids-projected/flat.png` and `Islands-projected/flat.png` (`-objects -units`; the A/B pairs and the
marker evidence), `121-projected/flat.png` (the identical-canvas pair, where only the pixels move)
and `61-projected/flat.png` (the +88-row outlier). Every other render, and the maps extracted from
`scenario.res`, were deleted at the end of the run.

## Conclusion

Every automated criterion SC-1…SC-14 passes. SC-15's measurable half is complete: **140 renders over
the full 38-map corpus at two scales — exit 0 on all, summary agreeing with the written file on all,
38 of 38 images changed.** R-1 is confirmed and bounded: five maps, all 256x256, all at scale 2,
refused with no file written where the flat render still fits the cap exactly. R-2 is discharged: ten
superseded digests re-taken, and the same ten reproduced byte-for-byte under `-flat` against records
written by older binaries — FR-1 on real data rather than on a fixture.

**AC-13 is therefore partially satisfied**: its completion, count and hash halves are evidenced, its
"relief is visible" half is measured but not *seen* and is declared not run above. No defect was found
by this pass.
