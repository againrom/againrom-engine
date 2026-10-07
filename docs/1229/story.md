# Writer census over changed worlds

## Intent

Owner direction, `pipeline/SAV-ENDGAME.md` execution order item 8, second
part. The permanent round-trip instrument resaves an unchanged load, where
carried original bytes hide writer gaps. This census changes the world before
it writes, then checks persisted fields against research rules instead of
against this engine's reader.

## As built

### Writer

- Town saves write `/GameOptions/{FlyingHP,ShowHP,ShowTimeFlow,Speed,Wimpy}`,
  `/Inventory/IsOpen`, `/SpellBook/{IsOpen,Pressed}` and `/View/{X,Y}` from
  current application state when the session has one, else from the loaded
  file's own state store (`projectCityApplication`,
  `pkg/game/nativecityapplication.go`). With no current state the loaded
  Formation is written back too. A town whose provenance is unavailable (for
  example a roster that did not restore faithfully) still keeps its loaded
  state-store records for this purpose; they are not persisted in the engine
  snapshot. Before, 27 of 40 town originals were written with `Speed` 0,
  three options 0 and `/View` (0,0).
- A Sack's `+0x1c` is gold plus each written item record's unit price, summed
  in 32 bits with no clamp (ITEM-SACK-010, SHOP-PRICE-011). A stack counts its
  unit price once. The engine's own ground draw reads the same sum
  (`World.SackValue`).
- A Sack the session constructs gets a runtime id and mask 2 at construction
  (MOVE-ID-016, SAV-655, SAV-1093): the lowest id no sack, actor, dead body,
  building, item record or effect record holds. This is simulation state, so
  a world with a new sack hashes differently.
- `mapWorld.entity` looks an entity up with the new `World.Entity` instead of
  copying the whole entity list per lookup. Behaviour is unchanged; a tick
  costs about 30 percent less on the census's worlds.

### Census

`pkg/game/savwritercensus_corpus_test.go`, build tag `sessioncorpusaudit`,
four tests. Each corpus file runs as its own parallel subtest with its own
front end, at most six at a time, larger files first. The walker skips
`exp*-*` directories through `corpusDirSkip`.

- `TestSAVWriterCensusChangedWorlds` restores each original, opens the
  mission at 1024x768 and changes it: a drop from the first party member's
  pack into a new sack, a pick-up of the nearest original sack, movement of
  the other members, a kill advanced until the victim reaches full decay, and
  last a camera pan through the viewer's clamp to the top-left corner (or on
  toward the centre when that leaves the loaded origin unchanged). It writes
  through `SaveDialogSeams` Prepare and Commit. Town files are written
  unchanged.
- `TestSAVWriterCensusReference` applies the sack and view-floor rules to
  every world original. It fails on zero world originals and on any break in
  `game0002-bigsack.sav`.
- `TestSAVWriterCensusSensitivity` writes `game0002-bigsack.sav` once and
  checks the unedited write plus nine edited copies (below).
- `TestSAVWriterCensusFailures` proves the enforcement returns a failure for
  a mismatch or refusal with no baseline row, a change below its floor, and an
  unnamed field, and pins the field classification.

Every Sack field, every contained item record and every state-store leaf is
compared with the loaded original. A difference is accepted only when an
exact rule confirmed the written value in that file, or when
`writerCensusDebt` names it with its cause; anything else fails. The rules
describe state the session creates or changes, not what SAVE writes for
carried state.

| Field | Rule | Check |
|---|---|---|
| Sack `+0x1c` | ITEM-SACK-010 | gold `+0x3c` plus each contained item's `+0x1c`, 32-bit, over item values each equal to the original's |
| surviving Sack | join on cell and runtime id | every field equal, or the difference fails; an original sack missing from the written file fails unless it is the one picked up |
| constructed Sack | MOVE-ID-016, SAV-655 | runtime id nonzero and held by no other record; `Token+0x18` = 2 |
| contained item | join on class and Identity key | every field equal to the original record's |
| item `Token+0x08` | ITEM-GROUNDMOVE-130 | a change must be the pickup stamp, original OR 1 |
| item split off a stack | none for the new key | every field but Identity and count repeats an original record, and the counts sum to its count |
| world `/View` | SESS-VIEW-030 | equal to the camera origin at save time, floored at 8 |
| `/Fog/Data` | SAV-FOG-061 | runs cover the loaded extent; no explored cell lost |
| party and roster supplement | PARTY-ROSTER-002 | no actor named twice |

Town leaves are their own family (`town:`); a town file's `/View` and options
are compared with the original like any other leaf.

`scripts/check-milestone2-acceptance.sh` selects the four tests as pooled
instruments (`TestSAVWriterCensus` in its pattern), whitelists
`SAV-WRITERCENSUS-` and `unreadable, skipped` lines, and clears
`AGAINROM_CENSUS_ONLY` for every instrument process.

## Proof

Census on the merge of engine main `39cdd83`, corpus `gameversions/saves`
(105 readable `.sav`; `game9000.sav` is unreadable and logged),
identical on both lawful roots:

