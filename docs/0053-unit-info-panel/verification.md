# Verification — 0053-unit-info-panel

Environment: Windows 11, Go toolchain pinned in `go.mod`. Developer runs are against the preserved
lawful installs at `gameversions/en` and `gameversions/ru`; no test reads either.

## Gate

```
(go build ./... && go vet ./... && go test -trimpath -count=1 ./... && sh scripts/check-no-game-assets.sh
 && sh scripts/check-doc-budget.sh && sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))")
EXIT=0     FAIL set: empty
```

`-trimpath` is load-bearing locally: Windows Defender quarantines one test binary without it.

## The disclosed divergences

This story's verdict is **AUTHORED**, and three things follow that are recorded here rather than
softened anywhere else.

**1. The panel's appearance reproduces nothing.** No claim settles the original's layout, art or
field positions, and `UNIT-PANEL-011` says so positively: the display reads its cache by computed
index, so a displacement sweep finds no reader, and what appears where needs the interface layer.
Corner, size, colours, border, labels and row order are ours. **The seam is `PanelLayout` in
`pkg/ui/panel.go`, produced by `AuthoredPanelLayout()` and installed by `(*Viewer).SetPanelLayout`.**
It carries a background picture, per-row offsets and its own colours, so the original's panel — when
the interface layer is read — is a *value*, not a rewrite. `TestPanelIsAFunctionOfItsLayout` holds
the property that two viewers differing only in that value differ only in the panel's pixels.

**2. The caption is a stand-in, not a reconstruction.** `UNIT-PANEL-010` establishes at High, by a
reference enumeration over the class array, that the display path reads **no `units.reg` field at
all** — so neither `DescText` nor `InfoPicture` reaches it, and the original's caption comes from a
source nobody has read. `DescText` is the only per-class name text this tree holds. What the panel
prints is therefore honest and is **not** what the original prints.

**3. Health is real and provisional at once.** `openMapWorld` builds every playable world through
`mapload.FromALM`, which passes no definition table, so every placed unit is born at `SpawnHP` —
100 on both fields. `FromALMWith`, which resolves the table's adjusted `HealthMax`, is reached only
by `cmd/classdump`. The panel states the number the simulation actually holds, and it moves under
damage; it is not the class table's health. Closing that is a story of its own.

## What the tree could not source, and so is not shown

`data.UnitDef` resolves some thirty named columns. Exactly **one** reaches a live entity — the
health maximum, via `FromALMWith`, which the game does not call. A `sim.Entity`'s whole field set is
an id, a cell, an optional target, an opaque class key, a stall count, `HP`/`MaxHP` and a movement
domain: **health is the only stat on it.**

Omitted for that reason, and not drawn as a zero, a dash or an empty label: the four primaries,
mana and its period, health regen period, speed, rotation speed, scan range, sight, reach, the
damage pair and its always-hits mark, to-hit, defence, absorption, attack charge and relax times,
five protections, five resistances, type id, face, token size, movement type, XP value.

## Evidence by criterion

| # | Evidence |
|---|---|
| AC-1, SC-1 | `TestLoadUnitsCarriesTheClassName` — three classes: ASCII, all-high-byte, and none. Byte-for-byte comparison, and every class loads frameless so the name cannot have arrived through art. PASS |
| AC-2, AC-3, SC-2 | `TestSnapshotCarriesTheEntitysOwnClassName` — alive/downed/dead of a class whose corpse class is **differently named**, plus an id naming no class. Each non-alive row also asserts `Art` **is** the corpse class, so the rows that witness the name also witness that the substitution they escaped happened. PASS |
| AC-4, AC-5, SC-3 | `TestPanelSubject` — four cases: empty selection; a selection whose first entries are dead and absent and whose survivor is **downed**; several selected; none surviving. PASS |
| AC-6, SC-5 | `TestComposePanelIsDeterministic` — two composes of one input compared byte for byte over the whole pixel buffer; a nil font and a record-less font both compose nothing. PASS |
| AC-7, SC-4 | `TestComposePanelStatesExactlyItsRows` — exactly three rows with their labels and values; the value's offset is the label's **pen** plus the gap, and zero after an empty label. PASS |
| AC-8 | `TestPanelNameSelectsARecordPerByte` — `"\xc0\xc1"` and `"\xc0\xc0"` measure the same and draw differently. PASS |
| AC-9, P-3, SC-6 | `TestPanelFitsItsContent` — content narrower and wider than the minimum; the box is never under the minimum, never short of the widest row, and the rightmost and lowest **text** pixel lie inside it. The frame is blacked out for the test so any coloured pixel is text and the containment is not a tautology over a filled rectangle. PASS |
| AC-10 | `TestPanelBackgroundAndRowModes` — supplied background drawn as given; fill-and-border; flowing rows at pad + index×(line+gap); placed rows at their own offsets; a row placed at (1000, 1000) in a 40×20 box paints nothing. PASS |
| AC-11, SC-7 | `TestPanelPresent` — no panel with no font, none with nothing selected, and with both the origin is the bottom-left corner at the margin for a 640×480 area. `TestPanelOrigin` covers all four corners and an oversized box, which is neither shrunk nor moved. PASS |
| AC-12, SC-8 | `TestPanelRebuildsOnlyOnAChange` — one build, then five unchanged frames and a camera move add none; a health change, a **different unit stating identical values**, a resize and a replaced layout each add one. The third case is invisible in pixels, which is why the subject's id is in the key. PASS |
| AC-13, SC-9 | `TestTheFontIsLoadedAndIsNeverFatal` (`cmd/againrom`) — both nodes, atlas missing, sidecar missing. **All three assemble**; the two broken ones carry a reason that names the font and are reported by the check line, which stays unchanged on a complete install. `TestAFontlessFrontEndStillOpensAMap` (`pkg/game`) opens a map on a front-end with no font, advances it and finds the entity. PASS |
| AC-14, SC-10 | `TestPanelDoesNotDisturbTheFrame` — the pass slice is identical with and without a font, and the case is guarded against being vacuous by asserting the font-bearing viewer really does present a panel and the bare one does not. The existing suites over `pkg/ui`, `pkg/game`, `pkg/render/terrain`, `cmd/againrom` and `internal/archtest` pass **unedited**. PASS |
| P-1 | `TestComposePanelIsDeterministic`. The no-file/no-clock/no-context half is **by construction, not by test**: `composePanel` is a package-level function over three value arguments and `pkg/ui/panel.go` imports only `fmt`, `image`, `image/color` and `pkg/render/text`. Stated as inspection, not claimed as an executed check. |
| P-2 | No file under `pkg/sim` is touched; `internal/archtest`'s import check and determinism source scan pass; the full suite including every digest and replay test is green. |
| P-4 | `TestPanelStatesOnlyWhatTheSubjectCarries` — an unnamed subject drops the name row and the kept rows close up; downed (0/100), dead (−37/100) and no-health-system (0/0) are each stated as they stand; an undefined field constant yields no row. PASS |
| P-5 | `TestPanelIsAFunctionOfItsLayout` — two viewers sharing a layout agree pixel for pixel, two differing in it do not, and mutating one `AuthoredPanelLayout()`'s rows cannot reach another's. PASS |
| SC-11 | Below. |

