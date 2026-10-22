# One part reader and one page cursor for the town talk window

## Intent

The town talk of both games, the mission notice and the map editor's message page now read event
text parts through one scan, and the town talk of both games pages through one cursor type. The
eight methods `pkg/ui` reads the talk window through are declared once. Neither game's frames,
World hashes, SAV bytes or scenario reports change.

## Authority

- Owner ruling "ROM1 is a reduced ROM2" (`pipeline/OWNER-RULINGS.md`) and the rule "One builder per
  kind": one builder, each game as profile data.
- Gap: architecture audit `pipeline/reviews/arch-audit-29d32509.md`, rank 3 and rank 20, and the
  owner's decision on it: one reader and one cursor.
- Refactor only; no ROM fact changes.

## Before

Readers (base `4f324f10`):

| Reader | Where | Used by |
| --- | --- | --- |
| `townDialogueState.lookup` | `pkg/game/towndialogue.go:29` | first-game town talk, `townLines` |
| `eventPartRange` | `pkg/game/eventtext.go:453` | `eventPartTag`, `eventPartTail`, `dialoguePart` (second-game town talk, mission notice), `dialoguePartTips`, `EventPart`, `EventPartSpeaker`, the map editor's `messageHeader` and `SetMessage` |

Both walk `<`...`>`, take the first tag that contains `part=<n>` (ASCII case folded) and survives
the audience, stop at an unterminated tag, cut the text at its first NUL and take the town words
with `acceptedDialogueTail` (from the first LF after the tag to the last CR before the next `<`, or
to the end; no LF is no part). Per case, both answer the same:

| Case | Answer |
| --- | --- |
| missing part, part past the last | not found; the first game keeps the words shown |
| empty part (`<part=2>\r\n\r\n<`) | found, empty words |
| no LF anywhere after the tag | not found |
| LF only after the next tag | found; the words run on from that LF |
| last part | the words run to the end of the text or the first NUL |

Cursors: the first game's `townScreen.said` (`pkg/game/townscreen.go:99`) with the payload in
`townDialogueState.payload`; the second game's `secondCampaign.payload` and `part`
(`pkg/game/secondcampaign.go:35`). Turns: `townScreen.turnTownTalk` (`townscreen.go:812`),
`secondCampaignScreen.turnTownTalk` (`secondcampaign.go:448`). Forwarding: eight `TownDialogue*`
methods on each screen.

The map editor's two expressions (`pkg/game/mapmessagepages.go:20`) are not a part rule:
`npc=([0-9]+)` finds the speaker to substitute in a header, `<[^>]*>` strips markup from a line.
Its part lookup already went through `eventPartRange`.

## As built

- `scanEventPart(payload, n, audience, candidate)` (`pkg/game/eventtext.go`) is the one scan;
  `eventPartRange` calls it without a candidate.
- `readDialoguePart` and `dialogueRead` (`pkg/game/dialogueinput.go`) are the one town words
  reader: the accepted tag, its words, and `tips()`. `dialoguePart` and `dialoguePartTips` call it.
- `talkCursor` (`pkg/game/towntalk.go`) holds the payload and part: `open`, `next`, `close`,
  `read`. The first game's screen holds one as `talkPage`; the second game's campaign holds one as
  `talk`. No other part or payload field remains on either.
- `townTalkSeat` (`pkg/game/towntalk.go`) holds the `ui.TownTalk` window and the screen as a
  `townTalkSource` (`talkView`, `turnTownTalk`), and declares the eight methods once. Both screens
  embed it; the screen is bound at construction and again by `openTalk`.
- `townDialogueState.lookup` takes a cursor and reads through `readDialoguePart`.

## Differences kept

Each is first-game data in `pkg/game/towndialogue.go`; the shared scan does not carry them.

1. Lowest part: the first game reads no part below 1 (`firstTownTalkLowestPart`); the scan reads
   part 0. No cursor of either game reaches part 0.
2. Speaker of a rejected tag: every tag naming the part, accepted or rejected by the audience,
   sets the first game's portrait flag and, when it names one, the speaker (`noteSpeaker`, passed
   as the scan's candidate). The second game reads no speaker.
3. Tips: the first game stores the accepted tag's `tips=` even when the words fail; the second
   game and the mission notice ask `dialoguePartTips` separately. Same rule.

The second game's `strings.TrimSpace` of the words stays in its `talkView`, as page data.

The first game's eight methods answered a nil `*townScreen`; the seat's methods do not. No caller
holds a nil screen (`pkg/game/tips_test.go` names the defensive convention).

No DIV row: no behaviour differs from main.

## Proof

- `TestTownTalkReaderPartCases`: both games' text shapes, every case in the table above, through
  each game's own reader.
- `TestTownTalkReaderLowestPart`, `TestTownTalkReaderFirstGameTagState`: the kept differences.
- `TestTalkCursorTurnsAndReopens`, `TestFirstTownTalkCursorThroughTheSeat`,
  `TestSecondTownTalkCursorThroughTheSeat`: turn, last page, reopen, through each screen's seat.
- `TestTownScreensHoldOneTownTalk`: the eight methods exist once, on the seat, and reach the
  window; neither screen declares one. `TestTownTalkHasOneCursor`: one cursor per game.
- Loss control: a broken part number in `scanEventPart` fails both games' reader and cursor
  tests; results are in the lane's review notes.
- `TestReleaseTownTalkOKThroughTheWindow` (EN, RU), `TestReleaseSecondTownTalkOKThroughTheWindow`
  (rom2-en, rom2-ru), the release set and the second-game save groups; results are in the lane's
  review notes.

## Open debt

- The mission notice keeps its own cursor (`world.go`, `m.part`) beside its page list; it reads
  through the shared reader.
- `pkg/ui` still reads the window through eight methods; one interface in `pkg/ui` would change
  its fakes in nine test files.
