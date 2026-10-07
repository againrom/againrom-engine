# Earned score and the campaign ending

## Intent and authority

The ending runs as the original runs it, and the earned score counts the bodies
the original counts. Authority: `FAME-026` (the score source is one cached
drawable, the first registered on an empty client map; its experience word
arrives by mask-4 state packets), `FAME-027` (the x87 control word at the score
call is Unknown), `FAME-028` (resume projects every actor with no stage
baseline, so a loaded hostile body at stage 2..4 meets the increment arm; the
counting consequence is Medium), `FAME-029` (credits end on a key, a mouse press
or the roll reaching its last line; the hall ends on its button; the terminal
route resets the campaign; message 442 is the one SAVE caller SAV-972 does not
exclude), `SAV-970..972`. Owner direction: where a claim is Unknown keep the
current behaviour and say so in a row.

## As built

- `fameObserver` binding keeps the baseline: loaded bodies earn nothing, so a load
  and an unchanged save keep the saved counter. The native recount of loaded
  hostile stage 2..4 bodies (FAME-028, Medium) is not adopted; DIV-1268 and
  DIV-2065 record it.
- The terminal Victory acknowledgment opens the installed credits roll. A key,
  a mouse press or the first visible line reaching the last line ends it into
  the hall page. The hall's button (also Enter on it and Escape) calls
  `leaveCampaignEnding`, which resets the session as a new game does, and shows
  the main menu. The summary page and its three-way choice are gone. Without an
  installed roll the ending's text page leads to the hall.
- A result the hall store has not accepted keeps the completed campaign and
  puts a note on the hall page; entering the hall retries the write.
- The main-menu hall view resets nothing.
- The arithmetic stays 53-bit nearest rounding (FAME-027 Unknown).
- `ui.flow` keeps its four ending callbacks in one field (`endingSeams`).

## Proof

- `TestFameBindKeepsSavedCounterWithLoadedBodies`.
- `TestEndingOpensCreditsRollThenHallAndKeysEndTheRoll`,
  `TestEndingCreditsRollEndsWhenLastLineIsFirstVisible`,
  `TestEndingHallEndsOnlyOnItsButton`, `TestEndingMenuHallDoesNotResetCampaign`,
  `TestEndingTextCreditsThenHallThenResetAndMenu`,
  `TestEndingHallShowsPendingResultNote`,
  `TestLeaveCampaignEndingResetsOnlyARecordedCampaign`.
- Release, EN and RU: `TestReleaseHeisdeadCampaignEnd` (the owner's mission-150
  SAV loads and runs Victory, credits, hall, reset),
  `TestReleaseCampaign1180EndingCreditsHallSaveAndReset`,
  `TestReleaseFame1181TerminalResultColdHallAndSAV`,
  `TestReleaseCampaign1176TerminalAndSideRecovery`,
  `TestReleaseTextSmoothingKeepsEveryGlyphOfEveryNamedScreen`.
- The owner's eight completed-campaign saves load on EN with `cmd/savecheck`.

## Open debt

DIV-1264, DIV-1267 and DIV-1268 narrowed; DIV-2060 credits step size and the
23 ms gate, DIV-2061 hall keys and message 442, DIV-2062 unrecorded result,
DIV-2063 reset field set, DIV-2064 source identity, DIV-2065 observation point and the loaded-body recount.
