# Verification

Code tested at `056f4dfda507792a020327057bc18b23f39594d6`, containing final
master `d53afa394675942d7461e772f1646efaf0fd2acc` (1099 and 1100).
Research pin remains `ba21c9aa9a949023b3d678b22ca29b3a3b0cd95f`.

## Observable result

Both lawful asset roots ran the actual App original-LOAD, school Train and
ordinary SAVE dispatch over the same Reniesta/Danath city source:
`gameversions/saves/2026-08-15/game0010.sav`, SHA256
`89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4`.
Reniesta school slot1 changed gold `683 -> 483`, HP maximum `22 -> 23`,
mana maximum `137 -> 139`. Existing book presence and four Spell instances
survived. Integer range refresh changed zero slots in this particular save.
Ordinary SAVE produced 3221-byte SAV, SHA256
`8aafe6f08a879764480be941976dbf7b4be2f4671503d550983b1a906dfbed0a`.
Fresh SAV -> AGS -> SAV processes reproduced those bytes exactly. A fresh
mission retained the trained sheet and book; native wind-up continuation
retained the same instance state and subsequent canonical hash.

Fergard's mission source `gameversions/saves/2026-08-24/game0021.sav`, SHA256
`7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c`, independently
checks all 28 five-byte Spell prefixes at body offsets `1119 + 11*i`.
On EN and RU, actual Light release followed eight ticks after the saved wind-up;
mana changed `145 -> 140`. A fresh AGS process retained all Book fields and
matched continuation. Before this slice, only membership reached actors and
source-backed mage school training was unavailable. These are App/headless
production-path observations, not original-runtime or independently recorded
locale observations. No GUI was driven or claimed observed.

## Independent controls

- Literal 113-byte book record; distinct instances of one ID; legacy, absent,
  present-empty and present-nonempty states; malformed decode is atomic.
- Actual read consumes one book, absent remains absent, duplicate read/equip
  leaves parameters unchanged. AI uses exact Defensive zero; point attribution
  skips only byte1. Signed mana boundaries include 32767, 32768 and 65535.
- Saved admission/retry/release uses cached range without a second power term.
  Refresh tests cover ordinary, zero and Teleport ranges, retaining mana and
  Defensive. Actual scroll use and weapon projection keep their own sources.
- Synthetic mage training checks literal final Human fields and a manual
  whole-graph allowed-field diff, not writer-versus-reader agreement alone.
  Shared Spell conflicts leave purse/party unchanged; agreeing aliases serialize.
- Every prior readable form widens to explicit legacy book state. Actual form70
  341-byte entities and 173-byte dead records remain intact. All previous digest
  controls remain asserted; form71 adds 113 bytes per entity.
- Historical gob envelopes `782e7e3e...` and `8a00d92c...` remain frozen decode
  fixtures. The latter was generated from exact published `176de1ab` using an
  outside-tree Go overlay and verified by full SHA before embedding. The clean
  temporary checkout was removed. Today's expanded-descriptor form69 control
  is `57fb1053...`; today's form71 envelope is `f0221513...`. Neither replaces
  an old-byte assertion.

## Gates and limits

`go test -trimpath -count=1 ./...`: PASS. Focused sim, game, mapload, data and
SAV tests: PASS. Changed Go files are gofmt-clean. Asset tree scan: clean.
Allocation sweeps before/after ledger edits: 32 ledgers, zero missing answers.
`check-div-claims.sh`: PASS; existing partial-retraction notices retained.
DIV-650/651 close; DIV-690/691 use only the reserved range. No Co-Authored-By
trailers. Preserved installs: 181 files, both roots unchanged.

The three new gated tests pass independently on EN and RU, including their
fresh-process children. The full paired population is 133 tests in eight
packages. Its first sandbox invocation failed on both roots solely at the
386 cutscene-helper build: Git VCS stamping exited128. One already-started
diagnostic rerun with `GOFLAGS=-buildvcs=false` built that helper but failed its
launch with `Access is denied` on both roots. Neither invocation is a passing
host release gate; no test was excluded or relabeled skipped. The root's final
merged host paired gate remains required.

Selected production scenarios `0154-synthetic-spells`, `1013-world-map-one-click`,
`0163-chargen-mission10` and `0163-mission-to-town`: 4/4 on each asset root,
using diagnostic `GOFLAGS=-buildvcs=false`. Endpoint fixtures are explicit
synthetic endpoints, not reconstructed original campaign playthroughs.

Built missionrun from this checkout. EN mission10/20 `-trace -ticks 1` reported
UNSUPPORTED `0 / 0`; script populations `16/27/12` and `14/15/11` match
`pipeline/milestone-baseline.txt`. Parent's prior1100 EN census was also `0 / 0`.
This slice moves the playable SAVE/cast result, not the script-gap census.

Raw gate receipts are outside Git under `.cache/fixture1101/` at the seat:
`full-go.log`, `release.log`, `release-buildvcsfalse.log`, `mage-en.log`,
`mage-ru.log`, and `scenarios.log`.

No new research, first-book allocator, generated Human/world SAV writer,
original process, lawful-install write, or root-master edit was performed.
Old native books remain table-backed (DIV-652). Shared-instance propagation
is conservatively refused (DIV-691). Later native Human rederivation and
mission-world export remain DIV-675.
