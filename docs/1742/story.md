# The second game's campaign flow follows the published claims; a town's gates add no companion

## Result

Leaving a town through its gates no longer brings Dina into the party. As in the original, only the victory in
mission 40 asks for her, and she joins before the next town opens. The rest of the campaign flow was checked
against the new published claims and already matched them: the hires each won mission offers, the movie after a
victory, the ending with credits, Hall of Fame and menu, and the defeat text each mission shows when a party
member falls. The first game is unchanged.

## Authority

- Owner: «продожаем по списку» (epic «ROM2: полная игра»): adopt the research.
- `R2-ENGINE-634` (High / Medium), `R2-ENGINE-744` (High / Medium), `R2-ENGINE-1592` (Medium): only the
  victory leave arm passes slot 773 to the addition producer; a town's gate leave adds no record. DIV-4188.
- `R2-ENGINE-1591` (Medium; High for the per-mission source table): defeat sections. DIV-4224.
- `R2-ENGINE-1593` (Medium): the departure movie and the ending sequence. DIV-2628, DIV-4752.

## Rules

1. When the party leaves a town through its gates, no party member joins, whatever slot 773 holds.
2. When a won mission's departure leaves slot 773 nonzero, that identity joins once before the next place opens.
3. When a fall loses a mission, reason 2..4 shows its mission text entry, else shared row 283 or 284.
4. The first game's town leave keeps its party.

## Code

| kind | its one builder | change |
|---|---|---|
| campaign stage step: joins | `pkg/game/secondparty.go` leftLocation | the won leave alone asks the producer |
| campaign stage step: town leave | `pkg/game/campaignsecond.go` destinationTownCampaign.leave | passes a gate leave |
| campaign stage step: departure | `pkg/game/winfinish.go` bankComplete | passes a won leave |
| defeat text | `pkg/game/secondgametext.go` secondGameFailureEntries | none; pinned on both roots |

## Proof

| test | rules |
|---|---|
| `TestSecondGateLeaveAddsNoDepartureCompanion` | 1, 4 |
| `TestSecondLeaveAfterFortyAddsIdentityTwentyThree`, `TestAbandonKeepsTheDepartureCompanion` | 2 |
| `TestReleaseSecondGameFailureSourceByMission`, `TestReleaseSecondGameFailureUsesInstalledFallback` | 3 |

Release tests at the release: `TestReleaseSecondGameFailureSourceByMission` on rom2-en and rom2-ru.

## Open debt

- DIV-2391, DIV-3111, DIV-4187, DIV-4225, DIV-4752: the original runs `R2-ENGINE-1590`..`1593` name.
- DIV-4696, DIV-4700: no claim answered them; the original runs of maps 101 and 53 settle them.