## Developer runs against both lawful installs (SC-11)

Both releases, through `game.OpenContainers` + `game.LoadFont` + `ui.RenderPanel`, from a program
outside both repositories:

```
en: font font1 = 224 records, line height 15, spacing 2
en: 34 unit classes;  0 of 34 class names carry a byte >= 0x80
ru: font font1 = 224 records, line height 15, spacing 2
ru: 34 unit classes;  0 of 34 class names carry a byte >= 0x80

en:  graphics/units/units.reg    31563 B  md5 02a6f419e51b12d8a9434b6e7264ded1
     graphics/font1/font1.16     32932 B  md5 6f67da5dd9ead6b916828d7d27b50035
     graphics/font1/font1.dat      896 B  md5 8c7dff8b7e1a7e805273ed61ab834d35
ru:  graphics/units/units.reg    31563 B  md5 02a6f419e51b12d8a9434b6e7264ded1
     graphics/font1/font1.16     32932 B  md5 6f67da5dd9ead6b916828d7d27b50035
     graphics/font1/font1.dat      896 B  md5 8c7dff8b7e1a7e805273ed61ab834d35
```

**A finding, and it matters for G1.** The unit registry and both font nodes are **byte-identical
between the English and Russian installs**, and every one of the 34 class names is ASCII in both.
So the panel is the same panel on both releases and its caption is **English on the Russian one**.
The unit roster is not localised in `units.reg`; wherever the Russian release's unit names live, it
is not here. That is a question for research, and it is why the contract says only that a byte
reaches its own glyph record and never that the glyph *reads* as the name.

Headless check, both roots, exit 0, and neither reports a font failure:

```
againrom.exe -check -assets gameversions\en   ->  againrom: 38 map rows, 8 of 8 buttons have a mask region
againrom.exe -check -assets gameversions\ru   ->  againrom: 34 map rows, 8 of 8 buttons have a mask region
```

Panels rendered to `<seat>\review\0053-unit-info-panel\`, six per release, at 3x:

```
en-class1.png    box 168x67   "Unarmed Fighter"   HP 100/100   CELL 41, 27
en-class2.png    box 264x67   "Unarmed Fighter with Shield"    (a name wider than the minimum widens the box)
en-class3.png    box 170x67   "Human Swordsman"
en-wounded.png   box 168x67   HP 60/100
en-downed.png    box 168x67   HP 0/100
en-unnamed.png   box 168x49   no name row; the box is one line shorter and the rows closed up
ru-*.png         identical boxes and text, from the Russian install's own archive
```

## Limitations, honestly

- **The upload is not covered by a test.** The one `ebiten` call that puts the picture on the GPU
  lives in `Draw` and needs a graphics context. What *is* tested is the flag that couples it to the
  picture — a mutant leaving that flag unset survived until the assertion existed, and is killed now.
- **The panel has not been seen inside the running window by this agent.** Everything above is the
  same composition code the window presents, rendered headlessly; the window itself is the owner's
  run. The PNGs exist so that judgement can be made before the binary is trusted.
- **A constraint on the seam that is not taste.** The font's blit *replaces* rather than blends and
  scales a partly-lit pixel toward black. Light text on a dark fill reads cleanly, which is what the
  authored layout uses; a substitute layout with a light background or dark text will get black
  fringing on every antialiased edge. That is a fact about the shipped blit, and a later layout has
  to live with it.
- **Mutants run**: minimum width as a ceiling; a skipped row still spending its gap; the value
  placed by the box rather than the pen; the health pair, the area size and the subject's identity
  each dropped from the refresh key; the freshness flag left unset; and the name read off `Art`
  instead of the live class. All killed. The third needed the fixture font corrected first — its
  glyphs did not overhang their advances, so pen and box were the same number and the case could not
  discriminate.

## Gate outputs

```
check-no-game-assets: ok
check-doc-budget:     analysis 5623/7168 · provenance 7815/16384 · spec 13100/13312 · plan 13092/13312
                      tasks T1 902 / T2 1008 / T3 840 / T4 707 (of 1400) · legend 587/1200
                      plan <= 1.2 x spec ok · tasks <= 1.2 x plan ok
check-sdd-audit:      4 trailered commits for this story, one per task, no id twice
```
