# Story 1255: music transitions at mission loss, Exit, Load and LOAD

## Intent

The music a player hears at a mission loss, at Exit to Main Menu, at Load Game
from the failure panel and at a LOAD of a mission or town save follows the
original's request sequence. Music is presentation only; no simulation, save or
hash state changes.

## Authority

Research EXP-0426, pinned at snapshot 109. Claims `VIDEO-MUSIC-061` through
`VIDEO-MUSIC-066`, `MISSION-DEFEAT-059`, `MISSION-LOAD-060`, `MISSION-LOAD-061`.
Confidence is High for the arm reads and Medium for the mask values and the
absence of other requests.

## As-built behaviour

The test seam is the controller's request log (`App.MusicRequestLog`) and the
device's stop/start order, not audio output. A log entry is `stop`,
`request:<list>` or `skip:<list>`.

| Route | Before | After |
|---|---|---|
| Failure panel displayed | no request | no request (witnessed) |
| Exit to Main Menu | stop, menu request | unchanged: `stop`, `request:menu.wav` |
| Load Game from the panel, dialog open | no request | unchanged (witnessed) |
| LOAD of a mission save | stop, mission request | unchanged: `stop`, `request:B00.wav..B11.wav`, also when that list plays |
| LOAD of a town save, Town list not playing | stop, Town request | unchanged: `stop`, `request:town.wav` |
| LOAD of a town save, Town list playing | stop, Town request | no stream change; `skip:town.wav` |

Only the last row changes behaviour. The mission owner never compares and
re-requests; the Town owner skips an equal list (`VIDEO-MUSIC-066`).

The original issues a stop at selection, another in the mission start, and one
inside the replace. The device receives one stop per request; the stop is
idempotent, so the observable order is stop then start.

Medium readings taken: the mask equals 1 at the failure panel and the main
menu mask is zero, so stops are issued where the claims place them; the skip
compares the request identity (list owner and school class order) in place of
the first array string.

## Proof

- `pkg/ui/music_load_test.go`: mission LOAD re-request, town LOAD skip control,
  town LOAD from a mission, Exit order, failure panel and load dialog silent,
  refused and cancelled LOAD silent.
- Release witnesses, run on the EN and RU installs:
  `TestReleaseLossThenExitStopsThenRequestsTheMenuList`,
  `TestReleaseLoadAfterLossRequestsTheMissionListAgain`,
  `TestReleaseTownSaveLoadRequestsTownListOnlyWhenItIsNotPlaying`. Each drives
  production App input and logs the request sequence.

## Open debt

`DIV-1531` and `DIV-1532` are closed. `DIV-1671` refused LOAD keeps the stream
running. `DIV-1672` cancelling the dialog stays on the panel, where the
original takes the Exit route. `DIV-1673` the skip compares request identity
and applies to owners no claim covers. `DIV-1674` mission-start exits that leave
the player stopped are not modelled.
