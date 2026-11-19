# Second-game triggers show the second game's operations

## Result

The trigger window of a second-game map in the editor lists the map's triggers. Each check and action shows the second
game's own operation name where the research reads what it does. Where it does not, the row says "Operation N, not
decoded" and keeps its raw parameters. Compiler parameters show as such and not as actions or checks. The window states
the second game's 1000 registers. The window only lists and shows: editing stays off.

## Authority

- Owner: «Еще задача это редактор rom2»; one builder per kind, the first game as profile data.
- R2-ASSET-247, R2-ENGINE-1312 to R2-ENGINE-1315: the second game's operation ids and the read effects of each.
- R2-ENGINE-624, R2-ENGINE-627: the failure reason that actions 4 and 5 write, and check 18 raises.
- R2-ENGINE-042, R2-ENGINE-043: full word 65538 is a compiler parameter and creates no executable arm.
- R2-SESSION-019: the second game's session holds 1000 registers.

## Rules

1. Every check and action of a second-game map is a named arm or a "not decoded" row with its raw id.
2. Operation 65538 shows as a compiler parameter.
3. The window and the variable rows use the second game's register limit, not the first game's.
4. A second-game map refuses every script edit and arm.
5. The first game's window, arms, limits and script writing behave as before.

## Code

| kind | its one builder | change |
|---|---|---|
| script profile | `scriptprofile.Profile` | a vocabulary field; `Editor.Profile` picks it from the map |
| operation names | `mapload.ScriptOperationNameIn` | a second table keyed by the profile |
| arm table | `mapedit.armTable` | keyed by profile and arm; the second game has no rows |
| trigger window rows | `inspectScript`, `MapEditor.opName` | take the profile from the map |

## Proof

| test | rules |
|---|---|
| `TestReleaseSecondMapScriptRowsAreNamedOrNotDecoded` | 1, 2, 3 |
| `TestSecondGameOperationNames` | 1 |
| `TestSecondMapFormatSurvivesCloneAndRefusesSave` | 4 |
| `TestReleaseFirstMapScriptKeepsItsProfile` | 5 |

Release tests at the release: both second-game installs and the first-game install.

## Open debt

- Operations whose effects the research leaves Medium or Unknown stay "not decoded"; no name or operand type is made up.
- Script writing for the second game awaits a supported second-game writer.
