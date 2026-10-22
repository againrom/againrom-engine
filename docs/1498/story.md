# One town dialogue builder for both games

## Intent

The town talk window of the first and the second game is one kind. One builder now owns its state,
its picture and its pointer contract; each game's screen holds one instance and gives it the game's
pages, art and profile. Neither game's frames, World hashes or SAV bytes change.

## Authority

- Owner ruling "ROM1 is a reduced ROM2" (`pipeline/OWNER-RULINGS.md`): a kind is built once; each
  game is that builder with profile data. Facts, texts, art and resources stay per game.
- Owner rule "One builder per kind" (the project `AGENTS.md`): data in, the existing instance
  built unchanged, the new one added as data.
- Gap: `pipeline/BACKLOG.md`, "Town dialogue".

## Before: what each path held

| Item | First game (`townScreen`) | Second game (`secondCampaignScreen`) |
| --- | --- | --- |
| Revision | `dialogueRevision`; +1 on open, on an acting turn, on the square's `dialogue-reset` | `dialogueRevision`; +1 on open and on every turn while a campaign is held |
| Button state, backdrop | `dialogueButtonState`, `dialoguePolicy`; cleared by `resetForNewGame` | `dialogueButtonState`, `dialoguePolicy`; never cleared (the screen is rebuilt per `TownScreen` call) |
| Shows | `townDialogueState.shows`: conversations opened while the window stayed open | 1 while open, else 0 |
| Open | `room == roomTalk` | `dialoguePart` finds the campaign's `payload` part |
| Page cursor | `said`; `townDialogueState` reads part text, speaker, tips, face | `secondCampaign.payload`, `part` |
| Text | the part body untrimmed, or the mercenary diagnostic | the part body, `strings.TrimSpace` |
| Portrait shape | mercenary talk or a named speaker | never |
| Face, face window | speaker or mercenary talk picture and its record window | none |
| Layout | `Words.OnLayout(AuthoredDialogueLayout())`, `Frame` = menu frame, portrait, button state, backdrop | same, `Frame` = `gameMenuArt`, portrait off |
| No font | no picture, no frame | picture call returns `nil, true` |
| `TownDialogueFrame` | the frame art and body | not implemented |
| Button | open and the rectangle has area | open; a closed window answered the rectangle with `false` |
| Turn | `said++`; a failed lookup closes into the room it came from and re-reads the room tip | `part++`; a missing part clears `payload` and `part` |

`TownDialogueActive` (town speech) is first-game speech state, not the window, and is unchanged.

## As built

`pkg/ui/towntalk.go`, next to `NoticeLayout`, `DialogFrame` and `DialogueButtonState`:

- `ui.TownTalk` holds revision, shows, button state and backdrop. `Open`, `Touch`, `Reset`,
  `SetVisual` change them. `Layout`, `Picture`, `Frame`, `Button`, `Shows` read a
  `ui.TownTalkView`; `Advance(turn)` runs the game's page turn and counts an acting turn.
- `ui.TownTalkView` is the data in: `Profile`, `Open`, `Page` (text, face, face window, portrait
  shape) and `Art` (words, frame, font), read from the game's own resources on each call.
- `ui.TownTalkProfile` switches: `CountShows`, `NeedsFont`, `ExposesFrame`, `ButtonNeedsArea`.

Profiles: `firstTownTalkProfile` (`pkg/game/townscreen.go`) sets all four;
`secondTownTalkProfile` (`pkg/game/secondcampaign.go`) sets none. Each screen holds one
`talkWindow ui.TownTalk`, builds its view in `talkView`, and turns its own pages in
`turnTownTalk`. Its eight `TownDialogue*` methods only pass the view to the window.

Removed: `dialogueRevision`, `dialogueButtonState`, `dialoguePolicy` on both screens,
`townDialogueState.shows`, `townScreen.townDialogueLayout`, `secondCampaignScreen.dialogueLayout`,
and each screen's own layout, render, frame, button and revision code.

## Differences kept as profile data

- Shows: the first game counts reopenings (`CountShows`); the second answers 1.
- No font: the first game draws nothing (`NeedsFont`); the second calls `RenderNotice`, which
  returns `nil`, and reports the window shown.
- Frame: the first game answers its frame to the frame composer (`ExposesFrame`); the second
  answers `nil`, as `ui.townDialogueFrame` did for a screen without the method.
- Button: the first game requires area (`ButtonNeedsArea`); the authored rectangle has area in both.
- Turn on a closed window: the first game's `turnTownTalk` does nothing; the second's turns and
  counts while a campaign is held. Callers reach it only while the window is shown.
- Portrait, face and text trimming are page data from each game's own reader.

Page cursors stay where they were. The second game's `payload` and `part` remain campaign state:
`savePoint`, `noticePending` and the world map read them, and a new campaign or a load replaces the
campaign and so closes the talk.

One answer changed shape without a player effect: a closed second-game window's `TownDialogueButton`
returns the zero rectangle instead of the authored one; both return `false`, and every caller
(`ui.townDialogueButton`, tests, `cmd/*` witnesses) reads the rectangle only after `true`.

## Proof

- `TestTownTalkPagesThroughOK` (both profiles): closed window offers no button and no picture;
  open counts revision 1; button is the authored rectangle; a press draws the pressed button and
  turns nothing; each release turns one page and counts one revision; the last closes; a turn that
  does not act counts nothing.
- `TestTownTalkProfilesKeepEachGame`: shows, frame and font answers per profile.
- `TestTownScreensHoldOneTownTalk`: each screen holds exactly one `ui.TownTalk` and no
  `DialogueButtonState` or `DialogueBackdrop` field; each of the 16 `TownDialogue*` methods reaches
  `talkWindow`; no method of either screen calls `RenderNotice`, `AuthoredDialogueLayout`,
  `WithDialogueButtonState` or `WithDialogueBackdrop`.
- `TestReleaseTownTalkOKThroughTheWindow` (EN, RU) and `TestReleaseSecondTownTalkOKThroughTheWindow`
  (rom2-en, rom2-ru): App pointer press and release on OK over every page of an installed inn talk;
  the pressed page equals the independently composed authored notice; the last release closes.
- The first-game release set and the second-game town, inn and save tests; results are in the
  lane's review notes.

## Open debt

- The two page readers stay separate: `townDialogueState.lookup` (first game) and `dialoguePart`
  (second game) differ in trimming and speaker handling. Unifying them is a pages story, not a
  window story.
- The second game builds a new `secondCampaignScreen` on each `FrontEnd.TownScreen` call, so a
  caller outside the App reads a window with revision 0.