| Root | discovered | changed | written | refused | mismatched | unnamed | census wall | per-file sum | slowest file | process wall, all four tests |
|---|---|---|---|---|---|---|---|---|---|---|
| EN | 105 | 65 | 105 | 0 | 0 | 0 | 48 s | 4m06s | 13.1 s `game0002-bigsack.sav` | 60 s |
| RU | 105 | 65 | 105 | 0 | 0 | 0 | 43 s | 4m06s | 13.1 s `game0002-bigsack.sav` | 55 s |

Before this pass the census ran serially at 5m41s (EN) and 5m27s (RU).

Changes over the 65 world files, each with a checked-in floor: camera 65,
drop into a new sack 63, pick-up 61 (4 incomplete), party movement 65, kill to
full decay 65.

Fields a rule confirmed, both roots: world `/View/X` 46, `/View/Y` 62, item
`Token+0x08` pickup stamp 8, split item records 2. Debt, both roots: world
`AgainromActions` 65, `AgainromRng` 65, `/Fog/Data` 57, `/Objects/Selection`
21, `/Projectiles/IDs` and the `Prj` leaves 2, `/SpellBook/Pressed` 1; town
`AgainromActions` 40, `/Objects/Selection` 31. No baseline row remains.

Reference: 65 world originals, 7 stacked items inside sacks, no break.

Sensitivity, both roots (11.3 s EN, 11.6 s RU, one write reused by every
edit); each edit must fire exactly the named keys:

| Edit | Caught by |
|---|---|
| unedited | nothing fires; the view is confirmed by rule and every sack value rests on original item values |
| every Sack `+0x1c` zero | ITEM-SACK-010, field `sack:T1C` |
| item prices zeroed and sack values made consistent | fields `sack-item:T1C`, `sack:T1C` |
| original sack re-identified as runtime id 0, gold 0 | missing original sack, MOVE-ID-016 |
| constructed sacks with runtime id 0 and mask 0 | MOVE-ID-016, SAV-655 |
| `/View/Y` 7 | SESS-VIEW-030, field `/View/Y` |
| `/View` carried from the loaded file | SESS-VIEW-030 |
| `/GameOptions/Speed` changed | field `/GameOptions/Speed` |
| fog plane written unexplored | SAV-FOG-061 |
| entry party restated into the roster | PARTY-ROSTER-002 |

Hotfix `7ee3346`: with its seven lines reverted in a scratch build, the EN
census still reported refused=0. The corpus originals import a native saved
Groups span, so a save takes the saved-roster projection and never the
live-only rebuild that refused. The census does not cover that route; the
hotfix's own test does.

## Open debt

- DIV-878: a town with neither current application state nor a loaded file
  (for example one reached after a mission) still writes the constructor's
  zeros for its options and view.
- Research question: how the Sack recomputation treats an item priced -1. The
  writer now sums in 32 bits, so such an item subtracts one; no claim fixes
  it.
- A town's `/Objects/Selection` is written empty where the loaded file names
  objects by index; no claim fixes what a town selection index names in a
  rebuilt document.
- A record split off a stack by a one-unit drop carries an engine-minted
  Identity key; no claim fixes the key.
- Item and effect runtime ids are reserved for a constructed sack because
  whether LOAD re-marks them is not established.
- Coverage limits: actor, group, dead-list, building and campaign fields are
  not compared. The walker admits upper-case `EXP-*` directories because
  `corpusDirSkip` matches lower-case `exp` only, and skips seven `exp*-*`
  files the review ran by hand.

## Reconciliation

Merged engine main `bd1e652` (story1225 and the chain-speed hotfix), then
`39cdd83` (the companion-detach hotfix). The
census leg that ran per root in `check-milestone2-acceptance.sh` is replaced
by the pooled instrument selection. `internal/storyguard/baseline.go` keeps
both sides' paragraphs and CommentBytes is re-measured at 7846553.

## Gates on the correction

- `go test -trimpath -count=1 ./...` on the merge: every package passed
  except `pkg/sim`, whose two pinned-method-set tests named the new
  `World.Entity`; the next commit registers it, and `go test ./pkg/sim`
  passes on it.
- `gofmt` clean; `scripts/check-no-game-assets.sh` clean;
  `internal/storyguard` passes at CommentBytes 7846553.
- Release tests from every `pkg/game` release file that touches sacks, town
  options or town saves (127 tests), EN and RU: 111 pass on each root,
  including `TestReleaseSackTokenValueRoundTripsFromTheOriginal`,
  `TestReleaseRestoredSackDrawsItsOwnValueBand`,
  `TestReleaseMilestone2Sacks1151` and `TestReleaseNewSack1200EmptyCell`. The
  other 16 fail with the same messages on engine main `bd1e652` under the same
  environment outside `check-release-tests.sh`; the seat's release gate
  decides them.
- Headless scenarios that drop or pick up items, EN and RU:
  `1060-campaign-040-reachable` and `1060-campaign-081-reachable` pass;
  `1005-doll-and-shop` stops at step 23 (`assert_shop`) exactly as engine main
  `bd1e652` does.
- `cmd/missionrun -trace -ticks 1`, EN: UNSUPPORTED nodes 0 on mission 10 and
  0 on mission 20, unchanged.
