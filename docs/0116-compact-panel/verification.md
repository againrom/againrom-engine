# Verification — 0116-compact-panel

## The measurement

`cmd/paneldump` against the lawful install at `gameversions/en`, mission 10 (`scenario/10.alm`),
font `font1`, line height 15. Both subjects are read out of the mission's own started world: the
first entity the loader knew a character for, and the first it did not.

```
go run -trimpath ./cmd/paneldump -assets <install> -mission 10 [-png <dir>]
```

**Before** — run on this branch at `40a76ee`, which is the instrument alone with no layout change:

| subject | rows | box |
|---|---|---|
| party, `Human Swordsman` | 18 | 255 x 337 |
| placed, `Human ClubMan` | 9 | 168 x 175 |

**After** — `76c22b3`:

| subject | rows | box | height | area |
|---|---|---|---|---|
| party, `Human Swordsman` | 12 | **268 x 229** | **-32 %** | **-29 %** |
| placed, `Human ClubMan` | 6 | **227 x 121** | **-31 %** | **-7 %** |

The window is 1024 x 768. The party panel was 44 % of its height and is now 30 %.

**Width grew in both cases and this is stated rather than buried.** The party panel is 13 pixels
wider — its width was already pinned by `WEAPON Iron Short Sword`, which no pairing can shorten.
The placed panel is 59 pixels wider because its old width was not content at all: it was the
layout's 168-pixel floor, which nine one-value rows never reached. Height is what the panel was
spending badly, and height is what came back.

An intermediate reading is worth keeping because it is why this story built the instrument first.
At `0cb1f75`, with both tasks landed and the plan followed exactly, the party panel measured
**363 x 229**: a 32 % cut in height and none at all in area. DD-4 had let a full-width row place the
second column, so the weapon name pushed the grid 108 pixels right. No test could have found it —
every acceptance criterion passed. `76c22b3` corrects DD-4 and the plan paragraph that produced it.

## What an ordinary mission-10 unit's panel now states

```
Human ClubMan
HP 15/15
CELL 24, 54  SPEED 16
DMG 2-3      HIT 3
DEF 6        ABS 0
SWING 7/4
```

Six rows for nine values. No character rows are drawn and none is drawn empty: `ui.PanelSubject`
names no party type, `Char.Known` gates the six character rows and `Combat.Known` the combat block,
so a unit the loader knows no character for states neither and its rows are skipped. The panel was
already reusable for an ordinary unit before this story; what it lacked was proportion.

The party member, for comparison — twelve rows for eighteen values:

```
Human Swordsman
HP 100/100
CELL 17, 66      SPEED 17
BODY 43          REACT 26
MIND 15          SPIRIT 15
SKILL Blade 10   XP 1593
WEAPON Iron Short Sword
DMG 10-16        HIT 49
DEF 8            ABS 0
SWING 7/4
PROT 7 7 7 7 7
RES 0 0 0 0 0
```

## Nothing stated was lost

The two stated sets above hold every value their `before` counterparts held, and no new one. The
before/after `PanelStatement` diff is exactly: rows joined, labels shortened, `XP`/`PROT`/`RES`
moved from the middle of the combat block into the character block. `TestAuthoredPanelLayoutStates
ExactlyTheTwentyOneFields` reads the same thing off the layout — every row and every `Right` cell,
compared both directions against twenty-one field constants written out by hand.

**A value that belongs on the panel and is absent:** none was found. Nothing is deferred on that
axis.

## Acceptance

| id | witness |
|---|---|
| AC-1 | `TestPanelTwoCellRowFourSubjectShapes` (four subtests: both cells, left only, right only, neither) |
| AC-2 | `TestPanelRightColumnOriginFollowsTheDrawnLeftCells`, and `TestAFullWidthRowDoesNotPlaceTheSecondColumn` for the half the measurement found |
| AC-3 | `TestPanelTwoCellRowPaintStaysInsideTheBox`, over the fixture font's overhanging glyph |
| AC-4 | every pre-existing `pkg/ui` test, `readout_test.go` included, passing unmodified. The readout names no right cell, so both new passes are inert for it by construction |
| AC-5 | `TestAuthoredPanelLayoutStatesExactlyTheTwentyOneFields` |
| AC-6 | `TestTheShippedLayoutDrawsTheCanonicalRowCounts` — `12`, `6`, `18`, `9` as literals, counted through `PanelStatement` and not derived from the layout |
| AC-7 | `TestTheWholeSheetFitsTheDefaultWindow`, extended to state a mana pool so it is genuinely the fullest case, and asserting 13 rows first |
| AC-8 | `TestPanelStatementIsOneEntryPerDrawnRow`, `TestPanelStatementSkipsARowWithNothingToState`, `TestPanelStatementReadsBothSubjectKinds` |
| AC-9 | the tables above, against `gameversions/en` |
| AC-10 | `TestPanelSubjectIsUsableAsAMapKey` — will not compile if the subject stops being comparable |
| AC-11 | `internal/archtest`, with `cmd/paneldump`'s own row added and `pkg/ui`'s untouched |

## Derived properties

- **P-1** — `TestPanelTwoCellRowFourSubjectShapes`'s fourth subtest: the dropped row's successor
  sits at `Pad.Y`, so it left no gap. `TestPanelRightColumnOriginFollowsTheDrawnLeftCells` covers
  the second pass's half: a dropped wide row does not place the column.
- **P-2** — AC-5 and AC-6 read it directly; the resolver `panelText` is untouched by this story,
  so the values themselves could not have moved.
- **P-3** — `PanelCell` and `Right` are fields of `PanelLayout`, which `SetPanelLayout` replaces
  whole. `TestPanelIsAFunctionOfItsLayout` still passes unchanged.

## Gate

Run on the clean committed tree at `76c22b3`:

```
go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -trimpath -count=1 ./...
```

`EXIT=0`; 31 packages `ok`, no failures, `gofmt` printed nothing.
`scripts/check-no-game-assets.sh` `EXIT=0`. `scripts/check-doc-budget.sh` `EXIT=0`.
`scripts/check-sdd-audit.sh` — zero `FAIL` lines with this file present. Note counts are not
comparable from a lane worktree, which has no `builds/`.

`git log --format='%h %(trailers:key=Co-Authored-By)' 84aa355..HEAD` prints an empty trailer field
for every commit.

## Divergence and what is ours

The panel's layout is **authored**, and `UNIT-PANEL-011` is why: the original's cache is addressed
by a computed index from `actor+0x14a` on, so no displacement sweep can name a consumer of the five
protection or five resistance bytes, and *which cached value appears at which position cannot be
settled without the index's own table and the drawing loop*. A consumer has the value set and its
arithmetic on evidence and its layout on nothing.

So the pairing, the row order, every label including the seven abbreviations, the column gap and
the slide rule are all ours. Nothing here reproduces anything and nothing here was fitted to
anything. `PanelLayout` remains the substitution point: the original's arrangement, if it is ever
decoded, arrives as a different value of that type and moves no drawing code.
