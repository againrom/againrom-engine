# Story 1306: mission-start exits and cancelling the Load dialog from the failure panel

## Intent

Bring `DIV-1672` and `DIV-1674` to the evidence of snapshot 142. No engine
behaviour changes; the result is the pin, the two ledger rows and one test.

## Authority

Research EXP-0459, pinned at snapshot 142. Claims `MISSION-067`, `MISSION-068`,
`MISSION-069`, `VIDEO-078`, `VIDEO-079`, and the amended `MISSION-DEFEAT-059`,
`MISSION-LOAD-061`, `VIDEO-MUSIC-063`, `VIDEO-MUSIC-064`. Owner policy
`DIV-610` keeps Cancel and Escape on the failure panel.

## As-built behaviour

- Original: Cancel, Escape and Load with no selection return `0x446` and take
  the Exit route (stop, menu list). A window close stops the player and
  requests no list (`MISSION-067`, `VIDEO-078`).
- Engine: Cancel and Escape return to the failure panel with no music call
  (`DIV-610`, `DIV-1672`, DEVIATION). A window close ends the run through the
  App step and requests no list; the stop and the absent list match the
  original, so no window-close difference remains.
- `DIV-1674`: the engine has no wait loop. The loop's zero exits need a quit
  message, 60,000 ms of silence or a failed map open (`MISSION-069`). No
  shipped-data condition feeding them was found (Medium). The row is narrowed
  to the Unknown writers of `frame+0x3b4`, the absence of a shipped mission map
  file and the surface left behind.

## Proof

`pkg/ui/music_load_test.go`:
`TestWindowCloseOverTheLoadDialogEndsWithoutAMusicRequest` opens the Load
dialog from a mission and asserts the close ends the run with no device event
and no logged request. The Cancel and Escape routes are covered by the
existing `TestRefusedAndCancelledLoadsLeaveTheMusicAlone` and
`TestMissionLossPanelMakesNoMusicRequest`.

## Open debt

`DIV-1672` stays an owner decision under `DIV-610`; Escape is Medium and other
inputs are Unknown (`MISSION-067`). `DIV-1674` stays Unknown until the writers
of `frame+0x3b4` and an absent shipped map file are established.
`DIV-2137` through `DIV-2140` were not used.
