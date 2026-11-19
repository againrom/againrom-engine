# The second game's outcome report and teleport movie stand where the original puts them

## Result

After a won mission in the second game the report is a framed popup at the original's size, centred on the screen,
with the Victory and Continue buttons where the original puts them. The failure page has the original's width and
button places. A movie, the teleport movie included, plays at its original place, drawn at its own size, centred on
the screen with black around it. The first game's reports and movies look as before.

## Authority

- Owner: triage card o-r2t-1, fix by researching the original's layout and repeating it.
- `R2-ENGINE-1300`, `R2-ENGINE-1301`, `R2-ENGINE-1302`, `R2-ENGINE-1307`, `R2-ASSET-243`, `R2-ASSET-244`.
- DIV-2393 and DIV-2396 narrow to what the claims leave Unknown.

## Rules

1. When the second game shows the victory report, the popup is 296x168 at ((W-296)/2, (H-168)/2) of the screen.
2. The victory buttons are page-relative (40,84,248,108) and (40,108,248,132); the words are centred in (40,32,232,80).
3. The failure page is 488 wide at (W-488)/2 and its buttons span x 61..427, each 24 high.
4. A 640x360 movie in the second game stands at (0,60) on a 640x480 screen and (192,204) on 1024x768, undoubled.
5. The first game's pages and movie place do not change.

## Code

| kind | its one builder | change |
|---|---|---|
| outcome report window | `pkg/ui/notice.go` layouts, `pkg/ui/dialog.go` | rectangles from the game's profile data |
| screen-mode layout | `pkg/ui/missionscreen.go` MissionScreen | `Outcome`, `MovieOnScreen` fields |
| movie placement | `pkg/ui/cutscene.go` composeCutscene | `movieRect` centres on the profile's screen |

## Proof

| test | rules |
|---|---|
| `TestSecondGameSuccessPopupAndButtons` | 1, 2 |
| `TestSecondGameFailurePageGeometry` | 3 |
| `TestMovieRectAtEachMode`, `TestCutsceneScreenFollowsProfile` | 4, 5 |
| `TestFirstGameOutcomePagesAreAuthored`, `TestMissionScreenIsEachGamesData` | 5 |

Release tests at the release: none.

## Open debt

- The failure page's height, vertical place, title row and screen darkening stay as authored (Unknown in claims).
