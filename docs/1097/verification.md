# Verification

## Measured result

The pre-story source loaded as a world with spent trigger latches and no
actionable outcome. The implementation now presents saved Victory at tick zero.
`scenarios/1097-original-terminal-victory.json` passes all 13 production App
steps on EN and RU: original load, unchanged purse600 across 64 paused frames,
Victory, world-map return, automatic town arrival, purse1100, and two permanent
characters with XP1795/1593. It does not move an escort or force WIN.

The source is `gameversions/saves/2026-08-02/game0009.sav`, SHA256
`60267c82072c77446ab9b34913318e89eab8f70e49f3510ae64aaaf423819bd6`.
Research's archive walker independently locates the Player body. Literal reads
at decoded offsets109 and119 give participant0 and outcome1. Session location
77244 originated in the production reader; independent literal reads at81606,
81614 and77651 give WIN1, LOSE0 and latch7=1. That session locator is not a
second structural oracle.

Separate `savecheck` processes import the original, SAVE native, exit, and
load that native file on both roots. Both stages report tick0, purse600,
57 entities and hash `54753128cef5f5dd`. The first frame retains Victory.
All 28 direct corpus SAV files load on each root. No original executable runs.

The release test also saves after town arrival and reloads through App. Purse
stays1100 and character state is unchanged. The source's already-spent script
reward is not replayed; the installed completion reward is exactly500. The
separate fresh-endpoint legacy fixture legitimately ends with1600 instead.

## Contract coverage

- Outcome and counters: 48 synthetic combinations of outcome0/1/2 and each
  counter0/1/2/uint32-max survive exact native byte/hash round trips. Reporter
  tests retain exact-one, loss-first comparisons, permit Won-to-Lost, and keep
  Lost terminal. The byte layout and version remain unchanged.
- Validation: invalid outcome, absent/duplicate human participants, truncated
  session spans and malformed latches are refused before mutation. The existing
  detached-slice and canonical rollback tests remain active. Builder-owned
  trigger registers are not overwritten.
- Player route: first-frame original/native terminal panels, native before
  acknowledgement, Continue-to-failure, reload/exit-only defeat, unchanged XP,
  no old dialogue or reward replay, and city-to-fresh-mission reset are tested.
- Saved defeat: a disclosed synthetic mutation of the source sets outcome2 and
  counters0/0. EN/RU native continuation keeps hash `2cdef4eafd7b1003`, shows
  failure and changes no XP or purse. This is not an original lost-file witness.

## Gates and remaining work

Focused packages `pkg/game`, `pkg/sim`, `pkg/formats/sav` and
`internal/gatedtests` pass on base70403c7d with this slice. Four selected release
tests pass on EN and RU. The executable App scenario and two-process logs are
in seat-local `review/story1097/`.

The mission census is unchanged: baseline m10 has16 checks/27 instants/12
triggers; m20 has14/15/11 on both roots. The built `missionrun -trace -ticks1`
reports zero UNSUPPORTED nodes for each mission/root. This story changes saved
continuation, not the population compiled from fresh campaign maps.

Reconciled code `f299cf1ca74ee2743654f252fb55b4643b26db85` includes master
`b2e8945eba430eec2264170de3de7c7b5c869b9f`, including the sole1096 correction.
Full Go, gofmt, no-assets and diff checks pass. Both outcome release tests and
all13 actual App scenario steps pass again on EN/RU. Separate newly built
`savecheck` processes retain the original/native victory hash54753128cef5f5dd,
tick0, purse600 and57 entities on both roots. Synthetic defeat remains
2cdef4eafd7b1003. Receipts in `review/story1097/`: `reconciled-final-go.log`,
`reconciled-release-{en,ru}.log`, `reconciled-drive-{en,ru}.log` and
`f299cf1c-{save,load}-{en,ru}.log`.

The sole fresh review returned one legacy-native defect: a version56 Won world
opened Victory before its upgrade disclosure, then permanently lost the panel
when that disclosure was dismissed. The one correction makes load disclosure
defer both Won and Lost panels. The unchanged independent overlay now passes.
Eight permanent cases cover both outcomes, original/extended disclosure pages,
and immediate/deferred Victory: no page loss, no world ticks while pending,
no Continue permission invented by dismissal, one-shot Victory and terminal
reload-only failure. `correction-independent-overlay.log` records these tests
and the existing no-font/defeat controls. No second review is required.

The seat owns final merge gates and current-build promotion. Full-world SAV
authoring, pending orders/effects and original-game read/resave compatibility
remain outside this result.
